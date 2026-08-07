package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage wraps a MinIO client bound to one bucket. presignedClient is a
// second client pointed at PublicEndpoint — the host a browser can reach —
// used only for signing download URLs; every other operation (put/remove)
// goes through the internal client, which is what actually needs network
// access to MinIO (e.g. the "minio" hostname on the Docker Compose
// network, unreachable from outside it).
type Storage struct {
	client          *minio.Client
	presignedClient *minio.Client
	bucket          string
}

// Connect opens a MinIO client at endpoint and ensures bucket exists.
// publicEndpoint is the host:port a browser can reach for presigned
// download URLs — pass the same value as endpoint when they coincide
// (e.g. local dev, where both the API and the browser reach MinIO via
// localhost).
func Connect(ctx context.Context, endpoint, publicEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*Storage, error) {
	creds := credentials.NewStaticV4(accessKey, secretKey, "")

	client, err := minio.New(endpoint, &minio.Options{Creds: creds, Secure: useSSL})
	if err != nil {
		return nil, err
	}

	presignedClient := client
	if publicEndpoint != endpoint {
		presignedClient, err = minio.New(publicEndpoint, &minio.Options{Creds: creds, Secure: useSSL})
		if err != nil {
			return nil, err
		}
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &Storage{client: client, presignedClient: presignedClient, bucket: bucket}, nil
}

func (s *Storage) Upload(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *Storage) PresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := s.presignedClient.PresignedGetObject(ctx, s.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *Storage) Delete(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}
