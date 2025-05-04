package domain

import "errors"

// Standard repository errors
var (
    // ErrNotFound indicates that a requested item was not found.
    ErrNotFound = errors.New("item not found")
    // ErrForbidden indicates that the user does not have permission to access the requested item.
    ErrForbidden = errors.New("forbidden")
    // ErrThemeNotFound indicates a specific theme was not found.
    ErrThemeNotFound = errors.New("theme not found") // Kept for potential specific handling
    // ErrEntryNotFound indicates a specific entry was not found.
    ErrEntryNotFound = errors.New("entry not found") // Kept for potential specific handling
    // ErrCannotDeleteDefaultTheme indicates an attempt to delete a default theme.
    ErrCannotDeleteDefaultTheme = errors.New("cannot delete default theme")
    // ErrCannotUpdateDefaultTheme indicates an attempt to update a default theme.
    ErrCannotUpdateDefaultTheme = errors.New("cannot update default theme")
    // ErrAlreadyExists indicates an attempt to create an item that already exists.
    ErrAlreadyExists = errors.New("item already exists")
)
