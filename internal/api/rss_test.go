package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"technews-tui/internal/model"
)

func TestRSSClient_Metadata(t *testing.T) {
	client := NewRSSClient([]string{"https://example.com/feed.xml"})
	if got := client.ID(); got != "rss" {
		t.Errorf("ID() = %q, want %q", got, "rss")
	}
	if got := client.Name(); got != "RSS" {
		t.Errorf("Name() = %q, want %q", got, "RSS")
	}
	opts := client.SortOptions()
	if len(opts) != 1 || opts[0] != "default" {
		t.Errorf("SortOptions() = %v, want [default]", opts)
	}
}

func TestRSSClient_FetchComments_ReturnsEmpty(t *testing.T) {
	client := NewRSSClient([]string{"https://example.com/feed.xml"})
	post := model.Post{Source: "rss", SourceID: "abc"}
	comments, err := client.FetchComments(post, 3)
	if err != nil {
		t.Fatalf("FetchComments() error = %v, want nil", err)
	}
	if len(comments) != 0 {
		t.Errorf("FetchComments() returned %d comments, want 0", len(comments))
	}
}

func TestRSSClient_FetchPosts_NoTargets(t *testing.T) {
	client := NewRSSClient([]string{})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 0 {
		t.Errorf("FetchPosts() returned %d posts, want 0", len(posts))
	}
}

func TestRSSClient_FetchPosts_LimitZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Test</title>
<item><title>Item</title><link>https://example.com/1</link><guid>abc</guid></item></channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 0)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 0 {
		t.Errorf("FetchPosts(limit=0) returned %d posts, want 0", len(posts))
	}
}

func TestRSSClient_FetchPosts_SkipsBlankTargets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Test</title>
<item><title>Real Item</title><link>https://ex.com/1</link><guid>x</guid></item>
</channel></rss>`))
	}))
	defer server.Close()

	// When blank targets are skipped and only valid targets remain,
	// the valid target's feed should be fetched successfully
	client := NewRSSClient([]string{"", "  ", server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Errorf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	if posts[0].Title != "Real Item" {
		t.Errorf("Title = %q, want %q", posts[0].Title, "Real Item")
	}
}

func TestRSSClient_FetchPosts_SingleFeed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>LWN.net</title>
<item>
<title>Test Article</title>
<link>https://lwn.net/Articles/123</link>
<guid>first-guid</guid>
<author>dan</author>
<pubDate>Wed, 01 Jan 2025 12:00:00 +0000</pubDate>
<description>&lt;p&gt;This is a &lt;b&gt;test&lt;/b&gt; article.&lt;/p&gt;</description>
</item>
<item>
<title>Another Article</title>
<link>https://lwn.net/Articles/456</link>
<guid>second-guid</guid>
<pubDate>Wed, 01 Jan 2025 13:00:00 +0000</pubDate>
</item>
</channel>
</rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 2 {
		t.Fatalf("FetchPosts() returned %d posts, want 2", len(posts))
	}

	// Posts are sorted by CreatedAt descending, so "Another Article" (13:00) comes first
	p := posts[0]
	if p.Title != "Another Article" {
		t.Errorf("Title = %q, want %q (sorted by date desc, this is newer)", p.Title, "Another Article")
	}
	if p.URL != "https://lwn.net/Articles/456" {
		t.Errorf("URL = %q, want %q", p.URL, "https://lwn.net/Articles/456")
	}
	if p.SourceURL != "https://lwn.net/Articles/456" {
		t.Errorf("SourceURL = %q, want %q", p.SourceURL, "https://lwn.net/Articles/456")
	}
	if p.SourceID != "second-guid" {
		t.Errorf("SourceID = %q, want %q", p.SourceID, "second-guid")
	}
	if p.Source != "rss" {
		t.Errorf("Source = %q, want %q", p.Source, "rss")
	}
	if p.SourceLabel != "LWN.net" {
		t.Errorf("SourceLabel = %q, want %q", p.SourceLabel, "LWN.net")
	}
	if p.Author != "" {
		// Author is empty because <author> in RSS <item> is not parsed by gofeed into item.Author
		// gofeed parses <author> as a Dublin-Core extension, not the author field
		t.Errorf("Author = %q, want empty string (gofeed doesn't parse <author> into Author.Name for RSS)", p.Author)
	}
	if p.Points != 0 {
		t.Errorf("Points = %d, want 0", p.Points)
	}
	if p.CommentCount != 0 {
		t.Errorf("CommentCount = %d, want 0", p.CommentCount)
	}
	if p.Rank != 0 {
		t.Errorf("Rank = %d, want 0 (first post after sort by time desc)", p.Rank)
	}

	// Second post (older)
	p2 := posts[1]
	if p2.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", p2.Title, "Test Article")
	}
	if p2.SourceID != "first-guid" {
		t.Errorf("SourceID = %q, want %q", p2.SourceID, "first-guid")
	}
}

func TestRSSClient_FetchPosts_AtomFeed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>Phoronix</title>
<entry>
<title>Phoronix Test Article</title>
<link href="https://phoronix.com/123"/>
<id>phoronix-guid-123</id>
<updated>2025-01-01T14:00:00Z</updated>
<summary>Phoronix description</summary>
</entry>
</feed>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	p := posts[0]
	if p.Title != "Phoronix Test Article" {
		t.Errorf("Title = %q, want %q", p.Title, "Phoronix Test Article")
	}
	if p.SourceLabel != "Phoronix" {
		t.Errorf("SourceLabel = %q, want %q", p.SourceLabel, "Phoronix")
	}
	if p.SourceID != "phoronix-guid-123" {
		t.Errorf("SourceID = %q, want %q", p.SourceID, "phoronix-guid-123")
	}
}

func TestRSSClient_FetchPosts_EmptyFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Empty</title></channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 0 {
		t.Errorf("FetchPosts() returned %d posts, want 0", len(posts))
	}
}

func TestRSSClient_FetchPosts_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	_, err := client.FetchPosts("default", 30)
	if err == nil {
		t.Fatal("FetchPosts() error = nil, want non-nil")
	}
}

func TestRSSClient_FetchPosts_MalformedXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`this is not xml at all <><><`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	_, err := client.FetchPosts("default", 30)
	if err == nil {
		t.Fatal("FetchPosts() error = nil, want non-nil")
	}
}

func TestRSSClient_FetchPosts_TransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close without writing anything to cause a transport error
	}))
	server.Close()

	client := NewRSSClient([]string{server.URL})
	_, err := client.FetchPosts("default", 30)
	if err == nil {
		t.Fatal("FetchPosts() error = nil, want non-nil (closed server)")
	}
}

func TestRSSClient_FetchPosts_PartialFailure(t *testing.T) {
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Good</title>
<item><title>Good Item</title><link>https://good.com/1</link><guid>good-guid</guid></item>
</channel></rss>`))
	}))
	defer goodServer.Close()

	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()

	client := NewRSSClient([]string{goodServer.URL, badServer.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() returned error %v, want nil (partial failure is OK)", err)
	}
	if len(posts) != 1 {
		t.Errorf("FetchPosts() returned %d posts, want 1 from good feed", len(posts))
	}
	if posts[0].Title != "Good Item" {
		t.Errorf("Title = %q, want %q", posts[0].Title, "Good Item")
	}
}

func TestRSSClient_FetchPosts_AllFeedsFail(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	client := NewRSSClient([]string{server1.URL, server2.URL})
	_, err := client.FetchPosts("default", 30)
	if err == nil {
		t.Fatal("FetchPosts() error = nil, want error when all feeds fail")
	}
}

func TestRSSClient_FetchPosts_MultipleFeedsSuccess(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Feed1</title>
<item><title>Feed1 Old</title><link>https://feed1.com/1</link><guid>feed1-old</guid><pubDate>Wed, 01 Jan 2025 10:00:00 +0000</pubDate></item>
<item><title>Feed1 New</title><link>https://feed1.com/2</link><guid>feed1-new</guid><pubDate>Wed, 01 Jan 2025 14:00:00 +0000</pubDate></item>
</channel></rss>`))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Feed2</title>
<item><title>Feed2 Middle</title><link>https://feed2.com/1</link><guid>feed2-mid</guid><pubDate>Wed, 01 Jan 2025 12:00:00 +0000</pubDate></item>
<item><title>Feed2 Oldest</title><link>https://feed2.com/2</link><guid>feed2-old</guid><pubDate>Wed, 01 Jan 2025 8:00:00 +0000</pubDate></item>
</channel></rss>`))
	}))
	defer server2.Close()

	client := NewRSSClient([]string{server1.URL, server2.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 4 {
		t.Fatalf("FetchPosts() returned %d posts, want 4", len(posts))
	}

	// Verify sorted by CreatedAt descending (most recent first)
	if posts[0].Title != "Feed1 New" {
		t.Errorf("posts[0].Title = %q, want %q (most recent first)", posts[0].Title, "Feed1 New")
	}
	if posts[1].Title != "Feed2 Middle" {
		t.Errorf("posts[1].Title = %q, want %q", posts[1].Title, "Feed2 Middle")
	}
	if posts[2].Title != "Feed1 Old" {
		t.Errorf("posts[2].Title = %q, want %q", posts[2].Title, "Feed1 Old")
	}
	if posts[3].Title != "Feed2 Oldest" {
		t.Errorf("posts[3].Title = %q, want %q (oldest last)", posts[3].Title, "Feed2 Oldest")
	}

	// Verify Ranks are assigned sequentially after final sort
	for i, p := range posts {
		if p.Rank != i {
			t.Errorf("posts[%d].Rank = %d, want %d", i, p.Rank, i)
		}
	}

	// Verify SourceLabels
	if posts[0].SourceLabel != "Feed1" {
		t.Errorf("SourceLabel = %q, want %q", posts[0].SourceLabel, "Feed1")
	}
}

func TestRSSClient_FetchPosts_EnforcesLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		items := []string{}
		for i := 0; i < 10; i++ {
			items = append(items, `<item><title>Item `+strings.Repeat("0", i)+`</title><link>https://ex.com/`+strings.Repeat("0", i)+`</link><guid>guid-`+strings.Repeat("0", i)+`</guid></item>`)
		}
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>X</title>` + strings.Join(items, "") + `</channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL, server.URL}) // two identical feeds
	posts, err := client.FetchPosts("default", 5)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 5 {
		t.Errorf("FetchPosts(limit=5) returned %d posts, want 5", len(posts))
	}
}

func TestRSSClient_FetchPosts_SourceLabel_FallsBackToHostname(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title></title>
<item><title>Item</title><link>https://lwn.net/ Articles/1</link><guid>x</guid></item>
</channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	if posts[0].SourceLabel == "" {
		t.Error("SourceLabel is empty, should fall back to hostname")
	}
}

func TestRSSClient_FetchPosts_SourceID_FallsBackToLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>X</title>
<item><title>Item</title><link>https://example.com/123</link></item>
</channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	if posts[0].SourceID != "https://example.com/123" {
		t.Errorf("SourceID = %q, want fallback to Link %q", posts[0].SourceID, "https://example.com/123")
	}
}

func TestRSSClient_FetchPosts_CreatedAt_FallsBackToUpdated(t *testing.T) {
	// Use Atom format since the fallback to UpdatedParsed is Atom-specific
	// (RSS uses <pubDate> which maps to PublishedParsed, not UpdatedParsed)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>X</title>
<entry>
<title>Item</title><link href="https://ex.com/1"/><id>x</id>
<updated>2025-01-02T12:00:00Z</updated>
</entry>
</feed>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	expected := time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)
	if posts[0].CreatedAt.Unix() != expected.Unix() {
		t.Errorf("CreatedAt = %v, want fallback to Updated %v", posts[0].CreatedAt, expected)
	}
}

func TestRSSClient_FetchPosts_Text_PrefersContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>X</title>
<item><title>Item</title><link>https://ex.com/1</link><guid>x</guid>
<content:encoded xmlns:content="http://purl.org/rss/1.0/modules/content/"><![CDATA[<p>Full content here</p>]]></content:encoded>
<description>Short desc</description>
</item>
</channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	if posts[0].Text != "Full content here" {
		t.Errorf("Text = %q, want %q (should prefer content:encoded)", posts[0].Text, "Full content here")
	}
}

func TestRSSClient_FetchPosts_Text_UsesDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>X</title>
<item><title>Item</title><link>https://ex.com/1</link><guid>x</guid>
<description>Just description</description>
</item>
</channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	if posts[0].Text != "Just description" {
		t.Errorf("Text = %q, want %q", posts[0].Text, "Just description")
	}
}

func TestRSSClient_FetchPosts_Text_StripsHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>X</title>
<item><title>Item</title><link>https://ex.com/1</link><guid>x</guid>
<description>&lt;b&gt;Bold&lt;/b&gt; and &lt;i&gt;italic&lt;/i&gt; and &lt;code&gt;code&lt;/code&gt;</description>
</item>
</channel></rss>`))
	}))
	defer server.Close()

	client := NewRSSClient([]string{server.URL})
	posts, err := client.FetchPosts("default", 30)
	if err != nil {
		t.Fatalf("FetchPosts() error = %v, want nil", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchPosts() returned %d posts, want 1", len(posts))
	}
	expected := "Bold and italic and `code`"
	if posts[0].Text != expected {
		t.Errorf("Text = %q, want %q (HTML should be stripped)", posts[0].Text, expected)
	}
}