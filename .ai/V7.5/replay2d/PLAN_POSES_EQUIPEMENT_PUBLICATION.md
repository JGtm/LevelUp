# Plan — Publier les POSES d'equipement (mur, capteur) : la largeur qui manque, puis le schema 9

> Ecrit le 2026-08-17, suite du PLAN_IDENTITE_TI37 (`193b3de5a`, `5e7634669`) : l'identite
> d'un objet `ti=37` est PROUVEE (GlobalID du tag `eqip` dans le mot 32 bits du bloc
> `object-multiplayer-properties` du record de creation ; diagonale mur 90 % / capteur 92 %,
> geometrie 0,575 m mediane vs 14 m temoin, verite terrain 2/2). La PUBLICATION est bloquee par
> UNE largeur de configuration de replication qui varie par film (default-state ti=37 : 60 bits
> sur les films d'arene, 57 sur `06dfe6d9` / `00ba2e1c` ; `mppLeadBits` 9 vs 6), dont la
> detection automatique echoue sur les gros films. Branche `feat/v75`, principal, contrat
> `plan-execution`. Le RENDU reste hors lot (decision UI utilisateur en attente).
>
> **CORRECTION DU 2026-08-18, a lire avant le reste** : la phrase ci-dessus (« UNE largeur »,
> « `mppLeadBits` 9 vs 6 ») est le diagnostic du 17/08 et il est FAUX. La largeur qui varie tient
> en DEUX champs, et c'est ce qui la rendait indetectable : `9/5` sur les films Quick Play,
> `8/3` sur les films Big Team Battle. L'hypothese `6/5` donnait le bon TOTAL mais decalait
> l'identifiant de 32 bits — elle rendait 19 a 37 poses par film portant un identifiant
> FABRIQUE. Detail et mesure : phase 0 ci-dessous. Le plan est conserve tel qu'ecrit, avec sa
> correction : c'est l'ecart entre les deux qui enseigne.

## Ce qui est acquis (ne pas re-mesurer)

- `filmdec/equipment_creation.go` : balayage des records NEW ti=37 (`[0][01][slot:13][gen:2][ti:6]`),
  default-state bit-exact (85 = 24 + 60 + 1 sur arene), hook `SetEquipmentCreationHook`,
  `MPPWord32` publie, stats de denominateur. **PERIME au 18/08** : `DetectMPPLeadBits` et son
  heuristique (« la largeur qui retient le plus de records, si leurs identifiants se repetent »)
  sont SUPPRIMES — c'etait un critere NEGATIF, et il a valide un decoupage faux. Remplaces par
  `CalibrateMPPWidths` (oracle de position sur le DECOUPAGE complet).
- Oracle de position : le corps du record retombe sur le premier point de la vie deja decodee
  en delta (`equipment_creation_offset_test.go`) — c'est l'INSTRUMENT qui mesure la largeur
  TOTALE d'un film (85 vs 82). **Il ne separe PAS les deux champs** : le total suffit a la
  position. Ce qui les separe est le nombre de records survivants (cf. phase 0).
- Resolution `MPPWord32 -> tag eqip` : `himap/sonde_ti37_gamefiles_test.go` (11/11) ; les
  identifiants connus : mur `0x008e2dc574` (+ un second par film : panneaux ?), capteur
  `0x0072199cba`, mur autre palette `0x00686b40c9` / `0x002974c233`, grappin `0x008c77ffe7`,
  propulseur `0x00eef5d48d` ; 4 identifiants « plats » = objets du monde sans poseur.
- Le porteur = bipede le plus proche a 250 ms (aucune reference de createur a la naissance —
  negatif mesure 503/503).

## Decisions tranchees avant execution

1. **La largeur ne se devine pas par film au moment du build de facon opaque** : on cherche
   d'abord CE QUI LA FIXE (structurel), sinon on la MESURE par l'oracle de position au debut
   du balayage (auto-calibration sur les N premiers records, comme `killsource.calibrate`) et
   on l'ECRIT dans les stats — jamais un defaut silencieux.
2. **Le nommage passe par un manifeste** : `replay_labels.toml` (section nouvelle
   `[[equipment_objects]]` : GlobalID `eqip` -> famille `wall` / `sensor` / `other`, avec la
   provenance en commentaire — mesure du 17/08). Jamais de GlobalID en dur cote Go.
   Un GlobalID inconnu -> famille `other`, jamais « mur » par defaut.
3. **Le mur porte DEUX identifiants** (appareil lance + panneaux, vraisemblablement) : publier
   les DEUX poses telles quelles, avec leur GlobalID ; le lot UI decidera d'en dessiner une ou
   deux — proposition ecrite ci-dessous, decision utilisateur.
4. **Le cap du mur** : le record de creation n'en porte pas ; le cap de VISEE du poseur au
   meme instant est deja decode (`CaptureDirs`, `Point.h`) — publie comme `poseHeading`
   (mesure de la couverture : part des poses avec un cap a < 200 ms), sans pretendre que
   c'est l'orientation exacte du mur.
5. Un seul decodage filmdec par process ; instruments gardes ; aucune base en ecriture ;
   JAMAIS `git add -A` (fichiers d'une autre session dans l'arbre) ; jamais d'attente passive.

## Phases

### Phase 0 — LA LARGEUR : ce qui la fixe, ou l'auto-calibrer — CLOSE le 2026-08-18

- [x] 0.1 **LA LARGEUR N'EST PAS UN CHAMP, C'EN EST DEUX — et c'est ce qui debloque tout.**
      Le plan (et le lot precedent) attribuaient les 3 bits manquants au PREMIER champ du bloc
      MPP. C'est FAUX, et la mesure le dit sans ambiguite : deux champs INCONDITIONNELS du
      chemin minimal peuvent porter l'ecart — le premier du bloc (`R(9)` au decompile) et le
      champ inline `R(5)` qui SUIT l'identifiant de 32 bits. Les deux donnent le meme TOTAL,
      donc la meme position, donc le meme accord avec l'oracle sur les records du chemin
      minimal. Ils ne donnent PAS la meme lecture : retrecir le premier decale l'identifiant
      de 32 bits et rend un identifiant faux.

      Mesure sur les films BTB, decoupage `6/5` (l'hypothese du lot precedent) contre `8/3` :

          film       6/5 (hypothese)              8/3 (mesure)
          00ba2e1c   19 poses · 1 identifiant     537 poses · 12 identifiants
          06dfe6d9   37 poses · 1 identifiant     892 poses · 14 identifiants
          084a804d   23 poses · 1 identifiant     922 poses · 13 identifiants

      A `6/5` le pic du default-state n'existait pas (largeurs eparpillees 89, 110, 133...) ;
      a `8/3` il tombe sur **57 bits**, exactement ce que le deserialiseur porte predit
      (487, 991 et 1 268 records). Et les identifiants BTB rejoignent alors ceux de l'arene.

      **Le tableau des 12 films** (decoupage · axes i0 · slots ti=37 · slots bipede · vies ·
      poses confirmees · identifiants distincts), avec le contexte du registre de matchs :

          film      liste            decoup.  axes i0     slots  bip.  vies  poses  ids
          000d5950  Quick Play        9/5     13/13/14      960    99   477    295    9
          00162144  Quick Play        9/5     15/15/17      890   100   370    181    8
          00502e52  Quick Play        9/5     17/17/16      872   100   385    246    9
          07aa428d  Quick Play        9/5     18/18/17      875    96   405    239    9
          331ff98d  Quick Play        9/5     15/15/17     1115    96   545    238    5
          cfb85a58  Quick Play        9/5     15/15/17      882    94   424    155    4
          64e8adfa  Quick Play        9/5     15/15/15     1332   138   610    229    6
          9edfcaa9  Big Team Battle   8/3     15/15/14     1800   161   631    316   13
          00ba2e1c  Big Team Battle   8/3     15/15/17     2134   209   993    537   12
          06dfe6d9  Big Team Battle   8/3     15/15/17     3493   256  1601    892   14
          084a804d  Big Team Battle   8/3     15/15/17     4623   256  2532    922   13
          0014603f  Quick Play        AUCUN   13/12/11      284    88    70      0    0

- [x] 0.2 **AUCUNE REGLE STRUCTURELLE N'EST DERIVEE, et le refus est motive.** La seule
      quantite qui separe les 11 films mesures est la LISTE (Quick Play -> 9/5, Big Team
      Battle -> 8/3), 7/7 et 4/4. Elle ne peut pas fonder une regle :
      1. **elle n'est pas dans le film** — c'est une donnee du registre de matchs ; en deriver
         la largeur ferait dependre le decodeur de la base, ce que la couche interdit ;
      2. le proxy cote film (taille de la bande de slots, nombre de bipedes) separe le corpus
         mais sa FRONTIERE n'est pas observee : rien entre 138 et 161 slots bipede. Poser le
         seuil serait l'inventer ;
      3. **deux champs retrecissent de 1 et de 2 bits** — aucune quantite en log2 n'explique
         cette paire, donc il n'y a pas de mecanisme a sourcer, seulement une correlation.

      La CALIBRATION est donc la source (`filmdec.CalibrateMPPWidths`), elle est publiee dans
      les statistiques ET dans la couverture du document, et un film qui ne tranche pas ne
      publie rien. L'oracle : la position lue a la fin du record doit retomber sur le premier
      point de la vie que l'en-tete ANNONCE — un decodage (les paquets delta) qui ne sait rien
      du record de creation. Seuils ecrits avant : >= 12 accords ET >= 3x la concurrente.
      Arret anticipe des que le verdict est net (3 a 29 chunks selon le film, 15 a 60 s).

**Gate 0 : PASSE**, avec une reserve chiffree. **11 films sur 12** ont un decoupage etabli, et
sur ces 11 le balayage rend des identifiants `eqip` **resolus 21 sur 21** contre les modules du
jeu (148 097 tags parcourus, `himap/sonde_ti37_gamefiles_test.go`). Le douzieme, `0014603f`,
NE TRANCHE PAS — 41 ancres et 70 vies d'objet sur tout le film, zero accord a n'importe quel
decoupage — et il publie une liste vide avec un `slog.Warn` explicite. C'est le comportement
que le gate prescrit, pas une defaillance : ce film ne porte aucune pose mesurable.

**Ce que le lot precedent avait mesure et qui reste vrai** : la largeur TOTALE (60 bits sur
arene, 57 ailleurs). Ce qu'il avait mal attribue : le champ qui la porte. La consequence etait
lourde — a `6/5` les identifiants BTB etaient faux, et l'un d'eux (`0x10c64ad2`) aurait ete
publie comme une identite alors qu'il etait un artefact de decalage.

### Phase 1 — MANIFESTE et distribution complete — CLOSE le 2026-08-18

- [x] 1.1 `replay_labels.toml` section `[[equipment_objects]]` : GlobalID `eqip` -> famille
      (`wall` / `sensor` / `other`), avec la provenance ET les diagonales EN COMMENTAIRE, y
      compris celles qui ECHOUENT — c'est la partie qui compte, elle dit pourquoi tel
      identifiant n'est pas nomme. Loader Go (`mappings.parseEquipmentObjects`, trois
      invariants fataux : identifiant hexadecimal 32 bits, famille dans la liste FERMEE,
      aucun doublon), accesseur `EquipmentObjects()`, pose dans `LabelCatalog` par la couche
      titre (`replaylabels.Load`) comme les icones et les teintes. Un GlobalID hors table
      vaut `other` — jamais « mur » par defaut.
- [x] 1.2 **Distribution 12/12** (11 mesurees, 1 refusee), avec 21 identifiants distincts sur
      le corpus, TOUS resolus dans le groupe `eqip` du jeu. La cohorte n'est plus le balayage
      brut mais les POSES CONFIRMEES par l'oracle — l'instrument de croisement
      (`equipment_creation_owner_test.go`) passe desormais par
      `ScanFilmEquipmentPlacements`, ce qui change la mesure autant que le chiffre : le brut
      portait 353 identifiants distincts pour 393 records sur `00ba2e1c` (du bruit), la
      cohorte confirmee en porte 12 pour 537 poses.

      **Geometrie du poseur, en METRES** (bornes du catalogue par carte), sur les 11 films :

          porteur le plus proche   mediane 0,522 a 0,596 m   p90 0,73 a 0,86 m (arene)
          TEMOIN (autre bipede)    mediane 11,1 a 35,9 m     p90 19,3 a 70,0 m

      Soit un facteur 20 a 45 entre la mesure et son temoin, sur les 11 films sans exception.

      **Diagonales agregees sur les rangs CONNUS** (le rang de capacite `i48` du poseur) :

          0x8e2dc574  rang 19 (fam. B)  22/25 = 88,0 %  temoin 30 %  -> wall     NOMME
          0x72199cba  rang 22 (fam. B)  28/32 = 87,5 %  temoin 22 %  -> sensor   NOMME
          0x528fce46  rang 19 (fam. B)  22/27 = 81,5 %               -> other
          0x72b63d69  rang  1 (fam. A)  41/50 = 82,0 %               -> other
          0x2974c233  rang  2 (fam. A)  42/54 = 77,8 %               -> other
          0x686b40c9  rang  2 (fam. A)  21/28 = 75,0 %               -> other

**Gate 1 : PASSE** — manifeste versionne, distribution 12/12, et AUCUN nom sans diagonale : les
deux seuls identifiants nommes sont au-dessus de 85 % avec un temoin plat a 22-30 %.

**CE QUE LE SEUIL COUTE, et il faut le dire** : quatre identifiants echouent de peu, dont les
DEUX que la famille A utilise pour le mur et le capteur. Consequence assumee — sur un film a
palette de famille A (les quatre films BTB du corpus), le mur et le capteur se publient
`other`, avec leur identifiant et leur pose, sans nom. Le seuil ne se rebaisse pas ; la
condition de reprise (elargir le corpus a des films de famille A dont plus de porteurs ont un
rang `i48` lu) entre au registre.

### Phase 2 — PUBLIER (schema 9) — CLOSE le 2026-08-18

- [x] 2.1 `filmdec.ScanFilmEquipmentPlacements(dir, wr)` : calibration, balayage, puis FILTRE
      par l'oracle de position, une pose par vie (la plus ancienne). **Le filtre EST la
      mesure** : l'en-tete NEW seul n'est pas selectif (111 022 ancres pour 922 poses sur
      `084a804d`) et les records acceptes sans lui portaient 866 identifiants distincts pour
      1 987 records. La cle de deduplication est (slot, generation, debut de la vie) et non le
      seul couple : la generation ne fait que 2 bits, un slot y repasse au cours d'un match.
      Le POSEUR et le CAP sont assembles une couche plus haut (`replay/equipment_placements.go`),
      qui seule detient le nuage des bipedes : plus proche a 250 ms et moins de 3 m (`owner`
      = -1 sinon), cap de visee du meme slot a moins de 200 ms (`h` absent sinon).
      `replay.Options.Placements` + `PlacementStats` — entree de DONNEES, absence non fatale
      et loggee ; la calibration voyage AVEC la liste, sans quoi une liste vide serait
      ambigue.
- [x] 2.2 Document : `equipmentPlacements`, `SchemaVersion` 8 -> 9 (chronique ecrite dans
      `document.go` ET dans le garde-rail `structure_test.go`), contrat Go
      (`contracttest`, 30 -> 31 champs, deux schemas ajoutes), OpenAPI regenere + 
      `generated.ts`, normalisation web (`replayNormalize.ts` + `NULLABLE_ARRAYS` de
      `replayContract.test.ts`), fixture d'entrees v5 -> **v6** (les poses ET la calibration),
      golden rejoue avec une SECTION dediee, `Coverage.Placements`.
- [x] 2.3 Artefacts temoins re-cuits : `000d5950` (295 poses · 32 nommees · 289 avec poseur ·
      254 avec cap), `00ba2e1c` (537 poses · 0 nommee · 505 avec poseur · 437 avec cap),
      `06dfe6d9`. `06dfe6d9` etait declare NON CONSTRUCTIBLE le 16/08 (carte `Threshold` hors
      catalogue de bornes) — elle y est desormais, et l'artefact se construit.

**Gate 2 : PASSE** — chiffres exacts au journal ci-dessous.

**LA COUVERTURE PUBLIEE, et ce qu'elle dit qu'un compte ne dirait pas.** `coverage.placements`
porte le DECOUPAGE calibre sur le film. Sans lui, « 0 pose » a deux causes indistinguables :
un film sans equipement pose, et un film dont la largeur n'a pas ete tranchee. Avec lui,
`calibrated: false` nomme la seconde. C'est la raison pour laquelle la couverture est publiee
MEME quand la liste est vide.

### Phase 3 — PROPOSITION UI (a trancher par l'utilisateur, PAS a coder ici) — ECRITE

- [x] 3.1 La proposition ci-dessous, AMENDEE par ce que la mesure a rendu : le cap du poseur
      EXISTE (254 poses sur 295 en portent un sur `000d5950`, 437 sur 537 sur `00ba2e1c`), donc
      l'option « rectangle oriente » est ouverte et non plus hypothetique. En revanche, sur les
      films Big Team Battle, AUCUNE pose n'est nommee (cf. gate 1) : la bascule y montrerait
      des centaines de marqueurs `other`. La decision utilisateur porte donc aussi sur ce que
      l'on dessine des `other` — rien, un point neutre, ou un point neutre au survol seulement.

Capteur : disque pulse (rayon en metres a fixer — le jeu affiche une zone radar ; sans donnee
de portee, un rayon fixe declare) a la position, de `t0` a `t1`, couleur de l'equipe du poseur.
Mur : rectangle centre sur la position, oriente par `poseHeading` quand il existe (sinon
carre), un seul marqueur par pose (le second identifiant du mur ne se dessine pas tant que sa
nature n'est pas etablie). Bascule « Equipements poses » dans le tiroir. Sons `Drop Wall -
Activate` / `Threat Sensor - Activate` a `t0` (bibliotheque utilisateur, categorie
Equipements des filtres).

## Regles dures

Aucun nom sans manifeste ni diagonale ; largeur jamais devinee en silence ; zero fix hors
perimetre ; decouvertes au registre ; commits sur `feat/v75`, pas de push.

## Journal

### 2026-08-18 — les trois phases closes, le rendu reste a trancher

**CE QUE LE LOT A CHANGE AU PLAN LUI-MEME.** Le plan (comme le lot precedent) tenait la largeur
pour celle d'UN champ. La mesure dit DEUX. C'est la difference entre publier 19 poses portant un
identifiant fabrique et publier 537 poses portant l'enumeration reelle du jeu — sur le meme film,
avec le meme code, a trois bits pres. Le garde-fou qui a evite la premiere issue n'est pas une
relecture : c'est le SEUIL de nommage. A `6/5`, `0x10c64ad2` sortait seul sur trois films BTB et
« se repetait » — le critere de l'ancienne detection ; il n'aurait jamais franchi la diagonale
contre le rang du poseur, et c'est ce qui a fait rouvrir la question de la largeur.

**LE PATRON A RETENIR, parce qu'il resservira** : quand deux champs inconditionnels peuvent
porter le meme ecart de largeur, un oracle de POSITION ne les separe pas (le total suffit a la
position). Ce qui les separe, c'est le nombre de records SURVIVANTS : un decoupage faux lit les
portes du bloc a des bits decales, prend des branches qui n'existent pas, et perd tout ce qui
n'est pas sur le chemin minimal. L'ecart est massif (12 contre 2, 19 contre 1, 28 contre 1) —
il suffit d'enumerer les decoupages plutot que les largeurs.

**COUVERTURE PUBLIEE, artefacts sur disque** (schema 9, `data/cache/replays/halo_infinite/`) :

    film      decoupage  poses  nommees  other  avec poseur  avec cap
    000d5950  9/5          295       32    263          289       254
    00ba2e1c  8/3          537        0    537          505       437
    06dfe6d9  8/3          892        0    892          (cf. artefact)

`06dfe6d9` etait declare NON CONSTRUCTIBLE le 16/08 (carte `Threshold` hors catalogue de
bornes) ; elle y figure desormais et l'artefact se construit.

**GATES, resultats exacts du 2026-08-18 :**

    go build ./...                                                    exit 0
    go vet ./...                                                      exit 0
    go test ./internal/analysis/... ./internal/replaybuild/...
            ./internal/games/... ./contracttest/...                   exit 0 (37 paquets)
    golangci-lint run --new-from-merge-base=origin/main               0 issues
    make check-types (web)                                            exit 0
    npm run lint (web)                                    0 erreur (19 warnings preexistants)
    vitest replayContract + grappleLayer                              13 tests verts
    himap/sonde_ti37_gamefiles_test.go                    21/21 resolus en `eqip`

**INSTRUMENTS versionnes et gardes** (`EQUIP_CREATION_FILM`, sautes en CI) :
`filmdec/equipment_creation_width_test.go` (largeur par film + structure du film),
`equipment_creation_owner_test.go` (croisement `eqip` x rang, geometrie en metres, temoin —
desormais branche sur les POSES confirmees et non sur le balayage brut),
`equipment_creation_test.go`, `equipment_creation_offset_test.go`,
`himap/sonde_ti37_gamefiles_test.go`.

**DECOUVERTES non traitees, portees au registre** : (1) le mur et le capteur de la palette
famille A sont mesures mais non nommes (75 a 82 %, le denominateur `i48` est le frein sur les
films BTB) ; (2) la correlation decoupage/liste est consignee sans etre erigee en regle ;
(3) l'oracle body-first (`equipment_creation_offset_test.go`) montre DEUX pics de distance
en-tete -> masque par film (85/425 sur arene, 82/426 sur BTB) : le second correspond a un
chemin ou la plupart des portes du bloc sont ouvertes. Il est decode correctement par la
grammaire portee (les poses confirmees en viennent aussi), mais sa structure n'a pas ete
detaillee — hors perimetre.
