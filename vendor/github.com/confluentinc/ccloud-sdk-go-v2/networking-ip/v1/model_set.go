package v1

import (
	"bytes"
	"encoding/json"
	"reflect"
)

/*
	Adding this type as an alternate to the following types when not available:
	- networking-ip/v1/model_multiple_search_filter.go
*/
// Set The connection types that may be used with the network.
type Set struct {
	Items []string
}

// NewSet instantiates a new Set object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSet() *Set {
	this := Set{}
	return &this
}

// NewSetWithDefaults instantiates a new Set object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSetWithDefaults() *Set {
	this := Set{}
	return &this
}

// Redact resets all sensitive fields to their zero value.
func (o *Set) Redact() {
}

func (o *Set) recurseRedact(v interface{}) {
	type redactor interface {
		Redact()
	}
	if r, ok := v.(redactor); ok {
		r.Redact()
	} else {
		val := reflect.ValueOf(v)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < val.Len(); i++ {
				// support data types declared without pointers
				o.recurseRedact(val.Index(i).Interface())
				// ... and data types that were declared without but need pointers (for Redact)
				if val.Index(i).CanAddr() {
					o.recurseRedact(val.Index(i).Addr().Interface())
				}
			}
		}
	}
}

func (o Set) zeroField(v interface{}) {
	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))
}

func (o Set) MarshalJSON() ([]byte, error) {
	toSerialize := make([]interface{}, len(o.Items))
	for i, item := range o.Items {
		toSerialize[i] = item
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(toSerialize)
	return buffer.Bytes(), err
}

func (o *Set) UnmarshalJSON(bytes []byte) (err error) {
	return json.Unmarshal(bytes, &o.Items)
}

type NullableSet struct {
	value *Set
	isSet bool
}

func (v NullableSet) Get() *Set {
	return v.value
}

func (v *NullableSet) Set(val *Set) {
	v.value = val
	v.isSet = true
}

func (v NullableSet) IsSet() bool {
	return v.isSet
}

func (v *NullableSet) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSet(val *Set) *NullableSet {
	return &NullableSet{value: val, isSet: true}
}

func (v NullableSet) MarshalJSON() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v.value)
	return buffer.Bytes(), err
}

func (v *NullableSet) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
