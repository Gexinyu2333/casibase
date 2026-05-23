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

package object

import "net/http"

// siteEndpointNeedsAutoFill is true while site-built-in's Endpoint is empty.
// Initialized at startup by InitSiteEndpoint and updated whenever site-built-in is saved.
var siteEndpointNeedsAutoFill bool

// InitSiteEndpoint reads site-built-in from the DB and sets siteEndpointNeedsAutoFill.
// Call once at startup after the adapter is ready.
func InitSiteEndpoint() {
	site, err := GetBuiltInSiteWithSecret()
	if err != nil || site == nil {
		return
	}
	siteEndpointNeedsAutoFill = site.Endpoint == ""
}

// AutoFillSiteEndpoint fills site-built-in's Endpoint from the request host when it is
// still empty. It is a no-op once the field has been set (or auto-filled).
func AutoFillSiteEndpoint(req *http.Request) {
	if !siteEndpointNeedsAutoFill {
		return
	}

	site, err := GetBuiltInSiteWithSecret()
	if err != nil || site == nil || site.Endpoint != "" {
		siteEndpointNeedsAutoFill = false
		return
	}

	host := req.Host
	if host == "" {
		return
	}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	site.Endpoint = scheme + "://" + host
	_, _ = UpdateSite("admin/site-built-in", site)
}
