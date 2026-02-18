// SPDX-FileCopyrightText: 2026 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type session struct {
	data      map[string]string
	expiresAt time.Time
}

// Session store (maps sessionID -> data)
var (
	sessions    = make(map[string]*session)
	sessionTTL  = 24 * time.Hour
	cleanupTick = 1 * time.Hour
	mtx         sync.Mutex
)

func init() {
	go cleanupSessions()
}

func cleanupSessions() {
	ticker := time.NewTicker(cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		mtx.Lock()
		for id, s := range sessions {
			if time.Now().After(s.expiresAt) {
				delete(sessions, id)
			}
		}
		mtx.Unlock()
	}
}

// Middleware to manage sessions
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing session cookie
		cookie, err := r.Cookie("session_id")
		var sessionID string

		if errors.Is(err, http.ErrNoCookie) {
			// No session, create a new one
			sessionID = uuid.New().String()
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    sessionID,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			})

			mtx.Lock()
			sessions[sessionID] = &session{
				data:      make(map[string]string),
				expiresAt: time.Now().Add(sessionTTL),
			}
			mtx.Unlock()
		} else {
			sessionID = cookie.Value
			mtx.Lock()
			if s, ok := sessions[sessionID]; ok {
				s.expiresAt = time.Now().Add(sessionTTL)
			}
			mtx.Unlock()
		}

		r.Header.Set("X-Session-ID", sessionID)
		next.ServeHTTP(w, r)
	})
}
