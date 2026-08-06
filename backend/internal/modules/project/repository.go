package project

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("project not found")

type Repository interface {
	Create(ctx context.Context, p *Project) error
	FindByID(ctx context.Context, id uuid.UUID) (*Project, error)
	// ListForMember returns projects the given user belongs to, optionally
	// filtered by status/name search, paginated.
	ListForMember(ctx context.Context, userID uuid.UUID, filter ListFilter) ([]Project, int64, error)
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id uuid.UUID) error

	AddMember(ctx context.Context, m *Member) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	FindMember(ctx context.Context, projectID, userID uuid.UUID) (*Member, error)
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]Member, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p *Project) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	var p Project
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) ListForMember(ctx context.Context, userID uuid.UUID, filter ListFilter) ([]Project, int64, error) {
	query := r.db.WithContext(ctx).Model(&Project{}).
		Joins("JOIN members ON members.project_id = projects.id").
		Where("members.user_id = ?", userID)

	if filter.Status != "" {
		query = query.Where("projects.status = ?", filter.Status)
	}
	if filter.Search != "" {
		query = query.Where("projects.name ILIKE ?", "%"+filter.Search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var projects []Project
	err := query.
		Order("projects.created_at DESC").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&projects).Error
	if err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

func (r *repository) Update(ctx context.Context, p *Project) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Project{}, "id = ?", id).Error
}

func (r *repository) AddMember(ctx context.Context, m *Member) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *repository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Member{}, "project_id = ? AND user_id = ?", projectID, userID).Error
}

func (r *repository) FindMember(ctx context.Context, projectID, userID uuid.UUID) (*Member, error) {
	var m Member
	err := r.db.WithContext(ctx).First(&m, "project_id = ? AND user_id = ?", projectID, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *repository) ListMembers(ctx context.Context, projectID uuid.UUID) ([]Member, error) {
	var members []Member
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&members).Error
	return members, err
}
