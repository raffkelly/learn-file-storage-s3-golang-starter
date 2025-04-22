package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	type stream struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	type videoInfo struct {
		Streams []stream `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	var videoData videoInfo
	err = json.Unmarshal(buf.Bytes(), &videoData)
	if err != nil {
		return "", err
	}
	vidHeight := videoData.Streams[0].Height
	vidWidth := videoData.Streams[0].Width
	aspectRatio := float64(vidWidth) / float64(vidHeight)
	if (aspectRatio > 1.7) && (aspectRatio < 1.8) {
		return "16:9", nil
	} else if (aspectRatio > .5) && (aspectRatio < .6) {
		return "9:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	newPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return newPath, nil
}

/*
func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	params := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	req, err := presignClient.PresignGetObject(context.Background(), &params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	substrings := strings.Split(*video.VideoURL, ",")
	videoBucket := substrings[0]
	videoKey := substrings[1]
	presignedURL, err := generatePresignedURL(cfg.s3Client, videoBucket, videoKey, time.Minute*10)
	if err != nil {
		return video, err
	}
	signedVideo := video
	signedVideo.VideoURL = &presignedURL
	return signedVideo, nil
}
*/
