package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

// functional options pattern
func presignOptsNew(options ...func(*s3.PresignOptions)) *s3.PresignOptions {
	opts := &s3.PresignOptions{}
	for _, o := range options {
		o(opts)
	}
	return opts
}

func WithTime(expires time.Duration) func(*s3.PresignOptions) {
	return func(p *s3.PresignOptions) {
		p.Expires = expires
	}
}

// generatePresignedURL retrieves the URL of a presigned request
// it creates a presign client and uses functional options pattern
// returns an error if the presigned request fails

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)

	objectInput := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	req, err := presignClient.PresignGetObject(context.TODO(), &objectInput, (s3.WithPresignExpires(expireTime)))

	if err != nil {
		return "", fmt.Errorf("Error creating the presigned request: %v", err)
	}

	return req.URL, nil
}

// dbVideoToSignedVideo is a method of apiConfig that receives a database.Video as input
// and return a database.Video with the video URL field set to a presigned URL and an error
// returns an error if
func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	videoURL := video.VideoURL
	fmt.Println("videoURL: ", *videoURL)

	bucket, key, err := getBucketAndKey(*videoURL)
	if err != nil {
		return database.Video{}, fmt.Errorf("Error getting the bucket and key from URL: %v", err)
	}

	fmt.Println("Bucket: ", bucket)
	fmt.Println("Key: ", key)

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Duration(1*time.Hour))
	if err != nil {
		return database.Video{}, fmt.Errorf("Error generating presigned URL: %v", err)
	}

	video.VideoURL = &presignedURL
	fmt.Println("Video URL after presign: ", *video.VideoURL)
	return video, nil
}

// getBucketAndKey returns a clean s3 bucket and key from a videoURL
func getBucketAndKey(videoURL string) (bucket, key string, err error) {
	parsedURL, err := url.Parse(videoURL)
	if err != nil {
		return "", "", fmt.Errorf("Error parsing URL: %v", err)
	}

	path := strings.TrimPrefix(parsedURL.Path, "/")

	parts := strings.SplitN(path, ",", 2)

	if len(parts) < 2 {
		return "", "", fmt.Errorf("URL path must contain bucket and key")
	}

	bucket = parts[0]
	key = parts[1]

	return bucket, key, nil
}
