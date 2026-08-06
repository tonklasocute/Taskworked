package task

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("task not found")

type Repository interface {
	Create(ctx context.Context, t *Task) error
	FindByID(ctx context.Context, id uuid.UUID) (*Task, error)
	List(ctx context.Context, filter ListFilter) ([]Task, int64, error)
	Update(ctx context.Context, t *Task) error
	Delete(ctx context.Context, id uuid.UUID) error

	SetTags(ctx context.Context, taskID uuid.UUID, tags []string) error
	Tags(ctx context.Context, taskID uuid.UUID) ([]string, error)

	AddChecklistItem(ctx context.Context, item *ChecklistItem) error
	UpdateChecklistItem(ctx context.Context, item *ChecklistItem) error
	DeleteChecklistItem(ctx context.Context, id uuid.UUID) error
	FindChecklistItem(ctx context.Context, id uuid.UUID) (*ChecklistItem, error)
	ListChecklistItems(ctx context.Context, taskID uuid.UUID) ([]ChecklistItem, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, t *Task) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*Task, error) {
	var t Task
	err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repository) List(ctx context.Context, filter ListFilter) ([]Task, int64, error) {
	query := r.db.WithContext(ctx).Model(&Task{}).Where("project_id = ?", filter.ProjectID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.AssigneeID != "" {
		query = query.Where("assignee_id = ?", filter.AssigneeID)
	}
	if filter.Search != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []Task
	err := query.
		Order("created_at DESC").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *repository) Update(ctx context.Context, t *Task) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Task{}, "id = ?", id).Error
}

func (r *repository) SetTags(ctx context.Context, taskID uuid.UUID, tags []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&Tag{}, "task_id = ?", taskID).Error; err != nil {
			return err
		}
		for _, tag := range tags {
			if err := tx.Create(&Tag{TaskID: taskID, Tag: tag}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) Tags(ctx context.Context, taskID uuid.UUID) ([]string, error) {
	var rows []Tag
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Find(&rows).Error; err != nil {
		return nil, err
	}
	tags := make([]string, len(rows))
	for i, row := range rows {
		tags[i] = row.Tag
	}
	return tags, nil
}

func (r *repository) AddChecklistItem(ctx context.Context, item *ChecklistItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *repository) UpdateChecklistItem(ctx context.Context, item *ChecklistItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *repository) DeleteChecklistItem(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&ChecklistItem{}, "id = ?", id).Error
}

func (r *repository) FindChecklistItem(ctx context.Context, id uuid.UUID) (*ChecklistItem, error) {
	var item ChecklistItem
	err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) ListChecklistItems(ctx context.Context, taskID uuid.UUID) ([]ChecklistItem, error) {
	var items []ChecklistItem
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("position ASC, created_at ASC").Find(&items).Error
	return items, err
}
