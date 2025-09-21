package main

import (
  "database/sql"
  "fmt"
  _ "github.com/lib/pq"
)



type DBConfig struct {
	URL string
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
		user_id TEXT NOT NULL
	`)
}

func InsertImage(image ImageSchema) error {
	_, err := DBConnection.Exec("INSERT INTO images (filename, size, format, width, height, user_id) VALUES ($1, $2, $3, $4, $5, $6)", image.Filename, image.Size, image.Format, image.Width, image.Height, image.UserId)
	return err
}