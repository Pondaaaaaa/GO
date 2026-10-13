package core

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
type UserRecord struct {
	User
	Hash []byte
}
type UserRepository interface {
	ByEmail(string) (UserRecord, bool)
	ByID(int64) (UserRecord, bool)
}
