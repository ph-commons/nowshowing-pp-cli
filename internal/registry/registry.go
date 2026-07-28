// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

// Package registry is the curated set of tracked Metro Manila and Iloilo
// cinemas, plus verified ticket prices, ported from the ~/src/nowshowing
// THEATERS and THEATER_PRICING constants. This is static reference data the
// CLI maintains, not fetched from any API.
package registry

import "strings"

// Theater is one tracked cinema.
type Theater struct {
	// Slug is the ClickTheCity theater slug used in the API path.
	Slug string `json:"slug"`
	// PopcornURL is the popcorn.app per-cinema page used for cross-checking.
	PopcornURL string `json:"popcorn_url"`
	// FallbackName matches ClickTheCity's own name; used if the API is down.
	FallbackName string `json:"fallback_name"`
	// DisplayName overrides the rendered name (casing/shortening); optional.
	DisplayName string `json:"display_name,omitempty"`
}

// Name returns the preferred display name: DisplayName if set, else FallbackName.
func (t Theater) Name() string {
	if t.DisplayName != "" {
		return t.DisplayName
	}
	return t.FallbackName
}

// Theaters is the tracked cinema set (16 as of the ~/src/nowshowing snapshot).
var Theaters = []Theater{
	{Slug: "robinsons-galleria-ortigas", PopcornURL: "https://www.popcorn.app/ph/robinsons-movieworld/galleria-ortigas/cinema/550", FallbackName: "Robinsons Galleria Ortigas"},
	{Slug: "power-plant-mall", PopcornURL: "https://www.popcorn.app/ph/powerplant/power-plant-mall/cinema/2633", FallbackName: "Power Plant"},
	{Slug: "ortigas-cinemas-estancia", PopcornURL: "https://www.popcorn.app/ph/ortigas-cinema/estancia-cinemas/cinema/2766", FallbackName: "Ortigas Cinemas Estancia"},
	{Slug: "robinsons-place-manila", PopcornURL: "https://www.popcorn.app/ph/robinsons/manila/cinema/552", FallbackName: "Robinsons Place Manila"},
	{Slug: "sm-megamall", PopcornURL: "https://www.popcorn.app/ph/sm-cinemas/sm-city-megamall/cinema/2763", FallbackName: "SM MegaMall", DisplayName: "SM Megamall"},
	{Slug: "sm-city-north-edsa", PopcornURL: "https://www.popcorn.app/ph/sm-cinemas/sm-city-north-edsa/cinema/512", FallbackName: "SM City North EDSA", DisplayName: "SM North EDSA"},
	{Slug: "the-podium", PopcornURL: "https://www.popcorn.app/ph/sm-cinema/podium/cinema/517", FallbackName: "The Podium"},
	{Slug: "greenbelt-3", PopcornURL: "https://www.popcorn.app/ph/ayala-malls-cinemas/greenbelt-3/cinema/543", FallbackName: "Greenbelt 3"},
	{Slug: "glorietta-4", PopcornURL: "https://www.popcorn.app/ph/ayala-malls-cinemas/glorietta-4/cinema/541", FallbackName: "Glorietta 4"},
	{Slug: "trinoma-mall", PopcornURL: "https://www.popcorn.app/ph/ayala-malls-cinemas/trinoma/cinema/548", FallbackName: "TriNoma Mall", DisplayName: "Trinoma"},
	{Slug: "up-town-center", PopcornURL: "https://www.popcorn.app/ph/ayala-malls-cinemas/up-town-center/cinema/549", FallbackName: "Ayala U.P. Town Center", DisplayName: "UP Town Center"},
	{Slug: "sm-city-iloilo", PopcornURL: "https://www.popcorn.app/ph/sm-cinemas/sm-city-iloilo/cinema/2764", FallbackName: "SM City Iloilo"},
	{Slug: "robinsons-place-iloilo", PopcornURL: "https://www.popcorn.app/ph/robinsons-movieworld/place-iloilo/cinema/553", FallbackName: "Robinsons Place Iloilo"},
	{Slug: "robinsons-place-jaro", PopcornURL: "https://www.popcorn.app/ph/robinsons-movieworld/place-jaro/cinema/555", FallbackName: "Robinsons Place Jaro"},
	{Slug: "festive-walk-iloilo", PopcornURL: "https://www.popcorn.app/ph/megaworld-cinemas/festive-walk-iloilo/cinema/2765", FallbackName: "Festive Walk Iloilo"},
	{Slug: "vista-mall-iloilo", PopcornURL: "https://www.popcorn.app/ph/vista-cinema/vista-mall-iloilo/cinema/2766", FallbackName: "Vista Mall Iloilo"},
}

// BySlug returns the tracked theater with the given slug, or ok=false.
func BySlug(slug string) (Theater, bool) {
	for _, t := range Theaters {
		if t.Slug == slug {
			return t, true
		}
	}
	return Theater{}, false
}

// Pricing is a verified per-tier ticket price snapshot (PHP). Only theaters with
// a confirmed price source appear here; everything else reports an unavailable
// price rather than a guess.
type Pricing struct {
	Regular  float64 `json:"regular"`
	Premium  float64 `json:"premium,omitempty"`
	Verified string  `json:"verified"`
}

// pricing is keyed by the stable ClickTheCity slug (not the live theater name),
// so a change in the API's returned name never silently drops the price.
var pricing = map[string]Pricing{
	"ortigas-cinemas-estancia": {Regular: 440.00, Premium: 590.00, Verified: "2026-07-12"},
	"power-plant-mall":         {Regular: 470.00, Verified: "2026-07-12"},
}

var premiumKeywords = []string{"screening room", "vip", "premiere", "dolby atmos", "imax"}

// PriceFor returns the verified per-seat price (PHP) for a room at the theater
// with the given slug and ok=true, or ok=false when no verified price source
// exists for that theater/tier. A room whose name matches a premium keyword uses
// the premium tier; everything else uses regular.
func PriceFor(slug, cinemaName string) (float64, bool) {
	p, found := pricing[slug]
	if !found {
		return 0, false
	}
	lower := strings.ToLower(cinemaName)
	for _, k := range premiumKeywords {
		if strings.Contains(lower, k) {
			if p.Premium == 0 {
				return 0, false
			}
			return p.Premium, true
		}
	}
	return p.Regular, true
}
