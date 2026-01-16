package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	http.MaxBytesReader(w,r.Body,maxMemory)

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

	viedeo_db ,err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No video id here", err)
		return
	}

	if viedeo_db.UserID != userID {
    respondWithError(w, http.StatusUnauthorized, "Not video owner", nil)
    return
	}

	//r.ParseMultipartForm(maxMemory)

	multiPartfile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer multiPartfile.Close()

	contentTypes := header.Header.Get("Content-Type")

	media_type ,_,err := mime.ParseMediaType(contentTypes)
	if  media_type != "video/mp4"{
		respondWithError(w, http.StatusInternalServerError, "header contentType not video", err)
		return
	}

	tmp ,err := os.CreateTemp("","tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create tmp", err)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()


	if _, err = io.Copy(tmp, multiPartfile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving file", err)
		return
	}

	tmp.Seek(0,io.SeekStart)

	preprocessed_file_path ,err:=processVideoForFastStart(tmp.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error preprocess file", err)
		return
	}
	defer os.Remove(preprocessed_file_path)

	processedFile, err := os.Open(preprocessed_file_path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error can't open preprocessed_file", err)
		return
	}
	defer processedFile.Close()
	
	

	

	key := make([]byte, 32)
	rand.Read(key)
	filename := base64.RawURLEncoding.EncodeToString(key)
	assetPath := getAssetPath(filename, contentTypes)

	ratio,err := getVideoAspectRatio(processedFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can't get aspectratio", err)
		return
	}
	switch ratio {
	case "16:9":
		assetPath = path.Join("landscape",assetPath)
	case "9:16":
		assetPath = path.Join("portrait",assetPath)
	default:
		assetPath = path.Join("other",assetPath)
	}


	cfg.s3Client.PutObject(context.Background(),&s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &assetPath,
		Body: processedFile,
		ContentType: &media_type,
	})

	url := fmt.Sprintf("%s,%s", cfg.s3Bucket, assetPath)
	viedeo_db.VideoURL = &url

	err = cfg.db.UpdateVideo(viedeo_db)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can't update vod data", err)
		return
	}

	signed_viedeo_db , err := cfg.dbVideoToSignedVideo(viedeo_db)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can't update signed vod data", err)
		return
	}

	
	respondWithJSON(w, http.StatusOK, signed_viedeo_db)


}



