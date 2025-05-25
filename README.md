# HTTP Cache Proxy

A minimal Go CLI tool that caches JSON API responses to avoid hitting rate limits and speed up development.

*Fully cooked by Claude.*

## Usage

```bash
go run main.go <port> <target-url>
```

Examples:
```bash
go run main.go 8080 https://api.github.com
go run main.go 3000 https://jsonplaceholder.typicode.com
```

Then make requests to `http://localhost:<port>` instead of the API directly.

## How it works

- Only caches `GET` and `HEAD` requests (idempotent methods)
- Stores responses as readable JSON files in `./cache/`
- Handles gzip compression properly (decompresses for storage, re-compresses when serving)
- Cache key = MD5 hash of method + path + query params

## Cache behavior

- **First request**: `Cache MISS` → fetches from API and caches
- **Subsequent requests**: `Cache HIT` → serves from disk (with gzip if client supports it)
- **POST/PUT/DELETE**: `No cache` → always proxies through

## Compression handling

The proxy perfectly mirrors the original API's compression behavior:
- If the API sends gzipped responses → cached responses will also be gzipped when served
- If the API sends uncompressed responses → cached responses remain uncompressed
- Always stores readable JSON in cache files (decompressed for inspection)

This ensures your application behaves identically with both the proxy and the real API.

## Useful for

- Avoiding API rate limits during development
- Working offline with previously fetched data
- Speeding up repetitive API calls in scripts
- Testing your app's gzip handling without hitting the real API

## Cache management

- Cache files: `./cache/*.json`
- Clear cache: `rm -rf cache/`
- Inspect responses: `cat cache/<hash>.json`