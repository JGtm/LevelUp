# Plan — Assainissement archi & code (hors backlog UX)

> Analyse réalisée le 2026-04-11. Issues identifiées à froid, indépendamment du backlog fonctionnel.
> Ce document couvre uniquement le code et l'archi — pas les features.

---

## Statut d'avancement — 2026-04-11

| ID | Item | Statut | Commit |
|---|---|---|---|
| F0 | Nettoyage git .tmp.* + .bak | ✅ Fait | branche `v7/cockpit` |
| F1 | Déplacer _exp_spawn_*.py | ✅ Fait | `77646df3` |
| F2 | Configurer thumbnails-watcher.service | ✅ Fait | `bdbd2be9` |
| F3 | Archiver experimental/ → _archive/ | ✅ Fait | `77646df3` |
| F4 | Archiver scripts one-shot | ✅ Fait | `a7950a6d` |
| F5 | docs/FR/ mise à jour + règle CLAUDE.md | ✅ Fait | `bd0689bf` |
| F6 | README dans dossiers archive | ✅ Fait | `e252ae8c` |
| F7 | Règle docs/FR/ dans CLAUDE.md | ✅ Fait | fusionné en F5 |
| F8 | Refactoring launcher.py (split 6 modules) | ✅ Fait | branche `v7/cockpit` |
| H1a | Fix test yaxis.autorange | ✅ Fait | `66db6b53` |
| H1b | Fix test llp-squad-start | ✅ Fait | `66db6b53` |
| H2 | Legacy kwargs → DeprecationWarning + audit | ✅ Fait | `7c987dc7` |
| H3 | Split migrations.py par domaine DB | ✅ Fait | `78ae2f42` |
| H4 | launcher_i18n.py → JSON | ✅ Fait | `a9e8ab55` |
| H5 | Stratégie cache unifiée | 🔲 Post-v7 | — |
| H6 | Extraire logique API de profile_api.py | 🔲 Post-v7 | — |
| H7 | Enforcer src/ports/ comme frontière | 🔲 Post-v7 | — |
| H8 | Règle TypedDict/dataclass + migration _page_context.py | ✅ Fait | branche `v7/cockpit` |
| H9 | Split _weapon_kills_repo.py + _materialized_views.py | ✅ Fait | branche `v7/cockpit` |

---

## Recommandation de timing : Pre-v7 ET Post-v7

Deux fenêtres distinctes. La règle de découpe est simple :

| Critère | Fenêtre |
|---|---|
| Touche uniquement la couche data/sync/utils → zéro impact UI | **Pre-v7** |
| Réduction de dette qui bénéfice directement à la stabilité des tests | **Pre-v7** |
| Travaux que la refonte v7 va de toute façon remettre en question | **Post-v7** |
| Travaux qui nécessitent de connaître la forme finale du shell v7 | **Post-v7** |

---

## Bloc F — Nettoyage structure fichiers et dossiers

> Réponses aux questions du point structure :
> - **Point 3 (docs/FR/)** : Option A — maintenu rigoureusement (plan ci-dessous)
> - **Point 6 (docs/FR/ décalée)** : prise en charge via le processus docs ci-dessous
> - **Point 7 (experimental/)** : archiver → cf. F5
> - **Point 8 (thumbnails-watcher.service)** : username = `deploy` sur VPS → cf. F3
> - **Launcher** : refactor post-v7 → cf. F8

---

### F0 — Nettoyage git immédiat (1 commit)

**Priorité : immédiate. Un seul commit sur la branche courante.**

Trois `.tmp.*` créés lors d'un hot-reload Streamlit sont trackés par erreur dans git, et
`app_settings.json.bak` pollue l'index sans intérêt.

**Commandes :**

```bash
# Supprimer les .tmp.* faux-fichiers Streamlit
git rm "scripts/index_media.py.tmp.14124.1774609003784"
git rm "scripts/index_media.py.tmp.14124.1774609010690"
git rm "scripts/index_media.py.tmp.14124.1774609025891"

# Dé-tracker le .bak (garder le fichier local, juste retirer de l'index)
git rm --cached app_settings.json.bak

# Protéger contre une future récidive
echo "scripts/*.tmp.*" >> .gitignore
echo "*.json.bak" >> .gitignore

# Supprimer les tmp Python qui trainent dans data/
rm -f data/tmp_check.py data/tmp_fanout_check.py data/tmp_lusr_check.py

git add .gitignore
git commit -m "chore(cleanup): retirer .tmp.* et .bak de l'index git"
```

**Vérification post-commit :** `git ls-files | grep -E "\.tmp\.|\.bak"` doit renvoyer 0 ligne.

---

### F1 — Déplacer les scripts expérimentaux orphelins

**Priorité : pré-v7, faible effort.**

`scripts/_exp_spawn_download.py` et `scripts/_exp_spawn_scan.py` sont à la racine de
`scripts/` alors qu'ils appartiennent thématiquement à `scripts/experimental/`
(même série de recherche spawn-detection).

```bash
git mv scripts/_exp_spawn_download.py scripts/experimental/
git mv scripts/_exp_spawn_scan.py scripts/experimental/
git commit -m "chore(structure): déplacer _exp_spawn_*.py dans experimental/"
```

---

### F2 — Configurer et relocaliser `thumbnails-watcher.service`

**Priorité : pré-v7, trivial à exécuter une fois les placeholders connus.**

Le fichier `scripts/thumbnails-watcher.service` contient les placeholders `YOUR_USERNAME`
et `/path/to/levelup`. L'utilisateur sur le VPS est `deploy`. Le fichier appartient
logiquement à `packaging/` (aux côtés des configs nginx).

**Actions :**

- [ ] **Mettre à jour les placeholders** dans `scripts/thumbnails-watcher.service` :
  ```ini
  User=deploy
  WorkingDirectory=/home/deploy/levelup
  ExecStart=/home/deploy/levelup/.venv/bin/python -m streamlit run streamlit_app.py
  ```
  *(ajuster le WorkingDirectory selon le chemin réel sur le VPS — vérifier avec `pwd` en SSH)*

- [ ] **Déplacer vers `packaging/`** :
  ```bash
  git mv scripts/thumbnails-watcher.service packaging/thumbnails-watcher.service
  ```

- [ ] **Créer `packaging/INSTALL_SERVICES.md`** (si inexistant) avec les instructions
  d'installation systemd :
  ```bash
  sudo cp packaging/thumbnails-watcher.service /etc/systemd/system/
  sudo systemctl daemon-reload
  sudo systemctl enable thumbnails-watcher
  sudo systemctl start thumbnails-watcher
  ```

- [ ] Commit :
  ```bash
  git add packaging/
  git commit -m "chore(packaging): configurer thumbnails-watcher.service (user=deploy) + déplacer dans packaging/"
  ```

---

### F3 — Archiver `scripts/experimental/`

**Priorité : pré-v7, faible effort, fort gain de lisibilité.**

**Analyse détaillée :**

Les 26 fichiers trackés dans `scripts/experimental/` sont exclusivement des artefacts
de la recherche weapon-extraction (inv#1–136). Le statut est **fermé** :
- `FINDINGS_weapon_extraction_EN.md` : en-tête "CLOSED", ERRATUM datée du 2026-03-18
  (conclusions invalidées par la suite de l'investigation)
- `FORMULA_C_TECHNICAL_MODEL.md` : modèle de l'état intermédiaire — supplanté par
  l'implémentation dans `src/data/`
- Scripts Python (`inv106–136`, `compare_acurtis`, `weapon_extraction`, etc.) :
  écrits pour la v5 du schéma, non maintenus, incompatibles avec le schéma v6 actuel
  (`gamertag` supprimé de `highlight_events`, vue `v_weapon_kills` inexistante au
  moment de la rédaction)
- Dernier commit sur `experimental/` : 2026-03-25 (refactor de renommage DB, pas de
  recherche active)

**Les résultats utiles sont déjà extraits** : les findings ont alimenté les mémo-repo
dans `/memories/repo/film-weapons-investigation-*.md` (24 fichiers). La connaissance est
préservée.

**Recommandation :** déplacer dans `scripts/_archive/experimental/` (le dossier
`scripts/_archive/` existe déjà).

```bash
git mv scripts/experimental scripts/_archive/experimental
# Ajouter un README d'entrée dans _archive pour expliquer le contexte
git commit -m "chore(archive): déplacer experimental/ → _archive/experimental/ (recherche weapon-extraction fermée)"
```

**Note :** Le dossier `scripts/_archive/` doit avoir un `README.md` expliquant qu'il
contient des scripts historiques non maintenus (voir F6).

---

### F4 — Archiver les scripts obsolètes de setup one-shot

**Priorité : pré-v7, faible effort.**

Deux scripts à la racine de `scripts/` sont des reliquats de setup one-shot
désormais couverts par le système de migrations automatiques :

| Script | Statut | Raison |
|---|---|---|
| `scripts/create_match_citations_table.py` | Obsolète | La table `match_citations` est créée par migration automatique dans `migrations.py → ensure_match_citations_table()`. Ce script n'est référencé que dans un commentaire de diagnostic ("Exécuter si manquant..."). |
| `scripts/compute_sessions.py` | Obsolète | La logique est dans `src/analysis/sessions.py` (Polars) et `scripts/post_sync_compute.py → _precompute_sessions()`. Ce script standalone n'apporte plus rien. |

**Action :**

```bash
git mv scripts/create_match_citations_table.py scripts/_archive/
git mv scripts/compute_sessions.py scripts/_archive/
git commit -m "chore(archive): archiver scripts one-shot obsolètes (migrations couvrent create_citations, post_sync_compute couvre sessions)"
```

---

### F5 — Processus de maintenance de `docs/FR/`

**Priorité : pré-v7 pour établir le process, puis continu.**

**Constat actuel :** `docs/FR/CHANGELOG.md` est à v6.4.0 alors que la version courante
est 6.5.0. Le décalage s'est créé silencieusement.

**Règle à appliquer** : toute PR qui touche un fichier EN dans `docs/` doit inclure la
mise à jour du correspondant FR dans le même commit (ou un commit du même PR).

**Actions ponctuelles :**

- [ ] **Mettre à jour `docs/FR/CHANGELOG.md`** pour couvrir v6.4.1 à v6.5.0 :
  portage des sections correspondantes de `docs/CHANGELOG.md`

- [ ] **Vérifier les autres docs FR** par rapport à leurs équivalents EN :
  ```bash
  # Comparer les dates de dernier commit pour chaque paire
  for f in docs/FR/*.md; do
    base=$(basename "$f")
    echo "=== $base ==="
    echo "FR:"; git log --oneline -1 -- "$f"
    echo "EN:"; git log --oneline -1 -- "docs/$base" 2>/dev/null || echo "(pas de correspondant EN)"
  done
  ```

- [ ] **Ajouter dans `CLAUDE.md`** la règle :
  > **docs/FR/ — règle de synchronisation** : tout commit qui modifie un fichier dans
  > `docs/` doit inclure la mise à jour du fichier correspondant dans `docs/FR/` si ce
  > fichier existe. Les deux commits peuvent être séparés mais doivent être dans le même PR.

**Commit :**

```bash
git add docs/FR/CHANGELOG.md
git commit -m "docs(fr): mettre à jour CHANGELOG FR v6.4.1 → v6.5.0"
```

---

### F6 — Ajouter un README dans `docs/archive/` et `scripts/_archive/`

**Priorité : pré-v7, 5 minutes.**

Les deux dossiers d'archive n'ont pas de README explicatif. Un visiteur ou un agent IA
ne sait pas que ces dossiers sont historiques.

- [ ] Créer `docs/archive/README.md` :
  ```markdown
  # docs/archive/

  Ce dossier contient des documents historiques des versions précédentes de LevelUp.
  Ces fichiers ne sont plus à jour et sont conservés à titre de référence uniquement.
  Ne pas modifier. Ne pas utiliser comme documentation courante.
  ```

- [ ] Créer `scripts/_archive/README.md` :
  ```markdown
  # scripts/_archive/

  Scripts one-shot et scripts de recherche historiques. Non maintenus.
  - `experimental/` : artefacts de la recherche film weapon-extraction (inv#1–136, fermée 2026-03)
  - Scripts de setup one-shot couverts par les migrations automatiques
  ```

```bash
git add docs/archive/README.md scripts/_archive/README.md
git commit -m "docs: ajouter README dans dossiers archive"
```

---

### F7 — Mise à jour règle `docs/FR/` dans `CLAUDE.md`

Couvert en F5 (règle à ajouter dans CLAUDE.md).

---

### F8 — Refactoring `launcher.py` (post-v7) ✅ COMPLÉTÉ

**Statut : Complété le 2026-04-11 sur la branche `v7/cockpit`.**

`launcher.py` faisait **2084L** avec 49 fonctions. Découpé en 6 modules :

| Module extrait | Contenu réel | Taille |
|---|---|---|
| `src/utils/launcher_env.py` | détection langue, venv, `_find_system_python`, `_cmd_setup` | 325L |
| `src/utils/launcher_players.py` | `PlayerInfo`, `_list_players`, `_cmd_doctor` | 333L |
| `src/utils/launcher_migrations.py` | `_run_migrations()`, `_run_db_healthcheck()` | 142L |
| `src/utils/launcher_startup.py` | signal handlers, `_launch_streamlit()` | 189L |
| `src/utils/launcher_sync.py` | `_sync_player_duckdb`, `_cmd_sync`, `_cmd_info` | 265L |
| `src/utils/launcher_onboarding.py` | `_onboard_first_player`, `_cmd_add_player`, MSAL | 462L |
| `launcher.py` résiduel | menu interactif, argparse, re-exports compat tests | ~440L |

---

### Vue d'ensemble bloc F

| ID | Item | Effort | Fenêtre |
|---|---|---|---|
| F0 | git rm .tmp.* + .bak + rm data/tmp_*.py | XS | **Immédiat** |
| F1 | git mv _exp_spawn_*.py → experimental/ | XS | Pré-v7 |
| F2 | Configurer thumbnails-watcher.service (user=deploy) + déplacer packaging/ | XS | Pré-v7 |
| F3 | Archiver experimental/ → _archive/experimental/ | XS | Pré-v7 |
| F4 | Archiver create_citations + compute_sessions | XS | Pré-v7 |
| F5 | Mettre à jour docs/FR/ + règle CLAUDE.md | S | Pré-v7 |
| F6 | README dans docs/archive/ et scripts/_archive/ | XS | Pré-v7 |
| F8 | Refactoring launcher.py (split en 5 modules) | L | Post-v7 |

**Estimation bloc F pré-v7 : 1 session (F0–F6 peuvent tenir en 2–3 commits groupés).**

---

## Bloc 1 — Pre-v7 (avant de partir sur `v7/cockpit`)

> Objectif : arriver en v7 avec une base saine. Ces items sont tous dans des couches
> indépendantes de l'UI et peuvent se faire sur `main` ou une branche dédiée courte.

---

### H1 — Corriger les 2 tests en échec dans la suite stable

**Priorité : immédiate — à faire avant tout commit sur main.**

Les deux failures sont silencieuses (la CI tourne avec `--ignore`), ce qui masque de futures régressions.

#### H1a — `test_intensity_heatmap_viz.py`

```
AssertionError: assert None == 'reversed'
fig.layout.yaxis.autorange
```

Fichier : `tests/test_intensity_heatmap_viz.py` — `TestPlotMatchIntensityHeatmap::test_five_matches_returns_figure`

**Cause probable :** `yaxis.autorange = "reversed"` a été retiré ou conditionné dans
`src/visualization/match_intensity_heatmap.py` lors d'une mise à jour visuelle récente.
Le test attend un comportement qui n'est plus produit.

**Action :**
- [ ] Lire `match_intensity_heatmap.py` — vérifier si `autorange = "reversed"` a été
  retiré intentionnellement ou accidentellement
- [ ] Si intentionnel : mettre à jour le test pour refléter le nouveau comportement
- [ ] Si régressif : restaurer `autorange = "reversed"` dans la figure

---

#### H1b — `test_teammates_legend.py`

```
AssertionError: assert 'llp-squad-start' in '<div id="llp-fixed-panel" ...>'
```

Fichier : `tests/test_teammates_legend.py` — `TestRenderPlayerLegendPanelFixed::test_fixed_html_has_start_sentinel`

**Cause probable :** le sentinel `llp-squad-start` a été renommé ou supprimé dans le HTML
généré par la fonction de légende Teammates lors d'une refonte du composant.

**Action :**
- [ ] Lire `src/ui/pages/teammates_legend.py` — trouver le nouveau nom du sentinel ou
  confirmer sa suppression
- [ ] Mettre à jour le test en conséquence (ou restaurer le sentinel si nécessaire pour
  un mécanisme de scroll/navigation JS)

---

### H2 — Retirer les legacy kwargs de la couche sync

**Priorité : haute — ils court-circuitent `SyncScope` sans avertissement.**

`SyncScope` est l'API cible depuis v5 (`scope=SyncScope(...)`), mais 4 fonctions
conservent leurs 30+ kwargs individuels marqués `# LEGACY` :

- `backfill_player_data`
- `backfill_all_players`
- `_backfill_with_api`
- `find_matches_missing_data`

Tant qu'ils restent actifs, de nouveaux callers (scripts one-shot, tests ad hoc) peuvent
les utiliser sans que personne s'en aperçoive.

**Action :**
- [ ] **Audit des callers actifs** : lancer `grep -rn "backfill_player_data\|backfill_all_players" scripts/ tests/ --include="*.py"` et lister tous les appels qui passent encore des kwargs individuels au lieu de `scope=`
- [ ] **Migrer les callers** identifiés vers `SyncScope`
- [ ] **Ajouter un `DeprecationWarning` temporaire** sur chaque kwarg LEGACY (logging warning à l'appel) — ne pas supprimer encore, mais rendre le problème visible
- [ ] **Planifier la suppression** au prochain cycle : une fois qu'aucun caller actif n'utilise les kwargs individuels, les retirer proprement
- [ ] Mettre à jour `tests/test_backfill_cli.py` si des tests passent des kwargs legacy directement

**Règle à ajouter dans CLAUDE.md :**
> Les kwargs individuels `medals=True`, `force_medals=True`, etc. dans `backfill_player_data`
> sont marqués `# LEGACY` — ne jamais les utiliser dans du nouveau code. Toujours passer
> `scope=SyncScope(medals=True, force_medals=True)`.

---

### H3 — Découper `migrations.py` par domaine cible

**Priorité : moyenne — le fichier fait 1862L avec 31 fonctions `ensure_*`.**

`src/data/sync/migrations.py` regroupe les migrations de 4 bases distinctes dans un seul
module. Le système de steps `src/data/migration/steps/` existe déjà mais `migrations.py`
continue de croître en parallèle.

**Cibles :**

| Fichier cible | Contenu | Fonctions `ensure_*` concernées |
|---|---|---|
| `migrations_player.py` | Évolutions `stats.duckdb` (player) | `ensure_player_match_enrichment_*`, `ensure_sessions_*`, `ensure_media_*`, `ensure_sync_meta_*`, `ensure_mv_*`, `ensure_pme_*`... |
| `migrations_shared.py` | Évolutions `shared_matches_v2.duckdb` | `ensure_match_participants_*`, `ensure_highlight_events_*`, `ensure_xuid_*`, `ensure_weapon_kills_*`, `ensure_match_registry_*`, `ensure_resolution_views`... |
| `migrations_metadata.py` | Évolutions `metadata.duckdb` | `ensure_weapon_labels`, `ensure_asset_translations_*`, `ensure_medal_*`, `ensure_mode_*`... |
| `migrations.py` (résiduel) | Helpers partagés : `_add_column_if_missing`, `_WEAPON_KILLS_LEGACY_COLUMNS`, etc. | Reste isolé comme module utilitaire bas niveau |

**Action :**
- [ ] Identifier et classer chaque `def ensure_*` par base cible (`grep "^def ensure_"`)
- [ ] Créer les 3 nouveaux modules avec les fonctions déplacées
- [ ] Mettre à jour tous les imports dans `src/data/migration/steps/` et dans `launcher.py → _run_migrations()`
- [ ] `migrations.py` résiduel conserve uniquement `_add_column_if_missing` et les constantes partagées
- [ ] Vérifier que la suite de tests `tests/migration/` passe intégralement après le split
- [ ] **Aucune régression tolérée** : les fonctions `ensure_*` sont idempotentes, le split ne change pas leur comportement

**Note :** ne pas déplacer de logique — déplacer les définitions uniquement.

---

### H4 — Convertir `launcher_i18n.py` en JSON

**Priorité : basse — travail trivial, gain de lisibilité et performance de parse.**

`src/utils/launcher_i18n.py` fait **813L** alors qu'il contient exclusivement un `dict`
de 368 clés i18n. C'est le fichier Python le plus long du projet `src/utils/`. Un fichier
Python qui ne contient que des données statiques est un anti-pattern.

**Action :**
- [ ] Créer `src/utils/launcher_i18n.json` — exporter le `STRINGS` dict au format JSON
- [ ] Modifier `src/utils/launcher_i18n.py` : ne garder que la fonction `t()` (< 15L)
  qui charge le JSON au démarrage via `importlib.resources` ou `pathlib`
- [ ] Mettre à jour `scripts/enforce_size_limits.py` — retirer `launcher_i18n.py` de la whitelist
- [ ] Vérifier que `launcher.py` fonctionne toujours (`_t("setup_done", lang)`)

---

## Bloc 2 — Post-v7 (après validation de `v7/cockpit` sur sous-domaine)

> Objectif : consolider l'archi une fois que la v7 a stabilisé la forme de l'app.
> Ces items ont tous une dépendance sur la structure finale du shell v7.

---

### H5 — Consolider la stratégie de cache

**Contexte :** le cache est actuellement fragmenté sur **17 fichiers** :
`_cache_core.py`, `_cache_loading.py` (3), `_cache_queries.py` (16!), `_cache_sessions.py`,
`_sync_duckdb_ops.py`, `cache_filters.py`, `cache_loaders.py`, `cache_social.py`,
`cache_control.py`, `commendations.py`, `medals.py`, `multiplayer.py`,
`explorer.py`, `match_view_helpers.py`, `teammates_helpers.py`, `data_loader.py`
— plus les `lru_cache` dans `translations.py`, `aliases.py`, `career_ranks.py`...

Aucune règle décisionnelle n'existe sur `lru_cache` vs `st.cache_data` vs rien.

**Pourquoi post-v7 :** la v7 redessinera les patterns de chargement de données
(hub-first, fragment, L2 filtres). Consolider le cache avant d'avoir la forme finale
du shell risque de forcer un deuxième passage.

**Action :**
- [ ] **Audit** : inventaire de toutes les fonctions cachées, leur TTL actuel (par défaut ou explicite), leur scope (process vs session Streamlit)
- [ ] **Règle décisionnelle à écrire** (à ajouter dans `CLAUDE.md`) :
  - `st.cache_data` → données lues depuis DuckDB qui ne changent pas pendant une session (matchs, médailles, gamertags)
  - `st.cache_resource` → connexions, objets lourds partagés inter-sessions (repos, clients)
  - `lru_cache` → lookups de référentiels statiques (traductions, labels, rangs) — ne pas mettre de données joueur ici
  - Rien → tout ce qui dépend des filtres courants ou du `session_state`
- [ ] **TTL explicites** : tout `@st.cache_data` doit avoir un `ttl=` explicite ou un commentaire justifiant l'absence
- [ ] **Clés d'invalidation** : `app/cache_control.py` devient le point d'entrée unique pour l'invalidation — supprimer les `clear_cache()` locaux qui court-circuitent cette logique
- [ ] **Regroupement** : `_cache_core.py`, `_cache_loading.py`, `_cache_queries.py`, `cache_loaders.py` ont des responsabilités qui se recoupent — consolider en `cache_data.py` (données matchs) + `cache_ui.py` (éléments UI lourds) + garder `cache_control.py` seul pour l'invalidation

---

### H6 — Extraire la logique API de `profile_api.py` vers un service

**Contexte :** `src/ui/profile_api.py` (466L) importe directement :
- `SPNKrAPIClient` depuis `src.data.sync.api_client`
- `Tokens` depuis `src.data.sync._tokens`
- `refresh_halo_tokens` depuis `src.data.sync._auth`
- `create_api_client` depuis `src.data.sync.api_factory`

L'UI orchestre des appels API Halo. C'est une violation du principe que les modules `ui/`
ne touchent pas `data.sync.*` directement.

**Pourquoi post-v7 :** la v7 introduit un `Profil` hub qui regroupera les paramètres et
la gestion du compte. La forme finale de cet hub détermine où cette logique doit atterrir.

**Action :**
- [ ] Créer `src/data/services/profile_api_service.py` — extraire la logique d'appel API (fetch gamertag, fetch career rank progression, fetch profile assets) hors de `profile_api.py`
- [ ] `profile_api.py` devient un module UI pur qui appelle `profile_api_service.py` et rend les composants Streamlit
- [ ] `profile_api_tokens.py` et `xbox_oauth.py` : évaluer si la logique d'auth peut être absorbée par `src/auth/provider.py` (qui est déjà le point d'entrée auth)
- [ ] Mettre à jour les imports dans `streamlit_app.py` et `streamlit_app_v7.py`

---

### H7 — Enforcer `src/ports/` comme frontière entre UI et data

**Contexte :** `DataRepository` et `HaloAPIPort` sont définis dans `src/ports/` et
documentés comme l'abstraction cible, mais il existe **50 imports directs** de
`src.data.repositories.*` et `src.data.sync.*` depuis `src/ui/`.

Les violations actuelles identifiées :
- `src/ui/pages/career_top_matches_data.py` → `_career_encounters_repo`
- `src/ui/pages/match_view_encounters.py` → `_encounter_loader`
- `src/ui/profile_api.py` → `_tokens`, `api_client`, `api_factory` (couvert en H6)
- `src/ui/translations.py` → `_asset_langs` (3 endroits)
- `src/ui/_sync_duckdb_ops.py` → `DuckDBSyncEngine`, `SyncOptions`

**Pourquoi post-v7 :** la v7 restructure les hubs — certains de ces imports existeront
dans des fichiers qui seront réécrits de toute façon. Enforcer maintenant introduit du
refactoring qui sera partiellement rejeté pendant la v7.

**Action :**
- [ ] **Audit complet** : `grep -rn "from src.data" src/ui/ src/app/ --include="*.py"` — inventaire exhaustif
- [ ] **Règle dans `CLAUDE.md`** : les modules `src/ui/` et `src/app/` ne doivent pas importer depuis `src/data/repositories/_*` (sous-modules privés) ni depuis `src/data/sync/_*`. Imports autorisés : `src/ports/`, `src/data/services/`, `src/data/repositories/duckdb_repo.py` (façade publique uniquement)
- [ ] **Ajouter un test d'architecture** dans `tests/test_architecture_review_*.py` qui vérifie l'absence de ces imports cross-couche
- [ ] Traiter les violations une à une : soit déplacer la logique dans un service, soit exposer la fonction via `DuckDBRepository` ou un nouveau port

---

### H8 — Clarifier `TypedDict` vs `dataclass` vs `BaseModel`

**Contexte :** les trois coexistent sans règle explicite :
- `src/data/domain/models/` : Pydantic `BaseModel` ✔ (validation frontière API)
- `src/app/_page_context.py` : `TypedDict` pour 5 contextes inter-modules
- `src/analysis/_kv_types.py` : `TypedDict` pour stats joueur
- Partout ailleurs : `@dataclass`, dicts nus, tuples

Les `TypedDict` de `_page_context.py` (`MatchViewParams`, `TeammateContext`,
`FilterSidebarCallbacks`...) sont passés entre modules et ont des valeurs par défaut
implicites — ce sont de bons candidats à `dataclass(frozen=True)`.

**Action :**
- [ ] **Ajouter dans `CLAUDE.md` la règle décisionnelle** :
  - `BaseModel` (Pydantic v2) → données qui traversent une frontière externe (API, JSON, CSV) et nécessitent validation/coercion
  - `@dataclass(frozen=True)` → structures internes entre modules avec types explicites et immuabilité souhaitée (contextes UI, paramètres de page, résultats d'analyse)
  - `TypedDict` → uniquement pour annoter des dicts dont la structure est imposée par une lib externe (plotly layout, spy kwargs...)
  - Dict nu → uniquement dans du code throwaway ou des fonctions locales < 10L
- [ ] **Migrer `_page_context.py`** : convertir `MatchViewParams`, `TeammateContext`, `FilterSidebarCallbacks`, `TeammateFilterOptions`, `TeammateCallbacks` en `@dataclass(frozen=True)`
- [ ] Vérifier que les tests de page continuent de passer après la migration

---

### H9 — Découper `_weapon_kills_repo.py` (566L) et `_materialized_views.py` (511L)

**Contexte :** deux fichiers dans `src/data/repositories/` dépassent le seuil 500L
sans être dans la whitelist de `enforce_size_limits.py`. Ils sont actuellement ignorés
mais seront touchés lors de l'intégration du hub Escouade (armes, stats squad).

**Action :**
- [ ] `_weapon_kills_repo.py` (566L) → extraire les helpers de réconciliation
  weapon_id dans `_weapon_kills_reconcile.py`, garder les loaders dans le fichier principal
- [ ] `_materialized_views.py` (511L) → séparer les views player (`_mv_player.py`) des
  views shared (`_mv_shared.py`)
- [ ] Vérifier les tests de weapons + materialized views après chaque split

---

## Vue d'ensemble priorisée

### Bloc Pre-v7 (à traiter sur une branche courte avant de partir sur `v7/cockpit`)

| ID | Item | Effort | Risque |
|---|---|---|---|
| H1a | Fix test `yaxis.autorange` | XS | Nul |
| H1b | Fix test `llp-squad-start` | XS | Nul |
| H2 | Legacy kwargs sync → DeprecationWarning + audit callers | S | Faible |
| H3 | Split `migrations.py` par domaine | M | Faible (refactor structurel, 0 logique changée) |
| H4 | `launcher_i18n.py` → JSON | XS | Nul |

**Estimation bloc pre-v7 : 2 à 3 sessions.**

### Bloc Post-v7 (après validation preview sur sous-domaine)

| ID | Item | Effort | Risque |
|---|---|---|---|
| H5 | Stratégie cache unifiée | M | Moyen (impact sur tous les chargements) |
| H6 | Extraire logique API de `profile_api.py` | S | Faible |
| H7 | Enforcer `src/ports/` comme frontière cross-couche | M-L | Moyen (50 imports à traiter) |
| H8 | Règle TypedDict/dataclass + migration `_page_context.py` | S | Faible |
| H9 | Split `_weapon_kills_repo.py` + `_materialized_views.py` | S | Faible |

**Estimation bloc post-v7 : 3 à 4 sessions.**

---

## Fichiers impactés — référence rapide

| Fichier | Item |
|---|---|
| `tests/test_intensity_heatmap_viz.py` + `src/visualization/match_intensity_heatmap.py` | H1a |
| `tests/test_teammates_legend.py` + `src/ui/pages/teammates_legend.py` | H1b |
| `src/data/sync/engine.py`, `_engine_fanout.py`, `backfill_data.py` | H2 |
| `src/data/sync/migrations.py` → `migrations_player.py`, `migrations_shared.py`, `migrations_metadata.py` | H3 |
| `src/utils/launcher_i18n.py` → `launcher_i18n.json` | H4 |
| `src/ui/_cache_*.py`, `src/app/cache_control.py`, 7+ autres | H5 |
| `src/ui/profile_api.py`, `profile_api_tokens.py`, `xbox_oauth.py` | H6 |
| `src/ui/pages/career_top_matches_data.py`, `match_view_encounters.py`, `translations.py` + 40+ fichiers | H7 |
| `src/app/_page_context.py`, `CLAUDE.md` | H8 |
| `src/data/repositories/_weapon_kills_repo.py`, `_materialized_views.py` | H9 |
