package main

import (
	"encoding/json"
	"fmt"
	"strings"

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
			Div(ID("sync")),
			Board(s),
			Script(Src("/static/clock.js"), Defer()),
		},
	})
}

func Board(s Session) Node {
	spec, err := json.Marshal(struct {
		Cues   []Cue `json:"cues"`
		Rounds int   `json:"rounds"`
	}{Cues: s.Cues(), Rounds: s.rounds()})
	if err != nil {
		spec = []byte(`{"cues":[]}`)
	}
	return Div(ID("board"),
		Face(s),
		Deck(s),
		Script(Type("application/json"), ID("spec"), Raw(string(spec))),
	)
}

// Face is the Echo's LED panel: two green round digits, then the red
// MM:SS. The digits are drawn as seven-segment LEDs by clock.js.
func Face(s Session) Node {
	return Section(Class("echo"),
		Div(Class("bezel"),
			Div(Class("screen"),
				Div(ID("rounds"), Class("seg green"), Attr("aria-label", readyRounds(s))),
				Div(ID("digits"), Class("seg red"), Attr("aria-label", readyDigits(s))),
			),
			Div(Class("bezel-mark"), Text("ROGUE")),
		),
	)
}

func Deck(s Session) Node {
	return Section(ID("deck"), Class("deck"),
		Div(Class("deck-inner"),
			Div(Class("panel-head"),
				Button(Type("button"), ID("xray"), Class("xray-toggle"), Attr("data-act", "xray"), Attr("aria-pressed", "true"), Text("X-RAY")),
			),
			Panel(s),
			Div(Class("transport"),
				Button(Type("button"), ID("go"), Class("go"), Attr("data-act", "toggle"), Text("START")),
				Button(Type("button"), ID("stop"), Class("stop"), Attr("data-act", "stop"), Text("STOP")),
				Button(Type("button"), ID("reset"), Class("reset"), Attr("data-act", "reset"), Text("RESET")),
			),
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
