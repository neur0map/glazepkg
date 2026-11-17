package manager

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestMavenCheckUpdatesReadsMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/junit/junit/maven-metadata.xml":
			w.Write([]byte(`<metadata><versioning><latest>4.13.2</latest><release>4.13.2</release></versioning></metadata>`))
		case "/org/example/old/maven-metadata.xml":
			w.Write([]byte(`<metadata><versioning><latest>1.1</latest></versioning></metadata>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := mavenBaseURL
	mavenBaseURL = srv.URL
	defer func() { mavenBaseURL = old }()

	got := (&Maven{}).CheckUpdates([]model.Package{
		{Name: "junit", Version: "4.12", Description: "junit:junit"},
		{Name: "old", Version: "1.0", Description: "org.example:old"},
		{Name: "missing", Version: "1.0", Description: "org.example:missing"},
	})
	want := map[string]string{"junit": "4.13.2", "old": "1.1"}
	if len(got) != len(want) || got["junit"] != want["junit"] || got["old"] != want["old"] {
		t.Fatalf("got %v, want %v", got, want)
	}
}
