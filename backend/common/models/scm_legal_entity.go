package models

type SCMLegalEntity struct {
	ID           ExternalResourceID `json:"external_id"`
	Type         LegalEntityType    `json:"type"`
	Name         ResourceName       `json:"name"`
	LegalName    string             `json:"legal_name"`
	EmailAddress string             `json:"email_address"`
}
