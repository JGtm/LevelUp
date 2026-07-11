# PLAN — Résidus Halo 5 match view (2026-07-07)

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

### LOT D — Ratings par match : instrumentation + backfill ciblé
Périmètre fermé :
- [ ] D1. `livesync.PersistPerMatchRatings` : compteurs + logs de skip par raison
      (carnage_err / owner_absent_du_carnage / placement_csr_null / persist_err) —
      règle n°3, plus aucun `continue` silencieux. Exposer le bilan en fin de run.
- [ ] D2. `cmd/h5-csr-match-backfill` : flag `--missing-only` (ne traite que les
      classés sans ligne CSR dans la player DB — ~388 matchs sur 4 joueurs au lieu de
      ~5900 fetches). Run pour JGtm, Madina97294, Chocoboflor, XxDaemonGamerxX ;
      consigner la ventilation des skips dans §7.
- [ ] D3. Selon D2 (DEC-4) : si placement majoritaire → écriture ligne « Placement »
      (SkillRankInsert rating NULL + tier_label, via PlayerPersister, append-only
      ADR 0019/0026) + affichage front du label ; sinon `[!]` justifié ici.
- [ ] D4. Vérifier les 3 matchs témoins : ligne CSR (ou Placement) présente, header
      match view non vide.
Gate LOT D :
```
cd apps/go-api && go test -tags=integration ./internal/persist/... ./internal/games/halo_5/...
# (persist touché → suite integration OBLIGATOIRE, delivery-checklist)
go run cmd/tmpdbq/main.go "../../data/titles/halo_5/players/JGtm/stats.duckdb" \
  "SELECT rating_type, count(*) FROM match_skill_rank_latest GROUP BY 1"  # CSR > 1003
# comptage « classés sans ligne » par joueur : doit tendre vers 0 ou être justifié D3
```

### LOT F — Médias : libellés match + double indexation (issu du volet VPS)
Périmètre fermé :
- [ ] F1. Libellés match de la galerie média (DEC-7) : `computedMapLabel` /
      `computedModeLabel` / `computedPlaylistLabel`
      (`media_repo_q37_enrich.go:22-67`) tombent en fallback sur la cascade
      asset_translations (ResolveAssetNamesBulk — RÉUTILISER, pas dupliquer) quand les
      colonnes registre sont vides ; mode : même fallback game_variant que le lot A2.
      Les filtres map/mode/playlist de la galerie doivent se peupler pour H5.
- [ ] F2. Routage titre de l'indexeur média (DEC-8) : filtre par motif de nom de
      fichier par titre dans `IndexMedia` (`internal/ops/media.go`), motifs déclarés
      dans la config titre (mappings TOML), pas de `slug ==` en dur.
- [ ] F3. Purge one-shot des 84 clips H5 du shared_social halo_infinite (cleanup avec
      --dry-run d'abord ; vérifier `cmd/cleanup_media_index` avant d'écrire un
      nouvel outil). Écritures shared_social = Persister + CHECKPOINT (ADR 0022).
- [ ] F4. Tests : enrich média H5 avec registre à noms NULL → libellés résolus ;
      indexeur ignore les fichiers d'un autre titre ; purge idempotente.
Gate LOT F :
```
cd apps/go-api && go test ./internal/platform/duckdb/... ./internal/ops/...
go test -tags=integration ./internal/ops/...   # écritures shared_social touchées
# serveur local avec copie prod de shared_social H5 :
curl -s -X POST -H "X-LevelUp-Title: halo_5" localhost:8000/api/v1/players/JGtm/pages/media \
  | jq '.table.items[] | select(.match_id=="f88f6d8b-7f05-43bf-91bf-4198a6ccee9f") | {map_name, mode_name}'
# attendu : map "Plaza", mode "Assassin" (plus de « Carte inconnue »)
```

### LOT E — Clôture chantier local
- [ ] E1. Statuer chaque item A→D+F (`[x]`/`[~]`/`[!]`), remplir §7-§8.
- [ ] E2. `.ai/thought_log.md` : entrée de clôture (obligatoire).
- [ ] E3. Skill `delivery-checklist` avant commit final / proposition de merge.
Gate : gate global = tous les gates A→D+F verts dans la même session + lint + types.

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
- [ ] V2. Après merge des lots A-F : rejouer en prod les opérations data — B2 (seed
      Tidal metadata), D2 (CSR --missing-only, 4 joueurs), F3 (purge 84 copies
      Infinite). Fenêtre creuse, ressources serrées (2 vCPU / 2 Go), PRÉVENIR avant
      toute écriture. Un writer par DB : passer par les CLI (jamais deux process RW).
- [ ] V3. Vérification visuelle prod par l'utilisateur : galerie média H5 (libellés),
      clic média → match f88f6d8b, matchs témoins (map Tidal, mode, Dominance,
      rating/Placement).
- [ ] V4. Déploiement des fixes : merge → push main = deploy auto (PRÉVENIR
      l'utilisateur avant le push).

## 7. Tracker (à remplir en exécution)

| Lot | Statut | Date | Notes |
|---|---|---|---|
| A | COMPLÉTÉ | 2026-07-11 | mode/playlist/tug vérifiés réel (JGtm h5 + non-rég Infinite) ; explorer hors périmètre (§8) |
| B | COMPLÉTÉ | 2026-07-11 | Tidal seedé local (--overrides-only) ; map_ui='Tidal' vérifié réel ; garde-fou 0 map non résolue. PROD : rejouer en V2 |
| C | COMPLÉTÉ | 2026-07-11 | Résistance + Résultat attendu masqués selon capability/null ; 5 tests vitest ; 112/112 match-view |
| B | à faire | | |
| C | à faire | | |
| D | à faire | | D2 : ventilation skips = |
| F | à faire | | copies prod dispo dans le scratchpad session (h5_social etc.) |
| E | à faire | | |
| V | V1 fait ; V2-V4 après merge | 2026-07-07 | prod = local ; « introuvable » = LOT F |

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
