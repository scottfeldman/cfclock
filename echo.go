package main

import (
	"fmt"
	"strconv"
	"strings"
)

// The Echo Gym Timer's remote is infrared. The published capture of the
// GX-IR03 gym-clock remote (the same NEC family these timers use) is the
// signal map below: address 0x01, one command byte per key.
//
// Color keys in that capture are labeled Red Clock, Green Up, Yellow Down,
// and Blue Stopwatch. Those line up with the Echo remote's Clock, count-up,
// countdown, and stopwatch keys. Tabata and Fight Gone Bad are separate
// keys on the Rogue remote; their command bytes are not in the capture, so
// this controller does not invent them. It programs those workouts as an
// interval, which is the sequence the manual documents with Edit, digits,
// and OK.
//
// https://gist.github.com/cfultz/65db5aac7a2d22de17f0a372f8f85ced

const irAddress = 0x01

type Key struct {
	Name string
	Cmd  byte
}

func (k Key) Line() string {
	return fmt.Sprintf("01 %02X  %s", k.Cmd, k.Name)
}

var (
	keyEdit  = Key{"EDIT", 0x0B}
	keyOK    = Key{"OK", 0x5A}
	keyStart = Key{"START", 0x5E}
	keyStop  = Key{"STOP", 0x59}
	keyReset = Key{"RESET", 0x5D}
	keyClock = Key{"CLOCK", 0x53}
	keyUp    = Key{"UP", 0x52}
	keyDown  = Key{"DOWN", 0x51}
)

var digitCmd = [10]byte{
	0: 0x03,
	1: 0x0E,
	2: 0x06,
	3: 0x0F,
	4: 0x12,
	5: 0x07,
	6: 0x13,
	7: 0x16,
	8: 0x02,
	9: 0x17,
}

func digitKey(n int) Key {
	return Key{Name: strconv.Itoa(n), Cmd: digitCmd[n]}
}

// Format is one detent of the format knob.
type Format struct {
	ID    string
	Label string
}

// Format is one detent of the format pot. Order is the live arc of an
// Alps RK09-class pot (~300°): INTERVALS at the CCW end through TABATA
// at the CW end, with the ~60° dead zone between TABATA and INTERVALS.
var formats = []Format{
	{"intervals", "INTERVALS"},
	{"clock", "CLOCK"},
	{"emom", "EMOM"},
	{"amrap", "AMRAP"},
	{"fortime", "FOR TIME"},
	{"tabata", "TABATA"},
}

func formatIndex(id string) int {
	for i, f := range formats {
		if f.ID == id {
			return i
		}
	}
	return -1
}

// The three value knobs. Their units are printed on gears driven by the
// format knob, so each format reads the same knob positions differently.
var valueKnobs = []string{"count", "time", "rest"}

var unitTable = map[string][3]string{
	"emom":      {"ROUNDS", "", ""},
	"amrap":     {"MINUTES", "", ""},
	"fortime":   {"CAP", "", ""},
	"tabata":    {"", "", ""},
	"intervals": {"ROUNDS", "WORK", "REST"},
	"clock":     {"", "", ""},
}

// Printed on the value disks, one stop per detent.
var (
	countStops = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 24, 25, 30, 35, 40, 45, 50, 60}

	timeStops = []int{10, 15, 20, 30, 40, 45, 60, 75, 90, 120, 150, 180, 240, 300, 360, 420, 480, 600}
	restStops = []int{0, 5, 10, 15, 20, 30, 45, 60, 90, 120, 180, 240, 300}
)

// Session is the position of the four Alps RK09-class pots. Dial, Count,
// Time, and Rest are absolute stop indices with end stops. Format Dial
// runs INTERVALS…TABATA on the live arc; the pot dead zone sits between
// TABATA and INTERVALS.
type Session struct {
	Dial  int
	Count int
	Time  int
	Rest  int
	Slot  int
	Note  string
}

func defaultSession() Session {
	return Session{
		Dial:  formatIndex("emom"),
		Count: stopIndex(countStops, 16),
		Time:  stopIndex(timeStops, 60),
		Rest:  stopIndex(restStops, 60),
		Slot:  1,
		Note:  "odd: 12 toes-to-bar / even: 15 wall balls",
	}
}

func stopIndex(stops []int, v int) int {
	for i, s := range stops {
		if s == v {
			return i
		}
	}
	return 0
}

func mod(a, n int) int {
	return ((a % n) + n) % n
}

func (s Session) Format() string { return formats[s.Dial].ID }

func (s Session) Units(knob int) string { return unitTable[s.Format()][knob] }

func (s *Session) Apply(op, value, text string) {
	switch op {
	case "turn":
		knob, dir, ok := strings.Cut(value, ":")
		if !ok {
			return
		}
		d := 1
		if dir == "-" {
			d = -1
		}
		switch knob {
		case "format":
			s.Dial += d
		case "count":
			s.Count += d
		case "time":
			s.Time += d
		case "rest":
			s.Rest += d
		}
	case "format":
		t := formatIndex(value)
		if t < 0 {
			return
		}
		s.Dial = t
	case "note":
		s.Note = text
	}
}

func (s *Session) Normalize() {
	s.Dial = clamp(s.Dial, 0, len(formats)-1)
	s.Count = clamp(s.Count, 0, len(countStops)-1)
	s.Time = clamp(s.Time, 0, len(timeStops)-1)
	s.Rest = clamp(s.Rest, 0, len(restStops)-1)
	s.Slot = clamp(s.Slot, 0, 9)
	s.Note = clip(s.Note, 80)
}

func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

func clip(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func (s Session) Clone() Session { return s }

func (s Session) workSec() int {
	switch s.Format() {
	case "emom":
		return 60
	case "intervals":
		return timeStops[s.Time]
	case "tabata":
		return 20
	}
	return 0
}

func (s Session) restSec() int {
	switch s.Format() {
	case "intervals":
		return restStops[s.Rest]
	case "tabata":
		return 10
	}
	return 0
}

func (s Session) rounds() int {
	switch s.Format() {
	case "emom", "intervals":
		return s.count()
	case "tabata":
		return 8
	}
	return 0
}

func (s Session) count() int { return countStops[s.Count] }

func (s Session) lengthSec() int { return s.count() * 60 }

// mode is what the Echo itself runs for this format.
func (s Session) mode() string {
	switch s.Format() {
	case "amrap":
		return "down"
	case "fortime":
		return "up"
	case "clock":
		return "clock"
	default:
		return "interval"
	}
}

// Signals is the key sequence the remote would send to load this program.
// START is not included; that key runs whatever is loaded.
func (s Session) Signals() []Key {
	switch s.mode() {
	case "interval":
		return intervalSignals(s)
	case "down":
		return append([]Key{keyDown, keyEdit}, append(mmssKeys(s.lengthSec()), keyOK)...)
	case "up":
		return append([]Key{keyUp, keyEdit}, append(mmssKeys(s.lengthSec()), keyOK)...)
	default:
		return []Key{keyClock}
	}
}

func intervalSignals(s Session) []Key {
	keys := []Key{digitKey(s.Slot), keyEdit}
	keys = append(keys, mmssKeys(s.workSec())...)
	keys = append(keys, keyEdit)
	keys = append(keys, mmssKeys(s.restSec())...)
	keys = append(keys, keyOK)
	keys = append(keys, roundKeys(s.rounds())...)
	keys = append(keys, keyOK)
	return keys
}

func mmssKeys(sec int) []Key {
	m := sec / 60
	s := sec % 60
	return []Key{
		digitKey(m / 10),
		digitKey(m % 10),
		digitKey(s / 10),
		digitKey(s % 10),
	}
}

func roundKeys(n int) []Key {
	if n > 99 {
		n = 99
	}
	return []Key{digitKey(n / 10), digitKey(n % 10)}
}

func (s Session) Summary() string {
	switch s.Format() {
	case "emom":
		return fmt.Sprintf("EMOM %d", s.rounds())
	case "amrap":
		return fmt.Sprintf("AMRAP %d", s.count())
	case "fortime":
		return fmt.Sprintf("FOR TIME  ·  %d:00 CAP", s.count())
	case "tabata":
		return "TABATA  ·  :20 / :10 × 8  ·  4:00"
	case "intervals":
		w, r, n := s.workSec(), s.restSec(), s.rounds()
		return fmt.Sprintf("%s ON / %s OFF × %d  ·  %s", formatLoose(w), formatLoose(r), n, formatLoose((w+r)*n))
	default:
		return "TIME OF DAY"
	}
}

// Cue is one stretch of the simulated Echo face.
type Cue struct {
	Count   string `json:"count"`
	Seconds int    `json:"seconds"`
	Round   int    `json:"round"`
	Phase   string `json:"phase"`
}

func (s Session) Cues() []Cue {
	switch s.mode() {
	case "interval":
		n, w, r := s.rounds(), s.workSec(), s.restSec()
		cues := make([]Cue, 0, n*2)
		for i := 1; i <= n; i++ {
			cues = append(cues, Cue{Count: "down", Seconds: w, Round: i, Phase: "WORK"})
			if r > 0 {
				cues = append(cues, Cue{Count: "down", Seconds: r, Round: i, Phase: "REST"})
			}
		}
		return cues
	case "down":
		return []Cue{{Count: "down", Seconds: s.lengthSec(), Phase: "DOWN"}}
	case "up":
		return []Cue{{Count: "up", Seconds: s.lengthSec(), Phase: "UP"}}
	default:
		return []Cue{{Count: "clock", Phase: "CLOCK"}}
	}
}

func readyDigits(s Session) string {
	switch s.mode() {
	case "interval":
		return formatLED(s.workSec())
	case "down":
		return formatLED(s.lengthSec())
	case "up":
		return "00:00"
	default:
		return "--:--"
	}
}

func readyRounds(s Session) string {
	if s.mode() != "interval" {
		return ""
	}
	return strconv.Itoa(s.rounds())
}

func formatLED(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

func formatLoose(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}
