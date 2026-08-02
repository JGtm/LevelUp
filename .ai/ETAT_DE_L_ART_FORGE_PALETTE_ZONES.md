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

### Q1.0-septies LA VARIANTE DE CAISSE EST NOMMEE — et elle refute l'espoir de la piste 1 *(2026-08-02, reprise)*

> PISTE 1 du handoff, jouee integralement. Elle demandait : « chercher une carte ou deux
> instances du MEME `type_id` portent des variantes DIFFERENTES ; cela prouverait que la
> variante choisit l'objet ». **Les divergences existent. La conclusion esperee est fausse.**

**Le balayage.** `tmp_forgeshape cratevar` lit le champ `/#3[]/#8/#1[]/#0[]/#0` sur les
**380 464 objets** des 199 cartes : **35 239 porteurs**, **49 valeurs distinctes** non nulles.
Chemin verifie sur pieces avant tout comptage (`cratedump` sur Vagabond objet 357).

**Les divergences existent** — 51 couples (carte, type) portent au moins deux variantes. La
plus nette : `catalyst_map.mvar`, type `493070541`, **deux** variantes sur la meme carte.

**LA VARIANTE EST UN murmur3 DE NOM SIMPLE.** Passee au craqueur, elle rend d'abord
`equipment` — qui est **exactement** la variante partagee par les trois grenades du groupe
`eqip` (frag, plasma, spike) — puis `default`, porte par `skull_weapon` et des centaines de
`bloc`. Deux ancrages semantiques independants, pour 0,000 collision fortuite attendue.

**LES QUATRE ENTREES « EMPLACEMENT » SONT NOMMEES, ET L'ENSEMBLE EST COMPLET :**

| entree de palette | Crate Variant | nom craque |
|---|---|---|
| `1486653438` | `-88833402` | **`banished_kinetic`** |
| `1882451900` | `-1319554708` | **`banished_plasma`** |
| `-1062552774` | `-224029073` | **`banished_shock`** |
| `801517767` | `-257275188` | **`banished_hardlight`** |

Les **quatre** classes de degat Banished de Halo Infinite, sans trou et sans doublon. Quatre
collisions fortuites tombant pile sur l'ensemble complet d'une taxonomie du jeu : l'esperance
de la passe entiere valait 0,137.

**CE QUE CELA REFUTE.** La piste 1 esperait que la variante soit le selecteur d'objet. Le
recoupement terrain dit le contraire :

| carte | objets | variante | reapparition | lecture |
|---|---|---|---|---|
| **Vagabond** | `1486653438` a z=51,4 **et** `-1062552774` a z=54,8 | **`banished_kinetic` pour les DEUX** | 120 s | l'utilisateur atteste un lance-roquettes ET un camouflage : **la variante ne les separe pas** |
| **Catalyst** | `493070541` x3 | `banished_shock` x2 (symetriques) + `banished_plasma` x1 | 90 s | la variante varie PAR OBJET, mais c'est une famille d'arme, pas un objet |
| **Cliffhanger** | `1882451900` x4 | `banished_plasma` | 120 s | dont **un a `up.z = 0,046`** : le **RATELIER MURAL** que l'utilisateur signalait, enfin epingle |

**Verdict : la variante de caisse est une FAMILLE (classe de degat, style, materiau), pas
l'identite de l'objet pose.** Le selecteur reste le `Representation Name`.

**UN CINQUIEME TYPE D'EMPLACEMENT, absent de la table du handoff** : `493070541`, 8 cartes,
102 instances. C'est lui qui porte les emplacements de Catalyst.

### Q1.0-septies-bis L'ESPACE DE NOMMAGE DES OBJETS EST OUVERT — 30 noms de plus

Le meme craqueur, applique au champ **`/#3[]/#8/#24[]/#1/#0`** (848 valeurs distinctes sur
les 199 cartes), etablit deux choses.

**1. Les deux champs partagent UN SEUL espace de nommage.** `banished`, `default`, `forge`,
`apple`, `base_green`, `base_grime` apparaissent dans les deux. Ce sont des StringID.

**2. Ce champ NOMME L'OBJET.** Passe au vocabulaire du binaire a profondeur 1 — **136 906
essais, 0,027 collision fortuite attendue, 26 correspondances** :

| categorie | noms |
|---|---|
| vehicules | `warthog` `mongoose` `ghost` `banshee` `scorpion` `wasp` `phantom` |
| armes | `assault_rifle` `sniper_rifle` `bandit` `commando_rifle` |
| grenades | `frag_grenade` `plasma_grenade` `spike_grenade` |
| tourelles | `shade_turret` `plasma_turret` `unsc_turret` |
| styles / etats | `default` `forge` `forge_mp` `campaign` `banished` `emissive` `white` `dirt` `mossy` `glacier` `space` `locked` `closed` `arrow` `apple` `fo_ai_zone` `base_green` `base_yellow` `base_grime` |

**Le controle croise gratuit** : `1192059526 = skull_weapon` est **deja** dans la table de
`mapvar/objectives.go`, atteinte par un chemin totalement independant. La recette est validee
une seconde fois.

**CE QUI RESISTE, ET C'EST DESORMAIS BIEN BORNE.** Les quatre `Representation Name` des
emplacements (`-1412311642`, `-245254093`, `-1351408675`, `-219174009`) restent non craques
apres **cinq vocabulaires independants** :

| vocabulaire | profondeur | essais | correspondances |
|---|---:|---:|---:|
| binaire complet (136 906 mots) | 1 | 136 906 | 0 |
| binaire alphabetique (43 483) | 2 | 1 890 814 772 | 1, rejetee (`xvw_rmc`) |
| interface Forge (2 648) | 2 | 7 014 552 | 0 |
| Halo curate (327) | 3 | 35 073 039 | 0 |
| **noms d'auteurs de cartes** (2 762, vivier neuf) | 2 | 7 631 406 | 0 |

Les jetons `rocket`, `launcher`, `active`, `camo`, `camouflage`, `overshield`, `powerup`,
`spawner`, `pad`, `rack`, `weapon` etaient tous dans le vocabulaire curate a profondeur 3 :
`rocket_launcher`, `active_camo` et `powerup_active_camo` ont donc ete **essayes et refutes**.

Sorties : `noms_craques_stringid.txt` (table complete + rejets nommes),
`crate_variants.txt` (balayage), `representation_names.csv` (les 848).

### Q1.0-octies RECONNAITRE UN EMPLACEMENT SANS LISTE EN DUR — le predicat qui tient

> Question posee en cours de session : « au rejeu, on ne sait pas quels types d'emplacement
> la carte utilise ; y a-t-il un denominateur, ou faut-il balayer tous les cas de figure ?
> Selon le theme graphique, les auteurs privilegient un composant ou un autre. »
>
> **La premisse est exacte et elle est mesurable** : les cinq types se repartissent sur
> 27 / 21 / 20 / 11 / 8 cartes, et `493070541` est celui de Catalyst, absent de Vagabond.
> Une liste de `type_id` en dur aurait rate Catalyst. Reponse : **on ne balaye pas, et on
> ne liste pas — on branche sur un predicat.**

**D'abord, le faux probleme.** Au rejeu il n'y a rien a deviner : le `.mvar` de la carte
enumere ses propres `type_id`. Le catalogue est bati **par carte et hors ligne**. La vraie
question est celle de la REGLE DE CLASSEMENT — comment reconnaitre un emplacement sur une
carte jamais vue, sans table a maintenir.

**Trois predicats testes sur les 199 cartes** (`tmp_forgeshape spawners`) :

| predicat | portee mesuree |
|---|---|
| **P1** groupe de palette dans `{weap, vehi, eqip}` | trop large : 2 785 types, dont 2 669 `bloc` a exclure |
| **P2** porte un delai de reapparition (`/#8/#1[]/#4`) | **filtre grossier utile** : 65 `type_id` sur 2 785, 86 cartes sur 199 — `weap` 440 objets, `vehi` 184, `eqip` 37, et 56 `bloc` parasites |
| **P3** signature d'emprise du modele (dx/dy/dz au 10⁻⁴) | **LE PREDICAT** |

**P3 tranche net.** Parmi toutes les signatures portees par un objet a delai, **une seule
regroupe plus de deux `type_id`** :

```
-> 0.1306/0.1308/0.2617    5 type_id   387 objets   59 cartes
                           [-1062552774  493070541  801517767  1486653438  1882451900]
```

**Les cinq emplacements, et eux seuls.** Toutes les autres signatures ne portent qu'un ou
deux types, et ce sont des MODELES REELS : `2.2440/1.0139/0.8315` = le Warthog,
`0.0615/0.0615/0.0863` = la grenade a fragmentation. La taxonomie tombe d'elle-meme :

| lecture | signature | ce qui est place |
|---|---|---|
| **emplacement** | `0.1306/0.1308/0.2617` (petit socle) | un PAD ; ce qui apparait dessus est nomme par le `Representation Name` |
| **objet direct** | l'emprise du modele reel | l'objet lui-meme (arme, vehicule, grenade) |

**Pourquoi P3 survit aux themes et pas la liste de `type_id`** : le `type_id` est une entree
de PALETTE (une par habillage, donc une par theme), tandis que l'emprise vient du **modele
d'objet**, constante au niveau du titre. `493070541` reference d'ailleurs un tag d'objet
DIFFERENT des quatre autres (`-1269928936` contre `-370671751`) et porte pourtant **la meme
emprise a la sixieme decimale**. Ni le `type_id` ni le tag ne sont l'invariant : l'emprise
l'est.

**La limite, dite plutot que cachee** : l'emprise se lit dans la palette Forge du jeu
installe, pas dans le `.mvar`. Le catalogue `cls_all.csv` couvre les 2 785 types vus sur
199 cartes ; une carte neuve employant une entree de palette inconnue exige de rejouer
`tmp_forgename classify` contre le module. C'est une operation bornee et hors ligne, pas
un balayage.

Sortie : `.ai/V7.5/dumps/forge_zones/emplacements_predicats.txt`.

### Q1.0-nonies LA PISTE 2 EST SANS OBJET — il manque un DICTIONNAIRE, pas un SCHEMA

> Question posee en cours de session : « les definitions `weap.xml` / `eqip.xml` du depot
> Z-15, ce ne sera pas utile ? » Reponse mesuree, pas raisonnee.

**Son but d'origine est atteint par ailleurs.** La piste 2 visait « le groupe qui porte la
liste des variantes de caisse », pour pouvoir les enumerer et les nommer. Elles sont nommees
(§Q1.0-septies), par murmur3 direct. Ce but-la est clos.

**Son but residuel a ete teste, et il ne paie pas.** Une definition de tag decrit une
DISPOSITION DE CHAMPS. Or notre blocage n'est pas « ou est le champ de nom » — on le sait,
et le diff differentiel des quatre tags `food` le confirme a l'octet pres :

| offset | contenu | statut |
|---|---|---|
| 16, 20 | en-tete du tag | — |
| **688 et 732** | **une valeur d'identite par entree** : `1067333871` / `-2060575876` / `1128236838` | **NOUVEAU, non identifie** |
| 708 et 1300 | **Crate Variant** | craque (§Q1.0-septies) |
| 728, 736, 752 | le `type_id` lui-meme | — |
| **1296** | **Representation Name** | non craque |

Notre blocage est « quelle chaine hache vers `-1412311642` ». **Un schema ne repond jamais a
cette question ; seul un dictionnaire le peut.** `weap.xml` et `eqip.xml` sont des schemas.

**Et le mur est le meme un cran plus bas.** Le tag d'objet reference par les quatre
emplacements, `-370671751`, est un vrai `weap` de **14 272 octets** ; celui de Catalyst,
`493070541`, un `weap` de **14 088 octets** (ce n'est donc pas une entree de palette `food`,
contrairement aux quatre autres). Extraits et passes au crible : **49 suites imprimables**,
toutes des marqueurs fourCC en petit-boutiste (`tam ` = `mat `, `edom` = `mode`, `effe`,
`cshd`, `dfos`). **Aucun nom.** Parser ces tags avec `weap.xml` rendrait des champs, pas des
libelles.

**La troisieme valeur d'identite, publiee plutot que comblee.** `1067333871`,
`-2060575876`, `1128236838` — une par entree, ecrite deux fois dans le tag. Ce n'est
**ni un murmur3 de nom** (0 correspondance sur les trois vocabulaires, esperance de
collision <= 0,057) **ni un identifiant de tag** (`gid introuvable` sur les 88 modules).
Un troisieme espace de nommage est donc en jeu, vraisemblablement des identifiants d'asset.

**Ce qui reste realiste pour nommer les quatre**, par valeur decroissante :

1. **L'attribution par observation** — la voie que le handoff proposait deja, et la seule
   qui ne demande aucune percee. Le `Representation Name` est un identifiant STABLE : on
   peut s'en servir sans connaitre son libelle humain. Une seule confrontation au Theatre
   suffit a nommer une entree pour toujours, et le releve terrain de Vagabond en epingle
   deja deux (le bas / le haut).
2. **Un dictionnaire communautaire couvrant le build Forge** — les trois essayes ne le
   couvrent pas (intersection **exactement 0** sur 4 235 `food`).
3. **Ghidra sur le parseur du tag `food`** — session de retro-ingenierie a part entiere,
   et sans garantie : le binaire doit encore contenir la table de resolution.

### Q1.0-sexies LE FORGE KIT ET L'APPARIEMENT `fosp` — deux impasses, mesurees

**Le `Forge Kit` (groupe `kit!`) ne mene nulle part ici.** Sa definition fait 337 Ko et porte
90 champs `name` et **71 champs `folder name`** litteraux — l'arborescence du navigateur
Forge, en principe. Mais dans le jeu installe il n'existe **qu'UN SEUL tag `kit!`**
(gid 1824735348, 26 600 o, dans `globals-rtx-new.module`), et son extraction ne rend
**10 suites imprimables**, toutes des marqueurs de structure (`ucsh`, `SFDM`…). Les gids
`kit!` references par les `foki` de la palette (1769181442, 1747229976, 240986752…) sont
**introuvables** dans `deploy/any`. Le catalogue de dossiers n'est pas dans les fichiers
locaux du jeu.

**L'appariement direct dans `fosp` est REFUTE.** La definition montre un bloc « Strings »
qui met cote a cote `_2 String` (StringID) et `_1 String Literal` — l'esperance etait que le
StringID soit le murmur3 du litteral, ce qui aurait donne un dictionnaire mecanique. Test sur
le tag `fosp -631325601` (24 512 o) qui contient « Active Camouflage » : les murmur3 des
**cinq** ecritures plausibles du libelle (`Active Camouflage`, `active_camouflage`,
`activecamouflage`, `active_camo`, `ActiveCamouflage`) sont cherches dans les 24 512 octets
du tag, en petit-boutiste ET en gros-boutiste : **0 occurrence sur 10 recherches**.

Le StringID et le litteral sont donc **poses independamment** : l'un designe une propriete
interne, l'autre est le texte affiche. Recuperer les couples exige de **parser reellement le
bloc** avec la definition, pas de deviner l'appariement. C'est la prochaine action concrete
pour qui reprend.

**Un label de carte craque au passage** : `-534119345 = assault_bomb`, trouve dans le
dictionnaire Halo de 616 697 noms — a ajouter a `objectives.go` avec les quatre autres.

### Q1.0-quinquies LES CHAINES DE L'INTERFACE FORGE — trouvees, EN CLAIR, dans les tags

Les definitions `foki.xml` et `fosp.xml` (depot Z-15) portent des champs de type **`_1` =
chaine LITTERALE**, la ou tout le reste du format n'a que des StringID :

- `foki` / « Kit Groups » -> « Scriptable Properties » : chaque propriete (bool, numerique,
  chaine, tag, couleur) porte un **`_1 name`** en clair ;
- `fosp` / « Menu Item Definitions » : **`_1 Property Name Literal`**, et un bloc
  « Strings » qui APPARIE **`_2 String` (StringID)** avec **`_1 String Literal`**.

Extraction des 79 tags `fosp` + `foki` du module de la palette : **575 chaines**, dont
461 exploitables. Ce sont les libelles de l'interface Forge, en clair.

**Les CLES D'EMPLACEMENT — la reponse directe a la question « quels types d'emplacements
existe-t-il »** :

```
PowerUpPadPlacementKey        <- socle de power-up
PowerWeaponPadPlacementKey    <- socle d'arme lourde
WeaponRackPlacementKey        <- LE RATELIER signale par l'utilisateur
WeaponTrunkPlacementKey       <- coffre d'armes
EquipmentPadPlacementKey      <- socle d'equipement
GrenadePadPlacementKey        <- socle de grenades
OrdnancePodPlacementKey       <- capsule de largage
VehiclePadPlacementKey        <- socle de vehicule
```

**L'avertissement de l'utilisateur est confirme par le jeu lui-meme** : le râtelier est une
categorie d'emplacement DISTINCTE (`WeaponRackPlacementKey`), separee du socle d'arme lourde
et du socle de power-up. La confusion qu'il signalait est reelle et nommee.

**Les power-ups et equipements, nommes en clair** : `Overshield` · `Active Camouflage` ·
`Threat Sensor` · `Threat Seeker` · `Grappleshot` · `Repair Field` · `Shroud Screen` ·
`Quantum Translocator` · `Thruster`. Plus le catalogue d'armes complet (`Cindershot`,
`Diminisher of Hope`, `MA5K Avenger`, `Scout DMR`, `VK78 Commando`, `Vestige Carbine`,
`Skewer`, `Shock Rifle`, `Stalker Rifle`, `Tactical Rifle`, `S7 Sniper`, `M41 SPNKr`,
`Fuel Rod SPNKr`, `Mk50 Sidekick`, `M392 Bandit`…), les vehicules (`Razorback`, `Rockethog`,
`Gungoose`, `Wasp`, `Wraith`, `Ghost`, `Mongoose`, `Falcon`) et les categories de largage
(`AirDrop Alpha/Bravo/Charlie/Delta`, `Custom Equipment A/B/C`).

**Ce qui ne marche PAS, et c'est mesure** : le murmur3 de ces huit cles d'emplacement
n'apparait **nulle part** dans nos donnees — ni parmi les 3 995 hachages de nom de la
palette, ni parmi les labels des 199 cartes. Elles sont donc employees comme **chaines de
script**, pas comme StringID d'objet. Le lien entre une entree de palette et sa cle
d'emplacement passe par autre chose (vraisemblablement le `Forge Kit`, groupe `kit!`, dont
la definition fait 337 Ko — non explore).

Sorties : `forge_ui_chaines_litterales.txt`, `reference_foki_tag_definition.xml`,
`reference_fosp_tag_definition.xml`.

### Q1.0-quater LES DEPOTS COMMUNAUTAIRES — ce qu'ils donnent, ce qu'ils ne donnent pas

Trois depots proposes par l'utilisateur, evalues sur pieces.

| depot | ce qu'il apporte ICI |
|---|---|
| [`Surasia/InfiniteExt`](https://github.com/Surasia/InfiniteExt) | **rien directement** : un greffon DLL d'edition a chaud, pas de definitions. Sert de point d'entree vers les deux suivants |
| [`Gamergotten/Infinite-runtime-tagviewer`](https://github.com/Gamergotten/Infinite-runtime-tagviewer) | **`Plugins/food.xml`** — la structure nommee (cf. Q1.0-ter). Son `files/tagnames.txt` est anterieur a Forge : **0 / 150** de nos `type_id` |
| [`Z-15/Halo-Infinite-Tag-Editor`](https://github.com/Z-15/Halo-Infinite-Tag-Editor) | 4 fichiers de donnees, dont **`all_trimmed.txt` : 616 697 correspondances `hachage:nom` deja calculees** |
| [`Gravemind2401/Reclaimer`](https://github.com/Gravemind2401/Reclaimer) | code C# de lecture (deja utilise par le chantier pour confirmer les offsets de la chaine des triangles) — **aucun dictionnaire de noms** |

**LE RESULTAT LE PLUS SOLIDE DE CETTE PASSE — mon murmur3 est valide par une source
independante.** Le dictionnaire de Z-15 donne `gravity_hammer -> 3C2CDAAC` ; cette session
calculait `0xACDA2C3C`. **Ce sont les memes octets, en ordre inverse** : ils stockent en
gros-boutiste, le decodeur du depot lit en petit-boutiste. Verifie sur quatre noms :

| nom | leur hachage | le notre | relation |
|---|---|---|---|
| `gravity_hammer` | `3C2CDAAC` | `ACDA2C3C` | octets inverses |
| `needler` | `07E63B7A` | `7A3BE607` | octets inverses |
| `assault_rifle` | `AF1EAA12` | `12AA1EAF` | octets inverses |
| `warthog` | `B7491EF0` | `F01E49B7` | octets inverses |

La recette de Q1.0 est donc confirmee de l'exterieur, pas seulement par coherence interne.

**Ce que ces dictionnaires ne couvrent PAS, mesure et non suppose** :

- leurs `tagnames.txt` sont d'un **autre build** : sur les 3 120 `food` que leur dump liste
  pour `forge_objects-rtx-new.module` et nos 4 235, l'intersection des identifiants est
  **exactement 0**. Les identifiants de tag ont bouge entre les versions ;
- leur `all_trimmed.txt` (616 697 noms) ne contient **aucun** des huit StringID des quatre
  emplacements — ni les `Representation Name`, ni les `Crate Variant` ;
- il ne contient meme pas nos noms deja craques (`gravity_hammer` y est, mais pas
  `skull_weapon`) : c'est un dictionnaire partiel, collecte sur d'autres tags.

**Gain net de la passe** : +3 noms (`warthog_gauss`, `primitive_teleporter`, `shell_casing`),
soit **67 entrees nommees** au total, et surtout la validation externe de la methode.

### Q1.0-ter LA STRUCTURE DU TAG `food` EST NOMMEE — definition communautaire

Piste ouverte par l'utilisateur (`Surasia/InfiniteExt`, qui mene a
`Gamergotten/Infinite-runtime-tagviewer`). Ce depot porte **479 definitions de tag**, une
par groupe de quatre lettres, dans `Plugins/*.xml` — dumpees par Lord Zedd et Exhibit.

**`Plugins/food.xml` nomme exactement ce que cette session avait mesure a l'aveugle** :

```
_38  struct interne : vtable space / global tag ID / local tag handle
_41  Static IO Representation                      <- reference de tag
_2   Asset variant ...                             <- StringID
_40  Object Representations            (BLOC)      <- c'est NOTRE « bloc 1 »
       _2   Representation Name                    <- mot 0 = NOTRE hash de nom
       _2   Crate Variant                          <- mot 1, non nul sur les 4 emplacements
       _41  Configuration                          <- ref vide observee
       _41  Object Definition (Crate)              <- LA reference `weap` observee
       _41  Menu Item Definitions                  <- ref vide observee
_40  Runtime Variants (Variant Name, geo, collision, materiaux, Style)
_41  Forge Kit
_2   Default Representation
_E   Property Flags : Yaw Rotation Only · Allow In-Game Variant Toggle ·
     Disable Primary/Secondary/Tertiary Color · Disable Normal/Fixed/Phased Physics ·
     Allow Boundary Scale · Disable Rotation
```

**Correspondance parfaite avec la mesure**, champ par champ : un bloc de 92 octets
commencant par deux StringID puis trois references de tag dont seule la deuxieme est
remplie. Le type `_2` est un **StringID**, c'est-a-dire un murmur3 — ce que le
`hashnames.txt` du meme depot confirme noir sur blanc : *« any block that says mmr3Hash
derives a string from a Murmur3 Hash »*.

**Ce que cela requalifie** :

| ce que la session appelait | le vrai nom |
|---|---|
| « le mot de nom » (mot 0) | **Representation Name** |
| « le second mot, non nul sur 4 entrees » | **Crate Variant** |
| « la reference `weap` » | **Object Definition (Crate)** |
| « les deux references vides » | **Configuration** et **Menu Item Definitions** |

Les quatre entrees « emplacement » sont donc **quatre VARIANTES DE CAISSE d'une meme
representation** — et c'est le `Crate Variant` qui separe le lance-roquettes du camouflage
sur Vagabond. Ce n'est pas un identifiant 64 bits comme le supposait Q1.0-bis : **c'est
deux StringID**, donc deux noms, donc craquables — il manque le vocabulaire, pas la methode.

**Ce que ce depot n'apporte PAS** : son `files/tagnames.txt` (315 927 chemins de tag indexes
par identifiant global) **ne couvre aucun de nos tags** — controle : **0 / 150** `type_id`
tires au hasard de nos 2 785. Le dump est anterieur a Forge. Il reste utile pour d'autres
chantiers, pas pour celui-ci.

Definition conservee : `.ai/V7.5/dumps/forge_zones/reference_food_tag_definition.xml`.

### Q1.0-bis LES QUATRE EMPLACEMENTS RESISTENT — ce qui a ete tente, et refute

Quatre voies essayees pour les nommer, quatre negatives. Ecrites ici pour qu'on ne les
rejoue pas.

| voie | mesure | verdict |
|---|---|---|
| dictionnaire elargi | tous les jetons snake_case du binaire, non ancres — **105 291 mots** apres fusion des deux extractions | **0 correspondance** sur les 4 mots de nom |
| le SECOND mot porte-t-il le nom ? | les 4 valeurs `0xFAB48286` `0xF2A5966F` `0xB159316C` `0xF0AA4ACC` contre le meme dictionnaire | **0 correspondance** — ni l'un ni l'autre des deux mots n'est un nom |
| vocabulaire cible armes/power-ups | `rocket_launcher`, `active_camouflage`, `overshield`, `weapon_spawner`… en combinaisons jusqu'a profondeur 3 (666 159 essais) | **0 correspondance** |
| Ghidra — la constante murmur3 | `0xCC9E2D51` : **plus de 700 occurrences** dans le binaire | trop diffus pour cibler ; nommer ces entrees demande de retrouver le **parseur du tag `food`**, c'est une session de retro-ingenierie a part entiere |

**Ce que le negatif apprend quand meme** : puisque, chez les 4 209 autres entrees, le second
mot vaut 0 et le premier est un nom, la paire `(mot0, mot1)` de ces quatre-la n'obeit pas au
meme schema. L'hypothese la plus economique est un **identifiant sur 64 bits** (asset), pas
deux noms — ce qui est coherent avec le fait qu'aucune moitie ne se craque.

**Le test du RATELIER, sur l'avertissement de l'utilisateur** (« il y a des rateliers sur
certains murs, ce sont des emplacements pour des armes de base — ne pas confondre »). Un
ratelier est fixe au mur : son vecteur `Up` n'est pas vertical. Mesure sur les 199 cartes :

| type_id | n | pose au SOL | fixe au MUR |
|---|---:|---:|---:|
| `1486653438` | 98 | 90 % | 10 % |
| `-1062552774` | 46 | 89 % | 9 % |
| `1882451900` | 117 | 96 % | 3 % |
| `801517767` | 24 | **100 %** | 0 % |
| `-721267272` (`generic_ball`) | 66 | 82 % | 0 % |
| `-1342546397` (`skull_weapon`) | 21 | **100 %** | 0 % |

**Aucun des quatre n'est un ratelier mural** : ils sont poses au sol a 89-100 %. Les rateliers
sont donc un AUTRE type d'objet, encore non identifie — la confusion signalee par
l'utilisateur n'a pas eu lieu, et le test le prouve plutot que de le supposer.

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

### Q2 — LE TEMOIN EST JOUE : un joueur qui capture une zone est DEDANS *(2026-08-02)*

Ce temoin manquait a la premiere passe. Il est desormais produit, et **il tranche le depart
demi-extents / tailles pleines** que les mesures hors ligne ne separaient pas.

**Ce qu'il a fallu produire** (autorise par le superviseur le 2026-08-02) :

1. **Les bornes de dequantification de Vagabond**, absentes du catalogue. Ajout de
   `"Vagabond": "fo08_wetland"` a la table de `cmd/mapquant-build`, avec sa RAISON ecrite :
   le `level_id` 88891201 du `.mvar` rend exactement une occurrence sur les 88 modules
   (groupe `levl`), preuve etablie hors de toute mesure de largeur (plan maitre §J0.2).
   Catalogue regenere : **15 cartes**, Vagabond W=15/15/17, etendue 462,6 / 453,4 / 1188,5.
2. **L'artefact de rejeu du film `696a9d7c`** : 110 traces, **31 216 points**, 5 337 images
   a 100 ms, 533,7 s. Verdicts du pont slot->joueur, des tirs et des grenades : **nominaux**.

**Le protocole** : aux **quatre instants du releve terrain** de J0.6, on demande quels
joueurs sont dans quelle zone, sous chacune des deux lectures, avec un **temoin negatif** —
les memes formes translatees de 12 m en x et en y. Test 3D : la hauteur compte.

| instant du releve | ce que le releve dit | reels DEMI | temoin DEMI | reels PLEIN | temoin PLEIN |
|---|---|---:|---:|---:|---:|
| 48 s | flyguy8773 capture la base B | 2 | **0** | 2 | **0** |
| 90 s | une equipe controle les trois bases | 2 | **3** | 1 | 1 |
| 190 s | score 69-30 | 3 | **4** | 1 | **0** |
| 334 s | controle des trois bases | 3 | **0** | 2 | **0** |
| **total** | | **10** | **7** | **6** | **1** |

**Le verdict.** Sous la lecture « demi-extents », les zones sont si grandes que **le temoin
negatif attrape AUTANT voire PLUS de joueurs que les vraies zones** (3 contre 2 a 90 s,
4 contre 3 a 190 s) : etre « dedans » n'y est pas informatif. Le rapport signal/temoin vaut
**10 / 7 = 1,4**. Sous la lecture « tailles pleines », il vaut **6 / 1 = 6,0**.

**La lecture « tailles pleines » est donc confirmee par un oracle d'execution**, et non plus
seulement par les 11 coincidences cylindre/boite. Les deux mesures independantes concordent.

**Et le temoin nominal tombe juste** : a 48 s, le releve terrain dit « capture de la base
B », une seule base ; le test rend **exactement une zone occupee** (la zone 2, par deux
joueurs), sous les deux lectures, avec un temoin negatif a **0**. Un joueur qui capture une
zone est bien dedans.

Outil : `cmd/tmp_zonetest <artefact.json> <carte.mvar> <t_secondes...>`.

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

## LE CONTRAT DE DONNEES — `map_objectives.json` **LIVRE en schema_version 2** *(2026-08-02)*

> **N'est plus une proposition.** Le schema est implemente, teste et le catalogue est
> regenere. Le RENDU, lui, n'existe toujours pas — et pour une raison mesuree : le
> catalogue n'a **aucun lecteur**, ni Go ni web (grep sur `MapObjectivesPath` et
> `map_objectives` : zero consommateur). Le brancher est un chantier a part entiere,
> tenu par `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md` etapes 1-2.

**Ce qui est livre :**

| element | ou |
|---|---|
| lecture du sac de forme | `internal/analysis/replay/mapvar/shape.go` (`readShape`, `Object.Shape()`) |
| brut conserve dans le record | `mapvar.Object.ShapeRaw` |
| champ de sortie | `mapvar.Objective.Shape` (`omitempty` — absent sur un ponctuel) |
| bump de schema + note de contrat | `cmd/mapobj-build/catalog.go` (`catalogSchemaVersion = 2`) |
| **regeneration HORS LIGNE** | `cmd/mapobj-build --refresh-from <dossier de .mvar>` (`refresh.go`) |
| catalogue regenere | 34 cartes, 597 objectifs, **264 avec forme** (157 boites, 107 cylindres), 333 ponctuels sans forme |

**Le mode `--refresh-from` existe parce qu'ajouter un champ au contrat ne doit pas
obliger a re-solliciter l'UGC pour 34 cartes.** Il re-parse les `.mvar` locaux en
appariant par `mvar_file`, PRESERVE les metadonnees issues du reseau (`map_id`,
`version_id`, `public_name`, `fetched_at`), et signale bruyamment toute carte dont le
fichier manque plutot que de l'effacer. Migration v1 -> v2 : 34/34, zero manquant.

**Quatre tests ancrent la lecture** (`mapvar_test.go`) :

1. `TestShapeAnchorCliffhangerStronghold` — l'objet 178 de `cliffhanger_map.mvar` porte
   exactement les valeurs brutes citees ci-dessous (s5=445644, s6=393216, s7=262144, s8=0)
   et doit rendre 3,400 x 3,000 m de demi-extents, 4 m au-dessus du centre ;
2. **`TestShapeFullSizeReadingBeatsHalfExtent`** — LE temoin : le cylindre 185 (r = 5,0999)
   et la boite 188 (s5 = 668441) de la meme carte coincident **au dixieme de millimetre**
   sous la lecture retenue, et seraient a un facteur 2 sous la lecture rejetee ;
3. `TestShapePresenceFollowsSurfaceRule` — 100 % des surfaciques avec forme, 0 % des
   ponctuels ;
4. `TestShapeDerivesFromRaw` — le derive ne peut pas diverger du brut.

**Piege confirme au passage** : `mapobj-build` resout la racine du depot en remontant
l'arborescence. Depuis un worktree il ecrit dans l'ARBRE PRINCIPAL — exporter
`LEVELUP_REPO_ROOT` avant de le lancer.

Le schema tel qu'il sort aujourd'hui, et les cinq regles qui le justifient :

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
