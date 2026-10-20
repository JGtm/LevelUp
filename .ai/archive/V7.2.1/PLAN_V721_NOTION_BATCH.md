# PLAN — Chantier v7.2.1 (section Notion « Pour la v7.2.1 »)

> Branche `feat/v7.2.1-notion-batch`, worktree dédié depuis `main` local (HEAD `9f0f7d48b`).
> Ouvert le 2026-07-25. Contrat d'exécution : skill `plan-execution` (ordre strict, aucun
> report d'action exécutable, statut obligatoire par item, gate + thought_log par étape).
> Reconnaissance : 6 agents, rendus le 2026-07-25 — chaque état ci-dessous est **vérifié sur
> pièces**, pas déduit des cases cochées des plans sources.

## Périmètre

13 items `V721-01` à `V721-13`, issus de la section Notion « Pour la v7.2.1 » :
6 découvertes de sortie de chantier v7.2 (réponses utilisateur déjà données), 4 plans
`.ai/` à analyser/corriger/traiter, la question `openspartan`, le triage du backlog, et
le lot final de documentation de release.

## Hors périmètre (décidé, ne pas traiter ici)

- **Incident prod** : le VPS est injoignable (SSH + HTTPS en timeout depuis ~19:03 UTC le
  25/07) et `v7.2.0` n'est PAS déployée (2 échecs du workflow `Deploy to VPS`, mort pendant
  le `go build` CGO). Signalé à l'utilisateur, qui a tranché « rien pour l'instant ».
  Aucune opération VPS/prod dans ce chantier. Aucun backfill prod.
- Tauri, Kills environnementaux, Housekeeping post-cutover (exclus explicitement par
  l'utilisateur du triage backlog).
- Échec CI `Go Coverage` sur `main` : flake connu `SharedProvider reopen`, non traité ici.

---

## V721-01 · Alertes Dependabot (4 ouvertes sur `main`)

**Réponse utilisateur** : « Ok je te laisse voir, on prend pas le risque de rendre notre app
instable pour autant donc je te laisse juger. »

| Alerte | Paquet | Sévérité | Actuel → corrigé | Nature | Décision |
|---|---|---|---|---|---|
| #10 | `js-yaml` | high | 4.2.0 + 4.1.1 → 4.3.0 | transitif, **outillage build seul** (`openapi-typescript` → `@redocly/openapi-core`) | **Bumper** |
| #9 | `brace-expansion` | high | 2.1.1 → 2.1.2 | transitif (même chaîne) | **Bumper** |
| #8 | `brace-expansion` | high | 5.0.6 → 5.0.7 | transitif (devDeps) | **Bumper** |
| #6 | `echarts` | moderate | 5.6.0 → **6.1.0** | **directe, runtime, moteur de TOUS les charts** | **Ne PAS bumper en v7.2.1** |

**Décision tranchée (moi)** : les 3 `high` sont des DoS dans des dépendances d'outillage de
build, jamais chargées par le navigateur, et les bumps sont des patchs — risque nul, gain
réel (les 3 alertes `high` disparaissent). `echarts` est un **bump majeur** sur le composant
le plus sollicité de l'UI : c'est exactement le « risque de rendre l'app instable » que
l'utilisateur refuse. Il devient un chantier dédié v7.3 avec spike, typecheck, vitest et
tournée visuelle — consigné au backlog **avec date cible**, pas un « au cas où ».

- [x] 01.1 — Relever `overrides.js-yaml` à `^4.3.0` (`apps/web/package.json:72-74`)
- [x] 01.2 — Ajouter les overrides `brace-expansion` (2 lignées : `^2.1.2` et `^5.0.7`)
- [x] 01.3 — Régénérer `package-lock.json`, vérifier que les 3 alertes tombent
- [x] 01.4 — Entrée backlog datée pour `echarts` 5→6 (cible v7.3, critère : suite charts verte + tournée visuelle)

**Gate** : `npm run typecheck` + `npm run test` + `make generate-types` inchangé (js-yaml est dans cette chaîne).

---

## V721-02 · Étendre `match_objective_stats` aux 4 blocs restants

**Réponse utilisateur** : « OK ».

**Faisabilité — TRANCHÉE (blocage levé)** : la recon concluait « bloqué faute de payload ».
Vérification en base (`match_registry`, requête `diag_q` du 25/07) :

| Mode | Matchs en base | Bloc visé |
|---|---|---|
| Stockpile | **39** | `StockpileStats` |
| Extraction | **2** | `ExtractionStats` |
| Attrition | **2** | `EliminationStats` — **INFIRMÉ au P0 : aucun bloc dans le payload** |
| « Survive The Undead 3.0 » (UGC) | **1** | `InfectionStats` — **INFIRMÉ au P0 : c'est un Firefight** |

**Périmètre après P0 : 2 blocs (Stockpile + Extraction), 11 colonnes — puis ÉLARGI à
3 blocs / 18 colonnes avec `VipStats` (cf. « PÉRIMÈTRE FINAL » ci-dessous).** Elimination
et Infection restent NON SPÉCIFIABLES (aucun payload existant ne porte ces blocs) — ils
sortent du chantier tant qu'un match d'un de ces modes n'aura pas été joué et synchronisé.
Interdiction explicite d'inventer leur schéma par analogie.

Patron intégral à répliquer (V72-03, vérifié) : `StatsBundle` (`internal/openspartan/halo_api_payload.go:82-96`)
→ `internal/sync/objective/objective_stats.go:27-113` → migration `shared_create_objective_stats`
(`internal/games/halo_infinite/migrations/steps_shared_objective_stats.go`) → `persist.ObjectiveStatsInsert`
(`internal/persist/rows.go:271-300`) → `persistObjectiveStats` (`internal/persist/shared_persister.go:494-538`,
**INSERT pur**) → `cmd/backfill_objective_stats`.

- [x] 02.1 — **P0 CLOS (2026-07-25)** : 10 payloads réels capturés (CLI jetable `cmd/tmp_objcap2`
      supprimé, dépôt principal propre). Résultat bloc par bloc, sur pièces :
      - `StockpileStats` **CAPTURÉ** (3 matchs BTB:Stockpile, 79 blocs, 6 champs stables) :
        `KillsAsPowerSeedCarrier`, `PowerSeedCarriersKilled`, `PowerSeedsDeposited`,
        `PowerSeedsStolen` (INT) + `TimeAsPowerSeedCarrier`, `TimeAsPowerSeedDriver` (ISO 8601).
      - `ExtractionStats` **CAPTURÉ** (2 matchs BTB:Extraction, 52 blocs, 5 champs stables,
        AUCUNE durée) : `ExtractionConversionsCompleted`, `ExtractionConversionsDenied`,
        `ExtractionInitiationsCompleted`, `ExtractionInitiationsDenied`, `SuccessfulExtractions`.
      - `EliminationStats` **NON CAPTURABLE** : les 2 matchs Arena:Attrition
        (`2000143c-…`, `f395b462-…`, GameVariantCategory 7, même UgcGameVariant) n'ont
        AUCUN bloc de mode — `Players[].PlayerTeamStats[].Stats` = `CoreStats` seul
        (idem `Teams[].Stats`). Aucun mode Elimination en base (inventaire des 65 modes
        distincts de `match_registry`). Hypothèse INFIRMÉE, aucun schéma déduit.
      - `InfectionStats` **NON CAPTURABLE** : « Survive The Undead 3.0 » est un Firefight UGC
        (GameVariantCategory 41, `PveStats` + `PvpStats`), pas un Infection. Aucun mode
        Infection en base. Hypothèse INFIRMÉE.
      - Découverte initialement hors périmètre, **RÉINTÉGRÉE** (arbitrage utilisateur) :
        bloc `VipStats` RÉEL et NON DÉCLARÉ dans `StatsBundle` (`models.go`), présent sur
        les 3 matchs Arena:VIP en base (7 champs, dont 2 durées ISO). Traité dans la même
        migration que Stockpile/Extraction — cf. « PÉRIMÈTRE FINAL ».
      Fixtures anonymisées : `testdata/objective_stats/{stockpile,extraction}_match.json`.
**PÉRIMÈTRE FINAL (élargi en cours d'exécution, arbitrage utilisateur du 2026-07-25)** :
**3 blocs, 18 colonnes**. Le bloc `VipStats` — découverte « hors périmètre » du P0, réelle
et non déclarée dans `StatsBundle` — a été intégré à la MÊME migration : mécanique
identique, coût marginal nul, alors qu'un report aurait imposé une 2e migration. Payload
relevé sur le dump réel du match Arena:VIP `00761d27-487c-4d7d-ac4c-bf7584de652c`
(8 blocs joueur, `GameVariantCategory` 23) — 7 champs : `KillsAsVip`, `VipKills`,
`VipAssists`, `TimesSelectedAsVip`, `MaxKillingSpreeAsVip` (INT) + `TimeAsVip`,
`LongestTimeAsVip` (ISO 8601). Le bloc existe aussi sous `Teams[].Stats` : comme pour
les autres modes, seul le niveau JOUEUR est extrait.

- [x] 02.2 — Migration : step **SÉPARÉ** `shared_objective_stats_add_stockpile_extraction`
      (`steps_shared_objective_stats.go`, `TargetShared`) — 18 × `ALTER TABLE
      match_objective_stats ADD COLUMN IF NOT EXISTS … ` nullable **+ `CREATE OR REPLACE
      VIEW match_objective_stats_latest`** dans le MÊME step. Enregistré en fin de
      `canonicalOrder` (`internal/migration/order.go`, 1 ligne).
      Le CREATE existant n'est PAS édité (migrations name-keyed : les DBs déjà migrées ne
      le rejoueraient pas). **Recréation de la vue OBLIGATOIRE, pas optionnelle** : DuckDB
      fige la liste de colonnes d'un `SELECT *` à la création de la vue — sans elle, les
      18 colonnes seraient invisibles pour TOUS les lecteurs (Q12, `ObjectiveStatsRepo`,
      backfill). ALTER ADD COLUMN = DDL pur, hors périmètre ART #23046 ; aucun garde-rail
      ART touché.
- [x] 02.3 — `ObjectiveStatsInsert` (+18 champs pointeurs), `buildObjectiveRow` (+3 blocs
      `StockpileStats`/`ExtractionStats`/`VipStats`), INSERT de `persistObjectiveStats`
      **43 colonnes / 43 placeholders / 43 arguments** (recomptés). INSERT reste PUR :
      aucun UPSERT, aucun `ON CONFLICT`, aucun `INSERT OR REPLACE`. Les 4 durées passent
      par `objectiveDurationSeconds` (fractions préservées), jamais `parsePTDuration`.
      Chaîne de lecture étendue en parallèle (sinon l'UI ne voit rien) : `ObjectiveRaw`
      (+`HasStockpile`/`HasExtraction`/`HasVip`), Q12, scan scoreboard,
      `buildScoreboardObjective`, `MatchScoreboardObjective`. `VipStats` ajouté à
      `StatsBundle` (`internal/openspartan/halo_api_payload.go`) — il y manquait.
- [x] 02.4 — Tests : 3 tests golden (`TestExtractObjectiveStats_{Stockpile,Extraction,Vip}`)
      sur fixtures réelles anonymisées, assertions d'exclusivité mutuelle dans les 2 sens
      (les nouveaux blocs vérifient CTF/Zones/Oddball nuls, le test CTF vérifie
      Stockpile/Extraction nuls), « aucun bloc → aucune ligne » déjà couvert
      (`_SlayerProducesNoRow`). Test de forme migration étendu +
      `TestSharedObjectiveStatsStockpileExtractionColumns` (18 colonnes, nullable, famille
      de type, round-trip PAR LA VUE `_latest`). Test d'intégration persist étendu
      (`TestSharedPersister_ObjectiveStats_StockpileAndExtraction` : 3 lignes, valeurs
      toutes distinctes → attrape un décalage d'un cran dans l'INSERT). Garde-rails ART
      inchangés. Fixtures DDL miroir mises à jour (`player_repos_test.go`,
      `sync_pipeline_fixture_test.go`) — sans elles Q12 casse.
- [x] 02.5 — UI : `ObjectiveMode` += `stockpile` | `extraction` | `vip`, +3 jeux de
      colonnes, `detectObjectiveMode`/`objectiveColsFor` étendus
      (`MatchScoreboard.logic.ts`), libellés FR **et** EN dans `match-view/i18n.ts`
      (13 clés × 2 locales). Go : DTO exposés étendus. **À jouer par le pilote** :
      `make openapi-gen` puis `make generate-types` (le contrat et `generated.ts` sont
      générés — `api/openapi.yaml` jamais édité à la main ; seule la *description* du
      schéma dans `openapi_manual_fragment.yaml` a été mise à jour).
- [x] 02.6 — Backfill LOCAL : **le bit EST déjà posé** — `markObjectiveDone` le pose pour
      TOUT match fetché, y compris ceux qui n'ont produit aucune ligne (c'était le cas des
      matchs Stockpile/Extraction/VIP en v7.2, blocs non extraits). `listCandidateMatchIDs`
      les exclut donc. **Reset ciblé requis avant relance** (SQL exact dans le rapport et
      dans le doc-comment de `cmd/backfill_objective_stats/main.go`). Le sync natif ne pose
      JAMAIS ce bit : les matchs synchronisés depuis restent candidats sans reset.
      Exécution du backfill NON lancée (hors périmètre de l'agent).
- [!] 02.7 — `EliminationStats` **NON TRAITÉ — manque de données, pas un renoncement** :
      les 2 seuls matchs Arena:Attrition en base (`GameVariantCategory` 7) ne portent
      AUCUN bloc de mode (`Players[].PlayerTeamStats[].Stats` = `CoreStats` seul, idem
      `Teams[].Stats`), et aucun mode Elimination ne figure parmi les 65 modes distincts de
      `match_registry`. Aucun schéma n'est déduisible par analogie sans inventer des noms
      de champs. **Condition de reprise** : un match matchmaking Elimination joué puis
      synchronisé → re-capture du payload → réplication du même patron (1 migration ALTER,
      1 bloc dans `buildObjectiveRow`, 1 jeu de colonnes UI).
- [!] 02.8 — `InfectionStats` **NON TRAITÉ — manque de données**, même raison : « Survive
      The Undead 3.0 » (seul candidat en base) est un **Firefight UGC**
      (`GameVariantCategory` 41, blocs `PveStats`/`PvpStats`), pas un Infection. Aucun mode
      Infection en base. Même condition de reprise que 02.7.

**Multi-titre** : aucune nouvelle capability. `CapMatchObjectiveStats`/`CapObjectiveStats`
sont déjà génériques à la table ; Halo 5 reste `not_exposed` par contrainte de source
(le carnage report H5 n'a aucun sous-objet par mode) — la table reste simplement vide pour
ce titre, la section UI est masquée par `useCapability('objective_stats')` + absence de
bloc. **Aucun `slug == …` introduit** (ratchet `no_slug_comparison_test.go` intact).

---

## V721-03 · ~19 nouvelles citations depuis les stats d'objectifs

**Réponse utilisateur** : « Pour les citations propose-moi des visuels en te basant sur ceux
qu'on a déjà ou ce qui est [sur la planche d'icônes de types de partie Halo Infinite]. Si tu
peux pas je m'en occuperai, j'ai Photoshop sur un autre ordi. »

**État vérifié** : 22 colonnes dans `match_objective_stats`, **4 déjà consommées** par des
citations v7.2 (`zone_captures`, `flag_returns`, `zone_secures`, `flag_carriers_killed`),
**19 libres** — ce sont exactement les 19 annoncées. Le moteur est prêt (`mappingTypeObjectiveStat`
routé dans `citations_engine.go:143-172`, source branchée par `loadObjectiveStats`).

**Stock d'assets** : `static/commendations/halo_5_guardians/` (165 PNG 100×100) +
`halo_infinite/` (26) — **105 fichiers libres**. Les 5 assets nommés au backlog existent tous
et sont libres. Le rendu front est un `<img src>` sur une URL statique
(`CitationProgressRing`, `MedalIcon`) : **un SVG monochrome fait main s'intègre sans aucune
adaptation de code**, et se met à l'échelle proprement (les tuiles vont de 52 à 98 px, là où
les PNG 100×100 existants flouent).

**Arbitrage utilisateur rendu (2026-07-25)** — **périmètre réel : 10 citations retenues sur
les 19**, les 9 autres écartées (« elles n'en valent pas la peine ») : ne pas les
re-proposer. Les paliers ont été **recalibrés sur les données réelles**
(`match_objective_stats_latest`, joueur de référence : 304 matchs à objectifs) ; les
premières propositions étaient fausses — quatre citations auraient été littéralement
inatteignables.

| Nom FR | Nom EN | Norm | Colonne source | Total réel (meilleur joueur) | Paliers |
|---|---|---|---|---|---|
| Capture du drapeau | Flag Captures | `flag_captures` | `flag_captures` | 93 | 10,25,50,75,**125** |
| Sécurisation du drapeau | Flag Secures | `flag_secures` | `flag_secures` | 508 | 50,100,200,350,**600** |
| Vol du drapeau | Flag Steals | `flag_steals` | `flag_steals` | 256 | 25,50,100,175,**300** |
| Chasse au rapatrieur | Returner Takedown | `returner_takedown` | `flag_returners_killed` | 48 | 5,10,20,35,**60** |
| Porteur imparable | Unstoppable Carrier | `unstoppable_carrier` | `kills_as_flag_carrier` | 3 | 1,2,3,5,**10** |
| Rapatriement agressif | Aggressive Return | `aggressive_return` | `kills_as_flag_returner` | 40 | 5,10,20,30,**50** |
| Défense de zone | Zone Defense | `zone_defense` | `zone_defensive_kills` | 278 | 25,50,100,200,**350** |
| Crâne intouchable | Untouchable Carrier | `untouchable_carrier` | `kills_as_skull_carrier` | 1 | 1,2,3,5,**10** |
| Chasse au porteur | Skull Carrier Takedown | `skull_carrier_takedown` | `skull_carriers_killed` | 4 | 2,5,10,20,**40** |
| Prise du crâne | Skull Grabs | `skull_grabs` | `skull_grabs` | 3 | 2,5,10,20,**40** |

Les paliers très bas des 4 dernières lignes sont **assumés** : exploits intrinsèquement
rares (frags en portant le drapeau / le crâne) ou mode peu joué (26 matchs Oddball en base).
Un réalignement sur les paliers génériques (`10,20,30,50,100`) les rendrait inatteignables.

- [x] 03.1 — Publier le tableau des 19 dans Notion (colonne commentaire utilisateur)
- [x] 03.2 — Arbitrages rendus : 10 retenues / 9 écartées, libellés FR+EN fixés, paliers recalibrés sur les données réelles
- [x] 03.3 — Seed : 10 lignes dans `defaultCitationMappings()` (`seed_citation_data.go`, bloc « Mode de jeu · objectifs CTF/Zones/Oddball ») + 8 constantes de paliers calibrés (`seed.go`) + **parité EN complète** (`citationDisplayEN` ET `citationDescriptionEN` : 98/98, zéro orphelin) + garde-rails de test (unicité des norms, existence disque de tout `image_path`, couverture EN du nom, fixture des 10 colonnes/paliers)
- [!] 03.4 (POST-MERGE) — Re-seed + backfill citations LOCAL — procédure ci-dessous, à jouer par le pilote (aucune commande Go/`make` jouée par l'agent)
- [!] 03.5 — **BLOCAGE AVANT DÉPLOIEMENT — visuels définitifs en attente de l'utilisateur.** Les 8 autres citations réutilisent des visuels Halo 5 déjà présents (vérifiés sur disque), mais « Capture du drapeau » et « Vol du drapeau » pointent deux **SVG provisoires** générés en interne, que l'utilisateur a écartés. Il produit les siens sous Photoshop ; cibles attendues : `static/commendations/halo_infinite/HI_citation_Capture_du_drapeau.png` et `.../HI_citation_Vol_du_drapeau.png`. Les SVG restent en place comme bouche-trous (chaîne complète et testable en local, pas de vignette cassée) — **ne rien pousser en prod avec eux**. Remplacement = déposer les 2 PNG, changer `.svg` → `.png` dans les 2 `ImagePath`, re-seeder. Le garde-rail `TestCitationImagePaths_ExistOnDisk` échoue si l'extension bascule avant l'arrivée des fichiers.
- [!] 03.6 (UTILISATEUR) — Vérification visuelle utilisateur (après 03.4 **et** 03.5)

**Procédure de re-seed + backfill (pilote, LOCAL)** — commandes littérales de
`docs/COMMENDATIONS.md`. Le seed écrit dans `metadata.duckdb` et le backfill dans les
`stats.duckdb` joueur : **arrêter le serveur (air / go-api) avant**, un seul process writer
par DB (ADR 0013/0016).

```powershell
# 0. arrêter le serveur de dev (air / go-api) — un seul process writer par DB.
#    CGO obligatoire (driver DuckDB) :
$env:CGO_ENABLED=1; $env:PATH="C:\msys64\ucrt64\bin;$env:PATH"

# 1. re-seed du registre de règles (metadata.duckdb du titre par défaut,
#    halo_infinite — les colonnes objectifs ne sont pas exposées pour halo_5).
#    Attendu : "10 insérées, 88 mises à jour".
go run ./apps/go-api/cmd/levelup seed citation-mappings

# 2. recalcul TOTAL + contrôle des invariants V1–V4. Obligatoire, pas un
#    `--citations` incrémental : les matchs déjà traités portent une ligne
#    sentinelle et ne seraient PAS rejoués, donc les 10 nouvelles citations
#    resteraient à zéro sur tout l'historique.
go run ./apps/go-api/cmd/levelup backfill --all --citations-recompute-all

# 3. relancer le serveur, puis ouvrir la page Citations d'un joueur à objectifs
```

---

## V721-04 · Reliquats openapi V72-01

**Réponse utilisateur** : « Ok ».

- [x] 04.1 — `DefaultStatus` sur les **11 handlers** répondant 201/202 (liste exacte établie) → retire ≈ 79 lignes du fragment manuel (3090 → ~3011)
- [x] 04.2 — **38 routes absentes** du contrat publié : assouplir les 4 gates du harnais de rendu (`BuildDemoRouter`) pour monter Prestige (29), catalog (3), diag auto-sync (3), assets_metadata (3). **Option retenue : un seul harnais** (stubs/`:memory:`), pas de second harnais — un second harnais réouvre le canal de drift que H6 a fermé
- [x] 04.3 — `make openapi-gen` + golden `TestOpenAPIYAMLIsUpToDate` régénéré dans le même commit
- [x] 04.4 — Vérifier `TestSharedOpenAPIDocCoversAllHumaRoutes` et les compteurs d'ops dans les tests/plans

**Non traité ici** (décision produit, reste au backlog) : passage des 16 corps `RawBody` en
`Body` typé — change le contrat d'erreur 400 → 422.

---

## V721-05 · Résidus armes

**Réponse utilisateur** : « Ok mais attention à ne pas multiplier les sources, le but est
d'unifier les structures et références pour faciliter la maintenance et l'ajout d'un nouveau
titre. »

**Constat majeur — la prémisse du backlog est FAUSSE sur un des deux points** (2 agents
concordants) : `weapon_labels.name_fr` n'est **pas** shadowée/morte. `weapon_resolver.go:76-93`
la lit comme **3ᵉ maillon** de `COALESCE(wnl.name_fr, wnl.name_en, wl.name_fr, wl.name_en, '')`,
pour tout `weapon_id` sans `weapon_key` dans le registre. Retirer l'écriture de `seedWeapons`
priverait donc les armes hors-registre de tout nom FR — dégradation silencieuse.

**Décision tranchée (moi)** : la consigne « ne pas multiplier les sources » est **déjà
satisfaite** — la source unique du nom d'affichage est `weapon_name_labels` keyée par
`weapon_key`, alimentée par `config/titles/{slug}/mappings/weapon_names.toml` pour Infinite
ET H5 (V72-06). `weapon_labels` n'est pas une source concurrente mais un **filet documenté**
pour les ids non encore mappés. On ne retire donc rien : on **corrige le texte du backlog**
(doc inversée = anti-pattern #9) et on garde le filet.

- [x] 05.1 — Purger la colonne inerte `weapons.name_fr` sur les DB déjà migrées : migration rebuild CTAS-swap (patron `internal/migration/append_only_rebuild.go`) — `purge_weapons_name_fr_column` (`internal/migration/steps_metadata_purge_weapons_name_fr.go`), enregistrée dans `canonicalOrder` après `add_weapon_registry` (positionnée entre `create_prestige_metadata_schema` et `rebuild_catalog_fetch_queue_drop_art_indexes` pour respecter l'invariant `TestSortByCanonicalIsNoOpOnCurrentRegistry` — ordre alphabétique des fichiers globaux) + dépendance déclarée dans `order_dependency_test.go`, tests dans `steps_metadata_purge_weapons_name_fr_test.go` (retrait colonne, no-op DB neuve/table absente, préservation rows + PK composite, idempotence). Commentaire de non-DROP de `weapon_registry.go:200-209` mis à jour (pointe vers la migration).
- [~] 05.2 — Corriger l'entrée backlog `[ops/h5] seedWeapons` : documenter que `wl.name_fr` est un fallback VIVANT, pas de la plomberie vestigiale — déjà traité par un autre agent (hors périmètre de cette passe, `.ai/BACKLOG.md` non touché ici)
- [~] 05.3 — Supprimer l'entrée backlog « Unifier la SOURCE des noms d'armes » (livrée en V72-06, preuves au dossier) — déjà traité par un autre agent (hors périmètre de cette passe, `.ai/BACKLOG.md` non touché ici)

---

## V721-06 · Explorer « Matchs par saison » complet (A2)

**Réponse utilisateur** : « Ok ! »

**Blocage levé** : le backlog disait « exige de sourcer les chemins CMS des 14 saisons ».
En réalité le chemin primaire est **déductible** (`Seasons/Season{N}.json`) et le seul cas
non déductible est l'opération intra-saison (`Season6-2.json`) — que le subquery live
remonte déjà quand elle existe. Aujourd'hui `computeSeasonBreakdown` ne tente QUE les
saisons présentes dans `rec.SeasonIDs`, d'où le « 5/14 » diagnostiqué.

**Décision tranchée (moi)** : union des deux sources — tenter systématiquement le chemin
déterministe pour les 14 saisons du catalogue, **fusionné** avec les chemins remontés par le
live (qui apportent les opérations intra-saison). Aucun sourcing externe nécessaire.

- [x] 06.1 — Générer le chemin primaire par saison du catalogue + union avec `playedByNum` — `deterministicSeasonPath` (ID `^season(\d+)$` → `Seasons/Season{N}.json`, override `extra.matchmade_path` pour l'opération hivernale) + `seasonCMSPaths` (union dédupliquée, déterministe en tête) dans `explorer_target_seasons.go`
- [x] 06.2 — Respecter le plafond de concurrence (`seasonBreakdownConcurrency = 6`) — inchangé ; le fan-out SR passe de ≤6 à ~15 chemins, mais le pic CSR (le plus coûteux : 1 appel par playlist ranked engagée) n'est désormais demandé QUE pour une saison à `matches > 0` → budget CSR identique à avant
- [x] 06.3 — Distinguer « saison non jouée » de « saison non remontée » dans la réponse — `domain.SeasonMatchCount.Unresolved` (bool optionnel, `omitempty`) + statut de section (`ok` / `local_partial` si résolution partielle / `failed` si aucune saison résolue). **Contrat à régénérer** : `make openapi-gen` puis `make generate-types`
- [x] 06.4 — Tests : saison au catalogue absente du subquery live → doit être tentée — `TestComputeSeasonBreakdown_CatalogSeasonMissingFromLive` (+ dédup, 3 états, tout-en-échec, repli sans chemin CMS, garde-rail catalogue réel 14/14)
- [!] 06.5 — Validation empirique sur un joueur à faible couverture live — NON FAIT par l'agent : exige un serveur lancé + tokens Halo valides (créneau Go exclusif détenu ailleurs, aucune commande `go`/`make` autorisée). À rejouer par le pilote après régénération du contrat : rechercher Nilton410 dans l'Explorer et vérifier `seasons_played` dans le log `explorer_season_breakdown` (attendu > 5)

---

## V721-07 · `PLAN_COACH_V3_GENERATION.md` — analyser, corriger, traiter

**État vérifié : LIVRÉ à ~95 % le 2026-06-09**, plan jamais mis à jour (statut affiché
« Proposé » — piège : un agent pourrait réimplémenter les phases A et C).

- [x] 07.1 — **Corriger le plan** : statuts réels par phase, références de lignes périmées (`types.go`, `service_pilot_pool.go`), retrait de la mention « aucun signal coach niveau escouade » (invalidée par `squad_coach.go`) — `.ai/archive/PLAN_COACH_V3_GENERATION.md` (tableau « État d'exécution » + refs corrigées `:268` / `:181-239` + § Phase C réécrite + journal d'exécution en fin de fichier)
- [x] 07.2 — **Amender l'ADR 0014 §6.1** — seul écart réel : l'ADR dit encore « alertes positives uniquement » alors que le soft-négatif est livré (doc inversée). Documenter les garde-fous : seuil −0,10 sur ≥14 j, catégorie neutre, jamais `outcome-loss` — `docs/adr/0014-progression-tracking-v2-ascension.md` (nouvelle section datée 2026-07-25 + correction ponctuelle des 2 passages périmés, en anglais, ADR EN-only)
- [x] 07.3 — Migrer les strings locales de `CoachFocusCard.tsx:26-43` vers un manifest i18n (ADR 0003 — dernier composant coach avec un objet `STR` en dur) — manifest `profile.coach.*` dans `apps/web/src/lib/i18n/manifests/profile.toml`, `apps/web/src/lib/i18n/generated/profile.ts` mis à jour à la main (à régénérer/vérifier au gate), composant migré vers `useProfileI18n().t(...)`
- [x] 07.4 — Clôturer et archiver le plan + retirer la ligne de `.ai/BACKLOG.md:286-287` — `git mv` vers `.ai/archive/PLAN_COACH_V3_GENERATION.md`, ligne retirée de `.ai/BACKLOG.md` (édition chirurgicale, 2 lignes, sans toucher aux 41 lignes déjà modifiées par un autre agent en parallèle sur ce fichier)

**Cooldown du cap soft-négatif** : demandé par le plan, jamais implémenté. **Décision
tranchée (moi)** : ne pas l'ajouter — la carte est déjà « 1 axe max, seuil 0,02 », un cooldown
introduirait un état persistant pour un bénéfice non mesuré. Statué `[!]` avec justification.

---

## V721-08 · `PLAN_SQUAD_CHALLENGES.md` — analyser, corriger, traiter

**État vérifié : lots 1 à 5 clos** (y compris 5.3/5.4, commit `93812f0ca`) ; le journal du
plan se contredit lui-même (dit « différés » ce que les cases disent « fait »).

- [x] 08.1 — **Fix D1, cause localisée du 500 prod** : `prestige_metadata_repo.go:33` utilise `r.db.Query` non recovered → `dblease.ErrDBLocked` n'est jamais émis, donc le mapping 503 du handler (`prestige_squads.go:451-455`) ne se déclenche jamais. Passer en variante recovered
- [x] 08.2 — Filtrer les templates `eval_type=threshold` hors du pool d'escouade — proposer un défi dont la règle affichée ne correspond pas à l'évaluation réelle est le vrai défaut (l'évaluation est cumulative pour tous en V1)
- [x] 08.3 — Corriger le journal du plan (lignes 264-268) + clôturer
- [!] 08.4 (POST-MERGE, donnee locale) — Nettoyer le résidu de test local (squad `sq_15c462211708c632`, défi `sc_e0997701f9a5e98a`)

---

## V721-09 · `PLAN_CROSS_TITLE_ARCS_2026-07.md` — analyser, corriger, traiter

**État vérifié : 0 % exécuté**, l'état des lieux du plan est exact. Le prérequis
« arrivée du 2e titre » est **déjà satisfait** (`halo_5` actif).

- [x] 09.1 — Retirer la ligne contradictoire du plan (« à confirmer » vs décision actée le 18/07) — + 2 accroches oubliées ajoutées au plan
- [x] 09.2 — Step `drop_arc_titles` + ajout à `canonicalOrder` (garder l'entrée historique `create_arc_titles_join` : c'est un enregistrement de `schema_migrations`, pas du code mort exécutable) — + `stepDependencies` créateur→dropper, `ApplyBackfill` du créateur retiré
- [x] 09.3 — Supprimer `ArcTitlesRepo` + implémentation + invariant `Create` + `creditTitlesFor` (callers → `[challenge.TitleSlug]`)
- [x] 09.4 — Supprimer les 3 fichiers de tests dédiés + l'entrée `"arc_titles"` de `order_audit_test.go:156` — remplacée par l'assertion inverse (table ABSENTE)
- [x] 09.5 — **Ratchet Phase 2** : `internal/prestige/no_cross_title_aggregation_test.go` (scan AST, 4 détecteurs, allowlist datée) — morsure prouvée sur le code exact retiré
- [x] 09.6 — Documenter la règle dans le skill `arch-rules`
- [x] 09.7 — Audit de lecture + clôture/archivage du plan (`.ai/archive/PLAN_CROSS_TITLE_ARCS_2026-07.md`)

**Découvertes non traitées (décision produit requise)** : `GetUserPrestigeCrossTitle` somme
les PP tous titres confondus (atteignable par `GET /prestige/me` sans `title_slug`) et
`GetLeaderboard(titleSlug=nil)` fait de même — cette dernière **sans aucun appelant de
production** (code mort). Les deux contredisent la décision produit du 2026-07-18 mais ne
sont pas des résidus d'`arc_titles` → allowlistées avec justification datée dans le ratchet.
Autre gap relevé : `wire.NewPrestigeBundle` épingle `titlePkg.DefaultSlug` → tout Prestige
est rattaché à Halo Infinite quel que soit le titre de la requête.
**Restes à faire par le pilote** (fichiers réservés) : entrée `.ai/thought_log.md` et retrait
de la ligne « Arcs multi-titres » de `.ai/BACKLOG.md:187`.

**Risque à signaler** : la migration `DROP TABLE` s'exécute au boot sur toutes les DB joueur.
Aucune donnée perdue (miroir 1:1 de `arc.title_slug`), mais à annoncer avant tout déploiement.

---

## V721-10 · `PLAN_REVUE_ANALYTIQUE_TIMESERIES_SQUAD_2026-07.md` — analyser, corriger, traiter

**État vérifié : 0 % exécuté, et ~40 % du contenu est caduc ou contredit par du code livré
depuis le 23/07.** Le plan doit être RÉÉCRIT avant exécution — le lancer tel quel ferait
perdre du temps sur des items sans objet et rouvrirait un bug corrigé le 24/07.

| Item | Correction établie |
|---|---|
| B2 rendement combat | **Prémisse invalide** : aucun composant orphelin. Il faut exposer `offensive_conversion`/`defensive_resistance` dans le contrat timeseries — effort S → M |
| B3 radar | Recalibrage Score **déjà fait** (`642ef31f8`). Ne reste que le tooltip valeur brute |
| B4 heatmaps | **Sans objet** : remplacées par le profil médian + enveloppe P25–P75 |
| B5 axes WR/MMR | Défaut réel sur Timeseries **seulement** — le chart Escouade est déjà correct. 1 badge, pas 2 |
| B7 / DEC-6 | **Contredit** par `ea84e242f` (« historique en composition exacte », + ratchet `no_raw_squad_intersection_test.go`). Décision à rouvrir |
| F1a dominance | Sous-estimé : **2 DTOs** à étendre, pas « zéro nouveau calcul » |

- [x] 10.1 — Réécrire le plan (statuts `[~]`/`[!]`, effort révisé, items caducs fermés)
- [x] 10.2 — Lot A : outillage badges de revue (prérequis de la tournée visuelle)
- [x] 10.3 — B1 : exposer `avg_life_seconds` et cesser d'utiliser le proxy `time_played/(deaths+1)`
- [x] 10.4 — B5 : axe % dédié [0,100] + axe MMR séparé
- [x] 10.5 — B3(3) : tooltip radar valeur brute
- [x] 10.6 — F1 : `dominance_flag` sur les 2 DTOs + losange sur la tape
- [!] 10.7 (ABANDONNE, arbitrage utilisateur « option c ») — B2 : selon arbitrage utilisateur
- [!] 10.8 (UTILISATEUR) — Tournée visuelle utilisateur + retrait des badges

**Décisions utilisateur requises** : DEC-6 (baseline « vs historique » escouade), sort de B2.

---

## V721-11 · « Pourquoi un dossier et des références openspartan ? »

**Réponse factuelle** : ce n'est **pas** un reliquat. 743 occurrences, **zéro code mort**,
aucune hors du périmètre strict de la feature d'import (paquet dédié + service + handler +
UI onboarding + commentaires expliquant ses effets de bord dans le pipeline). Le package est
vivant, testé (73 % de couverture), sécurisé (`RequireAuth`), et a été mis en conformité
anti-ART récemment (`0a27412f7`). Le nom vient du projet communautaire OpenSpartan Workshop
(crédité dans `README.md:294` et `docs/ACKNOWLEDGMENTS.md:6`), dont la feature importe la
base SQLite — l'usage exact que l'utilisateur juge légitime.

**Seul point discutable** : `internal/openspartan/halo_api_payload.go` décrit des payloads de l'API
Halo officielle sous un package « openspartan ». Sans effet de bord (ces types ne sortent
jamais du duo `openspartan`/`mapper`), mais trompeur au premier coup d'œil.

- [x] 11.1 — Répondre à l'utilisateur dans Notion (faits + options)
- [x] 11.2 — Ajouter un paragraphe dans un guide d'architecture : la feature n'est documentée nulle part hors code + thought_log, **c'est la vraie cause de la question**
- [x] 11.3 — Renommer `models.go` → nom fidèle (cosmétique, 0 import externe cassé) — **si l'utilisateur le souhaite**

---

## V721-12 · Triage et nettoyage du backlog

**Réponse utilisateur** : « Quels items du backlog peuvent être faits ? Hors Tauri, Kills
environnementaux et housekeeping. S'ils sont faisables maintenant, je veux que tu les fasses
pour cette v7.2.1. Si des choses terminées sont faites, nettoyer le backlog. »

**4 entrées à supprimer (déjà livrées, preuves vérifiées)** :

| Entrée | Preuve |
|---|---|
| Unifier la SOURCE des noms d'armes | `weapon_resolver.go:76-94`, `weapon_names.toml` ×2, garde-rail `weapon_names_completeness_test.go`, commit `cebe2fed9` |
| Sunburst niveau 2 classe Grenade | `fragdist.go:226-258` (`grenadeRoles`), i18n `frags.toml:124-146`, commit `0eb523bb2` |
| H5 double-comptage sunburst | `fragdist.go:93-107` (retrait `MechanicKills`), commit `0eb523bb2` |
| Retrait du fallback LIVE Match view | `match_view_service.go:229-244`, commit `468154424` — déjà auto-marqué CLOS, à déplacer en « Récemment complété » |

Tous les autres items faisables sont **traités par les items V721 ci-dessus** (objectifs →
V721-02, citations → V721-03, openapi → V721-04, armes → V721-05).

- [x] 12.1 — Supprimer les 4 entrées livrées, les consigner en « Récemment complété »
- [x] 12.2 — Corriger l'entrée `seedWeapons` (prémisse fausse, cf. V721-05)
- [x] 12.3 — Ajouter l'entrée datée `echarts` 5→6 (cf. V721-01)

---

## V721-13 · [FINAL] Documentation de release depuis v7.2.0

- [x] 13.1 — Sections « What's new » des README FR **et** EN (public **end-user**, orienté usage) — `README.md` + `docs/FR/README.md`, bloc condensé (2 sous-sections, 5 puces) inséré en tête, ligne de version `v7.2` → `v7.2.1`. Badges de version laissés à 7.2.0 (précédent `cc9d82b2d` : bump au tag).
- [x] 13.2 — Changelogs FR et EN (public technique, granularité fine) — section `## [7.2.1] - 2026-07-25` dans `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`, format Keep a Changelog des entrées 7.1.0/7.2.0 (Ajouté API Go / Ajouté React-TS / Modifié / Corrigé / Supprimé / Ops).
- [x] 13.3 — Fichier des notes de version in-app — `docs/RELEASE_NOTES.md` + `docs/FR/RELEASE_NOTES.md` (source unique de `service.ReleaseNotesService`). **Contrainte de parsing** : `extractVersionKey` tronque à `major.minor` et `extractWhatsNewBlocks` garde la 1re occurrence d'une clé → un bloc `v7.2.1` empilé aurait effacé le bloc `v7.2` des notes in-app. Forme retenue : UN bloc pour le train 7.2, ligne de version renommée `v7.2.1`, sections v7.2.1 insérées en tête. Aucun code Go touché.
- [x] 13.4 — Vérifier la parité FR/EN (règle 15 du CLAUDE.md) — même découpage, même nombre de puces et même nombre de lignes ajoutées par paire (28/28 notes, 11/11 README, 62/62 changelog) ; libellés FR des modes repris du référentiel `mode_playlist_fr.go` (Stockpile → « Stockage »).

**Omissions délibérées** (anti-survente) : visuels définitifs des 2 citations `flag_captures` /
`flag_steals` (une ligne factuelle en `Ops` du changelog, rien côté joueur) ; modes Elimination
et Infection (documentés `Ajouté` comme non spécifiés faute de payload + condition de reprise) ;
**purge de `weapons.name_fr` (05.1)** — omise du changelog à la rédaction, à raison : la
vérification sur pièces ne trouvait ni `steps_metadata_purge_weapons_name_fr.go`, ni le nom
d'étape dans `internal/migration/order.go`.

> **RÉSOLU le 2026-07-26 (pilote).** Cause : l'agent V721-05 avait écrit ses 5 fichiers dans
> le **dépôt principal** (branche `main`) au lieu du worktree — d'où leur absence de la
> branche. C'est cette omission de changelog qui a levé le lièvre. Fichiers portés dans la
> branche, les 2 lignes de `order.go` / `order_dependency_test.go` réinsérées SANS écraser
> les ajouts des autres agents (`drop_arc_titles`, objectifs), dépôt principal restauré
> propre. Les 5 tests d'intégration de la purge passent, dont
> `TestPurgeWeaponsNameFR_RemovesColumnWhenPresent` (base créée AVEC la colonne) et
> `..._PreservesRowsAndPrimaryKey`.
>
> **Rectification d'une preuve invalide** : la « vérification sur copie des vraies bases »
> annoncée pour cette purge était un FAUX POSITIF — la `metadata.duckdb` locale n'avait
> DÉJÀ PLUS la colonne avant migration (`weapons_a_name_fr = 0` à l'état AVANT). Seuls les
> tests d'intégration démontrent la purge. Les deux autres vérifications sur copie (18
> colonnes d'objectifs, 88 → 98 citations) restent valides : elles ont bien observé une
> transition d'état.
>
> Ligne `Supprimé`/`Removed` à ajouter aux deux changelogs — cf. item 13.5 ci-dessous.

- [x] 13.5 — Ajouter la purge `weapons.name_fr` en `Supprimé` / `Removed` des deux changelogs

---

## Ordonnancement retenu

Contrainte transverse : **aucune commande Go concurrente** (corruption de cache) — les gates
Go se jouent en série. Les agents de reconnaissance/rédaction peuvent tourner en parallèle.

1. **Vague 1 — clôtures à faible risque, zéro conflit de fichiers** : V721-07, V721-08, V721-11, V721-12
2. **Vague 2 — backend isolé** : V721-09, V721-04, V721-05, V721-06, V721-01
3. **Vague 3 — données** : V721-02, puis V721-03 (dépend des arbitrages visuels)
4. **Vague 4 — UI** : V721-10 (réécriture du plan d'abord, puis exécution)
5. **Vague 5** : V721-13 (en dernier — il documente tout ce qui précède)

---

## Découvertes — REQUALIFIÉES EN LOT À TRAITER (décision utilisateur 2026-07-25 soir)

> L'utilisateur a demandé explicitement que les découvertes consignées soient traitées.
> Elles sortent donc du régime « règle 7 » (noter, ne pas traiter) et deviennent le lot
> **V721-14**. Consolidation des rapports des 12 agents + du gate.

### Déjà traitées en cours de chantier (4)

| # | Découverte | Traitement |
|---|---|---|
| D-01 | 5 lectures plates dans `prestige_social_repo.go` — même trou que la cause du 500 | Corrigées + ratchet `bare_db_recovery_routing_test.go` (V721-08b) |
| D-02 | Ratchet existant aveugle aux repos à champ `db *duckdb.DB` nu | Nouveau ratchet, mordant prouvé sur les 11 violations d'origine |
| D-03 | Chaîne 503 cassée sur rejoindre/abandonner un défi (400 au lieu de 503) | Cause conservée dans la chaîne d'erreur (2e `%w`) |
| D-04 | **3 routes catalogue montées SANS garde d'auth**, avec `?xuid=` + `only_played` → fuite d'activité d'un joueur, sans authentification. Invisible au ratchet parce que le harnais de démo ne les montait jamais | `RequireAuth` sur les 2 branches (aucun appelant front, vérifié) |

### Lot V721-14a — correction (effort S/M, faisable immédiatement)

| # | Découverte | Effort |
|---|---|---|
| D-05 | `GetUserPrestigeCrossTitle` scanne `MAX(updated_at)` dans un `time.Time` : NULL pour un joueur sans PP → erreur de scan → **500** sur `GET /prestige/me` sans `title_slug` | S |
| D-06 | `wire.NewPrestigeBundle` épingle `titleSlug := DefaultSlug` pour `shared_social` ET `metadata` → **tout Prestige est rattaché à Halo Infinite quel que soit le titre demandé** (les PP `halo_5` atterrissent dans le `shared_social` d'Infinite) | M |
| D-07 | `CountActiveParticipants` : zéro appelant de production (code mort, règle 7) | S |
| D-08 | `PrestigeRepo.GetLeaderboard(titleSlug=nil)` : branche cross-titre sans appelant, en plus interdite par le ratchet | S |
| D-09 | `buildCorrelationPoints` est un **3e consommateur** du proxy de durée de vie — l'histogramme et le nuage racontent deux histoires de la même métrique | S — **[x]** confirmé sur pièces (`buildCorrelationPoints`, `internal/service/timeseries_service_tabs.go`, calculait bien son propre proxy `time_played/(deaths+1)` en dur au lieu de réutiliser le helper). Rebasé sur `matchAvgLifeSeconds` (`timeseries_service_buckets.go`, même helper que `buildLifeBuckets` — même priorité valeur réelle/repli, même télémétrie Debug fallbacks/used/total, ajoutée côté nuage). Doc du champ `LifeBuckets` (`domain/timeseries.go`) corrigée en cohérence. `buildCorrelationPoints` extrait vers `timeseries_service_correlation.go` (CLAUDE.md règle 5 : l'ajout de la télémétrie faisait passer `timeseries_service_tabs.go` à 517 L, > seuil 500 — god-file split, pas de nouvelle fonctionnalité). Tests `TestBuildCorrelationPoints_LifespanPrefersRealValue` / `..._LifespanFallbackWhenRealMissing` (déplacés avec les tests `buildCorrelationPoints` existants dans `timeseries_service_correlation_test.go`, même miroir) distinguent valeur réelle et repli — échouent sans le correctif |
| D-10 | `MatchHeader.card.tsx` garde son propre `DOMINANCE_TOKENS` → 2e copie, à migrer vers `lib/narrative/dominance.ts` | S — **[x]** confirmé sur pièces (`DOMINANCE_TOKENS` local dupliquait exactement `DOMINANCE_COLOR_TOKENS`). Migré vers le module central (+ `asDominance` pour normaliser le flag) ; constante locale supprimée (règle 7) ; aucune 3e copie résiduelle (grep sur tout `apps/web/src`). Garde-rail neuf : `lib/narrative/dominance.guard.test.ts` interdit toute redéfinition locale de `DOMINANCE_TOKENS` |
| D-11 | Commentaire faux dans `stats_canonical.go` (OC/DR « restent nil » alors qu'ils sont calculés) | S — **[x]** confirmé sur pièces (`OffensiveConversion`/`DefensiveResistance` sont bien calculés via `ComputeCombatYield` dans `StatsMatchRowFromCanonical`, lignes 96-116 ; seul `MedalExploitScore` reste effectivement nil dans ce converter). Commentaire corrigé pour distinguer les deux cas |
| D-12 | Bloc de doc mal placé dans `seed.go` : documente `defaultCitationMappings` mais surmonte `SeedMedalDefinitions`, dont il couvre le `//nolint` par accident | S — **[x]** confirmé sur pièces (nolint dans `seed.go` ne peut de toute façon pas traverser vers `seed_citation_data.go` : inerte pour `defaultCitationMappings`, et sans effet réel sur `SeedMedalDefinitions` qui ne dépasse aucun seuil). Bloc de doc + `//nolint:funlen,maintidx` déplacés au-dessus de `func defaultCitationMappings()` (`seed_citation_data.go`) avec note sur `maintidx` non activé dans `.golangci.yml`. `SeedMedalDefinitions` recoit un commentaire propre décrivant son comportement réel (vérifie + compte, n'insère plus rien). Aucun retrait de nolint actif : gate non affecté |
| D-13 | `internal/openspartan/doc.go` référence `.ai/SPRINT_OPENSPARTAN_IMPORT.md`, supprimé | S — **[x]** confirmé (fichier absent du worktree, racine `.ai/` et `.ai/archive/`). Référence remplacée par `docs/ARCHITECTURE_V6.md` (section « OpenSpartan import », lignes 56-58, + pendant FR confirmé) |
| D-14 | `docs/COMMENDATIONS_REFERENCE.md` + pendant FR **périmés depuis v7.2** : annoncent 88 citations (98 désormais) et décrivent 4 d'entre elles avec l'ancien mécanisme | S — **[x]** confirmé sur le seed (98 `Norm:` dans `seed_citation_data.go`, 4 constantes `citationNorm{Charge,GotYou,Stakeholder,FlagCarrierHunter}` basculées `objective_stat`, 10 constantes `citationNorm*` v7.2.1 ajoutées à la section « Mode de jeu »). `COMMENDATIONS_REFERENCE.md` : compte 88→98, table « PvP — Game mode » 11→21 lignes (4 corrigées `award`→`objective_stat` + 10 nouvelles), note de mécanisme v7.2. `FR/CITATIONS_REFERENCE.md` (redirection pure vers l'EN) : compte 88→98 dans la seule phrase qui le citait |
| D-15 | `ObjectiveAggregate` non étendu aux 3 nouveaux modes (Stockpile/Extraction/VIP) — aucune surface UI aujourd'hui, mais l'asymétrie est une dette | M |

#### Statuts — découvertes Prestige D-05 à D-08 (traitées le 2026-07-25)

**[x] D-05 — `GetUserPrestigeCrossTitle` : 500 pour un joueur sans PP. CONFIRMÉ, CORRIGÉ.**
Vérifié sur pièces : `platform/duckdb/prestige/prestige_social_repo.go` scannait
`MAX(updated_at)` (agrégat, donc NULL sur un ensemble vide) directement dans
`prestige.UserPrestige.UpdatedAt` (`time.Time`, `types.go:136`). Chemin HTTP réel :
`GET /prestige/me` sans `title_slug` → `handlers/prestige.go:382` → `service.go:649`
(`if titleSlug == ""` → voie cross-titre) → scan en erreur → aucune sentinelle reconnue
par `serviceError` → branche `default` = **500**. Correctif : scan en `sql.NullTime` puis
conversion ; joueur sans PP → prestige vide sans erreur, même contrat que
`GetUserPrestige`. Audit des autres agrégats du fichier : `COALESCE(SUM(total_pp), 0)`
(jamais NULL) et `COUNT(*)` de `CountActiveParticipants` (jamais NULL, méthode supprimée
par D-07) — aucun autre site à risque. Test :
`prestige_social_cross_title_test.go` (`//go:build cgo`, donc gate par défaut),
`TestGetUserPrestigeCrossTitle_NoPP_ReturnsEmptyNotError` (mordant : le code pré-fix
échoue au `Scan`) + `..._WithPP_KeepsUpdatedAt` (contrôle négatif : la date n'est pas
écrasée quand elle existe).

**[!] D-06 — Prestige épinglé au titre par défaut. CONSTAT CONFIRMÉ ; CORRECTIF NON FAIT
(décision motivée : le correctif propre exige une décision produit, un demi-correctif
serait pire).**

*Faits établis.* `wire/prestige_setup.go` fixe `titleSlug := titlePkg.DefaultSlug` et
ouvre AVEC lui `shared_social.duckdb` ET `metadata.duckdb`. Portée exacte :

| Donnée Prestige | Base | Isolée par titre aujourd'hui ? |
|---|---|---|
| arcs, défis, moments, télémétrie, baselines | player DB (`pdb.Player`) | OUI — `ServiceRegistry.resolve` lit `ctxkeys.TitleSlug` (`wire/registry.go:228`) |
| `prestige_events`, `user_prestige_history` | shared_social du titre PAR DÉFAUT | NON (chemin) — mais chaque ligne porte le vrai `title_slug` et les lectures filtrent dessus |
| `squad`, `squad_member`, `squad_challenge`, participants | shared_social du titre PAR DÉFAUT | NON |
| `challenge_template`, `preset_arc` (catalogue) | metadata du titre PAR DÉFAUT | NON |

Donc : **violation de l'isolation par CHEMIN (ADR 0008), pas fusion sémantique** — aucun
calcul ne mélange deux titres (hors les deux voies cross-titre connues), les PP `halo_5`
atterrissent physiquement dans le `shared_social` d'Infinite avec `title_slug='halo_5'`.

*Deux blocages structurels, vérifiés sur pièces.*
1. **Catalogue.** `challenge_template` / `preset_arc` sont un référentiel possédé par Halo
   Infinite (`internal/games/halo_infinite/migrations/{steps.go,prestige.go}`, seedé depuis
   `config/titles/halo_infinite/{challenges,arcs}/*.toml`). Le set de migrations Halo 5
   possède son target `metadata` (`OwnsTarget == TargetMetadata`,
   `games/halo_5/migrations/metadata.go:315`) et ne crée PAS ces tables — c'est un
   invariant **testé** : `games/halo_5/migrations/metadata_test.go:88-89` échoue si
   `challenge_template`/`preset_arc` apparaissent dans la metadata h5 (« POLLUTION »).
   Et `config/titles/halo_5/` n'a ni `challenges/` ni `arcs/` : il n'existe aucun
   catalogue Prestige pour un 2e titre. Ouvrir la metadata du titre courant casserait
   donc toute lecture de catalogue hors titre par défaut.
2. **Escouades.** `squad` et `squad_member` n'ont AUCUNE colonne `title_slug`
   (`games/halo_infinite/migrations/steps_shared_social.go:177-189`) : une escouade
   n'appartient à aucun titre. Isoler `shared_social` par titre **scinderait les escouades
   elles-mêmes** (une escouade créée depuis Infinite disparaîtrait depuis Halo 5) et
   orphelinerait les `squad_challenge` (qui, eux, portent un `title_slug`). Ce n'est pas un
   bug de câblage : le modèle de données de la couche cross-joueurs n'est pas per-titre.

*État réel des données : NON ÉTABLI.* `data/titles/halo_infinite/warehouse/shared_social.duckdb`
est tenue RW par le serveur local (lecture directe refusée : « Device or resource busy »),
et aucune CLI DuckDB n'est disponible dans l'environnement. Estimation par le code :
le volume de PP `halo_5` doit être nul ou marginal — le catalogue h5 étant vide,
`SuggestTemplates(halo_5)` ne rend rien et l'UI ne peut proposer aucun objectif ; seule la
création libre (`template_id` optionnel, `handlers/prestige.go:148`) permettrait un défi h5,
et son évaluation post-sync ne tourne que si le joueur h5 passe par le cycle V2
(`sync/v2/cycle.go:294`, `p.TitleSlug`) — le chemin `TitleSyncRunner`/`liveRunner` utilise
`ProgressionAfterSync`, explicitement câblé SANS le PrestigeBundle (`config/config.go:80-87`).
**À vérifier avant toute migration** :
`SELECT title_slug, count(*), sum(total_pp) FROM user_prestige_latest GROUP BY 1` et
l'équivalent sur `prestige_events`, dans le shared_social d'Halo Infinite.

*Options chiffrées (aucune exécutée).*
- **Option 1 — bundle par titre (correctif « complet »)** : `map[slug]*PrestigeBundle`
  construit à la demande, résolution depuis `ctxkeys.TitleSlug` dans `LazyPrestigeService`,
  `Close()` de tous les bundles, `RunPostSync` routé par titre. Effort L (câblage) **+
  BLOQUÉ** par les deux points ci-dessus : demande de créer un catalogue Prestige par titre
  (schéma + TOML + seed + levée de l'invariant anti-pollution h5) et de trancher le statut
  des escouades (per-titre ou cross-titre). **Décision produit requise.**
- **Option 2 — n'isoler que `shared_social`** : effort S/M, mais c'est le demi-correctif
  interdit : catalogue Infinite + événements per-titre = asymétrie nouvelle, escouades
  scindées, et les lignes déjà écrites deviennent invisibles.
- **Option 3 — statu quo documenté (RETENUE)** : l'épinglage est désormais décrit, daté et
  justifié dans `wire/prestige_setup.go` (bloc « ÉPINGLAGE AU TITRE PAR DÉFAUT ») avec
  renvoi vers cette entrée, pour qu'aucune relecture future ne le prenne pour un oubli.
- **Coût d'une migration de données (si l'option 1 est un jour tranchée)** : copier vers le
  shared_social du titre les lignes `title_slug = <slug>` de `prestige_events` et
  `user_prestige_history` (append-only : recopier l'historique complet, pas seulement
  `_latest`, sinon les vues `_latest` du titre cible reconstruisent un carry-forward faux),
  plus les `squad_challenge` du titre et leurs participants — mais SANS pouvoir déplacer
  `squad`/`squad_member`, qui n'ont pas de titre. Non écrite, comme demandé.

**[x] D-07 — `CountActiveParticipants` : code mort. CONFIRMÉ, SUPPRIMÉ.**
Grep exhaustif du dépôt avant suppression : 1 déclaration d'interface
(`prestige/repository.go:150`), 1 implémentation (`prestige_social_repo.go:485`), 2 stubs
de test (`service_full_test.go`, `service_coverage_test.go`) et 1 étape du test de reprise
(`prestige_social_reopen_test.go`, étape 5) — **zéro appelant de production**. Les 5 ont été
retirés (règle 7) ; l'en-tête du test de reprise dit désormais pourquoi la 5e lecture a
disparu. Grep de contrôle après suppression : plus aucune occurrence hors ce commentaire.

**[x] D-08 — `GetLeaderboard(titleSlug=nil)` : branche cross-titre morte. CONFIRMÉ, SUPPRIMÉE.**
Vérifié : `GetLeaderboard` n'a AUCUN appelant de production (interface + implémentation +
1 test d'intégration + 2 stubs ; la page « Leaderboard PP » du front est un placeholder qui
ne fetch rien — `apps/web/src/features/prestige/LeaderboardPPPage.tsx:23-25`). La branche
`SUM(...) GROUP BY user_id` est supprimée et le paramètre passe de `*string` à `string` :
la voie cross-titre n'est plus exprimable à la compilation ; un slug vide est refusé
(`prestige.ErrInvalidInput`) au lieu d'être dégradé en « tous titres ». Morsure runtime :
`TestPrestigeSocialRepo_Leaderboard_RejectsEmptyTitle` (échoue si un classement est rendu
au lieu d'une erreur) + `TestPrestigeSocialRepo_Leaderboard_PerTitle` enrichi d'un 2e titre
pour le même joueur (échoue si les PP des deux titres se somment). **Allowlist du ratchet :
rien à retirer** — les 2 entrées de `allowlistCrossTitle` visent `GetUserPrestigeCrossTitle`
(`repository.go` + `service.go`), toutes deux toujours présentes, donc
`TestCrossTitleAllowlistNotStale` reste vert ; la branche `GetLeaderboard` vivait dans
`platform/duckdb/prestige/`, hors périmètre de scan. Le commentaire « PORTÉE » du ratchet,
qui décrivait deux agrégations cross-titre survivantes, est mis à jour (doc inversée).

*Découverte annexe (non traitée, règle 7)* : **D-20** — `PrestigeRepo.GetLeaderboard` est
mort dans son ENTIÈRETÉ, pas seulement sa branche cross-titre (aucun appelant de
production ; seul un test d'intégration l'exerce). Non supprimé : le périmètre confié
portait sur la branche, et la surface UI « Leaderboard PP » est livrée en placeholder —
supprimer le repo entier est un arbitrage produit (abandonner ou finir la feature).

### Lot V721-14b — à arbitrer / différé (justifié)

| # | Découverte | Pourquoi différé |
|---|---|---|
| D-16 | Halo 5 possède entièrement `TargetMetadata` : la purge de `weapons.name_fr` n'atteint pas une `metadata.duckdb` H5 provisionnée avant V72-06 | Même limite que le seul précédent du dépôt ; sans impact fonctionnel (colonne inerte) |
| D-17 | ~25 accès DB plats subsistent dans la racine `internal/platform/duckdb/` (sinks de persistance, auth, notifications, catalogues) — ratchet volontairement borné à `prestige/` | Une allowlist de 25 entrées ferait un ratchet décoratif ; à traiter par sous-paquet |
| D-18 | Duplication de la connaissance du schéma JSON Halo : typée dans `internal/openspartan/halo_api_payload.go`, re-parsée en `map[string]any` dans `internal/sync/transforms_helpers.go` | Effort L, risque moyen-élevé (pipeline de sync critique) |
| D-19 | Les 35 routes nouvellement publiées arrivent avec les descriptions par défaut de Huma, sans réponses d'erreur documentées | Aucune source FR à recopier ; enrichissement du fragment à faire sciemment |

---

## Découvertes résiduelles (hors périmètre, non requalifiées)

- **Prod injoignable + `v7.2.0` non déployée** (cf. « Hors périmètre »). Deux causes annexes
  relevées dans le log de deploy : le fix de dédup du build Go `9f0f7d48b` n'a pas pris (les
  2 images se buildent toujours séparément, args `VERSION` différents : `v7.2.0` vs `dev`),
  et l'image démo porte encore `main.version=dev`.
- **Duplication de connaissance du schéma JSON Halo** : typé dans `internal/openspartan/halo_api_payload.go`,
  reparsé en `map[string]any` dans `internal/sync/transforms_helpers.go:79`. Vraie dette,
  effort L, risque moyen-élevé (pipeline sync critique). Candidat v7.3.
- **6 handlers supplémentaires** répondant 201/202 (`backfill.go:148`, `settings.go:549/638`,
  `sync_handler.go:575/658`, `user_auth.go:246`) sans correction dans le fragment manuel —
  à confirmer sur pièces pendant V721-04.
- **V721-08b — même trou de routage recovery sur `shared_social` (TRAITÉ).** Après le correctif
  `metadata` (V721-08.1), `platform/duckdb/prestige/prestige_social_repo.go` portait 5 lectures
  mono-ligne PLATES (`r.db.QueryRow`) sur la handle RW `shared_social` : `GetUserPrestige`,
  `GetUserPrestigeCrossTitle`, `SquadRepo.Get`, `SquadChallengeRepo.Get`,
  `CountActiveParticipants` — soit un 500 permanent (jusqu'au restart) sur le prestige du
  joueur et les lectures d'escouade dès qu'un Reopen concurrent invalide la handle. Routées
  vers `QueryRowRecovered` ; contention fichier traduite en `dblease.ErrDBLocked` (→ 503 +
  `Retry-After`) par `translateSocialLockErr` (`prestige_social_recovery.go`, 2e copie du
  patron ; centralisation + garde-rail à la 3e). Régression verrouillée par
  `prestige_social_reopen_test.go` et par un ratchet structurel
  (`bare_db_recovery_routing_test.go`) qui interdit les accès plats sur un champ NU
  `db *duckdb.DB` dans le sous-paquet `prestige/` — le trou par lequel les DEUX incidents sont
  passés, `player_db_recovery_routing_test.go` ne voyant que les formes `pdb.Player.Query(`.
  Découvertes annexes **non traitées** (hors périmètre) :
  1. `internal/prestige/service_squad_challenges.go:115` et `:154` cassent la chaîne d'erreur —
     toute erreur de `SquadChallenges.Get` (y compris `ErrDBLocked`) est ré-emballée en
     `ErrInvalidInput` → HTTP **400** au lieu de 503 sur join/abandon d'un défi d'escouade
     (fichier réservé à un autre agent). Les 2 autres callers (`GetSquadChallenge`,
     `EvaluateSquadChallenge`) et `Squads.Get` propagent en `%w` : chaîne intacte.
  2. `GetUserPrestigeCrossTitle` scanne `MAX(updated_at)` dans un `time.Time` : NULL pour un
     joueur sans aucun PP → erreur de scan → 500 sur `GET /prestige/user` sans `title_slug`.
     Antérieur au correctif, comportement inchangé.
  3. `SquadChallengeRepo.CountActiveParticipants` n'a **aucun caller de production** (interface
     + repo + stubs de test seulement) — candidat suppression (règle 7).

---

## Journal d'exécution

_(rempli à chaque clôture d'étape)_

- **2026-07-25** — Ouverture. Worktree `v721-notion` depuis `main` local (`9f0f7d48b`).
  Reconnaissance 6 agents rendue. Blocages levés sur V721-02 (payloads capturables : 39
  matchs Stockpile, 2 Extraction, 2 Attrition, 1 Infection UGC) et V721-06 (chemins CMS
  déductibles, pas de sourcing externe nécessaire). Incident prod signalé, mis hors périmètre
  sur décision utilisateur.
- **2026-07-25** — **V721-02 items 02.2 à 02.6 CLOS** (code écrit, gates joués par le pilote
  en passe sérialisée). Périmètre élargi en cours de route sur arbitrage utilisateur :
  3 blocs / **18 colonnes** (Stockpile 6 + Extraction 5 + **VIP 7**) au lieu de 2/11 — le
  bloc `VipStats`, découverte du P0, coûtait un delta quasi nul dans la même migration
  alors qu'un report imposait une 2e migration. `VipStats` ajouté à `StatsBundle` (il y
  manquait). 02.7/02.8 statués `[!]` : Elimination et Infection non spécifiables faute de
  payload (aucun match de ces modes en base) — condition de reprise écrite.
  **Reste au pilote** : `make openapi-gen` + `make generate-types`, la passe de gates, puis
  le reset ciblé du bit `MBitObjectiveStats` avant de relancer
  `cmd/backfill_objective_stats` en LOCAL.
