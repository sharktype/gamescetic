package test_auth

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
)

type IDP struct {
	JWKSURL  string
	Issuer   string
	Audience string
	Cache    *jwk.Cache

	signingKey jwk.Key
}

func NewIDP(t *testing.T) *IDP {
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

	cache, err := auth.NewJWKSet(t.Context(), server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return &IDP{
		JWKSURL:    server.URL,
		Issuer:     "issuer",
		Audience:   "audience",
		Cache:      cache,
		signingKey: signingKey,
	}
}

func (i *IDP) Config() auth.Config {
	return auth.Config{
		JWKSURL:  i.JWKSURL,
		Issuer:   i.Issuer,
		Audience: i.Audience,
	}
}

func (i *IDP) Token(t *testing.T, sub string) string {
	t.Helper()

	return i.TokenWithScopes(t, sub, nil)
}

func (i *IDP) TokenWithScopes(t *testing.T, sub string, scopes []string) string {
	t.Helper()

	token := jwt.New()

	if err := token.Set(jwt.SubjectKey, sub); err != nil {
		t.Fatal(err)
	}

	if err := token.Set(jwt.IssuerKey, i.Issuer); err != nil {
		t.Fatal(err)
	}

	if err := token.Set(jwt.AudienceKey, []string{i.Audience}); err != nil {
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

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), i.signingKey))
	if err != nil {
		t.Fatal(err)
	}

	return string(signed)
}
