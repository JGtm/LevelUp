# ETAT DE L'ART — la palette Forge : nommer les objets, mesurer les zones

> Session du 2026-08-01, branche `feat/re-mode-score` (worktree dedie). Analyse
> **hors ligne** de fichiers locaux : 199 variantes de carte `.mvar`, 88 modules du jeu
> installe. Aucun jeu lance, aucune capture, aucune modification du jeu.
>
> Outillage neuf, jetable : `cmd/tmp_forgename` (palette), `cmd/tmp_forgeshape` (record
> d'objet `.mvar`), `cmd/tmp_forgedraw` (rendu + mesure). Rien du code livre n'a ete touche.
>
> Sorties versionnees : `.ai/V7.5/dumps/forge_zones/`.

---

## RESUME — ce que la session etablit

| question | verdict |
|---|---|
| **Q1 nommer** | Les modules ne portent **AUCUN nom** (table des chaines vide, 88/88). Le nom d'objet est **indecidable hors ligne** — voie fermee sur pieces. Mais la **classification** l'est : `type_id -> groupe de tag` resolu a **99,0 % (2758/2785)** sur les 199 cartes, contre 45 types auparavant. Controle positif **45/45 identiques**. |
| **Q1 power-ups** | Le `[!]` de J0.2 est **CLOS, par la negative** : la variante de carte ne place **pas d'equipement**. `eqip` = **3 types sur 2785**, presents sur **5 cartes sur 199**, aucun sur Catalyst ni Vagabond. La question « quels power-ups porte cette carte » n'a **pas** de reponse dans le `.mvar` — elle n'y est pas ecrite. |
| **Q2 mesurer** | **RESOLU par la source (b)** : le record d'objet porte un **sac de forme** jamais decode — famille + 4 nombres en virgule fixe 16.16. 16 434 formes lues sur 197 cartes. La source (a), l'emprise par type, est **inutilisable pour les zones** (modele vide, ±0,0005). La source (c) n'a **pas ete ouverte** — inutile. |
| **Q3 orientation** | **OUI, orientee.** Sur les formes tournees, la boite alignee capte **+31 %** (boite a 20 deg) et **+18 %** (cylindre) de positions joueur en plus que la forme reelle ; sur les formes non tournees les deux comptages sont **identiques au millieme**. |
| **Q4 regle generale** | **100 % des objectifs SURFACIQUES** portent une forme (zones Bastion 430/431, Extraction 434/434, reperes Ravitaillement 186/186). **0 % des objectifs PONCTUELS** en portent (apparitions de drapeau 0/669, socles 0/855). Ce n'est pas un trou de couverture : c'est la structure. |
| **noms de zone (A/B/C)** | **ABSENTS du `.mvar`.** Trois zones d'une meme carte ne different que par leur position, leurs dimensions et leur `team_index`. La lettre est attribuee a l'execution. |

**Une correction a porter au chantier** : l'emprise de la palette (`forge_object_types.csv`)
est en **unites monde**, pas en metres — facteur **3,048**. Les positions du `.mvar` et les
formes de zone, elles, sont en **metres**. Melanger les deux donne un facteur 3 silencieux.

---

## Q1 — NOMMER

### Q1.1 Le nom d'objet n'existe pas dans les fichiers du jeu

Quatre voies essayees, quatre fermees, chacune sur une mesure :

| voie | mesure | verdict |
|---|---|---|
| Table des chaines du `.module` (en-tete +0x24) | **0 octet sur 88 modules** de `deploy/any` et `deploy/ds` | fermee |
| Chaines lisibles dans le tag `food` du type | Un tag `food` de 1 388 o ne contient que des marqueurs de structure (`ucsh`, `SFDM`, `colb`) | fermee |
| Le tag `forg` (globals, **752 984 o**) | 3 chaines seulement, toutes des noms de noeud de script (`forge_mode_object_spawning_enabled`, `forge_ngs_node_name_object_reference`, `forge_ngs_node_name_spawn_mode_object`) — **aucun nom d'objet** | fermee |
| `GlobalID = murmur3(chemin de tag)` | Teste sur 2 identifiants de niveau **connus** (Vagabond 88891201, Catalyst -1044063363) x 6 formes de chemin : **aucune correspondance** | non etablie |

Les groupes de tag de la palette Forge (`forge_objects-rtx-new.module`, 43 685 entrees) :
`rtmp` 7322 · `scgt` 5636 · `mat ` 5119 · **`food` 4235** · `bloc` 4007 · `hlmt` 3621 ·
`phmo` 3611 · `coll` 3606 · … · `fpal` 110 · `fosp` 52 · `foki` 27. Les `fpal` (« palette »)
font 272 octets et ne contiennent aucune chaine.

**Conclusion.** Sur ce build, un `type_id` ne peut pas etre nomme hors ligne. La regle du
chantier tient : **un `type_id` inconnu reste SANS NOM**, jamais un nom approchant.

### Q1.2 Ce qui EST decidable : la classification, a 99,0 %

La chaine de palette etablie a la session precedente
(`food -> bloc|scen|mach|eqip|weap|vehi -> hlmt -> mode -> bloc « compression info » 84 o`)
a ete **rejouee par un lecteur neuf** et etendue a tous les `type_id` observes.

**CONTROLE POSITIF — 45/45 identiques**, groupe ET dimensions (tolerance 5e-4), sur les
45 lignes de `forge_object_types.csv`. Zero ecart de groupe, zero ecart de dimension.

Resolution, sur les **2 785 `type_id` distincts des 199 cartes** :

| groupe | types | lecture |
|---|---:|---|
| `bloc` | 2 669 | blocs et volumes Forge |
| `weap` | 33 | armes |
| *(irresolu)* | 27 | `food` sans reference d'objet — **publie, pas comble** |
| `mach` | 19 | dispositifs (socles d'objectif, portes) |
| `scen` | 18 | decor |
| `vehi` | 14 | vehicules |
| **`eqip`** | **3** | equipement |
| `ctrl` | 2 | terminaux |

**2 758 / 2 785 = 99,0 %.** Pour memoire, l'etat au 2026-07-31 : 33/42 sur Catalyst (78 %)
et **15/468 sur Vagabond (3 %)**. Les deux cartes sont desormais a **36/36** et **479/479**.

### Q1.0 LE NOMMAGE EST RESOLU — le nom est un murmur3, dans le tag

> **Cette section renverse le verdict Q1.1 ci-dessous.** Q1.1 concluait « le nom d'objet est
> indecidable hors ligne ». C'etait faux, et l'erreur etait une prémisse non testee : j'avais
> conclu que le binaire etait depouille sur la foi d'un constat de la session J0 qui portait
> sur les noms d'ARCHETYPE en memoire vive. **Le binaire est plein de chaines lisibles.**

**Le mecanisme.** Le tag `food` d'une entree de palette porte, **en tete de son second bloc
de donnees**, un mot de 32 bits qui est le **murmur3_x86_32(seed=0) du nom snake_case de
l'objet** — le meme hachage, le meme espace de nommage que les labels d'objectif deja
craques dans `objectives.go`.

**Ce qui l'etablit, et le controle croise gratuit** : `0xACDA2C3C = murmur3("gravity_hammer")`
et `0x7A3BE607 = murmur3("needler")`, trouves en passant les **82 281 chaines de type
identifiant extraites de `HaloInfinite.exe`** au craqueur. Et surtout :
**`1192059526 = murmur3("skull_weapon")` etait DEJA dans la table de `objectives.go`** —
la meme valeur, atteinte par deux chemins independants.

**Recette** (3 etapes, entierement hors ligne) :

1. extraire les chaines identifiant du binaire (`[a-zA-Z][a-zA-Z0-9_]{3,63}`) ;
2. lire le mot de 32 bits en tete du bloc 1 de chaque `food` (`tmp_forgename slots`) ;
3. craquer par murmur3 — directement, puis par combinaison de jetons (profondeur 2).

**Rendement : 64 entrees de palette nommees** sur 4 213, avec 0,08 puis 5,6 collisions
fortuites attendues. Toutes les correspondances retenues sont semantiquement coherentes et
plusieurs se controlent par leur usage reel sur les 199 cartes :

| nom | type_id | usage mesure | ce qui le confirme |
|---|---|---|---|
| `skull_weapon` | -1342546397 | **21 cartes / 21 instances** | **exactement 1 par carte** — le crane d'Oddball |
| `generic_ball` | -721267272 | 35 cartes / 61 | 1 ou 3 par carte, aux points d'apparition d'Oddball |
| `kill_volume` | 937132837 | 48 cartes / 164 | barrieres de mort |
| `cubemap_volume` | 43333489 | 57 cartes / 562 | sondes d'eclairage |
| `unsc_turret` | -269578988 | 41 cartes / 93 | tourelle, delai de reapparition 88 s |
| `frag_grenade` · `plasma_grenade` · `spike_grenade` | -113703634 · -1336095237 · 475600440 | 1 / 3 / 1 carte | **les 3 entrees `eqip` de toute la palette** |
| vehicules | — | — | `warthog` `wraith` `scorpion` `banshee` `phantom` `ghost` `mongoose` `wasp` `brute_chopper` `falcon` |
| armes | — | — | `assault_rifle` `sniper_rifle` `energy_sword` `gravity_hammer` `needler` `stalker_rifle` `plasma_pistol` |

**Deux corrections que ce nommage impose, et qui portent sur MES conclusions :**

1. **Le groupe `eqip` de la palette Forge, ce sont les GRENADES** — frag, plasma, spike, et
   rien d'autre. Le raisonnement « aucun `eqip` place, donc aucun power-up » etait donc un
   non-sequitur, comme la section Q1.3-bis le soupconnait. On sait maintenant pourquoi.
2. **`-721267272` n'est PAS un emplacement de power-up : c'est `generic_ball`.** Le motif
   « centre exact + deux symetriques » sur Catalyst est une disposition d'apparition
   d'Oddball, pas de surbouclier. Le candidat de Q1.3-bis est **refute** — et il l'est par la
   methode elle-meme, pas par un argument.

**Ce qui resiste** : les **quatre** entrees « emplacement » (`1486653438`, `-1062552774`,
`1882451900`, `801517767`, toutes sur le modele -370671751). Elles sont les seules de la
palette a porter un **second mot de 32 bits non nul** — chez les 4 209 autres entrees, ce
mot vaut 0. Leur mot de nom n'est craque par aucun des deux dictionnaires (ni les 82 281
chaines du binaire, ni les combinaisons de jetons a profondeur 3 sur un vocabulaire d'armes
et de power-ups cible). Ce sont donc bien des objets d'une nature differente — et ce sont
eux qui portent les deux emplacements de Vagabond (lance-roquettes en bas, camouflage en
haut, cf. Q1.3-ter).

Sorties : `.ai/V7.5/dumps/forge_zones/palette_noms.csv` (4 213 entrees, hachage + nom quand
il est craque) et `noms_craques_murmur3.txt` (les 33 correspondances hachage -> nom).

### Q1.3-bis REPRISE DU 2026-08-01 (soir) — le controle differentiel de l'utilisateur

> **Ce qui a declenche la reprise** : la formulation de Q1.3 ci-dessous est TROP FERME et
> elle est retiree. Elle supposait que « power-up » implique le groupe de tag `eqip`, ce qui
> n'a jamais ete etabli. L'utilisateur a fourni le controle qui manquait :
> **Cliffhanger n'a pas de power-up ; Catalyst et Vagabond en ont.** Un negatif, deux positifs.

**D'abord, tester ma propre faiblesse.** `resolveType` retient la premiere reference d'une
liste de groupes ORDONNEE : un `food` qui reference a la fois un `bloc` (le socle) et un
`eqip` (l'objet) serait classe `bloc`. Une commande `groups` rend donc l'ENSEMBLE des groupes
atteignables a profondeur <= 2 (`food -> refs` et `food -> foki -> refs`), sans aucun ordre de
priorite. Sur les 2 785 types : `bloc` 2669 · `weap` 36 · *(aucun)* 26 · `mach` 23 ·
`scen` 22 · `vehi` 15 · **`eqip` 3** · `ctrl` 2. **Le chiffre ne bouge pas.** La faiblesse
etait reelle, elle ne masquait rien.

**Le fait dur du controle differentiel** : sur les six variantes (`catalyst_catalyst`,
`catalyst_map`, `vagabond_map`, `vagabond_fo08_wetland`, `cliffhanger_map`,
`cliffhanger_ridgeline`), **AUCUN `type_id` n'est present a la fois sur Catalyst et sur
Vagabond et absent de Cliffhanger.** L'intersection est vide. Aucun objet unique ne peut donc
etre « le power-up » des deux cartes a la fois.

**Le meilleur candidat, et il ne tient qu'a moitie** : `type_id` **-721267272**.

| critere | mesure |
|---|---|
| groupe / emprise | `weap`, **0,28 x 0,29 x 0,30 m** — un cube de 30 cm, pas une arme |
| presence | **35 cartes**, 61 instances, **toujours dans la variante de BASE `*_map`**, jamais dans une variante de mode |
| nombre par carte | **1 ou 3**, jamais plus |
| Catalyst | **3**, aux positions (0,00 · -14,80 · 24,90), (0,00 · +14,80 · 24,90), (0,00 · 0,00 · 27,50) — **deux points parfaitement symetriques sur l'axe x = 0, plus un au centre exact, 2,6 m plus haut** |
| Aquarius (temoin de motif) | 3, a (+19,00 · 0), (-19,00 · 0), (0,02 · 0,02) — **meme motif** |
| Cliffhanger | **ABSENT** — conforme au negatif de l'utilisateur |
| Vagabond | **ABSENT** — **contredit le positif de l'utilisateur** |

Le motif « deux symetriques + un au centre conteste » est la disposition classique des objets
de puissance d'une arene. Il tient sur Catalyst et sur Cliffhanger. **Il tombe sur Vagabond.**

**La piste des labels est fermee, et c'est mesure.** Les trois instances de Catalyst portent
chacune 4 labels, dont deux DIFFERENT d'une instance a l'autre — ce qui ressemblait a un
discriminateur d'identite. Une recherche exhaustive murmur3 (craqueur valide **4/4 sur des
labels deja connus** avant usage) en a craque un : **`248451123 = minigame_include`**. Ce sont
donc des **filtres de mode**, pas l'identite de l'objet. Ils disent dans quels modes l'objet
apparait, jamais ce qu'il est.

**Nouveaux labels craques au passage** (a porter dans `objectives.go` ; 0,006 collision
fortuite attendue sur 1,6 M d'essais, garde-fou du fichier respecte) :

| hash | nom | objets porteurs |
|---|---|---|
| `-903313158` | `extraction_include` | 200 |
| `2140598169` | `firefight_include` | 1 357 |
| `248451123` | `minigame_include` | 297 |
| `-1875636905` | `forge_include` | 178 |

**Verdict revise.** Ce qui est etabli :

1. aucun objet de groupe `eqip` sur Catalyst, Vagabond ou Cliffhanger (mesure sans ordre de
   priorite) ;
2. aucun `type_id` commun a Catalyst et Vagabond et absent de Cliffhanger ;
3. un candidat serieux pour Catalyst (**-721267272**), qui echoue sur Vagabond ;
4. les labels ne nomment pas l'objet — ils filtrent les modes.

**Ce qui manque pour trancher est une observation, pas un calcul** : les trois positions
Catalyst ci-dessus correspondent-elles aux emplacements reels du surbouclier et du
camouflage ? L'image `catalyst_powerup_candidat.png` les marque. Si oui, le candidat est
identifie et le verdict s'inverse pour Catalyst. Si non, le `.mvar` ne porte pas les
power-ups et il faut les chercher dans le scenario de base ou dans le mode de jeu.

### Q1.3-ter L'EMPLACEMENT GENERIQUE — l'hypothese de l'utilisateur, verifiee

> Retour utilisateur du 2026-08-01 : « au milieu de Catalyst on a bien un emplacement de
> spawn d'arme/power-up, c'est un objet au-dessus duquel levite l'equipement » ; « sur
> Vagabond il y a un emplacement avec un lance-roquettes et un autre avec un camouflage,
> diametralement opposes » ; et l'hypothese : « un objet-emplacement reutilise par le jeu,
> l'arme ou le power-up etant place dessus en parametre ».

**L'hypothese est la bonne, et elle se lit dans les fichiers.**

**Recensement de la palette ENTIERE** (4 235 entrees `food` du module Forge, pas seulement
les 2 785 utilisees par les cartes) : `bloc` 4063 · **`weap` 76** · *(aucun)* 32 · `mach` 31 ·
`vehi` 21 · `scen` 21 · **`eqip` 4** · `ctrl` 2. Ces 80 entrees armes/equipement pointent vers
seulement **47 tags d'objet distincts** — donc plusieurs entrees de palette partagent un meme
modele. C'est exactement la signature d'un emplacement generique.

**Deux familles nettes, separees par la taille du modele :**

| famille | taille du modele | tags | lecture |
|---|---|---|---|
| **vraies armes** | 0,92 a 2,04 m | ~40 tags distincts | des modeles d'arme (fusil ~1 m, sniper 1,5 m, epee 2,04 m) |
| **emplacements** | **0,22 a 0,40 m** | 6 tags | des petits volumes — pas des armes |

Le detail des emplacements, avec leur usage reel sur les 199 cartes :

| entree de palette | tag d'objet | taille | cartes / instances | par carte |
|---|---|---|---|---|
| `1486653438` | -370671751 | 0,40 x 0,40 x 0,80 | 27 / 98 | jusqu'a 9 |
| `1882451900` | **-370671751** *(le meme)* | 0,40 x 0,40 x 0,80 | 20 / 117 | jusqu'a 16 |
| `-1062552774` | **-370671751** *(le meme)* | 0,40 x 0,40 x 0,80 | 21 / 46 | jusqu'a 4 |
| `801517767` | **-370671751** *(le meme)* | 0,40 x 0,40 x 0,80 | 11 / 24 | jusqu'a 4 |
| `-721267272` | 1072582607 | 0,28 x 0,29 x 0,30 | **35 / 61** | **1 ou 3** |
| `-2013147197` | 1379616328 | 0,28 x 0,29 x 0,30 | 2 / 4 | jusqu'a 3 |
| `-1342546397` | 1530156 | 0,22 x 0,16 x 0,20 | 21 / 21 | **exactement 1** |
| `-1136587198` | 1737103770 | 0,22 x 0,16 x 0,20 | 0 / 0 | — |

**Quatre entrees de palette pour UN SEUL modele** (-370671751) : la preuve directe que
l'objet place est un **emplacement**, et que ce qui apparait dessus est choisi ailleurs.

**Ou est le parametre.** Les quatre entrees `food` de cette famille sont **identiques octet
pour octet** (1 388 o) a **40 octets pres**, dont 8 dans l'en-tete du tag et **8 en tete de
leur second bloc de donnees** — un **u64 propre a chaque entree** :

| entree | u64 |
|---|---|
| `1486653438` | `0xFAB48286ABD1D5A6` |
| `-1062552774` | `0xF2A5966FF161B833` |
| `1882451900` | `0xB159316CAF7323DD` |

La reference `weap` qui suit dans le meme bloc est **identique** dans les trois. Ce u64 est
donc **le selecteur d'objet** — l'« en parametre » de l'hypothese utilisateur.

**Ce qu'il n'est pas** (teste, pour ne pas le re-tester) : ni un identifiant d'arme de film
(aucun ne se termine par `42C9679F`, la basse moitie commune de la table
`REFERENCE_WEAPON_IDS.md`), ni un identifiant d'asset de module (balayage des 88 modules a
tous les offsets de l'entree fichier : **0 occurrence** pour les trois).

**Ce que les cartes de reference disent :**

- **Vagabond** porte exactement **deux** emplacements, `1486653438` a (121,1 · 50,7 · 51,4)
  et `-1062552774` a (145,0 · 53,4 · 54,8) — **deux entrees DIFFERENTES a deux endroits
  opposes**. L'utilisateur y voit un lance-roquettes et un camouflage. Les deux entrees
  partagent le meme modele et ne different que par leur u64 : **le u64 distingue le
  lance-roquettes du camouflage.**
- **Catalyst** porte `-721267272` (3 instances : centre exact + deux symetriques) — modele
  different (0,28 m), et l'utilisateur confirme que celui du centre est bien un emplacement
  au-dessus duquel levite l'equipement. Une carte livree et une carte Forge n'emploient donc
  pas les memes entrees de palette pour la meme fonction.
- **Cliffhanger** ne porte ni `-721267272` ni les deux entrees de Vagabond — conforme a
  « pas de power-up ».

**Ce qui reste a faire, et c'est desormais une seule chose** : donner un nom au u64. Deux
voies, dans l'ordre : (1) **Ghidra** — trouver le lecteur de ce champ dans le binaire, c'est
ce que le projet `HI` permet ; (2) a defaut, l'observation — confirmer lequel des deux
emplacements de Vagabond porte le lance-roquettes, ce qui nomme deux u64 d'un coup.

### Q1.3 Le `[!]` power-ups de J0.2 — clos par la negative *(FORMULATION RETIREE, cf. Q1.3-bis)*

Les 3 types `eqip` de la palette et leurs porteurs :

| type_id | emprise (u. monde) | cartes |
|---|---|---|
| 475600440 | 0,309 x 0,100 x 0,100 | `nadair_map` |
| -113703634 | 0,062 x 0,062 x 0,086 | `cole_protocol_map` |
| -1336095237 | 0,059 x 0,061 x 0,065 | `argyle_map`, `empyrean_map`, `takamanohara_map` |

**5 cartes sur 199.** Ni Catalyst ni Vagabond n'en portent. Les emprises (18 a 94 cm une
fois converties) sont celles de petits objets poses, pas de socles de surbouclier.

Le blocage du 2026-07-31 disait : « la seule voie restante est la palette Forge ». La palette
est desormais lue a 99 % — et **elle repond que la donnee n'y est pas**. Surbouclier et
camouflage ne sont pas places par la variante de carte ; ils viennent du mode de jeu ou du
scenario de base. **Ce n'est plus un blocage d'acces, c'est un resultat.**

### Q1.4 Controle d'echelle — la palette est en unites monde

Trouve en confrontant les 14 `vehi` a des dimensions connues :

| type_id | emprise brute | x 3,048 | lecture |
|---|---|---|---|
| -411259918 | 10,73 x 7,35 x 3,82 | **32,7 x 22,4 x 11,6 m** | Pelican (~30 m) |
| 1503350133 | 3,31 x 2,51 x 1,10 | **10,1 x 7,7 x 3,3 m** | char Scorpion (~9 m) |
| -262750720 | 2,24 x 1,01 x 0,83 | **6,8 x 3,1 x 2,5 m** | Warthog (5,9 x 3,0 x 2,4 m) |

Lues en metres, ces valeurs donneraient un Warthog de 2,2 m. **L'emprise de la palette est en
unites monde ; le facteur est 3,048.** Les positions et les formes du `.mvar`, elles, sont en
metres (l'empreinte foulee de Cliffhanger mesure 85 x 75 m, taille d'une arene 4v4).

---

## Q2 — MESURER : LA FORME D'UNE ZONE

### Q2(a) L'emprise par type — inutilisable pour les zones

Le type de zone Bastion (`type_id` 1818458590) est un `bloc` dont l'emprise vaut
**±0,0005 sur les trois axes** — c'est la colonne `geom = modele_vide` de
`forge_object_types.csv`, qui marque un volume invisible. Meme constat pour le type
`-1476457415`. La colonne `geom` distingue donc bien la famille « modele reel » de
« volume invisible », mais elle ne donne **aucune dimension de zone**. Voie (a) close.

### Q2(b) L'echelle par objet — LA SOURCE, et elle etait sous nos yeux

`mapvar.parseObject` extrait 6 champs du record et `readGameplayBag` 3 du sac, alors que le
lecteur Bond decode l'arbre **entier**. Inventaire des champs reellement presents sur les
**380 464 objets** des 199 cartes :

| champ | type | occurrences | fichiers | lu par `mapvar` |
|---|---|---:|---:|---|
| #2 #3 #4 #5 #7 #10 | struct/u8 | 380 464 | 199 | oui |
| **#6** | float | **240** | **7** | **non** |
| **#8** | struct | 380 464 | 199 | partiellement |
| **#9** | struct | 380 464 | 199 | non (toujours vide sur l'echantillon) |

Le sac de forme vit a **`#8 -> #0[0] -> #0[0]`**, juste a cote de la categorie et des labels
que le depot lit deja :

```
#0  i32     famille de forme   (2 ou 3 — aucune autre valeur sur 16 434 formes)
#1..#4      toujours absents   (emplacements d'autres familles, presumes)
#5  i32     dimension A        virgule fixe 16.16
#6  i32     dimension B        virgule fixe 16.16
#7  i32     haut  (au-dessus du centre)
#8  i32     bas   (au-dessous du centre)
```

**Le pas est etabli** : 393216 = 6 x 65536 **exactement**, et **39,2 %** des 62 299 valeurs
brutes sont des multiples exacts de 65536 (valeurs rondes de concepteur), **54,6 %** des
multiples de 16384 (le quart de metre). Converties, les dimensions vont de **0,05 m a
248,5 m** — plage plausible, avec des valeurs rondes qui dominent (2,000 · 1,000 · 105,000 ·
125,000).

Deux familles seulement, sur 16 434 formes / 197 cartes :

| famille | objets | lecture retenue |
|---|---:|---|
| **2** | 4 624 | **cylindre** — `s5` = rayon, `s7`/`s8` = haut/bas |
| **3** | 11 810 | **boite** — `s5` = largeur, `s6` = profondeur, `s7`/`s8` = haut/bas |

### Le point qui a demande un depart : demi-extents ou tailles pleines ?

Trois mesures, dont deux **rejetees comme non concluantes** — c'est le resultat le plus
important a ne pas perdre.

1. **Part de l'empreinte sur du sol foule** (maille 0,5 m sur 171 116 positions joueur de
   Cliffhanger) : 58,9 % pour la lecture « taille pleine » contre 48,4 % pour « demi ».
   **REJETEE** : la mesure favorise mecaniquement la forme la plus petite (le temoin
   « demi / 2 » redonne exactement le score de « plein »). Le critere est **tautologique**,
   exactement comme le « critere en or par l'AABB » que le chantier a deja du retirer.
2. **Medianes par role** : rapport boite/cylindre de 1,49 (Bastion) et 1,67 (Extraction) —
   **ni 1 ni 2**. Non concluant : les concepteurs dimensionnent chaque site.
3. **Coincidences exactes sur une meme carte et un meme role** (79 paires cylindre+boite) :

   | lecture testee | paires a moins de 5 cm | dont exclusives |
   |---|---:|---:|
   | `largeur_boite ~ 2 x rayon` (**taille pleine**) | 11 | **11** |
   | `largeur_boite ~ 1 x rayon` (demi-extent) | 1 | 1 |

   Cas exacts : Cliffhanger cylindre r = 5,0999 / boite 10,1996 ; High Ground r = 2 /
   boite 4,000 ; Forest r = 2,5 / boite 5,000. Une coincidence a la quatrieme decimale
   n'est pas un hasard de dimensionnement.

**Retenu : `s5`/`s6` sont des TAILLES PLEINES pour la boite, `s5` est un RAYON pour le
cylindre** (c'est la presentation habituelle d'une interface Forge : Largeur/Profondeur
contre Rayon). `s7`/`s8` sont des distances **au centre**, vers le haut et vers le bas.

**Ce que ce depart ne prouve pas** : 11 paires sur 79. Le juge de paix reste l'oracle
d'execution — un evenement `zone_captured` date a la ms (etabli dans ce meme worktree,
`HANDOFF_EVENEMENTS_NOMMES_2026-08-01.md`) confronte a la position du joueur au meme
instant. Ce test **n'a pas ete joue** : il exige les bornes de dequantification de Vagabond,
absentes du catalogue `map_quant_bounds.json`. **C'est la premiere chose a faire ensuite.**

### Q2(c) L'entite de zone a l'execution — NON OUVERTE

(a) ne suffisait pas, (b) suffit. La consigne etait de n'ouvrir (c) que dans le cas
contraire. Elle reste necessaire pour **une colline qui se deplace en cours de partie**
(KOTH), que la variante de carte ne peut par construction pas decrire.

---

## Q3 — LA FORME EST-ELLE ORIENTEE ?

Oui. `Up` + `Forward` donnent une base complete, et la mesure le confirme sur Cliffhanger
(171 116 positions joueur, temoin negatif = la meme forme translatee de 25 m) :

| zone | orientation | part des positions dans la forme ORIENTEE | dans la boite ALIGNEE | temoin negatif |
|---|---|---:|---:|---:|
| Extraction, boite 10,20 x 4,19 | **-20,0 deg** | 0,800 % | **1,047 %** *(+31 %)* | 0,000 % |
| Extraction, cylindre r = 5,10 | *(cercle vs carre)* | 16,323 % | **19,214 %** *(+18 %)* | 0,033 % |
| Extraction, boite 5,98 x 9,00 | **+6,0 deg** | 2,656 % | 2,584 % | 1,929 % |
| Bastion, boite 6,80 x 6,00 | 0 deg | 12,553 % | 12,534 % | 0,000 % |
| Bastion, boite 6,50 x 4,50 | 0 deg | 9,138 % | 9,138 % | 0,090 % |

Les orientations lues sont des valeurs de concepteur : `Forward = (0,93969, -0,34202)` est
**exactement** cos/sin de -20 deg ; `(-0,70687, 0,70734)` sur Vagabond est **45 deg**.

**Lecture.** Sur une forme non tournee, boite orientee et boite alignee donnent le meme
comptage au millieme — la mesure ne bruite pas. Sur une forme tournee, la boite alignee
**deborde** : c'est le piege deja documente du fond de carte (« la boite alignee d'une piece
tournee deborde de la piece »). Ignorer `Forward` sur une zone d'Extraction de Cliffhanger,
c'est declarer dedans 31 % de positions qui sont dehors.

---

## L'IMAGE — `.ai/V7.5/dumps/forge_zones/zones_cliffhanger.png`

Cliffhanger, 8 zones d'objectif dessinees par-dessus le sol reellement foule.

- **gris** : les cases de 0,5 m visitees par au moins une position joueur (10 223 rectangles
  du BSP de la toile Ridgeline ont ete ecartes comme fond : ils couvrent toute la toile et ne
  disent rien de la surface jouable d'une carte Forge) ;
- **vert** : les 171 116 positions joueur ;
- **couleur d'equipe, trait plein** : la forme orientee en lecture « demi-extent » ;
- **blanc** : la meme en lecture « taille pleine » — celle qui est retenue ;
- **pointille** : la boite alignee sur les axes du monde.

Ce que l'image montre et que les chiffres seuls ne montraient pas : les formes en lecture
« demi » **debordent dans le vide** hors de la surface jouable (zone haut-droite, zone
gauche, zone basse), tandis que les blanches se posent sur les plates-formes. Et les
pointilles des deux zones tournees mordent visiblement a cote.

---

## Q4 — LA REGLE GENERALE : COUVERTURE SUR 199 CARTES

`.ai/V7.5/dumps/forge_zones/coverage.csv` (une ligne par carte).

| role d'objectif | avec forme | total | couverture |
|---|---:|---:|---:|
| `extraction_zone` | 434 | 434 | **100,0 %** |
| `stockpile_navpoint` | 186 | 186 | **100,0 %** |
| `strongholds_zone` | 430 | 431 | **99,8 %** |
| `flag_delivery` | 68 | 454 | 15,0 % |
| `oddball_spawn` | 1 | 245 | 0,4 % |
| `flag_spawn` | 0 | 669 | **0 %** |
| `stockpile_socket` | 0 | 855 | **0 %** |
| **total** | **1 119** | **3 274** | **34,2 %** |

Le 34,2 % global **n'est pas un trou de couverture** : c'est le partage entre objectifs
surfaciques et objectifs ponctuels. Un point d'apparition de drapeau et un socle de depot
**sont** des points — ils n'ont pas de forme parce qu'ils n'en ont pas besoin.

- 195 cartes portent au moins un objectif ; **188** portent au moins une zone.
- **7 cartes** ont des objectifs mais aucune zone (`corpo`, `critical_dewpoint`,
  `fortitude_heavies`, `nadair`, `oasis_heavies`, …) : ce sont des cartes CTF/Oddball pures.
- **1 seule** `strongholds_zone` sur 431 est sans forme, sur une carte UGC.

**La regle vaut donc pour les 30 cartes, et au-dela** : elle est verifiee sur 199 variantes,
dont les toiles Forge et les cartes de la communaute.

---

## NOMS DE ZONE (A / B / C) — ABSENTS

Question posee en cours de session. Reponse mesuree, pas supposee.

Les trois zones Bastion de Cliffhanger portent **le meme `type_id`** (1818458590), **les
memes labels** (`strongholds_include` + `strongholds_zone`), **le meme hachage** en
`#8/#24[0]/#1` (1038316458). Elles ne different que par : position, dimensions, orientation,
et **`team_index`** (0 / absent / 1). Meme constat sur les trois zones de Vagabond.

Aucun champ de nom, aucun index de zone, aucune lettre. Les labels non resolus portes par des
objets de zone ont ete passes au crible : le plus frequent (200 occurrences) a ete **craque**
— `murmur3("extraction_include") = -903313158`, correspondance exacte, a ajouter a la table
de `objectives.go`. Il en reste **un** non resolu (`-831896525`, 65 occurrences) ; deux lots
de candidats semantiques n'ont rien donne — **il reste inconnu, on ne devine pas**.

**Deux pistes, dans cet ordre, pour la lettre :**

1. **L'ordre des objets dans le fichier.** Les zones apparaissent dans l'ordre des index
   d'objet (Cliffhanger 178/179/180 ; Vagabond 385/387/2933). Hypothese testable : la lettre
   suit cet ordre. Le test est le meme oracle que Q2 — un `zone_captured` date, la position
   du joueur, la zone qui le contient.
2. **`team_index`.** Sur les deux cartes examinees, les trois zones se repartissent
   exactement en equipe 0 / neutre / equipe 1. C'est le cote de carte, probablement pas la
   lettre, mais cela distingue deja les zones deux a deux.

---

## LES CHAMPS DU RECORD QUE LE DEPOT LIT PUIS JETTE

Demande en cours de session. **Consigne, pas cable.** Inventaire complet dans
`.ai/V7.5/dumps/forge_zones/inventory.txt` (chemin, type, occurrences, fichiers,
nombre de valeurs distinctes, etendue).

### Etabli

| chemin | type | objets | fichiers | lecture | ce qui l'etablit |
|---|---|---:|---:|---|---|
| **`/#6`** | float | 240 | 7 | **echelle uniforme** | absente quand elle vaut 1 ; valeurs 0,561 a 2,063 groupees autour de 1 (1,034 x149 ; 2,063 x32 ; 1,200 x6) ; portee par 4 cartes livrees (bazaar 145, streets 52, forbidden 20, behemoth 2), sur des `bloc` |
| **`/#8/#1[0]/#4`** | u16 | 730 | 86 | **delai de reapparition, en secondes** | valeurs 30 (251) · 60 (123) · 120 (116) · 240 (42) · 45 · 180 ; porte par **`weap` 117 et `vehi` 94** — les seuls objets qui reapparaissent. Chez les armes : 120 (56), 45 (26), 30 (23), 240 (10) |
| **`/#8/#1[0]/#5`** | u16 | 280 | 63 | **second delai**, quasi exclusivement `vehi` (94) | 30 · 60 · 10 · 20 · 3 |
| **`/#8/#0[0]/#0[0]`** | struct | 16 434 | 197 | **le sac de forme** (cf. Q2) | — |

### Mesure, mais NON identifie — a ne pas cabler en l'etat

| chemin | type | objets | fichiers | ce qu'on sait | ce qu'on a REFUTE |
|---|---|---:|---:|---|---|
| `/#8/#0[0]/#10` | u16 | 30 991 | 66 | 0..318 | **pas un ordre d'apparition** : sur `merchant's_square` 2 172 objets pour 248 valeurs distinctes, **toutes repetees**. C'est un **index de groupe** — a rapprocher de `root[6]`, « tables de regroupement », que `mapvar` ne lit pas non plus |
| `/#8/#1[0]/#13` | bool | 22 120 | 115 | toujours 1 quand present | **pas « present au depart » lie a la reapparition** : 10 % des objets a delai le portent contre 34 % de ceux sans delai — aucune correlation |
| `/#8/#0[0]/#3` | bool | 10 585 | 12 | toujours 1 ; `scen` 325, `mach` 86, `bloc` 83 | — |
| `/#8/#0[0]/#15` | bool | 1 580 | 67 | toujours 1 | — |
| `/#8/#0[0]/#11` | u8 | 2 215 | 170 | 1..64, majoritairement 2 | — |
| `/#8/#24[0]` | struct | 380 289 | 199 | present sur **99,95 %** des objets : `#0` 1..4, `#1/#0` un hachage (848 valeurs), `#2` 1..2 | — |
| `/#8/#23[0]/#0` | 3 floats | 299 539 | 156 | un triplet par objet, etendues tres larges | — |
| `/#8/#18[0]` | struct | 10 030 | 70 | ~40 champs (portees, cadences, vitesses) — panneau de reglage d'objet | — |
| `/#9` | struct | 380 464 | 199 | present sur **tous** les objets, **vide** sur tout l'echantillon inspecte | — |

**« Present au depart » n'est pas identifie.** Trois drapeaux booleens sont candidats
(`#8/#0/#3`, `#8/#0/#15`, `#8/#1/#13`) ; aucun ne passe un test de correlation. En Bond un
champ absent vaut son defaut : un drapeau dont le defaut est « vrai » **ne s'ecrit jamais**,
ce qui rend ces trois-la ambigus par construction. Trancher demande un oracle d'execution
(un objet visible ou non a t=0 dans le film), pas une lecture de plus.

---

## LE CONTRAT DE DONNEES PROPOSE — `map_objectives.json`

Proposition. **Rien n'a ete implemente** : ni le schema, ni le rendu n'ont ete touches.

```jsonc
{
  "role": "strongholds_zone",        // inchange
  "type_id": 1818458590,             // inchange
  "team_index": 0,                   // inchange (-1 = neutre)
  "pos": { "x": 29.108, "y": 6.535, "z": -1.602 },   // metres, inchange
  "shape": {                         // NOUVEAU — absent quand la forme est inconnue
    "family": "box",                 // "box" | "cylinder"
    "half_x": 3.400,                 // demi-largeur, le long de `forward`
    "half_y": 3.000,                 // demi-profondeur (absent si "cylinder")
    "radius": 4.000,                 // rayon (absent si "box")
    "up_z": 4.000,                   // hauteur au-dessus du centre
    "down_z": 0.000,                 // profondeur au-dessous du centre
    "forward": { "x": 0.99999, "y": -0.00397, "z": 0.0 },
    "up": { "x": 0.0, "y": 0.0, "z": 1.0 }
  }
}
```

**Cinq regles, dans l'ordre de priorite :**

1. **Demi-extents en sortie, metres.** Le fichier brut porte des tailles pleines pour la
   boite (`half_x = s5/2`, `half_y = s6/2`) et un rayon pour le cylindre
   (`radius = s5`, sans division). La conversion se fait **une fois**, a la lecture, et le
   contrat ne publie que des demi-extents — pour qu'aucun consommateur n'ait a rejouer ce
   depart. Conversion depuis le brut : **diviser par 65536**.
2. **Toujours orientee.** `forward` et `up` sont **obligatoires** des qu'il y a une forme.
   Un consommateur qui ignore `forward` se trompe de 31 % sur une zone tournee (Q3). Une
   boite alignee sur les axes du monde n'est jamais une reponse acceptable.
3. **Pas de forme = pas de champ `shape`.** Un objectif ponctuel (apparition de drapeau,
   socle de depot) sort **sans** `shape`. Le rendu doit alors afficher un **point**.
   **JAMAIS de disque par defaut, jamais de rayon invente** : 65,8 % des objectifs sont des
   points, un rayon par defaut inventerait 2 155 zones qui n'existent pas.
4. **Conserver le brut a cote du derive.** Le contrat porte aussi
   `"raw": { "family": 3, "s5": 445644, "s6": 393216, "s7": 262144, "s8": 0 }` — le brut est
   la donnee du jeu, le derive notre interpretation. C'est la precaution deja posee pour le
   decoupage des zones (`SUIVI_REPLAY_2D.md` rang 4) et elle s'applique telle quelle : si le
   depart « taille pleine » est un jour infirme par l'oracle d'execution, le brut permet de
   recalculer sans re-extraire les 199 cartes.
5. **Pas de nom de zone.** Aucun champ `label`/`letter` : la donnee n'existe pas dans le
   fichier (voir plus haut). Si la lettre devient necessaire, elle viendra de l'execution et
   se posera dans un champ **distinct**, jamais devinee depuis l'ordre du fichier sans que
   l'hypothese ait ete testee contre le film.

**Ce que le contrat ne couvre pas** : la colline KOTH qui se deplace en cours de partie. Par
construction, une variante de carte decrit un etat initial ; une zone mobile releve de
l'entite d'execution (source (c), non ouverte).

---

## OUTILLAGE ET SORTIES

| outil (neuf, jetable) | ce qu'il fait |
|---|---|
| `cmd/tmp_forgename` | `hdr` `list` `dump` `ascii` `entry` `survey` `where` (sondes module/tag) · `hash` (murmur3) · **`control`** (rejoue les 45 types, 45/45) · **`classify`** (type_id -> groupe, 2785 types) |
| `cmd/tmp_forgeshape` | `fields` `inventory` (champs du record) · **`shapes`** (extraction des formes) · **`coverage`** (couverture par carte) · `props` (champs jetes) · `types` `zones` `obj` |
| `cmd/tmp_forgedraw` | rendu PNG + mesure orientee/alignee/temoin + test d'empreinte |

| sortie versionnee | contenu |
|---|---|
| `.ai/V7.5/dumps/forge_zones/shapes.csv` | 16 434 formes : carte, objet, type, role, famille, `s1..s8` **bruts**, position, orientation, equipe, labels |
| `.ai/V7.5/dumps/forge_zones/coverage.csv` | couverture par carte et par role |
| `.ai/V7.5/dumps/forge_zones/cls_all.csv` | 2 785 `type_id` -> groupe de tag + emprise |
| `.ai/V7.5/dumps/forge_zones/inventory.txt` | inventaire profond du record d'objet |
| `.ai/V7.5/dumps/forge_zones/props_stats.txt` | croisements des champs jetes par groupe de tag |
| `.ai/V7.5/dumps/forge_zones/zones_cliffhanger.png` | l'image |

---

## CE QUI RESTE OUVERT — dans l'ordre

1. **L'oracle d'execution**, qui tranche a la fois le depart « taille pleine » et la lettre
   de zone : `zone_captured` date a la ms x position du joueur x zone qui le contient.
   Pre-requis unique : **les bornes de dequantification de Vagabond** (`fo08_wetland`,
   module present sur `D:` et sur la cle), a produire par `cmd/mapquant-build` et a ajouter
   au catalogue `map_quant_bounds.json`. Non fait : cela ecrit dans une donnee **livree**,
   hors du perimetre « documents et outillage de recherche » de cette session.
2. **Le label `-831896525`** (65 objets de zone) — non craque.
3. **Les 27 `type_id` irresolus** (« aucune ref objet dans le food ») — publies tels quels.
4. **« Present au depart »** — trois candidats booleens, aucun etabli.
5. **La colline KOTH mobile** — source (c), entite d'execution.
