package repo

import (
	"errors"
	"strings"

	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/core"
	"golang.org/x/crypto/bcrypt"
)

type UserMem struct {
	users map[string]core.UserRecord
	byID  map[int64]core.UserRecord
}

func NewUserMem() (*UserMem, error) {
	r := &UserMem{users: make(map[string]core.UserRecord), byID: make(map[int64]core.UserRecord)}
	for _, u := range []core.User{{ID: 1, Email: "admin@example.com", Role: "admin"}, {ID: 2, Email: "user@example.com", Role: "user"}} {
		hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		if len(hash) == 0 {
			return nil, errors.New("empty hash")
		}
		rec := core.UserRecord{User: u, Hash: hash}
		r.users[u.Email] = rec
		r.byID[u.ID] = rec
	}
	return r, nil
}
func (r *UserMem) ByEmail(email string) (core.UserRecord, bool) {
	u, ok := r.users[strings.ToLower(strings.TrimSpace(email))]
	return u, ok
}
func (r *UserMem) ByID(id int64) (core.UserRecord, bool) { u, ok := r.byID[id]; return u, ok }
