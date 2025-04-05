package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"

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
	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, 500, "unable to read file data", nil)
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
	thumb := thumbnail{
		data:      fileData,
		mediaType: contentType,
	}
	videoThumbnails[videoID] = thumb
	newVideo := database.Video{}
	cfg.db.UpdateVideo(newVideo)

	// TODO: implement the upload here

	respondWithJSON(w, http.StatusOK, struct{}{})
}
