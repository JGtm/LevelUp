# Lot 6.11 — les passages de drapeau rapides (2026-09-11)

> Worktree `LevelUp-wt-couverture-objectifs`, branche `wt/couverture-objectifs`, base `0fcb22ad9`.
> Preuves d'entree : `.ai/V7.5/RAPPORT_COUVERTURE_OBJECTIFS_B1_2026-09-11.md` §4 et §10 (D1, D2),
> `.ai/V7.5/RAPPORT_COUVERTURE_OBJECTIFS_B2_2026-09-11.md` §8 (D2),
> `.ai/V7.5/AUDIT_COUVERTURE_OBJECTIFS_2026-09-10.md` §3 et §12.4.
> **Aucune base DuckDB n'a ete ouverte**, aucun artefact du parc n'a ete recuit : tous les
> chiffres viennent de cuissons HORS LIGNE (`cmd/replay-build --facts`, racine de travail en
> scratchpad) confrontees a l'oracle API par `.ai/V7.5/outillage/couverture_objectifs`.

## 0. Verdict en cinq lignes

1. **Le ratio publie/oracle du drapeau passe de 1,130 a 1,066** sur les 11 films CTF a calque
   (2 396,9 s -> 2 262,2 s pour 2 121,2 s d'oracle). L'exces tombe de 275,7 s a **141,0 s**
   (-48,9 %). Le gate a 1,05 n'est PAS atteint, et le residu est decompose au §6.
2. **Le cas nomme au gate de B1 est ferme** : `16ea3668` / 2535417044536883 passe de 55,1 s
   publiees a **1,3 s** pour 0,9 s d'oracle ; le film passe de 1,443 a **1,031**.
3. **L'hypothese D1 de B1 est REFUTEE par la mesure** : le passage de main en main existe, mais
   il ne pese presque rien (15 portages, 15,0 s). Ce qui pesait, c'est le drapeau qui **RENTRE
   CHEZ LUI** pendant qu'on le croyait porte : 17 portages, 119,7 s.
4. **`fb1a1a72` est corrige a la source** : ses 10 prises de drapeau fausses tombent a 0, et sur
   les **69 films du parc c'est la SEULE action qui change**.
5. **`4ecdf3e7` / 2533274877168586 est statue `[!]`** : la cause est etablie sur piece — un
   SIEGE du statborg occupe successivement par trois personnes dans la meme manche — et le
   correctif (identite par fenetre d'occupation) est hors du perimetre de ce lot.

---

## 1. Etat des items

| # | item | statut | gate |
|---|---|---|---|
| 1 | la prise d'un AUTRE joueur du MEME drapeau ferme le portage | `[x]` | tenu, population chiffree (petite) |
| 2 | reprise a soi-meme apres un lacher bref | `[x]` | mesure faite, borne prouvable trouvee et posee |
| 3 | `fb1a1a72`, 10 prises pour 0 a l'oracle sur le slot 24 | `[x]` | tenu |
| 4 | `4ecdf3e7`, 2533274877168586 a 0 publie pour 4,1 s | `[!]` | cause instruite, correctif hors perimetre (§5) |
| — | gate de lot (ratio <= 1,05) | `[!]` | **1,066** atteint ; residu decompose au §6 |

Commits, dans l'ordre :

| item | SHA | message |
|---|---|---|
| 1 | `6bcca3456` | la PRISE D'UN AUTRE JOUEUR du MEME drapeau ferme le portage |
| 2 | `0e35b3235` | le DRAPEAU QUI RENTRE CHEZ LUI ferme le portage |
| 3 | `6066e462c` | le DOMAINE des compteurs au niveau de l'ENREGISTREMENT |

## 1.1 Protocole de mesure

Quatre parcs cuits HORS LIGNE dans le scratchpad de session, avec les memes faits de match que
la phase B1 (`b1/facts/*.facts.json`, 83 fichiers exportes avant le lot) :

| parc | binaire | contenu |
|---|---|---|
| `parcA` | HEAD `0fcb22ad9` | **la reference AVANT**, 69 films |
| `parcI1` | + item 1 | 12 films CTF |
| `parcI2` | + item 2 | 12 films CTF |
| `parcI3` | + item 3 | **la reference APRES**, 69 films |

**`parcA` reproduit B1 a la decimale** : ratio 1,1300 (B1 : 1,130), 2 396,9 s pour 2 121,2 s,
**469 periodes**, **45** joueurs au-dessus de leur oracle a 0,5 s pres, **2** sous 0,8. Les quatre
valeurs sont celles du tableau de B1 §4, a l'identique — la chaine de mesure est donc la meme.

Sorties APRES committees sous `.ai/V7.5/replay2d/registre_film/vague6_611_*.tsv|log`.

---

## 2. Item 1 — la prise d'un AUTRE joueur du MEME drapeau (`6bcca3456`)

**Cause verifiee sur pieces.** `boundFlagCarries` (`replay/flag_carries.go:259`) ne connait que la
prise SUIVANTE DU MEME SLOT (`nextOpeningOfSlot`). Un drapeau qui passe a un coequipier laissait
donc le portage precedent courir jusqu'au fait suivant — mort, capture, fin de match — et le
calque publiait deux porteurs du meme drapeau au meme instant.

**Savoir de QUEL drapeau il s'agit, sans geometrie.** Le bornage precede l'attribution
(`flag_assign.go` a besoin des positions, qui dependent de `t1`) : `flagIndex` n'existe pas encore.
Il n'est pas necessaire — l'invariant dur du mode suffit, et c'est celui que `flag_assign.go`
pose deja : **on ne porte jamais son propre drapeau**. Donc deux coequipiers portent le MEME, et
deux adversaires en portent deux DIFFERENTS. Un seul drapeau en jeu (variante neutre, carte hors
catalogue) : toute prise d'un autre joueur borne. Equipe non lue ou plus d'un socle adverse :
abstention, et elle SE COMPTE (`carrierTeamUnknown`).

**Le filtre par equipe porte tout le poids de la regle, et c'est mesure.** Sans lui — en fermant
sur la prise de n'importe quel autre joueur — 57 portages et **775,1 s** seraient retires a tort
sur les 11 films, soit un ratio de **0,739** contre un oracle a 1,000. Le temoin negatif
`TestUnePriseDUnADVERSAIRENeFermeRien` fige exactement ce chiffre.

**Gate (11 films CTF, `parcA` -> `parcI1`).**

| grandeur | avant | apres |
|---|---|---|
| ratio des 11 films | 1,1300 | **1,1229** |
| publie | 2 396,9 s | **2 381,9 s** (-15,0 s) |
| portages fermes par passage (`closedByHandoff`) | — | **15** |
| `overlaps` / `closedOverlaps` | 0 / 0 | **0 / 0** |
| periodes | 469 | 469 |
| joueurs au-dessus de leur oracle a 0,5 s pres | 45 | 45 |
| actions (toutes, tout le parc) | — | **inchangees a l'unite** |

**La population est PETITE, et c'est un resultat, pas un echec.** 15 portages sur 469. Le lacher
suivi d'une reprise par un coequipier est deja date, presque toujours, par la VIE LIBRE de l'objet
(`closedByObject`, lot 6.7-B1) : l'objet touche le sol entre les deux mains. **L'hypothese D1 de
B1 — « c'est la cause du residu de 275,7 s » — est donc refutee par la mesure.** La regle reste :
elle rend le calque vrai par construction (un drapeau n'a qu'un porteur) la ou il ne l'etait que
par constat, et elle ferme ce que le canal objet ne voit pas.

**Repartition par film** (`closedByHandoff`) : `f8efc5ca` 4, `b8a44fe8` 3, `8bc6074f` 2,
`a0c36016` 2, `16ea3668` 1, `58864b3c` 1, `7fce3219` 1, `4ecdf3e7` 1, `bc60b4d9` 0, `bf5ced1b` 0,
`cde26226` 0.

**Tests.** `flag_carries_handoff_test.go`, 6 tests portant sur `closeByHandoff` directement — un
choix delibere : dans le document PUBLIE, un portage borne par la prise suivante est de toute
facon coupe a la frame de cette prise (`spansOfTransitions`), si bien qu'un recouvrement et une
fermeture se ressemblent a l'oeil ; ce qui change est la BORNE. L'effet de bout en bout est fige
par `TestFlagOverlapsComptesParDrapeau`, qui a desormais **deux moities** : sans equipe lue le
recouvrement subsiste et se compte (constat C3, inchange), avec les equipes il n'existe plus.

**Mutations jouees.** (a) supprimer le test `drapeaux[o.xuid] != mien` de
`flagFirstOtherOpening` -> `TestUnePriseDUnADVERSAIRENeFermeRien` rouge, et lui seul ;
(b) debrancher `closeByHandoff` de `buildFlagCarries` -> `TestFlagOverlapsComptesParDrapeau` rouge.

---

## 3. Item 2 — le drapeau qui RENTRE CHEZ LUI (`0e35b3235`)

### 3.1 La mesure d'abord, comme l'item l'exige

Population des REPRISES (suites de spans portes CONTIGUS du meme joueur — un portage ferme par
la prise suivante du meme slot, dont le lacher n'est date par rien, donc sans `dropped`
intercale), relevee sur `parcI1` :

| grandeur | valeur |
|---|---|
| suites de reprise | **18** |
| duree publiee dans ces suites | **228,2 s** |
| dont AU-DELA du premier span | **54,1 s** |
| spans isoles | 451, pour 2 153,7 s |
| exces total des 11 films | 260,7 s |

La reprise **n'explique donc pas a elle seule** l'exces : 54,1 s de 260,7 s. Le releve nomme en
revanche le pire cas — `16ea3668` / 2535417044536883, une suite de 55,1 s pour 0,9 s d'oracle,
exactement le cas laisse ouvert par B1.

### 3.2 La borne prouvable, et pourquoi elle l'est

Un VOL (`flag_steals`) se fait AU SOCLE (`flag_assign.go`) : pour que ce joueur RE-VOLE le
drapeau a 286,5 s apres l'avoir vole a 231,7 s, il a fallu que le drapeau **RENTRE CHEZ LUI**
entre-temps. Et un drapeau chez lui n'est dans la main de personne.

Deux chaines datent cette rentree, **et toutes deux NOMMENT leur drapeau** :

| chaine | ce qui nomme le drapeau | compteur |
|---|---|---|
| RETOUR CREDITE (`flag_returns`) | l'EQUIPE de qui le rend : on ne renvoie que le SIEN | `closedByReturn` |
| RENTREE DE L'OBJET | le SOCLE ou l'objet est re-cree (`flagObjectHomecomings`) | `closedByHome` |

Les deux etaient deja lues — elles servent l'etat du sol dans `flag_carries_lives.go` — et
**aucune ne fermait un portage** : le drapeau rentrait chez lui pendant qu'un joueur etait cense
courir avec. Comme les autres chaines du calque, celle-ci ne retient qu'un instant STRICTEMENT
interieur a `]t0, t1[` : elle ne peut que RACCOURCIR.

**Le refus d'ambiguite de la rentree est propre a cet etage.** Une naissance au socle du drapeau F
peut aussi etre le drapeau ADVERSE tombe au pied de ce socle — ce que `applyFlagHomecoming` ecarte
par la POSITION des laches. Ici les positions n'existent pas encore, donc le refus se fait sur le
TEMPS : une rentree dont un portage d'un AUTRE drapeau s'acheve dans la meme seconde
(`flagFreeDropWindowMS`, aucun seuil neuf) n'est pas retenue.

**Une fin CHEZ LUI n'est pas un lacher.** `flagCarryRaw.endsHome()` regroupe la capture et la
rentree ; les trois endroits qui posent le drapeau apres une fin — l'etat du sol (`poser`), le
repositionnement du lacher (`repositionFlagDrops`) et la transition publiee (`flagLifeClose`) —
la traitent pareil. Un retour qui vient de fermer un portage **ne se compte plus en abstention**
(`ambiguousReturns` de `16ea3668` : 4 -> 1).

### 3.3 Gate (11 films CTF, `parcI1` -> `parcI2`)

| film | publie avant | publie apres | oracle | ratio avant | ratio apres |
|---|---|---|---|---|---|
| `16ea3668` | 189,0 s | **135,0 s** | 131,0 s | 1,443 | **1,031** |
| `58864b3c` | 113,7 s | **112,5 s** | 110,1 s | 1,033 | **1,022** |
| `7fce3219` | 188,5 s | **183,4 s** | 173,9 s | 1,084 | **1,055** |
| `8bc6074f` | 213,6 s | **213,5 s** | 209,3 s | 1,021 | **1,020** |
| `a0c36016` | 166,4 s | **160,5 s** | 156,5 s | 1,063 | **1,026** |
| `b8a44fe8` | 538,0 s | **495,1 s** | 401,1 s | 1,341 | **1,234** |
| `bc60b4d9` | 128,7 s | **128,6 s** | 120,2 s | 1,071 | **1,070** |
| `bf5ced1b` | 39,0 s | **36,2 s** | 33,1 s | 1,178 | **1,094** |
| `cde26226` | 507,1 s | **499,5 s** | 480,4 s | 1,056 | **1,040** |
| `f8efc5ca` | 200,0 s | **200,0 s** | 196,0 s | 1,020 | **1,020** |
| `4ecdf3e7` | 97,9 s | **97,9 s** | 109,6 s | 0,893 | **0,893** |
| **total** | **2 381,9 s** | **2 262,2 s** | **2 121,2 s** | **1,1229** | **1,0665** |

| critere | avant | apres |
|---|---|---|
| joueurs au-dessus de leur oracle a 0,5 s pres | 45 | **39** |
| joueurs sous 0,8 de leur oracle | 2 | **2, LES MEMES** |
| periodes | 469 | **477** |
| `closedByHome` / `closedByReturn` | 0 / 0 | **17 / 0** |
| actions (tout le parc) | — | **inchangees a l'unite** |

**`closedByReturn` vaut ZERO sur les onze films, et ce n'est pas une chaine morte.** Un retour
CREDITE re-cree aussi l'objet a son socle : la chaine de l'OBJET voit donc les deux populations —
les retours credites ET les retours AUTOMATIQUES que personne ne credite — et elle les date au
meme instant ou plus tot, si bien qu'elle gagne toujours le `min`. La chaine creditee reste le
repli du film dont l'objet est muet (`scan.Free` vide), et elle est la seule des deux a n'avoir
besoin d'aucun refus d'ambiguite.

**Tests.** `flag_carries_home_test.go`, 6 tests, dont le temoin negatif
`TestUnRetourDuDRAPEAUADVERSENeFermeRien` (le retour credite a un joueur du camp du PORTEUR
nomme l'AUTRE drapeau) et l'abstention `TestUneRentreeAmbigueNeFermeRien`.
**Mutations jouees** : (a) rendre `flagOfOwner` insensible a l'equipe -> temoin negatif rouge ;
(b) retirer `flagAutreLacherProche` -> abstention rouge.

---

## 4. Item 3 — `fb1a1a72` : dix prises pour zero (`6066e462c`)

**Cause etablie sur le film, et ce n'est aucune des deux hypotheses du plan.**

| hypothese | verdict |
|---|---|
| erreur d'identite de slot | **REFUTEE** — le pont par le triplet nomme les huit slots (10 a 24), sans collision, et aucun autre joueur ne perd la prise : les sept autres slots sont exacts a l'unite |
| manche fantome (film a 2 manches) | **REFUTEE** — le film declare les manches 0 (1 017 enr.), 2 (149) et 5 (1) ; depuis B1, `RealRounds` ne retient deja que la 0 |
| **ancrage FORTUIT** | **ETABLIE** — voir ci-dessous |

Le slot 24 emet, a `t = 764 967 ms`, **un enregistrement de DOUZE composants** la ou ce slot n'en
emet jamais plus de six :

```
t=764967 manche=0  7={1,2415919104}  14={0,0}  19={0,0}  22={10,8}  23={0,105}
                  28={0,4}  32={3,8192}  36={56,14}  38={0,0}  39={0,0}
                  43={8,79}  46={-30456,0}
```

`comp 22 A` (= `flag_grabs`) y saute de rien a **10**, et `comp 23 B` (= `flag_secures`) a
**105**. Le second etait deja refuse par la borne de deroulage de B1 (16) ; le premier passait
**SOUS** cette borne. Un second enregistrement du meme genre suit a `t = 774 276` (quatorze
composants). **La borne PAR PAS ne pouvait pas voir ce defaut : il est au niveau de
l'ENREGISTREMENT** — exactement la decouverte D5 de B1, qui nommait le filtre sans le poser.

**Le seuil est mesure, pas choisi.** Sur 11 films du parc, tous modes (CTF, Oddball, Strongholds,
KOTH, Slayer, BTB) :

| population | valeur |
|---|---|
| pire valeur d'un enregistrement dont AUCUN canal n'atteint 2^20 | **102 934** (`4f77afc1`) ; 32 518 sur dix films sur onze |
| plus petite valeur de la population aberrante | **2 415 919 104** (0x90000000), jusqu'a 4,1 milliards |
| enregistrements hors domaine par film | 0 a 15, sur 327 a 9 111 |

Entre 102 934 et 1 048 576, **rien**. `statMaxCounter = 1 << 20` est dix fois au-dessus de la pire
valeur saine et exactement sous la plus petite aberrante. **Canaux A et B seulement** : les seuls
mesures, et les seuls que les tables d'objectif lisent — on ne borne pas ce qu'on n'a pas observe.

**Gate.**

| critere | attendu | mesure | verdict |
|---|---|---|---|
| `fb1a1a72` : joueurs au-dessus de leur oracle sur `flag_grabs` | 0 | **0** (10 -> 0 sur le slot 24) | `[x]` |
| `fb1a1a72` : 7 autres slots | inchanges | **inchanges** (1, 6, 4, 9, 1 prises = oracle) | `[x]` |
| actions du parc entier (69 films) | seule celle-ci bouge | **une seule action change dans tout le parc** | `[x]` |
| films non-CTF identiques a l'octet | oui | **5 changent** — voir §7 | `[!]` |

**Tests.** `objectiveevents/statborg_domaine_test.go` : temoin positif sur les deux vecteurs REELS
du corpus (creux et dense), le cas `fb1a1a72` reproduit a l'identique avec ses douze composants,
et les deux bords de la borne (signe compris). **Limite assumee** : aucun vecteur negatif au
niveau du BIT n'a ete fabrique — la position des valeurs dans le flux n'est pas adressable depuis
les aides de test existantes (`setBitsBE` adresse les en-tetes, pas les valeurs). Le garde-rail de
regression est donc le predicat plus le gate sur film.

---

## 5. Item 4 — `4ecdf3e7` / 2533274877168586 : `[!]`, et la cause est etablie

**Ce qui est publie.** Zero action, zero periode, 0,0 s pour 4,1 s d'oracle (1 prise, 1
securisation). Toutes les autres actions du film sont exactes a l'unite.

**Ce que le film dit.** Le pont d'identite nomme HUIT slots statborg (10 a 24) et
2533274877168586 **n'en a aucun** : le slot 12 est nomme 2535434009919524.

**Ce que la feuille de match dit — et c'est la preuve** :

| joueur | arrivee | depart | temps joue |
|---|---|---|---|
| 2533274877168586 | debut | **19:39:43** (`left_in_progress`) | 157 s |
| `bid(29.0)` (bot) | 19:39:43 | 19:39:44 | 1 s |
| 2535434009919524 | **19:39:44** (`joined_in_progress`) | — | 85 s |

**Un seul SIEGE, trois occupants successifs dans la meme manche.** Et le film le date : la serie
`comp 22 A` du slot 12 ne porte qu'un point, `t = 155 197 : 0` — le compteur du siege REMIS A
ZERO, a 155,2 s, quand le premier occupant part a 157 s de jeu. Les deux horloges concordent.

**Pourquoi ce n'est pas corrigeable dans ce lot.** `SlotIdentityByRound` resout un slot pour une
MANCHE ENTIERE, par le triplet (frags, morts, assistances) de fin de match : avec deux occupants
dans la meme manche, il ne peut en nommer qu'un, et il nomme celui dont le triplet final colle.
Le correctif est une identite par **fenetre d'OCCUPATION** — les instants existent deja
(`MatchPlayerFact.JoinMatchMS` / `LeaveMatchMS`, alimentes par `first_joined_time` /
`last_leave_time`), et le film porte le marqueur de la remise a zero. C'est un chantier du pont
d'identite (`internal/analysis/objectiveevents/`), pas de `replay/` : hors perimetre, **statue
`[!]`** comme le plan l'autorise.

**Le second joueur de `4ecdf3e7`** (2535469190789936, 3,9 s publiees pour 14,3 s) releve d'une
autre cause, deja comptee : `coverage.flagCarries.noTrack = 1` — une prise dont le porteur n'a
aucune trajectoire publiee a l'instant de la prise, donc aucun drapeau a dessiner. Consigne, non
traite.

---

## 6. Gate du lot — decomposition du residu

| critere (audit §11 L5) | attendu | mesure | verdict |
|---|---|---|---|
| ratio des 11 films | <= 1,05 | **1,0665** (1,130 avant) | `[!]` |
| joueurs au-dessus de leur oracle a 0,5 s pres | minimum atteignable | **39** (45 avant) | chiffre ci-dessous |
| aucun joueur sous 0,8 de son oracle | 0 nouveau | **2, LES MEMES 2** qu'avant | `[x]` |
| nombre de periodes | >= 469 | **477** | `[x]` |
| actions (prises, captures, vols, retours, securisations) | 0 unite de mouvement | **1 seule dans tout le parc**, celle que l'item 3 corrige | `[x]` |
| films non CTF identiques a l'octet | oui | **5 changent**, tous des corrections (§7) | `[!]` |

### 6.1 Ou est le residu — il tient dans UN film

Exces total : **141,0 s** (275,7 s avant le lot).

| film | exces | part |
|---|---|---|
| `b8a44fe8` | **94,0 s** | **67 %** |
| `cde26226` | 19,1 s | 14 % |
| `7fce3219` | 9,5 s | 7 % |
| `bc60b4d9` | 8,4 s | 6 % |
| `16ea3668` / `a0c36016` / `8bc6074f` / `f8efc5ca` | 4,0 a 4,2 s chacun | 11 % |
| `bf5ced1b` | 3,1 s | 2 % |
| `58864b3c` | 2,4 s | 2 % |
| `4ecdf3e7` | -11,7 s (deficit, cf. §5) | — |

**Sans `b8a44fe8`, les dix autres films sont a 1,027** (1 767,1 s pour 1 720,1 s) — sous le gate.

### 6.2 Les joueurs qui restent au-dessus, et leur cause

39 joueurs sur 73. **Quatre d'entre eux portent 133,2 s des 141,0 s**, et tous les quatre sont sur
`b8a44fe8` :

| joueur | periodes | publie | oracle | ecart | cause |
|---|---|---|---|---|---|
| 2535442462807197 | 13 | 140,4 s | 59,0 s | **+81,4 s** | reprise au SOL par lui-meme, canal objet muet |
| 2533274858911298 | 2 | 56,4 s | 28,4 s | +28,0 s | idem |
| 2535455799302553 | 10 | 30,9 s | 14,9 s | +16,0 s | idem |
| 2533274823110022 | 2 | 25,0 s | 17,2 s | +7,8 s | idem |

Les **35 autres** se repartissent 7,8 s : trente d'entre eux depassent de moins de 1,5 s, c'est-a-
dire de l'ordre de la demi-seconde par periode — le grain de la mesure.

**La cause du residu, nommee.** Le pire span du parc est `b8a44fe8` / 2535442462807197,
frames 6 590 a 7 169 : **58,0 s publiees** pour un joueur dont l'oracle vaut 59,0 s sur SEIZE
prises. Il prend le drapeau a la frame 6 590 et le REPREND a la frame 7 171 ; entre les deux,
(a) personne d'autre ne le prend — la regle de l'item 1 se tait, (b) le drapeau ne rentre JAMAIS
chez lui — il est repris AU SOL (`flag_grabs`, pas `flag_steals`), donc la regle de l'item 2 se
tait, et (c) aucune vie libre de l'objet ne nait a ses pieds — la regle de B1 se tait. **Les trois
chaines qui datent un lacher sont muettes sur ce cas, et c'est le residu.**

### 6.3 Le canal qu'on a REFUSE d'utiliser, et pourquoi

Le MARQUEUR de portage des images-cles (`flag_carries_marker.go`) pourrait borner ces spans : une
image-cle sans marqueur sur le slot du porteur prouverait qu'il ne porte plus rien. **Il n'a pas
ete utilise, et c'est un choix argumente** : ce marqueur est le **CONTROLE INDEPENDANT** du
calque — la seule chaine disjointe de celle des compteurs. S'en servir comme SOURCE rendrait
`markerObserved` / `markerConfirmed` tautologiques, et le calque perdrait sa seule preuve
exterieure. Mesure a l'appui : sur `b8a44fe8`, `markerObserved = 15` et `markerConfirmed = 15` —
le controle ne contredit d'ailleurs pas ces spans (il dit qu'au moins UNE image-cle de chacun
porte le marqueur, ce qui est compatible avec un lacher plus tard).

---

## 7. Films a recuire — la liste exacte

**17 des 69 films mesures changent** (comparaison a l'octet `parcA` -> `parcI3`, les 52 autres
sont IDENTIQUES).

| films | raison | item |
|---|---|---|
| `16ea3668` `58864b3c` `7fce3219` `8bc6074f` `a0c36016` `b8a44fe8` `bc60b4d9` `bf5ced1b` `cde26226` `f8efc5ca` `4ecdf3e7` | passage de main en main et drapeau rentre chez lui | 1 et 2 |
| `fb1a1a72` | dix prises fausses retirees | 3 |
| `28c9b538` | `coverage.flagCarries.steals` 2 -> 0 (bruit sur un film sans drapeau) | 3 |
| `32d9a94f` | une emission de score parasite retiree (`coverage.score.points` 1 172 -> 1 170) | 3 |
| `8a485699` | une serie `assists` parasite retiree (points 868 -> 866) | 3 |
| `bfecd02b` `d8b13ec2` | un slot dont la PROVENANCE du pont passe de `triplet_feuille` a `instants_de_mort` — **meme xuid** | 3 |

**Aucune montee de `SchemaVersion`** : les quatre champs neufs
(`closedByHandoff`, `carrierTeamUnknown`, `closedByReturn`, `closedByHome`) sont `omitempty` des
DEUX cotes (stocke et servi), et absents de tous les artefacts ou ils valent zero.

**Le gate « films non CTF identiques a l'octet » n'est donc PAS tenu**, et les cinq ecarts sont
listes ci-dessus : aucun ne retire une action, quatre retirent du bruit, un ameliore la provenance
d'un pont. C'est le prix du filtre au niveau de l'enregistrement, et il etait deja nomme par B1
(decouverte D5) comme le bon niveau.

---

## 8. Gates techniques, joues reellement

| commande | resultat |
|---|---|
| `CGO_ENABLED=0 go test ./internal/analysis/replay/ ./internal/analysis/objectiveevents/ ./internal/replaybuild/` | **ok**, 3 paquets |
| `CGO_ENABLED=1 go test ./internal/service/...` | **ok**, 4 paquets |
| `golangci-lint run` sur `analysis/replay`, `analysis/objectiveevents`, `domain/replaydoc`, `service/replayview` | **0 issues** |
| `go vet` (hook de pre-commit, 3 fois) | propre |
| `make openapi-gen` | `api/openapi.yaml` regenere, diff additif de 12 lignes |
| `openapi-typescript` | `generated.ts` regenere, diff additif de 8 lignes, tous champs OPTIONNELS |

**Goldens.** Aucun golden de CI ne change (les goldens de rejeu portent sur `000d5950`, un Slayer
sans calque de drapeau). La SEULE reference de test qui bouge est
`TestFlagOverlapsComptesParDrapeau`, et elle est RENFORCEE : la moitie historique (constat C3,
equipes inconnues) est conservee telle quelle, une seconde moitie fige le fait neuf.

**Web** : non touche. Les quatre compteurs neufs entrent au contrat (`openapi.yaml`,
`generated.ts`) en additif et optionnel ; aucun code web ne les lit, aucune chaine i18n n'est
creee.

**Corpus d'equivalence** : non joue ici, il appartient au superviseur (instruction du lot).

---

## 9. Decouvertes, consignees et NON traitees

- **D1 — `bc60b4d9` (Illusion) declare TROIS socles de drapeau, dont DEUX pour l'equipe 0.** Les
  regles par equipe de ce lot s'abstiennent donc sur les 10 portages du camp adverse
  (`carrierTeamUnknown = 10`, le seul film du parc a en porter). Le catalogue d'objectifs de cette
  carte melange vraisemblablement le socle de la variante neutre avec un socle d'equipe. A verifier
  au catalogue, pas dans `replay/`.
- **D2 — `4ecdf3e7` : un SIEGE pour trois occupants successifs dans une manche** (§5). Le correctif
  est une identite par fenetre d'occupation, dans `objectiveevents/` ; les instants necessaires
  sont deja aux faits de match.
- **D3 — le residu du drapeau tient dans une seule forme** : le lacher suivi d'une REPRISE AU SOL
  par le meme joueur, sur un drapeau qui ne rentre pas chez lui et dont l'objet ne replique pas sa
  naissance (§6.2). Les trois chaines de datation sont muettes ; la quatrieme (le marqueur
  d'image-cle) est le controle independant du calque et ne doit pas devenir une source (§6.3).
- **D4 — `closedByReturn` vaut 0 sur les onze films** : la chaine de l'OBJET voit aussi les retours
  credites et les date au meme instant ou plus tot. La chaine creditee n'est pas morte pour autant
  — elle est le repli du film dont l'objet est muet — mais elle ne se verifiera sur piece que le
  jour ou un tel film entrera au parc.
- **D5 — le filtre de domaine se limite aux canaux A et B.** C et D (les deux canaux
  conditionnels, ouverts le 2026-08-31) n'ont pas ete mesures : un ancrage fortuit qui ne salirait
  que ceux-la passerait encore. A mesurer le jour ou un lot les lira.

---

## 10. Reproduction

```bash
# 1. cuisson HORS LIGNE d'un parc (aucune base ouverte, chunks lus en lecture seule)
LEVELUP_REPO_ROOT=<racine de travail> \
  go run ./apps/go-api/cmd/replay-build --map <carte> --facts <short8>.facts.json <matchId> \
  <racine partagee>/data/cache/film_chunks/<short8>

# 2. confrontation a l'oracle API
cd .ai/V7.5/outillage/couverture_objectifs
CGO_ENABLED=0 go run . \
  -parc   <parc cuit> \
  -oracle <racine>/.ai/V7.5/replay2d/registre_film \
  -out    <sortie>
```

Les sorties `vague6_611_*.tsv|log` committees sont exactement celles de cette commande sur
`parcI3` (69 artefacts), renommees avec le prefixe du lot.

**Instruments de recherche, supprimes avant livraison** (recette pour les rejouer) :

1. `internal/analysis/replay/zz_611_fb_test.go` — sous garde `ZONE_FILM`, un film par processus :
   manches declarees et `RealRounds`, pont slot -> xuid, series `comp 22 A` / `comp 24 A` /
   `comp 23 B` par slot, emissions BRUTES d'un slot, histogramme du nombre de composants par
   enregistrement et comptage des enregistrements hors domaine. C'est lui qui a rendu les chiffres
   des §4 et §5.
2. `mesure611` — module Go de scratchpad lisant les artefacts cuits et l'oracle TSV : ratio par
   film, population des fermoirs candidats par famille (passage / retour / capture), detail par
   span, et population des suites de reprise (`-reprise`). C'est lui qui a rendu les 775,1 s du
   temoin negatif de l'item 1 et le releve du §3.1.
