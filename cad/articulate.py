# FreeCAD articulate helper for cfclock panel assembly.
# Open cad/cfclock.FCStd, then:
#   exec(open("/Users/sfeldma/work/cfclock/cad/articulate.py").read())
#   articulate(fmt=120)
#
# Stack (+Z toward user):
#   knob + shaft (knob's own shaft, rear blind bore)
#   plate
#   units disk  (~0.3 mm behind plate)
#   value disk
#   hub / idler / drive gears
#   pot body with short shaft plugged into knob-shaft bore

import FreeCAD as App
import math

S = 0.5
panelW, panelH = 680 * S, 468 * S
ox, oy = panelW / 2, panelH / 2
colX0, colPitch = 172 * S, 168 * S
knobY = 276 * S
fmtX, fmtY = 340 * S, 108 * S
hubR, idlerR = 56 * S, 28 * S
names = ["count", "time", "rest"]

T_PLATE = 3.0
DISK_T = 2.0
GAP = 0.3
GEAR_T = 4.0
Z_PLATE_BOT = -T_PLATE
Z_UNITS_F = Z_PLATE_BOT - GAP
Z_VALUE_F = Z_UNITS_F - DISK_T - GAP
Z_UNITS_B = Z_VALUE_F - GAP
Z_VALUE_B = Z_UNITS_B - DISK_T - GAP
Z_HUB = Z_VALUE_B - 2.0 - GEAR_T
Z_SHAFT_BOT = Z_HUB - 1.0
KNOB_H = 12.0
Z_KNOB = 1.2


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


def articulate(fmt=None, count=None, time=None, rest=None):
    doc = App.ActiveDocument
    ss = doc.getObject("Angles")
    if fmt is not None:
        ss.set("B2", str(fmt))
    if count is not None:
        ss.set("B3", str(count))
    if time is not None:
        ss.set("B4", str(time))
    if rest is not None:
        ss.set("B5", str(rest))
    doc.recompute()
    f = float(ss.FormatDeg)
    c = float(ss.CountDeg)
    t = float(ss.TimeDeg)
    r = float(ss.RestDeg)
    idler_a = -f * (hubR / idlerR)

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

    val = {"count": c, "time": t, "rest": r}
    for k, name in enumerate(names):
        x = colX(k)
        z_u = Z_UNITS_F if k != 1 else Z_UNITS_B
        z_v = Z_VALUE_F if k != 1 else Z_VALUE_B
        vd = val[name]
        spin(doc.getObject("Shaft_" + name), x, knobY, Z_SHAFT_BOT, vd)
        spin(doc.getObject("ValueDisk_" + name), x, knobY, z_v, vd)
        spin(doc.getObject("UnitsDisk_" + name), x, knobY, z_u, f)
        spin(doc.getObject("Hub_" + name), x, knobY, Z_HUB, f)
        p = doc.getObject("Pointer_" + name)
        if p:
            rad = math.radians(vd)
            w = float(p.Width)
            base = App.Vector(sx(x), sy(knobY), Z_KNOB + KNOB_H)
            out = App.Vector(math.sin(rad) * (w * 0.15), math.cos(rad) * (w * 0.15), 0)
            p.Placement = App.Placement(base + out, App.Rotation(App.Vector(0, 0, 1), vd))
        if k > 0:
            spin(doc.getObject("Idler_" + str(k)), colX(k) - colPitch / 2, knobY, Z_HUB, idler_a)

    doc.recompute()
    return {"Format": f, "Count": c, "Time": t, "Rest": r}


if __name__ == "__main__":
    print(articulate())
