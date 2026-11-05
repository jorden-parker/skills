package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const sitemapIndexXML = `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://www.skills.sh/sitemap-skills-1.xml</loc></sitemap>
  <sitemap><loc>https://www.skills.sh/sitemap-skills-2.xml</loc></sitemap>
  <sitemap><loc>https://www.skills.sh/sitemap-owners.xml</loc></sitemap>
</sitemapindex>`

const firstSkillSitemapXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://www.skills.sh/acme/tools/lint</loc></url>
  <url><loc>https://www.skills.sh/obra/superpowers-skills/test-driven-development-(tdd)</loc></url>
  <url><loc>https://www.skills.sh/acme/tools</loc></url>
  <url><loc>https://www.skills.sh/a/../b</loc></url>
</urlset>`

const secondSkillSitemapXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://www.skills.sh/acme/tools/lint</loc></url>
  <url><loc>https://www.skills.sh/acme/tools/alpha</loc></url>
</urlset>`

var wantIndex = []string{
	"acme/tools/alpha",
	"acme/tools/lint",
	"obra/superpowers-skills/test-driven-development-(tdd)",
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (Client, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	client := Client{
		BaseURL:   server.URL,
		IndexPath: filepath.Join(t.TempDir(), "cache", "skills-sh-index.txt"),
		HTTP:      server.Client(),
		Now:       time.Now,
	}
	return client, &hits
}

func serveSitemaps(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sitemap.xml":
		_, _ = w.Write([]byte(sitemapIndexXML))
	case "/sitemap-skills-1.xml":
		_, _ = w.Write([]byte(firstSkillSitemapXML))
	case "/sitemap-skills-2.xml":
		_, _ = w.Write([]byte(secondSkillSitemapXML))
	default:
		http.Error(w, "unexpected "+r.URL.Path, http.StatusInternalServerError)
	}
}

func failEverything(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "down", http.StatusInternalServerError)
}

func TestIndexFetchesSortedValidIDsAndCachesThem(t *testing.T) {
	client, hits := newTestClient(t, serveSitemaps)
	got, err := client.Index()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, wantIndex) {
		t.Fatalf("Index = %v, want %v", got, wantIndex)
	}
	data, err := os.ReadFile(client.IndexPath)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(data)), "\n"); !reflect.DeepEqual(lines, wantIndex) {
		t.Fatalf("cache lines = %v", lines)
	}
	if hits.Load() == 0 {
		t.Fatal("no requests were made")
	}
}

func TestIndexNeverFetchesOwnersSitemap(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sitemap-owners.xml" {
			t.Error("owners sitemap was fetched")
		}
		serveSitemaps(w, r)
	})
	if _, err := client.Index(); err != nil {
		t.Fatal(err)
	}
}

func TestIndexUsesFreshCacheWithoutRequests(t *testing.T) {
	client, hits := newTestClient(t, serveSitemaps)
	writeCache(t, client.IndexPath, []string{"cached/one/skill"}, time.Now())
	got, err := client.Index()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"cached/one/skill"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Index = %v, want %v", got, want)
	}
	if hits.Load() != 0 {
		t.Fatalf("requests = %d, want 0", hits.Load())
	}
}

func TestIndexRefetchesStaleCache(t *testing.T) {
	client, _ := newTestClient(t, serveSitemaps)
	writeCache(t, client.IndexPath, []string{"old/one/skill"}, time.Now().Add(-25*time.Hour))
	got, err := client.Index()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, wantIndex) {
		t.Fatalf("Index = %v, want %v", got, wantIndex)
	}
}

func TestIndexKeepsStaleCacheWhenServerFails(t *testing.T) {
	client, _ := newTestClient(t, failEverything)
	writeCache(t, client.IndexPath, []string{"old/one/skill"}, time.Now().Add(-25*time.Hour))
	got, err := client.Index()
	if err != nil {
		t.Fatalf("Index error = %v", err)
	}
	if want := []string{"old/one/skill"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Index = %v, want %v", got, want)
	}
}

func TestIndexFailsWithoutCacheWhenServerFails(t *testing.T) {
	client, _ := newTestClient(t, failEverything)
	_, err := client.Index()
	if err == nil || !strings.Contains(err.Error(), "skills.sh") {
		t.Fatalf("Index error = %v", err)
	}
}

func writeCache(t *testing.T, indexPath string, ids []string, modified time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(strings.Join(ids, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(indexPath, modified, modified); err != nil {
		t.Fatal(err)
	}
}

const lintID = "acme/tools/lint"

var lintFiles = []File{
	{Path: "SKILL.md", Contents: "---\nname: lint\n---\n"},
	{Path: "rules/a.md", Contents: "rule"},
}

func serveFiles(files []File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(map[string]any{"files": files, "hash": "abc"})
		_, _ = w.Write(body)
	}
}

func TestFilesReturnsEveryFile(t *testing.T) {
	client, _ := newTestClient(t, serveFiles(lintFiles))
	got, err := client.Files(lintID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, lintFiles) {
		t.Fatalf("Files = %v, want %v", got, lintFiles)
	}
}

func TestFilesEscapesEachSegment(t *testing.T) {
	var gotPath string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		serveFiles(lintFiles)(w, r)
	})
	id := "obra/superpowers-skills/test-driven-development-(tdd)"
	if _, err := client.Files(id); err != nil {
		t.Fatal(err)
	}
	want := "/api/download/obra/superpowers-skills/" + url.PathEscape("test-driven-development-(tdd)")
	if gotPath != want {
		t.Fatalf("path = %s, want %s", gotPath, want)
	}
}

func TestManifestReturnsSkillFile(t *testing.T) {
	client, _ := newTestClient(t, serveFiles(lintFiles))
	got, err := client.Manifest(lintID)
	if err != nil {
		t.Fatal(err)
	}
	if got != lintFiles[0].Contents {
		t.Fatalf("Manifest = %q", got)
	}
}

func TestManifestNeedsSkillFile(t *testing.T) {
	client, _ := newTestClient(t, serveFiles([]File{{Path: "rules/a.md", Contents: "rule"}}))
	_, err := client.Manifest(lintID)
	if err == nil || !strings.Contains(err.Error(), "no SKILL.md") {
		t.Fatalf("Manifest error = %v", err)
	}
}

func TestDownloadWritesFilesWithoutExecuteBits(t *testing.T) {
	client, _ := newTestClient(t, serveFiles(lintFiles))
	dest := filepath.Join(t.TempDir(), "registry", "acme", "tools", "lint")
	if err := client.Download(lintID, dest); err != nil {
		t.Fatal(err)
	}
	nested, err := os.ReadFile(filepath.Join(dest, "rules", "a.md"))
	if err != nil || string(nested) != "rule" {
		t.Fatalf("rules/a.md = %q, %v", nested, err)
	}
	fi, err := os.Stat(filepath.Join(dest, "SKILL.md"))
	if err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("SKILL.md mode = %v, %v", fi.Mode(), err)
	}
	di, err := os.Stat(dest)
	if err != nil || di.Mode().Perm() != 0o755 {
		t.Fatalf("dest mode = %v, %v", di.Mode(), err)
	}
}

func TestDownloadRejectsUnsafeFilesAndLeavesNothing(t *testing.T) {
	cases := map[string][]File{
		"../evil":     {{Path: "SKILL.md", Contents: "x"}, {Path: "../evil", Contents: "x"}},
		"/etc/x":      {{Path: "SKILL.md", Contents: "x"}, {Path: "/etc/x", Contents: "x"}},
		"a/../../x":   {{Path: "SKILL.md", Contents: "x"}, {Path: "a/../../x", Contents: "x"}},
		"":            {{Path: "SKILL.md", Contents: "x"}, {Path: "", Contents: "x"}},
		"no SKILL.md": {{Path: "rules/a.md", Contents: "rule"}},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			client, _ := newTestClient(t, serveFiles(files))
			parent := filepath.Join(t.TempDir(), "owner", "repo")
			dest := filepath.Join(parent, "lint")
			err := client.Download(lintID, dest)
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("Download error = %v", err)
			}
			if _, statErr := os.Lstat(dest); !os.IsNotExist(statErr) {
				t.Fatalf("dest exists: %v", statErr)
			}
			entries, _ := os.ReadDir(parent)
			if len(entries) != 0 {
				t.Fatalf("parent is not empty: %v", entries)
			}
		})
	}
}

func TestDownloadReplacesExistingCopy(t *testing.T) {
	client, _ := newTestClient(t, serveFiles(lintFiles))
	dest := filepath.Join(t.TempDir(), "lint")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "stale.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := client.Download(lintID, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "stale.md")); !os.IsNotExist(err) {
		t.Fatalf("stale.md still present: %v", err)
	}
}

func TestDownloadReportsMissingSkill(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	err := client.Download("acme/tools/missing", filepath.Join(t.TempDir(), "missing"))
	if err == nil || !strings.Contains(err.Error(), "not found on skills.sh") {
		t.Fatalf("Download error = %v", err)
	}
}

func TestFilesRejectsNonRegistryIDWithoutRequest(t *testing.T) {
	client, hits := newTestClient(t, serveFiles(lintFiles))
	if _, err := client.Files("me/alpha"); err == nil {
		t.Fatal("Files(me/alpha) returned no error")
	}
	if hits.Load() != 0 {
		t.Fatalf("requests = %d, want 0", hits.Load())
	}
}
