package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Panel geometry, in SVG user units.
//
// Each value knob carries a value disk. In front of it, on a sleeve
// around the knob shaft, is a units disk: the units are printed on it and
// its cut-outs are the shutter over the value disk. The plate has two
// windows under each knob, one for the units and one for the value.
//
// The disks are wider than the column pitch, so the middle column's pair
// sits one layer behind the outer columns' pairs and they overlap without
// touching. The windows are below the knobs, where no neighbor reaches.
//
// A hub gear on each units sleeve turns it. The hubs, the drive gear on
// the format shaft, and the idlers between them are all the same pitch,
// and each hub is the same size as the drive gear, so every units disk
// turns 60° per format detent, the same way as the format knob.
const (
	panelW     = 680.0
	panelH     = 468.0
	colX0      = 172.0
	colPitch   = 168.0
	knobY      = 276.0
	knobR      = 36.0
	diskR      = 152.0
	valueR     = 132.0
	unitR      = 98.0
	valueWinH  = 34.0
	unitWinW   = 80.0
	unitWinH   = 26.0
	shutterIn  = 114.0
	shutterOut = 150.0
	labelY     = knobY + 172
	fmtX       = 340.0
	fmtY       = 108.0
	fmtKnobR   = 56.0
	fmtLabelR  = 82.0
	hubR       = 56.0
	hubTeeth   = 28
	idlerR     = 28.0
	idlerTeeth = 14
)

func columnX(k int) float64 { return colX0 + colPitch*float64(k) }

// valueWinW is one stop of knob k's disk, so its neighbors stay hidden.
func valueWinW(k int) float64 {
	_, n, _ := diskSpec(Session{}, k)
	return min(58, valueR*detent(n)*math.Pi/180-1)
}

// frontLayer is true for the outer columns, whose disks sit in front of
// the middle column's.
func frontLayer(k int) bool { return k != 1 }

func Panel(s Session) Node {
	disks := []Node{}
	for _, front := range []bool{false, true} {
		for k := range valueKnobs {
			if frontLayer(k) == front {
				disks = append(disks, valueDisk(s, k), unitsDisk(s, k))
			}
		}
	}
	return Div(Class("panel-wrap"),
		El("svg",
			Class("panel"),
			Attr("viewBox", fmt.Sprintf("0 0 %g %g", panelW, panelH)),
			Attr("role", "group"),
			Attr("aria-label", "Controller panel"),
			panelDefs(),
			rect(0, 0, panelW, panelH, Class("cavity")),
			Group(disks),
			gearTrain(s),
			rect(0, 0, panelW, panelH, Class("plate"), Attr("mask", "url(#holes)")),
			frames(s),
			formatKnob(s),
			Group(valueKnobNodes(s)),
		),
	)
}

func panelDefs() Node {
	holes := []Node{rect(0, 0, panelW, panelH, Attr("fill", "white"))}
	for k := range valueKnobs {
		x := columnX(k)
		holes = append(holes,
			rect(x-unitWinW/2, knobY+unitR-unitWinH/2, unitWinW, unitWinH, Attr("fill", "black")),
			rect(x-valueWinW(k)/2, knobY+valueR-valueWinH/2, valueWinW(k), valueWinH, Attr("fill", "black")),
		)
	}
	return El("defs",
		El("mask", ID("holes"), Group(holes)),
		El("radialGradient", ID("cap"), Attr("cx", "38%"), Attr("cy", "30%"),
			El("stop", Attr("offset", "0"), Attr("stop-color", "#4a4a4a")),
			El("stop", Attr("offset", "1"), Attr("stop-color", "#161616")),
		),
	)
}

// gearTrain runs from the drive gear on the format shaft, through one
// idler, to the middle hub, and along the row through an idler between
// each pair of hubs.
func gearTrain(s Session) Node {
	a := float64(s.Dial) * 60
	gear := func(id, kind string, cx, cy, r float64, n int, angle float64) Node {
		return El("g",
			ID("gear-"+id),
			Class("gear "+kind),
			Style(rotate(cx, cy, angle)),
			El("path", Class("teeth"), Attr("d", gearPath(cx, cy, r, n))),
			El("circle", Class("hub"), cxy(cx, cy), Attr("r", "6")),
		)
	}
	nodes := []Node{
		gear("drive", "drive", fmtX, fmtY, hubR, hubTeeth, a),
		gear("feed", "idler", fmtX, fmtY+hubR+idlerR, idlerR, idlerTeeth, -a*hubR/idlerR+180.0/idlerTeeth),
	}
	for k, name := range valueKnobs {
		nodes = append(nodes, gear(name, "units", columnX(k), knobY, hubR, hubTeeth, a))
		if k > 0 {
			nodes = append(nodes, gear("idler-"+strconv.Itoa(k), "idler", columnX(k)-colPitch/2, knobY, idlerR, idlerTeeth, -a*hubR/idlerR))
		}
	}
	return El("g", Class("train"), Group(nodes))
}

// valueDisk is fixed to knob k. Stop p is printed at -p·detent, so it is
// under the value window when the knob is on p.
func valueDisk(s Session, k int) Node {
	x := columnX(k)
	name := valueKnobs[k]
	pos, n, label := diskSpec(s, k)
	d := detent(n)
	texts := []Node{}
	for p := range n {
		cls := "digit"
		if p == pos {
			cls += " on"
		}
		rot := fmt.Sprintf("rotate(%g %g %g)", -float64(p)*d, x, knobY)
		w := valueWinW(k)
		texts = append(texts,
			rect(x-w/2, knobY+valueR-valueWinH/2, w, valueWinH, Class("chip"), Attr("transform", rot)),
			El("text",
				Class(cls),
				num("x", x), num("y", knobY+valueR),
				Attr("transform", rot),
				Text(label(p)),
			),
		)
	}
	sx, sy := polar(x, knobY, valueR-12, 180+d)
	ex, ey := polar(x, knobY, valueR+12, 180+d)
	return El("g",
		ID("vdisk-"+name),
		Class("disk"),
		Style(rotate(x, knobY, float64(pos)*d)),
		El("circle", Class("disk-face"), cxy(x, knobY), num("r", diskR)),
		El("line", Class("stop-pin"), num("x1", sx), num("y1", sy), num("x2", ex), num("y2", ey)),
		Group(texts),
	)
}

// unitsDisk rides on the knob's sleeve and turns with the format train.
// Format f's units are printed at -60f so they reach the units window when
// the dial is on f.
func unitsDisk(s Session, k int) Node {
	x := columnX(k)
	cur := mod(s.Dial, len(formats))
	labels := []Node{}
	for f, fm := range formats {
		label := unitTable[fm.ID][k]
		if label == "" {
			continue
		}
		cls := "unit"
		if f == cur {
			cls += " on"
		}
		rot := fmt.Sprintf("rotate(%d %g %g)", -60*f, x, knobY)
		labels = append(labels,
			rect(x-unitWinW/2, knobY+unitR-unitWinH/2, unitWinW, unitWinH, Class("chip"), Attr("transform", rot)),
			El("text",
				Class(cls),
				num("x", x), num("y", knobY+unitR),
				Attr("transform", rot),
				Text(label),
			),
		)
	}
	return El("g",
		ID("udisk-"+valueKnobs[k]),
		Class("disk units-disk"),
		Style(rotate(x, knobY, float64(s.Dial)*60)),
		El("path", Class("shutter"), Attr("fill-rule", "evenodd"), Attr("d", shutterPath(x, knobY, k))),
		Group(labels),
	)
}

// shutterPath is the units disk with an opening for every format that uses
// knob k. Format f is cut at 180-60f, so it sits over the value window when
// the dial is on f. Neighboring used formats are cut as one opening.
func shutterPath(cx, cy float64, k int) string {
	n := len(formats)
	used := make([]bool, n)
	start := -1
	for f, fm := range formats {
		used[f] = unitTable[fm.ID][k] != ""
		if !used[f] && start < 0 {
			start = f
		}
	}
	var b strings.Builder
	arc := func(r, a0, a1 float64, sweep int) {
		x, y := polar(cx, cy, r, a1)
		large := 0
		if math.Abs(a1-a0) > 180 {
			large = 1
		}
		fmt.Fprintf(&b, "A%g %g 0 %d %d %.2f %.2f", r, r, large, sweep, x, y)
	}
	circle := func(r float64) {
		x, y := polar(cx, cy, r, 0)
		fmt.Fprintf(&b, "M%.2f %.2f", x, y)
		arc(r, 0, 180, 1)
		arc(r, 180, 360, 1)
		b.WriteString("Z")
	}
	circle(diskR)
	if start < 0 {
		circle(shutterOut)
		circle(shutterIn)
		return b.String()
	}
	for i := 1; i <= n; i++ {
		f := (start + i) % n
		if !used[f] || used[(f+n-1)%n] {
			continue
		}
		last := f
		for used[(last+1)%n] && (last+1)%n != f {
			last = (last + 1) % n
		}
		span := float64(mod(last-f, n)+1) * 60
		a0 := 180 - 60*float64(f) + 30
		a1 := a0 - span
		x, y := polar(cx, cy, shutterOut, a1)
		fmt.Fprintf(&b, "M%.2f %.2f", x, y)
		arc(shutterOut, a1, a0, 1)
		x, y = polar(cx, cy, shutterIn, a0)
		fmt.Fprintf(&b, "L%.2f %.2f", x, y)
		arc(shutterIn, a0, a1, 0)
		b.WriteString("Z")
	}
	return b.String()
}

// detent spaces n stops around a disk with one blank slot for the end stop.
func detent(n int) float64 { return 360 / float64(n+1) }

func diskSpec(s Session, k int) (pos, n int, label func(int) string) {
	switch valueKnobs[k] {
	case "count":
		return s.Count, len(countStops), func(p int) string { return strconv.Itoa(countStops[p]) }
	case "time":
		return s.Time, len(timeStops), func(p int) string { return formatLoose(timeStops[p]) }
	default:
		return s.Rest, len(restStops), func(p int) string { return formatLoose(restStops[p]) }
	}
}

func frames(s Session) Node {
	nodes := []Node{}
	for k, name := range valueKnobs {
		x := columnX(k)
		cls := "frame"
		if s.Units(k) == "" {
			cls += " blank"
		}
		nodes = append(nodes,
			rect(x-unitWinW/2, knobY+unitR-unitWinH/2, unitWinW, unitWinH, Class(cls)),
			rect(x-valueWinW(k)/2, knobY+valueR-valueWinH/2, valueWinW(k), valueWinH, Class(cls)),
			El("text", Class("plate-label"), num("x", x), num("y", labelY), Text(strings.ToUpper(name))),
		)
	}
	return Group(nodes)
}

// formatKnob is a pointer knob. The formats are printed on the plate
// around it at 60f.
func formatKnob(s Session) Node {
	cur := mod(s.Dial, len(formats))
	marks := []Node{}
	for f, fm := range formats {
		a := 60 * float64(f)
		cls := "flabel"
		if f == cur {
			cls += " on"
		}
		x, y := polar(fmtX, fmtY, fmtLabelR, a)
		anchor := "middle"
		switch {
		case a > 0 && a < 180:
			anchor, x = "start", fmtX+fmtLabelR*0.8
		case a > 180:
			anchor, x = "end", fmtX-fmtLabelR*0.8
		}
		tx1, ty1 := polar(fmtX, fmtY, fmtKnobR+4, a)
		tx2, ty2 := polar(fmtX, fmtY, fmtKnobR+11, a)
		marks = append(marks,
			El("line", Class("tick"), num("x1", tx1), num("y1", ty1), num("x2", tx2), num("y2", ty2)),
			El("text", Class(cls), Attr("data-format", fm.ID), Style("text-anchor: "+anchor), num("x", x), num("y", y), Text(fm.Label)),
		)
	}
	ridges := make([]Node, 12)
	for i := range ridges {
		a := float64(i) * 30
		x1, y1 := polar(fmtX, fmtY, fmtKnobR-10, a)
		x2, y2 := polar(fmtX, fmtY, fmtKnobR, a)
		ridges[i] = El("line", Class("ridge"), num("x1", x1), num("y1", y1), num("x2", x2), num("y2", y2))
	}
	px1, py1 := polar(fmtX, fmtY, 14, 0)
	px2, py2 := polar(fmtX, fmtY, fmtKnobR-14, 0)
	return Group{
		Group(marks),
		El("text", Class("plate-label"), num("x", 90), num("y", fmtY), Text("FORMAT")),
		knobShell("format", "Format knob", formats[cur].Label, 60, fmtX, fmtY, fmtKnobR+4,
			El("g", ID("rotor-format"), Class("rotor"), Style(rotate(fmtX, fmtY, float64(s.Dial)*60)),
				El("circle", Class("cap"), cxy(fmtX, fmtY), num("r", fmtKnobR)),
				Group(ridges),
				El("line", Class("pointer"), num("x1", px1), num("y1", py1), num("x2", px2), num("y2", py2)),
			),
		),
		turnButtons("format", fmtX, fmtY+fmtLabelR, 150),
	}
}

func valueKnobNodes(s Session) []Node {
	out := []Node{}
	for k, name := range valueKnobs {
		x := columnX(k)
		pos, n, label := diskSpec(s, k)
		d := detent(n)
		units := s.Units(k)
		text := label(pos)
		if units != "" {
			text += " " + strings.ToLower(units)
		} else {
			text += ", not used"
		}
		knurl := make([]Node, 24)
		for i := range knurl {
			a := float64(i) * 15
			x1, y1 := polar(x, knobY, knobR-6, a)
			x2, y2 := polar(x, knobY, knobR, a)
			knurl[i] = El("line", Class("ridge"), num("x1", x1), num("y1", y1), num("x2", x2), num("y2", y2))
		}
		px1, py1 := polar(x, knobY, 8, 0)
		px2, py2 := polar(x, knobY, knobR-9, 0)
		pointer := "pointer"
		if units == "" {
			pointer += " idle"
		}
		out = append(out,
			knobShell(name, strings.ToUpper(name[:1])+name[1:]+" knob", text, d, x, knobY, knobR+4,
				El("g", ID("rotor-"+name), Class("rotor"), Style(rotate(x, knobY, float64(pos)*d)),
					El("circle", Class("cap"), cxy(x, knobY), num("r", knobR)),
					Group(knurl),
					El("line", Class(pointer), num("x1", px1), num("y1", py1), num("x2", px2), num("y2", py2)),
				),
			),
			turnButtons(name, x, knobY+30, 52),
		)
	}
	return out
}

func knobShell(name, label, value string, detent, x, y, r float64, rotor Node) Node {
	return El("g",
		ID("knob-"+name),
		Class("knob"),
		Attr("data-knob", name),
		num("data-detent", detent),
		Attr("tabindex", "0"),
		Attr("role", "slider"),
		Attr("aria-label", label),
		Attr("aria-valuetext", value),
		El("circle", Class("knob-hit"), cxy(x, y), num("r", r)),
		rotor,
	)
}

func turnButtons(name string, x, y, dx float64) Node {
	btn := func(dir, glyph, label string, bx float64) Node {
		return El("g",
			Class("turn"),
			Attr("data-turn", name+":"+dir),
			Attr("role", "button"),
			Attr("tabindex", "0"),
			Attr("aria-label", label),
			El("circle", cxy(bx, y), Attr("r", "13")),
			El("text", num("x", bx), num("y", y), Text(glyph)),
		)
	}
	return Group{
		btn("-", "↺", "Turn "+name+" counterclockwise", x-dx),
		btn("+", "↻", "Turn "+name+" clockwise", x+dx),
	}
}

func gearPath(cx, cy, r float64, n int) string {
	ra, rr := r+4, r-4.5
	step := 360.0 / float64(n)
	var b strings.Builder
	for k := 0; k < n; k++ {
		a := float64(k) * step
		pts := [][2]float64{
			{rr, a - step/2},
			{rr, a - step*0.28},
			{ra, a - step*0.15},
			{ra, a + step*0.15},
			{rr, a + step*0.28},
		}
		for i, p := range pts {
			x, y := polar(cx, cy, p[0], p[1])
			if k == 0 && i == 0 {
				fmt.Fprintf(&b, "M%.2f %.2f", x, y)
			} else {
				fmt.Fprintf(&b, "L%.2f %.2f", x, y)
			}
		}
	}
	b.WriteString("Z")
	return b.String()
}

// polar measures a clockwise from 12 o'clock.
func polar(cx, cy, r, a float64) (float64, float64) {
	rad := a * math.Pi / 180
	return cx + r*math.Sin(rad), cy - r*math.Cos(rad)
}

func rotate(cx, cy, deg float64) string {
	return fmt.Sprintf("transform-origin: %gpx %gpx; transform: rotate(%gdeg)", cx, cy, deg)
}

func rect(x, y, w, h float64, attrs ...Node) Node {
	return El("rect", num("x", x), num("y", y), num("width", w), num("height", h), Group(attrs))
}

func cxy(x, y float64) Node { return Group{num("cx", x), num("cy", y)} }

func num(name string, v float64) Node {
	return Attr(name, strconv.FormatFloat(v, 'f', -1, 64))
}
