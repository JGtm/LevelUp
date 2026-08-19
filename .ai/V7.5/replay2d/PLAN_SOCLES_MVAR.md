# PLAN — TOUS LES SOCLES DEPUIS LES FICHIERS DE CARTE (.mvar)

Lot de MESURE SEULE, ouvert le 2026-08-19. Worktree frere `LevelUp-wt-socles-mvar`
(branche `wt/socles-mvar`, base `feat/v75`). Aucune publication, aucun changement de
production : ce plan mesure et rend un verdict. Une section « production proposee » ne
s'ecrit QUE si la mesure la porte.

## 1. La question

La detection actuelle des socles (`ground_weapon_pads.go`) exige une RECURRENCE : une
position ne devient un socle qu'a partir de DEUX apparitions d'armes de meme famille dans
le film. Consequence mesuree et acceptee jusqu'ici : un socle dont l'arme n'a ete servie
qu'une fois est INVISIBLE, et l'objet deja pose a t=0 n'est pas repliqué. Le film sous-
compte donc les socles par construction (temoin : `530820e5`, Catalyst CTF, rend 5 socles
la ou deux autres films de la meme carte en rendent 10 aux memes coordonnees).

D'ou la demande : **chercher tous les socles DES LE DEPART**, dans les fichiers de carte,
pour ne rien manquer.

## 2. Acquis verifies sur pieces (2026-08-19)

| Piece | Chemin absolu | Ce qu'elle donne |
|---|---|---|
| Decodeur Bond CB2 | `C:\Users\Guillaume\Projects\LevelUp\apps\go-api\internal\analysis\replay\mapvar\cb2.go` | `DecodeRoot` generique ; tout tag inconnu = erreur, jamais de saut silencieux |
| Grammaire `.mvar` | `...\mapvar\mapvar.go` | `root[3]` objets (type_id, pos, up, forward, flags, categorie, team, labels, instance_id, forme) ; `root[10][1]` table de chaines ; **`root[6]` et `root[11]` NON EXPLOITES** |
| Roles d'objectif | `...\mapvar\objectives.go` | labels = murmur3_x86_32(seed 0) du nom snake_case ; 28 noms resolus ; les `*_include/_exclude` sont des FILTRES DE MODE, pas des roles |
| Generateur du catalogue | `...\apps\go-api\cmd\mapobj-build\` | `--from-file` + `--dump-objects` (dump JSON de TOUS les objets), `--refresh-from <dossier>` hors ligne, `--save-mvar` |
| Depot local des `.mvar` | `C:\Users\Guillaume\Projects\LevelUp\.ai\re_dump\mapvar\` | **199 fichiers**, nommes `{carte}_{fichier}.mvar` |
| Catalogue fige | `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_objectives.json` | schema 2, cle = map_id, 72 cartes, champs `names[]`, `objectives[]`, `bounds`, `level_id`, `objects_n` |
| Artefacts de rejeu cuits | `C:\Users\Guillaume\Projects\LevelUp\data\cache\replays\halo_infinite\{film}.json` | `weaponPads[]` = L'ORACLE (x, y, z au centimetre, `weapon`, `spawns`, `presence`) |
| Correspondance film -> carte | `...\internal\analysis\replay\objectifs_phase1_drapeau_test.go` (`objCTFModules`, `objCTFCarteNom`) | Catalyst : `64e8adfa`, `530820e5`, `01e1f945` ; Cliffhanger : `bcb6d393`, `000d5950` |
| Piege connu | memoire du depot | une carte Forge `.mvar` = **canevas + rack** ; Smallhalla est Forge (canevas `fo08_wetland`) |

### 2.1 L'oracle, extrait des artefacts locaux (pas des artefacts publies)

**Catalyst** (`e859cf75-9b8a-429a-91be-2376681c8537`, module `catalyst`) — 10 socles
d'armes aux MEMES coordonnees dans `01e1f945` (KOTH) et `64e8adfa` (CTF), au centimetre :

```
 1  -9.738    -0.003    22.403
 2   9.472    12.411    24.003
 3   9.481   -12.404    24.015
 4 -11.046    -0.003    25.344
 5   5.160    -0.003    26.501
 6   0.003   -25.204    26.501
 7   0.003    25.298    26.501
 8  11.597    -0.003    22.600
 9   6.277     6.939    27.018
10   6.286    -6.945    27.018
```

Plus **1 socle de power-up** mesure au lot precedent : `0.257 ; -0.003 ; 21.36`
(`powerup_overshield`, presente en KOTH, absent des deux films CTF de la meme carte).
Total oracle Catalyst : **11 positions**.

**Cliffhanger** (`bcb6d393`, CTF, module `cliffhanger_ridgeline`) : 10 socles.
**Smallhalla** (`00162144`, module `smallhalla_map`) : 11 socles.
**Cliffhanger Super Fiesta** (`000d5950`) : **aucune cle `weaponPads`** — 0 socle mesure.

Fait etabli du lot precedent, qui borne ce qu'on peut esperer : **le socle appartient a la
carte, l'arme qui y apparait appartient au match** (epee et marteau au meme point selon le
film ; trois des dix socles de Catalyst portent une arme differente entre deux films).

## 3. Hypotheses, ecrites AVANT la mesure

- **H-SCENARIO (en tete, precision utilisateur du 2026-08-19)** — ce qui dicte ce que les
  socles font apparaitre, voire ce qui les MET EN PLACE, ce sont les **scenarios** (le
  variant de MODE), pas la carte seule. Une carte peut avoir davantage de socles sur
  certains modes ; en Fiesta il n'y en a en principe aucun. Deux formes a distinguer :
  - **H-SCENARIO-a (activation)** : les spawners sont dans le `.mvar` de carte et le mode
    les allume/eteint (labels `*_include` / `*_exclude`, deja connus du decodeur).
  - **H-SCENARIO-b (pose)** : les spawners vivent dans un fichier de MODE (variant de
    scenario / gametype, ou un `.mvar` par combinaison carte x mode) et le `.mvar` de
    carte n'en porte aucun.
  Le **test discriminant est deja disponible et hors ligne** : Cliffhanger, meme carte,
  deux modes, 10 socles (`bcb6d393`, CTF) contre 0 (`000d5950`, Super Fiesta). Si les deux
  matchs partagent le meme `.mvar` et que ce fichier porte les 10 spawners, H-SCENARIO-a
  est retenue et H-SCENARIO-b refutee pour la POSE (le mode ne fait qu'activer).
- **H1 (portage)** : le `.mvar` d'une carte DEV porte des objets a moins d'un metre de
  chaque socle de l'oracle.
- **H2 (identite)** : l'objet du `.mvar` porte de quoi savoir CE QU'IL FAIT SPAWN
  (reference d'arme `weap` / d'equipement `eqip`), ou bien il est generique et l'arme vient
  d'ailleurs. Les deux champs jamais decodes (`root[6]` regroupements, `root[11]`
  surcharges de proprietes indexees) sont les candidats a inspecter en priorite.
- **H3 (Forge)** : sur une carte Forge (Smallhalla), le rack d'objets porte les socles et
  le canevas n'en porte aucun — le piege « canevas + rack » impose de mesurer les DEUX
  fichiers avant de conclure.

## 4. Seuils, ecrits AVANT la mesure

| Mesure | Seuil de reussite | Refutation |
|---|---|---|
| **Appariement (Q2)** | par carte, **>= 90 %** des positions de l'oracle ont un objet du `.mvar` a **< 1 m** (distance 3D) | < 90 % : le `.mvar` de carte ne porte pas les socles |
| **Temoin negatif** | 100 tirages de N positions aleatoires dans l'emprise XY de la carte (Z conserve de l'oracle, graine fixe) : **<= 20 %** apparies a < 1 m | temoin > 20 % : la carte est si dense que « a moins d'un metre » ne discrimine rien — l'appariement ne prouve alors rien |
| **Rapport signal/temoin** | **>= 3** | < 3 : verdict negatif meme si le taux brut passe |
| **Diagnostic XY** (declare AVANT la mesure, ne remplace aucun seuil) | la distance est aussi mesuree en XY seul | si XY apparie a < 1 m et pas la 3D, l'objet est au bon endroit du plan mais a une autre ALTITUDE (origine au sol contre arme flottante) : a dire, sans requalifier le verdict |
| **Identite (Q3)** | l'arme est declaree LISIBLE si un champ de l'objet apparie prend **>= 2 valeurs distinctes** entre socles portant des armes distinctes ET est **constant** entre les deux films de la meme carte | sinon : « spawner generique, arme non lisible dans le `.mvar` » |
| **Generalisation (Q5)** | les 3 cartes passent le seuil d'appariement | 1 carte sur 3 : resultat partiel, ecrit comme tel |

**Gate d'arret global** : si la phase 2 (Catalyst, la carte la mieux mesuree) echoue au
seuil d'appariement OU au rapport signal/temoin, les phases 3 et 5 ne s'executent pas —
on ecrit le negatif avec ce qui a ete vu (phase 6 directement). La phase 4 s'execute dans
tous les cas : elle repond a H-SCENARIO, qui vaut meme en cas de negatif.

## 5. Contraintes d'execution

Lecture seule ; aucune base de donnees ouverte ; chemins ABSOLUS vers
`C:\Users\Guillaume\Projects\LevelUp\data\...` et `...\.ai\re_dump\mapvar\` ; instruments
sous garde d'environnement (`MVAR_FILE`, `MVAR_DIR`, `MVAR_ORACLE`) — sans la garde, le
test SKIPPE ; `CGO_ENABLED=0` ; `GOCACHE` local au worktree ; **un seul `go` a la fois** ;
jamais de push, de stash, ni de `git add -A`. Gates par phase :
`go vet ./internal/analysis/...`, `go test ./internal/analysis/...`, golangci 0 nouveau.


## 6. Phases

### Phase 0 — l'instrument (commit 1)

- [x] 0.1 Ecrire `internal/analysis/replay/mapvar/socles_research_test.go` : chargement
      d'un `.mvar` sous garde `MVAR_FILE`, inventaire des objets (type_id, categorie,
      labels resolus/non resolus, forme, nom d'instance si atteignable).
- [x] 0.2 Ajouter la lecture BRUTE de `root[6]` et `root[11]` (champs jamais decodes) —
      forme, cardinalite, correlation avec l'index d'objet.
- [x] 0.3 Gate : `go vet` + `go test` du paquet sans garde (skip propre).

### Phase 1 — que contient le `.mvar` de Catalyst ? (Q1) (commit 2)

- [x] 1.1 Inventaire complet de `catalyst_catalyst.mvar` (337 objets) : histogramme des
      type_id, des categories, des labels.
- [x] 1.2 Idem pour `catalyst_map.mvar` (357 objets) — deux fichiers pour la meme carte,
      il faut savoir lequel est le bon avant d'apparier.
- [x] 1.3 Verdict Q1 ecrit : le `.mvar` porte-t-il des objets de type « spawner » ?

### Phase 2 — appariement de l'oracle Catalyst (Q2) (commit 3)

- [x] 2.1 Coder les 11 positions de l'oracle (section 2.1) dans l'instrument, sous garde.
- [x] 2.2 Pour chaque position : objet du `.mvar` le plus proche, distance 3D, type_id,
      labels.
- [x] 2.3 Temoin negatif : 100 tirages, graine fixe, meme emprise, meme N.
- [x] 2.4 **GATE D'ARRET** : >= 90 % a < 1 m et signal/temoin >= 3, sinon aller en phase 4
      puis 6.

### Phase 3 — que fait spawn le socle ? (Q3) (commit 4)

- [x] 3.1 Sur les objets apparies : quels champs varient avec l'arme observee ?
- [x] 3.2 Croiser avec les deux films de Catalyst (armes differentes sur 3 socles).
- [x] 3.3 Inspecter `root[11]` (surcharges indexees) sur les index des objets apparies.
- [x] 3.4 Verdict Q3 ecrit : arme fixee dans le fichier, ou spawner generique ?

### Phase 4 — le variant de MODE (Q4 + H-SCENARIO) (commit 5)

- [x] 4.1 Inventorier les `.mvar` du depot par carte : combien de fichiers, quels noms
      (`ctf_breaker.mvar`, `ridgeline.mvar`, `map.mvar`...), quel `level_id`.
- [x] 4.2 Cliffhanger : comparer objet par objet les fichiers servant le match CTF
      (`bcb6d393`) et le match Super Fiesta (`000d5950`) — meme fichier ? memes objets ?
- [x] 4.3 Chercher, dans la chaine `himap` et le catalogue UGC, ce qu'une combinaison
      carte x mode expose comme fichiers (tags de scenario / gametype referencant des
      spawners).
- [x] 4.4 Verdict : H-SCENARIO-a (activation) ou -b (pose), ou « hors de portee hors
      ligne » ecrit comme tel.

### Phase 5 — generalisation (Q5) (commit 6)

- [x] 5.1 Cliffhanger (`bcb6d393`, 10 socles) : appariement + temoin.
- [x] 5.2 Smallhalla (`00162144`, 11 socles) : appariement + temoin, sur le rack ET sur le
      canevas (piege Forge).
- [x] 5.3 Verdict Q5.
- [x] 5.4 AJOUTE EN COURS D'EXECUTION (decouverte de 5.2, elle change la lecture de
      H-SCENARIO) : sur Forge, les socles portent des LABELS de mode que les cartes DEV
      n'ont pas. Tenter de resoudre les hashs inconnus les plus frequents par recherche
      murmur3 ciblee, sous le meme garde-fou anti-collision que `objectives.go`.

### Phase 6 — cloture (commit 7)

- [x] 6.1 Section « production proposee » (catalogue statique `map_weapon_pads.json`) OU
      negatif ecrit.
- [x] 6.2 Gates finaux : `go vet ./internal/analysis/...`,
      `go test ./internal/analysis/...`, golangci 0 nouveau.
- [x] 6.3 CR : reponses aux 5 questions avec chiffres, textes journal et registre.

## 7. Journal

- **2026-08-19** — Plan ouvert. Acquis verifies sur pieces (chaine `mapvar`, `mapobj-build`,
  depot des 199 `.mvar`, catalogue fige) ; oracle re-extrait des artefacts LOCAUX plutot que
  des artefacts publies (10 socles Catalyst identiques au centimetre entre `01e1f945` et
  `64e8adfa`). Hypothese H-SCENARIO placee en tete a la demande de l'utilisateur, avec son
  test discriminant deja disponible hors ligne (Cliffhanger 10 vs 0). Aucune mesure lancee
  a cette heure.
- **2026-08-19, phase 0 close** — Instrument `socles_research_test.go` ecrit dans le paquet
  `mapvar`, sous gardes `MVAR_FILE` / `MVAR_DIR` / `MVAR_ORACLE` / `MVAR_ORACLE_POINTS`.
  L'oracle n'est PAS code en dur : il se lit dans un artefact de rejeu cuit
  (`weaponPads`), plus des points fournis a la main pour le socle de power-up, qui vient
  d'une autre voie de mesure. Gates : `go vet` propre, `go test -run TestSocles` vert avec
  trois SKIP explicites sans garde. Report assume : la consigne du lot demande « ni journal
  ni registre », donc AUCUNE entree `.ai/thought_log.md` n'est ecrite — les textes partent
  au CR. C'est la seule derogation au contrat `plan-execution`, et elle est ordonnee.
- **2026-08-19, phase 1 close** — Inventaire des deux `.mvar` de Catalyst (meme `level_id`
  `-1044063363`, deux map_id distincts au catalogue).
  `catalyst_catalyst.mvar` : 337 objets, **36 type_id distincts**, 3 categories (-1, 1, 2),
  31 objets avec forme, 124 avec index d'equipe, table de chaines **VIDE**. Labels resolus :
  `infection_include` 109, `ctf_include` 8, `strongholds_include` 6, `extraction_zone` 6,
  `extraction_include` 5, `assault_include` 4, `flag_spawn` 3, `strongholds_zone` 3,
  `flag_delivery` 2, plus 4 unitaires. **9 hashs de label INCONNUS** (le plus frequent
  `-886053664`, 18 fois).
  `catalyst_map.mvar` : 357 objets, 42 type_id, 4 categories, 3 noms, **20 hashs inconnus**.
  `root[6]` DECODE (il n'etait pas seulement « non exploite », il n'avait jamais ete
  regarde) : 11 blocs, chacun `{.0 = offset de depart, .1 = liste d'entrees, .5.0 = un
  int32 de type hash}`. Les offsets s'enchainent (0, 135, 143, 167, 268, 293, 294, 300,
  337, 340, 346) et la somme des entrees vaut 348 = `root[6].2`. Chaque entree vaut
  `{0:1, 1:identifiant sequentiel, 6:0xFFFFFFFF, 8:index, 9:struct vide}` : c'est une
  **table d'allocation d'identifiants**, sans position ni reference d'objet. La coincidence
  « 11 blocs / 11 socles » est fortuite et le disque le montre.
  `root[11]` : present, type BT_MAP, **vide** sur Catalyst.
  **Verdict 1.3** : le fichier porte 36 a 42 types d'objets dont un seul est identifie
  (`-1239931096`, le volume d'objectif que `map_objectives.json` publie deja) ; RIEN ne les
  nomme (table de chaines vide sur les cartes DEV) ; aucune reference d'arme n'est lisible.
  Le seul discriminant disponible est donc la POSITION — c'est l'objet de la phase 2.
- **2026-08-19, phase 2 close — H1 CONFIRMEE, ET LARGEMENT** — les socles d'armes SONT dans
  le `.mvar` de la carte. **11 / 11 apparies** a moins d'un metre, et pas de justesse :
  distance mediane **0,01 m**, maximum **0,01 m** — le centimetre, la meme resolution que
  l'oracle. Temoin negatif 4,5 % (50 sur 1 100 tirages, graine 20260819) : rapport
  signal/temoin **22**, seuil du plan 3.
  Le resultat tient sur les TROIS films de la carte et sur les DEUX `.mvar` : `01e1f945`
  (KOTH) 11/11, `64e8adfa` (CTF) 11/11, `530820e5` (CTF, film qui ne voit que 5 socles) 6/6.
  Ce sont a chaque fois les MEMES objets, aux memes index (271 a 287).
  **Et les type_id sont discriminants** : `1597478195` porte les socles 4, 5, 6, 7 (marteau
  ou epee, SPNKr ou Cremateur, les deux snipers) ; `1649659840` porte les six autres ;
  `1585893648` porte le socle de power-up, a lui seul. Le fichier distingue donc au moins
  trois familles de socle.
  **Signal decisif pour la demande de depart** : l'inventaire compte **7** objets de type
  `1649659840` et **5** de type `1597478195`, soit **12 socles d'armes** la ou le meilleur
  film n'en montre que **10**. Deux socles existent que le film ne voit jamais. Gate d'arret
  NON declenche : phases 3 et 5 s'executent.
- **2026-08-19, phase 3 close — H2 TRANCHEE : la FAMILLE est lisible, l'ARME ne l'est pas.**
  Les objets de socle sont NUS : categorie absente, aucun index d'equipe, aucun label,
  aucune forme, et un sac de proprietes RIGOUREUSEMENT identique d'un socle a l'autre
  (`#8.0[0].13` vide, `#8.1[0]` vide, `#8.24[0] = {0:4}` sur les treize). Le seul champ qui
  varie est le **type_id**, et il code une FAMILLE, confirmee par le croisement des trois
  films et du catalogue d'armes :
  - `1597478195` — **armes de pouvoir** : Duelist Energy Sword et Diminisher of Hope au meme
    objet 276 selon le match, Fuel Rod SPNKr / M41 SPNKr / Cindershot a l'objet 287,
    S7 Sniper aux objets 282 et 283 dans les trois films ;
  - `1649659840` — **armes de rack** : CQS48 Bulldog, Disruptor ou Mangler, VK78 Commando ou
    Vestige Carbine ou BR75, Sentinel Beam ;
  - `1585893648` — **le socle de power-up**, seul de son type sur la carte.
  Un MEME objet porte donc des armes differentes selon le match : l'arme n'est pas dans le
  fichier de carte, et le fait etabli du lot precedent (« le socle appartient a la carte,
  l'arme au match ») se verifie a la source.
  `root[11]` est present mais VIDE sur Catalyst : aucune surcharge indexee a croiser (3.3).
  **ET LE COMPTE HONNETE** : 13 objets de socle, mais **11 emplacements distincts** — les
  deux objets « jamais vus » sont a 4,7 cm et 9 mm d'un socle deja vu, donc le meme
  emplacement declare deux fois, pas deux socles de plus. Sur Catalyst, le fichier de carte
  ne revele AUCUN emplacement que le film ignorait. Ce qu'il apporte est ailleurs : il les
  donne TOUS sans exiger de recurrence (le film `530820e5` n'en montre que 6 sur 11) et il
  donne leur famille.
- **2026-08-19, phase 4 close — H-SCENARIO-b REFUTEE, H-SCENARIO-a RETENUE.**
  Corpus : **199 fichiers lus, 0 en echec**. Une carte expose un ou plusieurs `.mvar`, et
  les seconds fichiers portent souvent un nom de MODE (`ctf_breaker.mvar`,
  `ctf_aquarius.mvar`, `va_behemoth.mvar`, `ridgeline.mvar`) — la piste « un fichier par
  combinaison carte x mode » avait donc une base de nommage. Elle ne resiste pas a la
  mesure. Cliffhanger expose `cliffhanger_map.mvar` (453 objets) et
  `cliffhanger_ridgeline.mvar` (443 objets), meme `level_id` `-1009396204` : **les deux
  portent EXACTEMENT les memes 17 socles, aux memes positions, avec les memes type_id**
  (10/10 apparies contre l'oracle CTF `bcb6d393`, mediane 0,01 m, temoin 5,6 %). Le choix
  du fichier ne change donc rien.
  **Et c'est ce qui tranche H-SCENARIO** : la meme carte, en Super Fiesta (`000d5950`),
  rend **zero** socle mesure — son artefact cuit n'a meme pas de cle `weaponPads`. Le
  fichier de carte, identique dans les deux cas, POSE les socles ; il n'explique pas leur
  extinction. Ce qui allume ou eteint un socle vient d'ailleurs, et **le `.mvar` n'en dit
  rien** : ces objets ne portent AUCUN label, donc aucun filtre `*_include` / `*_exclude`
  comme en portent les objets d'objectif. La lecture « le mode active » est retenue par
  elimination, pas par lecture directe — c'est une nuance, elle est ecrite.
  4.3, sur pieces : l'asset `Maps` de l'UGC n'expose que des `.mvar`
  (`cmd/mapobj-build/fetch.go`, `Files.FileRelativePaths`). Mais le depot SAIT DEJA parler
  aux variants de mode : `internal/platform/halo/discovery_types.go` declare
  `AssetTypeGameVariant -> "ugcGameVariants"`, et `internal/openspartan/mapper/mapper.go`
  enregistre `UgcGameVariant.AssetId` / `VersionId` de chaque match. L'endpoint
  `/hi/ugcGameVariants/{assetId}/versions/{versionId}` est donc atteignable avec
  l'outillage existant — un appel reseau, hors du perimetre hors ligne de ce lot, mais la
  piste est nommee et l'identifiant necessaire est deja en base.
- **2026-08-19, phase 5 close — LA GENERALISATION TIENT, 3 CARTES SUR 3, ET C'EST LA QUE LE
  GAIN APPARAIT.**
  **Cliffhanger** (`cliffhanger_ridgeline.mvar`, 443 objets) contre `bcb6d393` : **10 / 10**
  a moins d'un metre, mediane 0,01 m, temoin 5,6 %. Le fichier porte **17 objets de socle,
  17 emplacements distincts** — le film n'en montre que 10. **SEPT EMPLACEMENTS QUE LE FILM
  NE VOIT JAMAIS**, et ce ne sont pas des doublons : (14,3 ; 22,7), (19,9 ; 3,2),
  (32,0 ; -0,4), (24,5 ; -18,4), (26,5 ; -17,8), (19,3 ; 14,2), (13,7 ; -15,1).
  **Smallhalla**, carte Forge, mesuree sur les DEUX fichiers du piege canevas+rack :
  - le rack `smallhalla_map.mvar` (3 901 objets, 222 type_id) rend **11 / 11**, mediane
    0,01 m — mais **le temoin large ECHOUE a 75,5 %**. C'est le seuil du plan qui parle :
    sur une carte Forge, « un objet a moins d'un metre » est vrai presque partout, et
    l'appariement brut n'y prouve RIEN. Le temoin n'est pas efface ; un temoin RESTREINT
    aux deux type_id de socle a ete ajoute (37 objets sur 3 901) et rend **4,7 %**, soit un
    rapport signal/temoin de 21. C'est cette mesure-la qui conclut, et la nuance est ecrite.
    Le rack porte **37 objets de socle, 26 emplacements distincts**, 11 vus par le film :
    **QUINZE emplacements invisibles**.
  - le canevas `smallhalla_fo08_wetland.mvar` (100 objets) rend **0 / 11**, distances 81 a
    102 m. Le piege canevas+rack est confirme dans le bon sens : sur Forge, TOUT est dans le
    rack, le canevas ne porte aucun socle.
  **Decouverte (item 5.4, ajoute en cours d'execution)** : sur Forge, les socles portent des
  LABELS, ce que les cartes DEV ne font pas — `stockpile_include`, `stockpile_exclude`,
  `infection_exclude`, `ctf_multi_exclude` y cotoient des hashs inconnus dont un
  (`-831896525`) revient trois a quatre fois sur le meme objet. C'est la premiere trace
  LISIBLE d'une activation par mode. Tentative de resolution par recherche murmur3 ciblee
  sur un vocabulaire du domaine : **1 010 100 candidats, 0 hash resolu sur 9**. Ils restent
  inconnus — on ne devine pas un libelle (garde-fou `objectives.go`).

## 8. Verdict

**H1 CONFIRMEE : LES SOCLES SONT DANS LE FICHIER DE CARTE, AU CENTIMETRE.** Trois cartes,
32 positions d'oracle, **32 appariees a moins d'un metre** — mediane 0,01 m partout.

| Carte | Fichier | Oracle | Apparies | Mediane | Temoin large | Temoin restreint | Objets socle | Emplacements | Vus | **Invisibles** |
|---|---|---|---|---|---|---|---|---|---|---|
| Catalyst | `catalyst_catalyst.mvar` | 11 | 11/11 | 0,01 m | 4,5 % | 0,5 % | 13 | 11 | 11 | **0** |
| Catalyst | `catalyst_map.mvar` | 11 | 11/11 | 0,01 m | 4,5 % | — | 13 | 11 | 11 | **0** |
| Cliffhanger | `cliffhanger_ridgeline.mvar` | 10 | 10/10 | 0,01 m | 5,6 % | 0,4 % | 17 | 17 | 10 | **7** |
| Cliffhanger | `cliffhanger_map.mvar` | 10 | 10/10 | 0,01 m | 5,8 % | — | 17 | 17 | 10 | **7** |
| Smallhalla (rack) | `smallhalla_map.mvar` | 11 | 11/11 | 0,01 m | **75,5 %** | 4,7 % | 37 | 26 | 11 | **15** |
| Smallhalla (canevas) | `smallhalla_fo08_wetland.mvar` | 11 | 0/11 | 81 a 102 m | 0 % | — | 0 | — | — | — |

**Reponses aux cinq questions**

1. **Le `.mvar` d'une carte DEV porte-t-il des spawners avec leurs positions ?** OUI. Trois
   type_id, jamais identifies jusqu'ici : `1597478195`, `1649659840`, `1585893648`.
2. **Chaque socle de l'oracle a-t-il un objet a moins d'un metre ?** OUI, 32 sur 32,
   mediane 0,01 m, contre un temoin de 0,4 a 5,6 % sur les cartes DEV.
3. **L'objet porte-t-il ce qu'il fait spawn ?** NON pour l'arme, OUI pour la FAMILLE. Un
   meme objet porte l'epee ou le marteau selon le match. Le type_id, lui, separe
   **armes de pouvoir** (`1597478195` : epee, marteau, SPNKr, Cindershot, S7 Sniper),
   **armes de rack** (`1649659840` : Bulldog, Disruptor, Mangler, Commando, Vestige, BR75,
   Sentinel Beam) et **socle de power-up** (`1585893648`). Le sac de proprietes est
   identique d'un socle a l'autre ; `root[11]` est vide ; la table de chaines aussi sur les
   cartes DEV.
4. **Le variant de mode est-il un fichier different ?** NON. Les deux `.mvar` d'une meme
   carte portent les MEMES socles. La difference 10 socles (CTF) contre 0 (Super Fiesta) sur
   Cliffhanger ne vient d'aucun fichier de carte : **le fichier POSE, le mode ALLUME**.
   L'activation n'est pas lisible dans le `.mvar` des cartes DEV (aucun label sur ces
   objets) ; elle l'est partiellement sur Forge, ou les socles portent `stockpile_include`,
   `infection_exclude`, `ctf_multi_exclude` et des hashs non resolus.
5. **Generalisation ?** OUI, 3 cartes sur 3, avec une reserve ecrite : sur une carte Forge,
   le temoin large echoue (75,5 %) et seul le temoin restreint aux type_id de socle conclut.

**LE GAIN, chiffre** : **22 emplacements de socle que le film ne montre jamais** (7 sur
Cliffhanger, 15 sur Smallhalla, 0 sur Catalyst). La demande de depart — « ne rien manquer » —
est satisfaite, et elle l'est la ou on ne l'attendait pas : Catalyst, la carte la mieux
mesuree, n'avait rien de cache.

## 8 bis. Phase de production proposee (a decider, RIEN N'EST FAIT)

Un catalogue statique **`data/titles/halo_infinite/reference/map_weapon_pads.json`**, sur le
modele exact de `map_objectives.json` : cle `map_id`, produit HORS LIGNE depuis
`.ai/re_dump/mapvar/`, jamais appele au rejeu.

- **Producteur** : etendre `cmd/mapobj-build` (il a deja `--refresh-from`, le depot de
  `.mvar` et l'ecriture de catalogue) ou un `cmd/mappad-build` jumeau. Aucune requete
  reseau supplementaire : les 199 `.mvar` sont deja en depot.
- **Contenu par socle** : `pos {x,y,z}`, `type_id` brut, `famille` derivee
  (`power_weapon` / `rack_weapon` / `powerup`), `emplacement` (regroupement a moins d'un
  metre, pour ne pas compter deux fois une position declaree deux fois).
- **La famille est une INFERENCE**, mesuree sur 3 cartes par correlation avec les armes
  observees. Elle se publie a cote du `type_id` brut, jamais a sa place — le jour ou elle
  est infirmee, on recalcule sans re-extraire.
- **Croisement au rejeu** : le statique dit OU sont tous les socles ; le film dit ce qu'ils
  ont fait apparaitre et quand.
- **LE PIEGE A NE PAS TENDRE, et il est serieux** : en Super Fiesta, Cliffhanger a 17 socles
  au fichier et ZERO actif en jeu. Publier le catalogue brut afficherait 17 socles fantomes
  sur ce rejeu. La regle de croisement doit donc etre conditionnee par le film (par exemple :
  ne montrer les socles statiques que si le film en confirme au moins un), et cette regle
  demande sa propre mesure. **C'est une decision produit, elle revient a l'utilisateur.**

## 9. Decouvertes (notees, NON traitees)

- `root[6]` n'est pas une table de regroupement d'objets mais une **table d'allocation
  d'identifiants** : 11 blocs, offsets enchaines, somme = `root[6].2`. Le commentaire de
  grammaire de `mapvar.go` est a preciser le jour ou on y touche.
- `root[11]` est present mais VIDE sur toutes les cartes mesurees, alors que la grammaire
  le presente comme « surcharges de proprietes indexees ».
- La table de chaines `root[10][1]` est VIDE sur les cartes DEV et pleine sur les cartes
  Forge (Smallhalla : 99 noms). `map_objectives.json` ne publie donc de `names[]` que pour
  les cartes communautaires.
- Une carte expose 1 a 3 `.mvar` ; les seconds portent des noms de mode
  (`ctf_breaker.mvar`, `va_behemoth.mvar`, `ridgeline.mvar`) sans porter de socles
  differents. Le catalogue enregistre parfois DEUX map_id pour la meme carte
  (`e859cf75-...` et `f7e8cde9-...` pour Catalyst, meme `level_id`).
- 9 a 20 hashs de label restent inconnus par carte ; la recherche murmur3 ciblee
  (1 010 100 candidats) n'en resout aucun.
- Les type_id de socle (`1597478195` = `0x5F379533`, `1649659840` = `0x6253CFC0`,
  `1585893648` = `0x5E86D110`) ne figurent dans aucun index du depot. Les chercher dans les
  tags du jeu installe (chaine `himap`) donnerait leur nom — piste, hors perimetre.

## 10. Journal de cloture

- **2026-08-19, phase 6 close** — Verdict et section « production proposee » ecrits.
  Gates : `go vet ./internal/analysis/...` = 0, `go test ./internal/analysis/...` = 0
  (aucun echec, les instruments SKIPPENT sans garde), `golangci-lint run
  ./internal/analysis/replay/mapvar/...` = **0 issues**. 25 items sur 25 statues, aucune
  case vide. Rappel du seul report du lot, ordonne par la consigne : ni entree
  `.ai/thought_log.md` ni entree au registre — les textes partent au CR.
