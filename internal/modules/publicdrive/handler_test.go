package publicdrive

import "testing"

func TestPublicPreviewable(t *testing.T) {
	tests := map[string]bool{
		"application/pdf":           true,
		"image/jpeg":                true,
		"text/plain; charset=utf-8": true,
		"video/mp4":                 true,
		"text/html":                 false,
		"application/octet-stream":  false,
		"application/x-msdownload":  false,
	}
	for mimeType, want := range tests {
		if got := publicPreviewable(mimeType); got != want {
			t.Errorf("publicPreviewable(%q) = %v, want %v", mimeType, got, want)
		}
	}
}
