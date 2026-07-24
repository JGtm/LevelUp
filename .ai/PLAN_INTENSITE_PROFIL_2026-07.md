# PLAN — Refonte « Intensité » : profil médian + enveloppe de variabilité

> Date : 2026-07-24 · Branche : `feat/intensite-profil` (worktree dédié, base main
> c3dd697e2 = lot Dynamique mergé). Contrat : skill `plan-execution` (ordre strict,
> gates, statuts `[x]`/`[~]`/`[!]`, zéro fix hors périmètre, découvertes consignées).
> Décisions produit validées par l'utilisateur le 2026-07-24 (artifact comparatif,
> solution S1 + grille 2 colonnes) — NE PAS rouvrir.

## Objectif

Remplacer la heatmap « Intensité » (matchs × 10 phases, cellules 0..1) par un
« profil d'activité par phase » : pour chaque joueur, la MÉDIANE des parts de
frags par phase sur les manches du scope (trait épais) + l'ENVELOPPE
interquartile P25–P75 (aplat), qui rend visible l'irrégularité (cas « joueur en
dents de scie »). Le nom UI reste « Intensité ». Aucune pastille narrative.

Surfaces : Escouade (onglet Dynamique, multi-joueurs), Timeseries (solo,
remplace TimeseriesIntensityHeatmap), Sessions (solo, nouveau).

## Décisions (TRANCHÉES)

| Sujet | Décision |
|---|---|
| Statistique | médiane des parts par phase ; enveloppe = P25–P75 des manches |
| Part par manche | `phases[i] / Σ phases` (la normalisation per-match du payload préserve les ratios — vérifié) |
| Manches exclues | somme de phases = 0 (aucun frag) → la manche ne contribue pas |
| < 4 manches exploitables | médiane seule, PAS d'enveloppe (bande dégénérée) — seuil nommé en constante |
| Repère | ligne pointillée à 10 % = activité uniforme |
| Escouade — disposition | grille 2 colonnes de panneaux par joueur : 1 joueur = pleine largeur ; 2 = une rangée ; 3 = 2+1 ; 4 = 2×2. Échelle Y PARTAGÉE (max commun aux panneaux). Un joueur sans manche exploitable n'a pas de panneau. |
| Toggle « all » | supprimé — le multi-panneaux remplace le switch joueur ; la ligne `all` du payload n'est plus consommée |
| Couleurs | Escouade : couleur par joueur (`colorByPlayer` existant) ; solo : couleur série standard du chart ; enveloppe = même teinte, opacité faible (~0.16) — tokens uniquement |
| i18n | FR titre « Intensité », sous-titre « Répartition des frags par phase de match » ; tooltip expliquant médiane / enveloppe / repère 10 % ; EN équivalents — parité typée |
| Heatmap | SUPPRIMÉE avec builder + wrappers + tests (0 code mort) après migration des 2 surfaces existantes |

## Phase 1 — Coeur + Escouade (Dynamique)

- [x] 1.1 Helper pur `apps/web/src/lib/charts/phaseProfile.ts` :
      `phaseShares(phases)` (null si somme 0), `phaseProfile(rows)` →
      `{ median[10], p25[10], p75[10], nMatches }` (quantiles R-7 sur manches
      exploitables). Constantes `PHASE_COUNT` + `MIN_MATCHES_FOR_ENVELOPE`.
      Tests unitaires `phaseProfile.test.ts` (dents de scie : enveloppe large ;
      manche vide exclue ; < 4 manches → nMatches signale « médiane seule »).
- [x] 1.2 Builder `features/squad/charts/squadIntensityProfileChart.ts` :
      option ECharts multi-grilles (`computeGrids` : 2 colonnes, N panneaux ;
      N=1 pleine largeur ; échelle Y partagée `sharedYMax` ; repère 10 % via
      markLine ; bande = paire de séries stack `env-{gi}` (base P25 transparente
      + P75−P25 areaStyle opacité 0.16), médiane en ligne ; titres multi via
      `title[]`). Réutilise `SQUAD_INTENSITY_PHASE_LABELS` (pas de 2e copie).
      Tests builder (N=1/2/3/4, échelle commune, panneau absent si aucune manche
      exploitable, médiane seule si < 4, markLine à 0,1).
- [x] 1.3 Composant `features/squad/SquadIntensityProfileChart.tsx` : consomme
      `intensity_profile.rows` par joueur (PAS la ligne `all`, filtrée par
      `playerOrder` + garde `key !== 'all'`), couleurs `colorByPlayer`, titres de
      panneaux = gamertags, hauteur = f(rangées). Monté dans
      `SquadDynamiquePage.tsx` à la place de `SquadIntensityHeatmapChart` (titre
      « Intensité » conservé, sous-titre + InfoTooltip ajoutés).
- [x] 1.4 i18n `features/squad/i18n.ts` : `subtitle` + `tooltip` + `medianLabel`
      + `envelopeLabel` + `refLabel` FR/EN (parité typée) ; retiré les clés
      heatmap mortes (`description`, `toggleLabel`, `allLabel`, `zLabel`) — plus
      aucun consommateur côté squad (Timeseries passe son propre littéral zLabel).
- [x] 1.5 Tests composant `SquadIntensityProfileChart.test.tsx` (rendu, exclusion
      `all`, ordre, couleur, état vide) + mock adapté dans
      `SquadDynamiquePage.test.tsx` (`SquadIntensityProfileChart`).

**Gate P1** : `npm run typecheck` (tsc -b, purger `node_modules\.tmp` AVANT —
le gate de référence, PAS `tsc --noEmit`) · `npx vitest run src/features/squad
src/lib/charts` (dangerouslyDisableSandbox).

## Phase 2 — Timeseries (solo)

- [x] 2.1 `features/timeseries/TimeseriesSquadAdapted.tsx` : NOUVEAU composant
      `TimeseriesIntensityProfile` (un panneau pleine largeur, médiane +
      enveloppe) réutilisant le builder P1 `buildSquadIntensityProfileOption` en
      N=1 (panel `label:''` → titre ECharts vide, sans impact layout ; couleur
      `chart-series-2`). ZÉRO duplication de géométrie. Rows =
      `data.intensity_rows` (tri start_time ASC conservé). Vide détecté via
      l'absence de `series` dans l'option (le builder l'omet si aucune manche
      exploitable). `TimeseriesIntensityHeatmap` CONSERVÉ (suppression Phase 3).
- [x] 2.2 `TimeseriesPage.progression.tsx` : import + montage swappés (titre
      « Intensité » conservé, sous-titre + InfoTooltip). Clés ajoutées dans
      `timeseries.toml` (surface = manifeste timeseries) : `intensity_subtitle`,
      `intensity_tooltip`, `intensity_median`, `intensity_envelope`,
      `intensity_ref` (FR+EN) ; `node build_i18n_manifests.mjs` régénéré
      (idempotent, seul `generated/timeseries.ts` change). `intensity_z` (heatmap)
      conservé → retiré en Phase 3.
- [x] 2.3 Aucun test Timeseries existant n'était cassé (aucun ne montait
      l'intensité) → `[~]` pour « adaptation ». NOUVEAU test
      `TimeseriesIntensityProfile.test.tsx` (rendu, liste vide, manches sans frag
      → état vide) calqué sur `TimeseriesFdaGapTrend.test.tsx`.

**Gate P2** : typecheck (référence) · `npx vitest run src/features/timeseries`.

## Phase 3 — Sessions (solo) + suppression heatmap

- [ ] 3.1 VÉRIFIER sur pièces la source de données phases pour le scope session :
      le payload session expose-t-il les phases par match ? Sinon, deux voies
      DÉFINIES : (a) si un endpoint existant (voie Timeseries/`buildIntensityRows`)
      peut être requêté sur les match_ids de la session, l'utiliser (query key
      dans `lib/query/keys.ts`, jamais inline) ; (b) sinon, exposer les phases
      dans le payload session côté Go en MIROIR du calcul intensité existant
      (service session + openapi — attention : schémas manuels réconciliés par
      TestOpenAPISchemaDrift, cf. leçon P4 du lot Dynamique) puis
      `npm run generate-types` idempotent. Choisir la voie (a) si possible,
      documenter le choix dans le plan.
- [ ] 3.2 `features/session-detail/SessionIntensityProfile.tsx` : panneau solo
      médiane + enveloppe sur les manches de la session ; intégré dans
      `SessionChartStack.tsx` ; échelle comparaison A/B (`_compareScale`) SI le
      pattern s'applique (domaine = parts 0..max — trancher comme les autres
      charts session). i18n session.toml FR/EN.
- [ ] 3.3 SUPPRESSION heatmap : `SquadIntensityHeatmapChart.tsx`,
      `squadIntensityHeatmapChart.ts`, `TimeseriesIntensityHeatmap`, leurs tests
      et imports — APRÈS grep des callers (`grep -rn IntensityHeatmap
      apps/web/src`). `heatmapRampTokens('frequency')` : vérifier les autres
      callers avant toute suppression de helper partagé.
- [ ] 3.4 Tests composant session (calquer SessionFdaGapCumulative.test).

**Gate P3** : typecheck (référence) · vitest `src/features/session-detail` +
suites touchées · si Go modifié : `go test ./internal/service/...` (CGO msys64 :
`$env:CGO_ENABLED='1'` + PATH `C:\msys64\ucrt64\bin`) + gofmt sur tout Go touché.

## Gates finaux du lot

- `npm run typecheck` cache purgé · `npm run lint` (baseline 8 warnings) ·
  `npx vitest run` complet · si Go touché : `go test ./...` + `go vet` +
  `make go-api-lint` (Git Bash).
- Entrée `.ai/thought_log.md` (règle obligatoire) avant remise au superviseur.
- Vérification visuelle = passe finale utilisateur (doctrine du projet).

## Hors périmètre

- Pistes S2/S3/S4 de l'artifact (non retenues).
- Toute retouche des autres charts Dynamique/Timeseries/Sessions.

## Découvertes en cours d'exécution

(consigner ici, ne pas traiter)

- **P1** Après le remontage, le composant wrapper `SquadIntensityHeatmapChart.tsx`
  n'a plus aucun consommateur côté squad (seul le BUILDER
  `charts/squadIntensityHeatmapChart.ts` reste utilisé, par Timeseries
  `TimeseriesSquadAdapted.tsx`). Conforme au périmètre : sa suppression est
  planifiée en Phase 3 (avec son test). Non traité en P1. Son test
  `charts/squadIntensityHeatmapChart.test.ts` reste vert (le builder est intact).
- **P1** Gate `npx vitest run src/features/squad src/lib/charts` : aucun besoin de
  `dangerouslyDisableSandbox` (a tourné dans le sandbox par défaut).
- **P2** Après le swap, la fonction `TimeseriesIntensityHeatmap`
  (TimeseriesSquadAdapted.tsx) n'a plus de consommateur (import retiré de la page)
  mais reste EXPORTÉE (non flaggée unused par tsc) — suppression planifiée Phase 3
  (item 3.3). Elle référence toujours `buildSquadIntensityHeatmapOption` → cet
  import reste utilisé. La clé i18n `timeseries.progression.intensity_z` devient
  inutilisée (aucun garde-rail de clé morte : validation manifest = fr+en présents
  seulement) — retirée avec la heatmap en Phase 3.
- **P2** Aucune duplication de la logique d'exploitabilité : le builder P1 signale
  le vide en omettant `series` ; le wrapper solo teste `opt.series`. Pas de retouche
  des fichiers P1 commités (le builder gère déjà `label:''`).
