package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage interface defines common operations for storage backends
type Storage interface {
	Setup() error
	Save(key string, bucket string, data []byte) error
	Read(key string, bucket string) ([]byte, error)
	ReadStream(key string, bucket string) (io.ReadSeekCloser, error)
	Remove(key string, bucket string) error
	Exists(key string, bucket string) bool
	GetPath(key string, bucket string) string
}

// FileStorage implements Storage interface using the local local
type FileStorage struct {
	BasePath string
	Logger   *Logger
}

func NewStorage(config *Config, logger *Logger) Storage {
	switch config.Storage.Type {
	case "local":
		return &FileStorage{
			BasePath: config.Storage.Path,
			Logger:   logger,
		}
	default:
		return nil
	}
}

func (fs *FileStorage) Setup() error {
	return os.MkdirAll(fs.BasePath, 0755)
}

func (fs *FileStorage) getFullPath(key, bucket string) string {
	return filepath.Join(fs.BasePath, bucket, key)
}

func (fs *FileStorage) GetPath(key, bucket string) string {
	return fs.getFullPath(key, bucket)
}

func (fs *FileStorage) Save(key, bucket string, data []byte) error {
	fullPath := fs.getFullPath(key, bucket)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

func (fs *FileStorage) Read(key, bucket string) ([]byte, error) {
	fullPath := fs.getFullPath(key, bucket)
	return os.ReadFile(fullPath)
}

func (fs *FileStorage) ReadStream(key, bucket string) (io.ReadSeekCloser, error) {
	fullPath := fs.getFullPath(key, bucket)
	return os.Open(fullPath)
}

func (fs *FileStorage) Remove(key, bucket string) error {
	fullPath := fs.getFullPath(key, bucket)
	return os.Remove(fullPath)
}

func (fs *FileStorage) Exists(key, bucket string) bool {
	fullPath := fs.getFullPath(key, bucket)
	_, err := os.Stat(fullPath)
	return err == nil
}
