// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

// Package gohue controls hue lights.
package gohue

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "net/url"
)

const (
  // Bright represents the brightest setting
  Bright = uint8(255)

  // Dim represents the dimmest setting.
  Dim = uint8(0)
)

var (
  // Pointer to true
  TruePtr = &trueVal

  // Pointer to false
  FalsePtr = &falseVal
)

var (
  trueVal = true
  falseVal = false
)  

const (
  maxu16 = float64(10000.0)
)

// Color represents a particular color and brightness. Programs using Colors
// should typically store and pass them as values, not pointers.
type Color struct {
  x uint16
  y uint16
  bri uint8
}

// NewColor returns a new Color. x and y are the coordinates of the color
// in the color XY space; brightness is the brightness where 255 is brightest
// and 0 is dimmest.
func NewColor(x, y float64, brightness uint8) Color {
  return Color{x: uint16(x * maxu16), y: uint16(y * maxu16), bri: brightness}
}

// X returns the X value of this Color.
func (c Color) X() float64 {
  return float64(c.x) / maxu16
}

// Y returns the Y value of this Color.
func (c Color) Y() float64 {
  return float64(c.y) / maxu16
}

// Brightness returns the brightness of this color.
func (c Color) Brightness() uint8 {
  return c.bri
}

// Blend blends this color with another color returning the blended Color.
// ratio=0 means use only this color; ratio=1 means use only the other color.
func (c Color) Blend(other Color, ratio float64) Color {
  invratio := 1.0 - ratio
  return NewColor(
      c.X() * invratio + other.X() * ratio,
      c.Y() * invratio + other.Y() * ratio,
      uint8(float64(c.bri) * invratio + float64(other.bri) * ratio))
}

// LightProperies represents the properties of a light.
type LightProperties struct {
  // C is the Color. nil means leave the color as-is
  C *Color

  // On is true if light is on or false if it is off. If nil,
  // it means leave the on/off state as is.
  On *bool
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
func (c *Context) Set(lightId int, properties *LightProperties) (response []byte, err error) {
  jsonMap := make(map[string]interface{})
  if properties.C != nil {
    jsonMap["bri"] = properties.C.Brightness()
    jsonMap["xy"] = []float64{properties.C.X(), properties.C.Y()}
  }
  if properties.On != nil {
    jsonMap["on"] = *properties.On
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
  return respBuffer.Bytes(), nil
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

