package models

var SecretCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: SecretResourceKind,
}

var SecretReadOperation = &Operation{
	Name:         "read",
	ResourceKind: SecretResourceKind,
}

var SecretReadPlaintextOperation = &Operation{
	Name:         "read_plaintext",
	ResourceKind: SecretResourceKind,
}

var SecretUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: SecretResourceKind,
}

var SecretDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: SecretResourceKind,
}

var secretAccessControlOperations = []*Operation{
	SecretReadOperation,
	SecretUpdateOperation,
	SecretDeleteOperation,
	SecretReadPlaintextOperation,
}
