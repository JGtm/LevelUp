# Axe 1 · CHECKLIST — Parité Python↔Go + Streamlit↔React

> Cocher chaque item au fur et à mesure. Pour tout écart détecté : classif 🔴🟠🟡🟢 + fichier:ligne + une ligne dans la section correspondante du `claude_review.md` ou `chatgpt_review.md`.

## Phase de préparation

- [ ] SHAs Python / Go / React figés et notés dans `SCOPE.md`
- [ ] Accès lecture confirmé aux deux worktrees
- [ ] Template vide copié vers `claude_review.md` ou `chatgpt_review.md`
- [ ] CLAUDE.md des deux worktrees lus

---

## Bloc 1 — Endpoints API (section A du template)

### 1.1 Endpoints principaux (matrice OpenAPI)

> **Source de vérité** : `OPENAPI_MVP_P0_P1.md` au SHA figé. Lister chaque endpoint présent dans cette matrice et le cocher individuellement. Ne pas supposer de liste hors-matrice. Pour chaque endpoint :
> - Vérifier existence côté Go
> - Vérifier contrat (payload entrée / sortie / codes d'erreur)
> - Vérifier parité pagination / tri / filtres si applicable

- [ ] Matrice OpenAPI lue intégralement
- [ ] Chaque endpoint de la matrice a une ligne dédiée dans le tableau `claude_review.md` / `chatgpt_review.md` §A
- [ ] Tout endpoint Go présent **mais absent** de la matrice listé à part (🟢 si justifié, sinon 🟠/🔴)
- [ ] Tout endpoint de la matrice **absent** du Go = 🔴 bloquant

### 1.2 Vérification post-Sprint 49 (ex-`notYetImplemented`)

> Statut attendu après S49 : tous implémentés. L'audit **vérifie** ce statut ; toute absence résiduelle = 🔴 bloquant.

- [ ] Aucun endpoint ne figure encore dans une liste `notYetImplemented` / `TODO` côté Go
- [ ] Chaque endpoint précédemment listé a une implémentation réelle (pas un stub renvoyant `501 Not Implemented`)
- [ ] Les tests d'intégration Go couvrent ces endpoints post-S49

### 1.3 Middlewares

- [ ] `request_id` — présent Python / présent Go
- [ ] `cors` — politique identique
- [ ] `rate_limit` — seuils identiques ou modernisés avec motivation
- [ ] `session` — cookie set/read/expired compatible
- [ ] `csrf` — protection en place
- [ ] `shadow` — mode shadow utilisable pour validation finale
- [ ] `contract_validate` — validation OpenAPI au runtime

---

## Bloc 2 — Pages UI (section B du template)

Pour chaque page ci-dessous :
1. Ouvrir la page Streamlit + la page React côte-à-côte
2. Lister les widgets / sections visibles
3. Cocher la présence côté React

### 2.1 Home

- [ ] KPIs principaux
- [ ] Battle pass / challenges
- [ ] Cartes de forme / intensité
- [ ] Distributions
- [ ] Narrative (comeback badges)

### 2.2 Career

- [ ] Résumé carrière (CareerSummaryCard)
- [ ] Charts (progression LUSR, CSR, win rate rolling)
- [ ] Top matches
- [ ] Encounters (nemesis top)
- [ ] Rang actuel + progression palier

### 2.3 Synthesis

- [ ] KPIs globaux
- [ ] Tableau synthétique
- [ ] Sections temporelles

### 2.4 Match history

- [ ] Tableau paginé
- [ ] Filtres (map, mode, playlist, outcome, range)
- [ ] Tri multi-colonnes
- [ ] Export (CSV / JSON ?)

### 2.5 Match view

- [ ] Scoreboard complet
- [ ] Détail joueur (panel latéral)
- [ ] Onglets (stats / médailles / citations / weapon kills / encounters / timeline)
- [ ] Rank delta (CSR/LUSR)
- [ ] Participation
- [ ] Charts internes

### 2.6 Last match

- [ ] Raccourci match le plus récent
- [ ] Gestion absence de match récent
- [ ] Distinction PvE / PvP

### 2.7 Explorer

- [ ] Recherche gamertag
- [ ] Filtres avancés
- [ ] Résultats paginés
- [ ] Enrichissement (si feature Python)

### 2.8 Session compare

- [ ] Sélection de 2 sessions
- [ ] Vue comparative (extras, histoire, viz)
- [ ] Gestion overlap

### 2.9 Timeseries

- [ ] Granularités jour / semaine / mois
- [ ] Distributions
- [ ] Forme, intensité
- [ ] Weapons

### 2.10 Squad / Teammates

- [ ] Vue solo
- [ ] Vue duo
- [ ] Vue trio (f2_xuid optionnel v6.2)
- [ ] Charts (charts, intensity, impact, synergy, weapons, map)
- [ ] Légende
- [ ] Sélecteur coéquipiers + badges narrative

### 2.11 Citations

- [ ] Liste des citations
- [ ] Recompute button

### 2.12 Media

- [ ] Grille media
- [ ] Filtres
- [ ] Likes
- [ ] Association ↔ match
- [ ] Temporal view
- [ ] Upload si supporté

### 2.13 Settings

- [ ] Sections de configuration identifiées côte-à-côte
- [ ] Gestion tokens / auth
- [ ] DB profiles

### 2.14 Setup wizard

- [ ] Scénario premier lancement
- [ ] Device code flow utilisateur
- [ ] Xbox bootstrap
- [ ] Smoke test

### 2.15 Pages spécifiques React

- [ ] Changelog — justifier comme 🟢 (nouvelle feature)

---

## Bloc 3 — Algorithmes métier (section C du template)

Pour chaque algo :
1. Lire le module Python + le package Go
2. Comparer entrées/sorties
3. Vérifier l'existence de golden values
4. Rouler les goldens côté Go si possible

- [ ] Performance score — golden Python vs Go
- [ ] LUSR — golden
- [ ] CSR — golden
- [ ] Sessions — golden (stitching, labels)
- [ ] Citations — golden (tous mappings médailles)
- [ ] Killer/victim — golden (paires top)
- [ ] Weapon parser — golden (UBIGINT weapon_id)
- [ ] Spawn / comeback detection — golden

---

## Bloc 4 — Sync / Backfill (section D du template)

- [ ] Sync delta : même flux + même idempotence
- [ ] Backfill : tous les flags `SyncScope` couverts côté Go
- [ ] Write lease : sémantique `~5s timeout, 1 writer/path` reproduite
- [ ] Bitmask `backfill_completed` 18 bits : **numériquement identique** (Python bit N ↔ Go bit N)
- [ ] Bit 18 `weapon_kills` posé au bon endroit
- [ ] Dégradation gracieuse sur `nil` (Go ne panique pas)

---

## Bloc 5 — CLI / scripts (section E du template)

- [ ] `sync.py` — équivalent Go présent
- [ ] `backup_player.py` — équivalent
- [ ] `restore_player.py` — équivalent (ops/restore.go)
- [ ] `backfill_data.py` — équivalent (backfill_cli.go)
- [ ] `check_env.py` — équivalent (validation/gate.go)
- [ ] Lister et cocher les autres scripts (`scripts/*.py`)

---

## Bloc 6 — Données (section F du template)

### 6.1 Tables

- [ ] `match_registry` — colonnes, types, contraintes
- [ ] `match_participants` — 31 colonnes incl. MMR
- [ ] `medals_earned` — identique
- [ ] `killer_victim_pairs` — identique
- [ ] `xuid_aliases` — identique
- [ ] `weapon_kills` — identique (weapon_id UBIGINT)
- [ ] `highlight_events` — identique
- [ ] `pve_match_stats` — identique (tous ennemis)
- [ ] `player_match_enrichment` — identique
- [ ] `match_skill_rank` — identique (PK + exclusivité LUSR/CSR)
- [ ] `sessions` — identique
- [ ] `media_files` / `media_match_associations` — identique
- [ ] `personal_score_awards` — identique
- [ ] `match_citations` — identique
- [ ] `career_progression` — identique
- [ ] `sync_meta` — identique
- [ ] Tables metadata (career_ranks, citation_mappings, mode_*, weapon_labels)

### 6.2 Vues

- [ ] `v_gamertag_lookup`
- [ ] `v_match_full`
- [ ] `v_killer_victim_full`
- [ ] `v_weapon_kills`

---

## Bloc 7 — i18n (section G du template)

- [ ] 14 langues présentes côté Go/React
- [ ] Source des traductions = DuckDB metadata (pas statiques)
- [ ] `Accept-Language` résolu correctement (avec fallback)
- [ ] Pages React utilisent un hook i18n cohérent

---

## Bloc 8 — Observabilité & erreurs (section H du template)

- [ ] Notifier Discord porté
- [ ] Taxonomie erreur provider appliquée (`HALO_PROVIDER_ERROR_TAXONOMY.md`)
- [ ] Format logs cohérent (structuré JSON ?)

---

## Bloc 9 — Modernisations (section I du template)

- [ ] Chaque item 🟢 a une motivation écrite
- [ ] Aucune modernisation n'a cassé un scénario utilisateur valide

---

## Validation finale de l'axe

- [ ] Template rempli à 100% (aucune case vide)
- [ ] Récap §J cohérent avec les sections A-I
- [ ] Tous les écarts ont un fichier:ligne
- [ ] Tous les écarts ont une classification
- [ ] Commit sur la branche `phase11/sprint50-triple-audit`
