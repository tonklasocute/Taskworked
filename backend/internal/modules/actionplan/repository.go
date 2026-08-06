package actionplan

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	CreateGoal(ctx context.Context, g *Goal) error
	FindGoal(ctx context.Context, id uuid.UUID) (*Goal, error)
	ListGoalsForProject(ctx context.Context, projectID uuid.UUID) ([]Goal, error)
	UpdateGoal(ctx context.Context, g *Goal) error
	DeleteGoal(ctx context.Context, id uuid.UUID) error

	CreateMilestone(ctx context.Context, m *Milestone) error
	FindMilestone(ctx context.Context, id uuid.UUID) (*Milestone, error)
	ListMilestonesForGoal(ctx context.Context, goalID uuid.UUID) ([]Milestone, error)
	ListMilestonesForProject(ctx context.Context, projectID uuid.UUID) ([]Milestone, error)
	UpdateMilestone(ctx context.Context, m *Milestone) error
	DeleteMilestone(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateGoal(ctx context.Context, g *Goal) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *repository) FindGoal(ctx context.Context, id uuid.UUID) (*Goal, error) {
	var g Goal
	err := r.db.WithContext(ctx).First(&g, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *repository) ListGoalsForProject(ctx context.Context, projectID uuid.UUID) ([]Goal, error) {
	var goals []Goal
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at ASC").Find(&goals).Error
	return goals, err
}

func (r *repository) UpdateGoal(ctx context.Context, g *Goal) error {
	return r.db.WithContext(ctx).Save(g).Error
}

func (r *repository) DeleteGoal(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Goal{}, "id = ?", id).Error
}

func (r *repository) CreateMilestone(ctx context.Context, m *Milestone) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *repository) FindMilestone(ctx context.Context, id uuid.UUID) (*Milestone, error) {
	var m Milestone
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *repository) ListMilestonesForGoal(ctx context.Context, goalID uuid.UUID) ([]Milestone, error) {
	var milestones []Milestone
	err := r.db.WithContext(ctx).Where("goal_id = ?", goalID).Order("created_at ASC").Find(&milestones).Error
	return milestones, err
}

func (r *repository) ListMilestonesForProject(ctx context.Context, projectID uuid.UUID) ([]Milestone, error) {
	var milestones []Milestone
	err := r.db.WithContext(ctx).
		Joins("JOIN goals ON goals.id = milestones.goal_id").
		Where("goals.project_id = ?", projectID).
		Order("milestones.created_at ASC").
		Find(&milestones).Error
	return milestones, err
}

func (r *repository) UpdateMilestone(ctx context.Context, m *Milestone) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *repository) DeleteMilestone(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Milestone{}, "id = ?", id).Error
}
