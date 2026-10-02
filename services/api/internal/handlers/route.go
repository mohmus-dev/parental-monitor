package handlers

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		h.handleHealth(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
		h.handleParentLogin(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/logout":
		h.handleLogout(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/parents/register":
		h.handleCreateParent(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/parents/me":
		h.handleCurrentParent(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/parents/me":
		h.handleDeleteParent(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/parents/me/children":
		h.handleListParentChildren(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/parents/me/children":
		h.handleCreateChild(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/parents/me/children":
		h.handleDeleteParentChildren(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "logged out successfully"})
}
