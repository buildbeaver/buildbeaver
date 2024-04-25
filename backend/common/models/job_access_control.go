package models

var JobCreateOperation = &Operation{
	Name:         "create",
	ResourceKind: JobResourceKind,
}

var JobReadOperation = &Operation{
	Name:         "read",
	ResourceKind: JobResourceKind,
}

var JobUpdateOperation = &Operation{
	Name:         "update",
	ResourceKind: JobResourceKind,
}

var JobAccessControlOperations = []*Operation{
	JobCreateOperation,
	JobReadOperation,
	JobUpdateOperation,
}
