package delivery

import (
	"context"
	"net/http"
	"strings"

	"github.com/ebk-tech/be-booking-events/internal/auth/application"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"
)

// UserIDFromCtx mengambil user ID dari request context (diset oleh AuthMiddleware).
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyUserID).(string)
	return v
}

// RoleFromCtx mengambil role dari request context (diset oleh AuthMiddleware).
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyRole).(string)
	return v
}

// AuthMiddleware memvalidasi JWT dari header Authorization: Bearer <token>.
// Jika valid, cek Redis blocklist (logout), lalu set user_id dan role ke context.
func AuthMiddleware(tokenProvider application.TokenProvider, redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "Authorization header wajib diisi")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			userID, role, err := tokenProvider.Verify(tokenStr)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "token tidak valid atau sudah kadaluarsa")
				return
			}

			// Cek apakah token sudah di-logout (ada di Redis blocklist jika Redis tersedia).
			if redisClient != nil {
				blocklisted, _ := redisClient.Exists(r.Context(), "jwt_blocklist:"+tokenStr).Result()
				if blocklisted > 0 {
					writeError(w, http.StatusUnauthorized, "token sudah tidak valid, silakan login kembali")
					return
				}
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole adalah middleware yang memastikan user memiliki salah satu dari role yang diizinkan.
// Harus dipasang setelah AuthMiddleware.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[RoleFromCtx(r.Context())] {
				writeError(w, http.StatusForbidden, "tidak memiliki akses untuk operasi ini")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
