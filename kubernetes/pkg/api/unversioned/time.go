/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package unversioned

import (
	"encoding/json"
	"time"

	"github.com/openshift/kubernetes/pkg/genericapiserver/openapi/common"

	"github.com/openshift/github.com/go-openapi/spec"
	"github.com/openshift/github.com/google/gofuzz"
)

// Time is a wrapper around time.Time which supports correct
// marshaling to YAML and JSON.  Wrappers are provided for many
// of the factory methods that the time package offers.
//
// +protobuf.options.marshal=false
// +protobuf.as=Timestamp
// +protobuf.options.(gogoproto.goproto_stringer)=false
type Time struct {
	time.Time `protobuf:"-"`
}

// DeepCopy returns a deep-copy of the Time value.  The underlying time.Time
// type is effectively immutable in the time API, so it is safe to
// copy-by-assign, despite the presence of (unexported) Pointer fields.
func (t Time) DeepCopy() Time {
	return t
}

// String returns the representation of the time.
func (t Time) String() string {
	return t.Time.String()
}

// NewTime returns a wrapped instance of the provided time
func NewTime(time time.Time) Time {
	return Time{time}
}

// Date returns the Time corresponding to the supplied parameters
// by wrapping time.Date.
func Date(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) Time {
	return Time{time.Date(year, month, day, hour, min, sec, nsec, loc)}
}

// Now returns the current local time.
func Now() Time {
	return Time{time.Now()}
}

// IsZero returns true if the value is nil or time is zero.
func (t *Time) IsZero() bool {
	if t == nil {
		return true
	}
	return t.Time.IsZero()
}

// Before reports whether the time instant t is before u.
func (t Time) Before(u Time) bool {
	return t.Time.Before(u.Time)
}

// Equal reports whether the time instant t is equal to u.
func (t Time) Equal(u Time) bool {
	return t.Time.Equal(u.Time)
}

// Unix returns the local time corresponding to the given Unix time
// by wrapping time.Unix.
func Unix(sec int64, nsec int64) Time {
	return Time{time.Unix(sec, nsec)}
}

// Rfc3339Copy returns a copy of the Time at second-level precision.
func (t Time) Rfc3339Copy() Time {
	copied, _ := time.Parse(time.RFC3339, t.Format(time.RFC3339))
	return Time{copied}
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (t *Time) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}

	var str string
	json.Unmarshal(b, &str)

	pt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	t.Time = pt.Local()
	return nil
}

// UnmarshalQueryParameter converts from a URL query parameter value to an object
func (t *Time) UnmarshalQueryParameter(str string) error {
	if len(str) == 0 {
		t.Time = time.Time{}
		return nil
	}
	// Tolerate requests from older clients that used JSON serialization to build query params
	if len(str) == 4 && str == "null" {
		t.Time = time.Time{}
		return nil
	}

	pt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	t.Time = pt.Local()
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		// Encode unset/nil objects as JSON's "null".
		return []byte("null"), nil
	}

	return json.Marshal(t.UTC().Format(time.RFC3339))
}

func (_ Time) OpenAPIDefinition() common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:   []string{"string"},
				Format: "date-time",
			},
		},
	}
}

// MarshalQueryParameter converts to a URL query parameter value
func (t Time) MarshalQueryParameter() (string, error) {
	if t.IsZero() {
		// Encode unset/nil objects as an empty string
		return "", nil
	}

	return t.UTC().Format(time.RFC3339), nil
}

// Fuzz satisfies fuzz.Interface.
func (t *Time) Fuzz(c fuzz.Continue) {
	if t == nil {
		return
	}
	// Allow for about 1000 years of randomness.  Leave off nanoseconds
	// because JSON doesn't represent them so they can't round-trip
	// properly.
	t.Time = time.Unix(c.Rand.Int63n(1000*365*24*60*60), 0)
}

var _ fuzz.Interface = &Time{}
