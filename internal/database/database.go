package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
	DB *sql.DB
}

func NewClient(dataSourceName string) (Client, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return Client{}, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return Client{}, fmt.Errorf("failed to connect to database: %w", err)
	}

	c := Client{DB: db}
	if err := c.autoMigrate(); err != nil {
		return Client{}, fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Database client initialized and migrated successfully.")
	return c, nil
}

// autoMigrate creates all necessary tables if they don't already exist.
func (c *Client) autoMigrate() error {
	couplesTable := `
    CREATE TABLE IF NOT EXISTS couples (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        side TEXT NOT NULL CHECK(side IN ('BRIDE', 'GROOM')),
				otp TEXT,
				otp_expiry TIMESTAMP,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`

	guestCategoriesTable := `
    CREATE TABLE IF NOT EXISTS guest_categories (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        side TEXT NOT NULL CHECK(side IN ('BRIDE', 'GROOM')),
        max_guests INTEGER NOT NULL,
        invitation_token TEXT NOT NULL UNIQUE,
        couple_id TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (couple_id) REFERENCES couples(id)
    );`

	rsvpsTable := `
    CREATE TABLE IF NOT EXISTS rsvps (
        id TEXT PRIMARY KEY,
        guest_name TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        phone TEXT NOT NULL UNIQUE,
        number_of_guests INTEGER NOT NULL,
        status TEXT NOT NULL DEFAULT 'PENDING' CHECK(status IN ('PENDING', 'APPROVED', 'REJECTED')),
        category_id TEXT,
        submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (category_id) REFERENCES guest_categories(id)
    );
		
		CREATE INDEX IF NOT EXISTS idx_rsvps_category_id ON rsvps(category_id);

		`

	// Execute tables in order of dependency
	if _, err := c.DB.Exec(couplesTable); err != nil {
		return fmt.Errorf("failed to create couples table: %w", err)
	}
	if _, err := c.DB.Exec(guestCategoriesTable); err != nil {
		return fmt.Errorf("failed to create guest_categories table: %w", err)
	}
	if _, err := c.DB.Exec(rsvpsTable); err != nil {
		return fmt.Errorf("failed to create rsvps table: %w", err)
	}

	return nil
}
