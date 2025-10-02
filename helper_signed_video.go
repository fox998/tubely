package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fox998/tubely/internal/aws"
	"github.com/fox998/tubely/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	urlParts := strings.Split(*video.VideoURL, ",")
	if len(urlParts) != 2 || urlParts[0] == "" || urlParts[1] == "" {
		return database.Video{}, fmt.Errorf("invalid video url: %s", *video.VideoURL)
	}

	bucket := urlParts[0]
	key := urlParts[1]

	presignedurl, err := aws.GeneratePresignedURL(cfg.s3Client, bucket, key, 5*time.Minute)
	if err != nil {
		return database.Video{}, err
	}

	presignedVideo := video
	presignedVideo.VideoURL = &presignedurl

	return presignedVideo, nil
}
