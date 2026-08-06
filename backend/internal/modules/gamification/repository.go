package gamification

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	FindCharacter(ctx context.Context, userID uuid.UUID) (*Character, error)
	SaveCharacter(ctx context.Context, c *Character) error
	ListCharacters(ctx context.Context) ([]Character, error)

	HasBadge(ctx context.Context, userID uuid.UUID, code BadgeCode) (bool, error)
	AwardBadge(ctx context.Context, b *Badge) error
	ListBadges(ctx context.Context, userID uuid.UUID) ([]Badge, error)

	FindMissionProgress(ctx context.Context, userID uuid.UUID, typ MissionType, periodKey string) (*MissionProgress, error)
	SaveMissionProgress(ctx context.Context, m *MissionProgress) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindCharacter(ctx context.Context, userID uuid.UUID) (*Character, error) {
	var c Character
	err := r.db.WithContext(ctx).First(&c, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) SaveCharacter(ctx context.Context, c *Character) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *repository) ListCharacters(ctx context.Context) ([]Character, error) {
	var characters []Character
	err := r.db.WithContext(ctx).Order("exp DESC").Find(&characters).Error
	return characters, err
}

func (r *repository) HasBadge(ctx context.Context, userID uuid.UUID, code BadgeCode) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Badge{}).Where("user_id = ? AND code = ?", userID, code).Count(&count).Error
	return count > 0, err
}

func (r *repository) AwardBadge(ctx context.Context, b *Badge) error {
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *repository) ListBadges(ctx context.Context, userID uuid.UUID) ([]Badge, error) {
	var badges []Badge
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("awarded_at ASC").Find(&badges).Error
	return badges, err
}

func (r *repository) FindMissionProgress(ctx context.Context, userID uuid.UUID, typ MissionType, periodKey string) (*MissionProgress, error) {
	var m MissionProgress
	err := r.db.WithContext(ctx).First(&m, "user_id = ? AND type = ? AND period_key = ?", userID, typ, periodKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *repository) SaveMissionProgress(ctx context.Context, m *MissionProgress) error {
	return r.db.WithContext(ctx).Save(m).Error
}
