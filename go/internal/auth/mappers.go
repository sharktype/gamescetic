package auth

import "github.com/sharktype/gamescetic/generated"

func rowToUser(row generated.User) User {
	return User{
		ID:    row.ID,
		Sub:   row.Sub,
		Name:  row.Name,
		Email: row.Email,
	}
}
