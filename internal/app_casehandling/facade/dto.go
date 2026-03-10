package facade

// InquiryDTO holds the data needed to handle an inbound inquiry.
type InquiryDTO struct {
	SenderPartyID   string
	OperatorPartyID string
	AgentPartyID    string
	SubjectID       string
	Body            string
}
