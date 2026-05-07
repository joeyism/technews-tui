package api

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"technews-tui/internal/model"
)

type RSSClient struct {
	http    *http.Client
	parser  *gofeed.Parser
	targets []string
}

func NewRSSClient(targets []string) *RSSClient {
	return &RSSClient{
		http: &http.Client{Timeout: 15 * time.Second},
		parser: gofeed.NewParser(),
		targets: targets,
	}
}

func (c *RSSClient) ID() string   { return "rss" }
func (c *RSSClient) Name() string { return "RSS" }
func (c *RSSClient) SortOptions() []string {
	return []string{"default"}
}

func (c *RSSClient) FetchComments(post model.Post, maxDepth int) ([]model.Comment, error) {
	return []model.Comment{}, nil
}

func (c *RSSClient) FetchPosts(sortOption string, limit int) ([]model.Post, error) {
	if limit <= 0 {
		return []model.Post{}, nil
	}

	// Filter out blank targets
	var nonBlankTargets []string
	for _, t := range c.targets {
		if strings.TrimSpace(t) == "" {
			continue
		}
		nonBlankTargets = append(nonBlankTargets, t)
	}
	if len(nonBlankTargets) == 0 {
		return []model.Post{}, nil
	}

	type feedResult struct {
		posts []model.Post
		err   error
	}

	results := make(chan feedResult, len(nonBlankTargets))
	var wg sync.WaitGroup

	for _, target := range nonBlankTargets {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			posts, err := c.fetchFeed(t)
			results <- feedResult{posts: posts, err: err}
		}(target)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allPosts []model.Post
	var lastErr error
	for res := range results {
		if res.err != nil {
			lastErr = res.err
			continue
		}
		allPosts = append(allPosts, res.posts...)
	}

	if len(allPosts) == 0 && lastErr != nil {
		return nil, fmt.Errorf("all feeds failed: %w", lastErr)
	}

	// Sort by CreatedAt descending; for stability, also sort by original order
	// We use the order they appear in the slice as the tiebreaker (which is
	// target index + item index within feed, set during fetchFeed).
	sort.SliceStable(allPosts, func(i, j int) bool {
		if allPosts[i].CreatedAt.Equal(allPosts[j].CreatedAt) {
			return allPosts[i].Rank < allPosts[j].Rank
		}
		return allPosts[i].CreatedAt.After(allPosts[j].CreatedAt)
	})

	// Trim to limit
	if len(allPosts) > limit {
		allPosts = allPosts[:limit]
	}

	// Assign global rank after final ordering
	for i := range allPosts {
		allPosts[i].Rank = i
	}

	return allPosts, nil
}

func (c *RSSClient) fetchFeed(target string) ([]model.Post, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, nil
	}

	resp, err := c.http.Get(target)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed %s returned status %d", target, resp.StatusCode)
	}

	feed, err := c.parser.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	sourceLabel := c.extractSourceLabel(feed, target)

	var posts []model.Post
	for i, item := range feed.Items {
		post := c.mapItemToPost(item, sourceLabel, target)
		post.Rank = i
		posts = append(posts, post)
	}

	return posts, nil
}

func (c *RSSClient) extractSourceLabel(feed *gofeed.Feed, target string) string {
	if feed.Title != "" {
		return feed.Title
	}
	u, err := url.Parse(target)
	if err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return target
}

func (c *RSSClient) mapItemToPost(item *gofeed.Item, sourceLabel, target string) model.Post {
	sourceID := ""
	if item.GUID != "" {
		sourceID = item.GUID
	} else if item.Link != "" {
		sourceID = item.Link
	}

	articleURL := item.Link
	sourceURL := item.Link
	if sourceURL == "" {
		sourceURL = target
	}

	author := ""
	if item.Author != nil && item.Author.Name != "" {
		author = item.Author.Name
	}

	createdAt := time.Time{}
	if item.PublishedParsed != nil {
		createdAt = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		createdAt = *item.UpdatedParsed
	}

	text := ""
	if item.Content != "" {
		text = StripHTML(item.Content)
	} else if item.Description != "" {
		text = StripHTML(item.Description)
	}

	return model.Post{
		ID:           sourceID,
		Title:        item.Title,
		URL:          articleURL,
		SourceURL:    sourceURL,
		Author:       author,
		Points:       0,
		CommentCount: 0,
		Source:       "rss",
		SourceLabel:  sourceLabel,
		SourceID:     sourceID,
		CreatedAt:    createdAt,
		Text:         text,
	}
}