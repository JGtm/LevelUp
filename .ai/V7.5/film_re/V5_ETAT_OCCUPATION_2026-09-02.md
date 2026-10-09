# RAPPORT — lot V5 : OU EST L'ETAT D'OCCUPATION COURANT DES VEHICULES ?

> Execute le 2026-09-02/03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`, aucune ecriture DuckDB. Mesures en AVANT-PLAN, `CGO_ENABLED=0`,
> GOCACHE isole (`scratchpad/gocache_v5`), donnees reelles du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data/cache`, LECTURE SEULE).
> `internal/analysis/replay/` (production), `apps/web/`, `cmd/weapon-sounds/` : NON touches.

---

## 0. LE RESULTAT EN SIX LIGNES

1. **L'IDENTITE DE L'OCCUPANT N'A ETE TROUVEE NULLE PART.** Les quatre pistes qui cherchaient
   un CHAMP nommant l'occupant (etat par joueur dans l'image-cle, composants du vehicule,
   entites-enfants, flux delta) rendent toutes un negatif, chacune avec son temoin et ses
   chiffres. Le GATE (a) — « donner l'occupant a n'importe quelle frame d'un episode » —
   **ECHOUE**, et le GATE (c) — le siege — **ECHOUE** aussi.
2. **UN SIGNAL D'OCCUPATION A ETE TROUVE, ET IL PASSE SES TEMOINS** : le record d'IMAGE-CLE
   d'un vehicule (`ti=40`) est **plus long quand le vehicule est occupe**. Test apparie
   vehicule-contre-lui-meme : **16 paires sur 17 (94,1 %)**, ecart moyen **+151 bits**, contre
   **3/12 (25,0 %)** au temoin par decalage de 37 s. Test des signes : p ~ 2,7 x 10^-4.
3. **L'ECART EST QUANTIFIE ET RECURRENT** : **+89 bits**, la meme valeur sur **trois films
   independants** (`53ce4390`, `a89a3d23`, `21468645`), et des multiples approximatifs ailleurs
   (+105, +197, +279, +314). C'est un BLOC de taille fixe qui s'ajoute, pas du bruit.
4. **LE CONTROLE DE SPECIFICITE TIENT** : la MEME mesure cote BIPEDE (le joueur est-il a bord ?)
   ne donne **rien** — 18/28 = 64,3 % au reel contre **9/11 = 81,8 % au temoin** (le temoin fait
   MIEUX). Le signal est propre au vehicule ; il n'est pas un artefact de la methode.
5. **UN CONFONDANT A ETE TROUVE ET NEUTRALISE, et il changeait le chiffre d'un facteur 9** :
   `WalkKeyframeWorld` SAUTE les records dont `Field26` n'est pas nul, donc une « longueur »
   peut couvrir plusieurs records. Sans filtre l'ecart apparent vaut **+1 348 bits** ; restreint
   aux records a voisin de slot immediat il vaut **+151**. C'est la valeur filtree qui est vraie.
6. **GHIDRA EST MORT** : `curl http://127.0.0.1:8089/mcp/schema` rend **exit 7 (connexion
   refusee)**. Tout ce rapport est de la mesure, aucune lecture d'executable.

**Ce qui est livre** : deux ajouts a `filmdec/`, exportes, testes SANS environnement, non
branches dans le rejeu de production — `KeyframeRecordSpans` / `ScanFilmKeyframeRecordSpans`
(l'emprise en bits des records d'image-cle, avec son garde-fou `SlotGap`) et `SetUnitRefHook`
(la sonde des champs de reference des composants `unit-*`, qui a servi a REFUTER la piste delta).

---

## 1. LA VERITE TERRAIN — les episodes attestes

Un episode = `[debut du trou du flux de position d'un slot bipede, instant de la SORTIE qui le
referme]`. La SORTIE est deja decodee et validee (occupant 100 % en bande, fermeture de trou
90,7 % contre 0 % au temoin — V3_EMBARQUEMENT § 2.4). Seuils repris a l'identique : trou >= 3 s,
fermeture a +/- 2 s, temoin par decalage de 37 s.

Instrument : `vehicules_v5_keyframe_test.go`, `TestV5KeyframeOracle`.

| film | episodes | images-cles | episodes contenant >= 1 image-cle | instants (episode x image-cle) |
|---|---|---|---|---|
| `0d76e8f1` (Behemoth SF) | **10** | 34 | **6/10** | 11 |
| `fccc61cd` (Launch Site SF) | **2** | 29 | **1/2** | 1 |

Detail `0d76e8f1` (les 10 trajets attestes annonces par la mission, retrouves a l'unite) :

| ep | slot | debut (s) | fin (s) | duree | siege | images-cles dedans |
|---|---|---|---|---|---|---|
| 0 | 514 | 2155,61 | 2163,75 | 8,14 s | 0 | 0 |
| 1 | 512 | 2157,03 | 2159,15 | 2,12 s | **2** | 0 |
| 2 | 514 | 2166,62 | 2184,56 | 17,93 s | 0 | 1 |
| 3 | 522 | 2200,89 | 2203,48 | 2,58 s | 0 | 0 |
| 4 | 515 | 2212,39 | 2231,19 | 18,80 s | 0 | 1 |
| 5 | 531 | 2293,86 | 2299,66 | 5,81 s | 0 | 0 |
| 6 | 554 | 2405,62 | 2420,65 | 15,03 s | 0 | 1 |
| 7 | 559 | 2413,39 | 2521,95 | 108,56 s | 0 | 5 |
| 8 | 551 | 2422,05 | 2443,15 | 21,10 s | 0 | 1 |
| 9 | 602 | 2700,35 | 2744,57 | 44,22 s | 0 | 2 |

**PREMIER CHIFFRE, ET IL BORNE TOUT LE LOT** : **7 episodes sur 12 (58,3 %)** contiennent au
moins une image-cle. Une image-cle toutes les ~20 s ne peut pas couvrir des trajets de 2 a 8 s.
**Le GATE (a) a 90 % est donc STRUCTURELLEMENT INATTEIGNABLE par la voie image-cle sur ce
corpus**, quel que soit le champ trouve — c'etait a dire AVANT de balayer, et c'est dit.

### 1.1 L'appariement episode -> vehicule (necessaire aux tests de presence)

L'evenement de sortie ne nomme pas le vehicule (sa ref 2, domaine 7, est gardee-absente en
pratique — V3_EMBARQUEMENT § 4). Le vehicule est donc APPARIE par la position : c'est le `ti=40`
le plus proche du premier echantillon de l'occupant apres le trou (modele V1a.4 : l'occupant
reapparait au contact du vehicule qu'il quitte).

Instrument : `vehicules_v5_appariement_test.go`, `TestV5Appariement`. **Appariement 10/10 sur
`0d76e8f1`, 2/2 sur `fccc61cd`**, distances 28,9 a 243,7 quanta.

---

## 2. PISTE 4 — GHIDRA : MORT, dit tel quel

```
curl -s -o ... -w "HTTP=%{http_code}" --max-time 6 http://127.0.0.1:8089/mcp/schema
HTTP=000 SIZE=0 ; curl_exit=7   (Couldn't connect to server)
```

Le pont MCP ET le serveur HTTP direct sont hors service. Le deserialiseur des records d'etat
par joueur n'a donc pas pu etre lu. **Tout ce qui suit est de la correlation mesuree.**

---

## 3. PISTE 5 + PISTE 2 — LE FLUX DELTA, ET LES COMPOSANTS DU VEHICULE

### 3.1 Ce qui a ete instrumente (additif, zero bit change)

Les composants `unit-*` lisent des CHAMPS DE REFERENCE d'entite sous trois formes, toutes
consommees et jetees jusqu'ici :

| forme | source | grammaire |
|---|---|---|
| `varw` | `FUN_1408f0ac4` -> `FUN_1406d3140` | `R(1)` porte ; si ouverte `[sonde R(1)] + R(13) + R(2)` |
| `w32g` | `FUN_14080d69c` | `R(1)` porte + `R(32)` |
| `w32` | `FUN_141d0f344` | `R(32)` inconditionnel |

`unit_ref_probe.go` (neuf, production) expose `SetUnitRefHook` ; les trois sites de lecture
publient desormais leur valeur. **Aucun bit consomme ne change** — le garde-rail
`TestUnitRefProbePublieSansChangerLesBits` l'exige explicitement (meme curseur avec et sans
sonde), et il tourne sans environnement.

**UN BOGUE DE METHODE TROUVE ET CORRIGE EN COURS DE ROUTE, note car il fabriquait 8 000 fausses
lectures.** `inferUnboundArchetype` essaie CHAQUE archetype du registre sur les memes bits pour
resoudre un delta de slot non lie : ce sont des lectures SPECULATIVES. Elles deposaient des
valeurs a des positions que la traversee retenue ne lit jamais, et l'attribution par position
les creditait a un composant au hasard (`object-position` se voyait attribuer des mots de 32
bits qu'il ne lit pas). La fonction mettait deja `posCaptureHook` et `dynPrecHook` a nil pour
exactement cette raison : `unitRefHook` les rejoint (idem `validatedResync`,
`repairUnportedComponent`). Lectures attribuees : **9 613 -> 1 210**, et l'attribution devient
semantiquement juste (plus aucune lecture creditee a un composant qui n'en fait pas).

### 3.2 Le recensement, et pourquoi la piste delta est STERILE (`0d76e8f1`)

`TestV5Recensement` : 40 189 paquets delta, 148 121 records, 112 579 propres, 35 542 desync.

| archetype | records propres | records tronques (desync) |
|---|---|---|
| `ti=35` bipede | **33 531** | 102 |
| `ti=40` vehicule | **176** | 138 |

Note de methode : les records DESYNC sont retenus jusqu'a leur point de rupture. `DesyncAt` est
l'index du premier composant present NON PORTE ; tout ce qui precede est consomme par la
grammaire portee. Les 13 composants `vehicle-*` non portes de `ti=40` (i30..i42) sont TOUS apres
les `unit-*` (i18..i26) : les jeter aurait prive la mesure de l'essentiel du corpus vehicule.

**TROIS FAITS TUENT LA PISTE :**

1. **Les composants porteurs de references n'arrivent quasiment jamais.** `i10`
   (object-parent-state) est lu **50 fois sur 33 531 records bipede propres — 0,15 %**. Meme
   ordre pour i18/i19/i20/i24/i25. Un etat d'occupation ré-emis pour permettre le seek serait
   present a chaque paquet, pas dans un record sur 670.
2. **Les valeurs ne se repetent JAMAIS.** Sur chaque canal, le nombre de valeurs DISTINCTES
   egale le nombre de lectures ouvertes (`n=12 ouverte=12 distinctes=12`, `n=21 ouverte=21
   distinctes=21`, ...). Un champ qui designe un conducteur rendrait le MEME slot pendant tout
   un trajet. C'est la signature d'une lecture desalignee, pas d'un etat.
3. **Aucune valeur ne tombe dans une bande de slots connue.** Sur la totalite des canaux
   `ti=35` et `ti=40`, `enBandeBipede = 0` et `enBandeVehicule = 0` (a une exception isolee
   pres). Bande bipede = 103 slots, bande vehicule = 47 slots.

Cote `ti=40` cela CONFIRME et EXPLIQUE le negatif deja publie : la grammaire de `ti=40`
desynchronise avant i18 (i2/i3 refutes en V1a/V2b, contamination 1247/1249 mesuree). Les 314
records `ti=40` exploitables du film ne sont pas un corpus.

**VERDICT PISTES 2 et 5 : REFUTEES.** Le flux delta ne porte pas d'etat d'occupation lisible.

---

## 4. PISTE 1 — L'ETAT PAR JOUEUR DANS L'IMAGE-CLE

L'image-cle porte un ETAT COMPLET (tous les composants, sans masque epars) et c'est le point de
reprise du seek : c'est l'endroit designe par l'axiome. Son corps n'est PAS bit-exact (aucun
decalage ne rend une marche exacte, cf. `keyframe_record_walk.go` ; la boucle d'etat complet
`keyframe_fullstate_loop.go` plafonnait a 0,85 % d'atterrissage). On ne le PARSE donc pas : on
le BALAIE, comme `ScanFilmKeyframeLoadouts` balaie les identifiants d'arme.

### 4.1 Balayage a decalage fixe (`TestV5Balayage`)

4 extracteurs (`s13`, `g15h`, `g15l`, `h32`) x 2 ancres (debut / fin du record) x tous les
decalages = **28 610 canaux**. Cible : la valeur lue designe-t-elle un vehicule VIVANT a cet
instant ? Temoin integre : les memes lectures sur les bipedes A PIED au meme instant.

Seuils ecrits avant mesure : a bord >= 90 %, temoin <= 10 %.

| corpus | lectures a bord | temoin | decalages passant | meilleur ecart |
|---|---|---|---|---|
| `0d76e8f1` | 11 | 229 | **0** | `fin s13 d=469` : 2/11 = 18,2 % contre 0,9 % |
| cumul 2 films | 12 | 441 | **0** | `fin g15h d=467` : 2/12 = 16,7 % contre 0,5 % |

(`fccc61cd` seul rend « 6 decalages passants » sur **n = 1** : sans valeur, et signale comme tel.)

**ECHEC.** Faiblesse identifiee et non masquee : les bornes de record ne sont pas fiables (le
balayeur saute les records a `Field26` non nul), et le contenu qui precede un champ est de
longueur variable — un decalage fixe ne peut pas etre exige a priori.

### 4.2 Presence SANS decalage, avec le vehicule apparie (`TestV5Appariement`)

Question affaiblie mais robuste : le slot du vehicule apparie apparait-il QUELQUE PART (fenetre
normalisee de 2 800 bits, les 4 extracteurs) dans le record d'image-cle de son occupant ?

| sens | classe positive | temoin |
|---|---|---|
| occupant -> vehicule (`0d76e8f1`) | **0/11 = 0,0 %** | 17/75 = **22,7 %** |
| vehicule -> occupant (`0d76e8f1`) | 1/11 = 9,1 % | 14/84 = **16,7 %** |
| occupant -> vehicule (`fccc61cd`) | 0/1 | 1/7 = 14,3 % |

**ECHEC, et le negatif est plus fort qu'un simple negatif** : la classe positive est SOUS le
taux de fond. Un champ absent rendrait le taux de l'occupant EGAL au fond. Un taux nul dit que
le record de l'occupant n'est pas le meme OBJET — c'est ce constat qui a ouvert le § 6.

**VERDICT PISTE 1 (identite de l'occupant) : REFUTEE.**

---

## 5. PISTE 3 — LES ENTITES-ENFANTS (sieges, tourelles)

Hypothese : un siege occupe ou une tourelle d'artilleur existe comme entite repliquee, dont la
VIE coincide avec l'episode. Mesure sans decoder aucun corps : premiere et derniere apparition
de chaque entite dans les tables d'image-cle, contre les episodes (`TestV5Enfants`).

| film | entites distinctes | part du temps couverte par les episodes | vies contenues dans un episode : **reel** | **temoin decale 37 s** |
|---|---|---|---|---|
| `0d76e8f1` | 854 | 37,0 % | **114** | **125** |
| `fccc61cd` | 1 068 | 2,8 % | 3 | 0 |
| `829abef9` | 984 | 24,4 % | **67** | **83** |
| `4898d586` | 930 | 23,9 % | **109** | **121** |

**ECHEC NET : sur les trois films a corpus suffisant, le TEMOIN fait MIEUX que le reel.** Les
vies « contenues dans un episode » ne sont que la consequence de la part de temps couverte
(37 %, 24 %, 24 %). Aucune entite n'apparait ni ne disparait avec un trajet. Resolution de la
piste, dite : 389/854, 487/1068, 373/984 et 453/930 entites ne sont vues qu'a UNE image-cle.

**VERDICT PISTE 3 : REFUTEE.**

---

## 6. CE QUI A ETE TROUVE — LA LONGUEUR DU RECORD D'IMAGE-CLE DU VEHICULE

### 6.1 D'ou vient la piste

Du negatif anormal du § 4.2 : 0/11 contre un fond a 22,7 %. Si le slot du vehicule n'est pas
dans le record de l'occupant PLUS RAREMENT que le hasard, c'est que le record n'a pas la meme
FORME. Le modele V1a.4 le predit : une entite attachee n'a plus de position, de vitesse ni
d'orientation propres — son etat serialise change de taille.

### 6.2 Premiere mesure, brute (`TestV5Forme`, `0d76e8f1`)

Longueur du record d'image-cle, en bits :

| classe | n | min | q1 | med | q3 |
|---|---|---|---|---|---|
| bipede A BORD | 11 | 2 816 | 2 829 | 2 860 | 3 054 |
| bipede a pied | 245 | 712 | 2 727 | **2 773** | 2 898 |
| vehicule **OCCUPE** | 11 | 1 761 | 2 060 | **2 060** | 2 119 |
| vehicule « vide » | 251 | 1 180 | 1 537 | **1 747** | 2 119 |

**Le temoin « vide » est FAUX, et il faut le dire** : le corpus n'atteste que les trajets qui
finissent par une sortie decodee (ratio board:exit = 1:15), donc la classe « vide » contient une
majorite de vehicules REELLEMENT occupes. Tout ecart mesure contre lui est une BORNE BASSE.

### 6.3 Le test APPARIE, vehicule contre lui-meme (`TestV5PaireVehicule`, 8 films)

Corpus : `0d76e8f1`, `fccc61cd`, `e232ffce`, `829abef9`, `53ce4390`, `4898d586`, `a89a3d23`,
`21468645`. Chaque vehicule apparie est compare A LUI-MEME : sa longueur mediane de record aux
images-cles PENDANT son episode attesté, contre ses images-cles hors episode. Cela elimine le
confondant du TYPE de vehicule (un Wraith et une Mongoose n'ont pas le meme nombre de
composants). Test des signes.

| mesure | n paires | plus long occupe | ecart moyen |
|---|---|---|---|
| **reel** | 27 | **25 (92,6 %)** | +1 348 bits |
| temoin decale 37 s | 19 | 8 (42,1 %) | -4 016 bits |
| **reel, records a voisin de slot immediat** | **17** | **16 (94,1 %)** | **+151 bits** |
| temoin decale, records a voisin immediat | 12 | **3 (25,0 %)** | +39 bits |

**LE CONFONDANT, ET SA NEUTRALISATION.** `WalkKeyframeWorld` n'accepte une ancre que si les 26
bits de `Field26` sont nuls ; un record dont ce champ ne l'est pas est SAUTE, et l'emprise
rendue couvre alors plusieurs records. Restreindre aux records dont le voisin de slot suivant
est immediat (`SlotGap == 1`) fait tomber l'ecart apparent de **+1 348 a +151 bits** — soit un
facteur 9. La proportion, elle, MONTE (92,6 % -> 94,1 %) et le temoin DESCEND (42,1 % -> 25,0 %) :
le filtre enleve du bruit, pas du signal.

Test des signes sur 16/17 sous H0 = 1/2 : **p ~ 2,7 x 10^-4**.

Ecart par film (records a voisin immediat) :

| film | paires | plus long occupe | ecart moyen |
|---|---|---|---|
| `0d76e8f1` | 2 | 2 (100 %) | **+314 bits** |
| `e232ffce` | 1 | 1 (100 %) | **+279 bits** |
| `829abef9` | 3 | 3 (100 %) | **+197 bits** |
| `53ce4390` | 2 | 2 (100 %) | **+89 bits** |
| `4898d586` | 6 | 5 (83,3 %) | **+105 bits** |
| `a89a3d23` | 1 | 1 (100 %) | **+89 bits** |
| `21468645` | 2 | 2 (100 %) | **+89 bits** |

**+89 bits, la MEME valeur sur trois films independants**, et des valeurs voisines de ses
multiples ailleurs. C'est un BLOC de taille fixe qui s'ajoute au record quand le vehicule est
occupe — l'hypothese naturelle (NON verifiee, et donc pas portee) est un bloc PAR OCCUPANT.

### 6.4 Le controle de specificite : la meme mesure cote bipede ne donne RIEN

`TestV5PaireBipede`, meme corpus, meme methode, classe positive = « ce joueur est a bord » :

| mesure | n paires | plus long a bord | ecart moyen |
|---|---|---|---|
| reel | 28 | 18 (**64,3 %**) | -5 016 bits |
| temoin decale 37 s | 11 | 9 (**81,8 %**) | +15 853 bits |

**Le temoin fait MIEUX que le reel.** La longueur du record BIPEDE ne dit pas si le joueur est a
bord. Le signal du § 6.3 est donc PROPRE AU VEHICULE : ce n'est pas un artefact de la methode
appariee, qui rendrait un positif des deux cotes.

---

## 7. LES GATES, ecrits avant la mesure

| gate | enonce | mesure | verdict |
|---|---|---|---|
| **(a)** | donner l'OCCUPANT a n'importe quelle frame d'un episode, >= 90 % des trajets des 2 films-demo | aucun signal ne nomme l'occupant (§ 3, § 4, § 5). Et la voie image-cle plafonne de toute facon a **7/12 = 58,3 %** de couverture | **ECHOUE** |
| **(b)** | le signal est NUL hors episode (temoin : joueurs a pied ~0 %) | pour le signal trouve : le MEME vehicule hors de son episode est plus court **16/17** ; temoin decale **3/12 = 25,0 %** ; controle bipede **64,3 % contre 81,8 %** au temoin | **PASSE** (pour le signal d'occupation) |
| **(c)** | distinguer le SIEGE (conducteur / passager / artilleur) | rien. Le seul siege != 0 du corpus (ep1, siege 2, `0d76e8f1`) dure 2,12 s et ne contient aucune image-cle | **ECHOUE** |
| **(d)** | survivre a un temoin par decalage temporel | 94,1 % contre **25,0 %** a 37 s de decalage | **PASSE** |

**Lecture honnete** : la mission disait « si un signal passe (a)+(b) mais pas (c), c'est deja une
victoire ». Ce qui a ete trouve passe **(b)+(d) mais pas (a)** — il donne l'OCCUPATION d'un
vehicule, pas l'IDENTITE de son occupant. C'est moins que la victoire annoncee, et c'est dit tel
quel. L'axiome de l'utilisateur reste intact : l'etat existe, et le § 6 montre meme OU il gonfle
le flux — il n'a simplement pas ete lu.

---

## 8. CE QUI EST LIVRE

| fichier | etat | role |
|---|---|---|
| `internal/analysis/filmdec/keyframe_record_spans.go` | **NEUF** (production, additif) | `KeyframeRecordSpan`, `KeyframeRecordSpans`, `ScanFilmKeyframeRecordSpans` — l'emprise en bits des records d'image-cle, avec le garde-fou `SlotGap` |
| `internal/analysis/filmdec/unit_ref_probe.go` | **NEUF** (production, additif) | `UnitRefRead`, `SetUnitRefHook` — la sonde des champs de reference des composants `unit-*` |
| `internal/analysis/filmdec/keyframe_record_spans_test.go` | **NEUF** (garde-rail SANS env) | payloads fabriques ; exige les bornes, exige que `SlotGap` denonce un voisin saute, exige que la sonde ne change AUCUN bit |
| `internal/analysis/filmdec/unit_weaponstate.go` | MODIFIE (3 sites) | `consumeOpt32`, `consume1408f0ac4Probe`, `consume141d0f344` publient leur valeur — meme grammaire, memes bits |
| `internal/analysis/filmdec/frame_records.go` | MODIFIE (2 sites) | `unitRefHook` mis a nil pendant les lectures SPECULATIVES (`inferUnboundArchetype`, `validatedResync`), a cote de `posCaptureHook`/`dynPrecHook` qui l'etaient deja |
| `internal/analysis/filmdec/frame_chain_infer.go` | MODIFIE (1 site) | idem pour `repairUnportedComponent` |
| `vehicules_v5_{occupation,keyframe,balayage,appariement,forme,paire,enfants}_test.go` | NEUFS (instruments, garde `V5_ROOT`/`V5_FILMS`) | les sept mesures de ce rapport |

**Rien n'est branche dans `internal/analysis/replay/`** : l'integration est le lot suivant, et
il n'y a de toute facon pas de quoi integrer tant que (a) echoue.

Suite sans environnement (tout ce qui est garde saute ; les deux garde-rails neufs tournent) :

```
go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
ok  levelup/go-api/internal/analysis/filmdec   2.661s
ok  levelup/go-api/internal/analysis/replay   29.797s
EXIT=0 · grep -c '^--- FAIL:' = 0 · gofmt -l vide · go vet propre
```

Commandes de rejeu (avant-plan, GOCACHE isole, `CGO_ENABLED=0`) :

```
V5_ROOT=<data>/cache V5_FILMS=0d76e8f1,fccc61cd \
  go test ./internal/analysis/filmdec/ -run TestV5KeyframeOracle -v -timeout 60m
V5_ROOT=<data>/cache V5_FILMS=0d76e8f1 \
  go test ./internal/analysis/filmdec/ -run TestV5Recensement -v -timeout 60m
V5_ROOT=<data>/cache V5_FILMS=0d76e8f1,fccc61cd \
  go test ./internal/analysis/filmdec/ -run TestV5Balayage -v -timeout 120m
V5_ROOT=<data>/cache V5_FILMS=0d76e8f1,fccc61cd \
  go test ./internal/analysis/filmdec/ -run TestV5Appariement -v -timeout 120m
V5_ROOT=<data>/cache \
V5_FILMS=0d76e8f1,fccc61cd,e232ffce,829abef9,53ce4390,4898d586,a89a3d23,21468645 \
  go test ./internal/analysis/filmdec/ -run TestV5PaireVehicule -v -timeout 180m
V5_ROOT=<data>/cache V5_FILMS=<idem> \
  go test ./internal/analysis/filmdec/ -run TestV5PaireBipede -v -timeout 180m
V5_ROOT=<data>/cache V5_FILMS=0d76e8f1,fccc61cd,829abef9,4898d586 \
  go test ./internal/analysis/filmdec/ -run TestV5Enfants -v -timeout 180m
```

---

## 9. STATUT DES ITEMS

| item | statut | justification |
|---|---|---|
| Piste 4 — Ghidra sur le deserialiseur d'etat par joueur | `[!]` | serveur HTTP MORT (`curl` exit 7). Signale, non contournable. |
| Piste 1 — etat par joueur dans l'image-cle | `[x]` **refutee** | balayage 28 610 canaux, 0 passant ; presence sans decalage 0/11 contre 22,7 % de fond. |
| Piste 2 — composants non explores de `ti=40` | `[x]` **refutee** | 314 records `ti=40` exploitables sur tout le film, valeurs jamais repetees, 0 en bande. |
| Piste 3 — entites-enfants (sieges / tourelles) | `[x]` **refutee** | vies contenues dans un episode : reel 114/67/109 contre temoin 125/83/121 — le temoin fait mieux. |
| Piste 5 — flux delta (ajout utilisateur) | `[x]` **refutee** | sonde neuve sur les 3 formes de champ de reference ; i10 lu dans 0,15 % des records bipede ; aucune valeur repetee. |
| GATE (a) occupant a toute frame, >= 90 % | `[!]` **ECHOUE** | aucun signal ne nomme l'occupant ; couverture image-cle plafonnee a 58,3 %. |
| GATE (b) nul hors episode | `[x]` **PASSE** | pour le signal d'occupation : 16/17 apparies, controle bipede negatif. |
| GATE (c) siege | `[!]` **ECHOUE** | aucun signal ; le seul siege != 0 du corpus dure 2,12 s sans image-cle. |
| GATE (d) temoin par decalage | `[x]` **PASSE** | 94,1 % contre 25,0 % a 37 s. |
| Decodeur additif dans `filmdec/`, exporte, teste, non branche | `[x]` | `KeyframeRecordSpans` + `SetUnitRefHook`, 2 garde-rails sans environnement. |
| Rapport + thought_log | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md`. |

**Ce qui reste ouvert** (note, non traite — regle du perimetre) :

1. **LIRE le bloc de +89 bits.** C'est la suite evidente : localiser, dans le record d'image-cle
   d'un vehicule occupe, OU les ~89 bits s'ajoutent (diff bit a bit du meme vehicule occupe /
   libre), et verifier l'hypothese « un bloc par occupant » sur un vehicule a deux occupants
   simultanes. Si ce bloc porte le siege et l'occupant, (a) et (c) tombent d'un coup.
2. **Le corpus d'episodes est le facteur limitant, pas la methode.** 12 episodes attestes sur
   les 2 films-demo, dont 7 seulement contiennent une image-cle. Elargir l'oracle passe par la
   marche de la liste ENTIERE d'evenements (deja identifiee comme voie au § 4.3 de
   V3_EMBARQUEMENT), qui multiplierait les sorties decodees.
3. **La grammaire de `ti=40` en delta reste fausse** (i2/i3, refutes en V1a/V2b). Tant qu'elle
   l'est, aucune mesure delta cote vehicule n'a de corpus. C'est un prealable a toute reprise
   de la piste 2.
4. **Le corps du record d'image-cle n'est toujours pas bit-exact** (0,85 % d'atterrissage pour
   la boucle d'etat complet). C'est le blocage de fond de la piste 1 ; il n'a pas ete attaque
   ici, la mission demandant de la correlation, pas un portage de grammaire.
