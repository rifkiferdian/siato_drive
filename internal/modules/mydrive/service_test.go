package mydrive

import "testing"

func TestSafeName(t *testing.T) {
	for _, input := range []string{"../secret.txt", `..\secret.txt`, ".", "..", ""} {
		if got := safeName(input); got != "" {
			t.Fatalf("safeName(%q) = %q", input, got)
		}
	}
	if got := safeName(" laporan.pdf "); got != "laporan.pdf" {
		t.Fatalf("safe name = %q", got)
	}
}

func TestBlockedUpload(t *testing.T) {
	if !blocked("virus.exe", "application/octet-stream") {
		t.Fatal("executable extension accepted")
	}
	if !blocked("file.bin", "application/x-msdownload") {
		t.Fatal("executable MIME accepted")
	}
	if blocked("laporan.pdf", "application/pdf") {
		t.Fatal("PDF rejected")
	}
}

func TestPreviewableTypes(t *testing.T) {
	for _, mimeType := range []string{"application/pdf", "image/jpeg", "image/png", "image/webp", "video/mp4", "text/plain; charset=utf-8"} {
		if !previewable(mimeType) {
			t.Errorf("%s should be previewable", mimeType)
		}
	}
	for _, mimeType := range []string{"text/html", "image/svg+xml", "application/javascript", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"} {
		if previewable(mimeType) {
			t.Errorf("%s must not be previewable", mimeType)
		}
	}
}
