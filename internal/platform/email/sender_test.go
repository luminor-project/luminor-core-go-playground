package email_test

import (
	"context"
	"errors"
	"testing"
)

// MockSender is a test double that captures sent emails without actually sending them.
type MockSender struct {
	emails []SentEmail
	err    error
}

// SentEmail represents a captured email for testing verification.
type SentEmail struct {
	To       string
	Subject  string
	BodyText string
	BodyHTML string
}

// NewMockSender creates a new MockSender.
func NewMockSender() *MockSender {
	return &MockSender{
		emails: make([]SentEmail, 0),
	}
}

// Send captures the email to the mock's internal storage.
func (m *MockSender) Send(ctx context.Context, to, subject, bodyText, bodyHTML string) error {
	if m.err != nil {
		return m.err
	}
	m.emails = append(m.emails, SentEmail{
		To:       to,
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
	})
	return nil
}

// SetError configures the mock to return an error on the next Send call.
func (m *MockSender) SetError(err error) {
	m.err = err
}

// GetEmails returns all captured emails.
func (m *MockSender) GetEmails() []SentEmail {
	return m.emails
}

// GetLastEmail returns the most recently captured email.
func (m *MockSender) GetLastEmail() (SentEmail, bool) {
	if len(m.emails) == 0 {
		return SentEmail{}, false
	}
	return m.emails[len(m.emails)-1], true
}

// Clear resets the captured emails.
func (m *MockSender) Clear() {
	m.emails = make([]SentEmail, 0)
	m.err = nil
}

// Test MockSender

func TestMockSender_CapturesEmail(t *testing.T) {
	t.Parallel()

	mock := NewMockSender()
	ctx := context.Background()

	err := mock.Send(ctx, "user@example.com", "Test Subject", "Plain text body", "<p>HTML body</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	emails := mock.GetEmails()
	if len(emails) != 1 {
		t.Fatalf("expected 1 email, got %d", len(emails))
	}

	email := emails[0]
	if email.To != "user@example.com" {
		t.Errorf("expected To 'user@example.com', got %q", email.To)
	}
	if email.Subject != "Test Subject" {
		t.Errorf("expected Subject 'Test Subject', got %q", email.Subject)
	}
	if email.BodyText != "Plain text body" {
		t.Errorf("expected BodyText 'Plain text body', got %q", email.BodyText)
	}
	if email.BodyHTML != "<p>HTML body</p>" {
		t.Errorf("expected BodyHTML '<p>HTML body</p>', got %q", email.BodyHTML)
	}
}

func TestMockSender_CapturesMultipleEmails(t *testing.T) {
	t.Parallel()

	mock := NewMockSender()
	ctx := context.Background()

	// Send multiple emails
	for i := 0; i < 5; i++ {
		err := mock.Send(ctx, "user@example.com", "Subject", "Body", "HTML")
		if err != nil {
			t.Fatalf("unexpected error on email %d: %v", i+1, err)
		}
	}

	emails := mock.GetEmails()
	if len(emails) != 5 {
		t.Fatalf("expected 5 emails, got %d", len(emails))
	}
}

func TestMockSender_GetLastEmail(t *testing.T) {
	t.Parallel()

	mock := NewMockSender()
	ctx := context.Background()

	// No emails yet
	_, ok := mock.GetLastEmail()
	if ok {
		t.Error("expected GetLastEmail to return false when no emails")
	}

	// Send emails
	mock.Send(ctx, "user1@example.com", "Subject 1", "Body 1", "HTML 1")
	mock.Send(ctx, "user2@example.com", "Subject 2", "Body 2", "HTML 2")
	mock.Send(ctx, "user3@example.com", "Subject 3", "Body 3", "HTML 3")

	// Should get the last one
	last, ok := mock.GetLastEmail()
	if !ok {
		t.Fatal("expected GetLastEmail to return true")
	}
	if last.To != "user3@example.com" {
		t.Errorf("expected To 'user3@example.com', got %q", last.To)
	}
	if last.Subject != "Subject 3" {
		t.Errorf("expected Subject 'Subject 3', got %q", last.Subject)
	}
}

func TestMockSender_Clear(t *testing.T) {
	t.Parallel()

	mock := NewMockSender()
	ctx := context.Background()

	// Send some emails
	mock.Send(ctx, "user@example.com", "Subject", "Body", "HTML")
	mock.Send(ctx, "user@example.com", "Subject 2", "Body 2", "HTML 2")

	if len(mock.GetEmails()) != 2 {
		t.Fatal("expected 2 emails before clear")
	}

	// Clear
	mock.Clear()

	if len(mock.GetEmails()) != 0 {
		t.Errorf("expected 0 emails after clear, got %d", len(mock.GetEmails()))
	}

	_, ok := mock.GetLastEmail()
	if ok {
		t.Error("expected GetLastEmail to return false after clear")
	}
}

func TestMockSender_SetError(t *testing.T) {
	t.Parallel()

	mock := NewMockSender()
	ctx := context.Background()

	// Set error
	expectedErr := errors.New("simulated email failure")
	mock.SetError(expectedErr)

	// Send should fail
	err := mock.Send(ctx, "user@example.com", "Subject", "Body", "HTML")
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// No email should be captured
	if len(mock.GetEmails()) != 0 {
		t.Errorf("expected 0 emails when error is set, got %d", len(mock.GetEmails()))
	}
}

// MockSenderWithHooks extends MockSender to support pre/post send hooks for advanced testing.
type MockSenderWithHooks struct {
	MockSender
	preSendHook  func(to, subject, bodyText, bodyHTML string) error
	postSendHook func(to, subject, bodyText, bodyHTML string)
}

// NewMockSenderWithHooks creates a new MockSenderWithHooks.
func NewMockSenderWithHooks() *MockSenderWithHooks {
	return &MockSenderWithHooks{
		MockSender:   *NewMockSender(),
		preSendHook:  nil,
		postSendHook: nil,
	}
}

// SetPreSendHook sets a hook to be called before sending.
func (m *MockSenderWithHooks) SetPreSendHook(hook func(to, subject, bodyText, bodyHTML string) error) {
	m.preSendHook = hook
}

// SetPostSendHook sets a hook to be called after sending.
func (m *MockSenderWithHooks) SetPostSendHook(hook func(to, subject, bodyText, bodyHTML string)) {
	m.postSendHook = hook
}

// Send implements the Sender interface with hooks.
func (m *MockSenderWithHooks) Send(ctx context.Context, to, subject, bodyText, bodyHTML string) error {
	if m.preSendHook != nil {
		if err := m.preSendHook(to, subject, bodyText, bodyHTML); err != nil {
			return err
		}
	}

	// Call parent Send
	err := m.MockSender.Send(ctx, to, subject, bodyText, bodyHTML)

	if m.postSendHook != nil {
		m.postSendHook(to, subject, bodyText, bodyHTML)
	}

	return err
}

func TestMockSenderWithHooks_PreSendHook(t *testing.T) {
	t.Parallel()

	mock := NewMockSenderWithHooks()
	ctx := context.Background()

	var hookCalled bool
	mock.SetPreSendHook(func(to, subject, bodyText, bodyHTML string) error {
		hookCalled = true
		if to == "block@example.com" {
			return errors.New("blocked by pre-send hook")
		}
		return nil
	})

	// Normal send should work
	err := mock.Send(ctx, "user@example.com", "Subject", "Body", "HTML")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hookCalled {
		t.Error("pre-send hook should have been called")
	}

	// Send to blocked address should fail
	mock.Clear()
	hookCalled = false
	err = mock.Send(ctx, "block@example.com", "Subject", "Body", "HTML")
	if err == nil {
		t.Error("expected error from pre-send hook")
	}
	if !hookCalled {
		t.Error("pre-send hook should have been called for blocked address")
	}
	if len(mock.GetEmails()) != 0 {
		t.Error("email should not be captured when pre-send hook fails")
	}
}

func TestMockSenderWithHooks_PostSendHook(t *testing.T) {
	t.Parallel()

	mock := NewMockSenderWithHooks()
	ctx := context.Background()

	var capturedTo string
	mock.SetPostSendHook(func(to, subject, bodyText, bodyHTML string) {
		capturedTo = to
	})

	err := mock.Send(ctx, "user@example.com", "Subject", "Body", "HTML")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if capturedTo != "user@example.com" {
		t.Errorf("post-send hook should capture To address, got %q", capturedTo)
	}
}
