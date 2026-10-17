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
- [ ] P3.1 `tactical.plan.info_source_replay` (manifeste `lib/i18n/manifests/tactical.toml`) :
      retirer « artefacts de rejeu » ; constat factuel FR/EN.

### P4 — Clôture
- [ ] P4.1 `npx tsc -b --force`, vitest complet, lint web (+ colors, fields, knip), `go test` des
      paquets Go touchés + `internal/archlint`, golangci-lint sur ces paquets.
- [ ] P4.2 Entrée `.ai/thought_log.md`, découvertes hors périmètre au plan §3.
- [ ] P4.3 Un commit par étape sur `feat/retours-pages` ; pas de push.

## 3. Découvertes

(à remplir)
