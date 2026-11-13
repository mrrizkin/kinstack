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
	ErrUnavailable     = errors.New("saml provider is unavailable")
	ErrInvalidConfig   = errors.New("saml provider configuration is invalid")
	ErrInvalidResponse = errors.New("saml response is invalid")
	// ErrUnstableSubject means a missing or transient NameID.
	ErrUnstableSubject = errors.New("saml subject identifier is not persistent")
)

// Config is validated before startup whenever SAML is enabled.
type Config struct {
	EntityID       string
	RootURL        string
	IDPMetadataURL string
	Certificate    string
	PrivateKey     string
	HTTPClient     *http.Client
}

type Provider struct {
	serviceProvider *saml.ServiceProvider
}

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

func (p *Provider) Metadata() ([]byte, error) {
	if p == nil || p.serviceProvider == nil {
		return nil, ErrUnavailable
	}
	return xml.MarshalIndent(p.serviceProvider.Metadata(), "", "  ")
}

// Start returns the request ID so the callback can validate InResponseTo.
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
	assertion, err := parseResponseSafely(p.serviceProvider, request, ids)
	if err != nil {
		return platformoauth.Identity{}, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}
	return identityFromAssertion(assertion)
}

// parseResponseSafely converts the GO-2026-6909 nil-pointer panic, reachable
// from an unauthenticated response, into a rejected login. See PILLARS.md.
func parseResponseSafely(
	provider *saml.ServiceProvider,
	request *http.Request,
	ids []string,
) (assertion *saml.Assertion, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			assertion = nil
			err = fmt.Errorf("%w: parser panic", ErrInvalidResponse)
		}
	}()
	return provider.ParseResponse(request, ids)
}

func identityFromAssertion(assertion *saml.Assertion) (platformoauth.Identity, error) {
	subject, err := stableSubject(assertion.Subject)
	if err != nil {
		return platformoauth.Identity{}, err
	}
	identity := platformoauth.Identity{Provider: "saml", Subject: subject}
	fillFromAttributes(&identity, assertion.AttributeStatements)
	identity.Email = strings.ToLower(identity.Email)
	if identity.DisplayName == "" {
		identity.DisplayName = identity.Email
	}
	identity.EmailVerified = identity.Email != ""
	return identity, nil
}

// stableSubject rejects a missing or transient NameID: it would bind a new identity
// row on every login.
func stableSubject(subject *saml.Subject) (string, error) {
	if subject == nil || subject.NameID == nil {
		return "", ErrUnstableSubject
	}
	if isTransientNameIDFormat(subject.NameID.Format) {
		return "", ErrUnstableSubject
	}
	value := strings.TrimSpace(subject.NameID.Value)
	if value == "" {
		return "", ErrUnstableSubject
	}
	return value, nil
}

func fillFromAttributes(
	identity *platformoauth.Identity,
	statements []saml.AttributeStatement,
) {
	for _, statement := range statements {
		for _, attribute := range statement.Attributes {
			applyAttribute(identity, attribute)
		}
	}
}

func applyAttribute(identity *platformoauth.Identity, attribute saml.Attribute) {
	name := strings.ToLower(strings.TrimSpace(attribute.Name))
	if friendly := strings.ToLower(strings.TrimSpace(attribute.FriendlyName)); friendly != "" {
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

func normalizePEM(raw string) string {
	return strings.ReplaceAll(strings.TrimSpace(raw), "\\n", "\n")
}
