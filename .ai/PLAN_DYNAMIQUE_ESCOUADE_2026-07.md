# PLAN — Onglet Dynamique, Solde de vies, Écart d'engagement cumulé, essais visuels

> Date : 2026-07-23 · Branche cible : `feat/squad-dynamique`
> Contrat d'exécution : skill `plan-execution` (ordre strict, gates, statuts
> `[x]` fait / `[~]` couvert ailleurs (référence) / `[!]` non traité (justification écrite),
> aucune case vide à la clôture, zéro fix hors périmètre).
> Reprise de session : lire ce fichier (statuts) + `git log --oneline -10` sur la branche.

## Objectif et critère de succès

Quatre évolutions des concepts engagement / rendement / résistance :

1. Nouvel onglet Escouade **« Dynamique »** regroupant intensité, engagement,
   rendement, résistance (+ les nouveaux graphes ci-dessous).
2. **Séparation Rendement / Résistance** en deux graphes multi-joueurs (tous les
   joueurs affichés en même temps, plus de switch 1-joueur).
3. Nouvelle métrique **« Balance des dégâts »** (carry/liability) :
   `(dégâts infligés − dégâts subis) / PV-pour-tuer du titre`, cumulée — sur
   l'onglet Dynamique (1 courbe/joueur) et la page Session (aire cumulée + KPI).
4. **Écart d'engagement cumulé** (résidu × durée, en événements) — Timeseries,
   Dynamique, Session.
5. Essai visuel : **aire entre la courbe Rendement et la ligne « 1 vie »** sur
   Timeseries (jetable si non concluant).

Succès = les 3 onglets Escouade fonctionnels sur Halo Infinite ET Halo 5 (avec
dégradations propres), nouveaux graphes visibles et validés visuellement,
`make check-types` + vitest + suite Go vertes, i18n FR/EN complet.

## Décisions produit (TRANCHÉES — ne pas rouvrir en cours d'exécution)

| Sujet | Décision |
|---|---|
| Nom onglet | FR « Dynamique » / EN « Dynamics » |
| Titre carry | FR « Balance des dégâts » (cumul : « Balance des dégâts cumulée ») / EN « Damage balance » — axe/valeurs exprimés en vies (dégâts nets ÷ PV-pour-tuer) |
| Formule carry | `(damage_dealt − damage_taken) / hpToKill` par match ; `null` si l'un des deux absent |
| Carry sur H5 | MASQUÉ (pas de `damage_taken`) — aucun fallback FDA déguisé |
| Titre engagement cumulé | FR « Écart d'engagement cumulé » / EN « Cumulative engagement gap » |
| Unité engagement cumulé | événements en excès/déficit = Σ(résidu évén./min × durée_min) — la durée vient du backend (P4), pas de cumul du résidu brut |
| Rendement/Résistance | 2 ChartCards multi-joueurs (couleur = joueur, repère « 1 vie » sur les deux) ; le mode toggle 1-joueur est SUPPRIMÉ (code mort) |
| Aire Timeseries | Rendement uniquement, `areaStyle.origin: oneLife` (ECharts 5.6 OK), opacité ~0.10 ; Résistance reste en pointillé sans aire |
| Factorisation cumul | 3e copie du cumul signé → extraire un helper générique de `cumulativeFdaGap` (carry-forward D5 conservé) + garde-rail |
| Session — placement | Les 2 nouveaux charts s'insèrent dans `SessionChartStack`, avec échelle partagée A/B (`_compareScale`) comme `SessionFdaGapCumulative` |

## Pré-requis / blocker

- [ ] **B0** — Un WIP est en cours sur `feat/career-medals-and-fixes` (season pass
      badge, fichiers modifiés non commités). Règle : ne pas changer de branche
      pendant un travail en cours. Ce chantier démarre APRÈS livraison/commit du
      WIP, par `git checkout main && git pull` puis
      `git checkout -b feat/squad-dynamique`. L'utilisateur tranche le moment.

## Phase 1 — Onglet « Dynamique » (réceptacle)

- [ ] 1.1 Route file-based
      `apps/web/src/routes/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique.tsx`
      (calquer `contributions.tsx` ; ne jamais éditer `routeTree.gen.ts`).
- [ ] 1.2 `features/squad/SquadDynamiquePage.tsx` : y DÉPLACER depuis
      `SquadContributionsPage.tsx` : `SquadIntensityHeatmapChart`,
      `SquadEfficiencyChart`, `SquadEngagementSection` (retirés de Contributions,
      imports nettoyés). Vérifier sur pièces le hook de données utilisé par
      Contributions et le réutiliser (même query key — pas de nouvelle clé inline).
- [ ] 1.3 Navigation 3e onglet dans `SquadLayout.tsx` + libellés
      `features/squad/i18n.ts` (FR « Dynamique » / EN « Dynamics », parité typée).
- [ ] 1.4 Tests : adapter `SquadContributionsPage.test.tsx` (sections parties) +
      smoke test `SquadDynamiquePage` (sections présentes).

**Gate P1** : `make check-types` · `npx vitest run src/features/squad` (hors
sandbox, cf. mémoire vitest) · vérif visuelle des 3 onglets (dev :8000).

## Phase 2 — Séparation Rendement / Résistance (multi-joueurs)

- [ ] 2.1 `features/squad/charts/squadEfficiencyChart.ts` : généraliser
      `buildSquadRendementMultiOption` en un builder paramétré par métrique
      (`damagePerKill` | `damagePerDeath`), repère « 1 vie » conservé.
- [ ] 2.2 `SquadEfficiencyChart.tsx` : rendre DEUX ChartCards (« Rendement — dégâts
      par frag », « Résistance — dégâts par mort »), couleurs par joueur
      (`colorByPlayer`), légende ECharts togglable. H5 : `hasResistance === false`
      → seule la carte Rendement est rendue (comportement mono actuel).
- [ ] 2.3 Supprimer le mode toggle 1-joueur : `buildSquadEfficiencyTrackOption`,
      boutons segmentés, légende footer SVG — APRÈS grep des callers
      (`grep -r buildSquadEfficiencyTrackOption apps/web/src`). Supprimer les
      tests associés (règle 0 code mort).
- [ ] 2.4 i18n : titres des 2 cartes FR/EN dans `features/squad/i18n.ts`.
- [ ] 2.5 Tests : builder paramétré (2 métriques, bornes d'axe, joueur sans donnée).

**Gate P2** : `make check-types` · `npx vitest run src/features/squad` · vérif
visuelle Halo Infinite ET Halo 5 (dégradation mono-carte).

## Phase 3 — « Balance des dégâts » (frontend uniquement)

- [ ] 3.1 `lib/charts/cumulativeSeries.ts` : extraire de `cumulativeFdaGap.ts` le
      cumul signé générique avec report (carry-forward D5 : `null` ne fait pas
      avancer le cumul, la courbe reporte) + moyenne sur points valides.
      `cumulativeFdaGap` DÉLÈGUE (pas de duplication) ; étendre
      `cumulativeFdaGap.guard.test.ts` pour interdire toute réimplémentation de
      l'accumulateur hors de ce fichier (règle n°6 : helper + garde-rail).
- [ ] 3.2 `lib/charts/netLives.ts` : `netLives(damageDealt, damageTaken, hpToKill)`
      → `number | null` + tests unitaires (null si donnée manquante, division par
      le barème du titre).
- [ ] 3.3 Escouade (onglet Dynamique) : `SquadNetLivesChart` — 1 courbe CUMULÉE par
      joueur sur `match_order` partagé (pattern `squadFdaGapChart.ts`), couleur
      par joueur, ligne repère 0. Masqué si aucun joueur n'a de `damage_taken`.
- [ ] 3.4 Session : `SessionNetLivesCumulative` (pattern
      `SessionFdaGapCumulative.tsx`) — aire cumulée + pastille KPI « balance
      moyenne par match » ; intégration `_compareScale` (yDomain partagé A/B) ; branché
      dans `SessionChartStack.tsx`. Masqué si `damage_taken` absent des rows.
- [ ] 3.5 i18n FR/EN (`session.toml` + `features/squad/i18n.ts`) ; tooltip
      expliquant la formule avec le barème injecté via `substituteHpToken`
      (« 225 » Infinite / « 115 » H5) ; unité 0-centrée, positif = porte l'équipe.
- [ ] 3.6 Tests : `netLives.test.ts`, `cumulativeSeries.test.ts`, tests composants
      session + squad (pattern des équivalents FDA gap).

**Gate P3** : `make check-types` ·
`npx vitest run src/lib/charts src/features/session-detail src/features/squad` ·
vérif visuelle Session + Dynamique · vérif H5 = masqué proprement.

## Phase 4 — Écart d'engagement cumulé (Go + web)

Backend (durées absentes des contrats engagement — seul travail Go du chantier) :

- [ ] 4.1 `internal/domain/engagement_score.go` : ajouter `duration_seconds`
      (somme par bin) à `EngagementMatchSummary`.
- [ ] 4.2 `internal/service/engagement_player_service.go` +
      `engagement_timeseries_binning.go` : remplir la durée (match seul = sa
      durée ; bin session/semaine/mois = somme des durées). Test binning : somme
      par bin correcte.
- [ ] 4.3 `internal/service/engagement_squad_service.go` : ajouter
      `durations_seconds []int64` (aligné sur `labels`) au payload squad
      engagement. Test service.
- [ ] 4.4 `make generate-types` → `generated.ts` à jour (diff revu, committé).

Frontend :

- [ ] 4.5 Helper d'assemblage : écart pondéré par point =
      `engagement_score × (duration_seconds / 60)`, cumul via le helper 3.1
      (report si score ou durée `null`).
- [ ] 4.6 Timeseries : chart « Écart d'engagement cumulé » adjacent à
      `EngagementTimeseriesSection` (même onglet intensité), aire cumulée solo
      (pattern `TimeseriesFdaGapTrend`).
- [ ] 4.7 Dynamique : courbes cumulées par joueur — résidu joueur =
      `pace_observed − team_expected` par match, × durée (pattern
      `squadFdaGapChart`).
- [ ] 4.8 Session : `SessionEngagementCumulative` — cumul de
      `engagement_score × duration_seconds/60` (zip `match_series` ↔ rows triées
      par `start_time`, comme `SessionEngagementChart`) ; `_compareScale` ;
      branché dans `SessionChartStack` à côté des barres existantes (frontend
      seul — les rows session ont déjà la durée).
- [ ] 4.9 i18n FR/EN (`engagement.toml` + `timeseries.toml`/`session.toml` selon
      surface) ; axe/tooltip « événements (excès/déficit) ».
- [ ] 4.10 Tests : Go (4.2, 4.3) ; web : helper 4.5 + composants.

**Gate P4** : `cd apps/go-api && go test ./internal/service/... ./internal/api/...`
· `make generate-types` sans diff résiduel · `make check-types` · vitest ciblés ·
vérif visuelle Timeseries + Dynamique + Session.

## Phase 5 — Essai : aire Rendement → ligne « 1 vie » (Timeseries)

- [ ] 5.1 `features/timeseries/TimeseriesSquadAdapted.tsx` (section Rendement &
      Résistance) : `areaStyle` sur la SEULE série Rendement,
      `origin: oneLife`, opacité ~0.10, couleur alignée sur le dégradé existant.
      Commit isolé (revert facile si non concluant).

**Gate P5** : vérif visuelle par l'utilisateur (thème clair + sombre). Non
concluant → revert du commit P5, item passé `[!]` avec justification.

## Hors périmètre (ne pas traiter, consigner ici)

- Variante « part des dégâts − part des morts » intra-escouade (angle
  complémentaire acté mais NON retenu pour ce lot).
- Toute retouche des graphes FDA gap existants au-delà de la délégation 3.1.

## Découvertes en cours d'exécution

(noter ici, ne pas traiter)

## Livraison

- Chaque phase = 1+ commits sur `feat/squad-dynamique` (demander avant commit).
- Avant la clôture : gates complets (`make check-types`, `make test-web`,
  `go test ./...` apps/go-api, lint si Go touché) + entrée `thought_log.md`.
- Merge dans `main` = déploiement prod automatique : prévenir l'utilisateur.

## Estimation

| Phase | Effort |
|---|---|
| P1 onglet | moyen |
| P2 séparation | rapide |
| P3 solde de vies | moyen |
| P4 engagement cumulé | lourd (seule phase Go) |
| P5 aire | rapide |
