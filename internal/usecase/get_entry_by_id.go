package usecase

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// GetEntryByID handles the logic for getting a single entry by its ID.
// Returns a domain entry.
func (uc *UseCase) GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*entry.Entry, error) {
	e, err := uc.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) { // Use domain error
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found"})
		}
		// Log internal error if needed
		log.Printf("Error fetching entry from repository: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entry"})
	}

	// Return domain model directly
	return e, nil
}
