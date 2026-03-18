package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	images := http.FileServer(http.Dir("./static/images/"))
	http.Handle("/images/", http.StripPrefix("/images/", withCORS(withCache(images, 86400), "*")))

	videos := http.FileServer(http.Dir("./static/videos/"))
	http.Handle("/videos/", http.StripPrefix("/videos/", withCORS(withCache(videos, 86400), "*")))

	audio := http.FileServer(http.Dir("./static/audio/"))
	http.Handle("/audio/", http.StripPrefix("/audio/", withCORS(withCache(audio, 86400), "*")))

	// Translation endpoints
	http.HandleFunc("/translate", translateHandler)
	http.HandleFunc("/pronounce", pronounceHandler)
	http.HandleFunc("/translator", demoHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "English to Chinese Translator API. Use /translate for translation and /pronounce for pronunciation.")
	})

	// Get port from environment variable (Railway provides this)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default for local development
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func withCache(h http.Handler, maxAge int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
		h.ServeHTTP(w, r)
	})
}

func withCORS(h http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
