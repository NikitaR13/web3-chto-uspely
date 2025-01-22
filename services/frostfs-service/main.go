//services/frostfs-service/main.go

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/nspcc-dev/neofs-sdk-go/client"
	"github.com/nspcc-dev/neofs-sdk-go/client/object"
	"github.com/nspcc-dev/neofs-sdk-go/container"
	"github.com/nspcc-dev/neofs-sdk-go/session"
)

var (
	cl   *client.Client
	sess *session.Session
	cnr  container.ID
)

func main() {
	// Будем брать адрес FrostFS и containerID из ENV
	frostEndpoint := os.Getenv("FROSTFS_ENDPOINT") // например "frostfs:8080"
	containerID := os.Getenv("FROSTFS_CONTAINER_ID")

	if frostEndpoint == "" || containerID == "" {
		log.Fatal("FROSTFS_ENDPOINT or FROSTFS_CONTAINER_ID not set")
	}

	var err error
	cl, err = client.New(
		client.WithDefaultEndpoint("grpc://" + frostEndpoint),
	)
	if err != nil {
		log.Fatalf("Failed to create FrostFS client: %v", err)
	}

	// Генерируем или берем из ENV ключ
	key, err := neofsCrypto.GenerateKey()
	if err != nil {
		log.Fatalf("GenerateKey: %v", err)
	}

	sess = session.New()
	sess.SetAuthKey(key)

	cnr, err = container.NewIDFromString(containerID)
	if err != nil {
		log.Fatalf("Invalid container ID: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", uploadHandler)
	mux.HandleFunc("/download", downloadHandler)

	srvAddr := ":8081"
	log.Printf("FrostFS service started on %s", srvAddr)
	if err := http.ListenAndServe(srvAddr, mux); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	obj := object.New()
	obj.SetContainerID(cnr)
	obj.SetPayload(data)

	resp, err := cl.PutObject(r.Context(), obj, client.WithSession(sess))
	if err != nil {
		log.Printf("PutObject error: %v", err)
		http.Error(w, "PutObject error", http.StatusInternalServerError)
		return
	}

	objID := resp.StoredObjectID().String()
	fmt.Fprintf(w, `{"object_id":"%s"}`, objID)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	objIDStr := r.URL.Query().Get("object_id")
	if objIDStr == "" {
		http.Error(w, "Missing object_id", http.StatusBadRequest)
		return
	}

	oid, err := object.NewIDFromString(objIDStr)
	if err != nil {
		log.Printf("Invalid objID: %v", err)
		http.Error(w, "Invalid object_id", http.StatusBadRequest)
		return
	}

	getResp, err := cl.GetObject(r.Context(), cnr, oid, client.WithSession(sess))
	if err != nil {
		log.Printf("GetObject error: %v", err)
		http.Error(w, "GetObject error", http.StatusInternalServerError)
		return
	}

	payload := getResp.Object().Payload()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(payload)
}
