# PLAN — Rejeu 2D : capteur, traqueur, écran occultant et champ de réparation à nouveau dessinés (2026-10-28)

Branche `fix/rejeu-objets-engendres` (depuis `release/v8.1.0` 14eee03f2), worktree
`../LevelUp-wt-objets-engendres`. Exécution sous le contrat `plan-execution`. Go du user le
2026-10-28 (« Ok très bien je suis d'accord avec toi mais faisons attention à ne pas faire
régresser la faille du translocateur ou le mur qui sont bien visibles et dont le son est bien joué
actuellement »).

**Critère de succès** : sur les témoins, les quatre familles se dessinent (forme, révélation,
zone, survol, usages) de leur pose à leur fin lue ; mur et faille identiques à l'octet (dessin et
sons). **Effort** : moyen (Go ~1 fichier + couverture ; web ~1 modèle + branchements). **Exécution** :
ordre strict, étape N close (gate vert + items statués `[x]` / `[~]` réf / `[!]` justifié, aucune
case vide) avant N+1 ; reprise = relire ce fichier, reprendre à la première case non statuée ;
découvertes consignées en bas, jamais traitées. Implémentation par sous-agent Sonnet, pas de commit
ni de `git add -A` par l'agent, `GOCACHE=%LOCALAPPDATA%\go-build-objets-engendres`.
**Logging** : les compteurs neufs s'ajoutent à la ligne `slog` existante de couverture du calque
(`document_equipement_porte.go`, « objetsEngendres »). **Multi-titre** : code dans
`internal/games/halo_infinite/film/` (propre au titre), web branché sur les familles du document,
jamais sur le slug.

## Diagnostic (thought_log du 2026-10-28)

Le calque des poses ne dessine qu'une pose `deployed` (`placementKind`,
`apps/web/src/features/match-replay/layers/equipmentPlacementsLayer.ts`). Depuis la règle du 103
(décision user du 2026-09-15, `replay/equipment_origin.go` `origineDeLaPose`), seuls les panneaux
de mur le sont. Sur les 1 402 artefacts locaux : `deployed` = mur seul (7 528) ; capteur, traqueur,
écran, champ = 0. Leur son a été rebranché le 15/10 sur `carriedEquipment[].uses`, leur dessin non.

Le document porte l'objet engendré de chacun avec sa vie lue (`carriedEquipment[].tracked[]`,
{t0, t1}, identifié côté décodeur par sa `LifeKey` {Slot, Gen}) : 1 656 capteurs, 489 traqueurs,
1 005 écrans, 116 champs, SANS position. Les trajectoires `projectiles[]` viennent de
`types.ProjectileTrack`, identifiées par la MÊME `LifeKey` : sur le témoin `000d5950`, chaque
lancer de capteur coïncide à une ou deux images près avec une trajectoire qui se pose (`rest`).

## Contrat retenu

- **Go** : chaque `CarriedTracked` gagne la POSITION DE REPOS de l'objet engendré, lue sur la
  trajectoire de la même `LifeKey` : `land` (image du dernier point publié), `x`, `y`, `z`
  solidaires, absents ensemble quand aucune trajectoire ne porte la vie (rien ne se devine ; un
  repli éventuel n'entre que s'il est MESURÉ à l'étape 1 et nommé + compté par `fallback`).
  Couverture : `trackedPositioned` / `trackedUnpositioned`. Montée de schéma et de révision de
  couche selon `replay/layers.go`. `equipmentPlacements`, `riftPoses`, `projectiles`, les
  origines de pose et les résumés d'usage serveur NE CHANGENT PAS (la règle du 15/09 tient).
- **Web** : une source « objets engendrés » (modèle pur, hors composant) fabrique, pour les seules
  familles `sensor`, `threat_seeker`, `shroud_screen`, `repair_field` (liste nommée, garde-rail),
  une pose dessinable à la position de repos, de `land` à `t1` lu (repli : la durée officielle
  existante quand `t1` manque). Elle alimente le même calque, la révélation du capteur, la présence
  en zone, le survol et le compte « déployé » des usages du rejeu. Texte d'aide corrigé (FR + EN).
- **MUR et FAILLE HORS PÉRIMÈTRE DE MODIFICATION** : leurs canaux (`equipmentPlacements` mur,
  `riftPoses`), leur dessin, leur fenêtre (`WALL_DURATION_MS`) et leurs sons ne bougent pas. Les
  familles `wall` et `translocator_beacon` ont aussi des `tracked` (2 002 et 292 au parc) : elles
  sont EXCLUES de la nouvelle source, sinon le mur et la faille se dessineraient deux fois.

## Témoins avant/après (scratchpad `temoins_objets_engendres/`)

Films témoins (chunks présents en local) : `bc0bdda3` (traqueur 24, mur 13), `2ba491cf` (capteur
16, mur), `5676a9ba` (champ 11, capteur 14, écran, mur), `bc8b45bf` (champ, écran), `f59c9332`
(faille 6, écran 11), `a0c36016` (faille 6, mur 4, écran), `000d5950` (référence).

- **T-Go** : documents des 7 témoins cuits EN PROCESSUS par `replay-equiv` (un film à la fois, verrou
  solo, `-repo-root` du worktree, jonctions `data/cache/film_chunks` et `film_manifests` ; faits
  `data/cache/film_facts` du worktree mis de côté avant la passe « après », `depuis_les_faits=false`
  vérifié). Comparaison : documents identiques à l'égalité structurelle une fois retirés les seuls
  champs ajoutés (`tracked[].land/x/y/z`, compteurs neufs) et `schemaVersion` ; en particulier
  `equipmentPlacements`, `riftPoses`, `projectiles`, `carriedEquipment` (hors champs neufs) à
  l'identique.
- **T-Web** : script de caractérisation local (non versionné) sur les artefacts cuits AVANT des 7
  témoins : pour chaque image échantillonnée, la liste des poses dessinées (`placementKind`,
  fenêtre active, id, famille), les failles (`riftStations`) et le calendrier des sons du mur et de
  la faille (pose, téléportation, extinction). AVANT et APRÈS : sorties identiques à l'octet pour
  mur et faille ; seules s'ajoutent les poses des quatre familles.
- Tests ciblés existants : même ensemble vert avant et après (`layers/`, `model/`, `sound/` du rejeu,
  paquet `replay` Go).

## Étapes

### Étape 0 — Témoins AVANT
- [x] 0.1 Jonctions de données dans le worktree (mklink /J, jamais `ln -s`), node_modules web (jonctions `film_chunks` et `film_manifests` posées le 2026-10-28 ; node_modules relève de l'agent web).
- [x] 0.2 T-Go avant : 7 documents cuits sur le code de base, empreintes md5 (par `cmd/replay-build`, autorisé par le user le 2026-10-28 ; journal).
- [x] 0.3 T-Web avant : sorties du script de caractérisation, empreintes md5 (journal du 2026-10-28, étape 3).
- Gate : fichiers témoins présents, empreintes consignées au journal du plan.

### Étape 1 — Mesure (avant tout code de production)
- [x] 1.1 Mesure APPROCHÉE par le superviseur (appariement en TEMPS seul, artefacts publiés, 401
  documents, lecture seule) : une trajectoire naît à ±2 images de l'objet engendré pour 94,0 % des
  capteurs, 100 % des traqueurs, 81,2 % des écrans, 95,7 % des champs — contre 19,6 % (mur) et
  15,9 % (faille), soit le taux de coïncidence fortuite d'un appariement en temps seul. Les quatre
  familles SONT des objets lancés à trajectoire ; vol médian 1,8 à 10,5 m (le porteur n'est donc
  pas un repli acceptable) ; dernier point → détonation lue : médiane 0,05 à 0,62 m.
- [!] 1.2 Mesure EXACTE (jointure par `LifeKey`) = les compteurs de 2.2 sur les 7 témoins, lus AVANT
  3.x. Si un famille est < 90 % positionnée, STOP et rapport au superviseur.
- Décision : la position de repos est le DERNIER POINT PUBLIÉ de la trajectoire (la détonation lue
  est un message d'impact, pas un repos ; écart médian ≤ 0,62 m).
- Gate : tableau de mesure exacte au journal du plan.

### Étape 2 — Go : position de repos de l'objet engendré
- [x] 2.1 Test d'abord (rouge) : `document_objets_engendres_test.go` (rouge constaté : champ `projectiles` et `Land` absents ; vert ensuite). Pas de fixture de film réel : les lectures sont celles de 5676a9ba déjà employées par `TestChampDeReparationFinLue`.
- [x] 2.2 Implémentation (`replay/document_equipement_porte.go` ou fichier voisin ≤ 500 L), couverture,
  montée de schéma/révision, contrat OpenAPI/types web régénérés.
- [x] 2.3 T-Go après : identité hors champs neufs sur les 7 témoins (voir journal : documents « après » rejoués depuis les faits de la passe décodée).
- Gate : `go test ./internal/games/halo_infinite/film/...` + `./internal/archlint/...` +
  `./internal/contracttest/...` verts, `make go-api-lint` sans nouvelle entrée, T-Go identique.

### Étape 3 — Web : dessin et usages des quatre familles
- [x] 3.1 Tests d'abord : modèle « objets engendrés » (familles exclues mur/faille, `t1` lu, repli,
  objet sans position = rien).
- [x] 3.2 Branchement : calque, révélation du capteur, zones, survol, usages du rejeu.
- [x] 3.3 Texte d'aide FR + EN (manifests / dictionnaires typés), sans promesse fausse.
- [x] 3.4 Garde-rail sur extrait d'artefact RÉEL versionné (patron `test/carriedUseWitnesses.ts`) :
  toute famille dont `PLACEMENT_RENDER` est une règle de dessin produit au moins une pose dessinée
  sur son témoin réel. Mutation vérifiée (le test rougit si la source est débranchée).
- [x] 3.5 T-Web après : mur et faille identiques à l'octet.
- Gate : `make check-types`, vitest des dossiers `features/match-replay` et `lib/replay`, eslint 0
  sur les fichiers touchés, knip 0.

### Étape 4 — Clôture
- [x] 4.1 Revue adversariale (une ronde, relecteur Sonnet neuf) sur le diff.
- [x] 4.2 `delivery-checklist`, journal, thought_log, commit(s) sur accord du user (accord du 2026-10-28).
- [ ] 4.3 Fusion dans `release/v8.1.0` sur accord du user, puis DEMANDE de recuisson du parc local
  (protocole du 2026-10-06 : demandée après fusion, durée réelle rapportée).
- [ ] 4.4 Gate visuel du user, en lecture (pas à l'arrêt), sur un témoin par famille + un mur + une
  faille.

## Découvertes (non traitées)

- Le mur a lui aussi une fin lue (`tracked[].t1`, 1 951 au parc) mais garde sa durée officielle :
  hors périmètre par consigne du user.
- (2026-10-28, étape 3) Le texte d'aide `layerPlacementsHint` parle encore de la « BALISE » du translocateur alors que le calque dessine une faille (correction du user du 2026-08-27) : non traité (hors périmètre).
- (2026-10-28, étape 3) Sur le témoin 2ba491cf, la fin lue des 13 capteurs situés vaut entre 0,1 s et 20,2 s après le repos (4 sous la seconde, 5 entre 4,6 et 7,6 s, 4 à 20,1-20,2 s) et le `t1` d'un capteur égale le `t0` du suivant du même objet porté dans 10 cas sur 11 (2252, 2313, 2381) : à vérifier si la suppression lue est un remplacement par le lancer suivant ; la fenêtre suit `t1` tel que décidé.
- (2026-10-28, étape 3) L'écran occultant, sans durée, couvre les joueurs jusqu'à la fin du rejeu (zones des fiches : 944 et 3329 couples joueur-image sur a0c36016 et f59c9332) : à trancher avec la durée de l'écran (`SHROUD_DURATION_MS`).
- (2026-10-28, après revue) Objets non situés, diagnostic sur les faits persistés : 95 objets ti=41 sur les 4 témoins concernés, 88 avec une trajectoire sous leur clé née 15 à 60 ms après la pile ; sur les 11 non situés, 7 sans AUCUNE trajectoire écrite par le film, 4 avec 3 ou 4 échantillons seulement (environ 0,1 s de vol, sans repos, un seul point de grille). Limite du film, aucun repli. Le traqueur et l écran ont la même fin lue (0,1 à 0,4 s après le repos) : pas de durée d écran à lire dans le film ; aucune source publiée trouvée.

## Journal

### 2026-10-28 — Arrêt à l'étape 0.2 (exécuteur Go)

- 0.1 fait : jonctions `data/cache/film_chunks` (1 423 entrées visibles) et `data/cache/film_manifests`
  vers le principal, posées par `New-Item -ItemType Junction` (PowerShell). `ln -s` non utilisé.
- 0.2 BLOQUÉ, rien cuit, aucun code de production touché. Consigne de l'exécuteur : cuire EN PROCESSUS
  avec `cmd/replay-equiv`, et s'arrêter si l'outil ne sort pas les documents complets. Constat sur pièces :
  1. `cmd/replay-equiv/child.go` (`digestsDuFilm`) appelle `b.BuildBytes(...)` et JETTE les octets
     (`_, err :=`) ; il n'écrit que des digests par étape (`<film>.tsv`), jamais le document. Aucun
     drapeau ne conserve le document (`-out-dir` ne garde que les TSV).
  2. Il exige `<short8>.facts.json` sous `replay/testdata/equivalence/` : seul `000d5950` des 7 témoins
     y figure (corpus de 20 films). Les 6 autres demanderaient `replay-facts-export` (lecture de la base
     partagée du titre).
  3. Seul autre outil unitaire qui produit le document : `cmd/replay-build` (même verrou solo, même
     sentinelle mémoire), mais il écrit `data/cache/replays/<titre>/<matchId>.json` à la racine résolue par
     `title.FindRepoRoot()` (pas de `-repo-root`, variable `LEVELUP_REPO_ROOT` interdite par la consigne) et
     prend `--map` + `--facts` (même problème de faits). C'est « une autre voie de cuisson » : non
     empruntée sans accord du superviseur.
- Décision attendue du superviseur : (a) autoriser `replay-build` (un film à la fois, depuis le worktree,
  racine résolue par `FindRepoRoot` en CWD worktree, avec ou sans `--facts` pour avant ET après, ce qui
  suffit à comparer tracked/projectiles/equipmentPlacements/riftPoses), ou (b) ajouter à `replay-equiv`
  un drapeau de sortie du document (changement d'outil, hors périmètre de ce plan).

### 2026-10-28 (suite) — étapes 0.2, 2.1-2.3 faites ; 1.2 : STOP, trois familles sous 90 %

Outil : `cmd/replay-build` (autorisé par le user), un film à la fois, au premier plan, depuis
`apps/go-api` avec `LEVELUP_REPO_ROOT` = le worktree ; binaires `replay-build-avant.exe` (code de base) et
`replay-build-apres.exe` ; sans `--facts`. Cartes lues sur une COPIE de `shared_matches_v2.duckdb` :
bc0bdda3 Forbidden, 2ba491cf Recharge, 5676a9ba Insolence, bc8b45bf The Pit, f59c9332 Catalyst,
a0c36016 Forest, 000d5950 Cliffhanger. Durées « avant » (décodage, s) : 6, 7, 6, 18, 10, 8, 9 ; pics
mémoire 0,27 à 0,63 Gio ; aucun échec. 14 décodages au total (7 avant, 7 après) ; ensuite 7
REPUBLICATIONS depuis les faits (`depuis_les_faits=true`, 0 à 1 s chacune, aucun décodage de chunk) pour
chaque réglage du code, voir plus bas.

**Empreintes md5 AVANT (code de base, décodage)** : 000d5950 `72b151517a125fa778803727be878eb3` ·
2ba491cf `685a2548ce9539260a54ed070fb489c3` · 5676a9ba `721ddec5554d90e1a5648a26179e85cd` ·
a0c36016 `e178da2e59b84d96fc51e4868d3bebb2` · bc0bdda3 `6379f30e95955944843935cee460c5fe` ·
bc8b45bf `9286d86c23c3d31cd702f8a0b671cdaf` · f59c9332 `10f4efd30e79241870c8a59be6122a2a`.
**APRÈS (code final, republication depuis les faits écrits par la passe « après » décodée)** :
000d5950 `2a6e9efc6ca5e52431ffda18ae76e4c5` · 2ba491cf `056d1aa0a43cd458ec643610ddaf5868` ·
5676a9ba `6a1687b4a0b9986d126657c87f5afd61` · a0c36016 `24f5dde52d1daad9deea345c7bd86b16` ·
bc0bdda3 `253dbe45db930e89a0dacbfedc47cca5` · bc8b45bf `df2049add249f0da020216aed7c2d3fb` ·
f59c9332 `024f81f0ea6a8039b01df601bd1a9239`. Fichiers : scratchpad `temoins_objets_engendres/go/{avant,apres}/`
(+ `apres_decode1/` : documents de la première passe décodée, règle « dernière trajectoire de la clé »,
abandonnée).

**Contrat réalisé** : `CarriedTracked` gagne `land/x/y/z` (pointeurs, solidaires) ; couverture
`trackedPositioned` / `trackedUnpositioned` (+ ligne slog « objetsSitues », « objetsSansPosition »).
Code : `replay/document_objets_engendres.go` (nouveau), `replay/projectiles.go` (`volPublie` extrait de
`buildProjectiles`, sortie inchangée), `replay/document_equipement_porte.go`. Faits persistés : AUCUNE
révision (ni `SchemaDesFaits`, ni `grammar.Rev`) ; la position se calcule à la publication depuis
`FilmInputs.Projectiles` et les lectures de l'équipement porté, déjà persistés ; `SchemaVersion` 104 -> 105
(entrée de chronique v105). DÉCOUVERTE DE JOINTURE : la clé (slot, génération) NE SUFFIT PAS (la
génération revient dans le match : première version, `land` = 5923 pour un objet apparu à la frame 943) ;
la jointure est donc « trajectoire de la même vie, née à `toleranceDeNaissance` = 3 frames au plus de la
pile i28 qui nomme l'objet, la plus proche ». Fichiers régénérés : `openapi.yaml`, `generated.ts`,
goldens d'assemblage (8) et de forme, fixtures de contrat web `replay_schema_105_*` (8) et manifeste.
Plafonds `film_file_size_test.go` (chronique 3633, structure_test 1486) et compte du ratchet de polarité
(198 échec / 190 succès) mis à jour dans le même lot, comme l'exigent leurs commentaires.

**1.2 Mesure exacte (7 témoins, règle de jointure finale)** :

| famille | objets | situés | part | repos certifié (`rest`) | `land` <= `t1` + 1 |
|---|---|---|---|---|---|
| capteur (`sensor`) | 32 | 26 | 81,3 % | 22/26 | 26/26 |
| traqueur (`threat_seeker`) | 24 | 24 | 100,0 % | 0/24 | 24/24 |
| écran (`shroud_screen`) | 17 | 14 | 82,4 % | 0/14 | 14/14 |
| champ (`repair_field`) | 16 | 14 | 87,5 % | 7/14 | 14/14 |
| mur (info) | 23 | 0 | 0 % | | |
| faille (info) | 12 | 0 | 0 % | | |

Mur et faille ne sont pas des projectiles ti=41 : jamais situés, comme attendu (ils restent hors de la
nouvelle source). **Trois familles sont sous 90 %** : arrêt prescrit. Diagnostic des 11 objets non situés
des quatre familles (instrumentation temporaire retirée) : pour la plupart, AUCUNE trajectoire de cette vie
n'existe dans les pistes ti=41 lues (liste vide) ; quatre objets (rangs de palette 22 et 10, selon le film)
ont une trajectoire née à la bonne frame mais de 3 ou 4 échantillons, soit moins de deux points de grille,
que `buildProjectiles` ne publie pas ; un champ (rang 23) n'a que la trajectoire d'un autre passage de la
même clé (née 4 900 frames plus tôt). Les rangs des lignes du diagnostic n'ont pas été rapprochés des
familles : la ventilation exacte par famille n'est pas établie. Aucun repli n'a été ajouté.

**2.3 T-Go** : après retrait de `schemaVersion`, de `layers.*` valant `publication-<N>` (la révision de
publication suit le schéma), de `tracked[].land/x/y/z` et de `coverage.carriedEquipment.tracked{Positioned,
Unpositioned}`, les 7 documents sont IDENTIQUES à l'égalité structurelle ; `equipmentPlacements`,
`riftPoses` et `projectiles` sont identiques après sérialisation canonique sur les 7 (script
`tgo.js`, scratchpad). Réserve : les documents « après » sont des republications depuis les faits
(`depuis_les_faits=true`) de la passe « après » décodée une fois (`depuis_les_faits=false`, vérifié), pour
tenir 14 décodages ; avant = décodage, donc l'identité vaut aussi équivalence décodage/republication.

**Gates (apps/go-api)** : `go test ./internal/games/halo_infinite/film/...` ok ; `go test
./internal/archlint/...` ok (104 s) ; `go test ./contracttest/...` ok ; `go test ./internal/replaydiff/...
./internal/service/replayview/... ./internal/domain/replaydoc/...` ok ; `golangci-lint run
--new-from-merge-base=origin/main` sur replay, replaydoc, replaydiff, archlint : 0 issues ; `go run
./cmd/openapi-gen -check` à jour. Le test de fraîcheur de `generated.ts` (vitest) n'a pas tourné (pas de
`node_modules` dans le worktree) ; le fichier a été produit par openapi-typescript 7.13.0 depuis le
`openapi.yaml` régénéré (diff : 12 lignes).

### 2026-10-28 — Décision du superviseur sur 1.2 et contrat web par famille

- T-Go revérifié sur pièces par le superviseur (script indépendant) : 7/7 identiques hors champs neufs ;
  `equipmentPlacements`, `riftPoses`, `projectiles` identiques sur les 7.
- **1.2 `[!]` → seuil levé, justifié** : 81 à 100 % des objets des quatre familles sont situés, contre
  0 % dessinés aujourd'hui ; les non situés n'ont, pour la plupart, AUCUNE trajectoire lue, et le seul
  repli disponible (le porteur) est réfuté par la mesure (vol médian 1,3 à 15 m). Les objets non situés
  restent non dessinés et comptés (`trackedUnpositioned`) ; rien ne se devine. Le seuil de 90 % était
  une borne du superviseur, pas une décision du user.
- **Vie lue après le repos** (7 témoins, frame = 100 ms) : capteur médiane 152 images (15,2 s, la durée
  officielle) ; champ 112 (11,2 s) ; traqueur 2 ; écran 2. Le traqueur et l'écran sont des PROJECTILES
  détruits à l'impact (vol 0,3 à 2,4 s) : leur `t1` lu est la fin du projectile, PAS celle de l'effet.
- Contrat web retenu, par famille (fenêtre de dessin) :
  - capteur : de `land` à `t1` lu ; repli la durée officielle existante (15 s) si `t1` manque ;
  - champ : de `land` à `t1` lu ; repli la règle existante si `t1` manque ;
  - traqueur : impulsion unique ancrée à `land` (règle existante `seekerImpulseActive`) ;
  - écran : ancré à `land` ; fin = la règle EXISTANTE de la famille (aucune durée connue : jusqu'à la fin
    du rejeu, le comportement d'avant la régression), isolée dans une seule constante — décision du
    user demandée (durée de l'écran en jeu).

### 2026-10-28 — Étape 3 (web) et témoin 0.3 : faites, gate vert

**0.3 T-Web AVANT** (avant toute modification du code web). Script de caractérisation local, NON versionné
(vitest sous `features/match-replay/`, supprimé à la clôture), entrées FIXES : les 7 documents schéma 105 de
`temoins_objets_engendres/go/apres/`. Sorties par film, 4 fichiers, dans `web/avant/` : (a) poses dessinées par le
calque (`placementKind` + `isPlacementActive` sur TOUTES les images, en intervalles d'images actives, trois jeux de
bascules ; comptes `countDrawablePlacements` ; sonde de survol `placementAt` au centre de chaque pose) ; (b)
`riftStations`, `translocationLinks`, `riftPoses` ; (c) calendrier des sons : `buildSoundTimeline` complet (avec et
sans kill) + `carriedUseSoundEvents`, `carriedEndSoundEvents`, `equipmentChangeSoundEvents`, `riftPoseSoundEvents`,
`translocationSoundEvents` (ms, stem, variantes) ; (d) `sensorReveals` (pas de 5 images), `zonePresenceAt` (chaque vie
vivante, pas de 5 images) et `buildEquipmentUsage`. Deux passes AVANT identiques à l'octet (déterminisme). md5 AVANT :
`b` (faille) 000d5950 / 2ba491cf / 5676a9ba / bc0bdda3 / bc8b45bf `ec2f0878e6f05eef3bd0b0e844ca4916` (faille vide),
a0c36016 `837b777e4b2e38028ddbfb3a729a7859`, f59c9332 `410c87506b0b822c94d5f709f004f799` ; `c` (sons) 000d5950
`d620fc9373c12326291489611849ca22`, 2ba491cf `02e9c1c361b3590a54877e87b73ff833`, 5676a9ba
`fcc50ff0f1365d1ae095e86a669bd0aa`, a0c36016 `72551b6dc6e85105a66c8701a24d0377`, bc0bdda3
`8b77e75e4b541346400cdd6baf9e8075`, bc8b45bf `6e3792502ab1738a47cf8161d8b35a60`, f59c9332
`f8344b2aafd74885bebe1b13c2543338`. Liste complète des 28 empreintes : scratchpad
`temoins_objets_engendres/web/md5_avant.txt`.

**3.1-3.2 Code.** `model/spawnedObjects.ts` (nouveau, 185 L) : `spawnedObjectPlacements` (pose fabriquée par objet
engendré SITUÉ de `sensor`, `threat_seeker`, `shroud_screen`, `repair_field` ; liste nommée
`SPAWNED_OBJECT_FAMILIES` ; `wall` et `translocator_beacon` dans `SPAWNED_OBJECT_OWN_CHANNEL_FAMILIES`, test
d'exclusion), `drawablePlacements` (poses du document puis fabriquées, mémoïsé par document ; le tableau du document
tel quel s'il n'y a rien à fabriquer), `spawnedObjectDeployments` (compte d'usage). Fenêtre : `placementWindow.ts`
gagne le type `WindowedPlacement` (`endRead?`, même vocabulaire que `equipmentEpisodes[].endRead`) et UNE ligne de
règle (`if (p.endRead) return p.t1`) ; aucune pose du document ne porte `endRead` : le mur garde exactement sa règle.
`endRead` est posé pour le capteur et le champ quand `t1` est lu (`t1` ramené au repos s'il lui est antérieur) ; le
traqueur et l'écran ne lisent jamais leur `t1` (fin du projectile). Écran : la durée est la constante
`SHROUD_DURATION_MS` (`placementShapes.ts`, valeur `undefined`), branchée dans la table `OFFICIAL_DURATION_MS` de la
fenêtre : la renseigner suffit (décision du user attendue ; tant qu'elle manque, jusqu'à la fin du rejeu, règle
existante). Branchements : `useReplayPlacements` (champ `all`, comptes + survol), `ReplayCanvas` (`placements.all`,
dépendance du tracé), `ReplayTeams` (zones des fiches), `equipmentUsageLogic`. Les sons n'ont PAS été touchés
(`replaySound.ts` lit toujours `doc.equipmentPlacements`) ; garde-rail statique : aucun module de `sound/` n'importe
la source.

**Appelants statués** (`placementKind`, `placementIsDeployedObject`, `doc.equipmentPlacements`,
`isPlacementActive`) : calque (`drawEquipmentPlacementsLayer`, `countDrawablePlacements`) = branché par
`placements.all` ; `placementHitTest` / `usePlacementHover` / `ReplayPlacementTip` = branchés (source de survol =
`all` ; l'infobulle nomme par `PLACEMENT_RENDER`, ligne « Posé par », aucune « origine » affichée pour une pose
déployée) ; `equipmentZones` + `playerCardReadings` + `ReplayTeams` = branchés ; `sensorReveals` = branché (via le
calque) ; `equipmentUsageLogic` = branché (voir ci-dessous) ; `replaySound.ts` (`placementIsDeployedObject` sur
`doc.equipmentPlacements`) = VOLONTAIREMENT non branché (les quatre familles sonnent par `carriedEquipment[].uses`, un
second son serait un doublon) ; `carriedUseSound` / `equipmentChangeSound` = commentaires seulement ;
`useReplayAbilityFx`, `vehiclesLayer`, `weaponPadFamilies`, `placementDropped` = commentaires, aucun appel ;
`equipmentUsageColumns` (`PLACEMENT_RENDER` seul) = inchangé ; `lib/replay` (schéma, normalisation, types) = inchangé.

**Ce que compte « déployé »** : UN DÉPLOIEMENT PAR OBJET ENGENDRÉ des quatre familles, SITUÉ OU NON (le geste a eu lieu
même quand son lieu manque), daté à l'apparition de l'objet (`tracked.t0`, le lancer) et attribué au propriétaire de la
vie du porteur à cet instant. Le mur et la faille ne s'y ajoutent pas. Une famille que les poses du document déploient
déjà n'est pas recomptée (artefacts antérieurs à la règle du 15/09 : sinon double compte). Effet à l'écran :
uniquement `deployed` (colonnes et ouverture de la colonne fusionnée) ; la cellule « utilisé » de ces familles reste lue
sur les consommations (`usageUsedOf`, jumeau Go inchangé, `KEPT_FAMILIES_WITH_SPAWNED_PIECE` = mur seul).

**3.3 Texte d'aide** : `layerPlacementsHint` (FR + EN, seules langues de `config/locales.toml`) : retrait de « le film
ne dit pas quand un équipement disparaît » et de « 15 s pour le capteur » tels quels ; le texte dit désormais que les
quatre objets se dessinent là où ils se posent, qu'un objet sans lieu lu n'est pas dessiné, que la suppression du
capteur et du champ est écrite par le film (15 s officielles si le film n'en donne aucune), que le mur garde sa durée
officielle et le reste jusqu'à la fin du rejeu ; l'écran occultant est ajouté à la liste des objets déployés.

**3.4 Garde-rail** : `layers/placementRenderWitness.test.ts` + `test/spawnedObjectWitnesses.ts` (extraits RÉELS
minimaux des documents schéma 105 : capteur `2ba491cf` slot 543 rang 22, champ et écran `5676a9ba` slots 539 / 757,
traqueur `bc0bdda3` slot 513, mur `5676a9ba` ses trois poses `deployed`, faille `a0c36016` slot 520). Il exige
l'égalité entre les familles à règle de dessin de `PLACEMENT_RENDER` (hors `null` et `unnamed` : exemption nommée,
aucun `other` réel sur les 7 témoins) et les témoins, puis au moins une pose dessinée par famille (faille :
`riftStations`). MUTATION : `spawnedObjectPlacements` neutralisée dans `drawablePlacements` -> les 4 familles
engendrées rougissent (capteur, champ, écran, traqueur), mur et faille restent verts ; code remis, verts.

**3.5 T-Web APRÈS** (mêmes entrées, même script ; seule la source des poses passe de `doc.equipmentPlacements` à
`drawablePlacements(doc)`) : (b) faille et (c) calendrier des sons IDENTIQUES à l'octet sur les 7 films (14/14
fichiers, mêmes md5 que ci-dessus) ; (a) toutes les poses d'origine (mur compris, 3 jeux de bascules, intervalles
d'images) identiques, md5 de la sous-liste d'origine = md5 AVANT sur les 7 (murs seuls aussi) ; survol des poses
d'origine identique ; seules s'ajoutent les poses des quatre familles : 000d5950 +2 capteurs, 2ba491cf +13 capteurs,
5676a9ba +11 capteurs +9 champs +1 écran, a0c36016 +2 écrans, bc0bdda3 +24 traqueurs, bc8b45bf +5 champs +2 écrans,
f59c9332 +9 écrans ; (d) révélations 0 -> 8 / 16 / 24 (000d5950, 2ba491cf, 5676a9ba), zones 0 -> 27 / 47 / 220 / 944 /
0 / 349 / 3329 (l'écran, sans durée connue, couvre jusqu'à la fin du rejeu : 944 et 3329 sur les films d'écran seul),
usage identique hors `deployed` des quatre familles. `deployed` ajouté : 000d5950 sensor 2 ; 2ba491cf sensor 16 ;
5676a9ba sensor 14, repair_field 11, shroud_screen 1 ; a0c36016 shroud_screen 2 ; bc0bdda3 threat_seeker 24 ;
bc8b45bf shroud_screen 3, repair_field 5 ; f59c9332 shroud_screen 11 (l'écart avec les poses dessinées = les objets non
situés). Aucun écart inexpliqué (comparateur du scratchpad `cmp_web.js` : « ECARTS NON EXPLIQUES: 0 »). md5 APRÈS :
scratchpad `web/md5_apres.txt`.

**Gate étape 3** : `npx tsc -b` (apps/web) 0 erreur ; vitest `src/features/match-replay` + `src/lib/replay` +
`src/lib/api` : 276 fichiers, 3815 tests verts (4 fichiers / 7 tests skippés préexistants), dont
`generated-types-fresh.guard`, `goFixtures.contract`, `placementFamily.guard`, `testDoc.guard`, `arborescence.guard` ;
eslint 0 problème sur `src/features/match-replay` et `src/lib/replay` ; `npx knip` exit 0. Script de caractérisation
supprimé du dépôt. Non fait ici (interdit à l'exécuteur) : entrée `.ai/thought_log.md`, commit.

Revue adversariale, P1 corrigé (2026-10-28) : `TestReposNeAPeuPresBornes` (écart 3 accepté et 4 refusé, avant et après `t0`, aucune candidate) et `TestReposNeAPeuPresLaPlusProche` (la plus proche parmi trois, égalité = la première dans les deux ordres, candidate à 4 frames inéligible) ; mutations vérifiées rouges (tolérance 1000, `<=`, dernière candidate dans la tolérance), code restauré, aucun décodage.

### 2026-10-28 — Étape 3, correctifs P1 de la revue adversariale (web)

1. **Doc inversée corrigée** (règle 17 : contrat, pas historique) : `layers/equipmentPlacementsLayer.ts` (doc de
   `drawEquipmentPlacementsLayer` : fenêtre de `placementEndFrame`, fin lue puis durée officielle puis fin du rejeu),
   `model/threatSensor.ts` (`SENSOR_DURATION_MS` = repli de la fenêtre, pas son contrat ; elle ne borne ni les pings ni
   la révélation). Vérification du calcul : `sensorPing` / `sensorPingAgeMs` sont périodiques (modulo `SENSOR_PING_MS`)
   à tout âge, la révélation ne lit que l'âge du ping : AUCUN calcul de production n'était borné par la durée officielle,
   donc pas de bug (un capteur à fin lue de ~20 s garde ses pings). `SENSOR_PINGS_DECLARED` n'avait aucun lecteur de
   production (seulement son propre test) : supprimé (règle 7), son test remplacé par un test « le ping est périodique
   au-delà de 15 s » (`threatSensor.test.ts`).
2. **Câblage couvert** : `layers/useReplayPlacements.test.tsx` (hook, extrait réel du capteur : `all` et `counts.drawable`
   servent la pose fabriquée, le survol la désigne) + garde-rail de lecture de source dans
   `layers/placementRenderWitness.test.ts` (aucun module de la feature hors liste nommée — `sound/replaySound.ts`,
   `model/spawnedObjects.ts`, `model/equipmentUsageLogic.ts` — ne lit `doc.equipmentPlacements` hors commentaires ; les
   trois consommateurs doivent porter `drawablePlacements(doc)` / `placements.all`). MUTATIONS, chacune rougit puis
   code remis : M1 `ReplayCanvas.tsx` `placements.all` -> `doc.equipmentPlacements` (garde de source) ; M2
   `useReplayPlacements.ts` comptes -> `doc.equipmentPlacements` (garde + 2 tests du hook) ; M3 `useReplayPlacements.ts`
   survol -> `doc.equipmentPlacements` (garde + test de survol) ; M4 `ReplayTeams.tsx` zones -> `doc.equipmentPlacements`
   (garde + test des consommateurs).
3. **T-Web rejoué** (script recréé puis supprimé, mêmes entrées) : sorties `web/apres/` IDENTIQUES à celles de la
   ronde précédente ; (b) faille et (c) sons identiques à l'octet à l'AVANT sur les 7 films ; poses d'origine identiques ;
   « ECARTS NON EXPLIQUES: 0 ». md5 : scratchpad `web/md5_apres.txt` (regénéré).
4. **Gates** : `npx tsc -b` 0 erreur ; vitest `src/features/match-replay` + `src/lib/replay` + `src/lib/api` : 277
   fichiers, 3819 tests verts (4 fichiers / 7 tests skippés préexistants) ; eslint 0 sur `src/features/match-replay` et
   `src/lib/replay` ; `npx knip` exit 0. Aucun commit.

### 2026-10-28 — Revue et gates de clôture (superviseur)

- 4.1 Revue adversariale, une ronde, relecteur Sonnet neuf : 0 P0/P1 au sens strict, 4 P2 ; trois traités
  dans le lot comme P1 (doc inversée de la durée du capteur, câblage non couvert, tolérance de naissance
  non tenue — mutations vérifiées), le quatrième (journal, changelog, notes de version) fait à la clôture.
  Pas de seconde ronde (consigne de quota du user du 21/10).
- Gates rejoués par le superviseur dans le worktree : `go vet ./...` rc=0 ; `go test ./...` rc=0 (214
  paquets ok, aucun `--- FAIL`) ; `tsc -b --force` après purge de `node_modules/.tmp` rc=0 ; `npm run lint`
  0 erreur (27 avertissements, tous dans des fichiers non touchés). Persist/sync/migration non touchés :
  pas de passe `-tags=integration`.
- 4.2 : changelog et notes de version EN + FR à jour ; commit et thought_log sur accord du user.
