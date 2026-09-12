# `ridgeline` (Cliffhanger) — props attributed by footprint, not by a column

`map_objects.csv` carries **no map column**. It used to sit one level up and was drawn on every
match whatever the map. It is placed here because two independent lines of evidence point at
Cliffhanger, whose module is `ridgeline`:

1. **Footprint against the played area.** The 453 props span X [-10.56, 44.00], Y [-24.65,
   39.02]. Cliffhanger's played area, taken from the baked artifacts, is X [-7.7, 44.2],
   Y [-25.7, 36.4]: 90.8 % of the prop footprint falls inside it, and the two areas differ by
   8 % in size. No other map in the parc comes close on both counts at once — the runners-up
   either cover a fraction of the props (Catalyst 45 %, Aquarius 36 %) or are so large that they
   swallow any footprint while the props would occupy a few percent of the field (Fortitude
   98 % containment for an area 25 times too big).
2. **Provenance.** The commit that introduced this CSV (`2044b7139`) carries exactly two `.mvar`
   files in its RE dump, `cliffhanger_map.mvar` and `cliffhanger_ridgeline.mvar`, and
   Cliffhanger (`000d5950`) is the reference film of the whole replay effort.

**This is an argued attribution, not a read one.** If the props ever look wrong on Cliffhanger,
the falsification is cheap: bake `000d5950` and check that the props sit on the geometry rather
than beside it. Should that fail, move the file back under `map_geometry/UNATTRIBUTED/` rather
than guessing another map — the data may well be right, it is its attribution that is deduced.
