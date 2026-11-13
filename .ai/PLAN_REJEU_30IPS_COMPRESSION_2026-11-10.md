# Plan : rejeu à 30 images/s et rejeux compressés sur disque (2026-11-10)

Statut : **go du user le 2026-11-10, exécution en cours**. Exécution sous le skill `plan-execution`.

## 0. Objectif et critère de succès

- **Fluidité.** Tous les rejeux sont publiés à 30 images/s, tout le point compris (position, visée, bouclier). Essai concluant sur un match le 2026-11-09 (verdict user : « très propre et fluide »).
- **Disque.** Les rejeux sont stockés compressés. Mesuré sur 30 rejeux : ratio 0,20, décompression de 6 ms pour le rejeu médian et de 20 ms pour le plus gros, en Go.
- **Évolutivité.** Le user va comparer le décodeur à un décodeur concurrent et corriger des trous de données. Changer un champ, combler un trou ou changer à nouveau de cadence ne doit toucher ni le stockage ni les seuils.

Le plan est réussi quand :
- les quatre conditions de la porte de B9 sont tenues ;
- le parc local est entièrement à 30 images/s et compressé ;
- l'empreinte disque est mesurée (estimée à environ 1,8 Go, contre 3,6 Go aujourd'hui) ;
- tous les garde-rails sont verts.

## 1. Constats sur pièces (inventaires du 2026-11-10)

- **Le format du fichier est découplé du format envoyé à la page.**
  - `service/replay_service.go:199` décode l'artefact, l'enrichit, puis `replayview.FromArtifact` le projette. Huma ré-encode le résultat et `chimiddleware.Compress(5)` le compresse.
  - Conséquence : compacter le fichier ne réduirait rien sur le réseau, seulement sur le disque, et la compression couvre déjà le disque.
- **Lecteurs qui décodent dans `replay.ReplayDocument`** (`GetReplay`, `replayartifacts.LireArtefactRange` et les commandes `backfill-*`, `validateArtifact`) : ils sont insensibles à la forme JSON.
- **Lecteurs ad hoc**, qui casseraient si la forme des listes changeait :
  - `service/tactical_sequence_store.go:45` ;
  - `cmd/backfill_t0_film/artefacts.go:67` ;
  - `internal/replayverite/document.go:301` ;
  - `internal/replaydiff/empreinte.go:68` ;
  - `libs/film/replay/vehicules_chassis_inventaire_test.go:59` ;
  - les fixtures web `replay_schema_107_*.json.gz`.
- **Lectures d'octets directes, sensibles à la compression :**
  - les mêmes lecteurs ad hoc ;
  - `artifact_digest.go:126` ;
  - `artifact_meta.go:114`, dont `bytes == st.Size()` est rompu sous compression ;
  - `derivations_index.go`, qui compare `ArtifactBytes` à `st.Size()` ;
  - `cmd/zone-attribution/census.go:115` ;
  - `ops/seed_demo_replays.go:177` ;
  - `cmd/replay-corpus-gate/basecache.go:275` ;
  - les tests web de mesure activés par variable d'environnement.
  - Non inspectés : `cmd/replay-equiv/parent.go:350`, `cmd/levelup/cmd_replay_parc_vies*.go`, `sync/replayartifacts/zones_rattrapage.go:394`.
- **Extension `.json` en dur.** Source unique : `domain/title/film_id.go:84`. Mais `HasSuffix(".json")` est recopié dans `replay_service.go:133`, `replay_purge_cron.go:177,189,231`, `replayregen/scan.go:56` et `backfill_t0_film/artefacts.go:107`. Plus de deux copies : règle 6.
- **Seuils exprimés en images**, qui deviendraient 3 fois plus courts à 33 ms.
  - Go :
    - `zone_attribution.go:50` ;
    - `replayverite/seuils.go:20,37,46,51,55` ;
    - `analysis/tactical/sequence.go:38` ;
    - `projectiles_tirs.go:18` ;
    - `vehicle_turrets.go:94` ;
    - `projectiles.go:84`, un pas en mètres par image ;
    - `t0_film.go` vers `:72`, un plancher de bruit par pas.
  - Web : `bombBlastFx.ts:94`, `flagCaptureFx.ts:86`, `flagCarriesLayer.ts:259`, `carriedGlyphPulse.ts:29`, `replayProjectiles.ts:16`, `headingInterp.ts:27`, `vehiclesAim.ts:52`, `skullSound.ts:121,129`, `zoneSound.ts:344`, `weaponChangeSound.ts:96`. `placementWall.ts:110` reste à vérifier.
  - Le moteur de lecture, `useReplayTiming` et la Tactique web travaillent déjà en millisecondes.
- **Défaut de relecture.** `replayartifacts/raster.go:115` relit les artefacts sans `frameIntervalMs` avec `replay.DefaultFrameIntervalMS`. Si cette constante passe à 33, les anciens artefacts sont relus à 33 ms.
- **Révision de publication.** Elle vaut `"publication-" + SchemaVersion` (`layers.go:115`), et `frameIntervalMs` en dépend (`layers.go:133`).
  - Pour qu'un rejeu à 100 ms soit jugé « à re-cuire », `SchemaVersion` doit monter (108 depuis l'étape 4.1 du plan v8.1.2, donc 109).
  - Une montée ne déclenche **aucune** recuisson automatique en lot. La file du bureau ne démarre que sur un geste du user, et l'après-sync est plafonné à 5 rejeux par cycle.
- **Le pas de 33 ms donne 30,3 images/s.** Chaque champ `T` vaut un pas de 33 ms, ce qui est cohérent sur tout l'axe.
- **Essai du 2026-11-09.** Les fiches n'avaient pas leurs compteurs, parce que l'essai passait par `cmd/replay-build` et non par `backfill-replay`. La cause est à confirmer en B7.

## 2. Décisions

| # | Décision | Raison |
|---|---|---|
| D-1 | **Pas d'écriture compacte (colonnes) dans le fichier.** | Le serveur ré-encode de toute façon pour la page, donc aucun gain réseau. La compression donne déjà le gain disque. Six lecteurs ad hoc et les fixtures casseraient. Un codec par champ rendrait chaque correction du décodeur plus coûteuse. |
| D-2 | **Écriture compacte sur le réseau, seulement sur mesure** (étape D) | Règle « pas de sur-correction ». Si elle se fait, elle se fait en un seul endroit de chaque côté (`replayview` et `normalizeReplayDocument`), de façon générique, sans table par champ. |
| D-3 | **Grille à 33 ms pour tout le point** | C'est le réglage de l'essai validé. Une seule cadence évite de gérer deux grilles. |
| D-4 | **Fichier `{short8}.json.gz`, gzip de la bibliothèque standard, niveau 6** | gzip compresse mieux que zstd niveau 3 ici (0,204 contre 0,223). La compression coûte 110 ms une fois, à l'écriture. Aucune dépendance ajoutée. L'extension dit le contenu, et le parc mixte reste lisible. |
| D-5 | **Toute lecture et toute écriture d'octets d'artefact passent par une porte unique de `internal/replaybuild`**, tenue par un garde-rail | La compression devient un détail de la porte. Un futur changement de stockage touche un fichier. |
| D-6 | **Tout seuil temporel est écrit en millisecondes et converti à la lecture**, garde-rail Go et web | Changer de cadence redevient un réglage. Les corrections du décodeur ne rencontrent pas de seuil caché. |
| D-7 | **`SchemaVersion` passe à 109** à la bascule de cadence. Une constante `HistoricalFrameIntervalMS = 100` est figée pour relire les artefacts sans champ. | C'est la seule voie par laquelle les rejeux à 100 ms se voient « à re-cuire ». Le défaut de construction et le défaut de relecture sont deux faits distincts. |
| D-8 | **La recuisson du parc et la conversion des fichiers sont des gestes du user** | Règle « jamais de cuisson en lot sans demander ». La copie vers la prod porte sur les rejeux seuls, jamais sur les bases. |
| D-9 | **Bureau : la compression des rejeux non refaisables se fait à la fin de la mise à jour des rejeux lancée par l'utilisateur** (accord du user, 2026-11-10) | Une application installée n'a pas de ligne de commande. La recuisson à 30 images/s écrit déjà compressé ; seuls les rejeux dont le film n'est plus archivé resteraient bruts. |

## 3. Dépendances et gestes réservés au user

- **Dépendance : après la clôture du plan v8.1.2.** Son étape 6 bascule LevelUp sur les modules publiés, et `libs/film` en fait partie. Toute modification de `libs/film/replay` passe ensuite par un numéro de version du module, tag posé par le user. Branche `feat/rejeu-30ips` depuis la release qui suivra la v8.1.2.
- **Gestes du user :**
  - le go de ce plan ;
  - la porte visuelle B9 ;
  - la conversion du parc, en C1 ;
  - la recuisson, en C2 ;
  - la copie des rejeux vers la prod et le déploiement simultané du serveur et des ouvriers, en C4 ;
  - le tag de module ;
  - le merge vers `main`.

## 3 bis. Contrat d'exécution

- Skill `plan-execution` (contrat par défaut). Ordre strict : une étape n'est close que lorsque son gate est passé et consigné dans le Journal (section 9). Statuts : `[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (justification écrite). Aucune case vide à la clôture.
- Zéro correctif hors périmètre : toute trouvaille va dans « Découvertes » (section 8).
- Reprise de session : lire le Journal, puis `git log --oneline -10` sur `feat/rejeu-30ips`, et reprendre au premier item non coché.
- Effort : 0 rapide ; A moyen (une vingtaine de fichiers, surtout des remplacements par la porte) ; B lourd (seuils, re-mesures, goldens) ; C dépend du parc (recuisson séquentielle) ; D rapide à mesurer.
- Couches : la porte et la conversion dans `internal/replaybuild` (orchestration de stockage) ; le prédicat de nom dans `internal/domain/title` à côté de `ReplayArtifactFileName` ; les seuils restent dans leurs paquets (`libs/film/replay`, `internal/analysis/tactical`) ; la commande dans `cmd/levelup`. Aucun handler ne change. Les chemins passent tous par `PathResolver`. Le rejeu reste derrière sa capability existante, donc aucun cas propre à un titre.
- Journalisation : la porte écrit un `slog.ErrorContext` et incrémente un compteur sur un fichier illisible, sans dégradation silencieuse. Le parc mixte (`.json` et `.json.gz` du même match) donne un `slog.WarnContext` et un compteur. La commande `replay compress` rend un compte rendu chiffré : convertis, déjà compressés, échecs.
- Commandes du gate (exactes) : `make go-api-test` ; `cd apps/go-api && go test -tags=integration ./...` ; `make libs-test` ; `make test-web` ; `make check-types` ; `make go-api-lint` ; `make replay-equiv` ; `make replay-corpus-gate`.

## 4. Étapes

### Étape 0 — Témoins AVANT
- [x] 0.1 Fixer une liste fermée de 12 matchs : Assassin, CTF, Roi de la colline, Bastions, Oddball, Bombe, une carte Forge, un match avec véhicules, un match avec bots, plus 0076ebdc.
  Liste fermée (2026-11-10). Tous ont un film archivé (manifeste, chunks, faits) et un artefact au parc (schéma 107), ce qui a été vérifié sur pièces ; le choix vient des artefacts et du registre `shared_matches_v2.duckdb`.

  | short8 | Mode (variante) | Carte | Ce que le match couvre |
  |---|---|---|---|
  | `0076ebdc` | Super Fiesta (`Slayer:Arena Super Fiesta`) | Prism | obligatoire ; 5 043 images, 106 pistes, 691 projectiles |
  | `ef8cb72b` | Assassin (`Slayer:Arena`) | Vagabond | Assassin d'arène, sans véhicule |
  | `008e1bba` | CTF (`CTF:Arena`) | Critical Dewpoint | 2 prises de drapeau, sans bot |
  | `d2b74083` | Roi de la colline (`KOTH:Arena`) | Illusion | zone tenue, sans bot |
  | `de3cec8b` | Bastions (`Strongholds:Arena`) | Kiken'na | bases, sans bot |
  | `43716616` | Oddball (`Oddball:Arena`) | Smallhalla | balle |
  | `c75f33b8` | Bombe (`Assault:One Bomb`) | Curfew | bombe, 2 pistes de bot |
  | `e157a672` | Carte Forge (`Husky Raid:Super CTF`) | Pharaoh | trame Forge `fo11_blank` (REGISTRE_CARTES), 2 prises de drapeau |
  | `82f29378` | Véhicules (`BTB:CTF`) | Oasis | 22 véhicules, grande équipe (25 joueurs) |
  | `ce927bd2` | Bots (`Strongholds:Arena`) | Live Fire | 11 pistes de bot (noms « [bot] », xuid vide) |
  | `be565134` | Extraction (`BTB:Extraction`) | Refuge | Extraction, 21 véhicules, le plus gros de la liste (5,3 Mo) |
  | `4f862258` | Classé (`Ranked:Slayer`) | Solitude - Ranked | playlist classée |

  Non couvert : aucun match avec bots ET véhicules à la fois n'a été retenu (`0f9550e5` le ferait, non ajouté pour tenir 12). `82f29378` n'a aucun calque de drapeau (`flagCarries` vide) malgré son mode CTF ; la partie « drapeau » est tenue par `008e1bba` et `e157a672`.
- [x] 0.2 Capturer sur cette liste et conserver dans le dossier de gate (`data/investigation/gate_30ips/avant/`, commandes exactes dans `COMMANDES.md`, somme de contrôle de chaque fichier dans `MANIFEST.sha256`). Faits le 2026-11-10 avec le code de la branche (schéma 108, cadence 100 ms), HEAD `8b5fc378c7` :
  - [x] les 12 artefacts recuits un à un par `levelup backfill-replay --match <id> --force`, décodage complet, 2 à 16 s chacun, 78 s en tout ; `artefacts/` (json et meta), `sha256.txt`, `tailles.txt` (la taille de chaque fichier) ; cuisson déterministe (mêmes octets sur seconde cuisson, et depuis les faits ou depuis le film) ;
  - [x] `replay-equiv` : `make replay-equiv`, 20 identiques, 0 différent, sortie 0, 1253 s (`replay-equiv.txt`) ;
  - [x] les empreintes `replaydiff` : `replaydiff/<id>.empreinte.json` (toutes les mesures non nulles, par comparaison à un document vide, car `replay-diff` n'a pas de mode « empreinte seule »), `.auto.json`, et l'écart au parc 107 (`parc107_vs_108`) ;
  - [x] `replay-verite` sur les 12 (avant = parc 107, après = branche, faits et oracles exportés dans `faits/`) : 12 « ok », code 0 ;
  - [x] simulations (`derives/`) : `backfill-usage-summary`, `backfill-vehicle-takes`, `backfill-flag-grabs-net`, `backfill-pad-tiers`, chacune `--dry-run --force --match <id complet>` par match (sans `--force`, `vehicle-takes` et `pad-tiers` répondent « déjà en base » sans projeter) ; `tactical-rasters --backfill --dry-run` (aucun filtre par match : il couvre les 12 artefacts du worktree, comptes seulement, le contenu du sidecar n'est pas capturé) ;
  - [!] une séquence de mort Tactique, réelle : impossible sans serveur ni base joueur (`POST /api/v1/players/{player_slug}/tactical/{map_id}/sequence`, corps `match_id`, `victim_xuid`, `time_ms`, lit `stats.duckdb` du joueur, non monté dans ce worktree). Substitut capturé, sans serveur : le test de recherche existant `TestMesureLectureDesPistes` (`-tags research`) calcule la séquence d'une piste à mi-vie et rend nombre de pistes, de points, cadence et taille du corps (`tactique/sequence_mesure.txt`, 12 lignes). Pour une vraie séquence : monter la base joueur et lancer le serveur du worktree, puis poster la requête.

Gate : témoins rangés, liste écrite dans ce plan.

### Étape A — Porte unique et compression (aucun contenu ne change)
- [x] A1 **Tests d'abord.** Écrits avant le code, rouges, puis verts :
  - [x] aller-retour écriture compressée puis lecture identique, octet pour octet (`replayartifact/gate_test.go` `TestEcritureCompresseRelitIdentique`, `replaybuild/artifact_stockage_test.go` `TestPuits_RangeUnCompresseQuiSeRelitIdentique`) ;
  - [x] lecture d'un `.json` ancien, et détection par les octets magiques, pas par le nom (`TestLitUnJsonAncien`, `TestDetecteParLesOctetsMagiques`) ;
  - [x] parc mixte : le `.gz` gagne, doublon compté (`TestParcMixte`) ; une réécriture retire le `.json` ancien ;
  - [x] `.meta` avec `bytes` égal à la taille stockée, nom `{short8}.meta` inchangé (`TestPuits_EnteteBytesEgalTailleStockee`) ;
  - [x] `DerivationsUpToDate` sous compression, nom `{short8}.derived.json` inchangé (`TestPuits_DerivationsUpToDateSousCompression`) ;
  - [x] listage par un seul prédicat de nom (`title.IsReplayArtifactFile`, `TestListeUnSeulPredicat`).
- [x] A2 **Porte.** Décision d'écart : la porte vit dans un paquet feuille `internal/replayartifact` (`Read`, `ReadBytes`, `Locate`, `Exists`, `List`, `Write`, `Copy`, `Remove`, ...) et non dans `internal/replaybuild`, parce que deux ratchets existants l'exigent : la page Tactique et son rattrapage ne doivent pas nommer `replaybuild` (`no_cuisson_depuis_tactique_test.go`) et ils lisent des artefacts ; le périmètre du décodeur (`film_slog_contexte_test.go`) interdit un `context.Background()` dans `replaybuild`. `replaybuild` garde l'en-tête `.meta`, la marque de dérivation, la purge et le puits (`writeArtifactBytes` compresse par `replayartifact.Write`). `ReplayArtifactFileName` rend `.json.gz` ; `title.IsReplayArtifactFile(name)` remplace les `HasSuffix` ; constantes d'extension dans `film_id.go`.
  - **Sens de `bytes`** : la TAILLE STOCKÉE (octets du fichier sur disque, celle de `os.Stat`), partout : `.meta`, marque de dérivation (`artifactBytes`), `Digest.Bytes`, accusé d'écriture (`StoredArtifact.Bytes`, reçu de l'ouvrier, événement « rejeu prêt »). Le `.meta` reste fiable sans relire l'artefact (la règle « taille déclarée = taille du fichier » est inchangée) ; la ligne ancrée par `filmreplis/sites.go` n'a pas bougé.
  - **Transport** : l'ouvrier (`cmd/replay-worker`) hache et envoie le JSON BRUT (`built.Blob`) ; seul le disque du serveur compresse. Vérifié : `build_queue_transport_e2e_cgo_test.go` compare le JSON envoyé au contenu décompressé du fichier rangé.
- [x] A3 **Migrer tous les lecteurs** de la section 1 vers la porte, après réouverture de chaque fichier :
  - service : `replay_service.go` (`IsAvailable` par `Locate`, `AvailableSet` par `List`, `GetReplay`), `tactical_sequence_store.go` ;
  - `scheduler/replay_purge_cron.go` (un seul listage, marques orphelines par `Exists`, `Remove` des deux formes), `replayregen/scan.go`, `replaybuild/artifact_purge.go` (retire les deux formes) ;
  - `sync/replayartifacts` : `document.go` (`ouvrirArtefact` = `replayartifact.Read`, la taille rendue est la taille stockée), `backlog.go` ; `LireArtefactRange` et les commandes `backfill-*` en héritent ;
  - `cmd/backfill_t0_film`, `cmd/zone-attribution/census.go`, `cmd/mapplafond-mesure`, `cmd/replay-verite`, `cmd/replay-corpus-gate` (`verite.go`, `basecache.go` : le cache de base range et somme le JSON décompressé, donc les entrées d'avant restent valides), `replaydiff.LireDocument`, `cmd/levelup` (`--only-existing`, `classerReparation`), `ops/seed_demo_replays.go` (copie par `Copy`, élagage par `ReplayArtifactStem`) ;
  - les trois non inspectés, lus : `cmd/replay-equiv/parent.go:350` lit un fichier de corpus (lignes de texte), pas un artefact ; `cmd/levelup/cmd_replay_parc_vies*.go` n'ouvre aucun artefact du parc (faits et TSV ; `--garder-documents` DÉPOSE des documents bruts hors parc) ; `sync/replayartifacts/zones_rattrapage.go:394` lit un `.mvar` de carte. Aucun n'est migré (rien à migrer) ;
  - `libs/film/replay/vehicules_chassis_inventaire_test.go` : lecture locale gzip-ou-brut (ADR 0041, pas d'import de `apps/go-api`) ; trois tests de corpus Go (`cadence_corpus_test.go`, `grenade_corpus_test.go`, `drapeau_bases_film_parc_research_test.go`) passent par la porte ;
  - `scripts/bench-pages.sh` copie aussi `.json.gz` ; web : cinq tests de mesure lisent par `test/artefactFile.ts` (gunzip si octets magiques, le compressé gagne) ;
  - le chemin côté démo (`DemoLayout`) suit `ReplayArtifactFileName` : les rejeux figés de la démo sont rangés compressés, les anciens se lisent.
- [x] A4 **Garde-rails** (allowlist vide) :
  - `archlint/no_artefact_bytes_outside_gate_test.go` : aucun `os.ReadFile` / `Open` / `OpenFile` / `Stat` / `Lstat` / `WriteFile` dont l'argument ou la fonction englobante nomme un artefact, hors `internal/replayartifact` et `internal/replaybuild` (qui possède `.meta`, marque et purge) ; auto-preuve du motif. Variante « voisin » plutôt qu'un élargissement de `no_artefact_reread_in_cmd_test.go`, dont l'empreinte (`json.Unmarshal` + `replay.ReplayDocument`) vise une autre copie ;
  - `archlint/no_replay_json_literal_test.go` : aucun littéral `".json"`, `".json.gz"` ni `".derived.json"` dans une fonction qui construit ou balaie des chemins de rejeu, hors `film_id.go` ;
  - `archlint/no_auto_replay_compress_test.go` : `CompressDir` n'a qu'un appelant, la commande (D-8).
- [x] A5 **Commande `levelup replay compress [--dry-run] [--title] [--dir]`** (`cmd/levelup/cmd_replay.go`, `replaybuild/artifact_compress.go`).
  - Convertit sur place, atomiquement (`atomicfile`) ; relit le `.gz` décodé et exige des octets identiques AVANT de supprimer le `.json` (sinon le `.gz` est retiré et le `.json` reste) ; réécrit le `.meta` (version de l'application conservée) et reporte la marque de dérivation à jour sur la nouvelle taille stockée (aucune dérivation rejouée) ;
  - idempotente ; un match aux deux formes est compté en doublon et laissé tel quel ; compte rendu : convertis, déjà compressés, doublons, échecs, octets avant et après ; code de sortie non nul s'il y a un échec ; arrêt doux sur Ctrl-C ;
  - tests : `replaybuild/artifact_compress_test.go` (dossier temporaire : conversion, idempotence, simulation, illisible, relecture menteuse, doublon, contexte annulé) et `cmd/levelup/cmd_replay_test.go`.

Gate A :
- tous les témoins de 0.2 sont identiques, tailles exceptées ;
- `make go-api-test`, `go test -tags=integration ./...`, `make libs-test`, `make test-web` et `make check-types` passent ;
- la taille du parc est mesurée après une conversion à blanc sur une copie.

### Étape B — 30 images/s
- [x] B1 **Tests d'abord**, pour chaque seuil de la section 1 : la même décision, exprimée en millisecondes, aux cadences 100 et 33 ms. Côté Go, un seul helper de conversion des millisecondes en images (vérifier d'abord qu'il n'en existe pas). Côté web, `msToFrames` (`lib/replay/replayLogic.ts:348`), déjà utilisé par `useReplayTiming`.
  Fait, hors `libs/film/replay`, traité par le lot `feat/812-seuils-ms`. Aucun helper Go n'existait (grep `msToFrames`/`MsToFrames`/`ImagesPourDuree` : rien) : `replayverite` porte le sien, `(*Document).images(ms)` (arrondi vers le bas, pas du document, 100 ms si absent) ; `analysis/tactical` travaillait déjà en ms. Tests : `replayverite/seuils_cadence_test.go` (conversion, durée réelle des cinq seuils, décision à la limite et une image au-delà, aux pas 100 et 33), `tactical/sequence_test.go` `TestPositionA_ToleranceDeBordEstUneDuree`, `replayartifacts/raster_test.go` (cadence historique), web `model/seuilsEnMs.test.ts` (27 cas : durée réelle et décision à la limite à 100 et 33 ms, valeur historique en images à 100 ms). Rouges d'abord (symboles absents, et décisions à 33 ms pour les sons).
- [x] B2 **Convertir les seuils Go**. Le pas maximal des projectiles devient une vitesse : 100 m/s × pas.
  Fait pour les sites de `apps/go-api`, hors `libs/film/replay`, traité par le lot `feat/812-seuils-ms` (`zone_attribution.go`, `projectiles_tirs.go`, `vehicle_turrets.go`, `projectiles.go` et son pas en mètres, `t0_film.go`). Convertis : `replayverite/seuils.go` (`toleranceTranslocationMS` 200, `toleranceVieMS` 500, `toleranceEchantillonMS` 100, `toleranceMortMS` 200, `margeFinDeFilmMS` 200 : mêmes durées que 2, 5, 1, 2, 2 images à 100 ms ; valeurs de mesure gardées, la re-mesure à 33 ms est B4) ; `tactical/sequence.go` (`toleranceDeBordImages = 1` devient `toleranceDeBordMs = 100`, indépendante du pas) ; `replayartifacts/raster.go` (la relecture d'un artefact sans `frameIntervalMs` prend `cadenceHistoriqueMS = 100`, constante unique partagée avec `positions.go`, ne dépend plus de `replay.DefaultFrameIntervalMS`). Laissé : `cmd/zone-attribution` (`-max-gap` en images, défaut `replay.DefaultMaxGapFrames` du lot `libs/film/replay` ; `nullShiftFrames = 300`), outil de recherche, voir Découvertes.
- [x] B3 **Convertir les 11 constantes web** et vérifier `placementWall.ts:110`.
  Fait, 13 constantes (les 11 du plan, `skullSound.ts` et `zoneSound.ts` en portant chacun une ou deux) : `BOMB_BLAST_HOLD_MS` 1200 (`bombBlastHoldFrames(doc)`), `FLAG_CAPTURE_HOLD_MS` 600 (`flagCaptureHoldFrames(doc)`), `PROJECTILE_TAIL_MS` 700 (fenêtre `{frame, tailFrames}` de `drawProjectilesLayer`), `CARRIED_PULSE_PERIOD_MS` 2600 et `BLINK_PERIOD_MS` 1000 (le pas descend par le `style.frameMs` des quatre calques de glyphes portés), `HEADING_INTERP_MAX_SPAN_MS` et `VEHICLE_AIM_HOLD_MS` 1000 (paramètre `frameMs` des fonctions de cap et de visée, `time.frameMs` du calque), `TOLERANCE_MS` 1200 et `RETOMBEE_MS` 6000 (crâne), `TOLERANCE_MS` 2000 (zone), `NATIVE_PICKUP_MATCH_MS` 500 (arme), convertis par `msToFrames(…, doc)`. Durée à 100 ms identique à l'ancienne en images. `placementWall.ts:110` : seuil en mètres (distance au dernier point), indépendant du pas : commentaire corrigé seulement. `padPresenceRefine.ts` : `PAD_BETWEEN_FRAMES = 1/64` est une fraction d'image qui garde le sentinel libre, pas une durée : renommée `PAD_BETWEEN_EPSILON` (commentaire corrigé) ; de même `ETA_MIN_FRAMES` (nombre d'images encodées d'un export) devient `ETA_MIN_ENCODED`, pour que le garde-rail D-6 ait une allowlist vide.
- [ ] B4 **Re-mesurer à 33 ms** les seuils qui sont des mesures de corpus : `replayverite/seuils.go` et le plancher de bruit de `t0_film.go`. Les valeurs retenues et leur mesure vont dans le commit. Si une mesure contredit la valeur historique convertie, remonter au user avant de trancher.
  Mesures de préparation faites le 2026-11-11 (aucun code livré) : section 10. Restent les décisions du user sur `toleranceVieMS`, le comptage V-2 et `t0FilmCumulM`, puis les valeurs retenues dans le commit.
  - [x] côté LevelUp, 2026-11-12 (branche `feat/rejeu-30ips-bascule`, local) : `replayverite` `toleranceVieMS` 650, `toleranceMortMS` 250, V-2 par passage ; les 12 témoins à 33 ms contre 100 ms rendent 12 « ok » (journal) ;
  - [~] `t0FilmCumulM` (0,5 m conservé) et `t0FilmJumpM` (50 m/s) : côté décodeur, déjà dans la v1.3.0 (`replay/t0_film.go`) ; à reconfirmer au gate du corpus.
- [x] B5 **Garde-rail D-6.** Aucune constante numérique suffixée `Frames` / `Images` / `_FRAMES` dans `libs/film/replay`, `analysis/tactical`, `replayverite` et `features/match-replay`. Allowlist vide, sauf les outils de recherche.
  Fait, hors `libs/film/replay`, traité par le lot `feat/812-seuils-ms` (le périmètre `libs/film/replay` s'ajoutera à `racinesSeuilsEnMillisecondes` quand ce lot aura fusionné). Go : `archlint/no_seuil_en_images_go_test.go` (`analysis/tactical`, `replayverite`, `sync/replayartifacts` ; constante ou variable à valeur numérique suffixée `Frames`/`Images` ; allowlist vide ; auto-preuve du détecteur ; limite lexicale écrite en tête). Web : `features/match-replay/test/seuilsEnMs.guard.test.ts` (const/let/var `*_FRAMES` ou `…Frames` initialisée par un littéral numérique, sources hors tests ; allowlist vide ; auto-preuve). Les outils de recherche sous `cmd/` ne sont pas balayés.
- [ ] B6 **Bascule.**
  - `DefaultFrameIntervalMS = 33` ;
  - `HistoricalFrameIntervalMS = 100`, utilisé par `raster.go:115` et tout défaut de relecture ;
  - `SchemaVersion` à 109, avec son entrée de chronique ;
  - les commentaires périmés listés par l'inventaire sont corrigés ;
  - les goldens sont régénérés, chaque écart expliqué.
  État au 2026-11-12 : le côté décodeur est livré dans `halo-infinite-film-decoder` v1.3.0 (`DefaultFrameIntervalMS` 33, `HistoricalFrameIntervalMS` 100, `SchemaVersion` 111). Côté LevelUp, préparé en local : fixtures de contrat 111 et instantané des sources, tolérances du banc, `zone-attribution`, tests de cadence web ; reste la montée du `require` (dernier commit, au recalage).
- [ ] B7 **Recuire la liste fermée** par `backfill-replay --match` (et non `replay-build`). Vérifier que les compteurs de joueur sont présents. Si ce n'est pas le cas, en trouver la cause avant d'aller plus loin.
- [ ] B8 **Témoins APRÈS sur la liste fermée.**
  - Différences attendues seulement : axe `T` et nombre de points. Effets attendus : vies et tirs, rattachements, comptes de dérivés à ± 1 image de quantification.
  - Toute autre différence est expliquée ou corrigée.
  - Le gate de corpus et `replay-equiv` passent.
- [ ] B9 **Porte visuelle du user** sur 3 matchs de la liste (Assassin, CTF, Bastions). Il vérifie quatre points :
  - fluidité ;
  - durée des effets (explosion de bombe, capture de drapeau, clignotement, traînées) ;
  - sons (crâne, zone, ramassage) ;
  - fiches avec compteurs.

- [x] B10 **Applications de bureau déjà installées (décision D-9).** Vérifier d'abord sur pièces le comportement de la file `replayregen` (décompte au démarrage, démarrage sur geste, `isStale` qui exige le film archivé). Tests d'abord, puis : à la fin d'une mise à jour des rejeux lancée par l'utilisateur, la file compresse par la porte les artefacts encore en `.json` qu'elle n'a pas pu refaire (film non archivé). Jamais au démarrage, jamais sans ce geste : le garde-rail `no_auto_replay_compress_test.go` reçoit ce second appelant, nommé et justifié. Compte rendu visible dans l'état de la file (refaits, compressés, échecs). Vérification du user sur une application installée : après mise à jour, les anciens rejeux s'ouvrent ; l'écran propose la mise à jour des rejeux ; à la fin, plus aucun `.json` brut dans le dossier des rejeux et la place occupée a baissé.
  Fait le 2026-11-11 (journal) :
  - [x] vérification sur pièces du comportement de la file (journal) ;
  - [x] compression en fin de passe : `replayregen/compress.go`, par `replaybuild.CompressDirWith` (porte avant chaque fichier : pause, mode jeu, synchro ; un fichier à la fois) ;
  - [x] état de la file : `compressing`, `compressed`, `compress_failed` (type de domaine, expvar `replay_regen_compressed_total` et `_compress_failed_total`) ; web : une phase « compression » et deux lignes de compte rendu dans l'écran existant (FR et EN) ; type web écrit à la main (la route n'est pas dans `openapi.yaml`, aucun diff à régénérer) ;
  - [x] garde-rail `no_auto_replay_compress_test.go` : deux appelants nommés et justifiés ;
  - [x] exFAT et FAT32 : repli par création exclusive (journal) ;
  - [x] garde-rail D-6 étendu à `libs/film/replay` (deux entrées d'allowlist datées : la grille elle-même).

- [x] B10 bis **Offre de compression seule (décision du user, 2026-11-11).** Le décompte expose `raw` (rejeux encore en `.json`, comptés par le listage de la porte, sans ouvrir un fichier ; un match aux deux formes ne compte pas). L'offre apparaît si `stale > 0` OU `raw > 0` ; quand seuls des bruts restent, le texte et le bouton disent « compresser » (FR et EN), et la passe ne fait que compresser. Même geste, même file : le décompte ne compresse rien (`TestCompression_JamaisSansLeGeste`, garde-rail `no_auto_replay_compress_test.go` inchangés). Tests : `TestDecompte_ExposeLesBrutsSansLireLeurContenu`, `TestPasseDeCompressionSeule`, `TestDecompteFinal_LesBrutsEnEchecRestentComptes`, web `replayRegen.test.tsx`.

Gate B : B9 validé par un message du user, plus les commandes du gate A.

### Étape C — Parc (gestes du user)
- [ ] C1 Le user lance `levelup replay compress` sur le parc local.
- [ ] C2 Le user lance la recuisson : la file du bureau, ou `backfill-replay --only-existing`, séquentiel.
- [ ] C3 Rejouer les dérivés, dans l'ordre déjà en vigueur : `usage-summary` (révision montée à `us8`), puis `pad-tiers --force`, puis `killsource`, plus les rasters tactiques. Le user est prévenu de chacun.
- [ ] C4 Copier les rejeux vers la prod, et seulement eux. Déployer le serveur et les ouvriers ensemble : un ouvrier à un schéma antérieur voit ses dépôts refusés.

### Étape D — Compaction sur le réseau (conditionnelle)
- [ ] D1 **Mesurer.** Taille transférée, plus temps d'analyse et de normalisation dans le navigateur, sur le plus gros rejeu du parc à 30 images/s.
- [ ] D2 **Seuil.** On n'implémente que si la réponse compressée dépasse 2 Mo ou si analyse et normalisation dépassent 300 ms. Sinon, l'item est clos `[!]` avec les mesures.
  - Si on implémente : un encodage générique auto-décrit, où le fichier porte l'échelle de chaque champ.
  - Il vit dans `replayview` et dans `normalizeReplayDocument`.
  - Le reste de l'application voit les mêmes objets.
  - Le premier test est l'absence : `h` manque sur 42 % des points, `sh` sur 76 %, et « absent » n'est pas « 0 ».

### Étape E — Clôture
- [ ] E1 Mettre à jour `docs/COMMANDS.md` (FR et EN) pour `replay compress`, l'ADR 0034 (cadence et stockage) et `.ai/thought_log.md`.
- [ ] E2 Dresser la liste des reports et des découvertes.

## 5. Pour faire évoluer le décodeur (contrat d'évolutivité)

- **Ajouter ou corriger un champ** : un champ dans le type Go et son tag. Le stockage et la cadence ne bougent pas, et il n'y a pas de codec par champ à tenir.
- **Combler un trou de données** : recuire. La porte de lecture et d'écriture n'est pas concernée.
- **Changer à nouveau de cadence** : une constante et une montée de version. Les seuils sont en millisecondes et un garde-rail le tient.
- **Changer de compression** : la porte seule.

## 6. Risques

| Risque | Parade |
|---|---|
| Un seuil en images oublié | B5 (garde-rail) et B8 (témoins) |
| Une re-mesure de seuil contredit l'historique | Arrêt et remontée au user (B4) |
| Fiches sans compteurs après recuisson | B7 bloquant |
| Ouvrier non redéployé | C4 : serveur et ouvriers ensemble |
| Mémoire pendant la recuisson | Recuisson séquentielle, lancée par le user |
| Navigateur lent sur un gros rejeu | Mesure D1 |

## 7. Hors périmètre

Courses de drapeau (non validé). Mainmise, préréglages de vue, fil d'événements, fiche joueur et graphiques : plans séparés. Dessin des véhicules : planche d'essai des pions.

## 8. Découvertes

- **2026-11-10, étape 0 : trouvailles d'exécution** (aucune corrigée, aucun code touché).
  - **Montage des données d'un worktree neuf.** Il manque `data/titles/halo_infinite/reference/generated/` (`map_callouts.json`, `map_weapon_pads.json`, ignoré par git, rempli par chaque poste). Sans lui, `backfill-replay` cuit sans erreur mais perd les points d'apparition des cartes « - Ranked » : `4f862258` rendait `pickups.origin` 51 -> 8 et `spawnPointsState` `map_absent`. Un témoin AVANT fait ainsi aurait été faux sans le dire. Copié depuis `LevelUp/` (identique au parc). `replay-equiv` n'en dépend pas (catalogue hermétique). Les bases `shared_matches_v2.duckdb` et `metadata.duckdb` (+ `.wal`) viennent de `LevelUp-wt-812-film` ; `metadata.duckdb.wal` existait (642 716 octets, postérieur au `.duckdb`) et a été copié avec, sans être rejoué.
  - **`data/investigation/gate_30ips/` n'est pas ignoré par git** (`git status` le montre `??`) : le plan et la consigne le disent ignoré. Aucun `.gitignore` modifié ; à ajouter avant de commiter l'étape A, ou à écarter du commit.
  - **Les `backfill-*` ouvrent la base partagée en écriture même en `--dry-run`** (cycle de migration au démarrage : `migration: cycle terminé applied=0`). Aucune ligne écrite (`0 ecrits`), mais les octets de `shared_matches_v2.duckdb` changent d'une commande à l'autre : comparer des comptes et des sorties, pas des sommes de contrôle de base.
  - **`replay-diff` n'a pas de mode « empreinte »** ni de `-tout` qui liste les mesures identiques : l'empreinte complète s'obtient en comparant à `{}`. Si l'étape B veut une comparaison fine, un mode `-empreinte` serait utile (non fait).
  - **Écart code-parc.** Les 12 recuits au code de la branche ne diffèrent du parc 107 que par `schemaVersion`, sauf `82f29378` et `be565134` (gain « tourelle fixe » de l'étape 4 du plan v8.1.2). Aucune perte.
  - **`make replay-equiv` dure 1253 s** (21 min) sur un poste chargé, pas ~12 min.
  - **Sans `--force`**, `backfill-vehicle-takes` et `backfill-pad-tiers` répondent « déjà en base » et ne projettent pas : le témoin ne vaut qu'avec `--force`.
  - **`flag-grabs-net` : « sans calque de drapeau »** pour `82f29378` (BTB CTF, `flagCarries` vide) : ce match ne teste pas la partie drapeau ; `008e1bba` et `e157a672` la tiennent.
- **2026-11-10, étape A : trouvailles d'exécution** (aucune corrigée hors périmètre).
  - **Estimation de taille du parc.** Simulation de `replay compress --dir` sur le parc réel (lecture seule) : 1 445 artefacts (les « 1 574 fichiers » du dossier comptent aussi 129 marques `.derived.json`), 3,6 Gio -> 745,9 Mio, soit un rapport de 0,20 et environ 0,73 Gio, pas les 1,8 Go estimés en section 0 (cette estimation supposait un rapport de 0,5).
  - **`data/investigation/` est ignoré par git** (`.gitignore` ligne 117) : la remarque de l'étape 0 n'a plus lieu d'être.
  - **Un worktree neuf n'a pas `apps/web/node_modules`** : `npm ci` (47 s) est requis avant `make check-types` et `make test-web`.
  - **Les sidecars `rasters/{short8}.json` restent bruts** (hors périmètre : D-1 ne les compresse pas) ; leur extension est la constante `title.ReplaySidecarExt`.
  - **Cache du gate de corpus** (`replay-corpus-gate`) : ses entrées `.artefact.json` restent du JSON brut et leur somme porte sur le JSON décompressé, identique à celle des entrées d'avant : aucune invalidation.
  - **Doublons au parc** : `replay compress` ne supprime jamais l'un des deux fichiers d'un match qui a `.json` et `.json.gz` (le compressé gagne à la lecture) ; lequel est le bon est un geste de l'opérateur.
  - **Test instable hors périmètre** : `mapcatalog.TestAddOverlayEntryConcurrentNePerdPasDEntree` a échoué une fois sous charge (« verrou d'écriture tenu trop longtemps — passage forcé ») ; non corrigé.
  - **Les tests de corpus Go** (`REPLAY_CORPUS`, `cadence_corpus_test.go`, `grenade_corpus_test.go`) listaient aussi les marques `.derived.json` comme des artefacts (`filepath.Ext == ".json"`) : le listage par la porte les écarte.
- **2026-11-10, étape B (B1, B2, B3, B5 : côté page et serveur) : trouvailles d'exécution** (aucune corrigée hors périmètre).
  - **Commentaire faux sur `RETOMBEE_FRAMES` (`skullSound.ts`).** La constante valait 60 images, soit 6 s à 100 ms, sous un commentaire qui disait « une seconde ». Comportement gardé (6 s : `RETOMBEE_MS = 6000`), commentaire corrigé (la vérité terrain donne 5 s entre la mort dans le vide et la réapparition au socle).
  - **Repli de `msToFrames` / `frameToMs` côté web : 60 images par seconde** quand `frameIntervalMs` manque, alors que l'histoire des fichiers est 100 ms (le serveur, lui, relit à 100 ms depuis cette étape, `cadenceHistoriqueMS`). Les seuils convertis suivent ce repli sur un document sans pas : un artefact antérieur au champ aurait des tolérances de son six fois plus larges qu'avant. Aucun artefact du parc n'est concerné (schéma ≥ 20 publie le pas ; le seul témoin sans pas est un test). Un test de son (`skullSound.test.ts`) s'appuyait sur ce repli sans le dire : le document de test porte désormais `frameIntervalMs: 100`.
  - **`replayverite`, document sans pas.** V-1 (saut) lit un document sans `frameIntervalMs` comme un pas nul et ne juge rien ; les tolérances converties le lisent à 100 ms. Écart préexistant côté V-1, non corrigé.
  - **Tolérance de bord de la séquence Tactique.** `toleranceDeBordImages = 1` (33 ms à 30 images/s, commentaire « la réplication s'arrête ~34 ms après la mort ») devient 100 ms selon la consigne : à 33 ms elle est trois fois plus large que l'ancienne image. À confirmer à la re-mesure (B4) ; aucun effet à 100 ms.
  - **`cmd/zone-attribution`** garde `-max-gap` en images (défaut `replay.DefaultMaxGapFrames`, `libs/film/replay`) et `nullShiftFrames = 300` : outil de recherche, converti avec le lot `libs/film/replay` s'il y a lieu.
  - **Gates sous charge.** Lancés ensemble, `go test ./...` et `npm run test:run` ont donné des délais dépassés (16 gardes à balayage d'arbre à 5 s côté web ; archlint, ops, duckdb, sync à 10 min côté Go) : chaque échec relancé seul est vert (voir Journal).

- **2026-11-11, B10 : trouvailles d'exécution** (aucune corrigée hors périmètre).
  - **L'offre ne voit pas les bruts non refaisables.** `stale` ne compte que les rejeux refaisables (film archivé). Un utilisateur dont tous les bruts sont non refaisables n'a pas d'offre, donc pas de compression. À traiter si le cas existe : compter aussi les bruts dans l'offre.
  - **Pause pendant la compression.** Elle efface le drapeau de reprise (comportement existant) : après un arrêt, une passe en pause ne reprend pas ; les bruts restants attendent une nouvelle passe.
  - **Garde-rail D-6 sur `libs/film/replay`** : deux constantes en images subsistent, la grille elle-même (`vehicleSampleStrideFrames = 1`, `vehicleRelayMarginFrames = 0`) ; allowlist datée. `DefaultMaxGapFrames` est dérivée d'une durée, sans littéral : elle passe.
  - **Test chronométré instable hors périmètre** : `pkg/duckdbbackup` `TestExportTargetTablesHeld_TableTimeoutMakesItPartial` (échéance d'une nanoseconde) a échoué une fois sous charge, vert 3 fois isolé.

- **2026-11-11, revue de B10 : découvertes** (aucune corrigée).
  - **Durées converties à `msToFrames` avant l'étape B** (`TIMING_MS`, fenêtres de zone, de vie, de fiche, de pastilles, etc.) : un document sans pas les lit toujours à 60 images par seconde. Aucun artefact du parc n'est sans pas ; à unifier avec `durationToFrames` si le besoin se présente.
  - **Un brut en échec reste compté** : `raw > 0` après une passe tant qu'un `.json` illisible traîne, donc l'offre de compression reparaît (au pire à chaque lancement) ; « Plus tard » la masque pour la session.
  - **`internal/archlint` dépasse 10 minutes** sous charge (processeur saturé par d'autres sessions) : lancé seul avec `-timeout 40m`.

- **2026-11-12, préparation locale du côté LevelUp : découvertes** (aucune corrigée).
  - **Le rebase demandé sur `origin/feat/decodeur-v1.2.2` rejouait aussi huit commits d'architecture (lot 4.4) de `release/v8.2.0`**, absents de cette branche : refait avec `--onto` pour ne rejouer que le commit de doc du plan.
  - **Fixtures de contrat : 7 sur 8 ne gardent qu'un point de piste sur 15** (`pointsStride` du manifeste) : seul `000d5950` porte le poids réel des pistes ; les mesures de poids sur fixtures sont donc à lire avec cette réserve.
  - **`replayartifact/gate_test.go`** écrit `"schemaVersion":108` en littéral dans des octets opaques (la porte ne lit pas le schéma) : laissé.
  - **Commentaires qui citent encore 100 ms comme mesure historique** (`positions.go`, `isolation_facts.go`, `squad_emprise_placement.go`, `steps_shared_match_lives_placement.go` : grille de 100 ms propre au calcul, sans lien avec la cadence du rejeu) : laissés, ils décrivent des mesures datées ou une grille de calcul distincte.
  - **`MIN_RENDERABLE_SCHEMA_VERSION` (27)** non relevé : les rejeux 108 à 110 restent affichables et marqués « à recuire ».

## 9. Journal

- **2026-11-10, lancement.** Go du user (« vas-y orchestre l'implem, revue adversariale à la fin »). Le plan v8.1.2 n'est pas clos : ses étapes 1, 2, 4, 5 sont fusionnées dans `release/v8.1.2`, son étape 6 (bascule sur les modules publiés, qui retire `libs/`) reste à faire par une autre session. Aménagement de la dépendance de la section 3 : branche `feat/rejeu-30ips` créée depuis `origin/release/v8.1.2` (`afb963994c`), worktree dédié `LevelUp-wt-30ips`. Les étapes 0 et A (hors `libs/film`, sauf un test) s'exécutent maintenant ; l'étape B, qui modifie `libs/film/replay`, attend la bascule de l'étape 6 de la v8.1.2 (dépendance explicite, report valide). `SchemaVersion` vaut déjà 108 sur cette base : la bascule de cadence le portera à 109.
- **2026-11-10, étape 0 (témoins AVANT) : exécutée, à vérifier par le user.** Liste fermée de 12 écrite sous 0.1 ; témoins rangés sous `data/investigation/gate_30ips/avant/` (non versionné, `MANIFEST.sha256` en tête, commandes exactes dans `COMMANDES.md`). Code : `feat/rejeu-30ips`, HEAD `8b5fc378c7`, schéma 108, 100 ms. Résultats : 12 recuisons en 78 s ; `replay-equiv` 20 identiques en 1253 s ; `replay-verite` 12 « ok » ; empreintes et écart au parc 107 sans perte ; simulations des dérivés sans échec. Reste `[!]` : la séquence de mort Tactique réelle (serveur et base joueur requis), remplacée par une mesure de la séquence calculée sur artefact. Le premier jeu de témoins, fait sans `reference/generated/`, a été jeté et refait (section 8). Gate de l'étape 0 : à clore par le user après vérification ; aucune étape A engagée.
- **2026-11-10, étape A (porte unique et compression) : exécutée, gate consigné ci-dessous.** Branche `feat/rejeu-30ips`. Écart au libellé de A2 : la porte est le paquet `internal/replayartifact` (raisons en A2). Témoins APRÈS sous `data/investigation/gate_30ips/apres_A/` (non versionné) : 12 recuisons une à une depuis un dossier vidé et des faits supprimés (décodage complet, `depuis_les_faits=false`) puis depuis les faits : les 12 `.json.gz` sont identiques octet pour octet entre les deux, et leur contenu décompressé a le sha256 des références de l'étape 0 (`verdict_artefacts.txt`) ; `.meta` identiques hors `bytes` (taille stockée) ; `replay-diff` (empreintes, écart au parc 107), `replay-verite` (12 « ok », sorties identiques), dérivés en simulation (4 commandes) et `tactical-rasters` identiques aux sorties AVANT, hors chemins, extension `.gz` des étiquettes et durées. `replay compress` sur les 12 références copiées dans un dossier temporaire : simulation (rien d'écrit), conversion (17,7 Mio -> 3,6 Mio, 12 convertis, 0 échec), contenu décompressé identique (12/12), seconde passe 0 converti. Parc réel mesuré en lecture seule : voir Découvertes. Les sorties des gates sont rangées sous `apres_A/gate_*.txt`. **Gate A, commandes lancées pour de bon** : `make go-api-test` exit 0 ; `go test ./... -count=1` (go-api) 219 paquets ok, seuls échecs rencontrés sur deux passes : `TestCompressDir_ArretSurContexteAnnule` (test écrit par l'étape, rendu non déterministe par un délai de 1 ns, corrigé par une annulation explicite, 3 passes vertes) et `mapcatalog.TestAddOverlayEntryConcurrentNePerdPasDEntree` (verrou forcé sous charge, paquet non touché, vert 3 fois isolé et dans la passe d'intégration) ; `go test -tags=integration -p 1 ./...` exit 0 (222 paquets ok) ; `make libs-test` exit 0 (39 paquets ok, 0 FAIL) ; `make test-web` 1 079 fichiers, 11 175 tests passés (23 sautés par construction, variables de mesure absentes) ; `make check-types` exit 0 (cache `.tmp` purgé) ; `make go-api-lint` 0 issues, exit 0 ; `make replay-equiv` 20 identiques, 0 différent, exit 0. Garde-rails de l'étape : verts. Commit « étape A » sans push.
- **2026-11-10, revue de l'étape A (une ronde, chaque constat vérifié sur pièces).** Tests écrits avant les corrections : rouges par compilation (champs de compte rendu, points d'injection et `WriteIfAbsent` absents) pour P1-1, P1-2 et la garde de P1-3 (rouge à l'exécution : `orchestrate.go:137` signalé), rouge à l'exécution pour P2-3 (3 sous-cas échouent quand `checkStored` est neutralisé) ; le test de course échoue aussi à l'exécution quand la pose exclusive est remplacée par un `rename` (écrasement).
  - **P1-1 corrigé (confirmé).** Une suppression refusée ou un process mort entre la pose et la suppression laissait un doublon classé « laissé tel quel » à jamais. Maintenant : doublon dont le `.gz` décodé est identique au `.json` = `.json` supprimé (« résorbé », aussi en simulation) ; contenus différents = rien de supprimé, compté à part (`Different`) avec un message ; un `Remove` qui échoue = « à relancer » (`ToRetry`, code de sortie non nul), pas un échec de conversion ; le message final dit chaque cas. Tests : `TestCompressDir_ResorbeUnDoublonIdentique`, `..._NeTouchePasUnDoublonDifferent`, `..._EchecDuRemoveEstARelancerPuisResorbe` (état identique à un crash entre pose et suppression, puis relance), `cmd_replay_test.go` `..._DoublonsIdentiqueEtDifferent`.
  - **P1-2 corrigé (confirmé).** `atomicfile` n'a pas de pose exclusive : `replayartifact.WriteIfAbsent` écrit un temporaire, fsync, puis le pose par lien dur (atomique, échoue si la cible existe) ; un `.gz` existant n'est jamais remplacé, la conversion juge alors le doublon ; `ErrNotExist` au `Remove` du brut est toléré. En-tête de `cmd_replay.go` corrigé (il disait « serveur arrêté de préférence, un serveur réécrirait » : faux désormais). Test de course par injection (`apresLectureDuBrut`, sans délai) : `TestCompressDir_ServeurQuiPoseUnCompressePendantLaConversion` ; et `TestEcritureSansEcraserNePoseJamaisParDessusUnCompresse`.
  - **P1-3 corrigé (confirmé).** `replay-corpus-gate/orchestrate.go:137` faisait `os.Stat` du chemin `.json.gz` : un parc brut était cru absent ; passe par `replayartifact.Locate`. `cmd/mappos-build` lit par la porte. Garde-rail durci (`no_artefact_bytes_outside_gate_test.go`) : outre l'argument et le nom de fonction, il suit les identifiants affectés (`:=`, `=`, `var`, `range`, point fixe) depuis une expression qui nomme un artefact ou un rejeu ; motif étendu à « rejeu ». Limite écrite dans le test : règle lexicale, intra-fonction ; un chemin reçu par paramètre, champ ou retour sans nom parlant, dans une fonction au nom neutre, passe (c'est pourquoi `mappos-build`, dont les chemins viennent de `os.Args`, est tenu par sa migration). Le commentaire de doc des fonctions n'est PAS lu (essayé : il cite des artefacts à propos de sidecars, de faits et d'index, 9 faux positifs).
  - **P2-1 corrigé (confirmé).** `docs/RUNBOOK_DEPLOY_CHECKLIST.md` expédie les deux formes (`--ignore-missing-args`) ; `cmd/mappos-build` migré ; `sweep_vehicle_weapon_tags.mjs` lit les deux formes (octets magiques, le compressé gagne) et échoue (code 2) sur 0 document (vérifié : dossier vide, code 2 ; parc de 12 `.json.gz`, lu). Commentaires de `config/replay_corpus.toml` et de deux `capabilities.toml` mis à jour. Garde-rail littéral : étendu aux fonctions dont le NOM parle d'artefact ou de rejeu ; il n'existe pas de règle simple pour « `HasSuffix(x, ".json")` sur un listage du dossier des rejeux » (dossier reçu par paramètre, `.json` omniprésent dans les mêmes fichiers) : limite écrite en tête du test.
  - **P2-2 corrigé (confirmé).** Le `.gz` reprend la date du `.json` (`Chtimes`) ; le `.meta` est écrit APRÈS, donc jamais plus ancien que l'artefact. Test : `TestCompressDir_ReporteLaDateEtLeMetaResteFiable` (date du `.json` reculée de 72 h, `.meta` non plus ancien, compteur « artefact relu » inchangé : le `.meta` fait foi sans relire l'artefact, ce que l'ancien test ne prouvait pas).
  - **P2-3 corrigé (confirmé).** `replayartifact.Read` refuse tout fichier vide et tout fichier nommé `.json.gz` sans en-tête gzip (erreur journalisée + compteur) ; un `.json` brut reste accepté. Conséquence : une trentaine de tests qui écrivaient du JSON brut sous un nom `.json.gz` passent par la porte (`replayartifact.Write`) : helpers de `service`, `replayregen`, `sync/replayartifacts`, `ops`, `cmd/levelup`.
  - **P2-4 corrigé (confirmé).** Commentaires qui citaient `artifact_gate.go`, `replaybuild.ReadArtifactBytes` et `ArtifactExists` ; `docs/COMMANDS.md` (255, 741) et `docs/FR/COMMANDS.md` (262, 767) décrivent `.json.gz`.
  - **Tests ajoutés** (demandés) : crash simulé entre pose et suppression puis relance ; échec de `Remove` ; annulation de contexte EN COURS de parc (`TestCompressDir_ArretEnCoursDeParc`, coupure après le premier fichier) ; `.meta` jugé fiable sans relecture.
  - **Hors constats** : des exécutables de build (`apps/go-api/*.exe`, ignorés par git) avaient été laissés par mes `go build` sans `-o` : supprimés.
- **2026-11-10, page des vérifications du user.** La porte visuelle B9 est inscrite « à venir » dans la page unique tenue par la session pilote de la v8.1.2 : https://claude.ai/artifact/P9VJKKbSDsQu1PUBjyGwTv (collection `verifs`, document `30ips-rejeu`). À B9 : la passer à `a_faire` (lire la version d'abord) et y mettre les liens des trois matchs. La bascule de la v8.1.2 attend encore un contrôle à l'écran du user et la fin d'un chantier d'architecture ; la session pilote préviendra.
- **2026-11-10, amendement (accord du user).** Ajout de D-9 et de l'item B10 : applications de bureau déjà installées. Les anciens rejeux restent lisibles sans rien faire ; la recuisson lancée depuis l'application les réécrit compressés ; ceux qui ne peuvent pas être refaits sont compressés à la fin de cette même mise à jour.
- **2026-11-10, contrôle du pilote après la revue de l'étape A.** Commits `cf696a7c93` et `e4c3f8a112` vérifiés (dépôt propre, gates finaux présents sous `apres_A/revue/`). Point à traiter en B10, pas avant : `WriteIfAbsent` pose par lien dur, que exFAT et FAT32 ne savent pas faire. Sur un dossier de rejeux placé sur un tel disque, la conversion échouerait fichier par fichier (comptée, sans perte). B10 devra prévoir un repli sûr ou un message clair, test à l'appui. L'étape A est close ; l'étape B attend la bascule de l'étape 6 du plan v8.1.2.
- **2026-11-10, étape B, lot « seuils en millisecondes côté page et serveur » (B1, B2, B3, B5 ; B4, B6 à B10 non engagés).** Périmètre accordé par le user : `apps/go-api` et `apps/web` seulement ; aucun fichier sous `libs/` (les seuils de `libs/film/replay` sont au lot `feat/812-seuils-ms`, le garde-rail Go l'ajoutera à ses racines à la fusion de ce lot). Branche `feat/rejeu-30ips`. Tests écrits avant le code (rouges : symboles absents, et décisions à 33 ms des sons), puis verts. Sites convertis : voir B2 et B3 ci-dessus ; aucun changement à 100 ms (chaque constante garde sa durée, N images x 100 ms). **Gates lancés pour de bon** (sorties sous `data/investigation/gate_30ips/apres_B123/`) :
  - `go test ./... -count=1` : première passe sous charge (web et Go lancés ensemble) 211 paquets ok, 9 en échec par délai dépassé (archlint, filmrun, mapcatalog, ops, platform/duckdb, sync, sync/filmfetch, sync/skill, sync/v2) ; ces 9 relancés seuls, un à un (`-p 1`) : 9 ok (archlint 509 s, avec le nouveau garde-rail) ;
  - `go test -tags=integration -p 1 -count=1 ./...` : exit 0, 222 paquets ok ;
  - `make go-api-lint` : 0 issues sur `apps/go-api` ; la boucle sur `libs/*` du même `make` a dépassé son délai de 5 min (cache froid, modules non touchés) ; relancée à la main avec `--timeout 15m` sur les quatre modules (film, gamefiles, haloapi, himap) : 0 issues chacun ;
  - `npm run typecheck` : exit 0 ; `npm run test:run` : première passe sous charge 1 067 fichiers ok, 14 en échec (16 tests, tous « Test timed out in 5000ms », gardes à balayage d'arbre), relancés seuls : 14 fichiers, 48 tests ok ; `npx eslint src/features/match-replay` : 0 erreur (8 avertissements préexistants, fichiers non touchés) ;
  - **témoins à 100 ms, identiques** (`apres_B123/`) : `replay-verite` sur les 12 matchs, 12 sorties octet pour octet identiques à `avant/replay-verite/` (code 0 pour les 12) ; `tactical-rasters --backfill --dry-run` identique à `avant/derives/` (14 lignes, hors durée) ; séquence de mort (substitut `TestMesureLectureDesPistes`, 12 matchs) identique à `avant/tactique/` hors durées.
- **2026-11-11, B10 (applications de bureau déjà installées) : exécuté.** Branche `feat/rejeu-30ips`, base `release/v8.2.0` (ex `release/v8.1.2`, tête `91727b0c30`, la base est la tête). Rebase fait deux fois : sur la tête du 2026-11-10 (un conflit, `padPickupDating.test.ts` : identité de fixture de la release gardée, nom `PAD_BETWEEN_EPSILON` gardé), puis sur `91727b0c30` (sans conflit ; aucune commande que l'étape A avait migrée n'a été supprimée : `backfill_t0_film`, `mapplafond-mesure` et `zone-attribution` figurent à la liste des commandes conservées ; aucune commande abandonnée). Garde-rail D-6 étendu à `libs/film/replay`, deux entrées d'allowlist datées (la grille elle-même). Vérification sur pièces de la file : le décompte se lance au démarrage du serveur, à la fin d'une passe et sur demande (`Rescan`), jamais par la lecture de l'état ; la file ne part que sur `Start` (geste) ou `ResumeIfPending` (drapeau posé par ce geste) ; `isStale` exige `!UpToDate`, un schéma non plus récent et un producteur non plus récent, et `scanTitle` exige le film archivé : les rejeux sans film ne sont jamais candidats, d'où le trou que B10 ferme. Tests écrits avant le code (rouges par compilation : `CompressDirWith`, `CompressOptions`, `CounterLinkFallback`, `ecrireEtSynchroniser`, champs d'état absents ; rouges à l'exécution côté web). **Choix exFAT : repli par création exclusive**, pas arrêt propre. Raisons : (1) le repli est lui-même exclusif (`O_EXCL`), il ne remplace jamais un fichier ; (2) la porte refuse un gzip incomplet à toute longueur de coupure (test sur 5 longueurs : en-tête seul, moitié, sans somme de contrôle), donc un lecteur concurrent reçoit une erreur, jamais un document ; (3) pour ne pas laisser un `.json.gz` incomplet masquer un `.json` valide, la porte lit alors le `.json` voisin, et la conversion reprend un `.json.gz` incomplet laissé par un arrêt brutal (octets lus mais non décodables, `.json` valide ; jamais sur une erreur de lecture du disque) ; (4) un échec d'écriture supprime le partiel. Un arrêt propre aurait laissé ces utilisateurs sans compression ni message utile. Limite assumée : pendant l'écriture un lecteur peut voir une erreur ou le `.json` voisin. **Gates lancés** (sorties dans le dossier temporaire du poste, non versionnées), machine à 100 % de processeur par d'autres sessions : `go test ./... -count=1` (go-api) exit 0, 214 paquets ok (avant le rebase sur `91727b0c30` : 222 ok, 1 échec chronométré `pkg/duckdbbackup` à échéance d'une nanoseconde, vert 3 fois isolé) ; `go test -tags=integration -p 1 -count=1 ./...` : 214 paquets ok sauf `internal/sync`, arrêté par son délai par défaut de 10 min puis de 20 min sous charge, puis relancé seul avec `-timeout 60m` : ok, 801 s; `make go-api-lint` : `apps/go-api` 0 issues, boucle `libs/` expirée à 5 min (cache froid), relancée à la main avec `--timeout 15m` et les mêmes options : film, gamefiles, haloapi, himap 0 issues ; `make libs-test` exit 0, 39 paquets ok, 0 FAIL ; `make check-types` exit 0 ; `vitest` sur `src/components/replay-regen`, `src/lib`, `src/features/settings` : 2 017 tests ok, 4 échecs « Test timed out in 5000ms » (trois gardes à balayage d'arbre) verts relancés seuls (6 tests) ; `make openapi-gen` : aucun diff (la route n'est pas dans le contrat : type web écrit à la main, `make generate-types` sans objet). Pour le user, sur une application installée : voir le compte rendu.


- **2026-11-11, décision du user (accord sur la recommandation du pilote).** B10 bis : l'application de bureau propose aussi la mise à jour des rejeux quand il ne reste que des rejeux bruts à compresser (aucun rejeu refaisable), avec un texte adapté (libérer de la place). Même geste de l'utilisateur, même file ; jamais au démarrage sans ce geste. À faire avec les corrections de la revue de B10.
- **2026-11-11, revue de B10 (une ronde) et B10 bis : exécutés.** Rebase sur `release/v8.2.0` (têtes `d5b6a33d9d` puis `fe4579354b`, renommage des modules des briques : le seul import `github.com/JGtm/LevelUp/libs/film/replay` de nos fichiers, `replayregen/compress_test.go`, est passé au nouveau chemin ; les chemins `libs/film/replay` des garde-rails `archlint` sont des chemins de dossiers, inchangés ; aucun conflit). Chaque constat vérifié sur pièces, test écrit avant le code (rouge par compilation ou à l'exécution) :
  - **P1 confirmé, corrigé.** `Decode` rend tels quels des octets sans en-tête gzip : `relireCompresse` ne levait donc pas `errContenuIllisible` pour un `.json.gz` de 0 ou 1 octet. `replayartifact.ValidateStored` (vide, ou nommé `.json.gz` sans en-tête) est appelée par `relireCompresse`. Tests à 0, 1 octet et « pas du gzip », en simulation et en conversion : `TestCompressDir_CompresseVideOuSansEnteteEstReprisCommeIncomplet`.
  - **P2-2 confirmé, corrigé.** Constante unique `HISTORICAL_FRAME_INTERVAL_MS = 100`, `durationFrameMs(doc)` et `durationToFrames(ms, doc)` dans `lib/replay/replayLogic.ts` ; `msToFrames`, `frameToMs` et `framesPerSecond` (horloge, repli à 60 images par seconde) ne changent pas. Passent à la conversion de durées les sites de l'étape B : déflagration, onde de capture, traînée des projectiles, tolérances de son (crâne, zone, arme), périodes des glyphes portés (quatre hooks), maintien de visée et interpolation de cap des véhicules (champ facultatif `durationFrameMs` du temps des véhicules, la timeline d'explosion garde `frameMs`). Tests : `durationToFrames.test.ts` (anciens nombres d'images sans pas : 12, 6, 7, 10, 26, 5, 60), `seuilsEnMs.test.ts`, et un garde-rail dans `seuilsEnMs.guard.test.ts` (aucune des six durées de seuil ne se convertit par `msToFrames`). Les autres durées de la page, converties à `msToFrames` avant l'étape B, ne sont pas touchées (voir Découvertes).
  - **P2-3 confirmé, tranché : pas de repli.** `LireDocument` refuse `frameIntervalMs <= 0`, donc le repli à 100 ms de `(*Document).images` n'était atteignable que par un `Document` construit à la main. Règle unique : aucun repli ; `images` rend 0 sans pas, le commentaire le dit, `TestLireDocument_RefuseUnPasAbsent` tient la porte.
  - **P2-4 confirmé, corrigé.** Seul « non pris en charge » mémorise le dossier : `errors.ErrUnsupported` (la bibliothèque le fait correspondre à ENOSYS, ENOTSUP, EOPNOTSUPP et ERROR_NOT_SUPPORTED) plus Windows `ERROR_INVALID_FUNCTION` (1, celui d'exFAT et de FAT32), `ERROR_NOT_SUPPORTED` (50), `ERROR_CALL_NOT_IMPLEMENTED` (120), POSIX `EPERM`, `ENOTSUP`, `EOPNOTSUPP`, `ENOSYS`. Toute autre erreur : repli pour ce fichier seulement, `slog.WarnContext` et compteur `replay_artifact_link_error_total`. Tests par injection de `linkFile`, dont la table `TestLienNonPrisEnCharge` par OS.
  - **P2-5 confirmé, corrigé.** `Read` relit une fois le `.json.gz` avant de conclure à l'illisible quand le brut voisin n'est pas utilisable ; une seule erreur comptée. Tests : `TestLecture_CompresseEnCoursDEcriture_UneRelectureAvantDeConclure`, `..._UneSeuleRelecture`.
  - **P2-6 confirmé, corrigé.** `verifierLaPose` compare d'abord le fichier aux octets que NOUS avons posés ; s'ils diffèrent, rien n'est supprimé et le match est compté `Replaced` (nouveau champ du compte rendu, affiché par la commande), pas un échec. Test de la course par injection (`apresLaPose`) : `TestCompressDir_UnAutreEcrivainQuiRemplaceLeCompresseNEstPasSupprime`. Reste une fenêtre entre cette comparaison et la suppression d'un `.gz` qui est bien le nôtre mais dont le contenu relu diffère : elle ne supprime que notre fichier, vérifié mauvais.
  - **P2-7 confirmé, corrigé.** Un doublon dont le `.json` a été retiré par un autre acteur est compté « déjà compressé », plus « résorbé ».
  - **P2-8 confirmé, corrigé.** La reprise d'un `.json.gz` incomplet exige que le `.json` voisin soit du JSON bien formé (`json.Valid`) ; sinon rien n'est retiré, échec compté et journalisé. Test avec un `.json` vide et un `.json` tronqué.
  - **Test faible.** `TestCompression_UnFichierALaFois` retiré de `replayregen` (la file ne fait qu'appeler la conversion) ; remplacé par `TestCompressDirWith_UnFichierALaFois` au niveau de la conversion : la porte du fichier suivant n'est consultée qu'après le compte rendu du précédent. `TestCompression_PauseEntreDeuxFichiers` rendu robuste à la charge (attend la fin du deuxième fichier).
  - **Gates lancés après le dernier rebase** (`fe4579354b`, processeur saturé par d'autres sessions ; sorties dans le dossier temporaire du poste) : `go test ./... -count=1` : première passe 211 paquets ok, deux échecs, `archlint` (délai de 10 min dépassé sous charge, ok seul avec `-timeout 40m`, 837 s) et `replayregen` `TestCompression_PauseEntreDeuxFichiers` (test trop pressé sous charge, corrigé, 10 passes sous `-race` vertes) ; `go test -tags=integration -p 1 -count=1 -timeout 60m ./...` : exit 0, 215 paquets ok ; `golangci-lint` sur `apps/go-api` avec les options du `make` : 0 issues (le `make` lui-même expire sur la boucle `libs/`, cache froid : relancée à la main, film, gamefiles, haloapi, himap : 0 issues) ; `make libs-test` : exit 0, 39 paquets ok ; `make check-types` : exit 0 ; `make test-web` équivalent (`vitest run`, suite entière) : 1 081 fichiers ok, 5 sautés, 11 208 tests ok, 23 sautés par construction ; `make openapi-gen` : aucun diff (la route de la file n'est pas au contrat). `grep -rn "JGtm/LevelUp/libs" apps libs --include=*.go` : vide.


- **2026-11-12, préparation locale du côté LevelUp (B4, B6 partiel ; branche `feat/rejeu-30ips-bascule` sur `origin/feat/decodeur-v1.2.2`, rien poussé, `go.mod` intact).** Décodeur v1.3.0 (`32a7c3c`) lu par un espace de travail local ; `go build`, `go vet` et les paquets du rejeu verts avant tout changement (la cadence ne casse aucun test LevelUp).
  - `replayverite` : `toleranceVieMS` 650, `toleranceMortMS` 250, V-2 par passage de véhicule ; bulletin figé `bcb6d393` inchangé ; 12 témoins 33 ms contre 100 ms : 12 « ok ». Sortie à 100 ms modifiée là où la mesure l'impose : tolérance de vie de 5 à 6 images, V-2 par passage.
  - `cmd/zone-attribution` : `-max-gap-ms`, `nullShiftMS`, `clockOffsetsMS`.
  - Fixtures de contrat 111 (8 fichiers + manifeste, 4,54 Mio) et instantané `film-source` régénérés ; tests de cadence web (`model/cadenceEquivalence.test.ts`, `lib/replay/cadenceEquivalence.test.ts`).
  - Poids servi (gzip 5 de la réponse projetée, `replayview.FromArtifact`) : `be565134` 1,06 Mo (100 ms, schéma 109) -> 2,59 Mo (33 ms), `tracks` 85 %, `vehicles` 8 %, `projectiles` 3 % ; fixture `000d5950` (1 point sur 1) 1,0 Mo, `tracks` 90 % ; `e5adf7b2` (1 point sur 15) 0,93 Mo, `vehicles` 50 %, `tracks` 25 % ; `bcb6d393` 0,11 Mo. Dans `tracks`, par colonne (gzip 5, `be565134`) : `x` 330 Ko, `y` 325 Ko, `h` 224 Ko (présent sur 66 % des points), `p` 120 Ko, `z` 115 Ko, `t` 80 Ko, `sh` 6 Ko : ces colonnes gzippées à part font environ 1,2 Mo, contre 2,2 Mo entrelacées. D2 : seuil de 2 Mo franchi sur le plus gros ; décision du user, rien implémenté.

- **2026-11-12, le pas entre dans le critère « à recuire » (décision du pilote : la PR 203 monte à v1.3.1 en forçant 100 ms, un rejeu cuit entre les deux fusions porte le même schéma à 100 ms puis à 33 ms).** Règle unique : à jour = schéma courant ET pas (absent = 100 ms) égal au pas de cuisson courant ; source `replaybuild.CookFrameIntervalMS` (`cadence.go`), lue par la cuisson et par `Digest.Verdict`, pas écrit dans `.meta` ; refus du dépôt d'un autre pas ; le serveur pose `X-Replay-Cadence-Stale`, le badge admin le recopie (libellé existant). Points de décision et tests : entrée du 2026-11-12 du thought_log. À l'étape B6 : un rejeu à 100 ms au schéma courant sort « à recuire » dès que la cadence courante passe à 33 ms, sans nouvelle montée de schéma.

## 10. Mesures à 33 ms (B4, préparation)

Faites le 2026-11-11 sur la branche `feat/rejeu-30ips-bascule` (base `release/v8.2.0`, tête `0627a64b65`), sans aucune modification de code versionné (le décodeur n'a pas été touché, la cadence reste à 100 ms dans le dépôt). Témoins et outils rangés sous `data/investigation/gate_30ips/mesure_33ms/` (ignoré par git) : `artefacts/` (33 ms), `artefacts_100/` (100 ms, même binaire), `empreintes/`, `verite/`, `derives_100/`, `derives_33/`, `outil/` (scripts jetables, dont une copie du test jetable `zz_mesure33_jetable_test.go.txt`), `sorties/` (journaux, tableaux). Les 12 matchs de l'étape 0, un à un, jamais sur le parc.

### 10.1 Comment cuire à 33 ms, avec les compteurs de joueur

- **`backfill-replay` n'a aucun réglage de pas** : il prend `replay.DefaultFrameIntervalMS` (100 aujourd'hui). Il ne produira du 33 ms qu'après le changement de la constante (B6). Voie d'avant bascule : `cmd/replay-build --interval 33`, qui existe toujours (aucune commande n'a été ajoutée au dépôt).
- **Cause des compteurs absents (B7, levée).** `replay-build` sans `--facts` appelle `BuildMatch` avec `port.MatchFacts{}` (`cmd/replay-build/main.go`, `loadFacts`) : pas de lignes de match, donc pas de pont d'identité ni de `scoreTimeline.players`. Constaté : cuisson de `0076ebdc` à 100 ms sans `--facts` : `scoreTimeline` n'a que `teams`, 0 joueur ; avec `--facts` : 8 joueurs. L'aide de l'option le dit (« sans lui : artefact sans compteurs de joueur ni actions d'objectif »). Ce n'est donc ni un défaut des faits de film ni des identités : c'est l'entrée « faits du match » (celle de la base) que l'essai du 2026-11-09 n'a pas fournie.
- **Voie qui donne les compteurs** : les faits du match exportés par `levelup replay-facts-export` (déjà rangés dans `avant/faits/<short8>.facts.json`) passés à `--facts`. Vérifié **octet pour octet** que cette voie rend le même document que `levelup backfill-replay --match <id> --force` à 100 ms (4 matchs : `0076ebdc`, `ef8cb72b`, `82f29378`, `e157a672`, sha256 du JSON décompressé identiques). Commande exacte (racine « bac à sable » pour ne pas écrire dans `data/cache/replays`, voir `outil/cuire.ps1`) :

      LEVELUP_REPO_ROOT=<mesure_33ms>/racine \
        replay-build.exe --map <nom de carte> --interval 33 \
        --facts <avant>/faits/<short8>.facts.json <matchId complet>

  `racine/` = `config/` et `data/titles/` en jonctions vers le worktree, `data/cache/{film_chunks,film_manifests}` et `data/film_identites` en jonctions de lecture vers `LevelUp/`, `data/cache/replays` et `film_facts` propres à la racine. Le nom de carte est le premier de `mapNames` du fichier de faits.
- **Cuisson depuis les faits de film (C2).** Les `.filmfacts.bin` écrits par une cuisson à 100 ms servent tels quels à 33 ms : sur 3 matchs, le document cuit à 33 ms depuis les faits (100 ms) est identique octet pour octet au document cuit à 33 ms par décodage complet (`e157a672` 2,0 s, `ef8cb72b` 2,4 s, `82f29378` 0,5 s). Les faits ne dépendent pas de la cadence : la recuisson du parc peut partir des faits.

### 10.2 Taille et volume (12 matchs, 100 ms puis 33 ms, même code)

| match | brut Ko | gzip Ko | points de piste | tirs | échant. véhicule | frameCount | durationMs |
|---|---|---|---|---|---|---|---|
| 0076ebdc | 2 107 -> 5 523 | 445 -> 1 112 | 29 357 -> 87 912 | 973 = | 0 | 5 043 -> 15 282 | 504 300 -> 504 306 |
| ef8cb72b | 1 479 -> 3 753 | 295 -> 719 | 19 625 -> 58 666 | 1 608 = | 0 | 3 397 -> 10 292 | 339 700 -> 339 636 |
| 008e1bba | 565 -> 1 403 | 117 -> 277 | 7 710 -> 23 043 | 572 = | 0 | 1 701 -> 5 155 | 170 100 -> 170 115 |
| d2b74083 | 805 -> 1 926 | 164 -> 384 | 11 228 -> 32 877 | 716 = | 0 | 2 506 -> 7 593 | 250 600 -> 250 569 |
| de3cec8b | 906 -> 2 295 | 175 -> 417 | 11 525 -> 34 356 | 924 = | 0 | 2 258 -> 6 841 | 225 800 -> 225 753 |
| 43716616 | 1 189 -> 2 875 | 238 -> 568 | 15 187 -> 45 034 | 1 557 = | 0 | 3 144 -> 9 527 | 314 400 -> 314 391 |
| c75f33b8 | 1 499 -> 3 831 | 288 -> 703 | 21 434 -> 64 096 | 1 413 = | 0 | 4 117 -> 12 474 | 411 700 -> 411 642 |
| e157a672 | 331 -> 786 | 69 -> 157 | 4 066 -> 12 184 | 265 = | 0 | 997 -> 3 019 | 99 700 -> 99 627 |
| 82f29378 | 2 049 -> 5 439 | 442 -> 1 095 | 22 162 -> 65 119 | 728 = | 7 836 -> 22 480 | 1 988 -> 6 023 | 198 800 -> 198 759 |
| ce927bd2 | 711 -> 1 760 | 147 -> 345 | 9 817 -> 29 189 | 554 = | 0 | 2 102 -> 6 370 | 210 200 -> 210 210 |
| be565134 | 5 276 -> 13 866 | 1 051 -> 2 571 | 71 076 -> 205 697 | 2 831 = | 5 430 -> 15 058 | 4 600 -> 13 938 | 460 000 -> 459 954 |
| 4f862258 | 1 631 -> 4 383 | 323 -> 814 | 24 576 -> 73 721 | 974 = | 0 | 4 287 -> 12 990 | 428 700 -> 428 670 |
| **total** | 18 548 -> 47 841 (x2,58) | 3 754 -> 9 160 (x2,44) | x2,89 à x3,00 | identiques | x2,77 à x2,87 | x3,03 | écart <= 73 ms |

- Les tirs ne sont pas échantillonnés sur la grille : leur nombre ne bouge pas. Les points de piste, les échantillons de véhicule, les trajectoires de projectile et les courbes de zone suivent la grille (x3).
- `durationMs` varie de quelques dizaines de ms (la grille de 33 ms ne tombe pas sur la même image de fin) : `frameCount * 33`, pas `frameCount * 100`.
- **Disque.** Gzip niveau 6 : x2,44 sur le fichier compressé. Le parc mesuré à l'étape A (1 445 artefacts, 745,9 Mio compressés à 100 ms) passerait à **environ 1,8 Gio** à 33 ms (le chiffre de la section 0 se retrouve), contre 3,6 Gio bruts aujourd'hui.
- **Réseau (étape D).** Proxy : le document de l'artefact ré-encodé puis compressé niveau 5, sans la projection `replayview`. Le plus gros (`be565134`) passe de **1,06 Mo à 2,59 Mo** compressé : il franchit le seuil de 2 Mo de D2. La médiane des 12 : 0,27 Mo -> 0,64 Mo. À confirmer en D1 sur la réponse réelle, mais D2 ne sera probablement pas clos `[!]`.

### 10.3 Ce qui change dans le contenu (hors axe du temps)

Méthode : `replay-diff` contre `{}` donne l'empreinte de chaque document, comparée métrique par métrique entre 100 et 33 ms (`outil/comparer*.py`, durées ramenées en ms) ; 666 métriques diffèrent sur au moins un match, dont l'essentiel est la densité de grille. Vérifié par ailleurs : **la longueur de chaque calque est identique** (vies par joueur, tirs par slot, grenades, ramassages, objectifs, portages de drapeau, de crâne et de bombe, états de zone, véhicules, roster, changements d'arme, capacités, épisodes d'équipement, tractions) **sauf** `projectiles`, `equipmentPlacements`, `bursts` (-1 sur `82f29378`) et `stances` (-1 sur `4f862258`). Scores finals d'équipe et totaux par joueur (score, frags, morts, assistances) : **identiques sur les 12**.

| Changement | Ordre de grandeur | Explication, vérifiée dans le code |
|---|---|---|
| Points de piste, `h`/`p`/`s`/`sh`/`hp`, échantillons de véhicule et de visée, `objectiveObjects.pts`, `zoneStates.gauge`, `projectiles.p` | x2,8 à x3,0 | densité de la grille |
| `frameCount` | x3,03 | idem |
| `projectiles` publiés | `0076ebdc` 689 -> 1 004, `ef8cb72b` 174 -> 339, `d2b74083` 128 -> 176, `ce927bd2` 42 -> 76, `43716616` 97 -> 113 ; `be565134` 285 -> 283, `de3cec8b` 59 -> 58 | **gain** : `buildProjectiles` écarte une trajectoire d'un seul point de grille (« ne se dessine pas ») ; à 100 ms les vols de moins de 100 ms disparaissaient. Les deux baisses sont des vols coupés plus tôt : `coverage.projectiles.truncated` 4 -> 6 et 0 -> 1, le pas maximal étant 100 m/s x pas (10 m à 100 ms, 3,3 m à 33 ms) |
| `coverage.projectiles.shotLinked` | 235 -> 249, 14 -> 33, 1 -> 11 | suit les projectiles publiés en plus |
| `equipmentPlacements` | `d2b74083` 111 -> 96, `c75f33b8` 151 -> 137, `ce927bd2` 61 -> 68 | **bord de film** : tous les écarts sont des poses nées à la dernière image (`t0 = t1 = dernière frame`, fin `open`) vues par le dernier relevé. À 33 ms la dernière image de grille tombe avant ou après ce relevé ; `endOpen` 19 -> 4, 16 -> 2, 1 -> 3. Aucune pose en cours de match ne change |
| `grappleLines` durées | -6 % à -15 % (`de3cec8b` 1 700 ms -> 1 287 ms) | l'arrivée est la frame de plus grande proximité de l'ancre (`grappleArrival`) : plus précise à 33 ms. Nombre de tractions identique |
| `bounds` | jusqu'à 0,6 m (`0076ebdc` minZ -2,61 -> -3,24) | enveloppe des points : l'extrême est plus souvent échantillonné |
| `t0FilmMs` | -96 à +5 ms | quantisation du premier mouvement, voir 10.5 |
| `coverage.birthLoadouts.snapped` | 23 -> 48, 14 -> 24, 6 -> 17, 5 -> 16 | compteur de quantisation (instant de création ramené dans la fenêtre de la vie) ; `published` n'a pas bougé |
| `coverage.placements.*`, `coverage.pickups.origin*` | +-1 à +-3 | bord de film et quantisation (`pickups.originGround` 9 -> 10 sur `c75f33b8`, 4 -> 3 sur `ce927bd2`) |
| `coverage.continuousFire.shots` | 432 -> 412 (`0076ebdc`), 208 -> 210, 2 035 -> 2 012, 1 338 -> 1 332 | frontières des rafales ; `bursts` `82f29378` 67 -> 66 ; durées des rafales -2,9 % à +1,9 % |
| `coverage.flagCarries.closedByHome/closedByObject` | `008e1bba` 1 -> 0 et 14 -> 13 | les états de drapeau (mêmes porteurs, mêmes états, mêmes bornes à une image près) sont identiques ; seul le compteur de cause de fermeture bouge. À regarder |
| `coverage.zones.ownerChecked` | `ce927bd2` 17 -> 12, `ownerVoteDisagreed` 0 -> 1 | **à expliquer dans le lot** : `checkOwnerAgreement` lit la valeur du canal propriétaire à la première image à ou après la capture (`zoneValueAfter`) ; à 33 ms cette image tombe plus souvent avant que le canal ait changé (neutre, donc hors dénominateur). Captures attribuées (23 sur 24), états de zone et durées identiques : le nom reste retenu, aucun effet visible attendu |
| `stances` | `4f862258` 693 -> 692 | un intervalle de moins (sprint) |
| `coverage.carriedEquipment.tracked*` | `ce927bd2` : 0 -> 2 positionnés | gain de positions |
| `coverage.vehicles.samplesAfterEnd` | 164 -> 490, 213 -> 566 | densité |
| `coverage.score.points` | +2 (`e157a672`, `ce927bd2`) | points de courbe, scores finals identiques |
| Durées cumulées (`duree-totale`, en ms) | de -0,2 % à +0,8 % pour les calques à intervalles ; `grappleLines` et `bursts` plus larges | convention T1 inclus : une image de plus par intervalle (100 ms contre 33 ms) |

Aucun changement n'est une perte de données. Les « poses de bord » et les contrôles de zone sont les seuls effets à faire valider ; les projectiles en plus sont un gain visible (plus de traînées).

**Les témoins AVANT de l'étape 0 ne valent plus pour B8.** Recuits au code actuel (`0627a64b65`) à 100 ms, les 12 documents diffèrent de `avant/artefacts/` (le code a avancé depuis `8b5fc378c7`, par exemple 5 -> 12 `bursts`, `carriedEquipment` +30 sur `0076ebdc`). Pour B8, la référence à 100 ms est `mesure_33ms/artefacts_100/` (même binaire que le 33 ms, mêmes faits). Les sorties `avant/derives/` restent égales à celles du code actuel (voir 10.6).

### 10.4 `replay-verite` sur les 12 à 33 ms (avant = 100 ms, après = 33 ms, mêmes faits et oracles)

Résultat avec les seuils actuels : **10 « ok », 2 « FAUX »** ; les scores d'oracle (O-K1..3, O-V1, O-V2, O-S1, O-S3, O-T1) et V-1, V-4, V-5, V-6, V-7, V-8 sont identiques sur les 12. Le contrôle « 100 ms contre 100 ms » rend 12 « ok ». Les valeurs absolues ne sont pas 0 sur la référence (par exemple V-2 = 2 sur `0076ebdc`, V-3 = 4) : « 0 violation » veut dire « aucune hausse ».

| Règle | 100 ms -> 33 ms | Lecture |
|---|---|---|
| V-3 `0076ebdc` | 4 -> 5 | une capacité 561 ms avant le début de la vie : à 100 ms le début de vie est arrondi vers le bas et l'écart mesuré tombait à 500 ms, dans la tolérance. Le début de vie est mesuré plus finement à 33 ms |
| V-2 `82f29378` | 424 -> 1 236 | la règle compte des **échantillons** de véhicule hors emprise : x2,9 comme la densité (7 836 -> 22 480 échantillons). Même véhicule, mêmes passages. C'est un défaut de comparaison, pas une dérive |

Tolérances de `replayverite/seuils.go` (distribution mesurée à 33 ms ; entre parenthèses 100 ms) :

| Seuil | Valeur actuelle (effective à 33 ms) | Mesure | Valeur proposée |
|---|---|---|---|
| `vitesseSautMaxMPS` (V-1) | 100 m/s | aucun pas de piste à plus de 40 m/s sur les 12, aux deux cadences : aucun bruit de quantisation à 33 ms | inchangée (pas une durée) |
| `toleranceTranslocationMS` | 200 (198) | aucun saut hors portes de carte sur les 12 : **non mesurable ici** ; à refaire sur les 19 témoins du corpus | inchangée en millisecondes |
| `toleranceVieMS` (V-3, V-5) | 500 (495) | écarts hors vie (12 matchs, 80 actions à 33 ms, 52 à 100 ms) : tirs <= 132 ms (<= 200 ms), grenades <= 99 ms ; capacités : 99, 198, 264, 330, 363, 429, 462, **561 x3, 594, 627**, 726, 957 ms, puis 12 606 ms (capacité) et 16 467 et 100 683 ms (ramassages) (100 ms : 600 x4, 800, 900, 12 700, 16 500, 100 700). Violations sur 12 matchs : **9 à 100 ms, 10 à 33 ms** ; à 630 ms : **5 et 5** | **à trancher par le user.** 630-650 ms donne les mêmes 5 violations aux deux cadences (726, 957 et les trois de plus de 12 s, vrais défauts) ; garder 500 ms accepte une hausse de 1 (la capacité de `0076ebdc`) qui ne tient qu'à l'arrondi de 100 ms |
| `rayonTrajetM`, `rayonPorteM`, `margeEmpriseObjetsM` | mètres | indépendants de la cadence | inchangés |
| `toleranceEchantillonMS` (V-6) | 100 (99) | toutes les instances « passager loin du véhicule » ont leur point de passager le plus proche à >= 330 ms : la tolérance de 100 ms n'en juge aucune, 0 violation | inchangée |
| `toleranceMortMS` (V-8) | 200 (198) | 536 morts statborg : écart à la fin de vie la plus proche 0 ms x198, 33 ms x314, 66 ms x16, 132 ms x1, 165 ms x5, **198 ms x2** (100 ms : 0 x421, 100 x109, **200 x6**). 0 violation dès 198 ms | 200 ms convient mais **sans marge** (la conversion arrondit vers le bas : 6 images = 198 ms, le maximum observé). 250 ms (7 images, 231 ms) laisse une image de marge, sans changer le verdict à 100 ms |
| `margeFinDeFilmMS` (O-V2) | 200 | toutes les fins de piste voisines de la fin du film tombent à 0 ms (dernière image) | inchangée |
| V-2 (comptage) | -- | voir ci-dessus | **compter autrement** : par véhicule, ou par intervalle continu hors emprise, pas par échantillon ; sinon toute bascule de cadence déclenche « FAUX » |

### 10.5 `t0_film` : coup d'envoi à 100 et à 33 ms

| match | 100 ms | 33 ms | écart |
|---|---|---|---|
| 0076ebdc | 22 800 | 22 737 | -63 |
| ef8cb72b | 400 | 330 | -70 |
| 008e1bba | 22 700 | 22 671 | -29 |
| d2b74083 | 300 | 297 | -3 |
| de3cec8b | 22 800 | 22 770 | -30 |
| 43716616 | 22 700 | 22 704 | +4 |
| c75f33b8 | 22 700 | 22 638 | -62 |
| e157a672 | 22 700 | 22 638 | -62 |
| 82f29378 | 22 500 | 22 473 | -27 |
| ce927bd2 | 22 700 | 22 638 | -62 |
| be565134 | 22 600 | 22 605 | +5 |
| 4f862258 | 22 800 | 22 704 | -96 |

(Marge à la frame 0 du film, `coverage.t0Film.marginMs`.) Les 12 coups d'envoi sont détectés aux deux cadences, avec la même rafale. L'écart est la quantisation : la grille de 100 ms date le premier mouvement à l'image suivante (retard moyen 41 ms, jusqu'à 100 ms) ; deux matchs sont à +4 et +5 ms.

Plancher de bruit de la fenêtre calme (de la frame 0 à 1 s avant le coup d'envoi ; `d2b74083` et `ef8cb72b` n'ont pas de fenêtre calme, le film commence sur le départ) :

| | médiane du cumul max par piste | max sur les pistes | pas max | seuil actuel |
|---|---|---|---|---|
| 100 ms | 0,000 m | 0,25 m (`be565134`), sinon <= 0,07 m | 0,14 m (1,40 m/s) | `t0FilmCumulM` 0,5 m ; `t0FilmJumpM` 5 m par pas |
| 33 ms | 0,000 m | **0,36 m** (`be565134`), sinon <= 0,09 m | 0,06 m (1,82 m/s) | idem |

- **`t0FilmCumulM` (0,5 m).** Le bruit cumulé croît avec le nombre de pas de la fenêtre : la marge sur le pire film passe de x2,0 à **x1,4**. À 0,3 m, `be565134` date un faux coup d'envoi à 561 ms. Effet du seuil sur la date (33 ms) : chaque +0,1 m retarde d'environ 33 ms (0,5 : 22 737 ; 0,6 : 22 770 ; 0,75 : 22 836 ; 1,0 : 22 935 sur `0076ebdc`). **Proposition : conserver 0,5 m** (12 coups d'envoi corrects, plus précis que 100 ms) ; **0,6 m** si l'on veut une marge de x1,7 pour un retard de 33 ms. Décision du user.
- **`t0FilmJumpM` (5 m par pas).** À 33 ms, 5 m par pas valent 150 m/s, contre 50 m/s à 100 ms. Essai avec 1,65 m (50 m/s) : dates identiques sur les 12 : aucun effet mesurable. **Proposition : l'écrire en vitesse (50 m/s) et le convertir avec le pas**, pour garder le sens « téléportation ». La fenêtre de 1 000 ms et `t0FilmMaxDelayMS` sont déjà en millisecondes.

### 10.6 Dérivés en simulation sur les artefacts à 33 ms (aucune écriture en base)

`backfill-usage-summary`, `backfill-vehicle-takes`, `backfill-flag-grabs-net`, `backfill-pad-tiers` (`--dry-run --force --match <id complet>`, 12 matchs) et `tactical-rasters --backfill --dry-run` : sorties à 33 ms **identiques ligne pour ligne** aux sorties à 100 ms du même code, et aux `avant/derives/` de l'étape 0 (hors durée). Aucun écart à chiffrer (mêmes prises nommées et anonymes, mêmes lignes de prises de véhicule `82f29378` 20 / `be565134` 21, mêmes prises de drapeau brutes et nettes `008e1bba` 17 / 7, mêmes niveaux d'armes). `backfill-lead-share` n'existe plus sur cette branche. Séquence de mort Tactique (substitut `TestMesureLectureDesPistes`) : même nombre de pistes (12 matchs), points x2,9 à x3, **corps de la séquence 3,1 à 3,4 Ko aux deux cadences** (écart <= 2 %) : la fenêtre de séquence est en millisecondes. Limite : ces simulations ne lisent pas toutes les couches (la plupart des matchs n'ont aucune prise de véhicule ni de drapeau) ; elles ne démontrent l'égalité que sur la matière présente.

### 10.7 Coût

| | 100 ms | 33 ms |
|---|---|---|
| Cuisson par décodage complet, somme des 12 (durée interne, machine chargée par d'autres agents, bruit de +-30 %) | 166 s | 160 s |
| Le plus long (`be565134`) | 51 s | 36 s |
| Pic mémoire du process, 12 matchs (échantillonné) | 84 à 449 Mio | 85 à 478 Mio (le plus gros +6 %) |
| Cuisson depuis les faits (33 ms) | -- | 0,5 à 2,4 s |
| Lecture : gunzip + décodage `replay.ReplayDocument` + ré-encodage (`be565134`) | 19 + 113 + 34 ms | 44 + 282 + 92 ms |
| Compression gzip 6 à l'écriture (`be565134`) | 249 ms | 634 ms |
| Compression réseau niveau 5 (`be565134`) | 140 ms | 333 ms |

La grille ne coûte presque rien à la cuisson (le décodage domine) ; elle triple la lecture, l'écriture et le transfert, comme la taille.

### 10.8 Surprises

1. Les témoins AVANT ne valent plus pour B8 (voir 10.3) : à reconstituer à 100 ms avec le binaire de la bascule.
2. `replay-verite` n'est pas cadence-neutre sur V-2 (comptage par échantillon) : la bascule déclencherait « FAUX » sur tout match à véhicules.
3. La cause de B7 est une entrée manquante (`--facts`), pas un défaut du décodeur.
4. Les faits de film sont indépendants de la cadence : la recuisson de C2 peut se faire depuis les faits.
5. Les poses de bord de film changent de nombre (-15 à +7) : pas de perte en cours de match, mais le compte du calque bouge.
6. `be565134` franchit 2 Mo compressé à 33 ms (D2).
7. `TestViesDUnEchantillonOntLeurOracle` : les trois vies d'un seul échantillon mesurées le 2026-09-14 sur les huit builds disparaissent à 33 ms (la vie compte plus d'un point de grille) ; le test attend `{3,0,3,0}` et mesure `{0,0,0,0}`.
8. Une première passe de dérivés a été lancée deux fois en parallèle par erreur de ma part (verrou de base) ; les sorties retenues viennent d'une passe unique, séquentielle.

### 10.9 Contenu probable du lot final

Sur une copie hors dépôt des modules `libs/film`, `libs/himap`, `libs/gamefiles` (jonctions vers la configuration et les données du worktree), `DefaultFrameIntervalMS = 33` seul : `go test ./film/...` : **11 tests de premier niveau échouent dans `replay`** (en plus de 4 échecs d'environnement qui existent aussi à 100 ms : `TestTableStatborgConcordeAvecNamedStatSlots`, trois tests de `killicon` qui lisent des fichiers absents de la copie) :

| Test | Nature |
|---|---|
| `TestGoldenAssembly`, `TestGoldenBuildsAssembly` (8 builds), `TestProjectilesAndInventoryCounts` | goldens `testdata/assembly_*.golden` à régénérer (8) ; projectiles 463 -> 607 sur le build de référence |
| `TestContractFixturesMatchCommitted` | les 8 fixtures `apps/web/.../fixtures/go/replay_schema_108_*.json.gz` (2,97 Mo compressés, environ 7,3 Mo à 33 ms) à régénérer en 109 ; le test les lit depuis `apps/web` : à voir comment le dépôt dédié les reçoit |
| `TestFramesOfMS`, `TestPasHistoriqueDistinctDuDefautDeConstruction`, `TestToleranceDePositionDeZoneParDefautVaut200ms`, `TestLaToleranceDEchantillonnageEstBornee` | attendus écrits pour 100 ms (2 images pour 200 ms, défaut 100) ; décisions inchangées en millisecondes |
| `TestLinkShots_FenetreDUneFrame`, `TestReposNeAPeuPresBornes` | tests écrits en images (fenêtre d'une image, tolérance de 3 images) : à réécrire en durées |
| `TestViesDUnEchantillonOntLeurOracle` | voir 10.8 |

- **Dépôt du décodeur** (dépôt dédié) : `replay/build.go` (constante et commentaires), `replay/document.go` (`SchemaVersion` 109) et sa chronique, 7 à 8 fichiers de tests, 8 goldens, éventuellement `replay/t0_film.go` (si `t0FilmJumpM` passe en vitesse ou `t0FilmCumulM` à 0,6 m) : **environ 17 fichiers**, plus un tag de module posé par le user (nouvelle version de `libs/film`).
- **Dépôt LevelUp** : `go.mod`/`go.sum` (nouvelle version du décodeur, ou rien tant que `libs/film` est dans l'arbre), `internal/replayverite/` (`seuils.go`, `violations_positions.go` pour V-2, `seuils_cadence_test.go`, le golden `testdata/bcb6d393.bulletin.golden` à vérifier), 8 fixtures web et `manifest.json`, `docs/COMMANDS.md` + `docs/FR/COMMANDS.md`, ADR 0034 (cadence), `.ai/thought_log.md` : **environ 15 fichiers**. Aucun code applicatif Go ne lit `DefaultFrameIntervalMS` (seulement des tests) ; `raster.go` et `positions.go` lisent déjà la cadence historique. Côté web, la page lit `frameIntervalMs` du document : aucun changement de code identifié par cette mesure (non testée avec des fixtures à 33 ms).
- **Pas de code** pour : stockage, porte `replayartifact`, dérivés (identiques), séquence Tactique.
- Ordre pour que le lot reste court : (1) décisions du user sur `toleranceVieMS`, `toleranceMortMS`, `t0FilmCumulM`, comptage V-2 ; (2) décodeur : constante, version, tests, goldens, tag ; (3) LevelUp : montée de version, `replayverite`, fixtures web, docs ; (4) B7/B8 sur la liste fermée avec `--facts` ou `backfill-replay` ; (5) gates.
