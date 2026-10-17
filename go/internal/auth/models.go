package auth

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`

	AvatarURL *string `json:"avatar_url,omitempty"`
}
