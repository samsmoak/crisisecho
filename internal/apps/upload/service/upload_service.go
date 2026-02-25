package service

import (
	"fmt"
	"mime/multipart"
	"time"

	"crisisecho/internal/apps/upload/model"
	"crisisecho/internal/apps/upload/repository"
)

type UploadService interface {
	GeneratePresignedURL(fileType, fileName, folder string) (*model.PresignResponse, error)
	GenerateRetrievalURL(filePath string, expirySeconds int) (*model.RetrieveResponse, error)
	VerifyUploadCompletion(filePath string) (bool, error)
	DirectUpload(file *multipart.FileHeader, folder string) (*model.UploadResponse, error)
}

type uploadService struct {
	repo repository.UploadRepository
}

func NewUploadService(repo repository.UploadRepository) UploadService {
	return &uploadService{repo: repo}
}

func (s *uploadService) GeneratePresignedURL(fileType, fileName, folder string) (*model.PresignResponse, error) {
	uploadURL, filePath, publicURL, err := s.repo.GeneratePresignedURL(fileType, fileName, folder)
	if err != nil {
		return nil, fmt.Errorf("UploadService.GeneratePresignedURL: %w", err)
	}

	return &model.PresignResponse{
		URL:          uploadURL,
		PublicURL:    publicURL,
		FileName:     fileName,
		FileType:     fileType,
		FilePath:     filePath,
		UploadMethod: "PUT",
		ExpiresIn:    900, // 15 minutes
	}, nil
}

func (s *uploadService) GenerateRetrievalURL(filePath string, expirySeconds int) (*model.RetrieveResponse, error) {
	url, err := s.repo.GenerateRetrievalURL(filePath, time.Duration(expirySeconds)*time.Second)
	if err != nil {
		return nil, fmt.Errorf("UploadService.GenerateRetrievalURL: %w", err)
	}

	return &model.RetrieveResponse{
		URL:       url,
		ExpiresIn: expirySeconds,
	}, nil
}

func (s *uploadService) VerifyUploadCompletion(filePath string) (bool, error) {
	return s.repo.VerifyFileExists(filePath)
}

func (s *uploadService) DirectUpload(file *multipart.FileHeader, folder string) (*model.UploadResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("UploadService.DirectUpload: open file: %w", err)
	}
	defer src.Close()

	fileType := file.Header.Get("Content-Type")
	if fileType == "" {
		fileType = "application/octet-stream"
	}

	publicURL, err := s.repo.UploadFile(src, file.Filename, fileType, folder)
	if err != nil {
		return nil, fmt.Errorf("UploadService.DirectUpload: %w", err)
	}

	return &model.UploadResponse{
		PublicURL: publicURL,
		FileName:  file.Filename,
		FileType:  fileType,
		Size:      file.Size,
	}, nil
}
