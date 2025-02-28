package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hafiztri123/document-api/internal/user/model"
	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(ctx context.Context, user *model.User) error
	FindUserByEmail(ctx context.Context, email string) (*model.User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) Repository {
	return &authRepository{
		db: db,
	}
}
func (r *authRepository) 	CreateUser(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
func (r *authRepository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &user, nil
}
func (r *authRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound){
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}
