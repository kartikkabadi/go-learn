package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kartikkabadi/go-learn/internal/web/cookies"
	"github.com/kartikkabadi/go-learn/internal/web/middleware"
	"github.com/kartikkabadi/go-learn/internal/web/views"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "session"
	sessionTTL        = 30 * 24 * time.Hour // 30 days
	maxEmailLen       = 254
	maxPasswordLen    = 72 // bcrypt hard limit
	minPasswordLen    = 8
)

type signupPage struct {
	views.PageMeta
	Email string
	Error string
}

type loginPage struct {
	views.PageMeta
	Email string
	Next  string
	Error string
}

// Signup renders the signup form (GET) or creates a new user (POST).
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	if u := middleware.UserFromContext(r); u != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if r.Method == http.MethodGet {
		h.Views.Render(w, "signup.html", signupPage{
			PageMeta: views.PageMeta{Title: "Sign up — go-learn", Description: "Create a free go-learn account to track your Go progress."},
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if email == "" || password == "" {
		h.renderSignupError(w, r, email, "Email and password are required.")
		return
	}
	if len(email) > maxEmailLen || !strings.Contains(email, "@") {
		h.renderSignupError(w, r, email, "Enter a valid email address.")
		return
	}
	if len(password) < minPasswordLen {
		h.renderSignupError(w, r, email, "Password must be at least 8 characters.")
		return
	}
	if len(password) > maxPasswordLen {
		h.renderSignupError(w, r, email, "Password must be 72 characters or fewer.")
		return
	}
	if password != passwordConfirm {
		h.renderSignupError(w, r, email, "Passwords do not match.")
		return
	}

	existing, err := h.Store.GetUserByEmail(email)
	if err != nil {
		slog.Error("signup lookup", "email", email, "error", err)
		h.renderSignupError(w, r, email, "Something went wrong. Try again.")
		return
	}
	if existing != nil {
		h.renderSignupError(w, r, email, "An account with that email already exists. Log in instead.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("bcrypt hash", "error", err)
		h.renderSignupError(w, r, email, "Something went wrong. Try again.")
		return
	}

	user, err := h.Store.CreateUser(email, string(hash))
	if err != nil {
		slog.Error("create user", "email", email, "error", err)
		h.renderSignupError(w, r, email, "Something went wrong. Try again.")
		return
	}

	h.startSession(w, r, user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Login renders the login form (GET) or authenticates a user (POST).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if u := middleware.UserFromContext(r); u != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	next := strings.TrimSpace(r.URL.Query().Get("next"))
	// Restrict next to same-site relative paths to prevent open redirect.
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		next = "/"
	}
	if r.Method == http.MethodGet {
		h.Views.Render(w, "login.html", loginPage{
			PageMeta: views.PageMeta{Title: "Log in — go-learn", Description: "Log in to your go-learn account."},
			Next:     next,
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.renderLoginError(w, r, email, next, "Email and password are required.")
		return
	}

	user, err := h.Store.GetUserByEmail(email)
	if err != nil {
		slog.Error("login lookup", "email", email, "error", err)
		h.renderLoginError(w, r, email, next, "Something went wrong. Try again.")
		return
	}
	if user == nil {
		h.renderLoginError(w, r, email, next, "No account found with that email. Sign up first.")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		h.renderLoginError(w, r, email, next, "Incorrect password.")
		return
	}

	h.startSession(w, r, user.ID)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// Logout deletes the session and clears the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		if err := h.Store.DeleteSession(c.Value); err != nil {
			slog.Error("delete session", "error", err)
		}
	}
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// startSession creates a session row and sets the signed cookie.
func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, userID string) {
	expiresAt := time.Now().Add(sessionTTL).UTC().Format(time.RFC3339)
	sess, err := h.Store.CreateSession(userID, expiresAt)
	if err != nil {
		slog.Error("create session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    cookies.Sign(sess.Token, h.CookieKey),
		Path:     "/",
		Expires:  time.Now().Add(sessionTTL),
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) renderSignupError(w http.ResponseWriter, r *http.Request, email, msg string) {
	h.Views.Render(w, "signup.html", signupPage{
		PageMeta: views.PageMeta{Title: "Sign up — go-learn"},
		Email:    email,
		Error:    msg,
	})
}

func (h *Handler) renderLoginError(w http.ResponseWriter, r *http.Request, email, next, msg string) {
	h.Views.Render(w, "login.html", loginPage{
		PageMeta: views.PageMeta{Title: "Log in — go-learn"},
		Email:    email,
		Next:     next,
		Error:    msg,
	})
}
