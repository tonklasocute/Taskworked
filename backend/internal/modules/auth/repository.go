package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error

	ListAll(ctx context.Context) ([]User, error)
	// ListByOrganization is the tenant-safe counterpart to ListAll — see
	// Service.ListUsersByOrganization's doc comment for when to use which.
	ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]User, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role Role) error
	UpdateDepartment(ctx context.Context, id uuid.UUID, departmentID *uuid.UUID) error

	// Count backs the first-admin bootstrap check in Register: an empty
	// table means there's nobody yet who could grant the new account
	// admin access, so it grants itself.
	Count(ctx context.Context) (int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *repository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

func (r *repository) ListAll(ctx context.Context) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).Order("name ASC").Find(&users).Error
	return users, err
}

func (r *repository) ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).Where("organization_id = ?", organizationID).Order("name ASC").Find(&users).Error
	return users, err
}

func (r *repository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

func (r *repository) UpdateRole(ctx context.Context, id uuid.UUID, role Role) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("role", role).Error
}

func (r *repository) UpdateDepartment(ctx context.Context, id uuid.UUID, departmentID *uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("department_id", departmentID).Error
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Count(&count).Error
	return count, err
}
