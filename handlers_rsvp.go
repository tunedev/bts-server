package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/tunedev/bts2025/server/internal/database"
	"github.com/tunedev/bts2025/server/internal/email"
)

// handlerGetCategoryMeta fetches public data for an RSVP link
func (cfg *apiConfig) handlerGetCategoryMeta(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondWithError(w, http.StatusBadRequest, "Invitation token is required", nil)
		return
	}

	parsedToken, err := uuid.Parse(token)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid rsvp link", err)
		return
	}
	category, err := cfg.db.GetCategory(parsedToken)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "invalid rsvp link", err)
		return
	}

	approvedCount, err := cfg.db.GetApprovedGuestCount(category.ID)
	if err != nil {
		log.Printf("Error getting approved guest count: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Could not retrieve guest count", err)
		return
	}
	remainingSpots := category.MaxGuests - approvedCount

	// Prepare the data payload for the frontend
	payload := map[string]interface{}{
		"name":            category.Name,
		"side":            category.Side,
		"remainingGuests": remainingSpots,
	}

	respondWithJSON(w, http.StatusOK, payload)
}

func (cfg *apiConfig) handlerSubmitRSVP(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		Guests       int    `json:"guests"`
		Token        string `json:"token"`
		SelectedSide string `json:"selectedSide"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	var categoryID uuid.NullUUID
	var err error
	status := "PENDING"

	// Logic Branch 1: Guest used a direct invitation link with a token
	if params.Token != "" {
		parsedToken, err := uuid.Parse(params.Token)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "malformed request, unable to parse token to uuid", err)
			return
		}
		category, err := cfg.db.GetCategory(parsedToken)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Invalid invitation link.", err)
			return
		}

		categoryID = uuid.NullUUID{UUID: category.ID, Valid: true}
		approvedCount, _ := cfg.db.GetApprovedGuestCount(category.ID)
		if approvedCount+params.Guests > category.MaxGuests {
			status = "PENDING"
		} else {
			status = "APPROVED"
		}
	} else if params.SelectedSide != "" {
		categoryID = uuid.NullUUID{Valid: false}
		status = "PENDING"
	} else {
		respondWithError(w, http.StatusBadRequest, "Missing required RSVP information.", err)
		return
	}

	rsvpParams := database.CreateRSVPParams{
		GuestName:      params.Name,
		NumberOfGuests: params.Guests,
		Email:          params.Email,
		Phone:          params.Phone,
		CategoryID:     categoryID,
	}

	newRSVP, err := cfg.db.CreateRSVP(rsvpParams, status)
	if err != nil {
		log.Printf("Error creating RSVP: %v", err)
		if isUniqueConstraintError(err) {
			respondWithError(w, http.StatusConflict, "This email or phone number has already been used to RSVP.", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Could not save your RSVP.", err)
		return
	}

	switch newRSVP.Status {
	case "APPROVED":
		cfg.mailer.SendRSVPConfirmed(newRSVP.Email, email.SendRSVPConfirmedParam{
			GuestName:      newRSVP.GuestName,
			Phone:          newRSVP.Phone,
			RSVPID:         newRSVP.ID.String(),
			NumberOfGuests: newRSVP.NumberOfGuests,
		})
	case "PENDING":
		cfg.mailer.SendRSVPReceived(newRSVP.Email, newRSVP.GuestName)
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{"success": true, "status": newRSVP.Status})
}

func isUniqueConstraintError(err error) bool {

	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
