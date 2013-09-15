// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

package gohue_test

import (
  "github.com/keep94/gohue"
  "github.com/keep94/tasks"
  "reflect"
  "testing"
  "time"
)

var (
  kNow = time.Date(2013, 9, 15, 14, 0, 0, 0, time.Local)
)

func TestTransition(t *testing.T) {
  transition := gohue.Transition{
      LightId: 2,
      Cds: []gohue.ColorDuration{
          {C: gohue.NewColor(0.2, 0.1, 0), D: 0},
          {C: gohue.NewColor(0.3, 0.3, 30), D: 1000},
          {C: gohue.NewColor(0.9, 0.9, 100), D: 1000},
          {C: gohue.NewColor(0.8, 0.7, 100), D: 1000},
          {C: gohue.NewColor(0.2, 0.4, 10), D: 1750},
          {C: gohue.NewColor(0.29, 0.46, 22), D: 2500}},
      Refresh: 500,
      On: true}
  expected := []request {
      {L: 2, C: gohue.NewColor(0.2, 0.1, 0), Cset: true, On: true, Onset: true, D: 0},
      {L: 2, C: gohue.NewColor(0.25, 0.2, 15), Cset: true, D: 500},
      {L: 2, C: gohue.NewColor(0.3, 0.3, 30), Cset: true, D: 1000},
      {L: 2, C: gohue.NewColor(0.4, 0.5, 40), Cset: true, D: 1500},
      {L: 2, C: gohue.NewColor(0.23, 0.42, 14), Cset: true, D:2000},
      {L: 2, C: gohue.NewColor(0.29, 0.46, 22), Cset: true, D:2500}}
  clock := &tasks.ClockForTesting{kNow}
  context := &setterForTesting{clock: clock, now: kNow}
  tasks.RunForTesting(transition.AsTask(context), clock)
  if !reflect.DeepEqual(expected, context.requests) {
    t.Errorf("Expected %v, got %v", expected, context.requests)
  }
}

func TestTransition2(t *testing.T) {
  transition := gohue.Transition{
      LightId: 3,
      Cds: []gohue.ColorDuration{
          {C: gohue.NewColor(0.2, 0.1, 0), D: 0},
          {C: gohue.NewColor(0.3, 0.3, 30), D: 1000}},
      Refresh: 600}
  expected := []request {
      {L: 3, C: gohue.NewColor(0.2, 0.1, 0), Cset: true, D: 0},
      {L: 3, C: gohue.NewColor(0.26, 0.22, 18), Cset: true, D: 600},
      {L: 3, C: gohue.NewColor(0.3, 0.3, 30), Cset: true, D: 1200}}
  clock := &tasks.ClockForTesting{kNow}
  context := &setterForTesting{clock: clock, now: kNow}
  tasks.RunForTesting(transition.AsTask(context), clock)
  if !reflect.DeepEqual(expected, context.requests) {
    t.Errorf("Expected %v, got %v", expected, context.requests)
  }
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
  return
}
