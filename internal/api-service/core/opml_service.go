package core

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

// OPML represents the root element of an OPML document.
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    OPMLHead `xml:"head"`
	Body    OPMLBody `xml:"body"`
}

// OPMLHead contains metadata about the OPML document.
type OPMLHead struct {
	Title       string `xml:"title,omitempty"`
	DateCreated string `xml:"dateCreated,omitempty"`
	OwnerName   string `xml:"ownerName,omitempty"`
}

// OPMLBody contains the outline elements.
type OPMLBody struct {
	Outlines []OPMLOutline `xml:"outline"`
}

// OPMLOutline represents a single RSS feed or folder in the OPML.
type OPMLOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []OPMLOutline `xml:"outline,omitempty"` // Nested outlines for folders
}

// OPMLFeedItem represents a parsed feed from OPML for import preview.
type OPMLFeedItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// OPMLParseResult contains the result of parsing an OPML file.
type OPMLParseResult struct {
	Feeds []OPMLFeedItem `json:"feeds"`
	Total int            `json:"total"`
}

// OPMLImportResult contains the result of importing feeds from OPML.
type OPMLImportResult struct {
	Imported   int      `json:"imported"`
	Skipped    int      `json:"skipped"`
	Failed     int      `json:"failed"`
	SkippedIDs []string `json:"skipped_urls,omitempty"`
	FailedIDs  []string `json:"failed_urls,omitempty"`
}

// OPMLService handles OPML parsing and generation.
type OPMLService struct{}

// NewOPMLService creates a new OPML service instance.
func NewOPMLService() *OPMLService {
	return &OPMLService{}
}

// GenerateOPML creates an OPML document from a list of feeds.
// Uses custom_title if set, otherwise falls back to the original feed title.
func (s *OPMLService) GenerateOPML(feeds []*models.UserFeed, username string) ([]byte, error) {
	opml := OPML{
		Version: "2.0",
		Head: OPMLHead{
			Title:       "Phoenix RSS Subscriptions",
			DateCreated: time.Now().Format(time.RFC1123),
			OwnerName:   username,
		},
		Body: OPMLBody{
			Outlines: make([]OPMLOutline, 0, len(feeds)),
		},
	}

	for _, feed := range feeds {
		// Use custom title if set, otherwise use original title
		title := feed.Title
		if feed.CustomTitle != nil && *feed.CustomTitle != "" {
			title = *feed.CustomTitle
		}
		outline := OPMLOutline{
			Text:   title,
			Title:  title,
			Type:   "rss",
			XMLURL: feed.URL,
		}
		opml.Body.Outlines = append(opml.Body.Outlines, outline)
	}

	// Generate XML with proper formatting
	output, err := xml.MarshalIndent(opml, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OPML: %w", err)
	}

	// Add XML declaration
	xmlDeclaration := []byte(xml.Header)
	result := append(xmlDeclaration, output...)

	return result, nil
}

// ParseOPML parses an OPML document and extracts feed information.
func (s *OPMLService) ParseOPML(data []byte) (*OPMLParseResult, error) {
	var opml OPML
	if err := xml.Unmarshal(data, &opml); err != nil {
		return nil, fmt.Errorf("failed to parse OPML: %w", err)
	}

	feeds := make([]OPMLFeedItem, 0)
	s.extractFeeds(opml.Body.Outlines, &feeds)

	return &OPMLParseResult{
		Feeds: feeds,
		Total: len(feeds),
	}, nil
}

// extractFeeds recursively extracts feed items from OPML outlines.
// This handles both flat and nested (categorized) OPML structures.
func (s *OPMLService) extractFeeds(outlines []OPMLOutline, feeds *[]OPMLFeedItem) {
	for _, outline := range outlines {
		// Check if this is a feed (has xmlUrl) or a folder
		if outline.XMLURL != "" {
			title := outline.Title
			if title == "" {
				title = outline.Text
			}
			// Skip empty URLs
			url := strings.TrimSpace(outline.XMLURL)
			if url == "" {
				continue
			}
			*feeds = append(*feeds, OPMLFeedItem{
				Title: title,
				URL:   url,
			})
		}

		// Recursively process nested outlines (folders/categories)
		if len(outline.Outlines) > 0 {
			s.extractFeeds(outline.Outlines, feeds)
		}
	}
}

// FilterDuplicates removes feeds that already exist in the user's subscriptions.
func (s *OPMLService) FilterDuplicates(parsedFeeds []OPMLFeedItem, existingFeeds []*models.UserFeed) (toImport []OPMLFeedItem, duplicates []OPMLFeedItem) {
	existingURLs := make(map[string]bool)
	for _, feed := range existingFeeds {
		// Normalize URL for comparison
		existingURLs[normalizeURL(feed.URL)] = true
	}

	toImport = make([]OPMLFeedItem, 0)
	duplicates = make([]OPMLFeedItem, 0)

	for _, feed := range parsedFeeds {
		normalizedURL := normalizeURL(feed.URL)
		if existingURLs[normalizedURL] {
			duplicates = append(duplicates, feed)
		} else {
			toImport = append(toImport, feed)
			// Mark as existing to handle duplicates within the import file
			existingURLs[normalizedURL] = true
		}
	}

	return toImport, duplicates
}

// normalizeURL normalizes a URL for comparison purposes.
func normalizeURL(url string) string {
	url = strings.TrimSpace(url)
	url = strings.ToLower(url)
	url = strings.TrimSuffix(url, "/")
	return url
}

