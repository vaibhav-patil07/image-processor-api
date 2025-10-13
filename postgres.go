package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)



type DBConfig struct {
	URL string
}

type ImagesResponse struct {
	Images []ImageSchema `json:"images"`
	TotalCount int `json:"count"`
}

type ImageResponse struct {
	Image ImageSchema `json:"image"`
}

var DBConnection *sql.DB = nil

func GetDBConnection(config DBConfig) (*sql.DB, error) {
	if DBConnection != nil {
		return DBConnection, nil
	}
	db, err := sql.Open("postgres", config.URL)
	if err != nil {
		return nil, err
	}
	DBConnection = db
	err = CreateImageTable()
	if err != nil {
		return nil, err
	}
	fmt.Println("Database connected successfully")
	fmt.Println("Image table created successfully")
	return db, nil
}

func CloseDBConnection() {
	DBConnection.Close()
	DBConnection = nil;	
}

func CreateTable(db *sql.DB, tableName string, columns string) error {
	_, err := db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, columns))
	return err
}

func CreateImageTable() error {
	return CreateTable(DBConnection, "images", `
		id SERIAL PRIMARY KEY,
		filename TEXT NOT NULL,
		size INT NOT NULL,
		format TEXT NOT NULL,
		width INT NOT NULL,
		height INT NOT NULL,
		user_id TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		image_id TEXT NOT NULL UNIQUE,
		job_status TEXT NOT NULL,
		compressed_at TIMESTAMP,
		compressed_size INT
	`)
}

func InsertImage(image ImageSchema) error {
	_, err := DBConnection.Exec("INSERT INTO images (filename, size, format, width, height, user_id, created_at, updated_at, image_id,job_status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)", image.Filename, image.Size, image.Format, image.Width, image.Height, image.UserId, image.CreatedAt, image.UpdatedAt, image.ImageID,image.JOB_STATUS)
	return err
}

func GetImagesByUserId(userId string, skip int, limit int, jobsStatus string) (ImagesResponse, error) {
	query := ""
	var rows *sql.Rows = nil;
	var err error = nil;
	if jobsStatus == "" {
		query = "SELECT * ,count(*) OVER() AS total_count FROM images WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		rows, err = DBConnection.Query(query, userId, limit, skip)
	} else {
		query = "SELECT * ,count(*) OVER() AS total_count FROM images WHERE user_id = $1 AND job_status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4"
		rows, err = DBConnection.Query(query, userId, jobsStatus, limit, skip)
	}
	if err != nil {
		return ImagesResponse{}, err
	}
	defer rows.Close()
	images := []ImageSchema{}
	totalCount := 0
	// var totalCount int
	for rows.Next() {
		var image ImageSchema
		err := rows.Scan(&image.ImageID, &image.Filename, &image.Size, &image.Format, &image.Width, &image.Height, &image.UserId, &image.CreatedAt, &image.UpdatedAt, &image.ImageID, &image.JOB_STATUS, &image.COMPRESSED_AT, &image.COMPRESSED_SIZE, &totalCount)
		if err != nil {
			return ImagesResponse{}, err
		}
		images = append(images, image)
	}
	return ImagesResponse{Images: images, TotalCount: totalCount}, nil
}

func GetImageById(imageID string, userId string) (ImageResponse, error) {
	query := "SELECT * FROM images WHERE image_id = $1 AND user_id = $2"
	row := DBConnection.QueryRow(query, imageID, userId)
	var image ImageSchema
	err := row.Scan(&image.ImageID, &image.Filename, &image.Size, &image.Format, &image.Width, &image.Height, &image.UserId, &image.CreatedAt, &image.UpdatedAt, &image.ImageID, &image.JOB_STATUS, &image.COMPRESSED_AT, &image.COMPRESSED_SIZE)
	if err != nil {
		return ImageResponse{}, err
	}
	response := ImageResponse{Image: image}
	return response, nil	
}