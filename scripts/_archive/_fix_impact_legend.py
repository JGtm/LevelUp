# -*- coding: utf-8 -*-
from pathlib import Path

p = Path("src/ui/pages/teammates_impact.py")
c = p.read_text(encoding="utf-8")
old = 'st.caption("⚡ Premier sang | 🎯 Finisseur | 💀 Boulet | 🐌 Plus lent | 🪦 Première victime")'
new = 'st.caption(t("tm_impact_legend"))'
print("Found:", old in c)
c2 = c.replace(old, new, 1)
p.write_text(c2, encoding="utf-8")
print("Done")
