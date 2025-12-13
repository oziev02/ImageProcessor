package service

import (
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/oziev02/ImageProcessor/internal/config"
	"github.com/oziev02/ImageProcessor/internal/domain"
	"github.com/oziev02/ImageProcessor/internal/repo"
	kafkatransport "github.com/oziev02/ImageProcessor/internal/transport/kafka"
)

type ImageService interface {
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*domain.Image, error)
	GetByID(ctx context.Context, id string) (*domain.Image, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*domain.Image, error)
}

type imageService struct {
	imageRepo   repo.ImageRepository
	storageRepo repo.StorageRepository
	producer    kafkatransport.Producer
	cfg         *config.Config
}

func NewImageService(
	imageRepo repo.ImageRepository,
	storageRepo repo.StorageRepository,
	producer kafkatransport.Producer,
	cfg *config.Config,
) ImageService {
	return &imageService{
		imageRepo:   imageRepo,
		storageRepo: storageRepo,
		producer:    producer,
		cfg:         cfg,
	}
}

func (s *imageService) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*domain.Image, error) {
	// Validate file size
	if header.Size > s.cfg.Image.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size")
	}

	// Generate ID
	id := repo.GenerateID()

	// Determine format
	ext := strings.ToLower(filepath.Ext(header.Filename))
	format, err := parseFormat(ext)
	if err != nil {
		return nil, fmt.Errorf("unsupported format: %w", err)
	}

	// Save original file
	originalPath := filepath.Join("original", id+ext)
	if err := s.storageRepo.Save(ctx, originalPath, file); err != nil {
		return nil, fmt.Errorf("failed to save original file: %w", err)
	}

	// Read image dimensions
	file.Seek(0, 0)
	img, _, err := decodeImageForDimensions(file, format)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create image record
	now := time.Now()
	image := &domain.Image{
		ID:              id,
		OriginalPath:    originalPath,
		ProcessedPath:   "",
		ThumbnailPath:   "",
		Status:          domain.StatusPending,
		Format:          format,
		OriginalWidth:   width,
		OriginalHeight:  height,
		ProcessedWidth:  0,
		ProcessedHeight: 0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := image.Validate(); err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	// Save to database
	if err := s.imageRepo.Create(ctx, image); err != nil {
		return nil, fmt.Errorf("failed to create image record: %w", err)
	}

	// Send to Kafka for processing
	task := &domain.ProcessingTask{
		ImageID:   id,
		ImagePath: originalPath,
		Format:    format,
		Width:     width,
		Height:    height,
	}
	if err := s.producer.SendTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to send processing task: %w", err)
	}

	return image, nil
}

func (s *imageService) GetByID(ctx context.Context, id string) (*domain.Image, error) {
	return s.imageRepo.GetByID(ctx, id)
}

func (s *imageService) Delete(ctx context.Context, id string) error {
	img, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete files
	if img.OriginalPath != "" {
		_ = s.storageRepo.Delete(ctx, img.OriginalPath)
	}
	if img.ProcessedPath != "" {
		_ = s.storageRepo.Delete(ctx, img.ProcessedPath)
	}
	if img.ThumbnailPath != "" {
		_ = s.storageRepo.Delete(ctx, img.ThumbnailPath)
	}

	// Delete from database
	return s.imageRepo.Delete(ctx, id)
}

func (s *imageService) List(ctx context.Context, limit, offset int) ([]*domain.Image, error) {
	return s.imageRepo.List(ctx, limit, offset)
}

func parseFormat(ext string) (domain.ImageFormat, error) {
	switch ext {
	case ".jpg", ".jpeg":
		return domain.FormatJPEG, nil
	case ".png":
		return domain.FormatPNG, nil
	case ".gif":
		return domain.FormatGIF, nil
	default:
		return "", domain.ErrInvalidFormat
	}
}

func decodeImageForDimensions(r io.Reader, format domain.ImageFormat) (image.Image, string, error) {
	switch format {
	case domain.FormatJPEG:
		img, err := jpeg.Decode(r)
		return img, "jpeg", err
	case domain.FormatPNG:
		img, err := png.Decode(r)
		return img, "png", err
	case domain.FormatGIF:
		img, err := gif.Decode(r)
		return img, "gif", err
	default:
		return nil, "", domain.ErrInvalidFormat
	}
}
