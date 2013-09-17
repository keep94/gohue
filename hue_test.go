// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

package gohue_test

import (
  "errors"
  "github.com/keep94/gohue"
  "github.com/keep94/tasks"
  "reflect"
  "testing"
  "time"
)

var (
  kNow = time.Date(2013, 9, 15, 14, 0, 0, 0, time.Local)
  kSomeError = errors.New("gohue: someError.")
)

func TestGradient(t *testing.T) {
  action := gohue.Action{
      Lights: []int{2},
      G: &gohue.Gradient {
          Cds: []gohue.ColorDuration{
              {C: gohue.NewColor(0.2, 0.1, 0), D: 0},
              {C: gohue.NewColor(0.3, 0.3, 30), D: 1000},
              {C: gohue.NewColor(0.9, 0.9, 100), D: 1000},
              {C: gohue.NewColor(0.8, 0.7, 100), D: 1000},
              {C: gohue.NewColor(0.2, 0.4, 10), D: 1750},
              {C: gohue.NewColor(0.29, 0.46, 22), D: 2500}},
          Refresh: 500},
      On: true}
  expected := []request {
      {L: 2, C: gohue.NewColor(0.2, 0.1, 0), Cset: true, On: true, Onset: true, D: 0},
      {L: 2, C: gohue.NewColor(0.25, 0.2, 15), Cset: true, D: 500},
      {L: 2, C: gohue.NewColor(0.3, 0.3, 30), Cset: true, D: 1000},
      {L: 2, C: gohue.NewColor(0.4, 0.5, 40), Cset: true, D: 1500},
      {L: 2, C: gohue.NewColor(0.23, 0.42, 14), Cset: true, D:2000},
      {L: 2, C: gohue.NewColor(0.29, 0.46, 22), Cset: true, D:2500}}
  verifyAction(t, expected, action)
}

func TestGradient2(t *testing.T) {
  action := gohue.Action{
      G: &gohue.Gradient{
          Cds: []gohue.ColorDuration{
              {C: gohue.NewColor(0.2, 0.1, 0), D: 0},
              {C: gohue.NewColor(0.3, 0.3, 30), D: 1000}},
          Refresh: 600}}
  expected := []request {
      {L: 0, C: gohue.NewColor(0.2, 0.1, 0), Cset: true, D: 0},
      {L: 0, C: gohue.NewColor(0.26, 0.22, 18), Cset: true, D: 600},
      {L: 0, C: gohue.NewColor(0.3, 0.3, 30), Cset: true, D: 1200}}
  verifyAction(t, expected, action)
}

func TestOn(t *testing.T) {
  action := gohue.Action{On: true, C: gohue.NewColorPtr(0.4, 0.2, 80)}
  expected := []request {{L: 0, C: gohue.NewColor(0.4, 0.2, 80), Cset: true, On: true, Onset: true, D: 0}}
  verifyAction(t, expected, action)
}

func TestOff(t *testing.T) {
  action := gohue.Action{On: true}
  expected := []request {{L: 0, On: true, Onset: true, D: 0}}
  verifyAction(t, expected, action)
}

func TestSeries(t *testing.T) {
  action := gohue.Action{
      Series: []*gohue.Action {
          {Lights: []int{2, 3}, On: true},
          {Sleep: 3000},
          {Off: true}}}
  expected := []request {
      {L: 2, On: true, Onset: true,  D: 0},
      {L: 3, On: true, Onset: true,  D: 0},
      {L: 0, On: false, Onset: true, D: 3000}}
  verifyAction(t, expected, action)
}

func TestError(t *testing.T) {
  action := gohue.Action{
      Series: []*gohue.Action {
          {Lights: []int{2, 3}, On: true},
          {Sleep: 3000},
          {Off: true}}}
  expected := []request {
      {L: 2, On: true, Onset: true,  D: 0}}
  clock := &tasks.ClockForTesting{kNow}
  context := &setterForTesting{err: kSomeError, clock: clock, now: kNow}
  tasks.RunForTesting(action.AsTask(context, nil), clock)
  if !reflect.DeepEqual(expected, context.requests) {
    t.Errorf("Expected %v, got %v", expected, context.requests)
  }
}
  

func TestEachSunset(t *testing.T) {
  now := time.Date(2013, 1, 7, 0, 0, 0, 0, time.Local)
  r := &gohue.EachSunset{Lat: 40.0, Lon: -120.0}
  stream := r.ForTime(now)
  var atime time.Time
  stream.Next(&atime)
  verifyTime(t, time.Date(2013, 1, 7, 16, 51, 59, 0, time.Local), atime)
  stream.Next(&atime)
  verifyTime(t, time.Date(2013, 1, 8, 16, 52, 57, 0, time.Local), atime)

  r = &gohue.EachSunset{Lat: 40.0, Lon: -120.0, HourCap: 16, MinuteCap: 52}
  stream = r.ForTime(now)
  stream.Next(&atime)
  verifyTime(t, time.Date(2013, 1, 7, 16, 51, 59, 0, time.Local), atime)
  stream.Next(&atime)
  verifyTime(t, time.Date(2013, 1, 8, 16, 52, 0, 0, time.Local), atime)

  r = &gohue.EachSunset{Lat: 40.0, Lon: -120.0, HourCap: 16, MinuteCap: 51}
  stream = r.ForTime(time.Date(2013, 1, 7, 16, 51, 0, 0, time.Local))
  stream.Next(&atime)
  verifyTime(t, time.Date(2013, 1, 8, 16, 51, 0, 0, time.Local), atime)
  stream.Next(&atime)
  verifyTime(t, time.Date(2013, 1, 9, 16, 51, 0, 0, time.Local), atime)
}

type request struct {
  L int
  C gohue.Color
  Cset bool
  On bool
  Onset bool
  D time.Duration
}

type setterForTesting struct {
  err error
  clock *tasks.ClockForTesting
  now time.Time
  requests []request
}

func (s *setterForTesting) Set(lightId int, p *gohue.LightProperties) (result []byte, err error) {
  var r request
  r.L = lightId
  if p.C != nil {
    r.C = *p.C
    r.Cset = true
  }
  if p.On != nil {
    r.On = *p.On
    r.Onset = true
  }
  r.D = s.clock.Current.Sub(s.now)
  s.requests = append(s.requests, r)
  if s.err != nil {
    err = s.err
  }
  return
}

func verifyTime(t *testing.T, expected, actual time.Time) {
  if expected != actual {
    t.Errorf("Expected %v, got %v", expected, actual)
  }
}

func verifyAction(t *testing.T, expected []request, action gohue.Action) {
  clock := &tasks.ClockForTesting{kNow}
  context := &setterForTesting{clock: clock, now: kNow}
  tasks.RunForTesting(action.AsTask(context, nil), clock)
  if !reflect.DeepEqual(expected, context.requests) {
    t.Errorf("Expected %v, got %v", expected, context.requests)
  }
}
