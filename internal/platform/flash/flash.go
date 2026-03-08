package flash

import (
	"context"
	"encoding/gob"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
	appSession "github.com/luminor-project/luminor-core-go-playground/internal/platform/session"
)

type contextKey string

const flashContextKey contextKey = "flash_messages"

type Type string

const (
	TypeSuccess Type = "success"
	TypeError   Type = "error"
	TypeInfo    Type = "info"
	TypeWarning Type = "warning"
)

type Message struct {
	Type    Type
	Content string
}

// Set adds a flash message to the session.
func Set(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, flashType Type, content string) {
	sess, err := store.Get(r, appSession.SessionName)
	if err != nil {
		slog.Warn("flash: failed to get session", "error", err)
		return
	}
	sess.AddFlash(Message{Type: flashType, Content: content})
	if err := sess.Save(r, w); err != nil {
		slog.Warn("flash: failed to save session", "error", err)
	}
}

// SetKey translates a key in the active locale and stores it as flash content.
func SetKey(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, flashType Type, key string, args ...any) {
	Set(w, r, store, flashType, i18n.T(r.Context(), key, args...))
}

// Middleware loads flash messages from the session into the request context.
func Middleware(store *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := store.Get(r, appSession.SessionName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			flashes := sess.Flashes()
			if len(flashes) > 0 {
				if err := sess.Save(r, w); err != nil {
					slog.Warn("flash: failed to save session after reading", "error", err)
				}
			}

			var messages []Message
			for _, f := range flashes {
				if msg, ok := f.(Message); ok {
					messages = append(messages, msg)
				}
			}

			ctx := context.WithValue(r.Context(), flashContextKey, messages)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext returns flash messages from the request context.
func FromContext(ctx context.Context) []Message {
	messages, _ := ctx.Value(flashContextKey).([]Message)
	return messages
}

func init() {
	// Register the Message type with gob for session serialization
	gob.Register(Message{})
}
