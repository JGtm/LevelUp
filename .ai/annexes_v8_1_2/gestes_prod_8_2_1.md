# Gestes de production, LevelUp 8.2.1 : rejeux au schéma 124 (2026-11-16)

Pendant de G5 à G7 de `gestes_prod_8_2_0.md` pour la montée du décodeur v1.3.9 (schéma 124). Les autres gestes
de la 8.2.1 relèvent de la session pilote de la release.

## R1. Régénération du parc, en LOCAL (FAIT le 2026-11-16, accord du user du 2026-11-16 matin)

- Binaire de `release/v8.2.1` à `e657d05bc8` (décodeur v1.3.9, schéma 124, 33 ms), worktree détaché sans jonction,
  serveur local arrêté, `backfill-replay --only-existing`, un film à la fois.
- Rapport : 1 455 construits (tous redécodés, révisions de grammaire et d'objectifs neuves), 0 erreur de décodage,
  0 mort mémoire, 0 mort subite ; 801 hors registre, 2 Firefight, 6 sans artefact. 2 h 26 (8 750 s).
- Comptes avant -> après : films 1 614 -> 1 614 ; manifestes 2 267 -> 2 267 ; `.json.gz` 1 455 -> 1 455 (tous au 124) ;
  faits de film 1 458 -> 1 458.
- Dérivés en local, même ordre que le G7 de la 8.2.0, 0 échec partout (25 min) : usage-summary 1 455 écrits (us7,
  schéma 124) ; pad-tiers --force 1 451 ; killsource --workers 1 : 1 460 matchs, 146 249 morts ; vehicle-takes --force
  1 456 ; flag-grabs-net --force 186 (1 270 sans calque de drapeau) ; tactical-rasters --backfill 1 456 en v7.

## R2. Mise en place sur le VPS (dossier d'attente PRÊT, bascule NON appliquée)

- `/opt/levelup/data/replays-staging-v8.2.1/halo_infinite/` : 1 455 `.json.gz` + 1 455 `.meta` (schéma 124), 1,9 Go,
  `deploy:deploy`, 1 965 183 987 octets identiques au local. Ni `.derived.json`, ni `rasters/`, ni le rejeu brut d'un
  joueur tiers, ni le `.meta` orphelin local, aucune base.
- État du dossier servi au 2026-11-16 (prod 8.2.0, `248884e99`) : 3 537 `.json.gz`, 3 537 `.meta`, 332 `.derived.json`,
  0 `.json` brut.
- **Quand** : après le déploiement de la 8.2.1 et la fin du premier démarrage. Une 8.2.0 ne connaît pas le schéma 124 :
  jamais avant.

```bash
cd /opt/levelup
bash data/replays-staging-v8.2.1/BASCULE.sh            # simulation : ne touche à rien
# mesuré le 2026-11-16 : « 1455 rejeux copiés (1455 remplacent un rejeu servi, 0 nouveaux), 0 anciens .json bruts retirés, 0 erreurs »
bash data/replays-staging-v8.2.1/BASCULE.sh --apply    # copie par temporaire + renommage, contrôle de taille
```

- Le script remplace, pour chaque match du dossier d'attente, le `.json.gz` puis le `.meta` du même match (temporaire puis
  renommage, une erreur de taille est comptée), et retire un ancien `.json` brut du même match s'il en restait un (aucun
  aujourd'hui). Les rejeux servis sans équivalent régénéré (environ 2 080, sans film pour les refaire) ne sont pas touchés.
  Sa dernière ligne compte les `.meta` au schéma 124 du dossier servi : 1 455 attendus après `--apply`.
- Retour arrière : aucun pour les `.json.gz` remplacés (restic). Le dossier d'attente de la 8.2.0 (1,9 Go, schéma 116)
  est toujours sur le VPS : sa suppression suit le contrôle écrit du G7 de la 8.2.0.

## R3. Dérivés sur le serveur (serveur arrêté, une fenêtre, après R2)

```bash
cd /opt/levelup && docker compose stop levelup
LU() { docker compose run --rm --no-deps levelup levelup "$@"; }
LU backfill-usage-summary                 # sans --force : le schéma 124 des rejeux suffit à re-résumer
LU backfill-pad-tiers --force
LU backfill-vehicle-takes --force         # obligatoire après une régénération
LU backfill-flag-grabs-net --force
LU tactical-rasters --backfill            # v7, réécrit les calques des rejeux remplacés
```

- `backfill-killsource` : sans objet en prod (aucun film), fait en local.
- Contrôle après la passe : `grep -l '"schemaVersion":124' /opt/levelup/data/cache/replays/halo_infinite/*.meta | wc -l`
  égal à 1 455, et un rejeu régénéré ouvert à l'écran en lecture. Ensuite seulement :
  `rm -rf /opt/levelup/data/replays-staging-v8.2.1`.
