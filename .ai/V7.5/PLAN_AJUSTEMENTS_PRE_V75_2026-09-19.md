# Plan — Ajustements UI pré-v7.5 (retours du 2026-09-19)

Source unique : `C:\Users\Guillaume\Downloads\Ajustements pre-v75.txt` + la conversation du
2026-09-19. Contrat d'exécution : skill `plan-execution`. Branche d'intégration :
`wt/ajustements-pre-v75` (worktree `LevelUp-wt-ajustements-pre-v75`, base `feat/v75`
`de7cb7af0`). Lots parallèles sur `wt/ajustements-matchview` et `wt/ajustements-heatmap`,
fusionnés dans `wt/ajustements-pre-v75`, puis un seul merge dans `feat/v75`.

## Règle de lecture (utilisateur, 2026-09-19)

| Le retour dit « virer »… | Action |
|---|---|
| un **bloc / une carte / un graphe** | **DÉPLACER** vers sa page cible (le « Priorité une » nomme Timeseries pour tout le Contexte Solo) |
| une **mention, un libellé, un texte, une légende, un axe** | **SUPPRIMER** ; si le texte porte de la compréhension → infobulle `(i)` à côté du titre du bloc |

## Décisions tranchées (aucune question ouverte — ne pas re-décider)

1. Onglet nouveau de la vue match : **« Contrôle » / "Control"**.
2. « Où ça se joue » → **« Occupation du terrain » / "Ground occupancy"**.
3. « Le compte » et « Combien, et à quelle vitesse » (Escouade) : vérifiés au niveau du camp du joueur — **restent tels quels**.
4. « Pourquoi la vengeance ne vient pas » : l'utilisateur veut **TOUTES les morts** affichées, la médiane par joueur (gros point) conservée. Sur une session (usage nominal de la page), un point par session ne rend qu'un point par joueur. **Décision** : un petit point **par mort**, axes continus — X = distance au coéquipier visible le plus proche rapportée à la portée du radar du match (`PlusProcheM / rayon`, repère vertical à 1,0 = « portée du radar », `nil` = bande « hors de vue » à droite) ; Y = délai avant vengeance en secondes (`DelaiMs`), mort jamais vengée = bande haute « jamais vengée ». Gros point par joueur = médiane X × médiane Y des morts vengées, taille = nombre de morts ; son infobulle porte encore la part isolée % et le taux d'échange %. Jointure Go exacte sur `(match_id, victim_xuid, time_ms)` entre `MortContexte` (isolement) et `MortSuivie` (riposte). Infobulle des points : « couverture = portée / proximité du radar ».
5. « Taux d'échange par session » : sous 3 sessions, **plus d'état vide** — afficher le taux de la ou des soirées sélectionnées face à l'habituel (`echange.habituel`, déjà servi) ; à partir de 3 sessions, la courbe comme aujourd'hui.
6. Vocabulaire équipement (vue match) : « actif » et « utilisé » disent la même chose (les épisodes actifs alimentent déjà le côté « utilisé », P2). **Le groupe de colonnes « États actifs » est retiré** ; seules restent Utilisé / Gardé / Lâché.
7. « Portée des frags » (Explorer, section « matchs joués ensemble ») : **la cible seule**, sur les matchs communs (le scope de la section). Retirer la bande « toi », sa légende, `frag_range_self` et sa lecture Go (règle 7).
8. Backfills (projection d'usage 62/1147, niveaux d'armes vides) : **hors chantier**, consignés dans Notion « Séquence à dérouler à la release » (fait le 19/09).
9. **Infobulles (i)** : toujours concises — **3 phrases grand maximum, phrases simples**, jamais un pavé. Un lexique de plusieurs paragraphes se résume, il ne se recopie pas. FR et EN.

## Lot 1 — Escouade + Timeseries (Go + web) — worktree `LevelUp-wt-ajustements-pre-v75`

### 1.A Escouade, bloc « Pourquoi la vengeance ne vient pas » (nuage)
- [x] Go : nouveau contrat `SquadNuageIsolement.morts[]` (un élément par mort du roster sur le scope : xuid, gamertag, match_id, time_ms, `distance_ratio` nullable, `hors_de_vue` bool, `vengee` bool, `delai_ms` nullable) + `reperes[]` par joueur (médianes, nb morts, part isolée, taux d'échange) — `internal/domain/squad_isolement.go`, construction dans `teammates_squad_isolement.go` (jointure `MortContexte` × `MortSuivie` sur `(match_id, victim_xuid, time_ms)`), openapi régénéré (`make generate-types`).
- [x] Web : `SquadIsolementNuageCard` rend un petit point par mort + gros point par joueur ; axes X (ratio distance / portée radar, ligne de repère à 1,0) et Y (délai s, bande « jamais vengée ») ; infobulle des points nomme la portée / proximité du radar.
- [x] Supprimer les deux pavés (« Portée du radar : lue par match… » et « Les deux vont ensemble… ») et leurs clés i18n.
- [x] Tests : Go (jointure, mort sans contexte, mort non vengée, ratio nil), logique web, garde-rail contrat.

### 1.B Escouade, bloc « Taux d'échange par session »
- [x] Sous 3 sessions : rendu « soirée(s) sélectionnée(s) vs habituel » au lieu de `EmptyStateNotice`.
- [x] Supprimer la mention « Une seule série, donc pas de légende… ».

### 1.C Escouade, bloc « Qui couvre qui »
- [x] Supprimer : « Vengeur × Vengé », « Mesuré sur N des M matchs de la sélection. », « Échanges réalisés sur N matchs d'escouade », le pavé « Ligne : celui qui venge… Un échange : … ».
- [x] Supprimer la barre de dégradé (visualMap) en bas à gauche.

### 1.D Escouade, bloc « Assistances dans l'escouade »
- [x] Positionné À DROITE de « Qui couvre qui » (même rangée, grille 2 colonnes).
- [x] Supprimer « Même donnée que la matrice, en plus grossier… », « Échanges donnés et reçus, par joueur », « Doublonne en partie avec la matrice… », « Mesuré sur N des M matchs de la sélection ».
- [x] Style de bloc canonique (`SectionCard`, comme les autres blocs de la page).
- [x] « Une barre par larbin (celui qui prépare le frag), un segment par patron (celui qui l'encaisse), sur les matchs de la sélection. » → infobulle `(i)` à côté du titre.
- [x] Barres empilées **horizontales** ; étiquettes d'axes « Patron » / « Larbin » (EN : "Boss" / "Minion").

### 1.E Escouade, section « Les formes retenues » (équipements, armes spéciales, objectifs)
- [x] DÉPLACER vers Timeseries les 9 cartes du Contexte Solo : `EquipmentSharesCard`, `EquipmentByMatchCard` (« Cadence de gestes, match par match »), `EquipmentSpreadCard` (« Étendue et moyenne de la période »), `PadsGapSoloCard` (« Écart à la parité, par famille d'arme »), `PadsShareSoloCard` (« Ma part des prises de socle, et sa dispersion »), `PadsWeaponGridCard` (« Taux de rafle par arme »), `ObjectivesGapRoleCard`, `ObjectivesSharesByFamilyCard`, `ObjectivesRawGridCard`. Le bloc `formes_retenues` est ajouté à la réponse Timeseries (même builder Go, scope solo de la page) ; sur Escouade ne restent que les cartes du Contexte Escouade. Les intertitres « Contexte Solo — … » / « Contexte Escouade — … » disparaissent (une page = un contexte).
- [x] Supprimer tous les pavés `constat` et `lexique` des trois blocs et l'`intro` de section ; le `lexique` de chaque bloc passe en infobulle `(i)` du titre du bloc (le `constat` disparaît).
- [~] Sortir les donuts de « Notre part de l'équipement du lobby » et « Notre part des armes spéciales du lobby » de leur bloc : le donut devient sa propre carte à droite, même rangée. — VÉRIFIÉ SUR PIÈCES : déjà en place depuis le découpage du 2026-09-13 (`EquipmentUsageSection.tsx`, deux `SectionCard` dans une grille `lg:grid-cols-2`). Ce qui restait — le libellé central — est traité par l'item jumeau de 1.G.
- [x] `HeaderStrip` (Ensemble / Lobbies / Parité / Modes) : conservé.

### 1.F Escouade, « Intensité »
- [x] Légende joueur / équipe / lobby sur le graphe.

### 1.G Timeseries
- [x] « Premier frag / première mort » : même hauteur que « Stats par minute » (même rangée, `items-stretch`, hauteur du graphe = hauteur de la carte voisine) + légende propre centrée en bas.
- [x] Supprimer le pied « Mesuré sur N matchs sur M » (équipement ET armes spéciales) — `measuredFooter` retiré. RÉSERVE : la clé `measuredFooterFmt` est CONSERVÉE, elle a un second consommateur hors périmètre (`padTiersCoverage`, pied de la rangée « Niveaux d'armes », dont la couverture est propre et vient d'une autre passe). Ses tests d'accord en nombre restent, ré-ancrés sur ce consommateur.
- [x] « Ma part de l'équipement du lobby » et « Ma part des armes spéciales du lobby » : le donut sort du bloc (carte propre à droite, même rangée) ; supprimer le libellé central « objets pris dans le lobby » (`donutEquipmentCenterLabel`, `donutWeaponCenterLabel`).
- [x] La carte « Ma part des armes spéciales du lobby » se rend TOUJOURS (état vide titré quand `weapon_pad_parties` est nul ou à 0), parallèle à l'équipement.
- [x] Supprimer le scroll horizontal des grilles d'usage (`overflow-x-auto` → les colonnes se répartissent en `grid` / `min-w-0`), sur Timeseries ET vue match (`UsageCountsGrid`, `UsageForms`).
- [x] « Intensité » : courbes équipe et lobby ajoutées (même modèle que `SquadIntensityProfileChart` : équipe = vrais alliés, lobby = tout le match) + légende joueur / équipe / lobby.
- [x] Accueillir les 9 cartes solo de 1.E dans l'onglet Progression, sous « Usages d'équipement ».

### Gate lot 1
```
cd apps/go-api && go build ./... && go test ./internal/domain/... ./internal/analysis/coordination/... ./internal/service/teammates/... ./internal/service/timeseries/...
make generate-types puis git diff --exit-code apps/web/src/lib/api/generated.ts
cd apps/web && npx tsc -b --force && npx eslint src/features/squad src/features/timeseries src/features/_shared/usage && npx vitest run src/features/squad src/features/timeseries src/features/_shared/usage
grep -rn "Vengeur × Vengé\|Mesuré sur .* des .* matchs\|Une seule série, donc pas de légende\|objets pris dans le lobby\|Contexte Solo\|Contexte Escouade" apps/web/src --include=*.ts --include=*.tsx  → 0 résultat hors tests de non-régression
```

## Lot 2 — Vue match (web) — worktree `LevelUp-wt-ajustements-matchview`

- [ ] Onglet « Contrôle » (`MATCH_VIEW_TABS` + `'control'`, i18n `tabControl` FR « Contrôle » / EN "Control", `MatchViewPage.tsx`, nouveau `MatchViewTabControl.tsx`) regroupant : « Usages d'équipement » (`MatchEquipmentUsageSection`, retiré de Chronologie), « Contrôle des armes » (`MatchPadControlSection`, sur le même modèle de carte que l'équipement), « Occupation du terrain » (`MatchPositionsHeatmap`, retiré de son emplacement actuel).
- [ ] « Où ça se joue » → « Occupation du terrain » / "Ground occupancy" (i18n).
- [ ] « Occupation du terrain » : hauteur réduite d'au moins 40 % ; supprimer la mention « Grille de 2,0 m · … Camps attribués par regroupement spatial, sans nom de joueur. ».
- [ ] « Part de chaque équipe, geste par geste » → « Part de chaque équipe » (FR) / EN équivalent.
- [ ] Supprimer « Replier / Voir plus » : tout est affiché par défaut. `collapsed-items-toggle.tsx` et son garde-rail : supprimés avec ses deux appelants (équipement + armes) — règle 7, aucun composant orphelin.
- [ ] Retirer le groupe de colonnes « États actifs » (décision 6) : `activeEpisodesGroup` + clés i18n `activeColumnFmt`, `activeCellTipFmt`, `activeFamily` supprimés avec leurs tests.
- [ ] Supprimer le scroll horizontal du bloc « Usages d'équipement » (vue match).
- [ ] Deep-link : `?tab=control` valide ; tests `MatchViewTabs.test.tsx` mis à jour.

### Gate lot 2
```
cd apps/web && npx tsc -b --force && npx eslint src/features/match-view src/features/match-replay src/components/ui && npx vitest run src/features/match-view src/features/match-replay src/components/ui
grep -rn "geste par geste\|Où ça se joue\|Grille de 2,0\|CollapsedColumnsToggle\|CollapsedWeaponsToggle\|collapsed-items-toggle" apps/web/src --include=*.ts --include=*.tsx → 0
```

## Lot 3 — Carte de chaleur canonique + Explorer (web + un retrait Go) — worktree `LevelUp-wt-ajustements-heatmap`

- [ ] Extraire le rendu de `SynthesisHeatmapChart` (« Activité par jour et heure ») en composant unique générique dans `components/charts/` (cellules aérées, étiquettes d'axes, infobulle, SANS barre de dégradé) et le consommer sur : Synthesis « Activité par jour et heure », Relations « Rythme des rencontres » (`RelationsMomentsSection`), Explorer « Carte de chaleur d'activité commune », Escouade « Performance par joueur × carte » (`SquadMapHeatmapChart`, sans le libellé d'axe « Carte »).
- [ ] Supprimer la barre de dégradé (visualMap) de Synthesis « Activité par jour et heure ».
- [ ] Garde-rail grep : aucune autre construction locale de heatmap catégorielle hors du composant canonique.
- [ ] Explorer, recherche par joueur : retirer le doublon « Top médailles » de la section « Carrière complète » (ne garder que celui de l'encart cible).
- [ ] Explorer « Portée des frags » : la cible seule (décision 7) — `ExplorerTargetFragRange` sans `self`, sans légende à deux entrées ; Go : `frag_range_self` retiré de `ExplorerEncounterStats` et sa lecture dédiée supprimée ; openapi régénéré.
- [ ] Explorer : rangée « Portée des frags » / « Répartition des résultats » / « Part des assistances » en colonnes de hauteur égale (`grid` + `items-stretch` + `h-full` sur les cartes).

### Gate lot 3
```
cd apps/go-api && go build ./... && go test ./internal/service/... -run 'Explorer|WeaponRange|Encounter'
make generate-types puis git diff --exit-code apps/web/src/lib/api/generated.ts
cd apps/web && npx tsc -b --force && npx eslint src/components/charts src/features/synthesis src/features/palmares src/features/explorer src/features/squad/SquadMapHeatmapChart.tsx && npx vitest run src/components/charts src/features/synthesis src/features/palmares src/features/explorer
grep -rn "frag_range_self\|fragRangeSelf" apps/web/src apps/go-api/internal --include=*.ts --include=*.tsx --include=*.go → 0
```

## Clôture
- [x] Fusion lot 2 et lot 3 dans `wt/ajustements-pre-v75` — sans conflit (56de738db, 34dd0c9be).
- [x] Gates sur la branche fusionnée (2026-09-20) : tsc 0, vitest 743 fichiers / 7 967 tests verts, `go test ./...` 40 ok / 0 FAIL, `make go-api-lint` 0 issue.
- [ ] Gate visuel utilisateur (Firefox, pile de dev basculée sur le worktree).
- [x] Entrée `.ai/thought_log.md` (2026-09-20).
- [ ] Fusion dans `feat/v75`, CI verte au niveau job (sur signal après le gate visuel).

## Découvertes (non traitées ici)
- `usageI18n.ts` porte un second gabarit « Mesuré sur N des M matchs de Capture du drapeau… »
  (pied des niveaux d'armes) : littéral voisin de celui retiré en 1.C/1.D, hors périmètre.
- La légende de rampe de « Qui couvre qui » était en DOM (`squad-echange-ramp`), pas un
  `visualMap` ECharts : `showVisualMap={false}` était déjà passé au wrapper.
- Les quadrants nommés du nuage d'isolement (`quadrantDuPoint`, `markArea`, quatre clés
  i18n) reposaient sur deux TAUX par point ; le contrat par mort ne les porte plus. Ils ont
  été supprimés avec leurs clés (règle 7) — les deux bandes nommées (« hors de vue »,
  « jamais vengée ») et le repère de portée du radar les remplacent.
- Projection d'usage : 62 matchs résumés / 1 147, 173 artefacts sur disque, fenêtre arrêtée au 07/09 → `backfill-usage-summary` (Notion, release).
- `match_pad_pickups_by_tier_latest` vide → `backfill-pad-tiers` (déjà dans Notion).
- Worktree `LevelUp-wt-explorer-rangee3` entièrement fusionné dans `feat/v75` : à supprimer.
- Notion « Ajustements pré-v7.5 » contient une liste plus large que le .txt (Accueil, Sessions, Citations, Médailles, Tactique…) — hors périmètre de ce plan.

## Journal
- 2026-09-20 : **LOT 1 EXÉCUTÉ ET CLOS** (1.A → 1.G, tous les items statués). Go : contrat
  `SquadNuageIsolement` refondu (un point par mort, jointure exacte (match_id, victim_xuid,
  time_ms) entre `MortContexte` et `coordination.Ripostes`), bloc `formes_retenues` ajouté à
  la réponse Timeseries (MÊME builder `squadagg.BuildSquadFormesBlock`, scope solo de la
  page), deux courbes de référence d'intensité (`intensity_rows_team` / `_lobby`, noyau
  commun `buildIntensityRowsPour`, équipe lue via `sessionusage.BuildTeamContext`). Web :
  nuage refondu, « Taux d'échange par session » parle sous 3 soirées, « Qui couvre qui » et
  « Assistances dans l'escouade » sur une rangée à deux colonnes, « Les formes retenues »
  scindée par contexte (solo → Timeseries, escouade → Escouade), pavés constat/lexique
  résumés en infobulles de 3 phrases, légendes d'intensité et de « Premier frag / première
  mort ». Défilement horizontal retiré de `ValueGrid`, `UsageCountsGrid` et `UsageForms`
  (correction du superviseur : la vraie cause était `ValueGrid.tsx`, pas seulement les deux
  formes d'usage). Gate joué : `go build ./...` + `go test ./...` verts, openapi et
  `generated.ts` sans dérive, `tsc -b --force` vert, `eslint` 0 erreur (2 avertissements
  TanStack préexistants), `vitest` 745 fichiers / 7 988 tests verts, grep des littéraux
  retirés à 0 hors tests de non-régression.
- 2026-09-19 : investigations closes — voir « Découvertes » et décisions 3, 4, 5, 7, 8. Mesures : `POST /api/v1/players/JGtm/pages/teammates` sans filtre = 361/451 mesurés, 18 points de nuage (9/joueur), 57 points de session ; `equipment_usage` solo = `weapon_pad_parties.lobby_total 1249`, `pad_pickups` non nuls pour les 4 joueurs suivis.
