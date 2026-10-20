package pdf

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConvertHTMLPostsMultipartDocument(t *testing.T) {
	var receivedHTML string
	var receivedFields map[string]string
	var receivedFilename string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != convertRoute {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Errorf("read files part: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()
		content, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("read files body: %v", err)
		}
		receivedHTML = string(content)
		receivedFilename = header.Filename

		receivedFields = map[string]string{}
		for key, values := range r.MultipartForm.Value {
			receivedFields[key] = values[0]
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 fake"))
	}))
	defer server.Close()

	client := New(Config{URL: server.URL + "/"})
	if !client.Enabled() {
		t.Fatal("expected configured client to be enabled")
	}

	content, err := client.ConvertHTML(
		context.Background(),
		[]byte("<html><body>Laporan</body></html>"),
		Options{PaperWidth: 8.27, PaperHeight: 11.69, MarginTop: 0.4, Landscape: false},
	)
	if err != nil {
		t.Fatalf("convert html: %v", err)
	}
	if string(content) != "%PDF-1.4 fake" {
		t.Fatalf("unexpected pdf payload %q", content)
	}
	if receivedFilename != "index.html" {
		t.Fatalf("unexpected part filename %q", receivedFilename)
	}
	if !strings.Contains(receivedHTML, "Laporan") {
		t.Fatalf("unexpected document body %q", receivedHTML)
	}
	if receivedFields["paperWidth"] != "8.27" || receivedFields["paperHeight"] != "11.69" {
		t.Fatalf("unexpected paper fields %#v", receivedFields)
	}
	if receivedFields["marginTop"] != "0.4" {
		t.Fatalf("unexpected margin fields %#v", receivedFields)
	}
	if receivedFields["printBackground"] != "true" {
		t.Fatalf("expected printBackground, got %#v", receivedFields)
	}
	if _, ok := receivedFields["landscape"]; ok {
		t.Fatalf("landscape must not be sent for portrait documents: %#v", receivedFields)
	}
}

func TestConvertHTMLWithoutConfigurationIsUnavailable(t *testing.T) {
	client := New(Config{})
	if client.Enabled() {
		t.Fatal("expected client without url to stay disabled")
	}
	if _, err := client.ConvertHTML(
		context.Background(),
		[]byte("<html></html>"),
		Options{},
	); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestConvertHTMLReportsUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("chromium missing"))
	}))
	defer server.Close()

	client := New(Config{URL: server.URL})
	_, err := client.ConvertHTML(context.Background(), []byte("<html></html>"), Options{})
	if !errors.Is(err, ErrConversion) {
		t.Fatalf("expected conversion error, got %v", err)
	}
	if !strings.Contains(err.Error(), "chromium missing") {
		t.Fatalf("expected upstream detail, got %v", err)
	}
}
