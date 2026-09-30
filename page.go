package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "maragu.dev/gomponents"
	comp "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func Page(s Session) Node {
	return comp.HTML5(comp.HTML5Props{
		Title:       "Echo Controller",
		Description: "Controller for a Rogue Echo Gym Timer",
		Language:    "en",
		Head: []Node{
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(Rel("preconnect"), Href("https://fonts.googleapis.com")),
			Link(Rel("preconnect"), Href("https://fonts.gstatic.com"), Attr("crossorigin", "anonymous")),
			Link(Rel("stylesheet"), Href("https://fonts.googleapis.com/css2?family=Barlow+Condensed:wght@500;600;700&family=Share+Tech+Mono&display=swap")),
			Link(Rel("stylesheet"), Href("/static/clock.css")),
			Script(Src("https://unpkg.com/htmx.org@2.0.4")),
		},
		Body: []Node{
			// Outside the swapped board so posts can queue across renders.
			Div(ID("sync")),
			Board(s),
		},
	})
}

func Board(s Session) Node {
	return Div(ID("board"), If(s.Xray, Class("xray")),
		Face(s, time.Now()),
		Deck(s),
	)
}

// Face is the Echo's LED panel: two green round digits, then the red MM:SS.
func Face(s Session, now time.Time) Node {
	digits, rounds := s.Display(now)
	every := s.tickEvery()
	return Section(ID("face"), Class("echo"),
		If(every != "", Group{
			Attr("hx-get", fmt.Sprintf("/face?rev=%d", s.Rev)),
			Attr("hx-trigger", "every "+every),
			Attr("hx-swap", "outerHTML"),
			Attr("hx-sync", "this:drop"),
		}),
		Div(Class("bezel"),
			Div(Class("screen"),
				Div(ID("rounds"), Class("seg green"), Attr("aria-label", strings.TrimSpace(rounds)),
					Segments(rounds, "rounds"),
				),
				Div(ID("digits"), Class("seg red"), Attr("aria-label", strings.TrimSpace(digits)),
					Segments(digits, "digits"),
				),
			),
			Div(Class("bezel-mark"), Text("ROGUE")),
		),
	)
}

func Deck(s Session) Node {
	return Section(ID("deck"), Class("deck"),
		Div(Class("deck-inner"),
			Div(Class("panel-head"),
				Button(Type("button"), ID("xray"), Class("xray-toggle"),
					Attr("aria-pressed", strconv.FormatBool(s.Xray)),
					hxProgram("xray", ""),
					Text("X-RAY"),
				),
			),
			Panel(s),
			Transport(s, ""),
			P(Class("legend"), Text(transportLegend())),
			Div(Class("tape-head"),
				Span(Class("knob-label"), Text("REMOTE SIGNALS")),
				Span(Class("tape-count"), Text(fmt.Sprintf("%d KEYS", len(s.Signals())))),
			),
			Pre(Class("tape"), Text(tape(s))),
			P(Class("hardware"),
				Text("NEC address 01. Command bytes are the published GX-IR03 gym-clock capture. Clock, count-up, and countdown use that file's red, green, and yellow keys. Confirm them on your Echo remote before a blaster is wired."),
			),
		),
	)
}

func Transport(s Session, oob string) Node {
	label := "START"
	cls := "go"
	switch s.status() {
	case "paused":
		label = "RESUME"
	case "running":
		cls = "go running"
	}
	attrs := []Node{ID("transport"), Class("transport"), Attr("data-rev", strconv.Itoa(s.Rev))}
	if oob != "" {
		attrs = append(attrs, Attr("hx-swap-oob", "outerHTML:"+oob))
	}
	return Div(Group(attrs),
		Button(Type("button"), ID("go"), Class(cls),
			If(s.status() == "running", Disabled()),
			hxProgram("start", ""),
			Text(label),
		),
		Button(Type("button"), ID("stop"), Attr("hx-preserve", "true"), Class("stop"), hxProgram("stop", ""), Text("STOP")),
		Button(Type("button"), ID("reset"), Attr("hx-preserve", "true"), Class("reset"), hxProgram("reset", ""), Text("RESET")),
	)
}

func hxProgram(op, value string) Node {
	return Group{
		Attr("hx-post", "/program"),
		Attr("hx-target", "#board"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-sync", "#sync:queue all"),
		Attr("hx-vals", fmt.Sprintf(`{"op":%q,"value":%q}`, op, value)),
	}
}

func tape(s Session) string {
	lines := make([]string, len(s.Signals()))
	for i, k := range s.Signals() {
		lines[i] = k.Line()
	}
	return strings.Join(lines, "\n")
}

func transportLegend() string {
	return keyStart.Line() + "    " + keyStop.Line() + "    " + keyReset.Line() + "    " + keyClock.Line()
}

// Seven-segment LEDs. Unlit segments stay faintly visible, as on the real panel.
var (
	segOrder = []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g'}
	segPts   = map[byte]string{
		'a': "13,4 47,4 53,10 47,16 13,16 7,10",
		'b': "54,13 60,19 60,43 54,49 48,43 48,19",
		'c': "54,53 60,59 60,83 54,89 48,83 48,59",
		'd': "13,86 47,86 53,92 47,98 13,98 7,92",
		'e': "6,53 12,59 12,83 6,89 0,83 0,59",
		'f': "6,13 12,19 12,43 6,49 0,43 0,19",
		'g': "13,45 47,45 53,51 47,57 13,57 7,51",
	}
	segLit = map[rune]string{
		'0': "abcdef", '1': "bc", '2': "abged", '3': "abgcd", '4': "fgbc",
		'5': "afgcd", '6': "afgedc", '7': "abc", '8': "abcdefg", '9': "abcdfg",
		'-': "g", ' ': "",
	}
)

func Segments(text, id string) Node {
	var off, on []Node
	x := 0.0
	for _, ch := range text {
		if ch == ':' {
			on = append(on,
				El("circle", num("cx", x+10), num("cy", 32), Attr("r", "5")),
				El("circle", num("cx", x+10), num("cy", 70), Attr("r", "5")),
			)
			x += 22
			continue
		}
		lit := segLit[ch]
		for _, s := range segOrder {
			poly := El("polygon", Attr("transform", fmt.Sprintf("translate(%g 0)", x)), Attr("points", segPts[s]))
			if strings.ContainsRune(lit, rune(s)) {
				on = append(on, poly)
			} else {
				off = append(off, poly)
			}
		}
		x += 74
	}
	glow := "glow-" + id
	return El("svg",
		Attr("viewBox", fmt.Sprintf("-10 -6 %g 114", x+6)),
		Attr("aria-hidden", "true"),
		El("filter", ID(glow), Attr("x", "-20%"), Attr("y", "-20%"), Attr("width", "140%"), Attr("height", "140%"),
			El("feGaussianBlur", Attr("stdDeviation", "3.5"), Attr("result", "b")),
			El("feMerge",
				El("feMergeNode", Attr("in", "b")),
				El("feMergeNode", Attr("in", "SourceGraphic")),
			),
		),
		El("g", Class("off"), Group(off)),
		El("g", Class("on"), Attr("filter", "url(#"+glow+")"), Group(on)),
	)
}
