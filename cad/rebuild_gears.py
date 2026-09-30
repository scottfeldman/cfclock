# Rebuild the cfclock gear train in the open FreeCAD document.
#
# Involute teeth (module 2 mm, 20°) replace the trapezoid gears.
# 28-tooth gears on the format shaft and the three units hubs,
# 14-tooth idlers between them. Center distance is 42 mm, so the
# pitch circles meet and a small negative profile shift leaves
# backlash instead of a bind.
#
# Each idler turns on its own axle, retained in the back plate.
# The format gear is bored onto the format shaft. Each hub gear is
# bored clear of the value shaft it spins around.
#
# Run with the cfclock document active:
#   exec(open("/home/sfeldma/Work/cfclock/cad/rebuild_gears.py").read())

import math

import FreeCAD as App
import Part
from InvoluteGearFeature import fcgear

MODULE = 2.0
PRESSURE = 20.0
SHIFT = -0.08  # negative profile shift: backlash at the fixed 42 mm centers
THICK = 4.0
GEAR_Z = -15.2  # behind the middle value disk; see Z_HUB in articulate.py

# Tooth centers of both gears fall on local +X. A 14-tooth gear therefore
# has spaces on local +Y. The feed idler meshes along Y and needs no extra
# turn. The row idlers mesh along X, so they are turned half a circular
# pitch to put a space on the line of centers.
FEED_PHASE = 0.0
ROW_PHASE = -180.0 / 14.0

DRIVE_BORE = 4.15  # format shaft OD is 8 mm; this is a close mount
HUB_BORE = 4.60    # value shaft passes through; the hub is not fixed to it
IDLER_BORE = 2.90
AXLE_R = 2.60

BRASS = (0.72, 0.55, 0.28)
STEEL = (0.68, 0.70, 0.72)
PLATE_COLOR = (0.16, 0.16, 0.17)


def gear_blank(teeth):
    w = fcgear.FCWireBuilder()
    fcgear.involute.CreateExternalGear(
        w, MODULE, teeth, PRESSURE, True, 1.0, 1.25, 0.38, SHIFT
    )
    wire = Part.Wire([o.toShape() for o in w.wire])
    return Part.Face(wire).extrude(App.Vector(0, 0, THICK))


def bore(solid, radius, z0, height):
    return solid.cut(Part.makeCylinder(radius, height, App.Vector(0, 0, z0)))


def with_bore(solid, bore_r):
    """Bore stays inside the tooth thickness so the disk stack can sit just ahead."""
    return bore(solid, bore_r, -0.5, THICK + 1.0).removeSplitter()


def place(obj, x, y, z, deg):
    obj.Placement = App.Placement(
        App.Vector(x, y, z),
        App.Rotation(App.Vector(0, 0, 1), deg),
    )


def style(obj, color):
    obj.ViewObject.ShapeColor = color
    obj.ViewObject.Deviation = 0.02


def idler_axle(cx, cy):
    # Teeth occupy GEAR_Z .. GEAR_Z+THICK (-15.2 .. -11.2).
    head = Part.makeCylinder(6.5, 1.3, App.Vector(cx, cy, -31.5))
    journal = Part.makeCylinder(AXLE_R, 31.5 - 10.5, App.Vector(cx, cy, -31.5))
    shoulder = Part.makeCylinder(6.0, 1.25, App.Vector(cx, cy, -16.53))
    retainer = Part.makeCylinder(5.5, 0.7, App.Vector(cx, cy, -11.12))
    axle = head.fuse(journal).fuse(shoulder).fuse(retainer)
    return axle.removeSplitter()


def rebuild():
    doc = App.ActiveDocument
    if doc is None or doc.Name != "cfclock":
        raise RuntimeError("Open cad/cfclock.FCStd first")

    ss = doc.getObject("Angles")
    fmt = float(ss.FormatDeg)

    hub = with_bore(gear_blank(28), HUB_BORE)
    drive = with_bore(gear_blank(28), DRIVE_BORE)
    idler = bore(gear_blank(14), IDLER_BORE, -0.5, THICK + 1.0).removeSplitter()

    specs = [
        ("FormatDriveGear", drive, 0, 63, fmt),
        ("FeedIdler", idler, 0, 21, -2 * fmt + FEED_PHASE),
        ("Hub_count", hub, -84, -21, fmt),
        ("Hub_time", hub, 0, -21, fmt),
        ("Hub_rest", hub, 84, -21, fmt),
        ("Idler_1", idler, -42, -21, -2 * fmt + ROW_PHASE),
        ("Idler_2", idler, 42, -21, -2 * fmt + ROW_PHASE),
    ]
    for name, shape, x, y, deg in specs:
        obj = doc.getObject(name)
        obj.Shape = shape.copy()
        place(obj, x, y, GEAR_Z, deg)
        style(obj, BRASS)

    for name in ("BackPlate", "Axle_feed", "Axle_idler_1", "Axle_idler_2"):
        old = doc.getObject(name)
        if old is not None:
            doc.removeObject(name)

    # Frame rather than a solid wall: a border, a spine under the format
    # shaft, and a rail under the three value shafts. The openings leave
    # the gears visible from the back.
    plate = Part.makeBox(340, 234, 3.0, App.Vector(-170, -117, -30.2))
    openings = (
        (-154, -7, 140, 108),
        (-154, -101, 140, 66),
        (14, -7, 140, 108),
        (14, -101, 140, 66),
    )
    for x, y, w, h in openings:
        plate = plate.cut(Part.makeBox(w, h, 5.0, App.Vector(x, y, -31.0)))
    for cx, cy in ((0, 63), (-84, -21), (0, -21), (84, -21)):
        seat = Part.makeCylinder(12.0, 0.6, App.Vector(cx, cy, -27.2))
        plate = plate.fuse(seat)
    for cx, cy in ((0, 21), (-42, -21), (42, -21)):
        plate = plate.cut(Part.makeCylinder(3.05, 5.0, App.Vector(cx, cy, -30.6)))
        boss = Part.makeCylinder(7.0, 1.6, App.Vector(cx, cy, -27.2))
        boss = boss.cut(Part.makeCylinder(3.05, 2.2, App.Vector(cx, cy, -27.4)))
        plate = plate.fuse(boss)
    plate = plate.removeSplitter()

    back = doc.addObject("Part::Feature", "BackPlate")
    back.Shape = plate
    back.Label = "Back plate"
    style(back, PLATE_COLOR)

    axles = [
        ("Axle_feed", "Feed axle", 0, 21),
        ("Axle_idler_1", "Count idler axle", -42, -21),
        ("Axle_idler_2", "Rest idler axle", 42, -21),
    ]
    for name, label, cx, cy in axles:
        obj = doc.addObject("Part::Feature", name)
        obj.Shape = idler_axle(cx, cy)
        obj.Label = label
        style(obj, STEEL)

    for name in ("Shaft_format", "Shaft_count", "Shaft_time", "Shaft_rest"):
        doc.getObject(name).Visibility = True

    doc.recompute()
    return {"format_deg": fmt, "feed_phase": FEED_PHASE, "row_phase": ROW_PHASE}


rebuild()
