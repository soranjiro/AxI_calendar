package converter

import (
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/soranjiro/axicalendar/internal/domain/user"
	"github.com/soranjiro/axicalendar/internal/presentation/api"
)

// --- User Converters ---

// ToApiUser converts internal User to API User
func ToApiUser(du user.User) api.User {
	userID := du.UserID // Copy UUID
	email := du.Email   // Copy string
	apiEmail := openapi_types.Email(email)

	return api.User{
		UserId: &userID,
		Email:  &apiEmail,
	}
}
