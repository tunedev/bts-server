package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Couple represents an admin user in the database.
type Couple struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Side      string     `json:"side"`
	OTP       *string    `json:"-"`
	OTPExpiry *time.Time `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateCoupleParams defines the parameters for creating a new couple's account.
type CreateCoupleParams struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Side  string `json:"side"`
}

// CreateCouple inserts a new couple record into the database.
func (c Client) CreateCouple(params CreateCoupleParams) (Couple, error) {
	id := uuid.New()
	query := `
    INSERT INTO couples (id, name, email, side)
    VALUES (?, ?, ?, ?)`

	_, err := c.DB.Exec(query, id, params.Name, params.Email, params.Side)
	if err != nil {
		return Couple{}, err
	}

	return c.GetCouple(id)
}

// GetCouple retrieves a single couple by their ID.
func (c Client) GetCouple(id uuid.UUID) (Couple, error) {
	query := `SELECT id, name, email, side, created_at FROM couples WHERE id = ?`

	var couple Couple
	err := c.DB.QueryRow(query, id).Scan(
		&couple.ID,
		&couple.Name,
		&couple.Email,
		&couple.Side,
		&couple.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return an empty struct if not found
			return Couple{}, nil
		}
		return Couple{}, err
	}
	return couple, nil
}

// GetCoupleByEmail retrieves a single couple by their email address.
func (c Client) GetCoupleByEmail(email string) (Couple, error) {
	query := `SELECT id, name, email, side, created_at FROM couples WHERE email = ?`

	var couple Couple
	err := c.DB.QueryRow(query, email).Scan(
		&couple.ID,
		&couple.Name,
		&couple.Email,
		&couple.Side,
		&couple.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Couple{}, nil
		}
		return Couple{}, err
	}
	return couple, nil
}

// StoreOTPForCouple saves a generated OTP and its expiry time for a user.
// NOTE: You need to add `otp` and `otp_expiry` columns to your `couples` table for this.
func (c Client) StoreOTPForCouple(email string, otp string, expiry time.Time) error {
	query := `UPDATE couples SET otp = ?, otp_expiry = ? WHERE email = ?`
	_, err := c.DB.Exec(query, otp, expiry, email)
	return err
}

// VerifyOTPForCouple checks if the provided OTP is valid and not expired.
func (c Client) VerifyOTPForCouple(email string, otp string) (Couple, error) {
	query := `
    SELECT id, name, email, side, created_at
    FROM couples
    WHERE email = ? AND otp = ? AND otp_expiry > CURRENT_TIMESTAMP`

	var couple Couple
	err := c.DB.QueryRow(query, email, otp).Scan(
		&couple.ID,
		&couple.Name,
		&couple.Email,
		&couple.Side,
		&couple.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// This means the OTP was incorrect or expired
			return Couple{}, errors.New("invalid or expired OTP")
		}
		return Couple{}, err
	}

	// Optional: Clear the OTP after successful verification
	// query = `UPDATE couples SET otp = NULL, otp_expiry = NULL WHERE email = ?`
	// c.DB.Exec(query, email)

	return couple, nil
}
