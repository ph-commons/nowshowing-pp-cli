// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package ctc

import "testing"

const sample = `{
  "status": true,
  "theater": {"name": "SM Megamall", "address": "Ortigas", "slug": "sm-megamall"},
  "now_showing": [
    {"movieId": 1, "title": "Superman", "mtrcb_rating": "PG", "running_time": "2 hr 9 min", "in3d": false},
    {"movieId": 2, "title": "Disney: Moana 2", "mtrcb_rating": "G", "running_time": "1 hr 40 min", "in3d": true}
  ],
  "schedules": [
    {"movieId": 1, "date": "2026-07-28", "theaterName": "Cinema 2", "showtimes": ["2:20 PM", "5:00 PM"]},
    {"movieId": 1, "date": "2026-07-28", "theaterName": "Cinema 10", "showtimes": ["11:00 AM"]},
    {"movieId": 2, "date": "2026-07-28", "theaterName": "Cinema 1", "showtimes": ["1:00 PM"]},
    {"movieId": 1, "date": "2026-07-29", "theaterName": "Cinema 2", "showtimes": ["9:00 PM"]}
  ]
}`

func TestParse(t *testing.T) {
	res, err := Parse([]byte(sample), "2026-07-28")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.TheaterName != "SM Megamall" {
		t.Errorf("theater = %q", res.TheaterName)
	}
	if len(res.Movies) != 2 {
		t.Fatalf("want 2 movies, got %d", len(res.Movies))
	}
	// Movies sorted by title: "Disney: Moana 2" (d) before "Superman" (s).
	moana := res.Movies[0]
	if moana.RawTitle != "Disney: Moana 2" || moana.Key != "moana 2" {
		t.Errorf("first movie = %q key=%q", moana.RawTitle, moana.Key)
	}
	sup := res.Movies[1]
	// Superman: two cinema rows, 3 showtimes today (the 2026-07-29 row excluded).
	if len(sup.CinemaRows) != 2 {
		t.Errorf("Superman cinema rows = %d, want 2", len(sup.CinemaRows))
	}
	if got := len(sup.Showtimes); got != 3 {
		t.Errorf("Superman showtimes = %v, want 3", sup.Showtimes)
	}
	// Natural cinema order: "Cinema 2" before "Cinema 10".
	if sup.CinemaRows[0].Cinema != "Cinema 2" {
		t.Errorf("cinema order wrong: %q first", sup.CinemaRows[0].Cinema)
	}
	// Showtimes sorted chronologically: 11:00AM, 2:20PM, 5:00PM.
	if sup.Showtimes[0] != "11:00AM" || sup.Showtimes[2] != "5:00PM" {
		t.Errorf("showtime order = %v", sup.Showtimes)
	}
}

func TestParseStatusFalse(t *testing.T) {
	if _, err := Parse([]byte(`{"status": false}`), "2026-07-28"); err == nil {
		t.Error("status=false should be an error, not empty result")
	}
}
