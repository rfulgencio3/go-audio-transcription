# go-audio-transcription

A Go POC for end-to-end audio transcription with document persistence.

## Overview

This project validates a complete audio transcription pipeline:

**Receive audio -> Transcribe full content with Gemini -> Persist**

## Stack

| Layer        | Technology                                   |
|--------------|----------------------------------------------|
| HTTP Server  | Go standard `net/http`                       |
| Transcription| Google Gemini API                            |
| Database     | MongoDB                                      |
| API Docs     | Swagger via `swaggo/swag`                    |
| Config       | Minimal environment variables                |

## Architecture

Audio transcription records are naturally document-shaped: a single record aggregates filename, the complete transcript, and optional enrichments with no relational joins. MongoDB stores the document exactly as the domain represents it, without schema migrations.

```text
POST /transcribe  (multipart: field "audio")
  |- Validate file size and content-type
  |- Transcribe audio -> full text (Gemini)
  |- Optionally enrich transcript  (Gemini)
  |- Persist document          (MongoDB)
  `- Return JSON 201
```

## Endpoints

| Method | Path             | Description                           |
|--------|------------------|---------------------------------------|
| GET    | /health          | Service healthcheck                   |
| POST   | /transcribe      | Upload audio -> full transcript        |
| GET    | /transcriptions  | List paginated transcriptions         |
| GET    | /swagger/*       | Swagger UI                            |

## Getting Started

### Prerequisites

- Go 1.21+
- MongoDB instance (local or cloud - [Railway](https://railway.app) recommended)
- Google Gemini API key

### Configuration

Configure only the essential variables in the Railway service settings:

| Variable             | Description                        | Default                 |
|----------------------|------------------------------------|-------------------------|
| `PORT`               | Platform-provided port fallback    | optional                |
| `GEMINI_API_KEY`     | Google Gemini API key              | optional at startup, required for `/transcribe` |
| `MONGODB_URI`        | MongoDB connection URI             | required                |

`MONGO_URL` is also accepted as a fallback for `MONGODB_URI` when you want to reference the Mongo service variable directly.

If `GEMINI_API_KEY` is missing, the server still starts so the container does not enter a restart loop, but `POST /transcribe` returns `503 Service Unavailable` until Gemini is configured.

All other runtime settings use internal defaults:

- listen address: `:$PORT` or `:8080`
- upload limit: `25MB`
- Gemini model: `gemini-1.5-flash`
- MongoDB database: `AudioTranscriptions`
- MongoDB collection: `transcriptions`
- Swagger public domain: inferred from Railway `RAILWAY_PUBLIC_DOMAIN` when available

### Run

```bash
# Install swag CLI (first time only)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger docs
swag init -g cmd/server/main.go --output docs

# Run the server
go run ./cmd/server/
```

### API Docs

Open [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) in your browser.

### Transcribe an audio file

```bash
curl -X POST http://localhost:8080/transcribe \
  -F "audio=@path/to/your/audio.mp3"
```

### Example response

```json
{
  "Id": "67e3e8f6b4f54d9f0cf0aa11",
  "audioFilename": "meeting.mp3",
  "fileSizeBytes": 2097152,
  "transcript": "Hello, this is a test recording...",
  "language": "en",
  "audioDuration": 47.3,
  "createdAt": "2026-03-25T14:30:00Z"
}
```

## Development

```bash
# Run tests with race detector
go test -race ./...

# Lint
golangci-lint run ./...
```

## Supported Audio Formats

`mp3`, `mp4`, `mpeg`, `mpga`, `m4a`, `wav`, `webm`, `ogg`, `flac`

## License

MIT
