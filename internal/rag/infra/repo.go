package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
	"github.com/luminor-project/luminor-core-go-playground/internal/rag/domain"
)

// PostgresRepository implements domain.Repository using PostgreSQL with pgvector.
type PostgresRepository struct {
	pool *pgxpool.Pool
	db   dbExecutor
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewPostgresRepository creates a new PostgreSQL-backed RAG repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, db: pool}
}

func (r *PostgresRepository) withTx(tx pgx.Tx) *PostgresRepository {
	return &PostgresRepository{pool: r.pool, db: tx}
}

func (r *PostgresRepository) ExecuteInTx(ctx context.Context, fn func(repo domain.Repository) error) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		return fn(r.withTx(tx))
	})
}

func (r *PostgresRepository) CreateDocument(ctx context.Context, doc domain.Document) error {
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO rag_documents (id, title, source_type, content, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		doc.ID, doc.Title, doc.SourceType, doc.Content, metadataJSON,
		doc.CreatedAt, doc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	return nil
}

func (r *PostgresRepository) CreateChunks(ctx context.Context, chunks []domain.Chunk) error {
	for _, chunk := range chunks {
		vec := pgvector.NewVector(chunk.Embedding)
		_, err := r.db.Exec(ctx,
			`INSERT INTO rag_chunks (id, document_id, chunk_index, content, token_count, embedding, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			chunk.ID, chunk.DocumentID, chunk.ChunkIndex, chunk.Content,
			chunk.TokenCount, vec, chunk.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert chunk %d: %w", chunk.ChunkIndex, err)
		}
	}

	return nil
}

func (r *PostgresRepository) FindSimilarChunks(ctx context.Context, embedding []float32, limit int, threshold float64) ([]domain.SearchResult, error) {
	vec := pgvector.NewVector(embedding)

	rows, err := r.db.Query(ctx,
		`SELECT c.id, c.document_id, c.content,
		        1 - (c.embedding <=> $1::vector) AS score,
		        d.title, d.source_type, d.metadata
		 FROM rag_chunks c
		 JOIN rag_documents d ON c.document_id = d.id
		 WHERE 1 - (c.embedding <=> $1::vector) > $3
		 ORDER BY c.embedding <=> $1::vector
		 LIMIT $2`,
		vec, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("query similar chunks: %w", err)
	}
	defer rows.Close()

	var results []domain.SearchResult
	for rows.Next() {
		var sr domain.SearchResult
		var metadataJSON []byte
		if err := rows.Scan(&sr.ChunkID, &sr.DocumentID, &sr.Content,
			&sr.Score, &sr.Title, &sr.SourceType, &metadataJSON); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		if err := json.Unmarshal(metadataJSON, &sr.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
		results = append(results, sr)
	}

	return results, rows.Err()
}

func (r *PostgresRepository) DeleteDocumentAndChunks(ctx context.Context, documentID string) error {
	// Chunks are deleted by ON DELETE CASCADE.
	_, err := r.db.Exec(ctx, `DELETE FROM rag_documents WHERE id = $1`, documentID)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}
