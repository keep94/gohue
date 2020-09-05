// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

// Package json_structs contains structures used internally for parsing json.
// These should not be used directly.
package json_structs

type LightState struct {
	State *LightProperties
}

type LightProperties struct {
	On  bool
	Bri uint8
	XY  []float64
}

type GeneralResponse struct {
	Error *SingleError
}

type SingleError struct {
	ErrorId     int `json:"type"`
	Address     string
	Description string
}
