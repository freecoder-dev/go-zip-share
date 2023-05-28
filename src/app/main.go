package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL       = "http://localhost:8080" // Replace with your base URL
	uploadsDir    = "./uploads/"
	maxUploadSize = 10 * 1024 * 1024 // 10MB
)

type UploadResponse struct {
	Message string `json:"message"`
	FileURL string `json:"file_url"`
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/file/", serveFile)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func index(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./../src/web/index.html"))
	tmpl.Execute(w, nil)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit the size of the uploaded file
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate a unique file name
	fileName := strconv.FormatInt(time.Now().UnixNano(), 10)

	// Create the uploads directory if it doesn't exist
	err = os.MkdirAll(uploadsDir, os.ModePerm)
	if err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Create the new zip file in the uploads directory
	zipFilePath := uploadsDir + fileName + ".zip"
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		http.Error(w, "Failed to create zip file", http.StatusInternalServerError)
		return
	}
	defer zipFile.Close()

	// Create a zip writer
	zw := zip.NewWriter(zipFile)

	// Create a new file in the zip writer
	zf, err := zw.Create(handler.Filename)
	if err != nil {
		http.Error(w, "Failed to create file in zip", http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file content to the zip entry
	_, err = io.Copy(zf, file)
	if err != nil {
		http.Error(w, "Failed to save file in zip", http.StatusInternalServerError)
		return
	}

	// Close the zip writer
	err = zw.Close()
	if err != nil {
		http.Error(w, "Failed to close zip writer", http.StatusInternalServerError)
		return
	}

	// Generate a shareable link
	fileURL := fmt.Sprintf("%s/file/%s.zip", baseURL, fileName)

	// Create the JSON response
	response := UploadResponse{
		Message: "File uploaded successfully!",
		FileURL: fileURL,
	}

	// Convert the response to JSON
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to generate JSON response", http.StatusInternalServerError)
		return
	}

	// Set the response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Send the JSON response
	w.Write(jsonResponse)
}

func serveFile(w http.ResponseWriter, r *http.Request) {
	fileName := strings.TrimPrefix(r.URL.Path, "/file/")

	// Retrieve the file from the uploads directory
	filePath := uploadsDir + fileName
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Set the appropriate Content-Type header
	w.Header().Set("Content-Type", "application/zip")

	// Copy the file data to the response
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Failed to serve file", http.StatusInternalServerError)
		return
	}
}
