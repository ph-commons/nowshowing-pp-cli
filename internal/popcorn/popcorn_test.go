// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package popcorn

import "testing"

// Fixture mirrors the popcorn.app page shape: an inline `allShowtimes: {...}`
// blob wrapped in surrounding JS. Keep this in sync with the live page; a
// failure here means the source structure drifted.
// Note the deliberate brace inside a movie title ("{Reboot}") — the extractor
// must not treat it as JSON structure. A naive brace-counter truncates here.
const fixture = `<html><head><script>
window.__data = {
  allShowtimes: {"2026-07-28": [
    {"MovieName": "Superman {Reboot}", "Cinemas": {"Cinema 2": [{"ShowTime": "2:20 PM"}, {"ShowTime": "5:00 PM"}]}},
    {"MovieName": "Disney: Moana 2", "Cinemas": {"Cinema 1": [{"ShowTime": "1:00 PM"}]}}
  ]},
  other: 1
};
</script></head></html>`

func TestExtractAllShowtimes(t *testing.T) {
	raw, err := ExtractAllShowtimes([]byte(fixture))
	if err != nil {
		t.Fatalf("ExtractAllShowtimes: %v", err)
	}
	idx, err := BuildIndex(raw, "2026-07-28")
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	e, ok := idx["superman reboot"]
	if !ok {
		t.Fatalf("superman not indexed (brace-in-title truncation?): %+v", idx)
	}
	if len(e.Showtimes) != 2 || e.Showtimes[0] != "2:20PM" {
		t.Errorf("superman showtimes = %v", e.Showtimes)
	}
	if _, ok := idx["moana 2"]; !ok {
		t.Errorf("moana 2 not indexed")
	}
}

func TestExtractMissingMarker(t *testing.T) {
	if _, err := ExtractAllShowtimes([]byte("<html>no blob here</html>")); err == nil {
		t.Error("expected error when marker absent")
	}
}

func TestCheckShowtimes(t *testing.T) {
	// Verified: upcoming CTC times exactly equal popcorn times.
	pc := &Entry{RawTitle: "Superman", Showtimes: []string{"2:20PM", "5:00PM"}}
	res := CheckShowtimes([]string{"11:00AM", "2:20PM", "5:00PM"}, pc, 720) // now = 12:00PM
	if res.Status != Verified {
		t.Errorf("want verified, got %s (%+v)", res.Status, res)
	}
	if res.ElapsedCount != 1 {
		t.Errorf("want 1 elapsed (11:00AM), got %d", res.ElapsedCount)
	}

	// Partial: one upcoming time agrees, one differs.
	pc2 := &Entry{RawTitle: "Superman", Showtimes: []string{"2:20PM", "8:00PM"}}
	res2 := CheckShowtimes([]string{"2:20PM", "5:00PM"}, pc2, 0)
	if res2.Status != Partial || res2.AgreeCount != 1 {
		t.Errorf("want partial agree=1, got %s agree=%d", res2.Status, res2.AgreeCount)
	}

	// Mismatch: no overlap.
	res3 := CheckShowtimes([]string{"2:20PM"}, &Entry{Showtimes: []string{"9:00PM"}}, 0)
	if res3.Status != Mismatch {
		t.Errorf("want mismatch, got %s", res3.Status)
	}

	// CTC-only: no popcorn data.
	res4 := CheckShowtimes([]string{"2:20PM"}, nil, 0)
	if res4.Status != CTCOnly {
		t.Errorf("want ctc-only, got %s", res4.Status)
	}
}
