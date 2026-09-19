"""Generate the Windows icon from the master artwork in assets/.

Ported from ED Voyage Companion's tools/genicons.py, keeping its application
icon half: assets/application-icon.png becomes a multi-size .ico beside it.
That one file is the whole identity. build.ps1 embeds it in the executable,
where the tray icon and the shortcuts take theirs from.

Run it when the master changes:

    python tools/genicons.py

It is not part of the build. The output is committed, so a clone needs neither
Python nor Pillow to build the application.
"""

from __future__ import annotations

import pathlib
import sys

try:
    from PIL import Image
except ImportError:  # pragma: no cover - a plain message beats a traceback
    sys.exit("Pillow is required: python -m pip install pillow")

# ICO_SIZES are the sizes Windows chooses between: the small tray and menu
# sizes, the taskbar and shortcut sizes, then the large one Explorer uses in
# its biggest view. Leaving one out makes Windows scale a neighbour, which
# looks soft.
ICO_SIZES = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]

# HEADER_SIZE is the setup window's header mark: about twice the 126 px it is
# drawn at, so it stays crisp on a high-density display. That page has no
# bundler, so it loads the file as it finds it; shipping the master there
# would put over a megabyte behind one picture.
HEADER_SIZE = 256

REPO = pathlib.Path(__file__).resolve().parent.parent
MASTER = REPO / "assets" / "application-icon.png"
HEADER = REPO / "installer" / "frontend" / "dist" / "icon.png"


def trimmed(master: pathlib.Path) -> Image.Image:
    """Open a master and crop away the transparent margin around its artwork."""
    image = Image.open(master).convert("RGBA")
    box = image.getbbox()
    return image.crop(box) if box is not None else image


def render_ico(master: pathlib.Path, target: pathlib.Path) -> int:
    """Write the multi-size Windows icon; return its byte size."""
    image = trimmed(master)
    # Square it before saving. An .ico entry is square by definition, so a
    # source that is not would be stretched into every size rather than padded
    # once.
    side = max(image.width, image.height)
    square = Image.new("RGBA", (side, side), (0, 0, 0, 0))
    square.paste(image, ((side - image.width) // 2, (side - image.height) // 2), image)
    square.save(target, "ICO", sizes=ICO_SIZES)
    return target.stat().st_size


def main() -> int:
    if not MASTER.exists():
        sys.exit(f"no application icon at {MASTER}")
    ico = MASTER.with_suffix(".ico")
    written = render_ico(MASTER, ico)
    print(f"{MASTER.name} {MASTER.stat().st_size:,} -> {written:,} bytes ({ico.name})")

    header = trimmed(MASTER)
    side = max(header.size)
    canvas = Image.new("RGBA", (side, side), (0, 0, 0, 0))
    canvas.paste(header, ((side - header.width) // 2, (side - header.height) // 2), header)
    canvas.resize((HEADER_SIZE, HEADER_SIZE), Image.LANCZOS).save(HEADER, "PNG", optimize=True)
    print(f"setup header {HEADER.stat().st_size:,} bytes ({HEADER.relative_to(REPO)})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
