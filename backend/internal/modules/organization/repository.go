package organization

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("organization not found")

// Repository is intentionally minimal for P1.1 — just enough to read the
// default organization the migration created. Create/Update/List and the
// service/handler layers land in later P1 phases once something actually
// needs them.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	var org Organization
	if err := r.db.WithContext(ctx).First(&org, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *Repository) FindBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization
	if err := r.db.WithContext(ctx).First(&org, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}
