# go-audio-transcription

A Go POC for end-to-end audio transcription with AI analysis and document persistence.

## Overview

This project validates a complete audio transcription pipeline:

**Receive audio → Transcribe (Whisper) → Analyze with AI (Gemini) → Persist (RavenDB)**

## Stack

| Layer        | Technology                                   |
|--------------|----------------------------------------------|
| HTTP Server  | Go standard `net/http`                       |
| Transcription| OpenAI Whisper API                           |
| AI Analysis  | Google Gemini API                            |
| Database     | RavenDB (document store)                     |
| API Docs     | Swagger via `swaggo/swag`                    |
| Config       | Environment variables + `godotenv`           |

## Architecture

Audio transcription records are naturally document-shaped — a single record aggregates filename, transcript, key points, summary, and sentiment with no relational joins. RavenDB stores the document exactly as the domain represents it, without schema migrations.

```
POST /transcribe  (multipart: field "audio")
  ├─ Validate file size and content-type
  ├─ Transcribe audio → text  (OpenAI Whisper)
  ├─ Analyze transcript        (Google Gemini)
  ├─ Persist document          (RavenDB)
  └─ Return JSON 201
```

## Endpoints

| Method | Path             | Description                          |
|--------|------------------|--------------------------------------|
| POST   | /transcribe      | Upload audio → transcript + analysis |
| GET    | /transcriptions  | List paginated transcriptions        |
| GET    | /swagger/*       | Swagger UI                           |

## Getting Started

### Prerequisites

- Go 1.21+
- RavenDB instance (local or cloud — [Railway](https://railway.app) recommended)
- OpenAI API key
- Google Gemini API key

### Configuration

```bash
cp .env.example .env
# Edit .env with your API keys and RavenDB URL
```

| Variable             | Description                        | Default              |
|----------------------|------------------------------------|----------------------|
| `ADDR`               | HTTP listen address                | `:8080`              |
| `MAX_UPLOAD_BYTES`   | Max audio file size in bytes       | `26214400` (25MB)    |
| `OPENAI_API_KEY`     | OpenAI API key                     | required             |
| `GEMINI_API_KEY`     | Google Gemini API key              | required             |
| `GEMINI_MODEL`       | Gemini model name                  | `gemini-1.5-flash`   |
| `RAVENDB_URLS`       | Comma-separated RavenDB URLs       | `http://localhost:8080` |
| `RAVENDB_DATABASE`   | RavenDB database name              | `AudioTranscriptions` |

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
  "Id": "transcriptions/1-A",
  "audioFilename": "meeting.mp3",
  "fileSizeBytes": 2097152,
  "transcript": "Hello, this is a test recording...",
  "language": "en",
  "audioDuration": 47.3,
  "summary": "A brief test recording introducing the team.",
  "keyPoints": ["Introduction", "Team structure overview"],
  "sentiment": "positive",
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
