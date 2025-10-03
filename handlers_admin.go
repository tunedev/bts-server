package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tunedev/bts2025/server/internal/auth"     // Adjust import path
	"github.com/tunedev/bts2025/server/internal/database" // Adjust import path
	"github.com/tunedev/bts2025/server/internal/email"

	"github.com/google/uuid"
)

// handlerLoginStart initiates the passwordless sign-in process.
func (cfg *apiConfig) handlerLoginStart(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	couple, err := cfg.db.GetCoupleByEmail(params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}
	if couple.ID == uuid.Nil {
		respondWithError(w, http.StatusNotFound, "Account not found for that email", nil)
		return
	}

	otp, err := auth.GenerateOTP()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not generate OTP", err)
		return
	}

	expiry := time.Now().Add(10 * time.Minute)
	if err := cfg.db.StoreOTPForCouple(params.Email, otp, expiry); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not save OTP", err)
		return
	}

	// Send the OTP via your emailer utility
	if err := cfg.mailer.SendLoginOTP(params.Email, otp); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to send OTP email", err)
		return
	}

	respondWithJSON(w, http.StatusOK, responseStructure{
		Data:    map[string]any{"message": "OTP sent to your email."},
		Success: true,
		Message: "OTP Sent successfully",
	})
}

// handlerLoginVerify validates an OTP and returns a session JWT.
func (cfg *apiConfig) handlerLoginVerify(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}
	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	couple, err := cfg.db.VerifyOTPForCouple(params.Email, params.OTP)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired OTP", err)
		return
	}

	token, err := auth.MakeJWT(couple.ID, cfg.jwtSecret, time.Hour*24)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create session token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, responseStructure{
		Data:    map[string]any{"token": token},
		Success: true,
		Message: "Login is successful",
	})
}

func (cfg *apiConfig) handlerCreateCategory(w http.ResponseWriter, r *http.Request) {
	// Assume coupleID is retrieved from context via auth middleware
	coupleID, _ := GetCoupleIDFromContext(r.Context())

	params := database.CreateCategoryParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format", err)
		return
	}
	params.CoupleID = coupleID

	category, err := cfg.db.CreateCategory(params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create category", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, responseStructure{
		Data:    category,
		Message: "Created category successfully",
		Success: true,
	})
}

func (cfg *apiConfig) handlerListCategories(w http.ResponseWriter, r *http.Request) {
	coupleID, _ := GetCoupleIDFromContext(r.Context())

	categories, err := cfg.db.ListCategoriesByCouple(coupleID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not retrieve categories", err)
		return
	}

	respondWithJSON(w, http.StatusOK, responseStructure{
		Data:    categories,
		Message: "Categories retrieved successfully",
		Success: true,
	})
}

func (cfg *apiConfig) handlerListRSVPs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	coupleDetails, ok := GetCoupleDetailsFromCtx(r.Context())
	if !ok {
		respondWithError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden), nil)
		return
	}

	rsvps, err := cfg.db.ListAllRSVPs(status, coupleDetails.Side)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not retrieve RSVPs", err)
		return
	}

	respondWithJSON(w, http.StatusOK, responseStructure{
		Data:    rsvps,
		Message: "Retrieved RSVP list successfully",
		Success: true,
	})
}

func (cfg *apiConfig) handlerApproveRSVP(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		RSVPID     uuid.UUID `json:"rsvpId"`
		Action     string    `json:"action"`
		CategoryID uuid.UUID `json:"categoryId"`
	}
	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	rsvp, err := cfg.db.GetRSVP(params.RSVPID)
	if err != nil || rsvp.ID == uuid.Nil {
		respondWithError(w, http.StatusNotFound, "RSVP not found", err)
		return
	}

	if params.Action == "APPROVE" {
		if !rsvp.CategoryID.Valid {
			if params.CategoryID == uuid.Nil {
				respondWithError(w, http.StatusBadRequest, "A category must be assigned to approve this RSVP", nil)
				return
			}
			if err := cfg.db.AssignCategoryToRSVP(rsvp.ID, params.CategoryID); err != nil {
				respondWithError(w, http.StatusInternalServerError, "Could not assign category", err)
				return
			}
		}
	}

	newStatus := params.Action + "D"
	if err := cfg.db.UpdateRSVPStatus(rsvp.ID, newStatus); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update RSVP status", err)
		return
	}

	switch newStatus {
	case "APPROVED":
		cfg.mailer.SendRSVPConfirmed(rsvp.Email, email.SendRSVPConfirmedParam{
			GuestName:      rsvp.GuestName,
			Phone:          rsvp.Phone,
			NumberOfGuests: rsvp.NumberOfGuests,
			RSVPID:         rsvp.ID.String(),
		})
	case "REJECTED":
		cfg.mailer.SendRSVPRejected(rsvp.Email, rsvp.GuestName)
	}

	respondWithJSON(w, http.StatusOK, responseStructure{
		Data:    map[string]any{"message": "RSVP status updated successfully."},
		Message: "Approved RSVP successfully",
		Success: true,
	})
}
