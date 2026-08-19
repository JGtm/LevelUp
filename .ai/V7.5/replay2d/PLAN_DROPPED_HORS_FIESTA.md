# PLAN — Les objets de PUISSANCE lâchés à la mort, hors Fiesta (rejeu 2D, lot web R2)

> Branche `wt/dropped-hors-fiesta`, base `feat/v75` = `e4a15e7c6`. Worktree frère
> `C:\Users\Guillaume\Projects\LevelUp-wt-dropped`. Contrat d'exécution : skill
> `plan-execution` (ordre strict, aucun report d'une action faisable maintenant).

## 1. La décision produit

Registre des reports, ligne du 2026-08-18 :

> DECISION PRODUIT (utilisateur, 2026-08-18) : HORS Fiesta, dessiner les armes speciales et
> equipements LACHES a la mort d'un joueur — « hors Fiesta c'est tactiquement interessant
> (power-ups, armes de puissance, equipements laches a la mort) ».

Elle amende la règle R2/W4 (« on n'affiche pas les objets lâchés »), qui reste entière EN
Fiesta.

## 2. Ce que la lecture sur pièces a établi (avant de coder)

### 2.1 Ce que le document publie — et ce qu'il ne publie pas

`equipmentPlacements` porte `origin: deployed | dropped | unknown` et la `family` du
manifeste. Le filtre actuel (`placementIsDeployedObject`, `equipmentPlacementsLayer.ts`) ne
dessine que `deployed` ; `PLACEMENT_RENDER` met les grenades et les capacités à `null`
explicite, et laisse les deux power-ups HORS table.

**Les ARMES lâchées à la mort ne sont PAS dans le document.** `equipmentPlacements` est un
canal d'ÉQUIPEMENT (tag `eqip`) ; `weaponPads` est un canal de SOCLES (position récurrente
d'apparition), pas de lâchers. Aucun champ ne publie « telle arme est au sol en (x, y) depuis
telle image ». **Les armes de puissance lâchées demandent donc une publication de données
supplémentaire, HORS de ce lot web** → report au registre.

Le lot se limite aux ÉQUIPEMENTS et POWER-UPS lâchés, familles du manifeste.

### 2.2 Recensement des deux témoins (mesuré, `data/cache/replays/halo_infinite/`)

| témoin | schéma | poses | familles `dropped` présentes |
|---|---|---|---|
| `01e1f945` (KOTH Catalyst) | 16 | 151 | `powerup_overshield` 1 · `grapple` 9 · `grenade_frag` 108 · `grenade_plasma` 6 · `grenade_spike` 5 · `repulsor` 11 · `other` 4 |
| `000d5950` (Super Fiesta Cliffhanger) | 17 | 295 | `sensor` 15 · `wall` 11 · `grapple` 29 · `thruster` 27 · grenades 168 |

Conséquences chiffrées, attendues du lot :
- `01e1f945` : **+1 primitive** (le surbouclier lâché, `t0 = 4104`, poseur slot 589) ;
- `000d5950` : **+0** — la garde Fiesta annule les 26 (`sensor` 15 + `wall` 11) qui seraient
  sinon dessinés. C'est l'assertion qui fait de ce témoin une mesure et pas un décor.

Les identifiants confirment qu'un lâcher n'est PAS l'objet actif : les 11 `wall/dropped`
portent l'identifiant de l'APPAREIL (`0x8e2dc574`), jamais celui des panneaux
(`0x528fce46`). Dessiner l'arc du mur pour un appareil au sol serait un mensonge — d'où une
forme DISTINCTE et unique pour tous les lâchés (§ 3.2).

### 2.3 Comment le web connaît (mal) le MODE — la question centrale

- **Le document de rejeu ne porte AUCUN mode** : `ReplayDocument` (schéma généré) n'a ni
  `mode`, ni `playlist`, ni `pair_name` ; `Coverage` non plus. Le calque ne peut donc pas
  décider seul.
- **La PAGE, elle, a la Match View** : la route `.../replay.tsx` appelle déjà
  `useMatchView(playerSlug, matchId)` (pour le fil et les fiches). Son `header` porte
  `mode_ui` et `playlist_label`.
- `playlist_label` est INUTILE : les deux témoins valent « Quick Play » (mesuré en base).
- `mode_ui` = `NormalizeModeLabel(pair_name)`. Il conserve l'identité de playlist pour
  `Super Fiesta` (428 matchs du corpus) et rend `Castle Wars` (1), mais pour un `pair_name`
  « Fiesta:Slayer on … » il extrait le SOUS-mode (« Slayer » → FR « Assassin ») : **3 matchs
  du corpus perdent l'indice**. Vérifié en base sur les 1 855 matchs.
- `expected_stats.hist_mode_category` est `omitempty` et conditionné à l'existence de matchs
  historiques de la même catégorie : pas un oracle.

**RÈGLE RETENUE** — celle du repli écrit dans la commande : *ne dessiner les lâchés QUE si le
match ne porte AUCUN indice Fiesta*. Indices = les libellés de la catégorie Fiesta canonique
(`modePrefixToCategory`, Go) : `Fiesta`, `Super Fiesta`, `Castle Wars` — identiques FR et EN
(`assets.toml`). Husky Raid en est EXCLU : le Go le promeut en catégorie propre et ce n'est
pas un mode à équipement aléatoire.

Corollaires assumés, tous deux écrits dans le code :
1. **Match View absente ou en cours de chargement = indice INCONNU = on ne dessine pas.** Une
   absence n'est pas une preuve de non-Fiesta.
2. **Trou mesuré : 3 matchs sur 432 de catégorie Fiesta (0,7 %)** — ceux dont le `pair_name`
   commence par `Fiesta:` — verraient leurs lâchés. Report au registre : fermer le trou
   demande de publier la catégorie de mode (ou `pair_name`) dans l'en-tête Match View, côté Go.

La bascule reste donc ALLUMÉE par défaut (demande utilisateur), mais elle ne commande
quelque chose que hors Fiesta.

## 3. Ce qu'on livre

### 3.1 Les familles de PUISSANCE dont le lâcher se dessine

`placementDropped.ts` (module neuf, sans cycle : il ne dépend que des types et de
`weaponPadFamilies`) :
- les cinq équipements DÉPLOYABLES du manifeste : `wall`, `sensor`, `translocator_beacon`,
  `threat_seeker`, `repair_field` ;
- les deux POWER-UPS, repris de `PAD_EQUIPMENT_FAMILIES` (`weaponPadFamilies.ts`) — la liste
  explicite que l'utilisateur a demandée le 18/08, jamais recopiée une 3e fois ;
- **jamais** les grenades ni les capacités (`grapple`, `thruster`, `repulsor`) — les `null`
  explicites de `PLACEMENT_RENDER` — ni `other`, dont la bascule de diagnostic ne change pas.

### 3.2 Le rendu

Une règle de rendu NEUVE (`PlacementKind = 'dropped'`), une seule forme pour toutes les
familles : petit anneau POINTILLÉ à alpha réduit, encre du camp du lâcheur (neutre sinon).
Un objet au sol n'est pas un objet actif : ni disque de portée, ni arc, ni pulsation.

### 3.3 L'infobulle

Famille (libellé bilingue : `placementFamily[kind]` pour les équipements,
`padEquipmentFamily[famille]` pour les power-ups), « lâché par X » (le champ `owner` du
document, -1 = non mesuré), et l'instant du lâcher au chronomètre du rejeu (`formatClock`
sur `t0`) — un instant absolu, jamais un âge qui se périme sous un pointeur immobile.

### 3.4 La bascule

`replay-show-dropped-placements`, persistée, ALLUMÉE par défaut, visible seulement quand le
film porte au moins un lâcher de puissance ET que le match n'est pas Fiesta.

## 4. Étapes

### Étape 1 — la règle et la forme (logique pure)
- [ ] `placementDropped.ts` : `PLACEMENT_ORIGIN_DROPPED`, `PLACEMENT_DROPPED_FAMILIES`,
      `placementIsDroppedPower()`.
- [ ] `placementShapes.ts` : `drawDroppedObject()` + ses constantes (aucune couleur écrite).
- [ ] `equipmentPlacementsLayer.ts` : `PlacementKind` += `'dropped'`, `placementKind()` prend
      désormais les DEUX bascules, `drawPlacement` aiguille, `hoverRadiusPx`,
      `countDrawablePlacements` rend `dropped`.
- [ ] `placementWindow.ts` : un lâché reste jusqu'à la fin du rejeu (aucun changement de
      code attendu — à VÉRIFIER sur pièces, pas à supposer).
- [ ] Tests : familles de puissance seulement, grenades/capacités jamais, `deployed`
      inchangés, bascule éteinte = rien.
- Gate : `vitest run src/features/match-replay` + `tsc -b --force`.

### Étape 2 — la garde de mode, la bascule, l'infobulle
- [ ] `replayFiesta.ts` : `matchFiestaGuard(header)` → `'fiesta' | 'clear' | 'unknown'`, pur
      et testé.
- [ ] `useReplaySettings` : `showDroppedPlacements` / `toggleDroppedPlacements`.
- [ ] `useReplayPlacements.ts` : EXTRACTION préalable imposée par le cliquet de taille de
      `ReplayCanvas.tsx` (812 lignes, plafond atteint) — comptes, axe de temps, bascules du
      calque et survol dans un seul hook, sur le modèle de `useReplayWeaponPads`.
- [ ] `ReplayCanvas` : prop `droppedAllowed` (défaut faux), câblage du calque, du survol et
      du tiroir. Le fichier doit RESTER sous 812 lignes.
- [ ] `ReplaySettingsDrawer` : la bascule et son aide.
- [ ] `ReplayPlacementTip` : famille, lâcheur, instant.
- [ ] `i18n.ts` + `i18nContract.ts` : FR et EN, parité par typage.
- [ ] Route `replay.tsx` : `droppedAllowed = matchFiestaGuard(matchView?.header) === 'clear'`.
- Gate : `tsc -b --force`, `eslint`, `vitest run src/features/match-replay`.

### Étape 3 — témoins mesurés et garde-rails
- [ ] Test des deux témoins : recensement du § 2.2 rejoué en fixture, comptage des
      PRIMITIVES de canvas (`recordingContext`) — `01e1f945` : +1 ; `000d5950` : +0 sous la
      garde Fiesta, et +26 sans elle (sinon le témoin ne prouverait rien).
- [ ] Garde-rail de vocabulaire : `PLACEMENT_DROPPED_FAMILIES` cohérente avec
      `PLACEMENT_RENDER` et avec les power-ups de `weaponPadFamilies`.
- [ ] Garde-rail couleurs : aucune valeur hex ni classe Tailwind couleur dans les fichiers
      touchés (le garde `canvasInk.guard` / `fxInk.guard` existe — vérifier la portée).
- [ ] Parité i18n FR/EN sur les clés neuves.
- [ ] Cliquet `ReplayCanvas.tsx` toujours vert.
- Gate : la suite `src/features/match-replay` entière.

### Étape 4 — plan statué, journal, registre
- [ ] Toutes les cases statuées `[x]` / `[~]` / `[!]`.
- [ ] Entrée `.ai/thought_log.md`.
- [ ] Registre des reports : la ligne du 18/08 passe à TRAITÉE (périmètre équipements +
      power-ups) ; deux reports NEUFS ouverts (armes lâchées = données ; trou de 0,7 % sur la
      détection de mode).

## 5. Contraintes de livraison

- `npm ci` dans le frère (fait), commits `feat(v7.5-rejeu-r2):` / `docs(...)`, une seule
  première ligne, jamais `git add -A`, aucun Go touché.
- Fichiers ≤ 500 lignes, fonctions ≤ 80, ≤ 5 paramètres.
- Gates sans pipe qui masque l'exit ; codes consignés dans `dropped_gates.log`.

## 6. Découvertes (à NE PAS traiter dans ce lot)

_(rempli en cours d'exécution)_

## 7. Journal d'exécution

_(rempli en cours d'exécution)_
