# Plan : le mode d'un match est fourni par le titre, une fois, pour toutes les pages

> Rédigé le 2026-10-26. Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la
> fois, statut de chaque item `[x]` / `[~]` réf / `[!]` justifié, aucune case vide à la clôture,
> zéro fix hors périmètre, découvertes notées en fin de plan).
> Branche : `feat/modes-par-titre`, depuis `release/v8.1.0` (6d3cfa8ca), worktree
> `LevelUp-wt-modes-titre`. Fusion dans `release/v8.1.0` à la fin, sur accord du user.

## Objectif et critère de succès

Le mode d'un match est porté par le champ canonique `MatchSummary.PairMode`, rempli par la
source du titre : Halo Infinite = nom de partie (inchangé), Halo 5 = type de jeu
(`game_variant`, nom traduit). Les pages lisent ce champ sans repli propre ; les replis
existants côté consommateur (« PairMode sinon GameVariant ») disparaissent, et un garde-rail
empêche d'en réécrire.

Critère : sur la base réelle, la Synthèse Halo 5 affiche « Par mode » (noms lisibles, pas
d'identifiant) ; le filtre Modes de l'Explorateur Halo 5 propose ces modes et filtre
correctement ; Halo Infinite est strictement inchangé (mêmes réponses d'API sur les pages
touchées, à l'octet hors champs volatils, pour un joueur témoin).

## Décisions (user, 2026-10-26)

- **D1 — Pas de capability** : Halo 5 a des modes ; c'est la source du titre qui dit où le
  trouver. Aucun branchement sur le titre dans les pages.
- **D2 — Correction à la source de LECTURE** (`platform/duckdb/halo5/halo5_match_history_source.go`,
  `projectH5MatchSummary`), pas dans l'ingestion : `games/halo_5/mapping.go` et
  `games/halo_5/ingest/` ne changent pas (sinon le type de jeu serait écrit dans les colonnes
  `pair_*` de la base partagée).
- **D3 — Le nom du champ reste `PairMode`** ; son commentaire de contrat devient « mode du match,
  fourni par la source du titre ». Le renommer serait un refactor transverse hors de ce plan
  (à noter en Découvertes si jugé utile).
- **D4 — La source du mode est déclarée au manifeste du titre (2026-10-26, demande du pilote, « un plus haut
  niveau »)** : clé `[title].mode_source` de `config/titles/{slug}/title.toml` (`"pair"` par défaut, clé absente ;
  `"game_variant"` pour Halo 5), validée par `domain/title/config_loader.go` (valeur inconnue = erreur de chargement),
  portée par `TitleDescriptor.ModeSource` (type `title.ModeSource`, interne : `json:"-"`, pas de changement d'API).
  Elle remplace la table Go `analysis.modeSourceByTitle` (supprimée avec `slugHalo5`). Injection : `config`
  (`buildPoolConfig`, `resolveDemoPlayer`) lit `title.DefaultRegistry().ModeSourceFor(slug)` dans
  `PlayerPoolConfig.ModeSource`, copié dans `PlayerDB.ModeSource` ; `PlayerMatchesRepo` lit `pdb.ModeSource`. Écart assumé avec
  le modèle `CSRSeasonID` (résolu par service dans `api/wire`) : `PlayerMatchesRepo` est construit en deux endroits
  (`NewCachedPlayerMatchesRepo` via wire, et `SquadV2LoaderAdapter.LoadFor` pour les coéquipiers) et dans six fichiers de
  test ; porter la source sur `PlayerDB`, comme `TitleSlug`, couvre tous les appels sans changer de signature. La source
  Halo 5 la reçoit de wire (`td.ModeSource`, constructeur à 3 paramètres). Garde-rail :
  `archlint/title_slug_literal_test.go` interdit les littéraux de slug dans `analysis/` et `platform/duckdb/`
  (allowlist datée des 7 sites existants, dont `campaign_exclusion.go`, non migrés).

## Points de départ relevés (2026-10-26, à re-vérifier en E0)

- Source H5 : `PairMode` construit depuis `pair_id/pair_name/pair_name_fr` du registre, vides en
  Halo 5 ; `GameVariant` construit depuis `game_variant_id/game_variant_name` (nom souvent NULL,
  aucun FR).
- Repli côté consommateur existant : `analysis/home_canonical.go` `modeLabels` (PairMode puis
  GameVariant), `home_canonical_recent.go`, `home_canonical_sessions.go` ; traductions des noms
  de type de jeu : `platform/duckdb/home_repo_translations_canonical.go`
  (`asset_translations`, `asset_type='game_variant'`) — à ce jour appliquées par l'Accueil.
- Consommateurs de `PairMode` (hors tests) : `analysis/home_canonical.go`,
  `analysis/stats_canonical.go`, `games/halo_5/ingest/registry.go`,
  `platform/duckdb/home_repo_translations_canonical.go`, `platform/duckdb/player_matches_cache.go`,
  `service/squadagg/aggregates.go`, `service/squadagg/intersect.go`, `service/synthesis_service.go`,
  `service/synthesis_service_legacy.go`, `service/tactical_service_cellule_enrichir.go`,
  `service/timeseries_service_sections.go`.

## Étapes

### E0 — Audit sur pièces (lecture seule ; bases : COPIES uniquement)
- [x] Pour chacun des consommateurs listés : ce qu'il fait de `PairMode` et ce que produit une
      valeur Halo 5 (« Slayer », « Capture the Flag »…) : normalisation (`NormalizeModeLabel`,
      skill `halo-modes`), chaîne de performance, regroupement, filtre. Tableau dans Mesures.
      Tout consommateur qui casserait ou fausserait = item ajouté à E2. (Tableau en Mesures ; 3 replis
      à supprimer, 1 traduction à adapter, 0 consommateur qui casse : items en E2.)
- [x] Filtres côté SQL : toute requête qui filtre ou groupe par `pair_name` / `pair_id` (filtre
      Modes de l'Explorateur, `/filters/resolve`, cascades d'escouade) — que se passe-t-il pour
      Halo 5 ? Lister fichier:ligne. (Liste en Mesures : aucun filtre SQL de mode sur la voie
      canonique ; la famille « ligne brute » filtre en Go par `ResolveModeUIWithVariant`.)
- [!] Noms : sur une copie de `data/titles/halo_5/warehouse/metadata.duckdb` (+ shared), part des
      `game_variant_id` des matchs de Chocoboflor ayant un nom EN et FR dans `asset_translations`.
      Si moins de 95 % des matchs ont un nom : ARRÊT, relevé au user.
      NON MESURÉ : la métadonnée H5 est tenue en exclusivité par le serveur local (PID 17420), aucune
      sauvegarde ni copie n'existe (détail et requête de mesure en Mesures). Seuil de 95 % non évalué, donc
      non déclenché ; à mesurer par le user AVANT E4. Le code face à un nom absent ne montre aucun
      identifiant (testé).
- [x] Où brancher la traduction : la source H5 peut-elle joindre `asset_translations` dans sa
      requête (même base attachée ?) ou faut-il passer par l'enrichissement existant ? Décision
      écrite ici avec sa raison ; une seule voie, pas de copie de la cascade. (Enrichissement existant,
      choix du dictionnaire par `Kind` : voir Mesures, « Décision : voie de traduction ».)
- Gate : section Mesures remplie, aucune case ouverte. (Remplie ; une case `[!]` justifiée, la mesure
  des noms, faute d'accès à la métadonnée H5.)

### E1 — La source Halo 5 fournit le mode
- [x] `projectH5MatchSummary` : `PairMode` = type de jeu (id, nom EN, nom FR) quand le registre n'a
      pas de pair ; un pair présent garde la priorité. (Fait dans les DEUX sources par un seul helper,
      `duckdb.ModeAssetReference` + `TitleDescriptor.ModeSource` (D4) : `projectH5MatchSummary` ET la source
      générique `projectMatchSummary` qui sert réellement les pages, cf. Mesures. Noms EN/FR posés par
      l'enrichissement existant. Source du mode : D4, manifeste du titre, plus de table Go.)
- [x] Commentaire de contrat de `canonical.MatchSummary.PairMode` réécrit (mode du match, fourni
      par la source du titre ; Halo Infinite : nom de partie ; Halo 5 : type de jeu)
- [x] Test de la source (DuckDB `:memory:` ou fixtures existantes) : match H5 sans pair -> PairMode
      = type de jeu avec ses libellés ; match avec pair -> inchangé ; variante sans nom -> pas
      d'identifiant brut affiché (comportement aligné sur la règle existante des assets).
      (`mode_reference_test.go`, `player_matches_mode_test.go` [chaîne complète H5 et Infinite],
      `home_repo_mode_variant_test.go`, `halo5_match_history_source_test.go`, `domain/title/config_loader_test.go`
      [clé `mode_source` : absente, valeurs valides, valeur inconnue refusée, manifeste réel de Halo 5].)
- Gate : `go test ./internal/platform/duckdb/... ./internal/games/...`
  Passé : `go test ./internal/platform/duckdb/... ./internal/games/... ./internal/analysis/` vert (exit 0) ;
  `go test -tags=integration ./internal/platform/duckdb/ ./internal/platform/duckdb/halo5/` vert (exit 0, 480 s).

### E2 — Les pages héritent, les replis disparaissent
- [x] Supprimer les replis « PairMode sinon GameVariant » devenus inutiles (`modeLabels` et
      équivalents relevés en E0), avec leurs tests propres ; garder ce qui sert à autre chose
      (ex. `GameVariantID` exposé à l'API). (Supprimés : `analysis.modeLabels` [repli `GameVariant`, et
      repli quand PairMode est présent sans libellé : 0 ligne concernée sur les 10 026 matchs Infinite],
      `squadagg.modeLabelForFilter`, `squadagg.buildSharedMatch`. Tests retirés : `TestModeLabels_FallbackGameVariant`,
      `TestModeLabels_PairModeEmptyFallbackGameVariant`, `TestFilterRowsByCascade_ModeGameVariantFallback`
      [remplacés par des tests « aucun repli »]. Gardés : `GameVariantID/Name` de l'API, `variantNameOf`
      [clé `rounds_decide`], clé du modèle d'écart attendu, `session_page_service.go:684` [repli de la
      famille « ligne brute », sert Infinite : `pair_name_fr` NULL sur 10 026 matchs].)
- [x] Corriger les consommateurs signalés en E0 (items ajoutés ici, un par consommateur). E0 n'a
      relevé aucun consommateur qui casse ou fausse ; deux adaptations nécessaires, un item chacune :
  - [x] `HomeRepo.EnrichCanonicalAssetTranslations` : un mode porté par le type de jeu se traduit par
        les dictionnaires des types de jeu (choix par `Kind`), plus de cascade `pair`/`mode_name_tr` pour lui
        (fait en E1, testé).
  - [x] Nom de type de jeu à préfixe technique (« Conteneur:Mode ») : normalisé UNE fois à l'enrichissement
        (`normalizeGameVariantMode`), sans quoi le filtre de modes de l'Escouade (compare le libellé brut aux
        options normalisées) ne retrouverait plus les matchs Halo 5 que l'ancien repli normalisait.
        Alternative écartée : normaliser dans `modeLabelForFilter` (changeait le résultat Infinite : des libellés de
        repli non résolus, ex. « Arena:Legacy Slayer BR on Narrows », ne retenaient plus leurs matchs).
  - [x] Fixture `mockHomePlayerMatches` (`service/home_service_test.go`) : posait le mode sur `GameVariant`
        (l'ancien repli) ; elle le pose maintenant sur `PairMode`, comme le fait la source de lecture.
- [x] Filtres SQL relevés en E0 : Halo 5 doit filtrer sur la même clé que celle affichée (sinon
      un mode proposé ne renvoie aucun match). (Constat, rien à corriger : aucun filtre SQL de mode sur la
      voie canonique ; options et filtrage de l'Explorateur et d'`/filters/resolve` passent par la même
      fonction `ResolveModeUIWithVariant` ; le filtre Escouade lit `PairMode` des deux côtés. Liste en Mesures.)
- [x] Garde-rail `internal/archlint` : interdit de lire `Summary.GameVariant` comme repli de mode
      (motif à définir sur pièces, allowlist datée et justifiée pour les usages légitimes).
      (`archlint/mode_gamevariant_fallback_test.go` : AST, tout accès `<expr>.Summary.GameVariant` hors
      allowlist de 6 fichiers [compte exact par fichier, raison datée]. Auto-preuve sur extraits.) Et, par D4,
      `archlint/title_slug_literal_test.go` : aucun littéral `"halo_5"` / `"halo_infinite"` dans `analysis/` et
      `platform/duckdb/` hors tests, hors allowlist datée de 7 sites existants.
- [x] Tests service : Synthèse Halo 5 -> `top_modes` non vide, noms lisibles ; Accueil Halo 5 ->
      mode dominant inchangé par rapport à avant (même valeur qu'avec l'ancien repli).
      (`service/synthesis_service_modes_titre_test.go`, `analysis/home_canonical_mode_titre_test.go` [compare à
      `assetLabels(GameVariant)`, ce que rendait l'ancien repli], `squadagg/mode_du_match_test.go`,
      `platform/duckdb/player_matches_mode_test.go` [chaîne de lecture complète, H5 et Infinite].)
- Gate : `go test ./internal/...` (suite Go complète, chaîne CGO) + `golangci-lint` sans nouvelle
  entrée de baseline + `go test ./internal/archlint/`.
  Résultat : voir « Gates » en Mesures (suite complète verte sauf `TestReleaseNotesDocs_NewestBlockMatchesChangelog`,
  rouge sur la base `release/v8.1.0` avant tout changement : docs non touchées).

### E3 — Non-régression Halo Infinite et contrat
- [x] Joueur témoin JGtm (Halo Infinite) : réponses API Synthèse, Accueil, Explorateur (briefing),
      Séries temporelles comparées avant/après sur le même jeu de données (serveur local sur
      COPIE ou tests de contrat existants) : identiques hors champs volatils. Méthode écrite ici.
      (Méthode et résultat en Mesures, « E3 ». Identique à l'octet sur 16 des 17 sections ; la 17e,
      les tuiles de faits marquants, est non déterministe d'une exécution à l'autre dans les DEUX arbres.)
- [x] `make openapi-gen` : aucun changement attendu ; s'il y en a un, le justifier. (`go run ./cmd/openapi-gen
      -check` : « api/openapi.yaml est à jour » ; aucun type d'API touché.)
- [x] Web : aucun changement attendu ; `npm run typecheck` et `npm run test:run` verts. (`npm ci` puis
      `tsc -b` exit 0 ; vitest : 1006 fichiers passés, 10 383 tests passés, 23 ignorés ; aucun fichier web modifié.)
- Gate : relevé écrit, tests verts. (Relevé en Mesures.)

### E4 — Vérification visuelle (avec le user) et clôture
- [x] Halo 5 (Chocoboflor, puis JGtm) : Synthèse « Par mode », filtre Modes de l'Explorateur,
      Accueil ; Halo Infinite (JGtm) : mêmes pages inchangées (2026-10-27, serveur relancé depuis la copie
      principale passée sur `release/v8.1.0` : Synthèse H5 Chocoboflor 12 modes à l'écran avec courbe, JGtm 13 ;
      filtre Modes 12 options ; Accueil H5 nomme les modes des sessions ; Synthèse Infinite JGtm 21 modes, inchangée.
      Premier essai faux négatif : le serveur lisait `config/` d'une copie restée en v8.0.7, d'où le log
      `mode_source` au boot, 47f3e6c4a)
- [x] Changelog EN/FR : une ligne dans la section `[8.1.0]` existante (Fixed), plus une puce au bloc v8.1 des
      notes de version (EN/FR)
- [x] `.ai/thought_log.md` : entrée Complété
- [x] Skill `delivery-checklist` ; commit et fusion dans `release/v8.1.0` sur accord du user (accord du 2026-10-26,
      fusion a2a369f2b puis 9240e1c4a ; non poussé)
- Gate : validation visuelle du user + checklist verte.

## Reprise de session

Lire ce fichier (cases cochées = fait), `git log --oneline -10` sur la branche, puis l'entrée du
thought log qui le cite. Reprendre à la première case vide de l'étape la plus basse.

## Mesures

### E0 — relevé du 2026-10-26 (bases : copies sous %TEMP%\modes-titre, supprimées en fin de session)

**Constat qui amende D2 (premier fait de l'audit).** `projectH5MatchSummary` (source H5,
`Halo5MatchHistorySource`) ne sert AUCUNE page : seul `LoadMatchSummaries` de l'adaptateur H5 la lit, et
personne n'appelle `LoadMatchSummaries` hors interface (grep, 0 appelant de production). Toutes les pages
canoniques (Synthèse, Accueil, Séries temporelles, Escouade, Tactique) lisent `PlayerMatchesRepo.Load`
(`platform/duckdb/player_matches_*.go`) via `PlayerMatchesAdapter` puis `HomeRepo.EnrichCanonicalAssetTranslations` :
une source de lecture GÉNÉRIQUE, la même pour les deux titres. Corriger seulement `projectH5MatchSummary`
n'aurait changé aucune page. La règle (« la lecture, pas l'ingestion ») est tenue : on corrige à la
source de lecture qui sert réellement les pages, et `projectH5MatchSummary` applique la même règle par le
même helper (une seule règle, pas deux copies).

**Seconde famille de consommateurs (non listée au plan).** Les lignes « brutes » (`MatchHistoryRawRow`,
`FilterMatchRow`, pages Explorateur, filtres, briefing, patterns, événements de séries) résolvent déjà le
mode par la convention centralisée `analysis.ResolveModeUIWithVariant` (pair sinon type de jeu), alimentée par
des requêtes SQL qui lisent `game_variant_*`. Options et filtrage de l'Explorateur passent par la même fonction
(`filters_service.modeUI`, `match_history_service_filters.go:226,353`) : une option proposée filtre donc
correctement, pour Halo 5, sans changement. Cette famille est hors du champ de ce plan (type de ligne
différent, ~12 requêtes SQL) : voir Découvertes.

**Tableau des consommateurs de `Summary.PairMode` (voie canonique), effet d'une valeur Halo 5 et décision**

| Consommateur | Ce qu'il fait de PairMode | Effet d'une valeur H5 (« Slayer », « Capture the Flag »...) | Décision |
|---|---|---|---|
| `analysis/home_canonical.go:96` `modeLabels` (+ `home_canonical_recent.go:79`, `home_canonical_sessions.go:215`) | libellés EN/FR, puis repli `GameVariant` | `NormalizeModeLabel` via `normalizeHomeModeLabel` : un nom sans `:` ni ` on ` passe tel quel ; mode dominant identique (mêmes libellés que le repli) | E2 : supprimer le repli |
| `analysis/stats_canonical.go:154` | recopie dans `PairName/PairNameFR` de la ligne brute ; `GameVariantName(FR)` recopiés à part | `ResolveModeUI(PairNameFR)` normalise ; plus de dépendance au repli `session_page_service.go:684` pour H5 | aucun changement ; le repli de `session_page_service` reste (sert Infinite : `pair_name_fr` toujours NULL, 0 ligne remplie sur 10026) |
| `platform/duckdb/home_repo_translations_canonical.go` | traduit PairMode via `asset_type='pair'` et `mode_name_tr` | l'identifiant d'un type de jeu cherché sous `pair` : aucune traduction ; cascade `ResolvePairNameFR` inadaptée | E1 : traduire par le type d'asset de la référence (voir décision) |
| `platform/duckdb/player_matches_cache.go:104` | clone la référence | `Kind`/`Labels` clonés tels quels | aucun |
| `service/squadagg/aggregates.go:333` `modeLabelForFilter` | PairMode, sinon `GameVariant` normalisé | identique au repli | E2 : supprimer le repli |
| `service/squadagg/intersect.go:105` | PairMode, sinon `GameVariant` | identique au repli | E2 : supprimer le repli |
| `service/synthesis_service_legacy.go:229` `breakdownModeLabel` | FR sinon EN puis `NormalizeModeLabel` ; sert « Par mode » (`top_modes`) | c'est ici que H5 est VIDE aujourd'hui ; avec une valeur : lignes de mode | aucun code (hérite) ; test E2 |
| `service/synthesis_service.go:331` | compteur de log `pair_fr_count` | aucun effet fonctionnel | aucun |
| `service/tactical_service_cellule_enrichir.go:162` | `NormalizeModeLabel(labelPourLocale)` | H5 obtient un mode (vide aujourd'hui) | aucun code (hérite) |
| `service/timeseries_service_sections.go:283` | `MatchMeta.ModeLabel` des formes d'escouade | H5 obtient un mode (vide aujourd'hui) | aucun code (hérite) |
| `games/halo_5/ingest/registry.go:18-27` | ÉCRIT `pair_*` depuis le résumé issu du mapping API | le résumé vient de `mapping.go` (PairMode nil), jamais de la source de lecture | aucun (D2 respectée : pas d'écriture du type de jeu en `pair_*`) |
| `service/squad_service_v2_history.go:97` | `ModeLabel = GameVariant.DefaultLabel` (les deux titres) | n'est pas un repli pair/type de jeu : usage propre | aucun (hors périmètre) |
| `service/teammates/teammates_squad_expected_fda.go:61` | clé de modèle = nom du type de jeu | usage propre | aucun |
| `analysis/home_canonical_recent.go:260` `variantNameOf` | clé de `[rounds_decide]` = nom du type de jeu | usage propre | aucun |

**Filtres SQL par `pair_name` / `pair_id` (fichier:ligne) et effet pour Halo 5**
- `analysis/match_filter.go:140` (catégories de mode par préfixes de `pair_name`) : concept propre à Infinite
  (`InferModeCategoryFromPairName`) ; pour un titre sans catégories le filtre est ignoré (`ignored`). Aucun changement.
- `platform/duckdb/career_repo_highlights.go:58` (`ModeRawSources`, page Carrière) : options et filtre tous deux
  sur `pair_name` ; Halo 5 n'a pas d'options de mode sur cette page, donc rien ne se propose qui ne filtre pas.
- `platform/duckdb/player_matches_repo.go:182,191,300`, `queries_home_citations.go:64,344`, `queries_match.go:130`,
  `csr_coverage_repo.go:186` : détection « classé » / BTB par motif de `pair_name`, sans rapport avec le libellé de mode.
- `platform/duckdb/prestige/prestige_squad_match_provider.go:244`, `queries_career*.go`, `queries_squad.go:84-135`,
  `queries_match.go`, `media_repo_*.go`, `leaderboard_repo.go:127`, `match_exclusion_repo.go` : lectures de `pair_name`
  pour affichage ou identification ; la famille « ligne brute » résout le mode par `ResolveModeUIWithVariant` (ci-dessus).
- Aucun filtre SQL de mode sur la voie canonique : l'item « filtres SQL » de E2 est un constat, pas une correction.

**Noms (item « moins de 95 % ») : NON MESURABLE, bloqué.**
- Base partagée H5 copiée (3 032 matchs ; `game_variant_id` renseigné sur 3 032, `game_variant_name` NULL sur 3 032,
  `pair_*` vide sur 3 032).
- Chocoboflor : 1 855 matchs, dont 149 Campagne (masqués à la lecture) ; 1 706 matchs sur 12 types de jeu ; le premier
  (`257a305e...`) porte 1 500 matchs (88 %), les 11 autres 206.
- Traductions : `data/titles/halo_5/warehouse/metadata.duckdb` est tenue en exclusivité par le serveur local (PID 17420,
  démarré 19:42) : copie refusée par le système (Copy-Item, flux en partage lecture/écriture, robocopy /B, 6 essais sur
  plusieurs minutes). Pas de sauvegarde de la métadonnée H5 sous `data/backups/` (les `metadata.duckdb` sauvegardées sont
  celles d'Infinite) ; `metadata-prebuilt.zip` contient une `asset_translations` VIDE. Arrêter le serveur ou lire la prod
  est hors de ce que cette étape m'autorise. La part de matchs nommés n'est donc pas chiffrée ici ; le comportement du code
  face à un nom absent est testé (aucun identifiant affiché).
- Mesure à faire par le user (serveur arrêté quelques secondes, ou sur la copie de prod) avant E4 :
  `duckdb <copie metadata H5> "SELECT lang, COUNT(*) FROM asset_translations WHERE asset_type='game_variant' AND asset_id IN (<12 identifiants ci-dessus>) AND COALESCE(name,'')<>'' GROUP BY 1"` : 12 identifiants x `en-US` et `fr-FR` attendus ;
  couverture en matchs = somme des matchs des identifiants nommés / 1 706.

**Décision : voie de traduction.** Une seule voie : l'enrichissement existant
(`HomeRepo.EnrichCanonicalAssetTranslations`, appelé par `PlayerMatchesAdapter` pour toutes les pages). Il résout déjà
les noms EN/FR des types de jeu (`asset_type='game_variant'`, dictionnaires `variantNames`/`variantNamesEN`) sur
`Summary.GameVariant`. La source de lecture pose `PairMode` comme une COPIE de la référence du type de jeu, avec
`Kind = "game_variant"` ; l'enrichissement choisit le dictionnaire d'après `Kind` (type de jeu : dictionnaires des
types de jeu, sans cascade `mode_name_tr`/`pair`). Pas de jointure dans la requête (la métadonnée est une base séparée
de la base partagée : `r.pdb.Metadata` vs `v_match_full`), pas de seconde cascade, `GameVariant` inchangé (identifiant
exposé à l'API, clé de `[rounds_decide]`). La règle « où est le mode » est une donnée par titre
déclarée au manifeste (`mode_source`, D4) : pas de capability (D1), aucune comparaison ni littéral de slug dans les pages.

### E3 — non-régression Halo Infinite : méthode et résultat (2026-10-26)

**Contrainte** : aucun serveur ni navigateur (ports tenus par d'autres sessions, le :8000 tient les bases en
écriture). Méthode retenue : un harnais Go jetable (non versionné, supprimé), lancé sur COPIES, qui rejoue la
chaîne réelle « lecture canonique -> pages » et sérialise tout en JSON ; lancé sur l'arbre d'AVANT (export
`git archive` de HEAD `6d3cfa8ca`, sans mes changements) et sur l'arbre d'APRÈS, mêmes copies, mêmes commandes.
- Copies (sous `%TEMP%\modes-titre`, supprimées en fin de session) : base partagée Infinite copiée en direct (2,5 Go,
  10 026 matchs) ; base du joueur JGtm Infinite : la live est verrouillée, SAUVEGARDE du 07/10
  (`data/backups/avant_rattrapage_2026-10-08/players/JGtm`, 1 182 matchs lus) ; métadonnée Infinite : la live est
  verrouillée, SAUVEGARDE du 02/10 (`avant_rattrapage_2026-10-08/metadata.duckdb`). Mêmes entrées avant et après.
- Chaîne rejouée (même pool, `GetOrOpen`, topologie réelle) : `CachedPlayerMatchesRepo.LoadPlayerMatches` (enrichissement
  inclus) + 2e enrichissement (comme Synthèse et Accueil) puis : lignes canoniques complètes (`rows`), Accueil
  (`BuildRecentMatchesWithFavoritesFromCanonical`, `BuildSessionSummariesFromCanonical` solo et escouade,
  `BuildHighlightsFromCanonical`, FR et EN), Synthèse (`buildBreakdownsFromCanonical` : cartes et modes),
  Séries temporelles (`timeseriesFormesMetas`, FR et EN), Tactique (libellé de mode par match, FR et EN), filtre de
  modes de l'Escouade (`FilterRowsByCascade`, matchs retenus pour chacun des 44 libellés de mode).
- Explorateur / briefing : ses modules « dimensions » et le filtre de modes sont bâtis sur les lignes brutes
  (`MatchHistoryRawRow`, SQL + `ResolveModeUIWithVariant`), aucun fichier de cette famille n'est modifié ; son socle de KPI
  lit les lignes canoniques, dont la sérialisation est identique à l'octet (section `rows`).
- Résultat : 6 exécutions de l'arbre d'après et 6 de l'arbre d'avant. 16 sections sur 17 sont IDENTIQUES à l'octet entre
  les 12 exécutions (`rows`, `count`, `forme`, `recent_fr/en`, `sessions_solo_fr/en`, `sessions_squad_fr/en`,
  `synthese_breakdowns`, `formes_fr/en`, `tactique_mode_fr/en`, `escouade_filtre_modes`). Forme : 1 182 lignes, 1 182 avec
  une partie comme mode, 0 sans mode, 0 mode de type de jeu. La 17e, `highlights_fr/en` (faits marquants de l'Accueil), est NON
  DÉTERMINISTE d'une exécution à l'autre dans chacun des deux arbres (jusqu'à 3 variantes en 6 exécutions de l'arbre d'avant :
  séries, pente, valeurs des tuiles « maîtrise », « par minute », « volume », « série » ; une tuile alterne entre deux
  sessions à égalité) : ce n'est pas un écart de ce plan ; les champs de mode de ces tuiles ne varient que dans la
  limite de ce tirage, des deux côtés.
- Premier essai écarté : la même comparaison avec un `modeLabelForFilter` qui normalisait le libellé changeait des
  résultats du filtre Escouade Infinite (matchs au libellé de repli non résolu, par ex. « Arena:Legacy Slayer BR on
  Narrows ») ; la normalisation a été déplacée à l'enrichissement du seul mode « type de jeu » (item E2).

**Halo 5 sur la forme réelle (arbre d'après seul, base partagée H5 copiée, Chocoboflor, 1 706 matchs hors Campagne)** :
1 706 lignes avec un mode, toutes de type de jeu, 0 partie. Les noms ne sont pas mesurables (métadonnée H5 inaccessible) :
avec la métadonnée préconstruite (dictionnaire vide) aucun libellé ne sort, aucun identifiant n'est affiché ; avec 12 noms
SYNTHÉTIQUES posés sur les 12 identifiants réels (copie jetable), la Synthèse produit 12 lignes « Par mode » totalisant
1 706 matchs (1 500 / 89 / 48 / 19 / 17 / 12 / 7 / 5 / 4 / 3 / 1 / 1), l'Accueil un mode dominant, les formes d'escouade un mode,
le filtre de modes de l'Escouade 12 valeurs qui retiennent chacune leurs matchs. Cela valide le tuyau, pas la couverture des
noms réels.

### Gates (2026-10-26)
- E1 : `go test ./internal/platform/duckdb/... ./internal/games/... ./internal/analysis/` vert ; `go test -tags=integration
  ./internal/platform/duckdb/ ./internal/platform/duckdb/halo5/` vert.
- E2 : `go test ./internal/...` (suite complète, chaîne CGO, 2 exécutions) : tous les paquets verts SAUF
  `internal/service` > `TestReleaseNotesDocs_NewestBlockMatchesChangelog` (« docs/RELEASE_NOTES.md : bloc le plus récent
  v8.0, première entrée de docs/CHANGELOG.md : v8.1 », idem FR) : rouge sur `release/v8.1.0` avant ce plan, docs non
  touchées, hors périmètre (Découvertes). `go test ./internal/archlint/` vert (inclut le nouveau garde-rail et
  `TestNoNewSlugComparison`). `golangci-lint run --new-from-rev=HEAD` : 0 issues (un premier passage, avant D4, avait levé `goconst` sur la table Go
  par titre, depuis supprimée). `gofmt -l` vide.
  `go test -tags=integration` ciblé (`Mode|Enrich|Halo5MatchHistory|PlayerMatches|Translation`) sur
  `platform/duckdb` et `platform/duckdb/halo5` : vert.
- E3 : `go run ./cmd/openapi-gen -check` : à jour ; `npm ci` + `npm run typecheck` (exit 0) + `npm run test:run`
  (10 383 tests passés, 23 ignorés) : vert. Non-régression Infinite : voir ci-dessus.
- Reste pour E4 (non fait ici) : mesure des noms H5 par le user, vérification à l'écran, changelog, `delivery-checklist`,
  commit. Caches de compilation dédiés à supprimer à la clôture du lot :
  `scripts/disk-hygiene.ps1 -Lot modes-titre -Apply`.

## Découvertes hors périmètre

- Synthèse : en interface anglaise, les noms de mode sont affichés en français
  (`breakdownModeLabel` lit le libellé FR) — constaté le 2026-10-26, pré-existant, non traité ici.
- **La source nommée en D2 ne sert aucune page** (constat E0) : `Halo5MatchHistorySource.GetMatchSummaries` n'a pas
  d'appelant de production (`LoadMatchSummaries` n'est appelée nulle part hors interface et tests). Le code mort
  éventuel (règle 7) et la capability `match.history` déclarée « supportée » pour H5 sur cette base sont à trancher ;
  non traité ici (la règle du mode y est appliquée pour que les deux sources ne divergent pas).
- **Famille « ligne brute »** : `analysis.ResolveModeUIWithVariant` (pair sinon type de jeu) reste utilisée par 7 sites
  (`filters_service.go:344`, `match_history_service_filters.go:226,353`, `match_history_service_briefing_dimensions.go:146`,
  `match_history_service_enrich.go:188`, `timeseries_service_events.go:243`, `patterns_repo.go:260`) et ses requêtes SQL
  lisent `game_variant_*` à côté de `pair_*` (`queries_squad.go:129-135`, `match_view_repo.go:210`,
  `match_history_fr_translations.go`). Même repli que celui supprimé ici, centralisé (une définition) donc non
  re-divergeant ; l'unifier avec le mode de la source exige que les lignes brutes portent aussi le mode (~12 requêtes).
  Conséquence : le critère « le filtre Modes de l'Explorateur H5 propose ces modes et filtre » était probablement déjà vrai
  avant ce plan ; à confirmer à l'écran en E4.
- `service/squad_service_v2_history.go:97` : `ModeLabel = GameVariant.DefaultLabel` pour les DEUX titres (l'historique
  Escouade d'Infinite affiche le nom du type de jeu, pas la partie) ; usage propre non traité, allowlisté au garde-rail.
- `TestReleaseNotesDocs_NewestBlockMatchesChangelog` est rouge sur `release/v8.1.0` (RELEASE_NOTES porte v8.0, CHANGELOG
  v8.1) avant tout changement de ce plan : docs non touchées ici ; à corriger par la session de la release.
- Faits marquants de l'Accueil non déterministes d'une exécution à l'autre (ex-æquo départagés par l'ordre d'une map Go) :
  mesuré en E3 sur l'arbre d'avant comme d'après ; un test de contrat qui les comparerait à l'octet serait instable.
- Données par titre écrites en Go, à migrer vers le manifeste comme `mode_source` (non migrées, D4) :
  `analysis/campaign_exclusion.go` (`campaignExcludedVariantIDs`, + son appel `SQLExcludeCampaignVariants("halo_5", ...)`
  dans `platform/duckdb/halo5`), `analysis/perfect_kills.go` (`perfectKillMedalIDs`), et les slugs par défaut de
  `analysis/world_stats.go`, `platform/duckdb/media_repo_translations.go`, `platform/duckdb/world_player_stats_repo.go`.
  Allowlistés au garde-rail `title_slug_literal_test.go`.
- Synthèse en anglais : voir ci-dessus. Pour Halo 5, `breakdownModeLabel` lit le libellé FR : l'anglais verra aussi des noms FR.
- Cartes Halo 5 : `map_name` NULL sur les 3 032 matchs du registre (déjà noté par la session « performance par carte »).
- Précision sur l'incertain : si les noms de type de jeu Halo 5 portent un suffixe de carte ou un préfixe inattendu, la
  normalisation (`NormalizeModeLabel`) les traite comme n'importe quel mode ; non vérifiable sans la métadonnée H5.
