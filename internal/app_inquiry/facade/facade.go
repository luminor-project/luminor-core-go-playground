package facade

// SubmitInquiryDTO holds data for a tenant submitting an inquiry.
type SubmitInquiryDTO struct {
	TenantPartyID string
	OrgID         string
	Body          string
}
