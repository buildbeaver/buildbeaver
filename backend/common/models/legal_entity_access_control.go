package models

var LegalEntityCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: LegalEntityResourceKind,
}

var LegalEntityReadOperation = &Operation{
	Name:         "read",
	ResourceKind: LegalEntityResourceKind,
}

var LegalEntityUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: LegalEntityResourceKind,
}

var LegalEntityDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: LegalEntityResourceKind,
}

// LegalEntityAccessControlOperations is the set of permissions made available to a Legal Entity's identity for
// resources that the Legal Entity owns.
var LegalEntityAccessControlOperations = []*Operation{
	LegalEntityCreateOperation,
	LegalEntityReadOperation,
	LegalEntityUpdateOperation,
	LegalEntityDeleteOperation,
	ArtifactCreateOperation,
	ArtifactReadOperation,
	ArtifactUpdateOperation,
	ArtifactDeleteOperation,
	BuildCreateOperation,
	BuildReadOperation,
	BuildUpdateOperation,
	RunnerCreateOperation,
	RunnerReadOperation,
	RunnerUpdateOperation,
	RunnerDeleteOperation,
	CredentialCreateOperation,
	CredentialReadOperation,
	CredentialUpdateOperation,
	CredentialDeleteOperation,
	GrantCreateOperation,
	GrantReadOperation,
	GrantUpdateOperation,
	GrantDeleteOperation,
	RepoCreateOperation,
	RepoReadOperation,
	RepoUpdateOperation,
	RepoDeleteOperation,
	GroupCreateOperation,
	GroupReadOperation,
	GroupUpdateOperation,
	GroupDeleteOperation,
	GroupMembershipCreateOperation,
	GroupMembershipReadOperation,
	GroupMembershipUpdateOperation,
	GroupMembershipDeleteOperation,
	SecretCreateOperation,
	SecretReadOperation,
	SecretUpdateOperation,
	SecretDeleteOperation,
	SecretReadPlaintextOperation,
}
