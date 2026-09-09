# Plan — Vague C : les formes (heatmaps, blocs manquants, médailles)

> Issu de `.ai/diagnostics/RETOURS_2026-09-08/DIAGNOSTIC_RETOURS_UTILISATEUR_2026-09-08.md`,
> §§ 1, 18, 18bis, 20, 20bis, 22. **Exécution sous le contrat du skill `plan-execution`.**

## Objectif et critère de succès

Rapprocher l'application des **deux maquettes validées** (`2ec1b8eb` « Les formes retenues »,
`4c520da6` « L'échange sur la page Escouade »), maintenant qu'elles sont dépouillées et que l'écart
est mesuré.

**Critère de succès :** les cinq lots ci-dessous sont livrés, et le garde-rail de C1 interdit
qu'une sixième implémentation de grille réapparaisse.

**Effort** : lourd. C'est la vague la plus parallélisable — **C1, C3 et C5 sont indépendants**, C2
dépend de C1, C4 est autonome.
**Branche d'intégration** : `feat/formes-maquettes` depuis `feat/v75`, une branche par lot fusionnée
dedans.

## Décisions PRISES — à ne pas rouvrir en exécution

| # | Décision | Valeur retenue |
|---|---|---|
| D1 | Il existe déjà un composant canonique | `components/charts/Heatmap2DChart.tsx`. **On l'étend, on n'en crée pas un second** |
| D2 | Padding entre cases | Obtenu par `itemStyle: { borderWidth, borderColor: <fond>, borderRadius }` sur la série heatmap. C'est le trait que l'utilisateur juge essentiel : « c'est aéré et joli » |
| D3 | Case sans mesure | **Hachurée, avec un tiret**, légendée « Aucune mesure sur cet axe ». La laisser non peinte contredit la doctrine du dépôt (« l'absence a sa propre forme, jamais un vide ») |
| D4 | Gros point du nuage d'isolement | **La MÉDIANE** par joueur, taille proportionnelle au total des morts examinées, étiquetée du gamertag. La maquette dit « un gros point par joueur » sans trancher : la médiane est cohérente avec les lignes de repère de la même carte, qui médianent déjà |
| D5 | Grenades | **Sorties des blocs d'équipement**, verbatim de la maquette : « ce ne sont pas des équipements ». Familles retenues : camouflage, surbouclier, mur de protection, grappin, objets lâchés au sol |
| D6 | Page Sessions | **Ne prend que des graphes normalisés** (parts en %, cadences par dix minutes). Déjà respecté, à ne pas casser |
| D7 | Médailles de la frise | **En IMAGES** avec titre et description en infobulle. Annule les décisions 9 et 14 de `PLAN_FRISE_POINT_DE_VUE_2026-09-06.md` |
| D8 | Variante multi-joueur de « Portée des engagements » | **HORS de cette vague.** Sa forme n'a jamais été arrêtée — elle demande une maquette avant du code |
| D9 | Bloc d'équipement : « déployé » et « lâché » | **Fusionnés en UNE colonne par famille, barre empilée** (utilisé / lâché en mourant / gardé sans l'utiliser), échelle commune par colonne. Validé sur maquette par l'utilisateur le 2026-09-09. **CORRIGÉ le 2026-09-09 (même jour, décision utilisateur explicite) : les power-ups NE SORTENT PLUS.** La rédaction initiale les excluait au motif qu'ils « ne se déploient jamais » — vrai pour le canal des poses, mais leur usage EST mesuré par `equipmentEpisodes`. La règle réelle n'est pas « bonus vs déployable », c'est **deux définitions de « utilisé »** : un équipement d'ACTIVATION (camouflage, surbouclier, translocateur, grappin, propulseur) est utilisé quand il est ACTIVÉ ; un DÉPLOYABLE (mur, capteur, écran occultant, traqueur, champ de réparation) est utilisé quand il est POSÉ. Les deux entrent dans la barre. Détail et canaux : `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` |

## Règles d'exécution

Identiques aux vagues A et B. **Une seule revue adversariale en fin de vague**, sur le diff
d'intégration complet — pas par lot.

---

## Lot C1 — Étendre le wrapper canonique de grille

**Périmètre fermé (2 fichiers + tests) :**

- [ ] `components/charts/Heatmap2DChart.tsx` — ajouter le padding des cases (décision D2) :
      `itemStyle` avec `borderWidth`, `borderColor` à l'encre du fond, `borderRadius`
- [ ] Même fichier — rendre l'absence VISIBLE (décision D3) : une case `value: null` reçoit une
      hachure et un tiret, et la légende la nomme. Aujourd'hui elle est « non peinte, hors échelle,
      sans étiquette », donc invisible
- [ ] Exposer un plafond de saturation optionnel (la maquette sature à trente points) — par défaut
      inchangé, pour ne rien casser chez les consommateurs existants
- [ ] Tests : une case `null` produit la hachure ; deux cases voisines ne se touchent pas
- [x] `components/charts/Heatmap2DChart.tsx` — ajouter le padding des cases (décision D2) :
      `itemStyle` avec `borderWidth`, `borderColor` à l'encre du fond, `borderRadius`. Fait :
      `borderWidth: CELL_BORDER_WIDTH (4)`, `borderColor: tc.card`, `borderRadius: CELL_BORDER_RADIUS (2)`
      au niveau série (hérité par toutes les cases, mesurées ou vides)
- [x] Même fichier — rendre l'absence VISIBLE (décision D3) : une case `value: null` reçoit une
      hachure et un tiret, et la légende la nomme. Fait : fond `tc.splitLine` + `decal` (hachure
      diagonale, encre `tc.axisLabel`) au niveau ITEM (seules les cases vides, pas les mesurées),
      étiquette `—` (`EMPTY_CELL_LABEL`), légende FR/EN rendue par le composant
      (`HEATMAP_EMPTY_CELL_TEXT`, pied de `ChartCard` via sa prop `legend`) uniquement quand la
      série contient au moins une case vide
- [x] Exposer un plafond de saturation optionnel (la maquette sature à trente points) — par défaut
      inchangé, pour ne rien casser chez les consommateurs existants. Fait : prop `saturationCap?:
      number`, ignorée si `valueRange` est fourni, absente par défaut (comportement historique
      inchangé, testé en non-régression)
- [x] Tests : une case `null` produit la hachure ; deux cases voisines ne se touchent pas. Fait,
      répartis sur deux fichiers (voir note ci-dessous) : `Heatmap2DChart.test.ts` (étendu — hachure/
      itemStyle/decal, tiret, borderWidth>0, plafond de saturation, non-régression) et
      `Heatmap2DChart.test.tsx` (créé — légende FR/EN au niveau composant, absente quand aucune
      case vide)

**Ce que ce lot ne fait PAS** : toucher aux quatre consommateurs. Ils héritent gratuitement.

**Gate :**
```bash
cd apps/web && npx vitest run src/components/charts/Heatmap2DChart
```

## Lot C2 — Migrer les deux implémentations hors wrapper (dépend de C1)

**Périmètre fermé (2 fichiers + 1 garde-rail) :**

- [ ] `features/synthesis/SynthesisHeatmapChart.tsx` — construit son option ECharts à la main.
      La faire passer par `Heatmap2DChart` en mode `divergent` (sa rampe autour de 50 % est
      légitime et doit être préservée)
- [ ] `features/session-detail/SessionUsageForms.tsx` → `UsageRegularityBand` — **NE PAS migrer**.
      C'est une miniature DOM/CSS assumée (cases de 14 px, valeur impossible à écrire) ; la statuer
      `[~]` avec cette référence, et vérifier seulement que son `gap-[3px]` reste
- [ ] **Garde-rail** (règle n°6 du dépôt : une factorisation sans garde-rail re-diverge) : un test
      grep qui échoue si un `type: 'heatmap'` ECharts apparaît hors de `Heatmap2DChart.tsx`
- [x] `features/synthesis/SynthesisHeatmapChart.tsx` — construit son option ECharts à la main.
      La faire passer par `Heatmap2DChart` en mode `divergent` (sa rampe autour de 50 % est
      légitime et doit être préservée). **FAIT le 2026-09-09 (lot 1.5, worktree
      `LevelUp-wt-vague1`)** : le report du lot 1.4 est levé, le superviseur a ordonné
      l'exécution. `valueRange={[0, 1]}` FIGE l'échelle du visualMap (le neutre reste à 50 %
      quelle que soit la plage réelle des taux de victoire — laisser le wrapper auto-ajuster
      min/max aurait décentré le neutre). `formatTooltip` reproduit le contenu exact de
      l'ancien tooltip (jour, heure, taux, nombre de matchs). Le wrapper n'exposant pas
      d'option `inverse` pour l'axe Y (contrairement à l'ancienne implémentation
      `yAxis.inverse: true`), les points sont émis Dimanche → Lundi pour que Lundi occupe le
      DERNIER index (le haut d'un axe catégoriel non inversé) — même rendu visuel. Effets de
      bord ACCEPTÉS, inhérents à l'unification (pas de régression au sens du plan, qui ne
      demande de préserver QUE la rampe) : légende horizontale en pied de carte au lieu de la
      barre verticale à droite, plus de titres d'axes ("Heure"/"Jour") ni de libellés
      "Victoires"/"" aux bornes du visualMap — le wrapper canonique n'expose aucune de ces
      deux options, et les 4 autres consommateurs vivent déjà sans elles.
- [~] `features/session-detail/SessionUsageForms.tsx` → `UsageRegularityBand` — **NE PAS migrer**.
      C'est une miniature DOM/CSS assumée (cases de 14 px, valeur impossible à écrire) ; statuée
      `[~]` avec cette référence. Vérifié sur pièces (2026-09-09) : `gap-[3px]` toujours présent
      ligne 276, aucune modification
- [x] **Garde-rail** (règle n°6 du dépôt : une factorisation sans garde-rail re-diverge) : un test
      grep qui échoue si une série heatmap ECharts apparaît hors de `Heatmap2DChart.tsx`. Fait :
      `components/charts/heatmapSingleImpl.guard.test.ts`, allowlist DATÉE 2026-09-09 (les 5 sites
      relevés par grep avant écriture : `SynthesisHeatmapChart.tsx`, `ActivityCalendarChart.tsx`,
      `ExplorerActivityHeatmapChart.tsx`, `RelationsMomentsHeatmap.tsx`, `squadMapHeatmapChart.ts`).
      Mordant PROUVÉ : ajout temporaire d'un 6e site (`features/tactical/_tmpHeatmapProbe.ts`),
      test rouge confirmé, fichier supprimé, test revert au vert — détail au thought_log.
      **MIS À JOUR le 2026-09-09 (lot 1.5)** : `SynthesisHeatmapChart.tsx` retiré de l'allowlist
      (plus aucun littéral `type: 'heatmap'` dans ce fichier après la migration ci-dessus) — 4
      sites restants, tous décision S6 (hors périmètre de la vague C).

**Gate :**
```bash
cd apps/web && npx vitest run src/components/charts src/features/synthesis
grep -rn "type: 'heatmap'" apps/web/src --include=*.ts --include=*.tsx | grep -v Heatmap2DChart
# doit ne rendre que des lignes allowlistees par le garde-rail
```
Exécuté (2026-09-09, lot 1.4) : vitest 283/283 (charts) + suite complète synthesis/squad/tactical/
session-detail 1089/1089 verts ; grep rend exactement les 5 sites ci-dessus ; `make check-types`
(`tsc -b`) vert.

**Ré-exécuté (2026-09-09, lot 1.5, après la migration de `SynthesisHeatmapChart.tsx`)** :
`cd apps/web && npx vitest run src/components/charts src/features/synthesis` → 44 fichiers,
409 tests verts (14 skipped, préexistants) ; `grep -rn "type: 'heatmap'" apps/web/src
--include=*.ts --include=*.tsx | grep -v Heatmap2DChart` rend exactement les 4 sites restants ;
`make check-types` (tsc -b) 0 erreur ; `npx eslint` sur les 3 fichiers touchés
(`SynthesisHeatmapChart.tsx`, `SynthesisHeatmapChart.test.tsx`,
`heatmapSingleImpl.guard.test.ts`) : 0 issue. Nouveau fichier de test dédié
`SynthesisHeatmapChart.test.tsx` (6 cas) : rampe divergente figée sur [0, 1], ordre Lundi/
Dimanche de l'axe Y, ordre des heures, contenu du tooltip, case vide (count 0 → value null),
état vide (aucune cellule mesurée).

## Lot C3 — Le nuage d'isolement : ce qui manque

**Exécuté le 2026-09-09 (lot 1.5, worktree `LevelUp-wt-vague1`, branche
`feat/vague1-integration`).** Vérifié sur pièces avant d'écrire : `SquadIsolementNuageCard.tsx`
avait bougé le 09-09 (chantier « légendes couleurs », commit `9212a1b0e` — infobulle ⓘ du
titre au lieu d'un pied de carte, socle de légende commun `getLegendBase`). Découverte : les
quatre libellés de quadrant étaient déjà AFFICHÉS dans les coins via `t.quadrant(...)`
consommé par `markArea` — seule la fonction `quadrantDuPoint` (classement PAR POINT) restait
sans appelant hors test. Rebranchée dans le tooltip d'un point (nomme son quadrant), pas dans
le rendu des coins (déjà fait).

**Périmètre fermé (3 fichiers) :**

- [x] `features/squad/SquadIsolementNuageCard.tsx` — ajouter **le gros point par joueur**
      (décision D4) : une série de plus, médiane par joueur, taille au total des morts, étiquetée.
      Fait : `pointMedianJoueur` (nouveau, `squadIsolement.logic.ts`) agrège les points d'un
      joueur (médiane par axe, somme des morts examinées) ; une seconde série ECharts par
      joueur (même nom → même entrée de légende), taille via `tailleMedianeDuPoint` (plage
      dédiée, visiblement plus grosse que le plus gros point de session), étiquette
      `label.formatter = gamertag` posée au-dessus du point, tooltip dédié
      (`t.tooltipMedian`, nouvelle clé manifest).
- [x] Même fichier — **afficher les quatre libellés de quadrant** dans les coins. `quadrantDuPoint`
      (`squadIsolement.logic.ts:66`) et les quatre clés (`squadIsolementStrings.ts:20-23`) existent
      déjà et **n'ont aucun consommateur** : c'est du code mort à rebrancher, pas à écrire.
      **Vérifié sur pièces (2026-09-09)** : les QUATRE LIBELLÉS DE COIN étaient déjà rendus
      (via `t.quadrant('procheCouvert')` etc. dans `markArea.data[i].name`, ajouté par un
      chantier antérieur) — seule la fonction `quadrantDuPoint` (classement par POINT, pas
      par région) n'avait aucun appelant hors test. Rebranchée dans le tooltip d'un point
      (nouvelle clé `squad.isolement.point_quadrant`, "Quadrant : {quadrant}") : chaque point
      nomme désormais explicitement le quadrant auquel il appartient, en plus des libellés de
      coin déjà en place.
- [x] Aligner le signal d'échantillon faible sur la maquette : **cercle pointillé** plutôt
      qu'opacité réduite. Fait : `itemStyleDuPoint` (nouveau) rend `{ color: 'transparent',
      borderColor: color, borderWidth: 1.5, borderType: 'dashed' }` pour un point atténué,
      `{ color }` (plein) sinon. `opaciteDuPoint`/`OPACITE_ATTENUEE`/`OPACITE_PLEINE` SUPPRIMÉS
      (plus aucun appelant après le remplacement — CLAUDE.md règle 7, zéro code mort), avec
      leurs tests. La mention textuelle « échantillon faible » du tooltip (lot Q8) est
      conservée intacte (`withLowSampleNote`, inchangé).
- [x] Tests : sur deux sessions et deux joueurs, deux gros points sont émis, à la médiane.
      Fait : `squadIsolement.logic.test.ts` (+9 cas : `pointMedianJoueur`,
      `tailleMedianeDuPoint`, migration de la suite `opaciteDuPoint` vers `pointAttenue` seul)
      + `SquadIsolementNuageCard.test.tsx` (+4 cas : deux gros points médians à la valeur
      attendue, quadrant nommé dans le tooltip, cercle pointillé vs plein).

**Gate passé (2026-09-09)** :
```bash
cd apps/web && npx vitest run src/features/squad/SquadIsolementNuageCard src/features/squad/squadIsolement
# 2 fichiers, 30 tests verts
grep -rn "quadrantDuPoint" apps/web/src --include=*.tsx | grep -v "\.test\."
# rend 3 lignes dans SquadIsolementNuageCard.tsx (import + commentaire + appel) : au moins un appelant
```
Gate élargi exécuté : `cd apps/web && npx vitest run src/features/squad` (56 fichiers, 487 tests
verts) · `make check-types` (tsc -b, 0 erreur) · `npx eslint` sur les 5 fichiers touchés (0 issue).

## Lot C4 — Les médailles de la frise, en images

**Périmètre fermé (3 fichiers) :**

- [ ] `features/match-replay/ui/ReplayMarkTrack.tsx` — remplacer l'anneau
      (`ring-1 ring-foreground`, 1 px autour d'une marque de 3 × 8 px, invisible en pratique) par le
      badge image (décision D7). `ui/MedalBadges.tsx` existe et le fil l'emploie déjà
      (`ReplayKillFeed.tsx:344` et `:508`)
- [ ] `ui/ReplayTimelineTracks.tsx` — **élargir la piste du joueur regardé** pour accueillir le
      badge. C'est ce qui fait de ce lot un vrai travail et non un correctif court : la hauteur de
      piste est structurelle (`rosterHeight.guard.test.ts`, `timelineGeometry.guard.test.ts`)
- [ ] L'infobulle porte le **titre ET la description** de la médaille : le document les publie déjà
      (`medal_label`, `medal_description`, `killFeedLogic.ts:86-92`)
- [ ] Mettre à jour l'en-tête de `ReplayTimelineTracks.tsx` : le « un seul code : anneau =
      médaille » n'est plus vrai (règle du dépôt : la doc se corrige dans le commit qui change le
      comportement)
- [x] `features/match-replay/ui/ReplayMarkTrack.tsx` — remplacer l'anneau
      (`ring-1 ring-foreground`, 1 px autour d'une marque de 3 × 8 px, invisible en pratique) par le
      badge image (décision D7). `ui/MedalBadges.tsx` existe et le fil l'emploie déjà
      (`ReplayKillFeed.tsx:344` et `:508`). Fait le 2026-09-09 : la marque nue perd son anneau, le
      badge se pose EN SURIMPRESSION à côté (span dédié, `pointer-events` par défaut pour que
      l'infobulle native du badge fonctionne au survol — seconde exception documentée à côté de la
      vignette média). `TrackMark.medals`/`TrackKill.medals`/`TrackMedal` portent désormais
      l'identité complète (`MedalEvent`), plus un simple libellé, jusqu'à `MedalBadges.tsx`
      (signature élargie en `readonly MedalEvent[]`, changement de glue nécessaire au typecheck).
- [x] `ui/ReplayTimelineTracks.tsx` — **élargir la piste du joueur regardé** pour accueillir le
      badge. C'est ce qui fait de ce lot un vrai travail et non un correctif court : la hauteur de
      piste est structurelle (`timelineGeometry.guard.test.ts`). Fait : 18 → 24 px (`h-[18px]` →
      `h-[24px]`), tops recalculés pour garder le même centre vertical (barre `top-2`, losange
      `top-[9px]`, badge `top-1`). `rosterHeight.guard.test.ts` s'est révélé, sur pièces, porter sur
      un tout autre sujet (le plafond `xl:max-h-[NN%]` des fiches joueur de la page de rejeu, pas la
      hauteur de piste) : `[~]` — rien à y changer, `timelineGeometry.guard.test.ts` est le seul
      garde-rail structurel réellement concerné et il est passé au gate.
- [x] L'infobulle porte le **titre ET la description** de la médaille : le document les publie déjà
      (`medal_label`, `medal_description`, `killFeedLogic.ts:86-92`). Fait : `MedalBadges` compose
      déjà `"${label} — ${description}"`, réutilisé tel quel — aucune logique de tooltip réécrite.
- [x] Mettre à jour l'en-tête de `ReplayTimelineTracks.tsx` : le « un seul code : anneau =
      médaille » n'est plus vrai (règle du dépôt : la doc se corrige dans le commit qui change le
      comportement). Fait, plus l'en-tête de `ReplayMarkTrack.tsx` (section badge, exception
      pointer-events) et le JSDoc de `useReplayTimeline.reduceFeed`.

**Découvertes (hors périmètre, non traitées) :**
- `rosterHeight.guard.test.ts` ne concerne pas la hauteur des pistes de la frise malgré son nom
  évocateur pour ce lot — il garde le plafond `xl:max-h-[NN%]` des fiches joueur sur la route de
  rejeu. Aucune action : le plan le citait par erreur d'association, pas le code.

**Gate :**
```bash
cd apps/web && npx vitest run src/features/match-replay/ui/ReplayMarkTrack src/features/match-replay/ui/timelineGeometry src/features/match-replay/ui/rosterHeight
```
Et de visu : rejeu Origin `8bc6074f`, point de vue JGtm — la médaille « Revirement » à 3:49 est
lisible sans survol.
# 3 fichiers, 28 tests, tous verts (2026-09-09)
cd apps/web && npx vitest run src/features/match-replay
# 178 fichiers passés + 1 skip préexistant, 2572 tests verts, aucune régression (2026-09-09)
make check-types
# tsc -b : 0 erreur (2026-09-09)
```
Et de visu : rejeu Origin `8bc6074f`, point de vue JGtm — la médaille « Revirement » à 3:49 est
lisible sans survol. `[~]` — contrôle de visu réservé au superviseur (cf. consignes d'exécution).

## Lot C5 — REPRIS AILLEURS

> **2026-09-09 — CE LOT EST ABSORBE** par `.ai/PLAN_EQUIPEMENT_GACHIS_2026-09-09.md`
> (etapes E5 et E6), apres validation d'une nouvelle maquette par l'utilisateur. Ne pas
> l'executer ici : deux agents travailleraient sur les memes fichiers.

### Perimetre d'origine, conserve pour memoire

## Lot C5 — Les blocs équipement / armes spéciales sur Escouade et Synthèse

> **C'est le lot le plus lourd, et le seul qui touche le backend.**

**Périmètre fermé — Go :**

- [ ] `internal/domain/` — le bloc d'usage n'existe que sur `session_page.go:145`. Publier
      l'équivalent pour la page Escouade et pour la Synthèse, en **types canoniques**
- [ ] `internal/service/` — l'orchestration ; **aucun SQL inline**, tout via repo/adapter
- [ ] `internal/platform/duckdb/` — la lecture, si une requête nouvelle est nécessaire ; sinon
      réutiliser celle de la page Sessions (à vérifier AVANT d'en écrire une)
- [ ] Brancher sur **capability**, jamais sur le slug (`no_slug_comparison_test.go` est un ratchet)
- [ ] `slog.InfoContext` / `ErrorContext` sur les dégradations ; jamais d'erreur avalée
- [ ] Tests : `analysis` purs, `service` avec mock `port.Repository`, `duckdb` en `:memory:`

**Périmètre fermé — Web :**

- [ ] Réutiliser `SessionUsageSection` / `SessionUsageForms` plutôt que réécrire : ils portent déjà
      les trois formes et **résolvent le contexte Solo/Escouade côté serveur**
- [ ] Respecter la décision D5 (pas de grenades) et D6 (Sessions reste normalisée)
- [ ] Strings FR **et** EN, jetons sémantiques uniquement, query keys dans `lib/query/keys.ts`

**Gate :**
```bash
make go-api-test && cd apps/go-api && go test -tags=integration ./...
cd apps/web && npx vitest run src/features/squad src/features/synthesis
make check-types
```

---

## Lot C6 — REMPLACE

> **2026-09-09 — CE LOT EST REMPLACE** par `.ai/PLAN_EQUIPEMENT_GACHIS_2026-09-09.md`
> (etapes E1, E2 et E3). La forme a change : TROIS issues (utilise / lache en mourant /
> garde sans l'utiliser) au lieu de deux, et les power-ups ENTRENT dans la colonne — la
> decision D9 a ete corrigee le meme jour. Ne pas l'executer ici.

### Perimetre d'origine, conserve pour memoire

## Lot C6 — Le bloc d'équipement : une colonne par famille, barre empilée

**Ce que la mesure établit, et qui ferme le débat de forme** (relevé du 2026-09-09, 128 artefacts) :

- `deployed` et `dropped` sortent du **MÊME canal** `equipmentPlacements`, discriminés par le seul
  champ `origin` : deux issues exclusives d'un même objet, pas deux grandeurs indépendantes.
- **Un lâcher est une MORT, jamais un geste.** `equipmentOrigin` ne classe en `dropped` qu'à moins
  de 200 ms ET 1,5 m de la dernière position du porteur ; les deux populations mesurées sont
  séparées par trois ordres de grandeur (lâchers à 20-38 ms / 0,63 m ; déploiements à 14-42 s /
  5,6-21,3 m).
- **Une mort lâche EXACTEMENT UN objet, jamais un par charge restante.** Groupées par (poseur,
  famille, instant), toutes les familles d'équipement sont à 1,00 ; seules les grenades montent à
  1,7, et c'est la PILE PORTÉE, pas des charges.
- **Le canal des charges ne couvre pas ces familles.** `abilityCharges` ne porte que `grapple` et
  `thruster` (valeurs 0..4), et rien n'est transmis au ramassage — même le maximum n'est pas
  établissable. Décision utilisateur du 2026-09-09 : **on ne compte pas les charges.**
- **Aujourd'hui les deux colonnes ont CHACUNE SON ÉCHELLE** (doctrine de `ValueGrid`) : sur le
  match `4f77afc1`, deux barres pleines côte à côte valent 3 et 2. La fusion est ce qui rend la
  comparaison légale.

**Périmètre fermé (5 fichiers + tests) :**

- [ ] `components/charts/valueGridModel.ts` — `ValueGridCell.segments?` OPTIONNEL ; la borne de
      colonne se calcule sur le TOTAL de la pile. Absent = comportement inchangé, donc la grille
      des objectifs de `match-view` n'est pas touchée
- [ ] `components/charts/ValueGrid.tsx` — rendu des segments
- [ ] `features/match-replay/model/equipmentUsageColumns.ts` — `UsageGroupKey` :
      `'deployed' | 'dropped'` → `'equipment'` ; une colonne par famille, deux valeurs par cellule.
      Le `Record` exhaustif force alors toutes les tables à suivre
- [ ] Même fichier — **EXCLURE les power-ups** de la colonne fusionnée (décision D9) : ils sont
      dans `PLACEMENT_DROPPED_FAMILIES` mais ne se déploient jamais. Les garder donnerait une barre
      100 % « lâché » qui ne signifie PAS une non-utilisation
- [ ] `features/match-replay/model/equipmentUsageChart.ts` — `USAGE_GROUP_TOKENS` perd `dropped` ;
      pour la famille fusionnée la couleur dit l'ISSUE (utilisé / lâché), plus la famille
- [ ] `features/match-replay/i18n/i18n.ts` — libellés FR **et** EN. **NE PAS nommer le total
      « ramassés »** : un usage est une CHARGE, un lâcher est un OBJET — le total n'est pas un
      compte de ramassages (un capteur pris une fois et lancé quatre fois donne 4 usages, 1 objet)
- [ ] Réserve à AFFICHER, pas à cacher : ~5 % des poses sont `origin: unknown` et celles à
      `owner: -1` partent dans `unattributed`. Le total est « poses attribuées », jamais « tout ce
      qui a existé »
- [ ] Tests : une cellule à deux segments s'empile sur la borne du total ; une colonne SANS segment
      garde exactement le rendu d'avant ; un power-up n'entre pas dans la colonne fusionnée

**Gate :**
```bash
cd apps/web && npx vitest run src/components/charts src/features/match-replay/model/equipmentUsage
make check-types
```

**Découverte à NE PAS traiter dans ce lot** : le capteur affiche **49 déploiements pour 302
lâchers** sur le parc (1:6), quand le mur est à 1:1 (295/251). Vrai comportement de jeu ou défaut
de classement d'origine — à vérifier dans un chantier à part.

---

## Tâche hors lot — la recuisson : 15 artefacts au schéma 38, 7 aux bornes fausses

Ni un correctif ni une conception : du **temps machine**, à lancer quand ça arrange.

- [ ] `cmd/replay-build` sur les 15 artefacts restés au schéma 38 (`0891225f`, `0d265ab0`,
      `1b2d9e08`, `28c9b538`, `30a23d15`, `4ecdf3e7`, `72b0a25e`, `7b0d89c4`, `94a28b8b`,
      `a03a5e65`, `bfecd02b`, `cde26226`, `f0220a96`, `f2966f08`, `faff9935`)
- [ ] Vérifier que `coverage.vehicles` apparaît sur les 15 — c'est ce qui débloque l'affichage des
      joueurs en véhicule (point 6)
- [ ] **Recuire aussi les 7 artefacts aux bornes fausses** (`0a44c6cc`, `30a23d15`, `3923bede`,
      `4f77afc1`, `81c02726`, `879a4dba`, `a4083bd2`). Le correctif est livré côté cuisson
      (branche `wt/bornes-aberrantes`, commit `490dc595e`) mais les artefacts déjà cuits gardent
      leurs bornes. C'est ce qui débloque le FOND DE CARTE d'Isolement (point 5) : vérifié en
      rejouant la règle, `coversPlayedArea` passe de faux à vrai sur `81c02726`

**Gate :**
```bash
for f in data/cache/replays/halo_infinite/*.json; do case "$f" in *derived*) continue;; esac
  c=$(jq -r 'if (.coverage|has("vehicles")) then 1 else 0 end' $f)
  [ "$c" = "0" ] && echo "$(basename $f .json) schema=$(jq -r .schemaVersion $f)"
done
# doit ne rien rendre
```

---

## Clôture de vague

- [ ] `make gate-push` vert (filet local avant merge)
- [ ] **Une** revue adversariale sur le diff d'intégration complet
- [ ] Entrée `.ai/thought_log.md`
- [ ] `delivery-checklist` avant la demande de merge
- [ ] Mettre à jour les §§ 18, 20 et 22 du diagnostic avec l'écart résiduel

## Ce que cette vague NE traite PAS, et pourquoi

| Sujet | Motif |
|---|---|
| Variante multi-joueur de « Portée des engagements » | Forme jamais arrêtée (décision D8) — demande une maquette |
| Sémantique binaire de `publishable` | Décision d'architecture, escaladée à l'utilisateur |
| Direction du cône de visée en véhicule | Décision produit qui inverse une mesure — escaladée |
| Troisième état des grenades (« sort inconnu ») | Manque de DONNÉE publiée, pas de forme : chantier backend à part |
| `coversPlayedArea` (fond de carte écarté) | **CAUSE CORRIGÉE** le 2026-09-08 (`wt/bornes-aberrantes`, commit `490dc595e`) : un échantillon aberrant définissait les bornes. Il ne reste que la RECUISSON, portée par la tâche hors lot |
| Plancher/pas de la grille tactique | Chantier autonome |
| Lisibilité des étiquettes du sunburst | Petit chantier de forme, mais sur un composant hors périmètre |

## Découvertes (à remplir — NE PAS TRAITER)

_(vide au démarrage)_
- **2026-09-09 (lot C1)** — `features/squad/squadEchange.logic.ts:200-203` documente l'ancien
  comportement de `Heatmap2DChart` sur une case vide (« que le wrapper ne peint ni n'étiquette »).
  Ce commentaire décrit maintenant l'ANCIEN défaut : depuis C1, le wrapper peint une hachure et un
  tiret sur ces cases (décision D3). Fix hors périmètre C1 (le fichier est un consommateur, « ils
  héritent » = ne pas y toucher) — à corriger quand `SquadEchangeMatrixCard` sera repris (lot C2
  différé, ou tâche dédiée), pour éviter la doc inversée (CLAUDE.md, diagnostic n°9).
- **2026-09-09 (lot C1)** — même remarque potentielle à vérifier sur tout AUTRE consommateur de
  `ChartPointHeatmap` qui commenterait l'ancien rendu invisible des cases `value: null` (non
  vérifié exhaustivement au-delà de `squadEchange.logic.ts`, seul cas trouvé par grep de
  `ne peint`/`non peinte` dans `features/squad`).
