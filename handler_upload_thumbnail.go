package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func getFileExtention(contentType string) (string, error) {
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}

	alloswedTypes := map[string]any{
		"image/jpeg": struct{}{},
		"image/png":  struct{}{},
	}

	if _, ok := alloswedTypes[mimeType]; !ok {
		return "", fmt.Errorf("file type %v not allowed", mimeType)
	}

	extentions, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return "", err
	}

	if len(extentions) == 0 {
		return "", errors.New("no extention found")
	}

	return extentions[0], nil
}

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		respondWithError(w, 500, "Failed to parse multipart form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 500, "Failed to get form file", err)
		return
	}

	contetnType := header.Header.Get("Content-Type")
	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Failed to get db video", err)
		return
	}

	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	fileExtention, err := getFileExtention(contetnType)
	if err != nil {
		respondWithError(w, 500, "failed to get file extention", err)
		return
	}

	assetPath := filepath.Join(cfg.assetsRoot, uuid.NewString()+fileExtention)
	assetFile, err := os.Create(assetPath)
	if err != nil {
		respondWithError(w, 500, "failed to create asset file", err)
		return
	}

	defer assetFile.Close()

	_, err = io.Copy(assetFile, file)
	if err != nil {
		respondWithError(w, 500, "failed to create asset file", err)
		return
	}

	assetUrl := fmt.Sprintf("http://localhost:%v/%v", cfg.port, assetPath)
	dbVideo.UpdatedAt = time.Now()
	dbVideo.ThumbnailURL = &assetUrl

	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, 500, "failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
