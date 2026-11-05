package registry

import (
	"encoding/xml"
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

func (c Client) get(requestPath string) ([]byte, error) {
	resp, err := c.HTTP.Get(c.BaseURL + requestPath)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: GET %s: %w", requestPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skills.sh: GET %s: %s", requestPath, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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
