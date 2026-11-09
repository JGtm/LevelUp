# NOTE 3.3 (volet RECHERCHE) — LES IDENTIFIANTS DE GRENADE DES BUILDS ANCIENS : LA GRAMMAIRE A-T-ELLE BOUGE, OU LES IDENTIFIANTS ?

> Ouverte le 2026-09-16, branche `feat/decfilm-33r`, base `e7b9bd48e`. Instrument sous
> `apps/go-api/tools/film_re/grenadeids/` et `tools/film_re/cmd/grenadeids/`, tag `research`.
> Etat au 2026-09-17 : CLOS. Instrument ecrit, compile, valide sur temoin positif ; **mesure 0
> JOUEE** (sept mini-bobines) ET **mesures 1 a 6 JOUEES** sur les six films du cache a la voie
> libre du pilote. VERDICT : la GRAMMAIRE a bouge d UN BIT, les identifiants n ont pas bouge (§8).
>
> Source de la question : V17 du plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md` §1.4 (hypothese de
> l utilisateur du 2026-09-18 etiquete, soit le 2026-09-16 reel par V18) et
> `.ai/PREPARATION_M3_3_2_A_3_4_2026-09-17.md` §2.

---

## 1. CE QUE L HYPOTHESE DE L UTILISATEUR DIT, ET CE QU ELLE INTERDIT

> « Le nombre et l ordre des types de grenade n ont pas change depuis la sortie du jeu ; des
> identifiants ou des rangs qui changent entre builds sont improbables ; c est plus
> vraisemblablement la GRAMMAIRE qui a bouge — la position lue apres le marqueur, ou le
> marqueur lui-meme. »

Ce que cela interdit : ouvrir 3.3 par une liste blanche « ancienne ». La note M3 §2.4 proposait
une entree de profil `Grenade.TypeIDsByRank` valant les quatre premieres valeurs de la famille
de huit relevee sur les versions 33, 37, 39 et 40 — avec la mention « ordre des rangs NON
etabli ». Cette entree serait un ORDRE DEVINE (M3-Q5 = B, un lancer identifie ou rien), et elle
serait ecrite AVANT d avoir teste l hypothese la moins couteuse.

L ordre de preuve impose par V17 est donc : **(1) chercher les quatre identifiants ACTUELS
ailleurs dans les films anciens** ; (2) si et seulement si (1) echoue, apparier la famille de
huit aux decrements d i22 ; (3) conclure seulement alors.

---

## 2. TROIS FAITS DE CODE, VERIFIES SUR PIECES A L ENTREE DU LOT

Les trois conditionnent la forme de l instrument. Ils sont lus dans l arbre a la base
`e7b9bd48e`, pas dans les notes.

### 2.1 Le « marqueur » est une fonction de l index d archetype, et elle se verifie

`grammar/projectiles.go` (en-tete, point 2) et `.ai/ADDENDUM_ETAT_DE_L_ART_2026-07-26.md` §3 :
le marqueur est `[5 bits bas de typeIndex][19 bits d amorce de l etat par defaut]`. La
derivation est donc

```
marqueur(ti) = ((ti & 31) << 19) | 0x40C00
```

et elle se controle : `marqueur(41) = (9 << 19) | 0x40C00 = 0x480000 | 0x40C00 = 0x4C0C00`,
exactement la constante `grenadeMarker` de `grammar/grenade_events.go`. L instrument REFUSE de
demarrer si cette egalite tombe (`VerifierMarqueurDeProduction`) — sans elle, toute la lecture
« le marqueur est une donnee de build » serait fausse.

**Consequence neuve, et elle n est ecrite nulle part** : le marqueur ne porte que CINQ bits de
l index. Sur un registre de 49 ou 50 blocs, `ti=41` et `ti=9` produisent donc LE MEME marqueur —
et `ti=9` est `managed-player` (`grammar/player_teams.go:61`). Le balayage de production ne
reconnait pas « une naissance de projectile » : il reconnait « une naissance d une entite dont
l index vaut 9 modulo 32 ». C est une ambiguite structurelle de l heureux accident, et
l instrument la publie film par film (champ `ambiguites`). Mesure 0 : `ambiguites=[9]` sur les
SEPT builds, sans exception.

**ET ELLE SE LEVE PAR UNE LECTURE, PAS PAR UNE HEURISTIQUE** (ajout du 2026-09-16 sur demande du
pilote). Le typeIndex d un record fait SIX bits — `grammar/traverse.go:94`
(`t.TypeIndex = uint32(br.ReadBits(6))`) et `grammar/keyframe_fullstate_loop.go:88`
(`kfReadBits(pay, recBit+keyframeRecordTIBit, 6)`). Le marqueur commence au DEUXIEME de ces six
bits : le bit de poids fort (valeur 32) est donc a `marqueur - 1`.

```
41 = 0b101001  -> bit a marqueur-1 : 1
 9 = 0b001001  -> bit a marqueur-1 : 0
```

L instrument lit ce bit (`typeindex.go`), compte les marqueurs PAR ARCHETYPE REEL et publie
l histogramme de la passe C **restreint a chaque archetype** : ce qui suit une naissance de
`managed-player` ne pollue donc plus la famille des grenades. La seule position ou le bit manque
(`marqueur = 0`, debut de payload) est comptee a part (`indetermines=`) et n entre dans AUCUN
histogramme par archetype — jamais devinee a zero en silence. La passe D accepte le meme filtre
(`-ti 41`).

### 2.2 L archetype projectile se resout PAR LE NOM, sans cabler 41

`grammar/projectiles.go` nomme les quatre composants qui identifient l archetype :
`projectile-at-rest-state`, `projectile-tether-state`, `projectile-command_tick`,
`projectile-deceleration-disabled-state`, et sa propre constante porte la consigne (« A VERIFIER
par le nom des composants du registre plutot qu a cabler : c est un index de build, pas une
constante du format »). Le registre de chaque film est lisible par
`grammar.ParseRegistryChunk(grammar.FilmRegistryChunk(film))`, et `Registry.Archetypes` porte
les noms. L instrument retient le bloc qui porte le plus de ces quatre noms et en derive le
marqueur du film. C est exactement la regle de l item 3.2.2 (« par NOM, jamais par rang »),
appliquee un lot plus tot.

### 2.3 Les SEPT MINI-BOBINES PAR BUILD NE PEUVENT PAS SERVIR CE LOT

La note M3 §0.3 les presente comme « la ressource la plus sous-utilisee du chantier : les sept
builds connus sont dans l arbre git, avec leur registre, sans toucher au cache de films ni au
verrou solo ». C est vrai pour un registre, et FAUX pour ce lot-ci. Leur `PROVENANCE.txt`
(lecture du 2026-09-16, `minifilm_111fa685`) dit ce que chaque chunk porte :

```
chunk_00.bin  LE CHUNK DE REGISTRE du film (type 1)
chunk_01.bin  14 paquet(s) d IMAGE-CLE (type 2) REELS
chunk_02.bin  LE PIED du film (chunk 30, type 3)
```

**Aucun paquet de type 0.** Or le marqueur de lancer vit dans les paquets DELTA, et eux seuls
(`grammar.ScanGrenadeThrows` filtre sur `PacketTypeDelta`). Les mini-bobines sont donc muettes
sur les passes A, B, C et D.

Elles restent utiles a UNE mesure, et elle est gratuite : la **mesure 0** ci-dessous.

---

## 3. L INSTRUMENT

`apps/go-api/tools/film_re/grenadeids/` (bibliotheque) et `tools/film_re/cmd/grenadeids/`
(binaire), tous fichiers sous `//go:build research` : `go build ./...`, `go vet ./...` et la CI
ne les voient pas. Il n ecrit AUCUN fichier, ne prend AUCUN verrou (voir §3.4) et lit les films
en place, un a la fois.

### 3.1 Les quatre passes

| Passe | Ce qu elle mesure | Ce qu elle tranche |
|---|---|---|
| A | pour CHAQUE marqueur, la valeur de 32 bits lue a `marqueur + 24 + d` pour tout `d` de `[-F, +F]` (F = 64 par defaut), comptee quand elle tombe dans la liste blanche actuelle | un decalage `d != 0` STABLE = la POSITION a bouge |
| B | un balayage ABSOLU : toute occurrence d un des quatre identifiants actuels a n importe quelle position de bit du flux delta, avec sa distance au marqueur le plus proche et le compte de celles qu aucun marqueur ne borde | les identifiants actuels vivent-ils dans un AUTRE champ, ou pas du tout ? |
| C | l histogramme SANS liste blanche de ce qui suit le marqueur, avec le champ d index joueur de 5 bits a +103, pour les DEUX marqueurs (production et registre) **et separement PAR ARCHETYPE REEL** — le sixieme bit d index, lu a `marqueur - 1`, separe `ti=41` de `ti=9` (§2.1) | la famille de huit est-elle la meme ? combien de ses valeurs viennent en realite d une naissance de `managed-player` ? |
| D | l appariement de chaque candidat de la passe C aux decrements UNITAIRES du compteur i22 du meme instant, rang par rang, avec un TEMOIN DE HASARD (memes appariements, instants decales) | question (2) |

Les passes A, B et C tiennent en UNE lecture du flux : a chaque position de bit, une seule
lecture de 32 bits sert de marqueur candidat (ses 24 bits de poids fort) ET d identifiant
candidat (ses 32 bits). Le cout est celui du balayage de production, a une lecture par position
pres.

### 3.2 Ce que l instrument publie de refutable AVANT toute interpretation

- **Le hasard attendu de la passe B**, calcule et imprime avec la mesure : quatre valeurs de
  32 bits cherchees a chaque position d un flux de `N` bits tombent par hasard `N x 4 / 2^32`
  fois. Un flux delta de l ordre de 40 Mio donne ~0,3 occurrence attendue : **tout compte
  superieur a 2 ou 3 est un signal**, tout compte de 0 ou 1 n en est pas un.
- **Le temoin de hasard de la passe D** (`-temoin-ms`) : le meme appariement joue sur des
  instants deplaces d une constante mesure ce que la seule densite des decrements produit sans
  aucun lien de cause. Un appariement qui ne bat pas son temoin ne prouve rien.
- **Le compte de marqueurs par source**, qui separe (a) « le marqueur n est pas trouve » de
  (b) « il est trouve mais l identifiant n est pas reconnu » — la correction importante de la
  note M3 §2.1 (`available: 0` ne dit PAS qu il n y a pas de marqueur).
- **Le controle de fonctionnalite du pont** en passe D : les couples (index joueur du lancer,
  slot du porteur d i22) que l appariement produit forment-ils une fonction ? L instrument
  OBSERVE ce pont, il ne le suppose jamais — c est un controle de l appariement, pas son
  fondement.

### 3.3 Ce que l instrument ne fait pas

Il n asserte rien, ne corrige rien, n ecrit rien, et ne prononce aucun verdict : les verdicts
sont dans cette note. Il ne touche ni `internal/`, ni `cmd/`, ni `config/`, ni `data/`.

### 3.4 Le verrou de decodage n est PAS pris, et c est un choix ecrit

`filmproc.AcquireSolo` ecrit un fichier `film_decode.lock` sous la racine du cache film. Le
cadre de ce lot interdit toute ecriture sous `data/`. La serialisation des decodages est donc
celle de l operateur — la « voie libre » du pilote. La **sentinelle memoire**, elle, est armee
(`filmproc.Arm`, plafond 4 Gio par defaut) : elle n ecrit rien et ne fait qu abaisser un plafond
dans ce processus.

---

## 4. LE PLAN DE MESURE (joue le 2026-09-17 ; resultats en §6 bis)

### Mesure 0 — JOUEE LE 2026-09-16 (autorisation du pilote) : LE MARQUEUR N A PAS BOUGE

Commande, 0,52 s, aucun film du cache touche :

```
go run -tags=research ./tools/film_re/cmd/grenadeids -fenetre 0 -voisinage 0 \
  -racine internal/games/halo_infinite/film/replay/testdata \
  -films minifilm_a521164d,minifilm_60ae07c4,minifilm_11de8353,minifilm_111fa685,minifilm_e5adf7b2,minifilm_bcb6d393,minifilm_fb1a1a72
```

Resultat colle, une ligne par build :

| mini-bobine | version | build | blocs de registre | `ti` projectile (noms trouves) | marqueur derive | ambiguites |
|---|---|---|---|---|---|---|
| `a521164d` | 33 | `HI_1_4_1` | 49 | **41** (4/4) | `0x4C0C00` | `[9]` |
| `60ae07c4` | 37 | `HI_1_8_0` | 49 | **41** (4/4) | `0x4C0C00` | `[9]` |
| `11de8353` | 38 | `HI_1_9_0` | 49 | **41** (4/4) | `0x4C0C00` | `[9]` |
| `111fa685` | 39 | `HI_1_10_0` | 49 | **41** (4/4) | `0x4C0C00` | `[9]` |
| `e5adf7b2` | 40 | `HI_1_11_0` | 49 | **41** (4/4) | `0x4C0C00` | `[9]` |
| `bcb6d393` | 40 | `HI_1_12_0` | 50 | **41** (4/4) | `0x4C0C00` | `[9]` |
| `fb1a1a72` | 41 | `HI_1_13_0` | 50 | **41** (4/4) | `0x4C0C00` | `[9]` |

**VERDICT DE LA MESURE 0 : le critere (1b) est ECARTE.** L archetype projectile est au rang 41
sur les SEPT builds, identifie par ses QUATRE noms de composant a chaque fois (aucune resolution
douteuse), et le marqueur derive vaut `0x4C0C00` partout. **Le marqueur n a pas bouge**, donc la
question (1) se reduit a la POSITION (passe A) et au CHAMP (passe B). Une entree de profil
`Grenade.Marqueur` serait aujourd hui une constante deguisee : elle ne s ecrira que si un build
futur deplace l archetype.

**Deux observations de bord, non traitees.**

1. Le cardinal du registre passe de **49 a 50 blocs** exactement entre `HI_1_11_0` et
   `HI_1_12_0` — c est-a-dire a la MEME frontiere que les lancers publies (0 avant, 55 a 395
   apres). C est une correlation, pas une cause : `ti` projectile vaut 41 des deux cotes, donc le
   bloc gagne est APRES le rang 41. A verser au lot 3.2 (le registre par build), pas ici.
2. Quatre des sept bobines font sonner `warnUnknownRegistry` (empreintes `4635077086892953806`
   pour `HI_1_4_1`, `3731204960007589061` pour `HI_1_8_0`, `11196959896536408664` pour
   `HI_1_9_0`, `9834140534605324359` pour `HI_1_10_0`, contre la connue
   `3948122217672317832`). C est exactement la matiere de l item 3.2.1 (« table des empreintes de
   registre connues PAR BUILD »), et ces quatre valeurs sont mesurees sur le domaine de hachage
   COURANT — ce que les huit empreintes du rapport H ne sont pas (F2 de la note M3).

### Mesure 0 — comment elle se rejoue

Les sept mini-bobines portent leur `chunk_00` COMMIS, donc leur registre. Passer l instrument
dessus avec `-fenetre 0` ne lit que l entete : build, nombre d archetypes, **ti projectile resolu
par nom**, marqueur derive, ambiguites.

```
cd apps/go-api
go run -tags=research ./tools/film_re/cmd/grenadeids -fenetre 0 \
  -racine internal/games/halo_infinite/film/replay/testdata \
  -films minifilm_a521164d,minifilm_60ae07c4,minifilm_11de8353,minifilm_111fa685,minifilm_e5adf7b2,minifilm_bcb6d393,minifilm_fb1a1a72
```

**Ce qu elle tranche a elle seule** : si `ti projectile` vaut 41 sur les sept builds, le MARQUEUR
n a pas bouge et la question (1) se reduit au DECALAGE. S il differe sur un build ancien, la
reponse est acquise sans ouvrir un seul film du cache — et la ligne de profil a ecrire est
`Grenade.Marqueur`, derive du `ti` lu dans le film, pas une liste blanche.

Arbitrage du pilote du 2026-09-16 : la contrainte « un decodage a la fois » est une contrainte de
RAM et de base partagee, elle ne s applique pas a ces bobines (1 Mio chacune, dans l arbre git,
hors parc). La mesure a donc ete jouee immediatement — resultat ci-dessus.

### Mesures 1 a 6 — les films du cache, un a la fois

| Ordre | Film | Version | Build | Role |
|---|---|---|---|---|
| 1 | `bcb6d393` | 40 | `HI_1_12_0` | **TEMOIN POSITIF** : 55 lancers publies. Si la passe A ne montre pas les quatre identifiants au decalage 0, c est l INSTRUMENT qui est faux, et rien d autre ne doit etre lu |
| 2 | `e5adf7b2` | 40 | `HI_1_11_0` | la MEME version de format que le temoin, 0 lancer : c est la paire qui prouve que la coupure est au BUILD |
| 3 | `111fa685` | 39 | `HI_1_10_0` | 476 images-cles d inventaire, 0 lancer — le film que la note M3 §2.5 designe comme la meilleure matiere d appariement |
| 4 | `60ae07c4` | 37 | `HI_1_8_0` | un build de plus, une version de plus |
| 5 | `a349fea8` | 33 | sans section | la grammaire la plus ancienne du corpus gate |
| 6 | `084a804d` | 39 | `HI_1_10_0` | SECOND temoin du meme build que 3 : un decalage qui ne se reproduit pas d un film a l autre du meme build n est pas une grammaire |

```
go run -tags=research ./tools/film_re/cmd/grenadeids \
  -racine <parc>/data/cache/film_chunks -films bcb6d393
```

La passe D ne se joue QUE si la question (1) est negative :

```
go run -tags=research ./tools/film_re/cmd/grenadeids \
  -racine <parc>/data/cache/film_chunks -films 111fa685 -apparier -temoin-ms 30000
```

---

## 5. LES CRITERES DE VERDICT, ECRITS AVANT LA MESURE (c est (1a) qui sort — §8)

Ils sont poses ici pour qu aucun ne soit choisi apres coup.

**(1a) LA POSITION A BOUGE.** Les quatre identifiants actuels apparaissent, sur les films
anciens, a un decalage `d != 0` qui est LE MEME sur les deux films d un meme build (3 et 6), avec
un compte du meme ordre que le nombre de marqueurs. Verdict : **grammaire**. 3.3.1 code une
entree de profil `Grenade.DecalageIdentifiant` par build, PAS une liste blanche, et
`GrenadeTypeIDsByRank` reste une constante du titre. L ordre des rangs n a jamais bouge : le
compteur `grenadeThrowsUnranked` de M3-Q5 reste a zero sur ces builds.

**(1b) LE MARQUEUR A BOUGE — ECARTE LE 2026-09-16 PAR LA MESURE 0** (`ti` projectile = 41 sur les sept builds, marqueur derive `0x4C0C00` partout). Le critere reste ecrit pour qu on sache ce qui l a ferme, et il se rouvrira le jour ou un build deplacera l archetype. Enonce d origine : la mesure 0 rend un `ti projectile` different de 41 sur un build
ancien, ET la passe A du marqueur DERIVE DU REGISTRE rend les quatre identifiants au decalage 0.
Verdict : **grammaire**, encore. 3.3.1 code `Grenade.Marqueur = marqueur(ti projectile lu dans le
film)` — ce qui rend le lot solidaire de 3.2 (le registre par build), comme la note M3 §2.2 le
predit.

**(1c) NI L UN NI L AUTRE.** Les quatre identifiants n apparaissent nulle part au-dela du hasard
attendu publie par la passe B, sur aucun des cinq films anciens. Verdict : la question (1) est
**negative** et elle est ALORS une mesure, pas une absence de mesure — c est ce qui autorise la
question (2). Passer a la passe D.

**(2) LES HUIT SONT LES IDENTIFIANTS ANCIENS.** Un candidat `X` de la famille de huit s apparie
aux decrements unitaires d UN SEUL rang `r` a un taux qui bat nettement son temoin de hasard, les
couples (index, slot) sont fonctionnels, et quatre candidats seulement apparaissent en
multijoueur. Verdict : **identifiants differents**, la regle `type = rang / 2` de `R-GRENADE`
tient, et l ORDRE est etabli PAR LES APPARIEMENTS (pas par la table `grenade_types`, qui ne donne
que l ordre des rangs du titre). L entree de profil de la note M3 §2.4 devient ecrivable, avec
son temoin par rang.

**(2 bis) L APPARIEMENT ECHOUE.** Si i22 lui-meme ne se lit pas sur ces builds (l instrument
publie `i22_lues` / `i22_non_lues` / `implausibles` pour le dire), l absence d appariement ne
prouve rien sur les identifiants : elle prouve que le canal d inventaire ne marche pas sur ce
build, ce qui est une DECOUVERTE a porter au §4 du plan et une dependance de 3.3 sur 3.2, pas un
verdict sur les grenades.

---

## 6. CE QUE GHIDRA A DEJA DIT, ET QU IL EST INUTILE DE REDEMANDER

Repris de la note M3 §2.3, mesure du 2026-09-17 sur `HaloInfinite.exe` base `0x140000000`, image
`269225.26.04.08.1618-1.hi_1_13_0`. Ces points sont FERMES ; les rejouer couterait une session
pour rien (regle : ne pas re-mesurer par statistique ce qu un ecrivain dit en clair, et ne pas
re-interroger un ecrivain qui a deja repondu).

- La table `grenade_types` existe (enregistreur `FUN_140104c20`, table `1443e2ab0`, 6 entrees) et
  confirme `rang = typeId - 1` : frag, plasma, lightning, spike, sapper, stasis. Elle donne
  l ORDRE DES RANGS DU TITRE, jamais un identifiant.
- PIEGE : une SECONDE enumeration, d ordre DIFFERENT (`FUN_1403523d0`, base `144d71ee0` :
  Frag=0, Plasma=1, Brute=2, Shock=3) — telemetrie / UI. Ne JAMAIS s en servir pour reordonner.
- Le marqueur `0x4C0C00` n est pas dans le binaire (3 motifs, 13 607 750 instructions,
  `match_count: 0`) — coherent avec ce qu il est : un champ de bits, jamais compare litteralement.
- Les identifiants de tag ne sont dans le binaire A AUCUN BUILD (24 motifs, 24 fois zero sur
  95 404 712 octets) : `FUN_140748d64` est un Murmur3_x86_32 sur le token normalise, le binaire
  hache des NOMS d enumeration et ne stocke aucun identifiant global de tag.
- Les archives `.module` des builds anciens ne sont plus distribuees : la voie « refaire la liste
  `gggl` du build ancien » est MORTE.

Ce qui reste a Ghidra pour ce lot : rien tant que la mesure n a pas parle. Si (1a) sort, l ecrivain
du record de creation d entite (`vtable[0x60]` de l archetype projectile, `0x1408EFB58` pour ti41
d apres `grammar/default_state_arch.go`) dira la largeur exacte du champ qui precede
l identifiant, et le decalage mesure se lira chez lui au lieu d etre une constante de profil.
C est la forme preferable — elle attend la mesure, elle ne la remplace pas.

---


---

## 6 bis. LES MESURES 1 A 6 — JOUEES LE 2026-09-17 (« voie libre » du pilote)

Six films, UN A LA FOIS, dans l ordre impose. Lecture seule, rien ecrit sous `data/`, verrou non
pris (la serialisation est celle de l operateur). Duree : 0,26 s a 1,2 s par film.

### 6 bis.1 Le temoin positif valide l instrument

`bcb6d393` (`HI_1_12_0`) : 312 marqueurs, et au decalage **+0** les quatre identifiants —
`0xB0171062` 37, `0xC0E34C44` 7, `0x3B2567D4` 7, `0x9212E428` 4, soit **55**. C est EXACTEMENT
le `grenades.available = 55` publie par l artefact cuit de ce temoin (note M3 §2.5). L instrument
reproduit la production au lancer pres ; ce qu il dira des films anciens est donc lisible.

### 6 bis.2 Le fait brut : un decalage d UN BIT, stable sur cinq films et cinq builds

Sur les cinq films anciens, AUCUN des quatre identifiants n apparait au decalage 0. Un seul y
apparait a un decalage stable : `0x3B2567D4` (rang 2) au decalage **-1**, `hors_marqueur = 0`,
sur quatre films (42, 66, 41, 5 occurrences). Les trois autres n ont AUCUN decalage stable dans
+/- 512 bits, tout en etant presents 13 a 287 fois ailleurs dans le flux — contre **0,07 a 0,26**
occurrence attendue par hasard. Ils sont donc ECRITS dans le film, hors de portee du marqueur.

Deux observations ont ferme l enigme.

1. **Sur les builds anciens, les neuf identifiants de la famille derriere le marqueur ont tous
   leur bit de poids faible a ZERO** (9 sur 9, `e5adf7b2`), alors que le build recent en melange
   pairs et impairs (6 pairs, 5 impairs). Un bit de poids faible toujours nul est la signature
   d une fenetre lue UN BIT TROP TARD.
2. **La suite de bits, relevee et posee cote a cote.** Un lancer de `bcb6d393` et un de
   `111fa685`, alignes sur le debut du marqueur :

```
bcb6d393  +0..23  01001100 00001100 00000000                   (0x4C0C00)
          +24..55 10110000 00010111 00010000 01100010          (0xB0171062 = frag)
          +56..87 01000010 11001001 01100111 10011111          (0x42C9679F)

111fa685  +0..23  01001100 00001100 00000000                   (0x4C0C00)
          +24..55 01110110 01001010 11001111 10101000          (0x764ACFA8)
          +56..87 10000101 10010010 11001111 00111110          (0x8592CF3E == 0x42C9679F << 1)
```

Le bloc de 32 bits qui SUIT l identifiant est, au bit pres, celui du build recent **decale d un
bit a gauche**. Autrement dit : `ancien[23..] == recent[24..]`.

### 6 bis.3 La consequence, et la prediction qu elle impose

Sur les builds anciens, l amorce fait **23 bits** (`0x260600`) et l identifiant commence a
**+23**. La production, elle, compare **24 bits** a `0x4C0C00` : son vingt-quatrieme bit n est
pas de l amorce, c est **le bit de poids fort de l identifiant**. Elle ne peut donc reconnaitre
un lancer ancien QUE si ce bit vaut 0 — et elle lit ensuite `identifiant << 1`, qui n est jamais
dans la liste blanche. Des quatre identifiants, un seul a son bit de poids fort a 0 :
`0x3B2567D4`. D ou les 42, 66, 41 et 5 occurrences observees au decalage -1, et rien d autre.

**Prediction falsifiable** : les lancers des trois autres grenades portent donc le motif de
24 bits `0x4C0C00` avec son dernier bit a 1, soit **`0x4C0C01`**, et leur identifiant se lit a
**+23**.

### 6 bis.4 La prediction est verifiee, sur les cinq films et les quatre rangs

| film | build | marqueurs `0x4C0C00` | marqueurs `0x4C0C01` | position | frag (0) | plasma (1) | dynamo (2) | spike (3) | **total** |
|---|---|---|---|---|---|---|---|---|---|
| `bcb6d393` | `HI_1_12_0` | 312 | 67 | **+24** | 37 | 7 | 7 | 4 | **55** |
| `e5adf7b2` | `HI_1_11_0` | 1 082 | 1 021 | **+23** | 27 | 39 | 42 | 30 | **138** |
| `111fa685` | `HI_1_10_0` | 1 792 | 998 | **+23** | 37 | 26 | 66 | 30 | **159** |
| `084a804d` | `HI_1_10_0` | 1 807 | 571 | **+23** | 207 | 28 | 41 | 13 | **289** |
| `60ae07c4` | `HI_1_8_0` | 245 | 668 | **+23** | 270 | 18 | 0 | 22 | **310** |
| `a349fea8` | v33, sans section | 763 | 1 092 | **+23** | 287 | 56 | 5 | 38 | **386** |

**1 282 lancers sur les cinq films qui en publiaient ZERO**, les quatre rangs presents (dynamo
seul manque sur `60ae07c4`). Controle negatif : sur le build recent, le marqueur `0x4C0C01` ne
porte AUCUN identifiant de la liste blanche (67 occurrences, zero reconnu) — la grammaire a
24 bits y est bien la bonne.

Controle croise : derriere le marqueur impair de `111fa685`, la lecture a +24 rend `0x602E20C4`
(37), `0x2425C850` (30), `0x81C69888` (26) — soit exactement `frag << 1`, `spike << 1`,
`plasma << 1`, aux memes comptes que la lecture a +23. Les deux lectures decrivent le meme
evenement.

---

## 7 bis. LA QUESTION (2) : ELLE NE SE POSE PLUS, ET SON INSTRUMENT NE DISCRIMINE PAS

La question (2) ne devait etre jouee que si (1) etait negative. (1) est POSITIVE. La passe D a
neanmoins ete jouee, et son resultat merite d etre consigne parce qu il ferme une voie que la
note M3 §2.3 presentait comme « la voie praticable ».

**Elle ne discrimine pas, et le temoin de hasard le prouve — y compris sur le temoin positif.**
Sur `bcb6d393`, ou les 55 lancers et leurs rangs sont connus : `0xB0171062` (frag, rang 0)
s apparie 24 fois sur 25 au rang 0 — mais le TEMOIN (memes appariements, instants decales de
37 s) le fait 13 fois sur 13. `0x3B2567D4` (dynamo, rang 2) s apparie **zero** fois au rang 2.
Sur `111fa685` les deux colonnes sont indiscernables (`0x764ACFA8` : 37 apparies, rangs
[5, 0, 31, 1] contre temoin 30 apparies, rangs [2, 0, 28, 0]).

**La cause est mesuree** : le canal i22 des paquets delta rend **87 lectures sur 1 481 records**
(`bcb6d393`) et **265 sur 5 982** (`111fa685`), soit 4 a 6 %. Avec 25 a 121 porteurs, les
intervalles entre deux lectures consecutives d un meme porteur durent des dizaines de secondes :
tout instant tombe dans plusieurs decrements a la fois, et l appariement par appartenance ne
designe aucun rang. C est le critere **(2 bis)** ecrit avant la mesure — a ceci pres qu il ne
vise pas les builds anciens, mais **la methode elle-meme, sur tous les builds**.

La note M3 §2.5 disait de `111fa685` qu il « porte 476 images-cles d inventaire de grenades et
0 lancer publie : c est exactement le materiau d appariement dont la methode d origine a besoin ».
Les 476 sont des lectures d IMAGE-CLE (`replay/inventory_decode.go`) ; le canal DELTA, seul a
porter un instant, en rend 265. La phrase est donc a corriger.

---

## 8. VERDICT (3)

**LA GRAMMAIRE A BOUGE, PAS LES IDENTIFIANTS. L hypothese de l utilisateur est CONFIRMEE**, et
de la facon la plus economique qu elle envisageait : ni le nombre, ni l ordre, ni les valeurs des
types de grenade n ont change depuis la sortie du jeu. Ce qui a change tient en **UN BIT** :

| build | amorce | identifiant | index auteur |
|---|---|---|---|
| >= `HI_1_12_0` | 24 bits, `0x4C0C00` | a **+24** | a **+103** (54 sur 55 dans 0..7) |
| <= `HI_1_11_0` (et v31 / v33 sans section) | **23 bits**, `0x260600` | a **+23** | **NON ETABLI** (point 6 ci-dessous) |

Ce que le critere (1a) prescrivait est donc ce qui s applique : **3.3.1 code une POSITION par
build, pas une liste blanche par build.** Precisement :

1. `GrenadeTypeIDsByRank` **reste une constante du titre** — aucune entree de profil, aucune
   liste « ancienne », aucun rang devine. La ligne de profil esquissee par la note M3 §2.4 ne
   doit PAS etre ecrite.
2. L entree de profil est la LARGEUR DE L AMORCE (23 ou 24 bits) et, solidairement, la position
   de l identifiant (+23 ou +24). Cle `build=`, provenance `mesuree`, preuve = ce document.
3. `grenadeMarker` cesse d etre une constante de paquet : il se derive de l amorce de 19 bits et
   du `ti` projectile LU DANS LE FILM (`marqueur(ti) = ((ti & 31) << 19) | 0x40C00`, verifie sur
   les sept builds, mesure 0), tronque a 23 ou 24 bits selon le build.
4. Le compteur `grenadeThrowsUnranked` de M3-Q5 reste a **ZERO** sur ces builds : chaque lancer
   recupere porte un rang connu. La decision B (« un lancer identifie ou rien ») est tenue sans
   rien mettre de cote.
5. Le sixieme bit d index (a `marqueur - 1`) DOIT etre lu : sans lui le balayage ramasse aussi
   les naissances de `managed-player` (`ti = 9`, meme marqueur) — 3 a 26 par film mesurees.
6. **Ce qui n est PAS etabli, et que 3.3.1 devra mesurer : la position du champ d INDEX AUTEUR
   sur les builds anciens.** Le decalage d un bit ne s y propage pas mecaniquement : les 47 bits
   qui separent l identifiant de l index sont un TOTAL mesure, pas une suite de champs lue
   (`grammar/grenade_events.go` le dit lui-meme). Mesure : a +103, `bcb6d393` rend 54 index sur
   55 dans 0..7 (position juste) et `111fa685` **49 sur 159** (position fausse) ; mais le seul
   critere « toutes les valeurs dans 0..7 » ne tranche pas — **18 decalages sur 49** le
   satisfont sur les deux films. Il faut un critere plus fort (recoupement avec la table des
   joueurs du film, ou avec le pont index / slot de `replay`). Tant qu il n est pas pose, les
   lancers anciens sont publiables avec leur TYPE et sans auteur — ce que la couverture distingue
   deja (`grenadesDisponibles` contre `grenadesRattachees`).

Gain attendu au corpus gate (item 3.3.2) : **1 282 lancers sur les cinq temoins anciens**, zero
perte sur les neuf autres (la grammaire a 24 bits est inchangee). Rapporte aux 82 films de builds
anciens du parc, l ordre de grandeur est de **20 000 lancers**, au-dela des « 5 000 a 10 000 »
que le plan annoncait.

---
## 9. ETAT — CLOS

| Question | Etat |
|---|---|
| mesure 0 — le marqueur a-t-il bouge ? | **NON** (2026-09-16). `ti` projectile = 41 sur les sept builds, marqueur derive `0x4C0C00` partout. Critere (1b) ecarte |
| (1) les 4 identifiants actuels ailleurs dans les films anciens | **OUI, a +23** (2026-09-17). Un seul les revele au marqueur PAIR (`0x3B2567D4`, le seul dont le bit de poids fort est a 0) ; les quatre au marqueur IMPAIR `0x4C0C01`. 1 282 lancers sur cinq films qui en publiaient zero |
| (2) appariement aux decrements i22 | **JOUEE malgre tout, et NEGATIVE POUR LA METHODE** : le temoin de hasard egale la mesure, y compris sur le temoin positif dont les rangs sont connus. Le canal i22 des paquets delta rend 4 a 6 % de lectures : les intervalles sont trop larges pour designer un rang (critere 2 bis) |
| (3) verdict | **LA GRAMMAIRE A BOUGE, PAS LES IDENTIFIANTS.** Un bit : amorce de 23 bits et identifiant a +23 jusqu a `HI_1_11_0`, 24 bits et +24 a partir de `HI_1_12_0`. §8 dit ce que 3.3.1 code |

Un seul point reste ouvert, et il est nomme : la position du champ d INDEX AUTEUR sur les builds
anciens (§8, point 6). Elle ne bloque pas la publication du TYPE, qui est ce que M3-Q5 exige.
