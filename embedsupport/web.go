// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package embedsupport

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/the-open-agent/openagent/conf"
)

var mainJSRe = regexp.MustCompile(`^static/js/main\.[a-f0-9]+\.js$`)

// ServeEmbedded serves a frontend asset from the embedded web/build FS.
// urlPath is the raw request URL path (e.g. "/", "/static/js/main.abc.js").
// Must only be called when WebFS() != nil.
func ServeEmbedded(w http.ResponseWriter, r *http.Request, urlPath string) {
	embedPath := strings.TrimPrefix(urlPath, "/")
	if embedPath == "" {
		embedPath = "index.html"
	}

	// Fall back to index.html for any path not in the embedded FS (SPA routing).
	if _, err := webFS.Open(embedPath); err != nil {
		embedPath = "index.html"
	}

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		serveEmbeddedFile(gzipWriter{Writer: gz, ResponseWriter: w}, r, embedPath)
	} else {
		serveEmbeddedFile(w, r, embedPath)
	}
}

// serveEmbeddedFile writes a single embedded asset to w, applying the
// Casdoor config substitution for the main.*.js bundle.
func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, embedPath string) {
	data, err := fs.ReadFile(webFS, embedPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	content := string(data)

	if mainJSRe.MatchString(embedPath) {
		serverUrl := conf.GetConfigString("casdoorEndpoint")
		clientId := conf.GetConfigString("clientId")
		appName := conf.GetConfigString("casdoorApplication")
		organizationName := conf.GetConfigString("casdoorOrganization")

		content = regexp.MustCompile(`serverUrl:"[^"]*"`).ReplaceAllString(content, fmt.Sprintf(`serverUrl:"%s"`, serverUrl))
		content = regexp.MustCompile(`clientId:"[^"]*"`).ReplaceAllString(content, fmt.Sprintf(`clientId:"%s"`, clientId))
		content = regexp.MustCompile(`appName:"[^"]*"`).ReplaceAllString(content, fmt.Sprintf(`appName:"%s"`, appName))
		content = regexp.MustCompile(`organizationName:"[^"]*"`).ReplaceAllString(content, fmt.Sprintf(`organizationName:"%s"`, organizationName))
	}

	if ct := mime.TypeByExtension(filepath.Ext(embedPath)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	http.ServeContent(w, r, filepath.Base(embedPath), time.Time{}, strings.NewReader(content))
}

type gzipWriter struct {
	io.Writer
	http.ResponseWriter
}

func (g gzipWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}
