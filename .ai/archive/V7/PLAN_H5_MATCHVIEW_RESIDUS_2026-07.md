# PLAN — Résidus Halo 5 match view (2026-07-07)

> **STATUT GLOBAL : COMPLÉTÉ (2026-07-12) — opérationnellement clos.** Chantier LOCAL
> clos le 2026-07-11 (lots A, B, C, D, F, E, gates verts) ; LOT V (volet prod) exécuté
> le 2026-07-12 lors de la salve post-deploy (merge PR #54 → main → prod ~09:25 UTC) :
> V2 rejoué sur le VPS (overrides Tidal → 48/48 maps résolues, 1002 lignes Placement,
> purge 84 clips média étrangers). Détail §6.
>
> **Restes utilisateur** (n'empêchent pas la clôture opérationnelle) :
> - V3 — vérification VISUELLE en prod par l'utilisateur (galerie média H5 + libellés +
>   filtres, clic média → match f88f6d8b dont l'association n'existe qu'en prod, matchs
>   témoins : map Tidal / mode / Dominance / rating Placement). Requête anonyme = 403
>   (ownership) → pas de repro authentifiée automatisable ; départage utilisateur.
>
> Branche du chantier local : `fix/h5-matchview-residus` (6 commits, poussée, CI verte).

> Exécution sous contrat du skill `plan-execution` (ordre strict, statuts obligatoires
> `[x]`/`[~]`/`[!]`, gates par lot, zéro fix hors périmètre — les découvertes vont en §8).
> Branche cible : `fix/h5-matchview-residus` (1 branche, commits par lot).
> Reprise de session : lire ce fichier §7 (tracker), puis le premier lot non clos.

## 1. Objectif et critère de succès

Résorber les 7 problèmes remontés le 2026-07-07 sur les matchs Halo 5 (match view).
Succès = les gates de chaque lot passent, et sur les 5 matchs témoins :
map, mode et playlist affichés correctement, graphe Dominance rempli, rating par match
présent (ou absence justifiée et affichée honnêtement), cards Résistance/Résultat attendu
conformes aux capabilities du titre, galerie média H5 avec libellés de match résolus.
Volet prod (V) : V1 diagnostiqué le 2026-07-07 (VPS revenu), V2-V4 après merge — §6.

## 2. Diagnostics prouvés (investigation du 2026-07-07, données locales)

Les 5 GUIDs utilisateur sont des `match_id` TOUS présents dans
`data/titles/halo_5/warehouse/shared_matches_v2.duckdb::match_registry`, avec
participants (2 équipes) et events. Import historique du 2026-06-23 (06h48-06h55,
first_sync_by=JGtm). Matchs témoins :

| match_id | playlist | map | classé | constat utilisateur |
|---|---|---|---|---|
| ccf64951… | Super Fiesta Party | d67fdcb9… (sans nom) | non | « pas de map », playlist « Fete » |
| 7e3fa711… | Slayer (FR : Assassin) | d67fdcb9… (sans nom) | oui | map/mode/LUSR absents |
| f88f6d8b… | Quick Play | Plaza | non | « introuvable » (média) |
| 14f762a2… | SWAT | Parallax | oui | incomplet, pas de LUSR |
| f6baea94… | Slayer | Eden | oui | incomplet, pas de LUSR |

1. **Map absente** : la map `d67fdcb9-6d9c-403e-960d-04202e19b244` = **Tidal** (canvas
   Forge océanique H5, identifié visuellement via l'image officielle halocdn du
   catalogue). Présente dans `maps_catalog` avec `name_canonical = ''` (l'API officielle
   ne nomme pas les canvas Forge dans /maps) et ABSENTE d'`asset_translations`.
   **Seul asset non résolu de tout le registre H5** — 129 matchs affectés.
2. **Playlist « Fête »** : reproduit en local (`playlist_label: "Fête"`).
   Traduction fr-FR OFFICIELLE de l'API 343 = « Super Fiesta Fête » (asset_translations),
   puis `analysis.NormalizePlaylistLabel` (documentée **spécifique Halo Infinite**,
   `playlist_label.go:2`) strippe le préfixe de catégorie « super fiesta » →
   il ne reste que « Fête ». Site : `match_view_repo.go:133` (seul call site).
   Touche aussi « Fiesta des fêtes » (Holiday Fiesta) → « Des fêtes ».
3. **Mode vide** (`mode_ui: ""`) : la résolution du mode passe par `pair_name` /
   `pair_asset_id` (`match_view_repo.go:141-169` + `ResolveModeUI`), or H5 n'a PAS de
   pair (`PairMode` nil dans l'adapter). Le `game_variant_id` est présent et SES
   traductions existent (`asset_translations` : 257a305e → Slayer/Assassin,
   a2949322 → Capture du drapeau) mais ne sont jamais utilisées pour le mode.
4. **Dominance vide** : `combat_tab.tug_of_war = []` alors que killer_victim_pairs
   est peuplé (87-109 paires/match, kd_timeline/killer_victim remplis). Cause :
   `durationMS = meta.PlayableDurationSeconds * 1000` (`match_view_data_loaders.go:229`)
   sans fallback, et `playable_duration_seconds` est **NULL pour les 3032 matchs H5**
   (0 NULL sur Infinite ; l'adapter H5 n'écrit jamais ce champ). `ComputeTugOfWar`
   retourne nil si durée ≤ 0 (`tug_of_war.go:38`). Dominance est donc vide sur
   **100 % des matchs H5**.
5. **« Pas de LUSR »** : les 3 matchs concernés sont CLASSÉS → l'affichage attend un
   CSR par match (`resolveMatchRatingType` : ranked → CSR, social → LUSR).
   - JGtm : 1306 classés dont **115 sans ligne CSR** ; 664 sociaux dont 190 sans LUSR.
     Madina : 114/125 ; Chocoboflor : 107/218 ; XxDaemonGamerxX : 52/56.
   - CSR manquants : le backfill (23-24/06) a couru APRÈS l'import — matchs dans le
     périmètre mais sautés en silence (`csr_match.go` : carnage KO → `continue` sans
     log ; gamertag introuvable ; `CurrentCsr` nil). Le taux de manque explose sur les
     années de jeu sporadique (2020 : 23/29, 2021 : 9/11, 2023 : 3/3) → forte
     présomption de **matchs de placement** (CurrentCsr null tant que non classé,
     ~10 placements/saison mensuelle H5). À départager par la sonde D2.
   - LUSR sociaux manquants (190 JGtm) : import ANTÉRIEUR au recalcul du 26/06 →
     sautés par les FILTRES du modèle, pas par le watermark : 123 matchs mono-équipe
     (customs), 29 FFA/multi-équipes (3+), 38 à 2 équipes (déséquilibre/outcome).
     Largement PAR DESIGN (modèle 2-équipes équilibrées).
   - `match_csrs`(_latest) shared : 0 ligne (table inutilisée en H5 — le CSR par match
     vit dans les player DBs `match_skill_rank`).
6. **« Introuvable » f88f6d8b** : IRREPRODUCTIBLE en local — HTTP 200 complet (Plaza,
   Partie rapide, Victoire 50-49), présent dans mv_player_matches + enrichment JGtm.
   Aucune association média locale (0 ligne dans les shared_social des 2 titres) :
   l'observation vient de la PROD (médias indexés sur le VPS). Bloqué VPS → lot V.
7. **Card Résistance** : la capability `damage_taken` est bien ABSENTE de
   `config/titles/halo_5/mappings/capabilities.toml` (comportement backend correct,
   DefensiveResistance nil) mais la card est RENDUE avec « N/A »
   (`MatchStatCards.tsx:489-494`). Le pattern de masquage existe déjà dans le même
   composant (card MMR : `providesTeamMmr && …`, ligne 435).
8. **Card Résultat attendu** : SUPPORTÉE pour H5 — `expected_win_prob` est calculé par
   LUSR v2 et rempli sur les matchs sociaux (JGtm H5 : 662/662 lignes LUSR avec ewp ;
   témoin ccf64951 : 0.446 servi par l'API). Jamais rempli sur les lignes CSR — sur les
   DEUX titres (Infinite : 951/951 LUSR, 0/8 CSR). La card grisée « — » apparaît donc
   sur tout match sans ligne LUSR (classés notamment). Comportement structurel, pas un
   bug H5 ; amélioration UX possible (masquer si null).

### Complément VPS (investigation prod du 2026-07-07 soir, copies /tmp → local)

9. **Prod = local** : match_registry H5 prod identique (3032 matchs, même taille de
   fichier, les 5 témoins présents, noms NULL pareil) ; JGtm prod = mêmes ratings
   (662 LUSR / 1003 CSR), f88f6d8b présent dans mv_player_matches + enrichment + LUSR.
   L'import historique du 2026-06-23 EST en prod. Tous les fixes/backfills locaux
   s'appliquent tels quels en prod.
10. **« Introuvable » f88f6d8b — cause identifiée** : le média existe en prod
    (media_files id=82, `JGtm/Halo_5_Guardians-2019-12-12_22h27.mp4`, association
    active vers f88f6d8b créée le 2026-06-25, delta 108 s). MAIS le pipeline média Q37
    résout les libellés match UNIQUEMENT depuis les colonnes du registre
    (`media_repo_q37_enrich.go:22-67` : map_name_fr/map_name, pair_name*,
    playlist_name*) — toutes NULL pour les 3032 matchs H5. La card média affiche donc
    « Carte inconnue »/aucun libellé (cf. tests MediaViewer) : le match paraît
    introuvable alors que le lien fonctionne (match view = HTTP 200 prouvé en local à
    données identiques). Les filtres map/mode/playlist de la galerie H5 sont vides
    pour la même raison. Le trafic d'origine de l'utilisateur n'est plus en logs
    (conteneur recréé au redéploiement) ; requête anonyme = 403 (ownership), pas de
    session testable — confirmation visuelle finale par l'utilisateur au merge.
11. **Double indexation cross-titre** : les MÊMES 84 clips H5 (fichiers
    `{captures}/JGtm/Halo_5_Guardians-*.mp4`) sont indexés dans les shared_social des
    DEUX titres (H5 : associés ; Infinite : id 307 etc., AUCUNE association possible).
    Structurel : `IndexMedia` scanne le répertoire captures du joueur (partagé entre
    titres) sans filtre de titre → chaque titre indexe tout. Sous Infinite ces clips
    polluent la section « Sans match » à perpétuité.

## 3. Décisions produit (tranchées AVANT exécution — veto utilisateur possible)

- **DEC-1 Libellés playlists H5** : après le fix B1, l'UI affichera les noms FR
  officiels 343 (« Super Fiesta Fête », « Fiesta des fêtes », « Assassin »). On les
  GARDE tels quels (source officielle, pas de mécanisme d'override à créer).
- **DEC-2 Card Résistance** : masquée entièrement quand `damage_taken` non supporté
  (aligné sur le précédent card MMR), au lieu de « N/A ».
- **DEC-3 Card Résultat attendu** : masquée quand `expected_win_prob` est null
  (les 2 titres — une card grisée « pas d'estimation » n'apporte rien).
- **DEC-4 Matchs de placement (CSR null)** : si D2 confirme le placement comme cause
  majoritaire, écrire une ligne `match_skill_rank` rating_type='CSR' avec rating NULL
  et `tier_label='Placement'` pour affichage honnête (sinon simple justification `[!]`).
- **DEC-5 LUSR non éligibles** (customs mono-équipe, FFA/multi-team) : PAS de calcul
  forcé (modèle 2-équipes = design) ; absence de rating assumée, aucune UI dédiée.
- **DEC-6 Dominance** : fix côté LECTURE (fallback durée), PAS de backfill de
  `playable_duration_seconds` en base (pas de migration pour un champ dérivable).
- **DEC-7 Libellés média** : le pipeline média résout map/mode/playlist via la MÊME
  cascade asset_translations que le match view (réutiliser ResolveAssetNamesBulk,
  pas de nouvelle copie du pattern), avec fallback game_variant pour le mode H5.
  Pas de backfill des colonnes noms du registre (résolution à la lecture partout).
- **DEC-8 Double indexation cross-titre** : router l'indexation par préfixe de nom de
  fichier par titre (`Halo_5_Guardians-*` → halo_5 uniquement ; captures Infinite →
  halo_infinite) + purge one-shot des 84 copies étrangères du shared_social Infinite
  (via cleanup dédié, précédé d'un dry-run). Un clip H5 ne doit plus apparaître
  « Sans match » sous Infinite.

## 4. Lots (ordre d'exécution strict)

### LOT A — Lecture Go : playlist, mode, durée (fixes match view)  [COMPLÉTÉ 2026-07-11]
Périmètre fermé :
- [x] A1. `NormalizePlaylistLabel` rendu title-aware via capability `playlist.label.strip_category`
      (`games.CapPlaylistCategoryStrip`) : déclarée `supported` dans le TOML halo_infinite +
      la CapabilityMap hardcoded HI (parité), ABSENTE de halo_5. Câblage : `registry.newMatchViewRepo`
      lit `capabilitiesForPDB(pdb).Has(CapPlaylistCategoryStrip)` → `WithPlaylistCategoryStrip`.
      Aucun `slug ==`. Site d'appel : `match_view_repo.go` (bloc PlaylistAssetID). Vérifié réel :
      ccf64951 → "Super Fiesta Fête" (non tronqué) ; Infinite inchangé (b955bf2a "Partie rapide").
- [x] A2. Mode H5 : fallback data-driven en fin de `GetMatchMeta` — pair (ID+nom) absent ET
      `GameVariantAssetID` présent → mode via `asset_translations` type `game_variant`. Vérifié réel :
      7e3fa711 mode_ui="Assassin", ccf64951 "Capture du drapeau". Les 2 autres chemins :
      match history H5 = [~] servi par l'ADAPTER CANONIQUE (`canonicalModeUI` priorise game_variant +
      `enrichCanonicalDetailTranslations`), la voie repo `applyMatchHistoryFRTranslations` ne traite
      que le player DB local (0 ligne en H5) ; explorer = [!] lit les colonnes BRUTES du registre
      (`Q19c` : map_name/pair_name, TOUTES NULL en H5, pas seulement le mode) → hors périmètre
      match view (cf. §8 Découvertes).
- [x] A3. Dominance : helper `tugDurationMS(meta)` extrait dans `match_view_data_loaders.go` —
      playable prioritaire (Infinite inchangé), fallback `headerGameplayDurationSeconds`
      (duration−T0) sinon 0. Title-agnostic. Vérifié réel : tug_of_war 7e3fa711=18, ccf64951=13.
- [x] A4. Tests : `match_view_repo_meta_test.go` (playlist title-aware strip, mode via game_variant,
      non-régression pair présent) + `match_view_tug_duration_test.go` (5 cas fallback) +
      `capabilities_test.go` count 17→18 + parité TOML/hardcoded HI. go test + lint verts.
Gate LOT A (commandes exactes) :
```
cd apps/go-api && go test ./internal/platform/duckdb/... ./internal/service/... ./internal/analysis/...
make go-api-lint
# serveur local puis :
curl -s -H "X-LevelUp-Title: halo_5" localhost:8000/api/v1/players/JGtm/matches/7e3fa711-e2bd-410d-99a4-84694d1dabe9 \
  | jq '.header.mode_ui, .header.playlist_label, (.combat_tab.tug_of_war|length)'
# attendu : "Assassin", "Assassin" (playlist), > 0
curl -s -H "X-LevelUp-Title: halo_5" localhost:8000/api/v1/players/JGtm/matches/ccf64951-4d5c-408a-949d-1c71e9eb07a0 \
  | jq '.header.playlist_label, .header.mode_ui'
# attendu : "Super Fiesta Fête", "Capture du drapeau"
# non-régression Infinite : un match Infinite au hasard, mode_ui/playlist inchangés.
```

### LOT B — Metadata : nom de la map Tidal (data + garde-rail)  [COMPLÉTÉ 2026-07-11]
Périmètre fermé :
- [x] B1. Nouveau mécanisme d'override keyé par asset_id (les canvas Forge n'ont PAS de
      nom EN sur lequel keyer, contrairement à `[maps]`) : section `[[maps_by_id]]` dans
      `asset_labels_fr.toml` (id/en/fr), entrée d67fdcb9 = « Tidal » (EN+FR). Struct
      `mapIDOverride` + `frLabels.MapsByID`. `applyMapIDOverrides` (UPDATE name_canonical
      + upsert asset_translations en-US/fr-FR) appelé EN DERNIER dans main (idempotent,
      survit à un re-fetch qui réécrirait name_canonical vide).
- [x] B2. Seed local rejoué via nouveau mode `--overrides-only` (local pur, sans clé API
      ni réseau — évite de marteler l'API dont les tokens sont morts). Vérifié :
      name_canonical='Tidal', asset_translations en-US/fr-FR='Tidal' ; curl match view
      ccf64951 ET 7e3fa711 → map_ui='Tidal'.
- [x] B3. Garde-fou `logUnresolvedMaps` : ouvre le registre H5 (RO), WARN slog les map_id
      référencés sans nom résolu dans asset_translations. Après B2 : « toutes les maps du
      registre sont résolues count_registry_maps=48 » (le 1 non résolu = Tidal, corrigé).
Test : `TestApplyMapIDOverrides` (name_canonical + traductions + idempotence). Lint OK.
Gate LOT B :
```
cd apps/go-api && go run cmd/tmpdbq/main.go ../../data/titles/halo_5/warehouse/metadata.duckdb \
  "SELECT name_canonical FROM maps_catalog WHERE lower(map_asset_id)='d67fdcb9-6d9c-403e-960d-04202e19b244'"
# attendu : Tidal ; + curl match view ccf64951 → .header.map_ui == "Tidal"
# requête « assets non résolus » (celle de l'investigation) → 0 ligne
```

### LOT C — Cards front (Résistance / Résultat attendu)  [COMPLÉTÉ 2026-07-11]
Périmètre fermé :
- [x] C1. `MatchStatCards.tsx` : card Résistance masquée par `providesDamageTaken && (…)`
      (aligné card MMR) — DEC-2. Constante morte `DR_NA_LABEL` retirée (règle 0 code mort ;
      les autres surfaces combat-yield gardent leur propre constante, hors périmètre).
- [x] C2. `MatchWinProbCard` rendue seulement si `expected_win_prob != null && isFinite`
      (call site) — DEC-3. Composant simplifié (prop `winProb: number`, branche null/`—`/
      `opacity-50` supprimée) ; clé i18n morte `no_win_prob_data` retirée du TOML + régen.
- [x] C3. Tests vitest `MatchStatCards.test.tsx` (5 cas : Infinite → Résistance+Résultat+MMR
      présents ; H5 sans damage_taken → Résistance absente ; sans team_mmr → MMR absente ;
      winProb null → Résultat absent ; winProb présent → « 62 % »). check-types OK ;
      dossier match-view 112/112 verts.
Gate LOT C :
```
make check-types && make test-web
# vitest hors sandbox (dangerouslyDisableSandbox) — cf. memoire vitest
```

### LOT D — Ratings par match : instrumentation + backfill ciblé  [COMPLÉTÉ 2026-07-11]
Périmètre fermé :
- [x] D1. `PersistPerMatchRatings` retourne un `PerMatchRatingsSummary` : chaque skip
      compté ET loggé par raison (skip_registry / skip_carnage / skip_owner_absent /
      placement_csr_null / skip_persist) + bilan slog Info en fin de run. Paramètre
      restreint à l'interface `carnageGetter` (testable). Test integration
      `csr_match_summary_integration_test` (fake carnage, ventilation 6 cas).
- [x] D2. Flag `--missing-only` (classés sans ligne CSR de la player DB uniquement).
      Runs des 4 joueurs FAITS en local (auth_as=JGtm pour les 3 RT morts) —
      ventilation §7 : **1002/1002 = placement_csr_null, 0 carnage KO, 0 owner absent**.
- [x] D3. DEC-4 CONFIRMÉ à 100 % → `buildPerMatchCSRInsert(nil)` écrit une ligne
      « Placement » (tier=skill.TierLabelPlacement réutilisé, rating_value=0 NOT NULL,
      via PlayerPersister append-only). Affichage front : [~] déjà couvert —
      `buildRankBlock` (Go) gère isPlacement (pas de valeur/progress) et
      `MatchRankBadge` (web) affiche tier_label tel quel → « Placement » rendu sans modif.
- [x] D4. Témoins vérifiés : 7e3fa711 / 14f762a2-970b / f6baea94-e0e9 → ligne
      `CSR / Placement` dans match_skill_rank_latest ET `rank={rating_type:CSR,
      tier_label:Placement}` servi par l'API ; header complet (map/mode/playlist).
      Couverture finale : classés avec ligne = 1306/1306 (JGtm), 1100/1100 (Madina),
      893/893 (Chocoboflor), 219/219 (XxDaemonGamerxX).
Gate LOT D :
```
cd apps/go-api && go test -tags=integration ./internal/persist/... ./internal/games/halo_5/...
# (persist touché → suite integration OBLIGATOIRE, delivery-checklist)
go run cmd/tmpdbq/main.go "../../data/titles/halo_5/players/JGtm/stats.duckdb" \
  "SELECT rating_type, count(*) FROM match_skill_rank_latest GROUP BY 1"  # CSR > 1003
# comptage « classés sans ligne » par joueur : doit tendre vers 0 ou être justifié D3
```

### LOT F — Médias : libellés match + double indexation (issu du volet VPS)  [COMPLÉTÉ 2026-07-11]
Périmètre fermé :
- [x] F1. Fallback des noms au POINT DE CHARGEMENT (`loadMediaMatchRegistry` →
      `resolveMediaRegistryNameFallbacks`) via ResolveAssetNamesBulk (RÉUTILISÉ) :
      map par map_id, playlist par playlist_id, mode par game_variant_id (champ
      DISTINCT `ModeNameFallback` — pas un pair : n'alimente ni PairNameRaw ni la
      classification par catégorie Infinite). `enrichMediaModeCategories` ne nil-e
      plus le ModeName sans pair ; filtre mode par égalité de libellé quand pas de
      pair. Vérifié réel (serveur local, 7 clips H5 associés) : maps résolues
      (Truth/Coliseum/Eden/Tyrant/Alpin/Plaza), mode="Assassin", filtres map/mode/
      playlist PEUPLÉS ("Super Fiesta Fête", "Partie rapide", ...). Non-régression
      Infinite vérifiée réel (catégories Assassin/Super Fiesta/Other inchangées).
- [x] F2. Routage titre de l'indexeur : `[title].media_filename_prefixes` dans
      title.toml (halo_5 = ["Halo_5_Guardians-"]) → `TitleDescriptor.MediaFilenamePrefixes`
      + `Registry.ForeignMediaFilenamePrefixes(slug)` ; `IndexMedia` saute les fichiers
      matchant un préfixe ÉTRANGER (opts.TitleSlug câblé aux 5 call sites : scan,
      reindex, post-sync, upload — via UploadRequest.TitleSlug —, CLI index-media).
      Aucun slug ==.
- [x] F3. `cmd/cleanup_media_index --foreign-only [--title <slug>] [--dry-run]` :
      purge des media_files matchant un préfixe étranger (+ associations_history +
      likes) + CHECKPOINT (ADR 0022). Exécuté en LOCAL : dry-run = 84 fichiers /
      0 assoc / 0 like → purge réelle 84 → re-run 0 (idempotence prouvée).
      « Sans match » Infinite : 101 → 17. À REJOUER EN PROD (V2).
- [x] F4. Tests : `media_repo_h5_fallback_test.go` (4 tests integration : libellés
      résolus, options de filtres peuplées, filtre mode par label, non-régression
      noms présents) ; `TestMatchesForeignPrefix` (5 cas) ;
      `TestMediaFilenamePrefixes_ParsedAndForeign` (parse TOML + foreign) ; purge
      idempotente prouvée en exécution réelle (run + re-run → 0).
Gate LOT F :
```
cd apps/go-api && go test ./internal/platform/duckdb/... ./internal/ops/...
go test -tags=integration ./internal/ops/...   # écritures shared_social touchées
# serveur local avec copie prod de shared_social H5 :
curl -s -X POST -H "X-LevelUp-Title: halo_5" localhost:8000/api/v1/players/JGtm/pages/media \
  | jq '.table.items[] | select(.match_id=="f88f6d8b-7f05-43bf-91bf-4198a6ccee9f") | {map_name, mode_name}'
# attendu : map "Plaza", mode "Assassin" (plus de « Carte inconnue »)
```

### LOT E — Clôture chantier local  [COMPLÉTÉ 2026-07-11]
- [x] E1. Tous les items A→D+F statués (`[x]`/`[~]`/`[!]`), §7-§8 remplis.
- [x] E2. `.ai/thought_log.md` : entrées par lot + entrée de clôture.
- [x] E3. Skill `delivery-checklist` déroulé : go test ./... complet (0 FAIL, exit 0),
      go test -tags=integration -p 1 (persist+halo_5+ops+duckdb), go vet 0,
      golangci-lint --new-from-rev=origin/main ./... = 0 issue, tsc -b purgé,
      npm run lint 0 erreur, vitest 247 fichiers / 2106 tests verts, build Vite OK.
Gate : gate global VERT (tous les gates A→D+F re-passés dans la session + lint + types +
CI branche verte — le job Frontend rouge du commit LOT C a été corrigé, cf. fix(C/ci)).

## 5. Réponses aux questions utilisateur (TL;DR intégré au plan)

- « Playlist Fete, traduction de ton fait ? » → NON : « Super Fiesta Fête » est le nom
  FR officiel de l'API 343 ; c'est notre normaliseur Infinite qui le tronque en
  « Fête » (fix A1, DEC-1).
- « Résistance supportée ? » → Non pour H5 (pas de damage_taken dans l'API) : card à
  masquer (C1). « Résultat attendu supporté ? » → OUI sur les matchs sociaux (LUSR v2,
  prouvé : 0.446 servi sur ccf64951), structurellement vide sur les classés des DEUX
  titres (C2 masque la card dans ce cas).

## 6. LOT V — Volet prod

- [x] V1. Diagnostic prod FAIT (2026-07-07 soir, VPS revenu — lecture seule sur
      copies /tmp, nettoyées) : prod = local (registre 3032, ratings JGtm 662/1003,
      f88f6d8b complet côté joueur ; média id=82 associé actif). Cause « introuvable »
      identifiée = libellés média NULL (diagnostic n°10) + double indexation (n°11)
      → traités par le LOT F. Trafic d'origine perdu (conteneur recréé) ; requête
      anonyme 403 → pas de repro authentifiée possible, confirmation visuelle par
      l'utilisateur après déploiement.
- [x] V2. EXÉCUTÉ EN PROD le 2026-07-12 (salve post-deploy, après merge PR #54 → main →
      déploiement ~09:25 UTC). Les trois opérations rejouées sur le VPS :
      1. B2 (overrides metadata, `--overrides-only`, local pur) : 26 armes / 184 médailles /
         1 map par id (Tidal, d67fdcb9) appliqués → **48/48 maps du registre H5 résolues**
         (garde-fou `logUnresolvedMaps` : 0 map non résolue).
      2. D2 (`h5-csr-match-backfill --missing-only`, 4 joueurs) : **1002 lignes « Placement »
         écrites** — JGtm 303 / Madina97294 293 / Chocoboflor 277 / XxDaemonGamerxX 129 ;
         `placement_csr_null = 100 %`, **zéro skip** (0 carnage KO, 0 owner absent). DEC-4
         confirmé en prod comme en local.
      3. F3 (`cleanup_media_index --foreign-only`) : dry-run préalable = **84 fichiers exacts**,
         puis purge réelle de **84 media_files étrangers** + CHECKPOINT (ADR 0022).
- [!] V3. Vérification VISUELLE prod par l'utilisateur (reste utilisateur — cf. en-tête) :
      galerie média H5 (libellés + filtres), clic média → match f88f6d8b (l'association
      n'existe qu'en prod), matchs témoins (map Tidal, mode, Dominance, rating/Placement).
      Requête anonyme = 403 (ownership) → départage utilisateur, non automatisable.
- [x] V4. Déploiement FAIT : merge PR #54 → main → deploy prod automatique le 2026-07-12
      (~09:25 UTC), avec accord utilisateur préalable.

## 7. Tracker (à remplir en exécution)

| Lot | Statut | Date | Notes |
|---|---|---|---|
| A | COMPLÉTÉ | 2026-07-11 | mode/playlist/tug vérifiés réel (JGtm h5 + non-rég Infinite) ; explorer hors périmètre (§8) |
| B | COMPLÉTÉ | 2026-07-11 | Tidal seedé local (--overrides-only) ; map_ui='Tidal' vérifié réel ; garde-fou 0 map non résolue. PROD : rejouer en V2 |
| C | COMPLÉTÉ | 2026-07-11 | Résistance + Résultat attendu masqués selon capability/null ; 5 tests vitest ; 112/112 match-view ; fix CI types test (tsc -b) |
| D | COMPLÉTÉ | 2026-07-11 | D2 ventilation : JGtm 303, Madina 293, Chocoboflor 277, XxDaemon 129 → **1002/1002 placement_csr_null** (0 carnage KO, 0 owner absent) ; lignes Placement écrites, couverture classés 100 % × 4 joueurs. PROD : rejouer D2 en V2 |
| E | COMPLÉTÉ | 2026-07-11 | delivery-checklist déroulée ; gate global vert (tests+lint+types+CI branche) |
| V | COMPLÉTÉ : V1/V2/V4 [x] ; V3 [!] user | 2026-07-12 | salve post-deploy : overrides Tidal → 48/48 maps, 1002 lignes Placement (JGtm 303/Madina 293/Choco 277/XxDaemon 129, 100 % placement_csr_null, 0 skip), purge 84 média (dry-run=84) ; V3 = vérif visuelle utilisateur |
| F | COMPLÉTÉ | 2026-07-11 | vérifié réel local (7 clips h5 associés, copies prod du scratchpad expirées) ; purge locale 84→0 ; f88f6d8b lui-même = assoc PROD → confirmé en V3. PROD : rejouer F3 en V2 |

## 8. Découvertes hors périmètre (NE PAS traiter dans ce chantier)

- **Explorer « matchs récents cible » (Q19c) non résolu pour H5** (constaté au LOT A2) :
  `Q19cTargetRecentMatches` lit `r.map_name` / `r.pair_name` / `r.pair_name_fr` BRUTS du
  registre — tous NULL sur H5 (map ET mode ET playlist vides, pas seulement le mode). Le
  « même fallback game_variant » demandé par A2 ne suffirait pas (la map serait toujours
  vide). Correctif propre = porter la cascade `asset_translations` complète (map+mode+
  playlist, + fallback game_variant) à Q19c/`scanTargetRecentMatch`, distinct et plus large
  que le périmètre match view/média. Statué [!] dans A2.

- `GetMatchKVPairs`/`GetMatchEvents` avalent les erreurs SQL (`return nil, nil`,
  `match_view_repo_extras.go:56/61`) — anti-pattern « swallowed error » : un échec de
  vue devient silencieusement « pas de données ».
- Vue `v_killer_victim_full` : `kvp.*` + re-jointure des MÊMES colonnes
  `killer_gamertag`/`victim_gamertag` (steps_shared.go:229-236) — colonnes dupliquées,
  fragile.
- `backfill_registry_names` ne répare que `*_name == *_id`, pas les noms NULL
  (cas H5 généralisé) — sans impact affichage (résolution à la lecture) mais
  incohérent avec sa promesse.
- ~188 matchs classés H5 (JGtm) portent une ligne LUSR héritée d'avant le filtre
  is_ranked : la vue _latest la sert et `resolveMatchRatingType` l'étiquette « CSR »
  → valeur LUSR affichée sous label CSR. À auditer (affichage trompeur potentiel).
- `present_at_beginning` NULL sur 100 % des participants H5 (fallback len(team) du
  LUSR OK, mais signal de présence absent → poids temps-joué dégradé).
- `match_csrs`/`match_csrs_latest` : table shared à 0 ligne (jamais alimentée) —
  candidate purge ou câblage.
- Logs prod incohérents sur une même requête : la ligne authz logge `title=halo_5`
  (header reçu) mais la ligne d'accès http logge `title_slug=halo_infinite` (défaut) —
  le middleware de log http ne voit pas le titre résolu. Gênant pour tout diagnostic
  par logs.
- Logs applicatifs prod non persistés hors conteneur : un redéploiement (conteneur
  recréé) efface l'historique — le trafic utilisateur d'avant la panne était perdu.

## 9. Effort estimé

A : 0,5-1 j · B : 0,5 j · C : 0,25 j · D : 0,5-1 j (hors runs) · F : 0,5-1 j ·
E : 0,25 j · V : 0,25 j (V2-V4, après merge). Total code local ~2,5-4 j-h.
