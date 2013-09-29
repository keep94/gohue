// Package json_structs contains structures used internally for parsing json.
// These should not be used directly.
package json_structs

type LightState struct {
  State *LightProperties
}

type LightProperties struct {
  On bool
  Bri uint8
  XY []float64
}
