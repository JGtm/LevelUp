# Plan — La precision des objets du monde : mesurer le defaut, puis le rendre impossible a oublier

> Ecrit le 2026-08-15, apres la decouverte du lot equipement (`d4be4ab95`). Branche
> `feat/v75`. Execution sous le contrat du skill `plan-execution` : ordre strict, un item =
> un statut, zero fix hors perimetre.

## Le defaut, verifie sur pieces

`WorldObjectPrecision = {IndexW: 1, AxisW: {13, 13, 14}}` (`filmdec/traverse.go:149`) est un
GLOBAL de paquet, documente comme les largeurs de **Cliffhanger**, avec la consigne ecrite
juste en dessous : « PROPRE A LA CARTE : sur une autre carte, installer les largeurs via
`SetWorldObjectPrecisionFromLayout` ».

**Aucun chemin de production n'appelle ce setter.** Ses deux seuls appelants sont des
fichiers de test (`equipment_state_test.go:124`, `replay/ground_weapon_research_test.go:360`).

`BuildFromFilm` appelle `ScanFilmProjectiles` (`replay/build.go:198`) sans jamais l'installer.
Les trajectoires de projectile publiees dans le rejeu 2D sont donc dequantifiees aux
largeurs de Cliffhanger sur toutes les autres cartes.

`DetectI0Layout` mesure autre chose sur **11 des 12 films** du corpus du 15/08 :
`[15 15 17]` x6, `[17 17 16]`, `[18 18 17]`, `[15 15 15]`, `[15 15 14]`, `[13 12 11]`.

Surface : le global est lu en **cinq points de production** — `traverse.go:251` et `:254`
(le chemin de traversee), `projectiles.go:57` et `:335`, `position_capture.go:241` — et
**un seul appelant de production** declenche le balayage (`build.go:198`).

## Objectif et critere de succes

Que les positions d'objets du monde (projectiles ti=41, armes au sol ti=42, equipement
ti=37, corps rigides ti=38) soient dequantifiees aux largeurs DU FILM, et qu'un appel
oublie ne puisse plus jamais redonner Cliffhanger en silence.

**Le correctif doit etre MESURE comme une amelioration, pas suppose.** Que les largeurs
different ne prouve pas que les nouvelles soient les bonnes : `DetectI0Layout` mesure le
decoupage de l'absolu du BIPEDE, et l'hypothese « les axes sont partages » est ECRITE dans
le commentaire mais n'a jamais ete verifiee sur le chemin world-object. Si la mesure dit que
le defaut fait mieux, on l'ecrit et on ne corrige pas.

## Decisions tranchees AVANT execution (ne pas re-arbitrer en cours de route)

1. **CORRIGE le 2026-08-15 — la source des largeurs est le CATALOGUE, pas `DetectI0Layout`.**
   La premiere ecriture de ce plan prescrivait `DetectI0Layout(dir)`. C'etait passer a cote
   de l'existant (regle 14 du depot), et la verification sur pieces le montre :

   - `data/titles/halo_infinite/reference/map_quant_bounds.json` porte, PAR CARTE, l'AABB du
     BSP **et** `AxisWidths` = `min(26, ceilLog2(ceil(60*extent)))` par axe
     (`filmdec/map_bounds.go:32-42`), produit par `cmd/mapquant-build`, offline-pur.
   - `map_bounds_test.go:49` verifie que `ridgeline` vaut **`{13, 13, 14}`** : le defaut cable
     de `WorldObjectPrecision` EST l'entree Cliffhanger du catalogue.
   - La chaine est validee de bout en bout par une source INDEPENDANTE : l'entree 0 du dump
     memoire runtime `ce_prec_ranges_14462cbe0.bin` egale au flottant pres l'entree
     `cliffhanger`/`ridgeline` du catalogue, et `ce_prec_widths_1445cc9e0.bin` est **32/32 en
     accord** avec la forme fermee (`.ai/ETAT_DE_L_ART_KILLWEAPON.md`, mesures du 2026-08-08).
   - Le geste est deja ecrit quelque part : `replay/ground_weapon_research_test.go:360-361`
     installe les largeurs **depuis l'entree du catalogue**, pas depuis le film.

   **Donc** : les largeurs viennent de la MEME entree de catalogue qui fournit deja
   `opt.WorldRange` a `BuildFromFilm`. Une seule source pour les bornes et les largeurs — les
   dissocier serait se donner deux verites a tenir.

   `DetectI0Layout` garde son role, et c'est celui que le commentaire du catalogue lui donne
   deja : **un CONTROLE**, pas une entree. Si le decoupage lu dans le film contredit les
   largeurs du catalogue, ce sont les BORNES qui sont fausses — cela se logge, cela ne se
   contourne pas.

1 bis. **Ou passe l'installation.** `ScanFilmWorldObjects` ne connait pas la carte du match ;
   l'appelant, si. Le descripteur doit donc DESCENDRE avec la plage, au meme endroit et par
   le meme chemin que `wr *Vec3Range` — c'est la seule facon de ne pas pouvoir armer l'un
   sans l'autre. Une carte hors catalogue garde le defaut ET le logge
   (`slog.WarnContext`), jamais de degradation silencieuse.
2. **Restauration systematique** : `prev := WorldObjectPrecision ; defer func(){ WorldObjectPrecision = prev }()`.
   Meme discipline que `SetAbilitySetHook` dans `ability_rank.go`.
3. **Le contrat de verrou est DOCUMENTE, pas ajoute** : l'appelant doit detenir
   `LockProcessDecode` (`BuildFromFilm` le fait deja). On ecrit la contrainte en tete de
   fonction, comme `ScanFilmAbilityRanks` l'ecrit.
4. **Pas de refonte en parametre.** Faire descendre le descripteur par signature toucherait
   cinq points de lecture et plusieurs signatures : hors perimetre. Le global reste, mais il
   devient impossible de l'oublier.

## Phases

### Phase 1 — MESURER le defaut (aucune ligne de production modifiee)

- [x] 1.1 Instrument versionne
      (`filmdec/world_object_precision_test.go`, gardes `WORLDPREC_FILM` /
      `WORLDPREC_BOUNDS` / `WORLDPREC_MAP`, `t.Skip` sans elles — verifie : les deux tests
      sautent). Il balaie `ScanFilmWorldObjects` DEUX FOIS — largeurs par defaut, puis
      largeurs **DU CATALOGUE** (decision n°1 corrigee) — et publie pour chacune :
      trajectoires, echantillons, et la marche detaillee (candidats / porte fermee /
      quantum saturE / acceptes). Une carte hors catalogue est dite telle et N'EST PAS
      mesuree (branche verifiee sur un nom de carte fabrique).
- [x] 1.2 **Le critere de qualite, enonce AVANT la mesure** : repris TEL QUEL du lot
      `d4be4ab95` — `equipBipedBox` + `equipBox.contains`, part d'echantillons dans
      l'emprise du nuage des BIPEDES du meme film, coordonnees NORMALISEES de l'AABB.
      Non circulaire, sans base.
- [x] 1.3 **7 films, 6 cartes, 6 valeurs de largeurs distinctes**, dont le temoin
      `000d5950`/Cliffhanger `[13 13 14]` (le correctif n'y change RIEN : mesure) et
      `0014603f`/Aquarius `[13 12 11]`. Cartes resolues hors DuckDB (le serveur de
      l'utilisateur tient les bases) : `match_registry.parquet` du snapshot 081, lu en
      DuckDB **en memoire** via `read_parquet` — aucun fichier de base ouvert.
- [x] 1.4 Ecart publie par film (trajectoires apparues / disparues / changees,
      echantillons entrant et sortant de l'emprise) — cf. tableau du journal ci-dessous.
- [x] 1.5 **CONTROLE DE COHERENCE catalogue <-> `DetectI0Layout`** (ajout de la decision
      n°1 corrigee) : **ACCORD 7 films sur 7**. Le decoupage lu dans le bitstream egale les
      largeurs deduites des bornes, sur les six cartes. Les bornes ne sont donc pas
      suspectes.

**Gate 1 : PASSE — les largeurs du catalogue font MASSIVEMENT mieux.** Tableau au journal.
Aucun film hors catalogue dans le corpus.

### Phase 2 — CORRIGER (ne s'ouvre que si le gate 1 montre une amelioration)

- [x] 2.1 **`replay.Options.WorldRange *filmdec.Vec3Range` devient
      `Options.MapQuant *filmdec.MapQuantEntry`** (decision 1 bis : le descripteur descend
      AVEC la plage, par le meme chemin — un seul champ, donc pas d'etat « bornes armees,
      largeurs oubliees »). `BuildFromFilm` installe les largeurs juste APRES
      `LockProcessDecode`, pour TOUT le decodage, et les restaure en differe
      (`installWorldObjectPrecision`). Largeurs absentes de l'entree -> defaut CONSERVE et
      `slog.Warn` (module, filmDir, defaut). `slog.Warn` et non `WarnContext` : `BuildFromFilm`
      ne prend pas de `ctx` et tout le fichier journalise ainsi ; ajouter un `ctx` a la
      signature serait hors perimetre. Appelants migres : `replaybuild.BuildMatch` (il tenait
      deja `entry`) et `cmd/zone-attribution`.
- [x] 2.2 Commentaire de `WorldObjectPrecision` et du setter reecrits dans le MEME commit :
      le defaut est nomme pour ce qu'il est (l'entree `cliffhanger`/`ridgeline` du catalogue),
      l'installateur de production est nomme, le contrat de verrou est ecrit, et
      `DetectI0Layout` est remis a sa place de CONTROLE. Les chiffres mesures y sont.
- [x] 2.3 Les cinq points de lecture, verifies SUR PIECES :
      `projectiles.go:57` et `:335` sont DANS `ScanFilmWorldObjects`, appele par
      `BuildFromFilm` -> ils prennent la valeur installee. `traverse.go:251`/`:254`
      (`consumeByName`) la prennent aussi pour tout balayage lance DANS `BuildFromFilm`.
      `keyframe_ground_weapons.go` **ne lit pas le global** : sa seule mention est un
      commentaire de parente d'archetypes (et il balaie des images-cles, pas des records
      delta). `position_capture.go:241` **est INATTEIGNABLE** : le repli est garde par
      `absoluteAxisW > 0`, dont le defaut vaut 14 et dont le seul ecrivant
      (`killsource/calibrate.go:85`) balaie `axisWMin=6 .. axisWMax=26` — jamais 0.
- [x] 2.4 Le regenerateur du golden (`decodeFilmInputs`) refait le MEME geste que la
      production (il pretendait « rejoue EXACTEMENT la sequence de BuildFromFilm » et
      n'installait pas les largeurs) : `goldenWorldRange` devient `goldenMapQuant` et rend
      l'entree ENTIERE.

**Gate 2 : PASSE.** `go build ./...` EXIT 0 · `go vet ./...` EXIT 0 ·
`go test ./internal/analysis/...` EXIT 0 · `golangci-lint run --new-from-merge-base=origin/main`
**0 issue** · `go test ./internal/replaybuild/... ./internal/api/wire/...` verts.

**Le golden NE BOUGE PAS, et c'est la mesure qui le predisait** : le film de reference
`000d5950` EST Cliffhanger, dont les largeurs de catalogue `[13 13 14]` SONT le defaut de
paquet. La phase 1 l'a mesure comme temoin — 580 trajectoires ti=41 identiques, 92,11 % des
deux cotes, ecart +0,00 point. Preuve qu'il ne s'agit pas d'un correctif non branche :
le fixture a ete **REGENERE depuis le vrai film** (`-update` + `REPLAY_FILM_DIR`, 83 s,
580 projectiles) et ressort **identique OCTET POUR OCTET** ; et les garde-rails de la
phase 3 ont ete vus ROUGES en retirant le branchement.

### Phase 3 — GARDE-RAIL (meme commit que la phase 2)

- [x] 3.1 `replay/world_object_precision_guard_test.go` :
      `TestInstallWorldObjectPrecision` (installe PUIS restaure — la restauration ne prouve
      rien seule, l'installation est donc verifiee pendant),
      `TestInstallWorldObjectPrecisionKeepsDefaultWithoutWidths` (entree sans largeurs ->
      defaut garde, degradation loggee), `TestBuildFromFilmRefusesWithoutMapQuant`, et
      `TestBuildFromFilmWiresWorldObjectPrecision` (le BRANCHEMENT : `BuildFromFilm` doit
      installer depuis `opt.MapQuant`, **en differe**, et **apres** la prise du verrou).
      **Pourquoi le branchement se verifie sur la SOURCE et non en executant `BuildFromFilm`**
      : le seul film versionne (`MiniFilmDir`) est une fenetre de paquets DELTA **sans
      image-cle de bipede** — `ScanFilmBipedPositions` echoue avant d'atteindre les objets du
      monde. Constate en l'essayant, pas suppose.
- [x] 3.2 `filmdec/world_object_precision_guard_test.go` :
      `TestWorldObjectPrecisionReadersAreAllowlisted` balaie **tout `internal/` + `cmd/`**
      (hors `_test.go`) et exige que chaque fichier de production mentionnant
      `WorldObjectPrecision` figure dans une **allowlist datee 2026-08-15 portant la raison
      de chaque entree**. Il tombe DANS LES DEUX SENS : lecteur non declare, et entree
      d'allowlist devenue morte.

**Gate 3 : PASSE, et les garde-rails ont ete VUS ROUGES** (quatre retraits, chacun suivi du
retour au vert) :

| retrait | test devenu ROUGE |
|---|---|
| `defer installWorldObjectPrecision(*opt.MapQuant, ...)` retire de `BuildFromFilm` | `TestBuildFromFilmWiresWorldObjectPrecision` |
| corps de `installWorldObjectPrecision` neutralise (`return func(){}`) | `TestInstallWorldObjectPrecision` — « largeurs NON INSTALLEES : [13 13 14], attendu [17 17 16] » |
| fichier de production ajoute lisant `WorldObjectPrecision` hors allowlist | `TestWorldObjectPrecisionReadersAreAllowlisted` |
| entree d'allowlist pointant un fichier qui ne le mentionne plus | `TestWorldObjectPrecisionReadersAreAllowlisted` |

## Regles dures

- Aucun autre correctif dans ce diff. Toute decouverte va en « Decouvertes ».
- Aucune base DuckDB ouverte en ecriture : le serveur de l'utilisateur tourne.
- Un seul decodage `filmdec` par process (globaux de paquet).
- `slog` structure pour toute degradation ; jamais `fmt.Println`.

## Statuts et cloture

`[x]` fait · `[~]` couvert ailleurs (reference) · `[!]` non traite (justification ecrite).
Aucune case vide a la cloture. Entree datee au `.ai/thought_log.md`. Toute piste non traitee
au `.ai/V7.5/REGISTRE_REPORTS.md` avec sa condition de reprise.

## Journal de la phase 1 (2026-08-15) — le tableau, avec ses denominateurs

Instrument : `filmdec/world_object_precision_test.go`. UN FILM PAR PROCESSUS. Duree 25 s a
4 min 15 par film. Critere = part d'echantillons dans l'emprise du nuage des BIPEDES du meme
film, coordonnees NORMALISEES de l'AABB.

**ti=41 — LE CHEMIN DE PRODUCTION** (`BuildFromFilm` -> `ScanFilmProjectiles`) :

| film | carte (module) | largeurs catalogue | controle | defaut | catalogue | ecart | echantillons |
|---|---|---|---|---|---|---|---|
| `000d5950` | Cliffhanger (ridgeline) | `[13 13 14]` | ACCORD | **92,11 %** | **92,11 %** | **+0,00** | 13 544 / 13 544 |
| `00162144` | Smallhalla (fo08_wetland) | `[15 15 17]` | ACCORD | 65,21 % | **99,79 %** | +34,58 | 11 725 / 11 725 |
| `00502e52` | Bazaar (ctf_bazaar) | `[17 17 16]` | ACCORD | 0,09 % | **99,41 %** | +99,32 | 12 694 / 12 694 |
| `07aa428d` | Illusion (ctf_illusion) | `[18 18 17]` | ACCORD | 0,51 % | **99,61 %** | +99,10 | 16 956 / 16 964 |
| `64e8adfa` | Catalyst (catalyst) | `[15 15 15]` | ACCORD | 28,46 % | **99,60 %** | +71,13 | 14 124 / 14 124 |
| `9edfcaa9` | Oasis (btb_exiled) | `[15 15 14]` | ACCORD | 31,31 % | **98,96 %** | +67,65 | 9 128 / 9 128 |
| `0014603f` | Aquarius (ctf_aquarius) | `[13 12 11]` | ACCORD | — | — | — | **aucun slot ti=41 dans les images-cles** |

**ti=37 — l'archetype du taux de reference (97,2 %)**, meme corpus, meme critere :

| film | defaut | catalogue | ecart | echantillons | trajectoires (apparues / disparues / changees) |
|---|---|---|---|---|---|
| `000d5950` | **92,28 %** | **92,28 %** | **+0,00** | 43 475 / 43 475 | 0 / 0 / 1 |
| `0014603f` | 2,14 % | **99,71 %** | +97,57 | 3 089 / 3 090 | 0 / 0 / 70 |
| `00162144` | 64,52 % | **98,89 %** | +34,38 | 43 060 / 43 069 | 6 / 3 / 364 |
| `00502e52` | 0,09 % | **98,03 %** | +97,94 | 34 198 / 34 200 | 1 / 0 / 384 |
| `07aa428d` | 0,55 % | **98,80 %** | +98,25 | 34 391 / 34 395 | 2 / 1 / 403 |
| `64e8adfa` | 27,03 % | **96,47 %** | +69,44 | 57 673 / 57 678 | 10 / 8 / 600 |
| `9edfcaa9` | 32,09 % | **96,93 %** | +64,84 | 37 491 / 37 512 | 11 / 6 / 620 |

Exemple d'ecart en echantillons (`00502e52`, ti=37) : **32 dans l'emprise -> 33 527**,
soit 34 166 hors emprise -> 673.

**CE QUE LA MESURE NE DIT PAS, ecrit avant de conclure :**

1. **L'emprise est une BOITE.** Un film dont les bipedes couvrent presque tout `[0,1]` sur un
   axe ne discrimine rien sur cet axe. Le critere est severe la ou la boite est etroite
   (`00502e52` : X`[0,823 0,867]` Y`[0,246 0,266]` Z`[0,147 0,162]` -> il separe 0,09 % de
   99,41 %) et faible la ou elle est large (`00162144`, Z`[0,000 0,882]` -> le defaut y garde
   65,21 %). **Le rang des films entre eux ne mesure donc pas la qualite du decodage.**
2. **Ni le 92,11 % de Cliffhanger ni le 99,79 % de Smallhalla ne sont un taux de justesse.**
   L'emprise des bipedes n'est pas la zone jouable : un objet pose sur une corniche ou une
   grenade lancee par-dessus une rambarde sort legitimement du nuage des pas. Le critere dit
   « meme repere », pas « bonne position au centimetre ».
3. **Aucune verification en coordonnees MONDE.** Tout est en normalise. Que l'echelle metrique
   soit juste tient aux BORNES, controlees ailleurs.
4. **Une seule direction testee.** On a compare le defaut au catalogue. On n'a pas cherche si
   un TROISIEME jeu de largeurs ferait mieux que le catalogue.
5. **Ni `ti=42` ni `ti=38` mesures** : le corpus porte ti=41 (production) et ti=37 (ancrage).
6. **La cle de trajectoire peut collisionner** (deux segments d'un meme couple slot/generation
   nes au meme instant) : les comptes « changent / identiques » sont approches a ~0,4 % pres.
   Les taux, eux, sont comptes sur les ECHANTILLONS et sont exacts.
7. **`0014603f` n'apporte rien a ti=41** : la carte la plus eloignee du defaut n'a aucun slot
   de projectile dans ses images-cles. Le temoin `[13 12 11]` ne vaut que par ti=37.

## Decouvertes — consigner, ne pas traiter

- L'hypothese « les axes du world-object sont ceux de l'absolu du bipede » est ecrite dans
  `traverse.go:137-148` et n'a jamais ete verifiee independamment. **Elle est VERIFIEE par ses
  consequences le 2026-08-15** : sur 6 cartes, poser les largeurs de la carte fait passer les
  objets du monde de 0,09-65 % a 96,5-99,8 % dans l'emprise des bipedes.
- `TraversalPrecision` (le chemin delta, `{6,6,6}`) porte le meme genre d'avertissement
  (« banc de calibration propre a Cliffhanger, pas un decodeur »). Hors perimetre.
- **`traverseComponentLoop` (traverse.go:1169) lit `WorldObjectPrecision` pour AVANCER LE
  CURSEUR sur `object-position-component`.** Les balayages qui passent par lui HORS
  `BuildFromFilm` — dont le decodeur de trames de `killsource` — gardent donc les largeurs de
  Cliffhanger, et une largeur fausse **desaligne la suite du record**, pas seulement la
  position. Non traite ici (regle « zero fix opportuniste ») : chaine d'appel distincte, qui
  demande de faire descendre l'entree de catalogue jusqu'a `killsource.Decode`. **Porte au
  registre des reports.**
- `position_capture.go:241` est un repli mort en l'etat (cf. item 2.3). Ne pas le supprimer
  dans ce diff (hors perimetre) ; sa mort est desormais ecrite dans l'allowlist du garde-rail.
