# Audit données de prod — LOT V9a (2026-07-07)

> Source : copie restaurée par V10a depuis restic `latest` (snapshot **9e96ed20**,
> 2026-06-27) vers `C:\Users\Guillaume\Downloads\Scripts\LevelUp-prod-copy\`.
> Toutes les lectures faites en READ-ONLY via `apps/go-api/cmd/tmpdbq` (driver DuckDB Go,
> `access_mode=read_only`). Aucune écriture sur la copie pendant V9a.
> **Verdict global : la copie prod (au 2026-06-27) est propre sur les 3 dettes DATA
> connues (TZ, is_ranked, intégrité). Les correctifs code correspondants sont déjà en
> place et testés. Restent : gaps LUSR d'intérieur (~0,8 %), 3 colonnes à DROP différé,
> médailles orphelines H5 (bruit d'ingestion), v3 jamais déployé.**

## Contexte : ce qui a déjà été corrigé AVANT le backup

Le backup date du 2026-06-27. Or les correctifs suivants ont été déployés sur prod avant :
- **TZ `first_joined_time`** : le backfill `cmd/backfill_first_joined_tz --commit` a déjà
  tourné en prod (les ~964 matchs décalés historiques sont corrigés).
- **`is_ranked` OpenSpartan** : (a) fix import-time dans
  `openspartan_import_service.go::writeOneMatch` (RankRecap présent ⟹ `is_ranked=true`,
  l.317-320) + test `openspartan_import_service_test.go:261-270` ; (b) migration boot
  `shared_backfill_is_ranked_and_season` + seed autoritatif ranked-playlists
  (`ranked_playlists.go`) ont reflaggé l'historique.
- **Désync watermark LUSR v2 (dominante)** : corrigée 2026-06-07 (`canonicalGate` +
  owner-only + fix `concurrentTeamSize`), branche `fix/enrichment-convergence`.

Conséquence : l'audit sur la copie mesure l'état POST-correctifs. Les chiffres attendus des
mémoires (964 matchs TZ, matchs classés non flaggés) sont **historiques**, pas l'état actuel.

---

## Tableau récapitulatif par item et par titre

| # | Item | Halo Infinite | Halo 5 |
|---|------|---------------|--------|
| 1 | Matchs `first_joined_time` décalés (offset Europe/Paris, T0>120s) | **0** (max T0 apparent = 118s) | **N/A** — `first_joined_time` non peuplé (0 / 24208 participants) |
| 2 | Matchs classés marqués `is_ranked=false` (signal RankRecap = `match_csrs`) | **0** (34 sur playlist classée, tous flaggés ; 0 CSR-porteur non flaggé) | **0** (0 CSR-porteur non flaggé ; 1601/3032 flaggés) |
| 3 | Orphelins participants / match_registry | **0** | **0** |
| 3 | Orphelins medals_earned / participant | **0** | **2149** (1190 xuid vide + 959 xuid non-participant ; bruit ingestion H5, cf. §Découvertes) |
| 3 | Doublons match_id (registry) | **0** | **0** |
| 3 | Doublons participants (match_id,xuid) | **0** | **0** |
| 3 | `match_skill_rank` vs `_latest` cohérent (par joueur) | **cohérent** (ex. Madina 27226 raw → 1172 latest = 1172 distinct) | — (non LUSR) |
| 4 | `known_teammates_count` + `friends_xuids` présents (PME, DEC-6) | **présents** (4/4 joueurs) | **présents** (4/4 joueurs) |
| 4 | `discord_notified` présent (media_files, G5) | **présent** (shared_social) | **présent** (shared_social) |
| 5 | Désync watermark LUSR v2 (watermark vs dernière ligne rated) | **watermark = dernière ligne** pour les 4 joueurs (pas de désync de tête) ; gaps d'intérieur : Madina 7, JGtm 10, Choco 6, XxDaemon 0 | — (H5 CSR, pas LUSR) |
| 6 | Counts globaux vs V10a | **1780 matchs / 26577 participants** (2021-11-19 → 2026-06-25) = OK | **3032 matchs / 24208 participants** = OK |

---

## Détail par item

### 1. TZ `first_joined_time` (V9b)
Requête = celle du diag `cmd/backfill_first_joined_tz` (offset = epoch(start_time as-UTC)
− epoch(start_time_utc), détection T0 apparent > 120s sur `present_at_beginning`).
- **Infinite** : 0 match décalé. Distribution T0 apparent : min −7075s, **max 118s** (sous
  le seuil 120s). Le dry-run de l'outil confirme : « Matchs décalés détectés : 0 ».
  `present_at_beginning` peuplé sur 23525/26577 participants, `first_joined_time` sur
  26577/26577 → détection valide, pas un faux zéro par colonne vide.
- **Halo 5** : `first_joined_time` NULL sur 100 % des 24208 participants → la dette TZ est
  **structurellement Infinite-only** (l'ingestion H5 ne peuple pas ParticipationInfo de la
  même façon). Rien à corriger côté H5.

### 2. `is_ranked` OpenSpartan (V9c)
Signal fiable = RankRecap → matérialisé dans `shared.match_csrs`. Croisement + validation
via le catalogue autoritatif ranked-playlists (`metadata.playlists_catalog.is_ranked=true`,
16 asset_ids).
- **Infinite** : 34 matchs sur playlist classée → **tous** `is_ranked=true` ;
  0 CSR-porteur non flaggé ; 0 match flaggé hors playlist classée. `first_sync_by` = le
  gamertag du joueur (pas `openspartan_import`) sur les imports historiques.
- **Halo 5** : 1601/3032 flaggés ranked ; 0 CSR-porteur non flaggé.
- Conclusion : la donnée is_ranked de la copie est **cohérente**. Le fix code + backfill
  boot ont déjà convergé.

### 3. Intégrité
Toutes les jointures orphelines et doublons = 0, **sauf** medals_earned H5 (2149 lignes
orphelines, voir §Découvertes — pas une corruption). `match_skill_rank_latest` cohérent
(1 ligne par match).

### 4. Colonnes à DROP différé (planifiées V9d)
Toutes physiquement présentes :
- `player_match_enrichment.known_teammates_count` + `.friends_xuids` (DEC-6/G14 : la vue ne
  les projette plus, DROP différé au prochain rebuild) — 8 player DBs (4 Infinite + 4 H5).
- `shared_social.media_files.discord_notified` (G5 : feature notif Discord supprimée
  end-to-end ; colonne encore créée par `ops/media_store.go` l.48/71) — 2 titres.

### 5. Watermark LUSR v2 (V9a5)
`player_skill_state_v2_latest.last_match_at` (shared) vs `MAX(start_time)` des lignes
`match_skill_rank_latest` type LUSR (player DB), par joueur possédé Infinite :

| Joueur | Watermark MAX | Dernière ligne LUSR | Écart de tête | Matchs non notés (LUSR-éligibles) |
|--------|---------------|---------------------|---------------|-----------------------------------|
| Madina97294 | 2026-06-10 20:14:52 | 2026-06-10 20:14:52 | **aucun** | 7 (4 en 2026, 3 en 2025) |
| JGtm | 2026-06-25 16:59:51 | 2026-06-25 16:59:51 | **aucun** | 10 (7 en 2026, 3 en 2025) |
| Chocoboflor | 2026-06-10 20:14:52 | 2026-06-10 20:14:52 | **aucun** | 6 (5 en 2026, 1 en 2025) |
| XxDaemonGamerxX | 2026-04-13 20:53:30 | 2026-04-13 20:53:30 | **aucun** | 0 |

Lecture : la désync de TÊTE (watermark en avance sur la dernière ligne → matchs récents
`rating_type:none`) **n'existe plus** — le fix `canonicalGate` du 2026-06-07 tient. Les
23 matchs non notés (Madina 7 + JGtm 10 + Choco 6) sont des gaps d'INTÉRIEUR : le résidu
« ~0,8 % de matchs à gros turnover roster qui échouent l'EP → skip + WARN nominatif »
documenté dans la mémoire lusr_v2_watermark_row_desync (propriété de sécurité : compute
échoué = jamais de note corrompue). Récupérables via `cmd/lusr_v2_canonical_backfill
--commit` (serveur arrêté), mais **hors périmètre V9** (ni TZ ni is_ranked ne les cause ;
ce sont des échecs EP légitimes). Consigné pour information.

### 6. Counts vs V10a
Infinite 1780/26577 (2021-11-19 → 2026-06-25), H5 3032/24208 — identiques aux références
consignées par V10a au plan. Copie cohérente.

---

## V9e — `weapon_kills_v3` : chiffres + reco (DÉCISION UTILISATEUR)

**Fait déterminant** : `weapon_kills_v3` n'existe **NI en prod NI sur la branche courante**.
Le package `internal/analysis/weaponv3/`, le repo `weapon_kills_v3_repo.go` et la table
`weapon_kills_v3` vivent uniquement dans la branche/worktree non mergée
`feat/weapon-attribution-v3`. Aucune table `%v3%` dans les DBs de la copie.

**Ce qui EST servi en prod (v2, la réalité actuelle)** :
- `shared.weapon_kills` (Infinite) : **1194 / 1780 matchs couverts = 67,1 %**, 109 744
  lignes. Vue de lecture active `v_weapon_kills` = dernière génération de `weapon_kills`
  (dense_rank par match+xuid), `effective_weapon_id = COALESCE(reconciled_as, weapon_id)`.
- v3 n'améliore pas la COUVERTURE (même source d'events) ; il corrige l'ATTRIBUTION du pi
  par-match sur les lobbies churnés (mémoire weapon_attribution_v3_status : v2
  `getXuidToPI` dense 0..N-1 est faux sur quits/joins ; v3 lit le vrai pi film). Gain =
  qualité d'attribution, pas taux de couverture. L'algo v3 a été explicitement laissé
  NON FINALISÉ (« finalisation différée », sollicitation d'aide externe).

**RECO chiffrée (à trancher par l'utilisateur)** :
- **Retirer** (recommandation par défaut) : v3 n'a jamais été promu, l'algo est incomplet,
  et il n'apporte pas de couverture. Supprimer la branche + worktree
  `feat/weapon-attribution-v3` (git garde l'historique) évite un « dead-code museum » à
  tests verts. Coût = nul, bénéfice = clarté.
- **Promouvoir** : exigerait de FINIR l'algo (multi-sessions, dépend d'aide RE externe),
  puis un backfill complet + bascule `LEVELUP_WEAPON_ATTRIB=v3` + rebuild de la vue
  `v_weapon_kills` → `weapon_kills_v3`. Non justifié tant que l'attribution churn n'est pas
  un problème produit remonté.

→ **Escaladé** : aucune action code prise sur v3 en V9 (périmètre : chiffrer + reco).

---

## V9d — Plan de rebuild append-only pour exécuter les DROP différés (recette ADR 0026)

**NE PAS EXÉCUTER en autonomie — décision opérateur, serveur arrêté.**

### Colonnes visées (confirmées physiquement présentes en §4)
| Table | DB | Colonnes à DROP | Origine |
|-------|-----|-----------------|---------|
| `player_match_enrichment` | player `stats.duckdb` (×8 : 4 Infinite + 4 H5) | `known_teammates_count`, `friends_xuids` | DEC-6 / G14 |
| `media_files` | `shared_social.duckdb` (×2 titres) | `discord_notified` | G5 |

### Pourquoi un rebuild et pas un simple `ALTER … DROP COLUMN`
- `player_match_enrichment` est **append-only** (ADR 0026, PK technique `id` + `written_at`,
  lecture via `player_match_enrichment_latest`). La recette d'ajout/retrait de colonne passe
  par `internal/migration/append_only_rebuild.go` (CREATE table cible sans la colonne →
  réinsertion des lignes `_latest` → swap → recréation de la vue `_latest`). Un DROP direct
  casserait l'invariant de reconstruction.
- `media_files` (shared_social) : écritures durables via `SharedSocialPersister` +
  `CHECKPOINT` (ADR 0022). Le rebuild doit se faire **serveur arrêté** (mono-process,
  ADR 0013) et suivi d'un CHECKPOINT sinon perte WAL.

### Procédure par titre (fenêtre serveur arrêté)
1. **Pré-check** : serveur `air`/prod arrêté (aucun writer). Backup restic frais (ou copie
   `.pristine` locale) — le rebuild est irréversible sans backup.
2. **PME (par player DB)** : pour chaque `stats.duckdb`, appliquer le rebuild append-only
   ciblant `player_match_enrichment` sans `known_teammates_count`/`friends_xuids`
   (extension de `pmeColumnStage`/`enrichmentFields()` déjà nettoyée ; il reste le DROP
   physique). Vérifier post : `PRAGMA table_info` ne liste plus les 2 colonnes, la vue
   `_latest` compile, `COUNT(*)` latest inchangé.
3. **media_files (par shared_social)** : rebuild table sans `discord_notified` +
   recréation des contraintes/index + `CHECKPOINT`. Retirer aussi la colonne de
   `ops/media_store.go` (CREATE + ensure-list l.48/71) DANS le même commit code.
4. **Post-rebuild global** : rebooter le binaire de la branche → les migrations boot
   passent (aucune tentative de re-add d'une colonne droppée), smoke lecture PME + médias.

### Fenêtre
À convenir avec l'opérateur (serveur arrêté ~quelques minutes/titre). **Ce rebuild est le
candidat naturel à combiner avec l'étape 2 « répétition générale sur la copie » du PLAN DE
MERGE** (le boot sur la copie prouvera que le code post-DROP ne retente pas d'ajouter les
colonnes). Non fait en V9.

---

## Fichiers modifiés sur la copie prod pendant V9
**Aucun en V9a** (audit read-only strict). V9b/V9c n'ont eu aucune écriture à valider (0
à corriger) — voir le plan, items V9b/V9c.

## Fichiers `.pristine` créés
Aucun : aucune validation écrivant sur la copie n'a été nécessaire (TZ = 0 à corriger,
is_ranked = 0 à corriger).
