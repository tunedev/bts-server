package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// RSVP represents a single RSVP record in the database.
type RSVP struct {
	ID             uuid.UUID     `json:"id"`
	GuestName      string        `json:"guest_name"`
	NumberOfGuests int           `json:"number_of_guests"`
	Email          string        `json:"email"`
	Phone          string        `json:"phone"` // Use a pointer for optional fields
	Status         string        `json:"status"`
	CategoryID     uuid.NullUUID `json:"category_id"`
	SubmittedAt    time.Time     `json:"submitted_at"`
}

// CreateRSVPParams defines the parameters for creating a new RSVP.
type CreateRSVPParams struct {
	GuestName      string        `json:"guest_name"`
	NumberOfGuests int           `json:"number_of_guests"`
	Email          string        `json:"email"`
	Phone          string        `json:"phone"`
	CategoryID     uuid.NullUUID `json:"category_id"`
}

func (c Client) CreateRSVP(params CreateRSVPParams, status string) (RSVP, error) {
	id := uuid.New()
	query := `
    INSERT INTO rsvps (
        id,
        guest_name,
        number_of_guests,
        email,
        phone,
        category_id,
				status
    ) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := c.DB.Exec(
		query,
		id,
		params.GuestName,
		params.NumberOfGuests,
		params.Email,
		params.Phone,
		params.CategoryID,
		status,
	)
	if err != nil {
		return RSVP{}, err
	}

	return c.GetRSVP(id)
}

// GetRSVP retrieves a single RSVP by its ID.
func (c Client) GetRSVP(id uuid.UUID) (RSVP, error) {
	query := `
    SELECT
        id,
        guest_name,
        number_of_guests,
        email,
        phone,
        status,
        category_id,
        submitted_at
    FROM rsvps
    WHERE id = ?`

	var rsvp RSVP
	err := c.DB.QueryRow(query, id).Scan(
		&rsvp.ID,
		&rsvp.GuestName,
		&rsvp.NumberOfGuests,
		&rsvp.Email,
		&rsvp.Phone,
		&rsvp.Status,
		&rsvp.CategoryID,
		&rsvp.SubmittedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return an empty RSVP and no error if not found, or you can return a specific error
			return RSVP{}, nil
		}
		return RSVP{}, err
	}

	return rsvp, nil
}

// ListRSVPsByCategory retrieves all RSVPs belonging to a specific guest category.
func (c Client) ListRSVPsByCategory(categoryID uuid.UUID) ([]RSVP, error) {
	query := `
    SELECT
        id,
        guest_name,
        number_of_guests,
        email,
        phone,
        status,
        category_id,
        submitted_at
    FROM rsvps
    WHERE category_id = ?
    ORDER BY submitted_at DESC`

	rows, err := c.DB.Query(query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rsvps []RSVP
	for rows.Next() {
		var rsvp RSVP
		if err := rows.Scan(
			&rsvp.ID,
			&rsvp.GuestName,
			&rsvp.NumberOfGuests,
			&rsvp.Email,
			&rsvp.Phone,
			&rsvp.Status,
			&rsvp.CategoryID,
			&rsvp.SubmittedAt,
		); err != nil {
			return nil, err
		}
		rsvps = append(rsvps, rsvp)
	}

	return rsvps, nil
}

// UpdateRSVPStatus updates the status of an RSVP (e.g., from PENDING to APPROVED).
func (c Client) UpdateRSVPStatus(id uuid.UUID, status string) error {
	query := `
    UPDATE rsvps
    SET status = ?
    WHERE id = ?`

	_, err := c.DB.Exec(query, status, id)
	return err
}

// DeleteRSVP removes an RSVP record from the database.
func (c Client) DeleteRSVP(id uuid.UUID) error {
	query := `DELETE FROM rsvps WHERE id = ?`
	_, err := c.DB.Exec(query, id)
	return err
}

// ListAllRSVPs retrieves all RSVPs from the database.
// If a non-empty status string is provided, it filters the results.
func (c Client) ListAllRSVPs(status, side string) ([]RSVP, error) {
	query := `
    SELECT
        id,
        guest_name,
        number_of_guests,
        email,
        phone,
        status,
        category_id,
        submitted_at
    FROM rsvps`

	args := []interface{}{}

	// If a status filter is provided, add it to the query
	if status != "" && side != "" {
		query += " WHERE status = ? AND side = ?"
		args = append(args, status)
		args = append(args, side)
	}

	if status == "" && side != "" {
		query += " WHERE status = ? AND side = ?"
		args = append(args, side)
	}

	query += " ORDER BY submitted_at ASC"

	rows, err := c.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rsvps []RSVP
	for rows.Next() {
		var rsvp RSVP
		var categoryID uuid.NullUUID

		if err := rows.Scan(
			&rsvp.ID,
			&rsvp.GuestName,
			&rsvp.NumberOfGuests,
			&rsvp.Email,
			&rsvp.Phone,
			&rsvp.Status,
			&categoryID,
			&rsvp.SubmittedAt,
		); err != nil {
			return nil, err
		}

		rsvp.CategoryID = categoryID
		rsvps = append(rsvps, rsvp)
	}

	return rsvps, nil
}

// AssignCategoryToRSVP updates an existing RSVP to assign it to a guest category.
// This is used when an admin approves an RSVP that was submitted from the main website.
func (c Client) AssignCategoryToRSVP(rsvpID uuid.UUID, categoryID uuid.UUID) error {
	query := `
    UPDATE rsvps
    SET category_id = ?
    WHERE id = ?`

	result, err := c.DB.Exec(query, categoryID, rsvpID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no RSVP found with the given ID to update")
	}

	return nil
}
