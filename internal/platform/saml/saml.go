// Package saml implements the SAML 2.0 service-provider boundary used by the
// auth feature. It wraps crewjam/saml so provider details never cross into the
// auth service.
package saml

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	platformoauth "kinstack/internal/platform/oauth"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
)

var (
	// ErrUnavailable identifies an unreachable or misconfigured provider.
	ErrUnavailable = errors.New("saml provider is unavailable")
	// ErrInvalidConfig identifies an incomplete provider configuration.
	ErrInvalidConfig = errors.New("saml provider configuration is invalid")
	// ErrInvalidResponse identifies a rejected or invalid SAML response.
	ErrInvalidResponse = errors.New("saml response is invalid")
	// ErrUnstableSubject identifies an assertion whose subject cannot identify
	// an account across logins (missing or transient NameID).
	ErrUnstableSubject = errors.New("saml subject identifier is not persistent")
)

// Config carries the service-provider settings. Every value is required when
// SAML is enabled; config validation guarantees completeness before startup.
type Config struct {
	EntityID       string
	RootURL        string
	IDPMetadataURL string
	Certificate    string
	PrivateKey     string
	HTTPClient     *http.Client
}

// Provider is the SAML service provider consumed by auth.
type Provider struct {
	serviceProvider *saml.ServiceProvider
}

// New builds and validates the service provider, fetching IdP metadata.
func New(ctx context.Context, cfg Config) (*Provider, error) {
	root, err := url.Parse(strings.TrimRight(cfg.RootURL, "/"))
	if err != nil || root.Scheme == "" || root.Host == "" {
		return nil, fmt.Errorf("%w: invalid root url", ErrInvalidConfig)
	}
	metadataURL, err := url.Parse(cfg.IDPMetadataURL)
	if err != nil || metadataURL.Scheme == "" || metadataURL.Host == "" {
		return nil, fmt.Errorf("%w: invalid idp metadata url", ErrInvalidConfig)
	}
	certificate, err := parseCertificate(cfg.Certificate)
	if err != nil {
		return nil, err
	}
	key, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	idpMetadata, err := samlsp.FetchMetadata(ctx, client, *metadataURL)
	if err != nil {
		return nil, fmt.Errorf("%w: fetch idp metadata: %v", ErrUnavailable, err)
	}
	return &Provider{serviceProvider: &saml.ServiceProvider{
		EntityID:    cfg.EntityID,
		Key:         key,
		Certificate: certificate,
		MetadataURL: *root.ResolveReference(&url.URL{Path: "/saml/metadata"}),
		AcsURL:      *root.ResolveReference(&url.URL{Path: "/login/saml/acs"}),
		IDPMetadata: idpMetadata,
		HTTPClient:  client,
	}}, nil
}

// Metadata renders the service-provider metadata XML for IdP registration.
func (p *Provider) Metadata() ([]byte, error) {
	if p == nil || p.serviceProvider == nil {
		return nil, ErrUnavailable
	}
	return xml.MarshalIndent(p.serviceProvider.Metadata(), "", "  ")
}

// Start builds the AuthnRequest redirect URL and returns its request ID so the
// callback can validate InResponseTo.
func (p *Provider) Start(relayState string) (string, string, error) {
	if p == nil || p.serviceProvider == nil {
		return "", "", ErrUnavailable
	}
	idpURL := p.serviceProvider.GetSSOBindingLocation(saml.HTTPRedirectBinding)
	request, err := p.serviceProvider.MakeAuthenticationRequest(
		idpURL,
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		return "", "", fmt.Errorf("%w: build authn request: %v", ErrUnavailable, err)
	}
	redirect, err := request.Redirect(relayState, p.serviceProvider)
	if err != nil {
		return "", "", fmt.Errorf("%w: encode authn request: %v", ErrUnavailable, err)
	}
	return redirect.String(), request.ID, nil
}

// Complete validates the SAML response and returns the asserted identity.
func (p *Provider) Complete(
	request *http.Request,
	requestID string,
) (platformoauth.Identity, error) {
	if p == nil || p.serviceProvider == nil {
		return platformoauth.Identity{}, ErrUnavailable
	}
	ids := make([]string, 0, 1)
	if trimmed := strings.TrimSpace(requestID); trimmed != "" {
		ids = append(ids, trimmed)
	}
	assertion, err := p.serviceProvider.ParseResponse(request, ids)
	if err != nil {
		return platformoauth.Identity{}, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return identityFromAssertion(assertion)
}

// identityFromAssertion extracts the base identity fields. A subject that is
// missing or explicitly transient cannot be bound to an account: accepting it
// would create a new identity row on every login.
func identityFromAssertion(assertion *saml.Assertion) (platformoauth.Identity, error) {
	identity := platformoauth.Identity{Provider: "saml"}
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		nameID := assertion.Subject.NameID
		if isTransientNameIDFormat(nameID.Format) {
			return platformoauth.Identity{}, ErrUnstableSubject
		}
		identity.Subject = strings.TrimSpace(nameID.Value)
	}
	if identity.Subject == "" {
		return platformoauth.Identity{}, ErrUnstableSubject
	}
	for _, statement := range assertion.AttributeStatements {
		for _, attribute := range statement.Attributes {
			name := strings.ToLower(strings.TrimSpace(attribute.Name))
			if friendly := strings.ToLower(
				strings.TrimSpace(attribute.FriendlyName),
			); friendly != "" {
				name = friendly
			}
			value := ""
			if len(attribute.Values) > 0 {
				value = strings.TrimSpace(attribute.Values[0].Value)
			}
			switch {
			case identity.Email == "" && isEmailAttribute(name):
				identity.Email = value
			case identity.DisplayName == "" && isNameAttribute(name):
				identity.DisplayName = value
			}
		}
	}
	identity.Email = strings.ToLower(identity.Email)
	if identity.DisplayName == "" {
		identity.DisplayName = identity.Email
	}
	identity.EmailVerified = identity.Email != ""
	return identity, nil
}

// transientNameIDFormat is the SAML 2.0 transient NameID format. Identities in
// this format are valid only for one session.
const transientNameIDFormat = "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"

func isTransientNameIDFormat(format string) bool {
	return strings.EqualFold(strings.TrimSpace(format), transientNameIDFormat)
}

func isEmailAttribute(name string) bool {
	switch {
	case name == "email", name == "mail", name == "emailaddress":
		return true
	case strings.HasSuffix(name, "/email"), strings.HasSuffix(name, "/mail"),
		strings.HasSuffix(name, "/emailaddress"), strings.HasSuffix(name, "emailaddress"):
		return true
	}
	return false
}

func isNameAttribute(name string) bool {
	switch {
	case name == "name", name == "displayname", name == "cn", name == "fullname":
		return true
	case strings.HasSuffix(name, "/name"), strings.HasSuffix(name, "displayname"):
		return true
	}
	return false
}

func parseCertificate(raw string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(normalizePEM(raw)))
	if block == nil {
		return nil, fmt.Errorf("%w: certificate is not valid PEM", ErrInvalidConfig)
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse certificate: %v", ErrInvalidConfig, err)
	}
	return certificate, nil
}

func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(raw)))
	if block == nil {
		return nil, fmt.Errorf("%w: private key is not valid PEM", ErrInvalidConfig)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse private key: %v", ErrInvalidConfig, err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: private key must be RSA", ErrInvalidConfig)
	}
	return key, nil
}

// normalizePEM accepts certificates and keys supplied with escaped newlines.
func normalizePEM(raw string) string {
	return strings.ReplaceAll(strings.TrimSpace(raw), "\\n", "\n")
}
