package main

import (
	"log"
	"net/http"
	"os"

	"github.com/tunedev/bts2025/server/internal/database"
	"github.com/tunedev/bts2025/server/internal/email"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	db        database.Client
	jwtSecret string
	platform  string
	port      string
	mailer    email.Mailer
}

func main() {
	godotenv.Load(".env")

	pathToDB := os.Getenv("DB_PATH")
	if pathToDB == "" {
		log.Fatal("DB_URL must be set")
	}

	db, err := database.NewClient(pathToDB)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	if resendAPIKey == "" {
		log.Fatal("RESEND_API_KEY environment variable is not set")
	}

	weddingFromEmail := os.Getenv("WEDDING_FROM_EMAIL")
	if weddingFromEmail == "" {
		log.Fatal("WEDDING_FROM_EMAIL environment variable is not set")
	}

	emailFromName := os.Getenv("EMAIL_SENDER_NAME")
	if emailFromName == "" {
		emailFromName = "noReply"
	}

	cfg := apiConfig{
		db:        db,
		jwtSecret: jwtSecret,
		platform:  platform,
		port:      port,
		mailer:    email.NewMailer(resendAPIKey, emailFromName, weddingFromEmail),
	}

	mux := http.NewServeMux()

	// Guest-Facing Routes
	mux.HandleFunc("GET /api/rsvp/meta", cfg.handlerGetCategoryMeta)
	mux.HandleFunc("POST /api/rsvp", cfg.handlerSubmitRSVP)

	// Admin-Facing Routes
	mux.HandleFunc("POST /api/admin/login/start", cfg.handlerLoginStart)
	mux.HandleFunc("POST /api/admin/login/verify", cfg.handlerLoginVerify)

	// These routes should be protected by middleware
	mux.HandleFunc("GET /api/admin/categories", middlewareAuth(cfg.handlerListCategories, cfg.db, cfg.jwtSecret))
	mux.HandleFunc("POST /api/admin/categories", middlewareAuth(cfg.handlerCreateCategory, cfg.db, cfg.jwtSecret))
	mux.HandleFunc("GET /api/admin/rsvps", middlewareAuth(cfg.handlerListRSVPs, cfg.db, cfg.jwtSecret))
	mux.HandleFunc("POST /api/admin/rsvps/approve", middlewareAuth(cfg.handlerApproveRSVP, cfg.db, cfg.jwtSecret))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: middlewareCORS(mux),
	}

	log.Printf("Serving on: http://localhost:%s/app/\n", port)
	log.Fatal(srv.ListenAndServe())
}
