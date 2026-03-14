package email

import (
	"context"
	"testing"
)

func TestNoOpSender_SendPasswordReset(t *testing.T) {
	sender := &NoOpSender{}
	err := sender.SendPasswordReset(context.Background(), "test@example.com", "http://example.com/reset?token=abc123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
