package models

var RepoCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: RepoResourceKind,
}

var RepoReadOperation = &Operation{
	Name:         "read",
	ResourceKind: RepoResourceKind,
}

var RepoUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: RepoResourceKind,
}

var RepoDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: RepoResourceKind,
}

var RepoAccessControlOperations = []*Operation{
	RepoReadOperation,
	RepoUpdateOperation,
	RepoDeleteOperation,
	BuildCreateOperation,
	SecretCreateOperation,
	SecretReadPlaintextOperation,
	BuildReadOperation,
	BuildUpdateOperation,
	ArtifactCreateOperation,
	ArtifactReadOperation,
	ArtifactUpdateOperation,
	ArtifactDeleteOperation,
}
