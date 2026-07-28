// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package registry

import "testing"

func TestTheatersWellFormed(t *testing.T) {
	if len(Theaters) == 0 {
		t.Fatal("no theaters registered")
	}
	seen := map[string]bool{}
	for _, th := range Theaters {
		if th.Slug == "" || th.PopcornURL == "" || th.FallbackName == "" {
			t.Errorf("incomplete theater: %+v", th)
		}
		if seen[th.Slug] {
			t.Errorf("duplicate slug %q", th.Slug)
		}
		seen[th.Slug] = true
	}
}

func TestName(t *testing.T) {
	if got := (Theater{FallbackName: "SM MegaMall", DisplayName: "SM Megamall"}).Name(); got != "SM Megamall" {
		t.Errorf("DisplayName should win, got %q", got)
	}
	if got := (Theater{FallbackName: "Greenbelt 3"}).Name(); got != "Greenbelt 3" {
		t.Errorf("FallbackName fallback, got %q", got)
	}
}

func TestBySlug(t *testing.T) {
	if _, ok := BySlug("sm-megamall"); !ok {
		t.Error("sm-megamall should resolve")
	}
	if _, ok := BySlug("does-not-exist"); ok {
		t.Error("unknown slug should not resolve")
	}
}

func TestPriceFor(t *testing.T) {
	if p, ok := PriceFor("ortigas-cinemas-estancia", "Cinema 1"); !ok || p != 440.00 {
		t.Errorf("regular price = %v ok=%v, want 440 true", p, ok)
	}
	if p, ok := PriceFor("ortigas-cinemas-estancia", "Screening Room 2"); !ok || p != 590.00 {
		t.Errorf("premium price = %v ok=%v, want 590 true", p, ok)
	}
	if p, ok := PriceFor("power-plant-mall", "IMAX"); ok {
		t.Errorf("Power Plant has no premium tier, want ok=false, got %v", p)
	}
	if _, ok := PriceFor("robinsons-place-manila", "Cinema 1"); ok {
		t.Error("unpriced theater should return ok=false")
	}
}
