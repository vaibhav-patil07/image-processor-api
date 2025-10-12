package main

import (
	"database/sql"
	"time"
)

type ImageSchema struct {
	Filename string `json:"filename"`
	Size int `json:"size"`
	Format string `json:"format"`
	Width int `json:"width"`
	Height int `json:"height"`
	UserId string `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ImageID string `json:"image_id"`
	JOB_STATUS string `json:"job_status"`
	COMPRESSED_AT sql.NullTime `json:"compressed_at"`
	COMPRESSED_SIZE sql.NullInt64 `json:"compressed_size"`
}