package service

import (
	"os"

	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type FileService struct {
	annotation.Injectable[inject.Default]
	file *os.File
}

func NewFileService(file *os.File) *FileService {
	return &FileService{file: file}
}
