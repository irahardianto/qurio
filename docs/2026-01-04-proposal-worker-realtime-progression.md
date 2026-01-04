# Proposal: Granular Real-Time Progress Reporting for Document Ingestion

**Date:** 2026-01-04
**Status:** Draft
**Priority:** Medium (UX Enhancement)

## 1. Problem Statement
The current document ingestion process is a "black box" operation from the user's perspective. When a user uploads a large document (e.g., a 500-page PDF), the system transitions to a "Pending" state which can persist for 10-30 minutes without providing any feedback.

**Current Limitations:**
- **Lack of Visibility:** Users cannot distinguish between a system hang and a long-running job.
- **Worker Opacity:** The `docling` library's `DocumentConverter.convert()` method blocks until the entire document is processed, offering no built-in hooks for page-level progress updates.
- **Process Isolation:** The worker runs conversion in a separate process (via `pebble`) which complicates communication back to the main application loop to report progress.

## 2. Business Value
- **Improved User Experience:** Providing a progress bar (e.g., "Processing page 45 of 200") builds trust and reduces frustration.
- **Better Observability:** Granular progress logs help developers identify specific pages that cause performance bottlenecks or crashes (e.g., "Stuck on page 89 for 5 minutes").
- **Early Failure Detection:** If progress stalls on a specific percentage for too long, the system (or user) can cancel the job earlier than the hard 30-minute timeout.

## 3. Technical Implementation Strategy

### 3.1 Custom Docling Pipeline
We cannot use `DocumentConverter` out-of-the-box for this. We must subclass `StandardPdfPipeline` or `PdfBackend` to inject a callback.

**Requirements:**
1.  **Custom Pipeline Class:** Create a class inheriting from `docling.pipeline.StandardPdfPipeline`.
2.  **Override Processing Loop:** Override the method responsible for iterating pages (likely `_process_pages` or similar internal method) to invoke a callback after each page.
3.  **Callback Interface:** Define a standard callback signature: `on_progress(current_page: int, total_pages: int)`.

### 3.2 Inter-Process Communication (IPC)
Since the worker runs in a separate process (isolated for stability), we need a way to send progress events back to the main asyncio loop.

**Mechanism:**
- **Shared Queue:** Pass a `multiprocessing.Queue` to the worker process.
- **Event Loop Integration:** The main `asyncio` loop polls this queue (or uses a thread to monitor it) and forwards events to the backend.

### 3.3 Backend & Frontend Integration
- **Webhook/GRPC:** The main worker process sends progress events to the Backend API (`POST /sources/{id}/progress`).
- **Server-Sent Events (SSE):** The Backend broadcasts these events to the Frontend via an SSE channel to update the UI in real-time.

## 4. Implementation Plan
1.  **Research:** Audit `docling` source code to identify the cleanest override point in `StandardPdfPipeline`.
2.  **Prototype:** Create a standalone script demonstrating a custom pipeline with a print-callback.
3.  **Integration:** Wire the custom pipeline into `apps/ingestion-worker/handlers/file.py` and set up the IPC queue.
4.  **API Hook:** Connect the worker events to the existing backend progress endpoint.
5.  **UI Update:** Add a progress bar component to the Source Card in the frontend.

## 5. Risks & Mitigation
- **Performance Overhead:** Frequent IPC calls might slow down processing. *Mitigation:* Report progress only every N pages or 5%.
- **Library Updates:** Internal `docling` APIs might change, breaking our custom subclass. *Mitigation:* Pin `docling` version strictly and add regression tests.
