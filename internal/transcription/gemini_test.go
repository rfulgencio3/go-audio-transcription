package transcription

import "testing"

func TestDetectAudioMIMEType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "mp3", filename: "sample.mp3", want: "audio/mpeg"},
		{name: "m4a", filename: "sample.m4a", want: "audio/mp4"},
		{name: "wav", filename: "sample.wav", want: "audio/wav"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := detectAudioMIMEType(tc.filename, nil)
			if got != tc.want {
				t.Fatalf("detectAudioMIMEType(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}
