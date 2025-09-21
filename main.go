package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ServerUp struct{
	Message string `json:"message"`
}

type AppError struct {
	Message string `json:"message"`
}

type File struct {
	File multipart.File
	Header *multipart.FileHeader
}

type ImageInfo struct {
	Filename string `json:"filename"`
	Size int `json:"size"`
	Format string `json:"format"`
	Width int `json:"width"`
	Height int `json:"height"`
	userId string
}

type ImageProcessorFolder string

const (
	Uploads ImageProcessorFolder = "uploads"
	Resized ImageProcessorFolder = "resized"
)

func initStorage() error{
	host := os.Getenv("STORAGE_END_POINT")
	region := os.Getenv("STORAGE_REGION")
	accessKeyID := os.Getenv("ACCESS_KEY")
	secretAccessKey := os.Getenv("SECRET_KEY")
	storageBucket := os.Getenv("STORAGE_BUCKET")

	s3Config := S3Config{
		Region: region,
		AccessKeyID: accessKeyID,
		SecretAccessKey: secretAccessKey,
		Host: host,
	}
	_, err := ConnectToS3(s3Config, storageBucket)
	if err != nil {
		fmt.Println("Error connecting to storage:", err)
		return err
	}
	return nil
}

func returnAppError(w http.ResponseWriter, message string, statusCode int) {
	appError := AppError{Message: message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(appError)
}

func parseFileFromForm(r *http.Request, sizeLimit int64) (File, error) {
	err := r.ParseMultipartForm(sizeLimit)
	if err != nil {
		return File{}, err
	}

	// Get the file from form data
	file, header, err := r.FormFile("image")
	if err != nil {
		return File{}, err
	}

	return File{File: file, Header: header}, nil;
}

func parseImageFromFile(file File) (ImageInfo, error) {
	header := file.Header
	// Check if it's an image file
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		return ImageInfo{}, errors.New("only image files (jpg, jpeg, png, gif) are allowed")
	}

	// Read file content
	fileBytes, err := io.ReadAll(file.File)
	if err != nil {
		return ImageInfo{}, errors.New("unable to read file")
	}

	// Decode image to get dimensions and format
	file.File.Seek(0, 0) // Reset file pointer
	img, format, err := image.Decode(file.File)
	if err != nil {
		return ImageInfo{}, errors.New("unable to decode image")
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create response
	imageInfo := ImageInfo{
		Filename: header.Filename,
		Size: len(fileBytes),
		Format: format,
		Width: width,
		Height: height,
	}
	return imageInfo, nil
}

// uploadHandler handles image file uploads and prints image information
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userId := r.Header.Get("x-user-id")
	if userId == "" {
		returnAppError(w, "User ID is missing", http.StatusBadRequest)
		return
	}
	
	file, err := parseFileFromForm(r, 10 << 20)
	if err != nil {
		returnAppError(w, "Unable to parse file", http.StatusBadRequest)
		return
	}
	defer file.File.Close()

	imageInfo, err := parseImageFromFile(file)
	imageInfo.userId = userId

	if err != nil {
		returnAppError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Try to insert into database if connection is available
	filePath := fmt.Sprintf("%s/%s/%s", Uploads, userId, imageInfo.Filename) 
	s3Object := s3.PutObjectInput{
		Bucket: aws.String(GetS3Bucket()),
		Key: aws.String(filePath),
		Body: file.File,
	}
	err = UploadFileToS3(&s3Object)
	if err != nil {
		fmt.Println("Error uploading file to storage:", err)
		returnAppError(w, "Unable to save file to storage", http.StatusInternalServerError)
		return
	}
	imageObject := ImageSchema{
		Filename: imageInfo.Filename,
		Size: imageInfo.Size,
		Format: imageInfo.Format,
		Width: imageInfo.Width,
		Height: imageInfo.Height,
		UserId: imageInfo.userId,
	}
	err = InsertImage(imageObject)
	if err != nil {
		fmt.Println("Error saving file to database:", err)
		returnAppError(w, "Unable to save file to database", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imageInfo)
}

func main() {

	// Register a handler function for the root path "/"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		helloMessage:= ServerUp{ Message: "Server is up and running :)"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(helloMessage)
	})

	// Register the upload handler
	http.HandleFunc("/upload", uploadHandler)

	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("No .env file found, using system environment variables.")
	}
	port := os.Getenv("PORT")
	// Database connection (optional for development)
	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		fmt.Println("No DATABASE_URL provided.")
		return
	}
	dbConfig := DBConfig{
		URL: dbURL,
	}
	_, err := GetDBConnection(dbConfig)
	if err != nil {
		fmt.Println("Warning: Could not connect to database:", err)
		return
	}
	defer CloseDBConnection()

	err = initStorage()
	if err != nil {
		fmt.Println("Error initializing storage:", err)
		return
	}
	defer CloseS3Connection()
	// Start the HTTP server on port 8080
	fmt.Println("Server listening on", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}