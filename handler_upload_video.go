package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/raffkelly/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/raffkelly/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	cappedBody := http.MaxBytesReader(w, r.Body, 1<<30)
	r.Body = cappedBody
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

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "unable to retrieve video from db", nil)
		return
	}
	if videoData.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "user not authorized to get video data", errors.New("unauthorized user"))
		return
	}

	videoFile, videoHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse data", nil)
		return
	}
	defer videoFile.Close()
	contentType := videoHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if mediaType != "video/mp4" || err != nil {
		respondWithError(w, 400, "invalid media type for video", err)
		return
	}
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 500, "error creating file in memory", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	io.Copy(tempFile, videoFile)
	tempFile.Seek(0, io.SeekStart)
	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, 500, "error determining aspect ratio", err)
		return
	}
	if aspectRatio == "16:9" {
		aspectRatio = "landscape"
	} else if aspectRatio == "9:16" {
		aspectRatio = "portrait"
	} else {
		aspectRatio = "other"
	}
	processedFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, 500, "error processing video", err)
		return
	}
	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, 500, "error opening processed video", err)
		return
	}
	defer processedFile.Close()

	contentTypeString := strings.Split(contentType, "/")[1]
	fileExtension := "." + strings.Split(contentTypeString, ";")[0]
	bytePath := make([]byte, 32)
	_, _ = rand.Read(bytePath)
	filenameString := aspectRatio + "/" + hex.EncodeToString(bytePath) + fileExtension

	putObject := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &filenameString,
		Body:        processedFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &putObject)
	if err != nil {
		respondWithError(w, 500, "error uploading to aws s3", err)
		return
	}

	videoURL := fmt.Sprintf("%v/%v", cfg.s3CfDistribution, filenameString)

	newVideo := database.Video{
		ID:                videoData.ID,
		CreatedAt:         videoData.CreatedAt,
		UpdatedAt:         time.Now(),
		ThumbnailURL:      videoData.ThumbnailURL,
		VideoURL:          &videoURL,
		CreateVideoParams: videoData.CreateVideoParams,
	}
	cfg.db.UpdateVideo(newVideo)

	respondWithJSON(w, http.StatusOK, newVideo)
}
