package main

import "time"

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
}