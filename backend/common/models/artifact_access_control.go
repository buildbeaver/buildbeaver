package models

var ArtifactCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: ArtifactResourceKind,
}

var ArtifactReadOperation = &Operation{
	Name:         "read",
	ResourceKind: ArtifactResourceKind,
}

var ArtifactUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: ArtifactResourceKind,
}

var ArtifactDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: ArtifactResourceKind,
}

var ArtifactAccessControlOperations = []*Operation{
	ArtifactReadOperation,
	ArtifactUpdateOperation,
	ArtifactDeleteOperation,
}
