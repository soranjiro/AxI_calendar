package services

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/soranjiro/axicalendar/internal/domain"
	"github.com/soranjiro/axicalendar/internal/domain/entry"
	dynamodbrepo "github.com/soranjiro/axicalendar/internal/adapter/persistence/dynamodb"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

type EntryService struct {
	entryRepo dynamodbrepo.EntryRepository
}

func NewEntryService(entryRepo dynamodbrepo.EntryRepository) *EntryService {
	return &EntryService{entryRepo: entryRepo}
}

func (s *EntryService) GetEntryByID(ctx context.Context, userID uuid.UUID, entryID uuid.UUID) (*entry.Entry, error) {
	e, err := s.entryRepo.GetEntryByID(ctx, userID, entryID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, api.Error{Message: "Entry not found"})
		}
		log.Printf("Error fetching entry from repository: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entry"})
	}
	return e, nil
}

func (s *EntryService) GetEntries(ctx context.Context, userID uuid.UUID, themeID uuid.UUID, startDate time.Time, endDate time.Time) ([]entry.Entry, error) {
	if startDate.IsZero() || endDate.IsZero() {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "start_date and end_date cannot be zero"})
	}
	if endDate.Before(startDate) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, api.Error{Message: "end_date cannot be before start_date"})
	}

	entries, err := s.entryRepo.ListEntriesByDateRange(ctx, userID, startDate, endDate, themeID)
	if err != nil {
		log.Printf("Error fetching entries from repository: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, api.Error{Message: "Failed to retrieve entries"})
	}
	return entries, nil
}
