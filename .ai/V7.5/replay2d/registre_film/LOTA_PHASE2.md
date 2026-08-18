# Lot A — phase 2 (WEB) : le score dans le temps s'affiche

> Worktree `../LevelUp-wt-score-web`, branche `wt/score-web`, base `fed987953`.
> Executeur : phase 2 du lot A du plan `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`.
> Aucun code Go, aucun push, aucune fusion, `routeTree.gen.ts` jamais touche.

## 0. Etat des items

| Item | Statut | Ou |
|---|---|---|
| A.2.0 Frontiere + garde etendue + helpers purs + regle `originResolved` | `[x]` | `cad702f63` |
| A.2.1 Rejeu : score vivant en-tete + fiches, retournements sur la frise | `[x]` | `a3eea81bb` |
| A.2.2 Match view : `MatchScoreCurveChart` dans le deroule du match | `[x]` | `0549a0539` |
| A.2.3 Gates + note pour la planche | `[x]` | ce document |

## 1. A.2.0 — la frontiere prend le calque de score

**Les cinq tableaux nullables passent la frontiere.** `normalizeReplayDocument` comble
`scoreTimeline.teams`, `.players`, et, dans leurs elements, `rounds`, `total`, `points`.
L'OBJET `scoreTimeline`, lui, garde le droit d'etre absent : un artefact de schema anterieur
a 12 n'en porte aucun, et un objet vide se lirait « le film a ete lu, il n'y avait pas de
score ». Types `*Ready` correspondants dans `replayNormalize.ts` ; alias de contrat
(`ReplayScoreTick`, `ReplayScoreRound`, `ReplayScoreSeries`, `ReplayTeamScore`,
`ReplayPlayerScore`, `ReplayScoreTimeline`, `ReplayScoreCoverage`) dans `lib/api/types.ts`.

**La garde `_ListeExhaustive` descend maintenant dans les etages.** Elle ne voyait que la
racine — elle avait deja laisse passer `weaponPads[].spawns` une fois, et le calque de score
empile QUATRE niveaux. Un walker de types (`NullableArrayPaths`, borne a 6 de profondeur,
ignorant les objets a signature d'index) cartographie **45 chemins** du contrat. Deux
assertions neuves :

- `_CarteExhaustive` : la carte ecrite = celle du contrat genere. Le Go publie un tableau de
  plus, ou que ce soit, et `tsc -b` refuse de compiler en NOMMANT le chemin.
- `_FrontiereProfonde` : de cette carte, le document NORMALISE ne laisse passer que
  `mapObjectives.markers` et `mapObjectives.zones` — deux chemins justifies (remplis a la
  requete, normalises par leur propre calque). C'est l'assertion qui compte : tout tableau
  nullable oublie par la frontiere, a quelque profondeur qu'il vive, y apparaitrait.

**Preuve que la garde mord** : la sonde a ete posee avant d'ecrire la liste, et c'est le
compilateur qui a dicte les 45 chemins (`tsc -p tsconfig.app.json --noErrorTruncation`).

**Le module pur `scoreTimelineLogic.ts`** (sans DOM, 30 tests) :

| Fonction | Ce qu'elle tranche |
|---|---|
| `scoreAtFrame(points, frame)` | derniere valeur <= frame ; **0 avant le premier palier** (dichotomie) |
| `teamSeriesFor` / `teamScoreAtFrame` | une equipe SANS serie vaut **zero partout** — verite du film |
| `playerCountersAt` | **`null`** pour un joueur non publie, jamais un objet de zeros |
| `roundAtFrame` | la manche courante et SA valeur (qui n'est pas le total) |
| `leadChanges` | les retournements : meneur UNIQUE, une egalite les suspend, la premiere prise de tete n'en est pas un |
| `allyOfTeamId` | le pont `teamId` du film -> allie/adverse de la page (via le scoreboard) |
| `filmClockTrusted` / `scoreTimelineOf` | la regle cliente du P2 (ci-dessous) |

**Regle `originResolved` (P2 (3) du registre).** Les calques dates par l'horloge du film sont
MASQUES quand `coverage.originResolved === false` **ET** `originMs` absent. Les DEUX
conditions sont exigees : le champ est un booleen non pointeur, donc un artefact de schema 11
servi tel quel dit `false` alors qu'il porte un `originMs` parfaitement valide — le masquer
serait perdre un calque juste. La garde est posee en UN point (`scoreTimelineOf`) pour le
score, et au sommet de `buildObjectivePulses` pour `objectives[]` : l'ecart d'origine est
mesure de 3,6 s a 50,8 s selon le match, et un pulse qui s'allume 30 s trop tot se lit comme
juste.

## 2. A.2.1 — le rejeu

**En-tete de colonne** (`ReplayTeamHeader`). Le TOTAL du match au frame courant, dans le token
du camp (`team-ally`/`team-enemy`), meme cascade de libelle qu'avant. Un mode a plusieurs
manches ajoute un rappel discret `M2 43` (infobulle « Manche 2 sur 2 : 43 »).

> **Choix ecrit (le plan laissait l'arbitrage a l'executeur)** : le TOTAL est toujours
> l'affichage principal — c'est le nombre que le jeu affiche et celui que l'oracle a valide
> (43/50, 3, 200/121) —, la manche courante n'apparait QUE s'il y en a plusieurs, en second
> et en encre attenuee. Sur un mode a manche unique elle repeterait le total.

Un camp SANS serie affiche **0** (temoin CTF 3-0 : ne pas marquer est une mesure). Mais si le
film ne publie AUCUNE serie d'equipe, la colonne n'affiche rien : « 0 » se lirait alors comme
une mesure alors que personne n'a compte.

**Fiches joueur.** Le badge affichait les totaux de FIN DE MATCH venus de la base — son propre
commentaire signalait le defaut depuis l'origine. Quand le film apparie le joueur, la fiche
montre son score personnel et ses frags/morts/assistances A L'INSTANT LU ; sinon elle garde
les totaux de la base. `null` veut dire « pas publie », jamais « a zero ». **Aucune ligne ne
s'ajoute** : la hauteur constante vivant/mort est preservee.

**Frise.** Une marque de 2 px a chaque changement de meneur, dans le token du NOUVEAU meneur,
datee en M:SS en infobulle, `pointer-events-none` (la piste reste saisissable dessous).

## 3. A.2.2 — la vue match

`MatchScoreCurveChart`, en tete du deroule du match (`sectionFlow`), pleine largeur.

- **Alimente par l'artefact**, meme endpoint et meme cle (`match-replay`) que le rejeu ;
  `useMatchReplay` recoit un `enabled` pilote par `header.replay_available` — **le meme gate
  que le lien « rejeu »**. Sans lui, chaque vue match telechargerait 1,5 a 2,7 Mio pour rien.
  La cle ne change pas : ouvrir le rejeu depuis la vue match ne refait aucun appel.
- **Rend `null`** sans artefact, sans calque, si `coverage.score.modeSupported === false`, ou
  si la garde d'horloge masque. Pas de cadre vide, pas de placeholder.
- **Escalier** (`step: 'end'`) : le film ne transmet que les changements, la valeur ATTEND
  entre deux paliers. Chaque serie est bornee — zero au coup d'envoi, dernier palier tenu
  jusqu'a la fin du match.
- **Un seul axe de valeurs** (regle dataviz), axe des temps en VALEURS (les paliers ne tombent
  pas sur une grille reguliere ; les espacer egalement mentirait sur le rythme), legende
  nommant les deux camps, tokens allie/adverse, retournements en traits verticaux poses une
  seule fois, infobulle FR/EN datee en M:SS avec les noms d'equipe echappes.
- **L'unite n'est pas recalculee** : `coverage.score.oracle` vaut `displayed`, les valeurs SONT
  le score affiche en jeu. `truncated` est DIT a l'ecran — une courbe incomplete qui a l'air
  complete est pire qu'une courbe absente.

## 4. Deux garde-rails ont mordu — suivis, pas desserres

1. **Cliquet de taille de `ReplayCanvas.tsx`** (`placementFamily.guard.test.ts`, plafond 861 :
   « le franchir se corrige en extrayant, pas en relevant le nombre »). Le fichier etait monte
   a 885. Extraction de `useLeadMarks.ts` (les trois derivations de la frise) et de
   `ReplayCountersBadge.tsx` (le badge des fiches, meme decoupage que `ReplayWeaponsRow`).
   **Retour a 861 exactement**, plafond inchange.
2. **`testDoc.guard.test.ts`** (une seule fixture de document). Le test du module pur appelait
   `normalizeReplayDocument` a la main : il passe desormais par `testReplayDoc`. L'allowlist du
   garde n'a pas ete elargie.

Un troisieme point a ete traite avant qu'il ne devienne une dette : le hook exporte depuis un
fichier de composant ajoutait un avertissement `react-refresh` (20 au lieu de 19). Il a son
propre module (`useLeadMarks.ts`, convention `useSlotIdentity.ts`) — **lint revenu a la
baseline exacte**.

## 5. Gates (joues dans la session, log persistant)

`.ai/V7.5/replay2d/registre_film/LOTA_phase2_gates.log`

```
=== [A.2.3] GATES DE CLOTURE — 2026-08-18 ===
EXIT_TSC=0        (npx tsc -b --force, apres purge de node_modules/.tmp)
EXIT_VITEST=0     73 fichiers, 930 tests (src/features/match-replay src/features/match-view)
EXIT_LINT=0       19 avertissements (baseline exacte), 0 erreur
```

`routeTree.gen.ts` : intouche (verifie `git diff HEAD --name-only`).

Tests ajoutes : **71** (scoreTimelineLogic 30, replayContract +4, ReplayTeams +12,
ReplayLeadMarks 10, _scoreCurve 18, MatchScoreCurveChart 13 — moins les recouvrements).

## 6. Note pour la planche (verification visuelle = la main de l'utilisateur)

Artefacts temoins sous `../LevelUp-wt-score-film/data/cache/replays/halo_infinite/`.
Valeurs calculees sur les artefacts reels — **c'est ce qui doit s'afficher**.

### `000d5950` — Slayer, 43/50, 8:18 (4 985 images x 100 ms)

| Instant | Equipe 0 | Equipe 1 |
|---|---|---|
| 0:00 | 0 | 0 |
| 2:05 | 6 | 10 |
| 4:09 | 20 | 23 |
| 6:14 | 28 | 37 |
| 8:18 (fin) | **43** | **50** |

- **En-tete** : les deux nombres tiquent en continu ; aucun rappel de manche (mode a manche unique).
- **Fiches** : **6 joueurs sur 8** montrent un score personnel + frags/morts/assistances vivants ;
  les **2 autres gardent les totaux de la base** (aucun zero invente — c'est le cas a regarder).
  Exemple a la fin : `2533274815845110` -> score 1520, 12 frags, 10 morts, 6 assistances.
- **Frise** : **aucune marque** — le camp gagnant mene de bout en bout (deux egalites, a 1 et a 2).
- **Dominance** : deux escaliers qui ne se croisent jamais, celui de l'equipe 1 toujours au-dessus.

### `530820e5` — CTF, 3-0, 7:55 (4 751 images)

| Instant | Equipe 0 | Equipe 1 |
|---|---|---|
| 0:00 | 0 | 0 |
| 1:59 | 0 | 0 |
| 3:58 | 2 | 0 |
| 7:55 (fin) | **3** | **0** |

- **Le cas a verifier en priorite** : le film ne publie **qu'UNE serie d'equipe**. La colonne du
  camp a 0 doit afficher **`0`**, et la courbe Dominance doit tracer **DEUX lignes** dont une
  plate a zero. Sans elle, un 3-0 se lirait comme un match a un seul participant.
- **Fiches** : **8 joueurs sur 8** ont des compteurs vivants (exemple : score 1735, 11 frags,
  15 morts, 4 assistances).
- **Frise** : aucune marque (un seul camp marque).
- Cet artefact porte aussi **183 actions d'objectif** : les pulses du canvas restent visibles
  (la garde d'horloge ne les masque pas, `originResolved = true`).

### `24dbb67d` — Oddball, 200/121 en 2 manches, 8:34 (5 137 images)

| Instant | Equipe 0 (total / manche) | Equipe 1 (total / manche) |
|---|---|---|
| 0:00 | 0 / M1 0 | 0 / M1 0 |
| 2:08 | 53 / M1 53 | 7 / M1 7 |
| 4:17 | 92 / M1 92 | 78 / M1 78 |
| 6:25 | 118 / **M2 18** | 105 / **M2 27** |
| 8:34 (fin) | **200** / M2 100 | **121** / M2 43 |

- **Le temoin des manches** : l'en-tete affiche le TOTAL en grand et `M2 18` en petit apres le
  passage a la seconde manche (vers 4:40) — la valeur de manche **repart de zero**, le total non.
- **Le temoin des retournements** : **3 marques** sur la frise, a **1:05** (equipe 0 passe),
  **3:01** (equipe 1 reprend), **3:55** (equipe 0 repasse). Les memes trois traits verticaux
  doivent apparaitre sur la courbe Dominance.
- **Fiches** : **aucun** joueur n'a de compteurs vivants (Oddball : 0/32 mesure en phase 0) —
  toutes les fiches gardent les totaux de la base. C'est le cas negatif a verifier : rien ne
  doit s'afficher a zero.

### Ce qui ne doit PAS apparaitre

- Sur un match **sans artefact de rejeu** (la quasi-totalite en production) : la vue match doit
  etre **exactement** comme avant — aucune carte « Score dans le temps », aucun cadre vide.
- Sur un artefact dont l'origine n'est pas resolue : ni score, ni pulses d'objectif.

## 7. Decouvertes (notees, NON traitees — hors perimetre)

- **`match-replay/i18n.ts` a 505 L** (etait 471). Fichier-dictionnaire : la parite FR/EN est
  garantie par le typage `Record<Locale, T>`, qui exige un seul objet. Le dictionnaire de
  `match-view` fait 833 L — c'est le regime etabli du depot pour cette forme de fichier.
- **`match-view/MatchViewPage.tsx` : 511 -> 523 L**, deja au-dessus du seuil avant ce lot.
  Aucun cliquet ne le garde. L'ajout est l'invocation du composant (6 props) — irreductible
  sans extraire la section, ce qui sort du perimetre.
- **`match-replay/ReplayTeams.test.tsx` a ~730 L.** Les deux blocs de score y ont ete ajoutes
  plutot que dans un fichier neuf pour **reutiliser `sbRow` et `TRACK`** : une copie de plus du
  constructeur de ligne de scoreboard aurait ete la 3e de la feature (cf. point suivant).
- **`sbRow` existe deja en 3 exemplaires** (`ReplayTeams.test.tsx`, `ReplayKillFeed.test.tsx`,
  `match-view/MatchCombatCtfOverlay.test.tsx`) avec des signatures divergentes. La regle
  « <= 2 copies » est deja franchie AVANT ce lot ; la centralisation (dans `test/`, avec son
  garde-rail) est un chantier a part.
- **Deux formateurs M:SS preexistants restent en place** : `MatchTugOfWarChart.formatMmSs`
  (secondes) et `replayLogic.formatSeconds`. Ce lot n'en a pas ajoute un troisieme — il a pose
  `lib/formatters/formatClockMMSS` (millisecondes, 0 = coup d'envoi et non « - ») et l'a
  branche sur ses DEUX appelants. Rapatrier les deux preexistants dessus serait un fix hors
  perimetre.
- **Les deux autres P2 du registre ne sont pas web et restent ouverts** : (1) l'ouvrier distant
  construit sans les faits du match, (2) `RealRounds` admet la manche du trou. Rien dans ce lot
  ne les touche.

## 8. Ce qui n'a PAS ete fait

Rien du perimetre A.2.x n'est reporte. Le **gate visuel** (planche + en app) appartient a
l'utilisateur, comme le prevoit le plan (Gate 2), et la **fusion `--no-ff`** au superviseur.
