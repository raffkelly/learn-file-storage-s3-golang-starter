package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raffkelly/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/raffkelly/learn-file-storage-s3-golang-starter/internal/database"
)

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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse data", nil)
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, 400, "invalid media type for thumbnail", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "unable to retrieve video from db", nil)
		return
	}
	if videoData.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "user not authorized to get video data", errors.New("unauthorized user"))
		return
	}
	contentTypeString := strings.Split(contentType, "/")[1]
	fileExtension := "." + strings.Split(contentTypeString, ";")[0]
	bytePath := make([]byte, 32)
	_, _ = rand.Read(bytePath)
	urlString := base64.RawURLEncoding.EncodeToString(bytePath)
	filePath := filepath.Join(cfg.assetsRoot, urlString+fileExtension)
	thumbFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, 500, "error creating thumbnail file", err)
		return
	}
	_, err = io.Copy(thumbFile, file)
	if err != nil {
		respondWithError(w, 500, "error copying data into thumbnail file", err)
		return
	}

	dataURL := "http://localhost:8091/assets/" + urlString + fileExtension

	newVideo := database.Video{
		ID:                videoData.ID,
		CreatedAt:         videoData.CreatedAt,
		UpdatedAt:         time.Now(),
		ThumbnailURL:      &dataURL,
		VideoURL:          videoData.VideoURL,
		CreateVideoParams: videoData.CreateVideoParams,
	}
	cfg.db.UpdateVideo(newVideo)

	respondWithJSON(w, http.StatusOK, newVideo)
}
