package repo

import (
	"context"
	"errors"

	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/core"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ErrUserNotFound = errors.New("user not found")
var ErrEmailTaken = errors.New("email already in use")

type UserStore interface {
	Create(context.Context, *core.User) error
	ByEmail(context.Context, string) (core.User, error)
}

type UserRepo struct{ db *gorm.DB }

func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Silent),
	})
}

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) AutoMigrate() error { return r.db.AutoMigrate(&core.User{}) }

func (r *UserRepo) Create(ctx context.Context, user *core.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrEmailTaken
	}
	return err
}

func (r *UserRepo) ByEmail(ctx context.Context, email string) (core.User, error) {
	var user core.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return core.User{}, ErrUserNotFound
	}
	return user, err
}
