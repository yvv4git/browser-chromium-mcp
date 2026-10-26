package fs

import (
	"os"
	"path/filepath"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Save(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

var _ interface {
	Save(path string, data []byte) error
} = (*Adapter)(nil)
