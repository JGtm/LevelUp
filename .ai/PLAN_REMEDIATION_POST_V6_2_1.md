# Plan de Remediation Post-v6.2.1

> Derniere revue croisee : 2026-04-07 (verification code vs constats)

## Objet

Ce document transforme la revue chirurgicale du delta Git entre v6.2.1 et HEAD en plan d'action executable.

Objectif: corriger d'abord les risques runtime et de structure, puis reduire la dette d'orchestration apparue autour de la persistance UI, des services de fond Streamlit, des migrations inter-bases et du pipeline de deploiement.

Annexe de classification: `.ai/ANNEXE_CLASSIFICATION_POST_V6_2_1.md`

## Constats Prioritaires

1. La persistance dite navigateur n'a pas un contrat clair entre le vrai composant localStorage et la logique Python actuellement branchee sur un fichier global serveur.
2. Le watcher media Linux n'est pas protege par le meme guard process-level que le polling legacy et peut etre relance au rerun.
3. Le guard des migrations shared valide trop tot la DB comme traitee, meme si une migration critique a echoue.
4. L'ordre des migrations reste fragile entre metadata et shared, avec fallback silencieux sur des vues degradees.
5. Le healthcheck auto-repair ne recompose pas le statut global apres reparation et le deploy ne distingue pas clairement repaired de ok.
6. La logique destructive de deploiement est executee **deux fois** a chaque deploy : le workflow fait `git fetch/reset/clean` **puis** appelle `deploy.sh` qui refait les memes 3 commandes.
7. La documentation agentique n'est plus alignee avec l'architecture v6 reelle.
8. Deux systemes de migrations coexistent sans coordination : les migrations inline de `_engine_connections._run_shared_migrations()` et le runner `migration/runner.py`.

## Principes De Correction

- Supprimer les ambiguities de contrat avant d'ajouter des gardes supplementaires.
- Preferer une source de verite unique par flux.
- Remplacer les fallbacks silencieux par des dependances explicites ou des statuts distincts.
- Ajouter des tests aux frontieres runtime, pas seulement sur la logique pure.
- Ne pas etendre davantage les modules deja en surcharge sans extraction prealable.

## Phase 0 - Stabilisation Runtime

### P0.1 Unifier la persistance UI

Effort estime: **moyen** (~2-4h)

Etat des lieux — il y a **trois** systemes de persistance, pas deux:

1. `data/ui_prefs.json` — prefs globales serveur (langue, dernier db_path). Ecrit par `src/ui/components/browser_storage/` malgre son nom. Risque d'ecrasement croise entre sessions navigateur.
2. `data/players/{gamertag}/ui_prefs.json` — prefs filtre par joueur (via `src/ui/filter_state.py`). Migration silencieuse depuis `.streamlit/`. Contrat serveur-side coherent pour un usage mono-utilisateur.
3. `src/ui/components/browser_storage/frontend/index.html` — composant Streamlit frontend localStorage. **Code mort** : present dans le repo, non cable dans le flux principal.

Decision a prendre:

- Le systeme (1) doit-il etre conserve? LevelUp est mono-utilisateur en pratique (un seul gamertag principal) et le VPS Docker n'a qu'un consommateur. La persistance serveur-side est **appropriee** dans ce contexte.
- Si oui, le composant frontend (3) est du code mort a supprimer, et les commentaires "localStorage navigateur" dans `data_loader.py` (priorite 3) doivent etre corriges.

Actions:

1. Supprimer le composant frontend mort `src/ui/components/browser_storage/frontend/` ou le brancher reellement.
2. Renommer/corriger les commentaires qui mentionnent "localStorage navigateur" alors que la persistance est fichier serveur.
3. Verifier que `data/ui_prefs.json` n'est pas ecrase entre sessions Streamlit concurrentes (bas risque en mono-utilisateur, mais a documenter).
4. La migration `.streamlit -> data/players` dans `filter_state._resolve_prefs_path()` est deja protegee par `try/except` — le risque de perte est negligeable (crash entre `copy2` et `unlink` sur un fichier de ~1Ko). Aucune modification necessaire.
5. Documenter le contrat retenu dans un commentaire en tete de `browser_storage/__init__.py`.

Critere d'acceptation:

- Aucun code mort frontend dans le repo (ou branche reellement).
- Aucun commentaire dans le code actif ne mentionne "localStorage navigateur" pour de la persistence fichier.
- Les tests `tests/test_ui_persistence_v64.py` couvrent le vrai chemin retenu.

### P0.2 Poser un guard process-level sur le watcher Linux

Effort estime: **faible** (~30min, fix de 5-10 lignes)

Cause racine: dans `media_background.background_media_indexing()`, le chemin Linux fait `start_media_watcher(...)` puis `return` **avant** le guard `_PERIODIC_STARTED` (L140-148). Chaque rerun Streamlit peut donc creer un nouvel `Observer()` watchdog + un nouveau thread de scan initial.

Correction recommandee — mutualiser le guard:

```python
# media_background.py, AVANT le branchement Linux
with _PERIODIC_LOCK:
    if _PERIODIC_STARTED:
        logger.debug("Indexation medias deja active (process-level guard)")
        return
    _PERIODIC_STARTED = True

# Puis le branchement Linux/Windows
if platform.system() == "Linux" and ...:
    start_media_watcher(base_path, settings)
    return
# ... thread periodique Windows
```

Actions:

1. Deplacer le guard `_PERIODIC_LOCK` + `_PERIODIC_STARTED` **avant** le branchement Linux dans `background_media_indexing()`.
2. Verifier que `start_media_watcher()` n'a pas besoin d'un guard supplementaire interne (actuellement non — le guard externe suffit).
3. Journaliser explicitement le mode actif: `watcher inotify (Linux)` ou `polling periodique (Windows/legacy)`.

Critere d'acceptation:

- Un seul observer watchdog par process.
- Un seul scan initial par process.
- Test de non-regression sur reruns repetes.

### P0.3 Rendre le guard des migrations success-based

Effort estime: **faible** (~1h)

Cause racine: dans `_engine_connections._run_shared_migrations()` (L246-249), `_SHARED_MIGRATIONS_DONE.add(db_key)` est appele **avant** les appels aux fonctions de migration. Si `ensure_resolution_views` ou `ensure_match_participants_columns` echouent, la DB reste marquee comme migree pour tout le processus.

Correction recommandee:

```python
# _engine_connections.py, _run_shared_migrations()
if db_key and db_key in _SHARED_MIGRATIONS_DONE:
    return
# NE PAS add() ici — le faire apres succes
try:
    ensure_match_participants_columns(conn)
    ensure_performance_indexes(conn)
    ensure_match_registry_spnkr_version(conn)
    ensure_weapon_kills_table(conn)
    ensure_resolution_views(conn)  # critique — depend de metadata
except Exception as e:
    logger.warning("Migration shared incomplete — sera retentee: %s", e)
    return  # PAS de add() → retry au prochain appel
if db_key:
    _SHARED_MIGRATIONS_DONE.add(db_key)  # succes confirme
```

Actions:

1. Deplacer `_SHARED_MIGRATIONS_DONE.add(db_key)` **apres** le bloc de migrations.
2. Distinguer migrations critiques (ensure_resolution_views) et best-effort (ensure_performance_indexes) — un echec best-effort ne doit pas bloquer le marquage.
3. Si une migration critique echoue, ne pas ajouter au set → le prochain `SyncEngine` retentera.

Critere d'acceptation:

- Une erreur transitoire sur ensure_resolution_views n'est plus memorisee comme succes process-level.
- Test: simuler un echec de migration → verifier que le retry suivant la relance.

## Phase 1 - Determinisme Structurel

### P1.1 Corriger l'ordre de dependance metadata -> shared

Effort estime: **moyen** (~2-3h)

Deux systemes de migration coexistent et sont tous deux impactes:

1. **runner.py** (`apply_pending_migrations`): ordre fixe `shared → shared_pve → player → metadata`. Metadata passe en dernier alors que shared en a besoin pour les vues i18n.
2. **_engine_connections._run_shared_migrations()**: migrations inline executees a la connexion. Elles appellent `ensure_resolution_views` qui construit `v_match_full` avec traductions FR de metadata. Si metadata n'est pas attachee, le fallback `NULL AS map_name_fr` s'applique silencieusement.

Les deux chemins doivent etre corriges.

Actions:

1. Dans `runner.py`: changer l'ordre `db_map` pour traiter **metadata avant shared**:
   ```python
   db_map = [
       (metadata_db_path, "metadata"),   # 1er — prerequis i18n
       (shared_db_path, "shared"),        # 2eme — peut attacher metadata
       (pve_db_path, "shared_pve"),
       (player_db_path, "player"),
   ]
   ```
2. Dans `_run_shared_migrations`: documenter en commentaire la dependance a metadata. Si metadata n'est pas encore attachee au moment de l'appel, `ensure_resolution_views` devrait emettre un warning explicite plutot que de creer des vues degradees silencieusement.
3. Supprimer le fallback `NULL AS ..._fr` ou le remplacer par un warning fort + flag dans le HealthCheckResult.

Critere d'acceptation:

- La sequence du runner est deterministe: metadata avant shared.
- Les vues degradees emettent un warning visible dans les logs.
- Sur un depot vierge, un seul passage du runner produit des vues completes.

### P1.2 Fiabiliser le healthcheck apres auto-repair

Effort estime: **faible** (~1-2h)

Bug precis: `HealthCheckResult.add()` (healthcheck_db.py L80-86) ne gere que deux transitions:
- `error/broken` → status = `"error"`
- `missing/warning` → status = `"warning"` (seulement si actuellement `"ok"`)

Le statut `"repaired"` n'a **aucun effet** sur le statut global. De plus, `_try_repair_views()` modifie les checks existants **apres** que `add()` a deja calcule le statut, sans recomposition.

Correction recommandee:

```python
def add(self, check: CheckDetail) -> None:
    self.checks.append(check)
    if check.status in ("error", "broken") and self.status != "error":
        self.status = "error"
    elif check.status == "repaired" and self.status == "ok":
        self.status = "warning"  # repaired = warning-level pour le deploy
    elif check.status in ("missing", "warning") and self.status == "ok":
        self.status = "warning"
```

Et ajouter une methode `recompute_status()` appelee apres `_try_repair_views()`.

Actions:

1. Ajouter le traitement de `"repaired"` dans `add()` : promouvoir en `"warning"`.
2. Ajouter `recompute_status()` pour recalculer le global apres mutation des checks.
3. Appeler `recompute_status()` apres `_try_repair_views()` dans `run_healthcheck()`.
4. Dans `deploy.sh`, ajouter le cas `repaired` au parsing JSON (le traiter comme `WARNING`).

Critere d'acceptation:

- Le JSON du healthcheck n'est plus incoherent entre checks details et status global.
- Le deploy sait distinguer ok, repaired/warning, error.

### P1.3 Sortir les fallbacks de structure silencieux

Effort estime: **faible** (~1h, en synergie avec P1.1)

Fallbacks identifies dans le code:

| Fichier | Ligne | Fallback | Impact |
|---------|-------|----------|--------|
| `src/data/sync/migrations.py` | ~L681 | `NULL AS map_name_fr, ...game_variant_name_fr` | Vue `mv_player_matches` sans traductions FR |
| `src/data/sync/migrations.py` | ~L1508-1515 | `NULL AS map_name_fr, ...playlist_canonical_fr` | Vue `v_match_full` sans traductions FR, ni mode_name |

Ces deux fallbacks sont dans la meme famille : absence de metadata attachee au moment de la creation des vues. Le second (L1508) est plus grave car `v_match_full` est la vue de reference pour toutes les pages UI. Les deux sont directement lies a P1.1 (ordre metadata/shared).

Actions:

1. Ajouter un `logger.warning()` explicite quand le fallback `NULL AS ..._fr` est declenche, pour rendre visible un eventuel probleme d'ordre.
2. Ajouter un check dans le healthcheck pour detecter des vues avec colonnes FR toutes NULL.
3. Si P1.1 est applique, le fallback peut etre conserve comme filet de securite (pas de suppression necessaire) tant qu'il est visible.

Critere d'acceptation:

- Le fallback NULL emmet un warning dans les logs.
- Le healthcheck detecte une vue degradee.

## Phase 2 - Hygiene Infra Et Documentation

### P2.1 Dedoublonner le deploiement

Effort estime: **faible** (~15min)

Etat reel: le workflow `.github/workflows/deploy.yml` fait `git fetch/reset/clean` (L22-24) **puis** appelle `bash /opt/levelup/deploy.sh` qui execute les **memes 3 commandes** (L21-23). Les commandes destructives s'executent donc **deux fois de suite** a chaque deploy — pas juste une duplication de code, mais une double execution.

Correction recommandee — supprimer les 3 lignes git du workflow et ne garder que l'appel a `deploy.sh`:

```yaml
# deploy.yml
script: |
  git config --global --add safe.directory /opt/levelup
  cd /opt/levelup
  bash /opt/levelup/deploy.sh
```

Actions:

1. Supprimer `git fetch`, `git reset --hard` et `git clean -fd` du workflow.
2. Conserver ces commandes uniquement dans `deploy.sh` (source de verite unique).
3. Verifier que `deploy.sh` est bien executable et que le `cd /opt/levelup` dans le workflow n'est pas necessaire (le script fait son propre `cd`).

Critere d'acceptation:

- Les commandes `git fetch/reset/clean` n'apparaissent qu'une seule fois dans le pipeline complet.

### P2.2 Recaler la doc agentique sur v6

Effort estime: **faible** (~30min)

Fichiers a corriger (references `shared_matches.duckdb` alors que le code utilise `shared_matches_v2.duckdb` — cf. `src/utils/paths.py:90`):

- `CLAUDE.md` : tableau Architecture des Donnees (L38)
- `.github/copilot-instructions.md` : tableau Architecture des Donnees (L43)
- `docs/ARCHITECTURE_V6.md` : a verifier

Actions:

1. Remplacer `shared_matches.duckdb` par `shared_matches_v2.duckdb` dans tous les documents d'instructions.
2. Verifier que les noms de vues SQL listees (`v_gamertag_lookup`, `v_match_full`, etc.) correspondent toujours aux vues reelles.
3. Retirer les references qui peuvent pousser un agent a recreer du legacy.

Critere d'acceptation:

- `git grep 'shared_matches\.duckdb'` ne retourne aucun hit dans les fichiers de documentation/instructions (hors archives `.ai/archive/`).

### P2.3 Ajouter les tests manquants aux frontieres runtime

Effort estime: **moyen** (~2-3h au total, a repartir au fil de P0/P1)

Tests existants: `tests/test_ui_persistence_v64.py` couvre `_resolve_prefs_path` et la migration legacy. Les tests ci-dessous sont **complementaires**.

Priorites de test:

1. **Guard watcher Linux** : appeler `background_media_indexing()` deux fois → verifier qu'un seul Observer est cree (mocker `watchdog.observers.Observer`).
2. **Retry migrations shared** : mocker `ensure_resolution_views` pour qu'il echoue → verifier que `_SHARED_MIGRATIONS_DONE` ne contient PAS le db_key → rappeler `_run_shared_migrations` → verifier qu'il retente.
3. **Healthcheck recompute** : creer un `HealthCheckResult` → appeler `_try_repair_views` qui passe un check en `repaired` → verifier que `result.status` est `"warning"` (pas `"ok"`).
4. **Persistance UI** : verifier que `browser_storage` ne cree pas de fichier parasite ou n'interfere pas avec `data/ui_prefs.json`.
5. **Deploy parsing** : tester le parsing Python inline de `deploy.sh` avec des JSON entries contenant `status: "repaired"` → verifier la sortie.

## Observations Complementaires

Les points ci-dessous sont reels, mais secondaires par rapport aux P0/P1 ci-dessus. Ils ne doivent pas re-prioriser le plan, sauf si un chantier adjacent y touche deja.

### O1 - Nommage et contrat trompeurs autour de browser_storage

Si la voie serveur-side est retenue, le nom du module et les commentaires mentionnant localStorage deviennent faux et doivent etre corriges. Sinon, le frontend localStorage actuellement present doit etre branche pour de vrai.

### O2 - Journalisation du fallback watchdog

Le fallback quand watchdog est indisponible est fonctionnellement acceptable, mais trop peu distinctif pour l'exploitation. Il faut rendre explicite le mode retenu et la raison du fallback.

### O3 - Retour de succes des retries media

L'indexation avec retry logue mal les cas ou une tentative intermediaire reussit apres erreur transitoire. Cela nuit a l'observabilite et au diagnostic, sans etre un bug metier direct.

### O4 - Dette de taille et de complexite sur les modules critiques

Les modules suivants restent proches ou au-dessus des seuils internes, ce qui augmente le cout de correction et le risque de workaround futur:

| Module | Lignes | Seuil 500L |
|--------|--------|------------|
| `src/data/sync/_engine_connections.py` | 535 | **depasse** |
| `src/ui/pages/media_tab.py` | 483 | proche |
| `src/utils/healthcheck_db.py` | 471 | proche |
| `src/ui/pages/settings.py` | 452 | ok |

Seul `_engine_connections.py` depasse le seuil. Les trois autres sont en zone de vigilance.

Cette dette n'est pas necessairement a traiter avant les P0/P1, mais elle doit borner tout nouveau changement dans ces fichiers.

### O5 - Documentation active a harmoniser au-dela de CLAUDE.md

CLAUDE.md est le cas le plus visible, mais la verification de coherence doit couvrir tous les documents d'instructions actifs et pas seulement la documentation publique.

### O6 - _bootstrap_shared_matches_db et launcher.py sans context manager

`_engine_connections._bootstrap_shared_matches_db()` (L155-170) et `launcher.py` (L323, L390, L1230) utilisent `duckdb.connect()` avec `try/finally conn.close()` au lieu de context managers — anti-pattern `Bare connect` liste dans les regles du projet. Au total 4 occurrences:

| Fichier | Ligne | Usage |
|---------|-------|-------|
| `_engine_connections.py` | ~L159 | Bootstrap shared (R/W) |
| `launcher.py` | L323 | Lecture stats joueur (R/O) |
| `launcher.py` | L390 | Count matchs joueur (R/O) |
| `launcher.py` | L1230 | Init metadata.duckdb vide (R/W) |

A corriger si ces fichiers sont touches par P0.3 ou un autre chantier.

### O7 - Double systeme de migrations non coordonne

Les migrations inline dans `_engine_connections._run_shared_migrations()` et le runner `migration/runner.py` coexistent sans coordination. Un echec dans l'un n'est pas visible par l'autre. Les migrations inline n'ont pas de table `schema_migrations` pour tracer leur application. Cela cree un angle mort ou une migration peut etre appliquee deux fois (runner + inline) ou ratee silencieusement (inline echoue, runner ne sait pas).

A terme, les migrations inline devraient etre absorbees par le runner. En attendant, documenter la coexistence et s'assurer que les deux chemins sont idempotents.

### O8 - PVE migrations sans guard process-level

`_engine_connections._get_pve_connection()` (L285-299) appelle `ensure_pve_schema(conn)` a chaque nouvelle connexion PVE sans guard equivalent a `_SHARED_MIGRATIONS_DONE`. Comme `ensure_pve_schema` utilise `IF NOT EXISTS` (idempotent), l'impact est nul en termes de corruption, mais le pattern est **inconsistant** avec shared et gaspille des cycles sur chaque SyncEngine instancie (fanout inclus). Correction de faible priorite : ajouter un guard `_PVE_MIGRATIONS_DONE` similaire.

### O9 - _SHARED_MIGRATIONS_DONE sans lock (TOCTOU theorique)

`_SHARED_MIGRATIONS_DONE` (set mutable module-level dans `_engine_connections.py`) est lu et ecrit sans `threading.Lock`, contrairement au guard `_PERIODIC_STARTED` dans `media_background.py` qui utilise `_PERIODIC_LOCK`. En pratique, `_run_shared_migrations` n'est **jamais** appele depuis des threads concurrents (le sync engine est sequentiel, le fanout est serialise). Le risque est donc **theorique** a ce stade. Si un parallelisme est introduit a l'avenir, ce set devra etre protege.

## Ordre Recommande

Reordonne par effort croissant a risque decroissant (quick wins d'abord):

1. **P0.2 Watcher Linux** (~30min) — fix critique de 5 lignes, zero risque de regression.
2. **P0.3 Guard migrations shared** (~1h) — deplacer une ligne, risque de regression nul.
3. **P2.1 Dedoublonnage infra** (~15min) — supprimer 3 lignes du workflow.
4. **P1.2 Healthcheck et deploy** (~1-2h) — ajouter un cas dans `add()` + recompute.
5. **P2.2 Documentation** (~30min) — search & replace dans 2-3 fichiers.
6. **P1.1 Ordre metadata/shared** (~2-3h) — changement d'ordre + validation.
7. **P0.1 Persistance UI** (~2-4h) — clarification architecturale, cleanup code mort.
8. **P2.3 Tests complementaires** — au fil de l'eau avec chaque fix.

## Decoupage En Lots De Travail

### Lot A - Runtime UI

- Persistance UI
- Watcher Linux

### Lot B - Schema Et Migrations

- Guard shared success-based
- Ordre metadata/shared
- Nettoyage des fallbacks silencieux

### Lot C - Observabilite Et Infra

- Healthcheck repaired
- deploy.sh et workflow
- Documentation agentique

## Risques A Surveiller Pendant La Remediation

- Regressions Streamlit dues aux reruns en chaine.
- Effets de bord multi-utilisateur si le serveur partage encore des etats globaux.
- Tests qui valident la logique pure mais pas le vrai runtime.
- Reintroductions de chemins legacy par documentation obsolete.
- Le double systeme de migrations (inline + runner) amplifie le risque de P0.3 et P1.1 — s'assurer qu'une correction dans un chemin n'est pas contredite par l'autre.

## Definition De Fini

Le plan sera considere termine quand:

1. Le contrat de persistance UI est unique et documente.
2. Le watcher Linux ne peut plus etre multiplie par rerun.
3. Les migrations shared ne sont plus bloquees en faux succes process-level.
4. Les dependances metadata/shared sont explicites et deterministes.
5. Le healthcheck et le deploy distinguent correctement ok, repaired, warning, error.
6. La doc operative des agents est alignee sur v6.
7. Les commandes destructives de deploiement ne s'executent qu'une seule fois.

## Notes De Revue (2026-04-07)

### Severites sureevaluees dans l'annexe

- **N2** (migration prefs non atomique): le `try/except` existant protege deja contre les echecs de copie. Le seul risque residuel est un crash processus entre `copy2` et `unlink` sur un fichier de ~1Ko — negligeable.
- **N6** (parsing deploy trop permissif): le script logue tout dans `$HC_LOG`. C'est un manque de granularite, pas une perte d'information.
- **T3** (retry media minimaliste): fonctionnel (3 tentatives, backoff lineaire). Le vrai probleme est l'observabilite (O3), pas le mecanisme.
- **T4** est redondant avec A2 — pas un finding distinct.

### Lacunes identifiees lors de la revue

- Le double systeme de migrations (inline `_run_shared_migrations` + runner `apply_pending_migrations`) n'etait pas identifie. Ajoute en O7.
- `_bootstrap_shared_matches_db` et 3 sites dans `launcher.py` utilisent des bare connects (anti-pattern liste dans les regles). Ajoute en O6.
- Le deploy n'est pas une duplication mais une **double execution** (le workflow fait fetch/reset/clean puis appelle deploy.sh qui les refait). Corrige dans le constat 6 et P2.1.
- Le fallback `NULL AS ..._fr` existe a **deux** emplacements (L681 pour `mv_player_matches` et L1508 pour `v_match_full`), pas un seul. Corrige dans P1.3.
- PVE migrations n'ont pas de guard process-level (contrairement a shared). Ajoute en O8.
- `_SHARED_MIGRATIONS_DONE` n'est protege par aucun lock (TOCTOU theorique, non exploitable dans l'architecture actuelle). Ajoute en O9.
