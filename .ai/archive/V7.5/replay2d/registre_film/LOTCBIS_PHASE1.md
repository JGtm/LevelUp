# Lot C-bis — phase 1 : le port de `ti=13`, et la mesure d'etat

> Perimetre : CB.1.1 (port) et CB.1.2 (mesure) + Gate 1 du plan `PLAN_EXPLOITATION_REGISTRE_FILM.md`.
> Grammaire lue en phase 0 : `LOTCBIS_PHASE0.md`. Base : `38f78a46a`, branche `wt/zones-ti13`.
> Mesures du 2026-08-18. Gates : `LOTCBIS_gates.log`. Sorties : `lotC/<short8>_ti13_*.tsv`.

## 1. CB.1.1 — le port

| ti | i | composant | statut table | deser_addr | code_source |
|---|---|---|---|---|---|
| 13 | 1 | `managed-object-property-component` | `porte` | `FUN_140ce5554` | `components_managed_property.go:189` |
| 13 | 2..33 | `managed-object-player-masked-property-component` (32 lignes) | `porte` | `FUN_140ce593c` | `components_managed_property.go:208` |

`traverse.go` ne grossit que de six lignes (deux `case` de routage, 867-872) ; le corps est dans le
fichier neuf. Les 33 lignes de table ECS sont editees DANS LE MEME COMMIT (regle G1), et le
garde-rail `TestG1TableSuitLeCode` passe dans les deux sens.

**Statut `porte` et non `partiel`, et c'est motive.** La largeur depend de la donnee (4 a 36 bits
selon le tag), mais le branchement porte sur une valeur LUE DANS LE FLUX, les seize branches sont
integralement consommees, et le cas rend toujours `true` : aucun desync n'est possible. Meme
raisonnement qu'au lot C phase 1b pour les `rtpc`.

**Un seul endroit ecrit la table de largeurs.** `managedPropertyPayloadBits` vit dans le code de
production, et le decodeur des vecteurs de test l'APPELLE au lieu d'en garder une copie — la copie
de la phase 0 a ete supprimee dans ce commit. Les vecteurs figes testent donc la table portee, pas
un double qui pourrait diverger.

**L'index de joueur n'est pas dans le deser, et c'est le bon partage.** `consumeByName` ne recoit
pas l'index du composant ; le jeu le lit dans le descripteur. L'appelant, lui, a le masque :
`ManagedPropertyPlayerIndex(i)` rend `i-2` (i2 porte le joueur 0). Meme partage des roles que pour
les quatre `rtpc` de ti=10.

### Vecteurs : 33 cas sur octets reels, verts du premier coup

| test | cas | ce qu'il contraint |
|---|---|---|
| `TestTi13VecteursFiges` | 33 | tag, largeur de charge utile, quantum, position de fin |
| `TestTi13VecteursPortDeser` | 33 | ce que le HOOK publie et combien de bits le deser de PRODUCTION consomme |
| `TestTi13HookConsommeLesMemesBitsSansHook` | 32 | poser un hook ne change pas la consommation, sur les 16 tags et les 2 modes |
| `TestTi13ConvertisseursFigent` | 12 | dequantification `[-100,+100]`, enumere ou 0 vaut -1, index de joueur |
| `TestTi13RampeTag3` | 3 | la rampe du tag 3 monte a pas constant |
| `TestTi13IdentifiantsPartagesEntreFilms` | 3 | les string-ids du tag 5 sont les memes sur deux films |

Aucun film n'est lu : les octets sont recopies, les tests tournent en CI. 31 des 33 vecteurs sont
CHAINES (leur largeur est confirmee par le flux lui-meme).

### Gate de portage : `DesyncAt` sur les 12 films, avant / apres

Instrument du lot 0 (`delta_walk_witness_test.go`, 12 premiers chunks). **Aucun film ne recule, LES
DOUZE progressent** — le lot C n'en avait fait progresser que sept.

| film | mode | records avant -> apres | traversee ABOUTIE avant -> apres | gain |
|---|---|---|---|---|
| `7344d24f` | Strongholds | 33 029 -> 33 109 | 25 021 -> **25 149** | **+128** |
| `8076f97f` | KOTH | 32 970 -> 33 012 | 24 889 -> **24 943** | **+54** |
| `696a9d7c` | Strongholds | 32 713 -> 32 737 | 24 653 -> **24 693** | **+40** |
| `64e8adfa` | CTF | 39 776 -> 39 806 | 31 935 -> **31 973** | **+38** |
| `24dbb67d` | Oddball | 39 634 -> 39 659 | 30 917 -> **30 954** | **+37** |
| `530820e5` | CTF | 35 542 -> 35 569 | 26 241 -> **26 273** | **+32** |
| `53ce4390` | CTF | 38 008 -> 38 030 | 28 586 -> **28 613** | **+27** |
| `01e1f945` | KOTH | 37 967 -> 37 986 | 29 140 -> **29 162** | **+22** |
| `000d5950` | Slayer | 38 862 -> 38 878 | 30 060 -> **30 080** | **+20** |
| `0a247154` | KOTH | 33 226 -> 33 239 | 24 468 -> **24 484** | **+16** |
| `606d9844` | KOTH | 34 512 -> 34 524 | 26 642 -> **26 657** | **+15** |
| `06dfe6d9` | — | 10 607 -> 10 613 | 8 494 -> **8 502** | **+8** |

La table figee des trois films de reference a ete mise a jour avec sa justification ecrite sur
place, comme son contrat l'exige, et la chronique du lot C a ete conservee a cote de la neuve.

**Le gain reste modeste, et il faut dire pourquoi** : une traversee n'aboutit que si TOUS les
composants annonces sont portes, et le trafic est domine par le bipede. **Ce qui est neuf, c'est
que le gain touche LES DOUZE films, y compris le temoin Slayer ou ti=13 est pourtant quasi muet
(5 records d'i1)** — signe que les records reparees ne sont pas seulement ceux de la bande ti=13,
mais aussi ceux qui la SUIVENT dans le paquet et que la marche sequentielle atteint desormais.

### Gates joues

`EXIT_VET=0` · `EXIT_TEST_FILMDEC=0` (filmdec, replay, objectiveevents) · `EXIT_BUILD_CGO=0`
(`go build ./...`, CGO actif) · `EXIT_LINT=0` (`golangci-lint --new-from-merge-base=origin/main
./...`, 0 issue) · `TestG1TableSuitLeCode` et `TestG3TableSuitLeDocument` verts.

## 2. CB.1.2 — la mesure d'etat

Seuils ecrits AVANT la mesure, dans `ti13_etat_test.go` (constantes `ti13Seuil*`), jamais ajustes
apres. Instrument : la boucle de PRODUCTION rejouee composant par composant sur les records ancres
(meme voie que le lot C), l'index de joueur reconstitue depuis l'ordre du masque. Six films, cinq
modes. Temoins d'ancrage publies a chaque passe.

| film | mode | records ti=13 ancres / consommes | i1 scalaire | i2..i33 par joueur | fantome | purete ti=4 |
|---|---|---|---|---|---|---|
| `7344d24f` | Strongholds | 35 687 / 10 590 | 3 635 | 11 464 | 26 899 | 0,68 % |
| `696a9d7c` | Strongholds | 36 270 / 8 285 | 4 086 | 7 084 | 26 217 | 1,10 % |
| `01e1f945` | KOTH | 6 483 / 4 891 | 3 423 | 3 603 | 14 236 | 0,73 % |
| `0a247154` | KOTH | 2 841 / 1 922 | 265 | 4 752 | 31 539 | 1,05 % |
| `530820e5` | CTF | 2 855 / 1 740 | 614 | 1 787 | 18 800 | 0,21 % |
| `000d5950` | Slayer (temoin) | 652 / 263 | **20** | 556 | 7 498 | 1,33 % |

L'ecart entre records ancres et records consommes (30 a 75 %) est la contamination de bande deja
chiffree par la sonde F5 : un record qui annonce un index que ti=13 ne possede pas ne se traverse
pas. Le temoin Slayer rend 20 valeurs d'i1 : la ou il n'y a pas d'objectif, le canal se tait.

### (a) TAG 5 — l'identifiant de chaine : NON JUGEABLE au seuil, mais la STABILITE est TENUE

| film | slot | identifiant | emissions |
|---|---|---|---|
| `7344d24f` | 1525 / 1526 / 1527 | `0x67F43AC3` / `0xD690D6B4` / `0xF2F9EB27` | 1 chacun |
| `696a9d7c` | 1525 / 1526 / 1527 | `0x67F43AC3` / `0xD690D6B4` / `0xF2F9EB27` | 1 chacun |
| `01e1f945` | 1395 / 1396 / 1397 | `0x6050ABD7` / `0x3327C7DA` / `0xF2F9EB27` | 1 chacun |
| `0a247154` | 1566 / 1567 | `0x3327C7DA` / `0x444AC900` | 1 / 2 |
| `530820e5` | 1376 / 1380 / 1381 | `0x30CE8ECB` / `0x14CFA482` / `0x1340FA63` | 1 chacun |

**VERDICT (a) : NON JUGEABLE au seuil ecrit** — aucun slot n'atteint les 10 emissions exigees.
L'identifiant est emis UNE FOIS, a la mise en place de la propriete, ce qui est coherent avec la
structure lue en phase 0 (le nom vit dans l'image-cle). Le seuil de volume n'est pas abaisse.

**Mais la clause de STABILITE, elle, est TENUE, et c'est le resultat de fond.** La carte
slot -> identifiant est **IDENTIQUE sur les deux Strongholds, 3 slots sur 3** (1525, 1526, 1527),
et Strongholds se joue sur **trois** zones. La coherence par slot vaut 100 % (une seule valeur
distincte par slot) — trivialement, puisqu'il n'y a qu'une emission, mais sans aucune exception sur
les cinq films. `0xF2F9EB27` se retrouve meme sur un KOTH : le vocabulaire deborde le mode.

**Ce que ce volet NE FAIT PAS, et pourquoi.** L'appariement de cette cle a une zone du CATALOGUE
(`replay.AttributeZones`, positions de bipedes) n'est pas fait : **`internal/analysis/replay`
importe `filmdec`, donc `filmdec` ne peut pas importer `replay`** — le pont geometrique appartient
au paquet `replay`, c'est-a-dire a la phase 2. Ni le pont slot de bipede vers joueur ni l'equipe du
capteur ne sont joignables depuis `filmdec` : `BipedPosition.Slot` migre a chaque reapparition et
n'est pas attribue, et `game-engine-team-mapping` lit ses bits sans les publier (aucun hook). La
clause « le proprietaire deduit concorde avec l'EQUIPE du capteur » est donc **NON MESUREE**, et
c'est ecrit plutot que contourne.

### (b) TAG 3 — la rampe contre les captures

| film | mode | rampes | amplitude mediane | captures couvertes | temoin +20 s | **niveau du hasard** | verdict |
|---|---|---|---|---|---|---|---|
| `7344d24f` | Strongholds | 84 | 82 822 (0,99 unite) | **69/71 = 97,2 %** | 46,5 % | **56,7 %** | **TENU** |
| `696a9d7c` | Strongholds | 85 | 82 693 | **73/77 = 94,8 %** | 53,2 % | **60,8 %** | NON TENU |
| `530820e5` | CTF | 21 | 33 556 | 24/56 = 42,9 % | 35,7 % | 17,4 % | NON TENU |
| `01e1f945` | KOTH | 60 | 81 090 | pas d'oracle | — | — | — |
| `0a247154` | KOTH | 0 | — | pas d'oracle | — | — | — |

**Clause principale : TENUE et largement** sur les deux Strongholds (97,2 % et 94,8 % pour un seuil
de 80 %). **Clause du temoin : tenue sur un film, manquee de 5,8 points sur l'autre.**

**Et il faut dire pourquoi, sans deplacer le seuil.** Le niveau du hasard vaut 56,7 % et 60,8 % :
une capture tombe pres d'un sommet de rampe par pur hasard plus d'une fois sur deux, parce que le
canal produit 84 et 85 rampes en dix minutes. La clause « temoin sous la MOITIE du reel » exige
donc un temoin sous 47-49 % alors que le hasard seul le place a 57-61 %. **Elle est inatteignable
par construction sur ce canal** — exactement le defaut que le lot C avait deja releve pour
`radial-progress` (temoin 36,6 % pour un seuil de 20 %, hasard a 46,1 %). Le signal, lui, est
franc : 97 % contre 57 % de hasard, et le temoin decale est AU niveau du hasard, donc non
informatif par construction.

Sur CTF le negatif est REEL et non un artefact de seuil : 42,9 % pour un hasard de 17,4 %. Le canal
reagit a quelque chose, mais pas aux prises de drapeau — attendu pour une jauge de zone.

### (c) TAG 4 et enumeres — lequel est l'ETAT ?

| film | candidat | valeurs / slots | **enumerabilite** | captures couvertes | temoin | hasard | verdict |
|---|---|---|---|---|---|---|---|
| `7344d24f` | TAG 4 (R32) | 150 / 10 | **6/6 = 100 %** | 71/71 = 100 % | 60,6 % | **93,1 %** | NON TENU |
| `696a9d7c` | TAG 4 (R32) | 148 / 9 | **6/6 = 100 %** | 77/77 = 100 % | 53,2 % | **96,6 %** | NON TENU |
| `530820e5` | TAG 4 (R32) | 83 / 7 | **2/2 = 100 %** | 54/56 = 96,4 % | 57,1 % | **63,0 %** | NON TENU |
| `01e1f945` | TAG 4 (R32) | 209 / 8 | **2/2 = 100 %** | pas d'oracle | — | — | — |
| `0a247154` | TAG 4 (R32) | 218 / 5 | **1/1 = 100 %** | pas d'oracle | — | — | — |
| tous | TAG 1 (enumere) | 1 a 19 | non jugeable | 0 % | 0 % | ~0 % | NON TENU |

**La clause d'ENUMERABILITE est TENUE partout : 100 % des slots juges portent au plus 8 valeurs
distinctes.** Le tag 4 est donc bien un ETAT enumerable, et c'est le seul candidat qui le soit.

**Mais la clause temporelle n'est pas jugeable** : avec 135 a 138 changements sur dix minutes, la
fenetre de +/- 2 s couvre 93 a 97 % du match. Une couverture de 100 % ne dit alors rien. Le critere
lui-meme est vide sur ce canal, et c'est ecrit plutot que presente comme un succes.

Le **tag 1 (enumere) est REFUTE** : 1 a 19 valeurs sur tout un film, 0 ou 1 changement. Ce n'est pas
le canal d'etat.

### (d) UNICITE — une seule zone active a la fois

| film | mode | rampes / slots | temps a une seule rampe | verdict |
|---|---|---|---|---|
| `01e1f945` | KOTH | 60 / 2 | **293 244 / 293 244 ms = 100,0 %** | **TENUE** |
| `530820e5` | CTF | 21 / 2 | 175 174 / 224 125 ms = 78,2 % | NON TENUE |
| `7344d24f` | Strongholds | 84 / 6 | 35 197 / 554 808 ms = 6,3 % | NON TENUE |
| `696a9d7c` | Strongholds | 85 / 5 | 23 896 / 516 496 ms = 4,6 % | NON TENUE |
| `0a247154` | KOTH | 0 / 2 | — | sans objet |

**C'est le resultat le plus net de la mesure, et il faut le lire dans le bon sens.** Sur le seul
film KOTH ou i1 parle, **une seule zone rampe a la fois, 100,0 % du temps, sur 60 rampes**. En
Strongholds, plusieurs rampent en parallele — ce qui n'est pas un echec du canal mais **le mode
lui-meme** : Strongholds se joue sur trois zones capturables simultanement. La clause discrimine
donc exactement ce qu'elle devait discriminer, a condition de la juger par mode.

C'est aussi la correction que le lot C reclamait : sa clause KOTH etait fausse au niveau du
NAVPOINT (6 a 34 marqueurs simultanes) ; portee au niveau de l'OBJET ti=13, elle rend 100 %.

### (d bis / e) Les valeurs par joueur (i2..i33)

| film | mode | echantillons | tag dominant |
|---|---|---|---|
| `0a247154` | KOTH | 4 752 | **t7 (R(24) quantifie) : 67,8 %** |
| `01e1f945` | KOTH | 3 603 | **t7 (R(24) quantifie) : 50,2 %** |
| `7344d24f` | Strongholds | 11 464 | t0 (vide) : 63,7 % · t2 : 18,8 % |
| `696a9d7c` | Strongholds | 7 084 | t0 (vide) : 49,4 % · t2 : 16,2 % |
| `530820e5` | CTF | 1 787 | t8 : 13,3 % · t15 : 12,2 % (disperse) |
| `000d5950` | Slayer | 556 | t8 : 12,6 % (disperse) |

**Decouverte : en KOTH le canal par joueur porte un FLOTTANT QUANTIFIE (tag 7) sur la moitie a
deux tiers de son trafic.** C'est une valeur continue par joueur, sur `[-100, +100]`, exactement le
miroir de la jauge scalaire du tag 3 — la forme d'une contribution individuelle a la colline. En
Strongholds au contraire le tag dominant est le **tag 0, la valeur VIDE** (49 a 64 %), et les tags
sont disperses : c'est la signature de la contamination d'ancrage que la phase 0 avait chiffree a
0 % de chainage sur ces memes composants. **Les deux lectures concordent, par deux voies
independantes.**

**(e) CTF** : les socles ne ressortent pas. Le trafic par joueur y est disperse (13,3 % au tag
dominant) et la jauge scalaire ne suit pas les prises (42,9 % pour 17,4 % de hasard). **Negatif
ecrit.**

## 3. GATE 1 — verdict

> Enonce : « un canal (a)+(b ou c) tient => forme PROPOSEE de `zoneStates` ; sinon negatif ecrit
> par canal. »

**GATE 1 : NON ATTEINT.** Le volet (a) n'est pas TENU a son seuil ecrit — il est NON JUGEABLE,
faute de volume (1 emission par slot contre 10 exigees) — et la conjonction exigee tombe donc, quel
que soit le sort de (b) et (c). Le seuil n'est pas abaisse, la forme de `zoneStates` n'est pas
ecrite.

**Ce que la mesure etablit malgre tout, et qui n'est pas rien :**

1. **Le tag 3 est bien la jauge de capture de zone** : 97,2 % et 94,8 % des captures ont un sommet
   de rampe a moins de 2 s, contre 57-61 % de hasard.
2. **Le tag 4 est un ETAT ENUMERABLE** : 100 % des slots juges portent au plus 8 valeurs
   distinctes, sur cinq films et quatre modes. C'est le seul canal du corpus qui satisfasse ce
   critere — ni `boundary-color` (996 quadruplets) ni `boundary-visibility` (32 bits actifs) ne
   l'avaient fait au lot C.
3. **La cle de nommage existe et elle est STABLE** : trois slots, trois identifiants de chaine,
   identiques sur les deux Strongholds — pour un mode qui se joue sur trois zones.
4. **En KOTH, une seule zone est active a la fois, 100,0 % du temps** sur 60 rampes.
5. **En KOTH, le canal par joueur porte un flottant quantifie** sur 50 a 68 % de son trafic.

**Ce qui bloque, nommement :**

- **L'appariement a une zone du catalogue est hors d'atteinte depuis `filmdec`** (cycle d'import :
  `replay` importe `filmdec`). Il appartient au paquet `replay`, donc a la phase 2.
- **Deux clauses temporelles sont vides par construction sur ce canal** : le temoin de (b) exige un
  taux sous 47-49 % la ou le hasard seul vaut 57-61 % ; la fenetre de (c) couvre 93-97 % du match.
  Le defaut est le meme qu'au lot C et il est desormais mesure deux fois.
- **L'equipe du capteur n'est pas joignable depuis `filmdec`** : `game-engine-team-mapping` lit ses
  bits sans les publier, et le pont slot de bipede vers joueur n'est pas etabli.

## 4. Statut des items

- [x] **CB.1.1** — port des 2 desers, 33 lignes de table, 33 vecteurs verts + 3 tests neufs,
  `DesyncAt` ameliore sur LES DOUZE films.
- [x] **CB.1.2** — mesure jouee sur 6 films et 5 modes, verdict par volet ci-dessus, denominateurs
  et niveaux du hasard publies.
- [!] **Gate 1** — NON ATTEINT : (a) non jugeable a son seuil de volume. Details section 3.
- [ ] **Phase 2** — NON FAITE, et hors de portee de `filmdec` pour le volet geometrique.

## 5. Cout machine (D17)

Un film par processus, avant-plan. Mesure d'etat : **15 a 64 s par film** (6 films). Temoin
`DesyncAt` : environ 12 s par film (12 films, deux passes). Pic memoire mesure en phase 0 sur le
meme ancrage : 16 Mo, plafond de 3 Go jamais approche.

## 6. Decouvertes (hors perimetre — notees, NON traitees)

1. **Le tag 4 est le premier canal ENUMERABLE du corpus** (au plus 8 valeurs par slot, 100 % des
   slots juges, 5 films). Les trois canaux du lot C avaient tous echoue sur ce critere. Si un etat
   de zone existe cote film, c'est celui-la — mais sa preuve demande un oracle qui ne soit pas une
   coincidence temporelle.
2. **Deux clauses de gate sont VIDES par construction quand le canal est bavard.** Le niveau du
   hasard devrait etre le DENOMINATEUR de tout temoin futur, et une fenetre devrait etre bornee par
   la densite des evenements, pas fixee a l'avance. Deuxieme occurrence apres le lot C.
3. **`game-engine-team-mapping` (ti=0 i0) lit ses bits et les jette** : `consumeGameEngineTeamMapping`
   consomme R(8)+R(9)+R(9)+R(8)+R(8)+R(8) puis un R(4) par equipe presente, sans hook. C'est la
   source d'equipe la plus directe du film, et elle est a un hook pres.
4. **`0xF2F9EB27` apparait en Strongholds ET en KOTH** (slots 1527 et 1397) : le vocabulaire des
   identifiants de chaine deborde le mode. Une table `string_id` resolue nommerait ces objets pour
   tous les modes d'un coup.
5. **`0a247154` n'a que 265 valeurs d'i1 et 0 rampe**, alors que `01e1f945` en a 3 423 et 60. Deux
   KOTH, deux comportements opposes du meme canal : la carte ou la variante de mode decide, pas le
   mode. A verifier avant de cabler quoi que ce soit.
