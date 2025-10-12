package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nfnt/resize"
	"github.com/rs/cors"
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

type ImageProcessorMessage struct {
	ImageID string `json:"image_id"`
	UserId string `json:"user_id"`
	Filename string `json:"filename"`
}

type ImageProcessorFolder string

const (
	Uploads ImageProcessorFolder = "uploads"
	Resized ImageProcessorFolder = "resized"
	Thumbnail ImageProcessorFolder = "thumbnail"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var addWSClientMutex sync.Mutex = sync.Mutex{}

// LogLevel represents different log levels
type LogLevel string

const (
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
	DEBUG LogLevel = "DEBUG"
)

// StackFrame represents a single stack frame
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     string `json:"line"`
}

// logStructured logs a structured message with optional stack trace
func logStructured(level LogLevel, message string, err error, statusCode int, includeStack bool) {
	logEntry := map[string]interface{}{
		"level":     level,
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   message,
	}
	
	if err != nil {
		logEntry["error"] = err.Error()
	}
	
	if statusCode != 0 {
		logEntry["statusCode"] = statusCode
	}
	
	if includeStack {
		// Parse stack trace into structured format
		stackTrace := string(debug.Stack())
		stackLines := strings.Split(stackTrace, "\n")
		
		var frames []StackFrame
		for i := 1; i < len(stackLines)-1; i += 2 { // Stack trace comes in pairs
			if i+1 < len(stackLines) && strings.TrimSpace(stackLines[i]) != "" {
				functionLine := strings.TrimSpace(stackLines[i])
				fileLine := strings.TrimSpace(stackLines[i+1])
				
				// Parse file and line number
				parts := strings.Split(fileLine, ":")
				file := ""
				line := ""
				if len(parts) >= 2 {
					file = parts[0]
					line = parts[1]
					// Remove extra info after line number
					if spaceIndex := strings.Index(line, " "); spaceIndex != -1 {
						line = line[:spaceIndex]
					}
				}
				
				frames = append(frames, StackFrame{
					Function: functionLine,
					File:     file,
					Line:     line,
				})
				
				// Limit to first 10 frames for readability
				if len(frames) >= 10 {
					break
				}
			}
		}
		
		logEntry["stackTrace"] = frames
		logEntry["rawStack"] = stackTrace // Keep for debugging
	}
	
	jsonLog, _ := json.MarshalIndent(logEntry, "", "  ")
	fmt.Println(string(jsonLog))
}

// jsonStringify safely converts any struct to JSON string
func jsonStringify(data interface{}) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

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

func returnAppError(w http.ResponseWriter, message string, statusCode int, err error) {
	appError := AppError{Message: message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(appError)
	
	// Log the error with structured stack trace
	if err != nil {
		logStructured(ERROR, message, err, statusCode, true)
	}
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

func resizeImage(file File) ( *bytes.Buffer, error) {
	img, format, err := image.Decode(file.File)
	if err != nil {
		logStructured(ERROR, "Unable to decode image", err, 0, true)
		return nil, errors.New("unable to decode image")
	}
	resizedImg := resize.Resize(300, 0, img, resize.Lanczos3)
	buf := new(bytes.Buffer)
	if format == "png" {
		err = png.Encode(buf, resizedImg)
	} else {
		err = jpeg.Encode(buf, resizedImg, nil)
	}
	if err != nil {
		logStructured(ERROR, "Unable to encode image", err, 0, true)
		return nil, errors.New("unable to encode image")
	}
	return buf, nil
}

// uploadHandler handles image file uploads and prints image information
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["user_id"]
	if userId == "" {
		returnAppError(w, "User ID is missing", http.StatusBadRequest, nil)
		return
	}
	
	file, err := parseFileFromForm(r, 10 << 20)
	if err != nil {
		returnAppError(w, "Unable to parse file", http.StatusBadRequest, err)
		return
	}
	defer file.File.Close()

	imageInfo, err := parseImageFromFile(file)
	imageInfo.userId = userId

	if err != nil {
		returnAppError(w, err.Error(), http.StatusBadRequest, err)
		return
	}
	imageID := uuid.New().String();
	// Try to insert into database if connection is available
	filePath := fmt.Sprintf("%s/%s/%s/%s", Uploads, userId, imageID,imageInfo.Filename) 
	file.File.Seek(0, 0)
	s3Object := s3.PutObjectInput{
		Bucket: aws.String(GetS3Bucket()),
		Key: aws.String(filePath),
		Body: file.File,
	}
	err = UploadFileToS3(&s3Object)
	if err != nil {
		fmt.Println("Error uploading file to storage:", err)
		returnAppError(w, "Unable to save file to storage", http.StatusInternalServerError, err)
		return
	}

	// Resize image
	file.File.Seek(0, 0)
	buf, err := resizeImage(file)
	if err != nil {
		returnAppError(w, "Unable to resize image", http.StatusInternalServerError, err)
		return
	}
	s3Object.Body = buf
	filePath = fmt.Sprintf("%s/%s/%s/%s", Thumbnail, userId, imageID, imageInfo.Filename)
	s3Object.Key = aws.String(filePath)
	err = UploadFileToS3(&s3Object)
	if err != nil {
		fmt.Println("Error uploading file to storage:", err)
		returnAppError(w, "Unable to save file to storage", http.StatusInternalServerError, err)
		return
	}

	imageObject := ImageSchema{
		Filename: imageInfo.Filename,
		Size: imageInfo.Size,
		Format: imageInfo.Format,
		Width: imageInfo.Width,
		Height: imageInfo.Height,
		UserId: imageInfo.userId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ImageID: imageID,
		JOB_STATUS: "in-queue",
	}
	err = InsertImage(imageObject)
	if err != nil {
		fmt.Println("Error saving file to database:", err)
		returnAppError(w, "Unable to save file to database", http.StatusInternalServerError, err)
		return
	}
	// Log successful upload
	logStructured(INFO, fmt.Sprintf("Image uploaded successfully: %s (%.2f KB)", imageInfo.Filename, float64(imageInfo.Size)/1024), nil, 200, false)
	
	imageProcessorMessage := ImageProcessorMessage{
		ImageID: imageID,
		UserId: imageInfo.userId,
		Filename: imageInfo.Filename,
	}
	
	// JSON stringify the message using helper function
	messageJSON, err := jsonStringify(imageProcessorMessage)
	if err != nil {
		logStructured(ERROR, "Failed to marshal message to JSON", err, 0, true)
		// Continue with upload success even if message publishing fails
	} else {
		err = PublishMessage(Message{
			Pattern: "image-processor",
			Message: messageJSON,
			MessageId: imageID,
		})
		if err != nil {
			logStructured(ERROR, "Failed to publish message", err, 0, false)
		} else {
			logStructured(INFO, fmt.Sprintf("Message published successfully for image: %s", imageID), nil, 0, false)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imageInfo)
}

func getImagesByUserId(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["user_id"]
	skip, err := strconv.Atoi(r.URL.Query().Get("skip"))
	if err != nil {
		skip = 0
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 100
	}
	jobsStatus := r.URL.Query().Get("jobs_status")
	if userId == "" {
		returnAppError(w, "User ID is missing", http.StatusBadRequest, nil)
		return
	}
	imagesResponse, err := GetImagesByUserId(userId, skip, limit, jobsStatus)
	if err != nil {
		returnAppError(w, "Unable to get images", http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imagesResponse)
}

func updateImageJobStatus(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["user_id"]
	if userId == "" {
		returnAppError(w, "User ID is missing", http.StatusBadRequest, nil)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		returnAppError(w, "Unable to upgrade to websocket", http.StatusInternalServerError, err)
		return
	}
	defer ws.Close()
	addWSClientMutex.Lock()
	clients[userId] = ws
	addWSClientMutex.Unlock()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			addWSClientMutex.Lock()
			delete(clients, userId)
			addWSClientMutex.Unlock()
			return
		}
	}
}

func main() {

	router :=  mux.NewRouter()

	// Register a handler function for the root path "/"
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		helloMessage:= ServerUp{ Message: "Server is up and running :)"}
		w.Header().Set("Content-Type", "application/json")
		err := PingPublisher()
		if err != nil {
			helloMessage.Message = "Server is up and running :) but publisher is not connected"
		}
		json.NewEncoder(w).Encode(helloMessage)
	}).Methods("GET")

	// Register the upload handler
	router.HandleFunc("/users/{user_id}/images", uploadHandler).Methods("POST")
	router.HandleFunc("/users/{user_id}/images", getImagesByUserId).Methods("GET")
	router.HandleFunc("/ws/users/{user_id}/images", updateImageJobStatus).Methods("GET")

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
	redisUrl := os.Getenv("REDIS_URL")
	queueName := os.Getenv("QUEUE_NAME")
	if redisUrl == "" {
		fmt.Println("No REDIS_URL provided.")
		return
	}
	publisherOptions := Options{
		QueueName: queueName,
	}
	err = InitializePublisher(redisUrl, publisherOptions)
	if err != nil {
		fmt.Println("Error initializing publisher:", err)
		return
	}
	err = InitializeEventSubscriber(redisUrl)
	if err != nil {
		fmt.Println("Error initializing event subscriber:", err)
		return
	}
	go SubscribeToEvent("image-processor-progress")
	defer CloseEventSubscriber()
	defer CloseS3Connection()
	corsHandler := cors.AllowAll().Handler(router)
	// Start the HTTP server on port 8080
	fmt.Println("Server listening on", port)
	err = http.ListenAndServe(":"+port, corsHandler)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}