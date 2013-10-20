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
  "github.com/keep94/gohue/json_structs"
  "github.com/keep94/tasks"
  "io"
  "net/http"
  "net/url"
  "time"
)

var (
  TrueVal = true
  FalseVal = false

  // Bright represents the brightest a light can be
  Bright = uint8(255)

  // Dim represents the dimmest a light can be.
  Dim = uint8(0)
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

// ColorPtr returns a pointer to the given color.
func ColorPtr(c Color) *Color {
  return &c
}

// Uint8Ptr returns a pointer to the given uint8
func Uint8Ptr(u uint8) *uint8 {
  return &u
}

// BoolPtr returns a pointer to the given bool
func BoolPtr(x bool) *bool {
  return &x
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

// LightProperies represents the properties of a light.
type LightProperties struct {
  // C is the Color. nil means leave the color as-is
  C *Color

  // Bri is the brightness. nil means leave brightness as-is.
  Bri *uint8
  
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
    jsonMap["xy"] = []float64{properties.C.X(), properties.C.Y()}
  }
  if properties.Bri != nil {
    jsonMap["bri"] = *properties.Bri
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

// Get gets the properties of a light. lightId is the ID of the light.
func (c *Context) Get(lightId int) (properties *LightProperties, err error) {
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
  jsonDecoder := json.NewDecoder(resp.Body)
  var jsonProps json_structs.LightState
  if err = jsonDecoder.Decode(&jsonProps); err != nil {
    return
  }
  if jsonProps.State != nil && len(jsonProps.State.XY) == 2 {
    state := jsonProps.State
    jsonColor := state.XY
    properties = &LightProperties{
        C: ColorPtr(NewColor(jsonColor[0], jsonColor[1])),
        Bri: Uint8Ptr(state.Bri),
        On: BoolPtr(state.On)}
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

  // The color the light should be. nil menas color should be unchanged.
  C *Color

  // The brightness the light should be. nil means brightness should be
  // unchanged.
  Bri *uint8

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
  C *Color

  // The brightness.
  Bri *uint8

  // If true, light(s) are turned on. May be used along with G field
  // to ensure light(s) are on.
  On bool

  // If true, light(s) are turned off.
  Off bool

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
  if a.C != nil || a.Bri != nil || a.On || a.Off {
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
    properties.On = &TrueVal
  } else if a.Off {
    properties.On = &FalseVal
  }
  properties.C = a.C
  properties.Bri = a.Bri
  multiSet(e, setter, lights, &properties)
}

func (a *Action) doGradient(setter Setter, lights []int, e *tasks.Execution) {
  startTime := e.Now()
  var currentD time.Duration
  var properties LightProperties
  if a.On {
    properties.On = &TrueVal
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
    properties.On = nil
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
    if _, err := setter.Set(0, properties); err != nil {
      e.SetError(err)
      return
    }
  } else {
    for _, light := range lights {
      if _, err := setter.Set(light, properties); err != nil {
        e.SetError(err)
        return
      }
    }
  }
}

func maybeBlendColor(first, second *Color, ratio float64) *Color {
  if first != nil && second != nil {
    return ColorPtr(first.Blend(*second, ratio))
  }
  return first
}

func maybeBlendBrightness(first, second *uint8, ratio float64) *uint8 {
  if first != nil && second != nil {
    return Uint8Ptr(uint8((1.0 - ratio) * float64(*first) + ratio * float64(*second) + 0.5))
  }
  return first
}

