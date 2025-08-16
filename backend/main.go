package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Database connection
var db *sql.DB

// Models
type Court struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BookingHour struct {
	ID        int       `json:"id"`
	CourtID   int       `json:"courtId"`
	DateStart time.Time `json:"dateStart"`
	DateEnd   time.Time `json:"dateEnd"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Clip struct {
	ID            int       `json:"id"`
	BookingHourID int       `json:"bookingHourId"`
	Filename      string    `json:"filename"`
	FilePath      string    `json:"file_path"`
	FileSize      *int64    `json:"file_size"`
	MimeType      *string   `json:"mime_type"`
	Duration      *int      `json:"duration_seconds"`
	CameraName    *string   `json:"camera_name"`
	UploadStatus  string    `json:"upload_status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Configuration
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ServerPort string
	UploadDir  string
}

func loadConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "cctv_system"),
		ServerPort: getEnv("SERVER_PORT", "5009"),
		UploadDir:  getEnv("UPLOAD_DIR", "./uploads"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Database initialization with retries
func initDatabase(config *Config) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	var err error
	maxRetries := 30
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Attempt %d/%d: Failed to open database connection: %v", i+1, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		err = db.Ping()
		if err != nil {
			log.Printf("Attempt %d/%d: Failed to ping database: %v", i+1, maxRetries, err)
			db.Close()
			time.Sleep(retryInterval)
			continue
		}

		// Connection successful
		log.Println("Database connected successfully")
		
		// Set connection pool settings
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		
		return
	}

	log.Fatalf("Failed to connect to database after %d attempts: %v", maxRetries, err)
}

// Helper functions
func sendJSONResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	sendJSONResponse(w, statusCode, APIResponse{
		Success: false,
		Message: message,
		Data:    nil,
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		
		next.ServeHTTP(w, r)
		
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Court handlers
func getCourts(w http.ResponseWriter, r *http.Request) {
	nameFilter := r.URL.Query().Get("name")
	log.Printf("Getting courts with name filter: '%s'", nameFilter)
	
	var query string
	var args []interface{}
	
	if nameFilter != "" {
		query = "SELECT id, name, description, is_active, created_at, updated_at FROM courts WHERE name ILIKE $1 AND is_active = true"
		args = append(args, "%"+nameFilter+"%")
	} else {
		query = "SELECT id, name, description, is_active, created_at, updated_at FROM courts WHERE is_active = true"
	}

	log.Printf("Executing query: %s with args: %v", query, args)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying courts: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch courts: %v", err))
		return
	}
	defer rows.Close()

	var courts []Court
	for rows.Next() {
		var court Court
		err := rows.Scan(&court.ID, &court.Name, &court.Description, &court.IsActive, &court.CreatedAt, &court.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning court: %v", err)
			sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error scanning court data: %v", err))
			return
		}
		courts = append(courts, court)
	}

	// Check for row iteration errors
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating over court rows: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error processing court data: %v", err))
		return
	}

	log.Printf("Found %d courts", len(courts))

	if len(courts) == 0 && nameFilter != "" {
		log.Printf("No courts found with name filter: %s", nameFilter)
		sendErrorResponse(w, http.StatusNotFound, "Court not found")
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Courts retrieved successfully (%d found)", len(courts)),
		Data:    courts,
	})
}

func createCourt(w http.ResponseWriter, r *http.Request) {
	var court Court
	if err := json.NewDecoder(r.Body).Decode(&court); err != nil {
		log.Printf("Error decoding court JSON: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if court.Name == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Court name is required")
		return
	}

	log.Printf("Creating court: %s", court.Name)

	query := `INSERT INTO courts (name, description, is_active) VALUES ($1, $2, true) RETURNING id, created_at, updated_at`
	err := db.QueryRow(query, court.Name, court.Description).Scan(&court.ID, &court.CreatedAt, &court.UpdatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			log.Printf("Court with name '%s' already exists", court.Name)
			sendErrorResponse(w, http.StatusConflict, "Court with this name already exists")
			return
		}
		log.Printf("Error creating court: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create court: %v", err))
		return
	}

	court.IsActive = true
	log.Printf("Court created successfully with ID: %d", court.ID)
	
	sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Court created successfully",
		Data:    court,
	})
}

// Booking hour handlers
func getBookingHours(w http.ResponseWriter, r *http.Request) {
	courtIDStr := r.URL.Query().Get("courtId")
	log.Printf("Getting booking hours with court ID filter: '%s'", courtIDStr)
	
	var query string
	var args []interface{}
	
	if courtIDStr != "" {
		courtID, err := strconv.Atoi(courtIDStr)
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, "Invalid court ID")
			return
		}
		query = "SELECT id, court_id, date_start, date_end, status, created_at, updated_at FROM booking_hours WHERE court_id = $1 ORDER BY date_start DESC"
		args = append(args, courtID)
	} else {
		query = "SELECT id, court_id, date_start, date_end, status, created_at, updated_at FROM booking_hours ORDER BY date_start DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying booking hours: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch booking hours: %v", err))
		return
	}
	defer rows.Close()

	var bookingHours []BookingHour
	for rows.Next() {
		var bh BookingHour
		err := rows.Scan(&bh.ID, &bh.CourtID, &bh.DateStart, &bh.DateEnd, &bh.Status, &bh.CreatedAt, &bh.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning booking hour: %v", err)
			continue
		}
		bookingHours = append(bookingHours, bh)
	}

	log.Printf("Found %d booking hours", len(bookingHours))

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Booking hours retrieved successfully (%d found)", len(bookingHours)),
		Data:    bookingHours,
	})
}

func createBookingHour(w http.ResponseWriter, r *http.Request) {
	var bh BookingHour
	if err := json.NewDecoder(r.Body).Decode(&bh); err != nil {
		log.Printf("Error decoding booking hour JSON: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if bh.CourtID == 0 {
		sendErrorResponse(w, http.StatusBadRequest, "Court ID is required")
		return
	}

	if bh.DateStart.IsZero() || bh.DateEnd.IsZero() {
		sendErrorResponse(w, http.StatusBadRequest, "Date start and date end are required")
		return
	}

	log.Printf("Creating booking hour for court %d from %v to %v", bh.CourtID, bh.DateStart, bh.DateEnd)

	// Verify court exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM courts WHERE id = $1 AND is_active = true)", bh.CourtID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking court existence: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to verify court: %v", err))
		return
	}
	if !exists {
		log.Printf("Court %d not found or inactive", bh.CourtID)
		sendErrorResponse(w, http.StatusBadRequest, "Court not found or inactive")
		return
	}

	if bh.Status == "" {
		bh.Status = "active"
	}

	query := `INSERT INTO booking_hours (court_id, date_start, date_end, status) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`
	err = db.QueryRow(query, bh.CourtID, bh.DateStart, bh.DateEnd, bh.Status).Scan(&bh.ID, &bh.CreatedAt, &bh.UpdatedAt)
	if err != nil {
		log.Printf("Error creating booking hour: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create booking hour: %v", err))
		return
	}

	log.Printf("Booking hour created successfully with ID: %d", bh.ID)

	sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Booking hour created successfully",
		Data:    bh,
	})
}

// Clip handlers
func uploadClip(w http.ResponseWriter, r *http.Request) {
	log.Printf("Starting clip upload")
	
	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get booking hour ID
	bookingHourIDStr := r.FormValue("bookingHourId")
	if bookingHourIDStr == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Booking hour ID is required")
		return
	}

	bookingHourID, err := strconv.Atoi(bookingHourIDStr)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid booking hour ID")
		return
	}

	log.Printf("Upload for booking hour ID: %d", bookingHourID)

	// Verify booking hour exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM booking_hours WHERE id = $1)", bookingHourID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking booking hour existence: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to verify booking hour: %v", err))
		return
	}
	if !exists {
		log.Printf("Booking hour %d not found", bookingHourID)
		sendErrorResponse(w, http.StatusBadRequest, "Booking hour not found")
		return
	}

	// Get the uploaded file
	file, handler, err := r.FormFile("video")
	if err != nil {
		log.Printf("Error getting uploaded file: %v", err)
		sendErrorResponse(w, http.StatusBadRequest, "No video file provided")
		return
	}
	defer file.Close()

	log.Printf("Received file: %s, size: %d bytes", handler.Filename, handler.Size)

	// Create upload directory if it doesn't exist
	config := loadConfig()
	uploadDir := filepath.Join(config.UploadDir, "clips")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Error creating upload directory: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	// Generate unique filename
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(handler.Filename)
	filename := fmt.Sprintf("clip_%d_%s%s", bookingHourID, timestamp, ext)
	filePath := filepath.Join(uploadDir, filename)

	log.Printf("Saving file to: %s", filePath)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer dst.Close()

	fileSize, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to save file")
		return
	}

	log.Printf("File saved successfully, size: %d bytes", fileSize)

	// Get MIME type
	mimeType := handler.Header.Get("Content-Type")
	if mimeType == "" {
		// Detect MIME type based on extension
		switch strings.ToLower(ext) {
		case ".mp4":
			mimeType = "video/mp4"
		case ".avi":
			mimeType = "video/x-msvideo"
		case ".webm":
			mimeType = "video/webm"
		default:
			mimeType = "video/mp4"
		}
	}

	// Get additional metadata from form
	description := r.FormValue("description")
	cameraName := r.FormValue("camera_name")
	if cameraName == "" && description != "" {
		parts := strings.Split(description, " ")
		for i, part := range parts {
			if part == "camera" && i > 0 {
				cameraName = parts[i-1]
				break
			}
		}
	}

	// Save clip metadata to database
	clip := Clip{
		BookingHourID: bookingHourID,
		Filename:      filename,
		FilePath:      filePath,
		FileSize:      &fileSize,
		MimeType:      &mimeType,
		UploadStatus:  "uploaded",
	}

	if cameraName != "" {
		clip.CameraName = &cameraName
	}

	query := `INSERT INTO clips (booking_hour_id, filename, file_path, file_size, mime_type, camera_name, upload_status) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`
	err = db.QueryRow(query, clip.BookingHourID, clip.Filename, clip.FilePath, clip.FileSize, clip.MimeType, clip.CameraName, clip.UploadStatus).
		Scan(&clip.ID, &clip.CreatedAt, &clip.UpdatedAt)
	if err != nil {
		log.Printf("Error saving clip metadata: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save clip metadata: %v", err))
		return
	}

	log.Printf("Clip uploaded successfully with ID: %d", clip.ID)

	sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Clip uploaded successfully",
		Data:    clip,
	})
}

func getClips(w http.ResponseWriter, r *http.Request) {
	bookingHourIDStr := r.URL.Query().Get("bookingHourId")
	log.Printf("Getting clips with booking hour ID filter: '%s'", bookingHourIDStr)
	
	var query string
	var args []interface{}
	
	if bookingHourIDStr != "" {
		bookingHourID, err := strconv.Atoi(bookingHourIDStr)
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, "Invalid booking hour ID")
			return
		}
		query = `SELECT id, booking_hour_id, filename, file_path, file_size, mime_type, 
				 duration_seconds, camera_name, upload_status, created_at, updated_at 
				 FROM clips WHERE booking_hour_id = $1 ORDER BY created_at DESC`
		args = append(args, bookingHourID)
	} else {
		query = `SELECT id, booking_hour_id, filename, file_path, file_size, mime_type, 
				 duration_seconds, camera_name, upload_status, created_at, updated_at 
				 FROM clips ORDER BY created_at DESC`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying clips: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch clips: %v", err))
		return
	}
	defer rows.Close()

	var clips []Clip
	for rows.Next() {
		var clip Clip
		err := rows.Scan(&clip.ID, &clip.BookingHourID, &clip.Filename, &clip.FilePath,
			&clip.FileSize, &clip.MimeType, &clip.Duration, &clip.CameraName,
			&clip.UploadStatus, &clip.CreatedAt, &clip.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning clip: %v", err)
			continue
		}
		clips = append(clips, clip)
	}

	log.Printf("Found %d clips", len(clips))

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Clips retrieved successfully (%d found)", len(clips)),
		Data:    clips,
	})
}

// Health check
func healthCheck(w http.ResponseWriter, r *http.Request) {
	// Test database connection
	var dbStatus string
	if err := db.Ping(); err != nil {
		dbStatus = "disconnected"
		log.Printf("Database health check failed: %v", err)
	} else {
		dbStatus = "connected"
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Service is healthy",
		Data: map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "1.0.0",
			"database":  dbStatus,
		},
	})
}

func main() {
	// Load configuration
	config := loadConfig()

	log.Printf("Starting CCTV Backend API Server")
	log.Printf("Database: %s:%s/%s", config.DBHost, config.DBPort, config.DBName)
	log.Printf("Upload Directory: %s", config.UploadDir)

	// Initialize database
	initDatabase(config)
	defer db.Close()

	// Create upload directory
	if err := os.MkdirAll(filepath.Join(config.UploadDir, "clips"), 0755); err != nil {
		log.Printf("Warning: Could not create upload directory: %v", err)
	}

	// Create router
	router := mux.NewRouter()

	// Add logging middleware
	router.Use(loggingMiddleware)

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", healthCheck).Methods("GET")

	// Court routes
	api.HandleFunc("/courts", getCourts).Methods("GET")
	api.HandleFunc("/courts", createCourt).Methods("POST")

	// Booking hour routes
	api.HandleFunc("/booking-hours", getBookingHours).Methods("GET")
	api.HandleFunc("/booking-hours", createBookingHour).Methods("POST")

	// Clip routes
	api.HandleFunc("/clips", uploadClip).Methods("POST")
	api.HandleFunc("/clips", getClips).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		Debug:            true,
	})

	handler := c.Handler(router)

	// Start server
	log.Printf("Server starting on port %s", config.ServerPort)
	log.Printf("API endpoints:")
	log.Printf("  Health: GET /api/v1/health")
	log.Printf("  Courts: GET/POST /api/v1/courts")
	log.Printf("  Booking Hours: GET/POST /api/v1/booking-hours")
	log.Printf("  Clips: GET/POST /api/v1/clips")
	
	if err := http.ListenAndServe(":"+config.ServerPort, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}