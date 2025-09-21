# Image Processor API

A Go-based REST API for image processing with S3 storage and PostgreSQL database integration.

## Features

- Image upload and processing
- Support for multiple image formats (JPEG, PNG, GIF)
- S3 storage integration
- PostgreSQL database for metadata storage
- Image information extraction (dimensions, format, size)
- User-based file organization

## Prerequisites

- Go 1.24.2 or later
- PostgreSQL database
- S3-compatible storage (AWS S3 or MinIO)

## Environment Variables

Create a `.env` file in the root directory with the following variables:

```env
PORT=8080
DATABASE_URL=postgres://username:password@localhost:5432/dbname
STORAGE_END_POINT=your-s3-endpoint
STORAGE_REGION=your-region
ACCESS_KEY=your-access-key
SECRET_KEY=your-secret-key
STORAGE_BUCKET=your-bucket-name
```

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd image-processor-api
```

2. Install dependencies:
```bash
go mod download
```

3. Set up your environment variables in a `.env` file

4. Run the application:
```bash
go run .
```

## API Endpoints

### Health Check
- **GET** `/` - Returns server status

### Image Upload
- **POST** `/upload` - Upload an image file
  - Headers: `x-user-id` (required)
  - Body: multipart/form-data with `image` field
  - Max file size: 10MB

## Project Structure

- `main.go` - Main application and HTTP handlers
- `model.go` - Data models and structures
- `postgres.go` - Database connection and operations
- `s3.go` - S3 storage operations
- `go.mod` - Go module dependencies

## Dependencies

- `github.com/joho/godotenv` - Environment variable loading
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/aws/aws-sdk-go-v2` - AWS SDK for S3 operations

## License

This project is licensed under the MIT License.
