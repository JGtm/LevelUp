# Procédure de régénération du parc, décodeur v1.6.1 (schéma 127)

Rédigée le 2026-11-24 à partir du code de la branche `chore/830-decodeur-v160`, du plan (item L.6), de la marche
précédente (8.2.1, schéma 124) et d'un essai sur la copie réduite du worktree (voir `essai.md`).
Statut des affirmations : « vérifié » = relu dans le code ou mesuré pendant l'essai ; « déduit » = raisonné, la
façon de le vérifier est dite.

## Ce que la procédure fait, en dix lignes

1. Avant : serveur arrêté, aucun processus LevelUp, sauvegarde de la base partagée et des rejeux (2,6 Go), 10 Go libres.
2. `backfill-replay --dry-run --only-existing`, puis sans `--dry-run` : régénère les 1 455 rejeux (un film à la fois), environ 2 h à 2 h 30.
3. `backfill-usage-summary`.
4. `backfill-pad-tiers --force`.
5. `backfill-killsource --workers 1` (décode les films : la plus longue des passes dérivées).
6. `backfill-vehicle-takes --force`.
7. `backfill-flag-grabs-net --force`.
8. `tactical-rasters --backfill` (environ 7 min).
9. `backfill-kickoff --dry-run`, puis `backfill-kickoff`.
10. Durée totale estimée : environ 3 h (2 h 30 de régénération + environ 30 min de dérivés). Espace : environ +1 Go sur le dossier
    des rejeux, +1,5 Go de marge pour les faits de film, 2,6 Go de sauvegarde : prévoir 10 Go libres (125 Go libres aujourd'hui).

## 0. Conditions communes (vérifié, sauf mention)

- Dossier principal à jour : branche portant le décodeur v1.6.1 (`apps/go-api/go.mod` exige `film-decoder v1.6.1`).
- `GOWORK=off`, cache Go dédié (`GOCACHE=%LOCALAPPDATA%\go-build-<lot>`).
- Compiler UNE fois le binaire, ne pas passer par `go run` : `cd apps/go-api && go build -o <dossier-hors-data>/levelup.exe ./cmd/levelup`.
- Lancer depuis la racine du dépôt, avec `LEVELUP_REPO_ROOT=<racine>` posé EXPLICITEMENT et AUCUNE autre variable `LEVELUP_*`
  (en particulier ni `LEVELUP_DATA_DIR`, ni `LEVELUP_LEGACY_FILM_CACHE_DIR`). Vérifier avant : `env | grep LEVELUP`.
  Sans `LEVELUP_REPO_ROOT`, la racine se devine depuis le dossier courant (`db_profiles.example.json` en remontant) : même résultat
  depuis la racine, mais l'explicite évite une surprise.
- Chemins résolus (vérifié dans le worktree d'essai : tout tombe sous la racine) : bases `data/titles/halo_infinite/warehouse/`,
  archive des films et rejeux `data/cache/` (`film_chunks/`, `film_manifests/`, `film_facts/`, `replays/halo_infinite/`),
  verrou de décodage `data/cache/film_decode.lock.json`, journaux `logs/`. ATTENTION côté dossier principal : `app_settings.json` peut
  porter `film_archive_dir` (chemin absolu, lu par `config.ResolveRoots`) ; s'il est posé, l'archive des films n'est plus sous
  `data/cache` et `--cache` doit être cohérent. À lire avant de lancer (déduit : je n'ai pas ouvert ce fichier du dossier réel).
- Serveur arrêté et aucun processus : `tasklist | grep -i -E "levelup|server|duckdb"` vide (le 2026-11-24 : vide). Arrêt propre =
  aucun `.wal` à côté de `shared_matches_v2.duckdb` (état au 2026-11-24 : aucun `.wal` de la base partagée).
- Nombre de fils du décodeur : 4. Vérifié dans le code : `decoderThreadsOf` rend 4 pour les profils `dev` et `desktop`, 1 pour
  `hosted` et `demo` ; le profil d'une commande lancée à la main est `dev` tant que `LEVELUP_ENV` n'est PAS déclaré (`hosted`
  sinon, donc 1 fil). Aucun journal n'imprime ce nombre : déduit pour le poste, à confirmer en s'assurant que `LEVELUP_ENV` est absent
  de l'environnement ET de `.env.local` (chercher la clé seulement). Mesure : 1 fil ou 4 fils ne change presque rien sur UN film
  (12,8 s contre 11,9 s sur un film moyen, essai ; mais le passage en `hosted` par `LEVELUP_ENV=production` n'est pas prouvé
  par un journal : mesure indicative). Un seul film est décodé à la fois (verrou `filmrun.AcquireSolo`, inter-processus).
- Un film à la fois : vérifié. `backfill-replay` lance un processus enfant par film, séquentiel, sous le verrou solo ; le plafond
  mémoire de chaque enfant est 3 Gio souple, 4 Gio dur (`--mem-limit-gib`, défaut 3).
- Mémoire attendue : pic d'un enfant 0,15 Go (petit film) à 2,66 Go (le plus gros film, 92 Mo) mesuré SANS les faits de match de la
  base (la copie n'en a pas). Marge de 17 % sous le plafond souple : si le plus gros film meurt « mémoire » (compté à part, la passe
  continue), relancer `backfill-replay --match <id> --mem-limit-gib 6`. Les passes dérivées ne dépassent pas 0,5 Go
  (`killsource` : pic 422 Mio sur le pire film, mesure de la passe).
- Pas pendant que l'utilisateur joue ni pendant qu'un autre agent écrit dans `data/` (mémoire : jamais de cuisson en lot sans le
  prévenir ; le go d'avance du 2026-11-18, rappelé le 2026-11-22, couvre cette passe, à rappeler au lancement).

## 1. Sauvegarde AVANT (serveur arrêté, copie simple, hors de `data/`)

| Quoi | Taille mesurée | Pourquoi |
|---|---|---|
| `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` | 483 Mo | toutes les passes dérivées et le rattrapage du coup d'envoi y écrivent ; un seul fichier, pas de `.wal` |
| `data/cache/replays/halo_infinite/` (`.json.gz`, `.meta`, `rasters/`) | 2,1 Go (1 455 rejeux : 1,98 Go ; 1 455 rasters : 163 Mo ; `.meta` : 3 Mo) | la régénération remplace chaque fichier ; l'ancien n'existe plus après (seule copie : celle-ci, plus la sauvegarde disque habituelle) |
| `data/titles/halo_infinite/warehouse/shared_social.duckdb`, `metadata.duckdb`, `shared_pve.duckdb` | non touchés | aucune des commandes ci-dessous n'y écrit (déduit de la lecture des commandes) |
| `data/cache/film_facts/` | 7,55 Go, 1 552 fichiers | réécrit film par film par la cuisson ; se refait des films, donc pas de sauvegarde obligatoire. À copier seulement si l'on veut pouvoir revenir sans refaire 2 h |

Commande type (une copie, jamais un lien ni une jonction) : `cp <fichier> <dossier-sauvegarde>/` et `cp -r` pour les rejeux.
Contrôle : tailles identiques (`stat -c %s`). Retour arrière : recopier les deux éléments, serveur arrêté.

## 2. Régénération des rejeux : `backfill-replay`

```
levelup backfill-replay --dry-run --only-existing    # plan : « films a construire : N » (attendu environ 1 455), rien d'écrit
levelup backfill-replay --only-existing              # la régénération
```

- Lit : `match_registry` (lecture courte, au démarrage du parent, puis une lecture par enfant pour les faits de match, relâchée avant
  de décoder) ; `data/cache/film_chunks/`, `film_manifests/`. Aucun réseau, aucun jeton (vérifié dans l'en-tête de la commande).
- Écrit : `data/cache/replays/halo_infinite/<short8>.json.gz` et `.meta` (atomique : un enfant tué ne laisse pas de fichier
  à moitié écrit), `data/cache/film_facts/halo_infinite/<short8>.filmfacts.bin` (réécrit), `film_decode.lock.json`. Aucune base
  écrite (vérifié : « la seule base touchée est le registre partagé, en lecture courte »).
- Serveur : doit être arrêté (la lecture passe par `OpenReadForQuery` ; tenue en écriture elle dégrade ou échoue).
- Reprenable : oui. La clé est la version de schéma de l'artefact (`replay.SchemaVersion`) ; un artefact au schéma 127 est sauté
  (compté « déjà à jour »). Relancer la même commande après une coupure reprend sans rien refaire de ce qui est fini.
- Avancement : une ligne par film sur la sortie standard (`<id> : N tracks, N octets (<module>)`) ; compter les `.meta` au 127
  (voir contrôle). Le plan de passe trie par coût croissant : les gros films sont à la fin (les 20 dernières minutes sont les
  plus lentes).
- Durée attendue : 2 h à 2 h 30. Base du calcul : essai 0,235 s par Mo de film (53,9 s pour 229 Mo, cinq films) contre 0,335 s
  avec le décodeur d'avant ; parc réel 1 455 films = 31,2 Go (moyenne 21,4 Mo) donc 2 h 02 ; plus le surcoût par film (lecture
  des faits de match en base, démarrage du processus) que l'essai ne voit pas ; précédent 8.2.1 : 8 750 s (2 h 26) avec le décodeur
  d'alors.
- Bilan de fin à relever : « N construits », échecs ventilés (carte hors catalogue, mémoire, mort subite). Attendu comme au
  précédent : 0 erreur, 0 mort mémoire, 0 mort subite ; 801 hors registre, 2 Firefight, 6 sans artefact (chiffres du 2026-11-16,
  à comparer sans les exiger identiques).
- Contrôle de fin : `grep -l '"schemaVersion":127' data/cache/replays/halo_infinite/*.meta | wc -l` = 1 455 ; nombre de
  `.json.gz` = 1 455 ; relancer `backfill-replay --dry-run --only-existing` doit rendre « films a construire : 0 ».
- Poids après : 2,1 Go devient environ 3,0 Go (rejeux 1,98 Go x 1,36 à 1,45 : 2,7 à 2,9 Go ; essai : +36 % sur cinq films sans faits
  de match, le plan dit +45 % avec).

## 3. Passes dérivées (toutes : serveur arrêté, une base écrite : `shared_matches_v2.duckdb`)

Toutes ouvrent la base partagée en écriture (verrou de processus unique, ADR 0013), y compris `--dry-run` (migrations jouées),
sauf mention. Aucune ne décode de film sauf `backfill-killsource`. Reprise : relancer la même commande.

| # | Commande | Lit | Écrit | Reprenable, clé | Durée (précédent 8.2.1) | Contrôle de fin |
|---|---|---|---|---|---|---|
| 3 | `backfill-usage-summary` (sans `--force` : le schéma 127 des rejeux suffit) | rejeux `.json.gz` un à un | `match_usage_players`, `match_usage_films` (append-only) | oui, (summary_rev, levelup_rev, schéma de l'artefact) via la vue `_latest` | quelques minutes (1 455 écrits) | bilan « 1 455 écrits » ; relancer : tout « à jour » |
| 4 | `backfill-pad-tiers --force` | rejeux | `match_pad_pickups_by_tier` (append-only) | oui, présence en base ; `--force` obligatoire (anciennes lignes) | 1 451 | bilan imprimé (socles confirmés) |
| 5 | `backfill-killsource --workers 1` | films en cache (décode), `highlight_events` (passe crédit) | `match_kill_events`, `match_weapon_shots`, faits d'isolement, résumés par match (append-only) | oui, `decoder_rev` : un match dont toutes les passes courantes portent la révision courante est sauté ; `--force` redécode tout | la plus longue des dérivées (1 460 matchs, 146 249 morts au précédent) | `--status` (autre terminal, ne lit aucune base) ; jauge « sans tête » = 0 (ADR 0039) ; relancer : tout sauté |
| 6 | `backfill-vehicle-takes --force` | rejeux + `match_kill_events_latest` (frags d'engin, donc APRÈS killsource) | `match_vehicle_takes` | oui, présence ; `--force` obligatoire après recuisson | 1 456 | bilan imprimé |
| 7 | `backfill-flag-grabs-net --force` | calque `flagCarries` des rejeux | `match_flag_grabs_net` | oui, présence ; `--force` si la fenêtre change | 186 écrits, 1 270 sans calque de drapeau | bilan imprimé, fenêtre appliquée |
| 8 | `tactical-rasters --backfill` | rejeux ; AUCUNE base ouverte (peut tourner serveur allumé) | `data/cache/replays/halo_infinite/rasters/<short8>.json.gz` | oui, idempotent : une 2e passe écrit 0 fichier | mesuré : 5 films en 2,3 s ; parc estimé 7 min | « N écrits, 0 échec » puis 2e passe « 0 écrits, N déjà à jour » |
| 9 | `backfill-kickoff --dry-run` puis `backfill-kickoff` | artefacts rangés (coup d'envoi source `film`) | coup d'envoi, source, temps de jeu, puis révision `kickoff_revision` dans `sync_meta` | oui, idempotent ; `--dry-run` en lecture seule, sans migration | minutes | la révision est rangée : le démarrage de la 8.3.0 n'a plus rien à rattraper |

Durée cumulée des dérivées : 25 min au précédent (sans `kickoff`, qui est neuf). Prévoir 30 à 40 min.
Le plan inscrit `backfill-killsource --workers 1` : un seul fil de décodage, car un seul processus tient la base ; par défaut c'est 3.
Autres passes présentes dans l'outil mais NON listées au plan L.6 (donc non lancées ici) : `backfill-bomb-stats`, `backfill-lead-share`,
`backfill-match-summaries` ; à statuer par l'utilisateur si une sortie du décodeur les touche.

## 4. Après

- Dossier d'attente et bascule du VPS : étape L.8 du plan, hors de cette procédure ; seuls les rejeux se copient, jamais une base.
- Mesures APRÈS (L.7) : compter schéma 127 (ci-dessus), poids du dossier des rejeux (`du -sh`), durée relevée à la ligne du bilan.
- Nettoyage : cache Go dédié (`powershell -File scripts/disk-hygiene.ps1 -Lot <lot> -Apply`).

## 5. Ce qui rend la procédure risquée (à lire avant de lancer)

- Le plus gros film mesure 2,66 Go de pic pour un plafond souple de 3,22 Go (3 Gio) : 17 % de marge, et la mesure est faite sans les
  faits de match de la base. Prévoir `--mem-limit-gib 6` en reprise ciblée.
- `backfill-replay` réécrit 1 552 faits de film (7,55 Go) en plus des rejeux : la passe est plus lourde en écriture disque que les
  2,1 Go de rejeux ne le laissent croire.
- Aucune des passes dérivées n'a pu être essayée dans la copie (elle n'a pas de base partagée) : leur durée est reprise du
  précédent, pas mesurée.
- Un `film_archive_dir` posé dans `app_settings.json` du dossier principal déplace l'archive : à relire avant de lancer.
