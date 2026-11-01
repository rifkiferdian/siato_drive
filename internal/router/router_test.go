package router

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud-drive/internal/config"
	"cloud-drive/internal/modules/auth"
	"cloud-drive/internal/modules/mydrive"
	"cloud-drive/internal/modules/publicdrive"
	"cloud-drive/internal/modules/shareddrive"
	"cloud-drive/internal/modules/trash"
	"github.com/google/uuid"
)

func TestLoginTemplateRenders(t *testing.T) {
	tpl, err := template.ParseGlob("../../web/templates/*.html")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err = tpl.ExecuteTemplate(&output, "login.html", map[string]any{"Title": "Masuk"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte(`class="login-shell"`)) {
		t.Fatal("login layout is missing")
	}
	if !bytes.Contains(output.Bytes(), []byte(`/static/js/login.js`)) {
		t.Fatal("login behavior script is missing")
	}
}

func TestWorkspaceTemplatesRender(t *testing.T) {
	tpl, err := template.ParseGlob("../../web/templates/*.html")
	if err != nil {
		t.Fatal(err)
	}
	session := auth.Session{CSRFToken: "csrf", User: auth.User{DisplayName: "Admin", Email: "admin@example.test"}}
	tests := []struct {
		name string
		data any
	}{
		{"dashboard.html", map[string]any{"Title": "Dashboard", "Session": session}},
		{"my-drive.html", map[string]any{"Title": "Drive Saya", "Session": session, "Page": mydrive.Page{Drive: mydrive.Drive{ID: uuid.New(), Name: "Drive Saya"}, Nodes: []mydrive.Node{}}, "Query": "", "Sort": "name", "MaxUpload": int64(1024)}},
		{"public-drive-manager.html", map[string]any{"Title": "Drive Publik", "Session": session, "Page": publicdrive.ManagerPage{Drive: publicdrive.Drive{ID: uuid.New(), Name: "Drive Publik"}, Nodes: []publicdrive.Node{}}, "Query": "", "Sort": "name", "MaxUpload": int64(1024)}},
		{"shared-drive-manager.html", map[string]any{"Title": "Drive Bersama", "Session": session, "Page": shareddrive.ManagerPage{Drive: shareddrive.Drive{ID: uuid.New(), Name: "Drive Bersama"}, Nodes: []shareddrive.Node{}}, "Query": "", "Sort": "name", "MaxUpload": int64(1024)}},
		{"trash.html", map[string]any{"Title": "Sampah", "Session": session, "Items": []trash.Item{}, "Query": "", "Type": ""}},
		{"public.html", map[string]any{"Title": "Publik", "Org": publicdrive.Organization{ID: uuid.New(), Slug: "default", Name: "Organisasi"}, "Nodes": []publicdrive.Node{}}},
		{"error.html", map[string]any{"Title": "Tidak ditemukan", "Message": "Halaman tidak tersedia."}},
	}
	for _, test := range tests {
		var output bytes.Buffer
		if err = tpl.ExecuteTemplate(&output, test.name, test.data); err != nil {
			t.Fatalf("render %s: %v", test.name, err)
		}
		if output.Len() == 0 {
			t.Fatalf("render %s produced no output", test.name)
		}
	}
}

func TestRoutesRegisterWithoutConflict(t *testing.T) {
	tpl := template.Must(template.New("root").Parse(`{{define "login.html"}}ok{{end}}{{define "dashboard.html"}}ok{{end}}{{define "public.html"}}ok{{end}}{{define "error.html"}}ok{{end}}`))
	cfg := config.Config{Env: "test", CookieName: "session", SessionTTL: time.Hour}
	service := auth.NewService(nil, time.Hour)
	_ = New(cfg, tpl, auth.NewHandler(service, cfg), service, publicdrive.NewHandler(nil, nil, "default"), mydrive.NewHandler(nil, 1024), publicdrive.NewManagerHandler(nil, 1024), shareddrive.NewManagerHandler(nil, 1024), trash.NewHandler(nil))
}

func TestRootRedirectsToLogin(t *testing.T) {
	tpl := template.Must(template.New("root").Parse(`{{define "login.html"}}ok{{end}}{{define "dashboard.html"}}ok{{end}}{{define "public.html"}}ok{{end}}{{define "error.html"}}ok{{end}}`))
	cfg := config.Config{Env: "test", CookieName: "session", SessionTTL: time.Hour}
	service := auth.NewService(nil, time.Hour)
	app := New(cfg, tpl, auth.NewHandler(service, cfg), service, publicdrive.NewHandler(nil, nil, "default"), mydrive.NewHandler(nil, 1024), publicdrive.NewManagerHandler(nil, 1024), shareddrive.NewManagerHandler(nil, 1024), trash.NewHandler(nil))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	app.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("Location = %q, want /login", location)
	}
}
