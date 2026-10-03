# Lot C-bis — phase 0 : la grammaire de `ti=13`, lue dans le binaire et confrontee aux octets

> Perimetre : CB.0.1, CB.0.2, CB.0.3 + Gate 0 du plan `PLAN_EXPLOITATION_REGISTRE_FILM.md`.
> **LECTURE SEULE.** Aucun code de production modifie : ni `traverse.go`, ni un `components_*.go`,
> ni une ligne de table ECS, ni un hook. Trois fichiers de TEST ajoutes, rien d'autre.
> Ghidra : instance de l'utilisateur, `HaloInfinite.exe` (311 104 fonctions), image base
> `0x140000000`, outils `mcp__ghidra__*` en lecture seule — aucun rename, aucun commentaire,
> aucun script, aucune analyse, aucune sauvegarde.
> Mesures du 2026-08-18, branche `wt/zones-ti13`. Gates : `LOTCBIS_gates.log`.
> Sorties : `lotC/<short8>_ti13_variant.tsv`, `lotC/<short8>_ti13_chainage.tsv`,
> journaux bruts `lotCbis_<short8>.log`.

## 1. Ce que la phase 1a du lot C avait laisse, et ce qui a debloque

Le lot C avait prononce un STOP sur ti=13 : « un type VARIANT a 11 branches » plus « un conteneur
de longueur variable », soit un ordre de grandeur au-dessus des trois canaux deja resolus. Les
acquis repris tels quels, sans les refaire : les adresses (`0x140ce5554` pour i1, `0x140ce593c`
pour i2..i33, coeur commun `0x140ce59bc`), le tag `R(4)`, les onze adresses de branche, deux
branches deja lues, et le fait que **le thunk +0x28 est un forwarder pur** (la charge utile
commence ou le masque finit).

**Ce qui debloque tout tient en une ligne de desassemblage.** Le contexte que le lecteur passe au
dispatcheur porte trois champs, et le troisieme n'avait pas ete identifie :

```
140ce555f: OR dword ptr [RSP + 0x30],0xffffffff   ; i1  : contexte+0x10 = 0xFFFFFFFF
...
140ce5980: MOV EAX,dword ptr [R9 + 0x8]           ; i2..i33 : contexte+0x10 = index du DESCRIPTEUR
140ce5987: MOV dword ptr [RBP + -0x39],EAX
```

Ce champ n'est PAS un compteur de bits restants (lecture provisoire de la phase 1a) : c'est un
**INDEX DE CHAMP**, et c'est lui qui choisit le mode. Le meme mecanisme que les `rtpc` du lot C
(`*(int *)(param_1 + 8)`), a un detail pres qui change tout : ici l'index ne sert pas a ranger le
resultat, il **selectionne la moitie du variant qui lit**.

## 2. La grammaire, bit a bit

### 2.1 Le contexte et les deux modes

`FUN_140ce59bc` lit `R(4)` (`+0x2c += 4`, `>> 0x3c`) puis appelle `FUN_140ce5aa4(tag, ctx)`.
Le contexte fait 0x18 octets :

| offset | contenu | i1 (`FUN_140ce5554`) | i2..i33 (`FUN_140ce593c`) |
|---|---|---|---|
| +0x00 | lecteur de bits | le lecteur | le lecteur |
| +0x08 | pointeur de sortie | `etat + 4` | tampon local de 136 octets |
| +0x10 | **index de champ** | **`0xFFFFFFFF`** | **`*(int*)(descripteur+8)`**, donc `0..31` |

Le dispatcheur teste `index < 0x20` et les deux moities du variant sont **disjointes** :

- branches **1 a 6** : `if (index < 0x20) return;` — elles ne lisent QUE si l'index est hors
  bornes, donc **seulement en mode A** ;
- branches **7 a 15** : `if (index < 0x20) { ...deserialiser... }` — elles ne lisent QUE si
  l'index est dans les bornes, donc **seulement en mode B**.

La valeur produite est rangee a `sortie+0` et le TAG DU VARIANT a `sortie+0x80` (pour i1 :
`etat+0x84`, que le lecteur remet a 0 avant de lire — d'ou la valeur « vide » par defaut).

### 2.2 Les douze alternatives

| tag | primitive | mode A (i1) | mode B (i2..i33) | sens probable |
|---|---|---|---|---|
| 0 | — | **0 bit** (valeur vide) | **0 bit** | absence de valeur |
| 1 | `FUN_1407ef804` | **R(4)** puis `-1` | 0 bit | enumere (0 = « absent ») |
| 2 | `FUN_1406cf008` | **R(1)** | 0 bit | booleen |
| 3 | `FUN_1406d84b4(...,0x18,...)` | **R(24)** quantifie `[-100, +100]` | 0 bit | flottant quantifie |
| 4 | en ligne (`FUN_140ce5720`) | **R(32)** | 0 bit | entier / handle |
| 5 | `FUN_14080dec4` **`"string-id-value"`** | **R(32)** | 0 bit | **identifiant de chaine** |
| 6 | `FUN_141d0f344` | **R(32)** | 0 bit | entier (code identique au tag 5) |
| 7 | `FUN_142ee59e0` | 0 bit | **R(24)** quantifie `[-100, +100]` | flottant PAR JOUEUR |
| 8 | `FUN_142ecf464` | 0 bit | **R(32)** | entier PAR JOUEUR |
| 9 | `FUN_14080dec4` **`"participant-string-id-value"`** | 0 bit | **R(32)** | chaine PAR JOUEUR |
| 10 | `FUN_1406cf008` | 0 bit | **R(1)** | booleen PAR JOUEUR |
| 11-15 | `FUN_141fce2f0` -> `FUN_1407ef804` | 0 bit | **R(4)** puis `-1` | enumere PAR JOUEUR |

Bornes du tag 3 / tag 7 lues en memoire : `DAT_143cd8f84` = `0xC2C80000` = **-100.0f**,
`DAT_143cd84a8` = `0x42C80000` = **+100.0f**.

**La moitie basse et la moitie haute sont le MEME jeu de types.** tag 3 / tag 7 (meme flottant),
tag 5 / tag 9 (meme identifiant de chaine), tag 2 / tag 10 (meme booleen), tag 1 / tags 11-15
(meme enumere), tag 4 / tag 8 (meme R(32)). Autrement dit : **le tag dit le TYPE de la propriete,
et le mode dit si on lit la valeur entiere ou l'element d'un joueur.** Les tags 12 a 15 tombent
sur le meme gestionnaire que 11 (le `switch` n'a que douze cas, `>= 11` est le defaut).

### 2.3 Largeur totale d'une valeur, tag compris

| tag | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11-15 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **mode A** (i1) | 4 | 8 | 5 | 28 | 36 | 36 | 36 | 4 | 4 | 4 | 4 | 4 |
| **mode B** (i2..i33) | 4 | 4 | 4 | 4 | 4 | 4 | 4 | 28 | 36 | 36 | 5 | 8 |

**La grammaire est COMPLETE et ne peut pas desynchroniser** : la largeur est entierement
determinee par 4 bits lus dans le flux, chaque branche est integralement consommee, et les deux
lecteurs finissent sur `MOV AL,0x1`. Statut de portage vise pour les deux : **`porte`**, pas
`partiel` (meme raisonnement qu'au lot C phase 1b pour les `rtpc`).

### 2.4 Le lecteur d'image-cle, qui donne le sens de l'archetype

`FUN_140ce55e8` (troisieme appelant de `FUN_140ce59bc`, trouve par `get_xrefs_to`) est le chemin
plein-etat, et il expose la structure que le chemin delta eparpille :

```
R(1) ; si 1 -> R(8)                                  (selecteur)
etat+0x84 = 0
FUN_14080dec4(rdr, "propertyName", etat)             R(32) : LE NOM DE LA PROPRIETE
R(1) : masquee par joueur ?
   si 0 -> UN SEUL variant                           (la valeur scalaire)
   si 1 -> BOUCLE DE 32 variants                     (une valeur PAR JOUEUR)
```

`ti=13` est donc un **sac de proprietes reseau nommees attache a un objet gere** — ce que confirme
la chaine `managed engine networked property creation: relevance=%5.3f` du binaire. L'archetype se
lit alors sans mystere : `i0` = le nom, `i1` = la valeur scalaire, `i2..i33` = les 32 valeurs par
joueur. **Il n'y a pas de « conteneur de longueur variable »** : le tampon local de 136 octets de
`FUN_140ce593c` est la copie SBO de la valeur composite, et chaque instance de composant y ecrit
UN champ. La lecture « conteneur » de la phase 1a est corrigee ici.

## 3. CB.0.1 — `i1 managed-object-property-component` : grammaire COMPLETE, confirmee sur octets

### 3.1 Distribution des tags, avec son temoin

Records a masque SINGLETON `{i1}`, bande reelle contre bande FANTOME (slots vides). Un tag de
4 bits lu sur du bruit est uniforme (6,3 % par valeur) ; lu sur un canal reel il est concentre.

| film | mode | records reels / fantome | tag dominant (reel) | concentration reel / fantome | coherence du tag par slot (mediane) |
|---|---|---|---|---|---|
| `7344d24f` | Strongholds | 3 759 / 290 | **t3 : 95,0 %** | 95,0 % / 15,9 % | **99,9 %** sur 12 slots |
| `696a9d7c` | Strongholds | 3 755 / 499 | **t3 : 95,4 %** | 95,4 % / 51,7 % | **99,9 %** sur 12 slots |
| `01e1f945` | KOTH | 3 694 / 124 | **t3 : 94,0 %** | 94,0 % / 25,8 % | **100,0 %** sur 3 slots |
| `530820e5` | CTF | 580 / 168 | **t3 : 85,7 %** | 85,7 % / 29,8 % | **100,0 %** sur 4 slots |
| `0a247154` | KOTH | 235 / 197 | **t4 : 91,1 %** | 91,1 % / 19,3 % | **100,0 %** sur 2 slots |
| `000d5950` | Slayer (temoin) | **5** / 25 | — | 40,0 % / 64,0 % | (aucun slot a >= 5 records) |

Lecture : sur les cinq films a objectif, i1 porte **un seul type de propriete par slot**, et ce
type est un flottant quantifie sur 24 bits (tag 3), avec un entier de 32 bits en second (tag 4).
Le temoin Slayer rend 5 records : **la ou il n'y a pas d'objectif, le canal se tait** — c'est le
controle negatif que la sonde F5 reclamait.

### 3.2 Le chainage : la largeur est confirmee PAR LE FLUX

Un record correctement dimensionne se termine la ou le suivant commence. On decode donc le record
entier et on regarde si un en-tete de record valide demarre au bit de fin. Temoin positif : `ti=4`
(1 slot, 1 composant, grammaire connue) ; temoin de hasard : le meme test decale de 3 bits.

| film | **i1 chaine** | temoin positif `ti=4` | fantome (tous composants) | temoin decale |
|---|---|---|---|---|
| `7344d24f` | **3 658 / 3 786 = 96,6 %** | 94,7 % | 2,1 % | 0,4 % |
| `696a9d7c` | **3 705 / 4 261 = 87,0 %** | 94,4 % | 2,7 % | 0,5 % |
| `01e1f945` | **3 681 / 3 708 = 99,3 %** | 95,8 % | 2,0 % | — |
| `530820e5` | **570 / 596 = 95,6 %** | 96,8 % | 2,9 % | — |

**i1 chaine au niveau du meilleur temoin du corpus.** La grammaire du mode A n'est pas une
hypothese : le flux lui-meme dit ou chaque valeur s'arrete, sur quatre films et trois modes.

### 3.3 Ce que disent les valeurs

- **tag 3 est une RAMPE.** Trois paquets consecutifs du meme slot (`01e1f945`, slot 1474) :
  8 394 200 -> 8 402 589 -> 8 410 977, soit un pas constant de ~1 199 quanta (~0,0143 unite sur
  `[-100, +100]`). `2^23 = 8 388 608` est le milieu de plage, donc zero : la valeur part de zero et
  monte regulierement. Meme forme sur `7344d24f` (8 389 606 -> 8 390 805 -> 8 392 003, pas 1 199)
  et `530820e5`. C'est la signature d'une jauge, la meme que `radial-progress` au lot C.
- **tag 5 est un identifiant de chaine, et son vocabulaire est PARTAGE ENTRE FILMS.** Les valeurs
  `1744059075`, `3599816372`, `4076464935` apparaissent sur les DEUX Strongholds, aux **memes
  slots** (1525, 1526, 1527) ; `2029524311` et `2267529471` sur les DEUX KOTH (slots 1471 et 1622).
  Un cadrage de bits faux ne reproduirait pas des constantes d'un film a l'autre — c'est exactement
  l'argument qui avait valide le cadrage des `rtpc` au lot C. **Trois slots, trois identifiants, en
  Strongholds : la forme d'un nommage des trois zones.**
- **tag 4** prend surtout `0`, `1`, `3` et `0xFFFFFFFF` — un petit entier avec une sentinelle.

### 3.4 Vecteurs

`ti13_vecteurs_test.go` : **33 vecteurs figes** (18 en mode A, 15 en mode B), tires d'octets reels
de 5 films, dont **31 CHAINES** (largeur confirmee par le flux). Toutes les branches qui lisent
sont couvertes. Deux exceptions honnetes, marquees dans le fichier : **tag 2 et tag 6 en mode A
n'ont AUCUN record chaine sur le corpus** — leur forme est lue dans le binaire, leur largeur n'est
pas confirmee par la donnee. Les tests tournent en CI (aucun film lu).

## 4. CB.0.2 — `i2..i33 player-masked-property` : grammaire COMPLETE, mode TRANCHE PAR LA DONNEE

Le desassemblage dit que l'index vient du descripteur, mais **la valeur que le registre y pose ne
se lit pas dans le binaire** (elle est ecrite a la construction de l'archetype). Deux hypotheses
restaient, et elles donnent des largeurs opposees. Le chainage les departage :

| film | mode B (element #k) | mode A (scalaire) | fantome (mode B) | temoin `ti=4` |
|---|---|---|---|---|
| `0a247154` (KOTH) | **53,8 %** | 13,9 % | 3,5 % | 96,5 % |
| `01e1f945` (KOTH) | **79,5 %** | 71,6 % | 2,0 % | 95,8 % |
| `7344d24f` (Strongholds) | 34,1 % | 34,4 % | 2,1 % | 94,7 % |
| `696a9d7c` (Strongholds) | 44,1 % | 44,1 % | 2,7 % | 94,4 % |
| `530820e5` (CTF) | 36,5 % | 33,8 % | 2,9 % | 96,8 % |
| `000d5950` (Slayer) | 3,4 % | 3,0 % | 7,4 % | 96,3 % |

Le verdict se lit composant par composant, la ou les player-masked parlent vraiment :

| film | composant | records | **chaine en mode B** | chaine en mode A |
|---|---|---|---|---|
| `0a247154` | i7 | 423 | **95 %** | 2 % |
| `0a247154` | i3 | 430 | **94 %** | 3 % |
| `0a247154` | i8 | 450 | **90 %** | 2 % |
| `0a247154` | i5 | 463 | **87 %** | 2 % |
| `0a247154` | i9 | 465 | **86 %** | 2 % |
| `0a247154` | i4 | 496 | **81 %** | 2 % |
| `01e1f945` | i7 | 249 | **90 %** | 6 % |
| `01e1f945` | i5 | 256 | **88 %** | 6 % |
| `01e1f945` | i8 | 259 | **86 %** | 6 % |
| `01e1f945` | i2 | 263 | **85 %** | 8 % |
| `01e1f945` | i4 | 275 | **81 %** | 7 % |

> **MODE B CONFIRME** : 81 a 95 % contre 2 a 8 %, sur onze composants et deux films. L'index du
> descripteur est bien dans `[0, 0x20[`, et `i2..i33` lisent chacun l'element d'UN joueur.

**Mais le trafic « player-masked » des Strongholds, lui, est de la CONTAMINATION — et c'est
mesure, pas suppose.** Sur `7344d24f` et `696a9d7c`, les composants les plus bavards de la bande
(i13, i17, i21 : 1 040 a 3 341 records, ceux-la memes que la sonde F5 avait releves) chainent a
**0 %** sous les DEUX hypotheses, soit **sous le fantome** (2,1-2,7 %). Le detecteur de motif
litteral le confirme : sur `696a9d7c`, i17 rend **22 valeurs de 64 bits distinctes pour 1 029
records (ratio 0,02)** — le meme motif rematche des centaines de fois, signature d'un faux positif
d'ancrage, alors qu'i1 est a 0,70 sur le meme film. Les 35 a 77 % de contamination chiffres par F5
se concentrent donc precisement la.

Vecteurs mode B : 15 figes, tous chaines, couvrant les tags 0, 6 (branches muettes) et 7, 8, 9, 10,
11, 12, 13, 14, 15 (branches lisantes), sur 4 films.

## 5. CB.0.3 — `i0 property-name` : verdict, et l'explication du « muet » de F5

`FUN_142ed69d8` = `FUN_14080dec4(rdr, "property-name", etat)` : **R(32), un identifiant de
chaine**, confirme par le nom de champ de debogage garde en retail. La table ECS est exacte.

| film | emissions reelles / fantome | valeurs distinctes (ratio) reel | fantome | i0 chaine |
|---|---|---|---|---|
| `7344d24f` | 121 / 568 | 65 (**0,54**) | 317 (0,56) | 5 / 148 = 3 % |
| `696a9d7c` | 68 / 539 | 51 (**0,75**) | 288 (0,53) | — |
| `0a247154` | 62 / 524 | 58 (**0,94**) | 390 (0,74) | — |
| `01e1f945` | 104 / 248 | 96 (**0,92**) | 195 (0,79) | — |
| `530820e5` | 56 / 250 | 48 (**0,86**) | 159 (0,64) | 3 / 204 = 1 % |
| `000d5950` | 9 / 558 | 9 (**1,00**) | 412 (0,74) | — |

**VERDICT : i0 ne porte rien d'exploitable sur le chemin DELTA, et le negatif de F5 est confirme
avec le controle qui lui manquait.** Le rapport « valeurs distinctes / emissions » du canal reel
n'est pas plus bas que celui du fantome — il est PLUS HAUT sur quatre films sur six. Un identifiant
de chaine tire d'un vocabulaire donnerait l'inverse. Et i0 ne chaine pas (1-3 %). Ce que F5 comptait
comme « emissions » est du bruit d'ancrage.

**POURQUOI le composant porte est muet, et c'est structurel** : `FUN_140ce55e8` (§2.4) montre que
le nom est ecrit dans le chemin d'IMAGE-CLE, une fois, a la creation de la propriete. Le chemin
delta ne le re-emet que si le nom CHANGE, ce qui n'arrive pas. **La ou ti=13 parle (i1, en delta),
le nom n'est pas ; la ou le nom est (l'image-cle), le walker du depot ne parse qu'un record par
table** (`LOTC_PHASE0.md` §4). C'est la vraie raison, et elle designe aussi la sortie : le nom se
prendra dans l'image-cle, pas dans le delta.

Reserve : une famille `0x37Exxxx01` revient sur trois films (`0x37EF8101`, `0x37E90101`,
`0x37E7C101`, `0x37E80102`) a 2-3 emissions chacune. Trop peu pour conclure, note pour la suite.

**Le pont vers le NOM existe malgre tout, et il est ailleurs** : le tag 5 d'i1 (§3.3) porte des
identifiants de chaine CONSTANTS par mode et par slot. C'est ce canal-la, pas i0, qui nomme.

## 6. Ce qui n'est PAS fait, et pourquoi

- **Le port Go** : hors perimetre par construction (phase 1, sur go du superviseur). Aucun `case`,
  aucune ligne de table ECS editee, aucun hook pose.
- **La resolution des identifiants de chaine en texte** : demande la table `string_id` du jeu ou
  les modules ; hors perimetre de la phase 0, et c'est un chantier a soi seul.
- **Les branches tag 2 et tag 6 en mode A** : forme lue dans le binaire, largeur NON confirmee par
  le flux (aucun record chaine sur 6 films). Signale dans les vecteurs.
- **Le sens exact du tag 3** (la rampe) : ce que la valeur MESURE est la question de CB.1.2, qui a
  ses seuils et ses temoins ecrits d'avance. La phase 0 publie la forme, pas le sens.
- **La convention de dequantification** (milieu d'intervalle contre bornes incluses) reste celle du
  lot C : non tranchee, ecart d'un demi-quantum, les vecteurs publient le quantum BRUT.
- **`FUN_140ce5b90`** (copie SBO du tampon de 136 octets) n'est pas desassemble : la fonction fait
  expirer l'outil MCP, et elle ne lit aucun bit (elle n'a pas le lecteur en argument).

## 7. Cout machine (D17)

Un film par processus, avant-plan, jamais `1b1e380f`, jamais le worktree principal.

| poste | mesure |
|---|---|
| duree par film (les deux instruments) | **21 a 42 s** (6 films) |
| pic memoire par processus | **16 Mo** (echantillonnage sur `0a247154`, le plus lourd) |
| plafond impose | 3 Go — jamais approche (0,5 % du plafond) |
| gates `go vet` + `go test` filmdec/replay/objectiveevents | 63 s |
| `golangci-lint --new-from-merge-base=origin/main` | 0 issue |

Cout mesure sur 2 films (`7344d24f`, `696a9d7c`) avant de lancer les 4 autres, comme D17 l'exige.

## 8. GATE 0 — verdict

> Enonce : « i1 ET le conteneur ont une grammaire COMPLETE ou PARTIELLE BORNEE (desync propre) ;
> sinon STOP ecrit avec les acquis. »

**GATE 0 : ATTEINT, et au-dela de l'enonce — les deux grammaires sont COMPLETES, pas partielles.**

| exigence | resultat |
|---|---|
| grammaire d'i1 | **COMPLETE** : 16 tags, 12 alternatives, largeur determinee par 4 bits, aucune desync possible ; chaine a **87-99 %** sur 4 films (temoin positif 94-97 %, fantome 2-3 %) |
| grammaire du « conteneur » i2..i33 | **COMPLETE** : ce n'est pas un conteneur mais l'element #k d'un composite ; mode B **confirme par la donnee** (81-95 % contre 2-8 %, 11 composants, 2 films) |
| desync propre | sans objet : **aucune branche ne peut desynchroniser**, les deux lecteurs rendent toujours `true` — statut de portage vise `porte` pour les deux |
| vecteurs | 33 figes, 31 chaines, en CI |

Reserve ecrite et non minimisee : **le volume exploitable depend du mode joue**. i1 est dense sur
Strongholds, KOTH et CTF (580 a 3 759 records) et muet en Slayer (5). Les player-masked ne parlent
vraiment qu'en KOTH ; en Strongholds, leur trafic apparent est de la contamination d'ancrage
mesuree a 0 % de chainage. La phase 1 devra donc porter les deux composants mais **ne pas attendre
d'etat par joueur sur les Strongholds**.

## 9. Statut des items

- [x] **CB.0.1** — `i1` : 12 branches portees sur papier (§2.2), largeurs (§2.3), portes et
  dependance a l'index d'etat (§2.1), grammaire bit a bit, 18 vecteurs dont 16 chaines,
  distribution des tags par film avec temoin fantome (§3.1) et chainage (§3.2).
- [x] **CB.0.2** — `i2..i33` : le « conteneur variable » est REFUTE (§2.4), la grammaire est celle
  de l'element #k, le mode est **tranche par la donnee** (§4), 15 vecteurs chaines, distribution
  par film ; contamination des Strongholds chiffree.
- [x] **CB.0.3** — `i0` : c'est bien un identifiant de chaine (R(32), nom de debogage
  `property-name`), mais **negatif sur le chemin delta**, avec l'explication structurelle du
  « muet » de F5 et le controle fantome qui manquait (§5).
- [x] **Gate 0** — ATTEINT (§8).

## 10. Decouvertes (hors perimetre — notees, NON traitees)

1. **Le champ « index de champ » d'un contexte de deserialisation est un motif de FORME reutilisable.**
   Un meme variant sert deux composants en lisant des moities disjointes selon un index pris dans le
   descripteur. Tout composant declare N fois dans un archetype est candidat au meme mecanisme — les
   `rtpc` du lot C en sont deja un cas (rangement seul). A verifier avant de porter tout composant
   multi-instance.
2. **Le chainage est un instrument d'ancrage a cout nul, et il manque partout.** « Un en-tete valide
   commence-t-il au bit de fin ? » separe le signal du bruit d'un facteur 30 (97 % contre 2-3 %) sans
   rien connaitre du sens. Les scanners de PRODUCTION par bande (ti=37/41/42, `projectiles.go`,
   `equipment_state.go`) n'ont toujours aucun chiffre de faux positifs (decouverte du lot C phase 0,
   toujours ouverte) : ce test le leur donnerait.
3. **Le detecteur de motif litteral (valeurs brutes distinctes / records) denonce un faux ancrage en
   une ligne.** i17 sur `696a9d7c` : 22 valeurs pour 1 029 records. Gratuit, et complementaire du
   fantome.
4. **`FUN_140ce55e8` est le lecteur d'IMAGE-CLE de ti=13** et il porte le nom de la propriete. Le
   nommage des objets geres passe par la, pas par le delta — mais le walker d'image-cle du depot ne
   parse qu'un record par table. Debloquer ce walker rendrait les noms de TOUS les archetypes, pas
   seulement ti=13.
5. **Les identifiants de chaine du tag 5 sont des constantes de MODE** (memes valeurs, memes slots,
   sur deux films d'un meme mode), comme les identifiants Wwise des `rtpc`. Il y en a exactement
   trois en Strongholds, sur trois slots consecutifs. Pont potentiel vers le catalogue de zones —
   c'est l'appariement que le lot C phase 1b declarait manquant.
6. **`0a247154` est le seul film ou i1 est domine par le tag 4 (91 %) et non le tag 3.** Deux KOTH,
   deux types de propriete dominants differents : le type depend de la carte ou de la variante de
   mode, pas seulement du mode. A garder en tete avant de cabler quoi que ce soit sur le tag.
