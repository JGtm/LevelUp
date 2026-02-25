# -*- coding: utf-8 -*-
"""Audit de couverture i18n."""
import re
from pathlib import Path

pages_dir = Path("src/ui/pages")
total_st = 0
fr_remaining = 0
details = []

ST_CALL = re.compile(r"st\.(write|info|warning|error|success|markdown|subheader|header|caption|spinner)\s*\(")
FR_CHARS = re.compile(r"[àâãäéèêëîïôùûüçÀÂÉÈÊËÎÏÔÙÛÜÇ]")
HAS_T_CALL = re.compile(r't\(["\']')

for f in sorted(pages_dir.glob("*.py")):
    c = f.read_text(encoding="utf-8")
    for i, line in enumerate(c.splitlines(), 1):
        stripped = line.lstrip()
        if stripped.startswith("#"):
            continue
        if ST_CALL.search(line):
            total_st += 1
            if FR_CHARS.search(line) and not HAS_T_CALL.search(line):
                fr_remaining += 1
                details.append(f"{f.name}:{i}: {line.strip()[:90]}")

pct = (total_st - fr_remaining) / total_st * 100 if total_st else 0
print(f"Total st.* calls : {total_st}")
print(f"FR restants       : {fr_remaining}")
print(f"Couverture        : {pct:.1f}%")
print()
if details:
    print("Strings FR restants :")
    for d in details:
        print(" ", d)
