# PDF Ingestion Reliability & Limits

**Date:** 2026-01-12
**Status:** Implemented

## Issue
PDF ingestion was failing for large documents (>10 pages) because the generated Markdown content exceeded the default NSQ message size limit of 1MB (`E_BAD_MESSAGE PUB message too big`).

## Solution
The system configuration was updated to support larger message payloads throughout the ingestion pipeline.

## Configuration Changes
1.  **NSQ Server (`nsqd`):**
    *   Configured with `--max-msg-size`.
    *   Default increased to **10MB** (10,485,760 bytes).
    *   Configurable via `NSQ_MAX_MSG_SIZE` environment variable in `docker-compose.yml`.

2.  **Backend (`go-nsq`):**
    *   Updated `nsq.Config` in Producer (`bootstrap.go`) and Consumer (`main.go`) to use the `NSQ_MAX_MSG_SIZE` value from configuration.
    *   Ensures the backend can both send and receive large payloads.

3.  **Workers (`pynsq`):**
    *   No code changes required as `pynsq` writers respect the server's negotiated limits.

## Verification
*   **Integration Test:** `TestTopicRouting` (and `integration_suite.go`) updated to configure the ephemeral NSQ container with the 10MB limit.
*   **Manual:** Verified successful parsing of large PDFs that previously failed.

## Defaults
*   **Default Limit:** 10MB
*   **Override:** Set `NSQ_MAX_MSG_SIZE` in `.env` (e.g., `NSQ_MAX_MSG_SIZE=52428800` for 50MB).
