# Porting your additions onto decoder v1.5.0: welcome pack

Hi Jordan,

Thank you for the work in `third_party/`, and for shot direction from aim (#17), which is already in the decoder since v1.3.7. This note tells you where to start, what changed upstream since your base, and what we need from you. We want your channels in the next release (v1.6.0), and nothing here has a deadline.

## 1. Where to start from

| Library | Your base | Start from | Public licence |
|---|---|---|---|
| `halo-infinite-film-decoder` | v1.3.6 | tag `v1.5.0` | AGPL-3.0 |
| `halo-infinite-map-images` | v1.0.1 | tag `v1.4.0` | AGPL-3.0 |
| `halo-infinite-files-reader` | v1.0.1 | tag `v1.2.0` | GPL-3.0 (because of `ooz`) |

At v1.5.0 the replay document is at schema 126 and the facts file at schema 30. The decoder still requires files-reader v1.0.1 and map-images v1.0.1; map-images v1.4.0 requires files-reader v1.2.0.

## 2. What changed upstream since v1.3.6

- **v1.3.7**: your shot direction (`shots[].h`, new `shots[].p`, `coverage.shots.aim`); one shared frame walk for damage source and replay build.
- **v1.3.9** (schema 124): timed medals, scores of all seats, presence and seats read from the film, round clocks and rules, personal score per action, end of each life (`tracks[].endReason`).
- **v1.4.0** (schema 125): `hits[]`, `damage[]` in points, `shotStats[]`, flag return gauge.
- **v1.5.0** (document 126, facts 30): `reloads[]`, `stances[]` of kind `moveCommand`, `eliminations[]`, named mode counters, `t0Film.source`. Shots are now read at every rank of a frame (about 30 % more shots); hit and damage authors come from the unit's responsible player; the kickoff `t0FilmMs` is the start of the first round clock (0.3 to 0.9 s earlier).
- **Removed**: 116 research instruments (v1.3.9) and two unreachable fallbacks (v1.3.8); many of your files may sit next to deleted ones. No public symbol was removed.

Full notes: `gh release view vX.Y.Z --repo JGtm/halo-infinite-film-decoder`.

## 3. Overlaps to resolve before porting

Our rule for all of them: **add fields; never change the meaning or the count of an existing field.** Where you and upstream publish the same thing, tell us which form you need and why, and we decide together.

- **Reloads.** Upstream: `reloads[]`, one entry per (frame, author, references), from the `weapon_reload` message; the four booleans are read but unpublished. You publish a reload as a gesture. Question: does your gesture list carry anything `reloads[]` lacks (weapon, duration)? If not, build on `reloads[]`.
- **Move command.** Upstream: held-command intervals in `stances[]` (`moveCommand`), cut at life limits. You publish the raw stick at full rate. These are two valid products: please add the raw channel under its own name and leave `moveCommand` as is.
- **Kickoff.** Upstream: `t0FilmMs` from the first round clock, with `coverage.t0Film`. Both repositories have `match_clock.go`; we have not compared contents. Question: does your copy publish a different kickoff? If yes, add a field.
- **Shots by seat.** Upstream: `coverage.seats.tirsParPlace` gives a shot to the seat's occupant, not the index. You add control-seat shooter (`places_de_controle.go`). Question: what does yours decide that this does not?
- **Aim.** `shots_visee.go` exists on both sides (your #17 is in). Question: do your aim-only tracks add anything beyond the heading and elevation that track points already carry? Also `damage[]` per seat and `hits[]` exist upstream (v1.4.0, changed in v1.5.0).
- **Posture.** Upstream has `stances[]` kinds for crouch and slide (recent film versions rarely announce them, so they are mostly empty) and a labelled derived jump (`jumpDerived`). The jump button read from the film is not published: it agreed with derived jumps only 63 % of the time.

## 4. Schema numbering

One bump for the whole release: document 127, facts 31, to be confirmed by us when we freeze. **Do not raise the numbers yourself.** We refreeze references, layer revision fingerprints and fixtures at integration; revision tests will go red when you touch the grammar or source layers, and that is expected. We write one chronicle entry covering all your channels, with you. For that, each PR should list the fields it adds.

## 5. Rules of the public repository

- No real player identifier or gamertag anywhere, fixtures included (Xbox ids start with 2533 or 2535).
- No file extracted from the game. Tests that read the installed game are named `*_gamefiles_test.go` and tagged `gamefiles`; research instruments carry the `research` tag.
- Files up to 500 lines, functions up to 80.
- No silent fallback: each one has a name, a counter, and an entry in the fallback registry (`internal/facts/fallback/`).
- The film's grammar beats any heuristic. If a value is derived instead of read, say so.
- semver v1: additions only. No public symbol removed, no signature changed. Callers go through the `decfilm` facade.
- Comments describe the contract, not the history. Keep each repository's language and naming as you find it (the decoder uses French comments and identifiers); do not translate existing files.
- Licence: AGPL-3.0 for the decoder and map-images, GPL-3.0 for files-reader. A new dependency (`woozymasta/bcn`) needs its licence checked by us before it enters.
- Locally: `go vet ./... && go test ./...` from the module directory.

## 6. Please do NOT bring back

- **Message translations** of existing files: each repository keeps its language.
- **Target-score table per variant**: it is a per-variant table, which the grammar rule excludes; LevelUp injects the values itself.
- **Jump derived from vertical speed**: a derivation, not a read; `jumpDerived` already covers it, labelled. A jump read from the film would be welcome.
- **`REPLAYINPUTS30` codec**: only if one of your channels needs it, with that channel's PR.

If you disagree with one, say so; it comes back to the maintainer.

## 7. Shots, hits, damage

Add fields only. We check mechanically, on 20 films, that the step fingerprints for shots, hits and damage are identical before and after your change. If they move, the PR comes back to you with the diff.

## 8. How we sync

Start now from `v1.5.0`. We work in parallel on `feat/v1.6.0` (decoding speed, fixes). When our part is merged we push a **freeze commit** and tell you; one rebase onto it is all that waits for you. After the freeze only your PRs and review fixes enter.

Files we will touch before the freeze, so you know where conflicts are likely: `replay/options.go`, `replay/film_scan*.go`, `replay/build_from_film.go`, `replay/document_stances*.go`, `replay/layers.go`, `replay/coverage.go`, `replay/document_schema.go`, `replay/document_chronicle.go`, `replay/filmfacts*.go`, `internal/source/`, `filmcache/`, `decfilm/`, `internal/grammar/` (`marche_*`, `distribuer*`, `keyframe_*`, `vue_a_lecture.go`, `movement_states.go`, `frame_vue_controle.go`), revision files, goldens, plus score, flag, zone and elimination files in `replay/` and `internal/facts/objectives/`.

One PR per channel or small coherent group; each says what changes in the output. Docs fragments (FORMAT, GLOSSARY, NOTICE) go in a separate file in the PR; we merge them.

## 9. Order across the three libraries

files-reader first, then map-images (it depends on it), then the decoder (it depends on both). We set the tags. A decoder channel that needs a new reader or maps function waits for that tag.

## 10. What we check on arrival

For each decoder PR, on a control branch: identifiers and game files scan; independent code review (heuristic instead of a read, unnamed fallback, undeclared output change, overlap with an existing field); full tests and revision gate; public-surface compatibility against `v1.5.0`; equivalence on 20 films with every difference matched to what the PR declares; a second pass from stored facts; the shots/hits/damage guard; added document weight; decoding time within 10 % of before. For the other two libraries: full tests, no unexpected skip, compatibility, no game file, licences. A red check returns the PR to you with the evidence; we do not fix your code without your agreement.

## 11. What we need from you now

1. Confirm the PR split you plan (channel, order, dependencies between them).
2. A rough, non-binding time estimate.
3. Which channels your viewer needs first.

Thank you again.
