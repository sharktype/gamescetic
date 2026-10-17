package routing

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/sharktype/satchel/internal/auth"
)

// No tests in this file; this is just the identity provider (IDP) helper.

type testIDP struct {
	JWKSURL  string
	Issuer   string
	Audience string

	cache      *jwk.Cache
	signingKey jwk.Key
}

func newTestIDP(t *testing.T) *testIDP {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
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

	// jwx ignores JWKS keys that do not name an algorithm.
	if err := signingKey.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
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

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(jwks)
	}))
	t.Cleanup(server.Close)

	cache, err := auth.NewJWKSet(t.Context(), server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return &testIDP{
		JWKSURL:    server.URL,
		Issuer:     "issuer",
		Audience:   "audience",
		cache:      cache,
		signingKey: signingKey,
	}
}

func (i *testIDP) token(t *testing.T, sub string) string {
	t.Helper()

	return i.tokenWithScopes(t, sub, nil)
}

func (i *testIDP) tokenWithScopes(t *testing.T, sub string, scopes []string) string {
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
		if err := token.Set("scopes", scopes); err != nil {
			t.Fatal(err)
		}
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), i.signingKey))
	if err != nil {
		t.Fatal(err)
	}

	return string(signed)
}
