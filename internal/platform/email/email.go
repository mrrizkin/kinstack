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

type Sender interface {
	Send(context.Context, Message) error
	Render(name string, data any) (string, error)
}

// UnavailableSender makes a missing provider explicit instead of silently
// dropping security-sensitive messages.
type UnavailableSender struct{ renderer }

func (UnavailableSender) Send(context.Context, Message) error { return ErrUnavailable }

// MemorySender deduplicates by idempotency key; failed sends are not recorded.
type MemorySender struct {
	renderer
	mu       sync.Mutex
	messages []Message
	failure  error
}

func NewMemorySender() *MemorySender { return &MemorySender{} }

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

func (s *MemorySender) Messages() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.messages...)
}

func (s *MemorySender) SetFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failure = err
}

// SMTPSender keeps provider credentials at the composition boundary.
type SMTPSender struct {
	renderer
	config config.EmailConfig
}

// New returns an unavailable sender when SMTP is unset in development; production
// configuration rejects that state at startup.
func New(cfg config.EmailConfig) (Sender, error) {
	if strings.TrimSpace(cfg.Host) == "" {
		return UnavailableSender{}, nil
	}
	if err := validateSMTPConfig(cfg); err != nil {
		return nil, err
	}
	return &SMTPSender{config: cfg}, nil
}

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
	client, err := s.dial(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := s.secure(client); err != nil {
		return err
	}
	return deliverMessage(client, message, addressOnly(s.config.From), addressOnly(message.To))
}

// dial gives the client the caller's deadline, so a stalled relay cannot hold the
// request open past its context.
func (s *SMTPSender) dial(ctx context.Context) (*smtp.Client, error) {
	conn, err := (&net.Dialer{}).DialContext(
		ctx,
		"tcp",
		net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port)),
	)
	if err != nil {
		return nil, fmt.Errorf("dial email relay: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open email relay: %w", err)
	}
	return client, nil
}

func (s *SMTPSender) secure(client *smtp.Client) error {
	if s.config.StartTLS {
		if err := client.StartTLS(
			&tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12},
		); err != nil {
			return fmt.Errorf("start email relay TLS: %w", err)
		}
	}
	if s.config.Username == "" {
		return nil
	}
	if err := client.Auth(
		smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host),
	); err != nil {
		return fmt.Errorf("authenticate email relay: %w", err)
	}
	return nil
}

func deliverMessage(client *smtp.Client, message Message, from, to string) error {
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
	raw, err := formatMessage(message, from, to)
	if err != nil {
		_ = writer.Close()
		return err
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write email body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close email body: %w", err)
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

func formatMessage(message Message, from, to string) ([]byte, error) {
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
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, fmt.Errorf("email multipart boundary: %w", err)
	}
	textHeader := make(map[string][]string)
	textHeader["Content-Type"] = []string{"text/plain; charset=UTF-8"}
	textPart, err := writer.CreatePart(header(textHeader))
	if err != nil {
		return nil, fmt.Errorf("email text part: %w", err)
	}
	if _, err := textPart.Write([]byte(message.TextBody)); err != nil {
		return nil, fmt.Errorf("email text body: %w", err)
	}
	if message.HTMLBody != "" {
		htmlHeader := make(map[string][]string)
		htmlHeader["Content-Type"] = []string{"text/html; charset=UTF-8"}
		htmlPart, err := writer.CreatePart(header(htmlHeader))
		if err != nil {
			return nil, fmt.Errorf("email html part: %w", err)
		}
		if _, err := htmlPart.Write([]byte(message.HTMLBody)); err != nil {
			return nil, fmt.Errorf("email html body: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("email multipart close: %w", err)
	}
	return body.Bytes(), nil
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
