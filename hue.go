// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

// Package gohue controls hue lights.
package gohue

import (
  "bytes"
  "encoding/json"
  "errors"
  "fmt"
  "github.com/keep94/gohue/json_structs"
  "github.com/keep94/maybe"
  "io"
  "net/http"
  "net/url"
)

var (
  // Bright represents the brightest a light can be
  Bright = uint8(255)

  // Dim represents the dimmest a light can be.
  Dim = uint8(0)
)

var (
  // Indicates that the light ID is not found.
  NoSuchResourceError = errors.New("gohue: No such resource error.")

  // Indicates that some general error happened.
  GeneralError = errors.New("gohue: General error.")
)

var (
  Red = NewColor(0.675, 0.322)
  Green = NewColor(0.4077, 0.5154)
  Blue = NewColor(0.167, 0.04)
  Yellow = Red.Blend(Green, 0.5)
  Magenta = Blue.Blend(Red, 0.5)
  Cyan = Blue.Blend(Green, 0.5)
  Purple = NewColor(0.2522, 0.0882)
  White = NewColor(0.3848, 0.3629)
  Pink = NewColor(0.55, 0.3394)
  Orange = Red.Blend(Yellow, 0.5)
)

const (
  maxu16 = float64(10000.0)
)

// Color represents a particular color. Programs using Colors
// should typically store and pass them as values, not pointers.
type Color struct {
  x uint16
  y uint16
}

// NewColor returns a new Color. x and y are the coordinates of the color
// in the color XY space.
func NewColor(x, y float64) Color {
  return Color{x: uint16(x * maxu16 + 0.5), y: uint16(y * maxu16 + 0.5)}
}

func (c Color) String() string {
  return fmt.Sprintf("(%.4f, %.4f)", c.X(), c.Y())
}

// X returns the X value of this Color.
func (c Color) X() float64 {
  return float64(c.x) / maxu16
}

// Y returns the Y value of this Color.
func (c Color) Y() float64 {
  return float64(c.y) / maxu16
}

// Blend blends this color with another color returning the blended Color.
// ratio=0 means use only this color; ratio=1 means use only the other color.
func (c Color) Blend(other Color, ratio float64) Color {
  invratio := 1.0 - ratio
  return NewColor(
      c.X() * invratio + other.X() * ratio,
      c.Y() * invratio + other.Y() * ratio)
}

// MaybeColor instances represent a Color or nothing. The zero value is nothing.
type MaybeColor struct {
  Color
  // True if this instance represents a Color or false otherwise.
  Valid bool
}

// NewMaybecolor returns a new instance that represents c.
func NewMaybeColor(c Color) MaybeColor {
  return MaybeColor{Color: c, Valid: true}
}

// Set makes this instance represent c.
func (m *MaybeColor) Set(c Color) {
  m.Color = c
  m.Valid = true
}

// Clear makes this instance represent nothing.
func (m *MaybeColor) Clear() {
  m.Color = Color{}
  m.Valid = false
}

func (m MaybeColor) String() string {
  if (!m.Valid) {
    return "Nothing"
  }
  return fmt.Sprintf("Just %s", m.Color)
}

// LightProperies represents the properties of a light.
type LightProperties struct {
  // C is the Color. Nothing means leave color as-is.
  C MaybeColor

  // Bri is the brightness. nothing means leave brightness as-is.
  Bri maybe.Uint8
  
  // On is true if light is on or false if it is off. If nothing,
  // it means leave the on/off state as is.
  On maybe.Bool

  // The transition time in multiples of 100ms. See
  // http://developers.meethue.com. Used only with Context.Set().
  // Context.Get() does not populate
  TransitionTime maybe.Uint16
}

// Context represents a connection with a hue bridge.
type Context struct {

  // The userId / developer Id. See hue documentation.
  UserId string

  // The private ip address of the hue bridge.
  IpAddress string
}

// Set sets the properties of a light. lightId is the ID of the light to set.
// 0 means all lights.
// response is the raw response from the hue bridge or nil if communication
// failed. This function may return both a non-nil response and an error
// if the response from the hue bridge indicates an error. For most
// applications, it is enough just to look at err.
func (c *Context) Set(
    lightId int, properties *LightProperties) (response []byte, err error) {
  jsonMap := make(map[string]interface{})
  if properties.C.Valid {
    jsonMap["xy"] = []float64{
        properties.C.X(), properties.C.Y()}
  }
  if properties.Bri.Valid {
    jsonMap["bri"] = properties.Bri.Value
  }
  if properties.On.Valid {
    jsonMap["on"] = properties.On.Value
  }
  if properties.TransitionTime.Valid {
    jsonMap["transitiontime"] = properties.TransitionTime.Value
  }
  var reqBuffer []byte
  if reqBuffer, err = json.Marshal(jsonMap); err != nil {
    return
  }
  var url *url.URL
  if url, err = c.lightUrl(lightId); err != nil {
    return
  }
  request := &http.Request{
      Method: "PUT",
      URL: url,
      ContentLength: int64(len(reqBuffer)),
      Body: simpleReadCloser{bytes.NewReader(reqBuffer)},
  }
  var client http.Client
  var resp *http.Response
  if resp, err = client.Do(request); err != nil {
    return
  }
  defer resp.Body.Close()
  var respBuffer bytes.Buffer
  if _, err = respBuffer.ReadFrom(resp.Body); err != nil {
    return
  }
  response = respBuffer.Bytes()
  err = toError(response)
  return
}

// Get gets the properties of a light. lightId is the ID of the light.
// properties is the returned properties.
// response is the raw response from the hue bridge or nil if communication
// failed. This function may return both a non-nil response and an error
// if the response from the hue bridge indicates an error. For most
// applications, it is enough just to look at properties and err.
func (c *Context) Get(lightId int) (
    properties *LightProperties, response []byte, err error) {
  var url *url.URL
  if url, err = c.getLightUrl(lightId); err != nil {
    return
  }
  request := &http.Request{
      Method: "GET",
      URL: url,
  }
  var client http.Client
  var resp *http.Response
  if resp, err = client.Do(request); err != nil {
    return
  }
  defer resp.Body.Close()
  var respBuffer bytes.Buffer
  if _, err = respBuffer.ReadFrom(resp.Body); err != nil {
    return
  }
  response = respBuffer.Bytes()
  var jsonProps json_structs.LightState
  if err = json.Unmarshal(response, &jsonProps); err != nil {
    err = toError(response)
    return
  }
  if jsonProps.State != nil && len(jsonProps.State.XY) == 2 {
    state := jsonProps.State
    jsonColor := state.XY
    properties = &LightProperties{
        C: NewMaybeColor(NewColor(jsonColor[0], jsonColor[1])),
        Bri: maybe.NewUint8(state.Bri),
        On: maybe.NewBool(state.On)}
  } else {
    err = GeneralError
  }
  return
}

func (c *Context) getLightUrl(id int) (*url.URL, error) {
  return url.Parse(fmt.Sprintf("http://%s/api/%s/lights/%d", c.IpAddress, c.UserId, id))
}

func (c *Context) lightUrl(id int) (*url.URL, error) {
  if id == 0 {
    return c.allUrl()
  }
  return url.Parse(fmt.Sprintf("http://%s/api/%s/lights/%d/state", c.IpAddress, c.UserId, id))
}

func (c *Context) allUrl() (*url.URL, error) {
  return url.Parse(fmt.Sprintf("http://%s/api/%s/groups/0/action", c.IpAddress, c.UserId))
}

type simpleReadCloser struct {
  io.Reader
}

func (s simpleReadCloser) Close() error {
  return nil
}

func toError(rawResponse []byte) error {
  var response []json_structs.GeneralResponse
  if err := json.Unmarshal(rawResponse, &response); err != nil {
    return nil
  }
  if len(response) > 0 && response[0].Error != nil {
    if response[0].Error.ErrorId == 3 {
      return NoSuchResourceError
    }
    return GeneralError
  }
  return nil
}
