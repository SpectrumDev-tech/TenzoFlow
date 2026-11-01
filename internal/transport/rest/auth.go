package rest

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

// requireBearerToken lets a request through only if it carries
// "Authorization: Bearer <token>" with exactly the configured token.
//
// TEMPORARY: this is a stop-gap for GET /api/v1/status until users and roles exist.
// TODO(admin): replace it with a real "current user must be an admin" middleware.
//
// SECURITY: tokens are compared in constant time. A plain == stops at the first
// differing byte, so response time would leak how much of a guess was correct.
// Both values are hashed first so the compared slices always have the same length
// (ConstantTimeCompare returns early when lengths differ).
func requireBearerToken(token string) Middleware {
	want := sha256.Sum256([]byte(token))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer")
			sum := sha256.Sum256([]byte(got))

			if !ok || subtle.ConstantTimeCompare(sum[:], want[:]) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="tenzoflow"`)
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
