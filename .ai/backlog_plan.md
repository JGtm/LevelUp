# Plan de traitement — Backlog consolidé — 2026-03-04

> **Mise à jour 2026-03-04** — Items A→L + J2/J3/J4/J5/M terminés sur `refactor/cleanup-all`.
> Tests : 3572 passed, 0 failed, 6 warnings. Reste : N, O, P, Q, R.

Synthèse de tous les soucis remontés (notes + logic_issues.md).
Chaque item est classé par état, priorité et effort.

---

## État de divergence des branches

**Ancêtre commun** : `bd59623` (fix(match-impact))

| Branche | Commits exclusifs |
|---|---|
| `main` | 8 commits (fixes bugs concrets) |
| `refactor/cleanup-all` | 11 commits (refactors architecture) |

Les deux branches ont divergé **sans conflit intentionnel** — main a reçu des fixes urgents
pendant que le refactor était en cours. **Action nécessaire : cherry-pick des 8 commits de main.**

---

## 🔴 Priorité immédiate

### A. Porter les 8 commits de `main` → `refactor/cleanup-all`

Ces commits résolvent des bugs déjà confirmés mais absents de la branche de travail.
**Ne pas cherry-pick en bloc aveuglément** : le refactoring majeur a remanié les fichiers
touchés par plusieurs de ces commits. Stratégie différenciée selon le risque.

#### ✅ Cherry-pick direct (logique autonome, fichiers non remaniés)

| Commit | Description | Résout |
|---|---|---|
| `6e23909` | fix(startup): charger .env.local avant vérification DISCORD_WEBHOOK_URL | Fausse alerte Discord webhook |
| `67cd282` | fix(startup): skip check Discord si Doppler actif | Fausse alerte Discord (Doppler) |
| `38d5215` | feat(sync): ajouter --with-citations (flag CLI) | Citations post-sync CLI |

**Action** : `git cherry-pick 6e23909 67cd282 38d5215`  
Risque de conflit faible — ces commits touchent `src/utils/startup_check.py` et
`scripts/sync.py`, tous deux peu remaniés par le refactor.

#### ⚠️ Cherry-pick à risque — vérifier le diff avant d'appliquer

| Commit | Risque | Stratégie |
|---|---|---|
| `c2cf9f5` + `69096f4` guard Tailscale | `tailscale.py` probablement intact, mais le **point d'appel** dans `streamlit_app.py` a été refactorisé (split en modules) → le guard peut se retrouver au mauvais endroit | Lire le diff, réimplémenter manuellement si l'appel a bougé |

**Action** : `git show c2cf9f5 -- src/utils/tailscale.py` puis comparer avec l'état actuel.

#### ❌ Ne pas cherry-pick — réimplémenter manuellement

| Commit | Pourquoi | Stratégie |
|---|---|---|
| `f156bb2` feat(citations) post-sync | Modifie le **bloc post-sync** de `streamlit_app.py` — **exactement la même zone que le Bug #0** (B). Conflit quasi-certain, et la logique doit s'intégrer dans la nouvelle architecture découpée | Réimplémenter dans `src/app/` après avoir appliqué le fix B |
| `36fc8ce` fix(citations) onglet match | Modifie `src/ui/pages/match_view.py` — potentiellement refactorisé, et dépend de `f156bb2` | Réimplémenter après `f156bb2` |
| `e1a5d39` revert was_master_before | Revert d'un filtre dans `match_view.py` — si la fonction a été découpée, le revert ne s'applique plus au bon endroit | Localiser le filtre actuel et appliquer le revert manuellement |

**Procédure pour chaque réimplémentation** :
1. `git show <commit>` pour voir exactement ce qui a été changé sur `main`
2. Localiser l'équivalent dans l'architecture refactorisée
3. Appliquer la même logique, adapter aux nouveaux modules

---

### B. Bug #0 — Match invisible après sync (filtres non réinitialisés)

**Statut** : ❌ Non corrigé sur aucune branche

**Root cause profonde** : `_filters_loaded_*` est une clé **write-once, never-expire** dans
`session_state`. La couche de filtres n'a aucun mécanisme de réactivité aux données —
elle fait confiance à ce flag pour toute la durée de vie de la session navigateur.
Suppression post-sync = patch superficiel : ça résout le cas sync in-app mais pas
les modifications de DB par scripts CLI, backfill, etc.

**Vraie correction (root cause)** : les filtres doivent s'auto-invalider quand la DB change,
indépendamment de la source de la modification. La clé `db_key` (mtime+size) est
déjà calculée à chaque rerun dans `main_helpers.py` — s'en servir comme signal :

```python
# Au lieu de _filters_loaded_{player_key} (booléen write-once)
# Stocker _filters_db_key_{player_key} = db_key au moment de l'init
# Comparer à chaque rerun : si db_key a changé → réinitialiser les filtres
_filters_db_key_stored = st.session_state.get(f"_filters_db_key_{player_key}")
if _filters_db_key_stored != current_db_key:
    # DB a changé depuis la dernière init → re-run _apply_default_last_session()
    st.session_state[f"_filters_db_key_{player_key}"] = current_db_key
    _apply_default_last_session(db_path, xuid, db_key, aliases_key)
```

Ce mécanisme:
- ✅ Couvre le sync in-app
- ✅ Couvre les modifications CLI/backfill
- ✅ Couvre le changement de profil A→B→A
- ✅ Couvre les nouvelles sessions post-sync
- ✅ Ne réinitialise **pas** les filtres si la DB n'a pas changé (pas de régression UX)

**Fichiers** : `src/app/filters_render.py` (guard L241) + `src/app/main_helpers.py`
(propager `db_key` jusqu'aux filtres)

---

## 🟠 Priorité haute (sprint en cours)

### C. Sync concurrent — SyncLock non câblé à l'UI

**Statut** : `SyncLock` existe dans `src/utils/sync_lock.py` mais **n'est pas utilisé**
dans `src/ui/sync.py` ni `streamlit_app.py`.

**Root cause profonde** : le sync UI et le sync CLI partagent la même ressource
(`shared_matches.duckdb`) sans aucune coordination inter-processus. Le `begin_sync_mode()`
existant protège les connexions *intra-processus* (threads Streamlit), mais pas contre
un second processus Python (CLI) qui ouvrirait la DB en R/W simultanément.

**Vraie correction (root cause)** : deux niveaux de protection nécessaires :
1. **Intra-processus** (déjà partiellement en place via `begin_sync_mode`) — à compléter
   avec `SyncLock` dans `sync_spnkr_db_via_engine` pour serialiser les clics UI concurrents
2. **Inter-processus** (manquant) — `SyncLock` s'appuie sur `filelock` (fichier `data/.sync.lock`)
   → protège aussi contre `scripts/sync.py` et `scripts/backfill_data.py` lancés en CLI

```python
# src/ui/sync.py — sync_spnkr_db_via_engine()
from src.utils.sync_lock import SyncLock, SyncAlreadyRunning
try:
    with SyncLock(timeout=0):  # fail-fast
        result = await engine.sync_delta(options)
except SyncAlreadyRunning:
    return False, "Un sync est déjà en cours (CLI ou autre onglet). Réessaie dans quelques instants."
```

`SyncLock` doit également être câblé dans `scripts/sync.py` et `scripts/backfill_data.py`
(vérifier s'il l'est déjà — il est référencé dans `sync_lock.py` docs mais pas confirmé
dans les scripts).

**Fichiers** : `src/ui/sync.py`, `scripts/sync.py`, `scripts/backfill_data.py`

---

### D. Spam Tailscale dans Discord et terminal — root cause confirmée

**Statut** : Corrigé sur `main` (`c2cf9f5` + `69096f4`) — **absent de `refactor/cleanup-all`**

**Deux problèmes distincts, une seule cause racine** :

Le guard actuel sur `refactor/cleanup-all` ([`streamlit_app.py#L339`](streamlit_app.py)) :

```python
if not st.session_state.get("_tailscale_started"):  # ← PAR SESSION navigateur ❌
    st.session_state[SK.TAILSCALE_STARTED] = True
    threading.Thread(target=_tailscale_worker, ...).start()
```

`session_state` est **par onglet navigateur**, pas par processus Python. Conséquence :
- Même onglet, rerun → OK, guard tient ✅
- Nouvel onglet / second utilisateur → `session_state` vide → thread relancé →
  `start_funnel()` rappelé → nouvelle notif Discord "🟢 LevelUp est disponible" ❌

Les commits `c2cf9f5` + `69096f4` déplacent le guard dans `src/utils/tailscale.py`
comme variable **module-level** (process-level), partagée entre toutes les sessions.

**Ce qui n'est PAS un bug** : le log `[Tailscale] OK (code 0)` en `INFO` — une fois le
guard process-level en place, `start_funnel()` n'est appelé qu'une seule fois par
démarrage du processus Python. Ce log est légitime, pas besoin de le passer en DEBUG.

**La notification Discord** `notify_app_started()` est un one-shot déclenché uniquement
depuis le thread Tailscale — pas de logique "changement d'état" nécessaire,
pas de boucle indépendante. Elle sera correcte une fois le guard corrigé.

**Correction** : intégrée dans A2 (porter le guard process-level depuis `main`).
Aucune autre action requise sur ce point.

---

### E. Mise à jour du baseline de taille de code

**Statut** : 8 nouvelles violations détectées (dont 7 pré-existantes déclenchées marginalement)  
**Violations** :

```
FUNC: src/app/_filters_session.py:134:_render_session_filter:262L
FUNC: src/app/filters_render.py:411:apply_filters:290L
FUNC: src/app/page_router.py:263:dispatch_page:148L
FUNC: src/ui/pages/citations.py:19:render_citations_page:169L
FUNC: src/ui/pages/match_view_participation.py:102:render_participation_comparison:81L
FUNC: src/ui/pages/objective_analysis.py:50:render_objective_analysis_page:356L
FUNC: src/ui/pages/session_compare.py:320:render_session_comparison_page:116L
MODULE: src/app/filters_render.py:702L
```

**Stratégie** :
- `render_participation_comparison` (81L) → split immédiat (1L de dépassement)
- Les autres (262L, 290L, 356L…) → dette structurelle, ajouter au baseline via
  `python scripts/check_code_size.py --update` puis planifier les splits séparément.
- `filters_render.py` (702L) → découpage en `filters_render.py` + `filters_logic.py` (sprint dédié)

---

## 🟡 Priorité normale (sprint suivant)

### F. Fausse alerte Discord webhook

**Statut** : ✅ Corrigé sur `main` (`6e23909`, `67cd282`) — **absent de `refactor/cleanup-all`**  
Résolu par le cherry-pick du point A.

### G. Fenêtre planificateur de tâches clignotante

**Statut** : ✅ Corrigé sur les deux branches (`7dadfe9` : utilisation de `pythonw.exe`)  
**Aucune action requise.**

### H. Bug #1 — `win_rate` incohérent (analytics vs trends)

**Statut** : ❌ Non corrigé

**Root cause profonde** : la formule `win_rate` est dupliquée en dur dans 7+ endroits sans
source de vérité unique. Uniformiser les 7 occurrences = patch qui se re-divergera au
prochain développeur qui ajoute une requête.

**Vraie correction (root cause)** : extraire la formule en une **constante SQL partagée**
dans un module dédié (`src/data/query/_sql_fragments.py` ou similaire) :

```python
# src/data/query/_sql_fragments.py
WIN_RATE_EXPR = """
    SUM(CASE WHEN outcome = {win} THEN 1 ELSE 0 END) * 1.0
    / NULLIF(SUM(CASE WHEN outcome IN ({win}, {loss}) THEN 1 ELSE 0 END), 0)
""".format(win=Outcome.WIN.value, loss=Outcome.LOSS.value)
```

Toutes les requêtes SQL importent et utilisent `WIN_RATE_EXPR` — une seule ligne à
corriger si la définition change.

**Fichiers** : `src/data/query/analytics.py` (L163, 193, 460, 487),
`src/data/query/trends.py` (L244, 275, 308)

### I. Bug #5 — NaN-check fragile dans `match_view.py`

**Statut** : ❌ Non corrigé

**Root cause profonde** : des données non typées et non validées traversent toute la
pipeline (DB → repository → DataFrame → UI) sans contrat de type. L'idiome `x == x`
est un symptôme : le développeur ne sait pas si la valeur est `None`, `float(NaN)`,
`int` ou `str` à ce stade.

**Vraie correction (root cause)** : la validation doit avoir lieu **à la frontière de
chargement**, pas à l'affichage. Le repository ou le modèle de données doit garantir
que `outcome` est toujours un `int | None` avant d'arriver dans l'UI.

1. Dans `DuckDBRepository.load_matches()` ou le transformer : caster `outcome` en `Int32`
   (Polars nullable) — les `NaN` flottants deviennent `null` proprement
2. Dans `match_view.py` L501 : `outcome_code = int(row["outcome"]) if row["outcome"] is not None else None`

Le patch L501 est nécessaire maintenant, mais le vrai fix est le cast upstream.

**Fichiers** : `src/ui/pages/match_view.py` (L501),
`src/data/repositories/duckdb_repo.py` (cast outcome en Int32)

---

## 🔵 Backlog (pas de sprint immédiat)

### J. Dettes techniques logic_issues #2, #3, #4, #6

**Root cause commune** : migration incomplète — des artefacts de la transition v4→v5
subsistent dans le code source sans justification active.

- **#2** Guard `_PERF_SCORE_AVAILABLE` : anti-pattern "compatibility guard forever" —
  Polars est obligatoire, le guard ne sera jamais `False`. Supprimer le `try/except`,
  importer directement. (`src/data/sync/_performance.py` L17–32)
- **#3** `_ensure_performance_score_column()` vide : anti-pattern "dead code museum" —
  supprimer méthode + vérifier absence d'appels (`src/data/sync/_performance.py` L36)
- **#4** `outcome == 4` : magic number — utiliser `Outcome.DID_NOT_FINISH` (enum existe
  mais non importé ici). Root cause = même problème qu'en I : pas de contrat de type
  uniforme sur `outcome`. (`src/data/sync/transformers/_match.py` L187)
- **#6** Magic numbers SQL `2`, `3` : résolu structurellement par le module
  `_sql_fragments.py` créé pour le fix H — les constantes `_WIN`/`_LOSS` y seront
  définies et réutilisées ici aussi.

### J2. Cleanup kwargs legacy SyncScope

**Root cause** : migration progressive vers `SyncScope` non terminée — les anciens kwargs
restent en place "le temps que tous les appelants migrent" sans deadline ni condition
de suppression claire.

**Fichiers** : `scripts/backfill/detection.py` (L46),
`scripts/backfill/orchestrator.py` (L104, L435, L774)

**Condition de suppression** : vérifier que tous les appelants (scripts, tests, UI)
passent `scope=SyncScope(...)` via `grep -r "backfill_player_data\|backfill_all_players"`,
puis supprimer les kwargs marqués `LEGACY` en une passe.

### J3. Migration `career.py` vers DuckDBRepository

**Root cause** : `src/ui/pages/career.py` (L27, L69) ouvre `duckdb.connect()` directement —
bypass du repository pattern, du cache de connexion et du context manager standard.
SQL correctement paramétré (pas de risque injection) mais viole l'architecture v5.

**Action** : remplacer par `get_cached_repository_st()` + méthode dédiée dans
`DuckDBRepository` si elle n'existe pas.

### J5. Déduplication citations + cohérence post-sync CLI

**Root cause** : deux implémentations du même algorithme de calcul de citations, créées
à des moments différents sans consolidation :

| Implémentation | Appelée par |
|---|---|
| `src/data/citations_backfill.py::backfill_citations_for_player()` | `engine.py` (post-sync UI + CLI via `DuckDBSyncEngine`) |
| `scripts/backfill/strategies.py::backfill_citations()` | `orchestrator.py` → `backfill_data.py --citations` |

Depuis A3, l'engine déclenche automatiquement les citations post-sync pour **les deux
chemins** (UI Streamlit et CLI `sync.py --delta` → `_try_sync_duckdb` → `DuckDBSyncEngine`).
Le flag `--with-citations` de `sync.py` est donc **redondant** : déclenche un second
calcul qui trouvera 0 match à traiter (incrémental), mais entretient la confusion.

**Sessions** : pas de duplication. `orchestrator.py` délègue déjà à
`sessions_backfill.py::backfill_sessions_for_player()`. La symétrie CLI/UI est assurée
via `DuckDBSyncEngine` (engine.py L688) ✅

**Corrections (root cause)** :

1. **Faire déléguer `strategies.py::backfill_citations()` vers la source canonique** :
   ```python
   # scripts/backfill/strategies.py::backfill_citations()
   from src.data.citations_backfill import backfill_citations_for_player
   result = backfill_citations_for_player(db_path, xuid, conn=conn)
   return result.get("citations_computed", 0)
   ```
   La logique SQL reste dans `src/data/citations_backfill.py` (seul endroit).

2. **Déprécier `--with-citations` dans `sync.py`** : ajouter un avertissement
   `logger.warning("--with-citations est redondant : les citations sont calculées automatiquement post-sync.")` et conserver le flag 1 release pour rétrocompatibilité, puis le supprimer.

3. **Vérifier `backfill_data.py --citations`** : il reste utile pour le rattrapage
   en bulk (force=True, tous les matchs existants) — ne pas supprimer l'option.

**Fichiers** : `scripts/backfill/strategies.py` (L1287–1433), `scripts/sync.py` (L1129, L1492)

### J4. TODO mineurs conservés volontairement

- `src/analysis/citations/custom_rules.py` (L103) — amélioration future dépendant
  de données API non disponibles. Conserver jusqu'à disponibilité.
- `scripts/migration/migrate_metadata_to_duckdb.py` (L72) — `# TODO: ajouter traductions FR`
  (impact faible, hors chemin chaud)

### K. i18n — corrections données (i18n-1, i18n-3)

- i18n-1 : restaurer 2 clés tronquées dans `PAIR_FR` (`src/ui/translations.py`)
- i18n-3 : supprimer doublon `tm_session_trend` dans `src/ui/i18n/widgets.py`

### L. i18n — nettoyage (i18n-2)

- Supprimer 342 entrées redondantes dans `PAIR_FR` (399 → 57 entrées)

### M. Documentation

- Mettre à jour `docs/ARCHITECTURE_V5.md` suite au refactoring majeur
- Mettre à jour `README.md` et `docs/README_FR.md`
- Mettre à jour `CHANGELOG.md`

### N. i18n — migrations JSON (i18n-5, i18n-6)

**Root cause** : `translations.py` = couche legacy non migrée. Les fonctions
`translate_pair_name()` et `translate_playlist_name()` maintiennent deux couches en
parallèle (dict Python + JSON), ce qui crée une asymétrie FR/EN et complique la
maintenance.

- Migrer les 57 entrées utiles de `PAIR_FR` vers `static/i18n/modes_fr.json`
- Archiver `translations.py` en couche legacy explicite avec deprecation warning
- Objectif final : une seule source de vérité JSON pour toutes les traductions

### O. i18n — câblage `t()` dans l'UI Streamlit

**Root cause** : les traductions EN ont été remplies (Phase 1b ✅) mais jamais câblées —
le registre i18n existe mais l'UI utilise encore des strings hardcodées FR. Dette de
finition de feature.

- Câbler `t()` dans les pages/widgets Streamlit
- Modifier `src/ui/translations.py` pour déléguer au registre i18n
- Supprimer les commentaires `⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO"`
  dans `src/ui/i18n/common.py`, `pages.py`, `viz.py`, `widgets.py`, `cli.py`

### P. Performance UI (optimisations profondes)

**Root cause commune** : l'UI recalcule et recharge les données à chaque rerun Streamlit
sans distinction entre ce qui a changé et ce qui est stable. Les vues matérialisées,
le lazy-loading et les projections fines sont des remèdes à ce problème structurel.

| # | Problème | Gain estimé | Root cause | Approche |
|---|---|---|---|---|
| P1 | Vues matérialisées reconstruites à chaque rafraîchissement | −70% | Calcul dans l'UI au lieu du moteur de sync | Reconstruction dans `engine.py` post-sync uniquement |
| P2 | `match_view` charge toutes sections même non consultées | −40% premier rendu | Pas de lazy-loading | `st.tabs` + `@fragment_if_available` + state par onglet |
| P3 | 2000+ matchs chargés entièrement avant filtrage Python | −50% RAM | Filtres appliqués en mémoire au lieu d'en SQL | Pousser filtres en DuckDB avec `LIMIT/OFFSET` |
| P4 | `performance_score` recalculé à l'affichage | — | Call sites non audités | Garantir lecture depuis `player_match_enrichment` |
| P5 | 30+ colonnes chargées pour pages n'en utilisant que 5-8 | −30% mémoire | Projection commune trop large | Projections fines par page dans `cache_loaders.py` |

### Q. CI/CD & Outillage

**Root cause** : la détection de régression existe (`scripts/demo_regression_detection.py`)
mais n'est pas intégrée au cycle de développement — elle ne protège que si on pense
à la lancer manuellement.

- Ajouter la détection de régression dans `.github/workflows/`
- Créer un pre-commit hook
- Benchmark avant/après optimisations P (`scripts/benchmark_pages.py`)

### R. Améliorations futures

- **Audit Pandas → Polars** : usages résiduels à la frontière Streamlit/Plotly
  (`.to_pandas()` toléré à la frontière, mais remonter du Pandas dans les modules
  métier = violation de règle)
- **`.ai/START_HERE.md`** : référence des phases 5-10 d'une migration v5 antérieures
  à v5.1 — vérifier pertinence ou archiver dans `.ai/archive/`

---

## Tableau récapitulatif

| Item | Description | Statut | Effort | Priorité |
|---|---|---|---|---|
| A1 | Cherry-pick direct (3 commits safe) | ✅ Fait | Très faible | 🔥 Immédiat |
| A2 | Tailscale guard process-level (`threading.Event`) | ✅ Fait | Très faible | 🔥 Immédiat |
| A3 | `citations_backfill.py` + câblage post-sync engine | ✅ Fait | Faible | 🔥 Immédiat |
| B | Bug #0 — filtres auto-invalidation via `_filters_db_key` | ✅ Fait | Très faible | 🔥 Immédiat |
| C | SyncLock câblé à l'UI (`SyncAlreadyRunning` + WAL flush) | ✅ Fait | Faible | 🟠 Haute |
| D | Spam Tailscale/Discord (guard session→process) | ✅ Inclus dans A2 | — | — |
| E | Baseline taille de code (247 violations connues) | ✅ Fait | Très faible | 🟠 Haute |
| F | Fausse alerte Discord | ✅ Cherry-pické (A1) | — | — |
| G | Fenêtre clignotante planificateur | ✅ Les deux branches | — | — |
| H | win_rate incohérent → `_sql_fragments.py` + `WIN_RATE_EXPR` | ✅ Fait | Faible | 🟡 Sprint suivant |
| I | NaN-check fragile `match_view.py` (`x==x` → `is not None`) | ✅ Fait | Très faible | 🟡 Sprint suivant |
| J | Dettes #2 #3 #4 #6 : `_PERF_SCORE_AVAILABLE` supprimé, dead method, `Outcome.DID_NOT_FINISH` | ✅ Fait | Faible | 🔵 Backlog |
| K | i18n clés tronquées + doublon `tm_session_trend` | ✅ Fait | Très faible | 🔵 Backlog |
| L | PAIR_FR nettoyage 399 → 56 entrées | ✅ Fait | Faible | 🔵 Backlog |
| J2 | Cleanup kwargs `LEGACY` SyncScope (`orchestrator.py`, `detection.py`) | ✅ Fait | Moyen | 🔵 Backlog |
| J3 | `career.py` → remplacer `duckdb.connect()` direct par `DuckDBRepository` | ✅ Déjà OK | Faible | 🔵 Backlog |
| J5 | Déduplication citations (`strategies.py` → délègue à `citations_backfill.py`) + dépréc. `--with-citations` | ✅ Fait | Faible | 🔵 Backlog |
| J4 | TODOs mineurs (`custom_rules.py`, `migrate_metadata_to_duckdb.py`) | ✅ Fait | Très faible | 🔵 Backlog |
| M | Docs (`ARCHITECTURE_V5.md`, `README`, `CHANGELOG`) | ✅ Fait (CHANGELOG) | Moyen | 🔵 Backlog |
| N | Migrer `PAIR_FR` → `static/i18n/modes_fr.json` (source unique) | ❌ À faire | Moyen | 🔵 Backlog |
| O | Câbler `t()` dans l'UI Streamlit (traductions EN existantes) | ❌ À faire | Moyen | 🔵 Backlog |
| P | Performance UI P1→P5 (MVs, lazy-loading, filtres SQL) | ❌ À faire | Élevé | 🔵 Backlog |
| Q | CI/CD détection régression (workflow GitHub) | ❌ À faire | Moyen | 🔵 Backlog |
| R | Améliorations futures (Pandas résiduel, `START_HERE.md`) | ❌ À faire | Variable | 🟢 Un jour |

---

## Ordre d'exécution recommandé

> ✅ = terminé sur `refactor/cleanup-all` (commit 2026-03-04)

```
✅ 1.  A1 — cherry-pick : 6e23909 67cd282 38d5215
✅ 2.  A2 — Tailscale guard process-level (threading.Event)
✅ 3.  B  — filtres auto-invalidation via _filters_db_key
✅ 4.  A3 — citations_backfill.py + engine.py post-sync
✅ 5.  E  — check_code_size --update (247 violations baseline)
✅ 6.  C  — SyncLock câblé à l'UI (WAL flush + SyncAlreadyRunning)
✅ 7.  H  — _sql_fragments.py + WIN_RATE_EXPR canonique
✅ 8.  I  — NaN-check fix match_view.py
✅ 9.  J  — Dettes #2 #3 #4 #6 supprimées
✅ 10. K  — i18n clés tronquées + doublon supprimé
✅ 11. L  — PAIR_FR 399 → 56 entrées

✅ 12. J5 — Déduplication citations (strategies.py délègue + dépréc. --with-citations)
✅ 13. J2 — Cleanup kwargs LEGACY SyncScope (backfill_player_data + backfill_all_players)
✅ 14. J3 — career.py → déjà OK (duckdb_read_only partout)
✅ 15. J4 — TODOs mineurs (custom_rules.py nettoyé, migrate_metadata conservé volontairement)
✅ 16. M  — CHANGELOG mis à jour

❌ 17. N  — PAIR_FR → static/i18n/modes_fr.json (source de vérité unique)
❌ 18. O  — Câbler t() dans l'UI Streamlit
❌ 19. P  — Performance UI P1→P5 (sprint dédié)
❌ 20. Q  — CI/CD détection régression (GitHub workflow)
❌ 21. R  — Améliorations futures (Pandas résiduel, START_HERE.md)
```
