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

// checks if the DockerConfig type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &DockerConfig{}

// DockerConfig struct for DockerConfig
type DockerConfig struct {
	// The default Docker image to run the job steps in, if the job is of type Docker
	Image string `json:"image"`
	// Determines if/when the Docker image is pulled during job execution, if the job is of type Docker
	Pull string `json:"pull"`
	BasicAuth *DockerBasicAuth `json:"basic_auth,omitempty"`
	AwsAuth *DockerAWSAuth `json:"aws_auth,omitempty"`
	// Path to the shell to use to run build scripts with inside the container
	Shell *string `json:"shell,omitempty"`
	AdditionalProperties map[string]interface{}
}

type _DockerConfig DockerConfig

// NewDockerConfig instantiates a new DockerConfig object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewDockerConfig(image string, pull string) *DockerConfig {
	this := DockerConfig{}
	this.Image = image
	this.Pull = pull
	return &this
}

// NewDockerConfigWithDefaults instantiates a new DockerConfig object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewDockerConfigWithDefaults() *DockerConfig {
	this := DockerConfig{}
	return &this
}

// GetImage returns the Image field value
func (o *DockerConfig) GetImage() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Image
}

// GetImageOk returns a tuple with the Image field value
// and a boolean to check if the value has been set.
func (o *DockerConfig) GetImageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Image, true
}

// SetImage sets field value
func (o *DockerConfig) SetImage(v string) {
	o.Image = v
}

// GetPull returns the Pull field value
func (o *DockerConfig) GetPull() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Pull
}

// GetPullOk returns a tuple with the Pull field value
// and a boolean to check if the value has been set.
func (o *DockerConfig) GetPullOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Pull, true
}

// SetPull sets field value
func (o *DockerConfig) SetPull(v string) {
	o.Pull = v
}

// GetBasicAuth returns the BasicAuth field value if set, zero value otherwise.
func (o *DockerConfig) GetBasicAuth() DockerBasicAuth {
	if o == nil || IsNil(o.BasicAuth) {
		var ret DockerBasicAuth
		return ret
	}
	return *o.BasicAuth
}

// GetBasicAuthOk returns a tuple with the BasicAuth field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DockerConfig) GetBasicAuthOk() (*DockerBasicAuth, bool) {
	if o == nil || IsNil(o.BasicAuth) {
		return nil, false
	}
	return o.BasicAuth, true
}

// HasBasicAuth returns a boolean if a field has been set.
func (o *DockerConfig) HasBasicAuth() bool {
	if o != nil && !IsNil(o.BasicAuth) {
		return true
	}

	return false
}

// SetBasicAuth gets a reference to the given DockerBasicAuth and assigns it to the BasicAuth field.
func (o *DockerConfig) SetBasicAuth(v DockerBasicAuth) {
	o.BasicAuth = &v
}

// GetAwsAuth returns the AwsAuth field value if set, zero value otherwise.
func (o *DockerConfig) GetAwsAuth() DockerAWSAuth {
	if o == nil || IsNil(o.AwsAuth) {
		var ret DockerAWSAuth
		return ret
	}
	return *o.AwsAuth
}

// GetAwsAuthOk returns a tuple with the AwsAuth field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DockerConfig) GetAwsAuthOk() (*DockerAWSAuth, bool) {
	if o == nil || IsNil(o.AwsAuth) {
		return nil, false
	}
	return o.AwsAuth, true
}

// HasAwsAuth returns a boolean if a field has been set.
func (o *DockerConfig) HasAwsAuth() bool {
	if o != nil && !IsNil(o.AwsAuth) {
		return true
	}

	return false
}

// SetAwsAuth gets a reference to the given DockerAWSAuth and assigns it to the AwsAuth field.
func (o *DockerConfig) SetAwsAuth(v DockerAWSAuth) {
	o.AwsAuth = &v
}

// GetShell returns the Shell field value if set, zero value otherwise.
func (o *DockerConfig) GetShell() string {
	if o == nil || IsNil(o.Shell) {
		var ret string
		return ret
	}
	return *o.Shell
}

// GetShellOk returns a tuple with the Shell field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DockerConfig) GetShellOk() (*string, bool) {
	if o == nil || IsNil(o.Shell) {
		return nil, false
	}
	return o.Shell, true
}

// HasShell returns a boolean if a field has been set.
func (o *DockerConfig) HasShell() bool {
	if o != nil && !IsNil(o.Shell) {
		return true
	}

	return false
}

// SetShell gets a reference to the given string and assigns it to the Shell field.
func (o *DockerConfig) SetShell(v string) {
	o.Shell = &v
}

func (o DockerConfig) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o DockerConfig) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["image"] = o.Image
	toSerialize["pull"] = o.Pull
	if !IsNil(o.BasicAuth) {
		toSerialize["basic_auth"] = o.BasicAuth
	}
	if !IsNil(o.AwsAuth) {
		toSerialize["aws_auth"] = o.AwsAuth
	}
	if !IsNil(o.Shell) {
		toSerialize["shell"] = o.Shell
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}

	return toSerialize, nil
}

func (o *DockerConfig) UnmarshalJSON(bytes []byte) (err error) {
	varDockerConfig := _DockerConfig{}

	if err = json.Unmarshal(bytes, &varDockerConfig); err == nil {
		*o = DockerConfig(varDockerConfig)
	}

	additionalProperties := make(map[string]interface{})

	if err = json.Unmarshal(bytes, &additionalProperties); err == nil {
		delete(additionalProperties, "image")
		delete(additionalProperties, "pull")
		delete(additionalProperties, "basic_auth")
		delete(additionalProperties, "aws_auth")
		delete(additionalProperties, "shell")
		o.AdditionalProperties = additionalProperties
	}

	return err
}

type NullableDockerConfig struct {
	value *DockerConfig
	isSet bool
}

func (v NullableDockerConfig) Get() *DockerConfig {
	return v.value
}

func (v *NullableDockerConfig) Set(val *DockerConfig) {
	v.value = val
	v.isSet = true
}

func (v NullableDockerConfig) IsSet() bool {
	return v.isSet
}

func (v *NullableDockerConfig) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableDockerConfig(val *DockerConfig) *NullableDockerConfig {
	return &NullableDockerConfig{value: val, isSet: true}
}

func (v NullableDockerConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableDockerConfig) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

