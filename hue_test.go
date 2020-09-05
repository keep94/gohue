// Copyright 2013 Travis Keep. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or
// at http://opensource.org/licenses/BSD-3-Clause.

package gohue_test

import (
	"github.com/keep94/gohue"
	"testing"
)

func TestColorBlend(t *testing.T) {
	c1 := gohue.NewColor(0.3, 0.2)
	c2 := gohue.NewColor(0.2, 0.6)
	expected := gohue.NewColor(0.23, 0.48)
	if actual := c1.Blend(c2, 0.7); actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
	expected = gohue.NewColor(0.2, 0.6)
	if actual := c1.Blend(c2, 1.0); actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
	expected = gohue.NewColor(0.3, 0.2)
	if actual := c1.Blend(c2, 0.0); actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestColorXY(t *testing.T) {
	c := gohue.NewColor(0.0, 1.0)
	if c != gohue.NewColor(c.X(), c.Y()) {
		t.Error("Round trip of X and Y failed.")
	}
	c = gohue.NewColor(1.0, 0.0)
	if c != gohue.NewColor(c.X(), c.Y()) {
		t.Error("Round trip of X and Y failed.")
	}
	c = gohue.NewColor(7.0, 0.2)
	if c != gohue.NewColor(c.X(), c.Y()) {
		t.Error("Round trip of X and Y failed.")
	}
}

func TestMaybeColor(t *testing.T) {
	var m, c gohue.MaybeColor
	v := gohue.NewColor(0.4, 0.6)
	m.Set(v)
	if m != gohue.NewMaybeColor(v) {
		t.Error("MaybeColor Set broken.")
	}
	verifyString(t, "Just (0.4000, 0.6000)", m.String())
	m.Clear()
	if m != c {
		t.Error("MaybeColor Clear broken.")
	}
	verifyString(t, "Nothing", m.String())
}

func verifyString(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}
