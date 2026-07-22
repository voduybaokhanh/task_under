package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/task-underground/backend/internal/storage"
)

// allowedContentTypes keeps the bucket to images: the presigned URL pins the
// content type, so this is what the client is allowed to upload.
var allowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

type UploadHandler struct {
	uploader *storage.Uploader
}

func NewUploadHandler(uploader *storage.Uploader) *UploadHandler {
	return &UploadHandler{uploader: uploader}
}

type presignRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
}

// Presign returns a short-lived URL the client PUTs the image to directly, so
// the file never travels through this server and AWS credentials never leave it.
func (h *UploadHandler) Presign(c *gin.Context) {
	if h.uploader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "image upload is not configured"})
		return
	}

	var req presignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contentType := strings.ToLower(strings.TrimSpace(req.ContentType))
	if !allowedContentTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content_type must be image/jpeg, image/png or image/webp"})
		return
	}

	upload, err := h.uploader.Presign(c.Request.Context(), req.Filename, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, upload)
}
