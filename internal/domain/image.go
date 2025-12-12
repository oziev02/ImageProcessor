package domain

import (
	"errors"
	"time"
)

// ProcessingStatus represents the status of image processing
type ProcessingStatus string

const (
	StatusPending    ProcessingStatus = "pending"
	StatusProcessing ProcessingStatus = "processing"
	StatusCompleted  ProcessingStatus = "completed"
	StatusFailed     ProcessingStatus = "failed"
)

// ImageFormat represents supported image formats
type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatPNG  ImageFormat = "png"
	FormatGIF  ImageFormat = "gif"
)

// Image represents a processed image entity
type Image struct {
	ID              string           `json:"id"`
	OriginalPath    string           `json:"original_path"`
	ProcessedPath   string           `json:"processed_path"`
	ThumbnailPath   string           `json:"thumbnail_path"`
	Status          ProcessingStatus `json:"status"`
	Format          ImageFormat      `json:"format"`
	OriginalWidth   int              `json:"original_width"`
	OriginalHeight  int              `json:"original_height"`
	ProcessedWidth  int              `json:"processed_width"`
	ProcessedHeight int              `json:"processed_height"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// ProcessingTask represents a task for background processing
type ProcessingTask struct {
	ImageID   string      `json:"image_id"`
	ImagePath string      `json:"image_path"`
	Format    ImageFormat `json:"format"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
}

// Validate validates image invariants
func (i *Image) Validate() error {
	if i.ID == "" {
		return ErrInvalidImageID
	}
	if i.OriginalPath == "" {
		return ErrInvalidImagePath
	}
	return nil
}

// Domain errors
var (
	ErrInvalidImageID   = errors.New("invalid image id")
	ErrInvalidImagePath = errors.New("invalid image path")
	ErrImageNotFound    = errors.New("image not found")
	ErrInvalidFormat    = errors.New("invalid image format")
)
