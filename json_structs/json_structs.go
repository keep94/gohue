package json_structs

type LightState struct {
  State *LightProperties
}

type LightProperties struct {
  On bool
  Bri uint8
  XY []float64
}