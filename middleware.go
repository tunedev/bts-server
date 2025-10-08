package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tunedev/bts2025/server/internal/auth"
	"github.com/tunedev/bts2025/server/internal/database"
)

type contextKey string

const coupleIDKey = contextKey("coupleID")
const coupleAuthDetailsKey = contextKey("coupleAuthDetailsKey")

// MiddlewareAuth is a middleware that protects admin routes.
// It validates the JWT and attaches the couple's ID to the request context.
func middlewareAuth(handler http.HandlerFunc, db database.Client, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, err.Error(), err)
			return
		}

		coupleID, err := auth.ValidateJWT(tokenString, jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
			return
		}

		coupleDetail, err := db.GetCouple(coupleID)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "User not found", err)
			return
		}

		ctx := context.WithValue(r.Context(), coupleIDKey, coupleID)
		ctx = context.WithValue(ctx, coupleAuthDetailsKey, coupleDetail)

		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

// middlewareCORS adds CORS headers to every request.
func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the allowed origin. Use "*" for development, or your specific frontend URL.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests (the browser sends an OPTIONS request first)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// GetCoupleIDFromContext is a helper function to retrieve the couple's ID from the context.
func GetCoupleIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	coupleID, ok := ctx.Value(coupleIDKey).(uuid.UUID)
	return coupleID, ok
}

func GetCoupleDetailsFromCtx(ctx context.Context) (database.Couple, bool) {
	coupleDetails, ok := ctx.Value(coupleAuthDetailsKey).(database.Couple)
	return coupleDetails, ok
}

func middlewareLogger(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		logger.Info("request handled", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}
