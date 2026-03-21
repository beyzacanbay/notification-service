package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beyzacanbay/notification-service/internal/model"
)

type postgresTemplateRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresTemplateRepo(pool *pgxpool.Pool) TemplateRepository {
	return &postgresTemplateRepo{pool: pool}
}

func (r *postgresTemplateRepo) Create(ctx context.Context, t *model.Template) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO templates (id, name, channel, content_template, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.Name, t.Channel, t.ContentTemplate, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *postgresTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	t := &model.Template{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, channel, content_template, created_at, updated_at
		FROM templates WHERE id = $1`, id).Scan(
		&t.ID, &t.Name, &t.Channel, &t.ContentTemplate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *postgresTemplateRepo) List(ctx context.Context) ([]*model.Template, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, channel, content_template, created_at, updated_at
		FROM templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*model.Template
	for rows.Next() {
		t := &model.Template{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Channel, &t.ContentTemplate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *postgresTemplateRepo) Update(ctx context.Context, t *model.Template) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE templates SET name = $1, channel = $2, content_template = $3
		WHERE id = $4`,
		t.Name, t.Channel, t.ContentTemplate, t.ID)
	return err
}

func (r *postgresTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM templates WHERE id = $1", id)
	return err
}
