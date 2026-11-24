# ADR 0049 — The Relief map background, the relief lines and the floor cut

Status: accepted (2026-11-18); implemented on the `feat/fonds-relief` work of the same date.
Source: the Relief plan of 2026-11-18 (`.ai/V8.3/PLAN_FONDS_RELIEF_2026-11-18.md`), built on the principles of OpenSpartan
Cartographer; the comparison mock-up of the same date and the decisions of the user, taken at the go and at the end of
the design gate (colour, marker outline, floors, light, weight).
Relates to: ADR 0048 (amended, see the last section), ADR 0043 (the relief is computed in a published brick; a
pseudo-version on the working branch, the tag before `main`), ADR 0041 (brick boundaries), ADR 0034 (the film decoder
indexes the published backgrounds by their `.json` files), ADR 0044 (the web client reads capabilities; no new one here).

## Vocabulary

- **Relief**: the height of the top surface of each pixel of a map background, seen from above, with the lines that follow
  from it. It has two files per map (D-6).
- **Top**: the height a pixel shows. The height of the cooked render, raised to the walkable floor where the mesh says so
  (D-7).
- **Kept zone**: the pixels whose height is known and that lie inside the navigation mesh (widened by `himap.MargeNavmesh`,
  narrow passages closed, enclosed hollows filled, specks removed). Only these pixels are tinted.
- **Floor**: a walkable surface of the navigation mesh. A pixel can carry up to four stacked floors. A **storey** is a
  height at which the floors of the map gather (a peak of their histogram).
- **Cut**: the choice of one height. The storey chosen is shown in full, the storeys below in pale, those above almost
  erased, and a floor covered by a roof reappears.
- **Game unit** (`u`): the unit of every height and distance of the relief. The code calls them metres
  (`metersPerPixel`); a unit is about 3 m (median ground speed 2.2 to 2.4 u/s, sprint 3.05 u/s).

## Problem

The rich background keeps what a player never stands on, and it says nothing of height: a map with stacked floors or a
natural terrain cannot be read from it, and a replay position on the second storey looks like one on the ground. The "Plan"
background draws the walls but not the terrain. The user liked the look of Cartographer's terrain and wanted it as a
complement of the existing backgrounds, not a replacement, and a way to look at one storey at a time.

## Decision

- **D-1 — A third background, "Relief", beside "Plan" (the default, unchanged) and "Rendu".** The three have the same
  dimensions and the same calibration, so changing background changes neither the projection, the orientation, the zoom nor
  the hover. They are offered in this order in every 2D view. The preference falls back on the Plan when the map has no
  relief or when its relief fails to read (`ReliefError`), and on the Rendu when it has no plan either; nothing is said to the
  viewer (`lib/replay/mapBackgroundVariant.ts`).
- **D-2 — A "Relief lines" layer.** Height breaks (a jump of more than 0.35 u between two neighbouring pixels), contour
  lines every 0.5 u on slopes, and the outline of the kept zone, as vectors drawn by the client, sharp at every zoom. It is a
  switch of its own, kept per background: on by default over the Relief, off over the Plan and the Rendu. The "Wall plan"
  switch is independent of it (D-3 of ADR 0048, amended). The file carries no height per line; the client keeps the lines
  that the cut keeps (a segment is kept if the cut keeps the pixel at its middle, in pieces of at most 0.5 u).
- **D-3 — The floor cut, in the three 2D views.** One control: "All" (default), one pill per storey, a height slider.
  The cut applies to the three backgrounds, by an opacity mask. In the replay the controls are in the "View" panel, and a
  "Follow the open player" mode takes the storey of the player's track at the instant read: the floor under the player's
  feet (the highest mesh floor at most 0.35 u above them), a change of storey only after the gap exceeds 0.5 u for
  300 ms, snapped to a storey within 0.7 u; a manual choice of storey stops the following; while the player is dead the
  storey stays until the respawn. The cut hides no player in the replay (guard
  `reliefNoPlayerMask.guard.test.ts`), only the background changes. "Terrain occupation" hides the positions outside the
  chosen storey. In Tactics the cut holds for the background and for the two layers whose data carry a height (the named
  zones and the opposing respawns by zone, through `z_bottom` and `z_top`); the other layers stay whole and say so in
  their legend.
- **D-4 — The heavy computation is done at baking time, in the brick; the browser tints, cuts and draws.** The heights, the
  flags, the stacked floors, the storeys and the lines are computed once, in Go, by `halo-infinite-map-images` v1.4.0
  (`himap.CalculeRelief`, `CalculeSols`, `CalculeTraits`; a minor version, API additions only). The client tints (a table
  of 256 colours drawn from the token), shades, cuts and draws, in a Web Worker (the first of the application, with a
  synchronous fallback when no worker starts). Three views consume the relief and the site is read on phones; the lines were
  what cost a second in the browser, and they leave with the baking.
- **D-5 — `mapfond-build --relief-seul` projects the relief into the PUBLISHED frame.** It is the fourth gesture of the
  tool. It bakes the map in memory, reads the calibration of the published sidecar, projects the relief into it, and
  writes only the two relief files and two sidecar fields; the background image, the plan image, the wall plan and the
  calibration do not change. A scale that differs from the published one by more than 1e-6 (relative) is an error of that
  map (`himap.ErrEchelleDifferente`) and nothing is written: sampling a fine grid into a coarse one would invent steps, hence
  false contour lines. A map without sidecar or background is left untouched. A complete baking redoes the relief in the
  same gesture; when it cannot, it publishes the background without relief and removes the stale files. A map whose
  published sidecar carries a wall plan but whose baking finds no mesh (blob absent or unreadable on the workstation) is
  an error of that map, named, in both gestures: nothing is written for it and the exit code is not zero, so that a
  relief "without mesh" is never published for a map that has one (D-9 is for the maps that have none). The need is
  measured: the rebake of Recharge on 2026-11-18 came out at 1605 x 1736 px where the published background is 1605 x 1771 px.
- **D-6 — Files `<key>.relief.webp` and `<key>.relief.json.gz`.**
  - The image is lossless WebP, with the size and the calibration of the background. Per pixel: A is 0 (height unknown, R, G
    and B are then 0) or 255, never in between, because a canvas stores pixels premultiplied and an intermediate alpha would
    crush the colour channels on reading; `q = R + 256 G`, height = `z0 + q x step`, `step` = 1/256 u (a coarser step gives a
    shading error of up to 4.2 levels out of 255, this one 1.9); B carries two flags, bit 0 "on the map" (height known and
    inside the kept zone) and bit 1 "floor taken from the mesh"; bits 2 to 7 are reserved and zero. The client reads it with
    `colorSpaceConversion: 'none'` and `premultiplyAlpha: 'none'`, at 1:1, and refuses an image that breaks the format (alpha
    outside {0, 255}, a reserved bit, colour on an unknown pixel, other dimensions than the calibration): the relief is then
    unreadable and the Plan is drawn. `mapfond-webp -convertir` refuses a round trip that is not identical to the bit.
  - The data are one-line JSON (`schemaVersion` 1), gzipped: a copy of the calibration the relief was projected to, the height
    coding and bounds, the floors (storeys, bounds of the slider, merge gap and tolerance, share taken from the mesh, mesh
    vertices in hundredths of a unit and the kept faces as triangles) and the lines (polylines in tenths of a pixel, the first
    point absolute and the others delta-encoded, the integer naming the pixel centre), and the degradations as closed codes
    (`sans-maillage`, `hauteurs-ecretees`, `pile-tronquee`). The format is that of a public brick, which produces the
    flag "floor taken from the mesh" and the three codes for any consumer: LevelUp reads only what it needs, the share
    taken from the mesh (`floors.meshShare`) and the code `sans-maillage`; the flag in the image is validated, not read,
    and the codes `hauteurs-ecretees` and `pile-tronquee` are decoded, not translated. A field of a public format that
    LevelUp does not read is legitimate, by the same rule as the replay document of the film decoder (rule 18 of
    `CLAUDE.md`).
  - **Not `.json`.** The decoder's background index (`replay.MapBackgroundIndexFor`) reads every file with the `.json`
    extension of the directory as a sidecar: a `<key>.relief.json` would register a phantom key `<key>.relief`. The extension
    of the data is `.json.gz`; the service decompresses (bounded at 32 MiB), checks the schema version and the calibration,
    and serves plain JSON.
  - The sidecar names both files (`reliefImage`, `reliefData`). Routes: `?variante=relief` on the two image routes (replay
    by match, Tactics by map) and a sibling route `background.relief.json` on both prefixes, with the cache of the plan image;
    absence is a 404, as for an absent plan.
- **D-7 — The top is raised to the walkable floor.** Where the highest floor of the mesh is more than 0.3 u above the height of
  the render (`himap.ToleranceDessusSol`), the floor becomes the top and the pixel carries the "floor taken from the mesh" flag.
  The top is never under a walkable floor. Measured on the four witness maps: a floor more than 0.3 u above the rendered top on
  9.5 to 28.4 % of the kept zone; without this rule the "All" view loses the high walkways, the lines gain noise and the cut shows
  only part of a storey.
- **D-8 — Ground without geometry takes its height from the mesh.** Where the mesh says "walkable" and the cooking has no matter
  (natural terrain filled flat in the published background), the height comes from the mesh and the pixel carries the same
  flag. The Rendu is unchanged: it stays the raw image of the geometry, with its flat fill. The tooltip of the Relief
  background says the share of the zone taken from the mesh ("relief taken from the navigation mesh over 46 % of the
  map"). That share is the one the brick writes in `floors.meshShare`, and it is what the tooltip reads; the per-pixel
  flag that the image carries for the same fact is not read by LevelUp (see D-6).
- **D-9 — A map without a mesh keeps a Relief, without cut.** The kept zone is the footprint of the matter, the file carries
  no floor and the degradation `sans-maillage`; the client hides the storey control and shows an explanation block (dotted
  border). The two Thunderhead variants are in this case: no `navmesh.blob` is published for them (`mapnav-fetch` reports none,
  and a direct request answered `BlobNotFound` on 2026-11-18).
- **D-10 — Live Fire takes its mesh from the shared module.** The geometry of Live Fire (`sgh_interlock`) lives in
  `pc/globals/common-rtx-new.module`, and its mesh in the module of the same name under `any/globals/`: resource 20390, 1 566
  vertices, 786 faces. The shared module holds the meshes of several arenas; `himap.ChargeMaillageNatifGeometriePartagee` keeps
  the mesh of the arena that the BSP choice retains for the geometry (at least half of its vertices inside the arena box, then
  the most objective anchors inside its box, then the larger share of vertices); the neighbouring arena (resource 20391, 287 faces) is counted as secondary. The function is a sibling of
  `ChargeMaillageNatif`, which is unchanged. 100 candidates of the module do not extract: their bytes are not in the file.
  The shared **playable boundary** (`sddt`) and the **water** are not applied, and the sidecar says so with the measure: the
  boundary is one closed surface of 124 triangles around several arenas, which would remove about 0.4 u2 of walkable ground
  without gaining on the clipping to the mesh; the water is 42 volumes under the playing floor, 84 % of whose cells would fall
  on walkable ground. Live Fire was rebaked in full (background, plan image, wall plan, relief) and validated on screen by the
  user on 2026-11-18.
- **D-11 — One tint per accessibility palette, from a token.** `map-relief-base`, one value for both themes, no colour picker:
  Graphite `#6B7078` (default), Violet `#7559B5` (Okabe-Ito, Tol Bright), Green `#3F8C4F` (Cividis). The ramp is computed in a
  single place (`lib/accessibility/reliefRamp.ts`): low = base mixed 30 % into black, mid = 78 % into black, high = base mixed
  30 % into white, linear in sRGB between the three. The choice is measured, not aesthetic: no candidate among the seven
  Cartographer presets and the theme grey keeps the marker cores 15 DeltaE away from every ramp step under the four visions
  (0 combinations out of 64), and the former marker outline reached 3:1 in none of 1 280 combinations; the tint is the least bad,
  and D-12 does the rest.
- **D-12 — A two-tone marker outline, on the three backgrounds.** A dark band (`marker-outline-band`, `#080C12`) against the
  core, edged by a light rim (`marker-outline-rim`, `#FFFFFF`), the same in every palette and both themes. The best of the two
  tones reaches at least 4.4:1 against any grey (4.43:1 in the worst combination measured), where one tone cannot hold a
  ground that goes from nearly black to nearly white. `mapReliefTokens.test.ts` requires at least 3:1 against every ramp step,
  every tint of the Plan, the surface of the Rendu and the card of the theme, under the four visions, by palette and theme. It
  changes the validated look of the replay markers, with the agreement of the user; the floor rings and the double outline of
  the page's player sit beyond the outer edge of the outline.
- **D-13 — Storeys are ranks, not metres.** The pills read "Low", "Floor 1", "Floor 2"...; the tooltip links them to the rings of
  the markers ("Floor 1: 1 ring"). A unit is about 3 m, so "+2.2 m" would be false. Storeys are the peaks of the histogram of
  floor heights (0.25 u step, at least 3 % of the samples, at least 1.4 u apart, five at most); on a natural terrain they are
  frequency steps, not floors, and the slider is the right control.
- **D-14 — The light comes from the upper left of the screen, whatever the orientation.** The shading is computed in the browser,
  so its direction follows the quarter turns (`lightForTurns`: (-1,-1), (-1,1), (1,1), (1,-1) in the image frame) and is cached
  per orientation. A map turned by 180 degrees and lit from the bottom right would invert to the eye.
- **D-15 — A stale relief is an absent relief.** The data copy the calibration the relief was projected to. The service compares it
  to the sidecar, checks the schema version, and answers as for an absent relief (404, logged) when the calibration differs, the
  version is unknown or the file is unreadable; the client then draws the Plan. A relief never outlives its background (D-5). The rule is the same for the data route and for the image route.
- **D-16 — Credit.** The principles (height per pixel, shading, contour lines, cut by storey) and the starting thresholds come from
  OpenSpartan Cartographer, by Den, under the MIT licence. The algorithm is written again in Go in the brick, in game units
  (the thresholds in feet are not copied; they were measured again on four maps); no line of the original is copied. The credit
  is in the README of the brick, in this ADR and in the release notes.

## Current state

Counted on the published directory `data/titles/halo_infinite/reference/map_backgrounds/` of the `feat/fonds-relief` work,
on 2026-11-22 (568 files; 114 backgrounds, each with its `.json` sidecar):

- **114 of 114** sidecars name a relief (`reliefImage` and `reliefData`); each has its `<key>.relief.webp` and its
  `<key>.relief.json.gz`. The Plan (D-1) is the fallback only where the relief fails to read.
- **2 reliefs carry the degradation `sans-maillage`** (D-9): Thunderhead (`28a3ac28-f69d-4fa9-9ebf-a0449c89c8da`) and
  Thunderhead Heavies (`37bc3df6-93e8-4d74-b16e-5ceaa30ebc23`). They are the two backgrounds without a wall plan.
- **Weight of the relief files**: 114 images and 114 data files, 51.1 MiB (53.6 MB) in all.
- **Thunderhead**: 13% of its published background has no height in the relief (the published background holds matter that today's bake does not have);
  a full re-bake of that map would close it. Thunderhead Heavies: 0% of its background has no height.

## Consequences

- Adding a map needs no code: bake it (`mapfond-build`, which now writes the relief), or add the relief to a published map with
  `--relief-seul`, then convert the image (`mapfond-webp -convertir`). A change of the relief computation is a change of the
  brick, adopted by a `require` bump (ADR 0043).
- The relief files are served with the calibration, by the background routes; no route outside the two prefixes and no capability.
- The Relief background has a hole where the Rendu has a flat fill of assumed floor (a pixel of the published background that has no
  height in the relief). The baking logs the count per map (`fond_sans_hauteur`).
- Under a cut, the Plan does not show a floor covered by a roof: its image keeps the highest floor of each pixel. Only the Relief
  reveals it.
- The first Web Worker of the application: it is built by Vite, needs no change to the content security policy of the Tauri
  shell, and a failure to start it falls back on a synchronous computation.
- Heights and distances in the relief are game units, about 3 m. Every distance shown "in metres" that comes from `metersPerPixel`
  (the radar radii of Tactics, shot distances) should be read again.

## Guard-rails

- `internal/mapfondsidecar`: `TestNommageRelief` (`<key>.relief.json.gz` is the data name, `<key>.relief.json` is not), `TestSidecar_ReliefAllerRetour`.
- `cmd/mapfond-build/relief_seul_test.go`: `--relief-seul` changes only the two sidecar fields; a different scale writes nothing; a
  map without a mesh gets the degraded relief; a map without sidecar is left intact; a complete baking redoes the relief and a
  baking without relief removes the stale one. `sans_plan_test.go`: the backgrounds without a plan are the two Thunderhead.
- `internal/service/replay_map_background_relief_test.go`: a different calibration is a stale relief, unreadable data are an absent
  relief, hostile names and keys are refused. `internal/api/handlers/background_relief_test.go`: the variant, the data route, a badly
  written variant, the ETag cycle.
- The brick: tests on synthetic grids (step, ramp, roof over a floor, enclosed hole, ground without matter), and `*_gamefiles_test.go`
  (tag `gamefiles`) on Recharge and on Live Fire (resource 20390, 786 faces, the shared boundary and water not applied).
- Web: `lib/accessibility/mapReliefTokens.test.ts` (tint per palette, ramp steps, contrast of the outline), the tests of
  `lib/replay/relief/` (reading and refusal of the image, floor stack, cut, lines, follow rule), `reliefNoPlayerMask.guard.test.ts`.

## Amendments to ADR 0048

- **D-3 (the wall plan).** There are three backgrounds, not two: Plan, Rendu, Relief. The wall plan vector is drawn over the Rendu
  AND the Relief (the Relief image does not carry the walls), independently of the relief lines; over the Plan it is not drawn.
- **D-4 (orientation).** The "Terrain occupation" view has a background control of its own (Plan, Rendu, Relief); it no longer only
  reads the replay's choice. The orientation and its rules are unchanged.
- **D-5 (a map without a mesh).** Live Fire is no longer one: its mesh is in the shared module (D-10). The maps without a mesh are
  the two Thunderhead variants. The "Current state" of ADR 0048 counted three sidecars without a plan; there are two.
- **D-8 (the sidecar).** Two fields join `plan` and `planImage`: `reliefImage` and `reliefData`, the names of the two relief files.
  `internal/mapfondsidecar` stays the only reader and writer of the complete sidecar, and `mapfond-webp -convertir` rewrites
  `reliefImage` through it.
