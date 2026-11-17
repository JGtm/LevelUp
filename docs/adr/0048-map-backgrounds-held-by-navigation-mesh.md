# ADR 0048 — Map backgrounds held by the navigation mesh, wall plan and orientation

Status: accepted (2026-11-13); implemented on the `feat/fonds-navmesh-natif` work of the same date.
Source: the navigation-mesh work on native and Forge map backgrounds, validated on screen on 2026-11-13 (twenty native
backgrounds recooked and reviewed).
Relates to: ADR 0043 (the bricks are published; the mesh decoder and the cooking options live in them), ADR 0041
(brick boundaries), ADR 0034 (the film decoder owns `replay.MapBackground`), ADR 0011 (adapters), ADR 0044 (the web
client reads capabilities, not profiles).

## Vocabulary

- **Navigation mesh**: the Havok `hkaiNavMesh` of a map, a set of walkable faces over shared vertices. It is the game's
  own description of where a player can stand.
- **Rich background**: the top-down image cooked from the map geometry (`himap`), with its calibration. It is what the
  player sees.
- **Wall plan**: the boundary edges of the mesh, the edges that exactly one face carries (walls, voids, platform
  edges), in world metres. It exists in two forms drawn from the same mesh, the same retained components and the same
  calibration: an image and a vector (D-3).
- **Sidecar**: the `{key}.json` file next to each background image, `replay.MapBackground` plus, since this ADR, `plan`
  (the vector) and `planImage` (the name of the plan image).

## Problem

The rich background keeps matter that is not playable (roofs, skybox, props above the play level), and on some maps
the surface it draws is not the one players run on. The roster of fixes was a per-map tuning of masks and thresholds,
each judged by eye. The game already ships the answer: the mesh of every native map is inside its module, and the
Forge maps have one in the repository (`navmesh.blob`). A second need appeared with it: a 2D view of positions is
unreadable without the walls, and a map is often easier to read turned.

## Decision

- **D-1 — One decoder, two sources.** A native map takes its mesh from a resource of its `any` module (signature
  `ucsh`, contiguous Havok tag files found by the `TAG0` signature); a Forge map takes it from `navmesh.blob`. Both go
  through `hinavmesh` of `halo-infinite-files-reader` and the one `OptionsNavmesh` of `halo-infinite-map-images`
  (`himap`); the two cooking chains do not carry their own mesh code.
- **D-2 — The rich background stays the base; the mesh frames and arbitrates.** The mesh never replaces the image. It
  clips (matter outside the mesh, dilated by a margin, is erased; 1.5 m by default on a native map, off by default on a
  Forge map) and it supplies the ground reference that a high surface is projected onto. Per-map levers in
  `map_fond_reglages.json`: `rogneAuNavmesh`, `margeNavmesh`, `toleranceNavmesh`, `maillageNiveauHaut`, `navmeshMode`, `sansNavmesh`,
  each with its `raison` and `gateLe`.
- **D-3 — The wall plan has two forms drawn from the same mesh: an image and a vector.** The image,
  `<key>.plan.webp` (`MapBackground.planImage`), has the dimensions and the calibration of the background; it is the
  "Plan" background, the client's default. The vector, `MapBackground.plan`
  (`MapBackgroundPlan { source, vertices, faces, edges [[x1,y1,x2,y2]] }`), holds the boundary edges in world metres,
  rounded to the millimetre, in the frame of the calibration; the web client draws it as a static layer between the
  background and the zones, over the "Rendu" background only (the "Plan" image already carries the walls). A map whose
  sidecar has no `plan` has no `planImage`, and the client serves the "Rendu" background.
- **D-4 — Orientation is part of the single projection, per map, shared by every 2D view.** The map turns by quarter
  turns, clockwise on screen, in the world before projection (`CanvasView.orient`, `projectTo`), so only the terrain
  turns and pixel-space text stays upright. The choice is stored per map in the browser
  (`replay-orientation.<mapId>`) with the wall-plan switch (`replay-show-wall-plan`). The replay, the Tactical tab and
  "Terrain occupation" read the same two preferences through `useMapViewPreferences`; only the replay holds the control.
- **D-5 — A map without a mesh is a named degradation.** Either its module carries no mesh (Live Fire,
  `sgh_interlock`) or, for a Forge map, the repository holds no `navmesh.blob`: the map is cooked by the historical
  chain, the sidecar lists the degradation in `degradations` (`aucun maillage de navigation dans le module any`,
  `aucun maillage de navigation au depot (navmesh.blob absent)`) and carries no `plan` and no `planImage`. The web client
  treats a missing plan as normal, not as an error. A read that fails while the mesh exists is an error of the build, not
  a degradation.
- **D-6 — A secondary mesh is ignored and counted.** A module may carry several meshes; the largest readable one is
  used, the others are reported (`secondaires`) and the unreadable candidates too (`illisibles`).
- **D-7 — `--plan-seul` redoes the plan without recooking.** For each map whose mesh is readable (a Forge map whose
  blob is present, a native map whose module carries a mesh) and whose sidecar is published, `mapfond-build --plan-seul`
  rewrites the vector (`plan`) AND the plan image (`planImage`), and leaves the background image, the calibration, the
  statistics, the degradations and `generatedAt` as they were. A map without a mesh or without a sidecar is left
  untouched.
- **D-8 — One reader and writer of the sidecar in LevelUp.** `internal/mapfondsidecar` reads and writes the complete
  sidecar. The film decoder's `replay.MapBackground` ignores `plan` on reading and would drop it on writing, so any tool
  that rewrites a sidecar (`mapfond-webp -convertir`, the build) goes through that package.

## Current state

Counted on the published directory `data/titles/halo_infinite/reference/map_backgrounds/` (339 files):

- 114 sidecars (`*.json`), 114 background images, 111 plan images (`*.plan.webp`, 7.3 MB), 111 sidecars that carry a
  `plan` and a `planImage`: 19 from a native module (`navmesh module any`), 92 from a Forge `navmesh.blob`.
- Three sidecars carry neither: Live Fire (`sgh_interlock`; no mesh in its module) and the two Thunderhead variants
  (`28a3ac28-...`, `37bc3df6-...`; no blob in the repository). The cooking names the Forge degradation since
  2026-11-16; the two published Thunderhead sidecars do not list it yet and get it at their next cooking.
- Of the 18 catalogue maps whose module is installed on the measuring machine, 17 hold a readable mesh. Secondary meshes
  are not decoded on Chasm (5), Launch Site (5) and Cliffhanger (1); 13 candidate resources on 9 of these maps are
  unreadable (`bloc N hors borne`, flags `0x6402`).
- Bazaar: 2 312 vertices, 670 faces, 1 110 edges in the published plan. On Bazaar the reference substitution is not
  exercised; the gain is the clipping.

## Consequences

- The decoder and the cooking options are changed in the two brick repositories and adopted by a `require` bump in
  `apps/go-api/go.mod`; a pseudo-version is allowed on a working branch, the tags are required before `main` (ADR 0043,
  amendments of 2026-11-12 and 2026-11-13).
- A new native or Forge map needs no code: cook it with `mapfond-build --maps`, then convert the sidecar with
  `mapfond-webp -convertir`. A Forge map needs its blob first (`mapnav-fetch`). Levers are added to
  `map_fond_reglages.json` only where a map was judged.
- The sidecar grows by the vector (up to a few thousand edges, 11 MB for the 114 sidecars) and the directory by the
  plan images (7.3 MB). Both are served with the calibration (`planImage` is read through the background route,
  `?variante=plan`).
- Unreadable and secondary meshes are a known, counted gap, not a silent one: widening the decoder to them is a change
  of the brick, judged map by map.
- The vector stays sharp at every zoom and under rotation, which is why it is the form drawn over the "Rendu"
  background; the image is the form that reads at a glance and is the default. The two are redone together by
  `--plan-seul` and by a cooking, so they cannot drift apart.

## Guard-rails

- `internal/mapfondsidecar` tests: a rewrite keeps `plan`; a sidecar without `plan` serializes as `replay.MapBackground`.
- `cmd/mapfond-build` tests: the per-map levers (`reglages_test.go`, where every declared lever has an entry in the
  published file or a dated exemption); `--plan-seul` leaves a map without a blob untouched; the set of published
  backgrounds without a plan is fixed at three, and a Forge map without a blob returns the named degradation
  (`sans_plan_test.go`).
- Web: `lib/replay/orientation.test.ts` fixes the quarter-turn direction, `wallPlan.test.ts` the normalization of the
  nullable transport, `useMapViewPreferences.test.ts` the sharing of the two preferences.

## Amendment of 2026-11-16 — the wall plan is an image and a vector

The 2026-11-13 text of D-3 said the wall plan was "a vector in the sidecar, never a second image". That was never the
state of the code: from the first cooking the plan was also written as an image (`<key>.plan.webp`), which the client
shows as the "Plan" background and uses by default, and the vector is drawn over the "Rendu" background. The image was
validated on screen on 2026-11-13. This amendment brings the text to the code:

- D-3 describes the two forms and their roles.
- D-7 says that `--plan-seul` redoes both.
- "Current state" counts the published files.
- The "Consequences" no longer claim that no image is duplicated.

No code, no published file and no served output changes with this amendment.
