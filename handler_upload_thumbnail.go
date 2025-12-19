package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing media type from Content-Type Header", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Media type is not jpeg or png", err)
		return
	}

	fileExtension := strings.Split(mediaType, "/")[1]

	// img data converted to a slice of bytes
	fileData, err := io.ReadAll(file)

	// construct the file path to save the file in file system
	absPath, err := filepath.Abs(cfg.assetsRoot)

	b := make([]byte, 32)
	rand.Read(b)

	encodedString := base64.RawURLEncoding.EncodeToString(b)
	fmt.Println("encodedString file string: ", encodedString)

	path := encodedString + "." + fileExtension

	filePath := filepath.Join(absPath, path)
	fmt.Println("file path: ", filePath)

	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating the file", err)
		return
	}

	// copy the contents of the data into the new file created in file system
	_, err = io.Copy(newFile, strings.NewReader(string(fileData)))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error copying the contents of file data into the new file in file system", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to get the video based on video ID provided", err)
		return
	}

	dbThumbnailURL := "http://localhost:" + cfg.port + "/assets/" + encodedString + "." + fileExtension

	videoMetadata.ThumbnailURL = &dbThumbnailURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video", err)
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
