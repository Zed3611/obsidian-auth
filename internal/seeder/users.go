package seeder

import (
	"context"
	"obsidian-auth/pkg/domain/models"

	"golang.org/x/crypto/bcrypt"
)

type UserCreator interface {
	Create(ctx context.Context, email, passwordHash string) (*models.User, error)
}

type UserSeeder struct {
	u UserCreator
}

type User struct {
	email    string
	password string
}

func NewUsersSeeder(u UserCreator) *UserSeeder {
	return &UserSeeder{u}
}

func (s *UserSeeder) Run(ctx context.Context) error {
	users := []*User{
		{"test1@example.com", "password1"},
		{"test2@example.com", "password2"},
		{"test3@example.com", "password3"},
		{"test4@example.com", "password4"},
		{"test5@example.com", "password5"},
		{"test6@example.com", "password6"},
		{"test7@example.com", "password7"},
		{"test8@example.com", "password8"},
		{"test9@example.com", "password9"},
		{"test10@example.com", "password10"},
	}

	for _, user := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(user.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = s.u.Create(ctx, user.email, string(hash))
		if err != nil {
			return err
		}
	}

	return nil
}
