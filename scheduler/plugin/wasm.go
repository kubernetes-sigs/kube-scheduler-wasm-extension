/*
   Copyright 2023 The Kubernetes Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package wasm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// getURL parses the URL manually, so that it can resolve relative file paths,
// such as "file://../../path/to/plugin.wasm"
func getURL(ctx context.Context, url string) ([]byte, error) {
	firstColon := strings.IndexByte(url, ':')
	if firstColon == -1 {
		return nil, fmt.Errorf("invalid URL: %s", url)
	}

	scheme := url[:firstColon]
	switch scheme {
	case "http", "https":
		return httpGet(ctx, http.DefaultClient, url)
	case "file":
		guestPath := url[7:] // strip file://
		return os.ReadFile(guestPath)
	default:
		return nil, fmt.Errorf("unsupported URL scheme: %s", scheme)
	}
}

// httpGet returns a byte slice of the wasm module found at the given URL, or
// an error otherwise.
func httpGet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// golang/go#60240 recommends to just close the client body, instead of
	// draining it first.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received %v status code from %q", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
