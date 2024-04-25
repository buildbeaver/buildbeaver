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

// checks if the DockerAWSAuthDefinition type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &DockerAWSAuthDefinition{}

// DockerAWSAuthDefinition struct for DockerAWSAuthDefinition
type DockerAWSAuthDefinition struct {
	// The ECR AWS region to authenticate to.
	AwsRegion *string `json:"aws_region,omitempty"`
	AwsAccessKeyId SecretStringDefinition `json:"aws_access_key_id"`
	AwsSecretAccessKey SecretStringDefinition `json:"aws_secret_access_key"`
	AdditionalProperties map[string]interface{}
}

type _DockerAWSAuthDefinition DockerAWSAuthDefinition

// NewDockerAWSAuthDefinition instantiates a new DockerAWSAuthDefinition object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewDockerAWSAuthDefinition(awsAccessKeyId SecretStringDefinition, awsSecretAccessKey SecretStringDefinition) *DockerAWSAuthDefinition {
	this := DockerAWSAuthDefinition{}
	this.AwsAccessKeyId = awsAccessKeyId
	this.AwsSecretAccessKey = awsSecretAccessKey
	return &this
}

// NewDockerAWSAuthDefinitionWithDefaults instantiates a new DockerAWSAuthDefinition object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewDockerAWSAuthDefinitionWithDefaults() *DockerAWSAuthDefinition {
	this := DockerAWSAuthDefinition{}
	return &this
}

// GetAwsRegion returns the AwsRegion field value if set, zero value otherwise.
func (o *DockerAWSAuthDefinition) GetAwsRegion() string {
	if o == nil || IsNil(o.AwsRegion) {
		var ret string
		return ret
	}
	return *o.AwsRegion
}

// GetAwsRegionOk returns a tuple with the AwsRegion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DockerAWSAuthDefinition) GetAwsRegionOk() (*string, bool) {
	if o == nil || IsNil(o.AwsRegion) {
		return nil, false
	}
	return o.AwsRegion, true
}

// HasAwsRegion returns a boolean if a field has been set.
func (o *DockerAWSAuthDefinition) HasAwsRegion() bool {
	if o != nil && !IsNil(o.AwsRegion) {
		return true
	}

	return false
}

// SetAwsRegion gets a reference to the given string and assigns it to the AwsRegion field.
func (o *DockerAWSAuthDefinition) SetAwsRegion(v string) {
	o.AwsRegion = &v
}

// GetAwsAccessKeyId returns the AwsAccessKeyId field value
func (o *DockerAWSAuthDefinition) GetAwsAccessKeyId() SecretStringDefinition {
	if o == nil {
		var ret SecretStringDefinition
		return ret
	}

	return o.AwsAccessKeyId
}

// GetAwsAccessKeyIdOk returns a tuple with the AwsAccessKeyId field value
// and a boolean to check if the value has been set.
func (o *DockerAWSAuthDefinition) GetAwsAccessKeyIdOk() (*SecretStringDefinition, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AwsAccessKeyId, true
}

// SetAwsAccessKeyId sets field value
func (o *DockerAWSAuthDefinition) SetAwsAccessKeyId(v SecretStringDefinition) {
	o.AwsAccessKeyId = v
}

// GetAwsSecretAccessKey returns the AwsSecretAccessKey field value
func (o *DockerAWSAuthDefinition) GetAwsSecretAccessKey() SecretStringDefinition {
	if o == nil {
		var ret SecretStringDefinition
		return ret
	}

	return o.AwsSecretAccessKey
}

// GetAwsSecretAccessKeyOk returns a tuple with the AwsSecretAccessKey field value
// and a boolean to check if the value has been set.
func (o *DockerAWSAuthDefinition) GetAwsSecretAccessKeyOk() (*SecretStringDefinition, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AwsSecretAccessKey, true
}

// SetAwsSecretAccessKey sets field value
func (o *DockerAWSAuthDefinition) SetAwsSecretAccessKey(v SecretStringDefinition) {
	o.AwsSecretAccessKey = v
}

func (o DockerAWSAuthDefinition) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o DockerAWSAuthDefinition) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.AwsRegion) {
		toSerialize["aws_region"] = o.AwsRegion
	}
	toSerialize["aws_access_key_id"] = o.AwsAccessKeyId
	toSerialize["aws_secret_access_key"] = o.AwsSecretAccessKey

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}

	return toSerialize, nil
}

func (o *DockerAWSAuthDefinition) UnmarshalJSON(bytes []byte) (err error) {
	varDockerAWSAuthDefinition := _DockerAWSAuthDefinition{}

	if err = json.Unmarshal(bytes, &varDockerAWSAuthDefinition); err == nil {
		*o = DockerAWSAuthDefinition(varDockerAWSAuthDefinition)
	}

	additionalProperties := make(map[string]interface{})

	if err = json.Unmarshal(bytes, &additionalProperties); err == nil {
		delete(additionalProperties, "aws_region")
		delete(additionalProperties, "aws_access_key_id")
		delete(additionalProperties, "aws_secret_access_key")
		o.AdditionalProperties = additionalProperties
	}

	return err
}

type NullableDockerAWSAuthDefinition struct {
	value *DockerAWSAuthDefinition
	isSet bool
}

func (v NullableDockerAWSAuthDefinition) Get() *DockerAWSAuthDefinition {
	return v.value
}

func (v *NullableDockerAWSAuthDefinition) Set(val *DockerAWSAuthDefinition) {
	v.value = val
	v.isSet = true
}

func (v NullableDockerAWSAuthDefinition) IsSet() bool {
	return v.isSet
}

func (v *NullableDockerAWSAuthDefinition) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableDockerAWSAuthDefinition(val *DockerAWSAuthDefinition) *NullableDockerAWSAuthDefinition {
	return &NullableDockerAWSAuthDefinition{value: val, isSet: true}
}

func (v NullableDockerAWSAuthDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableDockerAWSAuthDefinition) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


