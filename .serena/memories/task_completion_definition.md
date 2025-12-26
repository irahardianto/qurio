# MVP Part 5.1 Completion

## Completed Features
- **Failed Jobs Management:**
  - `failed_jobs` table and repository.
  - `JobService` and API (`GET /jobs/failed`, `POST /jobs/:id/retry`).
  - Worker `ResultConsumer` saves failed jobs.
  - Frontend `JobsView` and `job.store`.
- **Dashboard & Stats:**
  - `StatsService` and API (`GET /stats`).
  - Frontend `DashboardView` and `stats.store`.
- **Source Cleanup:**
  - `DeleteChunksBySourceID` in Weaviate adapter.
  - `SourceService.Delete` calls cleanup before DB delete.
- **Documentation:**
  - Updated `README.md` with full usage and architecture guide.

## Technical Details
- **Idempotency:** Failed jobs can be retried safely.
- **Data Integrity:** Deleting a source removes its chunks from Weaviate.
- **Frontend:** Dashboard is now the home page. Sidebar includes Failed Jobs link.
