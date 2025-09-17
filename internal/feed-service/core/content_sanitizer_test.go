package core

import (
	"testing"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFeedItem_RemovesDangerousTags(t *testing.T) {
	item := &gofeed.Item{
		Content: "<p>Safe</p><script>alert('xss')</script>",
	}

	content, description, err := sanitizeFeedItem(item, "https://example.com/article")
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.NotContains(t, content, "script")
	require.Equal(t, "Safe", description)
}

func TestSanitizeFeedItem_AbsolutizesRelativeURLs(t *testing.T) {
	item := &gofeed.Item{
		Content: `<a href="/post">Read</a><img src="images/pic.png" alt="pic">`,
	}

	content, _, err := sanitizeFeedItem(item, "https://example.com/base")
	require.NoError(t, err)
	require.Contains(t, content, `href="https://example.com/post"`)
	require.Contains(t, content, `src="https://example.com/images/pic.png"`)
}

func TestSanitizeFeedItem_PlainTextWrapped(t *testing.T) {
	item := &gofeed.Item{
		Content: "Plain text content",
	}

	content, _, err := sanitizeFeedItem(item, "https://example.com/base")
	require.NoError(t, err)
	require.Contains(t, content, "<pre>")
	require.Contains(t, content, "Plain text content")
}
