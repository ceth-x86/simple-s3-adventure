package service

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateUploadDir(uploadDir string) error {
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create upload directory")
	}
	return nil
}

func DeleteFile(uploadDir string, uuid string) error {
	filePath := filepath.Join(uploadDir, uuid)
	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found")
		} else {
			return fmt.Errorf("failed to delete file on server")
		}
	}
	return nil
}

func fileExists(uploadDir string, uuid string) (bool, error) {
	filePath := filepath.Join(uploadDir, uuid)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func openFile(uploadDir string, uuid string) (*os.File, error) {
	filePath := filepath.Join(uploadDir, uuid)
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}
