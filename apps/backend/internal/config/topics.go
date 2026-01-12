package config

const (
	// TopicIngestWeb is the NSQ topic for web crawling and ingestion tasks.
	TopicIngestWeb = "ingest.task.web"

	// TopicIngestFile is the NSQ topic for file processing tasks.
	TopicIngestFile = "ingest.task.file"

	// TopicIngestResult is the NSQ topic for ingestion results (success/failure).
	TopicIngestResult = "ingest.result"

	// TopicIngestEmbed is the NSQ topic for embedding generation tasks.
	TopicIngestEmbed = "ingest.embed"
)
