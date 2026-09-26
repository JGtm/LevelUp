# LOT D1 — la ligne de grappin joignait la MAUVAISE VIE (2026-09-04)

Branche `wt/grappin-vies`, worktree `LevelUp-wt-lecture-equipement`. Fichiers touches :
`apps/web/src/features/match-replay/grappleLayer.ts` et son test.

## Le defaut

`buildGrappleFx` indexait les trajectoires par SLOT en ecrasant :

```ts
const pointsBySlot = new Map(doc.tracks.map((t) => [t.slot, t.points]))
```

Or un slot de biped est REATTRIBUE a chaque reapparition — invariant ecrit noir sur blanc a
quatre endroits du dossier (`shotFx.ts:59`, `fireMark.ts:45`, `riftStations.ts:132`,
`replayMarkers.ts:216`) et une cinquieme fois depuis le lot P4 dans `thrusterDashFx.ts`. Le
`Map` ne conservait donc que la DERNIERE piste du slot. Une accroche appartenant a une vie
anterieure allait chercher les points d'une autre vie :

- si l'instant precede les points retenus, `positionAt` rend `null` et le cable DISPARAIT ;
- si l'instant tombe dans la fenetre de la vie usurpatrice, le cable se peint a la position
  d'UN AUTRE JOUEUR, tendu vers une ancre qui n'est pas la sienne.

Le defaut est anterieur au lot P4 : c'est le modele qui a ete recopie dans le dash, puis
corrige la-bas le 2026-09-03 (constat A1). Le calque du grappin est livre en production depuis
le 2026-08-20.

## Portee MESUREE sur le parc

Mesure sur les 106 artefacts du cache local
(`data/cache/replays/halo_infinite/*.json`, LECTURE SEULE), en rejouant la jointure ancienne
et la nouvelle sur chaque ligne de grappin.

| Grandeur | Valeur |
|---|---|
| Artefacts analyses | 106 |
| Artefacts portant des lignes de grappin | 62 |
| Lignes de grappin (fenetre non vide) | **1 101** |
| Lignes MAL JOINTES par l'ancien code | **0 (0,0 %)** |
| dont disparues / mal placees | 0 / 0 |

**Le chiffre est zero, et il s'explique — ce n'est pas un defaut imaginaire.** Le decoupage
d'une piste PAR VIE n'apparait qu'au SCHEMA 38 (le modele des sieges, valide le 2026-09-02).
Avant lui, un slot porte exactement une piste et le `Map` ne peut rien ecraser :

| schema | films | tracks | slots uniques | vies non finales | lignes de grappin |
|---|---|---|---|---|---|
| 6 | 1 | 35 | 35 | 0 | 0 |
| 20 | 27 | 2 222 | 2 222 | 0 | 328 |
| 21 | 4 | 351 | 351 | 0 | 15 |
| 23 | 7 | 678 | 678 | 0 | 3 |
| 28 | 4 | 288 | 288 | 0 | 1 |
| 31 | 1 | 77 | 77 | 0 | 0 |
| 32 | 9 | 1 016 | 1 016 | 0 | 25 |
| 34 | 51 | 5 102 | 5 102 | 0 | 692 |
| **38** | **2** | **213** | **195** | **18** | **37** |

Sur les DEUX seuls artefacts au schema 38, 15 slots sur 195 portent plusieurs vies, et
**2,6 % des images de vie appartiennent a une vie NON FINALE de son slot**. Une seule des 37
lignes de grappin de ces deux films tombe sur un slot multi-vie (`1cd3848a`, slot 578,
traction 3935-3946) — et elle tombe, par chance, dans la vie FINALE [3901..3977]. D'ou le zero.

**Ce que la correction rend, c'est donc l'avenir du parc, pas son passe.** A la RE-CUISSON
(chantier des sieges, en attente d'accord utilisateur), tous les artefacts passent au decoupage
par vie : en appliquant le taux mesure de 2,6 % aux 1 101 accroches du cache, **de l'ordre de
30 accroches** seraient perdues ou mal placees sur le seul parc local. Sur les donnees
observees, les vies d'un slot sont disjointes et ordonnees chronologiquement dans
`doc.tracks` : le mode de defaillance dominant serait la DISPARITION SILENCIEUSE (le son de
grappin part, le cable ne se dessine pas), la mauvaise position restant possible mais non
observee.

Corriger MAINTENANT, avant la re-cuisson, evite d'avoir a rediagnostiquer un calque qui se
tairait sans rien dire.

## Le correctif

Patron canonique du depot, repris tel quel de `buildThrusterDashFx` / `buildShotFx` /
`buildFireMarks` : grouper par slot dans un `Map<number, ReplayTrackReady[]>`, puis retenir la
vie qui COUVRE l'instant.

```ts
const track = (bySlot.get(l.slot) ?? []).find((v) => isAliveAt(v, l.t0))
if (!track || track.points.length === 0) continue
```

L'instant de reference est `t0`, le DEPART de la traction : c'est le tir qui appartient a une
vie. Une traction qui se termine apres la mort du porteur reste dessinee — `positionAt` fige
alors la derniere position, ce qui est exactement ce qu'on veut voir.

Effet de bord voulu : une accroche qu'AUCUNE vie du slot ne couvre (trou entre deux vies) ne
produit plus d'entree du tout, la ou l'ancien code lui collait les points de la derniere piste.

## La mutation tuee

Test ajoute : `DEUX VIES du meme slot : chaque traction prend la vie qui COUVRE son depart`,
plus deux compagnons. Les deux vies sont ORTHOGONALES et DISJOINTES — la premiere file plein
est en bas a gauche ([10..20], (2,2) -> (4,2)), la seconde plein nord en haut a droite
([30..40], (8,8) -> (8,9)) — pour que la confusion se voie.

Le defaut a ete REINTRODUIT (`.at(-1)` a la place du `.find(isAliveAt)`) et la suite est passee
ROUGE : **3 tests en echec sur 9**.

```
x DEUX VIES du meme slot : chaque traction prend la vie qui COUVRE son depart
  expected [ { t: 30, x: 8, y: 8 }, ...(1) ] to be [ { t: 10, x: 2, y: 2 }, ...(1) ]
x ecarte une traction qu'AUCUNE vie du slot ne couvre : entre deux vies
  expected [ { t0: 24, t1: 27, ...(2) } ] to have a length of +0 but got 1
x trace le cable depuis la position de la vie qui a tire, pas de la vie suivante
  expected undefined to deeply equal [ 30, 80 ]
```

Le correctif a ensuite ete restaure et la suite repasse VERTE (9/9).

## Autres porteurs du meme patron

Balayage de tout `apps/web/src` : indexations `slot -> points` / `slot -> track`, `new Map(...
.map(...))`, et l'ensemble des consommateurs de `doc.tracks`.

**AUCUN AUTRE PORTEUR.** Detail des sites inspectes :

- `fireMark.ts`, `shotFx.ts`, `thrusterDashFx.ts`, `riftStations.ts` — deja au patron par vie.
- `rosterLogic.ts:178` `indexBySlot` — ecrase VOLONTAIREMENT (agregat sur tout le match pour
  le comptage d'usage d'equipement) ; la dette est documentee sur place et le remede pour le
  rendu par image existe : `buildSlotOwnership` / `ownerAtFrame`.
- `rosterLogic.ts:119` `new Map(scoreboard.map(...))` — cle `xuid`, une ligne par joueur.
- `killFx.ts:147`, `livesPosition.ts` (`buildLivesByXuid`) — groupent en LISTES par xuid.
- `threatSensor.ts:284` — cle par slot mais resolue PAR IMAGE, avec `isAliveAt` en amont.
- `heatmapLayer.ts:225`, `killFeedLogic.ts:363`, `padPresenceRefine.ts` — parcourent toutes les
  pistes, sans index.
- `objectivesLayer.ts` — passe par `buildPlayerPosAt` (par xuid, par vie).
- `equipmentKillBadges.ts:64` (`features/match-view`) — cle par famille d'equipement, le slot
  n'est qu'une valeur portee.

## Gates

| Gate | Sortie |
|---|---|
| `make check-types` | EXIT 0 |
| `make test-web` | EXIT 0 — 562 fichiers, 5 843 tests passes, 14 skippes, 0 echec |
| `npx eslint` sur les 2 fichiers | EXIT 0 |
| Couleurs en dur | aucune (le calque recoit son encre de l'appelant) |
| Cliquets de taille | `grappleLayer.ts` 118 L, test 208 L — aucun releve |

Aucun commit, aucun push : le superviseur relit.
