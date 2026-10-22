package authtest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

// Client is an auth.Client verifying tokens from a throwaway IDP, which it also signs them with.
type Client struct {
	*auth.Client
	*Signer
}

// Signer mints tokens for a throwaway IDP.
type Signer struct {
	key      jwk.Key
	keys     jwk.Set
	issuer   string
	audience string
}

// NewUntrustedSigner creates a signer whose key no auth.Client trusts.
func NewUntrustedSigner(t testing.TB) *Signer {
	t.Helper()

	return newSigner(t)
}

func newSigner(t testing.TB) *Signer {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	key, err := jwk.Import(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	if err := key.Set(jwk.KeyIDKey, "test-kid"); err != nil {
		t.Fatal(err)
	}

	keys := jwk.NewSet()
	if err := keys.AddKey(key); err != nil {
		t.Fatal(err)
	}

	return &Signer{key: key, keys: keys, issuer: "issuer", audience: "audience"}
}

func New(t testing.TB, db db.Client) *Client {
	t.Helper()

	signer := newSigner(t)
	jwks := signer.publicJWKS(t)

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write(jwks)
		}),
	)
	t.Cleanup(server.Close)

	authClient, err := auth.New(
		t.Context(),
		db,
		zap.NewNop(),
		auth.Config{JWKSURL: server.URL, Issuer: signer.issuer, Audience: signer.audience},
	)
	if err != nil {
		t.Fatal(err)
	}

	return &Client{Client: authClient, Signer: signer}
}

func (s *Signer) publicJWKS(t testing.TB) []byte {
	t.Helper()

	publicSet, err := jwk.PublicSetOf(s.keys)
	if err != nil {
		t.Fatal(err)
	}

	jwks, err := json.Marshal(publicSet)
	if err != nil {
		t.Fatal(err)
	}

	return jwks
}

func (s *Signer) Token(t testing.TB, sub string) string {
	t.Helper()

	return s.TokenWithScopes(t, sub, nil)
}

func (s *Signer) TokenWithScopes(t testing.TB, sub string, scopes []string) string {
	t.Helper()

	token := jwt.New()

	if err := token.Set(jwt.SubjectKey, sub); err != nil {
		t.Fatal(err)
	}

	if err := token.Set(jwt.IssuerKey, s.issuer); err != nil {
		t.Fatal(err)
	}

	if err := token.Set(jwt.AudienceKey, []string{s.audience}); err != nil {
		t.Fatal(err)
	}

	if err := token.Set(jwt.ExpirationKey, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	if scopes != nil {
		if err := token.Set("scope", strings.Join(scopes, " ")); err != nil {
			t.Fatal(err)
		}
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), s.key))
	if err != nil {
		t.Fatal(err)
	}

	return string(signed)
}
