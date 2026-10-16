package auth

import "github.com/sharktype/satchel/db"

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`

	AvatarURL *string `json:"avatar_url,omitempty"`
}

func MapUser(user db.User) User {
	return User{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		AvatarURL: user.AvatarUrl,
	}
}
