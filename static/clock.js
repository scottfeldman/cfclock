// Plays the cue list and paints the Echo face: green rounds, red time.
// Programming is htmx. START / STOP / RESET are the remote's run keys.
(function () {
  const state = {
    status: "ready",
    cues: [],
    roundsTotal: 0,
    index: 0,
    origin: 0,
    elapsed: 0,
    lastSecond: null,
  };
  let painted = {};

  function arm() {
    painted = {};
    const el = document.getElementById("spec");
    state.cues = [];
    state.roundsTotal = 0;
    if (el) {
      try {
        const spec = JSON.parse(el.textContent);
        state.cues = spec.cues || [];
        state.roundsTotal = spec.rounds || 0;
      } catch (err) {
        state.cues = [];
      }
    }
    state.index = 0;
    state.elapsed = 0;
    state.origin = 0;
    state.status = "ready";
    state.lastSecond = null;
    paint();
  }

  function cue() { return state.cues[state.index]; }

  function elapsedNow() {
    if (state.status === "running") return performance.now() - state.origin;
    return state.elapsed;
  }

  function shownSeconds(c, elapsedMs) {
    if (!c || c.count === "clock") return 0;
    if (state.status === "done" && c.count === "down") return 0;
    if (c.count === "down") {
      const left = c.seconds * 1000 - elapsedMs;
      if (left <= 0) return 0;
      return Math.ceil(left / 1000);
    }
    let ms = elapsedMs;
    if (c.seconds > 0 && ms > c.seconds * 1000) ms = c.seconds * 1000;
    return Math.floor(ms / 1000);
  }

  function formatLED(sec) {
    sec = Math.max(0, sec | 0);
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
  }

  function wall() {
    const d = new Date();
    let h = d.getHours() % 12;
    if (h === 0) h = 12;
    return String(h).padStart(2, " ") + ":" + String(d.getMinutes()).padStart(2, "0");
  }

  // Seven-segment LEDs. Unlit segments stay faintly visible, as on the
  // real panel.
  const SEG = {
    a: "13,4 47,4 53,10 47,16 13,16 7,10",
    b: "54,13 60,19 60,43 54,49 48,43 48,19",
    c: "54,53 60,59 60,83 54,89 48,83 48,59",
    d: "13,86 47,86 53,92 47,98 13,98 7,92",
    e: "6,53 12,59 12,83 6,89 0,83 0,59",
    f: "6,13 12,19 12,43 6,49 0,43 0,19",
    g: "13,45 47,45 53,51 47,57 13,57 7,51",
  };
  const LIT = {
    "0": "abcdef", "1": "bc", "2": "abged", "3": "abgcd", "4": "fgbc",
    "5": "afgcd", "6": "afgedc", "7": "abc", "8": "abcdefg", "9": "abcdfg",
    "-": "g", " ": "",
  };

  function segSVG(text, id) {
    let x = 0;
    let off = "";
    let on = "";
    for (const ch of text) {
      if (ch === ":") {
        on += '<circle cx="' + (x + 10) + '" cy="32" r="5"/>' +
          '<circle cx="' + (x + 10) + '" cy="70" r="5"/>';
        x += 22;
        continue;
      }
      const lit = LIT[ch] || "";
      for (const s in SEG) {
        const poly = '<polygon transform="translate(' + x + ' 0)" points="' + SEG[s] + '"/>';
        if (lit.indexOf(s) >= 0) on += poly; else off += poly;
      }
      x += 74;
    }
    const glow = "glow-" + id;
    return '<svg viewBox="-10 -6 ' + (x + 6) + ' 114" aria-hidden="true">' +
      '<filter id="' + glow + '" x="-20%" y="-20%" width="140%" height="140%">' +
      '<feGaussianBlur stdDeviation="3.5" result="b"/>' +
      '<feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge></filter>' +
      '<g class="off">' + off + '</g><g class="on" filter="url(#' + glow + ')">' + on + "</g></svg>";
  }

  function setSeg(id, text) {
    if (painted[id] === text) return;
    painted[id] = text;
    const el = document.getElementById(id);
    if (!el) return;
    el.innerHTML = segSVG(text, id);
    el.setAttribute("aria-label", text.trim());
  }

  // Beeps: short on 3, 2, 1 and long when a piece starts or ends.
  let audio = null;
  function beep(long) {
    if (!audio) return;
    const t = audio.currentTime;
    const o = audio.createOscillator();
    const g = audio.createGain();
    o.type = "square";
    o.frequency.value = long ? 1180 : 880;
    g.gain.setValueAtTime(0.0001, t);
    g.gain.exponentialRampToValueAtTime(0.18, t + 0.01);
    g.gain.exponentialRampToValueAtTime(0.0001, t + (long ? 0.7 : 0.14));
    o.connect(g).connect(audio.destination);
    o.start(t);
    o.stop(t + (long ? 0.72 : 0.16));
  }

  function paint() {
    const c = cue();
    const go = document.getElementById("go");
    if (go) {
      const label = state.status === "paused" ? "RESUME" : "START";
      if (go.textContent !== label) go.textContent = label;
      go.disabled = state.status === "running";
      go.classList.remove("running");
    }
    if (!c) return;
    if (c.count === "clock") {
      setSeg("digits", wall());
      setSeg("rounds", "  ");
      document.title = "Echo Clock";
      return;
    }
    const shown = state.status === "ready" && c.count === "up" ? 0 : shownSeconds(c, elapsedNow());
    setSeg("digits", formatLED(state.status === "ready" && c.count === "down" ? c.seconds : shown));
    let roundText = "  ";
    if (c.round) {
      const n = state.status === "ready" && state.roundsTotal ? state.roundsTotal : c.round;
      roundText = String(n).padStart(2, "0");
    }
    setSeg("rounds", roundText);
    if (state.status === "running" && shown !== state.lastSecond) {
      if (c.count === "down" && shown >= 1 && shown <= 3) beep(false);
      state.lastSecond = shown;
    }
    document.title = (state.status === "running" || state.status === "paused")
      ? formatLED(shown) + "  Echo"
      : "Echo Controller";
  }

  function advance() {
    let elapsed = performance.now() - state.origin;
    let guard = 0;
    while (guard++ < 40 && state.status === "running") {
      const c = cue();
      if (!c || c.count === "clock") break;
      const dur = c.seconds * 1000;
      if (!(dur > 0) || elapsed < dur) break;
      elapsed -= dur;
      state.origin = performance.now() - elapsed;
      state.lastSecond = null;
      beep(true);
      if (state.index + 1 >= state.cues.length) {
        state.elapsed = dur;
        state.status = "done";
        break;
      }
      state.index += 1;
    }
  }

  function halt() {
    if (state.status !== "running") return;
    state.elapsed = performance.now() - state.origin;
    state.status = "paused";
    paint();
  }

  function start() {
    const c = cue();
    if (!c || c.count === "clock") return;
    if (state.status === "running") return;
    if (state.status === "done") arm();
    if (!audio && window.AudioContext) audio = new AudioContext();
    if (audio && audio.state === "suspended") audio.resume();
    if (state.status === "ready") {
      state.elapsed = 0;
      state.index = 0;
      state.lastSecond = null;
      beep(true);
    }
    state.origin = performance.now() - state.elapsed;
    state.status = "running";
    paint();
  }

  // Knobs. Every detent is one POST; they are chained so fast turns land
  // in order.
  let queue = Promise.resolve();
  function send(op, value) {
    queue = queue.then(function () {
      return htmx.ajax("POST", "/program", {
        target: "#board",
        swap: "outerHTML",
        values: { op: op, value: value },
      });
    }).catch(function () {});
  }
  function turn(knob, dir) { send("turn", knob + ":" + (dir > 0 ? "+" : "-")); }

  function xray(on) {
    document.body.classList.toggle("xray", on);
    const b = document.getElementById("xray");
    if (b) b.setAttribute("aria-pressed", String(on));
  }
  xray(localStorage.getItem("xray") !== "off");

  let drag = null;

  function knobCenter(name) {
    const hit = document.querySelector('[data-knob="' + name + '"] .knob-hit');
    if (!hit) return null;
    const r = hit.getBoundingClientRect();
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
  }
  function angleAt(c, e) { return Math.atan2(e.clientX - c.x, c.y - e.clientY) * 180 / Math.PI; }

  document.addEventListener("pointerdown", function (e) {
    const k = e.target.closest && e.target.closest("[data-knob]");
    if (!k) {
      focusedKnob = null;
      return;
    }
    const name = k.dataset.knob;
    const c = knobCenter(name);
    if (!c) return;
    drag = { name: name, step: parseFloat(k.dataset.detent) || 24, last: angleAt(c, e), acc: 0, moved: false };
  });
  window.addEventListener("pointermove", function (e) {
    if (!drag) return;
    const c = knobCenter(drag.name);
    if (!c) return;
    const a = angleAt(c, e);
    let d = a - drag.last;
    if (d > 180) d -= 360;
    if (d < -180) d += 360;
    drag.last = a;
    drag.acc += d;
    const step = drag.step;
    while (Math.abs(drag.acc) >= step) {
      const dir = drag.acc > 0 ? 1 : -1;
      turn(drag.name, dir);
      drag.acc -= dir * step;
      drag.moved = true;
    }
  });
  window.addEventListener("pointerup", function () {
    if (drag) setTimeout(function () { drag = null; }, 0);
  });

  let wheelAt = 0;
  document.addEventListener("wheel", function (e) {
    const k = e.target.closest && e.target.closest("[data-knob]");
    if (!k) return;
    e.preventDefault();
    const now = performance.now();
    if (now - wheelAt < 70 || e.deltaY === 0) return;
    wheelAt = now;
    turn(k.dataset.knob, e.deltaY > 0 ? 1 : -1);
  }, { passive: false });

  document.addEventListener("click", function (e) {
    if (!e.target.closest) return;
    const jump = e.target.closest("[data-format]");
    if (jump) {
      if (!(drag && drag.moved)) send("format", jump.dataset.format);
      return;
    }
    const t = e.target.closest("[data-turn]");
    if (t) {
      const parts = t.dataset.turn.split(":");
      turn(parts[0], parts[1] === "+" ? 1 : -1);
      return;
    }
    const btn = e.target.closest("[data-act]");
    if (!btn) return;
    const act = btn.dataset.act;
    if (act === "toggle") start();
    else if (act === "stop") halt();
    else if (act === "reset") arm();
    else if (act === "xray") {
      const on = !document.body.classList.contains("xray");
      localStorage.setItem("xray", on ? "on" : "off");
      xray(on);
    }
  });

  // A swap removes the focused knob for a moment and focus falls to the
  // body; keys pressed in that gap still belong to the knob.
  let focusedKnob = null;
  document.addEventListener("focusin", function (e) {
    const k = e.target.closest && e.target.closest("[data-knob]");
    focusedKnob = k ? k.dataset.knob : null;
  });

  document.addEventListener("keydown", function (e) {
    if (e.target && e.target.closest && e.target.closest("input, textarea")) return;
    const k = e.target.closest && e.target.closest("[data-knob]");
    const knob = k ? k.dataset.knob : (e.target === document.body ? focusedKnob : null);
    if (knob) {
      const dir = { ArrowRight: 1, ArrowUp: 1, ArrowLeft: -1, ArrowDown: -1 }[e.key];
      if (dir) {
        e.preventDefault();
        turn(knob, dir);
        return;
      }
    }
    const t = e.target.closest && e.target.closest("[data-turn]");
    if (t && (e.key === "Enter" || e.code === "Space")) {
      e.preventDefault();
      const parts = t.dataset.turn.split(":");
      turn(parts[0], parts[1] === "+" ? 1 : -1);
      return;
    }
    if (e.code === "Space") {
      e.preventDefault();
      if (state.status === "running") halt();
      else start();
    } else if (e.key === "r" || e.key === "R") {
      arm();
    }
  });

  document.addEventListener("htmx:afterSwap", function () {
    if (document.getElementById("spec")) arm();
    xray(document.body.classList.contains("xray"));
  });

  arm();
  requestAnimationFrame(function frame() {
    if (state.status === "running") advance();
    paint();
    requestAnimationFrame(frame);
  });
})();
