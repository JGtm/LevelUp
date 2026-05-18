# ADR 0015 — Player Profile V1 (Ascension)

**Date** : 2026-05-18
**Status** : Partially implemented (3/10 commits livrés, 7 reportés)
**Branch** : `feat/player-profile-ascension` (3 commits sur 10)

## Context

Le profil joueur V1 prévu par `PLAN_PLAYER_PROFILE_ASCENSION.md` propose une page riche
qui agrège narrative (radar 6 axes + rôles), LUSR (tier + 8 composantes), style
(FK/FD + engagement), et coaching (suggestions de défis + boucle de campagne
d'amélioration). C'est la couche structurelle dont V2 (Streaks/Records/Milestones,
ADR 0014) supposait l'existence — mais qui n'avait jamais été livrée dans le code.

V2 a contourné l'absence avec un mini-service `internal/progression/profile/`
construit pour le strict besoin du coach (μ + sub-tier + LOWESS sur μ, ~250 lignes).
Ce sprint V1 visait à construire le service complet et faire 9 livraisons supplémentaires
pour boucler la page profil + la mécanique de campagne.

**Décision d'engagement** : sprint V1 attaqué sur la branche
`feat/player-profile-ascension` créée depuis HEAD de V2 (feat/progression-tracking-
ascension). Sur les 10 commits prévus, **3 ont été livrés**, **7 sont en dette**.

## Decisions livrées (commits 1-3)

### Commit 1 — Tagging du catalogue

Extension du type `prestige.Template` avec 3 champs :
- `LUSRComponents []string` : composantes LUSR ciblées (référentiel `CompositeWeights`)
- `RadarAxes []string` : axes narrative 6 ciblés (optionnel)
- `IsLongTerm bool` : true si `window_type=rolling_days` OR (`last_n_matches`
  AND `eval_type=threshold`)

Migration `metadata.duckdb` ajoutant 3 colonnes (CSV simple + boolean) à
`challenge_template`. Loader TOML étendu. 27 templates Halo Infinite taggés via
mapping mécanique metric → tags. 4 templates marqués `# TODO: tag` pour métriques
identitaires (matches_played, modes_played, maps_explored, firefight_waves) sans
LUSR component direct.

### Commit 2 — Enrichissement catalogue (gap closing)

5 nouveaux templates et 2 nouveaux arcs preset pour fermer les gaps §3.2 du plan :
- Deaths vs Expected (24% du composite, **0 templates** auparavant) : 2 nouveaux
- Defensive Resistance (6%, **0 templates**) : 1 nouveau
- Accuracy long-terme (10%, manquait habitude rolling_days) : 1 nouveau
- Kills vs Expected long-terme : 1 nouveau

Arcs : `marksman` (axe Accuracy : session → 3sessions → 30 jours) et `survivor`
(axe Survival : kdr_session → deaths_vs_expected weekly → monthly).

Totals : 27 → 32 templates, 4 → 6 arcs. Couverture LUSR complète sur les 8 composantes
(au moins 1 template par composante, plusieurs sur les composantes lourdes).

### Commit 3 — Inversion math LUSR

`RequiredCompositeForTier(currentMu, targetMu, sigma) float64` dans
`internal/analysis/skill_rating.go`. Binary search [0, 1] avec guards aux bornes
pour éviter la convergence asymptotique. Permettra (commit 4 reporté) à la
Section B du profil d'afficher pour chaque composante "cible à atteindre par match
pour stabiliser au sub-tier suivant".

7 tests unitaires couvrant : stabilité (target=current → 0.5), delta atteignable
avec round-trip via `trueskillUpdate`, delta non-atteignable (clamp 0/1), parcours
des 6 tiers (Bronze → Onyx).

## Decisions reportées (commits 4-9, en dette)

Les décisions architecturales sont **prises** dans le plan §4-§5 mais **non
implémentées**. Inventaire de la dette :

### Commit 4 — Service PlayerProfile + endpoint (reporté)

**À implémenter** : `service/player_profile_service.go` (~300 lignes) avec
`BuildProfile(xuid, titleSlug, window=100)` qui orchestre 6 sous-méthodes :
- `aggregateNarrative` (rôles + radar 6 axes) — réutilise `narrative.*`
- `computeStyleSignature` (FK/FD ratio) — réutilise `narrative.first_events`
- `computeEngagement` (score + régularité) — réutilise `analysis/temporal/engagement_score`
- `computeLUSRComponents` (8 scores moyens + top 20% perso)
- `identifyLeverages` (top 2 leviers = (1-current) × weight + axes radar)
- `selectSuggestedChallenges` (matching catalogue via `LUSRComponents` taggés)

Endpoint `GET /api/v1/players/{slug}/profile?window=100` (422 si données insuffisantes
< 30 matchs).

**Refactor critique** : remplacer le mini-service `internal/progression/profile/`
(commit 6 du V2) par ce nouveau service complet. Refactor du orchestrateur
post-sync `EvaluateProgressionAfterSync` pour appeler le nouveau service.

### Commit 5 — ImprovementCampaign (reporté, le plus gros)

**À implémenter** :
- Migration `improvement_campaign` table (stats.duckdb) + ALTER `challenge` ADD `campaign_id`
- `service/improvement_campaign_service.go` (~250L) avec `StartCampaign`,
  `EvaluateCampaign` (LOWESS + Mann-Whitney U), `LinkChallengeToCampaign`,
  `PauseCampaign`, `CloseCampaign`, `AbandonCampaign`, `GetActiveCampaign`
- Hook post-sync : extension d'`EvaluateProgressionAfterSync` (V2 commit 6) pour
  appeler `EvaluateCampaign` si campagne active
- 6 endpoints HTTP `/campaigns/*`
- 5 raffinements algorithmiques (§4.5.3 du plan) : LOWESS lissé pas delta brut,
  phrasing no-causalité, filtre playlist, Mann-Whitney pour confirmation,
  auto-suggestion de clôture (pas auto-fermeture)
- 2 gardes-fous UX (§4.5.4) : copy mesurée, pas de fausse précision (2 décimales,
  toujours afficher sample_size)

**Alertes coach V2 déjà déclarées** (commit 5 du V2) mais jamais émises faute de
ce service : `AlertTypeCampaignProgress`, `AlertTypeCampaignCloseAuto`. Câblage
à compléter dans ce commit.

### Commits 6-7 — UI Frontend (reportés)

Composant `PlayerProfileCard` (5 sous-composants : Identity, StyleDiscipline,
Performance, Progression, InsufficientData) + `CampaignTracker` (sticky en tête)
+ mini-modale de démarrage. Hooks `usePlayerProfile`, `useActiveCampaign`,
`useCampaignMutations`. Tests Vitest.

### Commits 8-9 — Rename nav + i18n (reportés)

NavL1 : "Objectifs" → "Ascension". i18n manifests pour les phrases éditoriales
(identité, leviers, coaching, campagne) avec règle R2 stricte sur la copy
no-causalité.

### Commit 10 — Cet ADR

Documenté ci-présent.

## Key architectural choices (décidés)

### Self-benchmark (pas inter-joueurs)

Le pool de joueurs LevelUp est trop petit pour produire un benchmark inter-joueurs
fiable. Décision §2 du plan : afficher le **top 20% personnel** par composante
+ l'inversion mathématique LUSR pour la cible théorique. Évite de comparer un
joueur à des distributions tirées de 10-50 utilisateurs.

### Renaming "Faiblesses" → "Axes d'amélioration"

UX positif. Cohérent avec la philosophie V2 (feedback positif uniquement,
plan §6.1 du V2).

### Campagne d'amélioration vs Saisons (DEPRECATED)

Différences fondamentales (cf. §4.5.2 du plan) :
- Activation **volontaire**, pas calendaire
- Durée **indéterminée**, fermable à tout moment
- **1 axe ciblé** (pas multi-arcs)
- **Pas de pénalité régression**
- **Pas de reward de fin** — la progression visible suffit

Esprit Duolingo : trajectoire personnelle suivie volontairement, pas de note finale.

### 5 raffinements algorithmiques (commit 5 reporté)

R1. Delta lissé LOWESS (pas delta brut, variance ±0.05 sur 30 matchs)
R2. Phrasing no-causalité (corrélation temporelle, pas relation de cause)
R3. Filtre playlist optionnel
R4. Mann-Whitney U pour milestone "progression confirmée" (p < 0.05)
R5. Auto-suggestion de clôture (pas auto-fermeture)

### Coexistence avec le mode libre

La campagne est **opt-in**, jamais imposée. Le flow Prestige libre (mode `libre`
du CreateChallenge) reste 100% disponible. Un défi sans `campaign_id` (NULL)
existe à part, juste "pour le fun". L'UX ne hiérarchise pas la valeur des deux.

## Consequences

### Positives (livrées)

- Catalogue Prestige enrichi et tagué : couverture LUSR complète sur les 8
  composantes (gap critique Deaths vs Expected fermé).
- Inversion math LUSR disponible : peut être consommée immédiatement par d'autres
  features sans dépendance circulaire.
- Aucune régression sur V2 (le mini-service `progression/profile/` reste
  fonctionnel en attendant le refactor).

### Négatives (dette)

- **Page profil non livrée** : aucune UI n'expose les données enrichies. Les
  3 commits livrés sont des fondations sans usage utilisateur direct.
- **Coach V2 alertes campagne dormantes** : `AlertTypeCampaignProgress` et
  `AlertTypeCampaignCloseAuto` sont définies dans le code coach (V2 commit 5)
  mais ne sont jamais déclenchées tant que `ImprovementCampaign` n'existe pas.
- **Streaks perf-based toujours non câblés** (V2 dette identique) : nécessitent
  les médianes personnelles que `PlayerProfile.computeLUSRComponents` exposerait.
- **Refactor mini-service profile/** non fait : 2 sources de vérité différentes
  côtoient le code tant que le service V1 complet n'est pas livré.

### Reprise du sprint

La branche `feat/player-profile-ascension` est prête à reprendre. Commits 4-9
peuvent être attaqués dans une nouvelle session (estimation : 6-8j de dev cumulé).
Ordre recommandé :
1. Commit 4 (PlayerProfile service) — débloque tout le reste
2. Commit 5 (Campaign) — le plus gros, mais autonome
3. Commits 6-7 (UI) — réutilisent l'API du 4 + données du 5
4. Commits 8-9 (rename + i18n) — pure surface texte
5. Mettre à jour cet ADR au statut "Accepted" quand 10/10 livrés

## References

- Plan retenu : `.ai/PLAN_PLAYER_PROFILE_ASCENSION.md` (705 lignes, 10 commits cadrés)
- ADR 0004 : narrative engine (radar 6 axes + 8 rôles — consommé par
  `aggregateNarrative` commit 4)
- ADR 0014 : Progression Tracking V2 (Streaks/Records/Milestones/Coach) qui
  attend ce V1 pour ses 2 alertes campagne et ses 2 streaks perf-based
- Plan rejeté : `.ai/PLAN_SEASONS_ASCENSION.md` (DEPRECATED, justification du
  choix Campagne vs Saison dans §4.5.2 du plan V1)
