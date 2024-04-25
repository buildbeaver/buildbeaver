/*
BuildBeaver Dynamic Build API - OpenAPI 3.0

This is the BuildBeaver Dynamic Build API.

API version: 0.3.00
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

import (
	"encoding/json"
)

// checks if the BuildDefinition type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &BuildDefinition{}

// BuildDefinition struct for BuildDefinition
type BuildDefinition struct {
	// The version number of the build definition format used
	Version string `json:"version"`
	Jobs []JobDefinition `json:"jobs"`
	AdditionalProperties map[string]interface{}
}

type _BuildDefinition BuildDefinition

// NewBuildDefinition instantiates a new BuildDefinition object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewBuildDefinition(version string, jobs []JobDefinition) *BuildDefinition {
	this := BuildDefinition{}
	this.Version = version
	this.Jobs = jobs
	return &this
}

// NewBuildDefinitionWithDefaults instantiates a new BuildDefinition object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewBuildDefinitionWithDefaults() *BuildDefinition {
	this := BuildDefinition{}
	return &this
}

// GetVersion returns the Version field value
func (o *BuildDefinition) GetVersion() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Version
}

// GetVersionOk returns a tuple with the Version field value
// and a boolean to check if the value has been set.
func (o *BuildDefinition) GetVersionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Version, true
}

// SetVersion sets field value
func (o *BuildDefinition) SetVersion(v string) {
	o.Version = v
}

// GetJobs returns the Jobs field value
func (o *BuildDefinition) GetJobs() []JobDefinition {
	if o == nil {
		var ret []JobDefinition
		return ret
	}

	return o.Jobs
}

// GetJobsOk returns a tuple with the Jobs field value
// and a boolean to check if the value has been set.
func (o *BuildDefinition) GetJobsOk() ([]JobDefinition, bool) {
	if o == nil {
		return nil, false
	}
	return o.Jobs, true
}

// SetJobs sets field value
func (o *BuildDefinition) SetJobs(v []JobDefinition) {
	o.Jobs = v
}

func (o BuildDefinition) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o BuildDefinition) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["version"] = o.Version
	toSerialize["jobs"] = o.Jobs

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}

	return toSerialize, nil
}

func (o *BuildDefinition) UnmarshalJSON(bytes []byte) (err error) {
	varBuildDefinition := _BuildDefinition{}

	if err = json.Unmarshal(bytes, &varBuildDefinition); err == nil {
		*o = BuildDefinition(varBuildDefinition)
	}

	additionalProperties := make(map[string]interface{})

	if err = json.Unmarshal(bytes, &additionalProperties); err == nil {
		delete(additionalProperties, "version")
		delete(additionalProperties, "jobs")
		o.AdditionalProperties = additionalProperties
	}

	return err
}

type NullableBuildDefinition struct {
	value *BuildDefinition
	isSet bool
}

func (v NullableBuildDefinition) Get() *BuildDefinition {
	return v.value
}

func (v *NullableBuildDefinition) Set(val *BuildDefinition) {
	v.value = val
	v.isSet = true
}

func (v NullableBuildDefinition) IsSet() bool {
	return v.isSet
}

func (v *NullableBuildDefinition) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableBuildDefinition(val *BuildDefinition) *NullableBuildDefinition {
	return &NullableBuildDefinition{value: val, isSet: true}
}

func (v NullableBuildDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableBuildDefinition) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

