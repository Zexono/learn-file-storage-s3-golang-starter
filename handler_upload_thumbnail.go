package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

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

	//Bit shifting is a way to multiply by powers of 2. 
	//10 << 20 is the same as 10 * 1024 * 1024, which is 10MB.
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	multiPartfile, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer multiPartfile.Close()

	contentTypes := header.Header.Get("Content-Type")

	//img_data, err := io.ReadAll(multiPartfile)
	//if err != nil {
	//	respondWithError(w, http.StatusBadRequest, "Unable to read form file", err)
	//	return
	//}

	viedeo_db ,err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No video id here", err)
		return
	}

	if viedeo_db.UserID != userID {
    respondWithError(w, http.StatusUnauthorized, "Not video owner", nil)
    return
	}

	//vod_thumbnail := thumbnail{
	//	data: img_data,
	//	mediaType: contentTypes,
	//}


	//videoThumbnails[viedeo_db.ID] = vod_thumbnail

	//img_data_str := base64.StdEncoding.EncodeToString(img_data)
	//data_url := "data:"+contentTypes+";base64,"+img_data_str

	media_type ,_,err := mime.ParseMediaType(contentTypes)
	if  media_type != "image/jpeg" && media_type != "image/png" && media_type != "image/webp"{
		respondWithError(w, http.StatusInternalServerError, "header contentType not img", err)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)
	filename := base64.RawURLEncoding.EncodeToString(key)

	assetPath := getAssetPath(filename, contentTypes)
	assetDiskPath := cfg.getAssetDiskPath(assetPath)

	dst, err := os.Create(assetDiskPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create file on server", err)
		return
	}
	defer dst.Close()
	if _, err = io.Copy(dst, multiPartfile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving file", err)
		return
	}

	/*file_extendsion := strings.Split(contentTypes, "/")
	if len(file_extendsion) < 2 {
		respondWithError(w, http.StatusInternalServerError, "header contentType err", err)
		return
	}

	filename := videoIDString + "." + file_extendsion[1]
	img_file_path := filepath.Join(cfg.assetsRoot,filename)
	new_file,err := os.Create(img_file_path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can't create img file", err)
		return
	}
	defer new_file.Close()

	io.Copy(new_file,multiPartfile)
	

	thumbnail_url := "http://localhost:"+cfg.port+"/"+img_file_path*/
	url := cfg.getAssetURL(assetPath)
	
	

	viedeo_db.ThumbnailURL = &url

	err = cfg.db.UpdateVideo(viedeo_db)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can't update vod data", err)
		return
	}

	
	respondWithJSON(w, http.StatusOK, viedeo_db)
}
