package saml

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crewjam/saml"
)

func TestProviderMetadataAndStart(t *testing.T) {
	certificate, privateKey := testKeyPair(t)
	server := idpMetadataServer(t)
	defer server.Close()

	provider, err := New(context.Background(), Config{
		EntityID:       "https://sp.example.test",
		RootURL:        "https://sp.example.test",
		IDPMetadataURL: server.URL,
		Certificate:    certificate,
		PrivateKey:     privateKey,
		HTTPClient:     server.Client(),
	})
	if err != nil {
		t.Fatalf("build saml provider: %v", err)
	}

	metadata, err := provider.Metadata()
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if !bytes.Contains(metadata, []byte("https://sp.example.test")) {
		t.Fatalf("metadata missing entity id: %s", metadata)
	}

	redirect, requestID, err := provider.Start("relay-1")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !strings.Contains(redirect, "SAMLRequest=") ||
		!strings.Contains(redirect, "RelayState=relay-1") {
		t.Fatalf("unexpected redirect: %s", redirect)
	}
	if strings.TrimSpace(requestID) == "" {
		t.Fatalf("expected an authn request id")
	}
}

func TestNewRejectsInvalidCertificate(t *testing.T) {
	server := idpMetadataServer(t)
	defer server.Close()
	_, err := New(context.Background(), Config{
		EntityID:       "https://sp.example.test",
		RootURL:        "https://sp.example.test",
		IDPMetadataURL: server.URL,
		Certificate:    "not-a-certificate",
		PrivateKey:     "not-a-key",
		HTTPClient:     server.Client(),
	})
	if err == nil {
		t.Fatalf("expected invalid certificate error")
	}
}

func testKeyPair(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "kinstack-sp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	privateKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return string(certificate), string(privateKey)
}

func idpMetadataServer(t *testing.T) *httptest.Server {
	t.Helper()
	entity := saml.EntityDescriptor{
		EntityID: "https://idp.example.test/metadata",
		IDPSSODescriptors: []saml.IDPSSODescriptor{{
			SSODescriptor: saml.SSODescriptor{
				RoleDescriptor: saml.RoleDescriptor{
					ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
				},
			},
			SingleSignOnServices: []saml.Endpoint{{
				Binding:  saml.HTTPRedirectBinding,
				Location: "https://idp.example.test/sso",
			}},
		}},
	}
	payload, err := xml.Marshal(entity)
	if err != nil {
		t.Fatalf("marshal idp metadata: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/samlmetadata+xml")
		_, _ = w.Write(payload)
	}))
}

func TestIdentityFromAssertionRejectsUnstableSubjects(t *testing.T) {
	transient := &saml.Assertion{Subject: &saml.Subject{NameID: &saml.NameID{
		Value:  "session-only",
		Format: transientNameIDFormat,
	}}}
	if _, err := identityFromAssertion(transient); !errors.Is(err, ErrUnstableSubject) {
		t.Fatalf("expected ErrUnstableSubject for a transient NameID, got %v", err)
	}

	missing := &saml.Assertion{}
	if _, err := identityFromAssertion(missing); !errors.Is(err, ErrUnstableSubject) {
		t.Fatalf("expected ErrUnstableSubject for a missing NameID, got %v", err)
	}

	persistent := &saml.Assertion{Subject: &saml.Subject{NameID: &saml.NameID{
		Value:  "persistent-1",
		Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
	}}}
	identity, err := identityFromAssertion(persistent)
	if err != nil {
		t.Fatalf("persistent NameID: %v", err)
	}
	if identity.Provider != "saml" || identity.Subject != "persistent-1" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}
