package models

var CredentialCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: CredentialResourceKind,
}

var CredentialReadOperation = &Operation{
	Name:         "read",
	ResourceKind: CredentialResourceKind,
}

var CredentialUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: CredentialResourceKind,
}

var CredentialDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: CredentialResourceKind,
}

var CredentialAccessControlOperations = []*Operation{
	CredentialReadOperation,
	CredentialUpdateOperation,
	CredentialDeleteOperation,
}
