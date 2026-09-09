# Plan — Vague A : le lot court (7 correctifs prouvés)

> Issu de `.ai/diagnostics/RETOURS_2026-09-08/DIAGNOSTIC_RETOURS_UTILISATEUR_2026-09-08.md`.
> **Exécution sous le contrat du skill `plan-execution`** — ordre strict, aucun report, chaque item
> statué. Ce plan ne redit pas le contrat, il s'y soumet.

## Objectif et critère de succès

Fermer les **sept irritants dont la cause racine est établie ET mesurée**, sans en instruire aucun
nouveau. Aucun de ces correctifs ne touche la donnée, le décodeur, ni un contrat d'API.

**Critère de succès, vérifiable :** les sept gates d'étape passent, `make check-types` et
`make test-web` sont verts, et la mesure d'avant/après de A1 (la seule qui se chiffre) montre
`scrollWidth === clientWidth` sur le panneau des équipes.

**Effort** : rapide (six items d'une à quinze lignes ; A6 est le seul « moyen »).
**Branche** : `feat/retours-lot-court` depuis `feat/v75`.
**Worktree** : `LevelUp-wt-lot-court` / `wt/lot-court` — dédié, conformément à la règle du dépôt
(ne jamais travailler dans le worktree partagé de session).

## Décisions PRISES — rien à trancher en cours de route

| # | Décision | Valeur retenue |
|---|---|---|
| D1 | Largeur du trait de lecture (A3) | `w-[3px]`, opacité `bg-foreground/55` — reste une couleur structurelle, exception assumée déjà documentée dans le fichier |
| D2 | Ancrage de l'écran de fin (A4) | L'overlay descend **dans** un conteneur `relative` qui n'enveloppe QUE `ReplayCanvas`. La bannière de score et la frise restent dehors |
| D3 | Encre neutre des zones (A6) | **Nouveau** jeton `zone-neutral` (remplissage) + `zone-neutral-outline` (contour). Ni `divergent-neutral` (`#60A5FA`, bleu) ni `outcome-draw` (`#3B82F6`, bleu) ne conviennent : les deux sont bleus, c'est exactement le défaut signalé |
| D4 | Décalage du glyphe de drapeau (A7) | `FLAG_OFFSET_*` **conditionné à l'état `carried`**. Refusé : décaler l'anneau — le rayon de la zone de retour (1,3 m) est une distance de jeu mesurée, son centre doit rester la position vraie |
| D5 | Onglet Tactique en L1 (A2) | Ajouté avec `capability: 'replay'`, **en dernière position**, pour être le miroir exact de `AscensionLayout.tsx:128` |
| D6 | Médailles de la frise en images | **HORS de cette vague.** La piste doit être élargie pour accueillir un badge : ce n'est plus un correctif court. Va en vague C avec les autres travaux de forme |

## Règles d'exécution

- **Ordre strict.** L'étape N+1 ne commence pas avant que le gate de N soit passé.
- **Statuts** : `[x]` fait · `[~]` couvert ailleurs (avec la référence) · `[!]` non traité (avec la
  justification écrite). **Aucune case vide à la clôture.**
- **Zéro fix hors périmètre.** Toute découverte va en § Découvertes, sans être traitée.
- **Une seule revue adversariale, en fin de vague** — pas par étape (règle du dépôt).
- **Reprise de session** : lire les cases cochées de ce fichier, puis `git log --oneline -10` sur
  `wt/lot-court`. L'étape en cours est la première non cochée.

---

## Étape A1 — Les fiches d'équipe rognées

**Périmètre fermé (1 fichier) :**

- [ ] `apps/web/src/features/match-replay/ui/ReplayTeams.tsx:184` — remplacer
      `repeat(${groups.length}, 1fr)` par `repeat(${groups.length}, minmax(0, 1fr))`
- [ ] Ajouter au commentaire voisin la raison en une phrase : `1fr` vaut `minmax(auto, 1fr)`, et
      le minimum `auto` est le min-content de la piste, que le `truncate` du nom (donc
      `white-space: nowrap`) rend égal au texte entier
- [ ] Étendre `ui/ReplayTeams.density.test.tsx` (ou le fichier de test le plus proche) d'un cas qui
      échoue sans le correctif : deux colonnes dans un conteneur étroit, `scrollWidth` attendu égal
      à `clientWidth`

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/ui/ReplayTeams
```
Et la mesure dans le navigateur (rejeu Origin, viewport 1600×950) :
```js
const g=[...document.querySelectorAll('div')].find(d=>d.className.toString().includes('grid h-full min-h-0 gap-2.5'));
({tpl:getComputedStyle(g).gridTemplateColumns, cw:g.clientWidth, sw:g.scrollWidth})
// attendu : sw === cw (avant correctif : cw 480 / sw 563)
```

## Étape A2 — L'onglet Tactique dans le menu déroulant L1

**Périmètre fermé (3 fichiers + 1 régénération) :**

- [ ] `apps/web/src/lib/i18n/manifests/common.toml` — ajouter `[common.nav.tab_tactique]` avec
      `fr = "Tactique"` et `en = "Tactical"` (parité FR/EN obligatoire)
- [ ] Régénérer : `node apps/web/scripts/build_i18n_manifests.mjs` — le fichier généré
      `apps/web/src/lib/i18n/generated/common.ts` est commité (ADR 0003)
- [ ] `apps/web/src/components/shell/navL1Sections.tsx` — section `ascension`, ajouter en **dernier**
      onglet : `{ key: 'tactique', labelKey: 'common.nav.tab_tactique', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tactique', capability: 'replay' }`
- [ ] Étendre `components/shell/NavL1.test.tsx` : avec la capability `replay`, la section Ascension
      expose cinq onglets et le dernier est Tactique ; sans elle, quatre

**Gate :**
```bash
cd apps/web && npx vitest run src/components/shell/NavL1
git diff --stat apps/web/src/lib/i18n/generated/common.ts   # doit etre non vide
```

## Étape A3 — Élargir le trait de lecture

**Périmètre fermé (1 fichier) :**

- [ ] `apps/web/src/features/match-replay/ui/ReplayPlayhead.tsx` — `w-px` → `w-[3px]`,
      `bg-foreground/40` → `bg-foreground/55` (décision D1)
- [ ] Vérifier que `timelineGeometry.guard.test.ts` reste vert : le trait fait désormais 3 px, il
      n'est plus « sa gauche = son centre ». **Si le garde-rail casse**, ajouter
      `-translate-x-1/2` — c'est la même translation que les marques, et le commentaire du fichier
      l'anticipe (« la translation vit là où une largeur la rend nécessaire »)

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/ui/timelineGeometry src/features/match-replay/ui/ReplayTimelineTracks
```

## Étape A4 — Centrer l'écran de fin sur la carte

**Périmètre fermé (1 fichier) :**

- [ ] `apps/web/src/routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx` —
      envelopper **le seul** `<ReplayCanvas>` dans un `<div className="relative">` et y déplacer
      `<ReplayVictoryOverlay>` (décision D2)
- [ ] Vérifier que l'overlay garde `pointer-events-none` : la frise doit rester saisissable dessous
- [ ] Étendre `ui/ReplayVictoryOverlay.test.tsx` d'une assertion de centrage relatif au canvas

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/ui/ReplayVictoryOverlay
```
Et la mesure dans le navigateur, en fin de rejeu :
```js
const ov=[...document.querySelectorAll('div')].find(d=>d.className.toString().includes('absolute inset-0 z-10 flex items-center justify-center overflow-hidden'));
const cv=document.querySelector('canvas'); const c=e=>Math.round(e.getBoundingClientRect().top+e.getBoundingClientRect().height/2);
({overlay:c(ov), carte:c(cv)})   // attendu : ecart < 5 px (avant : 530 vs 441)
```

## Étape A5 — Le message d'état vide de « Distance par arme »

**Périmètre fermé (1 fichier, 2 locales) :**

- [ ] `apps/web/src/features/match-view/i18n.ts:399` (FR) et `:713` (EN) — remplacer le motif
      « le décodage du film […] n'a pas encore été joué ici », **factuellement faux** (le match
      Origin porte 65 positions de kill), par un motif qui dit la vraie cause : la passe de
      décodage n'a pas autorisé la publication ligne par ligne pour ce match
- [ ] Ne PAS toucher au reste de `MatchKillDistanceSection` : le comportement est correct, seul le
      texte ment

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-view/MatchKillDistanceSection
grep -n "pas encore été joué" apps/web/src/features/match-view/i18n.ts   # doit ne rien rendre
```

## Étape A6 — L'encre neutre des zones (le seul item « moyen »)

**Périmètre fermé (4 fichiers) :**

- [ ] `apps/web/src/styles/globals.css` — ajouter `--ac-zone-neutral` et `--ac-zone-neutral-outline`
      dans **les trois blocs** (`:root`, `@media (prefers-color-scheme: dark)` guardé
      `:root:not([data-theme="light"])`, `:root[data-theme="dark"]`) — jamais une seule définition
      dans un bloc conditionnel
- [ ] `apps/web/src/lib/accessibility/semantic-tokens.ts` — déclarer les deux jetons dans l'union
      `SemanticToken` et dans la liste de validation
- [ ] `apps/web/src/features/match-replay/layers/objectivesLayer.ts` — `ObjectivesStyle` gagne un
      `neutralOutline: string` ; `drawZone` et `drawMarker` l'emploient **uniquement** quand
      `e.team === -1`
- [ ] `apps/web/src/features/match-replay/layers/useZoneStates.ts` — fournir les deux encres
- [ ] Étendre `layers/objectivesLayer.test.ts` : une zone `team === -1` peint le remplissage neutre
      ET un contour d'encre distincte

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/layers/objectivesLayer
npm --prefix apps/web run lint:colors     # aucune couleur en dur introduite
```

## Étape A7 — Le glyphe de drapeau recentré hors portage

**Périmètre fermé (1 fichier) :**

- [ ] `apps/web/src/features/match-replay/layers/flagCarriesLayer.ts` — `FlagGlyphPaint` gagne un
      booléen `offset` ; `drawFlagGlyph` (l. 384-385) n'applique `FLAG_OFFSET_X/Y` que s'il est vrai
- [ ] L'appelant le met à vrai **pour le seul état `carried`** (décision D4) — `dropped`, `home` et
      `carried_open` ancrent sur la position brute
- [ ] Aligner la cible de survol (l. 443-444) sur la même règle, sinon les trois ancrages
      resteraient distincts
- [ ] Étendre `layers/flagCarriesLayer.test.ts` : à `dropped`, le pied de la hampe est à la position
      projetée exacte ; à `carried`, le décalage subsiste

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/layers/flagCarriesLayer src/features/match-replay/layers/flagReturnZone
```
Et de visu : rejeu Origin `8bc6074f`, image **790** — l'anneau et le pied du drapeau partagent leur
centre.

---

## Étape A8 — Les largeurs de fiches ne bougent plus quand le texte s'allonge

> Retour utilisateur du 2026-09-09 : « "Inventaire indisponible" fait aussi agrandir les largeurs
> de fiches, il faut que les largeurs de fiches soient fixes !! si du contenu apparaît en texte
> qui est plus long faut le tronquer ! »

**Ce n'est PAS le défaut de A1** : A1 portait sur les colonnes de CAMPS, celui-ci sur la largeur
d'une FICHE. Les deux correctifs sont indépendants.

**Ce que la lecture du code établit** (à re-vérifier sur pièces avant de coder — règle
d'exécution n°4, le code a pu bouger) :

- Le badge est **déjà** `min-w-0 truncate` (`ui/ReplayInventoryRow.tsx:436-440`) et son
  commentaire affirme qu'il « occupe la place qui reste et ne décale donc rien ». Le symptôme
  prouve que cette affirmation est fausse en pratique : **la chaîne de contrainte est rompue plus
  haut**, pas dans le badge.
- **Suspect n°1** — `ui/ReplayInventoryRow.tsx:138` : le conteneur de la rangée est
  `flex items-center gap-[5px] font-mono text-[9.5px] …`, **sans `min-w-0` ni `overflow-hidden`**,
  alors que toutes ses autres cellules sont `shrink-0`. Un élément flex a `min-width: auto` par
  défaut : un seul enfant souple ne suffit pas si le conteneur lui-même ne peut pas rétrécir.
- **Suspect n°2** — `ui/ReplayTeams.tsx:84` : `grid-cols-[repeat(auto-fill,minmax(115px,1fr))]`.
  Bornée en apparence. À confirmer que c'est bien ce conteneur qui sert au gabarit où le défaut
  se voit (l'autre branche est `SEATS_COLUMN_CLASS`, ligne 82).

**Périmètre fermé (1 fichier attendu + tests) :**

- [x] **MESURER D'ABORD**, dans le navigateur, quel maillon s'élargit — ne pas poser `min-w-0`
      sur les deux « au cas où » (règle : zéro fix opportuniste)
- [x] Poser `min-w-0` (et `overflow-hidden` seulement si la mesure le montre nécessaire) sur le
      SEUL maillon que la mesure désigne
- [~] Test qui échoue sans le correctif : une fiche portant l'état vide « Inventaire
      indisponible » n'est pas plus large que la même fiche sans
- [x] Vérifier que le badge reste TRONQUÉ et que son infobulle porte le texte entier — contrat
      existant, `ui/ReplayTeams.test.tsx:1175`, à ne pas casser

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/ui/ReplayTeams src/features/match-replay/ui/ReplayInventoryRow
```
Et la mesure dans le navigateur (un rejeu où un joueur est mort, viewport 1600×950) :
```js
// largeur des fiches AVEC et SANS le badge d'état vide, sur la même rangée
[...document.querySelectorAll('[class*="relative flex shrink-0 flex-col rounded"]')]
  .map(c => ({ w: c.getBoundingClientRect().width, vide: c.textContent.includes('Inventaire') }))
// attendu : toutes les largeurs EGALES, que `vide` soit vrai ou faux
```


### Journal A8 — CLOSE le 2026-09-09

**Le plan visait un suspect, la mesure en a designe un autre — et a montre que le defaut annonce
etait deja corrige.** Mesure faite dans le navigateur sur la feuille de style REELLE du depot
(la page de connexion la charge ; chaine de classes reproduite a l'identique depuis les sources) :

| Banc | Colonnes de camps | Fiche |
|---|---|---|
| `repeat(2, 1fr)` sans badge | `235px 235px` | 235,0 |
| `repeat(2, 1fr)` **avec** badge | `235.141px 234.859px` | 235,1 |
| `minmax(0, 1fr)` sans badge | `235px 235px` | 235,0 |
| `minmax(0, 1fr)` **avec** badge | `235px 235px` | 235,0 |

**A1 couvrait deja la croissance de largeur.** La fuite de min-content etait reelle en `1fr` ; le
`minmax(0, 1fr)` d'A1 la ferme, badge ou pas.

**Mais un second defaut, independant, subsistait** : le badge ne se tronquait JAMAIS. Son
conteneur (`ReplayInventoryRow.tsx:137`) est un element flex a `min-width: auto` sans `min-w-0` :
il prenait sa largeur hypothetique, texte entier compris, et debordait la rangee parente que
`overflow-hidden` clippait en silence. **Rangee de 102 px pour 221 px de contenu — 119 px coupes
net, sans points de suspension, cellules suivantes disparues.** Avec `min-w-0` : debordement 0,
troncature active (`badge_tronque` faux -> vrai).

**Suspect n°2 du plan (la piste `minmax(115px,1fr)`) : ECARTE par la mesure** — la fiche fait
115,5 px dans les quatre bancs, elle ne grandit pas. Aucune modification faite de ce cote.

**Item `[~]`** — le test de largeur demande est irreproductible : jsdom ne met rien en page. La
mesure navigateur ci-dessus tient le role de preuve du defaut ; le garde-rail versionne
(`ReplayTeams.test.tsx`, describe « la rangee d'inventaire se laisse retrecir ») fixe la CAUSE,
meme doctrine que le garde-rail voisin de la grille des camps.

**Fixations DOM regenerees** — diff verifie : **18 ajouts de `min-w-0`, aucune autre difference**.

**Gate passe** : `npx vitest run src/features/match-replay/ui/ReplayTeams` -> 79 passes,
3 skippes, **0 echec**.

## Étape A9 — Aucun véhicule ne peut paraître plus petit qu'un pion

> Retour utilisateur du 2026-09-09 : « sur le dernier match sur Isolement je vois les véhicules,
> le Ghost mais il est plus petit que le pion du joueur ça fait hyper bizarre. »

**La cause est écrite dans le code** (`model/vehiclesLayer.ts:313-368`) :

- `PION_REFERENCE_PX = CORE_RADIUS * 2` = **6,8 px**, le DIAMÈTRE du noyau d'un pion.
- `VEHICLE_FLOOR_PX = PION_REFERENCE_PX`, et ce plancher porte sur la **LONGUEUR** du véhicule.
  Un sprite allongé dont la longueur est ramenée à 6,8 px a une **largeur bien inférieure** à
  6,8 px : il occupe moins de surface qu'un pion et se lit « plus petit ». **Le plancher garantit
  une longueur, pas une masse visuelle.**
- Et la raison pour laquelle le Ghost y tombe : le commentaire de `MONGOOSE_REFERENCE_LENGTH_MM`
  dit que **le Mongoose et le Warthog sont les DEUX SEULES familles dont l'échelle est garantie**
  (`V4_RAPPORT_SPRITES_2026-08-31.md`) ; les **12 autres portent une valeur provisoire**. Le Ghost
  en fait partie — sa longueur à l'écran n'est pas mesurée.

**Décision tranchée pour cette vague** : deux corrections existent, une seule y tient.

| | Correction | Verdict |
|---|---|---|
| (a) | Mesurer le mm/px du Ghost et des 11 autres familles | **HORS vague A** — chantier de sprites, données à produire |
| (b) | Exprimer le plancher sur la dimension la PLUS PETITE du sprite | **Retenue** — forme pure, un fichier |

**Périmètre fermé (1 fichier + tests) :**

- [~] `model/vehiclesLayer.ts` — le plancher garantit que la LARGEUR rendue
      (`naturalWidthPx / naturalHeightPx × longueur`) ne descend pas sous `PION_REFERENCE_PX`.
      `naturalWidthPx` est **déjà** dans `VehicleSpriteSize` (ajouté le 2026-09-03 pour les ancres
      d'armement) : aucune donnée nouvelle, aucun chargement de plus
- [x] Redocumenter `VEHICLE_FLOOR_PX` dans le MÊME commit — la doc d'un seuil se met à jour quand
      le seuil change (anti-pattern n°9, « doc inversée »)
- [x] Ne PAS toucher au plafond doux `VEHICLE_SOFT_CEIL_PX` ni à `VEHICLE_PX_PER_MM` : les tailles
      RELATIVES entre familles doivent continuer à suivre le manifeste
- [~] Test : un sprite très allongé au mm/px provisoire rend une largeur ≥ au diamètre du pion ;
      un Mongoose (échelle garantie) garde EXACTEMENT sa taille d'avant

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/model/vehiclesLayer
```
Et le contrôle visuel sur le dernier match Isolement : le Ghost doit se lire au moins aussi gros
qu'un pion, sans que le Mongoose ni le Warthog n'aient bougé.

**Hors périmètre, consigné** : l'arme inconnue de ce même match (« ça a l'air d'être le
mutilateur ») est une question de DONNÉE, pas de forme — portée au registre, point 19.


### Journal A9 — CLOSE le 2026-09-09 (après arbitrage utilisateur)

**La première version de ce journal concluait à un blocage : la prémisse du plan était fausse.
Elle l'était bien, et voici ce que la mesure a mis à la place.**

**Ce que le plan supposait, et qui est réfuté :**

1. *« Le Ghost n'a pas d'échelle mesurée »* — FAUX. Le manifeste
   `static/vehicles-assets/halo_infinite/replay/index.json` le porte en `statut: "valide"`,
   **9,99 mm/px vérifié le 2026-09-02**. Les **18** familles sont mesurées
   (`ECHELLES_SPRITES_2026-09-02.md`). Le commentaire du code qui disait « avec le Warthog, la
   seule famille garantie » était périmé — retiré dans ce même commit (anti-pattern n°9).
2. *« Le Ghost tombe sur le plancher »* — FAUX. Tailles rendues calculées depuis les PNG réels,
   plancher à 6,80 px : mongoose 11,90 × 7,25 · warthog 20,64 × 10,41 · **ghost 15,71 × 13,02** ·
   banshee 23,80 × 22,41 · scorpion 36,07 × 24,17. **Aucune famille ne l'atteint** — la plus
   petite en est à 75 % au-dessus. Le correctif prévu (plancher sur la plus petite dimension)
   n'aurait rien changé.

**La vraie cause : l'ANCRE.** `PION_REFERENCE_PX = CORE_RADIUS * 2` = 6,80 px n'est pas la taille
visible d'un pion : au rez-de-chaussée il mesure `CORE_RADIUS + OUTLINE_PAD` de rayon, soit
**8,80 px**, et davantage avec ses anneaux d'étage. La cible « Mongoose = 1,75 pion de long » se
calculait contre une référence 1,29 fois trop petite — le Mongoose sortait à **7,25 px de large,
plus étroit que le pion** dont il est censé faire 1,75 fois la longueur.

**Arbitrage utilisateur du 2026-09-09** : re-ancrer toutes les familles. Ancre retenue = **le pion
visible, noyau + lisere (8,80 px)**, soit un facteur **×1,29** — l'option « anneau » (×1,91) a
été écartée, elle aurait porté le Scorpion à 69 px et écrasé la carte.

**Tailles après re-ancrage** (plafond doux porté à 61,60 px, aucune famille ne l'atteint) :

| Famille | longueur | largeur | ≥ pion (8,80) |
|---|---|---|---|
| mongoose | 15,40 | 9,38 | oui |
| ghost | 20,33 | 16,84 | oui |
| warthog | 26,71 | 13,48 | oui |
| banshee | 30,80 | 29,00 | oui |
| scorpion | 46,68 | 31,28 | oui |

**Forme du correctif (2 fichiers) :** `replayMarkers.ts` exporte `PION_VISIBLE_DIAMETER_PX`
(la valeur que `markerEdge` calcule déjà — pas une seconde vérité) ; `vehiclesLayer.ts` s'y
ancre. **`VEHICLE_PX_PER_MM` et `VEHICLE_SOFT_CEIL_PX` ne sont pas touchés** : ils sont dérivés
de l'ancre, leurs valeurs suivent donc le ×1,29 mais leurs formules et les **proportions
relatives entre familles restent identiques** — c'est ce que l'item l'interdisant protégeait.

**Items `[~]`** — deux énoncés du plan sont devenus caducs : le plancher sur la plus petite
dimension (remplaçé par le re-ancrage, l'invariant « jamais plus fin qu'un pion » est désormais
tenu par un test dédié), et « le Mongoose garde EXACTEMENT sa taille d'avant » (faux **par
construction** : le re-ancrage le fait grandir, c'est l'objet du correctif).

**Deux garde-rails posés** dans `vehiclesLayer.test.ts` : l'ancre est le pion VISIBLE et jamais
son seul noyau ; la plus petite famille reste au moins aussi LARGE qu'un pion.

**Gates passés** : `vitest` vehiclesLayer + vehicleWeaponMounts + vehiclesPaint → 94 passés.
`make check-types` → vert. `make test-web` → **658 fichiers, 7 035 tests, 0 échec**.
`lint:colors` → 0 violation.

## Clôture de vague

- [ ] `make check-types` vert
- [ ] `make test-web` vert
- [ ] `npm --prefix apps/web run lint:colors` vert
- [ ] **Une** revue adversariale (`adversarial-review`) sur le diff complet de la vague
- [ ] Entrée `.ai/thought_log.md` : items traités, statuts, découvertes consignées
- [ ] `delivery-checklist` avant la demande de merge

**Aucun test Go n'est requis** : la vague ne touche pas `apps/go-api/`.

## Découvertes (à remplir pendant l'exécution — NE PAS TRAITER)

_(vide au démarrage)_
