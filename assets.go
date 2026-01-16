package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(filename string, mediaType string) string {
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", filename, ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

//func (cfg apiConfig) getVideoAssetURL(bucketName string, region string, assetPath string) string {
//	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, assetPath)
//}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
return "." + parts[1]
}

func getVideoAspectRatio(filePath string) (string, error){
	cmd := exec.Command("ffprobe","-v", "error", "-print_format", "json", "-show_streams",filePath)
	var b bytes.Buffer
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		return "",err
	}

	var meta_data vid_streams
	err = json.Unmarshal(b.Bytes(),&meta_data)
	if err != nil {
		return "",err
	}

	if meta_data.Streams[0].DisplayAspectRatio == "16:9" {
		return "16:9",nil
	}
	if meta_data.Streams[0].DisplayAspectRatio == "9:16" {
		return  "9:16",nil
	}

	return "other",nil

}

type vid_streams struct {
	Streams []struct {
		Width              int    `json:"width,omitempty"`
		Height             int    `json:"height,omitempty"`
		SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
		DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
	} `json:"streams"`
}

func processVideoForFastStart(filePath string) (string, error){
	new_filepath := filePath+".processing"
	cmd := exec.Command("ffmpeg","-i",filePath,"-c","copy","-movflags","faststart","-f","mp4",new_filepath)
	err := cmd.Run()
	if err != nil {
		return "",err
	}
	return new_filepath,nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error){
	
	presignClient := s3.NewPresignClient(s3Client)
	object , err := presignClient.PresignGetObject(context.Background(),&s3.GetObjectInput{
		Bucket: &bucket,
		Key: &key,
	},s3.WithPresignExpires(expireTime))

	if err != nil {
		return "",err
	}
	return object.URL,nil

}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error){
	
	if video.VideoURL == nil {
    	return video, nil
	}

	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) < 2 {
    	return video, nil
	}
	bucket := parts[0]
	key := parts[1]
	
	url,err := generatePresignedURL(cfg.s3Client,bucket,key,3600)
	if err != nil {
		return video,err
	}
	video.VideoURL = &url
	
	return video,nil
}