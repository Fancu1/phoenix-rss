package handler

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticFrontendHandler serve the embedded SvelteKit frontend.
type StaticFrontendHandler struct {
	indexHTML  []byte
	fileServer http.Handler
}

// NewStaticFrontendHandler create a new handler for the embedded frontend.
func NewStaticFrontendHandler(staticFS fs.FS) (*StaticFrontendHandler, error) {
	// Read index.html into memory once at startup.
	indexHTML, err := fs.ReadFile(staticFS, "dist/index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to read index.html from embedded fs: %w", err)
	}

	// Create a sub-filesystem that strips the "dist" prefix, so that "/" corresponds to "dist/".
	subFS, err := fs.Sub(staticFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("failed to create sub-filesystem for frontend assets: %w", err)
	}

	return &StaticFrontendHandler{
		indexHTML:  indexHTML,
		fileServer: http.FileServer(http.FS(subFS)),
	}, nil
}

// RegisterRoutes set up the routes for serving the static frontend and the SPA fallback.
func (h *StaticFrontendHandler) RegisterRoutes(engine *gin.Engine) {
	// Use a middleware to set optimal cache headers for different assets.
	engine.Use(h.cacheControlMiddleware())

	// These routes serve the fingerprinted, immutable assets.
	// They will receive a long cache header from the middleware.
	engine.GET("/_app/*filepath", gin.WrapH(h.fileServer))
	engine.GET("/assets/*filepath", gin.WrapH(h.fileServer))
	engine.GET("/favicon.ico", gin.WrapH(h.fileServer))

	// NoRoute handles the SPA fallback. Any request not matching a static file
	// or an API route will be served the pre-loaded index.html.
	engine.NoRoute(func(c *gin.Context) {
		// Only act as a fallback for non-API routes.
		if !strings.HasPrefix(c.Request.RequestURI, "/api") {
			c.Data(http.StatusOK, "text/html; charset=utf-8", h.indexHTML)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		}
	})
}

// cacheControlMiddleware apply appropriate Cache-Control headers to frontend assets.
func (h *StaticFrontendHandler) cacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the request is for an immutable asset.
		if strings.HasPrefix(c.Request.URL.Path, "/_app/immutable/") || strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// For index.html and other root files, prevent caching.
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		c.Next()
	}
}
