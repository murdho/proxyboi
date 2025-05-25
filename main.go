package main

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const cacheDir = "./cache"

type CacheEntry struct {
	Data         []byte `json:"data"`
	WasGzipped   bool   `json:"was_gzipped"`
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run main.go <listen-port> <target-url>\nExample: go run main.go 8080 https://api.example.com")
	}

	port := os.Args[1]
	targetURL := os.Args[2]

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal("Invalid target URL:", err)
	}

	// Create cache directory
	os.MkdirAll(cacheDir, 0755)

	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Override the Director to modify requests
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	// Override ModifyResponse to cache responses
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode == 200 && isIdempotent(resp.Request.Method) {
			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body.Close()

			// Check if response was gzipped
			wasGzipped := strings.Contains(resp.Header.Get("Content-Encoding"), "gzip")
			
			// Decompress if gzipped
			var jsonData []byte
			if wasGzipped {
				reader, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					return err
				}
				defer reader.Close()
				jsonData, err = io.ReadAll(reader)
				if err != nil {
					return err
				}
			} else {
				jsonData = body
			}

			// Cache the entry with compression info
			cacheEntry := CacheEntry{
				Data:       jsonData,
				WasGzipped: wasGzipped,
			}
			cacheData, _ := json.Marshal(cacheEntry)
			cacheKey := getCacheKey(resp.Request)
			cachePath := filepath.Join(cacheDir, cacheKey+".json")
			os.WriteFile(cachePath, cacheData, 0644)

			// Restore the response body (original compressed if it was compressed)
			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
		}
		return nil
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only cache idempotent methods
		if isIdempotent(r.Method) {
			// Check cache first
			cacheKey := getCacheKey(r)
			cachePath := filepath.Join(cacheDir, cacheKey+".json")
			
			if cacheData, err := os.ReadFile(cachePath); err == nil {
				var cacheEntry CacheEntry
				if json.Unmarshal(cacheData, &cacheEntry) == nil {
					fmt.Printf("Cache HIT: %s %s\n", r.Method, r.URL.String())
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Cache", "HIT")
					
					// Only compress if original API used gzip AND client accepts it
					if cacheEntry.WasGzipped && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
						w.Header().Set("Content-Encoding", "gzip")
						gzWriter := gzip.NewWriter(w)
						defer gzWriter.Close()
						gzWriter.Write(cacheEntry.Data)
					} else {
						w.Write(cacheEntry.Data)
					}
					return
				}
			}
			
			fmt.Printf("Cache MISS: %s %s\n", r.Method, r.URL.String())
			w.Header().Set("X-Cache", "MISS")
		} else {
			fmt.Printf("No cache (non-idempotent): %s %s\n", r.Method, r.URL.String())
			w.Header().Set("X-Cache", "SKIP")
		}

		proxy.ServeHTTP(w, r)
	})

	fmt.Printf("Caching proxy listening on port %s, forwarding to %s\n", port, targetURL)
	fmt.Printf("Cache directory: %s\n", cacheDir)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getCacheKey(r *http.Request) string {
	key := fmt.Sprintf("%s_%s_%s", r.Method, r.URL.Path, r.URL.RawQuery)
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}

func isIdempotent(method string) bool {
	return method == "GET" || method == "HEAD"
}