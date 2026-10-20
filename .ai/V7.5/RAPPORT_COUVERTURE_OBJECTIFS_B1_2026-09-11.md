# Lot 6.7 phase B1 — couverture des calques d'objectif : le lot correctif Go (2026-09-11)

> Worktree `LevelUp-wt-couverture-objectifs`, branche `wt/couverture-objectifs`.
> Preuve d'entree de chaque item : `.ai/V7.5/AUDIT_COUVERTURE_OBJECTIFS_2026-09-10.md` (§10
> causes, §11 gates) et `.ai/V7.5/RAPPORT_ODDBALL_FANTOMES_2026-09-10.md` (§6, decouvertes).
> **Aucune base DuckDB n'a ete ouverte**, aucun artefact du parc n'a ete recuit : tous les
> chiffres viennent de cuissons HORS LIGNE (`cmd/replay-build --facts`, racine de travail dans
> un dossier de scratchpad) confrontees a l'oracle API par
> `.ai/V7.5/outillage/couverture_objectifs`.

## 0. Verdict en six lignes

1. **Le crane atteint 96,6 % de son oracle** (0,891 apres l'item 1, 0,822 avant le lot) : un
   train de n tics publie desormais n largeurs de tic, et la largeur se MESURE sur le film.
2. **Le drapeau perd 75,5 % de son exces** (1 125,2 s -> 275,7 s) : le lacher ferme aussi un
   portage deja ferme. Le gate a 1,05 n'est PAS atteint (1,130) et la cause du residu est
   nommee, mesuree et hors du perimetre de cet item.
3. **La manche fantome est fermee** : `e60aaf06` passe de 24 actions rattachees sur 154 a
   **154 sur 154**, et de 10 captures de zone a **38 pour 38 a l'oracle**.
4. **Le calque des actions se tait au-dela de huit joueurs**, avec son compteur
   (`refusedByRoster`) et son avertissement.
5. **La borne de deroulage est recalibree sur l'oracle** : 100 000 -> 16. Les trois compteurs
   aberrants de l'audit tombent a leur valeur reelle, et les `assists` explosives de trois
   films retrouvent la feuille de match **a l'unite**.
6. **L'hypothese `assists` de l'audit §10 est CONFIRMEE** : 40 806 pulses faux sur trois films,
   refutes par la feuille de match re-exportee.

---

## 1. Etat des items

| # | item | statut | gate |
|---|---|---|---|
| 1 | completion d'identite par residu de manche | `[x]` | tenu (commit `20f138e30`, re-mesure §2) |
| 2 | demi-fenetre de tic sur le crane | `[x]` | tenu |
| 3 | drapeau ferme sur le lacher | `[x]` livre, gate **partiellement** tenu | 3 criteres sur 6 ; §4 |
| 4 | manche fantome | `[x]` | tenu, et au-dela |
| 5 | garde d'effectif du calque `objectives` | `[x]` | tenu |
| 6 | bornes de deroulage par joueur et par comp | `[x]` | tenu |

Commits, dans l'ordre :

| item | SHA | message |
|---|---|---|
| 1 | `20f138e30` | completion d'identite par RESIDU DE MANCHE (agent precedent) |
| 2 | `1e32db6c4` | DEMI-FENETRE DE TIC aux deux bornes du portage du crane |
| 3 | `7ad82ea13` | le LACHER ferme aussi un portage DEJA FERME |
| 4 | `bb06cce5a` | la MANCHE FANTOME de `e60aaf06`, un trou VIDE dans la chaine |
| 5 | `ebd012e3b` | GARDE D'EFFECTIF du calque des actions d'objectif |
| 6 | `f22474816` | BORNE DE DEROULAGE recalibree sur l'oracle, 100 000 -> 16 |

## 1.1 Le protocole de mesure, et ce qui le rend rejouable

Quatre parcs d'artefacts cuits HORS LIGNE, chacun de 68 films, dans le scratchpad de session :

| parc | binaire | contenu |
|---|---|---|
| `parc0` | HEAD de la branche a l'ouverture (item 1 inclus) | **la reference AVANT** de tous les items 2 a 6 |
| `parc23` | + items 2 et 3 | 15 films recuits (4 Oddball + 11 CTF) |
| `parc4` | + item 4 | parc complet |
| `parc7` | + items 5 et 6 | parc complet — **la reference APRES** |

`parc0` reproduit l'audit du 2026-09-10 a la troisieme decimale : ratio drapeau **1,530**
(audit 1,531), **197** periodes, **69** joueurs au-dessus de leur oracle sur 73 mesures. Le
calibrage de l'instrument (`vague6_calibrage.log`) est vert : 11 films sur 11 sur le temoin des
spans, 1 249,0 s sur le temoin d'oracle Oddball, trois porteurs a la decimale.

Sorties APRES committees sous `.ai/V7.5/replay2d/registre_film/vague6_b1_*.tsv`.

---

## 2. Item 1 — completion d'identite par residu de manche (rappel, `20f138e30`)

Livre par l'agent precedent ; sa mesure est reportee ici pour que la chaine du lot se lise
d'un bloc, et elle est **re-verifiee** sur `parc0` (cuit au HEAD, donc item 1 inclus) :

| grandeur | avant l'item 1 | apres |
|---|---|---|
| `43716616` / 2533274978052136 | 0,0 s | **61,2 s** (oracle 62,3 ; exige >= 55) |
| `43716616` `coverage.skullCarries.noBridge` | 2 | **0** |
| crane, 4 films Oddball | 1 036,7 s / 1 249,0 s (0,830) | **1 113,0 s (0,891)** |
| joueurs au-dessus de leur oracle | 0 | **0** |
| `d9781168` `51ebbc0f` `c88ec007` | — | artefacts identiques a l'octet |
| `7fce3219` `cde26226` (CTF multi-manche) | — | identiques a l'octet, `noSlot = 0` |

Controle refait ce jour sur `parc0` : `vague6_couverture_crane.tsv` donne bien 1 113,0 s pour
1 249,0 s, 0 joueur au-dessus, et `vague6_identite.tsv` donne `noSlot = 0` sur `7fce3219` et
`cde26226`.

---

## 3. Item 2 — demi-fenetre de tic sur le crane

**Cause verifiee sur pieces.** Releve de la cadence des tics sur `43716616` (film decode,
10 trains, 213 tics) : histogramme des ecarts intra-train **900 ms x 13, 1 000 ms x 186**. Le
tic tombe a 1 Hz, et un train borne par son premier et son dernier tic couvre donc (n-1)
largeurs quand il temoigne de n secondes de possession.

**Correctif.** `skullTickWidthFrames` mesure la largeur du tic sur les trains du film lui-meme
(mediane des ecarts intra-train, plafonnee par `skullTickGapMS`) ; `skullHalfTickFrames` en
tire `(w-1)/2` images a poser de part et d'autre. La largeur n'est pas une constante ecrite
dans le code.

**Pourquoi `(w-1)/2` et non `w/2`.** L'intervalle publie est FERME : `[t0,t1]` compte
`t1-t0+1` images. `w/2` de chaque cote faisait publier n largeurs de tic PLUS une image et
portait **2 joueurs sur 28** au-dessus de leur oracle (+0,1 s chacun, mesure). `(w-1)/2` est la
plus grande fenetre symetrique en images entieres qui ne depasse jamais n largeurs.

**Gate (4 films Oddball, `parc0` -> `parc23`).**

| film | publie avant | publie apres | oracle | ratio avant | ratio apres |
|---|---|---|---|---|---|
| `43716616` | 200,6 s | **213,4 s** | 217,7 s | 0,921 | **0,980** |
| `51ebbc0f` | 238,0 s | **256,1 s** | 265,8 s | 0,895 | **0,964** |
| `c88ec007` | 320,1 s | **348,8 s** | 361,8 s | 0,885 | **0,964** |
| `d9781168` | 354,3 s | **388,6 s** | 403,7 s | 0,878 | **0,963** |
| **total** | **1 113,0 s** | **1 206,9 s** | **1 249,0 s** | **0,891** | **0,966** |

- exige `> 82,2 %` (rapport 6.2) et `> 0,891` (post-item 1) : **tenu**.
- **0 joueur au-dessus de son oracle**, avant comme apres (28 joueurs mesures) : **tenu**.
- periodes 100 avant, 100 apres : la fenetre n'en cree ni n'en supprime.

**Tests.** `skull_carries_halfwindow_test.go`, 4 tests rouges a la compilation avant le code.
**Mutation** : `TestSkullCarriesSansCadenceMesurableNeDeplaceRien` — un film dont tous les
trains tiennent en UN tic ne donne aucune cadence a mesurer et ses bornes ne bougent pas ; si
la largeur etait une constante, ce test rougirait.

**Goldens.** Aucun golden de CI ne change (`000d5950` est un Slayer, entree de crane vide).
Trois assertions de test portaient les bornes brutes ; elles portent desormais
`testHalfTickFrames` (499 images sur l'axe de test a 1 ms/image), le delta est explique en
commentaire. `TestSkullCarriesCarrierAbsent` et `TestSkullCarriesOpenAtAxisEnd` passent
INCHANGES : le rognage a la presence et le seuil d'ouverture absorbent la fenetre.

---

## 4. Item 3 — le lacher ferme aussi un portage deja ferme

**Cause verifiee sur pieces.** `closeByFreeLives` (`flag_objects.go:263` a l'audit) ne traitait
que les portages `!closed` — ceux que RIEN ne bornait, soit **UN SEUL span** sur tout le parc
(`b8a44fe8`, 18,7 s). Les 1 129,3 s de faux etaient donc dans les portages FERMES, par une
mort, une capture ou une reprise survenue APRES un lacher que rien ne datait. Le canal capable
de le dater etait lu, compte et publie (`objectLives = 37` sur `16ea3668`) et ne fermait rien :
`closedByObject` valait **zero sur les onze films**.

**Correctif.** La garde `if raws[i].closed { continue }` tombe. Le plus petit fermoir gagne, et
c'est structurel : `flagFreeDropInside` ne retient qu'une naissance strictement interieure a
`]t0, t1[`, donc tout instant rendu est deja plus petit que la borne en place ; les vies libres
etant triees par instant, la premiere trouvee est la plus precoce. Une CAPTURE dementie par un
objet au sol anterieur cesse d'etre une capture. **La sous-population validee ne s'elargit
pas** : seules servent les vies nees aux pieds d'un porteur et hors socle.

**En-tete de `flag_carries.go` reecrit** (anti-patron n° 9) : quatre faits de fermeture
deviennent CINQ ; le paragraphe « le lacher volontaire n'est donc PAS borne » est remplace par
ce qui est vrai depuis ce lot, et ce qui reste non borne est nomme.

**Gate (11 films CTF a calque, `parc0` -> `parc23`).**

| film | periodes | publie avant | publie apres | oracle | ratio avant | ratio apres |
|---|---|---|---|---|---|---|
| `16ea3668` | 16 -> 28 | 212,8 s | 189,0 s | 131,0 s | 1,624 | 1,443 |
| `58864b3c` | 10 -> 19 | 159,5 s | **113,7 s** | 110,1 s | 1,449 | **1,033** |
| `7fce3219` | 22 -> 85 | 261,7 s | **188,5 s** | 173,9 s | 1,505 | **1,084** |
| `8bc6074f` | 18 -> 51 | 348,4 s | **214,7 s** | 209,3 s | 1,665 | **1,026** |
| `a0c36016` | 24 -> 49 | 263,3 s | **166,4 s** | 156,5 s | 1,682 | **1,063** |
| `b8a44fe8` | 25 -> 60 | 696,2 s | 551,9 s | 401,1 s | 1,736 | 1,376 |
| `bc60b4d9` | 20 -> 29 | 163,1 s | **128,7 s** | 120,2 s | 1,357 | **1,071** |
| `bf5ced1b` | 5 -> 10 | 49,6 s | 39,0 s | 33,1 s | 1,498 | 1,178 |
| `cde26226` | 25 -> 79 | 672,9 s | **507,1 s** | 480,4 s | 1,401 | **1,056** |
| `f8efc5ca` | 23 -> 34 | 288,8 s | **200,0 s** | 196,0 s | 1,473 | **1,020** |
| `4ecdf3e7` | 9 -> 25 | 130,1 s | **97,9 s** | 109,6 s | 1,187 | **0,893** |
| **total** | **197 -> 469** | **3 246,4 s** | **2 396,9 s** | **2 121,2 s** | **1,530** | **1,130** |

Distribution du ratio par joueur (73 joueurs a oracle non nul) : mediane **1,380 -> 1,033**,
p25 1,016, p75 1,088, max 61,2 (l'unique cas nomme ci-dessous).

| critere du gate (audit L5) | attendu | mesure | verdict |
|---|---|---|---|
| ratio des 11 films | <= 1,05 | **1,130** | `[!]` non tenu |
| joueurs au-dessus de leur oracle a 0,5 s pres | 0 | **45** (59 avant) | `[!]` non tenu |
| nombre de periodes | >= 197 | **469** | `[x]` |
| aucun joueur ne perd du temps qu'il avait a l'oracle | 0 nouveau joueur sous 0,8 | **2 avant, LES MEMES 2 apres** | `[x]` |
| `closedByObject` cesse d'etre 0 | != 0 | **20 / 50 / 15** sur `16ea3668` / `b8a44fe8` / `58864b3c` | `[x]` |
| `16ea3668` / 2535417044536883 | <= 1,5 s | **55,1 s** (56,7 s avant) | `[!]` non tenu |

Grandeurs complementaires, qui disent l'ampleur reelle du gain :

| grandeur | avant | apres |
|---|---|---|
| exces total sur les 11 films | 1 125,2 s | **275,7 s** (-75,5 %) |
| joueurs au-dessus de plus de 2 s | 55 | **16** |
| joueurs au-dessus de plus de 5 s | 48 | **9** |

**Pourquoi le gate n'est pas atteint, et ce n'est pas une opinion.** Le residu de 275,7 s est
concentre : 9 joueurs sur 73 portent 9 depassements de plus de 5 s, et deux films en font 209 s
(`b8a44fe8` +150,8 s, `16ea3668` +58,0 s). Le cas nomme au gate — `16ea3668` /
2535417044536883, un unique span de 56,7 s pour 0,9 s a l'oracle — passe a 55,1 s parce
qu'**aucune vie libre ne nait aux pieds de ce porteur pendant son portage** : le canal de
l'objet n'a rien a dire de ce lacher-la. Sa cause est ailleurs, et elle est instruite au §8
(decouverte D1) : rien ne ferme un portage sur la prise d'un AUTRE joueur du MEME drapeau, alors
que `dropsWithheld` prouve que le passage de main a main existe (14 fins retenues sur
`16ea3668` avant ce lot). Elargir la sous-population des vies libres — la seule autre voie —
est explicitement interdit par le perimetre de l'item.

**Tests.** `flag_objects_drop_test.go`, 3 tests rouges avant le code. **Mutation** :
`TestUneVieLibreFermeUnPortageDejaFermeParLaMort` — retablir `if raws[i].closed { continue }` le
rougit, et lui seul. Temoins negatifs : une vie nee APRES la fermeture ne change rien, plus les
deux temoins deja en place (vie loin du porteur, vie nee a un socle).

---

## 5. Item 4 — la manche fantome de `e60aaf06`

**Cause verifiee sur pieces** (releve du film : 864 enregistrements, 72 morts) :

| manche declaree | enregistrements joueur | part | slots | intervalle |
|---|---|---|---|---|
| 0 | 313 | 100 % | 8 | [2 991, 411 036] |
| 1 | **0** | — | 0 | **absente** |
| 2 | 44 | **14 %** | 8 | [79 010, 172 093] |

La manche 2 est MATERIELLE (14 % >= `statMinRoundRecordShare` = 10 %) — elle tombe d'ailleurs
dans la bande 7-15 % que le corpus de calibrage declare indecidable — et son intervalle tient
ENTIEREMENT dans celui de la manche 0. La tolerance d'UNE manche vide dans la chaine
(`statMaxEmptyRoundRun`), ecrite pour une manche JOUEE mais trop courte, laissait la chaine
sauter par-dessus le vide et atteindre cet ancrage.

**Le cout ne passait pas par les series, mais par l'identite** — et c'est ce que l'audit ne
pouvait pas voir. Les points de la manche 2 sont deja ecartes par `ChronologicalTotal` (ils
precedent la fin de la manche 0). Ce qui coutait, c'est que `SlotIdentityByRound` tire ses
manches de `RealRounds` : avec trois manches, l'identite se resolvait PAR MANCHE ; la manche 2
n'a aucun slot emetteur du compteur de morts ; et `RoundIdentity.At` envoyait dans cette manche
vide tout evenement date apres 79 076 ms, soit la quasi-totalite du match.

**Correctif.** `presentRounds` rend les manches qui existent dans les enregistrements (au moins
UN, joueur ou equipe) ; `contiguousRounds` refuse de TOLERER dans le trou une manche qui
n'existe pas. En TETE de chaine, une manche absente reste toleree — un film qui numerote a
partir de 1 ne declare rien en manche 0, c'est un decalage et non un trou.

**Gate (`parc0` -> `parc4`).**

| grandeur | avant | apres | exige |
|---|---|---|---|
| `e60aaf06` manches (couverture) | 3 | **1** | concorde avec le fil de score (1) |
| `e60aaf06` actions rattachees | 24 / 154 | **154 / 154** | — |
| `e60aaf06` `noSlot` | 130 | **0** | <= 8 |
| `e60aaf06` `zone_captures` | 10 / 38 | **38 / 38** | >= 36 |
| `e60aaf06` `zone_secures` | 2 / 11 | **11 / 11** | — |
| `e60aaf06` actions d'objectif publiees | 12 | **49** | oracle 49 |
| `32d9a94f` `396cfc92` `572e236b` `81c02726` | 1,000 | **1,000 inchange** | aucune regression |
| concordance manches / fil de score, 68 films | 3 desaccords | **1 desaccord** | concordance |

**Un film de plus change, et c'est un gain** : `72b0a25e` (Slayer:Arena, un mode SANS manche,
deja nomme dans `round_bounds.go` parmi les trois films a etiquetage faux) passe de 3 manches a
1. Diff structurel complet de son artefact : la SEULE valeur qui bouge est
`coverage.score.rounds`.

**Ce qui reste** : `a4083bd2` (Slayer) declare toujours 3 manches pour 1 au fil de score. Ses
trois manches EXISTENT dans les enregistrements, la garde ne les touche donc pas. Il ne publie
aucun calque d'objectif : le desaccord ne coute rien de mesurable. Consigne au §8 (D3).

**Tests.** `statborg_rounds_absent_test.go`, 3 tests rouges avant le code, le premier
reproduisant la forme EXACTE du film. **Mutation** :
`TestRealRoundsGardeUneMancheCourteQuiEXISTE` — la meme forme avec une manche 1 de TROIS
enregistrements garde la chaine ouverte ; si la garde regardait la matiere ou la coherence
plutot que la PRESENCE, elle rougirait. Contre-epreuve :
`TestRealRoundsTolereUneManche0AbsenteEnTeteDeChaine`.

---

## 6. Item 5 — la garde d'effectif du calque des actions

**Preuve d'entree** (audit §6, L1). Le calque du PORTAGE avait sa garde (`IsFlagFilm`) et se
taisait sur les films BTB ; le calque des ACTIONS n'en avait aucune et publiait 129 actions sur
les trois films BTB du parc, dont **65 prises de drapeau sur `4f77afc1` la ou l'oracle en compte
QUATRE** pour les 36 lignes de sa feuille.

**Correctif.** `objectiveevents.RosterFitsStatborg(n)` + `StatPlayerSlots` (derive des
constantes de format : `(statSlotMax - statTeamSlotMax) / 2` = 8). `replaybuild.identifiedEvents`
refuse le film entier, rend `refusedByRoster` en couverture et emet un `slog.WarnContext`.

**CE QUI SE COMPTE EST UN SIEGE, PAS UNE PERSONNE — et c'est la mesure qui l'a impose.** La
premiere ecriture comptait les LIGNES de la feuille de match. Cuisson hors ligne du parc avec
cette version : **huit** films refuses au lieu de trois, dont cinq films d'arene PARFAITEMENT
mesures.

| film | lignes de feuille | dont bots | dont arrives en cours | **sieges** | verdict lignes | verdict sieges |
|---|---|---|---|---|---|---|
| `8bc6074f` (CTF) | 10 | 1 | 2 | **7** | refuse (198 actions) | publie |
| `396cfc92` (Strongholds) | 10 | 1 | 1 | **8** | refuse (152 actions) | publie |
| `572e236b` (Strongholds) | 10 | 1 | 2 | **7** | refuse (143 actions) | publie |
| `4ecdf3e7` (CTF neutre) | 10 | 1 | 2 | **7** | refuse (86 actions) | publie |
| `bf5ced1b` (CTF) | 9 | 1 | 1 | **7** | refuse (54 actions) | publie |
| `4f77afc1` (BTB:CTF) | 36 | 8 | 10 | **18** | refuse | **refuse** |
| `5676a9ba` (BTB:TC) | 31 | 4 | 6 | **21** | refuse | **refuse** |
| `879a4dba` (BTB:CTF) | 26 | 1 | 2 | **23** | refuse | **refuse** |

633 actions EXACTES auraient ete retirees, dont les 152 de `396cfc92` et les 143 de `572e236b`,
deux films a 1,000 joueur par joueur. Un siege est occupe par au plus une personne a la fois :
un bot remplit une place liberee, un remplacant en prend une. Les deux populations sont alors
separees d'un facteur superieur a DEUX (7-8 contre 18-23) et le plafond de format tombe
exactement a la frontiere haute des arenes.

**Gate.**

| film | actions publiees avant | apres | `refusedByRoster` |
|---|---|---|---|
| `4f77afc1` | 253 | **0** | 189 |
| `879a4dba` | 99 | **0** | 99 |
| `5676a9ba` | 178 | **0** | 178 |
| les 65 autres films | — | **inchangees** | 0 (champ absent du document) |

**`omitempty` ET PAS DE MONTEE DE SCHEMA.** Le compteur vaut zero sur 65 des 68 artefacts.
Ecrit sans `omitempty`, il changeait CHAQUE OCTET de chacun d'eux — donc obligeait a recuire le
parc entier pour un champ vide (verifie : les 68 artefacts differaient). Le tag porte
`omitempty` des DEUX cotes, stocke et servi, comme le garde-rail de parite l'exige
(`replayview/parity_test.go` compare les tags). Meme regle que `Coverage.Abilities` au lot 5.6.

**Contrat public.** `refusedByRoster` entre au DTO `replaydoc.LayerCoverage`, a
`api/openapi.yaml` (regenere par `make openapi-gen`) et a
`apps/web/src/lib/api/generated.ts` (regenere) — additif et OPTIONNEL, aucun code web ne le
lit encore.

**Tests.** `replaybuild/rosterguard_test.go`, 4 tests. **Mutations** : (a) retirer l'appel a
`RosterFitsStatborg` rougit `TestGardeDEffectifRefuseUnFilmAuDelaDeHuitJoueurs` ; (b) compter
`len(facts.Players)` au lieu des sieges rougit
`TestSiegesAuCoupDEnvoiNeCompteNiBotNiRemplacant`, qui reproduit la forme exacte des cinq films
d'arene. Contre-epreuve : sans faits de match, l'effectif est INCONNU et rien n'est refuse.

---

## 7. Item 6 — les bornes de deroulage

### 7.1 L'hypothese `assists` de l'audit §10, instruite d'abord

L'audit ne pouvait pas trancher : « l'export d'oracle ne porte PAS de colonne `assists` ».
L'export du 2026-09-10 22:30 (`oracle_vague6_participants.tsv`) porte `kills`, `assists` et
`score`. **L'hypothese est CONFIRMEE**, et par la feuille de match elle-meme :

| film | `assists` publiees | feuille de match | ratio | joueur en cause | son pas maximal |
|---|---|---|---|---|---|
| `16ea3668` | 9 513 | **31** | 306,9 | 2535469190789936 | 9 482 |
| `8bc6074f` | 15 648 | **38** | 411,8 | 2535449383340628 | 15 610 |
| `f8efc5ca` | 15 645 | **37** | 422,8 | 2535469190789936 | 15 608 |

**40 806 pulses pour 106 assistances reelles**, portees par UN joueur par film. On borne donc,
et l'oracle dit de combien.

### 7.2 La mesure qui cale la borne, et elle se lit dans l'artefact

`incrementTimes` emet une action par unite gagnee et les DATE TOUTES a l'instant du point qui
les porte : **n actions publiees au meme `timeMs` pour un meme (joueur, statistique) SONT un pas
de n**. La grandeur se mesure donc sans decoder un seul film. Releve sur les 68 artefacts :

| population | pas maximal observe | effectif |
|---|---|---|
| actions d'objectif (drapeau, zone) | **1** | 376 triples (film, action, joueur) |
| `kills` | 1 (138 triples), 2 (8), **3** (1) | 147 triples |
| `assists` | **1** | 130 triples |
| **aberrantes** | **64** (`4f77afc1` `flag_grabs`), **64** (`cde26226` `flag_steals`), **84** (`a0c36016` `flag_capture_assists`), 9 482 / 15 608 / 15 610 (`assists`) | 6 triples |

**Pire pas sain 3, plus petit pas aberrant 64, et RIEN entre les deux** — un facteur 21 de vide.
La borne passe de **100 000 a 16** : 5,3 fois au-dessus du pire pas sain (la marge couvre la
PREMIERE emission d'un slot vu en retard, qui date d'un coup les unites deja acquises) et
4 fois sous le plus petit pas aberrant. Les quatre bombes memoire connues (537 698 416 a
2 163 333 677) restent neutralisees avec une marge de 33 millions.

**L'en-tete est reecrit** (audit, decouverte 12.3, anti-patron n° 9) : il qualifiait de « pire
deroulage d'un film SAIN » les 17 306 unites du comp 20 B de `d9781168` ; l'oracle etablit que
ce MEME emplacement produit 84 assistances de capture pour ZERO reelle sur `a0c36016`. La
population dite saine ne l'etait pas.

### 7.3 Gate

| grandeur | avant | apres | exige |
|---|---|---|---|
| `a0c36016` `flag_capture_assists` / 2535427927026623 | 84 (oracle 0) | **0** | 0 |
| `cde26226` `flag_steals` / 2535469190789936 | 65 (oracle 1) | **1** | 1 |
| `4f77afc1` `flag_grabs` / 2533274818676576 | 64 (oracle 0) | **0** | — (film par ailleurs tu par l'item 5) |
| `cde26226`, les 7 autres voleurs | 4/1/1/2/2/1/5 | **inchanges**, tous a 1,000 | ne bougent pas |
| `16ea3668` `assists` | 9 513 | **31** = feuille | — |
| `8bc6074f` `assists` | 15 648 | **38** = feuille | — |
| `f8efc5ca` `assists` | 15 645 | **37** = feuille | — |

**Effet de bord bienvenu, et la decouverte 12.1 de l'audit tombe avec** : le denominateur de
couverture de ces trois films etait fausse par ces pulses (« 100 % de couverture sur un calque
dont 99 % du contenu est un seul compteur »). `coverage.objectives.available` de `8bc6074f`
passe de **15 808 a 198**. Le cout de rendu de `buildObjectivePulses` tombe dans la meme
proportion.

**Tests.** `named_bornes_test.go` : `TestIncrementTimesSerieSaineIntacte` recale sur le pire pas
sain MESURE (3, au lieu de 17 306, avec l'explication du changement de reference) ;
`TestIncrementTimesPasAberrantDeSoixanteQuatreRejete` (forme exacte de `cde26226`) et
`TestIncrementTimesAssistsExplosivesRamenentLaFeuilleDeMatch` (forme exacte de `16ea3668`,
six assistances reelles puis un pas de 9 482) sont neufs. **Mutation** : remettre
`maxUnrollPerStep` a 100 000 rougit les deux nouveaux.

---

## 8. Films a recuire — la liste exacte pour le superviseur

**26 des 68 films mesures changent** (comparaison a l'octet `parc0` -> `parc7`, les 42 autres
sont IDENTIQUES). Dans l'ordre, avec la raison :

| films | raison | item |
|---|---|---|
| `43716616` `d9781168` `51ebbc0f` `c88ec007` | demi-fenetre de tic du crane | 2 |
| `16ea3668` `58864b3c` `7fce3219` `8bc6074f` `a0c36016` `b8a44fe8` `bc60b4d9` `bf5ced1b` `cde26226` `f8efc5ca` `4ecdf3e7` | lacher qui ferme un portage deja ferme | 3 |
| `e60aaf06` `72b0a25e` | manche fantome fermee | 4 |
| `4f77afc1` `879a4dba` `5676a9ba` | calque des actions tu (effectif) | 5 |
| `0a44c6cc` `1b2d9e08` `bf2a9f05` | un joueur de plus au fil de score (budget d'evenements libere) | 6 |
| `0d265ab0` `46c3f91d` `a4083bd2` | compteur `flagCarries.steals` de bruit ramene a zero (films non-CTF) | 6 |

`16ea3668`, `8bc6074f`, `a0c36016`, `cde26226` et `f8efc5ca` cumulent les items 3 et 6 ; la
liste ci-dessus les nomme une seule fois.

**Les QUATRE films Oddball n'ont pas d'artefact au parc** : leur « recuisson » est une PREMIERE
cuisson, deja prevue au lot 6.8 point (b).

**Aucune montee de `SchemaVersion`** : le seul champ neuf (`coverage.objectives.refusedByRoster`)
est `omitempty`, et il est absent des 65 documents ou il vaut zero.

---

## 9. Gates de lot, joues reellement

| commande | resultat |
|---|---|
| `go test ./internal/analysis/replay/ ./internal/analysis/objectiveevents/ ./internal/replaybuild/` | **ok**, 3 paquets |
| `CGO_ENABLED=1 go test ./internal/service/...` | **ok**, tous les paquets |
| `CGO_ENABLED=0 go test ./internal/...` | aucun echec de test ; seuls les paquets a DuckDB echouent au SETUP faute de CGO sur ce poste (defaut d'environnement, pas du lot) |
| `golangci-lint run` sur `analysis/`, `replaybuild/`, `service/replayview/`, `domain/replaydoc/` | **9 avertissements, TOUS preexistants** et tous dans `analysis/filmdec/` et `analysis/home_test.go` — paquets que le lot ne touche pas. Sur les paquets touches (`analysis/replay`, `analysis/objectiveevents`) : **0 issues** |
| `go vet` (hook de pre-commit, 6 fois) | propre |
| `make openapi-gen` | `api/openapi.yaml` regenere, diff additif de 3 lignes |
| `openapi-typescript` | `generated.ts` regenere, diff additif de 2 lignes |

**Goldens.** Aucun golden de CI ne change sur les six items. Les goldens de rejeu portent sur
`000d5950` (mini-bobine Slayer), sans calque de crane ni de drapeau, et sur un corpus
d'equivalence que les items ne touchent pas. Les seules references de TEST qui bougent sont
celles de l'item 2 (trois assertions de bornes, delta explique en commentaire : +/- la
demi-fenetre mesuree) et celle de l'item 6
(`TestIncrementTimesSerieSaineIntacte`, reference 17 306 -> 3, avec la raison ecrite).

**Le corpus d'equivalence (`make replay-corpus-gate`) n'est PAS joue ici** : il appartient au
superviseur (instruction du lot), et il demande le parc reel.

---

## 10. Decouvertes, consignees et NON traitees

- **D1 — Rien ne ferme un portage de drapeau sur la prise d'un AUTRE joueur du MEME drapeau.**
  `boundFlagCarries` (`replay/flag_carries.go`) ne connait que la prise SUIVANTE DU MEME slot
  (`nextOpeningOfSlot`). Or le drapeau passe de main a main sans toucher le sol, et le calque le
  sait deja : `coverage.flagCarries.dropsWithheld` compte exactement ces fins retenues (14 sur
  `16ea3668`, 40 sur `b8a44fe8` avant l'item 3). C'est la cause du residu de 275,7 s de l'item 3,
  et notamment du cas nomme au gate (`16ea3668` / 2535417044536883, 55,1 s pour 0,9 s) : aucune
  vie libre ne nait pendant son portage parce que l'objet n'a jamais ete libre.
  *Gain estimable* : les 9 joueurs a plus de 5 s d'ecart, soit l'essentiel des 275,7 s.
- **D2 — `4ecdf3e7` perd toujours deux joueurs**, et ce n'est pas ce lot : 2533274877168586
  (4,1 s a l'oracle, 0 publiee, 0 periode) et 2535469190789936 (3,9 s pour 14,3 s). Les DEUX
  etaient deja dans cet etat avant le lot, aux memes valeurs a la decimale. Deja consigne a
  l'audit §12.4 pour le premier ; le second est neuf.
- **D3 — `a4083bd2` declare toujours 3 manches pour 1 a son fil de score.** Ses trois manches
  EXISTENT dans les enregistrements, la garde de l'item 4 ne les touche donc pas. C'est un
  Slayer (mode sans manche) qui ne publie aucun calque d'objectif : le desaccord ne coute rien
  de mesurable. Il reste le dernier des 68 films dont `coverage.score.rounds` contredit son
  propre fil de score.
- **D4 — `5676a9ba` publiait 112 `kills` pour 241 a la feuille** (ratio 0,465) avant l'item 5.
  Le film est desormais tu par la garde d'effectif, donc la question est SANS OBJET pour lui ;
  elle resterait ouverte le jour ou le lot L6 (portage BTB) rouvrirait ces films.
- **D5 — Le filtre juste serait au niveau de l'ENREGISTREMENT, pas du pas.** L'en-tete des
  bornes de `named_series.go` le dit depuis le lot 4b et le redit apres l'item 6 : rejeter le
  record entier quand l'un de ses canaux est hors domaine — comme `modeScoreInDomain` le fait
  deja pour le comp 0 — traiterait la cause plutot que la taille. La borne par pas reste un
  rempart, meme recalibree.

---

## 11. Reproduction

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
  -out    <racine>/.ai/V7.5/replay2d/registre_film
```

Les sorties `vague6_b1_*.tsv|log` committees sont exactement celles de cette commande sur le
parc cuit au HEAD de la branche apres l'item 6 (68 artefacts).
