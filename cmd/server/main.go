// Package main is the entry point for the audio transcription server.
//
//	@title			Audio Transcription API
//	@version		1.0
//	@description	POC: receives audio, transcribes with OpenAI Whisper, analyzes with Google Gemini, persists in RavenDB.
//	@host			localhost:8080
//	@BasePath		/
//	@accept			multipart/form-data
//	@produce		application/json
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/rfulgencio3/go-audio-transcription/config"
	_ "github.com/rfulgencio3/go-audio-transcription/docs"
	"github.com/rfulgencio3/go-audio-transcription/internal/ai"
	"github.com/rfulgencio3/go-audio-transcription/internal/handler"
	"github.com/rfulgencio3/go-audio-transcription/internal/storage"
	"github.com/rfulgencio3/go-audio-transcription/internal/transcription"
)

func main() {
	// Load .env in development — ignored if the file does not exist.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warn: could not load .env file: %v", err)
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// --- Build dependencies ---

	// Transcription: OpenAI Whisper
	whisper := transcription.NewWhisperTranscriber(cfg.OpenAI.APIKey)

	// AI Analysis: Google Gemini
	initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
	gemini, err := ai.NewGeminiAnalyzer(initCtx, cfg.Gemini.APIKey, cfg.Gemini.ModelName)
	initCancel()
	if err != nil {
		log.Fatalf("gemini init error: %v", err)
	}
	defer func() {
		if err := gemini.Close(); err != nil {
			log.Printf("warn: closing Gemini client: %v", err)
		}
	}()

	// Storage: RavenDB
	repo, err := storage.NewRavenDBRepository(cfg.RavenDB.URLs, cfg.RavenDB.DatabaseName)
	if err != nil {
		log.Fatalf("ravendb init error: %v", err)
	}
	defer repo.Close()

	// --- Wire HTTP routes ---
	h := handler.NewHandler(whisper, gemini, repo, cfg.Server.MaxUploadBytes)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /transcribe", h.Transcribe)
	mux.HandleFunc("GET /transcriptions", h.ListTranscriptions)
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// --- Start server in background goroutine with bounded lifecycle ---
	go func() {
		log.Printf("server listening on %s", cfg.Server.Addr)
		log.Printf("swagger UI: http://localhost%s/swagger/index.html", cfg.Server.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// --- Graceful shutdown on SIGINT / SIGTERM ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}
