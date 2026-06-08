package link

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"shortlink/internal/users"
)

func TestHandlers_Shorten_HappyPath(t *testing.T) {
	cleanDB(t)
	userID := insertTestUser(t, "alice")

	store := NewStore(testPool)
	handlers := NewHandlers(store)

	body := strings.NewReader(`{"url": "https://example.com"}`)
	r := httptest.NewRequest("POST", "/api/shorten", body)
	r = r.WithContext(users.WithUserID(r.Context(), userID))
	w := httptest.NewRecorder()

	handlers.Shorten(w, r)
	if w.Code != 200 {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var resp ShortenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code == "" {
		t.Error("response code should not be empty")
	}
	var url string
	err := testPool.QueryRow(context.Background(),
		"SELECT url FROM links WHERE code = $1", resp.Code).Scan(&url)
	if err != nil {
		t.Fatalf("verify in DB: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("url in DB: got %q, want %q", url, "https://example.com")
	}
}
