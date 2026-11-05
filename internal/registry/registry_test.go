package registry

import (
	"net/http"
	"net/http/httptest"
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

func newIndexClient(t *testing.T, handler http.HandlerFunc) (Client, *atomic.Int32) {
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
	client, hits := newIndexClient(t, serveSitemaps)
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
	client, _ := newIndexClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	client, hits := newIndexClient(t, serveSitemaps)
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
	client, _ := newIndexClient(t, serveSitemaps)
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
	client, _ := newIndexClient(t, failEverything)
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
	client, _ := newIndexClient(t, failEverything)
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
