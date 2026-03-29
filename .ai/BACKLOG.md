— Tâches et TODO centralisés

> Mis à jour le 2026-03-29.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-29 | **[v6.2.1] Renommage agrégats KDA → `efficiency` + QA finale** : `_performance_session.py` v1/v2 + `_cumulative_results.py` + `_cumulative_series.py` + `cumulative.py` — variables/champs/colonnes DF renommés. i18n `sc_kda_label` → `sc_efficiency_label`. `session_compare.py` `perf["kda"]` → `perf["efficiency"]`. **Logging** ajouté dans `_performance_session.py` et `cumulative.py` (`logger.debug/warning`). **Couverture tests** : `_performance_session.py` 88% → 99.6%, `_cumulative_results.py` 87% → 100%, `cumulative.py` 96% → 100%, `_cumulative_series.py` 79% → 94% — 18 nouveaux tests (composantes KD/win/obj/MMR/saturation, branches edge-cases, DF vides). `p.kda` per-match (API Halo) et `performance_score` DB non touchés. 5297 tests passent. |
| 2026-03-29 | **[v6.2.1] Normalisation labels modes — Phase 2 intégration UI** : `translate_pair_name()` délègue à `resolve_display_mode()` + `infer_mode_category_from_pair_name()`. `infer_custom_category_from_pair_name()` corrigée (format inversé, `PREFIX_TO_CATEGORY` direct). Dead code supprimé (`_normalize_pair_case`, `_mode_db_lookup`, `_mode_sep`). 18 nouveaux tests unitaires pour `infer_mode_category_from_pair_name`. 5261 tests passent. |
| 2026-03-29 | **[v6.2.1] Normalisation labels modes — Phase 1 : `resolve_display_mode()`** : fonction pure `src/analysis/mode_display.py`, `_PREFIX_RULES`, format inversé, qualificatif Heavies, 29 overrides ajoutés dans `mode_pair_overrides` (FR + EN : Assault, KOTH, Team Slayer, Team Snipers, BTB Fiesta, Tactical). Validation CSV utilisateur. 35 tests unitaires. |
| 2026-03-28 | **[v6.2] Badges narratifs** — Remontada / Débandade / Contre-Remontada + Héros silencieux / Faux-frère : `DominanceFlag`, `comeback_analysis.py`, `comeback_backfill.py`, `--comeback-badges` CLI, badge `MatchImpactEvent` stats-only. |
| 2026-03-28 | **[v6.2] Unification vue coéquipier unique → vue escouade** — `f2_xuid` optionnel, suppression `render_single_teammate_view`. |
| 2026-03-28 | **[v6.2] Graphe combiné Frags↑/Morts↓** — `plot_trio_kills_deaths()`, axe Y symétrique, `safe_chart_render()`. |
| 2026-03-27 | **Bug — `index_media.py --force` levait `ConstraintError: Duplicate key`** : `force_rescan` contourne uniquement le filtre delta mtime, `existing` toujours chargé. |
| 2026-03-26 | **Bug critique — `mv_player_matches` recalcule le KDA au lieu de lire la valeur API** : détection dynamique `has_kda_col` + SQL conditionnel. |
| 2026-03-26 | **Bug RÉCURRENT CRITIQUE — Session escouade absente du graphe "Évolution de la performance"** : `shared_read_only=True` dans `_engine_fanout.py` + LEFT JOIN défensif dans `_performance_squad`. |
| 2026-03-26 | **Bug — Colonne "Précédente rencontre" incohérente (Page Match · Encounters)** : CTE `filter_past` + `_fetch_match_start_time` + guard `days = max(0, delta.days)`. |
| 2026-03-26 | **UX — Score d'équipe supérieur aux scores individuels** : `_render_compact_team_card` calcule `bonus = score - base_avg`. |
| 2026-03-26 | **Bug — Médias mal rattachés aux matchs (décalage fuseau horaire)** : `epoch(timezone('UTC', capture_end_utc))` dans `associate_with_matches()`. |
| 2026-03-21 | **Bug — Frags vs. détail armes (double-comptage melee)** : remainder `api_total - film_kills` dans 3 fichiers + `load_total_kills_for_player()`. |
| 2026-03-21 | **UI — Graphe stats/min escouade : morts sous l'axe** — `dpm_neg`, ticks Y absolus via `build_symmetric_abs_ticks`. |
| 2026-03-21 | **UI — Notation de session escouade (Page Coéquipiers)** — `compute_squad_performance_score()`, `SQUAD_GRADE_THRESHOLDS`, `render_squad_session_header()`, 18 tests. |
| 2026-03-21 | **CI — Scripts exclus par `.gitignore`** — `enforce_size_limits.py`, `validate_imports.py`, stubs `test_page_router_smoke/regressions.py`. |
| 2026-03-19 | **Medal definitions en BDD** — table `medal_definitions` dans `metadata.duckdb` (167 médailles), CLI `--medal-metadata`, `MedalsMixin`, 16 tests unitaires. |
| 2026-03-19 | **Fix 11 — Fan-out multi-joueurs** : `FanoutEnrichmentMixin` + `shared_read_only=True`. |
| 2026-03-19 | **Audit post-V6** : `weapon_kills` bit sync, `v_gamertag_lookup` systématique, `shared_matches_v2.duckdb` production, LEGACY SyncScope supprimés. |
| ≤ 2026-03-16 | *(items antérieurs archivés — voir `.ai/archive/` et `git log`)* |

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

---

###  Perf — Vectoriser le backfill multi-flags performance scores (v7+)

**Noté le** : 2026-03-26 | **Priorité** : Basse

Quand `--force-performance-scores` est combiné avec d'autres flags, la boucle appelle `compute_performance_score_for_match()` une fois par match avec une requête SQL individuelle pour l'historique des 50 derniers matchs.

**Solution** : pré-charger l'historique complet avant la boucle (comme `batch_compute_performance_scores`), le passer en contexte, supprimer la requête interne per-match.

**Impact** : uniquement les backfills multi-flags. Le sync normal est déjà vectorisé.

---

### 🟡 Script — Analyse des kills par arme pour un match donné (v7+)

**Noté le** : 2026-03-27 | **Priorité** : Basse

Outil CLI de diagnostic : tous les kills d'un match donné pour un joueur donné.

**Entrée** : `match_id` + `gamertag`
**Sortie** : `killer / victim / timestamp mm:ss / weapon_id`
**Données** : `weapon_kills` JOIN `killer_victim_pairs` + `xuid_aliases` + `v_gamertag_lookup` (toutes dans `shared_matches_v2.duckdb`)
**Complexité** : Faible

---

### 🟡 Kills environnementaux — catégorie dédiée (v7+)

**Noté le** : 2026-03-28 | **Priorité** : Basse

La médaille **Kong** (kill via baril projeté) est dans `GRENADE_MEDALS` par défaut. Créer une catégorie `environmental_kills` :
1. Colonne `environmental_kills INTEGER` dans `match_participants` (migration)
2. Retirer `Kong` de `GRENADE_MEDALS` → `ENVIRONMENTAL_MEDALS`
3. Logique filmshell dans `_weapon_kills_repo.py` + backfill `--environmental-kills`

**Priorité** : Basse — barrel kills très rares, impact négligeable.

---

## 🔮 Roadmap v6.3+

---

### [v6.3] Score de forme — indice de progression court terme

**Noté le** : 2026-03-28 | **Priorité** : Moyenne

```
form_score = moy_perf_score(14 derniers matchs) - moy_perf_score(90 derniers matchs)
```

- Positif → en forme / Négatif → creux de forme
- Calculable par `mode_category` (Arena, BTB, Ranked)
- Données : `player_match_enrichment.performance_score` déjà disponible

**Implémentation** :
1. `compute_form_score(gamertag, anchor_date)` dans `src/analysis/performance_score.py`
2. Colonne `form_score FLOAT` dans `sessions` (migration)
3. Calculé au post-sync, affiché sur la page d'accueil / profil (sparkline 30j + indicateur ↑↓)

---

### [v6.3] Détection de changement de niveau (breakpoints)

**Noté le** : 2026-03-28 | **Priorité** : Basse

Moyenne mobile double (14j vs 90j) — croisements ascendant/descendant = pallier détecté.

**Implémentation** :
1. `detect_level_breakpoints(df: pl.DataFrame) -> list[Breakpoint]` dans `src/analysis/progression.py`
2. `Breakpoint(date, direction: "up"|"down", delta_perf, n_matches_confirmed)` — seuil ≥10 matchs consécutifs
3. Table `progression_breakpoints` dans `stats.duckdb`
4. Overlay "cap franchi" sur les courbes de tendance

---

### [v6.3] Page Adversaires — Head-to-head, Nemesis, Proie

**Noté le** : 2026-03-28 | **Priorité** : Moyenne

Nouvelle page dédiée aux adversaires récurrents.

**Données** : tout dans `shared_matches_v2.duckdb` — `match_participants`, `killer_victim_pairs`, `match_registry`, `v_gamertag_lookup`.

| Métrique | Source |
|----------|--------|
| `matches_vs` | `match_participants` |
| `win_rate_vs` | `match_registry.outcome` |
| `kills_on` / `deaths_from` | `killer_victim_pairs` |
| `nemesis_score` = `deaths_from / max(1, kills_on)` pondéré | dérivé |
| `prey_score` = `kills_on / max(1, deaths_from)` pondéré | dérivé |

**Implémentation** :
1. `src/data/services/rivals_service.py` — `load_rivals_stats(gamertag, min_matches=3, limit=50)`
2. Nouvelle page `src/ui/pages/rivals.py`
3. Filtres : mode_category, fenêtre temporelle (30j/90j/all)
4. Exclure bots (`xuid LIKE 'bid(%'`), min_matches configurable

---

### [v6.3] Discord — Résumé de session post-sync

**Noté le** : 2026-03-28 | **Priorité** : Basse

Bouton `📤` dans la sidebar, actif ≥5 min après `last_match_end_time` (configurable).

**Contenu embed** : W-L/win rate, meilleur match, top médaille, badge comeback, composition escouade, rôles de soirée (Champion 🏆 / Maillon Faible 🍌 via `compute_impact_scores()`).

**Données** :
- Colonne `discord_notified_at TIMESTAMP DEFAULT NULL` dans `sessions` (migration)
- `discord_session_notify_delay_minutes` dans `app_settings.json` (défaut : 5)
- `src/utils/discord_notifier.py` à étendre

**Opt-in** : visible uniquement si `discord_session_notify = true` ET webhook configuré.

---

### [v6.3] Clutch moments — kills décisifs

**Noté le** : 2026-03-28 | **Priorité** : Basse

Trois types de kills clutch, par ordre de fiabilité :

| Type | Définition | Données |
|------|-----------|---------|
| **Spree-stopper** | Kill sur joueur avec médaille de série dans ce match | `medals_earned` × `killer_victim_pairs` |
| **Comeback clutch** | Kill en match `DominanceFlag.COMEBACK` / `COUNTER_COMEBACK`, joueur top-2 killers | `match_registry.comeback_flag` × `match_participants` |
| **Last-minute** | Kill dans les 60 dernières secondes d'un Slayer à ≤2 pts d'écart | `killer_victim_pairs.timestamp_ms` × `match_registry` |

**Stockage** : colonnes `clutch_kills INTEGER` + `clutch_type TEXT` dans `player_match_enrichment`.
**Backfill** : `--clutch-kills` dans `scripts/backfill_data.py`, logique dans `src/analysis/clutch_analysis.py`.
**Limites** : spree-stopper approximatif (pas de timestamp par médaille) ; last-minute dépend de la couverture filmshell.
