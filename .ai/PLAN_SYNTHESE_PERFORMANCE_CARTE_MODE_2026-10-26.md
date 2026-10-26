# Plan : score de performance par carte et par mode dans la Synthèse

> Rédigé le 2026-10-26, relu sur pièces le même jour. Contrat d'exécution : skill `plan-execution`
> (ordre strict, une étape à la fois, statut de chaque item `[x]` / `[~]` réf / `[!]` justifié,
> aucune case vide à la clôture, zéro fix hors périmètre, découvertes notées en fin de plan).
> Branche : `feat/synthese-performance-carte-mode`, depuis la branche d'intégration de la release suivante (`release/v8.0.7` part en prod le 2026-10-26 ; à confirmer avec le user avant E1), dans un worktree dédié
> (`LevelUp-wt-synthese-perf`). Effort : moyen (un champ d'API, un graphique enrichi).

## Objectif et critère de succès

Section « Activité & répartition » de la Synthèse : les deux graphiques existants « Par carte » et
« Par mode » (bâtons empilés des issues) portent, au-dessus de chaque colonne, une barre courte du
score de performance moyen (0..100) colorée par palier. Même axe des colonnes, même ordre, aucune
nouvelle section. Critère : sur la base réelle, chaque carte et chaque mode listés montrent leur
score moyen ; une colonne sans aucun match noté montre un emplacement en pointillé ; la moyenne
d'une ligne est celle des matchs de cette ligne qui ont un score (règle déjà appliquée par la
Synthèse pour la vue d'ensemble, `stats_canonical.go`).

## Décisions produit (validées par le user le 2026-10-26)

- **D1 — On enrichit l'existant**, pas de nouveau graphique, pas de bloc sur la page Tendances
  (Tendances = évolution dans le temps ; un croisement mois × carte y serait presque vide).
- **D2 — La stat est le score de performance** (`performance_score`), pas la FDA.
- **D3 — Aucun minimum de matchs** : les graphiques listent toutes les cartes et tous les modes,
  la barre de performance aussi.
- **D4 — Une carte = une seule famille de modes** (pas de carte jouée à la fois en 4c4 et en
  12c12) : moyenne simple par carte, pas de découpage par chaîne de référence.
- **D5 — Variantes de carte** (« Live Fire » / « Live Fire - Ranked ») : hors de ce lot, traitées
  par `PLAN_CARTES_VARIANTES_STATS_2026-10-26.md` (étape E2, item Synthèse,
  worktree `LevelUp-wt-cartes-variantes`). Voir « Coordination ».

## Lecture sur pièces (2026-10-26)

| Élément | Constat |
|---|---|
| Agrégat | `buildBreakdownsFromCanonical` (`apps/go-api/internal/service/synthesis_service_legacy.go`, ~100 L, au-delà du seuil de 80) compte V/D/N/non terminé par libellé de carte et de mode ; le `switch` d'issue est écrit deux fois |
| Types | `domain.SynthesisMapEntry` / `SynthesisModeEntry` (`internal/domain/synthesis.go`) : comptes + `win_rate`, aucun champ de performance |
| Score | `canonical.PlayerMatchRow.Enrichment.PerformanceScore *float64` ; palier canonique `analysis.PerfTier` (seuils 80/65/50/35), déjà exposé ailleurs en `perf_tier` (`domain/explorer.go`, `domain/match_history.go`) |
| Libellé | `fields.performance_score` présent dans les deux titres (`config/titles/{halo_infinite,halo_5}/mappings/fields.toml`) -> `useFieldLabel('performance_score')`, aucune chaîne en dur |
| Graphique | `SynthesisOutcomesByGroupChart.tsx` (159 L) : bâtons VERTICAUX, catégories en abscisse inclinées à 45° ; `ChartCard.title` accepte un `ReactNode` (donc `titleWithInfo`) |
| Couleurs | tokens `perf-tier-1..5` (palettes par défaut et cividis) |
| Tests | aucun test Go de `buildBreakdownsFromCanonical` ; web : `SynthesisChartsEmpty.test.tsx`, `SynthesisPage.test.tsx` |

## Coordination avec le plan des variantes de carte

Les deux lots modifient `buildBreakdownsFromCanonical`. Partage :
- ce lot ne touche PAS à la clé de regroupement des cartes (libellé) ni au filtre de cartes ;
- il extrait le comptage d'une ligne dans un helper (`tallyBreakdown`) que le lot variantes
  réutilise tel quel : seul le calcul de la clé change de son côté ;
- le second lot fusionné dans la branche de release rebase et résout ; après les deux, le test
  « une seule entrée pour base + variante » du lot variantes doit aussi vérifier que la moyenne
  porte sur les matchs des deux (naturel si la clé est commune).
- avant E1, prévenir la session qui exécute le plan variantes (via le user si elle n'est pas
  identifiable) ; consigner la réponse ici.

## Étapes

### E0 — Vérifications sur pièces (copie des bases, jamais la base tenue par le serveur)
- [x] Halo Infinite : part des matchs du joueur principal ayant un `performance_score` non nul,
      par carte et par mode (les 10 plus jouées de chaque) — voir Mesures
- [x] Halo 5 : même relevé. Si AUCUN match n'a de score, ARRÊT et décision user (la barre serait
      en pointillé partout : masquer pour ce titre suppose une capability, hors de ce plan)
      — 89,5 % des matchs ont un score : pas d'arrêt
- [x] Matchs non terminés : ont-ils un score ? (la moyenne suit la règle de la vue d'ensemble
      quoi qu'il en soit ; le relevé sert à expliquer une moyenne au user) — non : 0 sur 39 (HI)
      et 0 sur 27 (H5)
- Gate : relevé écrit dans la section Mesures, aucune case ouverte.

### E1 — API
- [x] `domain.SynthesisMapEntry` et `SynthesisModeEntry` : `PerformanceAvg *float64
      json:"performance_avg,omitempty"` et `PerfTier int json:"perf_tier,omitempty"`
      (0 si aucun score, même convention que `domain/explorer.go`)
- [x] `synthesis_service_legacy.go` : helper `tallyBreakdown(e *entry, r canonical.PlayerMatchRow)`
      (issue + somme et nombre des scores non nuls), appelé pour la carte et pour le mode ; le
      `switch` dupliqué disparaît ; `buildBreakdownsFromCanonical` repasse sous 80 L
- [x] Construction des entrées : moyenne si au moins un score, `analysis.PerfTier(avg)` pour le palier
- [x] `go-features` / grep : vérifier qu'aucun helper existant ne fait déjà « moyenne des
      `*float64` non nuls » sur des `canonical.PlayerMatchRow` ; s'il existe, l'utiliser
      — aucun (la moyenne de la vue d'ensemble vient de `stats_canonical.go`, sur un autre
      type) ; `breakdownTally.performance()` nouveau, palier par `analysis.PerfTier`
- [x] Test `synthesis_service_breakdowns_test.go` : moyenne sur les seuls matchs notés ; ligne
      sans aucun score -> `nil` et palier 0 ; palier aux seuils (80 -> 1, 79.9 -> 2) ; issues
      inchangées (non-régression) ; même chose côté mode
- [x] `make openapi-gen` puis `make generate-types` (jamais d'édition à la main de `openapi.yaml`
      ni de `generated.ts`)
- Gate : `cd apps/go-api && go test ./internal/service/ ./internal/domain/ ./internal/api/`
  (inclut `TestOpenAPIYAMLIsUpToDate`) + `make go-api-lint` sans nouvelle entrée de baseline.
  Passé le 2026-10-26 : trois paquets `ok` ; `golangci-lint run --new-from-merge-base=origin/main`
  = 0 issues ; `openapi.yaml` et `generated.ts` régénérés (+12 lignes chacun, deux types) ;
  aucune ligne de baseline touchée.

### E2 — Graphique
- [x] `SynthesisPage.tsx` : `performance_avg` et `perf_tier` passés dans les `entries` des deux
      graphiques (aucune logique ajoutée dans la page)
      ; le prop `height={360}` est retiré, la hauteur vit dans le graphe
- [x] `buildOutcomesOption` : seconde grille AU-DESSUS des bâtons empilés, mêmes catégories
      (`xAxis` lié, étiquettes masquées sur la grille du haut), axe 0..100 nommé par
      `useFieldLabel('performance_score')`, une barre par colonne colorée `perf-tier-N` via
      `resolveToken`, valeur arrondie à l'entier au-dessus de la barre
- [x] Colonne sans score : emplacement en pointillé pleine hauteur (contour en tirets, fond
      transparent, token neutre existant), pas de valeur affichée
- [x] Infobulle : une ligne « <libellé du champ> : 72 » ajoutée aux lignes d'issues ; « — » si
      pas de score ; `axisPointer.link` sur les deux abscisses pour un survol commun
- [x] Titre des deux cartes via `titleWithInfo` : une bulle qui dit ce que montre la barre du
      haut et le sens des couleurs (chaîne FR + EN dans `lib/i18n/manifests/synthesis.toml`,
      clé `synthesis.breakdown.performance_info`, puis régénération du manifeste)
- [x] Hauteur des deux cartes ajustée (360 -> valeur mesurée pour garder la hauteur actuelle des
      bâtons empilés) — 450 px, CALCULÉE (360 + bande 64 + espace 26 ; bâtons empilés toujours
      230 px) et non mesurée à l'écran : à confirmer en E3
- [x] Tests `SynthesisOutcomesByGroupChart.test.ts` (nouveau) : couleur par palier, colonne sans
      score en pointillé, ordre des catégories identique dans les deux grilles, liste vide =
      option vide (comportement actuel)
- Gate : `make check-types`, `make test-web` (hors bac à sable), `npm run lint` dans `apps/web`,
  aucune valeur hex ni classe de couleur ajoutée (skill `color-tokens`).
  Passé le 2026-10-26 : `npm run typecheck` (tsc -b) exit 0 ; `npm run test:run` 1005 fichiers
  passés, 5 ignorés, 10 376 tests passés, 23 ignorés (dont le garde-rail
  `generated-types-fresh`) ; `npm run lint` 0 erreur, 26 avertissements, tous hors synthèse
  (React Compiler / TanStack Table, antérieurs) ; grep hex et classes Tailwind couleur sur le
  graphe : aucun ; `tools/lint-no-hardcoded-fields.mjs` : aucune violation.

### E3 — Vérification visuelle et clôture
- [ ] Serveur local (air + vite) : Synthèse du joueur principal, FR puis EN, thème clair et sombre,
      palette cividis ; captures envoyées au user, qui désigne les témoins (gate visuel)
- [ ] Un filtre qui laisse une seule carte ; une sélection vide (état vide du `ChartCard` intact)
- [ ] Changelog EN/FR de la release visée : une ligne (rédigé avec les notes de la v8.1.0, section pas encore ouverte)
- [ ] `.ai/thought_log.md` : entrée Complété (à poser après le gate visuel ; entrée « En cours » à jour) ; `.ai/REGISTRE_REPORTS.md` si un item finit en `[!]`
- [x] Skill `delivery-checklist` ; commit et fusion dans `release/v8.1.0` à la demande du user (2026-10-26), AVANT le gate visuel
      (coordination avec le lot variantes : voir Coordination)
- Gate : validation visuelle du user + checklist de livraison verte.

## Reprise de session

Lire ce fichier (cases cochées = fait), `git log --oneline -10` sur la branche, puis la dernière
entrée du thought log qui la cite. Reprendre à la première case vide de l'étape la plus basse.

## Mesures

Relevé E0 du 2026-10-26, sur COPIES (supprimées ensuite), via `cmd/diag_q`. Matchs du joueur
principal = `match_participants` du titre joint à `match_registry` ; score =
`player_match_enrichment_latest.performance_score`.

Écarts de méthode (à lire avant de se fier aux chiffres) :
- Halo Infinite : la base live de JGtm est tenue par le serveur (copie impossible, verrou
  exclusif). Le relevé porte sur la sauvegarde `data/backups/avant_recuisson_98_2026-10-16/players/JGtm`
  (1 182 matchs ; la base partagée copiée, elle, est à jour). Les parts sont stables d'une
  semaine à l'autre, mais ce n'est pas l'état du jour.
- Modes HI : libellé brut du registre dont on retire le suffixe « on <carte> » ; approximation de
  `analysis.NormalizeModeLabel` (qui est celui utilisé par le service), pas son résultat exact.
- Halo 5 : `match_registry` ne porte aucun nom de carte ni de mode (map_name / pair_name vides),
  seulement `map_id` ; les noms viennent du référentiel (verrouillé, non copié) : le relevé
  par carte H5 est donc par `map_id`. `PairMode` est nil pour H5 (`halo_5/mapping.go`) : la liste
  « Par mode » de la Synthèse est vide pour ce titre, seule « Par carte » est concernée.

| | Halo Infinite (JGtm) | Halo 5 (Chocoboflor) |
|---|---|---|
| Matchs | 1 182 | 1 855 |
| Avec score | 1 082 (91,5 %) | 1 661 (89,5 %) |
| Cartes distinctes / sans aucun score | 72 / 19 | 36 / 15 |
| Modes distincts / sans aucun score | 38 / 13 (approx.) | sans objet (liste vide) |
| Score min / moyenne / max | non relevé | 4,2 / 50,4 / 96 (aucun zéro) |

Par issue (avec score / total) : HI victoires 545/579, défaites 533/556, égalités 4/8, non
terminés 0/39. H5 victoires 827/966, défaites 834/862, non terminés 0/27. Aucun match non
terminé n'a de score ; les lignes sans score sont surtout des cartes ou modes rares (1 à 4 matchs).

10 cartes les plus jouées, HI (matchs, % avec score, moyenne) : Illusion 56, 100 %, 48,5 ;
Bazaar 53, 92,5 %, 49,9 ; Streets 52, 92,3 %, 60,3 ; Live Fire 50, 92 %, 57,3 ; Cliffhanger 50,
96 %, 44,1 ; Prism 48, 95,8 %, 53,5 ; Recharge 47, 93,6 %, 52,3 ; Aquarius 46, 95,7 %, 55,0 ;
Catalyst 45, 97,8 %, 50,3 ; Behemoth 44, 93,2 %, 52,3.

10 modes les plus joués, HI (approx.) : Super Fiesta:Slayer 433, 94,9 %, 51,2 ; Community:Team
Slayer 144, 93,8 %, 49,7 ; Arena:CTF 128, 95,3 %, 49,1 ; Arena:Team Slayer 103, 95,1 %, 51,6 ;
Community:Slayer 91, 90,1 %, 49,6 ; Arena:Slayer 83, 92,8 %, 56,2 ; Arena:Strongholds 66,
92,4 %, 47,5 ; Arena:King of the Hill 38, 94,7 %, 56,0 ; Arena:Neutral Flag CTF 24, 91,7 %,
48,4 ; Tactical:Slayer 13, 76,9 %, 39,7.

10 cartes les plus jouées, H5 (par map_id, 8 premiers caractères) : cebd854f 206, 98,1 %, 51,3 ;
caacb800 203, 98,0 %, 50,1 ; ce1dc2de 202, 98,5 %, 51,3 ; c74c9d0f 159, 96,9 %, 50,3 ;
d67fdcb9 109, 97,2 %, 49,9 ; cd844200 108, 97,2 %, 50,4 ; cb914b9e 93, 95,7 %, 45,9 ;
c7b7baf0 82, 98,8 %, 51,3 ; cc040aa1 71, 95,8 %, 49,3 ; b0426fa5 68, 100 %, 51,8.

Lecture : la moyenne d'une ligne exclut les non terminés (sans score) et les rares matchs sans
score ; toutes les lignes bien jouées (>= 40 matchs) ont plus de 90 % de matchs notés, sauf
Tactical:Slayer (77 %, 13 matchs).

## Découvertes hors périmètre

- La section « Lecture sur pièces » et l'objectif citent `trPerf` comme moyenne de la vue
  d'ensemble : `trPerf` (`synthesis_service_canonical.go`) est un suivi du MEILLEUR match, pas une
  moyenne. La règle « moyenne des matchs notés » est bien celle de la vue d'ensemble
  (`stats_canonical.go`) ; la référence du plan est à corriger.
- Halo 5 : `PairMode` nil, donc la liste « Par mode » est vide sur ce titre (graphe en état vide) ;
  seul « Par carte » porte la barre. Comportement antérieur au lot, non modifié.
- Halo 5 : `match_registry.map_name` est vide pour les 3 032 matchs de la base partagée (seul
  `map_id` est écrit) ; les noms viennent du référentiel. Non vérifié ici que le libellé s'affiche
  bien côté interface : à regarder en E3 sur le profil Halo 5.
- Environnement : la base live de JGtm (Halo Infinite) est verrouillée en lecture par le serveur
  local, donc E0 a lu la sauvegarde du 2026-10-16 ; `make` n'est pas dans le PATH PowerShell
  (les commandes de `openapi-gen` / `generate-types` ont été jouées directement) ; un
  `golangci-lint` d'une autre session tournait en parallèle (relancé avec `--allow-serial-runners`).
- Coordination : le plan variantes n'a PAS été prévenu depuis cette session (pas d'accès à son
  agent) ; à relayer par le user avant la fusion des deux lots dans la branche de release.
