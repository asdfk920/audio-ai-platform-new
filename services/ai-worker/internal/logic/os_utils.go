package logic

import (
	"os"
)

func osMkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func osCreate(name string) (*os.File, error) {
	return os.Create(name)
}
