# Plan de livraison v5.7.0

> Créé le 2026-03-12. Version cible : **5.7.0** (depuis 5.5.1).  
> Tag Git : `v5.7.0` → déclenche `release.yml` (build + publish GitHub Release automatique).

---

## Périmètre

### ✅ Inclus

| # | Chantier | Source backlog | Complexité |
|---|----------|---------------|------------|
| A | Couverture tests `migrations.py` (v5.5–v5.7) | Dette technique | M |
| B | Résidus Pandas → Polars (viz + analysis + UI) | Audit 2026-03-12 | M |
| C | Dead code performance_relative — guard Pandas | Audit 2026-03-12 | S |
| D | Hover thumbnail cartes (CSS pur, supprimer JS) | Backlog hover | M |
| E | Détection de langue `LevelUp.sh` / `LevelUp.bat` | Backlog i18n | M |
| F | Traductions FR manquantes migration metadata | Dette technique | S |
| G | Version bump + tag + release notes | Release | S |

### ❌ Exclus (explicitement hors scope)

- **Kwargs legacy SyncScope** — rétro-compat maintenue, nettoyage différé
- **Migration noms d'assets résolus → IDs bruts** — chantier L, impact trop large

### 🔜 Backlog v5.8+ (dépendances non réunies)

| # | Chantier | Bloqué par | Complexité |
|---|----------|-----------|------------|
| H | Top 10 meilleurs / pires matchs (onglet carrière) | Récupération médailles équipes alliée/ennemie via API | L |

---

## Stratégie de branche

**Branche unique** : `release/v5.7` (depuis `main`)

Commits séquentiels :
1. `test(migrations): couverture ensure_weapon_kills, ensure_bot_teammate, add_spartan_id`
2. `refactor(viz): éliminer to_pandas() dans participation_charts`
3. `refactor(analysis): supprimer guard Pandas dans _performance_relative`
4. `feat(ui): hover thumbnail CSS pur, supprimer JS sandboxé`
5. `feat(launcher): détection langue système FR/EN`
6. `fix(metadata): traductions FR rangs career`
7. `chore(release): bump version 5.5.1 → 5.7.0`

---

## A — Couverture tests `migrations.py`

### Objectif

Couvrir les 4 fonctions sans tests identifiées dans le backlog. Viser ≥ 85% sur `migrations.py`.

### A.1 — `ensure_weapon_kills_table()`

**Fichier** : `tests/test_migrations.py`  
**Classe** : `TestWeaponKillsMigration` (nouvelle)

| Test | Description | Assertion |
|------|-------------|-----------|
| `test_ensure_weapon_kills_creates_table` | Appel sur DB vide | Table `weapon_kills` existe, colonnes `match_id`, `xuid`, `time_ms`, `weapon_id` (UBIGINT), `delta_ms`, `confidence`, `swap_detected`, `delayed_damage` |
| `test_ensure_weapon_kills_idempotent` | Double appel | Pas d'erreur, schéma identique |
| `test_ensure_weapon_kills_creates_index` | Après création | Index `idx_wk_match_xuid` existe |
| `test_migrate_weapon_name_to_id_perkill` | Table legacy per-kill (`weapon_name VARCHAR`, `time_ms` présent) | Colonne `weapon_id` UBIGINT, `weapon_name` absent, données converties |
| `test_migrate_weapon_name_aggregated_drops` | Table legacy agrégée (`weapon_name + kills`, pas `time_ms`) | Table DROP+CREATE vide (schéma v5.7) |
| `test_upgrade_bigint_to_ubigint` | Table avec `weapon_id BIGINT` | Type devient UBIGINT, données préservées |

**Dépendances** :
- `src/analysis/_weapon_data.py` : `WEAPON_NAME_TO_INT`, `MELEE_WEAPON_ID`, `GRENADE_WEAPON_ID` — nécessaires pour le test de conversion
- Fixture `conn` (DuckDB in-memory) déjà disponible dans `conftest.py`

**Logging vérifié** :
- Chaque migration loggue `logger.info("✅ Migration weapon_kills …")` — asserter via `caplog` fixture

### A.2 — `ensure_bot_teammate_column()`

**Classe** : `TestBotTeammateColumn` (nouvelle)

| Test | Description | Assertion |
|------|-------------|-----------|
| `test_ensure_bot_teammate_noop_no_table` | Pas de table `player_match_enrichment` | Pas d'erreur, pas de table créée |
| `test_ensure_bot_teammate_adds_column` | Table `player_match_enrichment` sans `had_bot_teammate` | Colonne ajoutée, type BOOLEAN |
| `test_ensure_bot_teammate_idempotent` | Double appel | Pas d'erreur, schéma inchangé |
| `test_ensure_bot_teammate_preserves_data` | Table avec données existantes | Données intactes, valeur par défaut NULL ou FALSE |

### A.3 — `add_spartan_id_to_career_progression()`

**Classe** : `TestSpartanIdCareerProgression` (nouvelle)

| Test | Description | Assertion |
|------|-------------|-----------|
| `test_add_spartan_id_noop_no_table` | Pas de `career_progression` | Pas d'erreur |
| `test_add_spartan_id_adds_column` | Table sans `spartan_id` | Colonne ajoutée, type VARCHAR |
| `test_add_spartan_id_idempotent` | Double appel | Schéma identique |

**Pré-requis** : Lire la signature exacte de `add_spartan_id_to_career_progression()` dans `migrations.py:454` pour connaître le type exact et la valeur par défaut.

### A.4 — `_recreate_highlight_events_with_sequence()` (chemin idempotent)

**Classe existante** : `TestHighlightEventsMigration` — ajouter un test

| Test | Description | Assertion |
|------|-------------|-----------|
| `test_recreate_already_has_sequence` | Table avec séquence `nextval` déjà en place | Pas de double création, données intactes |

**Implémentation** : Créer la table avec séquence via DDL direct, insérer des données, appeler `ensure_highlight_events_autoincrement()`, vérifier que `nextval` fonctionne toujours et que les données sont préservées.

### A.5 — Mesure de couverture

```bash
python -m pytest tests/test_migrations.py --cov=src/data/sync/migrations --cov-report=term-missing -v
```

**Cible** : ≥ 85% (actuellement ~60%).  
**Fichier de rapport** : Inclure le résultat dans le commit message ou le thought_log.

---

## B — Résidus Pandas → Polars

### ⚠️ Précautions

Ce chantier touche les graphiques Plotly. Risque de régression visuelle si les types passés à Plotly changent. Règles :

1. **Plotly accepte des listes Python** — pas besoin de Pandas. Passer `.to_list()` sur les Series Polars.
2. **`px.sunburst()` accepte un dict ou DataFrame Pandas** — utiliser un dict de listes Polars converties.
3. **`st.dataframe()` accepte Polars nativement** depuis Streamlit ≥ 1.25.
4. **`st.line_chart()` accepte Polars nativement** avec `x=` et `y=` — pas besoin de `.set_index()`.
5. **Tester visuellement** chaque graphique après migration (pie, bar, sunburst, stacked bar, line chart).

### B.1 — `src/visualization/participation_charts.py` (3 occurrences)

**Fonctions impactées** :

#### `plot_participation_pie()` (L58–L100)

État actuel :
```python
pdf = agg.to_pandas()
pdf["label"] = pdf["award_category"].map(lambda x: ...)
pdf["color"] = pdf["award_category"].map(lambda x: ...)
penalties = pdf[pdf["total_score"] < 0]["total_score"].sum()
pdf_positive = pdf[pdf["total_score"] > 0].copy()
```

Migration :
```python
agg = agg.with_columns([
    pl.col("award_category").map_elements(
        lambda x: cat_labels.get(x, x.capitalize() if x else viz_t("cat_label_other", lang)),
        return_dtype=pl.Utf8,
    ).alias("label"),
    pl.col("award_category").map_elements(
        lambda x: CATEGORY_COLORS.get(x, CATEGORY_COLORS["other"]),
        return_dtype=pl.Utf8,
    ).alias("color"),
])
penalties = agg.filter(pl.col("total_score") < 0)["total_score"].sum()
agg_positive = agg.filter(pl.col("total_score") > 0)
```

Passage à Plotly :
```python
labels=agg_positive["label"].to_list(),
values=agg_positive["total_score"].to_list(),
marker={"colors": agg_positive["color"].to_list()},
```

**Risque** : `penalties` peut être `None` si aucune ligne négative → utiliser `penalties or 0`.

**Test** : Exécuter le pie chart avec des données de test (catégories kill, assist, penalty). Vérifier que le donut s'affiche correctement avec la bonne palette Okabe-Ito.

#### `plot_participation_bars()` (L145–L195)

État actuel :
```python
pdf = agg.to_pandas()
pdf["award_label"] = pdf["award_name"].map(lambda x: ...)
pdf["color"] = pdf["award_category"].map(lambda x: ...)
pdf = pdf.iloc[::-1]  # reverse
```

Migration :
```python
agg = agg.with_columns([
    pl.col("award_name").map_elements(
        lambda x: i18n_label("awards", x, lang=lang) or x,
        return_dtype=pl.Utf8,
    ).alias("award_label"),
    pl.col("award_category").map_elements(
        lambda x: CATEGORY_COLORS.get(x, CATEGORY_COLORS["other"]),
        return_dtype=pl.Utf8,
    ).alias("color"),
])
if orientation == "h":
    agg = agg.reverse()
```

Passage : `.to_list()` partout pour les traces.  

**Attention au `.empty`** : `pdf.empty` → `agg.is_empty()`.

#### `plot_participation_by_match()` (L280–L310)

État actuel :
```python
pdf = pivoted.to_pandas()
# Itère sur colonnes catégorie
for cat in categories_order:
    if cat in pdf.columns:
        fig.add_trace(go.Bar(x=pdf["match_id"], y=pdf[cat], ...))
```

Migration :
```python
# Polars pivot déjà fait — itérer sur les colonnes
for cat in categories_order:
    if cat in pivoted.columns:
        fig.add_trace(go.Bar(
            x=pivoted["match_id"].to_list(),
            y=pivoted[cat].to_list(),
            ...
        ))
```

**Risque nul** — le pivot Polars produit les mêmes colonnes.

### B.2 — `src/visualization/participation_charts_extra.py` (1 occurrence)

**Fonction** : `plot_participation_sunburst()` (L195)

État actuel :
```python
pdf = agg.to_pandas()
pdf["award_label"] = pdf["award_name"].map(...)
pdf["category_label"] = pdf["award_category"].map(...)
fig = px.sunburst(pdf, path=["category_label", "award_label"], values="score", ...)
```

Migration :
```python
agg = agg.with_columns([
    pl.col("award_name").map_elements(...).alias("award_label"),
    pl.col("award_category").map_elements(...).alias("category_label"),
])
fig = px.sunburst(
    agg.to_pandas(),  # px.sunburst EXIGE un DataFrame — garder cette conversion
    path=["category_label", "award_label"],
    values="score",
    ...
)
```

**⚠️ DÉCISION** : `px.sunburst()` ne supporte **pas** les DataFrame Polars (Plotly Express lit `.columns` et `.itertuples()` qui sont des API Pandas). **Conserver `.to_pandas()` ici** avec un commentaire explicite :
```python
# px.sunburst exige un DataFrame Pandas — conversion à la frontière Plotly
```

Alternative : construire le sunburst manuellement avec `go.Sunburst(ids=, labels=, parents=, values=)` pour éliminer Pandas. **Coût** : ~30 lignes de code supplémentaire pour calculer la hiérarchie parent/enfant. **Recommandation** : garder `to_pandas()` ici, c'est une frontière légitime.

### B.3 — `src/ui/pages/objective_analysis.py` (3 occurrences)

**Fonctions** : `_render_assist_table()` et `_render_awards_frequency()`

État actuel (x3) :
```python
tbl = some_df.to_pandas()
tbl.iloc[:, 0] = tbl.iloc[:, 0].map(lambda x: i18n_label(...))
tbl.columns = ["Award", "Points", "Occurrences"]
st.dataframe(tbl, ...)
```

Migration :
```python
tbl = some_df.rename({"col1": "Award", "col2": "Points", "col3": "Occurrences"})
tbl = tbl.with_columns(
    pl.col("Award").map_elements(
        lambda x: i18n_label("awards", x, lang=get_lang()) or x,
        return_dtype=pl.Utf8,
    )
)
st.dataframe(tbl, ...)
```

**Attention** : Les noms de colonnes originaux dans `assist_by_type` et `obj_freq` / `all_freq` doivent être vérifiés. Les fonctions `compute_award_frequency_polars()` et l'agrégation assist retournent probablement `award_name`, `score`, `count` → vérifier avant de renommer.

**Pattern pour les 3 occurrences** : Extraire un helper si le pattern se répète 3 fois :
```python
def _translate_award_df(df: pl.DataFrame, lang: str, columns: dict[str, str]) -> pl.DataFrame:
    """Traduit la colonne award_name et renomme les colonnes pour affichage."""
    name_col = next(c for c in df.columns if "name" in c.lower())
    return df.with_columns(
        pl.col(name_col).map_elements(
            lambda x: i18n_label("awards", x, lang=lang) or x,
            return_dtype=pl.Utf8,
        )
    ).rename(columns)
```

**Lieu** : Dans `objective_analysis.py` en fonction privée (pas de helper séparé — 1 seul fichier consommateur).

### B.4 — `src/ui/components/duckdb_analytics.py` (1 occurrence)

État actuel (L163) :
```python
st.line_chart(chart_df.to_pandas().set_index("Match"), width="stretch")
```

Migration :
```python
st.line_chart(chart_df, x="Match", y="KDA", width="stretch")
```

**Risque nul** — `st.line_chart` supporte Polars nativement avec les paramètres `x=` et `y=`.

### B.5 — Tests de non-régression Pandas

Ajouter un test qui vérifie l'absence de `import pandas` (en dehors des frontières autorisées) :

**Fichier** : `tests/test_legacy_free_ui_viz_wave_a.py` (existant ou nouveau)

Patterns à scanner :
```python
EXCLUDED_FILES = {
    "_compat.py",           # frontière Plotly
    "win_loss.py",          # pd.DataFrame.style (pas d'alternative Polars)
    "rag.py",               # LanceDB API
    "polars_compat.py",     # utilitaire conversion
    "_arrow_bridge.py",     # pont Arrow/Polars
    "streamlit_bridge.py",  # frontière UI (to_pandas timezone conversion)
    "participation_charts_extra.py",  # px.sunburst (frontière Plotly)
}
```

Vérifier que les fichiers migrés (B.1, B.3, B.4) n'importent plus `pandas`.

---

## C — Dead code guard Pandas `_performance_relative.py`

### C.1 — Supprimer le guard `was_pandas`

**Fichier** : `src/analysis/_performance_relative.py` (L220–L240)

État actuel :
```python
was_pandas = not isinstance(df, pl.DataFrame)
df_pl = _normalize_df(df)
...
return result.to_pandas() if was_pandas else result
```

Tous les appelants identifiés passent du Polars → le chemin Pandas n'est **jamais emprunté**.

Migration :
```python
df_pl = df if isinstance(df, pl.DataFrame) else pl.from_pandas(df)
...
return result
```

**Simplification** : Retirer les 3 occurrences de `return result.to_pandas() if was_pandas else result` → `return result`.

### C.2 — Évaluer les guards dans les helpers

#### `src/analysis/sessions.py` L56

```python
if not isinstance(df, pl.DataFrame):
    df = pl.from_pandas(df)
```

**Vérification nécessaire** : Lister tous les appelants de `compute_sessions()` :
- Si **tous** passent du Polars → supprimer la guard
- Si un appelant externe/script pourrait passer du Pandas → **conserver** la guard avec commentaire `# garde de compatibilité — supprimer quand tous les appelants migrés`

**Action** : `grep -rn "compute_sessions" src/ scripts/` pour lister les appelants.

#### `src/analysis/_performance_relative_helpers.py` L26

```python
def _normalize_df(df):
    if isinstance(df, pl.DataFrame):
        return df
    return pl.from_pandas(df)
```

**Même logique** : Vérifier si `_normalize_df` est appelé depuis du code qui pourrait encore passer du Pandas. Si tout est Polars → simplifier en `assert isinstance(df, pl.DataFrame)` ou supprimer.

### C.3 — `src/data/integration/streamlit_bridge.py`

**NE PAS MIGRER dans v5.7**. Ce module :
- A un contrat explicite "retourne du Pandas" (docstring + commentaires)
- Utilise `pd.to_datetime().dt.tz_convert()` pour la timezone — pas d'équivalent simple en Polars pur
- Est importé par l'UI Streamlit qui attend du Pandas

**Action v5.7** : Ajouter un commentaire `# v5.7 : frontière Pandas légitime — migration évaluée pour v5.8+` en tête de `matches_to_dataframe()`.

---

## D — Hover thumbnail cartes (CSS pur)

### D.1 — Supprimer le JS sandboxé

**Fichier** : `src/ui/styles.py`

1. Supprimer la variable `_MAP_TOOLTIP_SCRIPT` (L14–L55)
2. Modifier `load_css()` (L71) : retirer `\n{_MAP_TOOLTIP_SCRIPT}` du return

Le JS est sandboxé par Streamlit (`st.markdown(unsafe_allow_html=True)` n'exécute pas les `<script>`) → il ne fonctionnait pas. Sa suppression n'a aucun impact visuel.

### D.2 — Implémenter le CSS pur

**Approche** : Classes CSS uniques par cellule, popup via `background-image` + `:hover`.

#### Nouveau CSS dans `static/styles.css`

```css
/* ── Map thumbnail hover (CSS-only, pas de JS) ── */
.map-hover {
    position: relative;
    cursor: default;
}
.map-hover .map-popup {
    display: none;
    position: absolute;
    bottom: 100%;
    left: 50%;
    transform: translateX(-50%);
    z-index: 9999;
    width: 220px;
    border-radius: 6px;
    border: 1px solid rgba(255, 255, 255, 0.18);
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.75);
    pointer-events: none;
}
.map-hover:hover .map-popup {
    display: block;
}
```

**Problème** : CSS pur `background-image: url(...)` ne peut pas être injecté dynamiquement par cellule via une classe CSS statique.

**Solution retenue** : Inline style dans le HTML généré. Chaque `<td>` de carte devient :
```html
<td>
  <span class="map-hover">
    Aquarius
    <img class="map-popup" src="/app/static/maps/aquarius.jpg" alt="" />
  </span>
</td>
```

L'attribut `src` de l'`<img>` est sanitizé via `html.escape()`. L'image est servie par Streamlit via le dossier `static/`.

**Note** : `st.markdown(unsafe_allow_html=True)` permet le rendu HTML mais bloque les scripts. Les balises `<img>` sont autorisées par Streamlit quand `unsafe_allow_html=True`.

**⚠️ VÉRIFICATION NÉCESSAIRE** : Tester que `<img>` est rendu dans `st.markdown(..., unsafe_allow_html=True)`. Si Streamlit filtre les `<img>`, utiliser `background-image` en inline style :
```html
<span class="map-hover">
  Aquarius
  <span class="map-popup" style="background-image:url('/app/static/maps/aquarius.jpg');
    background-size:cover; width:220px; height:130px;"></span>
</span>
```

### D.3 — Modifier les générateurs HTML

#### `src/ui/pages/match_table_html.py` (L290–L296)

Remplacer :
```python
return f"<td><span class='map-cell' data-thumb-url='{esc_url}'>{esc_val}</span></td>"
```
Par :
```python
return (
    f"<td><span class='map-hover'>{esc_val}"
    f"<img class='map-popup' src='{esc_url}' alt='' /></span></td>"
)
```

#### `src/ui/pages/win_loss_table_style.py` (L53)

Même remplacement.

### D.4 — Améliorer `_build_map_url_index()`

**`lru_cache(maxsize=1)` → `lru_cache(maxsize=None)`** : Le `maxsize=1` est fragile — si la fonction est appelée avec des arguments différents un jour, le cache sera évincé. `maxsize=None` est sûr car la fonction ne prend aucun argument.

**Normalisation Unicode** :
```python
import unicodedata

stem = unicodedata.normalize("NFC", f.stem.lower())
```

### D.5 — Tests

| Test | Fichier | Description |
|------|---------|-------------|
| `test_map_hover_html_generated` | `tests/ui/test_match_table_html.py` | Vérifier que le HTML contient `class='map-hover'` et `<img class='map-popup'` pour une carte connue |
| `test_map_hover_no_url_fallback` | Idem | Carte sans thumbnail → `<td>` simple sans popup |
| `test_no_js_in_load_css` | `tests/test_styles.py` (nouveau ou existant) | `load_css()` ne contient pas `<script>` |
| `test_build_map_url_index_unicode` | `tests/ui/test_match_table_html.py` | Carte avec accent (ex. "Détente") resolve correctement |

---

## E — Détection de langue `LevelUp.sh` / `LevelUp.bat`

### E.1 — `LevelUp.sh` : Détection POSIX

Insérer **en tête du script** (après le shebang et les variables) :

```sh
# ── Détection langue système ──
_locale="${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}"
_lang_code=$(echo "$_locale" | cut -c1-2 | tr '[:upper:]' '[:lower:]')
case "$_lang_code" in
    [a-z][a-z]) : ;;
    *)           _lang_code="" ;;
esac
case "$_lang_code" in
    fr) SCRIPT_LANG="fr" ;;
    *)  SCRIPT_LANG="en" ;;
esac
```

### E.2 — `LevelUp.bat` : Détection via registre Windows

Insérer en tête (après `@echo off` et `setlocal`) :

```bat
:: ── Détection langue système ──
set "SCRIPT_LANG=en"
for /f "tokens=3" %%L in ('reg query "HKCU\Control Panel\International" /v LocaleName 2^>nul') do (
    set "_WIN_LOCALE=%%L"
)
if defined _WIN_LOCALE (
    set "_TMP=!_WIN_LOCALE:~0,2!"
    if /i "!_TMP!"=="fr" set "SCRIPT_LANG=fr"
)
```

**Note** : `LevelUp.bat` utilise déjà `setlocal EnableDelayedExpansion` et `chcp 65001` → les accents FR fonctionneront.

### E.3 — Inventaire des chaînes localisables

#### `LevelUp.sh` (~50 chaînes `echo`)

Regrouper les messages en bloc de variables en tête :

```sh
# ── Messages FR ──
if [ "$SCRIPT_LANG" = "fr" ]; then
    MSG_WSL_WARNING="  ⚠  WSL2 : projet sur un chemin Windows"
    MSG_WSL_PERF="     Les performances I/O seront dégradées."
    MSG_WSL_RECOMMEND="     Recommandé : déplacez le projet dans ~/LevelUp (ext4)."
    MSG_REINSTALL="  🔄 Suppression du venv (--reinstall)..."
    MSG_VENV_DEAD="  ⚠  Interpréteur du venv inaccessible (Python désinstallé ?), recréation..."
    MSG_VENV_INCOMPLETE="  ⚠  Environnement incomplet détecté, réinstallation..."
    MSG_DEPS_CHANGED="  🔄 pyproject.toml modifié — mise à jour des dépendances..."
    MSG_DEPS_OK="  ✓ Dépendances à jour."
    MSG_DEPS_PARTIAL="  ⚠  Mise à jour partielle"
    MSG_FIRST_LAUNCH_TITLE="     LevelUp - Premier lancement"
    MSG_PYTHON_NOT_FOUND="  ❌ Python 3.10+ introuvable sur ce système."
    MSG_VENV_MODULE_MISSING="  ❌ Le module 'venv' est absent de"
    MSG_CREATING_VENV="  Création de l'environnement virtuel..."
    MSG_VENV_FAIL="  ❌ Impossible de créer le venv."
    MSG_PIP_UPDATE="  Mise à jour de pip..."
    MSG_PIP_WARN="  ⚠  pip non mis à jour (réseau ou proxy). Poursuite avec la version installée."
    MSG_INSTALLING="  Installation des dépendances (quelques minutes à la première exécution)..."
    MSG_INSTALL_FAIL="  ❌ Échec de l'installation. Causes possibles :"
    MSG_INSTALL_FAIL_NETWORK="     - Pas de connexion internet"
    MSG_INSTALL_FAIL_READONLY="     - Dossier en lecture seule (déplacez LevelUp dans ~/Documents)"
    MSG_INSTALL_FAIL_DISK="     - Espace disque insuffisant (df -h)"
    MSG_READY="  ✓ Environnement prêt."
    # ... (compléter l'inventaire exhaustif)
fi

# ── Messages EN ──
if [ "$SCRIPT_LANG" = "en" ]; then
    MSG_WSL_WARNING="  ⚠  WSL2: project on a Windows path"
    MSG_WSL_PERF="     I/O performance will be degraded."
    MSG_WSL_RECOMMEND="     Recommended: move the project to ~/LevelUp (ext4)."
    MSG_REINSTALL="  🔄 Removing venv (--reinstall)..."
    MSG_VENV_DEAD="  ⚠  Venv interpreter inaccessible (Python uninstalled?), recreating..."
    MSG_VENV_INCOMPLETE="  ⚠  Incomplete environment detected, reinstalling..."
    MSG_DEPS_CHANGED="  🔄 pyproject.toml changed — updating dependencies..."
    MSG_DEPS_OK="  ✓ Dependencies up to date."
    MSG_DEPS_PARTIAL="  ⚠  Partial update"
    MSG_FIRST_LAUNCH_TITLE="     LevelUp - First launch"
    MSG_PYTHON_NOT_FOUND="  ❌ Python 3.10+ not found on this system."
    MSG_VENV_MODULE_MISSING="  ❌ The 'venv' module is missing from"
    MSG_CREATING_VENV="  Creating virtual environment..."
    MSG_VENV_FAIL="  ❌ Unable to create venv."
    MSG_PIP_UPDATE="  Updating pip..."
    MSG_PIP_WARN="  ⚠  pip not updated (network or proxy). Continuing with installed version."
    MSG_INSTALLING="  Installing dependencies (this may take a few minutes on first run)..."
    MSG_INSTALL_FAIL="  ❌ Installation failed. Possible causes:"
    MSG_INSTALL_FAIL_NETWORK="     - No internet connection"
    MSG_INSTALL_FAIL_READONLY="     - Read-only folder (move LevelUp to ~/Documents)"
    MSG_INSTALL_FAIL_DISK="     - Insufficient disk space (df -h)"
    MSG_READY="  ✓ Environment ready."
    # ... (compléter)
fi
```

#### `LevelUp.bat` (~45 chaînes `echo`)

Même pattern avec `set "MSG_..."` :

```bat
if /i "%SCRIPT_LANG%"=="fr" (
    set "MSG_REINSTALL=  Suppression du venv (--reinstall)..."
    set "MSG_READY=  Environnement pret."
    rem ... (toutes les chaînes)
) else (
    set "MSG_REINSTALL=  Removing venv (--reinstall)..."
    set "MSG_READY=  Environment ready."
    rem ...
)
```

### E.4 — Remplacer les `echo` littéraux

Remplacement systématique dans le corps des scripts :
```sh
# Avant
echo "  ✓ Environnement prêt."
# Après
echo "$MSG_READY"
```

### E.5 — Tests manuels

| Scénario | Commande | Résultat attendu |
|----------|----------|-----------------|
| SH — système FR | `LANG=fr_FR.UTF-8 ./LevelUp.sh --help` | Messages en français |
| SH — système EN | `LANG=en_US.UTF-8 ./LevelUp.sh --help` | Messages en anglais |
| SH — locale C | `LANG=C ./LevelUp.sh --help` | Messages en anglais (fallback) |
| SH — locale vide | `unset LANG LC_ALL; ./LevelUp.sh --help` | Messages en anglais |
| BAT — système FR | Exécuter sur Windows FR | Messages en français |
| BAT — système EN | Exécuter sur Windows EN | Messages en anglais |

### E.6 — Estimation

- **SH** : ~219 lignes actuelles → ~280 lignes avec bloc messages (~+60 lignes)
- **BAT** : ~233 lignes → ~310 lignes (~+77 lignes)
- Aucune dépendance externe ajoutée
- 100% backward-compatible (FR par défaut sur systèmes FR)

---

## F — Traductions FR manquantes migration metadata

### F.1 — Contexte

`scripts/migration/migrate_metadata_to_duckdb.py` L72 :
```python
"tier_name_fr": rank_data.get("RankTitle", {}).get("value", ""),  # TODO: ajouter traductions FR
```

Le champ `tier_name_fr` est identique à `tier_name_en` — pas de traduction.

### F.2 — Source des traductions

Les rangs Halo Infinite ont des noms officiels :
- Bronze, Silver, Gold, Platinum, Diamond, Onyx (rangs CSR)
- Recruit, Private, Corporal, Sergeant, etc. (rangs career/spartan)

**Options** :
1. **Table statique** : Dict `{eng_name: fr_name}` dans le script de migration
2. **Fichier i18n existant** : Vérifier si `src/ui/i18n/` contient déjà les traductions de rangs

### F.3 — Implémentation

Créer un mapping dans `src/ui/i18n/data_labels.py` ou un fichier dédié `src/ui/i18n/ranks.py` :

```python
RANK_NAMES_FR: dict[str, str] = {
    "Bronze": "Bronze",
    "Silver": "Argent",
    "Gold": "Or",
    "Platinum": "Platine",
    "Diamond": "Diamant",
    "Onyx": "Onyx",
    "Recruit": "Recrue",
    "Private": "Soldat",
    "Corporal": "Caporal",
    "Sergeant": "Sergent",
    # ... compléter avec les 50+ rangs career
}
```

Puis dans `migrate_metadata_to_duckdb.py` :
```python
from src.ui.i18n.ranks import RANK_NAMES_FR

"tier_name_fr": RANK_NAMES_FR.get(en_name, en_name),
```

### F.4 — Tests

| Test | Assertion |
|------|-----------|
| `test_rank_names_fr_complete` | Toutes les clés de `RANK_NAMES_FR` sont non-vides |
| `test_tier_name_fr_not_identical_to_en` | Au moins 80% des rangs ont une traduction FR différente de EN (exclure Bronze, Onyx qui sont identiques) |

---

## G — Version bump + tag + release

### G.1 — Pré-requis avant bump

Checklist obligatoire :

- [ ] Tous les tests passent : `python -m pytest --ignore=tests/integration -q`
- [ ] Pas d'erreurs de type
- [ ] Pas de `import pandas` hors frontières autorisées
- [ ] Thought log à jour
- [ ] CHANGELOG mis à jour

### G.2 — Mise à jour `pyproject.toml`

```toml
version = "5.7.0"
```

### G.3 — CHANGELOG

Ajouter une section dans `docs/CHANGELOG.md` :

```markdown
## [5.7.0] — 2026-03-XX

### Added
- Hover thumbnail sur les noms de cartes dans les tableaux (CSS pur)
- Détection automatique de la langue système dans les lanceurs (FR/EN)
- Traductions FR des rangs dans metadata.duckdb

### Changed
- Migration Pandas → Polars dans participation_charts, objective_analysis, duckdb_analytics
- Suppression du script JS tooltip sandboxé (remplacé par CSS pur)

### Fixed
- Guard Pandas inutile dans _performance_relative.py (dead code path)

### Tests
- +12 tests migrations.py (ensure_weapon_kills, ensure_bot_teammate, add_spartan_id, highlight idempotent)
- Couverture migrations.py : ~60% → ≥85%
- Test de non-régression import pandas
```

### G.4 — Tag et release

**Option A** — Via GitHub Actions (recommandée) :
1. Pusher la branche `release/v5.7`
2. Merger dans `main`
3. Lancer le workflow `Bump Version & Tag` avec `bump=minor` et `note=weapon_kills, CSS tooltips, i18n launchers`
4. Le workflow crée le commit de bump, le tag `v5.7.0`, et déclenche `release.yml`

**Option B** — Manuellement :
```bash
# Après merge dans main
git tag -a v5.7.0 -m "Release v5.7.0"
git push origin v5.7.0
```

Le tag `v5.7.0` déclenche automatiquement `.github/workflows/release.yml` qui :
1. Build les artefacts Windows portable + Unix
2. Crée la release GitHub avec les assets uploadés

### G.5 — Vérification post-release

- [ ] Release GitHub créée avec 2 assets (`.zip` Win64 + `.tar.gz` Unix)
- [ ] Release notes auto-générées par `softprops/action-gh-release@v2`
- [ ] Les artefacts sont téléchargeables et fonctionnels

---

## Ordre d'exécution recommandé

```
1. [A] Tests migrations      ← fondation, aucune dépendance
2. [C] Dead code Pandas      ← nettoyage préalable à B
3. [B] Migration Pandas→Polars ← sensible, tester visuellement
4. [F] Traductions FR ranks  ← indépendant, rapide
5. [D] Hover thumbnail CSS   ← UI, tester visuellement
6. [E] i18n lanceurs         ← indépendant, scripts bash/bat
7. [G] Version bump + tag    ← dernier, après validation complète
```

---

## Matrice de risque

| Chantier | Risque | Mitigation |
|----------|--------|------------|
| B — Pandas→Polars viz | **Moyen** — régression visuelle possible | Tester chaque chart manuellement avant commit |
| B.2 — Sunburst | **Bas** — `px.sunburst` maintient `to_pandas()` | Frontière documentée |
| D — Hover CSS | **Bas** — JS actuel ne fonctionnait pas | Amélioration pure |
| E — i18n lanceurs | **Bas** — backward-compatible | Test sur systèmes FR et EN |
| A — Tests migrations | **Nul** — ajout de tests uniquement | — |
| C — Dead code | **Bas** — chemin jamais emprunté | Grep appelants avant suppression |

---

## Logging à ajouter/vérifier

| Fichier | Logger | Messages attendus |
|---------|--------|-------------------|
| `src/data/sync/migrations.py` | `logger` existant | Déjà en place (`✅ Migration weapon_kills…`) |
| `src/ui/pages/match_table_html.py` | `_log` existant | `_log.debug("Index thumbnails cartes : %d entrées")` — déjà en place |
| `src/ui/styles.py` | Aucun | Pas nécessaire (CSS statique) |
| `LevelUp.sh` / `LevelUp.bat` | `echo` | Messages déjà verbeux — ajouter `[FR]`/`[EN]` en mode debug si besoin |
| `src/analysis/_performance_relative.py` | `logger` | Ajouter `logger.debug("compute_performance_series: %d matchs", len(df))` une fois le guard supprimé |

---

## Tests — Couverture cible globale

| Module | Avant v5.7 | Cible v5.7 | Delta |
|--------|-----------|-----------|-------|
| `src/data/sync/migrations.py` | ~60% | ≥85% | +25% |
| `src/visualization/participation_charts.py` | Existants | Idem + test no-pandas | +1 test |
| `src/ui/pages/match_table_html.py` | Existants | +2 tests hover | +2 tests |
| `src/ui/styles.py` | Peu testé | +1 test no-script | +1 test |
| `src/analysis/_performance_relative.py` | Existants | Idem (code simplifié) | 0 |

**Commande de mesure** :
```bash
python -m pytest --cov=src --cov-report=term-missing --ignore=tests/integration -q
```

---

## H — Top 10 meilleurs / pires matchs (onglet carrière) [BACKLOG v5.8+]

> **⛔ BLOQUÉ** : ce chantier nécessite la récupération des médailles par équipe (alliée et ennemie)
> via l'API Halo. La table `medals_earned` (shared_matches) couvre uniquement les médailles du
> joueur suivi — on n'a pas encore les agrégats par équipe adverse. À planifier **après** l'implémentation
> du backfill des médailles équipes.

### H.1 — Objectif

Afficher dans l'onglet Carrière deux tableaux :
- **Top 10 meilleures performances** : matchs dans lesquels le joueur a le plus brillé
- **Top 10 pires performances** : matchs les plus difficiles ou dominés par l'adversaire

Les matchs sélectionnés doivent être **nets** : victoire ou défaite franche, sans pollution
par des robots, des déconnexions ou des matchs abandonnés.

### H.2 — Critères d'exclusion (filtre préalable)

Les matchs suivants sont **retirés du pool** avant tout classement :

| Critère | Source | Logique |
|---------|--------|---------|
| Bots dans n'importe quelle équipe | `match_participants` — si au moins 1 participant a un flag bot (à confirmer via SPNKr) | Exclure — match non-fair quelle que soit l'équipe |
| Match non terminé (DNF) | `match_registry.outcome = DNF` (outcome code) | Exclure |
| Résultat nul (TIE) | `outcome = TIE` | Exclure — pas de vainqueur/perdant net |
| Durée anormalement courte | `match_registry.duration_seconds < MIN_MATCH_DURATION_SECONDS` (constante à calibrer sur p5 percentile) | Exclure — rage quit probable très tôt, le match n'a pas eu lieu |

> **Note bots** : La colonne `had_bot_teammate` (player_match_enrichment) ne couvre que
> l'équipe alliée. La détection correcte **passe par `match_participants`** : si au moins
> 1 ligne de `match_participants` pour ce `match_id` correspond à un bot (flag à confirmer
> selon schéma SPNKr — field `bot_attributes` ou pattern xuid), le match entier est exclu
> quelle que soit l'équipe impactée.

> **Note rage quit partiel** : Un coéquipier qui déconnecte en cours de match (mais le
> match continue) n'est pas exclu — les matchs courts sont le proxy correct pour les arrêts
> prématurés complets. Un match qui dure suffisamment longtemps avec 1 DC reste valide.

### H.3 — Critères de scoring

**Logique retenue : 3 niveaux de priorité**

Le scoring est intentionnellement simple et mode-agnostique grâce à la médaille "À table" :

| Priorité | Critère | Explication |
|----------|---------|-------------|
| **1 (principal)** | Présence de la médaille "À table" dans une équipe | Critère de domination mode-agnostique, fourni directement par le moteur Halo — pas besoin de normaliser le score par mode |
| **2 (départage)** | Durée du match (`duration_seconds`) courte | Court = une équipe a écrasé l'autre rapidement. Longue = le match était serré ou en overtime |
| **3 (départage fin)** | Écart de score brut | Différentiel points/kills/caps selon le mode, pour distinguer à l'intérieur d'un même groupe durée |

**Logique de sélection :**

- **Top 10 meilleures performances** :
  1. D'abord les matchs où l'équipe alliée a reçu "À table" (triés par durée croissante — plus court = domination plus nette)
  2. Puis les matchs de victoire sans "À table" (triés par écart de score décroissant)
  
- **Top 10 pires performances** :
  1. D'abord les matchs où l'équipe ennemie a reçu "À table" (triés par durée croissante — plus court = humiliation plus nette)
  2. Puis les matchs de défaite sans "À table" (triés par écart de score croissant — défaite la plus large)

> **Sur la durée courte = mauvais signe** : Si c'est serré, les deux équipes se sont défendues
> et c'est un bon match. Une domination/humiliation nette se manifeste précisément par une
> durée courte (l'équipe dominée n'a pas pu résister). Ce critère est cohérent.

> **"À table" / Steaktacular (ID `1169390319`)** : Décernée à TOUTE l'équipe gagnante en
> cas de victoire parfaite ou quasi-parfaite. Aucune relevance individuelle — c'est exactement
> ce qu'on veut : un proxy de domination collective, indépendant du mode de jeu.

### H.4 — Questions ouvertes / Décisions différées

| # | Question | Impact | Statut |
|---|----------|--------|--------|
| H.4.1 | Rage quit partiel (1 DC en cours, match continue) | Faible | **Résolu** : non exclu si durée suffisante — proxy durée min suffit |
| H.4.2 | Seuil de durée minimale | Faible | **Décision** : calibrer sur p5 percentile de l'historique du joueur. Constante `MIN_MATCH_DURATION_SECONDS` dans `src/analysis/_match_quality_filter.py` |
| H.4.3 | Bots ennemis : critère de détection dans `match_participants` | **Bloquant partiel** | Investiguer flag bot SPNKr dans le schéma — champ `bot_attributes` ? Ou pattern xuid `bot_*` ? À confirmer avant implémentation du filtre |
| H.4.4 | Agrégat médailles par équipe (alliée vs ennemie) | **Bloquant principal** | Nécessite `medals_earned` jointé avec `match_participants.team_id` pour séparer les deux équipes. Non disponible actuellement |
| H.4.5 | Modes FFA (Free-for-All) | Faible | Pas d'équipes formelles → "À table" non applicable → utiliser uniquement écart de score + durée pour ces modes |

### H.5 — Architecture de données envisagée

Une fois les médailles équipes disponibles (agrégat par `team_id`) :

```sql
-- CTE filtre de qualité (matchs "propres")
WITH clean_matches AS (
    SELECT mr.match_id
    FROM match_registry mr
    JOIN match_participants mp ON mp.match_id = mr.match_id AND mp.xuid = :player_xuid
    WHERE mr.outcome NOT IN ('DNF', 'TIE')
      AND mr.duration_seconds >= :min_duration            -- constante MIN_MATCH_DURATION_SECONDS
      AND NOT EXISTS (                                    -- aucun bot dans quelque équipe que ce soit
          SELECT 1 FROM match_participants bp
          WHERE bp.match_id = mr.match_id AND bp.is_bot = TRUE  -- flag à confirmer §H.4.3
      )
),

-- CTE médailles par équipe (bloquant — à construire avec team_id)
medals_by_team AS (
    SELECT
        me.match_id,
        mp.team_id,
        SUM(CASE WHEN me.medal_name_id = 1169390319 THEN me.count ELSE 0 END) AS steaktacular_count,
        SUM(me.count) AS total_medals
    FROM medals_earned me
    JOIN match_participants mp ON mp.match_id = me.match_id AND mp.xuid = me.xuid
    GROUP BY me.match_id, mp.team_id
),

-- Score final
match_scores AS (
    SELECT
        mr.match_id,
        mr.mode,
        mr.outcome,
        mr.duration_seconds,
        mr.start_time,
        mr.map_id,
        mp.score          AS player_score,
        mp.team_id        AS my_team_id,
        -- Médaille "À table" côté allié vs ennemi
        COALESCE(ally.steaktacular_count, 0)  > 0 AS ally_got_steaktacular,
        COALESCE(enemy.steaktacular_count, 0) > 0 AS enemy_got_steaktacular,
        -- Score brut de l'équipe (pour départage L3)
        (SELECT SUM(team_score) FROM match_participants tp
         WHERE tp.match_id = mr.match_id AND tp.team_id = mp.team_id LIMIT 1) AS ally_team_score,
        (SELECT SUM(team_score) FROM match_participants tp
         WHERE tp.match_id = mr.match_id AND tp.team_id != mp.team_id LIMIT 1) AS enemy_team_score
    FROM match_registry mr
    JOIN clean_matches cm ON cm.match_id = mr.match_id
    JOIN match_participants mp ON mp.match_id = mr.match_id AND mp.xuid = :player_xuid
    LEFT JOIN medals_by_team ally   ON ally.match_id = mr.match_id  AND ally.team_id = mp.team_id
    LEFT JOIN medals_by_team enemy  ON enemy.match_id = mr.match_id AND enemy.team_id != mp.team_id
)

-- Top 10 meilleures performances
SELECT * FROM match_scores
WHERE outcome = 'WIN'
ORDER BY
    ally_got_steaktacular DESC,   -- priorité 1 : on a eu "À table"
    duration_seconds ASC,         -- priorité 2 : plus court = domination plus nette
    (ally_team_score - enemy_team_score) DESC  -- priorité 3 : écart de score
LIMIT 10;

-- Top 10 pires performances (même CTE, clause ORDER BY inversée)
SELECT * FROM match_scores
WHERE outcome = 'LOSS'
ORDER BY
    enemy_got_steaktacular DESC,  -- priorité 1 : ennemis ont eu "À table"
    duration_seconds ASC,         -- priorité 2 : plus court = humiliation plus nette
    (enemy_team_score - ally_team_score) DESC  -- priorité 3 : écart de score
LIMIT 10;
```

### H.6 — UI envisagée

```
[Onglet Carrière]
  └── Section "Matchs marquants"
        ├── 🏆 Top 10 meilleures performances
        │     Tableau : Date | Mode | Carte | Résultat | Écart score | Médailles alliées | Durée
        └── 💀 Top 10 pires performances
              Tableau : idem
              
  [Filtre optionnel] : Mode de jeu | Saison
```

- Tri par défaut : écart score normalisé (desc victoires, desc défaites)
- Chaque ligne cliquable → ouvre le détail du match (lien vers vue match existante)
- Badge coloré sur l'écart de score (vert gradient pour victoires, rouge pour défaites)

### H.7 — Pré-requis avant implémentation

- [ ] **[Bloquant]** Confirmer le flag de détection bot dans `match_participants` (§H.4.3) — champ `is_bot` ? pattern xuid ? à valider avec schéma SPNKr/API
- [ ] **[Bloquant]** Construire l'agrégat médailles par équipe : `medals_earned` JOIN `match_participants` sur `(match_id, xuid)` → agréger par `team_id` → inclure `steaktacular_count` (medal_name_id = `1169390319`)
- [ ] Décision sur `MIN_MATCH_DURATION_SECONDS` — calibrer sur p5 percentile de l'historique (ex. 3 min = 180 s comme point de départ)
- [ ] Valider comportement FFA (§H.4.5) — "À table" applicable ou non ?
