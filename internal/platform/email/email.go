// Package email defines the provider-neutral outbound email boundary.
package email

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"sync"

	"kinstack/config"
)

// ErrUnavailable identifies email delivery that is not configured.
var ErrUnavailable = errors.New("email delivery is unavailable")

// Message is the complete outbound email contract. IdempotencyKey is stable
// across retries of the same mutation and lets providers with durable
// deduplication avoid duplicate delivery.
type Message struct {
	To             string
	From           string
	Subject        string
	TextBody       string
	HTMLBody       string
	IdempotencyKey string
}

// Sender is the only dependency application services need for email delivery.
type Sender interface {
	Send(context.Context, Message) error
	Render(name string, data any) (string, error)
}

// UnavailableSender makes a missing provider explicit instead of silently
// dropping security-sensitive messages.
type UnavailableSender struct{ renderer }

// Send always fails with ErrUnavailable.
func (UnavailableSender) Send(context.Context, Message) error { return ErrUnavailable }

// MemorySender is a deterministic fake for unit and HTTP tests. Successful
// messages are deduplicated by idempotency key, while failed sends are not.
type MemorySender struct {
	renderer
	mu       sync.Mutex
	messages []Message
	failure  error
}

// NewMemorySender constructs an empty in-memory sender.
func NewMemorySender() *MemorySender { return &MemorySender{} }

// Send records a message after validating it, deduplicating by idempotency key.
func (s *MemorySender) Send(_ context.Context, message Message) error {
	if err := validateMessage(message); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sent := range s.messages {
		if sent.IdempotencyKey == message.IdempotencyKey {
			return nil
		}
	}
	if s.failure != nil {
		return s.failure
	}
	s.messages = append(s.messages, message)
	return nil
}

// Messages returns a copy of the recorded messages.
func (s *MemorySender) Messages() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.messages...)
}

// SetFailure makes subsequent sends fail with err.
func (s *MemorySender) SetFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failure = err
}

// SMTPSender is the production adapter for an SMTP relay. The application can
// keep its provider-specific credentials at the composition boundary.
type SMTPSender struct {
	renderer
	config config.EmailConfig
}

// New returns an SMTP sender when configured. Development environments may
// intentionally omit SMTP and receive an explicit unavailable error on email
// mutations; production configuration rejects that state.
func New(cfg config.EmailConfig) (Sender, error) {
	if strings.TrimSpace(cfg.Host) == "" {
		return UnavailableSender{}, nil
	}
	if err := validateSMTPConfig(cfg); err != nil {
		return nil, err
	}
	return &SMTPSender{config: cfg}, nil
}

// Send delivers one message through the configured SMTP relay.
func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	if s == nil {
		return ErrUnavailable
	}
	if err := validateMessage(message); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	from := addressOnly(s.config.From)
	to := addressOnly(message.To)
	conn, err := (&net.Dialer{}).DialContext(
		ctx,
		"tcp",
		net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port)),
	)
	if err != nil {
		return fmt.Errorf("dial email relay: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open email relay: %w", err)
	}
	defer client.Close()
	if s.config.StartTLS {
		if err := client.StartTLS(
			&tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12},
		); err != nil {
			return fmt.Errorf("start email relay TLS: %w", err)
		}
	}
	if s.config.Username != "" {
		if err := client.Auth(
			smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host),
		); err != nil {
			return fmt.Errorf("authenticate email relay: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set email sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set email recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open email body: %w", err)
	}
	if _, err := writer.Write(formatMessage(message, from, to)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write email body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close email body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("complete email delivery: %w", err)
	}
	return nil
}

func validateSMTPConfig(cfg config.EmailConfig) error {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("SMTP host and port are invalid")
	}
	if cfg.Username != "" && cfg.Password == "" || cfg.Username == "" && cfg.Password != "" {
		return fmt.Errorf("SMTP username and password must be provided together")
	}
	if _, err := mail.ParseAddress(cfg.From); err != nil || strings.ContainsAny(cfg.From, "\r\n") {
		return fmt.Errorf("SMTP from address is invalid")
	}
	return nil
}

func validateMessage(message Message) error {
	if _, err := mail.ParseAddress(message.To); err != nil ||
		strings.ContainsAny(message.To, "\r\n") {
		return fmt.Errorf("email recipient is invalid")
	}
	if _, err := mail.ParseAddress(message.From); err != nil ||
		strings.ContainsAny(message.From, "\r\n") {
		return fmt.Errorf("email sender is invalid")
	}
	if strings.TrimSpace(message.Subject) == "" || strings.ContainsAny(message.Subject, "\r\n") {
		return fmt.Errorf("email subject is invalid")
	}
	if message.TextBody == "" && message.HTMLBody == "" {
		return fmt.Errorf("email body is required")
	}
	if strings.TrimSpace(message.IdempotencyKey) == "" || len(message.IdempotencyKey) > 200 {
		return fmt.Errorf("email idempotency key is invalid")
	}
	return nil
}

func formatMessage(message Message, from, to string) []byte {
	var body bytes.Buffer
	boundary := "kinstack-" + messageID(message.IdempotencyKey)
	fmt.Fprintf(
		&body,
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%s@kinstack>\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n",
		from,
		to,
		message.Subject,
		messageID(message.IdempotencyKey),
		boundary,
	)
	writer := multipart.NewWriter(&body)
	_ = writer.SetBoundary(boundary)
	textHeader := make(map[string][]string)
	textHeader["Content-Type"] = []string{"text/plain; charset=UTF-8"}
	textPart, _ := writer.CreatePart(header(textHeader))
	_, _ = textPart.Write([]byte(message.TextBody))
	if message.HTMLBody != "" {
		htmlHeader := make(map[string][]string)
		htmlHeader["Content-Type"] = []string{"text/html; charset=UTF-8"}
		htmlPart, _ := writer.CreatePart(header(htmlHeader))
		_, _ = htmlPart.Write([]byte(message.HTMLBody))
	}
	_ = writer.Close()
	return body.Bytes()
}

func header(values map[string][]string) textproto.MIMEHeader {
	result := make(textproto.MIMEHeader)
	for key, items := range values {
		for _, item := range items {
			result.Add(key, item)
		}
	}
	return result
}

func messageID(key string) string {
	digest := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", digest[:12])
}

func addressOnly(raw string) string {
	parsed, err := mail.ParseAddress(raw)
	if err != nil {
		return raw
	}
	return parsed.Address
}

var (
	_ Sender = (*SMTPSender)(nil)
	_ Sender = (*MemorySender)(nil)
	_ Sender = UnavailableSender{}
)
