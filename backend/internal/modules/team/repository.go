package team

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("department not found")

type Repository interface {
	CreateDepartment(ctx context.Context, d *Department) error
	ListDepartments(ctx context.Context) ([]Department, error)
	FindDepartment(ctx context.Context, id uuid.UUID) (*Department, error)
	DeleteDepartment(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateDepartment(ctx context.Context, d *Department) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *repository) ListDepartments(ctx context.Context) ([]Department, error) {
	var departments []Department
	err := r.db.WithContext(ctx).Order("name ASC").Find(&departments).Error
	return departments, err
}

func (r *repository) FindDepartment(ctx context.Context, id uuid.UUID) (*Department, error) {
	var d Department
	err := r.db.WithContext(ctx).First(&d, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *repository) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Department{}, "id = ?", id).Error
}
