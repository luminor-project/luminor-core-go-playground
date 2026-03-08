package facade

// DocumentIndexedEvent is dispatched when a document has been indexed.
type DocumentIndexedEvent struct {
	DocumentID string
	Title      string
}
