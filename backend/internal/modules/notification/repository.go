package notification

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, n *Notification) error
	ListForUser(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	MarkRead(ctx context.Context, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error

	FindPreference(ctx context.Context, userID uuid.UUID) (*Preference, error)
	UpsertPreference(ctx context.Context, p *Preference) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *repository) ListForUser(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, error) {
	var notifications []Notification
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

func (r *repository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	var n Notification
	err := r.db.WithContext(ctx).First(&n, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *repository) MarkRead(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Notification{}).Where("id = ?", id).Update("read", true).Error
}

func (r *repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true).Error
}

func (r *repository) FindPreference(ctx context.Context, userID uuid.UUID) (*Preference, error) {
	var p Preference
	err := r.db.WithContext(ctx).First(&p, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) UpsertPreference(ctx context.Context, p *Preference) error {
	return r.db.WithContext(ctx).Save(p).Error
}
