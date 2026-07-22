package storage

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withMinIO points the uploader at a local S3-compatible server. Skipped
// unless one is running:
//
//	docker run -d -p 9000:9000 -e MINIO_ROOT_USER=minioadmin \
//	  -e MINIO_ROOT_PASSWORD=minioadmin minio/minio server /data
//	S3_TEST_ENDPOINT=http://localhost:9000 go test ./internal/storage/
func withMinIO(t *testing.T) *Uploader {
	t.Helper()

	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set")
	}

	t.Setenv("AWS_BUCKET_NAME", os.Getenv("S3_TEST_BUCKET"))
	t.Setenv("AWS_ENDPOINT_URL", endpoint)
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "minioadmin")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "minioadmin")

	uploader := NewUploader(context.Background())
	require.NotNil(t, uploader, "uploader should be configured")
	return uploader
}

// The whole point of presigning: a client with no credentials can upload, and
// the result is readable at the public URL.
func TestPresignedURLAcceptsAnUpload(t *testing.T) {
	uploader := withMinIO(t)

	upload, err := uploader.Presign(context.Background(), "photo.PNG", "image/png")
	require.NoError(t, err)

	assert.Contains(t, upload.UploadURL, "task-images/", "key should live under the images prefix")
	assert.True(t, strings.HasSuffix(strings.Split(upload.PublicURL, "?")[0], ".png"),
		"extension should be preserved and lowercased: %s", upload.PublicURL)
	assert.Equal(t, int(PresignTTL.Seconds()), upload.ExpiresIn)

	body := strings.NewReader("not really a png, but bytes are bytes")
	req, err := http.NewRequest(http.MethodPut, upload.UploadURL, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "image/png")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	require.Less(t, resp.StatusCode, 300, "upload rejected: %s", raw)

	read, err := http.Get(upload.PublicURL)
	require.NoError(t, err)
	defer read.Body.Close()
	stored, _ := io.ReadAll(read.Body)
	assert.Equal(t, "not really a png, but bytes are bytes", string(stored))
}

// Two presigns must never collide, even for the same filename — otherwise one
// user could overwrite another's image.
func TestPresignGeneratesADistinctKeyEachTime(t *testing.T) {
	uploader := withMinIO(t)

	first, err := uploader.Presign(context.Background(), "photo.png", "image/png")
	require.NoError(t, err)
	second, err := uploader.Presign(context.Background(), "photo.png", "image/png")
	require.NoError(t, err)

	assert.NotEqual(t, first.PublicURL, second.PublicURL)
}

// A URL that has outlived its window must be refused by the storage service.
func TestExpiredURLIsRejected(t *testing.T) {
	uploader := withMinIO(t)

	upload, err := uploader.presignWithTTL(context.Background(), "photo.png", "image/png", time.Second)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	req, err := http.NewRequest(http.MethodPut, upload.UploadURL, strings.NewReader("late"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "image/png")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.GreaterOrEqual(t, resp.StatusCode, 400, "an expired URL should not be accepted")
}

func TestUploaderIsNilWithoutABucket(t *testing.T) {
	t.Setenv("AWS_BUCKET_NAME", "")
	assert.Nil(t, NewUploader(context.Background()))
}
