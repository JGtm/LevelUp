# Audit dette technique & documentation — LevelUp

> Date : 2026-07-02 — Revue seule ; ce document et l'entrée thought_log associée sont les seules écritures (a posteriori, sur demande).
> Périmètre : qualité de la documentation technique (READMEs, réfs ADR, commentaires d'invariants, docs/FR) + durabilité (guards/flags, angles morts multi-titre hors registre MT, dette silencieuse schéma DB/contrats).
> Méthode : 7 passes d'exploration parallèles (READMEs, réfs ADR, docs/FR, hardcodings Infinite, guards/flags, schéma DB, commentaires d'invariants) + contre-vérification manuelle de chaque finding porteur. 5 faux positifs d'agents éliminés sur pièces (section 5).
> Complémentaire des trois audits du même jour : [AUDIT_ARCHI_GO_API_2026-07.md](AUDIT_ARCHI_GO_API_2026-07.md) (couches/structure/perf), [AUDIT_QUALITE_SECURITE_GO_API_2026-07.md](AUDIT_QUALITE_SECURITE_GO_API_2026-07.md) (sécurité/robustesse/tests), [V7/CODE_REVIEW_2026-07-02.md](V7/CODE_REVIEW_2026-07-02.md) (qualité/duplication/code mort). Leurs findings ne sont pas re-reportés, ils sont cités.
>
> Note de correction : la version orale de ce rapport (session du 2026-07-02) affirmait « pipeline sync V1 encore défaut ». C'est faux — `shouldUseV2()` (`internal/scheduler/auto_sync.go:437-444`) retourne `v != "v1"`, donc V2 est le défaut runtime. Ce sont les commentaires qui sont inversés (finding n°2 ci-dessous, recoupe CODE_REVIEW_2026-07-02 qui l'avait détecté). Le présent document est la version corrigée.

---

## Résumé exécutif

| Axe | Verdict | Constat dominant |
|---|---|---|
| Doc d'entrée agents (CLAUDE.md, project_map) | Rouge | Les deux fichiers lus « avant toute action » décrivent un monde Python supprimé du repo et des chemins DB qui n'existent plus |
| ADRs & réfs code | Vert (2 exceptions) | 1 092 réfs cohérentes, renumérotations propres, CLOSED sans dépendance ; 1 pointeur fantôme (0014 -> réel 0016), ADR 0005 en dérive vs code |
| READMEs & doc in-code | Orange | Catalogues à jour sauf `temporal/` ; les invariants critiques (INSERT-only, no-lease post-sync, mono-process) vivent dans les ADRs mais pas aux points de mutation ; docs de kill-switches inversées vs comportement réel |
| docs/FR (règle CLAUDE.md n°18) | Orange | Zéro enforcement ; désyncs avérées dans les deux sens ; politique de traduction incohérente (2 ADRs sur 29) |
| Guards & flags | Rouge | Branche d'écriture legacy dangereuse (UPSERT pré-ART-fix) conservée derrière `LEVELUP_PERSIST_BATCH=0` ; V1 sync entretenu comme rollback sans date de retrait ; ADR 0023 Phase 5 en retard sur son propre critère |
| Multi-titre (angles morts hors registre MT) | Orange | Le socle est réellement solide (vérifié) ; restent le pipeline film Infinite logé dans `analysis/`, les goldens non paramétrés, l'absence de template « nouveau titre » |
| Schéma DB & contrats | Vert clair | OpenAPI ratchet strict, pas de table orpheline ; dettes ciblées : 2 colonnes mortes, ~105 migrations sans cycle-out, allowlist ART warning-only |

Paradoxe central : la discipline documentaire du code Go est excellente (1 092 réfs ADR, `persist/doc.go` exemplaire, drift-detector OpenAPI), mais les documents d'orientation que le workflow agentique rend obligatoires sont les plus périmés du repo — coût payé à chaque session d'agent, en contexte gaspillé et en fausses pistes.

---

## TOP 10 priorisé

| N° | Sév. | Finding | Emplacement | Effort |
|---|---|---|---|---|
| 1 | P1 | CLAUDE.md ~60% obsolète : environnement Python, commandes et règles sur du code supprimé, chemins DB v5 faux | `CLAUDE.md` | 0,5-1 j |
| 2 | P1 | Docs des kill-switches sync inversées vs comportement réel : V2 est le défaut (`shouldUseV2()` = `v != "v1"`) mais `cmd/server/main.go:1108-1111` (« dormant... le flow runtime reste 100% V1 ») et `internal/sync/v2/doc.go:17` (« défaut v1 ») affirment l'inverse ; V1 entretenu comme rollback vivant sans date de retrait. Recoupe CODE_REVIEW_2026-07-02 (inversions doc), + 1 site (main.go) ajouté ici | `apps/go-api/internal/scheduler/auto_sync.go:437-444` vs `cmd/server/main.go:1108-1111`, `internal/sync/v2/doc.go:17` | 0,5 j (docs) + décision retrait V1 |
| 3 | P1 | `LEVELUP_PERSIST_BATCH` = guard forever : défaut ON depuis 2026-05-24, la branche `=0` réactive précisément le chemin UPSERT concurrent qui corrompait l'ART (ADR 0017/0018/0019) ; aucune date d'expiration | `apps/go-api/cmd/levelup/cmd_sync.go:68-70`, `internal/sync/engine.go:338` (+6 sites) | ~2 j |
| 4 | P1 | ADR 0023 Phase 5 en retard (~28 fichiers / 65+ appels de fallbacks legacy) et le marqueur `legacy_source_used` documenté par CLAUDE.md n'existe pas dans le code (0 occurrence) : la suppression future est aveugle sans télémétrie | `internal/worldenrich/wiring.go:31-41`, `cmd/server/main.go:2227`, 13 CLIs | ~3 j |
| 5 | P2 | `.ai/project_map.md` gelé au 2026-04-28 et activement trompeur (« Film Chunks : NON EXPLOITABLES » — démenti depuis ; règles Pandas/src/), alors qu'estampillé « cartographie vivante » | `.ai/project_map.md` | 0,5 j |
| 6 | P2 | Pipeline film/armes 100% Infinite logé dans `internal/analysis/` (couche « algos purs » title-agnostic) — angle mort absent du registre MT-01..26 ; MT-15 a fait cette extraction pour la chaîne LUSR, pas pour le film | `internal/analysis/{weapon_data,highlight_event_parser,spawn_detection,kill_attribution,weapon_scanner,weapon_reconciliation}.go` | 1-2 j |
| 7 | P2 | ADR 0005/Prestige en dérive : l'ADR documente défaut OFF + ré-évaluation avant fin Q3 2026 (2026-09-30) ; le code est défaut ON via 2 gates aux sémantiques divergentes (env-only vs settings.json+env) | `internal/prestige/sync_hook.go:32` vs `internal/config/config_settings.go:73` | 0,5 j |
| 8 | P2 | Réf ADR fantôme : le doc du B-swap pointe `adr/0014-shared-db-provider-b-swap.md « (à créer au commit 9) »` — l'ADR existe sous le numéro 0016 ; statut « commit 2/9 » périmé | `internal/platform/duckdb/sharedprovider/doc.go:23`, `baseline_red_integration_test.go:93` | 15 min |
| 9 | P2 | Invariants non écrits aux points de mutation : INSERT-only absent de `SharedPersister.Persist`, design « pas de lease pendant post-sync V2 » absent du runner, contrainte mono-process absente de l'API `Provider`, recette 3-étapes absente d'`enrichmentFields()` | `internal/persist/`, `internal/sync/v2/`, `sharedprovider/provider.go` | 0,5 j |
| 10 | P2 | Règle docs/FR n°18 sans enforcement : `docs/CITATIONS.md` EN figé 2026-02-26 vs FR 2026-06-25 (4 mois, sens inverse), `COMMENDATIONS.md` sans FR, 2/29 ADRs traduites (politique incohérente), liens relatifs cassés dans `docs/FR/ARCHITECTURE_V6.md` | `docs/`, `docs/FR/` | décision + hook |

---

## 1. Qualité de la documentation technique

### 1.1 Les documents d'entrée du workflow agentique sont les plus périmés du repo — P1

Coût récurrent : CLAUDE.md et `.ai/project_map.md` sont, par règle projet, lus avant toute action par chaque agent.

CLAUDE.md (dernier commit 2026-06-25 — entretenu par ajouts Go, jamais purgé) :
- Vérifié sur disque : `src/` et `.venv/` n'existent plus ; il reste 3 fichiers `.py` dans tout le repo (2 générateurs de fixtures de test sous `apps/go-api/tests/`, 1 script d'analyse sous `scripts/`). Aucun Python dans le Dockerfile, le Makefile ni les workflows.
- Sections mortes : § Environnement Python entier, § Commandes Utiles (`scripts/sync.py`, `backup_player.py`, `restore_player.py`, `backfill_data.py` — inexistants), règles 2 à 17 (Pydantic, Pandas/Polars, SQLite, Streamlit, Plotly, fragments, SyncScope, `src/analysis/` vs `src/data/services/`), § `src/ai/`, § Modules Supprimés v4.1, § Stack Technique, § anti-patterns avec exemples Python.
- § « Architecture des Données (v5) » : les chemins `data/warehouse/...` et `data/players/{gt}/` n'existent plus — layout réel (vérifié) : `data/titles/{halo_infinite,halo_5}/warehouse/` et `data/titles/{slug}/players/{gt}/` (ADR 0008). L'exemple MCP `ATTACH 'data/warehouse/metadata.duckdb'` échoue tel quel.
- CLAUDE.md documente un warn log `legacy_source_used` qui n'existe nulle part dans `apps/go-api` (grep : 0 occurrence).
- Reste valide : liste des 29 ADRs (vérifiée exacte, titres conformes), règle auth ADR 0023 (hors marqueur ci-dessus), règle écritures Collect->Persist, stratégie de branches, skills.

`.ai/project_map.md` : gelé au 2026-04-28 (2 mois), auto-déclaré « cartographie vivante ». Affirmations démenties depuis (« Film Chunks : NON EXPLOITABLES pour l'identification d'armes » alors que l'extraction arme-par-kill offline a été résolue) ; règles du monde Python ; le corps a dérivé en changelog d'avril.

`.ai/thought_log.md` : 31 777 lignes, pas de rotation/archivage alors que le projet archive déjà par version (`.ai/archive/v5.0/`, `v6.0/`). Le corpus `.ai/` compte 309 fichiers .md ; la « doctrine RE-VÉRIFIER » du [PLAN_MULTITITRE_INDEX](V7/PLAN_MULTITITRE_INDEX.md) est un aveu lucide que les pointeurs de ce corpus rotent plus vite qu'ils ne sont maintenus.

### 1.2 Références ADR dans le code : excellentes, deux exceptions — P2

Inventaire : 1 092 références couvrant les 29 ADRs, toutes existantes ; renumérotations (0020->0027, 0021->0028, 0024->0029) documentées en en-tête et réfs code sémantiquement cohérentes par zone ; ADRs CLOSED 0017/0018 sans dépendance code résiduelle (transition 0019 propre). Point fort réel.

Exceptions :
- Pointeur fantôme B-swap (vérifié à la ligne) : `sharedprovider/doc.go:23` et `baseline_red_integration_test.go:93` renvoient vers `docs/adr/0014-shared-db-provider-b-swap.md « (à créer au commit 9) »`. L'ADR a été écrite — sous le numéro 0016 (0014 = progression Ascension). Le doc.go annonce aussi un état « commit 2/9 » alors que le B-swap est livré.
- ADR 0005 (Prestige) : voir 2.3.

### 1.3 READMEs de package — globalement frais, un trou — P2/P3

- `internal/analysis/temporal/README.md` omet les deux APIs d'engagement scoring (`ComputeEngagementScore`, `ComputeEngagementCoefficient` + types et erreurs sentinelles), importées par ~19 fichiers.
- `breakdown/README.md` et `narrative/README.md` : conformes (exports, types, consumers vérifiés par grep).
- `apps/web/src/components/charts/README.md` : le « catalogue des 11 wrappers » est exact (9 génériques + 2 spécialisés dans `features/timeseries/`, note présente mais tardive).
- `persist/doc.go`, `sync/v2/doc.go` (hors ligne 17, cf. TOP 2), READMEs observability/logging : à jour.
- Sans doc de package alors que critiques : `internal/games/halo_5/`, `internal/progression/`, `internal/domain/`, `internal/api/handlers/` ; et surtout `internal/sync` (v1, 111 fichiers) et `internal/migration` (système append-only central) n'ont pas de doc.go.

### 1.4 Invariants non écrits là où on modifie le code — P2

Le projet documente très bien ses décisions (ADRs, doc.go) mais pas toujours ses invariants au point de mutation. Quatre sites où un dev de passage peut casser la prod sans avoir lu 3 ADRs :
- `SharedPersister.Persist` : l'exigence INSERT-only sur shared (ADR 0019) n'est pas rappelée sur la méthode.
- Runner post-sync V2 : le design « pas de shared writer lease pendant la phase 6 » (ADR 0027 — c'est le gain V2) n'est écrit que dans le doc.go du package, pas dans l'implémentation.
- `sharedprovider/provider.go` : la contrainte « ne jamais ouvrir ce chemin via sql.Open direct — mono-process RO/RW » n'est pas sur l'API publique.
- `enrichmentFields()` : la recette 3-étapes (ADR 0019 / CLAUDE.md) n'est pas reliée depuis la fonction qui en est l'étape finale.

Contre-exemples positifs à répliquer (le standard interne existe) : `persist/doc.go` (108 L, problème->architecture->recettes), `art_probe.go` (workaround ART expliqué), `apps/web/src/stores/createFilterStore.ts:1-16` (piège de scope solo/squad exemplairement raconté — mais pas rappelé dans `soloFilterStore.ts` lui-même).

### 1.5 docs/FR — la règle n°18 vit sur la bonne volonté — P2

Les guides majeurs (FOUNDATIONS_GUIDE, COMMANDS, SYNC_GUIDE) sont synchronisés. Mais :
- `docs/CITATIONS.md` : EN figé au 2026-02-26, FR mis à jour au 2026-06-25 — 4 mois de divergence. Le sens (FR devant EN) échappe à la lettre de la règle n°18, qui ne couvre qu'une direction.
- `docs/COMMENDATIONS.md` (2026-06-25, doc active H5) : pas de pendant FR ; idem RUNBOOK_OPS_DUCKDB_CLI_TOOLS.
- ADRs : 2 traduites sur 29 (0013, 0021) — ni « tout EN » ni « tout bilingue » : absence de politique plutôt que violation formelle (la lettre de la règle n'oblige pas à créer les FR).
- Liens relatifs cassés dans `docs/FR/ARCHITECTURE_V6.md` (`../../.ai/` au lieu de `../.ai/`).
- Aucun hook (lefthook.yml existe pourtant) ni CI ne vérifie la règle.

---

## 2. Durabilité — dette technique & évolutivité

### 2.1 Guards de compatibilité : des « forever guards » sur le chemin le plus critique — P1

Les règles projet (« toute guard a une date d'expiration », « supprimer le flag une fois la feature en prod ») ne sont pas appliquées aux flags les plus importants :

- `LEVELUP_PERSIST_BATCH` (`cmd/levelup/cmd_sync.go:68-70` + 7 autres sites) : défaut ON depuis le 2026-05-24 (Phase 4.7). La branche `=0` réactive `insertFetchedMatch`, c'est-à-dire exactement le chemin UPSERT concurrent qui produisait la corruption ART éradiquée par les ADR 0017/0018/0019. Rollback vivant 5+ semaines après la bascule, vers un état connu dangereux. CODE_REVIEW_2026-07-02 signale de plus une doc « défaut OFF » inversée.
- `LEVELUP_SYNC_PIPELINE` : V2 est le défaut réel (`shouldUseV2()` : `v != "v1"`, `auto_sync.go:437-444`) — mais les commentaires `cmd/server/main.go:1108-1111` et `sync/v2/doc.go:17` disent l'inverse (cf. TOP 2). Le fond durabilité demeure : V1 complet reste entretenu comme opt-out sans date ni critère de retrait — deux pipelines sync à maintenir, chaque évolution multi-titre câblée deux fois (PMT-3 l'a déjà subi).
- `LEVELUP_PERSIST_BATCH_ASYNC` (défaut OFF), `MULTI_TITLE_API_ENABLED` (défaut OFF), `LEVELUP_EVENTS_CONVERGENCE`, `LEVELUP_CONTRACT_VALIDATE` : aucun ne documente son cycle de vie.
- Sur l'ensemble du repo, un seul objet porte une date d'expiration : `TODO(expiry:2026-08-01)` dans `internal/platform/duckdb/season_pass_repo_tracks.go:254` — le bon pattern existe, il n'est jamais réutilisé. 513 TODO/FIXME au total (~63 Go, ~450 TS), ~95% sans date ni ticket ; plusieurs référencent des phases (« P4.3 finale », « P4.4 », « Phase 1.5+ », « D6 ») qu'aucun plan daté ne porte.

### 2.2 ADR 0023 Phase 5 — la dette auth a dépassé son échéance implicite — P1

L'ADR conditionne la Phase 5 (suppression des fallbacks legacy `sync_meta.oauth_refresh_token` / `msal_token_cache` / `SPNKR_OAUTH_REFRESH_TOKEN_*`) à « ~1 semaine de stabilisation » après les Phases 2-4, livrées fin mai 2026. Au 2026-07-02 : ~4 semaines de retard sur ce critère, ~28 fichiers / 65+ appels encore branchés (dont `internal/worldenrich/wiring.go:31-41`, 13 CLIs, le boot `cmd/server/main.go:2227`) — et aucune télémétrie pour savoir si les fallbacks servent encore (le log documenté n'existe pas, cf. 1.1). La suppression sera aveugle tant que ce marqueur n'est pas implémenté puis observé une semaine.

### 2.3 Prestige / ADR 0005 : le flag a survécu à sa phase — P2

Vérifié dans les deux sens : l'ADR documente une activation phasée, défaut OFF, avec ré-évaluation avant fin Q3 2026 (2026-09-30). Le code est passé défaut ON (`prestige/sync_hook.go:32` et `config/config_settings.go:73`) sans mise à jour de l'ADR. Aggravant : les deux gates divergent — `prestige.IsEnabled()` ne lit que l'env var, `loadPrestigeEnabled()` lit `app_settings.json` avec override env. Poser `prestige_enabled: false` dans le JSON gaterait les surfaces HTTP mais pas le hook de sync, qui continuerait d'écrire. À trancher avant l'échéance : acter l'activation (mettre à jour/clore l'ADR, retirer le flag ou le déclarer kill-switch avec source unique).

### 2.4 Multi-titre : le socle tient, trois angles morts hors registre — P2

Ce que l'audit confirme (contre les a priori) : `EndpointResolver` est réellement câblé (constants.toml `[endpoints]` + consommateurs sync/platform + tests) ; le modèle de dégâts par titre est livré (`games/damage_model.go` : `EffectiveHpToKill(slug)`, P80 par titre, traits `ProvidesNativeKDA/DamageTaken/TeamMMR` capability-driven) — le plan PLAN_DAMAGE_MODEL_PER_TITLE est exécuté, restes volontaires documentés dans le code (LUSR const Infinite par design ; gradient escouade front). Le registre MT-01..26 est fiable.

Angles morts non tracés au registre :

1. Le pipeline film/armes Infinite entier vit dans `internal/analysis/` — `weapon_data.go` (catalogue d'IDs filmshell), `highlight_event_parser.go` (scan binaire à motifs fixes), plus `spawn_detection.go`, `kill_attribution.go`, `weapon_scanner.go`, `weapon_reconciliation.go`. Contenu 100% title-specific dans la couche déclarée « algos purs » title-agnostic. MT-15 a fait exactement cette extraction pour la chaîne LUSR (`halo_infinite/skillchain/`) ; le pipeline film est le morceau resté derrière. Destination naturelle : `internal/games/halo_infinite/film/` + entrée au registre MT.
2. Goldens calibrés Infinite non paramétrés par titre (`analysis/timeline/golden_test.go`, `home_canonical_test.go`, `synthesis_canonical_test.go`) : au premier KPI title-aware qui bouge, ils rougissent en bloc sans distinguer régression et divergence légitime de titre.
3. Pas de template « nouveau titre » : `games/halo_5/` a redéveloppé sa hiérarchie (client -> livesync -> migrations) en miroir d'Infinite, et ses écritures livesync (`halo_5/livesync/csr_match.go:70`) sont des INSERT purs directs, hors couche `persist/` — ART-safe sur le fond (vérifié : INSERT-only sur tables append-only), mais deuxième convention d'écriture qui contredit la lettre de la règle « toute nouvelle écriture per-match passe par BatchBuilder ». Un 3e titre copiera l'un des deux modèles au hasard. `docs/ADD_TITLE.md` existe — véhicule idéal pour figer la convention.

(Le contournement du ratchet `no_slug_comparison` par l'alias `titlePkg.DefaultSlug` et la classification de modes Infinite dans `media_repo.go` sont réels mais déjà consignés dans AUDIT_ARCHI_GO_API_2026-07, findings 30 et 28.)

### 2.5 Schéma DB & contrats : dette ciblée, pas systémique — P3

- 2 colonnes mortes dans `player_match_enrichment` : `known_teammates_count` et `friends_xuids` — aucun writer par stage, valeur figée à la baseline. À décider : réactiver un writer ou droper au prochain rebuild (silencieusement fausses si un futur consommateur les lit).
- ~105 migrations sans politique de cycle-out : aucune n'est retirée ni marquée d'un TTL ; le coût de provisioning d'un nouveau titre croît linéairement.
- Allowlist ART : `TestAllowlistJustifiesEverything` ne fait que logger un warning quand une entrée devient obsolète — ratchet mou comparé aux autres archlints du repo (allowlists vides bloquantes).
- Sains, vérifiés, à ne pas re-suspecter : colonnes `sync_meta` auth = fallback ADR 0023 intentionnel (pas des colonnes mortes) ; vues `mv_*` = vues simples recalculées au SELECT (pas des matérialisées à rafraîchir) ; migrations metadata H5 correctement isolées (`TitleMigrationSet`, pas de dérive silencieuse attendue) ; contrat OpenAPI robuste (YAML manuel 16,7 kL gardé honnête par `TestOpenAPISchemaDrift` ratchet strict + `TestNoJSONRouteBypassesHuma` + `generated.ts` régénéré via `make generate-types`).

### 2.6 Zones où la vitesse a primé sur la structure (constat, sans jugement)

À documenter comme dette assumée, chiffrée par les audits du même jour : racine `api/` devenue deuxième couche service (39 fichiers, ~9,5 kL — AUDIT_ARCHI Axe 1) ; seuils `.golangci.yml` relaxés puis désactivés par répertoire sans baseline gelée (CODE_REVIEW section 6) ; livraison H5 par chemins parallèles rapides (CLIs `h5-*` + livesync direct, cf. 2.4.3) ; corpus `.ai/` de 309 fichiers dont la maintenance repose sur la doctrine RE-VÉRIFIER plutôt que sur la fraîcheur des documents ; thought_log append-only de 31,8 k lignes. Aucun de ces choix n'était déraisonnable au moment où il a été fait ; leur coût est désormais surtout documentaire et d'onboarding.

---

## 3. Points sains à préserver (vérifiés)

Discipline ADR de référence (1 092 réfs cohérentes, renumérotations tracées, CLOSED propres) ; `persist/doc.go` et READMEs observability exemplaires ; drift-detector OpenAPI strict ; archlints à allowlist vide (outcomes, slug — modulo le bypass déjà audité) ; aucun `slug == "halo_infinite"` littéral en prod ; pas de table orpheline ; `EndpointResolver` et damage model per-title livrés ; `breakdown/` et `narrative/` READMEs exacts.

## 4. Recommandations techniques et fonctionnelles

Techniques — par ordre de rendement :
1. Réécrire CLAUDE.md (0,5-1 j) : purger tout le Python, corriger les chemins v5->v7 (`data/titles/{slug}/...`), pointer vers les skills `.claude/skills/` (à jour, elles) ; retirer ou implémenter le marqueur `legacy_source_used`. Rendement maximal : chaque session d'agent en bénéficie.
2. Corriger les 3 docs de kill-switch inversées (main.go:1108-1111, sync/v2/doc.go:17, doc PERSIST_BATCH « défaut OFF ») — 0,5 j ; puis dater le retrait de V1 dans l'ADR 0027 et supprimer `LEVELUP_PERSIST_BATCH` (la branche `=0` est un rollback vers un état dangereux ; si un kill-switch est voulu, le documenter comme tel avec date de retrait).
3. Implémenter le warn log legacy auth, observer 1 semaine, puis exécuter la Phase 5 ADR 0023 (~3 j, ~35 fichiers). L'ordre importe : le log d'abord, sinon la suppression est aveugle.
4. Trancher Prestige avant le 2026-09-30 : acter l'activation dans l'ADR 0005, unifier les 2 gates sur une seule source, retirer le flag ou le requalifier.
5. Corriger les 2 pointeurs `0014-b-swap` -> 0016 et rafraîchir le statut du doc.go sharedprovider (15 min).
6. Ajouter 4 rappels d'invariants aux points de mutation (1.4) + doc.go pour `internal/sync`, `internal/migration`, `internal/games` (0,5 j, patron : `persist/doc.go`).
7. Généraliser `TODO(expiry:YYYY-MM-DD)` (précédent existant) avec un lint léger qui échoue à date dépassée ; passer le check d'allowlist ART de warning à erreur.
8. Multi-titre : déplacer le pipeline film vers `games/halo_infinite/film/` (pattern MT-15), l'inscrire au registre MT ; paramétrer les goldens par slug ; figer la convention d'écriture des titres dans `docs/ADD_TITLE.md`.
9. Rotation du thought_log par trimestre vers `.ai/archive/` (pratique existante pour les plans) + mise à jour ou rétrogradation explicite de `project_map.md` (le marquer « historique » vaut mieux que « vivant » et faux).

Fonctionnelles / de gouvernance :
- Politique docs/FR explicite : soit « ADRs et runbooks = EN-only » écrit dans CLAUDE.md (la règle n°18 devient vérifiable), soit traduction — puis un hook lefthook qui liste les paires désynchronisées dans le diff. L'incohérence actuelle (2 ADRs traduites) est pire que chacune des deux politiques.
- Prioriser le retrait du pipeline V1 avant l'activation multi-titre 1b : chaque semaine où V1+V2 coexistent, la surface à rendre title-aware est double.
- Décisions produit sur `known_teammates_count`/`friends_xuids` (réactiver ou droper) et sur la ré-évaluation Prestige — deux échéances silencieuses qui ne se rappelleront pas d'elles-mêmes.

## 5. Faux positifs écartés pendant la revue (méthode)

Consignés pour éviter de futurs re-audits :
- « `EndpointResolver` créé mais consommé par aucun callsite » — FAUX : câblé (`games/endpoints.go`, `mappings/loader_endpoints.go`, `sync/endpoint_resolver.go`, `platform/halo/endpoints.go`, tests dédiés). MT-01 Exit Gate confirmé.
- « Littéral 225 HP câblé partout, `effective_hp_to_kill` jamais consommé front » — PÉRIMÉ : `games/damage_model.go` livré, callers compute branchés (`post_sync_progression_queries.go:127,337`), front consomme via bootstrap (`HelpPage.tsx:20`). Restes volontaires documentés (LUSR, gradient escouade).
- « Prestige défaut OFF » — FAUX : défaut ON dans les deux gates (`sync_hook.go:32`, `config_settings.go:73`) ; c'est l'ADR qui est en retard (cf. 2.3).
- « Colonnes `sync_meta.oauth_refresh_token`/`msal_token_cache` mortes » — FAUX : fallback ADR 0023 intentionnel en transition (Phase 5).
- « Pipeline sync V1 encore défaut » (version orale de ce rapport) — FAUX : `shouldUseV2()` = `v != "v1"`, V2 défaut ; ce sont les commentaires qui sont inversés (TOP 2).
