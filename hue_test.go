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
              {C: gohue.ColorPtr(gohue.NewColor(0.2, 0.1)),
               Bri: gohue.Uint8Ptr(0), D: 0},
              {C: gohue.ColorPtr(gohue.NewColor(0.3, 0.3)),
               Bri: gohue.Uint8Ptr(30), D: 1000},
              {C: gohue.ColorPtr(gohue.NewColor(0.9, 0.9)),
               Bri: gohue.Uint8Ptr(100), D: 1000},
              {C: gohue.ColorPtr(gohue.NewColor(0.8, 0.7)),
              Bri: gohue.Uint8Ptr(100),  D: 1000},
              {C: gohue.ColorPtr(gohue.NewColor(0.2, 0.4)),
               Bri: gohue.Uint8Ptr(10), D: 1750},
              {C: gohue.ColorPtr(gohue.NewColor(0.29, 0.46)),
               Bri: gohue.Uint8Ptr(22), D: 2500}},
          Refresh: 500},
      On: true}
  expected := []request {
      {L: 2, C: gohue.NewColor(0.2, 0.1), Cset: true, Briset: true, On: true, Onset: true, D: 0},
      {L: 2, C: gohue.NewColor(0.25, 0.2), Cset: true, Bri: 15, Briset: true, D: 500},
      {L: 2, C: gohue.NewColor(0.8, 0.7), Cset: true, Bri: 100, Briset: true, D: 1000},
      {L: 2, C: gohue.NewColor(0.4, 0.5), Cset: true, Bri: 40, Briset: true, D: 1500},
      {L: 2, C: gohue.NewColor(0.23, 0.42), Cset: true, Bri: 14, Briset: true, D:2000},
      {L: 2, C: gohue.NewColor(0.29, 0.46), Cset: true, Bri: 22, Briset: true, D:2500}}
  verifyAction(t, expected, action)
}

func TestGradient2(t *testing.T) {
  action := gohue.Action{
      G: &gohue.Gradient{
          Cds: []gohue.ColorDuration{
              {C: gohue.ColorPtr(gohue.NewColor(0.2, 0.1)), D: 0},
              {C: gohue.ColorPtr(gohue.NewColor(0.3, 0.3)), D: 1000}},
          Refresh: 600}}
  expected := []request {
      {L: 0, C: gohue.NewColor(0.2, 0.1), Cset: true, D: 0},
      {L: 0, C: gohue.NewColor(0.26, 0.22), Cset: true, D: 600},
      {L: 0, C: gohue.NewColor(0.3, 0.3), Cset: true, D: 1200}}
  verifyAction(t, expected, action)
}

func TestGradient3(t *testing.T) {
  action := gohue.Action{
      G: &gohue.Gradient{
          Cds: []gohue.ColorDuration{
              {Bri: &gohue.Bright, D: 0},
              {Bri: &gohue.Bright, C: &gohue.Red, D: 1000},
              {C: &gohue.Red, D: 2000},
              {Bri: &gohue.Dim, D: 3000},
              {Bri: &gohue.Dim, D: 4000}},
          Refresh: 500}}
  expected := []request {
      {L: 0, Bri: gohue.Bright, Briset: true, D: 0},
      {L: 0, Bri: gohue.Bright, Briset: true, D: 500},
      {L: 0, Bri: gohue.Bright, Briset: true, C: gohue.Red, Cset: true, D: 1000},
      {L: 0, Bri: gohue.Bright, Briset: true, C: gohue.Red, Cset: true, D: 1500},
      {L: 0, C: gohue.Red, Cset: true, D: 2000},
      {L: 0, C: gohue.Red, Cset: true, D: 2500},
      {L: 0, Bri: gohue.Dim, Briset: true, D: 3000},
      {L: 0, Bri: gohue.Dim, Briset: true, D: 3500},
      {L: 0, Bri: gohue.Dim, Briset: true, D: 4000}}
  verifyAction(t, expected, action)
}

func TestOnColor(t *testing.T) {
  action := gohue.Action{
      On: true, C: gohue.ColorPtr(gohue.NewColor(0.4, 0.2))}
  expected := []request {{L: 0, C: gohue.NewColor(0.4, 0.2), Cset: true, On: true, Onset: true, D: 0}}
  verifyAction(t, expected, action)
}

func TestOnBrightness(t *testing.T) {
  action := gohue.Action{
      On: true, Bri: gohue.Uint8Ptr(135)}
  expected := []request {{L: 0, Bri: 135, Briset: true, On: true, Onset: true, D: 0}}
  verifyAction(t, expected, action)
}

func TestOn(t *testing.T) {
  action := gohue.Action{On: true}
  expected := []request {{L: 0, On: true, Onset: true, D: 0}}
  verifyAction(t, expected, action)
}

func TestOff(t *testing.T) {
  action := gohue.Action{Off: true}
  expected := []request {{L: 0, On: false, Onset: true, D: 0}}
  verifyAction(t, expected, action)
}

func TestColorOnly(t *testing.T) {
  action := gohue.Action{C: &gohue.Yellow}
  expected := []request {{L: 0, C: gohue.Yellow, Cset: true, D: 0}}
  verifyAction(t, expected, action)
}

func TestRepeat(t *testing.T) {
  action := gohue.Action{On: true, Repeat: 3}
  expected := []request {
      {L: 0, On: true, Onset: true, D: 0},
      {L: 0, On: true, Onset: true, D: 0},
      {L: 0, On: true, Onset: true, D: 0}}
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

func TestSeries2(t *testing.T) {
  action := gohue.Action{
      Lights: []int{1, 4},
      Series: []*gohue.Action {
          {On: true},
          {Sleep: 3000},
          {Off: true}}}
  expected := []request {
      {L: 1, On: true, Onset: true,  D: 0},
      {L: 4, On: true, Onset: true,  D: 0},
      {L: 1, On: false, Onset: true, D: 3000},
      {L: 4, On: false, Onset: true, D: 3000}}
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
  if err := tasks.RunForTesting(action.AsTask(context, nil), clock); err != kSomeError {
    t.Error("Expected to get kSomeError.")
  }
  if !reflect.DeepEqual(expected, context.requests) {
    t.Errorf("Expected %v, got %v", expected, context.requests)
  }
}
  
type request struct {
  L int
  C gohue.Color
  Cset bool
  Bri uint8
  Briset bool
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
  if p.Bri != nil {
    r.Bri = *p.Bri
    r.Briset = true
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

func verifyAction(t *testing.T, expected []request, action gohue.Action) {
  clock := &tasks.ClockForTesting{kNow}
  context := &setterForTesting{clock: clock, now: kNow}
  tasks.RunForTesting(action.AsTask(context, nil), clock)
  if !reflect.DeepEqual(expected, context.requests) {
    t.Errorf("Expected %v, got %v", expected, context.requests)
  }
}
