# LOT 4 — SUIVI DELTA DE L'INVENTAIRE : execution du plan en 5 lots

> Execution du 2026-08-25, branche `wt/suivi-delta-inventaire` (base `feat/v75`, schema 19).
> Document fondateur : `FAISABILITE_SUIVI_DELTA_INVENTAIRE_2026-08-24.md` §4.
> Contrat : skill `plan-execution`. AUCUN COMMIT — travail en working tree.

## VERDICT EN UNE PHRASE

Le suivi delta des **GRENADES** est confirme multi-films (concordance 97,94 %, rappel 95,17 %,
controle croise 100,0 %) et il est **FUSIONNE** dans le rejeu au schema 20 ; le suivi delta des
**MUNITIONS** est implemente, mesure, et **REFUSE PAR SA PROPRE MESURE** — il n'alimente pas la
fiche ; l'identite de l'arme reste hors de portee de ce canal, comme l'etude l'annoncait.

---

## 1. STATUT DES 5 SOUS-LOTS

| lot | objet | statut |
|---|---|---|
| 4.0 | assainir la doctrine perimee | `[x]` |
| 4.1 | scanner i22 + i47, verrou multi-films | `[x]` |
| 4.2 | munitions delta i30/i31/i33/i34 | `[x]` avec RESERVE chiffree (porte par film) |
| 4.3 | rappel de l'ancre biped delta | `[x]` — 95,17 % publie |
| 4.4 | fusion dans le rejeu | `[x]` pour les GRENADES · `[!]` pour les MUNITIONS (§4.7) |

---

## 2. LOT 4.0 — DOCTRINE PERIMEE ASSAINIE

Deux commentaires affirmaient qu'i22 lit 91-92 % de comptes impossibles et en tiraient une
doctrine (« ne pas derouler la chaine de composants »). La mesure appartient a un AUTRE chemin
(`DecodeFrameRecords`, marche NON ancree) et elle est ANTERIEURE aux correctifs d'i0 (47 bits),
d'i25/i26/i27 et de la polarite de porte d'i30/i33.

- `filmdec/frame_records.go` — les 92,46 % sont desormais explicitement rattaches a
  `DecodeFrameRecords`, dates comme anterieurs aux correctifs, et confrontes a la mesure du
  chemin ANCRE (120 lectures sur 120, compteur == 4, valeurs dans {0,1,2}).
- `filmdec/grenade_events.go` — la justification « la chaine de composants ne marche pas » est
  remplacee par ce qui est vrai aujourd'hui : la chaine marche sur le chemin ancre, et ce
  decodeur RESTE parce qu'il donne un EVENEMENT (le type lance et son auteur) la ou i22 donne
  un ETAT. La mention « i22, non resolu » est corrigee.

**Gate** : `go build ./internal/analysis/...` et `go vet` verts, aucune ligne executable modifiee.

**Decouverte hors perimetre (non traitee)** : `filmdec/traverse.go:135` porte la meme mesure
perimee (« i22 passe de 90,02 % a 92,83 % de comptes impossibles ») pour justifier le choix
6/6/6. Elle sert d'argument comparatif et non de doctrine ; hors perimetre du lot 0.

---

## 3. LOT 4.1 — LE SCANNER DE GRENADES, ET LE VERROU MULTI-FILMS

### 3.1 Ce qui est livre

- `filmdec/inventory_delta.go` — `ScanFilmInventoryDeltas(dir) ([]InventoryDelta,
  InventoryDeltaStats, error)`, calque exact de `ScanFilmAbilityRanks` : ancre
  `matchBipedHeader` sur la bande de slots bipedes, marche des composants par les DESERS DE
  PRODUCTION, publication par hook.
- `filmdec/components_biped_ability.go` — hook `SetGrenadeSetHook` dans
  `consumeBipedDesiredGrenadeSet` (i47). Le deser jetait ses 9 bits ; il les publie desormais,
  a parcours de bits INCHANGE.
- `filmdec/ability_rank.go` — la marche biped est factorisee en **un seul exemplaire**
  (`walkRecordComponents`, visiteur), dont `walkRecordTo` n'est plus qu'une ligne. Regle des
  <= 2 copies tenue : i48, i28 et l'inventaire la partagent. Un seul parcours pour N cibles —
  l'inventaire en veut six.
- `filmdec/inventory_delta_stats.go` — les denominateurs du balayage, extraits pour tenir le
  seuil de 500 lignes (meme motif que `KeyframeInventoryStats`, sorti d'`inventory_decode.go`
  pour la meme raison). Tailles finales : 447 + 74 + 225 lignes.
- `filmdec/inventory_delta_test.go` — garde-rails PURS (aucun film, valables en CI) : largeur
  d'i47 figee a 9 bits, conversion 1-base -> rang base 0, test refutable d'i22.
- `filmdec/inventory_delta_corpus_test.go` — instrument de corpus, garde `INV_DELTA_FILMS`.

### 3.2 Le verrou multi-films — 25 films, filmdec

    records biped delta ancres              4 355 683
    i22 lues                                    4 065   dont 11 implausibles  -> 99,73 % plausibles
    i47 lues                                    2 103   dont 1 hors masque non vide -> 99,93 %
    ACCORD masque i47 <-> bitmap i22        1 925 / 1 925 = 100,00 %

**L'accord i22 <-> i47 est le controle le plus fort du chantier**, et il n'etait pas prevu par
l'etude. Les deux composants sont deserialises a des positions DIFFERENTES du meme record, par
des desers differents ; leur accord ne peut pas venir de la construction. C'est la meme regle
que le canal des images-cles impose deja a sa propre lecture d'i47.

### 3.3 La confrontation aux images-cles — 70 films, replay

    films balayes                                  70
    films rendant des couples confrontables        28    (>= 15 exige)
    images-cles a grenades lues                 2 738
    dont un delta anterieur du meme slot          729
    CONCORDANCE                             714 / 729 = 97,94 %
    selection i47                           213 / 225 = 94,67 %
    lectures delta                             12 454   dont 629 strictement entre deux images-cles
    AGE MEDIAN de la derniere lecture     10,00 s -> 8,09 s  (-19,1 %)

La concordance multi-films (97,94 %) est **superieure** au 97,2 % du film unique. Le plancher
du plan (95 %) est tenu avec marge. Minimum par film : 91,9 % (`07aa428d`).

### 3.4 Deux corrections que la mesure a imposees

**(a) Le masque i47 VIDE n'est pas une anomalie.** Le premier passage comptait 93,44 % de
conformite et faisait echouer le gate. Diagnostic : sur `03af54c3`, 75 lectures portent un
masque `000000` et une selection NON nulle. i47 est le jeu DESIRE — sur certains films le champ
de selection garde un rang residuel quand le masque retombe a zero. Ces lectures sont desormais
comptees a part (`MaskEmpty`) et publiees SANS selection : ne portant aucune grenade, le porteur
n'en selectionne aucune. Conformite reelle : 99,93 %.

**(b) Un film peut ne RIEN transmettre.** `0014603f` : 118 054 records ancres, **zero** annonce
d'i22, i47 ET i48. Ce n'est pas une panne de decodage — i28 y est lu 766 fois sur 766 (100 %),
donc l'ancre et la marche fonctionnent. Le film ne transmet simplement pas ces composants. La
fusion doit degrader sans bruit sur ce cas.

### 3.5 Reserve honnete sur l'ampleur du gain

**Le gain de fraicheur est reel mais plus modeste qu'espere** : -19,1 % (10,00 s -> 8,09 s), la
ou l'attente etait ~5 s. La cause est mesuree : i22 est RARE (4 065 transmissions pour
4 355 683 records, 0,09 %) parce que le film ne le transmet qu'au CHANGEMENT. 629 lectures
seulement tombent strictement entre deux images-cles.

### 3.6 Decouverte majeure hors perimetre

**Le canal des IMAGES-CLES est muet sur une large part du corpus.** Sur les 25 premiers films,
15 ne rendent AUCUNE lecture de grenades aux images-cles, alors que le canal delta y livre des
milliers de lectures. Sur 70 films, 28 seulement rendent des couples confrontables. C'est
l'inverse de la reserve attendue : le canal delta est le plus COUVRANT des deux. A instruire
hors de ce lot.

---

## 4. LOT 4.2 — LES MUNITIONS, ET LA RESERVE QUI LES ACCOMPAGNE

### 4.1 Ce qui est livre

- `filmdec/unit_weaponstate.go` — hooks `SetWeaponAmmoHook` (chargeur R(8) + fraction R(12),
  les deux portes ACTIVES-BAS) et `SetWeaponRoundsHook` (reserve R(11)). Parcours de bits
  INCHANGE, largeurs figees par test pur.
- `filmdec/inventory_delta_ammo.go` — types, capture par emplacement, enveloppes, et **la porte
  par film**.
- L'emplacement d'arme est deduit de l'ORDRE DES OCCURRENCES du nom dans l'archetype
  (`weapon-state-ammo` y apparait 4 fois), jamais d'index cable : un index est un numero de build.

### 4.2 Le film temoin reproduit l'etude AU CHIFFRE PRES

    emplacement 0  chargeur  n=563  min=1 p50=30 p90=72 max=80
    emplacement 1  chargeur  n=593  min=1 p50=24 p90=70 max=80
    emplacement 0  reserve   n=56   min=0 p50=4  p90=25 max=80
    emplacement 1  reserve   n=43   min=0 p50=6  p90=50 max=240

Identique, effectif par effectif, aux mesures de l'etude (§2.4). Le scanner de production n'a
donc pas devie de la sonde.

### 4.3 CE QUE LE MULTI-FILMS A REFUTE

Etendu a 25 films, le chargeur ne tient plus dans 1..80 : **4,50 % des 55 544 lectures
depassent 120**, avec un maximum a 250 et 251 valeurs distinctes sur 256 possibles.

Deux hypotheses testees et **REFUTEES** :

1. *« Je lis les 4 emplacements alors que l'etude n'en sondait que 2. »* Faux : les
   emplacements 2 et 3 ne transmettent quasi rien (n=1 et n=1 sur 25 films). Le bruit vient des
   emplacements 0 et 1 eux-memes.
2. *« La corroboration par i22 sur le meme record permettrait de filtrer. »* Inexploitable :
   **17 chargeurs sur 55 544** partagent un record avec une lecture d'i22 plausible. Les deux
   composants ne voyagent presque jamais ensemble.

### 4.4 Le vrai diagnostic : c'est BIMODAL, PAR FILM

Taux de chargeurs hors enveloppe (120), film par film, 25 films :

    films PROPRES (18)      0,00 %  x16,  0,09 % (034200db),  0,25 % (040c5e7e)
    films CONTAMINES (7)    1,73 %  2,73 %  11,06 %  13,93 %  17,93 %  22,68 %  24,99 %

**Les deux populations sont separees par un vide** entre 0,25 % et 1,73 %. Le seuil de 1 %
tombe dans ce vide : il n'est pas un reglage sensible, toute valeur entre 0,3 % et 1,7 % donne
le MEME classement.

### 4.5 La decision : une porte TOUT OU RIEN, par film

Sur un film ou le curseur derive, les lectures qui TOMBENT sous l'enveloppe n'en sont pas plus
vraies — elles sont indiscernables. Filtrer valeur par valeur fabriquerait des chargeurs
plausibles et FAUX, ce qui est pire qu'un trou. `refuseAmmoIfContaminated` refuse donc le canal
munitions du film EN BLOC ; les GRENADES du meme film restent publiees, elles ont leurs propres
tests.

    PORTE MUNITIONS   7 films sur 25 refuses -> canal exploitable sur 72 % du corpus
    films retenus, chargeur : n=34 975  min=0 p10=5 p50=20 p90=35 max=112  distinctes=86
    reserve : n=3 241  min=0 p10=2 p50=39 p90=216 max=256  (1 seule lecture hors enveloppe sur 4 980)

### 4.6 Confrontation aux images-cles — LE GATE DU PLAN N'EST PAS ATTEINT

Le plan exigeait « concordance >= 95 % avec les images-cles sur les couples comparables ».
Mesure sur 70 films, chargeurs dont la lecture delta precede l'image-cle de moins d'une seconde :

    chargeurs confrontables (ecart <= 1 s)   1 152
    CONCORDANCE                          1 069 / 1 152 = 92,80 %

**92,80 % < 95 %.** Une explication innocente existait : un chargeur change a CHAQUE tir, donc
le desaccord pourrait n'etre que de la consommation survenue entre les deux lectures. Elle a ete
testee — en refaisant la confrontation a plusieurs ecarts. Si le desaccord etait du tir, la
concordance MONTERAIT quand on rapproche les deux mesures.

    ecart <= 0,10 s    118 / 134 = 88,06 %
    ecart <= 0,25 s    214 / 239 = 89,54 %
    ecart <= 0,50 s    292 / 321 = 90,97 %
    ecart <= 1,00 s    431 / 465 = 92,69 %
    ecart <= 2,00 s    629 / 675 = 93,19 %

**Elle DESCEND.** L'hypothese de la consommation est refutee par le profil : le residu est une
erreur de lecture, pas du tir. (Reserve de lecture honnete : les couples a ecart court sont
aussi les moments les plus ACTIFS du match — ce biais existe, mais il joue dans le sens qui
aurait du faire MONTER le taux, pas descendre.)

### 4.7 Statut `[!]` des munitions, et ce qui en decoule

Les munitions delta sont **implementees, testees, mesurees — et NON fusionnees**. Le scanner
reste en place : ses hooks, ses enveloppes, sa porte par film et ses garde-rails purs sont livres
et verts, et l'instrument de corpus publie ses chiffres. Ce qui n'est pas fait, c'est
l'alimentation de la fiche : `Inventory.Am` continue de venir des seules images-cles.

CE QUI SERAIT NECESSAIRE POUR LEVER LE `[!]` : comprendre pourquoi 22 films sur 70 ont une
distribution de chargeurs contaminee **alors que l'ancre et la marche y fonctionnent** — les
grenades du MEME film passent tous leurs tests. L'empreinte de registre ECS n'est pas le
discriminant : elle est INCONNUE sur des films propres comme sur des films contamines.

---

## 5. LOT 4.3 — LE RAPPEL DE L'ANCRE : 95,17 %

L'etude nommait le rappel comme **le seul vrai risque residuel** (§2.6 : « on ignore combien de
records biped delta existent reellement »). Il est desormais mesure, par un chemin qui ne demande
pas de connaitre ce nombre.

**LA DEFINITION RETENUE.** Une TRANSITION REELLE est un changement de quadruplet de grenades
entre deux images-cles consecutives du meme slot : le canal de PRODUCTION l'atteste,
independamment des deltas. La transition est CAPTUREE si une lecture delta du meme slot, tombant
dans l'intervalle, rend deja le nouveau quadruplet.

    transitions attestees par les images-cles   145
    capturees par un delta                      138
    RAPPEL                                   138 / 145 = 95,17 %

Ce chiffre ne dit pas combien de records l'ancre manque — il dit quelque chose de plus utile
pour une fiche : **quand l'inventaire d'un joueur change reellement, le canal delta le rapporte
19 fois sur 20**. Les 4,83 % manquants BORNENT le gain de fraicheur ; ils ne rendent aucune
lecture fausse (la justesse est mesuree a part, §3.3).

---

## 6. LOT 4.4 — LA FUSION : UN AXE A PART, ET POURQUOI

### 6.1 La decision d'architecture, et le defaut qu'elle evite

`ReplayDocument` publie un nouvel axe **`grenadeReads`** (schema **20**), alimente par les DEUX
canaux, chaque lecture portant sa `src` (`kf` / `delta`) — le patron exact d'`abilities.go`.

**LES LECTURES DELTA N'ENTRENT PAS DANS `Inventory`, et c'est le point du lot.** Le client
retient, pour un slot et une image, la lecture d'`Inventory` la plus recente <= T, et lit sur
ELLE le chargeur, la reserve et l'emplacement degaine autant que les grenades. Y verser des
lectures delta — qui ne portent QUE des grenades — ferait masquer une lecture pleine par une
lecture partielle, et **la cellule de munitions se viderait**. C'est tres exactement le defaut
que la version 19 vient de fermer (« une lecture vide EFFACE »), sous une autre forme. Verrouille
par `TestGrenadeReadsNAffectePasLInventaire`.

### 6.2 Ce qui est livre

    replay/grenade_reads.go        l'axe, sa couverture, sa doctrine
    replay/grenade_reads_test.go   5 garde-rails purs
    document.go / coverage.go / build.go   champ, couverture, cablage du scanner
    golden_assembly_test.go        le golden REND l'axe (sinon il ne le protegerait pas)
    golden_inputs_test.go          fixture v10 : porte les lectures delta

Cote web : `grenadeReads` traverse la frontiere (`replayNormalize`), `grenadeReadingAt` choisit
la lecture, et `ReplayInventoryRow` la prefere pour la boite de grenades **avec repli** sur
`inventory` quand l'artefact ne porte pas l'axe. Aucun redesign de fiche — un seul bloc de
lecture modifie. Les regles de nommage et de selection sont factorisees
(`grenadesCarriedFrom`, `selectedGrenadeFrom`) : un seul exemplaire pour les deux canaux.

### 6.3 Le SchemaVersion : 19 -> 20, et les goldens

Le client CONSOMME l'axe, donc la version monte — politique du depot appliquee, chronique ecrite
aux DEUX endroits qui la portent (`document.go` et le garde-rail de `structure_test.go`).
`EXPECTED_REPLAY_SCHEMA_VERSION` cote web suit : son garde-rail de parite a d'ailleurs attrape
l'oubli. Un artefact v19 se lit « a re-cuire » : il ne peut porter aucune lecture delta.

**Goldens regeneres, et nature du changement :**

    assembly_000d5950.golden   schema 19 -> 20, et un bloc NEUF « GRENADES PORTEES » :
                               240 lecture(s) · 120 par image-cle · 120 par delta
                               164 lecture(s) portent le rang SELECTIONNE
    inputs_000d5950.bin.gz     format v9 -> v10 : le fixture porte les 120 lectures delta.
                               SANS CETTE REGENERATION le golden n'aurait JAMAIS exerce le
                               second canal — il aurait fige un document que la production ne
                               produit plus.

---

## 7. ETAT DES GATES

| gate | commande | resultat |
|---|---|---|
| Go tests | `go test ./internal/analysis/replay/ ./internal/analysis/filmdec/... ./contracttest/...` | **vert** (3 paquets) |
| Go vet | `go vet ./internal/analysis/...` | **vert** |
| Go lint | `golangci-lint run ./internal/analysis/filmdec/... ./internal/analysis/replay/...` | **0 issue** |
| Types web | `tsc -b --force` | **vert** |
| Tests web | `vitest run` | **483 fichiers, 4674 tests, 14 skipped** |
| ESLint | fichiers touches | **0** |
| Contrat | `openapi-gen` puis `openapi-typescript` | regenere, jamais ecrit a la main |
| Goldens | `-update` avec le film de reference | regeneres, changements documentes §6.3 |

Deux lints ont ete corriges avant de clore : `walkRecordComponents` rendait un booleen que
personne ne lisait (retire — c'est `visit` qui sait ce qu'il a obtenu), et `invDeltaFracLevels`
etait une constante morte (retiree, sa doctrine portee sur le champ `FracQ`).

**Instruments de corpus** (gardes par `INV_DELTA_FILMS`, sautes en CI) :

    CGO_ENABLED=0 INV_DELTA_FILMS=<repo>/data/cache/film_chunks INV_DELTA_MAX=70 \
      go test ./internal/analysis/filmdec/ -run '^TestInventoryDeltaCorpus$' -timeout 120m -v
    CGO_ENABLED=0 INV_DELTA_FILMS=<repo>/data/cache/film_chunks INV_DELTA_MAX=70 \
      go test ./internal/analysis/replay/ -run '^TestInventoryDeltaConfrontationCorpus$' -timeout 180m -v

**Sondes jetables de l'etude** : `filmdec/i22_delta_research_test.go` et
`replay/i22_confrontation_research_test.go` sont CONSERVEES en l'etat. Le §5 de l'etude prevoyait
leur suppression au lot 4 ; elles n'ont pas ete touchees parce que leur retrait sort du perimetre
des cinq sous-lots et qu'il est separable — a trancher avec l'utilisateur.

**Note d'environnement** : le worktree n'avait pas de `node_modules` (les gates web ne pouvaient
pas tourner). Une jonction Windows a ete posee vers celui du depot principal
(`apps/web/node_modules`) — repertoire ignore par git, aucun effet sur le diff.

---

## 8. DECOUVERTES HORS PERIMETRE (notees, NON traitees)

1. `filmdec/traverse.go:135` porte la meme mesure perimee que le lot 0 a corrigee ailleurs.
2. Le canal des IMAGES-CLES est muet sur ~60 % du corpus pour les grenades (15 films sur 25,
   42 sur 70) — a instruire : c'est le canal de PRODUCTION actuel de la fiche.
3. `0014603f` ne transmet ni i22, ni i47, ni i48 en delta, alors qu'i28 y est lu a 100 %.
4. Les 9 films au canal munitions contamine partagent-ils une propriete (build, mode, taille de
   registre) ? L'empreinte de registre ECS INCONNUE apparait sur certains d'entre eux mais AUSSI
   sur des films propres : ce n'est pas le discriminant.

---

## 9. CORRECTIONS DE REVUE (2026-08-26) — 5 constats, 5 corriges

Revue adversariale du diff du lot 4. Aucun autre changement n'a ete fait : chaque correction
porte SON verrou, nomme ci-dessous.

### 9.1 [P1] Magie du fixture non incrementee

`golden_inputs_test.go` annoncait v10 en commentaire et gardait `goldenInputsMagic =
"REPLAYINPUTS9\n"` alors que la section `InventoryDeltas` avait ete INSEREE au milieu du flux.
Un fixture v9 passait donc la garde de version, puis mourait plus loin sur « uvarint illisible a
l offset N » — bruyant par chance, pas par construction, et le message parlait d octets la ou le
probleme etait une version.

- Corrige : magie portee a `"REPLAYINPUTS10\n"` ; la doctrine « la magie s incremente dans le
  meme commit que la suite des sections » est ecrite au-dessus de la constante.
- Fixture regenere par la porte documentee (`REPLAY_FILM_DIR=.../000d5950 go test -run
  GoldenInputs -update`, puis `-run GoldenAssembly -update`) : 2 360 004 octets bruts,
  1 029 382 compresses — 171 826 positions, 519 tirs, 150 loadouts, 70 lancers, 580 projectiles,
  184 inventaires, 57 lectures grappin, 93 morts, 8 index.
- **Verrou** : `TestGoldenInputsVersionGuard` (relit le corps COURANT precede de la magie
  PRECEDENTE ; exige un refus contenant « version inconnue »).
- **Mutation controlee** : `git checkout HEAD -- testdata/inputs_000d5950.bin.gz` (le fixture v9)
  fait desormais tomber `TestGoldenInputsRoundTrip` sur `magie absente ou version inconnue —
  regenerer`, et non plus sur un debordement de varint.

### 9.2 [P1+P2] La boite de grenades affichait l avenir, et portait le mauvais age

Deux defauts lies, un seul correctif.

(a) `grenadeReadingAt` s appuie sur `nearestReading`, dont le repli « a venir » rend un age
NEGATIF ; la boite preferait cette lecture sans regarder le signe. Mesure : slot 554 du film de
reference, une plasma affichee ~60 s avant sa premiere mesure, a pleine opacite, sous une
infobulle « lu il y a X ».

MEME DOCTRINE QUE LA « LECTURE VIDE A VENIR » : une lecture a venir ne prime jamais une
information passee. Le departage vit dans une fonction PURE neuve, `grenadeBoxAt` :

    axe passe (age >= 0)                    -> l axe gagne (c est le gain du lot)
    axe A VENIR + inventaire passe porteur   -> les compteurs PASSES gagnent, avec LEUR age
    axe A VENIR + rien de passe              -> la lecture a venir s affiche, ASSUMEE :
                                                age negatif tel quel, infobulle « dans X s »
    aucun axe (artefact <= 19)               -> l inventaire, tel quel

`g` vide = compteurs NON LUS (pas « aucune grenade ») : une lecture d inventaire sans compteurs
ne departage donc rien.

(b) La boite portait l opacite et l infobulle de l INVENTAIRE. Elle porte desormais les siennes
(`grenadeBoxHint`, nouvelles cles i18n `grenadeAge` / `grenadeAhead`), meme patron que la cellule
de capacite. POUR QUE LE GAIN SOIT VISIBLE, l estompage a ete RETIRE du conteneur de la rangee :
laisse la, il MULTIPLIAIT l opacite propre de chaque cellule par celle de l inventaire, et une
lecture de grenades de 8,1 s s affichait plus PALE qu avant le lot. Chaque cellule porte
maintenant l age de la lecture qui la decrit — munitions : l inventaire ; capacite : i48 ;
grenades : l axe ; badge d etat vide : l age de la LECTURE VIDE (`empty.age`, celui que son
infobulle annonce deja).

Consequence assumee : `grenadesCarried` et `selectedGrenade` (enveloppes sur `*From`) n avaient
plus d appelant hors tests — supprimees avec leurs cas, dont les deux non couverts ailleurs
(nommage bilingue, compteurs non lus) ont ete portes sur `grenadesCarriedFrom`.

- **Verrous purs** : `grenadeBoxAt` — 6 cas dans `inventoryReading.test.ts`.

### 9.3 [P1] Aucun test du branchement client

Aucun test de rendu n eprouvait la boite sur cet axe : l inversion complete de la preference
laissait la suite verte.

- **Verrous** : `ReplayTeams.test.tsx`, describe « boite de grenades : quelle lecture, et de quel
  age » — 4 cas (delta plus recente que l image-cle ; artefact <= 19 sans axe ; lecture a venir
  contre inventaire passe ; lecture a venir seule).
- **Double mutation** :
  - preference inversee vers l axe meme a venir -> tombe « lecture de grenades A VENIR :
    l information PASSEE de l inventaire prime » (title observe : « Plasma x5 · Grenades lues
    dans 1.0 s ») ;
  - preference inversee vers l inventaire meme quand l axe est passe (annulation du benefice du
    lot) -> tombe « artefact 20 : la lecture DELTA plus recente que l image-cle gagne, avec SON
    age » (title observe : « Fragmentation x2 · Grenades lues il y a 1.5 s »).

### 9.4 [P2] Godoc decalee

`golden_assembly_test.go` : le commentaire de `renderAbilities` etait colle sans ligne vide a
celui de `renderGrenadeReads`, et godoc l attribuait a la mauvaise fonction. Le commentaire a ete
DEPLACE au-dessus de `func renderAbilities` — chaque fonction porte sa doc.

### 9.5 Etat des gates apres corrections

| gate | commande | resultat |
|---|---|---|
| Go tests | `go test ./internal/analysis/replay/ ./internal/analysis/filmdec/... ./contracttest/...` | **vert — 572 tests** |
| Go vet | `go vet` sur les 3 paquets | **vert** |
| Go lint | `golangci-lint run ./internal/analysis/replay/... ./internal/analysis/filmdec/...` | **0 issue** |
| Types web | `make check-types` (`tsc -b`) | **vert** |
| Tests web | `node_modules/.bin/vitest run src/features/match-replay/` | **76 fichiers, 1119 tests** (1113 avant les nouveaux cas) |
| ESLint | 6 fichiers touches | **0** |
