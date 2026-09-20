package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"aimili-vpngate-go/pkg/config"
)

type contextKey string

const secretPathVerifiedKey contextKey = "secret_path_verified"

const maxRequestBodyBytes = 1 << 20

type Middleware struct {
	cfg *config.Config
}

func NewMiddleware(cfg *config.Config) *Middleware {
	return &Middleware{cfg: cfg}
}

// SecurityHeaders injects defensive HTTP headers on all responses
func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "no-referrer")

		if strings.HasPrefix(r.URL.Path, "/api/") || strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) LimitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.cfg.IsUIAuthEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Allow subscription clients (Clash / Mihomo / Shadowrocket) to fetch subscription
		// without HTTP Basic Auth if accessed via verified secret path or with a valid token.
		if strings.HasPrefix(r.URL.Path, "/api/singbox/subscription") {
			if v, ok := r.Context().Value(secretPathVerifiedKey).(bool); ok && v {
				next.ServeHTTP(w, r)
				return
			}
			token := r.URL.Query().Get("token")
			settings := m.cfg.GetSettings()
			secret := strings.Trim(settings.UIPath, "/")
			subToken := strings.Trim(m.cfg.GetSubscriptionToken(), "/")
			if token != "" && ((subToken != "" && token == subToken) || (secret != "" && token == secret) || (settings.UIPassword != "" && token == settings.UIPassword)) {
				next.ServeHTTP(w, r)
				return
			}
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			qUser := r.URL.Query().Get("user")
			qPass := r.URL.Query().Get("pass")
			if qUser != "" && qPass != "" {
				user = qUser
				pass = qPass
				ok = true
			}
		}
		if !ok || !m.cfg.VerifyUICredentials(user, pass) {
			// Anti brute-force delay on invalid attempt
			time.Sleep(300 * time.Millisecond)
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted Access"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) SecretPathGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		settings := m.cfg.GetSettings()
		secret := strings.Trim(settings.UIPath, "/")
		subToken := strings.Trim(m.cfg.GetSubscriptionToken(), "/")
		reqPath := r.URL.Path

		// 1. Direct subscription fetch with valid query token bypasses secret path prefix
		token := r.URL.Query().Get("token")
		if strings.HasPrefix(reqPath, "/api/singbox/subscription") && token != "" {
			if (subToken != "" && token == subToken) || (secret != "" && token == secret) || (settings.UIPassword != "" && token == settings.UIPassword) {
				ctx := context.WithValue(r.Context(), secretPathVerifiedKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// 2. Short subscription path: /sub/<token> or /sub/<token>/<format>
		if strings.HasPrefix(reqPath, "/sub/") {
			subRemainder := strings.TrimPrefix(reqPath, "/sub/")
			parts := strings.SplitN(subRemainder, "/", 2)
			tokenPart := parts[0]
			if (subToken != "" && tokenPart == subToken) || (secret != "" && tokenPart == secret) || (settings.UIPassword != "" && tokenPart == settings.UIPassword) {
				subFormat := ""
				if len(parts) > 1 {
					subFormat = parts[1]
				}
				r2 := new(http.Request)
				*r2 = *r
				if subFormat == "clash" {
					r2.URL.Path = "/api/singbox/subscription/clash"
				} else if subFormat == "raw" || subFormat == "text" {
					r2.URL.Path = "/api/singbox/subscription/raw"
				} else {
					r2.URL.Path = "/api/singbox/subscription"
				}
				ctx := context.WithValue(r2.Context(), secretPathVerifiedKey, true)
				*r2 = *r2.WithContext(ctx)
				next.ServeHTTP(w, r2)
				return
			}
		}

		// 3. Subscription under random subToken safe path: /<sub_token>/api/singbox/subscription...
		if subToken != "" {
			subPrefix := "/" + subToken + "/"
			if strings.HasPrefix(reqPath, subPrefix) && strings.HasPrefix(reqPath[len(subPrefix)-1:], "/api/singbox/subscription") {
				r2 := new(http.Request)
				*r2 = *r
				r2.URL.Path = "/" + strings.TrimPrefix(reqPath, subPrefix)
				ctx := context.WithValue(r2.Context(), secretPathVerifiedKey, true)
				*r2 = *r2.WithContext(ctx)
				next.ServeHTTP(w, r2)
				return
			}
		}

		// If no UI secret path configured, pass through (subToken already handled above)
		if secret == "" {
			next.ServeHTTP(w, r)
			return
		}

		// 4. Direct /api/ or /metrics access without secret prefix:
		// Only allow if caller already holds valid authentication credentials.
		// If unauthenticated, stealthily return 404 to avoid revealing that this API exists to scanners.
		if strings.HasPrefix(reqPath, "/api/") || reqPath == "/api" || reqPath == "/metrics" {
			if m.cfg.IsUIAuthEnabled() {
				user, pass, ok := r.BasicAuth()
				if !ok {
					qUser := r.URL.Query().Get("user")
					qPass := r.URL.Query().Get("pass")
					if qUser != "" && qPass != "" {
						user = qUser
						pass = qPass
						ok = true
					}
				}
				if !ok || !m.cfg.VerifyUICredentials(user, pass) {
					http.NotFound(w, r)
					return
				}
			}
			next.ServeHTTP(w, r)
			return
		}

		// Exact match to /secret -> redirect to /secret/
		if reqPath == "/"+secret {
			http.Redirect(w, r, "/"+secret+"/", http.StatusFound)
			return
		}

		// Scoped under /secret/
		prefix := "/" + secret + "/"
		if strings.HasPrefix(reqPath, prefix) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/" + strings.TrimPrefix(reqPath, prefix)
			ctx := context.WithValue(r2.Context(), secretPathVerifiedKey, true)
			*r2 = *r2.WithContext(ctx)
			next.ServeHTTP(w, r2)
			return
		}

		// Reject all unauthorized access with generic 404
		http.NotFound(w, r)
	})
}
