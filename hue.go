// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

// Package gohue controls hue lights. See http://developers.meethue.com.
package gohue

import (
  "bytes"
  "encoding/json"
  "errors"
  "fmt"
  "github.com/keep94/gohue/json_structs"
  "github.com/keep94/maybe"
  "io"
  "net"
  "net/http"
  "net/url"
  "time"
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

var (
  kDefaultOptions = &Options{}
)

// Color represents a particular color. Programs using Colors
// should typically store and pass them as values, not pointers.
type Color struct {
  x uint16
  y uint16
}

// NewColor returns a new Color. x and y are the coordinates of the color
// in the color XY space. x and y are between 0.0 and 1.0 inclusive.
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

// NewMaybeColor returns a new instance that represents c.
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

  // Bri is the brightness. Nothing means leave brightness as is.
  Bri maybe.Uint8
  
  // On is true if light is on or false if it is off. Nothing
  // means leave the on/off state as is.
  On maybe.Bool

  // The transition time in multiples of 100ms. Nothing means the default
  // transition time. See http://developers.meethue.com.
  // Used only with Context.Set(). Context.Get() does not populate.
  TransitionTime maybe.Uint16
}

// Context represents a connection with a hue bridge.
type Context struct {
  ipAddress string
  userId string
  allUrl *url.URL
  client *http.Client
}

// Options contains optional settings for Context instance creation.
type Options struct {
  // Operations that take longer than this will fail with an error.
  // Zero or negative values means no timeout specified.
  Timeout time.Duration
}

// NewContext creates a new Context instance. ipAddress is the private ip
// address of the hue bridge, but could be a DNS name.
// userId is the user Id / developer Id (See hue documentation).
func NewContext(ipAddress, userId string) *Context {
  return NewContextWithOptions(ipAddress, userId, nil)
}

// NewContextWithOptions creates a new Context instance.
// ipAddress is the private ip address of the hue bridge, but could be a
// DNS name.
// userId is the user Id / developer Id (See hue documentation).
// options contains optional settings for the created context.
func NewContextWithOptions(
    ipAddress, userId string, options *Options) *Context {
  if options == nil {
    options = kDefaultOptions
  }
  allUrl := &url.URL{
      Scheme: "http",
      Host: ipAddress,
      Path: fmt.Sprintf("/api/%s/groups/0/action", userId),
  }
  var client http.Client
  if options.Timeout > 0 {
    client.Transport = &http.Transport{Dial: timeoutDialer(options.Timeout)}
  }
  return &Context{
      ipAddress: ipAddress,
      userId: userId,
      allUrl: allUrl,
      client: &client}
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
  request := &http.Request{
      Method: "PUT",
      URL: c.lightUrl(lightId),
      ContentLength: int64(len(reqBuffer)),
      Body: simpleReadCloser{bytes.NewReader(reqBuffer)},
  }
  client := c.client
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
  request := &http.Request{
      Method: "GET",
      URL: c.getLightUrl(lightId),
  }
  client := c.client
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

func (c *Context) getLightUrl(id int) *url.URL {
  return &url.URL{
      Scheme: "http",
      Host: c.ipAddress,
      Path: fmt.Sprintf("/api/%s/lights/%d", c.userId, id),
  }
}

func (c *Context) lightUrl(id int) *url.URL {
  if id == 0 {
    return c.allUrl
  }
  return &url.URL{
      Scheme: "http",
      Host: c.ipAddress,
      Path: fmt.Sprintf("/api/%s/lights/%d/state", c.userId, id),
  }
}

type simpleReadCloser struct {
  io.Reader
}

func (s simpleReadCloser) Close() error {
  return nil
}

func timeoutDialer(
    timeout time.Duration) func(net, addr string) (net.Conn, error) {
  return func(netw, addr string) (net.Conn, error) {
    deadline := time.Now().Add(timeout)
    conn, err := net.DialTimeout(netw, addr, timeout)
    if err != nil {
      return nil, err
    }
    conn.SetDeadline(deadline)
    return conn, nil
  }
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
