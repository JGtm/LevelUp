# Essai de régénération sur la copie du worktree (2026-11-24)

## Où l'essai a écrit (vérifié)

- Racine : worktree `LevelUp-wt-dec160` (branche `chore/830-decodeur-v160`). Aucun `app_settings.json`, `db_profiles.json` ni `.env.local`
  dans ce worktree (donc aucun chemin absolu pointant ailleurs) ; aucune variable `LEVELUP_*` dans l'environnement ; lancement avec
  `LEVELUP_REPO_ROOT` = le worktree, `GOWORK=off`.
- Preuve d'exécution : la commande `backfill-replay --dry-run` a échoué en nommant le chemin résolu de la base partagée,
  `...\LevelUp-wt-dec160\data\titles\halo_infinite\warehouse\shared_matches_v2.duckdb` (inexistante : la copie n'a que
  `metadata-prebuilt.zip`). Verrou de décodage : `...\LevelUp-wt-dec160\data\cache\film_decode.lock.json`. Faits de film et rejeux :
  sous `...\LevelUp-wt-dec160\data\cache\`.
- Binaires : `bin/levelup.exe` (branche, décodeur v1.6.1, schéma 127) et `bin/levelup-avant.exe` (extrait par `git archive` de
  `origin/release/v8.3.0`, décodeur v1.4.1, schéma 125), construits avec un cache Go dédié.

## Contrainte de la copie : pas de base partagée

La copie n'a aucune base de matchs. `backfill-replay` sans `--one` exige `match_registry` et refuse donc de partir. L'essai passe par la
forme `--one <short8> --map-name <carte>` (la forme interne que le parent utilise pour chaque film), avec la carte prise dans le
manifeste d'équivalence du décodeur. Conséquence : les artefacts sont cuits SANS faits de match (pas de compteurs de joueur) ; leur
poids est donc inférieur à celui d'un vrai rejeu, mais la comparaison avant / après est faite dans les mêmes conditions.
Les passes dérivées qui lisent la base n'ont pas pu être essayées (aucune base dans la copie, et je n'avais pas le droit de copier
celle du dossier réel).

## Cinq films, un à la fois, avant (v1.4.1) puis après (v1.6.1) sur le même film

Chaque passe a redécodé le film (le journal dit « faits cuits sous d'autres gardes — redécodage », `depuis_les_faits=false`
partout). Durée = mur, processus compris. Pic = mémoire de travail maximale du processus de cuisson.

| Film (carte, version du jeu) | Poids du film | Durée avant | Durée après | Pic avant | Pic après | Rejeu avant (gz) | Rejeu après (gz) | Rapport | Schéma après |
|---|---|---|---|---|---|---|---|---|---|
| `51101d1d` (Fortress, HI_1_13_0, version actuelle, le plus petit) | 9,5 Mo | 1,4 s | 1,1 s | 141 Mo | 146 Mo | 330 674 o | 490 541 o | x 1,48 | 127 |
| `9f57c612` (Curfew, HI_1_13_0, version actuelle) | 20,7 Mo | 2,4 s | 2,2 s | 329 Mo | 345 Mo | 836 152 o | 1 240 674 o | x 1,48 | 127 |
| `50247b26` (Oasis, version 31, la plus ancienne) | 45,1 Mo | 23,2 s | 10,5 s | 1 293 Mo | 1 030 Mo | 3 234 725 o | 3 940 435 o | x 1,22 | 127 |
| `084a804d` (Fortitude Heavies, HI_1_10_0) | 61,8 Mo | 17,1 s | 15,5 s | 1 395 Mo | 1 317 Mo | 7 493 750 o | 9 708 476 o | x 1,30 | 127 |
| `1c4c63c2` (Refuge, HI_1_10_0, le plus gros du parc) | 92,3 Mo | 32,8 s | 24,6 s | 2 735 Mo | 2 663 Mo | 7 775 257 o | 11 421 875 o | x 1,47 | 127 |
| Total | 229,4 Mo | 76,9 s | 53,9 s | | | 19 670 558 o | 26 802 001 o | x 1,36 | |

- Schéma 127 : vérifié dans le JSON de chaque rejeu « après » (`schemaVersion` = 127) ET dans son `.meta` ; les rejeux « avant » portent 125.
- Pic : donné par le film lui-même (`__levelup_pic_octets__`) : le plus gros film monte à 2,66 Go pour un plafond souple de 3,22 Go.
- Poids : rejeu non compressé « après » du plus gros film 63,3 Mo contre 41,8 Mo.
- Vitesse : x 1,4 sur le total (x 2,2 sur le film le plus ancien, presque rien sur les petits). 0,235 s par Mo de film après, 0,335 avant.

## Autres mesures

- 1 fil contre 4 fils sur `084a804d` (pris juste après une passe « avant » pour forcer le redécodage) : 12,8 s (1 fil, via
  `LEVELUP_ENV=production`, profil `hosted`) contre 11,9 s (4 fils, profil `dev`). Écart faible ; le passage effectif à 1 fil n'est
  pas prouvé par un journal (déduit du code).
- `tactical-rasters --backfill` (aucune base requise) : cinq films, 2,3 s, 5 écrits, 0 échec, pic 467 Mo ; deuxième passe
  immédiate : 0 écrit, 5 déjà à jour (0,2 s). `--dry-run` : 5 à écrire, 0 écrit.
- Passes dérivées avec base (`backfill-usage-summary`, `backfill-pad-tiers`, `backfill-killsource`, `backfill-vehicle-takes`,
  `backfill-flag-grabs-net`, `backfill-kickoff`) : NON lancées, pas de base dans la copie. Durées reprises du précédent (25 min).

## Extrapolation au parc réel (lecture seule du dossier réel)

- Parc réel : 1 455 rejeux (schéma de départ non relu), 1 455 films avec rejeu, 31,2 Go de films (moyenne 21,4 Mo, le plus gros 92,3 Mo),
  1 552 faits de film (7,55 Go), 163 Mo de rasters, 1,98 Go de `.json.gz`, 483 Mo de base partagée.
- Régénération : 31,2 Go x 0,235 s/Mo = environ 7 330 s = 2 h 02 ; avec le décodeur d'avant, 0,335 s/Mo donnerait 2 h 54 (le
  précédent a mesuré 2 h 26 avec un décodeur intermédiaire : cohérent). Les cinq films sont plus gros que la moyenne (46 Mo contre
  21 Mo) : le surcoût fixe par film (démarrage, lecture en base des faits de match) pèse plus sur le parc que sur l'essai. Fourchette
  retenue : 2 h à 2 h 30.
- Poids des rejeux après : 1,98 Go x 1,36 (essai) à x 1,45 (plan) = 2,7 à 2,9 Go, soit environ +0,8 à +0,9 Go.
