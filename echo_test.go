package main

import "testing"

func TestEMOM16RemoteSequence(t *testing.T) {
	s := defaultSession()
	s.Normalize()
	got := names(s.Signals())
	// P1, Edit, work 01:00, Edit, rest 00:00, OK, rounds 16, OK.
	want := []string{"1", "EDIT", "0", "1", "0", "0", "EDIT", "0", "0", "0", "0", "OK", "1", "6", "OK"}
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("keys %v", got)
	}
	if cmd := s.Signals()[0].Cmd; cmd != 0x0E {
		t.Fatalf("digit 1 cmd %02X", cmd)
	}
	if s.Signals()[1].Cmd != 0x0B || s.Signals()[11].Cmd != 0x5A {
		t.Fatalf("edit/ok bytes")
	}
	cues := s.Cues()
	if len(cues) != 16 || cues[0].Seconds != 60 || cues[0].Phase != "WORK" || cues[15].Round != 16 {
		t.Fatalf("cues %+v", cues)
	}
}

func TestTabataIgnoresValueKnobs(t *testing.T) {
	s := defaultSession()
	s.Apply("format", "tabata", "")
	s.Apply("turn", "count:+", "")
	s.Apply("turn", "time:+", "")
	s.Normalize()
	cues := s.Cues()
	if len(cues) != 16 {
		t.Fatalf("len %d", len(cues))
	}
	sum := 0
	for i, c := range cues {
		sum += c.Seconds
		want := "WORK"
		if i%2 == 1 {
			want = "REST"
		}
		if c.Phase != want {
			t.Fatalf("cue %d %s", i, c.Phase)
		}
	}
	if sum != 240 {
		t.Fatalf("sum %d", sum)
	}
}

func TestFormatKnobIsEndless(t *testing.T) {
	s := defaultSession()
	var seen []string
	for range formats {
		s.Apply("turn", "format:+", "")
		seen = append(seen, s.Format())
	}
	want := "amrap fortime tabata intervals clock emom"
	if stringsJoin(seen) != want {
		t.Fatalf("%v", seen)
	}
	if s.Dial != 6 {
		t.Fatalf("dial %d", s.Dial)
	}
}

func TestFormatJumpTakesShortWay(t *testing.T) {
	s := defaultSession()
	s.Apply("format", "clock", "")
	if s.Dial != -1 || s.Format() != "clock" {
		t.Fatalf("dial %d %s", s.Dial, s.Format())
	}
	s.Apply("format", "fortime", "")
	if s.Dial != -4 || s.Format() != "fortime" {
		t.Fatalf("dial %d %s", s.Dial, s.Format())
	}
}

func TestUnitsFollowFormat(t *testing.T) {
	s := defaultSession()
	for _, c := range []struct{ format, count, time, rest string }{
		{"emom", "ROUNDS", "", ""},
		{"amrap", "MINUTES", "", ""},
		{"fortime", "CAP", "", ""},
		{"tabata", "", "", ""},
		{"intervals", "ROUNDS", "WORK", "REST"},
		{"clock", "", "", ""},
	} {
		s.Apply("format", c.format, "")
		if s.Units(0) != c.count || s.Units(1) != c.time || s.Units(2) != c.rest {
			t.Fatalf("%s: %q %q %q", c.format, s.Units(0), s.Units(1), s.Units(2))
		}
	}
}

func TestCountKnobIsShared(t *testing.T) {
	s := defaultSession()
	s.Apply("format", "amrap", "")
	s.Normalize()
	got := names(s.Signals())
	want := []string{"DOWN", "EDIT", "1", "6", "0", "0", "OK"}
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("%v", got)
	}
	if s.Signals()[0].Cmd != 0x51 {
		t.Fatalf("down cmd %02X", s.Signals()[0].Cmd)
	}
	s.Apply("turn", "format:+", "")
	s.Apply("turn", "count:-", "")
	s.Normalize()
	got = names(s.Signals())
	want = []string{"UP", "EDIT", "1", "5", "0", "0", "OK"}
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("%v", got)
	}
}

func TestEMOMIgnoresTimeKnob(t *testing.T) {
	s := defaultSession()
	for range 3 {
		s.Apply("turn", "time:+", "")
	}
	s.Normalize()
	if s.workSec() != 60 || s.Summary() != "EMOM 16" {
		t.Fatalf("work %d summary %q", s.workSec(), s.Summary())
	}
	s.Apply("format", "intervals", "")
	s.Normalize()
	got := names(s.Signals())
	want := []string{"1", "EDIT", "0", "2", "0", "0", "EDIT", "0", "1", "0", "0", "OK", "1", "6", "OK"}
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("keys %v", got)
	}
}

func TestKnobsStopAtDrumEnds(t *testing.T) {
	s := defaultSession()
	s.Count = 0
	s.Apply("turn", "count:-", "")
	s.Rest = 0
	s.Apply("turn", "rest:-", "")
	s.Normalize()
	if s.count() != 1 || s.Rest != 0 {
		t.Fatalf("count %d rest %d", s.count(), s.Rest)
	}
	s.Count = len(countStops) - 1
	s.Apply("turn", "count:+", "")
	s.Normalize()
	if s.count() != 60 {
		t.Fatalf("count %d", s.count())
	}
}

func TestShutterOpeningsMatchUnits(t *testing.T) {
	// Outer disc plus one opening per run of formats that use the knob:
	// count is EMOM-AMRAP-FOR TIME and INTERVALS, time and rest are
	// INTERVALS only.
	for k, want := range []int{3, 2, 2} {
		d := shutterPath(0, 0, k)
		if got := len(splitM(d)); got != want {
			t.Fatalf("knob %d: %d subpaths in %s", k, got, d)
		}
	}
}

func splitM(d string) []string {
	var out []string
	for i, r := range d {
		if r == 'M' {
			out = append(out, d[i:])
		}
	}
	return out
}

func names(keys []Key) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = k.Name
	}
	return out
}

func stringsJoin(parts []string) string {
	return stringsJoinSep(parts, " ")
}

func stringsJoinSep(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}
