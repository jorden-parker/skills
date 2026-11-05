package registry

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jorden-parker/skills/internal/atomicfile"
	"github.com/jorden-parker/skills/internal/skills"
)

const (
	indexMaxAge      = 24 * time.Hour
	maxResponseBytes = 16 << 20
	skillSitemapGlob = "sitemap-skills-*.xml"
)

type Client struct {
	BaseURL   string
	IndexPath string
	HTTP      *http.Client
	Now       func() time.Time
}

func New(p skills.Paths) Client {
	return Client{
		BaseURL:   "https://www.skills.sh",
		IndexPath: p.RegistryIndex,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		Now:       time.Now,
	}
}

func (c Client) Index() ([]string, error) {
	info, statErr := os.Stat(c.IndexPath)
	if statErr == nil && c.Now().Sub(info.ModTime()) < indexMaxAge {
		return readIndexFile(c.IndexPath)
	}
	ids, err := c.fetchIndex()
	if err != nil {
		if statErr == nil {
			return readIndexFile(c.IndexPath)
		}
		return nil, err
	}
	if err := writeIndexFile(c.IndexPath, ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (c Client) fetchIndex() ([]string, error) {
	body, err := c.get("/sitemap.xml")
	if err != nil {
		return nil, err
	}
	var sitemapIndex struct {
		Sitemaps []struct {
			Loc string `xml:"loc"`
		} `xml:"sitemap"`
	}
	if err := xml.Unmarshal(body, &sitemapIndex); err != nil {
		return nil, fmt.Errorf("skills.sh: sitemap index: %w", err)
	}
	var ids []string
	for _, sitemap := range sitemapIndex.Sitemaps {
		sitemapPath, ok := skillSitemapPath(sitemap.Loc)
		if !ok {
			continue
		}
		body, err := c.get(sitemapPath)
		if err != nil {
			return nil, err
		}
		urlIDs, err := urlsetIDs(body)
		if err != nil {
			return nil, fmt.Errorf("skills.sh: %s: %w", sitemapPath, err)
		}
		ids = append(ids, urlIDs...)
	}
	slices.Sort(ids)
	return slices.Compact(ids), nil
}

func skillSitemapPath(loc string) (string, bool) {
	parsed, err := url.Parse(loc)
	if err != nil {
		return "", false
	}
	matched, _ := path.Match(skillSitemapGlob, path.Base(parsed.Path))
	return parsed.Path, matched
}

func urlsetIDs(body []byte) ([]string, error) {
	var urlset struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(body, &urlset); err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range urlset.URLs {
		parsed, err := url.Parse(entry.Loc)
		if err != nil {
			continue
		}
		if id := strings.Trim(parsed.Path, "/"); skills.IsRegistryID(id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

var errNotFound = errors.New("not found on skills.sh")

func (c Client) get(requestPath string) ([]byte, error) {
	resp, err := c.HTTP.Get(c.BaseURL + requestPath)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: GET %s: %w", requestPath, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	case http.StatusNotFound:
		return nil, fmt.Errorf("skills.sh: GET %s: %w", requestPath, errNotFound)
	}
	return nil, fmt.Errorf("skills.sh: GET %s: %s", requestPath, resp.Status)
}

type File struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

func (c Client) Files(id string) ([]File, error) {
	if !skills.IsRegistryID(id) {
		return nil, fmt.Errorf("not a skills.sh ID: %s", id)
	}
	body, err := c.get("/api/download/" + escapedSegments(id))
	if err != nil {
		return nil, err
	}
	var download struct {
		Files []File `json:"files"`
	}
	if err := json.Unmarshal(body, &download); err != nil {
		return nil, fmt.Errorf("skills.sh: download %s: %w", id, err)
	}
	return download.Files, nil
}

func escapedSegments(id string) string {
	segments := strings.Split(id, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

func (c Client) Manifest(id string) (string, error) {
	files, err := c.Files(id)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if file.Path == "SKILL.md" {
			return file.Contents, nil
		}
	}
	return "", fmt.Errorf("skills.sh: %s has no SKILL.md", id)
}

func (c Client) Download(id, dest string) error {
	files, err := c.Files(id)
	if err != nil {
		return err
	}
	if err := checkFiles(id, files); err != nil {
		return err
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".download-*")
	if err != nil {
		return err
	}
	err = writeFiles(staging, files)
	if err == nil {
		err = os.RemoveAll(dest)
	}
	if err == nil {
		err = os.Rename(staging, dest)
	}
	if err != nil {
		_ = os.RemoveAll(staging)
	}
	return err
}

func checkFiles(id string, files []File) error {
	hasManifest := false
	for _, file := range files {
		if !filepath.IsLocal(filepath.FromSlash(file.Path)) {
			return fmt.Errorf("skills.sh: %s has an unsafe file path %q", id, file.Path)
		}
		hasManifest = hasManifest || file.Path == "SKILL.md"
	}
	if !hasManifest {
		return fmt.Errorf("skills.sh: %s has no SKILL.md", id)
	}
	return nil
}

func writeFiles(root string, files []File) error {
	if err := os.Chmod(root, 0o755); err != nil {
		return err
	}
	for _, file := range files {
		target := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(file.Contents), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func readIndexFile(indexPath string) ([]string, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids, nil
}

func writeIndexFile(indexPath string, ids []string) error {
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return err
	}
	return atomicfile.Write(indexPath, []byte(strings.Join(ids, "\n")+"\n"))
}
