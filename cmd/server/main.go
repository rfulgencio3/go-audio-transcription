// Package main is the entry point for the audio transcription server.
//
//	@title			Audio Transcription API
//	@version		1.0
//	@description	POC: receives audio, transcribes and analyzes with Google Gemini, persists in RavenDB.
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

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/rfulgencio3/go-audio-transcription/config"
	_ "github.com/rfulgencio3/go-audio-transcription/docs"
	"github.com/rfulgencio3/go-audio-transcription/internal/ai"
	"github.com/rfulgencio3/go-audio-transcription/internal/handler"
	"github.com/rfulgencio3/go-audio-transcription/internal/storage"
	"github.com/rfulgencio3/go-audio-transcription/internal/transcription"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// --- Build dependencies ---

	// Transcription + analysis: Google Gemini
	var transcriber transcription.Transcriber
	var analyzer ai.Analyzer
	if cfg.Gemini.APIKey == "" {
		log.Printf("warn: GEMINI_API_KEY is not set; /transcribe pipeline is disabled")
		transcriber = transcription.NewDisabledTranscriber("GEMINI_API_KEY is not set")
		analyzer = ai.NewDisabledAnalyzer("GEMINI_API_KEY is not set")
	} else {
		initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
		geminiTranscriber, err := transcription.NewGeminiTranscriber(initCtx, cfg.Gemini.APIKey, cfg.Gemini.ModelName)
		if err != nil {
			initCancel()
			log.Fatalf("gemini transcription init error: %v", err)
		}
		gemini, err := ai.NewGeminiAnalyzer(initCtx, cfg.Gemini.APIKey, cfg.Gemini.ModelName)
		initCancel()
		if err != nil {
			_ = geminiTranscriber.Close()
			log.Fatalf("gemini init error: %v", err)
		}
		transcriber = geminiTranscriber
		analyzer = gemini
		defer func() {
			if err := geminiTranscriber.Close(); err != nil {
				log.Printf("warn: closing Gemini transcription client: %v", err)
			}
		}()
		defer func() {
			if err := gemini.Close(); err != nil {
				log.Printf("warn: closing Gemini client: %v", err)
			}
		}()
	}

	// Storage: RavenDB
	repo, err := storage.NewRavenDBRepository(cfg.RavenDB.URLs, cfg.RavenDB.DatabaseName)
	if err != nil {
		log.Fatalf("ravendb init error: %v", err)
	}
	defer repo.Close()

	// --- Wire HTTP routes ---
	h := handler.NewHandler(transcriber, analyzer, repo, cfg.Server.MaxUploadBytes)

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
