package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hionay/blgrep/blocklist"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: query parameter is required")
		os.Exit(1)
	}
	query := os.Args[1]

	s := &blocklist.Scanner{
		Query:       query,
		Concurrency: 20,
		Client:      &http.Client{Timeout: 30 * time.Second},
	}

	fmt.Println("Fetching source list from hagezi/dns-blocklists...")
	urls, err := s.FetchSources(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d source URLs. Searching for %q...\n\n", len(urls), s.Query)
	res := s.Scan(ctx, urls)
	if len(res.Matches) == 0 {
		fmt.Println("No matches found.")
	} else {
		fmt.Printf("Found %d match(es):\n\n", len(res.Matches))
		for _, m := range res.Matches {
			fmt.Printf("LIST: %s\n  LINE %d: %s\n\n", m.URL, m.Line, m.Text)
		}
	}
	if len(res.Errors) > 0 {
		fmt.Printf("\n--- %d fetch errors ---\n", len(res.Errors))
		for _, e := range res.Errors {
			fmt.Println(e)
		}
	}
}
