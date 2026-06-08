package link

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"shortlink/internal/users"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type Handlers struct {
	store *Store
}

func NewHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}

type ShortenRequest struct {
	URL string `json:"url"`
}
type ShortenResponse struct {
	Code string `json:"code"`
}

func generateCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)

}

func (h *Handlers) Shorten(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url required", http.StatusBadRequest)
		return
	}
	uuid, _ := users.UserIDFromContext(r.Context())
	code := generateCode()

	if err := h.store.Create(r.Context(), code, req.URL, uuid); err != nil {
		http.Error(w, "insert failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ShortenResponse{Code: code})
}

func (h *Handlers) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	linkID, url, err := h.store.FindByCode(r.Context(), code)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}
	go func(linkID int, ip, ua string) {
		_ = h.store.RecordClick(context.Background(),
			linkID, ip, ua)
	}(linkID, r.RemoteAddr, r.UserAgent())
	http.Redirect(w, r, url, http.StatusFound)
}
func (h *Handlers) ListMine(w http.ResponseWriter, r *http.Request) {
	uid, _ := users.UserIDFromContext(r.Context())

	links, err := h.store.ListByUser(r.Context(), uid)
	if err != nil {
		http.Error(w, "query faield", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(links)
}
