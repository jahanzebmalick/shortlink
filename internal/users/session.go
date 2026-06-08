package users

import (
	"net/http"
	"sync"
)

var (
	sessionsMu sync.Mutex
	sessions   = map[string]int{}
)

func setSession(sid string, userID int) {
	sessionsMu.Lock()
	sessions[sid] = userID
	sessionsMu.Unlock()
}
func deleteSession(sid string) {
	sessionsMu.Lock()
	delete(sessions, sid)
	sessionsMu.Unlock()
}
func userIDFromCookie(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0, false
	}
	sessionsMu.Lock()
	userID, ok := sessions[cookie.Value]
	sessionsMu.Unlock()
	return userID, ok
}
