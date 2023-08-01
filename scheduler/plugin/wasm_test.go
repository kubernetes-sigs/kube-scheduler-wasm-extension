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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/kube-scheduler-wasm-extension/scheduler/test"
)

var wasmMagicNumber = []byte{0x00, 0x61, 0x73, 0x6d}

func Test_getURL(t *testing.T) {
	testCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	bin, err := os.ReadFile(test.URLTestAllNoopWat[7:])
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	tmpFile, err := filepath.Abs(path.Join(tmpDir, "all_noop.wasm"))
	if err != nil {
		t.Fatal(err)
	}

	if err = os.WriteFile(tmpFile, bin, 0o0444); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bin)
	}))
	t.Cleanup(ts.Close)

	type testCase struct {
		name          string
		url           string
		expected      []byte
		expectedError string
	}

	tests := []testCase{
		{
			name:     "relative file valid",
			url:      "file://" + path.Join("..", "test", "testdata", "test", "all_noop.wasm"),
			expected: bin,
		},
		{
			name:     "file valid",
			url:      "file://" + tmpFile,
			expected: bin,
		},
		{
			name:          "not url",
			url:           "http",
			expectedError: "invalid URL: http",
		},
		{
			name:          "http invalid",
			url:           "http:// ",
			expectedError: `parse "http:// ": invalid character " " in host name`,
		},
		{
			name:          "https invalid",
			url:           "https:// ",
			expectedError: `parse "https:// ": invalid character " " in host name`,
		},
		{
			name:     "http",
			url:      ts.URL,
			expected: bin,
		},
		{
			name:          "unsupported scheme",
			url:           "ldap://foo/bar.wasm",
			expectedError: "unsupported URL scheme: ldap",
		},
		{
			name:          "file not found",
			url:           "file://testduta",
			expectedError: "open testduta: ",
		},
		{
			name: "relative dir not file",
			url:  "file://.",
			// Below ends in "is a directory" in unix, and "The handle is invalid." in windows.
			expectedError: "read .: ",
		},
		{
			name: "dir not file",
			url:  "file://" + tmpDir,
			// Below ends in "is a directory" in unix, and "The handle is invalid." in windows.
			expectedError: fmt.Sprintf("read %s: ", tmpDir),
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			bin, err := getURL(testCtx, tc.url)
			if tc.expectedError != "" {
				// Use substring match as the error can be different in Windows.
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected err %v to contain %s", err, tc.expectedError)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want, have := tc.expected, bin; !bytes.Equal(want, have) {
				t.Fatalf("unexpected binary: want %v, have %v", want, have)
			}
		})
	}
}

func Test_httpGet(t *testing.T) {
	wasmBinary := wasmMagicNumber
	wasmBinary = append(wasmBinary, 0x00, 0x00, 0x00, 0x00)
	cases := []struct {
		name          string
		handler       http.HandlerFunc
		expectedError string
	}{
		{
			name: "plain wasm binary",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(wasmBinary)
			},
		},
		// Compressed payloads are handled automatically by http.Client.
		{
			name: "compressed payload",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Encoding", "gzip")

				gw := gzip.NewWriter(w)
				defer gw.Close()
				_, _ = gw.Write(wasmBinary)
			},
		},
		{
			name: "http error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError: "received 500 status code",
		},
	}

	for _, proto := range []string{"http", "https"} {
		t.Run(proto, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					ts := httptest.NewServer(tc.handler)
					defer ts.Close()
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					_, err := httpGet(ctx, ts.Client(), ts.URL)

					if tc.expectedError != "" {
						// Use substring match as the error can be different in Windows.
						if !strings.Contains(err.Error(), tc.expectedError) {
							t.Fatalf("expected err %v to contain %s", err, tc.expectedError)
						}
					} else if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
				})
			}
		})
	}
}
