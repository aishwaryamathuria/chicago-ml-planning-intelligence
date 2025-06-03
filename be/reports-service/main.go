package main

import (
	"log"
	"net/http"
	"reports-service/reports"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables.")
	}
	r := mux.NewRouter()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // your frontend URL
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(r)

	r.HandleFunc("/api/report1", reports.HandleReport1).Methods("GET")
	r.HandleFunc("/api/report3", reports.HandleReport3).Methods("GET")
	r.HandleFunc("/api/report4", reports.HandleReport4).Methods("GET")
	r.HandleFunc("/api/report4a", reports.HandleReport4A).Methods("GET")
	r.HandleFunc("/api/report4b", reports.HandleReport4B).Methods("GET")

	port := ":8080"
	log.Printf("Reports service running on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, handler))
}
