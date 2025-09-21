package main
type ImageSchema struct {
	Filename string `json:"filename"`
	Size int `json:"size"`
	Format string `json:"format"`
	Width int `json:"width"`
	Height int `json:"height"`
	UserId string `json:"user_id"`
}