package shareddrive

import "testing"

func TestManagerSafeName(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../secret.txt", `..\secret.txt`} {
		if managerSafeName(name) != "" {
			t.Errorf("unsafe name accepted: %q", name)
		}
	}
	if got := managerSafeName("Dokumen Tim.pdf"); got != "Dokumen Tim.pdf" {
		t.Fatalf("name = %q", got)
	}
}
func TestManagerPreviewable(t *testing.T) {
	if !managerPreviewable("application/pdf") {
		t.Fatal("PDF should be previewable")
	}
	if managerPreviewable("text/html") {
		t.Fatal("HTML preview must be blocked")
	}
}
