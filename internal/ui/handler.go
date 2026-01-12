package ui

import (
	"io/fs"
	"net/http"
	"strings"

	dabbi "github.com/mjshashank/dabbi"
)

// Handler returns an HTTP handler that serves the embedded UI
// with SPA fallback for client-side routing
func Handler() http.Handler {
	uiFS, err := dabbi.GetUIFS()
	if err != nil {
		// Return a fallback handler if UI is not available
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>dabbi</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #0f0f1a;
            color: #e0e0e0;
        }
        .container { text-align: center; }
        h1 { font-size: 48px; margin-bottom: 10px; color: #00d4ff; }
        p { color: #8888aa; }
        code {
            background: #1a1a2e;
            padding: 2px 8px;
            border-radius: 4px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>dabbi</h1>
    </div>
</body>
</html>`))
		})
	}

	fileServer := http.FileServer(http.FS(uiFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to serve static files directly
		if path != "/" && !strings.HasSuffix(path, "/") {
			// Check if the file exists
			cleanPath := strings.TrimPrefix(path, "/")
			if f, err := uiFS.Open(cleanPath); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html for all routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// StaticHandler returns a handler that only serves static files without SPA fallback
func StaticHandler() http.Handler {
	uiFS, err := dabbi.GetUIFS()
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(uiFS))
}

// IndexHTML returns the index.html content for manual handling
func IndexHTML() ([]byte, error) {
	uiFS, err := dabbi.GetUIFS()
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(uiFS, "index.html")
}
