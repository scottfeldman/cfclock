# FreeCAD articulate + drag handles for cfclock.
#
# Setup (once per FreeCAD session, after opening cad/cfclock.FCStd):
#   exec(open("/Users/sfeldma/work/cfclock/cad/articulate.py").read())
#
# Drag a cyan Handle_* in the 3D view:
#   1. Select Handle_format / Handle_count / Handle_time / Handle_rest
#   2. Press T (Transform) or right-click → Transform
#   3. Drag the blue rotation ring (around Z)
#
# Or from the Python console:
#   articulate(fmt=120)

import FreeCAD as App
import Part
import math

S = 0.5
panelW, panelH = 680 * S, 468 * S
ox, oy = panelW / 2, panelH / 2
colX0, colPitch = 172 * S, 168 * S
knobY = 276 * S
fmtX, fmtY = 340 * S, 108 * S
hubR, idlerR = 56 * S, 28 * S
fmtKnobR = 56 * S
knobR = 36 * S
names = ["count", "time", "rest"]

T_PLATE = 3.0
DISK_T = 2.0
GAP = 0.3
GEAR_T = 4.0
Z_PLATE_BOT = -T_PLATE
Z_UNITS_F = Z_PLATE_BOT - GAP
Z_VALUE_F = Z_UNITS_F - DISK_T - GAP
# The middle pair is a full disk thickness behind the outer value disks.
# The old stack only gapped the bottoms, so the middle units disk ran
# through the count and rest value disks.
Z_UNITS_B = Z_VALUE_F - DISK_T - GAP
Z_VALUE_B = Z_UNITS_B - DISK_T - GAP
Z_HUB = -15.2
Z_SHAFT_BOT = -15.2
KNOB_H = 12.0
Z_KNOB = 1.2
Z_HANDLE = Z_KNOB + KNOB_H + 1.0

_busy = False
_observer = None


def sx(x):
    return x - ox


def sy(y):
    return (panelH - y) - oy


def colX(k):
    return colX0 + colPitch * k


def spin(obj, x, y, z, deg):
    if obj is None:
        return
    obj.Placement = App.Placement(
        App.Vector(sx(x), sy(y), z),
        App.Rotation(App.Vector(0, 0, 1), deg),
    )


def yaw_deg(pl):
    """Angle of +Y after rotation, degrees CW from 12 o'clock (matches panel)."""
    v = pl.Rotation.multVec(App.Vector(0, 1, 0))
    return math.degrees(math.atan2(v.x, v.y)) % 360.0


def clamp(n, lo, hi):
    return lo if n < lo else hi if n > hi else n


def ensure_handles(doc):
    """Bright lever on each knob — select + Transform (T) to drag-rotate."""
    specs = [("format", fmtX, fmtY, fmtKnobR * 0.85)]
    for k, name in enumerate(names):
        specs.append((name, colX(k), knobY, knobR * 0.9))

    for name, x, y, reach in specs:
        hname = "Handle_" + name
        h = doc.getObject(hname)
        if h is None:
            # radial lever + ball grip
            arm = Part.makeBox(reach, 3.0 * S, 2.5 * S)
            arm.translate(App.Vector(0, -1.5 * S, 0))
            ball = Part.makeSphere(4.0 * S)
            ball.translate(App.Vector(reach, 0, 1.0 * S))
            solid = arm.fuse(ball)
            h = doc.addObject("Part::Feature", hname)
            h.Shape = solid
            h.ViewObject.ShapeColor = (0.1, 0.85, 0.95)
            h.ViewObject.Transparency = 20
            h.Label = "Handle " + name
        # store axis xref as properties via Label/DocumentObjectGroup — use Placement only
    doc.recompute()


def place_handle(doc, name, x, y, reach, deg):
    h = doc.getObject("Handle_" + name)
    if not h:
        return
    rad = math.radians(deg)
    # lever local +X points outward at 12 o'clock before rotate; rotate around shaft
    # tip direction at deg: (sin, cos) in FC xy
    base = App.Vector(sx(x), sy(y), Z_HANDLE)
    h.Placement = App.Placement(base, App.Rotation(App.Vector(0, 0, 1), deg - 90))
    # -90 so local +X (along arm) aligns with 12 o'clock at deg=0


def articulate(fmt=None, count=None, time=None, rest=None):
    global _busy
    doc = App.ActiveDocument
    if doc is None:
        raise RuntimeError("No active document")
    ensure_handles(doc)
    ss = doc.getObject("Angles")
    if ss is None:
        raise RuntimeError("Angles spreadsheet missing")

    _busy = True
    try:
        if fmt is not None:
            ss.set("B2", str(clamp(float(fmt), 0, 300)))
        if count is not None:
            ss.set("B3", str(float(count) % 360))
        if time is not None:
            ss.set("B4", str(float(time) % 360))
        if rest is not None:
            ss.set("B5", str(float(rest) % 360))
        doc.recompute()
        f = float(ss.FormatDeg)
        c = float(ss.CountDeg)
        t = float(ss.TimeDeg)
        r = float(ss.RestDeg)
        # Same ratio as the 28/14 tooth pair. Row idlers are half a tooth
        # out of phase with the feed idler: they mesh along X, the feed
        # idler meshes along Y. See cad/rebuild_gears.py.
        idler_a = -f * (hubR / idlerR)
        row_a = idler_a - 180.0 / 14.0

        spin(doc.getObject("Shaft_format"), fmtX, fmtY, Z_SHAFT_BOT, f)
        spin(doc.getObject("FormatDriveGear"), fmtX, fmtY, Z_HUB, f)
        spin(doc.getObject("FeedIdler"), fmtX, fmtY + hubR + idlerR, Z_HUB, idler_a)
        ptr = doc.getObject("FormatPointer")
        if ptr:
            rad = math.radians(f)
            w = float(ptr.Width)
            base = App.Vector(sx(fmtX), sy(fmtY), Z_KNOB + KNOB_H)
            out = App.Vector(math.sin(rad) * (w * 0.15), math.cos(rad) * (w * 0.15), 0)
            ptr.Placement = App.Placement(base + out, App.Rotation(App.Vector(0, 0, 1), f))
        place_handle(doc, "format", fmtX, fmtY, fmtKnobR * 0.85, f)

        val = {"count": c, "time": t, "rest": r}
        for k, name in enumerate(names):
            x = colX(k)
            z_u = Z_UNITS_F if k != 1 else Z_UNITS_B
            z_v = Z_VALUE_F if k != 1 else Z_VALUE_B
            vd = val[name]
            spin(doc.getObject("Shaft_" + name), x, knobY, Z_SHAFT_BOT, vd)
            spin(doc.getObject("ValueDisk_" + name), x, knobY, z_v, vd)
            spin(doc.getObject("VL_" + name + "_chips"), x, knobY, z_v, vd)
            spin(doc.getObject("VL_" + name + "_texts"), x, knobY, z_v + 0.2, vd)
            spin(doc.getObject("UnitsDisk_" + name), x, knobY, z_u, f)
            spin(doc.getObject("UL_" + name + "_chips"), x, knobY, z_u, f)
            spin(doc.getObject("UL_" + name + "_texts"), x, knobY, z_u + 0.2, f)
            spin(doc.getObject("Hub_" + name), x, knobY, Z_HUB, f)
            p = doc.getObject("Pointer_" + name)
            if p:
                rad = math.radians(vd)
                w = float(p.Width)
                base = App.Vector(sx(x), sy(knobY), Z_KNOB + KNOB_H)
                out = App.Vector(math.sin(rad) * (w * 0.15), math.cos(rad) * (w * 0.15), 0)
                p.Placement = App.Placement(base + out, App.Rotation(App.Vector(0, 0, 1), vd))
            place_handle(doc, name, x, knobY, knobR * 0.9, vd)
            if k > 0:
                spin(doc.getObject("Idler_" + str(k)), colX(k) - colPitch / 2, knobY, Z_HUB, row_a)

        doc.recompute()
        return {"Format": f, "Count": c, "Time": t, "Rest": r}
    finally:
        _busy = False


class _HandleObserver:
    """When a Handle_* Placement changes (Transform drag), drive articulation."""

    def slotChangedObject(self, obj, prop):
        global _busy
        if _busy or prop != "Placement":
            return
        if not obj.Name.startswith("Handle_"):
            return
        name = obj.Name[len("Handle_") :]
        ang = yaw_deg(obj.Placement)
        _busy = True
        try:
            # snap handle back onto shaft axis while keeping dragged angle
            if name == "format":
                place_handle(App.ActiveDocument, "format", fmtX, fmtY, fmtKnobR * 0.85, ang)
                articulate(fmt=ang)
            elif name in names:
                k = names.index(name)
                place_handle(App.ActiveDocument, name, colX(k), knobY, knobR * 0.9, ang)
                articulate(**{name: ang})
        finally:
            _busy = False

    def slotRedoDocument(self, doc):
        pass

    def slotUndoDocument(self, doc):
        pass


def install_drag_handles():
    global _observer
    doc = App.ActiveDocument
    if doc is None:
        raise RuntimeError("No active document")
    ensure_handles(doc)
    articulate()  # place handles at current angles
    if _observer is not None:
        try:
            App.removeDocumentObserver(_observer)
        except Exception:
            pass
    _observer = _HandleObserver()
    App.addDocumentObserver(_observer)
    print("Drag handles ready: select Handle_* → press T → rotate around Z")
    return True


if __name__ == "__main__":
    install_drag_handles()
    print(articulate())
