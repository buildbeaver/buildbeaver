package models

// SystemName is the name of a system that has provided data stored in the database.
// This can include external SCMs like GitHub, other external systems, as well as our own server system
// and associated tools.
type SystemName string

func (s SystemName) String() string {
	return string(s)
}

// BuildBeaverSystem is the system name to use for data that is sourced from BuildBeaver itself.
const BuildBeaverSystem SystemName = "buildbeaver"

// BuildBeaverToolsSystem is the system name to use for data that is sourced from the bb-tools admin app.
const BuildBeaverToolsSystem SystemName = "bb-tools"

// TestsSystem is the system name to use when data is being created for unit or integration tests
const TestsSystem SystemName = "tests"
