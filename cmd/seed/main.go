package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"

	"github.com/tunedev/bts2025/server/internal/database"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	pathToDB := os.Getenv("DB_PATH")
	if pathToDB == "" {
		log.Fatal("DB_PATH must be set")
	}

	groomsEmail := os.Getenv("GROOMS_EMAIL")
	if groomsEmail == "" {
		log.Fatal("GROOMS_EMAIL is required for seed")
	}

	bridesEmail := os.Getenv("BRIDES_EMAIL")
	if bridesEmail == "" {
		log.Fatal("BRIDES_EMAIL is required for seed")
	}

	// Connect to the database
	db, err := database.NewClient(pathToDB)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}

	log.Println("Seeding database...")

	// 1. Seed the couples (Bride and Groom)
	bride, err := seedCouple(db, "Diamond", bridesEmail, "BRIDE")
	if err != nil {
		log.Fatalf("Failed to seed bride: %v", err)
	}

	groom, err := seedCouple(db, "Babatunde", groomsEmail, "GROOM")
	if err != nil {
		log.Fatalf("Failed to seed groom: %v", err)
	}

	// 2. Seed the guest categories
	seedCategory(db, "Bride's Family", 100, bride)
	seedCategory(db, "Bride's Friends", 50, bride)
	seedCategory(db, "Groom's Family", 100, groom)
	seedCategory(db, "Groom's Friends", 50, groom)
	seedCategory(db, "Groom's siblings Friends", 10, groom)
	seedCategory(db, "Bride's siblings Friends", 10, bride)

	log.Println("Database seeding complete. ✅")
}

// seedCouple creates a couple if they don't already exist.
func seedCouple(c database.Client, name, email, side string) (database.Couple, error) {
	existingCouple, err := c.GetCoupleByEmail(email)
	if err != nil {
		return database.Couple{}, err
	}
	if existingCouple.ID != uuid.Nil {
		log.Printf("Couple '%s' already exists, skipping.", name)
		return existingCouple, nil
	}

	log.Printf("Creating couple: %s", name)
	return c.CreateCouple(database.CreateCoupleParams{
		Name:  name,
		Email: email,
		Side:  side,
	})
}

// seedCategory creates a guest category if it doesn't already exist.
func seedCategory(c database.Client, name string, maxGuests int, couple database.Couple) {
	existingCategory, err := c.GetCategoryByName(name)
	if err != nil {
		log.Printf("Error checking category %s: %v", name, err)
		return
	}
	if existingCategory.ID != uuid.Nil {
		log.Printf("Category '%s' already exists, skipping.", name)
		return
	}

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Printf("Error generating token for %s: %v", name, err)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	log.Printf("Creating category: %s", name)
	_, err = c.CreateCategory(database.CreateCategoryParams{
		Name:            name,
		Side:            couple.Side,
		MaxGuests:       maxGuests,
		CoupleID:        couple.ID,
		InvitationToken: &token,
	})
	if err != nil {
		log.Printf("Error creating category %s: %v", name, err)
	}
}
