package publicdrive

import "testing"

func TestManagerSafeName(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../secret.txt", `..\secret.txt`} {
		if managerSafeName(name) != "" {
			t.Errorf("unsafe name accepted: %q", name)
		}
	}
	if got := managerSafeName("Laporan Publik.pdf"); got != "Laporan Publik.pdf" {
		t.Fatalf("name = %q", got)
	}
}
func TestManagerPreviewable(t *testing.T) {
	if !managerPreviewable("application/pdf") {
		t.Fatal("PDF should be previewable")
	}
	for _, mimeType := range []string{"text/html", "image/svg+xml", "application/javascript"} {
		if managerPreviewable(mimeType) {
			t.Errorf("unsafe preview accepted: %s", mimeType)
		}
	}
}
