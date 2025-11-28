package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

func newOPMLTestContext(method, path string, body io.Reader) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, body)
	ctx.Request = req
	return ctx, w
}

func attachOPMLUserContext(c *gin.Context, userID uint) {
	baseCtx := logger.WithRequestID(context.Background(), "test-request")
	baseCtx = logger.WithUserID(baseCtx, userID)
	c.Request = c.Request.WithContext(baseCtx)
	c.Set("userID", userID)
}

func TestOPMLHandler_ExportOPML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         uint
		feeds          []*models.Feed
		expectStatus   int
		expectContains []string
	}{
		{
			name:   "export empty feeds",
			userID: 1,
			feeds:  []*models.Feed{},
			expectStatus: http.StatusOK,
			expectContains: []string{
				`<?xml version="1.0" encoding="UTF-8"?>`,
				`<opml version="2.0">`,
				`Phoenix RSS Subscriptions`,
			},
		},
		{
			name:   "export with feeds",
			userID: 2,
			feeds: []*models.Feed{
				{ID: 1, Title: "Test Feed 1", URL: "https://example1.com/feed.xml"},
				{ID: 2, Title: "Test Feed 2", URL: "https://example2.com/feed.xml"},
			},
			expectStatus: http.StatusOK,
			expectContains: []string{
				`text="Test Feed 1"`,
				`xmlUrl="https://example1.com/feed.xml"`,
				`text="Test Feed 2"`,
				`xmlUrl="https://example2.com/feed.xml"`,
			},
		},
		{
			name:   "export with Chinese characters",
			userID: 3,
			feeds: []*models.Feed{
				{ID: 1, Title: "阮一峰的网络日志", URL: "https://ruanyifeng.com/feed"},
			},
			expectStatus: http.StatusOK,
			expectContains: []string{
				`阮一峰的网络日志`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubFeedService{
				listUserFeedsFn: func(ctx context.Context, userID uint) ([]*models.Feed, error) {
					require.Equal(t, tt.userID, userID)
					return tt.feeds, nil
				},
			}

			handler := NewOPMLHandler(stub, nil)
			ctx, w := newOPMLTestContext(http.MethodGet, "/api/v1/feeds/export", nil)
			attachOPMLUserContext(ctx, tt.userID)

			handler.ExportOPML(ctx)

			require.Equal(t, tt.expectStatus, w.Code)
			require.Equal(t, "application/xml", w.Header().Get("Content-Type"))
			require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
			require.Contains(t, w.Header().Get("Content-Disposition"), ".opml")

			body := w.Body.String()
			for _, expected := range tt.expectContains {
				require.Contains(t, body, expected)
			}
		})
	}
}

func TestOPMLHandler_PreviewOPML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Feed 1" title="Feed 1" type="rss" xmlUrl="https://example1.com/feed.xml"/>
    <outline text="Feed 2" title="Feed 2" type="rss" xmlUrl="https://example2.com/feed.xml"/>
  </body>
</opml>`

	tests := []struct {
		name           string
		userID         uint
		opmlContent    string
		existingFeeds  []*models.Feed
		expectStatus   int
		expectToImport int
		expectDups     int
	}{
		{
			name:           "preview with no existing feeds",
			userID:         1,
			opmlContent:    opmlContent,
			existingFeeds:  []*models.Feed{},
			expectStatus:   http.StatusOK,
			expectToImport: 2,
			expectDups:     0,
		},
		{
			name:        "preview with one duplicate",
			userID:      2,
			opmlContent: opmlContent,
			existingFeeds: []*models.Feed{
				{ID: 1, Title: "Feed 1", URL: "https://example1.com/feed.xml"},
			},
			expectStatus:   http.StatusOK,
			expectToImport: 1,
			expectDups:     1,
		},
		{
			name:        "preview with all duplicates",
			userID:      3,
			opmlContent: opmlContent,
			existingFeeds: []*models.Feed{
				{ID: 1, Title: "Feed 1", URL: "https://example1.com/feed.xml"},
				{ID: 2, Title: "Feed 2", URL: "https://example2.com/feed.xml"},
			},
			expectStatus:   http.StatusOK,
			expectToImport: 0,
			expectDups:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubFeedService{
				listUserFeedsFn: func(ctx context.Context, userID uint) ([]*models.Feed, error) {
					require.Equal(t, tt.userID, userID)
					return tt.existingFeeds, nil
				},
			}

			handler := NewOPMLHandler(stub, nil)

			// Create multipart form with file
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", "test.opml")
			require.NoError(t, err)
			_, err = part.Write([]byte(tt.opmlContent))
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			ctx, w := newOPMLTestContext(http.MethodPost, "/api/v1/feeds/import/preview", body)
			ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
			attachOPMLUserContext(ctx, tt.userID)

			handler.PreviewOPML(ctx)

			require.Equal(t, tt.expectStatus, w.Code)

			var result PreviewImportRequest
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
			require.Equal(t, tt.expectToImport, len(result.ToImport))
			require.Equal(t, tt.expectDups, len(result.Duplicates))
		})
	}
}

func TestOPMLHandler_PreviewOPML_InvalidFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubFeedService{}
	handler := NewOPMLHandler(stub, nil)

	// Create multipart form with invalid OPML
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.opml")
	require.NoError(t, err)
	_, err = part.Write([]byte("not valid xml"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	ctx, w := newOPMLTestContext(http.MethodPost, "/api/v1/feeds/import/preview", body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	attachOPMLUserContext(ctx, 1)

	handler.PreviewOPML(ctx)

	// Should have error in context (error middleware would convert this to HTTP error)
	require.True(t, len(ctx.Errors) > 0, "Expected error in context for invalid OPML")
	_ = w // suppress unused warning
}

func TestOPMLHandler_PreviewOPML_NoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubFeedService{}
	handler := NewOPMLHandler(stub, nil)

	ctx, w := newOPMLTestContext(http.MethodPost, "/api/v1/feeds/import/preview", nil)
	attachOPMLUserContext(ctx, 1)

	handler.PreviewOPML(ctx)

	// Should have error in context for missing file
	require.True(t, len(ctx.Errors) > 0, "Expected error in context for missing file")
	_ = w // suppress unused warning
}

func TestOPMLHandler_ImportOPML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         uint
		feeds          []core.OPMLFeedItem
		subscribeErr   error
		expectStatus   int
		expectImported int
		expectFailed   int
	}{
		{
			name:   "import single feed success",
			userID: 1,
			feeds: []core.OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example1.com/feed.xml"},
			},
			subscribeErr:   nil,
			expectStatus:   http.StatusOK,
			expectImported: 1,
			expectFailed:   0,
		},
		{
			name:   "import multiple feeds success",
			userID: 2,
			feeds: []core.OPMLFeedItem{
				{Title: "Feed 1", URL: "https://example1.com/feed.xml"},
				{Title: "Feed 2", URL: "https://example2.com/feed.xml"},
			},
			subscribeErr:   nil,
			expectStatus:   http.StatusOK,
			expectImported: 2,
			expectFailed:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubFeedService{
				subscribeToFeedFn: func(ctx context.Context, userID uint, url string) (*models.Feed, error) {
					require.Equal(t, tt.userID, userID)
					if tt.subscribeErr != nil {
						return nil, tt.subscribeErr
					}
					return &models.Feed{ID: 1, Title: "Imported", URL: url}, nil
				},
			}

			handler := NewOPMLHandler(stub, nil)

			reqBody, err := json.Marshal(ImportOPMLRequest{Feeds: tt.feeds})
			require.NoError(t, err)

			ctx, w := newOPMLTestContext(http.MethodPost, "/api/v1/feeds/import", bytes.NewReader(reqBody))
			ctx.Request.Header.Set("Content-Type", "application/json")
			attachOPMLUserContext(ctx, tt.userID)

			handler.ImportOPML(ctx)

			require.Equal(t, tt.expectStatus, w.Code)

			var result core.OPMLImportResult
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
			require.Equal(t, tt.expectImported, result.Imported)
			require.Equal(t, tt.expectFailed, result.Failed)
		})
	}
}

func TestOPMLHandler_ImportOPML_EmptyFeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubFeedService{}
	handler := NewOPMLHandler(stub, nil)

	reqBody, err := json.Marshal(ImportOPMLRequest{Feeds: []core.OPMLFeedItem{}})
	require.NoError(t, err)

	ctx, w := newOPMLTestContext(http.MethodPost, "/api/v1/feeds/import", bytes.NewReader(reqBody))
	ctx.Request.Header.Set("Content-Type", "application/json")
	attachOPMLUserContext(ctx, 1)

	handler.ImportOPML(ctx)

	// Should have error in context for empty feeds
	require.True(t, len(ctx.Errors) > 0, "Expected error in context for empty feeds")
	_ = w // suppress unused warning
}

func TestOPMLHandler_ImportOPML_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubFeedService{}
	handler := NewOPMLHandler(stub, nil)

	reqBody := `{"feeds": [{"title": "Test", "url": "https://example.com/feed"}]}`
	ctx, w := newOPMLTestContext(http.MethodPost, "/api/v1/feeds/import", strings.NewReader(reqBody))
	ctx.Request.Header.Set("Content-Type", "application/json")
	// Not attaching user context

	handler.ImportOPML(ctx)

	// Should have error in context for unauthenticated request
	require.True(t, len(ctx.Errors) > 0, "Expected error in context for unauthenticated request")
	_ = w // suppress unused warning
}

func TestOPMLHandler_PreviewOPML_RealFeedlyFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Partial content from real Feedly export
	feedlyOPML := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
    <head>
        <title>peixian subscriptions in feedly Cloud</title>
    </head>
    <body>
        <outline text="Every Day" title="Every Day">
            <outline type="rss" text="High Scalability" title="High Scalability" xmlUrl="http://feeds.feedburner.com/HighScalability" htmlUrl="http://highscalability.com/blog/"/>
            <outline type="rss" text="阮一峰的网络日志" title="阮一峰的网络日志" xmlUrl="http://feeds.feedburner.com/ruanyifeng" htmlUrl="http://www.ruanyifeng.com/blog/"/>
        </outline>
        <outline text="Blog" title="Blog">
            <outline type="rss" text="🍺 IceBeer" title="🍺 IceBeer" xmlUrl="https://www.icebeer.top/feed/" htmlUrl="https://www.icebeer.top/"/>
            <outline type="rss" xmlUrl="https://makeoptim.com/feed.xml" htmlUrl="https://makeoptim.com/"/>
        </outline>
    </body>
</opml>`

	stub := &stubFeedService{
		listUserFeedsFn: func(ctx context.Context, userID uint) ([]*models.Feed, error) {
			return []*models.Feed{}, nil
		},
	}

	handler := NewOPMLHandler(stub, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "feedly.opml")
	require.NoError(t, err)
	_, err = part.Write([]byte(feedlyOPML))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	ctx, w := newOPMLTestContext(http.MethodPost, "/api/v1/feeds/import/preview", body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	attachOPMLUserContext(ctx, 1)

	handler.PreviewOPML(ctx)

	require.Equal(t, http.StatusOK, w.Code)

	var result PreviewImportRequest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))

	// Should have 4 feeds from the nested structure
	require.Equal(t, 4, len(result.ToImport))

	// Check that Chinese and emoji titles are preserved
	foundChinese := false
	foundEmoji := false
	for _, feed := range result.ToImport {
		if feed.Title == "阮一峰的网络日志" {
			foundChinese = true
		}
		if feed.Title == "🍺 IceBeer" {
			foundEmoji = true
		}
	}
	require.True(t, foundChinese, "Chinese title should be preserved")
	require.True(t, foundEmoji, "Emoji title should be preserved")
}

