# Map geometry — Forge props, one directory per map

`forge_object_types.csv` lives **here**, at title level: it maps a `type_id` to a measured
footprint, and that table is the same for every map. Copying it under each map would create
copies that drift.

`map_objects.csv` lives **one level down**, under the map's **module** name — the same key as
`map_quant_bounds.json`, `map_structure/` and `map_backgrounds/`. Until 2026-09-11 it sat at
this level and `MapGeometryDir` took only the title, so the single file was drawn on **every**
match whatever the map: 382 identical props across all 76 baked artifacts. With a real map
background underneath, another map's scenery is not a stopgap — it is wrong data on a tool that
measures positions.

A map with no directory simply has no props. That is the normal case (nobody has extracted the
props of the 79 maps in the bounds catalogue), it is logged at Debug, and the replay draws
without contextual landmarks. A missing or unreadable `forge_object_types.csv`, on the other
hand, is a failure of the title and is logged as a warning.

To add a map: extract its props from the `.mvar` variant, write
`map_geometry/<module>/map_objects.csv` with the same columns, and state in the commit how the
map was identified.
