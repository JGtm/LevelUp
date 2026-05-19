# ADR 0015 — Player Profile V1 (Ascension)

**Date** : 2026-05-18 (créé) / 2026-05-18 (clôturé)
**Status** : Accepted (10/10 commits livrés)
**Branch** : `feat/player-profile-ascension` — prête à merger

## Context

Le profil joueur V1 prévu par `PLAN_PLAYER_PROFILE_ASCENSION.md` propose une page riche
qui agrège narrative (radar 6 axes + rôles), LUSR (tier + 8 composantes), style
(FK/FD + engagement), et coaching (suggestions de défis + boucle de campagne
d'amélioration). C'est la couche structurelle dont V2 (Streaks/Records/Milestones,
ADR 0014) supposait l'existence — mais qui n'avait jamais été livrée dans le code.

V2 avait contourné l'absence avec un mini-service `internal/progression/profile/`
construit pour le strict besoin du coach (μ + sub-tier + LOWESS sur μ, ~145 lignes).
Ce sprint V1 a construit le service complet, la boucle Campagne d'amélioration,
l'UI front + i18n, et le rename Ascension. **10/10 commits livrés.**

## Decisions livrées (commits 1-10)

### Commit 1 — Tagging du catalogue

Extension du type `prestige.Template` avec 3 champs (LUSRComponents, RadarAxes,
IsLongTerm). Migration `metadata.duckdb` ajoutant 3 colonnes à `challenge_template`.
Loader TOML étendu. 27 templates Halo Infinite taggés via mapping mécanique
metric → tags.

### Commit 2 — Enrichissement catalogue (gap closing)

5 nouveaux templates et 2 nouveaux arcs preset pour fermer les gaps §3.2 du plan :
- Deaths vs Expected (24% du composite, **0 templates** auparavant) : 2 nouveaux
- Defensive Resistance (6%, **0 templates**) : 1 nouveau
- Accuracy long-terme (10%) : 1 nouveau
- Kills vs Expected long-terme : 1 nouveau

Arcs : `marksman` et `survivor`. Totals : 27 → 32 templates, 4 → 6 arcs.
Couverture LUSR complète sur les 8 composantes.

### Commit 3 — Inversion math LUSR

`RequiredCompositeForTier(currentMu, targetMu, sigma) float64` dans
`internal/analysis/skill_rating.go`. Binary search [0, 1] avec guards aux bornes.
7 tests unitaires (stabilité, round-trip via `trueskillUpdate`, parcours des 6
tiers Bronze → Onyx).

### Commit 4 — Service PlayerProfile complet + endpoint

`internal/progression/profile/service.go` augmenté avec `NewServiceFromPlayerDB(pdb)`
+ `BuildProfile(...)` orchestrant 6 sous-méthodes idempotentes (fillSectionB,
aggregateNarrative, computeStyleSignature, computeEngagement, computeLUSRComponents,
selectSuggestedChallenges). Le V2 caller (`post_sync_progression.go:Load`)
reste API-compatible — aucune régression V2.

`queries.go` (nouveaux helpers SQL) + endpoint HTTP
`GET /api/v1/players/{slug}/profile?window_days=30`. Pattern aligné sur
`ProgressionHandler`. 8 tests unitaires Windows + 3 tests intégration CI Linux.

**Trade-off documenté** : `lusr_component.*` breakdown retourne placeholder 0
tant que la table `lusr_component_history` n'existe pas (V2 follow-up).

### Commit 5 — ImprovementCampaign full stack

Migration `improvement_campaign` + ALTER `challenge` ADD `campaign_id`. Domain
package `internal/campaign/` (300L service + 101L stats Mann-Whitney U + 96L types).
Service avec Start/Pause/Resume/Close/Abandon/Link/Evaluate via interfaces
`Repo` + `SampleProvider` (mockabilité).

5 raffinements algorithmiques (§4.5.3) :
- R1 : delta lissé LOWESS via `analysis/temporal/lowess.go`
- R2 : phrasing strict (UI commit 9)
- R3 : filtre playlist optionnel (`playlist_group`)
- R4 : Mann-Whitney U fait maison avec approximation CLT normale (n1+n2 ≥ 20),
  gestion ex-aequo, p-value two-tailed → ProgressionConfirmed si p < 0.05
- R5 : auto-suggestion de clôture (plateau 60j, jamais auto-fermeture)

Hook post-sync `EvaluateActive` câblé (idempotent, no-op si pas de campagne).
7 endpoints HTTP `/campaigns/*`. 16 tests unitaires Windows (6 MWU + 10 service
avec fakes).

### Commit 6 — PlayerProfileCard frontend (5 sous-composants)

Types TS dans `lib/playerProfile.ts` miroirs des structs Go. Hook
`usePlayerProfile` (stale 5min) + `useActiveCampaign` (stale 1min) +
`useCampaignMutations` avec invalidation centralisée.

5 sous-composants dans `components/profile/` :
- IdentitySection (A1) : rôles + radar 6 axes ECharts + forces/à renforcer
- StyleDisciplineSection (A2) : 2 cartes FK/FD + Engagement
- PerformanceSection (B) : tier + sub-tier progress + 8 composantes + badge LOWESS
- ProgressionSection (C) : leviers + suggestions + CTA campagne
- InsufficientDataPlaceholder : état < 30 matchs

Intégré en tête de `ParcoursTab` (ObjectifsPage). Couleurs via
`tokenCssVar('outcome-win'|'outcome-loss')` — pas de Tailwind color classes.

### Commit 7 — CampaignTracker sticky + StartCampaignModal

CampaignTracker (280L) sticky au-dessus du profil quand campagne active :
header tri-état (Active confirmé / En cours / Pause / Clôturée / Abandonnée),
bloc tendance (snapshot + actuel + delta signé Unicode), état < 20 matchs,
auto-closure notice (R5 info-only), actions Pause/Resume/Clore/Abandonner
avec `AlertDialog` pour les destructives.

StartCampaignModal (195L) pattern aligné sur AlertDialog (role=dialog, Escape,
clic backdrop). Selector playlist_group (6 options + "all"), phrase R2, option
"Skip — créer juste un défi libre" propagée via callback parent.

`CampaignAndProfileSection` helper dans ObjectifsPage orchestre tracker + profile
+ modale ; `onStartCampaign` propagé au profile **uniquement si pas de campagne
active** (plan §4.5.1 — 1 max active à la fois).

### Commit 8 — Rename Objectifs → Ascension + Mon parcours → Parcours

Label nav L1 + tab sous-onglet + ObjectifsPage h1 + page title. Le route path
`/players/$slug/objectifs` est **conservé** pour compat (pas de bookmark cassé).
Tests `NavL1.test.tsx` réécrits sur "Ascension" (6/6 pass). Ajustement
utilisateur in-sprint : "Mon parcours" raccourci en "Parcours".

### Commit 9 — i18n manifest profil + campagne FR/EN

Manifest unifié `profile.toml` (125 clés) pour profil + campagne. Hook
`useProfileI18n` (15L) — sucre `t(key, vars?)`. Plurals + selects ICU pour
compteurs et booleans. Clés dynamiques via cast `ProfileManifestKey` pour
rôles/axes/composantes/tiers.

8 composants refactorés — 0 warning eslint hardcoded-strings sur les fichiers
profil/campagne (50 résolus). Bonus : `coaching_tips.ts` (généré, jamais commité)
rattrapé.

### Commit 10 — Cet ADR

Mise à jour de l'ADR de "Partially implemented (3/10)" à **"Accepted (10/10)"**.

## Key architectural choices

### Self-benchmark (pas inter-joueurs)

Le pool de joueurs LevelUp est trop petit pour produire un benchmark inter-joueurs
fiable. Décision §2 du plan : afficher le **top 20% personnel** par composante
+ l'inversion mathématique LUSR pour la cible théorique. Évite de comparer un
joueur à des distributions tirées de 10-50 utilisateurs.

### Renaming "Faiblesses" → "Axes d'amélioration"

UX positif. Cohérent avec la philosophie V2 (feedback positif uniquement).
Manifest commit 9 : `profile.insights.improvements` ("À renforcer").

### Campagne d'amélioration vs Saisons (DEPRECATED)

Différences fondamentales (cf. §4.5.2 du plan) :
- Activation **volontaire**, pas calendaire
- Durée **indéterminée**, fermable à tout moment
- **1 axe ciblé** (pas multi-arcs)
- **Pas de pénalité régression**
- **Pas de reward de fin** — la progression visible suffit

Esprit Duolingo : trajectoire personnelle suivie volontairement, pas de note finale.

### 5 raffinements algorithmiques (livrés en commit 5)

R1. Delta lissé LOWESS (pas delta brut, variance ±0.05 sur 30 matchs)
R2. Phrasing no-causalité ("On t'aide à voir ta trajectoire — pas à la garantir.")
R3. Filtre playlist optionnel
R4. Mann-Whitney U fait maison + approximation CLT pour confirmation (p < 0.05)
R5. Auto-suggestion de clôture, jamais auto-fermeture

### Coexistence avec le mode libre

La campagne est **opt-in**, jamais imposée. Le flow Prestige libre (mode `libre`
du CreateChallenge) reste 100% disponible. Un défi sans `campaign_id` (NULL)
existe à part, juste "pour le fun". Le bouton "Skip — créer juste un défi libre"
de la modale matérialise ce choix.

### Route path /objectifs conservé

Décision pragmatique commit 8 : seul le label nav change ("Ascension"), pas le
route path `/players/$slug/objectifs`. Bookmark-friendly, zero refactor
routing/redirection. Le rename de la route est un commit isolé hors V1
(si l'équipe le souhaite plus tard).

## Consequences

### Positives (livrées)

- **Page profil V1 complète** : 5 sections (A1/A2/B/C + insufficient) consomment
  GET /profile et exposent radar 6 axes + rôles + style FK/FD + engagement +
  tier LUSR + 8 composantes + leviers + suggestions.
- **Boucle Campagne complète** : start (avec snapshot 100 matchs) → tracker
  sticky → pause/resume/close/abandon → auto-suggestion R5. 7 endpoints + UI
  + hook post-sync câblés.
- **i18n FR/EN** intégrale sur profil + campagne (125 clés, plurals ICU).
- **Coach V2 alertes campagne câblables** : `AlertTypeCampaignProgress` et
  `AlertTypeCampaignCloseAuto` (V2 commit 5) ont maintenant leur backend.
  L'orchestrateur post-sync les déclenchera naturellement quand une campagne
  active passe le seuil de confirmation MWU ou se met en plateau 60j.
- **Catalogue Prestige enrichi et tagué** : 32 templates, 6 arcs, couverture
  LUSR complète sur les 8 composantes.
- **Aucune régression V2** : le caller `post_sync_progression.go:Load` du
  mini-service reste API-compatible — le service V1 augmente sans casser.

### Trade-offs et follow-ups V2

Documentés dans les thought_logs commit par commit :

- **LUSR component history non matérialisée** : `loadLUSRComponentsBreakdown`
  retourne map vide → UI affiche current=0/top20=0/target avec tooltip
  "données indisponibles". À V2 : créer table `lusr_component_history(match_id,
  component_name, value)` alimentée par le scoring engine.
- **Mapping `personal_score_awards.award_name → narrative.ParticipationAxis`** :
  V1 utilise heuristique agrégats `match_participants` (kills/deaths/assists/
  score). Affinement title-specific en V2 pour Objective + Impact plus
  précis.
- **R5 "axe sort du bottom-3 du radar"** : V1 plateau-only. Le PlayerProfile
  n'est pas consulté par le hook campagne — ajout en V2 follow-up.
- **`lusr_component.*` axes Campaign** : SampleProvider retourne placeholder 0.
  UI limite le sélecteur aux axes radar V1. Câblé V2 avec la table history.
- **Template ID brut dans suggestions** : `ProgressionSection` affiche
  "halo_infinite.daily.kda_session" tel quel. V2 : exposer `label_fr`/`label_en`
  via `SuggestedChallenge` ou ajouter au manifest profile.toml.
- **Streaks perf-based toujours non câblés** (V2 dette identique) :
  nécessitent les médianes personnelles que `computeLUSRComponents` exposerait
  quand la table history sera créée.

### Rename de route /objectifs → /ascension

Hors scope V1. Commit séparé si l'équipe veut l'aligner avec le label nav.
Implique : redirection backward, mise à jour `routeTree.gen.ts`, refactor
`pageTitle.ts`, mise à jour ~20 références dans le code (HomePrestigeSection,
notifications navigation, classifyFeedback, tests). Aucune urgence — le
mapping label/route est déjà documenté in-code par les commentaires commit 8.

## References

- Plan retenu : `.ai/PLAN_PLAYER_PROFILE_ASCENSION.md` (705 lignes, 10 commits cadrés)
- ADR 0004 : narrative engine (radar 6 axes + 8 rôles — consommé par
  `aggregateNarrative` commit 4)
- ADR 0014 : Progression Tracking V2 (Streaks/Records/Milestones/Coach) qui
  attendait ce V1 pour ses 2 alertes campagne et ses 2 streaks perf-based
- Plan rejeté : `.ai/PLAN_SEASONS_ASCENSION.md` (DEPRECATED, justification du
  choix Campagne vs Saison dans §4.5.2 du plan V1)
- thought_log entries `[2026-05-18] feat(player-profile-v1)(commit-{1..10})`
