# Lay printable parts out on 10 inch plates and write one STEP file per group.
# Run: freecadcmd cad/print/make_plates.py

import os

import FreeCAD as App
import Import
import Part

SRC = "/home/sfeldma/Work/cfclock/cad/cfclock.FCStd"
OUT = "/home/sfeldma/Work/cfclock/cad/print"
BED = 10 * 25.4  # 254 mm
MARGIN = 8.0
GAP = 6.0

os.makedirs(OUT, exist_ok=True)
log_lines = []


def log(msg):
    print(msg)
    log_lines.append(msg)


def copied(doc, name):
    return doc.getObject(name).Shape.copy()


def seat_mark(disk_bb, solid):
    """Embed a marking 0.05 mm into the disk face it already sits on."""
    s = solid.copy()
    bb = s.BoundBox
    top, bot = disk_bb.ZMax, disk_bb.ZMin
    if bb.ZMin >= top - 0.45:
        s.translate(App.Vector(0, 0, top - bb.ZMin - 0.05))
    elif bb.ZMax <= bot + 0.45:
        s.translate(App.Vector(0, 0, bot - bb.ZMax + 0.05))
    return s


def fuse_parts(disk_name, mark_names, doc):
    disk = copied(doc, disk_name)
    marks = []
    bb = disk.BoundBox
    for name in mark_names:
        for solid in copied(doc, name).Solids:
            marks.append(seat_mark(bb, solid))
    solids = list(disk.Solids) + marks
    acc = solids[0]
    for solid in solids[1:]:
        acc = acc.fuse(solid)
    log(f"  {disk_name}: {len(solids)} pieces -> {len(acc.Solids)} solid(s)")
    return acc


def mass_z(shape):
    solids = shape.Solids or [shape]
    weight = 0.0
    moment = 0.0
    for solid in solids:
        weight += solid.Volume
        moment += solid.Volume * solid.CenterOfMass.z
    if weight <= 0:
        bb = shape.BoundBox
        return (bb.ZMin + bb.ZMax) / 2.0
    return moment / weight


def on_bed(shape, flip_heavy=False):
    s = shape.copy()
    if flip_heavy:
        bb = s.BoundBox
        if mass_z(s) > (bb.ZMin + bb.ZMax) / 2.0:
            s.rotate(App.Vector(0, 0, 0), App.Vector(1, 0, 0), 180)
    bb = s.BoundBox
    s.translate(App.Vector(-bb.XMin, -bb.YMin, -bb.ZMin))
    return s


def arrange(named):
    """Pack shapes that already sit on z=0 with their min corner at the origin."""
    x = MARGIN
    y = MARGIN
    row_h = 0.0
    placed = []
    for name, shape in named:
        bb = shape.BoundBox
        w, h = bb.XLength, bb.YLength
        if x + w > BED - MARGIN:
            x = MARGIN
            y += row_h + GAP
            row_h = 0.0
        if w + 2 * MARGIN > BED or h + 2 * MARGIN > BED or y + h > BED - MARGIN:
            raise RuntimeError(f"{name} does not fit on a {BED:.0f} mm plate ({w:.1f} x {h:.1f})")
        shape.translate(App.Vector(x - bb.XMin, y - bb.YMin, -bb.ZMin))
        placed.append((name, shape))
        x += w + GAP
        row_h = max(row_h, h)
    # Center the group on the plate.
    box = placed[0][1].BoundBox
    for _, shape in placed[1:]:
        box.add(shape.BoundBox)
    dx = (BED - (box.XMax - box.XMin)) / 2.0 - box.XMin
    dy = (BED - (box.YMax - box.YMin)) / 2.0 - box.YMin
    for _, shape in placed:
        shape.translate(App.Vector(dx, dy, 0))
    return placed


def export_plate(filename, named):
    path = os.path.join(OUT, filename)
    bed = App.newDocument("bed")
    for name, shape in named:
        obj = bed.addObject("Part::Feature", name)
        obj.Label = name
        obj.Shape = shape
    bed.recompute()
    Import.export(bed.Objects, path)
    App.closeDocument(bed.Name)
    bb = named[0][1].BoundBox
    for _, shape in named[1:]:
        bb.add(shape.BoundBox)
    log(
        f"{filename}: {len(named)} part(s), "
        f"span {bb.XLength:.1f} x {bb.YLength:.1f} x {bb.ZLength:.1f} mm"
    )


def export_raw(filename, name, shape):
    """A part that does not fit the 10 inch bed, unscaled."""
    path = os.path.join(OUT, filename)
    s = on_bed(shape)
    bed = App.newDocument("bed")
    obj = bed.addObject("Part::Feature", name)
    obj.Label = name
    obj.Shape = s
    bed.recompute()
    Import.export(bed.Objects, path)
    App.closeDocument(bed.Name)
    bb = s.BoundBox
    log(f"{filename}: {bb.XLength:.1f} x {bb.YLength:.1f} x {bb.ZLength:.1f} mm (larger than 10 in)")


def main():
    # freecadcmd executes this file twice in one process.
    if os.environ.get("CFCLOCK_PLATES_RAN"):
        return
    os.environ["CFCLOCK_PLATES_RAN"] = "1"
    src = App.openDocument(SRC)
    plates = {
        "hub-gears.step": ["Hub_count", "Hub_time", "Hub_rest"],
        "format-drive-gear.step": ["FormatDriveGear"],
        "idler-gears.step": ["FeedIdler", "Idler_1", "Idler_2"],
        "idler-axles.step": ["Axle_feed", "Axle_idler_1", "Axle_idler_2"],
        "value-shafts.step": ["Shaft_count", "Shaft_time", "Shaft_rest"],
        "format-shaft.step": ["Shaft_format"],
        "pointers.step": ["Pointer_count", "Pointer_time", "Pointer_rest", "FormatPointer"],
    }
    flip = {"value-shafts.step", "format-shaft.step"}
    for filename, names in plates.items():
        arranged = arrange([(n, on_bed(copied(src, n), filename in flip)) for n in names])
        export_plate(filename, arranged)

    disks = [
        ("value-disk-count.step", "ValueDiskCount", "ValueDisk_count", ["VL_count_chips", "VL_count_texts"]),
        ("value-disk-time.step", "ValueDiskTime", "ValueDisk_time", ["VL_time_chips", "VL_time_texts"]),
        ("value-disk-rest.step", "ValueDiskRest", "ValueDisk_rest", ["VL_rest_chips", "VL_rest_texts"]),
        ("units-disk-count.step", "UnitsDiskCount", "UnitsDisk_count", ["UL_count_chips", "UL_count_texts"]),
        ("units-disk-time.step", "UnitsDiskTime", "UnitsDisk_time", ["UL_time_chips", "UL_time_texts"]),
        ("units-disk-rest.step", "UnitsDiskRest", "UnitsDisk_rest", ["UL_rest_chips", "UL_rest_texts"]),
    ]
    for filename, label, disk_name, marks in disks:
        log(f"fusing {disk_name}")
        solid = fuse_parts(disk_name, marks, src)
        export_plate(filename, arrange([(label, on_bed(solid))]))

    export_raw("front-plate.step", "FrontPlate", copied(src, "Plate"))
    export_raw("back-plate.step", "BackPlate", copied(src, "BackPlate"))
    App.closeDocument(src.Name)
    with open(os.path.join(OUT, "layout.log"), "w") as fh:
        fh.write("\n".join(log_lines) + "\n")


main()
