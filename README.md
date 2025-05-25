# HTTP Cache Proxy

A minimal Go CLI tool that caches API responses to avoid hitting rate limits and speed up development.

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
- Handles gzip-compressed responses properly
- Cache key = MD5 hash of method + path + query params

## Cache behavior

- **First request**: `Cache MISS` → fetches from API and caches
- **Subsequent requests**: `Cache HIT` → serves from disk
- **POST/PUT/DELETE**: `No cache` → always proxies through

## Useful for

- Avoiding API rate limits during development
- Working offline with previously fetched data
- Speeding up repetitive API calls in scripts

## Cache management

- Cache files: `./cache/*.json`
- Clear cache: `rm -rf cache/`
- Inspect responses: `cat cache/<hash>.json`
