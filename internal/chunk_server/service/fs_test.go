package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateUploadDir(t *testing.T) {
	uploadDir := "./test_uploads"
	defer os.RemoveAll(uploadDir)

	err := CreateUploadDir(uploadDir)
	assert.NoError(t, err)

	// Check if directory was created
	info, err := os.Stat(uploadDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDeleteFile(t *testing.T) {
	uploadDir := "./test_uploads"
	fileName := "testfile.txt"
	filePath := filepath.Join(uploadDir, fileName)

	// Setup: create directory and file
	os.MkdirAll(uploadDir, os.ModePerm)
	f, _ := os.Create(filePath)
	f.Close()
	defer os.RemoveAll(uploadDir)

	// Test deleting existing file
	err := DeleteFile(uploadDir, fileName)
	assert.NoError(t, err)

	// Check if file was deleted
	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))

	// Test deleting non-existing file
	err = DeleteFile(uploadDir, fileName)
	assert.Error(t, err)
	assert.Equal(t, "file not found", err.Error())
}

func TestFileExists(t *testing.T) {
	uploadDir := "./test_uploads"
	fileName := "testfile.txt"
	filePath := filepath.Join(uploadDir, fileName)

	// Setup: create directory and file
	os.MkdirAll(uploadDir, os.ModePerm)
	f, _ := os.Create(filePath)
	f.Close()
	defer os.RemoveAll(uploadDir)

	// Test if file exists
	exists, err := fileExists(uploadDir, fileName)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test if non-existing file
	exists, err = fileExists(uploadDir, "nonexistent.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestOpenFile(t *testing.T) {
	uploadDir := "./test_uploads"
	fileName := "testfile.txt"
	filePath := filepath.Join(uploadDir, fileName)

	// Setup: create directory and file
	os.MkdirAll(uploadDir, os.ModePerm)
	f, _ := os.Create(filePath)
	f.Close()
	defer os.RemoveAll(uploadDir)

	// Test opening existing file
	file, err := openFile(uploadDir, fileName)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	file.Close()

	// Test opening non-existing file
	file, err = openFile(uploadDir, "nonexistent.txt")
	assert.Error(t, err)
	assert.Nil(t, file)
}
