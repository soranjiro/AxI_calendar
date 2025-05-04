package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/api"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	repo "github.com/soranjiro/axicalendar/internal/repository/dynamodb"
)

// GetEntryByIDUseCase defines the interface for the get entry by ID use case.
type GetEntryByIDUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*api.Entry, error)
}

// getEntryByIDUseCase implements the GetEntryByIDUseCase interface.
type getEntryByIDUseCase struct {
	entryRepo repo.EntryRepository
}

// NewGetEntryByIDUseCase creates a new GetEntryByIDUseCase.
func NewGetEntryByIDUseCase(entryRepo repo.EntryRepository) GetEntryByIDUseCase {
	return &getEntryByIDUseCase{entryRepo: entryRepo}
}

// Execute handles the logic for getting a single entry by its ID.
func (uc *getEntryByIDUseCase) Execute(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*api.Entry, error) {
	e, err := uc.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found"})
		}
		// Log internal error if needed
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entry"})
	}

	apiEntry := entry.ToApiEntry(*e)
	return &apiEntry, nil
}
