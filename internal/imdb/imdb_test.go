// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package imdb

import "testing"

func TestQueryFor(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Superman", "superman"},
		{"F1: The Movie", "f1_the_movie"},
		{"Lilo & Stitch", "lilo_stitch"},
	}
	for _, c := range cases {
		if got := queryFor(c.in); got != c.want {
			t.Errorf("queryFor(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseSingleRecent(t *testing.T) {
	// "Moana" collides: 2016 original + 2026 re-release; only the recent one wins.
	data := []byte(`{"d":[
	  {"id":"tt3521164","l":"Moana","qid":"movie","y":2016},
	  {"id":"tt9999999","l":"Moana","qid":"movie","y":2026},
	  {"id":"tt0000001","l":"Moana: Behind the Scenes","qid":"tvSeries","y":2026}
	]}`)
	res, err := parseSuggestions(data, "Moana", 2026)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if res == nil || res.ID != "tt9999999" || res.Ambiguous {
		t.Errorf("want tt9999999 not-ambiguous, got %+v", res)
	}
	if res.URL != "https://www.imdb.com/title/tt9999999/" {
		t.Errorf("url = %q", res.URL)
	}
}

func TestParseAmbiguous(t *testing.T) {
	// Two recent exact matches → ambiguous, first exact returned.
	data := []byte(`{"d":[
	  {"id":"tt111","l":"The Furious","qid":"movie","y":2025},
	  {"id":"tt222","l":"The Furious","qid":"movie","y":2026}
	]}`)
	res, err := parseSuggestions(data, "The Furious", 2026)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if res == nil || !res.Ambiguous {
		t.Errorf("want ambiguous, got %+v", res)
	}
}

func TestParseNoMatch(t *testing.T) {
	data := []byte(`{"d":[{"id":"tt1","l":"Something Else","qid":"movie","y":2026}]}`)
	res, err := parseSuggestions(data, "Superman", 2026)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if res != nil {
		t.Errorf("want nil for no exact match, got %+v", res)
	}
}
