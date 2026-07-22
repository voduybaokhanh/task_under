// Package storage issues presigned S3 URLs so clients upload straight to the
// bucket. Bytes never pass through the backend, and AWS credentials never
// reach the device.
package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// PresignTTL is how long an upload URL stays valid — long enough to pick a
// photo and upload it, short enough to be useless if leaked.
const PresignTTL = 15 * time.Minute

// Uploader hands out presigned PUT URLs for a single bucket.
type Uploader struct {
	presign   *s3.PresignClient
	bucket    string
	publicURL string
	prefix    string
}

// NewUploader builds an Uploader from the environment. It returns nil when
// AWS_BUCKET_NAME is unset, so the API can report the feature as unconfigured
// instead of failing at startup.
//
// Env: AWS_BUCKET_NAME, AWS_REGION, AWS_ENDPOINT_URL (for S3-compatible
// services such as MinIO), AWS_PUBLIC_URL (public read base, defaults to the
// virtual-hosted S3 URL) plus the usual AWS credential variables.
func NewUploader(ctx context.Context) *Uploader {
	bucket := os.Getenv("AWS_BUCKET_NAME")
	if bucket == "" {
		log.Println("AWS_BUCKET_NAME not set; image upload disabled")
		return nil
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		log.Printf("AWS config unavailable (%v); image upload disabled", err)
		return nil
	}

	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			// MinIO and friends serve buckets as a path, not a subdomain.
			o.UsePathStyle = true
		}
	})

	publicURL := os.Getenv("AWS_PUBLIC_URL")
	if publicURL == "" {
		if endpoint != "" {
			publicURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(endpoint, "/"), bucket)
		} else {
			publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
		}
	}

	log.Printf("Image upload enabled (bucket %s)", bucket)
	return &Uploader{
		presign:   s3.NewPresignClient(client),
		bucket:    bucket,
		publicURL: strings.TrimSuffix(publicURL, "/"),
		prefix:    "task-images",
	}
}

// Upload describes where the client should PUT the file and where the world
// will read it afterwards.
type Upload struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	ExpiresIn int    `json:"expires_in"`
}

// Presign returns an upload URL for one object. The key is generated here so
// a client cannot overwrite someone else's image by picking its name.
func (u *Uploader) Presign(ctx context.Context, filename, contentType string) (*Upload, error) {
	return u.presignWithTTL(ctx, filename, contentType, PresignTTL)
}

func (u *Uploader) presignWithTTL(ctx context.Context, filename, contentType string, ttl time.Duration) (*Upload, error) {
	key := fmt.Sprintf("%s/%s%s", u.prefix, uuid.NewString(), strings.ToLower(path.Ext(filename)))

	req, err := u.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}

	return &Upload{
		UploadURL: req.URL,
		PublicURL: fmt.Sprintf("%s/%s", u.publicURL, key),
		ExpiresIn: int(ttl.Seconds()),
	}, nil
}
