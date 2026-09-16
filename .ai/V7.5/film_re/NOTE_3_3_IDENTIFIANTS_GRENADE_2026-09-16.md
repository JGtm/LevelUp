# NOTE 3.3 (volet RECHERCHE) — LES IDENTIFIANTS DE GRENADE DES BUILDS ANCIENS : LA GRAMMAIRE A-T-ELLE BOUGE, OU LES IDENTIFIANTS ?

> Ouverte le 2026-09-16, branche `feat/decfilm-33r`, base `e7b9bd48e`. Instrument sous
> `apps/go-api/tools/film_re/grenadeids/` et `tools/film_re/cmd/grenadeids/`, tag `research`.
> Etat au 2026-09-16 : instrument ECRIT, COMPILE et VERIFIE sur ses invariants ; **mesure 0
> JOUEE** (les sept mini-bobines, autorisation du pilote — §4) ; les mesures 1 a 6 sur les films
> du cache attendent la « voie libre » (un seul decodage a la fois sur ce poste).
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

## 4. LE PLAN DE MESURE (a jouer a la voie libre, un film a la fois, dans cet ordre)

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

## 5. LES CRITERES DE VERDICT, ECRITS AVANT LA MESURE

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

## 7. ETAT

| Question | Etat |
|---|---|
| mesure 0 — le marqueur a-t-il bouge ? | **JOUEE le 2026-09-16 : NON.** `ti` projectile = 41 sur les sept builds (4/4 noms a chaque fois), marqueur derive `0x4C0C00` partout. Critere (1b) ECARTE |
| (1) les 4 identifiants actuels ailleurs dans les films anciens | instrument PRET, non joue — attend la voie libre. La question se reduit desormais au DECALAGE (passe A) et au CHAMP (passe B) |
| (2) appariement aux decrements i22 | instrument PRET, non joue — ne se joue que si (1) est negative |
| (3) verdict | non prononce |

Ce qui est acquis a cette heure :

1. **Le marqueur n a pas bouge** sur les sept builds connus (mesure 0), et l archetype projectile
   est au rang 41 partout, identifie par ses quatre noms de composant.
2. La derivation `marqueur(ti) = ((ti & 31) << 19) | 0x40C00` est verifiee, et elle fait de
   `ti=9` (`managed-player`) un HOMONYME de `ti=41` — l ambiguite se leve par la lecture du
   sixieme bit d index a `marqueur - 1`, que l instrument fait desormais.
3. Les sept mini-bobines NE PORTENT PAS de paquet delta : elles ne peuvent pas servir les passes
   A a D, seulement la mesure 0.

Ce qui reste ouvert : la POSITION lue apres le marqueur, et le CHAMP. C est exactement ce que
l hypothese de l utilisateur designe comme le plus vraisemblable une fois le marqueur ecarte.
