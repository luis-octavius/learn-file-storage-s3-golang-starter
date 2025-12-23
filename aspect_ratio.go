package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

type aspectRatio struct {
	Streams []struct {
		Width              int    `json:"width"`
		Height             int    `json:"height"`
		DisplayAspectRatio string `json:"display_aspect_ratio"`
	} `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var b bytes.Buffer
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		log.Println("error executing the command")
		return "", fmt.Errorf("Error executing the command: %v", err)
	}

	var aspRatio aspectRatio
	err = json.Unmarshal(b.Bytes(), &aspRatio)
	if err != nil {
		return "", fmt.Errorf("Error unmarshaling aspect ratio: %v", err)
	}

	// width := strconv.Itoa(aspRatio.Streams[0].Width / 120)
	// height := strconv.Itoa(aspRatio.Streams[0].Height / 120)

	finalAspectRatio := aspRatio.Streams[0].DisplayAspectRatio
	fmt.Println("final aspect ratio: ", finalAspectRatio)
	return finalAspectRatio, nil
}
