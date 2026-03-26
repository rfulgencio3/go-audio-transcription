# go-audio-transcription

A Go service that receives audio files, captures the full spoken transcript with Gemini, and stores the result in MongoDB.

## Overview

This project implements a simple audio transcription pipeline:

**Receive audio -> Transcribe full content with Gemini -> Optionally enrich -> Persist**

## Stack

| Layer        | Technology                                   |
|--------------|----------------------------------------------|
| HTTP Server   | Go standard `net/http`                      |
| Transcription | Google Gemini API                           |
| Persistence   | MongoDB                                     |
| API Docs      | Swagger via `swaggo/swag`                   |
| Config        | Minimal environment variables               |

## Architecture

Each transcription is stored as a single MongoDB document with:

- file metadata
- full transcript text
- optional language and duration metadata
- optional summary, key points, and sentiment
- creation timestamp

```text
POST /transcribe  (multipart: field "audio")
  |- Validate file size and content-type
  |- Transcribe audio -> full text (Gemini)
  |- Optionally enrich transcript (Gemini)
  |- Persist document             (MongoDB)
  `- Return JSON 201
```

If transcript enrichment fails, the service still saves the full transcript.

## Endpoints

| Method | Path             | Description                          |
|--------|------------------|--------------------------------------|
| GET    | /health          | Service healthcheck                  |
| POST   | /transcribe      | Upload audio and persist transcript  |
| GET    | /transcriptions  | List paginated transcriptions        |
| GET    | /swagger/*       | Swagger UI                           |

## Getting Started

### Prerequisites

- Go 1.21+
- MongoDB instance
- Google Gemini API key

### Configuration

Only two application variables are required:

| Variable         | Description                          |
|------------------|--------------------------------------|
| `GEMINI_API_KEY` | Google Gemini API key                |
| `MONGODB_URI`    | MongoDB connection URI               |

`MONGO_URL` is also accepted as a fallback for `MONGODB_URI` when you want to reference the Mongo service variable directly.
`GEMINI_MODEL` is optional when you need to override the default model.

If `GEMINI_API_KEY` is missing, the server still starts so the container does not enter a restart loop, but `POST /transcribe` returns `503 Service Unavailable`.

All other runtime settings use internal defaults:

- listen address: `:$PORT` or `:8080`
- upload limit: `25MB`
- Gemini model: `gemini-2.5-flash`
- MongoDB database: `AudioTranscriptions`
- MongoDB collection: `transcriptions`
- Swagger public domain: inferred from Railway `RAILWAY_PUBLIC_DOMAIN` when available

### Railway Example

For the `go-audio-transcription` service, a typical Railway setup is:

```env
GEMINI_API_KEY=...
GEMINI_MODEL=gemini-2.5-flash
MONGODB_URI=${{Mongo.MONGO_URL}}/?authSource=admin
```

### Run

```bash
export GEMINI_API_KEY="your-gemini-key"
export MONGODB_URI="mongodb://localhost:27017"
go run ./cmd/server
```

### API Docs

Open [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) in your browser.

### Test the API

```bash
# Healthcheck
curl http://localhost:8080/health

# Upload an audio file
curl -X POST http://localhost:8080/transcribe \
  -F "audio=@path/to/your/audio.mp3"

# List stored transcriptions
curl http://localhost:8080/transcriptions
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
  "summary": "A short summary of the recording.",
  "keyPoints": ["Greeting", "Short test recording"],
  "sentiment": "positive",
  "createdAt": "2026-03-25T14:30:00Z"
}
```

`summary`, `keyPoints`, and `sentiment` are optional. The full transcript is the primary persisted output.

## Development

```bash
go test ./...
go vet ./...
```

## Supported Audio Formats

`mp3`, `mp4`, `mpeg`, `mpga`, `m4a`, `wav`, `webm`, `ogg`, `flac`

## License

MIT
