// Package main is the entry point for the audio transcription server.
//
//	@title			Audio Transcription API
//	@version		1.0
//	@description	POC: receives audio, captures the complete transcript with Google Gemini, and can optionally enrich the transcript with extra analysis.
//	@host			example.com
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
	"github.com/rfulgencio3/go-audio-transcription/docs"
	"github.com/rfulgencio3/go-audio-transcription/internal/ai"
	"github.com/rfulgencio3/go-audio-transcription/internal/handler"
	"github.com/rfulgencio3/go-audio-transcription/internal/transcription"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	configureSwagger(cfg)

	// --- Build dependencies ---

	// Transcription + optional analysis: Google Gemini
	var transcriber transcription.Transcriber
	var analyzer ai.Analyzer
	if cfg.Gemini.APIKey == "" {
		log.Printf("warn: GEMINI_API_KEY is not set; /transcribe pipeline is disabled")
		transcriber = transcription.NewDisabledTranscriber("GEMINI_API_KEY is not set")
		analyzer = nil
	} else {
		initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
		geminiTranscriber, err := transcription.NewGeminiTranscriber(initCtx, cfg.Gemini.APIKey, cfg.Gemini.ModelName)
		if err != nil {
			initCancel()
			log.Fatalf("gemini transcription init error: %v", err)
		}
		transcriber = geminiTranscriber
		defer func() {
			if err := geminiTranscriber.Close(); err != nil {
				log.Printf("warn: closing Gemini transcription client: %v", err)
			}
		}()
		initCancel()

		if cfg.Feature.EnableTranscriptAnalysis {
			initCtx, initCancel = context.WithTimeout(context.Background(), 15*time.Second)
			gemini, err := ai.NewGeminiAnalyzer(initCtx, cfg.Gemini.APIKey, cfg.Gemini.ModelName)
			initCancel()
			if err != nil {
				_ = geminiTranscriber.Close()
				log.Fatalf("gemini analysis init error: %v", err)
			}
			analyzer = gemini
			defer func() {
				if err := gemini.Close(); err != nil {
					log.Printf("warn: closing Gemini client: %v", err)
				}
			}()
		} else {
			log.Printf("info: transcript analysis is disabled; returning transcript only")
			analyzer = nil
		}
	}

	// --- Wire HTTP routes ---
	h := handler.NewHandler(transcriber, analyzer, cfg.Server.MaxUploadBytes)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /transcribe", h.Transcribe)
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
		if cfg.Public.BaseURL != "" {
			log.Printf("swagger UI: %s/swagger/index.html", cfg.Public.BaseURL)
		} else {
			log.Printf("swagger UI: http://localhost%s/swagger/index.html", cfg.Server.Addr)
		}
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

func configureSwagger(cfg config.Config) {
	docs.SwaggerInfo.BasePath = "/"

	if cfg.Public.Host != "" {
		docs.SwaggerInfo.Host = cfg.Public.Host
		if cfg.Public.Scheme != "" {
			docs.SwaggerInfo.Schemes = []string{cfg.Public.Scheme}
		}
		return
	}

	docs.SwaggerInfo.Host = "localhost" + cfg.Server.Addr
	docs.SwaggerInfo.Schemes = []string{"http"}
}
