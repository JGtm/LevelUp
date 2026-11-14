# Gestes de production, LevelUp 8.2.0 (BROUILLON du 2026-11-12, mis à jour le 2026-11-13 ; G5 à G7 le 2026-11-14)

Base lue : `origin/release/v8.2.0` à `31f91a79e8` (PR #181 à #206 fusionnées). Sources : plan
`.ai/PLAN_V8_1_2_2026-11-09.md` (7.5), `docs/RUNBOOK_DEPLOY_CHECKLIST.md` (conventions `LU`, fenêtres,
R7 et R8), `docs/COMMANDS.md`, journal `.ai/thought_log.md`, code de `cmd/levelup` et de `internal/`.

Marque **À CONFIRMER** : dépend d'une PR encore ouverte ou d'un point non prouvé sur pièces.
Une seule PR reste en attente : la **#207** (cuisson à 30 images/s, décodeur v1.3.6, `SchemaVersion` 115 ; branche
`feat/rejeu-33ms-et-destructions`), suspendue au verdict de cinq lignes « non établi » du gate du corpus. Sont
fusionnées depuis le premier brouillon : #203 (décodeur v1.3.4, destructions de véhicule, `SchemaVersion` 113),
#205 (fonds de carte par le maillage de navigation), #206 (alertes de sécurité des dépendances, aucun geste) et
les étapes 4 à 8 du plan d'architecture (aucun geste de production). Sans la #207, la release part au schéma 113
et la cuisson reste à 100 ms (10 images/s).

## Règles qui valent pour toute la liste

- **JAMAIS de copie d'une base locale vers la prod.** La base locale n'est jamais alignée sur la prod. Seuls
  les rejeux se copient (dossier `data/cache/replays/<titre>/`) ; la seule suppression côté VPS est celle des
  anciens `.json` bruts remplacés par leur `.json.gz` régénéré (G6). Les dérivés se
  réécrivent sur le serveur par les passes `backfill-*`, qui lisent les rejeux copiés. `backfill-killsource`
  reste local : aucune base n'est copiée (D-17 du plan).
- Aucune cuisson en lot sans ton accord explicite, une fenêtre à part, un film à la fois.
- Push sur `main` = déploiement automatique : tu es prévenu avant, et c'est ton geste.
- `LU` = `docker compose run --rm --no-deps levelup levelup` (depuis `/opt/levelup`). Toute passe qui écrit
  la base partagée exige le serveur arrêté (`docker compose stop levelup`), `--dry-run` compris pour celles
  qui jouent les migrations.

## Avant les gestes (pré-requis, pas des gestes de production)

| # | Contrôle | Commande ou preuve | Durée | Risque |
|---|---|---|---|---|
| P1 | CI verte sur `release/v8.2.0`, dont `desktop-e2e` | page Actions | selon la CI | un déploiement lancé sur CI rouge est refusé par `attente-ci` |
| P2 | Montée en 8.2.0 faite à la clôture : `apps/desktop/src-tauri/Cargo.toml`, `Cargo.lock`, `tauri.conf.json` ; blocs de notes `**v8.2 — ...**` posés en tête de `docs/RELEASE_NOTES.md` et `docs/FR/RELEASE_NOTES.md` ; `## [8.2.0]` dans les deux changelogs | brouillons : `brouillon_versions_8_2_0_FR.md`, `brouillon_versions_8_2_0_EN.md` (même dossier) | 15 min | `release.yml` refuse le tag si la première section des notes ne porte pas `v8.2` ; deux tests Go de cohérence notes / changelog |
| P3 | Aucun `backfill` hôte en cours | `pgrep -f '[b]ackfill'` vide | secondes | le job `deploy-demo` saute la régénération si un backfill tient le verrou |
| P4 | Sauvegarde restic fraîche | `scripts/RESTIC_BACKUP.md` | quelques minutes | sans elle, pas de retour arrière si une base est touchée |
| P5 | Purge des `film_chunks` du VPS vérifiée | liste du dossier d'archive | secondes | un reste de chunks fait décoder le VPS lors d'une recuisson |
| P6 | Fenêtre annoncée | message | — | — |

## Gestes, dans l'ordre

| # | Geste | Serveur | Durée estimée | Risque |
|---|---|---|---|---|
| G1 | Fusion squash vers `main` et tag `v8.2.0` | déploiement auto | CI 15 à 25 min puis déploiement | moyen |
| G2 | Migration `shared_match_lead_share_v1` | au boot, automatique | secondes | faible |
| G3 | Journaux du serveur en JSON, à vérifier | actif | 2 min | faible |
| G4 | Recalcul LUSR et note de performance au premier démarrage | actif, automatique | non mesuré en prod ; 47 s pour un joueur de 1 108 matchs (mesure du runbook) | moyen, visible |
| G5 | Recuisson des rejeux, en LOCAL | serveur local arrêté | FAIT le 2026-11-14 (1 h 36) | — |
| G6 | Mise en place des rejeux régénérés (dossier d'attente déjà sur le VPS, `BASCULE.sh`) | actif | quelques minutes | moyen |
| G7 | Passes de dérivés sur le serveur, dont `backfill-lead-share` | arrêté | lead-share environ 5 min à 10 min ; autres non chiffrées | moyen |
| G8 | `levelup replay compress` | arrêté (recommandé) | environ 3 à 4 min de compression, non mesuré en entier | faible |
| G9 | Redémarrage, santé, sessions | actif | 5 min | faible |
| G10 | Application Windows : release, notes, mise à jour | — | 30 min | moyen |
| G11 | Vérifications à l'écran | actif | à ta main | — |

### G1. Fusion vers `main` et tag `v8.2.0`

- Commande : PR `release/v8.2.0` vers `main`, **Squash and merge** (seul mode activé), titre en Conventional
  Commits ; puis tag `v8.2.0` posé sur le commit de `main` (jamais avant le squash). Les tags `libs/<brique>/...`
  n'existent plus à poser : les briques sont consommées depuis leurs dépôts (`halo-api` v1.0.1,
  `halo-infinite-files-reader` et `halo-infinite-map-images` v1.1.0, `halo-infinite-film-decoder` v1.3.4 ; v1.3.6
  avec la #207). À la lecture du 2026-11-13, le `go.mod` de la release ne porte que des tags (les pseudo-versions
  de travail de la #205 ont été remplacées par `v1.1.0`) ; à relire au moment du squash. Les fonds de
  carte recuits et leur plan des murs (`data/titles/halo_infinite/reference/map_backgrounds/`) sont des données
  versionnées : ils partent avec le déploiement, aucun geste de production ne les concerne.
- Durée : le déploiement attend la CI du même sha (sonde toutes les 30 s), soit 15 à 25 min, puis le
  déploiement. `release.yml` compile l'installeur Windows, signe l'updater, joue `desktop-e2e` et publie
  `updates/latest.json`.
- Risque : moyen. Retour arrière = `git revert` du squash et push ; la migration du G2 est additive (une table
  et une vue), donc pas de restauration restic nécessaire pour elle. **Attention** : après un retour arrière,
  l'ancien binaire porte une autre révision de notes (`lusr-2026-11-08`) ; la comparaison se fait sur
  « différent » (ADR 0047), donc le prochain démarrage rejouerait le recalcul sous l'ancienne formule.

### G2. Migration `shared_match_lead_share_v1` (au boot)

- Commande : aucune, elle s'applique au démarrage du serveur. Crée la table append-only `match_lead_share` et sa
  vue `match_lead_share_latest` dans la base partagée de Halo Infinite.
- Durée : secondes. Pendant le boot `/health` répond 503 `server_starting` : normal.
- Risque : faible (création seule, aucune table existante touchée). Vérifier : `/health` 200,
  `grep -riE 'FATAL|panic' /opt/levelup/data/logs/*.log`, et dans `migration.log` la fin du cycle
  (`migration: cycle terminé`, avec `applied` au moins égal à 1).

### G3. Journaux du serveur en JSON

- Contexte : la variable héritée `LEVELUP_LOG_JSON` est supprimée ; l'image pose désormais
  `LEVELUP_LOG_FORMAT=json` (`Dockerfile`). La prod était déjà en JSON par l'ancienne variable : la sortie doit
  rester identique. Aucun des interrupteurs retirés en même temps n'était posé en prod (mesure du plan).
- Commande : `docker compose logs --tail 5 levelup` : chaque ligne doit être un objet JSON
  (`{"time":...`). Contrôler aussi qu'un fichier de `data/logs/` reste lisible par l'outil de lecture habituel.
- Durée : 2 min. Risque : faible ; seul effet d'un oubli, des lignes en format compact au lieu de JSON, ce qui
  casserait un analyseur de logs. Remède : poser `LEVELUP_LOG_FORMAT=json` dans l'environnement du service.

### G4. Recalcul des notes au premier démarrage (automatique, ADR 0047)

- Contexte : `LUSRRev` passe à `lusr-2026-11-09` (repli du solveur sans les comptes de frags et de morts quand
  il ne converge pas : un match non résolu reçoit enfin une note). Chaque joueur voit son historique rejoué une
  fois, joueur par joueur, au démarrage ; la valeur stockée est `sync_meta.scoring_revision`.
- Commande : aucune. Vérifier ensuite qu'un second redémarrage ne relance pas le recalcul (révision rangée).
- Durée : non mesurée en prod. Repère : 47 s pour un joueur de 1 108 matchs sur copie locale (runbook, E4.6).
- Risque : moyen et visible. D'après l'analyse du plan, environ 70 % des notes déjà posées bougent (jusqu'à
  130 points sur un match, sur un joueur), le plus souvent de peu. Le dire dans les notes de version (fait dans
  les brouillons). Ne pas redémarrer en boucle pendant le recalcul.

### G4b. Table héritée `streak` retirée des bases joueur (migration au démarrage, compléments de la 8.2.0)

- Contexte : la table `streak`, remplacée depuis longtemps par `streak_history` et la vue `streak_latest`, faisait
  refuser à chaque démarrage une réparation de clé (`clé déclarée absente, réparation refusée ... table=streak`).
  La migration `drop_streak_legacy_v1` la supprime de chaque base joueur, seulement si toutes ses lignes sont déjà
  dans `streak_history` ; sinon elle la laisse et le journal dit `streak héritée conservée` avec le nombre de
  lignes non recopiées. Les bases neuves ne la créent plus.
- Séquence : (1) la sauvegarde habituelle du VPS (restic) est faite avant le déploiement ; (2) démarrage du
  serveur : les migrations s'appliquent seules, joueur par joueur ; (3) contrôle du journal : plus aucune ligne
  `réparation refusée` avec `table=streak`, et, pour chaque joueur, `MatchRecomputer: terminé` (recalcul des notes
  de G4 achevé) ; la jauge `levelup.compaction.passes_par_match` reste lue par le script de nuit du serveur, que ces
  compléments ne changent pas.
- Interdit : copier une base locale vers le VPS, pour `streak` comme pour le reste. Toute réparation de données en
  prod passe par une migration au démarrage ou par une passe `backfill-*` lancée sur le serveur.
- Durée : secondes par joueur. Risque : faible (suppression d'une table dont aucune lecture ne dépend, vérifiée
  par la jauge de recopie du step).

### G4c. Précision des records (compléments de la 8.2.0)

- Contexte : la précision lue pour les records était en pourcentage (0 à 100) alors que ses bornes sont en ratio
  (0 à 1) ; les records de précision étaient refusés (`records: valeur hors bornes ignorée`), et ceux déjà rangés
  l'étaient en pourcentage. La lecture convertit désormais en ratio, et la migration
  `records_accuracy_to_ratio_v1` (au démarrage, INSERT seul) range en ratio une ligne neuve par dernier
  record de précision rangé en pourcentage. La colonne des matchs n'est pas touchée.
- Séquence : démarrage (le correctif et la migration sont dans le binaire), puis contrôle du journal : plus de ligne
  `valeur hors bornes ignorée ... metric=accuracy` au post-sync, et, dans `migration.log`, la ligne
  `records de précision convertis en ratio`.
- Risque : faible. Toute autre réparation de données serait une migration au démarrage.

### G5. Recuisson des rejeux, en LOCAL (à ton go, jamais en lot sans accord)

**FAIT le 2026-11-14** (go du user du 2026-11-13) : une seule passe `backfill-replay`, binaire de `release/v8.2.0`
à `76da40dc4f` (décodeur v1.3.7, schéma **116**, 33 ms), 1 455 rejeux en 1 h 36, archive des films intacte
(1 614) ; dérivés rejoués en local ensuite (killsource compris). Le texte qui suit garde la préparation.

- Contexte : le schéma des rejeux est à **113** sur la release (décodeur v1.3.4, prod au 107 depuis la 8.1.0) ;
  la lecture complète du film (#198) et les montées de décodeur de la #203 ont changé les révisions de grammaire
  (`grammar-2026-11-12.4`), de source des éliminations (`killsource-2026-11-10`) et de profil
  (`profile-2026-11-12.2`) : les faits de film déjà rangés sont donc périmés et la recuisson **redécode** au lieu
  de republier, tant que le `--dry-run` n'annonce pas « republier ». Les destructions de véhicule, les occupants
  tués avec leur véhicule et les corrections de lecture (postures, armes au sol) n'existent que dans un rejeu
  recuit : sans recuisson, la prod garde l'ancien contenu.
  **À CONFIRMER, #207** : elle monterait le schéma à **115** et la cuisson à 30 images/s (pas de 33 ms ; un rejeu
  resté à 100 ms est alors signalé « cadence périmée »). Ne recuire qu'**une seule fois, au schéma final** de la
  release : trancher la #207 AVANT de lancer la recuisson. Fusionnée, c'est 115 ; écartée de la 8.2.0, c'est 113
  et le passage ultérieur à 30 images/s imposerait une seconde recuisson complète.
- Commandes, serveur local arrêté, depuis `apps/go-api` :
  `go run ./cmd/levelup backfill-replay --only-existing --dry-run`, puis
  `go run ./cmd/levelup backfill-replay --only-existing` (séquentiel, un enfant par film, `--mem-limit-gib`
  pour relever le plafond). Sur le bureau, c'est la file de régénération lancée depuis l'application
  (bandeau d'offre), jamais automatique.
- Durée : non mesurée sur le parc. Repère : 12 films recuits en 78 s (environ 6,5 s par film), soit de l'ordre de
  2 h 40 pour 1 445 films ; davantage si redécodage complet. Fenêtre à part.
- Risque : élevé côté mémoire (bombe RAM connue : jamais de balayage du corpus en parallèle). Les rejeux
  produits sont déjà écrits en `.json.gz`.
- `backfill-killsource` : en local seulement, base non copiée (D-17). En prod les anciens matchs gardent leurs
  lignes actuelles, les nouveaux films profitent de la correction.

### G6. Mise en place des rejeux régénérés (dossier d'attente, `BASCULE.sh`)

**Déjà fait (2026-11-14, accord du user)** : les 1 455 rejeux régénérés (`.json.gz`, schéma 116, 33 ms) et leurs
1 455 `.meta` sont sur le VPS, dans un dossier d'attente SOUS `data/` (à l'abri du `git clean` du déploiement) :
`/opt/levelup/data/replays-staging-v8.2.0/halo_infinite/` (1,9 Go, `deploy:deploy`). Rien n'a été écrit dans le
dossier servi : la prod 8.1.3 ne lit pas les `.json.gz`, et un `.meta` neuf à côté d'un ancien `.json` y décrirait
un fichier absent. Ni marque `.derived.json` (elle dirait faits des dérivés de la base LOCALE), ni `rasters/`, ni
le seul rejeu brut local (match d'un joueur tiers sans film) ne sont dans le dossier d'attente. Aucune base.

**Quand** : juste après le déploiement de la 8.2.0 (G1), une fois le premier démarrage fini (G2, G4, G4b, G4c :
`MatchRecomputer: terminé` pour chaque joueur au journal). Serveur ACTIF toléré : chaque fichier est posé par
copie vers un temporaire puis renommage, et le serveur de prod ne cuit pas (pas de films).

```bash
cd /opt/levelup
bash data/replays-staging-v8.2.0/BASCULE.sh            # simulation : ne touche à rien
# attendu (mesuré le 2026-11-14) : « 1455 rejeux copiés, 1455 anciens .json bruts retirés, 0 erreurs »
bash data/replays-staging-v8.2.0/BASCULE.sh --apply    # copie, puis retire les .json remplacés
```

- **Ce que `--apply` retire, et seulement cela** : pour chaque match du dossier d'attente, l'ancien
  `<id>.json` brut du MÊME match, après la copie vérifiée (même taille) de son `<id>.json.gz`. Les 1 455 `.json`
  retirés sont donc tous remplacés par leur `.json.gz` régénéré. Les rejeux de prod SANS équivalent régénéré
  (matchs de joueurs présents seulement en prod, environ 2 080, à l'ancien schéma, sans film pour les refaire)
  ne sont PAS touchés : ils restent en `.json` et passent par le G8 (`replay compress`). Leurs `.meta` et leurs
  marques `.derived.json` restent aussi.
- **Pourquoi retirer l'ancien `.json`** : laissé à côté du `.json.gz`, il ferait un doublon de contenus
  différents à chaque match (WARN par match, compteur `replay_artifact_duplicate_files_total`), que le G8 ne
  résorbe pas (il ne supprime que les doublons identiques) ; la place occupée resterait celle d'aujourd'hui
  (8,7 Go pour 3 537 `.json`).
- Risque : moyen. Retour arrière possible tant que le dossier d'attente existe, pas pour les `.json` retirés
  (restic du P4). Le script s'arrête à la première erreur de copie (`set -e`).

### G7. Passes de dérivés sur le serveur (serveur arrêté, une fenêtre)

**Ordre dans la fenêtre** (un seul écrivain par base, ADR 0013) : G1 déployé, premier démarrage fini (G2, G4,
G4b, G4c au journal), puis G6 serveur actif ; ENSUITE `docker compose stop levelup`, les passes ci-dessous dans
l'ordre, puis le G8 dans la même fenêtre, puis le G9 (redémarrage). Jamais de passe pendant le premier démarrage :
G4b et G4c sont des migrations du démarrage, et le recalcul du G4 tient les bases joueur.

```bash
cd /opt/levelup && docker compose stop levelup
LU() { docker compose run --rm --no-deps levelup levelup "$@"; }
LU backfill-usage-summary                 # sans --force : le schéma 116 des rejeux suffit à tout re-résumer
LU backfill-pad-tiers --force             # niveaux d'armes de socle, relus au schéma 116
LU backfill-vehicle-takes --force         # prises de véhicule, obligatoire après une recuisson
LU backfill-flag-grabs-net --force        # prises de drapeau nettes, relues au schéma 116
LU tactical-rasters --backfill            # calques de la Tactique (fichiers sous rasters/, idempotent)
LU backfill-lead-share --dry-run          # lecture seule, aucune migration, une ligne par match et un bilan
LU backfill-lead-share                    # la table est neuve (G2) : pas de --force
```

- `backfill-killsource` : SANS OBJET en prod (aucun film ; il décode les films). Fait en local le 2026-11-14.
- Mesures locales du 2026-11-14 (1 455 rejeux, poste chargé) : `usage-summary` 3 min 33, `pad-tiers --force`
  4 min 05, `vehicle-takes --force` 6 min 52, `flag-grabs-net --force` 7 min, `tactical-rasters --backfill`
  8 min 45, `backfill-lead-share` 7 min 20 en simulation ; tous à 0 échec. La prod compte environ 3 500 rejeux
  (les anciens, non régénérés, sont relus tels quels) : compter de l'ordre de 1 h de fenêtre sur 2 vCPU,
  extrapolation non mesurée.
- `backfill-usage-summary` : la reprise saute un match seulement si la révision de projection ET le schéma de
  l'artefact sont ceux d'aujourd'hui (`cmd_backfill_usage_summary.go`) ; la révision reste `us7` (décodeur
  v1.3.7).
- Risque : moyen. Les passes sont INSERT seul (anti-ART) et reprenables ; une ligne `failed` non nulle rend la
  commande non nulle. Les matchs de modes à peu de points (drapeau, colline, bombe) sont écrits « sans chiffre »
  avec leur raison, jamais zéro (78 % mesurés en local).
- Les nouveaux matchs synchronisés ensuite reçoivent leurs dérivés par la chaîne post-synchronisation.

**Suppression du dossier d'attente : seulement après un contrôle écrit**, consigné dans le journal de la release :
(1) compte des rejeux servis, `ls /opt/levelup/data/cache/replays/halo_infinite | grep -c '\.json\.gz$'`, au moins
1 455 (plus ceux du G8) ; (2) un rejeu régénéré ouvert à l'écran en prod (lecture fluide à 30 images/s). Ensuite :
`rm -rf /opt/levelup/data/replays-staging-v8.2.0`.

### G8. `levelup replay compress`

- Commandes, depuis `/opt/levelup` : `LU replay compress --dry-run` (lit, compresse en mémoire, affiche le gain,
  n'écrit rien), puis `LU replay compress`. Convertit sur place les `.json` bruts restants (surtout les 2 079
  rejeux présents seulement en prod), jamais lancé automatiquement.
- Durée : compression de l'ordre de 110 ms par artefact (plan), soit 3 à 4 min pour 3 500 artefacts, relecture
  et disque en plus, non mesuré en entier. Gain mesuré en local : 3,6 Gio vers 746 Mio (rapport 0,20).
- Risque : faible. Pose exclusive du `.json.gz` (jamais d'écrasement), relecture identique avant la suppression
  du `.json`, idempotent, un échec est compté et laisse le fichier (code de sortie non nul : à relancer). Le
  serveur peut rester actif : la pose d'un `.json.gz` est exclusive et atomique, un `.json.gz` posé par le serveur
  pendant la conversion n'est jamais remplacé (en-tête de `cmd_replay.go`) ; la rejouer dans la fenêtre où le
  serveur est déjà arrêté reste une prudence, pas une exigence. Un match qui porte les deux formes se juge sur le
  contenu : identique, le `.json` est supprimé ; différent, rien n'est supprimé et la commande le compte (aucun
  cas attendu : le G6 retire lui-même l'ancien `.json` de chaque rejeu régénéré). Dans la fenêtre du G7, après
  ses passes. Contrôler l'espace libre du VPS avant et après.

### G9. Redémarrage, santé, sessions

- Commandes : `docker compose up -d levelup`, `curl -sf http://127.0.0.1:8000/health` (200 avant de continuer).
  Recalcul des sessions (route admin `POST /api/v1/settings/sessions/recalculate`, étape R8 du runbook) : **non
  nécessaire pour la 8.2.0**. Le recalcul ne lit que l'heure de début, les coéquipiers, le drapeau classé et la
  durée jouée de `match_registry` et `match_participants` et n'écrit que `session_id` et `session_label`
  (`sync/session_recalc.go`) : ni le recalcul LUSR du G4 ni les rejeux recuits ne les changent, et le code des
  sessions (`analysis/sessions.go`, `domain/session_*`) n'a reçu depuis `main` que des renommages d'import, le retrait
  de la probabilité de victoire et les champs de Mainmise, lus à la requête. À rejouer seulement si une règle de découpage change dans une release
  future.
- Durée : 5 min. Risque : faible.
- Démo : le job `deploy-demo` la régénère ; vérifier qu'elle répond 200.

### G10. Application Windows

- La release GitHub porte l'installeur, `updates/latest.json` (notes FR et EN tirées de la première section des
  deux fichiers de notes) et un paquet de ressources qui change (icônes de grenades et de capacités, fonds de
  carte recuits et plan des murs de la #205 : les images et leurs fichiers de calage pèsent plus lourd, un
  téléchargement unique à la mise à jour).
- À faire : essai de mise à jour depuis une 8.1.2 installée (offre, notes, redémarrage, données intactes) ;
  vérifier que les anciens fichiers de l'interface sont retirés ; première ouverture d'un ancien rejeu (lisible
  sans rien faire) et offre de régénération, qui ne démarre jamais seule.
- Durée : 30 min. Risque : moyen (installeur non signé, SmartScreen avertit à chaque installation).

### G11. Vérifications à l'écran (à ajouter à l'artefact des vérifications, collection `verifs`)

Rejeu : filtre du fil, clic sur une élimination, bouton Vue, fiche en panneau (Suivre, Seul, Masquer, Échap).
Vue du match : Écart de score, Supériorité numérique, Mainmise (tiret, « 0 retournement »). Sessions, Séries
temporelles, Escouade : Mainmise. Aperçu de lien Discord : lien copié depuis l'application (langue du partageur)
et lien nu (anglais). Après la recuisson : destructions de véhicule dans le fil et le filtre « Véhicules » (ligne
« détruit », parenthèse « à bord ») sur un match à véhicules. Fonds de carte : calque « Plan des murs » et
orientation par quarts de tour dans le panneau « Vue » sur une carte native et sur une carte Forge, Live Fire sans
plan, puis les mêmes réglages sur la Tactique et « Occupation du terrain ». Tableaux triés (administration, joueurs
croisés de la Carrière, Classement : sens du premier clic, valeurs absentes en dernier) et les deux graphes à
palette d'accessibilité. Avec la #207 : rejeu à 30 images/s en lecture, badge « cadence périmée » sur un rejeu
resté à 10 images/s.

## Récapitulatif des points À CONFIRMER

| Point | Dépend de | Effet sur les gestes |
|---|---|---|
| Cuisson à 30 images/s, schéma 115, décodeur v1.3.6, pictogramme et tués sur les destructions | PR #207, ouverte ; fusion suspendue aux cinq lignes « non établi » du gate et à l'acceptation de deux FAUX attribués à la grille | G5 : trancher AVANT la recuisson, qui ne se fait qu'une fois au schéma final (115 fusionnée, 113 sinon) ; G6 et G7 : ouvrier et serveur déployés ensemble ; notes : deux lignes marquées |

Points levés depuis le premier brouillon, sur pièces :

| Point | Verdict | Source |
|---|---|---|
| Schéma 109, destructions de véhicule (#203) | fusionnée ; le schéma de la release est 113 (décodeur v1.3.4) | `replay/document.go` du décodeur v1.3.4, PR #203 |
| Étape 4 du plan d'architecture (4.4 à 4.8) | close, ainsi que les étapes 5 à 8 ; aucun geste de production | journal du 2026-11-13, historique de la release |
| Commande de copie des rejeux | remplacée le 2026-11-14 : dossier d'attente déjà en place sur le VPS, `BASCULE.sh` en simulation puis `--apply` juste après le déploiement (G6) | simulation sur le VPS du 2026-11-14 : 1 455 copiés, 1 455 `.json` remplacés, 0 erreur |
| Révision des résumés d'usage (« us8 ») | n'existe pas : `us7` dans v1.3.4 et v1.3.6 ; le schéma d'artefact suffit à re-résumer | `replay/usage_summary.go` du décodeur, `cmd_backfill_usage_summary.go` |
| Recalcul des sessions | non nécessaire : les sessions ne dépendent ni des notes ni des rejeux | `sync/session_recalc.go`, comparaison de `analysis/sessions.go` avec `main` |
| `replay compress` : serveur arrêté ? | non exigé : pose exclusive et atomique, serveur actif toléré | en-tête de `cmd_replay.go` |
