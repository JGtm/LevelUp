# Lot D — phase 0 : l'usage d'un equipement, date par ses charges. MESURE ET VERDICT

> Branche `wt/usages-film`. Lecture seule : aucune publication, aucun champ de document, aucune
> ecriture DuckDB. Instrument versionne et garde :
> `apps/go-api/internal/analysis/replay/equipment_uses_research_test.go` +
> `equipment_uses_join_test.go` (garde `EQ_FILM`, un film par processus).
>
> **VERDICT GATE 0 : NON ATTEINT, et le negatif est PROPRE.** L'appariement prescrit
> (decroissance d'`equipment-charges-remaining` i27 contre les evenements de grappin i59) rend
> **0 / 501** la ou le plan exige 80 %. Le temoin decale de +7 s rend lui aussi 0 / 501 : le
> zero ne vient pas d'un instrument muet, il vient de ce qu'il n'y a rien a apparier.
> **i27 sort de la reserve comme NEGATIF** (registre `:157`, mise a jour = superviseur), lot D
> clos `[!]`.

## 0. Ce qui a ete joue, et sur quoi

| | |
|---|---|
| Corpus prescrit (12 films) | `000d5950` `06dfe6d9` `64e8adfa` `530820e5` `53ce4390` `0a247154` `01e1f945` `7344d24f` `696a9d7c` `24dbb67d` `00162144` `02784ce1` |
| Complement (2 films) | `606d9844` `8076f97f` — joues pour reconcilier le corpus du plan §1.2 avec la liste prescrite, qui differe de deux films |
| Films lus | cache du principal, chemins ABSOLUS, lecture seule. `1b1e380f` JAMAIS ouvert |
| Bornes de carte | catalogue `map_quant_bounds.json` du principal, carte AUTO-DETECTEE par la signature des largeurs d'axe lues dans le film (meme methode qu'`i59_anchor_test.go`) — 14 films sur 14 tranches |
| Seuils | ecrits AVANT la mesure, repris du plan : appariement >= 80 %, temoin +7 s <= 10 %, coherence du pont >= 90 %, controle croise >= 80 %. Fenetre d'appariement ± 0,5 s |

**Trois temoins, pas un.** (a) les MEMES evenements decales de +7 s ; (b) les memes evenements
contre les signaux des AUTRES familles d'equipement ; (c) le plancher de bruit de l'ancrage
lui-meme, publie par le balayage des poses.

**LE NEGATIF N'EST PAS UN ARTEFACT DE BUILD, et le corpus permet de le dire.** Le garde-rail
d'empreinte pose au lot 0 a crie sur **6 films des 14** (`06dfe6d9` et `0a247154` : 116 blocs ;
`00162144`, `24dbb67d`, `02784ce1`, `8076f97f` : 118 blocs, trois empreintes distinctes) —
c'est-a-dire **au moins quatre versions du jeu dans le corpus**, exactement ce que le lot 0
avait mesure. Les 8 autres films portent l'empreinte connue, `000d5950` compris. Le resultat est
le MEME des deux cotes : les index de composant sont resolus PAR NOM (`equipmentFieldIndices`)
et le dispatch du deser l'est aussi, donc i26 et i27 sont lus au bon endroit sur chaque build.
Un negatif qui ne tiendrait que sur les films d'un build serait suspect ; celui-ci tient sur
tous. Traces : `lotD/logs/<short8>.err`.

## 1. D.0.1 — volumes, valeurs, transitions `[x]`

### 1.1 Les volumes attendus sont retrouves

| | 12 films prescrits | registre `:157` (2026-08-15) |
|---|---|---|
| records delta ti=37 | 881 551 | 1 097 619 |
| i27 `charges-remaining` annonce au masque | **13 384** (1,52 % des records) | 16 125 (1,47 %) |
| i26 `energy-delay-ticks-left` | **9 750** (1,11 %) | 10 608 (0,97 %) |

L'ecart des totaux vient du CORPUS (celui du registre est 24 % plus gros), pas du decodage :
les deux taux par record sont du meme ordre, et sur `000d5950` l'instrument rend **exactement**
les chiffres du lot 0 — i26 820 annonces / 820 lues, i27 883 / 883, 0 porte fermee. Sur le
corpus du plan §1.2 (avec `606d9844` et `8076f97f` a la place de `00162144` / `02784ce1`) :
i26 8 625, i27 12 219.

### 1.2 Les transitions du lot 0 sont reproduites — puis expliquees

Groupees par la SEULE paire (slot, generation), c'est-a-dire comme le lot 0 les comptait :

| | 12 films | dont `000d5950` | lot 0 sur `000d5950` |
|---|---|---|---|
| i26 transitions / paires consecutives | 1 233 / 1 988 | 126 / 172 | 126 / 172 |
| i27 transitions / paires consecutives | 2 300 / 3 490 | 194 / 263 | 194 / 263 |

Accord a l'unite. **Mais ce comptage additionne deux objets differents.** La generation ne fait
que 2 bits : un slot repasse par la meme paire au cours d'un match, et `confirmPlacements` le
sait deja (sa cle de deduplication porte le debut de la vie). Rattachees a une VIE D'OBJET
identifiee par son record de creation, ces 2 300 « transitions » tombent a **33 decroissances**
sur 2 896 vies.

### 1.3 Le partage REEL / FANTOME — la mesure qui tranche

Une lecture est REELLE quand sa cle (slot, generation) retombe sur une vie d'objet CONFIRMEE
par l'oracle de position du balayage des poses ; FANTOME sinon. C'est le meme oracle que la
production, et il n'a rien de circulaire : il vient des paquets delta, qui ne savent rien du
record de creation.

| champ | lectures REELLES | lectures FANTOMES | part reelle |
|---|---|---|---|
| i27 `charges-remaining` | **368** | **12 954** | **2,8 %** |
| i26 `energy-delay-ticks-left` | 2 419 | 7 304 | 24,9 % |

La forme des deux populations acheve la demonstration. Sur `000d5950` la population FANTOME
d'i27 compte **118 valeurs distinctes sur 256 possibles, max 255** — une occupation quasi
uniforme de la plage R(8), la signature d'un curseur mal aligne, pas d'un compteur. Sur les
12 films, la population fantome ne descend jamais sous 60 classes distinctes.

### 1.4 Ce que les valeurs REELLES disent — et ce n'est pas « des charges »

Sur les 14 films instrumentes, **241 vies d'objet portent au moins une lecture d'i27** :

| lectures d'i27 dans la vie | vies |
|---|---|
| 1 | **198** |
| 2 | 23 |
| 4 | 3 |
| 5 | 1 |
| 20 / 25 / 67 | 1 chacun |

- **198 vies sur 241 ne portent QU'UNE SEULE lecture** : aucune transition n'y est possible,
  quelle que soit la semantique du champ.
- **8 vies sur 241 seulement** ont un maximum `<= 3`, la plage d'un compteur de charges
  d'equipement. Les valeurs observees se massent au contraire dans le haut de l'octet
  (215, 223, 231, 243, 244, 245, 251, 255).
- Les 33 decroissances ont des amplitudes ARBITRAIRES — `255 -> 4`, `251 -> 90`, `223 -> 22`,
  `244 -> 14` — jamais le pas de −1 d'un compteur d'usages.

i26 REEL est d'une autre forme : 5 a 43 valeurs distinctes par film, **ecrasees sur 0 et 1**
(sur `000d5950` : 261 lectures a 0 et 57 a 1 sur 328), avec une poignee de valeurs hautes
isolees. Ce n'est pas le profil d'un compte a rebours qui s'ecoule.

### 1.5 Temoin (c) — le plancher de bruit de l'ancrage

Le balayage des poses publie le rapport reel/fantome de l'ANCRAGE lui-meme. Sur `000d5950` :
6 651 ancres reconnues, 401 records acceptes, **295 confirmes par l'oracle de position
(73,6 % des acceptes)**, 295 poses. Sur `06dfe6d9` (BTB, 26 participants) : 57 228 ancres pour
892 poses. L'en-tete NEW de ti=37 n'est pas selectif, c'est ecrit dans
`equipment_placements.go` et la mesure le confirme.

Puretes etablies AILLEURS et citees ici comme reference (non re-mesurees) : ancrage ti=37 en
delta a **97,2 % de 628 368 echantillons i0** dans l'emprise des bipedes du meme film, 12 films
sur 12 (registre ECS) ; temoin d'ancrage a 1 slot **ti=4 : 98,7-99,8 %** (lot C phase 1a).

**Limite assumee et ecrite** : `ScanFilmEquipmentState` ne prend pas de bande en parametre, un
tirage sur bande FANTOME du balayage d'etat exigerait de modifier du code de production — hors
phase 0 (lecture seule). Le partage §1.3 en tient lieu, et il est plus severe.

## 2. D.0.2 — l'oracle du grappin `[x]` (GATE 0 : NON ATTEINT)

**L'identite n'est pas devinee.** Les objets ti=37 de famille `grapple` sont ceux dont le
GlobalID du tag `eqip` du record de CREATION est declare `grapple` au manifeste du titre — LA
MEME table que la production (`replay_labels.toml`, `0x273fe0eb ability_grapple_hook` et
`0x8c77ffe7` meme modele `hlmt`).

| | |
|---|---|
| evenements de grappin (i59 tag==3) | **501** sur 12 films (285 usages, paires tir/accroche a <= 0,5 s) |
| objets ti=37 `grapple` identifies | **198 vies** — `0x273fe0eb` sur 11 films, `0x8c77ffe7` sur `000d5950` |
| films sans aucun objet `grapple` | 1 (`0a247154`, KOTH : 3 lectures tag==3 dont 3 corps casses, 0 evenement) |

| canal candidat | signaux `grapple` | APPARIES | temoin (a) +7 s | temoin (b) autres familles | verdict (seuil 80 % / 10 %) |
|---|---|---|---|---|---|
| **i27 decroissance (prescrit)** | 2 | **0 / 501 (0,0 %)** | 0 / 501 (0,0 %) | 2 / 501 (0,4 %) | **NON TENU** |
| **i26 hausse (repli prescrit)** | 12 | **1 / 501 (0,2 %)** | 0 / 501 (0,0 %) | 10 / 501 (2,0 %) | **NON TENU** |
| naissance de l'objet (controle hors plan) | 198 | 35 / 501 (7,0 %) | **31 / 501 (6,2 %)** | 88 / 501 (17,6 %) | NON TENU, et AU NIVEAU DU TEMOIN |

**Lecture.** Le grappin a bien des objets ti=37 — la question ouverte par le plan (« si le
grappin n'a AUCUN objet ti=37, dis-le ») recoit donc un OUI : 198 vies sur 12 films, une
identite propre, un seul identifiant par film. Ce sont leurs CHARGES qui ne bougent pas : sur
198 vies d'objet grappin, **une seule** porte une decroissance d'i27, et elle ne tombe pas dans
la fenetre d'un evenement. Le repli prescrit (i26 qui repart) ne fait pas mieux : 12 hausses
pour 501 evenements.

Le canal de controle (la NAISSANCE de l'objet grappin) est le seul a apparier quelque chose —
et son temoin decale l'egale (7,0 % contre 6,2 %). C'est la confirmation, sur une population
restreinte a l'identite `grapple`, du negatif deja ecrit au registre pour les naissances ti=37
toutes familles confondues (densite 4,7-5,3/s, temoins au niveau du reel). Sur `06dfe6d9` le
temoin (b) monte a 36,8 % : l'appariement des naissances ne distingue meme pas les familles.

## 3. D.0.3 — le pont objet -> joueur `[!]` non etabli (le canal porteur n'existe pas)

Le pont prescrit repose sur les appariements de D.0.2. **Le canal prescrit n'appariant rien
(0/501), le pont n'est pas calculable par la voie du plan.** Mesure publiee pour memoire sur le
canal de controle (naissances), le seul a produire des paires :

| | 12 films |
|---|---|
| vies portant au moins un appariement | 19 |
| vies attribuees (>= 2 evenements d'un MEME slot) | 13 |
| vies ambigues (deux slots pretendants) | 1 |
| **coherence** | **12 / 13 (92,3 %)** — seuil 90 % franchi |

**Ce chiffre ne vaut rien et il faut le dire** : il porte sur 13 attributions issues d'un canal
dont le taux d'appariement (7,0 %) est indiscernable de son temoin (6,2 %). Une coherence
elevee sur une population fabriquee par le hasard reste du hasard coherent.

**Alternative `item-ignore-player` (i19, ti=37) — mesure ecrite, valeur non lisible en
phase 0.** Le composant est annonce au masque (59 fois sur `000d5950`) et sa grammaire est
`R(1)[+R(5)]` (`components_world.go:124`) : **5 bits**. C'est EXACTEMENT la forme d'i23
`equipment-creator`, deja refutee deux fois — 0 valeur sur 1 328 dans la bande des slots de
bipede (qui se comptent en milliers, 13 bits), et 28 valeurs distinctes sur un film a 8 joueurs
donc pas davantage un index compact de joueur. Sa VALEUR n'est pas lisible ici : le deser la
jette, et lui poser un hook est de la PLOMBERIE — interdite en phase 0 (lecture seule), et de
toute facon le pari est mauvais par construction de largeur.

## 4. D.0.4 — generalisation `[!]` (aucune famille ne decremente)

**2 896 vies d'objet identifiees sur les 12 films, 15 familles.** Decroissances d'i27 :

| famille | vies | vies avec >= 1 decroissance | decroissances |
|---|---|---|---|
| `grenade_frag` | 1 668 | 15 | **29** |
| `grenade_plasma` | 259 | 0 | 0 |
| `grapple` | 198 | 1 | 1 |
| `grenade_spike` | 135 | 1 | 1 |
| **`wall`** | **136** | **0** | **0** |
| **`repulsor`** | **122** | **0** | **0** |
| **`thruster`** | **74** | **0** | **0** |
| **`sensor`** | **69** | **0** | **0** |
| `grenade_dynamo` | 99 | 0 | 0 |
| `other` | 81 | 1 | 1 |
| `repair_field` | 41 | 1 | 1 |
| **`translocator_beacon`** | **4** | **0** | **0** |
| `threat_seeker` | 4 | 0 | 0 |
| `powerup_overshield` / `powerup_camo` | 3 / 3 | 0 | 0 |
| **TOTAL** | **2 896** | **19** | **33** |

**Les CINQ familles nommees par le plan — repulseur, propulseur, mur, capteur, translocateur —
totalisent ZERO decroissance sur 405 vies d'objet.** Il n'y a donc aucun « nombre d'usages par
joueur et par match » a en tirer : le compteur ne compte pas.

**Controle croise pose <-> decroissance de la meme famille** (une pose de mur consomme-t-elle
une charge ?) : **5 / 1 468 (0,34 %)**, temoin +7 s 2 / 1 468 (0,14 %). Seuil du plan : 80 %.
Le meilleur film-famille atteint 1,6 % (2/126, `grenade_frag`).

**Relation i26 <-> decroissance** (le delai de recharge debute-t-il au decrement ?) :
**0 / 33 (0,0 %)** des decroissances de charge sont accompagnees d'une hausse d'i26 sur la MEME
vie dans ± 0,5 s. Les deux canaux ne se parlent pas.

**Poses par poseur** (mesure faite, en METRES, par `equipmentOwner` — LA fonction de
production) : elle FONCTIONNE, elle ne manquait pas. Sur `000d5950`, 90 slots poseurs (des
VIES, pas des joueurs), mediane 3 poses par slot, 6 poses sans poseur a portee. C'est bien
l'attribution qui existe deja pour les poses ; ce que ce lot devait ajouter — l'USAGE date d'un
equipement porte — n'a pas de source.

## 5. Cout machine (D17)

Regle tenue : **un film par processus** (`go test -c` puis un lancement par film), **avant-plan**,
**plafond memoire surveille** (`Start-Process -PassThru`, `PeakWorkingSet64`, kill au-dela de
3 Go), **cout mesure sur 3 films avant les 12**, **jamais pendant un `go build`**.

| | 3 films de calibrage | 14 films au total |
|---|---|---|
| duree | 40 / 52 / 106 s | 24 s a **427 s** (`06dfe6d9`, BTB 26 joueurs) ; mediane ~55 s |
| pic memoire | 68 / 22 / 24 Mo | **21,2 a 116,2 Mo** |
| plafond 3 Go | jamais approche (3,9 % du plafond au pire) | 0 kill, 0 echec |

L'instrument fait quatre passes disque par film (etat ti=37, poses ti=37 avec calibration MPP,
evenements i59, nuage des bipedes). Journal des couts : `LOTD_gates.log`.

## 6. Gates de cloture

| gate | commande | verdict |
|---|---|---|
| vet | `go vet ./internal/analysis/... ./internal/replaybuild/... ./internal/games/halo_infinite/film/... ./contracttest/...` | `EXIT_VET=0` |
| tests | `CGO_ENABLED=0 go test -count=1 ./internal/analysis/filmdec/ ./internal/analysis/replay/` | `EXIT_TEST=0` (filmdec 42,0 s, replay 41,3 s) |
| lint | `golangci-lint run --new-from-merge-base=origin/main ./...` | `EXIT_LINT_CGO1=0`, **0 issue** |

**Piege rencontre et note** : `golangci-lint` sous `CGO_ENABLED=0` echoue en `EXIT=7` avec une
erreur de typecheck (`could not import levelup/go-api/internal/ooz : build constraints exclude
all Go files`) — 0 issue de lint pour autant. Le lint doit tourner **CGO active** ; les tests
filmdec/replay, eux, restent a `CGO_ENABLED=0`.

## 7. Pieces

- Sorties par film : `lotD/logs/<short8>.log` (14 films).
- TSV : `lotD/<short8>_vies.tsv` (une ligne par vie d'objet identifiee : cle, instance,
  GlobalID, famille, bornes, lectures i27/i26, min/max d'i27, decroissances) et
  `lotD/<short8>_decroissances.tsv` (une ligne par decroissance, avec `de -> vers`).
- Couts et gates : `LOTD_gates.log`.

Rejouer un film :

```
CGO_ENABLED=0 \
EQ_FILM=C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/000d5950 \
EQ_BOUNDS=C:/Users/Guillaume/Projects/LevelUp/data/titles/halo_infinite/reference/map_quant_bounds.json \
EQ_OUT=<repo>/.ai/V7.5/replay2d/registre_film/lotD \
go test ./internal/analysis/replay/ -run '^TestEquipmentUsesPhase0$' -timeout 60m -v
```

## 8. Verdict et suites

**Gate 0 NON ATTEINT** : D.0.2 rend 0 / 501 la ou le plan exige >= 80 %. Le temoin (a) rend
0 / 501 — le negatif est propre, l'instrument n'est pas muet.

**Consequence prescrite par le plan** : `equipment-charges-remaining` (i27) **sort de la reserve
comme NEGATIF**, lot D clos `[!]`, phases D.1 (publication `equipmentUses`) et D.2 (web) NON
LANCEES. La mise a jour du registre `:157` et du journal §7 revient au superviseur.

**Ce que le negatif dit precisement**, et qu'il faut ecrire pour ne pas le re-tenter : le canal
i27 n'est pas « un compteur de charges qu'on n'a pas su lire ». Sur la population dont
l'identite est prouvee, **198 vies sur 241 ne portent qu'UNE lecture**, 8 sur 241 restent sous
la valeur 3, les amplitudes de variation sont arbitraires, et 97,2 % des annonces au masque
sont des fantomes d'ancrage. Les 16 125 annonces qui l'avaient designe comme « le gisement »
etaient, pour l'essentiel, du bruit compte comme du signal.

### Decouvertes (hors perimetre, NON traitees)

1. **Le recensement au masque de ti=37 surestime massivement le signal.** Le chiffre qui a mis
   i27 en reserve (16 125 annonces, « 3,1x l'energie ») melange reel et fantome ; rapporte aux
   vies confirmees par l'oracle de position, il tombe a 2,8 %. Tout classement de composants
   ti=37 fonde sur `MaskCensus` merite d'etre relu a cette aune — i25 `equipment-being-hacked`
   et i28 `tracked-object-handles-stack`, aujourd'hui cites comme prometteurs, n'ont jamais ete
   passes a ce filtre. Condition de reprise : le partage reel/fantome est deja outille
   (`eqUsesBuildLives`), il suffit de l'appliquer aux autres composants.
2. **`equipment-has-infinite-uses` (ti=37 i30, R(1), porte, jamais lu)** est le seul bit de
   l'archetype qui separerait le mobilier de carte de l'equipement pose par un joueur — la
   question laissee ouverte par le verdict 0.6. Il n'a pas de hook, donc pas de mesure possible
   en phase 0. Condition de reprise : un hook (plomberie), et le meme partage reel/fantome.
3. **`0a247154` (KOTH) ne rend AUCUN evenement de grappin** : 3 lectures tag==3, 3 corps casses
   (drapeaux != 000). Le registre chiffrait cette perte a ~4 % du corpus famille B ; sur ce film
   elle est de 100 %. Rien a corriger ici, mais le film est un candidat naturel pour le report
   « corps i59 tag==3 a drapeaux != 000 » du registre, qui attend un corpus ou ces variantes
   sont denses.
4. **`000d5950` est le seul film du corpus a porter le GlobalID de grappin `0x8c77ffe7`**
   (variante `sofa_modele`), les 11 autres portent `0x273fe0eb`. Les deux sont deja nommes au
   manifeste : rien a faire, mais la dualite est reelle et se retrouvera ailleurs.
5. **`golangci-lint` est incompatible avec `CGO_ENABLED=0`** dans ce depot (`internal/ooz`).
   Merite une ligne dans la doc des gates : deux reglages differents pour deux gates voisins.
