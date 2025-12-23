# Implementation Summary - MVP Part 3.3: Crawler Fixes & Recursion

## Implemented Features
- **Backend Handler Fix:** Updated `POST /api/v1/sources` to correctly decode `max_depth` and `exclusions`.
- **Backend Payload:** Updated `Service.Create` and `Service.ReSync` to use `max_depth` (was `depth`) in NSQ payload.
- **Worker Configuration:** Added `GEMINI_API_KEY` to `config.py` and `docker-compose.yml`. Updated `requirements.txt` to include `crawl4ai[google]`.
- **Recursive Crawling:** Implemented `BFSDeepCrawlStrategy` in `apps/ingestion-worker/handlers/web.py` to support `max_depth`.
- **Iterative Processing:** Updated `apps/ingestion-worker/main.py` to handle list of results from recursive crawl and publish individual messages to `ingest.result` topic.
- **Tests:** Added `TestCreate_FullPayload` to backend tests and fixed `source_test.go` broken signatures and assertions.

## Tests Completed
- `go test ./apps/backend/features/source` (Passed)
- `go test ./apps/backend/internal/worker` (Passed)

## How to Run
1.  **Start the stack:**
    ```bash
    export GEMINI_API_KEY=your_key_here
    docker-compose up --build
    ```
2.  **Create a recursive source:**
    ```bash
    curl -X POST http://localhost:8081/api/v1/sources \
      -H "Content-Type: application/json" \
      -d 
      {
        "url": "https://crawl4ai.com/mkdocs/",
        "max_depth": 1,
        "exclusions": ["/blog"]
      }
    ```
3.  **Verify:**
    - Check `ingestion-worker` logs for "Starting crawl... with depth 1".
    - Check `backend` logs for multiple "received result" messages.
    - Check `GET /api/v1/sources/{id}` for populated chunks.
