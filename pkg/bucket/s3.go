package bucket

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 provides a simple Uploader backed by Amazon S3 or S3-compatible storage.
type S3 struct {
	client   *s3.Client
	bucket   string
	region   string
	endpoint string // optional; S3-compatible endpoint (e.g., MinIO)
	// simple key prefix grouping, optional; not part of interface but useful for callers
	prefix string
}

// NewS3 constructs an S3 uploader with static credentials.
// If endpoint is non-empty, the client will target that endpoint (S3 compatible) and use path-style addressing.
func NewS3(ctx context.Context, accessKey, secretKey, region, bucketName, endpoint, prefix string) (*S3, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if region == "" {
		region = "us-east-1"
	}

	// Configure credentials
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var client *s3.Client
	if endpoint != "" {
		// Normalize endpoint; must be valid URL for BaseEndpoint
		ep := endpoint
		if !strings.HasPrefix(ep, "http://") && !strings.HasPrefix(ep, "https://") {
			ep = "https://" + ep
		}
		if _, err := url.Parse(ep); err != nil {
			return nil, fmt.Errorf("invalid s3 endpoint: %w", err)
		}
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = &ep
			// Most S3-compatible stores prefer path-style
			o.UsePathStyle = true
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	return &S3{
		client:   client,
		bucket:   bucketName,
		region:   region,
		endpoint: endpoint,
		prefix:   strings.Trim(prefix, "/"),
	}, nil
}

// Upload implements Uploader.Upload by putting the object into S3 and returning a public URL.
func (s *S3) Upload(ctx context.Context, key string, body []byte, contentType string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("s3 client is nil")
	}
	if key == "" {
		return "", fmt.Errorf("key is required")
	}
	if s.prefix != "" {
		key = strings.Trim(s.prefix+"/"+strings.TrimLeft(key, "/"), "/")
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(body),
		ContentType: &contentType,
		// ACL:         s3types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}

	// Construct a public URL. If endpoint provided, prefer it; else use standard AWS format.
	if s.endpoint != "" {
		ep := strings.TrimRight(s.endpoint, "/")
		if !strings.HasPrefix(ep, "http://") && !strings.HasPrefix(ep, "https://") {
			ep = "https://" + ep
		}
		return fmt.Sprintf("%s/%s/%s", ep, s.bucket, key), nil
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key), nil
}

// Key returns the bucket name as a simple identifier.
func (s *S3) Key() string { return strings.Trim(s.bucket+"/"+s.prefix, "/") }
