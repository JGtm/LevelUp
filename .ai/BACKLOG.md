# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-12.

---

## 🔴 Bugs actifs

### ~~Images citations d'armes incorrectes~~
> ✅ Traité le 2026-03-11 — voir section **Traité** ci-dessus.

---

## 🔴 Dette Technique (code source)

### Kwargs legacy SyncScope — nettoyage différé
> `scope=SyncScope(...)` opérationnel. Les 30+ kwargs `LEGACY` dans `detection.py` (L46) et `orchestrator.py` (L104, L435, L774) sont intentionnellement conservés pour rétro-compat — pas une urgence. À supprimer lors d'un prochain passage sur ces fichiers.

---

### Traductions FR manquantes dans migration metadata
> [`scripts/migration/migrate_metadata_to_duckdb.py`](../scripts/migration/migrate_metadata_to_duckdb.py#L72) L72 — `# TODO: ajouter traductions FR`

---

### Migration : noms d'assets résolus → IDs bruts en BDD
> Dans `match_registry`, les noms d'assets sont stockés en parallèle des IDs bruts (redondance + risque de stale data). À terme, l'UI doit résoudre les noms à la lecture depuis `metadata.duckdb`, pas les lire depuis les colonnes `*_name`.

**Contexte** : Au moment de l'insertion (sync initial), les noms publics (ex. `"Aquarius"`, `"Ranked Arena"`) sont récupérés depuis l'API SPNKr et stockés directement en BDD — en plus de l'ID brut. La `weapon_kills` (v5.7) et `medals_earned` montrent le bon modèle : ID brut uniquement, résolution à la lecture.

**Colonnes concernées dans `shared_matches.duckdb`** :

| Table | Colonnes ID (OK) | Colonnes nom résolu (à migrer) |
|-------|-----------------|-------------------------------|
| `match_registry` | `map_id`, `playlist_id`, `pair_id`, `game_variant_id` | `map_name`, `playlist_name`, `pair_name`, `game_variant_name` |
| `match_participants` | `xuid` | `gamertag` (redondant avec `xuid_aliases`) |
| `highlight_events` | `xuid` | `gamertag` (peut devenir stale si alias change) |

**Modèles de référence (déjà corrects)** :
- `medals_earned.medal_name_id` → UBIGINT, résolution via `metadata.duckdb`
- `weapon_kills.weapon_id` → UBIGINT post v5.7 (migré depuis `weapon_name`)

**Actions** :
- [ ] Auditer les usages UI/query des colonnes `*_name` dans `match_registry` pour identifier ce qui lit directement le nom stocké vs ce qui joint `metadata.duckdb`
- [ ] Créer une vue `v_match_registry` qui résout les noms à la lecture via JOIN sur les tables de référence `metadata.duckdb` (maps, playlists, game_variants)
- [ ] Migrer les requêtes consommatrices (pages Streamlit, repositories) vers la vue — supprimer les colonnes `*_name` de `match_registry` une fois toutes les requêtes migrées
- [ ] `match_participants.gamertag` et `highlight_events.gamertag` : évaluer si ces colonnes sont utilisées en lecture directe ou si le JOIN sur `xuid_aliases` est systématique — supprimer si redondant
- [ ] Ajouter un test de non-régression : aucune colonne `*_name` dans les nouvelles tables shared (hors `xuid_aliases`)

**Complexité** : L (impact UI + repositories + migrations)  
**Fichiers clés** : [`src/data/sync/migrations.py`](../src/data/sync/migrations.py), [`src/data/sync/_shared_writes.py`](../src/data/sync/_shared_writes.py), [`src/data/sync/transformers/_match.py`](../src/data/sync/transformers/_match.py), `data/warehouse/shared_matches.duckdb`

---

### Couverture tests `migrations.py` (lacunes v5.5–v5.7)
> [`src/data/sync/migrations.py`](../src/data/sync/migrations.py) — ~1290 lignes, couverture actuelle ~60%. Trois blocs sans aucun test depuis les versions 5.5–5.7.

| Fonction | Version ajoutée | Couverture actuelle |
|----------|----------------|---------------------|
| `ensure_weapon_kills_table()` | v5.7 | ❌ Aucun test |
| `ensure_bot_teammate_column()` | v5.5 | ❌ Aucun test |
| `add_spartan_id_to_career_progression()` | v5.x | ❌ Aucun test |
| `_recreate_highlight_events_with_sequence()` | v5.x | ⚠️ Chemin idempotent non testé |

**Actions** :
- [ ] `ensure_weapon_kills_table()` : tester création de table, conversion `weapon_name→weapon_id`, type BIGINT→UBIGINT, idempotence
- [ ] `ensure_bot_teammate_column()` : tester ajout de colonne, valeur par défaut, idempotence (double appel = même schéma)
- [ ] `add_spartan_id_to_career_progression()` : tester ajout colonne, contrainte, idempotence
- [ ] `_recreate_highlight_events_with_sequence()` : tester le chemin déjà-migré (si `nextval` existe, pas de double création)
- [ ] Viser couverture ≥ 85% sur `migrations.py` (mesurer via `python -m pytest --cov=src/data/sync/migrations`)

**Complexité** : M  
**Fichiers** : [`tests/test_migrations.py`](../tests/test_migrations.py), [`src/data/sync/migrations.py`](../src/data/sync/migrations.py)

---

## 🟠 Conflit `shared_matches.duckdb` — sync depuis UI Streamlit

> Source : audit logs 2026-03-09 — 339 warnings/sync, app stable, pas de panne fonctionnelle.
> **Ne pas traiter tant que le sync n'est pas stable depuis ≥ 1 semaine.**
> Signal de déclenchement : sync retourne `None` pour shared sur plusieurs runs consécutifs.

### Contexte

Quand le sync est lancé depuis l'UI Streamlit, `_engine_connections.py::_get_shared_connection()` tente d'ouvrir `shared_matches.duckdb` en **R/W direct**. Simultanément, Streamlit maintient une connexion **R/O + ATTACH** sur le même fichier via `@st.cache_resource` (ttl=3600, `get_cached_repository_st`). DuckDB refuse qu'un même fichier soit ouvert sous deux modes dans le même processus.

Le retry appelle `release_all_db_connections()` (WeakSet), mais le repo du cache Streamlit rétablit sa connexion R/O dès le rerun suivant → **cycle conflit → release → reconnexion R/O → conflit**.

### Plan — Refactoring connexion shared (branche dédiée)

**Mécanique** : Supprimer `duckdb.connect(shared, read_only=False)`. À la place, écrire dans shared via `ATTACH shared AS s (READ_WRITE)` depuis la connexion player, ou passer par un contexte de connexion partagé unique géré par un singleton. Supprime la cause racine — le conflit devient structurellement impossible.

**Effort** : XL — refactoring complet du sync engine  
**Risque** : Élevé — à traiter dans une branche dédiée, isolée de la prod

---

## 🟡 Hover thumbnail sur les noms de cartes (tableaux HTML)

> Commencé le 2026-03-11 — solution identifiée le 2026-03-12.

**Objectif** : Au survol d'un nom de carte dans les tableaux HTML (Historique, Explorer, Win/Loss), afficher la miniature correspondante `static/maps/*.jpg|png`.

**Solution retenue** : CSS pur avec classes uniques par élément via `st.markdown` — pas de JS, pas de sandbox.  
Source : https://github.com/everydaydigital/streamlit-image_hover_tooltip  
Principe : classes CSS `hoverable_i` / `tooltip_i` / `image-popup_i` générées par cellule, popup via `background-image` + `:hover { display: block }`.

**Ce qui a été fait (à nettoyer)** :
- `_MAP_TOOLTIP_SCRIPT` dans `src/ui/styles.py` — script JS injecté via `load_css()`, **ne fonctionne pas** (sandboxé par Streamlit), à supprimer
- `<span class='map-cell' data-thumb-url='...'>` dans `match_table_html.py` L296 et `win_loss_table_style.py` L53 — attributs `data-*` liés au JS, à remplacer

**Ce qui est réutilisable** :
- `map_thumb_url()` + `_build_map_url_index()` dans `match_table_html.py` — logique de résolution d'URL valide, à conserver

**Actions restantes** :
- [ ] Supprimer `_MAP_TOOLTIP_SCRIPT` de `src/ui/styles.py` et son injection dans `load_css()`
- [ ] Remplacer les `<span class='map-cell' data-thumb-url='...'>` par des divs CSS hover (classes uniques `hoverable_i` par ligne de tableau)
- [ ] `_build_map_url_index()` : passer `lru_cache(maxsize=None)` (actuellement `maxsize=1`, très fragile) et normaliser les noms via `unicodedata.normalize('NFC', name)`
- [ ] Créer une table de correspondance explicite `nom API Halo → fichier PNG` pour les maps avec caractères spéciaux ou noms divergents

---

## 🟢 Détection de langue système dans `LevelUp.sh` / `LevelUp.bat`

> Ajouté le 2026-03-12. Faisabilité confirmée pour les deux scripts.

**Objectif** : Détecter la langue du système au démarrage et afficher les messages du lanceur dans la langue de l'utilisateur (FR/EN minimum). Actuellement tous les messages sont en français.

### Faisabilité

#### `LevelUp.sh` (POSIX sh — macOS / Linux / WSL2)

**Oui, faisable** via les variables d'environnement POSIX standard, disponibles sur tous les systèmes :

| Variable | Exemple | Priorité |
|----------|---------|----------|
| `$LC_ALL` | `fr_FR.UTF-8` | Priorité maximale |
| `$LC_MESSAGES` | `en_US.UTF-8` | Messages uniquement |
| `$LANG` | `fr_FR.UTF-8` | Fallback général |

Logique de détection (POSIX strict, pas de bashisme) :

```sh
# Déterminer la langue (2 premières lettres de la locale)
_locale="${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}"
_lang_code=$(echo "$_locale" | cut -c1-2 | tr '[:upper:]' '[:lower:]')
# Valider le code : doit être 2 lettres a-z (exclut C, POSIX, chaîne vide)
case "$_lang_code" in
    [a-z][a-z]) : ;;  # valide
    *)           _lang_code="" ;;
esac
case "$_lang_code" in
    fr) SCRIPT_LANG="fr" ;;
    *)  SCRIPT_LANG="en" ;;  # anglais par défaut (inclut C, POSIX, non défini)
esac
```

**Couverture** : macOS (bash/zsh), Ubuntu (dash), Arch, Fedora, WSL2 — tous exposent ces variables. Si aucune n'est définie → fallback anglais.

---

#### `LevelUp.bat` (Windows CMD)

**Oui, faisable** via le registre Windows — toujours disponible (Vista+) sans dépendance externe :

```bat
:: Lire LocaleName depuis le registre (ex : "fr-FR", "en-US", "de-DE")
set "WIN_LOCALE="
for /f "tokens=3" %%L in ('reg query "HKCU\Control Panel\International" /v LocaleName 2^>nul') do set "WIN_LOCALE=%%L"

:: Extraire les 2 premières lettres
set "LANG_CODE=en"
if defined WIN_LOCALE (
    set "_tmp=!WIN_LOCALE:~0,2!"
    if /i "!_tmp!"=="fr" set "LANG_CODE=fr"
)
```

**Alternative via PowerShell** (plus robuste, PowerShell disponible depuis Windows 7) :
```bat
for /f %%L in ('powershell -NoProfile -Command "[System.Globalization.CultureInfo]::CurrentUICulture.TwoLetterISOLanguageName" 2^>nul') do set "LANG_CODE=%%L"
```

**Recommandation** : Priorité registre (pas de dépendance PowerShell), fallback `en` si introuvable. L'alternative PowerShell via `TwoLetterISOLanguageName` peut retourner des résultats inconsistants pour les variantes régionales (ex. `zh` pour `zh-CN` et `zh-TW` — même code, encodages différents) ; la lecture directe du registre (`LocaleName` = `fr-FR`, `en-US`) est plus fiable.

---

### Plan d'implémentation

- [ ] **Étape 1** : Ajouter la détection de langue en tête de `LevelUp.sh` et `LevelUp.bat`
- [ ] **Étape 2** : Inventaire exhaustif des chaînes localisables (passe préparatoire sur les deux scripts — les ~35 SH / ~30 BAT listés initialement sont non exhaustifs)
- [ ] **Étape 3** : Extraire toutes les chaînes dans un bloc de définition par langue (`# ── Messages ──` en haut de chaque script)
- [ ] **Étape 4** : Traduire en anglais + corriger les accents manquants dans les messages FR `.bat` (`chcp 65001` déjà présent)
- [ ] **Étape 5** : Remplacer les `echo` littéraux par des appels à la macro/variable traduite
- [ ] **Étape 6** : Test manuel : `LANG=en_US.UTF-8 ./LevelUp.sh` sur Linux + WSL2 ; test sur système EN Windows

**Langues cibles** : FR (actuel), EN (ajout prioritaire). Autres langues (DE, ES, PT…) extensibles au même pattern sans refactoring.

**Complexité** : M — modification chirurgicale de 2 scripts, aucune dépendance externe, totalement backward-compatible.  
**Fichiers** : [`LevelUp.sh`](../LevelUp.sh), [`LevelUp.bat`](../LevelUp.bat)

---

## �🟢 Améliorations futures / Backlog bas
### [UI] Heatmap performance par joueur × carte — Page Teammates
> Ajouté le 2026-03-12.

**Objectif** : Visualiser en un coup d'œil l'efficacité de chaque joueur (toi + coéquipier(s) sélectionnés) sur chaque carte des matchs filtrés.

**Format** : Heatmap Plotly (`go.Heatmap`) — lignes = cartes, colonnes = joueurs, valeur de cellule = `performance_score` moyen sur la carte.

**Option "Outcome"** : superposer en annotation le win_rate de l'équipe sur chaque carte (ex. `62%` affiché dans la cellule, ou couleur de contour Win/Loss).

**Données nécessaires** :
- `performance_score` : disponible dans `player_match_enrichment` (joueur principal) et calculable depuis `shared.match_participants` (coéquipier) via `compute_performance_series`
- `map_name` : disponible dans `match_registry` via `mv_player_matches`
- `outcome` : disponible dans `shared.match_participants`
- La logique de `compute_map_breakdown` est réutilisable — l'appliquer par joueur sur les `match_ids` communs

**Placement suggéré** : Nouvel onglet ou section dans la vue "single coéquipier" (après les graphes de comparaison existants). À adapter pour la vue multi-coéquipiers.

**Implémentation** :
- [ ] `compute_map_performance_matrix(players_dfs: dict[str, pl.DataFrame]) -> pl.DataFrame` dans `src/analysis/maps.py` — retourne un DataFrame long `(player, map_name, perf_avg, win_rate, n_matches)`
- [ ] `plot_map_performance_heatmap(df_matrix, show_outcome: bool = False) -> go.Figure` dans `src/visualization/maps.py`
- [ ] Intégration dans `teammates_charts.py` ou nouveau `teammates_maps.py` si >80L
- [ ] Clés i18n à ajouter dans `src/ui/i18n/pages/teammates.py`
- [ ] Filtre minimum de matchs (ex. ≥ 3 sur la carte) pour éviter les cellules non représentatives

**Complexité** : M

### [UI] Performance par carte vs historique — vues escouade et joueur
> Ajouté / recadré le 2026-03-12.

**Objectif produit** : démontrer l'efficacité sur les cartes des matchs sélectionnés par les filtres, en la comparant à l'historique sur ces mêmes cartes.

**Décision UX actuelle** :
- **Escouade** : graphe principal en barres horizontales par carte, centré sur le **delta de performance**.
- **Win rate** : affiché en **colonne texte à droite**, pas comme seconde série graphique.
- **Logique d'escouade** : conserver strictement la logique déjà utilisée sur la page (`amis sélectionnés + x joueurs inconnus de l'équipe`), sans redéfinir le périmètre pour ce composant.

**Définition escouade** :
- `perf_escouade_filtrée(carte)` = moyenne de la performance moyenne de l'équipe sur les matchs filtrés de cette carte
- `perf_escouade_historique(carte)` = même calcul sur tout l'historique de cette carte
- `delta_perf(carte)` = `perf_escouade_filtrée - perf_escouade_historique`
- Colonne droite : `WR filtré | WR historique`

**Format recommandé — vue escouade** :
- barre horizontale divergente, axe centré sur `0`
- valeur affichée = `delta_perf`
- sous-texte discret = `perf filtrée vs perf historique`
- colonne droite = `WR 63% | 54%`
- volume affiché discrètement = `n filtré / n hist`

**Piste à préciser — vue hors escouade / joueur seul** :
- comparer la performance moyenne du joueur sur les cartes filtrées à son historique personnel sur ces mêmes cartes
- éviter un simple doublon de la vue escouade : privilégier une lecture plus individuelle (constance, spécialisation map, forme récente)
- conserver le win rate en info secondaire légère

**Implémentation** :
- [ ] Créer un agrégat par carte pour la vue escouade : `(map_name, perf_filtered, perf_history, delta_perf, wr_filtered, wr_history, n_filtered, n_history)`
- [ ] Réutiliser la logique d'escouade existante de la page teammates pour définir le périmètre des matchs et de l'équipe
- [ ] Créer un composant de visualisation dédié avec barres de delta + colonne de win rate à droite
- [ ] Ajouter un seuil minimal de volume (`n`) et un traitement visuel des faibles échantillons
- [ ] Définir la vue "joueur seul" correspondante avant implémentation UI, pour éviter deux graphes redondants
- [ ] Ajouter les clés i18n nécessaires dans `src/ui/i18n/pages/teammates.py` et/ou `src/ui/i18n/pages/timeseries.py`

**Complexité** : M
**Dépendances** : `compute_map_breakdown` (existant), `shared.match_participants`, `player_match_enrichment`

#### [UI] Variante visuelle validée — superposition des deltas + transparence
> Ajouté le 2026-03-12 (complément, sans remplacement des pistes existantes).

**Contexte** : pour la vue par carte, conserver le `delta_perf` comme signal principal, et ajouter le `delta_ratio` sans surcharger le graphe.

**Décision de représentation (Option 1)** :
- Superposer `delta_perf` et `delta_ratio` **après normalisation** (échelles différentes)
- Conserver les **valeurs brutes en texte** à côté pour éviter l'ambiguïté
- Garder le `win rate` en colonne texte à droite (`WR filtré | WR hist`)

**Encodage visuel proposé** :
- Barre principale (pleine) : `delta_perf_norm`
- Barre secondaire (fine/contour) : `delta_ratio_norm`
- Couleur selon le signe (positif/négatif)
- Transparence plus faible en cas de sous-performance (négatif)

**Règles de lisibilité / confiance** :
- Opacité modulée par le volume `n_filtered` (faible n = plus transparent)
- Afficher systématiquement : `Δ perf`, `Δ ratio`, `n filtré / n hist`
- Si `n_filtered` sous seuil minimum : état visuel atténué + tooltip explicite

**Calculs à afficher (valeurs brutes)** :
- `delta_perf = perf_filtered - perf_history`
- `delta_ratio = ratio_filtered - ratio_history`

**Implémentation (complémentaire aux tâches déjà listées)** :
- [ ] Ajouter les champs `ratio_filtered`, `ratio_history`, `delta_ratio` à l'agrégat par carte
- [ ] Ajouter une normalisation robuste (`clamp` / bornes percentiles) pour la superposition des barres
- [ ] Implémenter la superposition visuelle (barre pleine + barre fine/contour) dans le composant chart
- [ ] Ajouter un mapping d'opacité basé sur `n_filtered` et le signe du delta
- [ ] Vérifier l'accessibilité (contraste couleurs + lecture sans couleur via labels)

---
- **Audit Pandas → Polars** : audit complet réalisé le 2026-03-12. Résultats détaillés ci-dessous.
- **START_HERE.md** : Le fichier `.ai/START_HERE.md` référence des phases 5-10 d'une migration v5 qui semblent antérieures à v5.1. À vérifier si encore pertinent ou à archiver dans `.ai/archive/`.

### Résidus Pandas — état audit 2026-03-12

#### 🔒 Légitimes, intouchables

| Fichier | Motif |
|---------|-------|
| `src/visualization/_compat.py` | Module frontière intentionnel (`to_pandas_for_plotly`, `ensure_polars`, etc.) |
| `src/ui/pages/win_loss.py` | `pd.DataFrame.style` pour styling conditionnel Streamlit — pas d'équivalent Polars (exception documentée dans `test_legacy_free_ui_viz_wave_a.py`) |
| `src/ai/rag.py` | `self.table.to_pandas()` — API LanceDB retourne nativement Pandas ; `iterrows()` dans la logique de recherche — hors périmètre code métier |
| `src/utils/polars_compat.py` | `pl.from_pandas()` — rôle utilitaire de conversion |
| `src/data/repositories/_arrow_bridge.py` | `pl.from_pandas()` — pont Arrow/Polars |

#### 🟢 Remplaçables immédiatement (aucune dépendance externe)

- [ ] **`src/visualization/participation_charts.py`** (3 occurrences) : `.to_pandas()` + `.map(lambda)` + filtre `.pdf[col < 0]` + `.iloc[::-1]` → remplacer par `pl.col().map_elements()`, `filter()`, `reverse()`. Passer `.to_list()` à Plotly.
- [ ] **`src/visualization/participation_charts_extra.py`** (1 occurrence) : même pattern sunburst → Polars pur.
- [ ] **`src/ui/pages/objective_analysis.py`** (3 occurrences) : `.to_pandas()` + rename colonnes + `st.dataframe()` → `st.dataframe()` accepte Polars nativement depuis Streamlit 1.25 ; utiliser `.with_columns(pl.col().map_elements(...))` + `.rename({})`.
- [ ] **`src/ui/components/duckdb_analytics.py`** (1 occurrence) : `chart_df.to_pandas().set_index("Match")` → `st.line_chart(chart_df, x="Match", y="KDA", width="stretch")`.

#### 🔴 Dead code à supprimer

- [ ] **`src/analysis/_performance_relative.py`** : guard `was_pandas = not isinstance(df, pl.DataFrame)` + retour conditionnel `.to_pandas()` — **tous les appelants identifiés** (`maps.py`, `timeseries_service.py`, `_timeseries_progression.py`, `match_history.py`, `_session_compare_history.py`, `_teammates_trio.py`) passent exclusivement du Polars. Chemin Pandas jamais emprunté.

#### 🟡 Guards probablement inutiles (vérifier avant de supprimer)

- [ ] **`src/analysis/sessions.py` L56** : `pl.from_pandas(df)` dans `compute_sessions()` — vérifier si un appelant passe encore du Pandas.
- [ ] **`src/analysis/_performance_relative_helpers.py` L26** : `_normalize_df()` — même vérification.

#### 🟡 À évaluer (breaking change potentiel)

- [ ] **`src/data/integration/streamlit_bridge.py`** : `matches_to_dataframe()` a un contrat "retourne Pandas" (construction Polars → `to_pandas()` → timezone via `pd.to_datetime().dt.tz_convert()`). Migration possible mais nécessite d'auditer tous les appelants d'abord.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-08 | Bug #0 : match invisible post-sync — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | Bug #1 : `win_rate` unifié sur `NULLIF(WIN+LOSS, 0)` dans `analytics.py` et `trends.py` |
| 2026-03-08 | Bug #5 : NaN-check fragile dans `match_view.py` → `is not None` |
| 2026-03-08 | Dette #2 : guard obsolète `_PERF_SCORE_AVAILABLE` supprimé dans `_performance.py` |
| 2026-03-08 | Dette #3 : dead code `_ensure_performance_score_column()` supprimé |
| 2026-03-08 | Dette #4 : magic number `outcome == 4` → `Outcome.DID_NOT_FINISH` |
| 2026-03-08 | Dette #6 : magic SQL `2`/`3` → constantes `_WIN`/`_LOSS` dans `analytics.py` |
| 2026-03-08 | i18n-1 : clés tronquées `PAIR_FR` restaurées dans `translations.py` |
| 2026-03-08 | i18n-2 : 342 entrées redondantes supprimées de `PAIR_FR` (399 → 57) |
| 2026-03-08 | i18n-3 : doublon `tm_session_trend` supprimé dans `widgets.py` |
| 2026-03-08 | Kwargs legacy SyncScope — dépréciés + `scope=SyncScope(...)` opérationnel ; kwargs conservés pour rétro-compat (suppression conditionnelle : quand tous les appelants migrés) |
| 2026-03-08 | `career.py` migré vers `get_cached_repository_st()` (plus de `duckdb.connect()` nu) |
| 2026-03-12 | `custom_rules.py:103` — TODO disparu, `compute_annexion_forcee` implémentée (extrapolation documentée) |
| 2026-03-12 | Pandas éliminé du code métier — résidus légitimes uniquement : `_compat.py` (frontière UI) + `distributions.py` (`TYPE_CHECKING` only, pas d'import runtime) |
| 2026-03-08 | Perf UI — vues matérialisées reconstruites uniquement post-sync dans `engine.py` |
| 2026-03-08 | Perf UI — lazy-loading `match_view` via `st.tabs` + `@fragment_if_available` |
| 2026-03-08 | Perf UI — pagination SQL `LIMIT/OFFSET` sur `mv_player_matches` |
| 2026-03-08 | Perf UI — projections Polars fines par page dans `cache_loaders.py` |
| 2026-03-08 | i18n câblage `t()` dans les pages/widgets Streamlit |
| 2026-03-08 | CI/CD — détection de régression + pre-commit hook |
| 2026-02-26 | Quick wins perf UI (cache TTL, `@lru_cache`, `@st.cache_data`) |
| 2026-02-25 | v5.3 LUSR stabilisation + UI Carrière |
| 2026-02-25 | i18n Phase 1b — traductions EN registres |
| 2026-02-20 | v5.2 : Filtres intent-based + Stats PvE Firefight |
| 2026-02-17 | Release v5.1 — architecture shared-only |
| 2026-02-15 | Remédiation P0/P1 sécurité SQL + conformité Streamlit |
