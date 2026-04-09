# Plan — Harmonisation de la couche informationnelle Timeseries / Teammates

> Contexte : analyse comparative réalisée le 2026-04-09.
> Objectif : aligner le niveau d'aide contextuelle de la page **Teammates** sur celui de **Timeseries** (captions conditionnels, notes post-graphe, couverture `hints_visible()`), et harmoniser les visualisations elles-mêmes (axes, constantes, graphes désactivés, données manquantes).
> Branche à créer : `feat/info-layer-teammates` depuis la branche courante.

---

## Partie A — Couche informationnelle

### Diagnostic résumé

| Critère | Timeseries | Teammates | Écart |
|---------|:----------:|:---------:|:-----:|
| Sections avec `hints_visible()` | 8 | 2 | ⚠️ -6 |
| Notes post-graphe (`_render_note`) | 4 | 0 | ❌ absentes |
| Captions conditionels protégés | 8 | 1 | ⚠️ -7 |
| Captions toujours visibles (non protégés) | 0 | 2 | ⚠️ |
| Clé i18n morte (définie, jamais appelée) | 0 | 1 | ⚠️ `tm_kd_half_caption` |
| Chaîne hardcodée hors système `t()` | 0 | 1 | ❌ `teammates_weapons.py:302` |

---

## Phase 1 — Extraction du composant partagé `_render_note`

**Fichier source** : `src/ui/pages/timeseries.py` (fonction `_render_note`, ~20 L)
**Fichier cible** : `src/ui/components/info_note.py` (nouveau)
**Impacté après** : `timeseries.py` (réimport), tous les sous-modules teammates qui en ont besoin

### Tâches

- [ ] Créer `src/ui/components/info_note.py` avec la fonction `render_info_note(text: str) -> None`
  - Corps identique à `_render_note` actuelle (HTML + CSS classe `ts-note`, condition `hints_visible()`)
  - Signature publique, docstring FR
- [ ] Dans `timeseries.py` : remplacer la définition locale par `from src.ui.components.info_note import render_info_note as _render_note`
- [ ] Vérifier que les 4 appels `_render_note(t("ts_note_*"))` fonctionnent toujours

**Risque** : aucun — refactoring pur, comportement inchangé côté timeseries.

---

## Phase 2 — Correction du caption hardcodé

**Fichier** : `src/ui/pages/teammates_weapons.py` ligne 302

### Tâches

- [ ] Ajouter la clé `tm_weapons_no_data` dans `src/ui/i18n/pages/teammates.py` :
  ```python
  "tm_weapons_no_data": {
      "fr": "Aucune donnée d'armes pour ces matchs.",
      "en": "No weapon data available for these matches.",
  },
  ```
- [ ] Remplacer le literal hardcodé par `st.caption(t("tm_weapons_no_data"))`

**Risque** : nul.

---

## Phase 3 — Suppression de la clé morte

**Clé** : `tm_kd_half_caption` — définie dans `src/ui/i18n/pages/teammates.py` ligne 138, **aucun appelant dans le codebase**.

### Option A — Supprimer
- [ ] Retirer la clé de `teammates.py` i18n
- [ ] Confirmer via `grep -r tm_kd_half_caption src/` qu'aucune référence ne subsiste

### Option B — Brancher sur la section K/D existante (teammates_charts.py)
- [ ] Localiser la section K/D dans `teammates_charts.py` (subheader `tm_kills_deaths`)
- [ ] Ajouter `if hints_visible(): st.caption(t("tm_kd_half_caption"))` sous le subheader

> **Recommandation** : Option B — la clé a une valeur sémantique réelle ("F/M 1ère → 2nde moitié"), sa suppression serait un gaspillage.

---

## Phase 4 — Protection des captions existants avec `hints_visible()`

Deux captions sont actuellement affichés **sans condition** alors qu'ils sont descriptifs (pas juste informatifs).

### 4a — `tm_map_timeline_caption` (`teammates_map_charts.py` lignes 74 et 155)

```python
# Avant
st.caption(t("tm_map_timeline_caption"))

# Après
if hints_visible():
    st.caption(t("tm_map_timeline_caption"))
```

- [ ] Appliquer aux **deux** occurrences (lignes 74 et 155)
- [ ] Ajouter `from src.ui.components.browser_storage import hints_visible` si absent (déjà importé ligne 20 ✅)

### 4b — `tm_history_tz_caption` (`teammates_views.py` ligne 232)

> Cas particulier : ce caption affiche le fuseau horaire — c'est **davantage contextuel** qu'explicatif. À évaluer : peut-on le garder toujours visible ?

- [ ] Décision à prendre : conserver toujours visible (info timezone utile) **ou** conditionner
- [ ] Option retenue : `if hints_visible(): st.caption(t("tm_history_tz_caption", tz=get_tz_name()))` (cohérence stricte) — **ou** laisser tel quel si la timezone est considérée info critique.

---

## Phase 5 — Ajout des captions manquants sur les sections clés

Pour chaque section sans caption, ajouter la protection `hints_visible()` + texte explicatif.

### 5a — Matrice d'impact (`teammates_impact.py`)

- [ ] Ajouter clé `tm_impact_caption` dans `src/ui/i18n/pages/teammates.py` :
  ```python
  "tm_impact_caption": {
      "fr": "Heatmap des rôles que chaque joueur a tenus le plus souvent sur les matchs communs.",
      "en": "Heatmap of the roles each player most frequently held across shared matches.",
  },
  ```
- [ ] Ajouter import `hints_visible` dans `teammates_impact.py`
- [ ] Insérer `if hints_visible(): st.caption(t("tm_impact_caption"))` après le subheader de la heatmap

### 5b — Bar chart armes (`teammates_weapons.py`)

Contexte : le graphe compare les kills par arme entre plusieurs joueurs. Aucune aide de lecture.

- [ ] Ajouter clé `tm_weapons_chart_caption` dans i18n :
  ```python
  "tm_weapons_chart_caption": {
      "fr": "Kills par arme — côte à côte sur les matchs communs. Révèle les profils d'armement de chaque joueur.",
      "en": "Kills by weapon — side by side across shared matches. Reveals each player's weapon preferences.",
  },
  ```
- [ ] Ajouter `if hints_visible(): st.caption(t("tm_weapons_chart_caption"))` dans `render_weapon_kills_bar_chart`

### 5c — Métriques comparatives (`teammates_charts.py` — `render_metric_bar_charts`)

- [ ] Ajouter clé `tm_metrics_caption` dans i18n :
  ```python
  "tm_metrics_caption": {
      "fr": "Comparaison des métriques clés entre joueurs sur les matchs communs.",
      "en": "Comparison of key metrics between players across shared matches.",
  },
  ```
- [ ] Ajouter `if hints_visible(): st.caption(t("tm_metrics_caption"))` dans `render_metric_bar_charts`

---

## Phase 6 — Ajout des notes post-graphe (`render_info_note`)

Notes interprétatives (dépend de la Phase 1).

### 6a — Radar de complémentarité (`teammates_synergy.py`)

- [ ] Ajouter clé `tm_note_radar` dans i18n :
  ```python
  "tm_note_radar": {
      "fr": "- Une **surface large** sur un axe → forte implication dans ce rôle\n"
            "- Deux profils **complémentaires** → leurs pointes couvrent des axes différents → bonne synergie\n"
            "- Deux profils **superposés** → même style de jeu — force sur les matchs agressifs, fragilité en cas de déficit offensif",
      "en": "- A **large area** on one axis → high involvement in that role\n"
            "- Two **complementary profiles** → their peaks cover different axes → good synergy\n"
            "- Two **overlapping profiles** → same playstyle — strong in aggressive matches, fragile when behind",
  },
  ```
- [ ] Appeler `render_info_note(t("tm_note_radar"))` après le chart dans `_render_radar_display`

### 6b — Profil de cadence (`teammates_map_charts.py`)

- [ ] Ajouter clé `tm_note_cadence` dans i18n :
  ```python
  "tm_note_cadence": {
      "fr": "- Des pics **synchronisés** → vous jouez aux mêmes moments → push coordonné\n"
            "- Des pics **décalés** → profils complémentaires en avant-poste / couverture\n"
            "- Courbes plates → peu de kills en dehors des phases d'objectif",
      "en": "- **Synchronized peaks** → you fight at the same time → coordinated push\n"
            "- **Offset peaks** → complementary front / cover split\n"
            "- Flat curves → few kills outside objective phases",
  },
  ```
- [ ] Appeler `render_info_note(t("tm_note_cadence"))` après le chart de cadence

### 6c — Matrice d'impact (`teammates_impact.py`)

> La légende `tm_impact_legend` existe et est affichée via `st.markdown`. À terme elle pourrait migrer vers `render_info_note` pour harmoniser le style visuel. Traiter en Phase 6c.

- [ ] Évaluer si le style `ts-note` améliore la lisibilité vs markdown brut
- [ ] Si oui : remplacer `st.markdown(t("tm_impact_legend"))` par `render_info_note(t("tm_impact_legend"))`

---

## Récapitulatif des fichiers impactés

| Fichier | Phases | Type de changement |
|---------|--------|--------------------|
| `src/ui/components/info_note.py` | 1 | **Créer** |
| `src/ui/pages/timeseries.py` | 1 | Réimport `_render_note` |
| `src/ui/i18n/pages/teammates.py` | 2, 3, 5, 6 | +6 clés, -1 clé morte (si option A) |
| `src/ui/pages/teammates_weapons.py` | 2, 5b | Clé hardcodée + caption chart |
| `src/ui/pages/teammates_map_charts.py` | 4a, 6b | `hints_visible()` + note cadence |
| `src/ui/pages/teammates_views.py` | 4b | `hints_visible()` optional |
| `src/ui/pages/teammates_impact.py` | 5a, 6c | Import + caption + note |
| `src/ui/pages/teammates_charts.py` | 3B, 5c | Caption K/D + caption métriques |
| `src/ui/pages/teammates_synergy.py` | 6a | Note radar |

---

## Ordre d'exécution recommandé

```
Phase 1  →  Phase 2  →  Phase 3 (décision opt A/B)
    ↓
Phase 4a + 4b (indépendantes, parallélisables)
    ↓
Phase 5a + 5b + 5c (indépendantes)
    ↓
Phase 6a + 6b + 6c (dépend Phase 1)
```

---

## Checklist pré-PR (Partie A)

- [ ] `python -m pytest -q --ignore=tests/integration`
- [ ] Aucune chaîne hardcodée restante dans les fichiers teammates modifiés
- [ ] `hints_visible()` importé dans chaque fichier qui l'utilise
- [ ] `render_info_note` importé uniquement depuis `src/ui/components/info_note`
- [ ] `timeseries.py` : comportement visuellement identique à avant la Phase 1
- [ ] Traductions EN cohérentes avec le ton des clés `ts_note_*` existantes

---

---

## Partie B — Graphes, données, tableaux

### Inventaire des visualisations actives

#### Timeseries (~30 visualisations)

| # | Graphe | Fonction viz | Type | Onglet |
|---|--------|-------------|------|--------|
| 1 | KDA timeseries | `plot_timeseries` | Scatter / ligne | Résumé |
| 2 | Distribution KDA (FDA) | `plot_kda_distribution` | Histogramme + KDE | Résumé |
| 3 | Résultats au fil du temps | `plot_outcomes_over_time` | Barres groupées | Résumé |
| 4 | Séries V/D | *(rendu interne win_loss)* | Métriques texte | Résumé |
| 5 | W/L par carte | `plot_stacked_outcomes_by_category` | Barres empilées | Cartes |
| 6 | W/L par mode | `plot_stacked_outcomes_by_category` | Barres empilées | Cartes |
| 7 | Win rate vs historique | `_render_winrate_perf_vs_history` | Bullet gauge | Cartes |
| 8–13 | 6 histogrammes (précision, kills, durée vie, perf, score/min, wr glissant) | `plot_histogram` | Histogramme + KDE | Distributions |
| 14–16 | 3 corrélations scatter (vie/kills, précision/kda, MMR) | `plot_correlation_scatter` | Scatter coloré | Distributions |
| 17 | Premier frag / première mort | `plot_first_event_distribution` | Histogramme groupé | Progression |
| 18 | Performance timeseries | `plot_performance_timeseries` | Multi-ligne | Progression |
| 19 | Assists timeseries | `plot_assists_timeseries` | Ligne | Progression |
| 20 | Stats par minute | `plot_per_minute_timeseries` | Multi-ligne | Progression |
| 21 | Durée de vie moy. | `plot_average_life` | Ligne | Progression |
| 22 | Folie meurtrière + HS + PK | `plot_spree_headshots_accuracy` | Multi-ligne | Progression |
| 23 | Tirs & précision | `plot_shots_accuracy` | Barres + ligne | Progression |
| 24 | Dégâts infligés / subis | `plot_damage_dealt_taken` | Barres groupées | Progression |
| 25 | Rang & score personnel | `plot_rank_score` | Barres + ligne | Progression |
| 26 | Score personnel agrégé | `_render_personal_score_section` | Tableau | Progression |
| 27 | Heatmap d'intensité intra-match | `plot_match_intensity_heatmap` | Heatmap | Avancé |
| 28 | Armes (kills, solo) | `plot_top_weapons` | Barres horizontales | Avancé (via Résumé) |
| 29 | Skill rank LUSR/CSR | `plot_lusr_timeseries` | Ligne | Avancé |
| 30 | Net score/heure cumulé | `plot_net_score_per_hour` | Aire colorée | Avancé |
| 31 | K/D cumulé avec IC | `plot_cumulative_kd_with_ci` | Ligne + bande | Avancé |
| 32 | EWMA K/D + tendance | `plot_ewma_kd` | Ligne lissée + droite | Avancé |
| 33 | Régression K/D | `plot_regression_trend` | Scatter + droite | Avancé |
| 34 | Heatmap W/L calendrier | `_render_wl_heatmap_section` | Heatmap calendrier | Avancé |
| 35 | Meilleures semaines | `_render_top_by_week` | Tableau | Avancé |

#### Teammates (~20 visualisations actives, 1 désactivée)

| # | Graphe | Fonction viz | Type | Fichier |
|---|--------|-------------|------|---------|
| 1 | KPIs section (mes stats) | `render_kpis_section` | Métriques | `teammates.py` |
| 2 | W/L bullet par carte | `plot_map_winrate_bullet` | Bullet gauge | `teammates_map_charts.py` |
| 3 | Perf vs historique par carte | `plot_map_perf_vs_history` | Bullet gauge | `teammates_map_charts.py` |
| 4 | Heatmap escouade joueur × carte | `plot_squad_map_heatmap` | Heatmap | `teammates_map_charts.py` |
| 5 | Timeline performance escouade | `plot_squad_performance_timeline` | Multi-ligne | `teammates_map_charts.py` |
| 6 | Kills↑ / Morts↓ (butterfly) | `plot_trio_kills_deaths` | Barres butterfly | `teammates_charts.py` |
| 7 | Assists | `plot_trio_metric` | Ligne multi | `teammates_charts.py` |
| 8 | KDA ratio | `plot_trio_metric` | Ligne multi | `teammates_charts.py` |
| 9 | Précision | `plot_trio_metric` | Ligne multi | `teammates_charts.py` |
| 10 | Durée de vie | `plot_trio_metric` | Ligne multi | `teammates_charts.py` |
| 11 | Performance score | `plot_trio_metric` | Ligne multi | `teammates_charts.py` |
| 12 | Killing Spree | `plot_multi_metric_bars_fn` | Barres + lissage | `teammates_charts.py` |
| 13 | HS + PK stacké | `plot_hs_pk_stacked` | Barres empilées | `teammates_charts.py` |
| 14 | Premier frag/mort butterfly (multi) | `_build_first_events_fig` | Barres butterfly | `teammates_charts.py` |
| 15 | Radar de complémentarité | `create_participation_profile_radar` | Radar 6 axes | `teammates_synergy.py` |
| 16 | Matrice d'impact | `_render_impact_ranking_html` | Tableau HTML custom | `teammates_impact.py` |
| 17 | Armes : bar chart comparatif | `render_weapon_kills_bar_chart` | Barres H groupées | `teammates_weapons.py` |
| 18 | Tableau d'armes | `render_weapon_kills_table` | Tableau HTML custom | `teammates_weapons.py` |
| 19 | Profil de cadence synchronisée | *(interne map_charts)* | Courbes phase | `teammates_map_charts.py` |
| 20 | Tableau historique matchs | `render_friends_history_table` | Tableau interactif | `teammates_helpers.py` |
| ~~21~~ | ~~Timeline par carte~~ | `plot_map_outcome_timeline` | ~~Scatter chronologique~~ | **DÉSACTIVÉ** `if False` |

---

### Asymétries identifiées

#### Asymétrie 1 — Métriques Timeseries sans équivalent comparatif dans Teammates

Ces graphes existent uniquement en mode solo. Certains sont **légitimement** absents (les distributions mono-joueur n'ont pas de sens en comparatif direct) ; d'autres pourraient être **adaptés** en version escouade.

| Métrique | Existe dans TS | Existe dans TM | Adaptable en multi ? |
|----------|:--------------:|:--------------:|:--------------------:|
| 6 histogrammes de distribution | ✅ | ❌ | ⚠️ en version superposée (KDE par joueur) |
| 3 corrélations scatter | ✅ | ❌ | ⚠️ 1 scatter par joueur / même axes = lisible |
| Dégâts infligés vs subis | ✅ | ❌ | ✅ directement (trio_metric) |
| Tirs & précision (shots fired/hit) | ✅ | ❌ | ✅ directement (trio_metric) |
| Rang & score personnel | ✅ | ❌ | ✅ score déjà dans trio_metric(performance) |
| Heatmap d'intensité intra-match | ✅ | ❌ | ✅ superposer les profils de 2–3 joueurs |
| Progressions cumulatives (NPH, K/D+IC, EWMA) | ✅ | ❌ | ⚠️ pertinent en mode 1 coéquipier seulement |
| Heatmap W/L calendrier | ✅ | ❌ | ❌ mono-joueur par nature |
| Meilleures semaines | ✅ | ❌ | ❌ mono-joueur par nature |

#### Asymétrie 2 — Métriques Teammates sans équivalent dans Timeseries

Ces visualisations sont propres à la dynamique escouade — leur absence dans Timeseries est normale et souhaitée.

| Métrique | Teammates | Timeseries | Commentaire |
|----------|:---------:|:----------:|-------------|
| Heatmap escouade (joueur × carte) | ✅ | ❌ | Concept multi-joueur uniquement |
| Radar de complémentarité (6 axes) | ✅ | ❌ | Concept multi-joueur uniquement |
| Matrice d'impact (événements clés) | ✅ | ❌ | Concept multi-joueur uniquement |
| Profil de cadence synchronisée | ✅ | ❌ | Concept multi-joueur uniquement |
| Premier frag/mort comparatif butterfly | ✅ | — (solo/density) | Les deux approches coexistent, lecture différente |

#### Asymétrie 3 — Graphes en double portée avec implémentations divergentes

Ces métriques existent dans les deux pages mais avec des **visuels ou axes différents**, ce qui peut désorienter un utilisateur qui navigue d'une page à l'autre.

| Métrique | Timeseries | Teammates | Nature de l'écart |
|----------|:----------:|:---------:|-------------------|
| KDA / ratio | `plot_timeseries` — axe X = `start_time` | `plot_trio_metric(ratio)` — axe X = index de match | **Axe X incohérent** |
| Performance | `plot_performance_timeseries` — axe X = temps | `plot_trio_metric(performance)` — axe X = index de match | **Axe X incohérent** |
| Assists | `plot_assists_timeseries` — axe X = temps | `plot_trio_metric(assists)` — axe X = index de match | **Axe X incohérent** |
| Durée de vie | `plot_average_life` — axe X = temps | `plot_trio_metric(average_life_seconds)` — axe X = index | **Axe X incohérent** |
| Précision | histogramme distribution | `plot_trio_metric(accuracy)` — axe X = index | **Type de graphe incompatible** |
| Armes (kills) | `plot_top_weapons` — barres H simples, échelle absolue | `render_weapon_kills_bar_chart` — barres H groupées, même échelle | **Conception similaire, code séparé** |
| Premier frag/mort | `plot_first_event_distribution` — density par tranche | `_build_first_events_fig` — butterfly barres groupées | **Approches fondamentalement différentes** |
| W/L bullet par carte | `_render_winrate_perf_vs_history` | `plot_map_winrate_bullet` | ✅ **Code partagé** (même fonction) |
| Perf vs historique par carte | `_render_winrate_perf_vs_history` | `plot_map_perf_vs_history` | ✅ **Code partagé** (même fonction) |

#### Asymétrie 4 — Graphe désactivé sans décision documentée

`plot_map_outcome_timeline` dans `teammates_map_charts.py` est encapsulé derrière `if False:` à **deux endroits** (lignes 65 et 134). Le code est maintenu mais ne s'exécute plus. La clé i18n `tm_map_timeline_caption` pointait vers ce graphe — elle est donc en partie orpheline.

```python
# teammates_map_charts.py — lignes 63–70 et 132–140
if False:  # timeline disabled — conserver le code pour usage futur  # noqa: SIM210
    st.markdown(f"##### {t('tm_map_timeline_title')}")
    st.caption(t("tm_map_timeline_caption"))
    with safe_chart_render():
        fig_tl = plot_map_outcome_timeline(...)
```

**Décision requise** : réactiver ou supprimer définitivement (avec la clé i18n).

#### Asymétrie 5 — Constante d'exclusion dupliquée (dette code)

La liste des weapon_ids à exclure est définie à deux endroits différents :

| Localisation | Définition | Valeur |
|---|---|---|
| `src/analysis/_weapon_data.py` | `EXCLUDED_WEAPON_IDS` | set complet |
| `src/ui/pages/teammates_weapons.py` | `_FILM_EXCLUDED_IDS = {0, 1}` | redéfini localement |

`teammates_weapons.py` n'importe pas `EXCLUDED_WEAPON_IDS` et redéfinit les IDs 0 et 1 comme sentinels locaux. Si `EXCLUDED_WEAPON_IDS` évolue dans `_weapon_data.py`, `teammates_weapons.py` ne reflétera pas le changement.

---

### Candidats à harmonisation — détail et aide à la décision

Les candidats sont classés par **impact utilisateur** (ce que voit l'utilisateur) et **effort d'implémentation**.

---

#### Candidat H1 — Étiquettes d'axe X adaptatives selon le contexte temporel 🔴

> **Reformulation suite à révision 2026-04-09** — La prémisse initiale (aligner Teammates sur l'axe date de Timeseries) était incorrecte. La vraie problématique est plus large et concerne **les deux pages**.

**Constat** : l'axe X d'un graphe de progression (KDA, performance, assists…) devrait afficher l'étiquette qui **discrimine le mieux** les matchs dans la fenêtre affichée. Or :

- Sur une **session** (même journée, quelques heures) : la date ne discrimine pas — tous les matchs ont la même. Le numéro de match + la carte est plus lisible.
- Sur **plusieurs jours** (< 2 semaines) : la date DD/MM + carte ancre chaque match dans la mémoire du joueur.
- Sur **plusieurs semaines** (2 sem – 3 mois) : la semaine (« S14 ») ou la date suffit.
- Sur **plusieurs mois** (> 3 mois) : le mois-année est le bon niveau de granularité.

Aujourd'hui Timeseries affiche toujours la date brute, Teammates toujours l'index ordinal. Les deux peuvent être inadaptés selon la fenêtre filtrée.

**Problématique ouverte** : définir une fonction `choose_axis_label(start_times: pl.Series) -> AxisLabelFormat` qui choisit le format selon l'étendue temporelle et la densité des matchs. Voir section dédiée [§ Problématique H1 — Axe adaptatif](#problématique-h1--axe-x-adaptatif-selon-le-contexte).

**Décision pour ce sprint** : **reporter** — pas de changement sur les axes dans ce sprint. H1 est réarchitecturé comme une feature à part entière, pas un fix de synchronisation TS↔TM.

---

### Problématique H1 — Axe X adaptatif selon le contexte

#### Principe directeur

L'étiquette doit **discriminer** les matchs dans la vue courante, pas afficher un format fixe. La variable discriminante change avec l'étendue temporelle :

| Étendue temporelle | Discriminant naturel | Format recommandé |
|--------------------|---------------------|-------------------|
| < 1 jour (session) | Position + carte | `"Match 3 · Bazaar"` |
| 1 – 13 jours | Date + carte | `"03/11 · Recharge"` |
| 14 – 89 jours | Semaine | `"S44 2025"` |
| ≥ 90 jours | Mois | `"Nov 2025"` |

La carte peut être tronquée (8 car.) si l'axe X est dense. Elle reste utile en tooltip hover même si supprimée du tick.

#### Heuristique proposée

```python
def choose_axis_label_format(start_times: pl.Series) -> AxisLabelFormat:
    span_days = (start_times.max() - start_times.min()).total_seconds() / 86400
    if span_days < 1:
        return AxisLabelFormat.MATCH_MAP      # "Match N · Carte"
    elif span_days < 14:
        return AxisLabelFormat.DATE_MAP       # "DD/MM · Carte"
    elif span_days < 90:
        return AxisLabelFormat.WEEK           # "S44 2025"
    else:
        return AxisLabelFormat.MONTH          # "Nov 2025"
```

#### Où s'applique-t-il ?

- `plot_trio_metric` (Teammates) — remplace l'index ordinal actuel
- `plot_timeseries` / `plot_performance_timeseries` (Timeseries) — remplace la date brute quand la fenêtre est courte

#### Implémentation suggérée

1. Définir `AxisLabelFormat` (enum) + `choose_axis_label_format()` dans `src/analysis/axis_labels.py`
2. Ajouter `build_axis_tick_labels(df: pl.DataFrame, time_col: str) -> pl.Series` qui construit les étiquettes selon le format choisi
3. Brancher sur `plot_trio_metric` (paramètre `x_labels: pl.Series | None = None`)
4. Brancher sur les fonctions Timeseries concernées (idem)

**Effort** : moyen (nouveau module + 2 fonctions viz modifiées). **Valeur** : haute — les deux pages en bénéficient.

**Sprint recommandé** : sprint dédié `feat/adaptive-axis-labels`, après ce sprint d'harmonisation.

---

#### Candidat H2 — Constante `_FILM_EXCLUDED_IDS` dupliquée 🔴

**Ce qui se passe** : `teammates_weapons.py` définit `_FILM_EXCLUDED_IDS = {0, 1}` localement. Si demain `EXCLUDED_WEAPON_IDS` dans `_weapon_data.py` ajoute un troisième ID sentinel, `teammates_weapons.py` ne filtrera pas correctement.

**Fichier concerné** : `src/ui/pages/teammates_weapons.py` lignes 20–22.

**Correction** :
```python
# Avant
_FILM_EXCLUDED_IDS = {0, 1}

# Après
from src.analysis._weapon_data import EXCLUDED_WEAPON_IDS as _FILM_EXCLUDED_IDS
```

**Effort** : 2 lignes. **Risque** : nul si les IDs 0 et 1 font bien partie de `EXCLUDED_WEAPON_IDS` (à vérifier).

---

#### Candidat H3 — Graphe désactivé `plot_map_outcome_timeline` 🔴

**Ce qui se passe** : le code de rendu est présent à deux endroits mais protégé par `if False:`. Il ne s'exécute jamais. Son seul effet est d'augmenter la charge cognitive de maintenance et de laisser une clé i18n en état semi-orphelin.

**Options** :

| Option | Description | Conséquence |
|--------|-------------|-------------|
| A — Supprimer | Retirer les deux blocs `if False:` + la fonction `plot_map_outcome_timeline` si exclusivement utilisée ici | Propre. Supprimer aussi `tm_map_timeline_title` si orpheline. |
| B — Réactiver | Retirer le `if False:` et vérifier que les données sont disponibles | Ajoute une vue utile (évolution chronologique par carte) |
| C — Garder en dormance | Ajouter un commentaire `# TODO: réactiver après test données` + date limite | Statu quo documenté |

**Recommandation** : vérifier si la fonction `plot_map_outcome_timeline` produit un résultat lisible sur données réelles. Si oui → B. Sinon → A.

---

#### Candidat H4 — Heatmap d'intensité intra-match dans Teammates 🟠

**Ce qui se passe** : la page Timeseries affiche une heatmap montrant le profil de kills normalisé par phase (10 buckets) pour chaque match, filtrable par V/D. La page Teammates n'a aucun équivalent, alors que **superposer les profils de 2–3 joueurs** sur les matchs communs produirait une lecture très directe de qui peake quand.

**Données disponibles** : `highlight_events` dans `shared_matches_v2.duckdb` contient les timestamps par xuid — les données sont là.

**Ce que ça donnerait** : plutôt qu'une heatmap (un axe = 1 match), on pourrait afficher des courbes d'intensité superposées (axe X = phases, axe Y = kills normalisés, une courbe par joueur). C'est plus proche du graphe de cadence existant que de la heatmap.

**Options** :

| Option | Description | Effort | Valeur ajoutée |
|--------|-------------|:------:|:--------------:|
| A — Courbes d'intensité superposées | Nouvelle fonction viz utilisant `compute_match_intensity_profiles` par joueur | Moyen | Haute — complète la cadence avec une vue normalisée |
| B — Réutiliser le graphe cadence existant | Le graphe cadence (`tm_squad_cadence`) couvre déjà cette lecture | Nul | — (doublon) |
| C — Ne pas ajouter | Le graphe cadence est suffisant | Nul | Statu quo |

**Décision (2026-04-09)** : **Reporter**. Le profil de cadence couvre le besoin. H4 reporté à un sprint dédié "analytics escouade avancé".

---

#### Candidat H5 — Distributions comparées (précision, KDA) dans Teammates 🟠

**Ce qui se passe** : Timeseries offre 6 histogrammes de distribution mono-joueur. Teammates n'a rien de similaire. En mode escouade, afficher 2–3 KDE de précision superposées sur les matchs communs serait instructif (qui est le plus régulier ?).

**Ce que ça donnerait** : une colonne de 2 graphes (précision, KDA) avec une KDE par joueur, dans un nouvel onglet ou à l'intérieur de la vue trio.

**Options** :

| Option | Description | Effort | Valeur ajoutée |
|--------|-------------|:------:|:--------------:|
| A — Onglet "Distributions" dans Teammates | Nouveau sous-onglet avec 2–3 KDE superposées (précision + KDA) | Moyen | Haute |
| B — Bloc ajouté dans la vue trio existante | Sous la comparaison de métriques, ajouter 2 histogrammes de distribution | Faible | Moyenne |
| C — Ne pas ajouter | | Nul | Statu quo |

**Décision (2026-04-09)** : **Reporter**. La page Teammates opère sur les matchs communs, qui peuvent être en nombre restreint (10–30 matchs). Une KDE sur un tel échantillon produit du bruit, pas une distribution interprétable. Valeur ajoutée insuffisante par rapport au risque de confusion.

---

#### Candidat H6 — Corrélations comparées dans Teammates 🟡

**Ce qui se passe** : Timeseries affiche 3 scatter plots montrant des corrélations (durée de vie vs kills, précision vs KDA, MMR vs résultat). En mode escouade, on pourrait afficher le même scatter mais avec des points colorés par joueur — ce qui révèle si les deux joueurs ont des patterns corrélés similaires ou opposés.

**Effort** : faible — `plot_correlation_scatter` accepte potentiellement une colonne de couleur ; sinon un scatter Plotly basique suffit.

**Valeur ajoutée** : révèle si le coéquipier est "du même profil" (précision ↔ KDA corrélés pareil) ou complémentaire.

**Décision (2026-04-09)** : **Reporter**. Candidat pour un sprint "analytics escouade avancé".

---

#### Candidat H7 — Unifier `plot_top_weapons` et `render_weapon_kills_bar_chart` 🟡

**Ce qui se passe** : les deux graphes d'armes font la même chose (barres horizontales de kills par arme) mais avec des implémentations distinctes, des styles légèrement différents et des logiques d'injection grenade/mêlée légèrement différentes aussi.

**Différences clés** :

| | `plot_top_weapons` (Timeseries) | `render_weapon_kills_bar_chart` (Teammates) |
|---|---|---|
| Joueurs | 1 | 1 à 4 |
| Grenade/mêlée | injection directe dans `weapons_data` | ajustement remainder film |
| Exclusion IDs | `EXCLUDED_WEAPON_IDS` | `_FILM_EXCLUDED_IDS` (local) |
| Style | barres simples | barres groupées + badges records |

**Option d'unification** : factoriser la logique de construction du DataFrame armes (résolution noms + injection grenade/mêlée + exclusion IDs) dans une fonction `build_weapon_kills_df(...)` dans `src/analysis/` et laisser chaque graphe appeler ses propres fonctions Plotly. L'unification totale n'est pas réaliste vu la divergence visuelle.

**Recommandation** : unifier uniquement la couche données (H2 + extraction `build_weapon_kills_df`). Laisser les deux rendus Plotly séparés.

---

#### Candidat H8 — Décision sur `plot_first_event_distribution` vs `_build_first_events_fig` 🟢

**Ce qui se passe** : deux graphes de "premier frag/premier mort" coexistent avec des approches fondamentalement différentes :
- Timeseries : `plot_first_event_distribution` → density curves (KDE), mono-joueur, tranches fines
- Teammates : `_build_first_events_fig` → butterfly barres groupées, multi-joueurs, tranches de 15s (`_BIN_SIZE_S = 15`)

La coexistence est **légitime** car les usages divergent. Le seul point d'harmonisation identifié : la constante `_BIN_SIZE_S = 15` est définie localement dans `teammates_charts.py` sans documentation. Si Timeseries utilisait un jour des bins similaires, les deux seraient désynchronisés.

**Correction mineure** : déplacer `_BIN_SIZE_S` en constante nommée dans un module partagé (`src/analysis/` ou une constante i18n-agnostique) et documenter le choix de 15s.

**Effort** : très faible. **Risque** : nul.

---

### Tableau de décision synthétique

| Candidat | Description courte | Priorité | Effort | Décision |
|----------|-------------------|:--------:|:------:|:---------|
| H1 | Axe X adaptatif (session/jour/semaine/mois) | 🔴 | Moyen | **Reformulé** — reporter, sprint dédié `feat/adaptive-axis-labels` |
| H2 | `_FILM_EXCLUDED_IDS` dupliqué | 🔴 | Très faible | **✅ Faire** — 2 lignes |
| H3 | Graphe timeline carte désactivé (`if False`) | 🔴 | Faible | **✅ Faire** — supprimer (dead code museum) |
| H4 | Heatmap intensité dans Teammates | 🟠 | Moyen | **Reporter** — cadence couvre le besoin |
| H5 | Distributions comparées dans Teammates | 🟠 | Faible-Moyen | **Reporter** — échantillon trop petit pour KDE fiable |
| H6 | Corrélations comparées dans Teammates | 🟡 | Faible | **Reporter** — sprint analytics escouade avancé |
| H7 | Unifier couche données armes | 🟡 | Moyen | **✅ Faire** — unifier données seulement (pas le rendu) |
| H8 | Constante `_BIN_SIZE_S` locale | 🟢 | Très faible | **✅ Faire** — extraire en constante nommée |

---

## Checklist pré-PR (Partie B)

- [ ] H2 : `_FILM_EXCLUDED_IDS` remplacé par `EXCLUDED_WEAPON_IDS` importé
- [ ] H3 : blocs `if False:` supprimés + fonction `plot_map_outcome_timeline` + clés i18n orphelines (`tm_map_timeline_title`, `tm_map_timeline_caption`) supprimées
- [ ] H7 : `build_weapon_kills_df` extrait dans `src/analysis/`, branché sur les deux pages
- [ ] H8 : `_BIN_SIZE_S` déplacé et documenté
- [ ] Aucune régression sur les graphes déjà fonctionnels (Timeseries + Teammates existant)

> H1 → sprint séparé `feat/adaptive-axis-labels`. H4, H5, H6 → backlog "analytics escouade avancé".

---

---

## Partie C — Sprint `feat/adaptive-axis-labels` : Références temporelles adaptatives

> Ce sprint est **indépendant** des Parties A et B. Il peut être développé en parallèle ou après.
> Branche : `feat/adaptive-axis-labels` depuis `main`.

### Objectif

Remplacer toutes les références temporelles fixes (dates brutes, index ordinaux) par un système centralisé qui choisit automatiquement le bon niveau de granularité selon la fenêtre de données affichée.

Le principe : **ce qui discrimine les matchs dans la vue courante** doit être l'étiquette. Pas un format imposé globalement.

---

### Vue d'ensemble du système

```
TemporalSpan              TemporalGranularity          Consumers
─────────────             ───────────────────          ──────────────────────────
from_series(pl.Series) ──► choose_granularity() ──────► build_axis_tick_labels()  → graphes
                                │                      ► format_temporal_ref()    → tooltips
                                │                      ► format_match_cell()      → tableaux
                                └──────────────────────► format_span_label()      → filtres UI / breadcrumbs
```

Tout le code métier est dans `src/analysis/temporal_format.py` — **zéro dépendance Streamlit ou Plotly**. Les adaptateurs UI sont dans des modules séparés.

---

### Phase C1 — Module core `src/analysis/temporal_format.py`

#### Types et constantes

```python
from enum import Enum
from dataclasses import dataclass
from datetime import datetime
import polars as pl

class TemporalGranularity(Enum):
    SESSION = "session"   # étendue < 1 jour
    DAY     = "day"       # 1 – 13 jours
    WEEK    = "week"      # 14 – 89 jours
    MONTH   = "month"     # ≥ 90 jours

# Seuils en jours — constantes nommées pour faciliter les tests et futures calibrations
_THRESHOLD_SESSION_DAYS: float = 1.0
_THRESHOLD_DAY_DAYS: float     = 14.0
_THRESHOLD_WEEK_DAYS: float    = 90.0

# Longueur max du nom de carte sur un tick (troncature)
_MAP_LABEL_MAX_LEN: int = 10
```

#### `TemporalSpan` — dataclass immuable

```python
@dataclass(frozen=True)
class TemporalSpan:
    min_dt: datetime
    max_dt: datetime
    match_count: int

    @property
    def duration_days(self) -> float:
        delta = self.max_dt - self.min_dt
        return delta.total_seconds() / 86400

    def granularity(self) -> TemporalGranularity:
        d = self.duration_days
        if d < _THRESHOLD_SESSION_DAYS:
            return TemporalGranularity.SESSION
        if d < _THRESHOLD_DAY_DAYS:
            return TemporalGranularity.DAY
        if d < _THRESHOLD_WEEK_DAYS:
            return TemporalGranularity.WEEK
        return TemporalGranularity.MONTH

    @classmethod
    def from_series(cls, start_times: pl.Series) -> "TemporalSpan":
        """Construit un TemporalSpan depuis une Series polars de datetimes."""
        if start_times.is_empty():
            raise ValueError("Impossible de construire un TemporalSpan depuis une Series vide")
        return cls(
            min_dt=start_times.min(),
            max_dt=start_times.max(),
            match_count=start_times.len(),
        )
```

#### Fonctions de formatage

```python
def format_match_tick(
    dt: datetime,
    granularity: TemporalGranularity,
    map_name: str | None = None,
    match_index: int | None = None,   # 1-based, utilisé en SESSION uniquement
    lang: str = "fr",
) -> str:
    """Étiquette d'axe pour un match unique."""
    ...
    # SESSION  → "Match 3 · Bazaar"     (index + carte)
    # DAY      → "03/11 · Recharge"     (date DD/MM + carte)
    # WEEK     → "S44 2025"             (numéro de semaine ISO)
    # MONTH    → "Nov 2025"             (mois-année)

def format_temporal_ref(
    dt: datetime,
    granularity: TemporalGranularity,
    lang: str = "fr",
) -> str:
    """Référence temporelle courte — tooltips, cellules tableau, breadcrumbs."""
    ...
    # SESSION  → "14:32"
    # DAY      → "03/11 14:32"
    # WEEK     → "S44 — Mer 03/11"
    # MONTH    → "Nov 2025"

def format_span_label(span: TemporalSpan, lang: str = "fr") -> str:
    """Label lisible pour représenter la fenêtre entière — filtres UI, titres."""
    ...
    # SESSION  → "Session du 03/11"
    # DAY      → "03/11 – 07/11/2025"
    # WEEK     → "S44 – S46 2025"
    # MONTH    → "Nov 2025 – Jan 2026"
```

#### `build_axis_tick_labels` — fonction principale d'intégration graphe

```python
def build_axis_tick_labels(
    df: pl.DataFrame,
    time_col: str = "start_time",
    map_col: str | None = "map_name",
    lang: str = "fr",
) -> pl.Series:
    """
    Retourne une Series de strings à passer comme axe X d'un graphe Plotly.
    Calcule automatiquement la granularité depuis df[time_col].
    Le résultat a la même longueur que df.
    """
    span = TemporalSpan.from_series(df[time_col])
    granularity = span.granularity()
    ...
```

**Contrat garanti** : même longueur que `df`, aucun `None`, chaînes non vides.

---

### Phase C2 — Tests `tests/test_temporal_format.py`

Les tests couvrent **uniquement** `src/analysis/temporal_format.py` — pas de Streamlit, pas de Plotly, pas de DB.

#### Couverture granularité

```python
def test_session_granularity_same_day():
    # 5 matchs sur 3 heures → SESSION

def test_session_granularity_boundary():
    # span = 23h59 → SESSION (< 1 jour)

def test_day_granularity_two_days():
    # span = 2 jours → DAY

def test_day_granularity_boundary_upper():
    # span = 13.9 jours → DAY (< 14)

def test_week_granularity():
    # span = 21 jours → WEEK

def test_week_granularity_boundary_upper():
    # span = 89 jours → WEEK

def test_month_granularity():
    # span = 90 jours → MONTH

def test_single_match_is_session():
    # 1 match → span = 0 → SESSION

def test_empty_series_raises():
    # Series vide → ValueError explicite
```

#### Couverture format_match_tick

```python
def test_session_label_with_map():
    # → "Match 3 · Bazaar"

def test_session_label_without_map():
    # → "Match 3"

def test_day_label_with_map():
    # → "03/11 · Recharge"

def test_week_label():
    # → "S44 2025"

def test_month_label():
    # → "Nov 2025"

def test_map_name_truncated_if_long():
    # carte de 15 caractères → tronquée à _MAP_LABEL_MAX_LEN

def test_lang_en_day_format():
    # lang="en" → "11/03 · Recharge" (MM/DD)
```

#### Couverture build_axis_tick_labels

```python
def test_output_length_matches_input():
    # len(result) == len(df)

def test_no_none_values():
    # result.null_count() == 0

def test_session_labels_include_match_index():
    # SESSION : "Match 1", "Match 2", "Match 3"...

def test_missing_map_col_graceful():
    # map_col absent du df → format sans carte, pas d'exception

def test_time_col_with_timezone():
    # start_time avec tz → pas d'erreur
```

#### Couverture format_span_label

```python
def test_span_label_session():
    # → "Session du 03/11"

def test_span_label_multi_month():
    # → "Nov 2025 – Jan 2026"

def test_span_label_same_month():
    # span dans le même mois → "Nov 2025" (pas de doublon mois)
```

**Objectif couverture** : 100 % des branches de `temporal_format.py`. Ce module est critique et pur — tout doit être testé.

---

### Phase C3 — Adaptateur Plotly `src/ui/chart_utils_temporal.py`

Séparation stricte : le module core ne connaît pas Plotly. L'adaptateur injecte les étiquettes dans les figures.

```python
# src/ui/chart_utils_temporal.py

from src.analysis.temporal_format import build_axis_tick_labels, TemporalSpan, format_temporal_ref

def apply_temporal_xaxis(
    fig: go.Figure,
    df: pl.DataFrame,
    time_col: str = "start_time",
    map_col: str | None = "map_name",
    lang: str = "fr",
) -> go.Figure:
    """
    Remplace les valeurs de l'axe X d'une figure Plotly par des étiquettes adaptatives.
    Modifie fig in-place et retourne fig (chaînable).
    Compatible avec tout type de trace (scatter, bar, line).
    """
    ...

def build_hover_temporal(
    df: pl.DataFrame,
    time_col: str = "start_time",
    lang: str = "fr",
) -> pl.Series:
    """Series de strings pour customdata hover — format adaptatif pour les tooltips."""
    span = TemporalSpan.from_series(df[time_col])
    return df[time_col].map_elements(
        lambda dt: format_temporal_ref(dt, span.granularity(), lang),
        return_dtype=pl.String,
    )
```

---

### Phase C4 — Adaptateur tableau `src/ui/table_utils_temporal.py`

```python
def format_date_column(
    df: pl.DataFrame,
    time_col: str = "start_time",
    lang: str = "fr",
) -> pl.DataFrame:
    """
    Remplace df[time_col] par une colonne string formatée adaptativement.
    À appeler juste avant le rendu HTML du tableau — ne modifie pas le df source.
    """
    ...
```

---

### Phase C5 — Intégration dans les graphes existants

**Stratégie de rollout** : opt-in progressif. Chaque fonction viz accepte un paramètre optionnel `temporal_labels: pl.Series | None = None`. Si fourni, il remplace l'axe X. Si absent, comportement actuel inchangé. Cela évite toute régression.

```python
# Exemple sur plot_trio_metric
def plot_trio_metric(
    df: pl.DataFrame,
    metric_col: str,
    ...,
    temporal_labels: pl.Series | None = None,   # ← nouveau, optionnel
) -> go.Figure:
    x_values = temporal_labels if temporal_labels is not None else df.get_column("start_time", default=pl.Series(range(len(df))))
    ...
```

L'appelant dans la page Streamlit construit les étiquettes une seule fois et les passe à tous les graphes de la section :

```python
# Dans teammates_charts.py
tick_labels = build_axis_tick_labels(df_matches, lang=current_lang())
fig1 = plot_trio_metric(df, "ratio", temporal_labels=tick_labels)
fig2 = plot_trio_metric(df, "performance_score", temporal_labels=tick_labels)
```

**Ordre d'intégration recommandé** (du plus visible au plus fin) :

| Priorité | Fonction | Page | Impact |
|----------|----------|------|--------|
| 1 | `plot_trio_metric` (×5) | Teammates | Supprime les "Match 1, 2, 3" sans contexte |
| 2 | `plot_performance_timeseries` | Timeseries | Très consulté |
| 3 | `plot_timeseries` (KDA) | Timeseries | Très consulté |
| 4 | `render_friends_history_table` (colonne date) | Teammates | Tableau principal |
| 5 | `plot_cumulative_kd_with_ci`, `plot_ewma_kd`, `plot_regression_trend` | Timeseries | Onglet Avancé |
| 6 | `plot_shots_accuracy`, `plot_damage_dealt_taken`, `plot_rank_score` | Timeseries | Onglet Progression |
| 7 | `plot_lusr_timeseries` | Timeseries | Onglet Avancé |
| 8 | Tooltips hover (via `build_hover_temporal`) | Tous | Fine-tuning |
| 9 | `format_span_label` dans les filtres UI | Global | Breadcrumb filtre |

---

### Phase C6 — Internationalisation

Les formats de date dépendent de la langue. Le système i18n existant (`t()`) doit exposer les formats de date par locale :

```python
# À ajouter dans src/ui/i18n/pages/common.py (ou un module dédié)
"date_format_day": {
    "fr": "%d/%m",       # 03/11
    "en": "%m/%d",       # 11/03
},
"date_format_day_time": {
    "fr": "%d/%m %H:%M",
    "en": "%m/%d %H:%M",
},
"week_label": {
    "fr": "S{week} {year}",
    "en": "W{week} {year}",
},
"month_format": {
    "fr": "%b %Y",       # "nov. 2025"
    "en": "%b %Y",       # "Nov 2025"
},
```

`format_match_tick` et `format_temporal_ref` acceptent `lang: str = "fr"` et délèguent la résolution des formats à ces constantes.

---

### Récapitulatif des fichiers

| Fichier | Action | Phase |
|---------|--------|-------|
| `src/analysis/temporal_format.py` | **Créer** | C1 |
| `tests/test_temporal_format.py` | **Créer** | C2 |
| `src/ui/chart_utils_temporal.py` | **Créer** | C3 |
| `src/ui/table_utils_temporal.py` | **Créer** | C4 |
| `src/ui/pages/teammates_charts.py` | Modifier — `temporal_labels=` | C5 |
| `src/ui/pages/timeseries.py` | Modifier — `temporal_labels=` | C5 |
| `src/ui/i18n/pages/common.py` | Modifier — formats de date | C6 |

---

### Points de vigilance

**Timezones** : `start_time` peut contenir des datetimes avec timezone (UTC). `TemporalSpan.from_series` doit normaliser avant de calculer la durée. Les tests couvrent ce cas (Phase C2).

**Performance** : `build_axis_tick_labels` est appelé côté rendu Streamlit. Il doit rester O(n) sur la taille du DataFrame. Pas de groupby ou join interne — uniquement des `map_elements` sur la colonne.

**DataFrames vides** : si le filtre actif produit 0 matchs, `TemporalSpan.from_series` lève une `ValueError`. L'appelant (page Streamlit) doit gérer ce cas avant d'appeler la fonction — c'est déjà le cas pour les graphes existants (guard `if df.is_empty(): return`).

**Rétrocompatibilité** : le paramètre `temporal_labels` est optionnel sur toutes les fonctions viz. Le comportement actuel est conservé si non fourni. La migration peut se faire graphe par graphe sans risque de régression.

---

## Checklist pré-PR (Partie C)

- [ ] `src/analysis/temporal_format.py` créé, aucune dépendance Streamlit/Plotly
- [ ] `tests/test_temporal_format.py` : couverture 100 % des branches
- [ ] `src/ui/chart_utils_temporal.py` et `table_utils_temporal.py` créés
- [ ] `plot_trio_metric` intégré (priorité 1)
- [ ] `plot_performance_timeseries` + `plot_timeseries` intégrés (priorité 2–3)
- [ ] `render_friends_history_table` : colonne date formatée (priorité 4)
- [ ] Formats i18n ajoutés dans le module commun
- [ ] Aucune régression : tous les graphes existants produisent un résultat au moins aussi lisible qu'avant
- [ ] `python -m pytest tests/test_temporal_format.py -v` → 0 failure

---

---

## Partie D — Audit séparation couches : `src/ui/pages/` vs `src/analysis/` vs `src/data/services/`

> Audit réalisé le 2026-04-09. Objectif : identifier toutes les fonctions `_compute_*` et `_load_*` qui vivent dans des fichiers de rendu UI alors qu'elles sont pures ou quasi-pures.
> Branche : `refactor/analysis-layer-separation` (sprint dédié, indépendant des Parties A–C).

---

### Règle fondamentale (pattern récurrent)

**Si une fonction commence par `_compute_` ou `_load_`, elle n'a pas sa place dans `src/ui/pages/`.**

- `_compute_*` → logique de calcul pure → `src/analysis/`
- `_load_*` (requêtes DuckDB) → accès données → `src/data/services/`
- Ce qui reste dans `src/ui/pages/` : uniquement du rendu (`st.*`, `go.Figure`, HTML)

Ce n'est pas seulement une convention de nommage — c'est ce qui rend le code testable sans instancier Streamlit ni Plotly.

---

### Candidats 🔴 Haute priorité

#### D1 — Binning timeline premier frag/mort (`teammates_charts.py`)

| Élément | Localisation actuelle | Cible |
|---------|----------------------|-------|
| `_format_bin_label()` | `teammates_charts.py:305` | `src/analysis/timeline_bins.py` |
| `_compute_bin_counts()` | `teammates_charts.py:313` | `src/analysis/timeline_bins.py` |
| `_bin_key()` (tri) | `teammates_charts.py:367` | `src/analysis/timeline_bins.py` |
| `_BIN_SIZE_S = 15` | `teammates_charts.py` local | Constante dans `timeline_bins.py` |

Ces fonctions sont **purement calculatoires** (binning de timestamps en tranches de 15s, comptage par joueur, tri). Elles sont actuellement enfouies dans `_build_first_events_fig()` (161 lignes) qui construit aussi la figure Plotly. Extraire la couche calcul permet de tester les bins sans instancier un graphe.

**Test cible** :
```python
# tests/test_timeline_bins.py
def test_bin_counts_two_players():
    # df avec kills à 5s, 12s, 45s pour deux joueurs
    # → dict attendu avec les bons bins

def test_format_bin_label_minutes():
    # 75s → "1m15s"

def test_bin_key_sort_order():
    # labels triés chronologiquement
```

---

#### D2 — Duplication logique grenade/mêlée (`teammates_weapons.py`)

La même logique de calcul du `remainder` (kills API - kills film = kills nets grenade/mêlée) est écrite **deux fois** :
- `_append_grenade_melee()` lignes 37–118 (appelée en mode single-player)
- `load_weapon_kills_data()` lignes 187–236 (appelée en mode multi-player)

Les lignes clés dupliquées :
```python
remainder = max(0, api_total - film_kills)
melee_net = min(melee, remainder)
grenade_net = min(grenade, max(0, remainder - melee_net))
```

**Cible** : `src/analysis/weapon_accounting.py`

```python
def compute_net_grenade_melee(
    grenade: int,
    melee: int,
    film_kills: int,
    api_total: int,
) -> tuple[int, int]:
    """
    Retourne (grenade_net, melee_net) en limitant au remainder
    pour éviter le double-comptage des kills filmés.
    """
    remainder = max(0, api_total - film_kills)
    melee_net = min(melee, remainder)
    grenade_net = min(grenade, max(0, remainder - melee_net))
    return grenade_net, melee_net
```

**Tests cibles** :
```python
def test_no_remainder_returns_zeros():
    # film_kills >= api_total → (0, 0)

def test_melee_capped_at_remainder():
    # melee > remainder → melee_net = remainder, grenade_net = 0

def test_both_fit_in_remainder():
    # grenade + melee < remainder → valeurs brutes retournées

def test_priority_melee_over_grenade():
    # melee consomme en priorité le remainder
```

---

#### D3 — Requêtes DuckDB pures dans `teammates_impact.py`

Trois fonctions sans aucun code UI vivent dans un fichier de rendu :

| Fonction | Lignes | Rôle | Cible |
|----------|--------|------|-------|
| `_load_match_participants()` | 31–62 | SELECT depuis `match_participants` | `src/data/services/teammates_impact_data.py` |
| `_load_highlight_events()` | 64–109 | SELECT depuis `highlight_events` | `src/data/services/teammates_impact_data.py` |
| `_load_match_outcomes()` | 111–142 | SELECT depuis `match_registry` | `src/data/services/teammates_impact_data.py` |

Ces fonctions acceptent une `conn` DuckDB et retournent un `pl.DataFrame` — elles sont déjà parfaitement isolées, il suffit de les déplacer.

Même chose pour les helpers de calcul dans `_render_impact_ranking_html()` :

| Fonction | Lignes | Rôle | Cible |
|----------|--------|------|-------|
| `_player_agg_counts()` | 190–197 | Agrégation events par joueur | `src/analysis/impact_matrix.py` |
| `_impact_extremes_from_agg()` | 200–210 | Min/max par colonne | `src/analysis/impact_matrix.py` |
| `_pivot_matrix_cells()` | 166–187 | Pivot DataFrame → structure HTML | `src/analysis/impact_matrix.py` |

---

#### D4 — `_compute_player_profile()` dans `teammates_synergy.py:81`

97 lignes de calcul pur (extraction de stats, scaling des seuils radar selon mode et nombre de matchs, construction du profil) sans aucun `st.*` ni `go.`. Elle prend un `DuckDBRepository` + un `pl.DataFrame` et retourne un profil.

**Cible** : déplacer dans `src/analysis/participation_profile.py` qui existe déjà et contient `compute_participation_profile()`. Ces deux fonctions sont cohérentes thématiquement.

**Note** : `_compute_player_profile` a un `# noqa: PLR0913` (trop d'arguments). La migration est l'occasion de la refactoriser avec un `dataclass` de paramètres.

---

#### D5 — `_load_first_events_data()` dans `teammates_charts.py:266`

36 lignes de requête DuckDB + transformation (mapping xuid → name) sans aucun code UI.

**Cible** : `src/data/services/teammates_charts_data.py` (ou fusionner avec `D3` dans un service `teammates_data.py` si le périmètre le justifie).

---

### Candidats 🟡 Priorité moyenne

#### D6 — `_reorder_profile_by_date()` dans `_timeseries_intensity.py:20`

21 lignes de tri pur par date sur un objet profil. Testable sans Streamlit.

**Cible** : `src/analysis/match_intensity.py` (module déjà existant pour les profils d'intensité).

#### D7 — Tri chronologique des cartes dupliqué dans `teammates_map_charts.py`

Le tri `sort(by="start_time")` + extraction des cartes ordonnées apparaît à deux endroits (lignes ~62–69 et ~143–150). Un helper `order_maps_by_first_played(df)` dans `src/analysis/map_stats.py` centralise ça.

---

### Récapitulatif des modules à créer/compléter

| Module | Action | Fonctions à y déplacer |
|--------|--------|----------------------|
| `src/analysis/timeline_bins.py` | **Créer** | `_format_bin_label`, `_compute_bin_counts`, `_bin_key`, `_BIN_SIZE_S` |
| `src/analysis/weapon_accounting.py` | **Créer** | `compute_net_grenade_melee()` (remplace la duplication D2) |
| `src/analysis/impact_matrix.py` | **Créer** | `_player_agg_counts`, `_impact_extremes_from_agg`, `_pivot_matrix_cells` |
| `src/analysis/participation_profile.py` | **Compléter** | `_compute_player_profile` depuis `teammates_synergy.py` |
| `src/analysis/match_intensity.py` | **Compléter** | `_reorder_profile_by_date` depuis `_timeseries_intensity.py` |
| `src/analysis/map_stats.py` | **Compléter** | helper `order_maps_by_first_played` |
| `src/data/services/teammates_impact_data.py` | **Créer** | `_load_match_participants`, `_load_highlight_events`, `_load_match_outcomes` |
| `src/data/services/teammates_charts_data.py` | **Créer** | `_load_first_events_data` |

---

### Ordre d'exécution recommandé

```
D2 (weapon_accounting)     — rapide, élimine une duplication active
D1 (timeline_bins)         — débloque des tests sur une logique complexe
D3 requêtes DuckDB         — déplacement mécanique, zéro risque
D5 (_load_first_events)    — idem, déplacement mécanique
D3 helpers calcul          — après les requêtes pour garder le contexte
D4 (_compute_player_profile) — plus de travail (refacto args + noqa)
D6, D7                     — en dernier, gains modestes
```

---

## Checklist pré-PR (Partie D)

- [ ] D1 : `src/analysis/timeline_bins.py` créé + `tests/test_timeline_bins.py`
- [ ] D2 : `src/analysis/weapon_accounting.py` créé + `tests/test_weapon_accounting.py` + duplication supprimée dans `teammates_weapons.py`
- [ ] D3a : requêtes DuckDB déplacées dans `src/data/services/teammates_impact_data.py`
- [ ] D3b : helpers calcul déplacés dans `src/analysis/impact_matrix.py` + tests
- [ ] D4 : `_compute_player_profile` déplacée dans `src/analysis/participation_profile.py`, `# noqa: PLR0913` résolu
- [ ] D5 : `_load_first_events_data` déplacée dans `src/data/services/teammates_charts_data.py`
- [ ] D6 : `_reorder_profile_by_date` déplacée dans `src/analysis/match_intensity.py`
- [ ] D7 : helper `order_maps_by_first_played` centralisé dans `src/analysis/map_stats.py`
- [ ] Aucun `_compute_*` ni `_load_*` restant dans `src/ui/pages/`
- [ ] `python -m pytest -q --ignore=tests/integration` → 0 failure
