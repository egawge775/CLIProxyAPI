package management

import (
	"archive/zip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// batchAuthFilesArchiveName is the download name of the packaged auth files.
const batchAuthFilesArchiveName = "auth-files.zip"

// DownloadAuthFilesArchive packages multiple auth files into one ZIP archive.
// It accepts the same "name" query parameters as batch delete, so a client can
// fetch the whole selection in a single response instead of downloading every
// file one by one.
func (h *Handler) DownloadAuthFilesArchive(c *gin.Context) {
	if h == nil || h.cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler not initialized"})
		return
	}

	names := uniqueAuthFileNames(c.QueryArray("name"))
	if len(names) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	for _, name := range names {
		if isUnsafeAuthFileName(name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid name"})
			return
		}
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name must end with .json"})
			return
		}
	}

	// Read every payload before writing the response so a missing file still
	// produces a JSON error instead of a truncated archive.
	contents := make(map[string][]byte, len(names))
	ordered := make([]string, 0, len(names))
	for _, name := range names {
		data, errRead := os.ReadFile(filepath.Join(h.cfg.AuthDir, name))
		if errRead != nil {
			if os.IsNotExist(errRead) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("file not found: %s", name)})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read file %s: %v", name, errRead)})
			}
			return
		}
		contents[name] = data
		ordered = append(ordered, name)
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", batchAuthFilesArchiveName, batchAuthFilesArchiveName))
	c.Header("Content-Type", "application/zip")
	c.Header("Cache-Control", "no-store")

	// The archive streams straight to the response; a late failure can only
	// truncate it, so report the cause through the request log.
	writer := zip.NewWriter(c.Writer)
	for _, name := range ordered {
		entry, errEntry := writer.CreateHeader(&zip.FileHeader{
			Name:     name,
			Method:   zip.Deflate,
			Modified: time.Now(),
		})
		if errEntry != nil {
			logBatchArchiveFailure(c, name, errEntry)
			break
		}
		if _, errWrite := entry.Write(contents[name]); errWrite != nil {
			logBatchArchiveFailure(c, name, errWrite)
			break
		}
	}
	if errClose := writer.Close(); errClose != nil {
		logBatchArchiveFailure(c, "", errClose)
	}
}

func logBatchArchiveFailure(c *gin.Context, name string, err error) {
	entry := log.WithError(err).WithField("archive", batchAuthFilesArchiveName)
	if name != "" {
		entry = entry.WithField("name", name)
	}
	if c != nil && c.Request != nil {
		entry = entry.WithField("client", c.ClientIP())
	}
	entry.Error("management: auth file archive download failed")
}
