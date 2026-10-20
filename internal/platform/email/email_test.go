package email

import (
	"context"
	"errors"
	"testing"

	"kinstack/config"
)

func TestMemorySenderDeduplicatesSuccessfulMessages(t *testing.T) {
	sender := NewMemorySender()
	message := Message{
		To:             "person@example.test",
		From:           "KinStack <no-reply@example.test>",
		Subject:        "Invitation",
		TextBody:       "Join us",
		IdempotencyKey: "invite-1",
	}
	if err := sender.Send(context.Background(), message); err != nil {
		t.Fatalf("send first message: %v", err)
	}
	if err := sender.Send(context.Background(), message); err != nil {
		t.Fatalf("send replay message: %v", err)
	}
	if len(sender.Messages()) != 1 {
		t.Fatalf("expected one message after replay, got %d", len(sender.Messages()))
	}

	sender.SetFailure(errors.New("relay down"))
	if err := sender.Send(
		context.Background(),
		Message{
			To:             "other@example.test",
			From:           message.From,
			Subject:        message.Subject,
			TextBody:       message.TextBody,
			IdempotencyKey: "invite-2",
		},
	); err == nil {
		t.Fatal("expected failed delivery")
	}
}

func TestConfiguredSenderRequiresSafeSMTPConfiguration(t *testing.T) {
	sender, err := New(config.EmailConfig{})
	if err != nil {
		t.Fatalf("empty development SMTP should be unavailable, not invalid: %v", err)
	}
	if !errors.Is(sender.Send(context.Background(), Message{}), ErrUnavailable) {
		t.Fatal("expected explicit unavailable sender")
	}
	if _, err := New(
		config.EmailConfig{
			Host:     "smtp.example.test",
			Port:     587,
			From:     "no-reply@example.test",
			StartTLS: true,
			Username: "user",
		},
	); err == nil {
		t.Fatal("expected partial SMTP credentials to be rejected")
	}
}
