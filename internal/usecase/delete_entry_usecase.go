package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain"
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"
)

// DeleteEntryUseCase defines the interface for the delete entry use case.
type DeleteEntryUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) error
}

// deleteEntryUseCase implements the DeleteEntryUseCase interface.
type deleteEntryUseCase struct {
	entryRepo repo.EntryRepository
}

// NewDeleteEntryUseCase creates a new DeleteEntryUseCase.
func NewDeleteEntryUseCase(entryRepo repo.EntryRepository) DeleteEntryUseCase {
	return &deleteEntryUseCase{entryRepo: entryRepo}
}

// Execute handles the logic for deleting an entry.
func (uc *deleteEntryUseCase) Execute(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) error {
	// Need EntryDate to delete. Get the entry first.
	e, err := uc.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found"})
		}
		// Log internal error if needed
		return echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entry before delete"})
	}

	// Now delete using the retrieved date
	err = uc.entryRepo.DeleteEntry(ctx, userID, entryID, e.EntryDate)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) { // Should not happen if GetEntryByID succeeded, but check anyway
			return echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found during delete attempt"})
		}
		// Log internal error if needed
		return echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to delete entry"})
	}

	return nil // Success indicates no content (204)
}
