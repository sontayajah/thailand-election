// DOPA mock server — simulates Thai Department of Provincial Administration ID verification.
// In dev: accepts any 13-digit numeric national_id as valid.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

var nationalIDRegex = regexp.MustCompile(`^\d{13}$`)

type verifyRequest struct {
	NationalID string `json:"national_id"`
}

type verifyResponse struct {
	Valid     bool   `json:"valid"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/verify", verifyHandler)

	log.Printf("[dopa-mock] listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok","service":"dopa-mock"}`)
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(verifyResponse{Valid: false, Status: "invalid_request"})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if !nationalIDRegex.MatchString(req.NationalID) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(verifyResponse{
			Valid:     false,
			Status:    "invalid_format",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Mock: always valid for any 13-digit ID
	log.Printf("[dopa-mock] verified national_id=****%s", req.NationalID[len(req.NationalID)-4:])
	json.NewEncoder(w).Encode(verifyResponse{
		Valid:     true,
		Status:    "verified",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
