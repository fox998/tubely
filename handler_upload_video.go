package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fox998/tubely/internal/auth"
	"github.com/fox998/tubely/internal/video"
	_ "github.com/fox998/tubely/internal/video"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	http.MaxBytesReader(w, r.Body, 1<<30)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}

	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You can't update this video", err)
		return
	}

	multipartFile, multipartHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get multipart file", err)
		return
	}
	defer multipartFile.Close()

	mediaType, _, err := mime.ParseMediaType(multipartHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse media type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type: "+mediaType, err)
		return
	}

	osTempFile, err := os.CreateTemp("", "tubely-upload_*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}
	defer os.Remove(osTempFile.Name())
	defer osTempFile.Close()

	io.Copy(osTempFile, multipartFile)
	osTempFile.Seek(0, io.SeekStart)

	processingVideo, err := video.ProcessVideoForFastStart(osTempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process video", err)
		return
	}
	defer os.Remove(processingVideo)

	processingVideoBytes, err := os.ReadFile(processingVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read processed video", err)
		return
	}

	aspectRatio, err := video.GetVideoOrientation(osTempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video aspect ratio", err)
		return
	}

	awsPutObjKey := fmt.Sprintf("%v/%v.mp4",
		aspectRatio,
		uuid.NewString())

	log.Println("putting object with key", awsPutObjKey)
	awsPutObjParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &awsPutObjKey,
		Body:        bytes.NewReader(processingVideoBytes),
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &awsPutObjParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't put object", err)
		return
	}

	newDbVideoUrl := fmt.Sprintf("https://%v/%v", cfg.s3CfDistribution, awsPutObjKey)
	dbVideo.VideoURL = &newDbVideoUrl
	dbVideo.UpdatedAt = time.Now()
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	log.Printf("Updated video url %v\n", newDbVideoUrl)
	respondWithJSON(w, http.StatusOK, dbVideo)
}
