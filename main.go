package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	DateAdded time.Time `json:"date_added"`
}



// JSON response helper functions

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	// Ensure the body contains only one JSON value.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain only one JSON value")
		}
		return err
	}

	return nil
}

// HTTP Handlers

func createUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input UserInput

		if err := decodeJSON(r, &input); err != nil {
			http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
			return
		}

		if input.Name == "" || input.Age <= 0 {
			http.Error(w, "Name is required and age must be positive",
				http.StatusBadRequest)
			return
		}

		user, err := InsertUser(db, input.Name, input.Age)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			http.Error(w, "Could not create user",
				http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, user)
	}
}

func getUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := GetUsers(db)
		if err != nil {
			log.Printf("Failed to retrieve users: %v", err)
			http.Error(w, "Could not retrieve users",
				http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, users)
	}
}

func getUserByIDHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id <= 0 {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		user, err := GetUserByID(db, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			log.Printf("Failed to retrieve user %d: %v", id, err)
			http.Error(w, "Could not retrieve user",
				http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

func DeleteUserByIDHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id <= 0 {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		err = DeleteUserByID(db, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			log.Printf("Failed to delete user %d: %v", id, err)
			http.Error(w, "Could not delete user",
				http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func UpdateUserByIDHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id <= 0 {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		var input UserInput
		if err := decodeJSON(r, &input); err != nil {
			http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
			return
		}

		if input.Name == "" || input.Age <= 0 {
			http.Error(w, "Name is required and age must be positive",
				http.StatusBadRequest)
			return
		}

		user, err := UpdateUserByID(db, id, input.Name, input.Age)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			log.Printf("Failed to update user %d: %v", id, err)
			http.Error(w, "Could not update user",
				http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

func main() {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword("postgres", os.Getenv("PASSWORD")),
		Host:   "localhost:5432",
		Path:   "postgres",
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to PostgreSQL")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", createUserHandler(db))
	mux.HandleFunc("GET /users", getUsersHandler(db))
	mux.HandleFunc("GET /users/{id}", getUserByIDHandler(db))
	mux.HandleFunc("DELETE /users/{id}", DeleteUserByIDHandler(db))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Server listening on :8080")
	log.Fatal(server.ListenAndServe())
}
