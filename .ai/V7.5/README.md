# V7.5 — recherche et rétro-ingénierie (rejeu 2D, événements de frags/morts)

Ce dossier rassemble les documents de **recherche et de rétro-ingénierie** produits dans les
deux worktrees `feat/filmdec-*` (décodage du film Theater, arme/source de dégât par kill) et
`feat/replay2d-*` (rejeu 2D), réunis sur `feat/replay2d-prod` le 2026-07-31.

Ce sont des **archives de chantier** : elles font foi sur ce qui a été prouvé, mesuré ou
réfuté. Ce qui reste à faire ou à terminer n'est pas ici mais à la racine de `.ai/`
(voir « Ce qui est resté à la racine » plus bas).

## Organisation

| Dossier | Contenu | Fichiers |
|---|---|---|
| `film_re/` | Format du film Theater : grammaire, chunks, décodeur ECS, keyframes, RE Ghidra, handoffs externes | 15 |
| `killweapon/` | Arme / source de dégât par kill : kill feed, dead-state, same-clock, walk biped, journal RE | 21 |
| `replay2d/` | POC du rejeu 2D : trajectoires, inventaire/loadout, vérité terrain | 6 |
| `cartes/` | Géométrie 2D des maps depuis les `.module`, triangles, noms de zones | 3 |
| `dumps/` | Captures binaires, CSV, PNG (ex-`.ai/re_dump/`) — 69 Mo, lus par du code | 40 entrées |

### Points d'entrée par sujet

- **Arme par kill** : `../README_KILLWEAPON_INDEX.md` (index maître, à greper en premier),
  puis `killweapon/RE_LOG_KILLWEAPON.md` (journal, ne jamais le lire par le haut).
- **Format du film** : `film_re/GRAMMAIRE_RECORD_FILM.md` puis
  `film_re/RECETTE_DECODAGE_FILM_CHUNKS.md`.
- **Reverse externe / handoff** : `film_re/HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md`,
  `film_re/GITHUB_RE_FINDINGS_EN.md` (EN).
- **Cartes** : `cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md`.

## Ce qui est resté à la racine de `.ai/`

Les documents **encore vivants** au 2026-07-31 : les états de l'art (référence courante),
l'index maître arme-par-kill, et les plans/handoffs créés le jour même dans les deux
worktrees, qui restent à traiter ou à terminer.

État de l'art et index : `ETAT_DE_L_ART_KILLWEAPON.md`,
`ADDENDUM_ETAT_DE_L_ART_2026-07-26.md`, `ETAT_DE_L_ART_CHANTIER_VOISIN.md`,
`README_KILLWEAPON_INDEX.md`.

Réconciliation et dette : `PLAN_RECONCILIATION_BRANCHES.md`, `PLAN_DETTE_AVANT_MERGE.md`.

Rejeu 2D : `PLAN_FINALISATION_REJEU_2D.md`, `PLAN_REJEU_2D_FIABILISATION.md`,
`SUIVI_REPLAY_2D.md`, `ETAT_DU_POC.md`, `CAHIER_DES_CHARGES_POC.md`,
`HANDOFF_REPLAY_2D_2026-07-29.md`, `PLAN_OBJECTIFS_TEMPS_REEL.md`,
`PLAN_CAPACITES_ACTIVES.md`, `PLAN_RECHERCHE_ASSETS_ICONES.md`,
`PLAN_BELLE_CARTE_TRIANGLES.md`, `CLE_USB_REJEU_2D.md`.

Killsource / armes : `HANDOFF_KILLSOURCE_REPRISE.md`, `PLAN_BRANCHEMENT_KILLSOURCE.md`,
`GUIDE_WEAPON_SHOTS.md`, `HANDOFF_PRECISION_PROJECTILES.md`, `PLAN_VARIABLES_JETEES.md`.

Captures et dumps : `HANDOFF_DUMPS_2026-07-31.md`, `SESSION_CAPTURE_AVANT_PC.md`.

Journaux : `thought_log.md` (principal) et `thought_log_replay.md` (chantier rejeu 2D).

## Pièges

- **Les journaux n'ont pas été réécrits.** `thought_log.md` et `thought_log_replay.md`
  citent les anciens chemins plats (de la forme `.ai/<NOM>.md`, `.ai/re_dump/...`) pour les
  entrées antérieures au 2026-07-31 : ce sont des archives, on ne réécrit pas l'histoire.
  Pour retrouver un document cité dans une vieille entrée, chercher son nom, pas son chemin.
- **`dumps/` est lu par du code**, pas seulement cité : `cmd/replay-build` (défaut de
  `-geometry`), plusieurs `cmd/tmp_*`, et `internal/analysis/replay/mapvar` (test). Déplacer
  ce dossier oblige à corriger ces chemins.
- Les `cmd/tmp_*` qui pointent en absolu vers `.claude/worktrees/filmdec-continuation/.ai/re_dump/`
  n'ont pas été touchés : ce worktree conserve son ancienne arborescence.
