package database

import (
	"crypto/rand"
	"encoding/hex"
	"log"

	"github.com/google/uuid"
)

// SeedDatabase orchestrates the seeding of all necessary startup data.
func (c Client) SeedDatabase() error {
	log.Println("Seeding database...")

	// 1. Seed the couples (Bride and Groom)
	bride, err := c.seedCouple("Diamond", "bride@example.com", "BRIDE")
	if err != nil {
		return err
	}

	groom, err := c.seedCouple("Babatunde", "groom@example.com", "GROOM")
	if err != nil {
		return err
	}

	// 2. Seed the guest categories for each couple
	err = c.seedCategory("Bride's Family", 100, bride)
	if err != nil {
		return err
	}
	err = c.seedCategory("Bride's Friends", 50, bride)
	if err != nil {
		return err
	}

	err = c.seedCategory("Groom's Family", 100, groom)
	if err != nil {
		return err
	}
	err = c.seedCategory("Groom's Friends", 50, groom)
	if err != nil {
		return err
	}

	// Add a "catch-all" category for website RSVPs that require approval
	err = c.seedCategory("Bride's Website RSVPs", 20, bride)
	if err != nil {
		return err
	}

	log.Println("Database seeding complete.")
	return nil
}

// seedCouple creates a couple if they don't already exist.
func (c Client) seedCouple(name, email, side string) (Couple, error) {
	existingCouple, err := c.GetCoupleByEmail(email)
	if err != nil {
		return Couple{}, err
	}
	// If couple already exists, return it
	if existingCouple.ID != uuid.Nil {
		log.Printf("Couple '%s' already exists, skipping.", name)
		return existingCouple, nil
	}

	// Otherwise, create it
	log.Printf("Creating couple: %s", name)
	return c.CreateCouple(CreateCoupleParams{
		Name:  name,
		Email: email,
		Side:  side,
	})
}

// seedCategory creates a guest category if it doesn't already exist.
func (c Client) seedCategory(name string, maxGuests int, couple Couple) error {
	existingCategory, err := c.GetCategoryByName(name) // You'll need to create this DB function
	if err != nil {
		return err
	}
	if existingCategory.ID != uuid.Nil {
		log.Printf("Category '%s' already exists, skipping.", name)
		return nil
	}

	// Generate a secure, random invitation token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	log.Printf("Creating category: %s", name)
	_, err = c.CreateCategory(CreateCategoryParams{
		Name:            name,
		Side:            couple.Side,
		MaxGuests:       maxGuests,
		CoupleID:        couple.ID,
		InvitationToken: &token,
	})
	return err
}
