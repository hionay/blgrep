package blocklist

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const defaultSourcesURL = "https://raw.githubusercontent.com/hagezi/dns-blocklists/refs/heads/main/sources.md"

// Scanner searches blocklist sources for a given query.
type Scanner struct {
	// Query is the case-insensitive search term.
	Query string

	// SourcesURL overrides the default hagezi sources.md URL.
	// If empty, the default is used.
	SourcesURL string

	// Concurrency controls how many lists are fetched in parallel.
	// Defaults to 20 if zero.
	Concurrency int

	// Client is the HTTP client used for all requests.
	// If nil, a default client is used.
	Client *http.Client
}

// Match represents a single hit in a blocklist.
type Match struct {
	URL  string
	Line int
	Text string
}

// Result holds the matches and errors from a Scan.
type Result struct {
	Matches []Match
	Errors  []string
}

// FetchSources fetches sources.md and extracts all https:// URLs from code blocks.
func (s *Scanner) FetchSources(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.sourcesURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching sources: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sources returned HTTP %d", resp.StatusCode)
	}

	var urls []string
	sc := bufio.NewScanner(resp.Body)

	inCodeBlock := false
	for sc.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock && strings.HasPrefix(line, "https://") {
			urls = append(urls, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading sources: %w", err)
	}
	return urls, nil
}

// Scan fetches all given URLs concurrently and searches each for s.Query.
func (s *Scanner) Scan(ctx context.Context, urls []string) Result {
	query := strings.ToLower(s.Query)

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		res Result
	)

	sem := make(chan struct{}, s.concurrency())
	for _, u := range urls {
		wg.Go(func() {
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			matches, err := s.searchURL(ctx, u, query)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				mu.Lock()
				res.Errors = append(res.Errors, err.Error())
				mu.Unlock()
				return
			}
			if len(matches) > 0 {
				mu.Lock()
				res.Matches = append(res.Matches, matches...)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return res
}

func (s *Scanner) searchURL(ctx context.Context, url, query string) ([]Match, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("FETCH ERROR: %s — %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	sc := bufio.NewScanner(resp.Body)
	// Intentionally large buffer for one-line big JSONs in some sources.md entries.
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)

	var matches []Match
	lineNum := 0
	for sc.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		lineNum++
		line := sc.Text()
		if strings.Contains(strings.ToLower(line), query) {
			matches = append(matches, Match{
				URL:  url,
				Line: lineNum,
				Text: line,
			})
		}
	}
	return matches, nil
}

func (s *Scanner) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return http.DefaultClient
}

func (s *Scanner) sourcesURL() string {
	if s.SourcesURL != "" {
		return s.SourcesURL
	}
	return defaultSourcesURL
}

func (s *Scanner) concurrency() int {
	if s.Concurrency > 0 {
		return s.Concurrency
	}
	return 20
}
