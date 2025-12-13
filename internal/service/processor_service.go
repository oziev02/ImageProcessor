package service

import (
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/nfnt/resize"
	"github.com/oziev02/ImageProcessor/internal/config"
	"github.com/oziev02/ImageProcessor/internal/domain"
	"github.com/oziev02/ImageProcessor/internal/repo"
)

type ProcessorService interface {
	ProcessImage(ctx context.Context, task *domain.ProcessingTask) error
}

type processorService struct {
	imageRepo   repo.ImageRepository
	storageRepo repo.StorageRepository
	cfg         *config.Config
}

func NewProcessorService(
	imageRepo repo.ImageRepository,
	storageRepo repo.StorageRepository,
	cfg *config.Config,
) ProcessorService {
	return &processorService{
		imageRepo:   imageRepo,
		storageRepo: storageRepo,
		cfg:         cfg,
	}
}

func (s *processorService) ProcessImage(ctx context.Context, task *domain.ProcessingTask) error {
	// Get image record
	img, err := s.imageRepo.GetByID(ctx, task.ImageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	// Update status to processing
	img.Status = domain.StatusProcessing
	img.UpdatedAt = time.Now()
	if err := s.imageRepo.Update(ctx, img); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Read original image
	originalReader, err := s.storageRepo.Read(ctx, task.ImagePath)
	if err != nil {
		img.Status = domain.StatusFailed
		img.UpdatedAt = time.Now()
		_ = s.imageRepo.Update(ctx, img)
		return fmt.Errorf("failed to read original image: %w", err)
	}
	defer originalReader.Close()

	// Decode image
	originalImg, _, err := decodeImage(originalReader, task.Format)
	if err != nil {
		img.Status = domain.StatusFailed
		img.UpdatedAt = time.Now()
		_ = s.imageRepo.Update(ctx, img)
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Process resized image
	processedImg := resize.Resize(
		uint(s.cfg.Image.ProcessedWidth),
		uint(s.cfg.Image.ProcessedHeight),
		originalImg,
		resize.Lanczos3,
	)

	// Process thumbnail
	thumbnailImg := resize.Resize(
		uint(s.cfg.Image.ThumbnailWidth),
		uint(s.cfg.Image.ThumbnailHeight),
		originalImg,
		resize.Lanczos3,
	)

	// Save processed image
	processedPath := filepath.Join("processed", task.ImageID+getExtension(task.Format))
	if err := s.saveImage(ctx, processedPath, processedImg, task.Format); err != nil {
		img.Status = domain.StatusFailed
		img.UpdatedAt = time.Now()
		_ = s.imageRepo.Update(ctx, img)
		return fmt.Errorf("failed to save processed image: %w", err)
	}

	// Save thumbnail
	thumbnailPath := filepath.Join("thumbnail", task.ImageID+getExtension(task.Format))
	if err := s.saveImage(ctx, thumbnailPath, thumbnailImg, task.Format); err != nil {
		img.Status = domain.StatusFailed
		img.UpdatedAt = time.Now()
		_ = s.imageRepo.Update(ctx, img)
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	// Add watermark if enabled
	if s.cfg.Image.WatermarkEnabled && s.cfg.Image.WatermarkPath != "" {
		// For simplicity, we'll skip watermark for now
		// In production, you'd overlay the watermark here
	}

	// Update image record
	img.ProcessedPath = processedPath
	img.ThumbnailPath = thumbnailPath
	img.Status = domain.StatusCompleted
	bounds := processedImg.Bounds()
	img.ProcessedWidth = bounds.Dx()
	img.ProcessedHeight = bounds.Dy()
	img.UpdatedAt = time.Now()

	if err := s.imageRepo.Update(ctx, img); err != nil {
		return fmt.Errorf("failed to update image record: %w", err)
	}

	return nil
}

func (s *processorService) saveImage(ctx context.Context, path string, img image.Image, format domain.ImageFormat) error {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "img-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Encode image
	switch format {
	case domain.FormatJPEG:
		if err := jpeg.Encode(tmpFile, img, &jpeg.Options{Quality: 90}); err != nil {
			return fmt.Errorf("failed to encode JPEG: %w", err)
		}
	case domain.FormatPNG:
		if err := png.Encode(tmpFile, img); err != nil {
			return fmt.Errorf("failed to encode PNG: %w", err)
		}
	case domain.FormatGIF:
		if err := gif.Encode(tmpFile, img, &gif.Options{}); err != nil {
			return fmt.Errorf("failed to encode GIF: %w", err)
		}
	default:
		return domain.ErrInvalidFormat
	}

	// Read temp file and save to storage
	tmpFile.Seek(0, 0)
	return s.storageRepo.Save(ctx, path, tmpFile)
}

func decodeImage(r io.Reader, format domain.ImageFormat) (image.Image, string, error) {
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

func getExtension(format domain.ImageFormat) string {
	switch format {
	case domain.FormatJPEG:
		return ".jpg"
	case domain.FormatPNG:
		return ".png"
	case domain.FormatGIF:
		return ".gif"
	default:
		return ".jpg"
	}
}
