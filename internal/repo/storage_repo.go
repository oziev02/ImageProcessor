package repo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type StorageRepository interface {
	Save(ctx context.Context, path string, data io.Reader) error
	Read(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
}

type storageRepo struct {
	basePath string
}

func NewStorageRepository(basePath string) StorageRepository {
	return &storageRepo{basePath: basePath}
}

func (r *storageRepo) Save(ctx context.Context, path string, data io.Reader) error {
	fullPath := filepath.Join(r.basePath, path)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (r *storageRepo) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(r.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %w", err)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

func (r *storageRepo) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(r.basePath, path)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (r *storageRepo) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(r.basePath, path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	return true, nil
}
