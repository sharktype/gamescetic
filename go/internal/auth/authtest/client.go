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

	signingKey jwk.Key
	issuer     string
	audience   string
}

func New(t *testing.T, db db.Client) *Client {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	signingKey, err := jwk.Import(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	if err := signingKey.Set(jwk.KeyIDKey, "test-kid"); err != nil {
		t.Fatal(err)
	}

	set := jwk.NewSet()
	if err := set.AddKey(signingKey); err != nil {
		t.Fatal(err)
	}

	publicSet, err := jwk.PublicSetOf(set)
	if err != nil {
		t.Fatal(err)
	}

	jwks, err := json.Marshal(publicSet)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write(jwks)
		}),
	)
	t.Cleanup(server.Close)

	issuer := "issuer"
	audience := "audience"

	authClient, err := auth.New(
		t.Context(),
		db,
		zap.NewNop(),
		auth.Config{JWKSURL: server.URL, Issuer: issuer, Audience: audience},
	)
	if err != nil {
		t.Fatal(err)
	}

	return &Client{
		Client:     authClient,
		signingKey: signingKey,
		issuer:     issuer,
		audience:   audience,
	}
}

func (c *Client) Token(t *testing.T, sub string) string {
	t.Helper()

	return c.TokenWithScopes(t, sub, nil)
}

func (c *Client) TokenWithScopes(t *testing.T, sub string, scopes []string) string {
	t.Helper()

	token := jwt.New()

	if err := token.Set(jwt.SubjectKey, sub); err != nil {
		t.Fatal(err)
	}

	if err := token.Set(jwt.IssuerKey, c.issuer); err != nil {
		t.Fatal(err)
	}

	if err := token.Set(jwt.AudienceKey, []string{c.audience}); err != nil {
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

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), c.signingKey))
	if err != nil {
		t.Fatal(err)
	}

	return string(signed)
}
