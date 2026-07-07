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

- [ ] **A1 — Labels de cartes dans les patterns** : `analyzeByMap` groupe par `MapID`
  brut et le handler `internal/api/handlers/patterns.go` le sert tel quel. Ajouter un
  champ `label` au DTO `ContextualPattern` résolu au niveau handler/service via le
  référentiel metadata du titre (même source que les autres pages : asset_translations
  / registre, title-agnostic via le registry — PAS de SQL inline dans le handler,
  passer par le reader/port existant). Fallback si non résolu : label = « Carte
  inconnue » + id court, jamais le GUID nu. Test handler : chaque pattern `by_map`
  servi a un `label` non vide.
- [ ] **A2 — Clés `with_friends`/`solo` et texte des leviers** : le front mappe les clés
  `by_squad` sur `squadVsSoloSolo`/`squadVsSoloSquad` (i18n existants) au lieu
  d'afficher la clé ; le texte du levier `map_avoidance` utilise le `label` de A1
  (« Améliore ton win rate sur Aquarius », plus jamais un GUID dans une phrase).
- [ ] **A3 — Clés de métriques Prestige** : `CreateChallengeForm` stocke `FieldKDA`,
  `StatsGlobales` l'affiche brut. Créer un helper front unique `metricLabel(key,
  locale)` (réutilisant `METRIC_LABEL_*` d'ascension + mapping des clés `Field*`
  Prestige) et migrer TOUS les affichages de clé métrique de `features/prestige` +
  `features/ascension` dessus. Garde-rail (règle n°6) : test vitest du helper —
  clé inconnue → libellé humanisé, jamais la clé brute ; grep-test interdisant
  `\bField[A-Z][a-zA-Z]+\b` dans le JSX rendu des deux features.
- [ ] **A4 — Bornes de vraisemblance records** : dans le detector records
  (`internal/progression/`), bornes par métrique (accuracy ∈ [0,1] côté stockage ADR
  0006, kda ∈ [-20, 50], kpm ∈ [0, 20], etc. — les fixer en constantes nommées,
  pas de magic numbers). Hors bornes : pas de persist + `slog.WarnContext`. À la
  lecture (endpoint `/records`), les métriques absentes du catalogue de labels (ex
  `best_kda` legacy) ne sont pas servies + log. Tests unitaires analysis purs.
- [ ] **A5 — Purge one-off des records corrompus** : commande diag/ops (pattern one-off
  container existant) qui liste puis supprime les lignes `player_records` /
  `record_history` hors bornes A4 pour les 4 joueurs. DuckDB : écritures via le repo
  existant (jamais de bare connect), règles ART respectées (tables concernées non
  append-only critiques — vérifier sur pièces avant, sinon recette ADR 0026).
- [ ] **A6 — Dates de jalons backfillées** : toutes les `earned_at` = date du seed
  (30/05/2026). Recalculer la vraie date de franchissement par jalon (cumul des
  métriques sur `match_registry` jusqu'au seuil — dérivable et déterministe) via une
  commande de backfill one-off ; si non dérivable pour un jalon, `earned_at` NULL et
  le front n'affiche pas de date (jamais une date fausse).
- [ ] **A7 — Levier sans valeur courante** : « Actuel — → Cible 37 % » ; si
  `current` absent, masquer la ligne Actuel/Cible et ne garder que l'horizon. Fix
  front `LeverList.tsx`.
- [ ] **A8 — Unités des séries hebdomadaires** : `streakBestLength` code « jour » en dur
  (i18n.ts:222) → « Record perso : 2 jours » sur une série hebdo. Paramétrer l'unité
  (jour/semaine) selon le type, FR + EN.
- [ ] **A9 — Condition de jalon lisible** : « accuracy >= 0.50 sur 90 jours distincts »
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

- [ ] **B1 — Restructuration 4 onglets** (DEC-3) — périmètre fermé :
  - [ ] route `objectifs.tsx` créée ; `index.tsx` rend le nouvel onglet Profil
    (routing file-based, ne jamais éditer routeTree.gen.ts)
  - [ ] `AscensionProfileTab` renommé `AscensionObjectivesTab` (contenu Prestige
    inchangé) ; nouveau `AscensionProfilTab` composant : PlayerProfileV3 (identité /
    style / performance) + PatternsSection (déplacée depuis le tab coaching) +
    SquadVsSoloCard + BehaviorAlertList
  - [ ] `ProgressionSection` + `StartCampaignModal` recomposées dans
    `AscensionCoachingTab` (sortent de PlayerProfileV3 — prop ou recomposition,
    vérifier sur pièces la plus simple)
  - [ ] `AscensionLayout` : 4 liens d'onglets, i18n FR/EN (labels + aria), sous-titres
    LayerSection relus
  - [ ] deep-links mis à jour (grep exhaustif) : `features/notifications/navigation.ts`,
    `lib/pageTitle.ts`, `features/feedback-drawer/classifyFeedback.ts`,
    `components/shell/navL1Sections.tsx` (+ NavL1MobileMenu)
  - [ ] tests : AscensionCoachingTab.test.tsx et AscensionProfileTab.test.tsx adaptés,
    nouveau test du tab Profil (mock echarts-for-react si radar rendu)
- [ ] **B2 — Deep-link du widget home** : `HomeAscensionWidget` montre des séries mais
  pointe l'onglet objectifs. Cible → `/ascension/realisations`. Vérifier le mapping
  des notifications (features/notifications/navigation.ts) vers le bon onglet aussi.
- [ ] **B3 — Câblage mode pilote** (DEC-1) : mutations front sur
  `POST /pilot-mode/enable|disable` (queryKeys dans `lib/query/keys.ts`, jamais
  inline), toggle réel avec état issu des défis `mode === 'pilote'` actifs, retrait du
  tooltip « non implémenté », i18n FR/EN, invalidation de `useChallenges` après
  attribution. CTA d'activation dans l'empty state « Aucun objectif libre actif ».
- [ ] **B4 — Coach proactif défaut ON** (DEC-2) : défaut `CoachProactiveMode` à true
  (`internal/platform/settings/store.go` + domain + test du défaut inversé), doc du
  changement dans le commit (anti-pattern doc inversée : mettre à jour TOUTE mention
  du défaut, grep `coach_proactive_mode`). Le hint d'activation dans
  `CoachProposalsCard` reste pour ceux qui l'ont explicitement désactivé.
- [ ] **B5 — Confirmations cohérentes** : `confirm()` natif d'abandon d'objectif
  (AscensionProfileTab.tsx:76) → `AlertDialog` (même composant que les campagnes) ;
  bouton « Abandonner » rattaché visuellement à sa carte (dans la carte, pas flottant
  sous la grille).
- [ ] **B6 — Abréviations et cibles** : tooltips sur OC (« Conversion offensive ») et DR
  (« Résistance défensive ») partout où ils apparaissent ; les « cible 98 % »
  uniformes de la section Performance : vérifier le calcul de cible par composante —
  si la cible est bien un même percentile global, l'étiqueter (« cible palier
  suivant ») ; si c'est un bug de calcul, le corriger (vérifier sur pièces côté
  service profile).
- [ ] **B7 — Dédoublonnage Solo/Escouade** : l'onglet Entraînement montre 2 cartes
  patterns `by_squad` PUIS une carte « Comparaison Solo / Escouade » avec les mêmes
  chiffres. Garder la comparaison, supprimer les 2 cartes du grid (filtre `by_squad`
  hors de `PatternContextGrid`). Code mort supprimé avec ses tests (règle n°7).
- [ ] **B8 — LUSR au lieu de μ/σ** (DEC-6) : section Performance affiche le tier + les
  points LUSR et l'écart au palier en points LUSR ; μ/σ retirés de l'UI.
- [ ] **B9 — Vocabulaire FR** (DEC-4) : « Mes milestones » → « Mes jalons »,
  `realisationsEmpty` sans « moment cards », relecture des strings FR de la feature
  (i18n.ts) pour anglicismes résiduels. EN inchangé sur le fond.
- [ ] **B10 — Barre Prestige lisible** (DEC-9) : progression intra-niveau explicite
  (« 0 / 1500 PP vers Mythique »), total PP en libellé secondaire, amis à 0 PP omis.
- [ ] **B11 — Badge petits échantillons** (DEC-8) : sous 10 matchs, badge neutre
  « Échantillon faible » à la place de Force/Faiblesse (front, seuil constant nommé
  partagé avec le backend via la réponse — pas de dur en double).

**Gate Lot B** :
```
cd apps/go-api && go test ./internal/platform/settings/... ./internal/api/handlers/...
make check-types && make test-web && make go-api-lint
# revue visuelle : 4 onglets rendus (Profil = index, patterns dans Profil, pistes de progression dans Entraînement) ; toggle pilote aller-retour (enable → défis pilotes visibles → disable) ; coach ON par défaut sur profil vierge
```

## Lot C — Historique (actif vs passé) [demande produit centrale]

Principe : l'onglet Objectifs ne montre QUE l'actif ; l'onglet Réalisations devient la
mémoire complète (séries, records, jalons, objectifs/arcs/campagnes passés).

- [ ] **C1 — Inventaire backend sur pièces** : vérifier ce que les endpoints servent
  déjà — statuts challenge existants : `draft/active/completed/expired/abandoned/
  archived` (enums.go) ; l'endpoint liste accepte-t-il un filtre statut ? Les
  campagnes closes sont-elles listables (pas seulement l'active) ? Arcs terminés ?
  Sortie de C1 : liste écrite des trous d'API (potentiellement zéro — ne rien
  réimplémenter qui existe, skill `go-features`).
- [ ] **C2 — Compléments d'API si trous** : au besoin, paramètre `status` sur la liste
  challenges et endpoint campagnes passées. Handlers minces, DTOs dédiés,
  openapi.yaml + `make generate-types` (le test contract force la doc).
- [ ] **C3 — Section « Historique » de l'onglet Réalisations** : sous les jalons,
  3 blocs chronologiques — objectifs passés (terminés / abandonnés / expirés, avec
  résultat), arcs terminés, campagnes closes (axe, delta snapshot→final, playlist).
  Les cartes moments existantes restent le bloc « célébration » ; l'historique est la
  liste exhaustive datée. Query keys centralisées, i18n FR/EN.
- [ ] **C4 — Nettoyage de l'actif** : onglet Objectifs = objectifs actifs + arcs en
  cours uniquement (plus de complétés dans les groupes) ; séries cassées : visibles
  dans Réalisations avec leur date, mais l'accueil (widget) ne montre que
  actives/protégées (déjà le cas).
- [ ] **C5 — Sorties vers les matchs** : cartes patterns cliquables → page Solo avec le
  filtre correspondant (le paramètre d'URL `f=` des pages stats existe déjà — réutiliser
  l'encodage existant, pas de format maison) : pattern by_mode → filtre mode, by_map →
  filtre carte. Pour les records : lien « voir la période » → page Solo bornée sur la
  fenêtre du record. Aucun cul-de-sac restant sur ces deux familles de cartes.

**Gate Lot C** :
```
cd apps/go-api && go test ./... (paquets touchés) ; make generate-types (si API)
make check-types && make test-web
# revue visuelle : onglet Objectifs sans items passés ; Réalisations montre l'objectif FDA abandonné/terminé dans l'historique ; clic pattern Super Fiesta → page Solo filtrée
```

## Lot D — Tendance visuelle (graphes minimaux, DEC-5)

- [ ] **D1 — Vérifier l'existant timeseries** : `internal/api/handlers/timeseries.go` +
  SPEC_ECHARTS_TIMESERIES (13 graphes) — si une série skill/LUSR par match existe,
  la réutiliser telle quelle (lecteurs : vues `_latest` uniquement, règle ART n°2).
  Sortie : décision écrite réutilisation vs nouvel endpoint (défaut : réutilisation).
- [ ] **D2 — Sparkline Performance** : mini-graphe 90 j (μ lissé LOWESS affiché en
  points LUSR) dans la section Performance, wrapper ECharts existant
  (`components/charts/`), tokens sémantiques, mock echarts-for-react dans les tests
  jsdom (piège mémorisé).
- [ ] **D3 — Calendrier d'activité** : heatmap 90 j des jours joués (données : counts
  par jour — vérifier l'existant côté home/engagement avant tout nouvel endpoint),
  rattachée au bloc séries de Réalisations. Timezone : fragment SQL canonique
  `COALESCE(start_time_utc, ...)` obligatoire.

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

## Découvertes en cours d'exécution (à consigner, ne pas traiter)

- (vide)
