// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

// Package actions controls hue lights via asynchronous tasks.
package actions

import (
  "errors"
  "github.com/keep94/gohue"
  "github.com/keep94/maybe"
  "github.com/keep94/tasks"
  "time"
)

var (
  kInvalidLightIdBytes = ([]byte)("Invalid light id")
)

// NoSuchLightIdError is the error that Task instances created from Action
// instances report when a light ID is unknown.
type NoSuchLightIdError struct {

  // The Unknown light ID
  LightId int

  // The raw response received from the hue bridge
  RawResponse []byte
}

func (e *NoSuchLightIdError) Error() string {
  return string(e.RawResponse)
}

// ColorDuration specifies the color and/or brightness a light should have a
// certain duration into a gradient.
type ColorDuration struct {

  // The color the light should be. nothing menas color should be unchanged.
  C gohue.MaybeColor

  // The brightness the light should be. nothing means brightness should be
  // unchanged.
  Bri maybe.Uint8

  // The Duration into the gradient.
  D time.Duration
}

// Interface Setter sets the properties of a light. lightId is the ID of the
// light to set. 0 means all lights.
type Setter interface {
  Set(lightId int, properties *gohue.LightProperties) (response []byte, err error)
}

// Gradient represents a change in colors and/or brightness over time.
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
  C gohue.MaybeColor

  // The brightness.
  Bri maybe.Uint8

  // If true, light(s) are turned on. May be used along with G field
  // to ensure light(s) are on.
  On bool

  // If true, light(s) are turned off.
  Off bool

  // Transition time in multiples of 100ms. Nothing means default transition
  // time. See http://developers.meethue.com. Right now it
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
  var properties gohue.LightProperties
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
  var properties gohue.LightProperties
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

func multiSet(
    e *tasks.Execution,
    setter Setter,
    lights []int,
    properties *gohue.LightProperties) {
  if len(lights) == 0 {
    if resp, err := setter.Set(0, properties); err != nil {
      e.SetError(fixError(0, resp, err))
      return
    }
  } else {
    for _, light := range lights {
      if light == 0 {
        e.SetError(fixError(0, kInvalidLightIdBytes, gohue.NoSuchResourceError))
        return
      }
      if resp, err := setter.Set(light, properties); err != nil {
        e.SetError(fixError(light, resp, err))
        return
      }
    }
  }
}

func fixError(lightId int, rawResponse []byte, err error) error {
  if err == gohue.NoSuchResourceError {
    return &NoSuchLightIdError{LightId: lightId, RawResponse: rawResponse}
  }
  return errors.New(string(rawResponse))
}

func maybeBlendColor(first, second gohue.MaybeColor, ratio float64) gohue.MaybeColor {
  if first.Valid && second.Valid {
    return gohue.NewMaybeColor(first.Blend(second.Color, ratio))
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


