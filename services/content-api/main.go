// services/content-api/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
)

type FrostResponse struct {
	ObjectID string `json:"object_id"`
}
type ContentRecord struct {
	ContentID string `json:"content_id"`
	BlurPath  string `json:"blur_path"`
	ObjectID  string `json:"object_id"`
}

var contentDB = map[string]ContentRecord{}

func main() {
	http.HandleFunc("/upload_content", uploadContentHandler)
	http.HandleFunc("/get_content", getContentHandler)

	addr := ":8082"
	log.Printf("content-api listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}

// uploadContentHandler:
//  1. /upload_content (POST multipart/form-data, field "file")
//  2. Отправляет в blur-service
//  3. Сохраняет blur локально
//  4. Отправляет оригинал в frostfs-service
//  5. Сохраняет инфу в contentDB
func uploadContentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 1) Blur
	blurData, err := callBlurService(data, header.Filename)
	if err != nil {
		http.Error(w, "Blur service error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 2) Сохраняем blur локально
	blurDir := "./blur_storage"
	os.MkdirAll(blurDir, 0755)
	blurFileName := header.Filename + ".blur.png"
	blurPath := path.Join(blurDir, blurFileName)
	if err := os.WriteFile(blurPath, blurData, 0644); err != nil {
		http.Error(w, "Save blur error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3) Оригинал -> frostfs-service
	objID, err := uploadToFrostFS(data)
	if err != nil {
		http.Error(w, "FrostFS upload error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	contentID := header.Filename
	record := ContentRecord{
		ContentID: contentID,
		BlurPath:  blurPath,
		ObjectID:  objID,
	}
	contentDB[contentID] = record

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func callBlurService(fileData []byte, filename string) ([]byte, error) {
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	fw.Write(fileData)
	mw.Close()

	blurURL := os.Getenv("BLUR_SERVICE_URL")
	if blurURL == "" {
		blurURL = "http://blur-service:5000/blur"
	}

	resp, err := http.Post(blurURL, mw.FormDataContentType(), buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

func uploadToFrostFS(data []byte) (string, error) {
	frostURL := os.Getenv("FROSTFS_SERVICE_URL")
	if frostURL == "" {
		frostURL = "http://frostfs-service:8081/upload"
	}
	resp, err := http.Post(frostURL, "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	var fr FrostResponse
	if err := json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return "", err
	}
	return fr.ObjectID, nil
}

// getContentHandler?user=0xABCDEF...&content_id=xxx
// Если user не имеет доступа -> вернуть blur. Иначе -> скачать из frostfs-service
func getContentHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	cid := r.URL.Query().Get("content_id")
	if user == "" || cid == "" {
		http.Error(w, "Missing user or content_id", http.StatusBadRequest)
		return
	}
	rec, ok := contentDB[cid]
	if !ok {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	has, err := hasAccess(user, cid)
	if err != nil {
		log.Printf("hasAccess error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if !has {
		// return blur
		data, err := os.ReadFile(rec.BlurPath)
		if err != nil {
			http.Error(w, "Blur file not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
		return
	}

	// download original from frostfs
	originalData, err := downloadFromFrostFS(rec.ObjectID)
	if err != nil {
		http.Error(w, "Failed to download from frostfs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(originalData)
}

func hasAccess(user, contentID string) (bool, error) {
	contractURL := os.Getenv("CONTRACT_CLIENT_URL")
	if contractURL == "" {
		contractURL = "http://contract-client:5001/has_access"
	}
	url := fmt.Sprintf("%s?user=%s&content_id=%s", contractURL, user, contentID)
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var j struct {
		Has bool `json:"has"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return false, err
	}
	return j.Has, nil
}

func downloadFromFrostFS(objID string) ([]byte, error) {
	frostURL := os.Getenv("FROSTFS_SERVICE_URL")
	if frostURL == "" {
		frostURL = "http://frostfs-service:8081"
	}
	url := fmt.Sprintf("%s/download?object_id=%s", frostURL, objID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}
