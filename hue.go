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
  "github.com/keep94/tasks"
  "io"
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

// ColorDuration specifies the color a light should be a certain duration
// into a gradient.
type ColorDuration struct {

  // The color the light should be. nothing menas color should be unchanged.
  C MaybeColor

  // The brightness the light should be. nothing means brightness should be
  // unchanged.
  Bri maybe.Uint8

  // The Duration into the gradient.
  D time.Duration
}

// Interface Setter sets the properties of a light. lightId is the ID of the
// light to set. 0 means all lights.
type Setter interface {
  Set(lightId int, properties *LightProperties) (response []byte, err error)
}

// Gradient represents a change in colors over time.
type Gradient struct {

  // The desired color at certain durations into the gradient. The specified
  // durations must be in increasing order.
  Cds []ColorDuration

  // Light color is refreshed this often.
  Refresh time.Duration
}

// Action represents some action to the lights.
// Callers should set exactly one of the
// Parallel, Series, G, any subset of {C, Bri, On, Off}, or Sleep fields.
// The one exception is that On can be used with G. The other
// fields compliment these fields.
type Action struct {

  // The light bulb ids. Empty means the default set of lights. For child
  // actions, the default set of lights is the parent's set of lights;
  // for top-level actions, the default set of lights are the light bulb
  // ids passed to AsTask.
  Lights []int

  // Repeat this many times. 0 or negative means do once.
  Repeat int

  // The Gradient
  G *Gradient

  // The single color
  C MaybeColor

  // The brightness.
  Bri maybe.Uint8

  // If true, light(s) are turned on. May be used along with G field
  // to ensure light(s) are on.
  On bool

  // If true, light(s) are turned off.
  Off bool

  // Transition time in multiples of 100ms.
  // See http://developers.meethue.com. Right now it
  // only works with the {C, Bri, On, Off} fields
  TransitionTime maybe.Uint16

  // Sleep sleeps this duration
  Sleep time.Duration

  // Actions to be done in series
  Series []*Action

  // Actions to be done in parallel
  Parallel []*Action
}

// AsTask returns a Task from this instance. setter is what changes the
// lightbulb. lights is the default set of lights empty means all lights.
func (a *Action) AsTask(setter Setter, lights []int) tasks.Task {
  if a.Repeat < 2 {
    return a.asTask(setter, lights)
  }
  return tasks.RepeatingTask(a.asTask(setter, lights), a.Repeat)
}

func (a *Action) asTask(setter Setter, lights []int) tasks.Task {
  if len(a.Lights) > 0 {
    lights = a.Lights
  }
  if len(a.Parallel) > 0 {
    parallelTasks := make([]tasks.Task, len(a.Parallel))
    for i := range parallelTasks {
      parallelTasks[i] = a.Parallel[i].AsTask(setter, lights)
    }
    return tasks.ParallelTasks(parallelTasks...)
  }
  if len(a.Series) > 0 {
    seriesTasks := make([]tasks.Task, len(a.Series))
    for i := range seriesTasks {
      seriesTasks[i] = a.Series[i].AsTask(setter, lights)
    }
    return tasks.SeriesTasks(seriesTasks...)
  }
  if a.G != nil {
    if len(a.G.Cds) == 0 || a.G.Cds[0].D != 0 {
      panic("D of first ColorDuration element must be 0.")
    }
    return tasks.TaskFunc(func(e *tasks.Execution) {
      a.doGradient(setter, lights, e)
    })
  }
  if a.C.Valid || a.Bri.Valid || a.On || a.Off {
    return tasks.TaskFunc(func(e *tasks.Execution) {
      a.doOnOff(setter, lights, e)
    })
  }
  return tasks.TaskFunc(func(e *tasks.Execution) {
    e.Sleep(a.Sleep)
  })
}

func (a *Action) doOnOff(setter Setter, lights []int, e *tasks.Execution) {
  var properties LightProperties
  if a.On {
    properties.On.Set(true)
  } else if a.Off {
    properties.On.Set(false)
  }
  properties.C = a.C
  properties.Bri = a.Bri
  properties.TransitionTime = a.TransitionTime
  multiSet(e, setter, lights, &properties)
}

func (a *Action) doGradient(setter Setter, lights []int, e *tasks.Execution) {
  startTime := e.Now()
  var currentD time.Duration
  var properties LightProperties
  if a.On {
    properties.On.Set(true)
  }
  idx := 1
  last := &a.G.Cds[len(a.G.Cds) - 1]
  for idx < len(a.G.Cds) {
    if currentD >= a.G.Cds[idx].D {
      idx++
      continue
    }
    first := &a.G.Cds[idx - 1]
    second := &a.G.Cds[idx]
    ratio := float64(currentD - first.D) / float64(second.D - first.D)
    acolor := maybeBlendColor(first.C, second.C, ratio)
    aBrightness := maybeBlendBrightness(first.Bri, second.Bri, ratio)
    properties.C = acolor
    properties.Bri = aBrightness
    multiSet(e, setter, lights, &properties)
    properties.On.Clear()
    if e.Error() != nil {
      return
    }
    if !e.Sleep(a.G.Refresh) {
      return
    }
    currentD = e.Now().Sub(startTime) 
  }
  properties.C = last.C
  properties.Bri = last.Bri
  multiSet(e, setter, lights, &properties)
}

type simpleReadCloser struct {
  io.Reader
}

func (s simpleReadCloser) Close() error {
  return nil
}

func multiSet(
    e *tasks.Execution,
    setter Setter,
    lights []int,
    properties *LightProperties) {
  if len(lights) == 0 {
    if resp, err := setter.Set(0, properties); err != nil {
      e.SetError(fixError(resp, err))
      return
    }
  } else {
    for _, light := range lights {
      if resp, err := setter.Set(light, properties); err != nil {
        e.SetError(fixError(resp, err))
        return
      }
    }
  }
}

func fixError(rawResponse []byte, err error) error {
  if err == GeneralError {
    return errors.New(string(rawResponse))
  }
  return err
}

func maybeBlendColor(first, second MaybeColor, ratio float64) MaybeColor {
  if first.Valid && second.Valid {
    return NewMaybeColor(first.Blend(second.Color, ratio))
  }
  return first
}

func maybeBlendBrightness(
    first, second maybe.Uint8, ratio float64) maybe.Uint8 {
  if first.Valid && second.Valid {
    return maybe.NewUint8(
        uint8((1.0 - ratio) * float64(first.Value) + ratio * float64(second.Value) + 0.5))
  }
  return first
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
