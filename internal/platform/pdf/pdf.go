// Package pdf renders HTML documents into PDF through a Gotenberg service.
//
// The renderer is optional: when no service URL is configured the client stays
// disabled and callers degrade gracefully. Browser printing stays outside this
// package; only server-side documents go through Gotenberg.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/i18n"
)

const (
	convertRoute     = "/forms/chromium/convert/html"
	defaultTimeout   = 30 * time.Second
	maxDocumentBytes = 32 << 20
	errorSnippetSize = 200
)

var (
	// ErrUnavailable identifies a missing renderer configuration.
	ErrUnavailable = errors.New(i18n.T("pdf.unavailable"))
	// ErrConversion identifies a renderer that refused or failed the request.
	ErrConversion = errors.New(i18n.T("pdf.conversion_failed"))
)

// Config contains the renderer service location and request timeout.
type Config struct {
	URL     string
	Timeout time.Duration
}

// Options describes the page geometry sent to the renderer. Lengths are inches.
type Options struct {
	PaperWidth   float64
	PaperHeight  float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
	Landscape    bool
}

// Client posts HTML documents to a Gotenberg Chromium route.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New constructs a renderer client. An empty URL produces a disabled client.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.URL), "/"),
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Enabled reports whether the renderer is configured.
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

// ConvertHTML renders one HTML document into PDF bytes.
func (c *Client) ConvertHTML(
	ctx context.Context,
	document []byte,
	options Options,
) ([]byte, error) {
	assert.Assert(len(document) > 0, "document body is required")
	if !c.Enabled() {
		return nil, ErrUnavailable
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="files"; filename="index.html"`)
	header.Set("Content-Type", "text/html; charset=utf-8")
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversion, err)
	}
	if _, err := part.Write(document); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversion, err)
	}

	fields := map[string]string{"printBackground": "true"}
	if options.PaperWidth > 0 {
		fields["paperWidth"] = formatLength(options.PaperWidth)
	}
	if options.PaperHeight > 0 {
		fields["paperHeight"] = formatLength(options.PaperHeight)
	}
	if options.MarginTop > 0 {
		fields["marginTop"] = formatLength(options.MarginTop)
	}
	if options.MarginBottom > 0 {
		fields["marginBottom"] = formatLength(options.MarginBottom)
	}
	if options.MarginLeft > 0 {
		fields["marginLeft"] = formatLength(options.MarginLeft)
	}
	if options.MarginRight > 0 {
		fields["marginRight"] = formatLength(options.MarginRight)
	}
	if options.Landscape {
		fields["landscape"] = "true"
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrConversion, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversion, err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+convertRoute,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversion, err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversion, err)
	}
	defer func() { _ = response.Body.Close() }()

	content, err := io.ReadAll(io.LimitReader(response.Body, maxDocumentBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversion, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: layanan menolak permintaan (HTTP %d) %s",
			ErrConversion,
			response.StatusCode,
			snippet(content),
		)
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrConversion, i18n.T("pdf.empty_file"))
	}
	return content, nil
}

func formatLength(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func snippet(content []byte) string {
	text := strings.TrimSpace(string(content))
	if len(text) > errorSnippetSize {
		text = text[:errorSnippetSize]
	}
	return text
}
