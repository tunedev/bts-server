package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GuestCategory represents a category of guests, like "Bride's Family".
type GuestCategory struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Side            string    `json:"side"`
	MaxGuests       int       `json:"max_guests"`
	InvitationToken *string   `json:"invitation_token"`
	DefaultCategory bool      `json:"default_category"`
	CoupleID        uuid.UUID `json:"couple_id"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateCategoryParams defines the parameters for creating a new guest category.
type CreateCategoryParams struct {
	Name            string    `json:"name"`
	Side            string    `json:"side"`
	MaxGuests       int       `json:"max_guests"`
	InvitationToken *string   `json:"invitation_token"`
	CoupleID        uuid.UUID `json:"couple_id"`
	DefaultCategory bool      `json:"default_category"`
}

// CreateCategory inserts a new guest category into the database.
func (c Client) CreateCategory(params CreateCategoryParams) (GuestCategory, error) {
	id := uuid.New()
	query := `
    INSERT INTO guest_categories (
        id,
        name,
        side,
        max_guests,
        invitation_token,
        couple_id,
				default_category
    ) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := c.DB.Exec(
		query,
		id,
		params.Name,
		params.Side,
		params.MaxGuests,
		params.InvitationToken,
		params.CoupleID,
		params.DefaultCategory,
	)
	if err != nil {
		return GuestCategory{}, err
	}

	return c.GetCategory(id)
}

// GetCategory retrieves a single guest category by its ID.
func (c Client) GetCategory(id uuid.UUID) (GuestCategory, error) {
	query := `
    SELECT
        id,
        name,
        side,
        max_guests,
        invitation_token,
        couple_id,
        created_at
    FROM guest_categories
    WHERE id = ?`

	var category GuestCategory
	err := c.DB.QueryRow(query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Side,
		&category.MaxGuests,
		&category.InvitationToken,
		&category.CoupleID,
		&category.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GuestCategory{}, nil
		}
		return GuestCategory{}, err
	}

	return category, nil
}

// GetCategoryByName retrieves a single guest category by its ID.
func (c Client) GetCategoryByName(name string) (GuestCategory, error) {
	query := `
    SELECT
        id,
        name,
        side,
        max_guests,
        invitation_token,
        couple_id,
        created_at
    FROM guest_categories
    WHERE name = ?`

	var category GuestCategory
	err := c.DB.QueryRow(query, name).Scan(
		&category.ID,
		&category.Name,
		&category.Side,
		&category.MaxGuests,
		&category.InvitationToken,
		&category.CoupleID,
		&category.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GuestCategory{}, nil
		}
		return GuestCategory{}, err
	}

	return category, nil
}

// ListCategoriesByCouple retrieves all guest categories managed by a specific couple.
func (c Client) ListCategoriesByCouple(coupleID uuid.UUID) ([]GuestCategory, error) {
	fmt.Println("Couple ID sent to me is ===>>>>>>>>>", coupleID)
	query := `
    SELECT
        id,
        name,
        side,
        max_guests,
        invitation_token,
        couple_id,
        created_at
    FROM guest_categories
    WHERE couple_id = ?
    ORDER BY created_at ASC`

	rows, err := c.DB.Query(query, coupleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []GuestCategory
	for rows.Next() {
		var category GuestCategory
		if err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Side,
			&category.MaxGuests,
			&category.InvitationToken,
			&category.CoupleID,
			&category.CreatedAt,
		); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// UpdateCategory modifies an existing guest category.
func (c Client) UpdateCategory(category GuestCategory) error {
	query := `
    UPDATE guest_categories
    SET
        name = ?,
        side = ?,
        max_guests = ?,
        invitation_token = ?
    WHERE id = ? AND couple_id = ?`

	_, err := c.DB.Exec(
		query,
		category.Name,
		category.Side,
		category.MaxGuests,
		category.InvitationToken,
		category.ID,
		category.CoupleID, // Ensure a couple can only update their own categories
	)
	return err
}

// DeleteCategory removes a guest category from the database.
func (c Client) DeleteCategory(id uuid.UUID) error {
	// Note: You may want to add logic to handle what happens to RSVPs in this category.
	query := `DELETE FROM guest_categories WHERE id = ?`
	_, err := c.DB.Exec(query, id)
	return err
}

func (c Client) GetApprovedGuestCount(categoryID uuid.UUID) (int, error) {
	query := `
    SELECT COALESCE(SUM(number_of_guests), 0)
    FROM rsvps
    WHERE category_id = ? AND status = 'APPROVED'`

	var count int
	err := c.DB.QueryRow(query, categoryID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetCategoryBySideDefault retrieves a single default category by its for a particular couple side.
func (c Client) GetCategoryBySideDefault(side string) (GuestCategory, error) {
	query := `
    SELECT
        id,
        name,
        side,
        max_guests,
        invitation_token,
        couple_id,
        created_at
    FROM guest_categories
    WHERE side = ? AND default_category = true
		ORDER BY created_at ASC
		LIMIT 1;
		`

	var category GuestCategory
	err := c.DB.QueryRow(query, side).Scan(
		&category.ID,
		&category.Name,
		&category.Side,
		&category.MaxGuests,
		&category.InvitationToken,
		&category.CoupleID,
		&category.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GuestCategory{}, nil
		}
		return GuestCategory{}, err
	}

	return category, nil
}
