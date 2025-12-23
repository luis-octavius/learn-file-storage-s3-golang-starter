package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// set the maximum size of the request
	const maxMemory = 1 << 30
	http.MaxBytesReader(w, r.Body, maxMemory)

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
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to get the video based on video ID provided", err)
	}

	// parse the file from the request
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	defer file.Close()

	// get the MIME type
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing media type from Content-Type header", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Media type is not video/mp4", err)
		return
	}

	// create a temporary file to handle the data
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error creating temporary file", err)
		return
	}

	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error reading the file data", err)
		return
	}

	// copy the contents of the file from the form to the temporary file
	_, err = io.Copy(tempFile, strings.NewReader(string(fileData)))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error copying the contents from the file", err)
		return
	}

	// create a processed version of the video to load faster
	processedVideo, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error processing video: %v", err)
		return
	}
	fmt.Println("processed video: ", processedVideo)

	processedFile, err := os.Open(processedVideo)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error opening the processed video: %v", err)
		return
	}

	defer processedFile.Close()

	// check the aspect ratio of the video
	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error getting the aspect ratio", err)
		return
	}

	var folderPrefix string

	switch aspectRatio {
	case "16:9":
		folderPrefix = "landscape"
	case "9:16":
		folderPrefix = "portrait"
	default:
		folderPrefix = "other"
	}

	// create a file name with the extension
	fileExtension := strings.Split(mediaType, "/")[1]

	b := make([]byte, 32)
	rand.Read(b)
	encodedString := base64.RawURLEncoding.EncodeToString(b)

	fileKey := folderPrefix + "/" + encodedString + "." + fileExtension
	log.Println("filekey: ", fileKey)

	// resets the temp file to the beginning
	tempFile.Seek(0, io.SeekStart)

	putObject := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        processedFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(context.Background(), &putObject)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error putting object in Bucket: %v", err)
		return
	}

	// structure the URL of the video in S3 Bucket
	videoURL := "https://" + cfg.s3Bucket + ".s3." + cfg.s3Region + ".amazonaws.com/" + fileKey

	videoMetadata.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
