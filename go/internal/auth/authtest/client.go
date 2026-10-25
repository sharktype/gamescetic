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

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/sharktype/gamescetic/internal/auth"
	"github.com/sharktype/gamescetic/internal/db"
)

// Client verifies tokens from a throwaway IDP.
type Client struct {
	*auth.Client
	*Signer
}

type Signer struct {
	key      jwk.Key
	issuer   string
	audience string
}

func NewSigner(t testing.TB) *Signer {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	key, err := jwk.Import[jwk.Key](privateKey)
	if err != nil {
		t.Fatal(err)
	}

	if err := key.Set(jwk.KeyIDKey, "test-kid"); err != nil {
		t.Fatal(err)
	}

	return &Signer{key: key, issuer: "issuer", audience: "audience"}
}

func New(t testing.TB, db *db.Store) *Client {
	t.Helper()

	signer := NewSigner(t)
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
		auth.Config{JWKSURL: server.URL, Issuer: signer.issuer, Audience: signer.audience},
	)
	if err != nil {
		t.Fatal(err)
	}

	return &Client{Client: authClient, Signer: signer}
}

func (s *Signer) publicJWKS(t testing.TB) []byte {
	t.Helper()

	publicKey, err := jwk.PublicKeyOf(s.key)
	if err != nil {
		t.Fatal(err)
	}

	publicSet := jwk.NewSet()
	if err := publicSet.AddKey(publicKey); err != nil {
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

	return s.TokenWithClaims(t, sub, nil)
}

func (s *Signer) TokenWithScopes(t testing.TB, sub string, scopes []string) string {
	t.Helper()

	var claims map[string]any
	if scopes != nil {
		claims = map[string]any{"scope": strings.Join(scopes, " ")}
	}

	return s.TokenWithClaims(t, sub, claims)
}

func (s *Signer) TokenWithClaims(t testing.TB, sub string, claims map[string]any) string {
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

	for name, value := range claims {
		if err := token.Set(name, value); err != nil {
			t.Fatal(err)
		}
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), s.key))
	if err != nil {
		t.Fatal(err)
	}

	return string(signed)
}
