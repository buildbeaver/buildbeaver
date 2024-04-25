package models

var RunnerCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: RunnerResourceKind,
}

var RunnerReadOperation = &Operation{
	Name:         "read",
	ResourceKind: RunnerResourceKind,
}

var RunnerUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: RunnerResourceKind,
}

var RunnerDeleteOperation = &Operation{
	Name:         "delete",
	ResourceKind: RunnerResourceKind,
}

var RunnerAccessControlOperations = []*Operation{
	RunnerReadOperation,
	RunnerUpdateOperation,
	RunnerDeleteOperation,
}
