# Plan — Couleurs dédiées des stats de combat (2026-09-17)

Branche : `claude/couleurs-stats-combat` (depuis `origin/feat/v75` 56f918258).
Décision utilisateur (2026-09-17) : jeux « validés » de l'artefact « Couleurs des stats de combat »
(https://claude.ai/artifact/DwzugjxXLTsGdpGSygdd1c, version 4).

## Contexte

Aucune stat de combat n'avait de jeton : une vingtaine de jetons empruntés à d'autres rôles
(issue, palier de performance, série de graphique, info, comparaison…). Les assistances avaient
cinq teintes selon la page ; le sens de l'assistance (Relations) empruntait `compare-a/b`, dont
l'ambre était identique au joueur d'escouade 2.

## Jetons et valeurs validées

| Jeton | Défaut | Okabe-Ito | Cividis | Tol |
|---|---|---|---|---|
| `stat-kills` | #059669 | #008E68 | #0072B2 | #228833 |
| `stat-deaths` | #F43F5E | #D55E00 | #D55E00 | #EE6677 |
| `stat-assists` | #0284C7 | #56B4E9 | #888888 | #2996BE |
| `assist-received` | #A16207 | #E69F00 | #E1C752 | #A75324 |
| `assist-given` | #7C3AED | #CC79A7 | #1B3158 | #AA4499 |

Règles (mêmes seuils que `squadPlayerTokens.test.ts`) : ΔE OKLab×100 ≥ 15 toutes paires ;
simulation protan/deutan ≥ 8 sur défaut et Okabe-Ito ; contraste ≥ 3:1 sur les deux surfaces
(défaut, Tol) ou sur une seule (Okabe-Ito : bleu ciel et orange infaisables sur fond clair sans
collision ; Cividis : rampe séquentielle).

## Périmètre de migration

MIGRÉ = la couleur dit « c'est un frag / une mort / une assistance / ce sens d'assistance ».
CONSERVÉ = la couleur encode autre chose (équipe, joueur, issue du match, qualité, récit).

Migrés :
- Tuile de match (`components/ui/match-card.tsx`) : barre FDA et libellés.
- `features/_shared/EncounterSplitBars.tsx` (KDSplitBar), `palmares/RelationsTable.tsx` (Frags/Morts),
  `palmares/PalmaresRelationsPage.tsx` (barre frags/morts de la bête noire).
- `career/CareerRivalsSection.tsx` (barres et nombres frags/morts des rivaux).
- `synthesis/SynthesisPage.tsx` (barre et nombres FDA), `synthesis/SynthesisWeaponRangeSection.tsx` (KILLS/DEATHS_TOKEN).
- `session-detail/SessionFdaBars.tsx`, `session-detail/SessionKillsDonut.tsx`.
- `explorer/combatChartOptions.ts`, `explorer/explorerCadenceChart.ts`, `explorer/ExplorerTargetSampleStats.tsx` (barre frags par arme).
- `timeseries/TimeseriesKdaTrend.tsx` (frags, morts, bonus assistances), `timeseries/TimeseriesFormCharts.tsx`,
  `timeseries/TimeseriesPage.distributions.tsx` (histogramme des frags ; les autres histogrammes ne sont pas des stats de combat).
- `match-view/MatchSummaryCharts.tsx` (barres F/D/A).
- `components/charts/FirstBloodLanes.tsx` (premier frag / première mort).
- `squad/charts/squadPerformanceLineCharts.ts` (segment bonus assistances).
- `match-replay/ui/ReplayCountersBadge.tsx` (triplet F/D/A), `ReplayAssistMark.tsx` (icône),
  `ReplayKillFeed.tsx` (fond de ligne assistée).
- `_shared/assists/AssistButterflyBar.tsx` : ASSIST_RECEIVED/GIVEN_TOKEN → `assist-received` / `assist-given`.

Conservés (justification) :
- Couleur d'ÉQUIPE : `MatchKDCumulChart`, `MatchCadenceChart`, `MatchTugOfWarChart`, noms du kill feed, marques de la piste du rejeu (`ReplayMarkTrack`).
- Couleur de JOUEUR : `squadPerMinuteChart`, frags/morts par joueur de `squadPerformanceLineCharts`, `MatchFragDiffChart`, `SquadAssistPairsChart`, cycle par tueur de `MatchAssistChart`.
- ISSUE du match : `TimeseriesKdaBars` (barre colorée par victoire/défaite).
- QUALITÉ : échelles de `SessionBriefing/KpiGrid`.
- RÉCIT : cartes Némésis / Souffre-douleur (`MatchNemesisCards`, `PlayerDetailPanel`), frags parfaits (`match-card`, perf-tier-3).
- ENCRES DU REJEU : marqueur de mort sur la carte (`ReplayCanvas`), icône de mort sans tueur (`divergent-neutral`).

## Étapes

- [x] E1 — Jetons : union + ALL_TOKENS (`semantic-tokens.ts`), 4 palettes, valeurs CSS de repli (`styles/globals.css`), snapshot de couverture.
- [x] E2 — Garde-fou famille : `lib/accessibility/combatStatTokens.test.ts` (contraste, ΔE, daltonisme selon la palette).
- [x] E3 — Migration des sites listés ci-dessus.
- [x] E4 — Garde-fou anti-emprunt : test ratchet qui interdit un jeton emprunté sur une ligne qui colore frags/morts/assistances, allowlist datée des conservés.
- [x] E5 — Règle écrite : skill `color-tokens` (famille, tons de part 35/65/100 %, bornes 25/50 %).
- [x] E6 — Gates (tsc purgé, eslint, lint couleurs/champs, vitest complet) + vérification visuelle locale (tuile, Synthèse, Relations, vue match, rejeu) + thought_log.
  Rendu vérifié en local sur tuiles de match, Synthèse et Relations (valeurs CSS calculées + captures). Vue match et rejeu : `[~]` couverts par les tests (fixtures HTML du rejeu, options de graphique), pas regardés à l'écran.

## Découvertes

- E2 : la conversion OKLab existait déjà en deux copies (`squadPlayerTokens.test.ts`, `scales/fragClass.guard.test.ts`) ; une troisième aurait violé la règle des 2 copies. Centralisée dans `lib/accessibility/colorDistance.ts`, les deux tests migrés, ratchet `colorDistance.guard.test.ts`.
- E3 : le jeton `bonus` ne sert plus qu'au cœur de la faille du rejeu (`useReplayInks`) ; ses commentaires « assistances » (semantic-tokens + 4 palettes) mis à jour dans le même lot.
- E3 : les frags des graphiques passent du bleu `chart-series-1` (convention retenue le 2026-09-09 pour la section portée des armes) au vert `stat-kills` — conséquence directe du choix du 2026-09-17, signalée à l'utilisateur.
- E4 : le ratchet ne voit qu'une ligne à la fois ; les sites conservés sans mot « kill/frag/death/assist » sur la ligne (couleurs d'équipe, de joueur) ne sont pas listés dans `KEPT`, car ils ne déclenchent rien. 7 accents de sous-type de frag sont justifiés dans `KEPT`.
- Okabe-Ito : contraste 3:1 sur fond clair infaisable (recherche exhaustive) — exemption « une surface » documentée dans le garde-fou.
