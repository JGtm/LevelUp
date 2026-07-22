# PLAN — Refonte UX page Ascension (2026-07)

> Rédigé le 2026-07-07 après revue complète (code + revue visuelle en conditions réelles
> sur JGtm, serveur dev local). Exécution sous contrat du skill `plan-execution`.
> Diagnostic source : conversation du 2026-07-07 (revue UX Ascension).

## Objectif et critère de succès

La page Ascension doit être crédible (zéro identifiant technique ni valeur aberrante à
l'écran), navigable (onglets nommés selon leur contenu, deep-links cohérents, sorties
vers les matchs), et donner une raison de revenir (historique du passé + tendance
visuelle). Critère de succès global : parcours des 3 onglets sur les 4 joueurs actifs
sans aucun GUID/clé brute/valeur absurde visible, mode pilote fonctionnel, section
Historique peuplée, 1 graphe de tendance + 1 calendrier d'activité rendus.

**Branche cible** : `refactor/ascension-ux-2026-07`, créée depuis `main` APRÈS le merge
du chantier audits (`refactor/audits-2026-07`). Dépendance : plan de merge 8 étapes du
chantier audits d'abord. Rappel : push sur `main` = déploiement prod automatique.

## Décisions produit (TRANCHÉES — modifier ici avant exécution si désaccord)

- **DEC-1 Mode pilote** : le backend est COMPLET (routes `POST
  .../pilot-mode/enable|disable`, quotas 3 daily / 5 weekly / 2 monthly,
  `internal/prestige/service_pilot_pool.go`). Le tooltip front « non implémenté côté
  backend » est une doc inversée. Décision : câbler le toggle (fonctionnel), défaut
  **OFF**, et promouvoir l'activation dans l'empty state des objectifs. Rationale :
  l'attribution auto crée des défis au palier Heroic d'office — l'imposer par défaut à
  un joueur qui n'a rien demandé brouille la frontière « mes objectifs » vs « le
  système », et gonfle les stats globales (X créés / 0 complétés). On réévalue le
  défaut ON après observation de la qualité d'attribution.
- **DEC-2 Coach proactif** : `coach_proactive_mode` passe à défaut **ON** (suggestions
  pures, dédup 24h existante, aucun write non sollicité). C'est lui qui répond au
  besoin « que des suggestions, autant l'activer ». Le hint « Active le coach
  proactif… » disparaît du premier contact.
- **DEC-3 Onglets** (validé utilisateur 2026-07-07) : restructuration en **4 onglets**,
  ordre = parcours utilisateur (qui je suis → ce que je vise → comment progresser → ce
  que j'ai accompli) :
  1. « Profil » — INDEX `/ascension` : identité/radar, style & engagement, performance
     (+ sparkline D2), patterns contextuels, solo vs escouade, comportements.
  2. « Objectifs » — NOUVELLE route `/ascension/objectifs` : couche Prestige inchangée
     (mode pilote opt-in, objectifs, arcs).
  3. « Entraînement » — route `/ascension/coaching` inchangée : cap du moment,
     suggestions coach, campagne active, leviers calibrés + pistes de progression
     (ProgressionSection extraite de PlayerProfileV3, avec StartCampaignModal).
  4. « Réalisations » — `/ascension/realisations` : surface du PASSÉ (Lot C).
  Recomposition pure : aucun composant feuille réécrit, aucun endpoint touché. Seule
  découpe réelle : ProgressionSection sort de PlayerProfileV3 (sections déjà en
  fichiers séparés sous `features/ascension/profile/`).
- **DEC-4 Vocabulaire FR** : « milestones » → « Jalons », « moment cards » → « cartes
  moments » (règle n°1 CLAUDE.md, pas d'anglicismes).
- **DEC-5 Graphes** : oui, mais minimal et motivationnel — (a) sparkline de tendance de
  skill (μ lissé LOWESS, affiché en points LUSR) 90 j dans la section Performance, (b)
  calendrier d'activité type heatmap 90 j adossé aux séries. PAS de dashboard : les
  pages Solo font déjà l'analytique lourde.
- **DEC-6 μ/σ bruts** : remplacés par l'affichage LUSR (métrique connue de
  l'utilisateur) ; μ/σ retirés de la surface (feedback mémorisé : montrer LUSR/CSR,
  jamais μ TrueSkill).
- **DEC-7 Records corrompus** : purge one-off des valeurs hors bornes (précision
  7333 %, `best_kda` 107) + bornes de vraisemblance par métrique dans le detector
  (INSERT refusé + `slog.WarnContext`) + filtre à la lecture (métrique inconnue du
  catalogue = non servie + log).
- **DEC-8 Petits échantillons patterns** : en dessous de 10 matchs par groupe, pas de
  badge Force/Faiblesse (badge « Échantillon faible » neutre à la place). Le seuil
  backend `MinMatchesPerGroup` reste, c'est un habillage du signal.
- **DEC-9 Barre Prestige** : la ligne du joueur affiche « PP dans le niveau courant /
  PP du prochain niveau » (progression intra-niveau), plus le total PP en second. Les
  amis sans données (0 PP partout) : conservés seulement s'ils ont ≥ 1 PP, sinon omis.

## Lot A — Crédibilité des données (backend + front) [prioritaire]

Gate d'entrée : brancher sur la branche cible, relire ce plan.

- [x] **A1 — Labels de cartes dans les patterns** : `analyzeByMap` groupe par `MapID`
  brut et le handler `internal/api/handlers/patterns.go` le sert tel quel. Ajouter un
  champ `label` au DTO `ContextualPattern` résolu au niveau handler/service via le
  référentiel metadata du titre (même source que les autres pages : asset_translations
  / registre, title-agnostic via le registry — PAS de SQL inline dans le handler,
  passer par le reader/port existant). Fallback si non résolu : label = « Carte
  inconnue » + id court, jamais le GUID nu. Test handler : chaque pattern `by_map`
  servi a un `label` non vide.
- [x] **A2 — Clés `with_friends`/`solo` et texte des leviers** : le front mappe les clés
  `by_squad` sur `squadVsSoloSolo`/`squadVsSoloSquad` (i18n existants) au lieu
  d'afficher la clé ; le texte du levier `map_avoidance` utilise le `label` de A1
  (« Améliore ton win rate sur Aquarius », plus jamais un GUID dans une phrase).
- [x] **A3 — Clés de métriques Prestige** : `CreateChallengeForm` stocke `FieldKDA`,
  `StatsGlobales` l'affiche brut. Créer un helper front unique `metricLabel(key,
  locale)` (réutilisant `METRIC_LABEL_*` d'ascension + mapping des clés `Field*`
  Prestige) et migrer TOUS les affichages de clé métrique de `features/prestige` +
  `features/ascension` dessus. Garde-rail (règle n°6) : test vitest du helper —
  clé inconnue → libellé humanisé, jamais la clé brute ; grep-test interdisant
  `\bField[A-Z][a-zA-Z]+\b` dans le JSX rendu des deux features.
- [x] **A4 — Bornes de vraisemblance records** : dans le detector records
  (`internal/progression/`), bornes par métrique (accuracy ∈ [0,1] côté stockage ADR
  0006, kda ∈ [-20, 50], kpm ∈ [0, 20], etc. — les fixer en constantes nommées,
  pas de magic numbers). Hors bornes : pas de persist + `slog.WarnContext`. À la
  lecture (endpoint `/records`), les métriques absentes du catalogue de labels (ex
  `best_kda` legacy) ne sont pas servies + log. Tests unitaires analysis purs.
- [x] **A5 — Purge one-off des records corrompus** : commande diag/ops (pattern one-off
  container existant) qui liste puis supprime les lignes `player_records` /
  `record_history` hors bornes A4 pour les 4 joueurs. DuckDB : écritures via le repo
  existant (jamais de bare connect), règles ART respectées (tables concernées non
  append-only critiques — vérifier sur pièces avant, sinon recette ADR 0026).
- [x] **A6 — Dates de jalons backfillées** : toutes les `earned_at` = date du seed
  (30/05/2026). Recalculer la vraie date de franchissement par jalon (cumul des
  métriques sur `match_registry` jusqu'au seuil — dérivable et déterministe) via une
  commande de backfill one-off ; si non dérivable pour un jalon, `earned_at` NULL et
  le front n'affiche pas de date (jamais une date fausse).
- [x] **A7 — Levier sans valeur courante** : « Actuel — → Cible 37 % » ; si
  `current` absent, masquer la ligne Actuel/Cible et ne garder que l'horizon. Fix
  front `LeverList.tsx`.
- [x] **A8 — Unités des séries hebdomadaires** : `streakBestLength` code « jour » en dur
  (i18n.ts:222) → « Record perso : 2 jours » sur une série hebdo. Paramétrer l'unité
  (jour/semaine) selon le type, FR + EN.
- [x] **A9 — Condition de jalon lisible** : « accuracy >= 0.50 sur 90 jours distincts »
  affiché brut. Le catalogue TOML (`config/titles/{slug}/milestones/catalog.toml`)
  porte déjà des libellés FR/EN par jalon : ajouter un champ condition FR/EN et le
  servir ; le front n'affiche plus jamais l'expression technique.

**Gate Lot A** :
```
cd apps/go-api && go test ./internal/analysis/patterns/... ./internal/progression/... ./internal/api/handlers/...
make check-types && make test-web
# revue visuelle locale : 3 onglets x JGtm — zéro GUID, zéro Field*, zéro % > 100 en précision
```

## Lot B — Navigation, IA, mode pilote

- [x] **B1 — Restructuration 4 onglets** (DEC-3) — périmètre fermé :
  - [x] route `objectifs.tsx` créée ; `index.tsx` rend le nouvel onglet Profil
    (routing file-based, routeTree.gen.ts régénéré via `@tanstack/router-generator`)
  - [x] `AscensionProfileTab` renommé `AscensionObjectivesTab` (contenu Prestige
    inchangé) ; nouveau `AscensionProfilTab` composant : PlayerProfileV3 (identité /
    style / performance) + patterns contextuels + SquadVsSoloCard + BehaviorAlertList
  - [x] `ProgressionSection` extraite de PlayerProfileV3 vers `AscensionCoachingTab`
    (recomposition via wrapper `usePlayerProfile` ; PlayerProfileV3 perd les props
    onStartCampaign/onLaunchTemplate — code mort supprimé) ; `StartCampaignModal`
    déjà dans le coaching tab ; leviers calibrés (LeverList) déplacés dans Entraînement
  - [x] `AscensionLayout` : 4 liens d'onglets, i18n FR/EN (tabProfile/tabObjectives/
    tabCoaching/tabRealisations + aria), sous-titres LayerSection relus (profilLayer*)
  - [x] deep-links mis à jour : `navigation.ts`, `lib/pageTitle.ts` (+objectifs/coaching),
    `navL1Sections.tsx` (4 onglets, `tab_profile`+`tab_objectives`), common.toml/
    generated.ts régénérés. `classifyFeedback.ts` : regex `(objectifs|ascension)` couvre
    déjà tous les sous-onglets → aucun changement (`[~]`). NavL1MobileMenu consomme
    NAV_L1_SECTIONS → couvert automatiquement.
  - [x] tests : AscensionObjectivesTab.test.tsx (renommé), AscensionCoachingTab.test.tsx
    adapté, nouveau AscensionProfilTab.test.tsx (PlayerProfileV3 mocké), PlayerProfileV3
    .test.tsx (progression/μσ retirés), NavL1.test.tsx (4 onglets)
- [x] **B2 — Deep-link du widget home + AM-5** : `HomeAscensionWidget` → `/ascension/
  realisations`. Notifs `objective_completed`/`challenge_completed` → `/ascension/
  realisations` avec `selectedObjectiveId`/`selectedChallengeId` ; ancrage/surlignage +
  scroll de la carte moment côté Réalisations (`useSearch`). `objective_assigned`/
  `challenge_added` → `/ascension/objectifs`. navigation.test.ts couvre les 4 cas.
- [x] **B3 — Câblage mode pilote** (DEC-1) : `usePilotMode` (mutations enable/disable),
  `prestigeApi.enable/disablePilotMode`, key `queryKeys.prestige.pilotMode`, toggle réel
  (état dérivé des défis `mode === 'pilote'` actifs), tooltip « non implémenté » retiré,
  i18n FR/EN, invalidation `challenge.list` + `prestige.meAll`, CTA dans l'empty state.
  **AM-4** : `DisablePilotMode` complété côté Go (archive les défis pilote actifs →
  statut `archived`, slog + tests).
- [x] **B4 — Coach proactif défaut ON** (DEC-2) : défaut `CoachProactiveMode` = true
  (`store.go` defaultSettings + applyAbsentDefaults, domain, wire ; tests inversés +
  « FalseRespected »). Mentions du défaut mises à jour (grep) : ADR 0020 (3 endroits),
  openapi.yaml comment, `AscensionCoachingTab` + `AnalyseTab` fallbacks `?? true`.
- [x] **B5 + AM-11 — Confirmations cohérentes** : `confirm()` natif → `AlertDialog`
  destructive (par carte, composant `ObjectiveCard`) ; bouton « Abandonner » i18n FR/EN.
- [x] **B6 + AM-10 — Abréviations et cibles** : composant partagé `CombatAbbr` (tooltip
  OC/DR) dans PatternContextGrid + SquadVsSoloCard ; `target_for_tier` vérifié sur pièces
  = composite global du palier suivant (par design, pas un bug) → étiqueté via note
  `profile.performance.target_note`.
- [x] **B7 — Dédoublonnage Solo/Escouade** : `by_squad` retiré de `CONTEXT_ORDER` (grille
  mode/carte) ; branche `contextLabel` by_squad supprimée (code mort ; aucun test dédié).
- [x] **B8 — LUSR au lieu de μ/σ** (DEC-6) : PerformanceSection affiche « {mu} pts LUSR »
  (clé `lusr_points`) ; `mu_sigma` supprimée (manifest + rendu) ; écart au palier déjà
  en points via `gap_to_next`. DTO déjà suffisant (aucun champ Go ajouté).
- [x] **B9 — Vocabulaire FR** (DEC-4) : « Mes jalons », « Aucun jalon », « cartes moments »,
  « Taux de victoire » (ex-« Win rate »). EN inchangé.
- [x] **B10 — Barre Prestige lisible** (DEC-9) : progression intra-niveau
  (`{inLevel} / {span} PP vers {next}`), total PP secondaire, amis à 0 PP omis. DTO déjà
  suffisant (`threshold_pp`/`next_threshold_pp` servis).
- [x] **B11 — Badge petits échantillons** (DEC-8) : seuil `MinMatchesForSignal` (10)
  promu const backend + servi via `PatternReport.min_matches_for_signal` (openapi +
  generate-types + drift OK) ; front affiche « Échantillon faible » (neutre) sous le
  seuil, sans 10 en dur.
- [x] **B12 — Pill de statut des séries lisible** (AM-6) : « Cassée » → « Interrompue »
  (FR ; EN « Broken » inchangé) + tooltip `streakBrokenTooltip` (date + reset
  multiplicateur PP), FR/EN (StreakCard).

**Gate Lot B** :
```
cd apps/go-api && go test ./internal/platform/settings/... ./internal/api/handlers/...
make check-types && make test-web && make go-api-lint
# revue visuelle : 4 onglets rendus (Profil = index, patterns dans Profil, pistes de progression dans Entraînement) ; toggle pilote aller-retour (enable → défis pilotes visibles → disable) ; coach ON par défaut sur profil vierge
```

## Lot C — Historique (actif vs passé) [demande produit centrale]

Principe : l'onglet Objectifs ne montre QUE l'actif ; l'onglet Réalisations devient la
mémoire complète (séries, records, jalons, objectifs/arcs/campagnes passés).

- [x] **C1 — Inventaire backend sur pièces** : vérifier ce que les endpoints servent
  déjà — statuts challenge existants : `draft/active/completed/expired/abandoned/
  archived` (enums.go) ; l'endpoint liste accepte-t-il un filtre statut ? Les
  campagnes closes sont-elles listables (pas seulement l'active) ? Arcs terminés ?
  Sortie de C1 : liste écrite des trous d'API (potentiellement zéro — ne rien
  réimplémenter qui existe, skill `go-features`). **Sortie → journal Lot C.**
- [x] **C2 — Compléments d'API si trous** : au besoin, paramètre `status` sur la liste
  challenges et endpoint campagnes passées. Handlers minces, DTOs dédiés,
  openapi.yaml + `make generate-types` (le test contract force la doc). **Voir journal.**
- [x] **C3 — Section « Historique » de l'onglet Réalisations** : sous les jalons,
  3 blocs chronologiques — objectifs passés (terminés / abandonnés / expirés, avec
  résultat), arcs terminés, campagnes closes (axe, delta snapshot→final, playlist).
  Les cartes moments existantes restent le bloc « célébration » ; l'historique est la
  liste exhaustive datée. Query keys centralisées, i18n FR/EN. **Voir journal C3.**
- [x] **C4 — Nettoyage de l'actif** : onglet Objectifs = objectifs actifs + arcs en
  cours uniquement (plus de complétés dans les groupes) ; séries cassées : visibles
  dans Réalisations avec leur date, mais l'accueil (widget) ne montre que
  actives/protégées (déjà le cas). **Voir journal C4** (défis `[~]`, séries `[~]`).
- [x] **C5 — Sorties vers les matchs** : cartes patterns cliquables → page Solo avec le
  filtre correspondant (le paramètre d'URL `f=` des pages stats existe déjà — réutiliser
  l'encodage existant, pas de format maison) : pattern by_mode → filtre mode, by_map →
  filtre carte. Pour les records : lien « voir la période » → page Solo bornée sur la
  fenêtre du record. Aucun cul-de-sac restant sur ces deux familles de cartes.
  **Voir journal C5.**

**Gate Lot C** :
```
cd apps/go-api && go test ./... (paquets touchés) ; make generate-types (si API)
make check-types && make test-web
# revue visuelle : onglet Objectifs sans items passés ; Réalisations montre l'objectif FDA abandonné/terminé dans l'historique ; clic pattern Super Fiesta → page Solo filtrée
```

## Lot D — Tendance visuelle (graphes minimaux, DEC-5)

- [x] **D1 — Vérifier l'existant timeseries** : `internal/api/handlers/timeseries.go` +
  SPEC_ECHARTS_TIMESERIES (13 graphes) — si une série skill/LUSR par match existe,
  la réutiliser telle quelle (lecteurs : vues `_latest` uniquement, règle ART n°2).
  Sortie : décision écrite réutilisation vs nouvel endpoint (défaut : réutilisation).
  **Décision tranchée — voir « Journal d'exécution — Lot D » ci-dessous.**
- [x] **D2 — Sparkline Performance** : mini-graphe 90 j (μ lissé LOWESS affiché en
  points LUSR) dans la section Performance, wrapper ECharts existant
  (`components/charts/`), tokens sémantiques, mock echarts-for-react dans les tests
  jsdom (piège mémorisé). **Voir journal D2.**
- [x] **D3 — Calendrier d'activité** : heatmap 90 j des jours joués (données : counts
  par jour — vérifier l'existant côté home/engagement avant tout nouvel endpoint),
  rattachée au bloc séries de Réalisations. Timezone : fragment SQL canonique
  `COALESCE(start_time_utc, ...)` obligatoire. **Voir journal D3.**

**Gate Lot D** :
```
cd apps/go-api && go test ./... (paquets touchés)
make check-types && make test-web
# revue visuelle : sparkline rendue avec ≥ 30 matchs ; heatmap cohérente avec les cartes séries (un jour joué = case remplie)
```

## Contrat d'exécution

- Skill `plan-execution` : ordre strict A → B → C → D, un lot clos (gate passé +
  thought_log) avant le suivant. Aucun report d'item exécutable ; statuts `[x]`/`[~]`
  (référence)/`[!]` (justification écrite) obligatoires à la clôture de chaque lot.
- Zéro fix opportuniste hors périmètre : consigner ici, section Découvertes.
- Reprise de session : lire ce fichier (statuts) + dernière entrée thought_log du
  chantier ; reprendre au premier item non coché du lot courant.
- Avant tout commit : entrée thought_log ; avant livraison finale : skill
  `delivery-checklist` + `go test -tags=integration` si persist/sync touchés (A4/A5/A6
  touchent persist → gate intégration OBLIGATOIRE, sérialisé `-p 1` + filtre ancré).
- Effort estimé : Lot A moyen (A1/A4/A6 = le gros), Lot B moyen (B1 restructuration
  4 onglets ~0,5-1 j + B3 câblage pilote), Lot C moyen, Lot D moyen (si réutilisation
  timeseries : rapide).

## Addendum 2026-07-22 — revue pré-exécution (vérification sur pièces, grille plan-review)

Dépendance levée : campagne audits mergée sur main (`28146aa3a`). Branche cible créée
depuis `main` à jour.

**Décision de chantier — croisement avec `.ai/PLAN_CROSS_TITLE_ARCS_2026-07.md`** : les
deux plans sont séparables proprement (vérifié : chevauchement limité à 2 fichiers Go sur
des déclarations disjointes — arcs vs challenges ; `arc_titles`/`ArcTitlesRepo`/
`creditTitlesFor` sans aucun consommateur prod ni surface HTTP/front). Exécution NON
combinée : ce plan d'abord ; le plan arcs reste sur sa branche dédiée
`feat/arcs-per-title-strict`, déclencheur inchangé (2e titre). Contrainte reprise ici :
le Lot C n'introduit AUCUN nouveau lecteur de `arc_titles`/`ArcsByTitle` — partition
en cours/terminés via `Arc.CompletedAt` uniquement.

**Feedback utilisateur consigné (collègue, 2026-07-22)** — 3 constats, tous dans le
périmètre : (1) aucun historique Objectifs/Arcs dans l'UI → Lot C (confirmé sur pièces :
seuls les défis `completed` apparaissent, en cartes moments) ; (2) notifs « objectif
complété » renvoient vers une page où l'objectif concerné n'est pas visible → AM-5 ;
(3) pill « Cassée » des cartes séries incomprise → AM-6.

**Amendements d'items (constats sur pièces — le code a bougé depuis le 07/07)** :

- **AM-1 (B1/B2, chemins)** : les routes vivent sous
  `src/routes/players/$playerSlug/ascension/` (pas `src/routes/ascension/`). Tous les
  greps deep-links sous ce préfixe. Le redirect legacy
  `players/$playerSlug/objectifs/index.tsx` (→ `/ascension`) doit pointer vers
  `/ascension/objectifs` après B1.
- **AM-2 (B1, périmètre réel)** : état actuel = 3 onglets ; PlayerProfileV3 + patterns
  sont rendus dans l'onglet Entraînement (`AscensionCoachingTab.tsx:70-74`), pas dans
  l'index. B1 = déplacer PlayerProfileV3 + PatternsSection vers le nouvel onglet Profil,
  extraire ProgressionSection vers Entraînement, renommer AscensionProfileTab →
  AscensionObjectivesTab. StartCampaignModal est DÉJÀ dans AscensionCoachingTab (rien à
  déplacer). Recomposition sans réécriture de feuilles : confirmée, mais plus large que
  la formulation initiale.
- **AM-3 (A5, append-only)** : hypothèse du plan OBSOLÈTE — depuis le 2026-05-30 les
  records sont append-only (`player_records_history` + vue `player_records_latest`,
  `records_repo.go` ; `record_history` append-only aussi). Purge = recette ADR 0026
  (supersession/rebuild), JAMAIS de DELETE direct. Vérifier sur pièces à l'exécution.
- **AM-4 (B3, disable)** : `DisablePilotMode` est un no-op sans état persistant
  (`service_pilot_pool.go:80-89`). B3 inclut : compléter le backend — disable archive
  les défis pilotes actifs non complétés (+ log) ; état du toggle dérivé des défis
  `mode === 'pilote'` actifs (comme prévu).
- **AM-5 (B2 étendu, feedback collègue)** : `objective_completed` et
  `challenge_completed` (features/notifications/navigation.ts:46-58) → cible
  `/ascension/realisations` avec ancrage/surlignage de l'item concerné (réutiliser le
  paramètre `selectedObjectiveId` existant, support à ajouter côté Réalisations).
  Aucun mapping vers realisations n'existe aujourd'hui.
- **AM-6 (nouveau B12, feedback collègue)** : pill de statut « Cassée » des cartes
  séries (StreakCard.tsx:39,82-85, gris neutre, sans explication) → libellé
  « Interrompue » + tooltip explicatif (série interrompue le {date}, multiplicateur PP
  réinitialisé), FR/EN, cohérent DEC-4. La date `broken_at` existe déjà côté back et
  front.
- **AM-7 (C1 PRÉ-REMPLI — inventaire fait en revue, C1 devient une vérification
  rapide)** : trous d'API = (1) filtre `status` non exposé sur `GET
  /prestige/challenges` (le repo est prêt : `ChallengeFilter.Status`,
  `prestige_player_helpers.go:83-86` ; le service force `active`,
  `service.go:576-581`) ; (2) AUCUNE liste de campagnes closes (repo campaign sans
  méthode List — nouvelle méthode + SQL + endpoint). NON-trou : arcs terminés déjà
  servis par `GET /arcs` (pas de filtre `completed_at`, le front partitionne sur
  `CompletedAt`).
- **AM-8 (D1)** : aucune série skill/LUSR réutilisable — seul le champ per-match
  `SkillRatingValue` existe (`domain/timeseries.go:243-248` ;
  `timeseries_service.go:53-56` note que MetricSeries ne couvre pas lusr). Décision D1
  orientée « construire la série depuis l'existant », à trancher sur pièces.
- **AM-9 (A8)** : le helper `streakUnit()` existe déjà (StreakCard.tsx:26-28) mais
  n'est pas appliqué à `streakBestLength` (i18n.ts:222 confirmé) — fix plus petit que
  prévu.
- **AM-10 (B6)** : la cible uniforme « 98 % » vient du champ API `target_for_tier`
  (service profile) — la vérification est côté backend, le front est un simple rendu.
- **AM-11 (B5)** : libellé « Abandonner » codé en dur FR
  (AscensionProfileTab.tsx:171) → basculer en i18n dans B5.

## Découvertes en cours d'exécution (à consigner, ne pas traiter)

- **(A2) Phrases de leviers en dur FR côté Go** — **RÉSOLU 2026-07-22 (F3)** :
  `internal/analysis/patterns/levers.go` codait les libellés de leviers en français
  dur. Refonte pérenne : le backend ne sert PLUS de phrase — le champ `label` du DTO
  `Lever` (et `source_pattern`, devenu code mort) sont supprimés ; le levier porte
  désormais des données structurées (`axis`, `context_key`, `context_label` résolu
  title-agnostic pour by_map). Le FRONT compose la phrase via des gabarits i18n FR/EN
  par axe (`features/ascension/i18n.ts:leverPhrase`, `LeverList.tsx:leverPhrase`).
  `rewriteMapLeverLabels` (handler) supprimé, remplacé par `setMapLeverContextLabels`
  (résout ContextLabel via `ResolveMapLabels`, même mécanisme A1). Tests : Go (DTO
  structuré servi, ContextLabel résolu sans GUID) + vitest (composition FR/EN par
  axe, GUID jamais rendu). openapi+generated régénérés (drift MISSING=0).
- **(A6) `accuracy_threshold_days` lit `start_time` brut** — **RÉSOLU 2026-07-22 (F2)** :
  `loadPlayerStats` (`post_sync_progression_queries.go`) utilise désormais
  `CAST(analysis.SQLStartTimeCanonical("mr") AS DATE)` (fragment timezone canonique,
  règle n°8) au lieu du `CAST(mr.start_time AS DATE)` brut — cohérent avec le backfill
  A6. Ratchet étendu : `TestNoNewRawStartTimeLiteral` (archlint) interdit désormais
  aussi `CAST(<alias>.start_time AS DATE)` brut (regex `rawCastStartTimeDateRE`,
  allowlist VIDE — précise, ne matche pas la forme canonique COALESCE). Autres
  `start_time` bruts du fichier (SELECT PlayedAt, ORDER BY allowlistés) hors périmètre
  F2 (chargement de matchs, concern distinct) — consignés, non traités.
- **(A5) Index `idx_rec_hist_achieved_desc` divergent** — **RÉSOLU 2026-07-22 (F4)** :
  définition canonique unique `(user_id, achieved_at DESC)` centralisée en constante
  `canonicalRecHistAchievedIndexDDL` (`steps_player.go`) ; la migration de dédup
  alignée dessus ; nouveau step correctif idempotent
  `repair_rec_hist_achieved_index_canonical_v1` (DROP + CREATE canonique, garde
  TableExists) ajouté à `playerSteps()` + `canonicalOrder`. Garde-rail
  `idx_rec_hist_canonical_test.go` : toute création de cet index DOIT être
  `(user_id, achieved_at DESC)` (scan sources non-test). Tests migration (intégration :
  répare un index divergent, idempotent, no-op sans table).
- **(A5/A6) LUSR/ratings hors périmètre** : les records purgés/backfillés ne
  concernent que PB/jalons ; aucun impact sur LUSR/CSR.
- **(B1) e2e `ascension-2tabs.spec.ts` obsolète** — **RÉSOLU 2026-07-22 (F6)** :
  fichier SUPPRIMÉ (référençait un layout mort 2/3 onglets). Le helper
  `skipObsoleteSpec` conservé (5 autres consommateurs). README e2e mis à jour (retrait
  de la référence). La réécriture e2e 4 onglets reste un CHANTIER DÉDIÉ (non fait ici).
- **(B3) télémétrie transition `archived`** — **RÉSOLU 2026-07-22 (F5)** :
  `DisablePilotMode` émet désormais `EmitTransition(ctx, c, TelemetryArchived)` par
  défi pilote archivé (nouvelle constante `TelemetryArchived = "archived"`, distincte
  d'`abandoned` — retrait système vs abandon volontaire). Test :
  `TestService_DisablePilotMode_EmitsArchivedTelemetry` (2 défis → 2 events `archived`).
- **(A2 rappel, B6) phrases de leviers FR en dur côté Go** — **RÉSOLU 2026-07-22 (F3)**
  (cf. bullet A2 ci-dessus : refonte structurée backend + gabarits i18n front).
- **(C4) Décompte des étapes d'arc faux dans l'onglet Objectifs** — **RÉSOLU 2026-07-22 (F1)** :
  `MyArcsSection` fusionne désormais `useChallenges` (actifs) + `useChallengeHistory`
  (terminaux) pour le décompte, via le helper pur `computeArcStepCounts` (dédup par id,
  `completed` = `status === 'completed'`, ignore `arc_id` absent). Test vitest du helper
  (décompte, dédup, défis détachés).
- **(C5) Lien by_map sensible à la locale** — **RÉSOLU 2026-07-22 (F7)** : le backend
  sert désormais sur chaque pattern contextuel un champ `filter_key` = la clé de
  filtrage STABLE que matche le pipeline (`mapUI = MapNameFR ?? MapName`, FR-first
  indépendant de la locale). Résolu via `ResolveMapFilterKeys` (nouvelle méthode port,
  `PreferredLangsForLocale("fr")` FIXE, ≠ `ResolveMapLabels` localisé). by_mode :
  `filter_key = key` (mode normalisé = modeUI ; confirmé sur pièces = identique pour
  Infinite ; Halo 5 by_mode dégénéré indépendamment). Front (`PatternContextGrid`)
  construit le lien sur `filter_key`, `label` reste l'affichage. openapi+generated
  régénérés (drift 0). Tests : Go (filter_key FR-first en requête EN, by_mode==key) +
  vitest (lien encode filter_key, pas le label localisé ; sans filter_key → pas de lien).
- **(C5) Warning jsdom « Not implemented: navigation to another Document »** : apparaît
  au run vitest (liens `<a href>` full-nav). Warning bénin, non bloquant (0 échec) et
  déjà présent ailleurs dans la suite — non traité.
- **(Clôture) HÉRITÉ DE MAIN — 2 tests Relations rouges** :
  `TestRelationsSegmentation_SoloVsSquad_CrossDB` + `_PlaylistFilter_CrossDB`
  (internal/service, tag integration) échouent À L'IDENTIQUE sur main `b21a59772`
  (vérifié en worktree isolé le 2026-07-22) : la fixture VALUES `r` du test n'a pas
  suivi les +4 colonnes Q4 du fix filtres H5 (`fae9921d9`) → Binder Error `r.map_id`.
  RÉSOLU le 2026-07-22 sur demande utilisateur : +4 colonnes ajoutées au DDL
  `match_registry` de la fixture (`relations_segmentation_integration_test.go`) —
  les 2 tests + le package service intégration repassent verts.
- **(Clôture) `make go-api-lint` au scope réduit** — **RÉSOLU 2026-07-22 (F8)** :
  investigation CI (`.github/workflows/ci.yml`) : le lint qui FAIT FOI est le job
  `go-lint` = golangci-lint v2.12.2 sur TOUT `apps/go-api` (config `.golangci.yml`)
  avec ratchet `--new-from-merge-base=origin/main` (dette baseline ~479 gelée invisible).
  Le target local ne faisait qu'un `go vet` domain+analysis (miroir du step vet du job
  `go-test`). Fix : le target reproduit désormais le lint CI (même ratchet) QUAND
  golangci-lint est installé, sinon REPLI documenté sur `go vet` + message renvoyant au
  job CI. Commentaire Makefile daté (scope réduit du repli assumé + critère de retrait).
  golangci-lint NON installé dans cet environnement → repli `go vet` exécuté (propre) ;
  le lint complet reste vérifié en CI.
- **(F2) autres `start_time` bruts dans `post_sync_progression_queries.go`** (NOUVELLE,
  non traitée) : hors du bucket `accuracy_threshold_days` corrigé, le fichier lit encore
  `mr.start_time` brut dans `loadProgressionSharedMatches` (SELECT PlayedAt + `WHERE
  start_time >= ?` + `ORDER BY start_time` allowlisté) et `loadComebackContext`. Concern
  distinct (chargement de matchs → PlayedAt bucketé en Go côté streaks/records), pas le
  bug jour-frontière de F2. À aligner sur le fragment canonique dans un lot dédié si le
  décalage TZ des imports OpenSpartan devient sensible sur ce chemin.
- **(F7) Halo 5 by_mode dégénéré** (NOUVELLE, non traitée) : le pattern engine dérive
  `Mode = NormalizeModeLabel(pair_name_fr ?? pair_name)` (patterns_repo `loadShared`),
  SANS repli `game_variant`. Halo 5 n'a pas de `pair_name` → `Mode = ''` pour tous les
  matchs → grouping by_mode dégénéré (une clé vide). Non aggravé par F7 (filter_key vide
  → pas de lien plutôt qu'un lien cassé). Le correctif pérenne = faire dériver le Mode du
  pattern via `ResolveModeUIWithVariant` (pair-puis-variant) — aligne display + filtre
  pour les deux titres, mais change le grouping/affichage Halo 5 → chantier dédié.

## Journal d'exécution — Lot A (2026-07-22)

Exécution complète A1→A9 sous contrat `plan-execution`, branche
`refactor/ascension-ux-2026-07`. Tous les items `[x]` (faits + vérifiés). Détail :

- **A1** : champ `Label` sur `ContextualPattern` (`json:"label,omitempty"`) ;
  port `PatternsRepository.ResolveMapLabels` + impl DuckDB (réutilise
  `MetadataRepo.ResolveAssetNamesBulk`, langs depuis `ctxkeys.Locale`) ; handler
  `enrichContextLabels` + repli localisé « Carte inconnue (id court) » / « Unknown
  map ». openapi.yaml + generated.ts régénérés. Test handler (label non vide,
  résolu vs repli, locale EN).
- **A2** : substitution GUID→nom dans le texte des leviers `map_avoidance` au
  handler (`rewriteMapLeverLabels`, via SourcePattern `by_map:{id}`) + test ;
  front `PatternContextGrid.contextLabel` (by_map→label, by_squad→i18n Solo/
  Escouade, by_mode→clé) + `label?` sur le type local.
- **A3** : helper unique `lib/i18n/metricLabel.ts` (clés canoniques + alias
  `Field*` + `PRESTIGE_METRIC_OPTIONS` + humanisation) ; migration de TOUS les
  affichages (RecordsTimeline, MilestonesGrid, SquadVsSoloCard, StatsGlobales,
  CreateChallengeForm) ; suppression du `metric` mort d'`ascension/i18n.ts` et des
  5 clés métriques mortes de `prestige/i18n.ts` ; garde-rails : `metricLabel.test.ts`
  (clé inconnue → humanisée) + `features/metric-key-guardrail.test.ts` (aucun
  `\bField[A-Z]` dans le JSX rendu prestige+ascension).
- **A4** : `records/bounds.go` (bornes nommées par métrique + `IsPlausibleValue` +
  `IsKnownMetric`) ; gate detector (hors bornes → pas de persist + `slog.WarnContext`) ;
  filtre read-side handler `/records` (métrique hors catalogue non servie + log).
  Tests purs. AUCUN changement de schéma servi → pas de MAJ openapi (le filtre
  masque des lignes, ne change pas le DTO).
- **A5** : `ops/records_purge.go` (rebuild CTAS filtré transactionnel, recette
  ADR 0026, JAMAIS de DELETE brut) sur `player_records_history` (+ vue) et
  `record_history` ; commande `cmd/purge_corrupt_records` (`--dry-run` défaut,
  `--apply`). Tests DuckDB temp (dry-run sans mutation, apply, retombée _latest sur
  version plausible, intégrité PK/séquence/vue, idempotence). **`--apply` NON exécuté
  (à faire par l'orchestrateur, serveur stoppé).**
- **A6** : moteur PUR `ops.ComputeMilestoneCrossings` (rejoue les matchs, cumul par
  métrique, fragment timezone canonique, constantes OC/DR/accuracy alignées sur le
  détecteur) ; `ops.BackfillMilestoneDates` (UPDATE ART-safe de `earned_at` non
  indexée, NULL si non dérivable) ; `milestone_earned.earned_at` rendue nullable
  (drop NOT NULL idempotent) ; repo `ListByUser` scan nullable + handler ne sert
  pas de date si zéro. Commande `cmd/backfill_milestone_dates`. Tests purs + DuckDB.
  **`--apply` NON exécuté (orchestrateur).**
- **A7** : `LeverList` masque la ligne « Actuel → Cible » si `current_val == 0`.
- **A8** : `streakBestLength` paramétré `{unit}` (jour/semaine via `streakUnit`),
  FR+EN.
- **A9** : `condition_fr`/`condition_en` ajoutés aux jalons du TOML HINF (H5 sans
  condition) ; `CatalogEntry`/`milestoneEntryTOML`/loader ; colonnes
  `milestone_catalog.condition_fr/_en` (CREATE + `AddColumnIfMissing` idempotent) ;
  seed bumpé `seed_milestone_catalog_v2` (re-seed DBs existantes) ; repo Upsert+Read ;
  DTO sert `condition_fr/_en` (plus la formule) ; openapi.yaml + generated.ts ;
  front `MilestonesGrid` choisit selon la locale, rien si absent. Test seed
  (intégration).

Reste à faire par l'orchestrateur : exécuter `--apply` de A5
(`cmd/purge_corrupt_records`) et A6 (`cmd/backfill_milestone_dates`) serveur
stoppé ; revue visuelle 3 onglets × JGtm (zéro GUID / zéro `Field*` / zéro
% > 100 en précision) ; suite intégration anti-ART avant commit (A4/A5/A6).

## Journal d'exécution — Lot B (2026-07-22)

Exécution complète B1→B12 sous contrat `plan-execution`, branche
`refactor/ascension-ux-2026-07` (par-dessus Lot A). Tous les items `[x]` sauf le
sous-point `classifyFeedback.ts` de B1 (`[~]`, regex couvre déjà les sous-onglets).
Gate Lot B PASSÉ intégralement (sorties vertes) :

- `go test ./internal/platform/settings/... ./internal/api/handlers/...
  ./internal/prestige/... ./internal/analysis/patterns/...` → ok
- `tsc -b` (check-types) → ok ; `vitest run` (test-web) → 280 fichiers,
  2459 passés / 14 skippés / 0 échec
- `gofmt -l .` → vide ; `go vet ./...` → clean ; `make go-api-lint` → clean
- Drift OpenAPI (`TestOpenAPISchemaDrift`, CGO) → ok (champ B11 `min_matches_for_signal`
  aligné struct↔yaml)

Faits notables :

- **B1** : restructuration 3→4 onglets. Nouveau `AscensionProfilTab` (index :
  identité + patterns + comportements), `AscensionObjectivesTab` (ex-ProfileTab, couche
  Prestige), `AscensionCoachingTab` (cap + proposals + campagne + `ProgressionSection`
  extraite de PlayerProfileV3 + leviers calibrés). `PlayerProfileV3` allégé (identité/
  style/perf uniquement, props CTA supprimées). routeTree régénéré via
  `@tanstack/router-generator` (jamais édité à la main). i18n `tab_profile`+`tab_objectives`
  (common.toml régénéré).
- **B3 + AM-4** : `DisablePilotMode` backend complété (archive les défis pilote actifs) ;
  toggle front réel dérivé des défis pilote, CTA d'activation en empty state.
- **B4** : défaut `coach_proactive_mode` → TRUE (DEC-2) ; doc anti-inversée mise à jour
  partout (ADR 0020, openapi comment, fallbacks front `?? true`).
- **B8/B10** : aucun champ DTO ajouté (mu/gap_to_next et threshold_pp/next_threshold_pp
  déjà servis). **B11** : seul ajout DTO du lot (`min_matches_for_signal`).

Reste à faire par l'orchestrateur : revue visuelle des 4 onglets × joueurs actifs
(Profil = index ; patterns dans Profil ; pistes de progression + leviers dans
Entraînement) ; aller-retour toggle pilote (enable → défis pilote visibles → disable →
archivés) ; coach ON par défaut sur profil vierge ; ancrage notif « objectif complété »
→ carte moment surlignée dans Réalisations.

## Journal d'exécution — Lot C (2026-07-22)

### C1 — Inventaire backend (vérifié sur pièces, confirme AM-7)

Deux trous d'API confirmés, un non-trou :

- **Trou n°1 — filtre statut non exposé sur `GET /prestige/challenges`.** Le repo est
  prêt : `ChallengeFilter.Status *ChallengeStatus` (`repository.go:40`) honoré par
  `buildChallengeListQuery` (`prestige_player_helpers.go:83-86`, `status = ?`). Mais
  l'API force `active` : handler `ListActiveChallenges` (`prestige.go:268-280`, input
  `listActiveChallengesInput` sans champ status `:263-266`) → service
  `ListActiveChallenges` (`service.go:576-582`, `Status: &active` en dur). Statuts
  disponibles : `draft/active/completed/expired/abandoned/archived` (`enums.go:24-31`),
  terminaux = completed/expired/abandoned/archived (`IsTerminal`:34-40).
- **Trou n°2 — aucune liste de campagnes closes.** Handler campagne : 7 routes
  create/active/byID/pause/resume/close/abandon (`campaign.go:51-60`), AUCUNE liste.
  `Repo` interface (`campaign/service.go:39-47`) : `GetByID`/`GetActive` mais pas de
  `List`. Statuts campagne `active/paused/completed/abandoned` (`types.go:23-28`),
  `IsEnded` = completed|abandoned (`types.go:75-77`). Colonnes disponibles pour le DTO
  (delta) : `snapshot_value`, `current_value_raw`, `current_value_lowess`, `axis`,
  `axis_kind`, `playlist_group`, `started_at`, `ended_at`, `status`
  (`campaign_repo.go:25-32` / `campaign/types.go:45-69`). Delta snapshot→final dérivable
  = `final - snapshot_value` avec `final = current_value_lowess ?? current_value_raw`.
- **NON-trou — arcs terminés déjà servis** par `GET /arcs` → `ListArcs` →
  `ArcRepo.ListByUser` (SQL sans filtre `completed_at`, `prestige_player_repo.go:315-322`
  ; scan `completed_at` nullable `:60-64`). Le front partitionne sur `Arc.CompletedAt`
  (décision de chantier : AUCUN nouveau lecteur `arc_titles`/`ArcsByTitle`).

### C2 — Compléments d'API (les 2 trous)

- **Trou n°1 — `status` sur `GET /prestige/challenges`** : ajout `Statuses
  []ChallengeStatus` à `ChallengeFilter` (repository.go) honoré par
  `buildChallengeListQuery` (`status IN (...)`, prioritaire sur `Status` mono ;
  callers existants inchangés). Nouvelle méthode service `ListChallenges(ctx, userID,
  titleSlug, statuses)` (interface `Service` + `LazyPrestigeService` + 2 mocks) ;
  `ListActiveChallenges` délègue à `ListChallenges([active])` → comportement historique
  identique. **Décision** : les défis ACTIFS restent enrichis (valeur courante + PP) ;
  les défis TERMINAUX sont servis bruts (leur fenêtre est passée, une valeur courante
  n'aurait pas de sens) — évite aussi N requêtes shared inutiles à l'ouverture de
  l'historique. Handler : `Status []string query:"status"` (CSV, convention Huma
  form/explode=false : `?status=completed,abandoned`) ; vide → défaut `active` appliqué
  par le service ; statut inconnu → 400 `invalid_status`. **Pas de MAJ openapi.yaml** :
  le path `/prestige/challenges` n'y est PAS documenté (réponse `mapOutput`
  non typée) — rien à émettre côté drift, la query est ajoutée en `doc:` de l'input Huma.
  Tests handler : filtre CSV → statuses parsés, défaut nil, statut invalide → 400 ;
  test service : défi terminal non enrichi.
- **Trou n°2 — `GET /players/{slug}/campaigns/history`** : nouvelle méthode repo
  `CampaignRepo.ListEnded` (SQL `status IN ('completed','abandoned')` scopé
  `user_id`+`title_slug`, tri `ended_at DESC NULLS LAST, started_at DESC`) + interface
  `Repo` + service `Service.ListEnded` + fakeRepo. Handler `handleListEnded` + DTO dédié
  `campaignHistoryItem` (id, axe, axis_kind, playlist, statut, dates, snapshot,
  final_value, delta) via `toCampaignHistoryItem`. Delta/FinalValue = méthodes PURES sur
  `ImprovementCampaign` (`FinalValue` = LOWESS lissé sinon brut ; `Delta` = final −
  snapshot ; nil si non évaluée) — colonnes vérifiées sur pièces
  (`campaign_repo.go:25-32`). openapi.yaml : path `listEndedCampaigns` + schémas
  `CampaignHistoryItem`/`CampaignHistoryResponse` (émis via `OPENAPI_EMIT_OUT`, drift
  MISSING=0) ; `generate-types` → generated.ts (2 schémas + operation). Tests : domain
  (FinalValue/Delta), service (ListEnded filtre+scope), handler (projection DTO), repo
  intégration (SQL scope+ordre, `//go:build integration`).

### C3 — Section « Historique » de l'onglet Réalisations

Sous les jalons, nouveau `HistorySection` (3 blocs datés) : objectifs passés (défis
terminaux), arcs terminés, campagnes closes. Hooks + query keys nouveaux :
`useChallengeHistory` (statuts terminaux via `prestigeApi.listChallenges`, clé
`queryKeys.challenge.history`), `useCampaignHistory` (`campaignApi.listEnded`, clé
`queryKeys.playerProfile.campaignHistory` sous `campaignAll` → invalidée par les
mutations de campagne). Types front `CampaignHistoryItem` + `prestigeApi.listChallenges`
(CSV status, chokepoint `/players/…` respecté + ratchet paths). i18n FR/EN
(historyTitle/…/historyResult*). Libellé `archived` = « Retiré » (neutre, distinct
d'« Abandonné »). États vides propres par bloc. Correction cohérente au passage : la
célébration (cartes moments) et `StatsGlobales` sont désormais alimentées par les défis
terminaux (avant : `useChallenges` = actifs seuls → 0 moment, complétion faussée) —
`allChallenges = actifs + terminaux` pour StatsGlobales, `completed` (terminaux) pour les
moments. Campagnes : libellés d'axe/statut/playlist réutilisés du manifest profile
(`useProfileI18n`, `profile.axis.*`/`profile.lusr.*`/`campaign.status.*`).

### C4 — Nettoyage de l'actif

`AscensionObjectivesTab` / `MyArcsSection` : arcs filtrés aux EN COURS (`!a.completed_at`)
— les arcs terminés migrent vers l'Historique. Défis : déjà actifs uniquement (l'API
force `active` pour `useChallenges`) → aucun item passé dans les groupes `[~]`. Séries :
le widget home (`HomeAscensionWidget:46`) filtre déjà `status !== 'broken'` (actives +
protégées) `[~]` ; les séries interrompues restent visibles dans Réalisations
(StreakDashboard, déjà le cas B12).

### C5 — Sorties vers les matchs

Helper partagé `features/filters/filterLink.ts` : `encodeFilterContextParam` (source
UNIQUE du format `?f=`, `createFilterStore.encodeToUrl` refactoré pour l'utiliser — plus
de duplication), `buildSoloFilterLink({playerSlug, titleSlug, cascade?, period?})` →
`/players/{slug}/stats/timeseries?f=…`, `dayWindowUTC`. **Format vérifié sur pièces** :
cascade.modes = valeur = `p.key` du pattern by_mode (= `NormalizeModeLabel(pair)` ===
`modeUI` du filtre, `filters_service.go:337`) ; cascade.maps = `p.label` (nom résolu,
JAMAIS le GUID `p.key`, = `mapUI`). Cartes patterns → `<a href>` (navigation PLEINE PAGE :
le `?f=` n'est décodé qu'au rehydrate du store solo). Records : lien « voir la période »
(PB cards + timeline) → Solo borné sur la JOURNÉE UTC du record (`dayWindowUTC`, le
backend traite `end_date` en fin de journée inclusive, `filters_service.go:381`). i18n
FR/EN (patternSeeMatches, recordSeePeriod). Tests vitest `filterLink.test.ts` (round-trip
encode/decode, by_mode/by_map cascade, période bornée, estampille titre, dayWindowUTC).

## Journal d'exécution — Lot D (2026-07-22)

### D1 — Décision timeseries (vérifiée sur pièces, confirme AM-8)

**Constat sur pièces.** `internal/api/handlers/timeseries.go` ne sert AUCUNE série
skill/LUSR (5 onglets : summary/cumul/intensity/distributions + first-events ;
`domain.TimeseriesPageResponse`). Seul un champ **per-match** `SkillRatingValue`
existe (`domain/timeseries.go:243-248`), non agrégé en série lissée.
`service/timeseries_service.go:53-57` confirme que `canonical.MetricSeries` ne
couvre pas lusr. Donc rien de directement réutilisable côté page Solo.

**Ce qui existe et se réutilise (règle n°14, zéro réimplémentation).**
- **Source de données** : la vue append-only `match_skill_rank_latest` (règle ART
  n°2 — lecteurs via `_latest` uniquement), `rating_value` où `rating_type='LUSR'`,
  ordonnée par `COALESCE(start_time, written_at)` (les rows LUSR ont `start_time`
  souvent NULL). C'est EXACTEMENT ce que lit déjà `profile.loadMuSeries`
  (`progression/profile/service.go:582-612`). LUSR = chemin v2 canonique ; la même
  colonne `rating_value` = μ = la valeur affichée « {mu} pts LUSR » côté Performance
  (`PerformanceSection.tsx:62`, `rating.mu.toFixed(0)`). Échelle 1000–2000+
  (`sync/skill/skill_config.go:230`) : le lissé est directement en points LUSR,
  aucune conversion.
- **Lissage LOWESS** : `temporal.LowessSmooth(series, LOWESSAlpha=0.3)`
  (`analysis/temporal/lowess.go`), DÉJÀ utilisé par `profile.ComputeMuTrend` et par
  la campagne (`current_value_lowess`). Réutilisé tel quel.

**Décision (D1).** CONSTRUIRE la série depuis l'existant (défaut amendé confirmé),
**PAS** de réutilisation d'un endpoint timeseries inexistant.
- **Surface D2 (sparkline)** : champ ajouté à la réponse `/profile`
  (`PlayerProfile.skill_trend`), calculé sur une fenêtre FIXE de 90 j dans
  `BuildProfile` (indépendante du `window_days` du profil — sémantique stable
  « tendance 90 j »). Rationale : la `PerformanceSection` consomme déjà la réponse
  `/profile` (skill_rating, components, mu_trend) dans l'onglet Profil → **aucune
  nouvelle query key, aucun hook, aucun 2e fetch**. Un endpoint dédié dupliquerait
  la résolution joueur + une clé + un hook pour une série minuscule (DEC-5 minimal).
  Le backend sert UNIQUEMENT les points **lissés** (`{date, value}`), jamais le μ
  brut (DEC-6 « jamais de μ brut à l'écran » garanti côté serveur). `< 3` points →
  série vide (LOWESS non fiable), front n'affiche rien.
- **Surface D3 (calendrier)** : les counts/jour n'existent NULLE PART (les hits
  « daily » = cadence de défis, sans rapport ; les heatmaps activité existantes
  Explorer/Synthesis sont jour×heure 7×24, PAS un calendrier 90 j). → **nouvel
  endpoint** `GET /players/{slug}/activity-calendar?days=90`, handler mince réutilisant
  `ProgressionResolver` + `profile.Service`, lecteur via `SharedReadDB` avec le
  fragment timezone canonique `StartTimeCanonicalSQL("mr")` + exclusion Campagne
  (`campaignExcl`), bucket jour = `t.UTC().Format("2006-01-02")` en Go (motif A6,
  évite les pièges TZ du CAST DATE DuckDB). Huma type l'output → schémas openapi
  émis + `generate-types`.

### D2 — Sparkline Performance (Profil)

Champ `PlayerProfile.SkillTrend []SkillTrendPoint{date, value}` servi par `/profile`,
calculé sur 90 j FIXE dans `BuildProfile` (const `SkillTrendWindowDays`). Backend :
`loadMuSeriesPoints` (vue `_latest`, timestamp `COALESCE(start_time, written_at)`) +
`buildSkillTrend` (réutilise `temporal.LowessSmooth`, LOWESSAlpha ; date =
`t.UTC().Format`). Sert UNIQUEMENT le lissé, `< 3` points → nil (front n'affiche rien).
openapi.yaml : schéma `SkillTrendPoint` + `skill_trend` sur `PlayerProfile` (émis via
`OPENAPI_EMIT_OUT`/`_DIVERGENT_OUT`, drift MISSING=0, PlayerProfile ré-aligné) +
`generate-types`. Front : type `SkillTrendPoint` (`lib/playerProfile.ts`),
`PerformanceSection` rend `<TimeseriesLineChart>` compact (height 96, `showSymbol=false`,
`smooth`, `xAxisType='category'`, valeurs arrondies pour un tooltip lisible), sous role
`img` + aria manifesté, i18n profile.toml (`trend_chart_title/_series/_aria`), `< 2`
points → rien. Tests : pur Go `buildSkillTrend` (lissage + date UTC), intégration
`SkillTrendPopulated`, vitest `PerformanceSection.test.tsx` (mock echarts-for-react :
rendu ≥ 2 pts / aria / états vides < 2 et absent).

### D3 — Calendrier d'activité (Réalisations)

Vérification existant (go-features) : AUCUN endpoint ne sert de counts/jour (les hits
« daily » = cadence de défis ; heatmaps Explorer/Synthesis = jour×heure 7×24). → nouvel
endpoint `GET /players/{slug}/activity-calendar?days=90` (clamp 7..180). Backend :
`profile.LoadActivityCalendar` (SharedReader, fragment canonique `StartTimeCanonicalSQL` +
`campaignExcl`, bucket jour `t.UTC().Format` en Go — motif A6, dédup match/jour), DTO
`ActivityCalendar{since, until, days[]ActivityDay{date, count}}` (jours vides omis).
Handler mince ajouté à `PlayerProfileHandler.Mount`. openapi.yaml : path
`getActivityCalendar` + schémas `ActivityCalendar`/`ActivityDay` (drift MISSING=0) +
`generate-types`. Front : query key `progressionActivity`, hook `useActivityCalendar`,
types `ActivityCalendar/ActivityDay` (ascension `types.ts`), composant
`ActivityCalendarChart` (compose `ChartCard` + `buildActivityCalendarOption` : heatmap
semaine×jour, lundi en haut, rampe NEUTRE `heatmapRampTokens('frequency')` CVD-safe,
`dowLabels`/`calendarChartText` réutilisés, légende sobre Moins/Plus, tokens uniquement),
i18n ascension.i18n (`activityCalendar*` FR/EN), rendu sous `StreakDashboard` dans
`AscensionRealisationsTab`. Tests : intégration Go `LoadActivityCalendar` (counts jours
distincts, fenêtre vide), vitest `ActivityCalendarChart.test.tsx` (grille pure semaine×jour,
builder option, rendu + état vide, mock echarts-for-react).

### Gate Lot D — sorties fidèles (exécutées cette session)

- `go test ./internal/progression/profile/... ./internal/api/handlers/...` → ok
- `CGO_ENABLED=1 go test ./internal/api/` (drift `TestOpenAPISchemaDrift` MISSING=0 +
  contract routes) → ok
- `CGO_ENABLED=1 go test -tags=integration ./internal/progression/profile/... -run
  'SkillTrend|ActivityCalendar'` → ok
- `gofmt -l` (profile+handlers) vide ; `go vet` (profile+handlers+api) clean ;
  `make go-api-lint` (vet domain/analysis, périmètre du target) EXIT 0
- `make generate-types` → generated.ts régénéré (ActivityCalendar/ActivityDay/
  SkillTrendPoint + skill_trend/getActivityCalendar) ; `make check-types` (tsc) → ok
- `make test-web` → 283 fichiers, 2480 passés / 14 skippés / 0 échec
- ESLint fichiers web touchés → 0 erreur (warning react-refresh supprimé sur les builders
  exportés, motif catalogue wrappers)

Reste à faire orchestrateur : revue visuelle (sparkline ≥ 30 matchs rendue ; heatmap
cohérente avec les cartes séries, un jour joué = case remplie) ; entrée thought_log
ajoutée en tête.
