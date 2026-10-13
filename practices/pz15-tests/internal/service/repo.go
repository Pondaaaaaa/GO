package service

import "errors"

type User struct {
	ID    int64
	Email string
}

var ErrNotFound = errors.New("not found")

type UserRepo interface {
	ByEmail(email string) (User, error)
}
