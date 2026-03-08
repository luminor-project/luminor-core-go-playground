# RAG (Retrieval-Augmented Generation) Vertical

Self-hosted RAG stack for indexing, searching, and querying documents with LLM-generated answers. Fully local — no cloud API dependency.

## Architecture

```
internal/rag/
├── domain/
│   ├── document.go      # Document, Chunk, SearchResult entities
│   ├── chunker.go       # Sliding-window text chunker (pure logic)
│   └── service.go       # RAGService: index, search, chat orchestration
├── infra/
│   └── repo.go          # PostgreSQL + pgvector repository
├── facade/
│   ├── dto.go           # IndexDocumentDTO, SearchResultDTO, ChatResponseDTO
│   ├── events.go        # DocumentIndexedEvent
│   └── impl.go          # Facade wiring
├── web/
│   ├── handler.go       # JSON API handlers
│   └── routes.go        # Route registration
└── testharness/
    └── factory.go       # Test factories
```

Follows the same vertical-slice pattern as `account/` and `organization/`. Cross-vertical communication only through `facade/` package.

## Stack Components

| Component           | Image                    | Purpose                                                        |
| ------------------- | ------------------------ | -------------------------------------------------------------- |
| **db-rag**          | `pgvector/pgvector:pg17` | Dedicated PostgreSQL instance with pgvector for vector storage |
| **local-inference** | `ollama/ollama:latest`   | Model inference: embeddings + chat via HTTP API                |

### Database Separation

RAG data lives on a dedicated `db-rag` instance (`pgvector/pgvector:pg17`, port 5433), fully separate from the `db-business` instance (`postgres:17`, port 5432) that holds business/domain data. Each has its own volume, own migration track, and own connection pool. The vector extension is installed automatically on first startup via `docker/dev/init-rag-db.sh`.

### Models

| Model              | Dimensions | Use                                |
| ------------------ | ---------- | ---------------------------------- |
| `nomic-embed-text` | 768        | Embedding generation (~275 MB RAM) |
| `llama3.1:8b`      | —          | Chat/generation (~4.7 GB RAM)      |
| `llama3.2:3b`      | —          | Lighter alternative for dev        |

## Setup

### 1. Start services

```bash
docker compose up -d
```

The local inference service starts automatically. The `db-rag` instance with pgvector is created alongside the `db-business` instance.

### 2. Pull models

```bash
docker compose exec local-inference ollama pull nomic-embed-text
docker compose exec local-inference ollama pull llama3.1:8b
```

First pull downloads the model weights. Subsequent starts load from the `local_inference_data` volume.

### 3. Run migrations

```bash
mise run migrate-db:business   # business database (accounts, orgs, etc.)
mise run migrate-db:rag        # RAG database (documents, chunks, vectors)
```

Each database has its own migration track under `migrations/business/` and `migrations/rag/`.

## Configuration

| Variable              | Default                                                                 | Description                                   |
| --------------------- | ----------------------------------------------------------------------- | --------------------------------------------- |
| `RAG_DATABASE_URL`    | `postgres://luminor:luminor@localhost:5443/luminor_rag?sslmode=disable` | Dedicated RAG database (separate instance)    |
| `LOCAL_INFERENCE_URL` | `http://local-inference:11434`                                          | Local inference service API base URL (Ollama) |
| `EMBED_MODEL`         | `nomic-embed-text`                                                      | Model for embedding generation                |
| `CHAT_MODEL`          | `llama3.1:8b`                                                           | Model for chat completion                     |

## API Endpoints

All endpoints accept and return JSON. No authentication required (add `auth.RequireAuth` middleware if needed).

### Index a document

```
POST /api/rag/documents
```

```json
{
    "title": "Project Architecture",
    "source_type": "markdown",
    "content": "Full document text here...",
    "metadata": { "author": "team", "version": "1.0" }
}
```

Response `201`:

```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Project Architecture",
    "source_type": "markdown"
}
```

The document is chunked (~500 tokens per chunk, 50 token overlap), each chunk is embedded via the local inference service, and everything is stored in a single PostgreSQL transaction.

### Delete a document

```
DELETE /api/rag/documents/{documentId}
```

Response `204` (no content). Chunks are cascade-deleted.

### Search documents

```
POST /api/rag/search
```

```json
{
    "query": "How does authentication work?",
    "limit": 5,
    "threshold": 0.3
}
```

Response `200`:

```json
{
    "results": [
        {
            "chunk_id": "...",
            "document_id": "...",
            "content": "The authentication module uses...",
            "score": 0.87,
            "title": "Project Architecture",
            "source_type": "markdown",
            "metadata": { "author": "team" }
        }
    ]
}
```

`limit` defaults to 5, `threshold` defaults to 0.3 (cosine similarity, 0-1 scale).

### Chat (RAG query)

```
POST /api/rag/chat
```

```json
{
    "query": "How does authentication work?",
    "limit": 5,
    "threshold": 0.3
}
```

Response `200`:

```json
{
    "answer": "Based on the documentation, authentication uses...",
    "sources": [
        {
            "chunk_id": "...",
            "document_id": "...",
            "content": "...",
            "score": 0.87,
            "title": "Project Architecture",
            "source_type": "markdown",
            "metadata": {}
        }
    ]
}
```

Pipeline: embed query -> cosine similarity search -> assemble context from top-k chunks -> inference chat completion with context -> return answer + sources.

## RAG Pipeline

### Indexing

```
Document text
  -> chunk (sliding window, ~500 tokens, ~50 overlap)
  -> embed each chunk via local inference service (concurrent, max 5 in-flight)
  -> store document + chunks + vectors in PostgreSQL (single transaction)
```

### Querying

```
User query
  -> embed via local inference service
  -> pgvector cosine similarity (HNSW index) -> top-k chunks
  -> assemble context from chunks
  -> inference chat with system prompt + context + query
  -> return answer + source references
```

## Database Schema

```sql
-- migrations/rag/001_create_rag_tables.up.sql
-- vector extension is installed by docker/dev/init-rag-db.sh on database creation

CREATE TABLE rag_documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'text',
    content     TEXT NOT NULL,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rag_chunks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES rag_documents(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    content     TEXT NOT NULL,
    token_count INT NOT NULL DEFAULT 0,
    embedding   vector(768) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- HNSW index for fast approximate nearest neighbor search
CREATE INDEX idx_rag_chunks_embedding ON rag_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
```

## New Dependencies

| Package                           | Purpose                                          |
| --------------------------------- | ------------------------------------------------ |
| `github.com/pgvector/pgvector-go` | pgvector `Vector` type for pgx (pure Go, no CGO) |

`golang.org/x/sync` (already transitive) is promoted to direct for concurrent chunk embedding.

## Production Notes

- **CGO_ENABLED=0 safe** — pgvector-go is pure Go, local inference service is HTTP. No Dockerfile changes needed.
- **Cold start** — First local inference request after restart loads the model into memory (~10s). Consider a warm-up request or health check.
- **Model changes** — Store the embedding model name in document metadata; provide a re-index mechanism when switching models (embeddings from different models are not comparable).
- **Scaling** — pgvector with HNSW is comfortable to ~1M vectors. Beyond that, consider a dedicated vector DB.
- **GPU acceleration** — Uncomment the `deploy.resources` block in `docker-compose.yml` for NVIDIA GPU support (dramatically faster LLM inference).

## Testing

```bash
# Unit tests (chunker + service with mocks + ollama client)
mise run in-app-container go test ./internal/rag/domain/ ./internal/platform/ollama/ -v

# Architecture boundary test (verifies RAG vertical isolation)
mise run in-app-container go test ./tools/archtest/ -run TestCurrentRepoPolicyHasNoViolations -v
```
