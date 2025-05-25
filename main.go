package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
)

const cacheDir = "./cache"

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
		if resp.StatusCode == 200 {
			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body.Close()

			// Cache the response
			cacheKey := getCacheKey(resp.Request)
			cachePath := filepath.Join(cacheDir, cacheKey)
			os.WriteFile(cachePath, body, 0644)

			// Restore the response body
			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
		}
		return nil
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check cache first
		cacheKey := getCacheKey(r)
		cachePath := filepath.Join(cacheDir, cacheKey)
		
		if data, err := os.ReadFile(cachePath); err == nil {
			fmt.Printf("Cache HIT: %s %s\n", r.Method, r.URL.String())
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write(data)
			return
		}

		fmt.Printf("Cache MISS: %s %s\n", r.Method, r.URL.String())
		w.Header().Set("X-Cache", "MISS")
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