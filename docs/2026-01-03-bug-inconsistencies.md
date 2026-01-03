Based on a forensic analysis of the code residing in the apps/ directory, several architectural and implementation inconsistencies remain present. These persist despite recent refactors and represent deviations from the project's established standards.
1. Ingestion Contract Inconsistency (Worker Handlers)
There is a functional mismatch between the web and file handlers in the ingestion worker regarding the metadata they return to the backend.
• Missing Path in File Ingestion: The web handler (apps/ingestion-worker/handlers/web.py) correctly extracts and returns a path field (breadcrumbs) derived from the URL. However, the file handler (apps/ingestion-worker/handlers/file.py) does not include a path field in its return dictionary.
• Impact on Contextual Embeddings: The Backend’s ResultConsumer expects a Path string to build the contextualString for vectorization. Because the file worker omits this, embeddings for uploaded documents will have an empty "Path" line in their context header, reducing semantic precision compared to web sources.
2. Frontend Import and Path Inconsistency
The frontend codebase exhibits inconsistent patterns for importing components and utilizing the established path aliases (@/).
• Mixed Relative vs. Aliased Paths: In apps/frontend/src/features/sources/SourceList.vue, the code uses relative paths for some components (e.g., ../../components/ui/StatusBadge.vue) while using aliases for others (e.g., @/components/ui/card) in the same file.
• Redundant Config: Path aliases are defined separately in both tsconfig.json and tsconfig.app.json, leading to potential resolution conflicts if one is updated without the other.
3. Data Type Casing Inconsistency
There is an inconsistency in the naming convention for data structures between the backend's internal types and the frontend's store interfaces.
• CamelCase vs. snake_case: In apps/frontend/src/features/sources/source.store.ts, the Chunk interface uses CamelCase fields (e.g., ChunkIndex, SourceURL, SourceID).
• Conflict with API Standards: This deviates from the project's general preference for snake_case in JSON/API interactions (e.g., source_id, max_depth, gemini_api_key). While the frontend store maps these, the mixed usage within the same file (e.g., total_chunks is snake_case in the Source interface) creates a disjointed developer experience.
4. Vector Storage vs. Contextual Metadata
While the project successfully implemented "Contextual Embeddings," there is a present inconsistency in what is stored versus what is vectorized.
• Vector Content vs. Property Metadata: The ResultConsumer prepends the Source Name to the text before embedding it into a vector. However, the Chunk struct in the worker types and the Weaviate StoreChunk implementation do not include the Source Name as a filterable property.
• Result: While the vector "knows" the source name semantically, an AI agent cannot explicitly filter for a source name (e.g., "Show me code only from the 'React Docs' source") because that field is not stored in the database's metadata schema, only the sourceId.
5. Present Cruft in Component Archetypes
The "Sage" design refresh implemented a specific aesthetic (Void Black/Cognitive Blue), but the apps/frontend folder still contains original template files that violate this archetype.
• Non-standard Components: apps/frontend/src/components/HelloWorld.vue still exists with its default Vite/Vue styles and colors, which do not align with the style.css brand variables or the shadcn-vue implementation used in the actual feature views.

--------------------------------------------------------------------------------
Analogy: The codebase is like a library that has just installed a high-tech digital catalog (the Source and Settings features). However, the librarian's assistant (the ingestion-worker) is labeling books from the internet with full shelf-paths but forgetting to put any path on the books physically handed to them (file uploads). Consequently, some books end up on the "Semantic" shelf without anyone knowing which room they actually belong to.