package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"shortlink/internal/db"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type meResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "hash failed", http.StatusInternalServerError)
		return
	}
	_, err = db.Pool.Exec(r.Context(),
		"INSERT INTO users (username, password_hash) VALUES ($1, $2)",
		req.Username, string(hash),
	)
	if err != nil {
		http.Error(w, "username taken", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var userID int
	var storedhash string
	err := db.Pool.QueryRow(r.Context(),
		"SELECT id, password_hash FROM users WHERE username = $1", req.Username,
	).Scan(&userID, &storedhash)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedhash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	sid := newSessionID()
	setSession(sid, userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		deleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}
func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCookie(r)
	if !ok {
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return
	}
	var username string
	err := db.Pool.QueryRow(r.Context(),
		"SELECT username FROM users WHERE id = $1", userID).Scan(&username)
	if err != nil {
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meResponse{ID: userID, Username: username})
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	uid, ok := ctx.Value(userIDKey{}).(int)
	return uid, ok
}

type userIDKey struct{}

func WithUserID(ctx context.Context, uid int) context.Context {
	return context.WithValue(ctx, userIDKey{}, uid)
}
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := userIDFromCookie(r)
		if !ok {
			http.Error(w, "not logged in", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), uid)))
	})
}
