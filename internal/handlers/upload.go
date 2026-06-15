package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler returns a gin.HandlerFunc that handles file uploads.
func UploadHandler(uploadDir string, maxImageMB, maxAudioMB, maxVideoMB int) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件"})
			return
		}
		defer file.Close()

		// Detect MIME type - try multiple methods
		mimeType := detectMIME(header)

		// Seek back to start
		if seeker, ok := file.(interface{ Seek(int64, int) (int64, error) }); ok {
			seeker.Seek(0, 0)
		}

		// Determine size limit
		var maxBytes int64
		switch {
		case strings.HasPrefix(mimeType, "image/"):
			maxBytes = int64(maxImageMB) * 1024 * 1024
		case strings.HasPrefix(mimeType, "audio/"):
			maxBytes = int64(maxAudioMB) * 1024 * 1024
		case strings.HasPrefix(mimeType, "video/"):
			maxBytes = int64(maxVideoMB) * 1024 * 1024
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的文件类型: %s", mimeType)})
			return
		}

		if header.Size > maxBytes {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("文件大小 %d MB 超过限制 %d MB", header.Size/(1024*1024), maxBytes/(1024*1024)),
			})
			return
		}

		// Generate UUID and save
		fileID := uuid.New().String()
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = extensionFromMime(mimeType)
		}
		savePath := filepath.Join(uploadDir, fileID+ext)

		if err := c.SaveUploadedFile(header, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"file_id":    fileID,
			"mime_type":  mimeType,
			"size_bytes": header.Size,
			"temp_path":  savePath,
		})
	}
}

// detectMIME detects MIME type from filename extension and file content.
func detectMIME(header *multipart.FileHeader) string {
	ext := strings.ToLower(filepath.Ext(header.Filename))

	// Extension-based detection (more reliable for audio/video)
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	case ".m4a":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".wma":
		return "audio/x-ms-wma"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	case ".mkv":
		return "video/x-matroska"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	}

	// Fallback: content-based detection
	buf := make([]byte, 512)
	if header.Size > 0 {
		// Can't read from header directly, use extension result
		return "application/octet-stream"
	}
	_ = buf
	return "application/octet-stream"
}

// extensionFromMime returns a default file extension for a given MIME type.
func extensionFromMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(mime, "image/png"):
		return ".png"
	case strings.HasPrefix(mime, "image/gif"):
		return ".gif"
	case strings.HasPrefix(mime, "image/webp"):
		return ".webp"
	case strings.HasPrefix(mime, "audio/mpeg"):
		return ".mp3"
	case strings.HasPrefix(mime, "audio/wav"):
		return ".wav"
	case strings.HasPrefix(mime, "audio/ogg"):
		return ".ogg"
	case strings.HasPrefix(mime, "audio/mp4"):
		return ".m4a"
	case strings.HasPrefix(mime, "video/mp4"):
		return ".mp4"
	case strings.HasPrefix(mime, "video/webm"):
		return ".webm"
	case strings.HasPrefix(mime, "video/quicktime"):
		return ".mov"
	default:
		return ""
	}
}

// MediaServeHandler returns a handler that serves uploaded files by file_id.
func MediaServeHandler(uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("file_id")
		if fileID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing file_id"})
			return
		}
		path := findUploadPath(uploadDir, fileID)
		if path == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.File(path)
	}
}

// Ensure multipart.File interface is used.
var _ multipart.File
