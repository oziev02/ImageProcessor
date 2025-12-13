package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oziev02/ImageProcessor/internal/domain"
)

type ImageRepository interface {
	Create(ctx context.Context, img *domain.Image) error
	GetByID(ctx context.Context, id string) (*domain.Image, error)
	Update(ctx context.Context, img *domain.Image) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*domain.Image, error)
}

type imageRepo struct {
	db *pgxpool.Pool
}

func NewImageRepository(db *pgxpool.Pool) ImageRepository {
	return &imageRepo{db: db}
}

func (r *imageRepo) Create(ctx context.Context, img *domain.Image) error {
	query := `
		INSERT INTO images (id, original_path, processed_path, thumbnail_path, status, format, 
			original_width, original_height, processed_width, processed_height, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(ctx, query,
		img.ID, img.OriginalPath, img.ProcessedPath, img.ThumbnailPath, img.Status,
		img.Format, img.OriginalWidth, img.OriginalHeight, img.ProcessedWidth, img.ProcessedHeight,
		img.CreatedAt, img.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}
	return nil
}

func (r *imageRepo) GetByID(ctx context.Context, id string) (*domain.Image, error) {
	query := `
		SELECT id, original_path, processed_path, thumbnail_path, status, format,
			original_width, original_height, processed_width, processed_height, created_at, updated_at
		FROM images
		WHERE id = $1
	`
	var img domain.Image
	err := r.db.QueryRow(ctx, query, id).Scan(
		&img.ID, &img.OriginalPath, &img.ProcessedPath, &img.ThumbnailPath, &img.Status,
		&img.Format, &img.OriginalWidth, &img.OriginalHeight, &img.ProcessedWidth, &img.ProcessedHeight,
		&img.CreatedAt, &img.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrImageNotFound
		}
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	return &img, nil
}

func (r *imageRepo) Update(ctx context.Context, img *domain.Image) error {
	query := `
		UPDATE images
		SET processed_path = $2, thumbnail_path = $3, status = $4,
			processed_width = $5, processed_height = $6, updated_at = $7
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		img.ID, img.ProcessedPath, img.ThumbnailPath, img.Status,
		img.ProcessedWidth, img.ProcessedHeight, img.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update image: %w", err)
	}
	return nil
}

func (r *imageRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM images WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	return nil
}

func (r *imageRepo) List(ctx context.Context, limit, offset int) ([]*domain.Image, error) {
	query := `
		SELECT id, original_path, processed_path, thumbnail_path, status, format,
			original_width, original_height, processed_width, processed_height, created_at, updated_at
		FROM images
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	defer rows.Close()

	var images []*domain.Image
	for rows.Next() {
		var img domain.Image
		if err := rows.Scan(
			&img.ID, &img.OriginalPath, &img.ProcessedPath, &img.ThumbnailPath, &img.Status,
			&img.Format, &img.OriginalWidth, &img.OriginalHeight, &img.ProcessedWidth, &img.ProcessedHeight,
			&img.CreatedAt, &img.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		images = append(images, &img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate images: %w", err)
	}

	return images, nil
}

func GenerateID() string {
	return uuid.New().String()
}
