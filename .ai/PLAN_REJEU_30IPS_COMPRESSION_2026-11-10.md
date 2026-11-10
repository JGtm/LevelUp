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
- [ ] B1 **Tests d'abord**, pour chaque seuil de la section 1 : la même décision, exprimée en millisecondes, aux cadences 100 et 33 ms. Côté Go, un seul helper de conversion des millisecondes en images (vérifier d'abord qu'il n'en existe pas). Côté web, `msToFrames` (`lib/replay/replayLogic.ts:348`), déjà utilisé par `useReplayTiming`.
- [ ] B2 **Convertir les seuils Go**. Le pas maximal des projectiles devient une vitesse : 100 m/s × pas.
- [ ] B3 **Convertir les 11 constantes web** et vérifier `placementWall.ts:110`.
- [ ] B4 **Re-mesurer à 33 ms** les seuils qui sont des mesures de corpus : `replayverite/seuils.go` et le plancher de bruit de `t0_film.go`. Les valeurs retenues et leur mesure vont dans le commit. Si une mesure contredit la valeur historique convertie, remonter au user avant de trancher.
- [ ] B5 **Garde-rail D-6.** Aucune constante numérique suffixée `Frames` / `Images` / `_FRAMES` dans `libs/film/replay`, `analysis/tactical`, `replayverite` et `features/match-replay`. Allowlist vide, sauf les outils de recherche.
- [ ] B6 **Bascule.**
  - `DefaultFrameIntervalMS = 33` ;
  - `HistoricalFrameIntervalMS = 100`, utilisé par `raster.go:115` et tout défaut de relecture ;
  - `SchemaVersion` à 109, avec son entrée de chronique ;
  - les commentaires périmés listés par l'inventaire sont corrigés ;
  - les goldens sont régénérés, chaque écart expliqué.
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

- [ ] B10 **Applications de bureau déjà installées (décision D-9).** Vérifier d'abord sur pièces le comportement de la file `replayregen` (décompte au démarrage, démarrage sur geste, `isStale` qui exige le film archivé). Tests d'abord, puis : à la fin d'une mise à jour des rejeux lancée par l'utilisateur, la file compresse par la porte les artefacts encore en `.json` qu'elle n'a pas pu refaire (film non archivé). Jamais au démarrage, jamais sans ce geste : le garde-rail `no_auto_replay_compress_test.go` reçoit ce second appelant, nommé et justifié. Compte rendu visible dans l'état de la file (refaits, compressés, échecs). Vérification du user sur une application installée : après mise à jour, les anciens rejeux s'ouvrent ; l'écran propose la mise à jour des rejeux ; à la fin, plus aucun `.json` brut dans le dossier des rejeux et la place occupée a baissé.

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
