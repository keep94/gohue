// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

package gohue_test

import (
  "errors"
  "github.com/keep94/gohue"
  "github.com/keep94/maybe"
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
              {C: gohue.NewMaybeColor(gohue.NewColor(0.2, 0.1)),
               Bri: maybe.NewUint8(0), D: 0},
              {C: gohue.NewMaybeColor(gohue.NewColor(0.3, 0.3)),
               Bri: maybe.NewUint8(30), D: 1000},
              {C: gohue.NewMaybeColor(gohue.NewColor(0.9, 0.9)),
               Bri: maybe.NewUint8(100), D: 1000},
              {C: gohue.NewMaybeColor(gohue.NewColor(0.8, 0.7)),
              Bri: maybe.NewUint8(100),  D: 1000},
              {C: gohue.NewMaybeColor(gohue.NewColor(0.2, 0.4)),
               Bri: maybe.NewUint8(10), D: 1750},
              {C: gohue.NewMaybeColor(gohue.NewColor(0.29, 0.46)),
               Bri: maybe.NewUint8(22), D: 2500}},
          Refresh: 500},
      On: true}
  expected := []request {
      {L: 2,
       C: gohue.NewMaybeColor(gohue.NewColor(0.2, 0.1)),
       Bri: maybe.NewUint8(0),
       On: maybe.NewBool(true),
       D: 0},
      {L: 2,
       C: gohue.NewMaybeColor(gohue.NewColor(0.25, 0.2)),
       Bri: maybe.NewUint8(15),
       D: 500},
      {L: 2,
       C: gohue.NewMaybeColor(gohue.NewColor(0.8, 0.7)),
       Bri: maybe.NewUint8(100),
       D: 1000},
      {L: 2,
       C: gohue.NewMaybeColor(gohue.NewColor(0.4, 0.5)),
       Bri: maybe.NewUint8(40),
       D: 1500},
      {L: 2,
       C: gohue.NewMaybeColor(gohue.NewColor(0.23, 0.42)),
       Bri: maybe.NewUint8(14),
       D:2000},
      {L: 2,
       C: gohue.NewMaybeColor(gohue.NewColor(0.29, 0.46)),
       Bri: maybe.NewUint8(22),
       D:2500}}
  verifyAction(t, expected, action)
}

func TestGradient2(t *testing.T) {
  action := gohue.Action{
      G: &gohue.Gradient{
          Cds: []gohue.ColorDuration{
              {C: gohue.NewMaybeColor(gohue.NewColor(0.2, 0.1)), D: 0},
              {C: gohue.NewMaybeColor(gohue.NewColor(0.3, 0.3)), D: 1000}},
          Refresh: 600}}
  expected := []request {
      {L: 0, C: gohue.NewMaybeColor(gohue.NewColor(0.2, 0.1)), D: 0},
      {L: 0, C: gohue.NewMaybeColor(gohue.NewColor(0.26, 0.22)), D: 600},
      {L: 0, C: gohue.NewMaybeColor(gohue.NewColor(0.3, 0.3)), D: 1200}}
  verifyAction(t, expected, action)
}

func TestGradient3(t *testing.T) {
  action := gohue.Action{
      G: &gohue.Gradient{
          Cds: []gohue.ColorDuration{
              {Bri: maybe.NewUint8(gohue.Bright), D: 0},
              {Bri: maybe.NewUint8(gohue.Bright),
               C: gohue.NewMaybeColor(gohue.Red),
               D: 1000},
              {C: gohue.NewMaybeColor(gohue.Red), D: 2000},
              {Bri: maybe.NewUint8(gohue.Dim), D: 3000},
              {Bri: maybe.NewUint8(gohue.Dim), D: 4000}},
          Refresh: 500}}
  expected := []request {
      {L: 0, Bri: maybe.NewUint8(gohue.Bright), D: 0},
      {L: 0, Bri: maybe.NewUint8(gohue.Bright), D: 500},
      {L: 0, 
       Bri: maybe.NewUint8(gohue.Bright),
       C: gohue.NewMaybeColor(gohue.Red),
       D: 1000},
      {L: 0, 
       Bri: maybe.NewUint8(gohue.Bright),
       C: gohue.NewMaybeColor(gohue.Red),
       D: 1500},
      {L: 0, C: gohue.NewMaybeColor(gohue.Red), D: 2000},
      {L: 0, C: gohue.NewMaybeColor(gohue.Red), D: 2500},
      {L: 0, Bri: maybe.NewUint8(gohue.Dim), D: 3000},
      {L: 0, Bri: maybe.NewUint8(gohue.Dim), D: 3500},
      {L: 0, Bri: maybe.NewUint8(gohue.Dim), D: 4000}}
  verifyAction(t, expected, action)
}

func TestOnColor(t *testing.T) {
  action := gohue.Action{
      On: true, C: gohue.NewMaybeColor(gohue.NewColor(0.4, 0.2))}
  expected := []request {
      {L: 0,
       C: gohue.NewMaybeColor(gohue.NewColor(0.4, 0.2)),
       On: maybe.NewBool(true),
       D: 0}}
  verifyAction(t, expected, action)
}

func TestOnBrightness(t *testing.T) {
  action := gohue.Action{
      On: true, Bri: maybe.NewUint8(135)}
  expected := []request {
      {L: 0, Bri: maybe.NewUint8(135), On: maybe.NewBool(true), D: 0}}
  verifyAction(t, expected, action)
}

func TestOn(t *testing.T) {
  action := gohue.Action{On: true}
  expected := []request {{L: 0, On: maybe.NewBool(true), D: 0}}
  verifyAction(t, expected, action)
}

func TestOff(t *testing.T) {
  action := gohue.Action{Off: true}
  expected := []request {{L: 0, On: maybe.NewBool(false), D: 0}}
  verifyAction(t, expected, action)
}

func TestColorOnly(t *testing.T) {
  action := gohue.Action{C: gohue.NewMaybeColor(gohue.Yellow)}
  expected := []request {
      {L: 0,
       C: gohue.NewMaybeColor(gohue.Yellow),
       D: 0}}
  verifyAction(t, expected, action)
}

func TestRepeat(t *testing.T) {
  action := gohue.Action{On: true, Repeat: 3}
  expected := []request {
      {L: 0, On: maybe.NewBool(true), D: 0},
      {L: 0, On: maybe.NewBool(true), D: 0},
      {L: 0, On: maybe.NewBool(true), D: 0}}
  verifyAction(t, expected, action)
}

func TestSeries(t *testing.T) {
  action := gohue.Action{
      Series: []*gohue.Action {
          {Lights: []int{2, 3}, On: true},
          {Sleep: 3000},
          {Off: true}}}
  expected := []request {
      {L: 2, On: maybe.NewBool(true),  D: 0},
      {L: 3, On: maybe.NewBool(true),  D: 0},
      {L: 0, On: maybe.NewBool(false), D: 3000}}
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
      {L: 1, On: maybe.NewBool(true),  D: 0},
      {L: 4, On: maybe.NewBool(true),  D: 0},
      {L: 1, On: maybe.NewBool(false),  D: 3000},
      {L: 4, On: maybe.NewBool(false),  D: 3000}}
  verifyAction(t, expected, action)
}

func TestError(t *testing.T) {
  action := gohue.Action{
      Series: []*gohue.Action {
          {Lights: []int{2, 3}, On: true},
          {Sleep: 3000},
          {Off: true}}}
  expected := []request {
      {L: 2, On: maybe.NewBool(true),  D: 0}}
  clock := &tasks.ClockForTesting{kNow}
  context := &setterForTesting{err: kSomeError, clock: clock, now: kNow}
  if err := tasks.RunForTesting(
      action.AsTask(context, nil), clock); err != kSomeError {
    t.Error("Expected to get kSomeError.")
  }
  if !reflect.DeepEqual(expected, context.requests) {
    t.Errorf("Expected %v, got %v", expected, context.requests)
  }
}
  
type request struct {
  L int
  C gohue.MaybeColor
  Bri maybe.Uint8
  On maybe.Bool
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
  r.C = p.C
  r.Bri = p.Bri
  r.On = p.On
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
