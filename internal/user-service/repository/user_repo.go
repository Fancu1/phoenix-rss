package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/user-service/models"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(user *models.User) (*models.User, error) {
	result := r.db.Create(user)
	return user, result.Error
}

func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	user := &models.User{}
	result := r.db.First(user, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return user, result.Error
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	user := &models.User{}
	result := r.db.Where("username = ?", username).First(user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return user, result.Error
}

func (r *UserRepository) Update(user *models.User) (*models.User, error) {
	result := r.db.Save(user)
	return user, result.Error
}
