package core

import (
	"strings"
	"testing"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

func TestOPMLService_GenerateOPML(t *testing.T) {
	service := NewOPMLService()

	tests := []struct {
		name     string
		feeds    []*models.Feed
		username string
		want     []string // strings that should be in the output
	}{
		{
			name:     "empty feeds",
			feeds:    []*models.Feed{},
			username: "testuser",
			want: []string{
				`<?xml version="1.0" encoding="UTF-8"?>`,
				`<opml version="2.0">`,
				`<title>Phoenix RSS Subscriptions</title>`,
				`<ownerName>testuser</ownerName>`,
				`</opml>`,
			},
		},
		{
			name: "single feed",
			feeds: []*models.Feed{
				{
					ID:    1,
					Title: "Test Feed",
					URL:   "https://example.com/feed.xml",
				},
			},
			username: "testuser",
			want: []string{
				`text="Test Feed"`,
				`title="Test Feed"`,
				`type="rss"`,
				`xmlUrl="https://example.com/feed.xml"`,
			},
		},
		{
			name: "multiple feeds",
			feeds: []*models.Feed{
				{ID: 1, Title: "Feed 1", URL: "https://example1.com/feed.xml"},
				{ID: 2, Title: "Feed 2", URL: "https://example2.com/feed.xml"},
			},
			username: "testuser",
			want: []string{
				`text="Feed 1"`,
				`text="Feed 2"`,
				`xmlUrl="https://example1.com/feed.xml"`,
				`xmlUrl="https://example2.com/feed.xml"`,
			},
		},
		{
			name: "feed with special characters",
			feeds: []*models.Feed{
				{
					ID:    1,
					Title: "Feed with <special> & \"characters\"",
					URL:   "https://example.com/feed.xml?foo=bar&baz=qux",
				},
			},
			username: "testuser",
			want: []string{
				`&lt;special&gt;`, // < and > should be escaped
				`&amp;`,           // & should be escaped
				`&#34;`,           // " is escaped as numeric entity by Go's XML encoder
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.GenerateOPML(tt.feeds, tt.username)
			if err != nil {
				t.Fatalf("GenerateOPML() error = %v", err)
			}

			output := string(got)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("GenerateOPML() output missing expected string %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestOPMLService_ParseOPML(t *testing.T) {
	service := NewOPMLService()

	tests := []struct {
		name      string
		opmlData  string
		wantCount int
		wantFeeds []OPMLFeedItem
		wantErr   bool
	}{
		{
			name: "valid OPML with flat structure",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline text="Feed 1" title="Feed 1" type="rss" xmlUrl="https://example1.com/feed.xml"/>
    <outline text="Feed 2" title="Feed 2" type="rss" xmlUrl="https://example2.com/feed.xml"/>
  </body>
</opml>`,
			wantCount: 2,
			wantFeeds: []OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example1.com/feed.xml"},
				{Title: "Feed 2", URL: "https://example2.com/feed.xml"},
			},
			wantErr: false,
		},
		{
			name: "valid OPML with nested structure (categories)",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline text="Tech">
      <outline text="TechCrunch" title="TechCrunch" type="rss" xmlUrl="https://techcrunch.com/feed/"/>
      <outline text="Ars Technica" title="Ars Technica" type="rss" xmlUrl="https://arstechnica.com/feed/"/>
    </outline>
    <outline text="News">
      <outline text="BBC" title="BBC" type="rss" xmlUrl="https://bbc.com/feed/"/>
    </outline>
  </body>
</opml>`,
			wantCount: 3,
			wantFeeds: []OPMLFeedItem{
				{Title: "TechCrunch", URL: "https://techcrunch.com/feed/"},
				{Title: "Ars Technica", URL: "https://arstechnica.com/feed/"},
				{Title: "BBC", URL: "https://bbc.com/feed/"},
			},
			wantErr: false,
		},
		{
			name: "OPML with missing title uses text",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline text="Feed Without Title" type="rss" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>`,
			wantCount: 1,
			wantFeeds: []OPMLFeedItem{
				{Title: "Feed Without Title", URL: "https://example.com/feed.xml"},
			},
			wantErr: false,
		},
		{
			name: "empty OPML",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body></body>
</opml>`,
			wantCount: 0,
			wantFeeds: []OPMLFeedItem{},
			wantErr:   false,
		},
		{
			name:      "invalid XML",
			opmlData:  `not valid xml`,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name: "OPML with empty URL is skipped",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline text="Valid Feed" type="rss" xmlUrl="https://example.com/feed.xml"/>
    <outline text="Invalid Feed" type="rss" xmlUrl=""/>
    <outline text="Another Invalid" type="rss" xmlUrl="   "/>
  </body>
</opml>`,
			wantCount: 1,
			wantFeeds: []OPMLFeedItem{
				{Title: "Valid Feed", URL: "https://example.com/feed.xml"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ParseOPML([]byte(tt.opmlData))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseOPML() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if result.Total != tt.wantCount {
				t.Errorf("ParseOPML() total = %d, want %d", result.Total, tt.wantCount)
			}

			if len(result.Feeds) != len(tt.wantFeeds) {
				t.Errorf("ParseOPML() feeds count = %d, want %d", len(result.Feeds), len(tt.wantFeeds))
				return
			}

			for i, want := range tt.wantFeeds {
				got := result.Feeds[i]
				if got.Title != want.Title {
					t.Errorf("ParseOPML() feed[%d].Title = %q, want %q", i, got.Title, want.Title)
				}
				if got.URL != want.URL {
					t.Errorf("ParseOPML() feed[%d].URL = %q, want %q", i, got.URL, want.URL)
				}
			}
		})
	}
}

func TestOPMLService_FilterDuplicates(t *testing.T) {
	service := NewOPMLService()

	tests := []struct {
		name           string
		parsedFeeds    []OPMLFeedItem
		existingFeeds  []*models.Feed
		wantToImport   int
		wantDuplicates int
	}{
		{
			name: "no duplicates",
			parsedFeeds: []OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example1.com/feed.xml"},
				{Title: "Feed 2", URL: "https://example2.com/feed.xml"},
			},
			existingFeeds:  []*models.Feed{},
			wantToImport:   2,
			wantDuplicates: 0,
		},
		{
			name: "all duplicates",
			parsedFeeds: []OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example1.com/feed.xml"},
			},
			existingFeeds: []*models.Feed{
				{ID: 1, Title: "Feed 1", URL: "https://example1.com/feed.xml"},
			},
			wantToImport:   0,
			wantDuplicates: 1,
		},
		{
			name: "mixed duplicates",
			parsedFeeds: []OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example1.com/feed.xml"},
				{Title: "Feed 2", URL: "https://example2.com/feed.xml"},
				{Title: "Feed 3", URL: "https://example3.com/feed.xml"},
			},
			existingFeeds: []*models.Feed{
				{ID: 1, Title: "Feed 1", URL: "https://example1.com/feed.xml"},
			},
			wantToImport:   2,
			wantDuplicates: 1,
		},
		{
			name: "case insensitive URL matching",
			parsedFeeds: []OPMLFeedItem{
				{Title: "Feed 1", URL: "HTTPS://EXAMPLE.COM/FEED.XML"},
			},
			existingFeeds: []*models.Feed{
				{ID: 1, Title: "Feed 1", URL: "https://example.com/feed.xml"},
			},
			wantToImport:   0,
			wantDuplicates: 1,
		},
		{
			name: "trailing slash normalization",
			parsedFeeds: []OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example.com/feed/"},
			},
			existingFeeds: []*models.Feed{
				{ID: 1, Title: "Feed 1", URL: "https://example.com/feed"},
			},
			wantToImport:   0,
			wantDuplicates: 1,
		},
		{
			name: "duplicates within import file",
			parsedFeeds: []OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example1.com/feed.xml"},
				{Title: "Feed 1 Duplicate", URL: "https://example1.com/feed.xml"},
			},
			existingFeeds:  []*models.Feed{},
			wantToImport:   1,
			wantDuplicates: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toImport, duplicates := service.FilterDuplicates(tt.parsedFeeds, tt.existingFeeds)

			if len(toImport) != tt.wantToImport {
				t.Errorf("FilterDuplicates() toImport count = %d, want %d", len(toImport), tt.wantToImport)
			}
			if len(duplicates) != tt.wantDuplicates {
				t.Errorf("FilterDuplicates() duplicates count = %d, want %d", len(duplicates), tt.wantDuplicates)
			}
		})
	}
}

func TestOPMLService_ParseRealFeedlyExport(t *testing.T) {
	service := NewOPMLService()

	// Real Feedly OPML export (partial)
	feedlyOPML := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
    <head>
        <title>peixian subscriptions in feedly Cloud</title>
    </head>
    <body>
        <outline text="Every Day" title="Every Day">
            <outline type="rss" text="High Scalability" title="High Scalability" xmlUrl="http://feeds.feedburner.com/HighScalability" htmlUrl="http://highscalability.com/blog/"/>
            <outline type="rss" text="阮一峰的网络日志" title="阮一峰的网络日志" xmlUrl="http://feeds.feedburner.com/ruanyifeng" htmlUrl="http://www.ruanyifeng.com/blog/"/>
            <outline type="rss" text="Stratechery by Ben Thompson" title="Stratechery by Ben Thompson" xmlUrl="https://stratechery.com/feed/" htmlUrl="https://stratechery.com"/>
        </outline>
        <outline text="Blog" title="Blog">
            <outline type="rss" text="🍺 IceBeer" title="🍺 IceBeer" xmlUrl="https://www.icebeer.top/feed/" htmlUrl="https://www.icebeer.top/"/>
            <outline type="rss" text="Stratechery by Ben Thompson" title="Stratechery by Ben Thompson" xmlUrl="https://stratechery.com/feed/" htmlUrl="https://stratechery.com"/>
            <outline type="rss" xmlUrl="https://makeoptim.com/feed.xml" htmlUrl="https://makeoptim.com/"/>
        </outline>
    </body>
</opml>`

	result, err := service.ParseOPML([]byte(feedlyOPML))
	if err != nil {
		t.Fatalf("ParseOPML() error = %v", err)
	}

	// Should have 6 feeds (including duplicates and feed without title)
	if result.Total != 6 {
		t.Errorf("ParseOPML() total = %d, want 6", result.Total)
	}

	// Check specific feeds
	foundFeeds := make(map[string]bool)
	for _, feed := range result.Feeds {
		foundFeeds[feed.URL] = true
		// Check that feed without title uses empty string (not panic)
		if feed.URL == "https://makeoptim.com/feed.xml" && feed.Title != "" {
			t.Errorf("Feed without title should have empty title, got %q", feed.Title)
		}
		// Check Chinese title is preserved
		if feed.URL == "http://feeds.feedburner.com/ruanyifeng" && feed.Title != "阮一峰的网络日志" {
			t.Errorf("Chinese title not preserved, got %q", feed.Title)
		}
		// Check emoji title is preserved
		if feed.URL == "https://www.icebeer.top/feed/" && feed.Title != "🍺 IceBeer" {
			t.Errorf("Emoji title not preserved, got %q", feed.Title)
		}
	}

	// Verify all expected URLs are present
	expectedURLs := []string{
		"http://feeds.feedburner.com/HighScalability",
		"http://feeds.feedburner.com/ruanyifeng",
		"https://stratechery.com/feed/",
		"https://www.icebeer.top/feed/",
		"https://makeoptim.com/feed.xml",
	}
	for _, url := range expectedURLs {
		if !foundFeeds[url] {
			t.Errorf("Expected URL %q not found in parsed feeds", url)
		}
	}
}

func TestOPMLService_RoundTrip(t *testing.T) {
	service := NewOPMLService()

	// Create test feeds
	originalFeeds := []*models.Feed{
		{
			ID:        1,
			Title:     "Test Feed 1",
			URL:       "https://example1.com/feed.xml",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			Title:     "Test Feed 2",
			URL:       "https://example2.com/feed.xml",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Generate OPML
	opmlData, err := service.GenerateOPML(originalFeeds, "testuser")
	if err != nil {
		t.Fatalf("GenerateOPML() error = %v", err)
	}

	// Parse OPML back
	result, err := service.ParseOPML(opmlData)
	if err != nil {
		t.Fatalf("ParseOPML() error = %v", err)
	}

	// Verify round-trip
	if result.Total != len(originalFeeds) {
		t.Errorf("Round-trip: got %d feeds, want %d", result.Total, len(originalFeeds))
	}

	for i, original := range originalFeeds {
		parsed := result.Feeds[i]
		if parsed.Title != original.Title {
			t.Errorf("Round-trip feed[%d].Title = %q, want %q", i, parsed.Title, original.Title)
		}
		if parsed.URL != original.URL {
			t.Errorf("Round-trip feed[%d].URL = %q, want %q", i, parsed.URL, original.URL)
		}
	}
}

