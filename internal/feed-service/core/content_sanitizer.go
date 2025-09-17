package core

import (
	"bytes"
	htmlstd "html"
	"net/url"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	htmlnode "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var htmlTagPattern = regexp.MustCompile(`(?i)<[a-z][\s\S]*>`)

// sanitizeFeedItem prepares article content and description for storage and rendering.
func sanitizeFeedItem(item *gofeed.Item, baseURL string) (string, string, error) {
	rawContent := firstNonEmpty(item.Content, item.Description)

	var sanitizedContent string
	var err error
	if strings.TrimSpace(rawContent) != "" {
		sanitizedContent, err = sanitizeHTML(rawContent, baseURL)
		if err != nil {
			return "", "", err
		}
	}

	description := sanitizePlainText(item.Description)
	if description == "" {
		description = sanitizePlainText(sanitizedContent)
	}

	return sanitizedContent, description, nil
}

func sanitizeHTML(raw, baseURL string) (string, error) {
	markup := ensureHTML(raw)
	absoluteMarkup, err := absolutizeMarkup(markup, baseURL)
	if err != nil {
		return "", err
	}

	policy := bluemonday.UGCPolicy()
	allowRichContent(policy)

	return policy.Sanitize(absoluteMarkup), nil
}

func ensureHTML(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if htmlTagPattern.MatchString(trimmed) {
		return raw
	}

	return "<pre>" + htmlstd.EscapeString(trimmed) + "</pre>"
}

func absolutizeMarkup(input, base string) (string, error) {
	if strings.TrimSpace(input) == "" || strings.TrimSpace(base) == "" {
		return input, nil
	}

	parsedBase, err := url.Parse(base)
	if err != nil || !parsedBase.IsAbs() {
		return input, nil
	}

	container := &htmlnode.Node{Type: htmlnode.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := htmlnode.ParseFragment(strings.NewReader(input), container)
	if err != nil {
		return input, err
	}

	for _, n := range nodes {
		rewriteRelativeURLs(n, parsedBase)
		container.AppendChild(n)
	}

	var buf bytes.Buffer
	for child := container.FirstChild; child != nil; child = child.NextSibling {
		if err := htmlnode.Render(&buf, child); err != nil {
			return input, err
		}
	}

	return buf.String(), nil
}

func rewriteRelativeURLs(node *htmlnode.Node, base *url.URL) {
	if node.Type == htmlnode.ElementNode {
		for i, attr := range node.Attr {
			switch attr.Key {
			case "href", "src":
				resolved := absolutize(attr.Val, base)
				if resolved != "" {
					node.Attr[i].Val = resolved
				}
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		rewriteRelativeURLs(child, base)
	}
}

func absolutize(value string, base *url.URL) string {
	s := strings.TrimSpace(value)
	if s == "" {
		return ""
	}

	parsed, err := url.Parse(s)
	if err != nil || parsed.IsAbs() {
		return s
	}

	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return s
	}

	return base.ResolveReference(parsed).String()
}

func allowRichContent(policy *bluemonday.Policy) {
	policy.AllowElements("pre", "code", "img", "figure", "figcaption")
	policy.AllowAttrs("src", "alt", "title", "width", "height", "loading").OnElements("img")
	policy.AllowURLSchemes("http", "https")
	policy.AllowAttrs("class").OnElements("code", "pre")
}

func sanitizePlainText(input string) string {
	cleaner := bluemonday.StrictPolicy()
	return strings.TrimSpace(cleaner.Sanitize(input))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
