# Plan — suite des retours du 2026-10-17 : Sessions, Synthèse, Séries temporelles, calendrier d'activité

Contrat : skill `plan-execution` (fait foi). Branche `feat/retours-pages`, worktree
`../LevelUp-wt-retours-escouade`, base `d819f74eb` (feat/v75 après fusion des lots
`feat/retours-escouade` et `feat/retours-tactique`). Plans de référence (clos) :
`.ai/PLAN_RETOURS_ESCOUADE_2026-10-17.md`, `.ai/PLAN_RETOURS_TACTIQUE_2026-10-17.md`.

## 0. Décisions de l'utilisateur (2026-10-17)

- Accord pour corriger les valeurs brutes des infobulles de Sessions, Synthèse et du calendrier
  d'activité (« « 1.3333333333333333 » est un format complètement nul »), et la bulle (i) de la
  Tactique qui dit « artefacts de rejeu ».
- « Regarde cette histoire de placeholder ou état vide pour ces pages aussi, ainsi que
  timeseries. » Règles déjà posées : on doit pouvoir distinguer un manque de fonction d'un manque
  de données (« sinon on pense à un bug ») ; respecter la convention de l'app (bloc parent +
  `EmptyStateNotice` dedans, `AnalysisPendingSlot` dans le bloc quand l'analyse du film est en
  cours) ; un état vide ne suggère jamais une action que l'utilisateur ne peut pas faire ni un
  vocabulaire interne (réanalyse, cuisson, artefact).

## 1. Acquis réutilisables (lots du jour)

- `components/charts/tipNumber.ts` (`tipNumber`, `tipNumberAuto`), `components/charts/axisTooltip.ts`,
  filet `valueFormatter` de `getTooltipBase`, garde `components/charts/rawTooltipValue.guard.test.ts`
  (surfaces non migrées nommées dans le test).
- Emprise de l'Escouade : `features/squad/emprise/EmpriseFilmGapSection.tsx`, `filmGapStrings.ts`,
  champs `matches_film_pending` / `matches_film_missing` du bloc `squad_emprise`
  (`service/teammates/teammates_service_emprise_gaps.go`, `AnalysisPendingResolver.ResolvePending`).
- `components/ui/pair-row.ts` (`PAIR_ROW_CLASS`, une carte seule prend la rangée).

## 2. Étapes

Statuts : `[x]` fait · `[~]` couvert ailleurs · `[!]` non traité (justification).

### P1 — Valeurs brutes et formats
- [x] P1.1 (tous migres : Sessions x14 graphes, Synthese x4 + carte chaleur + bipolaire, calendrier, HistogramChart, heatmap2DOption, sessionBarsTrendChart, Series temporelles x14, WeaponAccuracyChart/FragWeaponBreakdown/RadarChart ; nouveau tipNumberUpTo) Migrer sur `tipNumber`/`axisTooltip` : `features/session-detail` (`SessionEngagementChart`,
      `SessionModeBreakdown`, `SessionParticipationBars`, `SessionPerfTrend`,
      `SessionPlacementBreakdown`, `sessionBarsTrendChart.ts:223`), `features/synthesis`
      (`SynthesisOutcomesByGroupChart`, `SynthesisTopWeeksChart`),
      `features/ascension/ActivityCalendarChart`, `HistogramChart.tsx:99`, `heatmap2DOption.ts:371` ;
      et toute valeur brute des Séries temporelles (`features/timeseries`).
- [x] P1.2 (axisTooltipFormatter partout ou la bulle etait le rendu par defaut : une ligne par serie qui a une valeur, pile de haut en bas ; histogramme = plage + Matchs, nuage = libelles des deux grandeurs, calendrier = date a la langue, KdaDensity = abscisse a 2 decimales) Contenu des infobulles de ces pages : même règle qu'à l'Escouade (rien de redondant avec
      l'axe, ordre visuel du graphe, nombres à la langue).
- [x] P1.3 (garde 1 : interpolation de value/data sur TOUT src/, releve nul ; garde 2 : toFixed dans les fichiers de graphe de components/charts + Sessions + Synthese + Series temporelles + calendrier ; restes nommes et hors lot : career XP, explorer) Étendre le garde `rawTooltipValue.guard.test.ts` à ces features ; liste des surfaces
      non migrées vide si possible, sinon chaque reste justifié.

### P2 — États vides et sections absentes
- [x] P2.1 (tableau au rapport et au journal §4 ; sources lues sur pièces, page par page) Inventaire (tableau au rapport) pour Sessions (détail), Synthèse, Séries temporelles et
      le calendrier d'activité : chaque bloc, sa source (feuille de match / film / autre), ce qui
      s'affiche aujourd'hui quand ses données manquent (rien, titre sur du vide, pointillé nu,
      état conforme), et si le manque est « fonction absente » (capability du titre) ou
      « données absentes » (film pas encore analysé, pas analysé, périmètre vide).
- [x] P2.2 (squadagg.NoteFilmGaps commun aux 3 pages Emprise ; Sessions : une carte d etat par famille tiree du film + Portee ; Series temporelles Usages : 7 sections ; Synthese : weapon_records_gap + carte Records ; module partage features/_shared/filmGap + garde filmGap.guard.test.ts) Corriger : capability absente = bloc absent ; données absentes = bloc AFFICHÉ (titre +
      bloc parent + état vide conforme). Film en cours d'analyse → `AnalysisPendingSlot` ;
      film non analysé → `EmptyStateNotice`. Réutiliser `EmpriseFilmGapSection` / ses textes
      en le remontant dans un module partagé s'il sert à plusieurs pages (règle n°6 : à la 3e
      copie, helper + garde-rail). Le signal « en cours / non analysé » : réutiliser
      `ResolvePending` côté Go (aucune seconde règle), contrat openapi + `make generate-types`.
- [x] P2.3 (SessionSummaryCard, tiroir sans comparaison, onglet Usages vide, TimeseriesKdaBars : bloc parent ; calendrier : bloc conserve en echec de lecture ; erreur et vide de la Synthese traduits) Corriger tout `EmptyStateNotice` nu (sans bloc parent) de ces pages.
- [x] P2.4 (FILM_GAP_TEXT, synthesis.empty.*, session.detail.summary_unavailable_description, activityCalendarUnavailable ; textesSansPersonne etendu) Textes FR/EN factuels, sans personne ni impératif ni vocabulaire interne ; gardes de
      textes existantes (`textesSansPersonne.test.ts`, `noUnknownMentions`) vertes.

### P3 — Bulle (i) de la Tactique
- [x] P3.1 (positions du film / film positions ; test de chaînes FR+EN sans artefact) `tactical.plan.info_source_replay` (manifeste `lib/i18n/manifests/tactical.toml`) :
      retirer « artefacts de rejeu » ; constat factuel FR/EN.

### P4 — Clôture
- [x] P4.1 (gates au journal §4) `npx tsc -b --force`, vitest complet, lint web (+ colors, fields, knip), `go test` des
      paquets Go touchés + `internal/archlint`, golangci-lint sur ces paquets.
- [x] P4.2 (entree du journal 2026-10-17 ; decouvertes ci-dessous) Entrée `.ai/thought_log.md`, découvertes hors périmètre au plan §3.
- [x] P4.3 (un commit par etape, plus deux commits de suite P1 et P2 ; branche feat/retours-pages, aucun push) Un commit par étape sur `feat/retours-pages` ; pas de push.

## 3. Découvertes (non traitées, hors périmètre)

- `features/synthesis/SynthesisHighlightsSection.tsx` n'est importé par aucune page (seul son test le monte) : code mort avec
  une description d'état vide écrite en dur en français (« scope »). `knip` ne le voit pas parce que le test le référence.
- `SynthesisPage.test.tsx` : 14 tests `it.skip` d'une ancienne interface (dette annoncée dans le fichier depuis le merge 818a26bc).
- `SynthesisTopWeeksChart` : noms de séries en dur, non traduits (« Top 1 », « Autres », « Taux Top (%) », axe « Win % »).
- Nombres à point décimal restants, hors graphes des pages du lot : `toFixed` des cartes KPI de la Synthèse
  (`SynthesisPage.tsx`), de `_shared.ts formatPercent` (Sessions), des formateurs canoniques `lib/formatters` (`formatPercent`,
  `formatNumberFixed`), de `features/career/CareerChartsSection.xpHistory.tsx` et de `features/explorer` (carte d'activité, cadence).
  Le garde `rawTooltipValue.guard.test.ts` nomme le périmètre tenu et ces restes.
- Blocs dont l'absence dit « aucun match concerné » et que le lot laisse absents (à confirmer avec l'utilisateur) : l'Objectif des
  Sessions et des Séries temporelles (aucun match à objectif), le bandeau d'objectifs des Séries temporelles, les cartes de frags
  (aucun frag), `FragSunburst` (renvoie `null` sans frag alors que son voisin `FragWeaponBreakdown` affiche un état vide).
- `TimeseriesKdaBars` n'est monté que par la page de laboratoire `/lab/charts`.
- Vue match : l'Emprise solo (`buildSoloEmpriseBlock`) ne reçoit pas de résolveur (`Pending` nil) ; la Vue match dit « en cours »
  par ses propres `analysis_pending_parts`, donc `matches_film_*` y restent à zéro.
- Les causes de la Synthèse (`weapon_records_gap`) comptent « non analysé » dès qu'un scope porte des frags sans position : sur un
  historique ancien dont le film n'existe plus, le texte « Le film de ces matchs n'a pas été analysé » reste vrai mais permanent.
- `go-vet` du pre-commit tourne sans l'environnement CGO (déjà noté au lot Tactique).

## 4. Journal d'exécution

### P1 (2026-10-17)
- Périmètre migré sur `tipNumber` / `tipNumberAuto` / `tipNumberUpTo` (nouveau : décimales au plus, sans zéro inutile) et
  `axisTooltipFormatter` : 14 graphes de Sessions, 4 de la Synthèse (+ carte de chaleur et barres bipolaires), calendrier d'activité,
  14 des Séries temporelles, `HistogramChart`, `heatmap2DOption`, `sessionBarsTrendChart`, `WeaponAccuracyChart` (unité « tirs »
  traduite), `FragWeaponBreakdown`, `RadarChart`, `DonutChart` (gabarit `{d}%`), `ScatterChart`.
- Gardes : `rawTooltipValue.guard.test.ts` couvre tout `src/` (interpolation de `value`/`data`, gabarits `{c}`/`{d}`) et les
  fichiers de graphe du lot pour `toFixed`.

### P2 (2026-10-17)
- Règle : fonction absente (capability) = bloc absent ; film non lu sur un périmètre qui lit le film = bloc affiché (intertitre +
  bloc parent + `AnalysisPendingSlot` ou `EmptyStateNotice`) ; film lu partout et bloc sans sujet = bloc absent (même règle que
  l'Emprise de l'Escouade).
- Go : `squadagg.NoteFilmGaps` (ex-méthode de l'Escouade) pose `matches_film_pending` / `matches_film_missing` ; Sessions et
  Séries temporelles la branchent sur leur bloc d'Emprise via le résolveur partagé (`WithAnalysisPending`, garde de câblage
  `registry_pages_film_gap_wiring_test.go`) ; la Synthèse publie `weapon_records_gap`. `openapi.yaml` et `generated.ts` régénérés.
- Web : module partagé `features/_shared/filmGap` (décision, textes, `FilmGapState` / `FilmGapCard` / `FilmGapSection`), utilisé par
  l'Escouade, Sessions, Séries temporelles et la Synthèse ; garde `filmGap.guard.test.ts` (un seul lecteur des compteurs, un seul
  littéral « Film non analysé »). Texte « non analysé » rendu neutre (plus de « pas encore »).

| Page | Bloc | Source | Avant (données manquantes) | Après |
|---|---|---|---|---|
| Sessions | Bande KPI (résumé) | feuille de match | pointillé nu en tête de colonne | bloc parent + pointillé dedans ; texte sans « reconstruite » |
| Sessions | Issues, frags/morts/assistances (anneaux) | feuille de match | carte absente (`null`) | carte titrée + état vide |
| Sessions | Modes, placements, radars, FDA, perf, dégâts, MMR/OC-DR | feuille de match | ChartCard : état vide conforme | inchangé |
| Sessions | Écart au FDA attendu, balance de vies, engagement, MMR, XP | capability | absent | inchangé (fonction absente) |
| Sessions | Intensité, Premier sang | film (événements) | ChartCard : état vide conforme | inchangé |
| Sessions | Appui reçu | journal des morts | absent sans bloc, raison affichée si indisponible | inchangé |
| Sessions | Portée des engagements | film (positions) | carte absente, « aucun texte ne dit l'absence » | titre + « en cours » / « non analysé » dans la carte |
| Sessions | Ressources, Prendre, Vies, Équipement | film (résumé d'usage) | famille et intertitre absents | intertitre + carte d'état par famille sans carte dessinée |
| Sessions | Objectif | feuille de match | absent sans match à objectif | inchangé (périmètre sans objet) |
| Sessions | Tiroir de comparaison vide | — | pointillé nu | bloc parent + pointillé |
| Séries temporelles › Résumé | graphes (FDA, vies, assistances, performance, cartes) | feuille de match | ChartCard : état vide conforme | inchangé |
| Séries temporelles › Résumé | Répartition des frags (anneau) | feuille de match | `null` sans frag | inchangé (découverte) |
| Séries temporelles › Usages | Portée, Ressources, Par carte, Prises, Rendement, Isolement, Équipement | film | section et intertitre absents | intertitre + état par section sans donnée |
| Séries temporelles › Usages | onglet sans rien | — | pointillé nu | bloc parent + pointillé |
| Séries temporelles › Usages | Objectif | feuille de match | absent | inchangé |
| Séries temporelles | Appui reçu | journal des morts / capability | absent | inchangé |
| Synthèse | Records de distance par arme | film (positions) | section absente | titre + « en cours » / « non analysé » dans la carte |
| Synthèse | Meilleures semaines, par carte, par mode | feuille de match | chart absent si vide | carte titrée + état vide |
| Synthèse | Page sans charge utile / erreur de chargement | — | texte en dur FR avec impératif et mots internes / « Erreur : … » brut | constats traduits, bloc parent |
| Synthèse | Précision par arme | capability (natif) | absent | inchangé (fonction absente) |
| Calendrier d'activité | calendrier | feuille de match | bloc absent si la lecture échoue | bloc conservé, « Calendrier d'activité indisponible. » ; sans jour joué : état vide déjà conforme |

### P3 (2026-10-17)
- `tactical.plan.info_source_replay` : « artefacts de rejeu » / « replay artifacts » devient « positions du film » / « film positions ».

### P4 (2026-10-17) — gates rejoués sur l'arbre final
- Web : `npx tsc -b --force`, `npx vitest run` (complet), `npm run lint` (0 erreur, 26 avertissements préexistants), `lint:colors`,
  `lint:fields`, `node tools/knip-ratchet.mjs` (0/0/0), `node tools/lint-cross-feature-imports.mjs` (7 <= plafond).
- Go : `go test` des paquets touchés (service, squadagg, teammates, domain, api, api/wire, api/handlers, archlint, contracttest),
  `golangci-lint run --new-from-merge-base=origin/main` sur service, domain, api (0 issue), `TestOpenAPIYAMLIsUpToDate` vert.
