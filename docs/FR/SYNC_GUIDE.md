# Guide de synchronisation — LevelUp

Version anglaise : [../SYNC_GUIDE.md](../SYNC_GUIDE.md)

> Comment LevelUp garde vos matchs Halo à jour. Le backend est en Go (`apps/go-api`) ; le front est React/Vite (`apps/web`). La synchronisation est désormais **automatique** — il n'y a plus de `python scripts/sync.py`.

## Vue d'ensemble

La sync tourne **à l'intérieur du serveur Go**. Deux boucles indépendantes maintiennent les données fraîches, toutes deux adossées au même `SyncEngine` et au même pool de tokens :

- **Watcher de présence** (`internal/watcher`) — piloté par les événements. Un démon suit la présence Xbox/Steam de chaque joueur configuré (WebSocket RTA + pollers REST). Quand un joueur termine un match, une sync delta est mise en file pour ce joueur uniquement. Latence faible, quasi temps réel.
- **Scheduler d'auto-sync** (`internal/scheduler/auto_sync.go`) — périodique. À intervalle fixe, il lance une sync delta pour tous les joueurs de `db_profiles.json`, rattrapant ce que le watcher aurait manqué.

Les commandes CLI manuelles (`levelup sync-delta` / `sync-full` / `backfill`) existent pour le bootstrap, le comblement de trous et les recalculs locaux, mais l'usage courant ne nécessite aucune action manuelle.

## Architecture des données (V6)

Les données de matchs sont centralisées dans des bases **partagées** par titre ; les **enrichissements** par joueur restent dans la base du joueur. L'arborescence est title-agnostic sous `data/titles/{slug}/` (slug par défaut `halo_infinite`).

```
API Halo (client compatible SPNKr, Go)
        |
        v
SyncEngine (internal/sync) + Pool de tokens (internal/platform/auth/pool)
        |
        +-- match nouveau -> data/titles/{slug}/warehouse/shared_matches_v2.duckdb
        |     match_registry         (1 ligne par match unique)
        |     match_participants     (tous les joueurs, MMR inclus)
        |     highlight_events       (événements film)
        |     medals_earned          (médailles)
        |     killer_victim_pairs    (paires de kills)
        |     xuid_aliases           (xuid -> gamertag)
        |
        +-- PvE / Firefight -> data/titles/{slug}/warehouse/shared_pve.duckdb
        |
        +-- enrichissement -> data/titles/{slug}/players/{gamertag}/stats.duckdb
              player_match_enrichment (performance_score, session_id, is_with_friends)
              personal_score_awards   (awards objectifs)
              match_skill_rank        (LUSR / CSR par match)
              sync_meta               (état de sync)
```

Schéma complet et justification : [../ARCHITECTURE_V6.md](../ARCHITECTURE_V6.md).

## Synchronisation automatique

### Watcher de présence

Démarré par le serveur au boot (démon `internal/watcher`). Pour chaque joueur il exécute une FSM de présence (WebSocket RTA, fallback Steam/REST) et, à la fin d'un match, met en file une sync delta coordonnée. L'auth est déléguée au pool de tokens partagé. Aucune configuration au-delà de la déclaration du joueur dans `db_profiles.json` avec un token valide (voir Auth).

### Scheduler d'auto-sync

`AutoSyncScheduler` lit `app_settings.json` au boot et à chaque tick. Clés concernées :

| Clé (`app_settings.json`) | Signification |
|---|---|
| `spnkr_auto_sync_enabled` | Interrupteur maître. Doit valoir `true` pour que le scheduler agisse. |
| `spnkr_auto_sync_interval_hours` | Intervalle en heures (défaut 6 si absent). |
| `spnkr_auto_sync_interval_minutes` | Intervalle en minutes (prioritaire si défini). |

À chaque cycle, pour chaque joueur de `db_profiles.json` :
1. Skip si le joueur n'a pas d'entrée dans le pool de tokens, ou si le watcher a déjà une session active pour lui.
2. Construit un `PooledHaloClient` pinné sur ce joueur.
3. Lance `SyncEngine.RunDelta` (fetches internes parallèles). Des cycles répétés à zéro insertion déclenchent un warning (garde-fou : 14 jours de zéro insertion silencieuse en mai 2026).

Le diagnostic est exposé via l'endpoint admin `/api/v1/_diag/auto-sync/snapshot`.

## Pipeline de sync V2

Le moteur par joueur (`RunDelta`/`RunFull`) est le défaut (V1). Un **orchestrateur de cycle V2** opt-in (`internal/sync/v2`) traite *tous* les joueurs par cycle en 6 phases, supprimant la sérialisation sur le writer partagé et garantissant une dédup cross-player correcte :

1. **Discovery** — parallèle par joueur, lecture seule : charge les IDs connus + pagine l'API.
2. **Dedup** — single : union des IDs de matchs inconnus entre joueurs.
3. **FetchShared** — errgroup borné : `GetMatchStats` par match unique.
4. **FetchPlayer** — parallèle par joueur : awards/scores nécessitant son propre token.
5. **Persist** — writer unique : un méga-batch (shared + player) en une transaction.
6. **PostSync** — parallèle par joueur : heals, films, citations, etc.

V2 est l'unique moteur de sync du cycle auto-sync depuis la suppression du pipeline V1 (2026-07). Les titres moteur (Infinite) passent par l'orchestrator ; les titres live-only (Halo 5) par `syncPlayer`→`liveRunner`. Si l'orchestrator n'est pas câblé au boot (prérequis manquants), le cycle bascule sur un filet structurel `syncPlayer`. V2 partage les Persisters, le schéma et le WAL.

## Delta vs Full

- **Delta** — ne récupère que les matchs plus récents que le watermark de la dernière sync. Rapide, défaut du watcher et du scheduler.
- **Full** — parcourt les N derniers matchs API et insère les manquants (comblement de trous). À utiliser après une longue panne, un import, ou un problème de watermark.

Pour chaque match synchronisé, le moteur récupère toujours le payload complet : stats, médailles, personal scores, performance score, highlight events, skill/MMR par match, et aliases xuid -> gamertag.

## CLI manuelle

Construire/lancer la CLI `levelup` depuis `apps/go-api/cmd/levelup` (nécessite la toolchain CGO pour le driver DuckDB — voir [../testing.md](../testing.md)). `LEVELUP_REPO_ROOT` est auto-détecté si absent.

### Sync delta / full

```bash
# Delta pour un joueur
levelup sync-delta --gamertag VotreGamertag [--max-matches 25] [--match-type matchmaking] [--rps 1]

# Delta pour tous les joueurs configurés (via le pool de tokens)
levelup sync-delta --all [--max-matches 25] [--token-pool-size 0]

# Full (comblement) pour un joueur ou tous
levelup sync-full --gamertag VotreGamertag [--max-matches 150] [--match-type matchmaking] [--rps 1]
levelup sync-full --all [--token-pool-size 0]
```

| Flag | Concerne | Défaut | Notes |
|---|---|---|---|
| `--gamertag` | sync-delta, sync-full | — | Mutuellement exclusif avec `--all`. |
| `--all` | sync-delta, sync-full | — | Tous les joueurs de `db_profiles.json` via le pool. |
| `--max-matches` | sync-delta / sync-full | 25 / 150 | Delta : max de nouveaux matchs insérés. Full : matchs API parcourus. |
| `--match-type` | les deux | `matchmaking` | `all` \| `matchmaking` \| `custom` \| `local`. |
| `--rps` | les deux | 1 | Max de requêtes API par seconde. |
| `--token-pool-size` | `--all` uniquement | 0 | 0 = auto (toutes les sources découvertes), `MaxSize` du pool. |

### Backfill (recalculs locaux & backfills API)

```bash
levelup backfill (--gamertag X | --all) <selecteur...> [--force] [--dry-run]
```

Sélecteurs (un ou plusieurs requis) :

| Sélecteur | API Halo requise | Description |
|---|---|---|
| `--engagement-scores` | Non | Backfill du score d'engagement. |
| `--citations` | Non | Recalcul de `match_citations` depuis mappings + médailles + stats + awards. |
| `--citations-recompute-all` | Non | Recalcul total (force) + vérifications d'invariants V1-V4. |
| `--composite-only` | Non | Citations composites uniquement (additif). |
| `--lusr` | Non | Recalcul LUSR (TrueSkill 2 + poids médailles). `--dry-run` prévisualise par playlist_group. |
| `--perf` | Non | Recalcul du performance score relatif (v5). |
| `--assists-model` | Non | Modèle OLS expected_assists par mode. |
| `--csr` | Oui | CSR par match via `GetMatchSkill` (RankRecap), idempotent. |
| `--shared-csr` | Oui (pas d'API avec `--dry-run`) | CSR de tous les participants des matchs ranked dans `shared.match_csrs`. |
| `--weapons` | Oui | `weapon_kills` depuis le CDN film. |
| `--compare-formulas` | Non | Simule 5 variantes de formule LUSR sur `--last-n` matchs (défaut 20). |

`--force` retraite les données déjà persistées. `--dry-run` n'est valide qu'avec `--shared-csr` ou `--lusr`. Les sélecteurs adossés à l'API rafraîchissent les tokens Halo du joueur via le refresh token OAuth (voir Auth). Le recalcul LUSR utilise le chemin v2 canonical ; le v1 est mort.

Le backfill est aussi exposé en HTTP (`POST /backfill/start`) ; la CLI est la voie locale sans serveur.

#### Révisions du décodeur de film et backlog killsource

Le décodeur de film est fait de cinq couches (ADR 0034, D-1), et ses couches révisées portent leur propre révision (D-6) : `source.Rev`, `profile.Rev`, `grammar.Rev` et — depuis le 2026-09-26, une par consommateur de faits — `killsource.Rev` et `objectives.Rev`, chacune dans le `rev.go` de sa couche. C'est `killsource.Rev` qui commande le backlog (elle a repris la valeur de `facts.Rev`, `killsource-2026-09-24` : la scission n'a rouvert aucun backlog).

Une révision est une chaîne figée dans un golden à côté de l'empreinte du périmètre de sa couche : une source qui change sans sa révision fait rougir le test d'empreinte, et une révision qui change sans sa source le fait rougir aussi. L'empreinte hache les **jetons** des sources (commentaires et mise en page ignorés, directives `//go:` comptées) et les fichiers qu'elles embarquent. Le **périmètre** est la fermeture des imports de production de la couche dans le module : une autre couche révisée y entre par sa **valeur**, tout autre paquet par ses sources ; il est figé dans `testdata/<couche>_perimetre.golden`. Le calcul, la fermeture, la chronique, la porte de régénération et les messages d'échec sont partagés (`internal/games/halo_infinite/film/revision`) ; une couche ne déclare que ses exclusions, les valeurs des couches qu'elle importe, son golden, sa porte et la question que son échec pose.

| Révision | Ce qu'elle hache | Elle monte quand |
|---|---|---|
| `source.Rev` | la couche source (chargement, décompression, découpage en chunks et paquets, lecteur de bits canonique) et `film/types` | la façon d'atteindre les octets change — tout est redécodé |
| `profile.Rev` | la couche profil (la table par build et par carte) ; elle n'importe aucune autre couche | une ligne de la table du profil change |
| `grammar.Rev` | l'arbre de la grammaire et les paquets qu'il importe (`film/types`, `games/weapons/filmshell`...), plus les **valeurs** de `source.Rev` et `profile.Rev` | une largeur, un cadre, un ordre de composants, un lecteur neuf |
| `killsource.Rev` | `facts/killsource` et les paquets qu'il importe (`film/damagetag` et ses tables embarquées, `film/types`...), plus les **valeurs** de `source.Rev`, `profile.Rev`, `grammar.Rev` | la sortie du kill-feed peut changer — candidates au backlog |
| `objectives.Rev` | `facts/objectives` et les paquets qu'il importe (`internal/domain`, `film/types`...), plus la **valeur** de `source.Rev` | la sortie des objectifs / du statborg peut changer — pas de backlog, les faits et les calques d'objectifs deviennent périmés |

Chaque couche hache SES sources et les VALEURS des couches qu'elle importe — jamais leurs octets. C'est le seul lien qui ne va pas de soi, et le défaut qu'il ferme est mesuré : une correction de grammaire peut changer la sortie de la source de kill sans toucher un octet de la couche des faits, la révision restait alors immobile, et les lignes déjà écrites portaient la révision courante — exclues du backlog à vie. Chaîner les valeurs rend la règle mécanique : la couche la plus basse qui monte fait monter toutes celles du dessus, jusqu'au backlog. Un faux positif coûte un redécodage ; un faux négatif coûte un parc de lignes fausses.

**Ce qu'est le backlog** (en vigueur, sous `killsource.Rev`, anciennement `facts.Rev`). Chaque ligne de `match_kill_events` porte dans `decoder_rev` la révision qui l'a produite. Un match dont la passe courante — lue par la vue `match_kill_events_latest`, jamais la table brute (ADR 0026) — ne porte pas la révision courante redevient candidat (`conditionBacklog`, `internal/sync/killcollector/postsync.go`). L'étape de post-sync rattrape à cadence bornée (8 films par cycle, budget de cinq minutes) et publie le reste dans l'expvar `killsource_postsync_backlog_restant` ; `levelup backfill-killsource --online` le vide délibérément. Ces bornes valent par cycle et par titre, pas par joueur : le cycle v2 lance le post-sync de chaque joueur dans sa propre goroutine sur le même arriéré global, donc une seule passe tourne à la fois par processus et par titre — un appel concurrent se retire, rend 0 et se compte dans `killsource_postsync_passe_deja_en_cours` (normal dès que plusieurs joueurs se synchronisent ensemble, pas un défaut). La recuisson du parc entier reste un geste à part, pris sur signal de l'utilisateur, jamais par lot. Depuis le schéma 62, cette passe n'est plus forcément un redécodage : `replaybuild.Digest.Verdict` répond `a-jour`, `republier` — le décodage est intact et seule la publication a bougé, donc le document est rejoué depuis les faits persistés par film sous `data/cache/film_facts/`, en centaines de millisecondes au lieu de dizaines de secondes — ou `redecoder`, quand une révision de couche a bougé ou que les faits manquent ; `levelup backfill-replay --only-existing` rapporte les deux comptes.

**Ce qui n'ouvre pas de backlog.** Déplacer un fichier, en scinder un, faire descendre un type dans le paquet de contrat `film/types`, passer une valeur par paramètre au lieu d'une variable de paquet : les empreintes voient les sources, donc elles changent, mais une montée de la RÉVISION n'ouvre le backlog que si la *sortie* peut changer — et un pas structurel se clôt à zéro différence de contenu, prouvée par le corpus gate. Quand un tel pas ne change que la forme, la révision reste en place, le golden est régénéré et le choix est écrit dans le commit : le gate exige qu'il soit explicite. À sa naissance le 2026-09-16, `facts.Rev` **a repris la valeur de `KillSourceDecoderRev`** (`killsource-2026-09-16.2`) au lieu d'ouvrir une nouvelle série, précisément parce que rien du décodage n'avait changé — et la série garde ce préfixe, parce que les lignes en base portent ces chaînes-là. La règle « une montée de `facts.Rev` ouvre le backlog killsource » vaut à partir du premier changement de sortie qui suit.

**Lire les révisions sur un artefact** (né au schéma 61, étendu au 62). Un document de rejeu cuit porte `coverage.decoder.{sourceRev, profileRev, grammarRev, killsourceRev, objectivesRev, build}` (`factsRev` jusqu'au schéma 71) (`GET /players/{player_slug}/matches/{match_id}/replay`) : l'artefact dit sous quelles révisions il a été cuit, au lieu de se deviner à sa version de schéma. `build` est la clé du profil, lue en clair dans `chunk_00` (D-3) ; elle vaut la chaîne vide — le bloc restant présent — aussi bien quand le film n'écrit aucun build que quand la table de profil ne connaît pas celui qu'il a écrit (`ErrUnknownBuild`). L'**absence** du bloc signifie « artefact cuit avant le schéma 61 », jamais « build inconnu ». Le sous-bloc `coverage.decoder.registry` classe l'empreinte du registre ECS du film (`fingerprint`, `status` = `connue` / `presumee` / `inconnue`, `blocks`, `namedSlots`) ; il est absent quand le registre n'a pas été lu. Depuis le **schéma 62**, le document porte aussi `layers`, une révision par calque cuit : la présence d'un calque se lit dans sa révision, jamais dans l'absence d'un champ.

**Un film dont la clé est inconnue est mis de côté** (ADR 0034, D-4 ; en vigueur depuis le lot 3.1.1). Le profil est une table indexée par les clés que le film *écrit* — sa version de format, son build, et sa version majeure quand il n'écrit pas de build. Quand cette clé est absente de la table, ni `sync/killcollector` ni `replaybuild` ne décode le film : aucun fait killsource, aucune position, aucun tir, aucun artefact — et jamais un décodage au profil du build voisin, qui produirait un document plausible et faux. Chaque producteur émet un `WARN` qui nomme la clé refusée, et les compteurs sont ceux que `grammar` nomme : `filmdec_unknown_build_<build>` (`sans_section` pour un film sans section d'identification) et `filmdec_unknown_format_<n>`, plus `killsource_ecartes_cle_inconnue` pour les passes arrêtées. L'outcome killsource est `ecarte-cle-inconnue` et l'enfant de cuisson sort en `filmproc.CodeSkipped` : un **écarté**, jamais un échec. Aucun marqueur de registre n'est posé — `MBitFilmAbsent` est terminal, alors que le film revient de lui-même le jour où la table de profil porte la clé. La procédure pour l'ajouter est [RUNBOOK_FILM_PROFILES.md](../RUNBOOK_FILM_PROFILES.md).

Renvois : [../adr/0034-film-decoder-profile-and-layers.md](../adr/0034-film-decoder-profile-and-layers.md) — D-1 (les cinq couches et leur dépendance à sens unique), D-6 (une révision par chose qui peut changer), D-10 et D-10 bis (la grammaire décide, un repli est nommé, compté et retiré) — et le registre des replis lui-même, `internal/games/halo_infinite/film/internal/facts/fallback`, qui porte pour chaque repli sa condition typée, sa date de pose, sa cible et son critère de retrait.

## Auth

Les tokens proviennent de la source unique décrite dans [../adr/0023-auth-tokens-single-source.md](../adr/0023-auth-tokens-single-source.md) : `data/auth/watcher_tokens/{xuid}.json` via `MultiUserTokenStore`. Le joueur doit d'abord être déclaré dans `db_profiles.json` (avec `xuid`).

- Onboarding normal : flux SSO Xbox web -> `/auth/xbox/callback` persiste le refresh token.
- Onboarding avancé : `go run ./apps/go-api/cmd/token-capture/ <Gamertag>` (device-code) ou `go run ./apps/go-api/cmd/token-import/ <Gamertag>` (RT sur stdin) écrit directement dans le store — aucune édition de `.env.local`.

Les chemins de sync `--all` et les backfills adossés à l'API résolvent les tokens via le pool (Discovery -> Resolver -> Pool), dont le Discovery scanne le token store (`data/auth/watcher_tokens/{xuid}.json`) ; le pool gère le refresh OAuth, réécrit le refresh token rotaté dans ce store, et cache les Spartan tokens (~3h30). Les commandes mono-joueur `levelup sync-delta/sync-full --gamertag` et les backfills `--csr/--weapons` lisent le même store directement. Depuis l'ADR 0023 Phase 5 (2026-08-25), ce store est la SEULE source : ni variable d'environnement, ni `sync_meta`. Ne jamais re-capturer un token pour corriger un 401 : une sync verte signifie que les tokens sont bons.

## Écritures append-only / ART-safe

Toutes les écritures par match passent par l'architecture Collect -> Persist (un batch INSERT-only par cycle), et les tables d'état critiques sont en append-only. Cela éradique par construction le bug de corruption d'index ART de DuckDB. Ne pas réintroduire d'`UPDATE` concurrent ni d'`INSERT ... ON CONFLICT DO UPDATE` sur les tables shared/état. Références :

- [../adr/0019-collect-persist-architecture.md](../adr/0019-collect-persist-architecture.md)
- [../adr/0026-append-only-art-eradication.md](../adr/0026-append-only-art-eradication.md)

### Post-sync LUSR v2 : rien de nouveau, aucun écrivain

L'étape LUSR v2 du post-sync (`RunLUSRV2ShadowOwnerOnly`, `internal/sync/skill`) décide de ce qui est nouveau sur un **lecteur**, avant de demander l'écrivain partagé. Sous le segment de lecture, elle charge les matchs candidats du joueur, le filigrane (watermark) de chaque groupe de modes (`last_match_at` dans `player_skill_state_v2_latest`) et, pour les candidats situés au-dessus, le prédicat d'éligibilité partagé (deux équipes humaines, équilibre, issue notable). Quand rien au-dessus du filigrane n'est notable, le cycle ne prend **aucun écrivain** et écrit une ligne INFO par joueur, `lusr_v2: rien de nouveau`, avec `candidates`, `new` = 0 et les compteurs de candidats écartés. Sinon il prend l'écrivain en **rafales bornées** pour tous les candidats situés au-dessus du filigrane : une rafale rend l'écrivain après 50 matchs, ou dès qu'elle l'a tenu 2 s (le seuil du chien de garde du provider partagé), et le reste de la file reprend sous une nouvelle rafale dans le même cycle. Sous chaque rafale, chaque match repasse tous ses contrôles sur le handle en écriture (groupe tenu, filigrane, éligibilité) ; un groupe tenu par une écriture canonique en échec le reste d'une rafale à l'autre, jusqu'à la fin du cycle. Chaque rafale écrit une ligne INFO, `lusr_v2: rafale bornée` (`candidates`, `new`, `bursts`, `matches`, `held_ms`, `remaining`), et le cycle écrit `lusr_v2: rafale terminée` (`candidates`, `new`, `bursts`, `processed`, compteurs de candidats écartés). Ce qui est écrit ne change pas : des INSERT seuls, des lectures par les vues `_latest`, les mêmes lignes dans le même ordre. Avant ce changement (mesure du 2026-09-23), l'étape prenait l'écrivain tous les trois candidats et testait le « déjà traité » à l'intérieur de la rafale : 1 233 bascules lecture seule / écriture en moins de deux minutes pour zéro ligne écrite, chacune drainant les lecteurs HTTP et vidant le cache DuckDB.

## Runbook ops (verrou DuckDB cross-process)

DuckDB ne partage pas un file-lock OS entre processus distincts. Lancer un outil CLI qui ouvre une base **partagée** (metadata, shared_matches_v2, shared_pve, shared_social) en RW pendant que le serveur tient son handle échouera avec `IO Error: Cannot open file ... utilisé par un autre processus`.

Règle : ne pas lancer `levelup sync-* / backfill` (ni les autres outils CLI sur DBs partagées) contre des bases partagées tant que le serveur (`apps/go-api/server.exe` ou `air`) tourne. Arrêter le serveur d'abord pour toute écriture partagée cross-process. Procédure complète et inventaire des outils : [../RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md](../RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md).

## Dépannage

| Symptôme | Action |
|---|---|
| Auto-sync inactive | Vérifier `spnkr_auto_sync_enabled: true` dans `app_settings.json` ; inspecter `/api/v1/_diag/auto-sync/snapshot`. |
| Joueur skippé (`not_in_pool`) | Aucun token découvert pour ce joueur — onboarder via SSO ou `token-capture`/`token-import`. |
| Zéro insertion répétée | Surveiller le warning du scheduler ; vérifier que l'appel `/matches` utilise `xuid(NNN)` et non le gamertag brut, et que le watermark est sain. |
| 401 sur un backfill API | Tokens périmés en cache ; **ne pas** re-capturer. Laisser le pool rafraîchir. Voir [../adr/0023](../adr/0023-auth-tokens-single-source.md). |
| `Cannot open file ... utilisé par un autre processus` | Verrou DuckDB cross-process — arrêter le serveur avant la CLI (voir Runbook ops). |
