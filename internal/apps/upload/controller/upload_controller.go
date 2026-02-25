package controller

import (
	"strconv"
	"strings"

	"crisisecho/internal/apps/upload/service"

	"github.com/gofiber/fiber/v2"
)

type UploadController struct {
	service service.UploadService
}

func NewUploadController(s service.UploadService) *UploadController {
	return &UploadController{service: s}
}

func (ctrl *UploadController) RegisterRoutes(router fiber.Router) {
	router.Get("/presign", ctrl.GeneratePresignedURL)
	router.Get("/retrieve", ctrl.GenerateRetrievalURL)
	router.Post("/direct", ctrl.HandleDirectUpload)
	router.Get("/verify", ctrl.VerifyUpload)
}

// GET /api/upload/presign?type=image/jpeg&name=photo.jpg&folder=crisis-images
func (ctrl *UploadController) GeneratePresignedURL(c *fiber.Ctx) error {
	fileType := strings.TrimSpace(c.Query("type"))
	fileName := strings.TrimSpace(c.Query("name"))
	folder := strings.TrimSpace(c.Query("folder", "uploads"))

	if fileType == "" || fileName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameters",
			"message": "Both 'type' and 'name' query parameters are required",
		})
	}

	validTypes := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true,
		"application/pdf": true, "text/plain": true,
	}
	if !validTypes[fileType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":      "invalid_file_type",
			"message":    "Unsupported file type",
			"validTypes": getKeys(validTypes),
		})
	}

	response, err := ctrl.service.GeneratePresignedURL(fileType, fileName, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "upload_url_generation_failed",
			"message": "Failed to generate upload URL",
			"details": err.Error(),
		})
	}

	return c.JSON(response)
}

// GET /api/upload/retrieve?path=uploads/image/jpeg/12345_photo.jpg&expiry=900
func (ctrl *UploadController) GenerateRetrievalURL(c *fiber.Ctx) error {
	filePath := strings.TrimSpace(c.Query("path"))
	expiry := strings.TrimSpace(c.Query("expiry", "900"))

	if filePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "'path' query parameter is required",
		})
	}

	expirySeconds, err := strconv.Atoi(expiry)
	if err != nil || expirySeconds <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_expiry",
			"message": "Expiry must be a positive integer (seconds)",
		})
	}

	if expirySeconds > 604800 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "expiry_too_long",
			"message": "Maximum expiry time is 7 days (604800 seconds)",
		})
	}

	response, err := ctrl.service.GenerateRetrievalURL(filePath, expirySeconds)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "retrieval_url_generation_failed",
			"message": "Failed to generate retrieval URL",
			"details": err.Error(),
		})
	}

	return c.JSON(response)
}

// POST /api/upload/direct  (multipart form, field: "file", optional form field: "folder")
func (ctrl *UploadController) HandleDirectUpload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_file",
			"message": "No file uploaded",
		})
	}

	folder := strings.TrimSpace(c.FormValue("folder", "uploads"))

	response, err := ctrl.service.DirectUpload(file, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "upload_failed",
			"message": "Failed to upload file",
			"details": err.Error(),
		})
	}

	return c.JSON(response)
}

// GET /api/upload/verify?path=uploads/image/jpeg/12345_photo.jpg
func (ctrl *UploadController) VerifyUpload(c *fiber.Ctx) error {
	filePath := strings.TrimSpace(c.Query("path"))
	if filePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "'path' query parameter is required",
		})
	}

	exists, err := ctrl.service.VerifyUploadCompletion(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "verification_failed",
			"message": "Failed to verify upload status",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"exists": exists,
		"path":   filePath,
	})
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
