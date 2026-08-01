# ETAT DE L'ART — la recette mode vers score, et le systeme d'evenements du film

> Session de recherche du 2026-08-01. Branche `feat/re-mode-score` (worktree dedie).
> Outillage neuf, jetable : `cmd/tmp_modescore`, `cmd/tmp_modestate`, `cmd/tmp_modeticks`.
> Rien de livre n'a ete touche : ni `filmdec/`, ni `objectiveevents/`, ni aucun code de prod.
>
> Regle d'ecriture appliquee partout : **« N valeurs observees sur ces films »**, jamais
> « le jeu compte N ». Chaque verdict porte ses chiffres et son controle negatif.

---

## 0. CE QUI A CHANGE PENDANT LA SESSION — a lire avant le reste

Deux corrections d'entree, etablies sur pieces, qui invalident des premisses de l'ordre de
mission lui-meme :

**(1) `01e1f945` n'est PAS un Slayer.** L'ordre de mission le presente comme l'un des deux
« controles Slayer ». La base dit `KOTH:Arena` sur Catalyst, 540 s, score 3-2. Le seul
controle Slayer reel du corpus est `000d5950` (`Slayer:Arena Super Fiesta`, Cliffhanger,
43-50). C'est une **bonne** nouvelle : le corpus couvre en fait **quatre** modes
(Slayer, KOTH, CTF x2, Strongholds) au lieu de trois, donc un contraste inter-mode plus fort.

**(2) L'utilisateur a ouvert une source externe en cours de session** — le depot
`davidhouweling/guilty-spark`, PR 752 a 757, qui parse le meme format de film pour construire
des courbes de score par mode. Cette source enonce **exactement l'hypothese Q3** (« un signal
d'etat d'objectif universel, tous modes confondus »). Elle a donc ete traitee comme ce qu'elle
est : une hypothese a refuter, pas une reference. Deux de ses revendications sont **refutees
par la mesure** ci-dessous ; une troisieme est **confirmee** ; et sa table mode -> statistique
fournit une **chaine independante** qui a servi a Q4.

---

## 1. VERDICTS, EN UNE PAGE

| # | Question | Verdict | Ce qui le tient |
|---|---|---|---|
| Q1 | Cartographie du systeme d'evenements | **ETABLI (partiel)** | Recensement mesure sur 5 films / 4 modes ; appairage type-10 -> type-0 exact ; un code discriminant par famille de mode |
| Q2 | Porteur autoritaire du score d equipe | **ETABLI en Slayer · ABSENT de l espace balaye en Strongholds · INDECIDABLE en CTF-KOTH** | Slayer : serie de 26 valeurs reproduite, leurres a zero. Ailleurs : un candidat credible DEMASQUE par les ancres terrain (6bis) |
| Q3 | La recette mode -> score | **ETABLI — et la reponse deplace la question** | L intermediaire existe mais ce n est PAS du code : registre de stats plat adresse par identifiant, sans aucun branchement de mode ; c est le SCRIPT de la variante qui choisit l identifiant (5.4) |
| Q4 | Evenements d'objectif nommes | **ETABLI pour les zones · REFUTE pour CTF · INDECIDABLE pour KOTH** | Zones : 3 chaines independantes, egalite EXACTE 8/8 par joueur |
| Q5 | Famille pickup (sous-produit) | **CARTOGRAPHIE** | Bornes inferieures de changements d arme par film et par mode, avec controle de vraisemblance Fiesta (section 7) |

**Le resultat de la session, en une phrase** : en mode a zones, un evenement de mode du footer
type-3 **EST** une prise ou une securisation de zone, par joueur, a l'unite — total 77 = 77 et
**8 joueurs sur 8** en egalite exacte, avec l'ancre terrain qui tombe sur l'acteur ET sur la
seconde ; et l'hypothese d'un intermediaire universel mode -> score n'a **aucun** support
mesure sous la forme cherchee — mais Ghidra en donne une autre : le moteur n a AUCUN branchement
de mode sur le chemin de lecture du score, c est le script de la variante qui choisit
l identifiant de stat (5.4). La recette est de la DONNEE, pas du code.

---

## 2. LE CORPUS ET SES ORACLES

| film | mode (base) | carte | duree | score API |
|---|---|---|---|---|
| `000d5950` | Slayer:Arena Super Fiesta | Cliffhanger | 496 s | 43-50 |
| `01e1f945` | **KOTH:Arena** | Catalyst | 540 s | 3-2 |
| `64e8adfa` | CTF:Arena | Catalyst | 839 s | 2-3 |
| `530820e5` | CTF:Arena | Catalyst | 475 s | 3-0 |
| `696a9d7c` | Strongholds:Arena | Vagabond | 561 s | 200-94 |

Controle de coherence du Slayer : score API 43-50 = frags par equipe 43-50 (somme
`match_participants.kills`). Le score du Slayer EST le compte de frags — la prediction de
l'ordre de mission tient, et elle sert de garde-fou a tout le reste.

Oracles utilises : `match_registry` (score d'equipe), `match_participants` (roster xuid ->
equipe ; l'equipe ne vient JAMAIS du champ team du film, connu non fiable),
`match_objective_stats_latest` (compteurs d'objectif par joueur), et le relevé terrain ecrit
`.ai/RELEVE_TERRAIN_CAPTURES_2026-07-31.md` (4 ancres a l'oeil sur `696a9d7c`).

---

## 3. Q1 — CARTOGRAPHIE DU SYSTEME D'EVENEMENTS

Outil : `cmd/tmp_modescore <cacheRoot> <filmID>[:label] ...`

### 3.1 Le recensement des paquets — un appairage exact, jamais consigne

Sur les cinq films, le nombre de paquets de type 10 **egale exactement** le nombre de paquets
de type 0 (30 418 / 30 418 · 33 102 / 33 102 · 50 956 / 50 956 · 29 148 / 29 148 · 34 276 /
34 276). Verifie sur pieces par vidage des paquets d'un chunk : la structure est
`[type-10, 10 octets][type-0, ~210-250 octets]` repetee, un couple toutes les ~16 ms.

Ce n'est pas un artefact de parcours : le vidage montre l'alternance stricte, avec des
horodatages moteur croissants (6643521026, 6643527860, 6643544363...). **Le type-10 est un
en-tete de trame de 10 octets qui precede chaque trame de replication.** Notre
`filmdec/film_packets.go` le traverse sans le lire. C'est une donnee gratuite, jamais exploitee.

### 3.2 Le premier enregistrement de chaque trame — histogramme par mode

Mesure du champ 7 bits `payload[0] >> 1` du paquet type-0 (la convention du depot, cf.
`filmdec/fire_events.go`). **Attention a ce que ce chiffre est** : c'est le type du PREMIER
enregistrement de la trame, pas « le type de la trame ».

| code | Slayer `000d5950` | KOTH `01e1f945` | CTF `64e8adfa` | CTF `530820e5` | Strongholds `696a9d7c` |
|---|---|---|---|---|---|
| 80 (0x50) | 26 548 | 27 750 | 44 058 | 24 928 | 27 716 |
| 116 (0x74) | 587 | 898 | 1 002 | 646 | 1 138 |
| 105 (0x69) | 832 | 2 535 | 3 395 | 2 167 | 2 649 |
| **123 (0x7b)** | **0** | **117** | **0** | **0** | **249** |
| **125 (0x7d)** | **0** | 0 | **4** | 0 | 0 |
| 122 (0x7a) | 0 | 1 | 0 | 1 | 0 |
| 82 (0x52) | 1 | 1 | 0 | 0 | 1 |

(Codes presents partout, non discriminants : 64, 68, 69, 96, 97, 98, 99, 101, 107, 114, 115,
121. Total 16 a 18 codes distincts par film.)

**Le temoin discriminant** : le code **123 n'apparait que dans les deux modes a zones**
(KOTH 117, Strongholds 249) et **zero fois** dans le Slayer comme dans les **deux** CTF. Deux
positifs, trois negatifs. La CTF `64e8adfa` et le KOTH `01e1f945` sont **sur la meme carte**
(Catalyst) : le signal suit donc le mode, pas la carte — c'est le controle qui donne sa valeur
au resultat.

**Chaine independante** (source externe, `shared/src/halo/objective-stats.ts`) : l'API Halo
range Strongholds **et** KOTH sous le **meme** bloc de statistiques `ZonesStats`
(`getStrongholdsObjectiveStats` est typee `MultiplayerStrongholds | MultiplayerKingOfTheHill`),
la ou CTF a son `CaptureTheFlagStats` et Oddball son `OddballStats`. Deux voies sans aucune
etape commune — un histogramme de flux binaire d'un cote, une table de mapping d'API
communautaire de l'autre — **designent le meme regroupement de modes**.

Le code 125 n'apparait que sur une seule des deux CTF (4 occurrences, chunks 30-31) : **un seul
echantillon, aucune conclusion** — consigne, pas interprete.

### 3.3 Le footer type-3 — histogramme des `type_hint`

| film | mode | th10 (mode) | th20 (morts) | th50 (frags) | th100 | th150 | total | medailles |
|---|---|---|---|---|---|---|---|---|
| `000d5950` | Slayer | **0** | 93 | 112 | 25 | 0 | 230 | 44 |
| `01e1f945` | KOTH | 66 | 105 | 134 | 18 | 1 | 324 | 47 |
| `64e8adfa` | CTF | 68 | 123 | 149 | 27 | 1 | 368 | 56 |
| `530820e5` | CTF | 43 | 94 | 108 | 15 | 5 | 265 | 36 |
| `696a9d7c` | Strongholds | 77 | 103 | 127 | 9 | 2 | 318 | 36 |

**Le controle negatif que la mission demandait est passe** : le mode sans aucun objectif porte
**exactement zero** evenement de mode, quand les quatre modes a objectif en portent 43 a 77.

### 3.4 TYPE_2 — le jeton de score

Le jeton 12 bits `0x7B6` (l'ancre du bloc score etablie par l'archive) est present dans
**100 % des images-cles des cinq films** (26/26, 28/28, 43/43, 25/25, 29/29), a une position
qui derive d'un film a l'autre (premier hit entre les octets 727 et 847). La regle de l'archive
— « lecture ancree sur un marqueur local par match, jamais d'offset en dur » — est reconfirmee,
et le jeton est **universel au mode** : il ne discrimine rien, il localise.

---

## 4. Q4 — LES EVENEMENTS D'OBJECTIF NOMMES (le resultat fort de la session)

Outil : `cmd/tmp_modeticks <cacheRoot> <rosterTSV> <filmID:s0-s1:mode> ...`

Note de decodage : l'horloge des evenements de mode est **big-endian** aux octets 48-51 du bloc
(convention du depot, `objectiveevents/film.go`). Une lecture little-endian rend des valeurs de
plusieurs heures — hors de toute duree de match. Le controle est gratuit et il attrape l'erreur.

### 4.1 ZONES (Strongholds `696a9d7c`) — ETABLI, trois chaines independantes

**Chaine 1 — l'ancre terrain.** Le tout premier evenement de mode du match tombe a
**t = 48,90 s, equipe 0, FlyGuy8773**. Le relevé terrain, ecrit a l'oeil le 2026-07-31 **avant**
tout decodage, dit : « **0:48 — flyguy8773 capture la base B** ». L'acteur ET la seconde
concordent. Second point : le relevé note « 1:30 — une equipe controle les trois bases » ; le
decodage porte une **rafale de quatre evenements d'equipe 0** a t = 89,29 / 89,96 s
(Otti1614, SunburntMonk740, NeonKnight3166, FlyGuy8773), soit 1:29-1:30.

**Chaine 2 — le compte total.** 77 evenements de mode decodes du film ;
`SUM(zone_captures + zone_secures)` de l'API sur les 8 joueurs = **77**.

**Chaine 3 — l'egalite par joueur, le critere fort.** Les deux multisets sont **identiques** :

| compte film | 16 | 10 | 10 | 10 | 9 | 9 | 7 | 6 |
|---|---|---|---|---|---|---|---|---|
| `zone_captures + zone_secures` API | 16 | 10 | 10 | 10 | 9 | 9 | 7 | 6 |

et les deux joueurs dont le gamertag est resolu par ailleurs tombent nominativement :
NeonKnight3166 = 16 (xuid 2535458126310341 : 10 + 6) et JGtm = 9 (xuid 2533274823110022 :
7 + 2). **8 joueurs sur 8, a l'unite.**

**Controle negatif** : le Slayer porte 0 evenement de mode et 0 statistique de zone.

**Ce que ca rend decodable** : en mode a zones, la timeline « qui a pris/securise une zone, et
quand » est lisible hors ligne, avec l'acteur (gamertag decode du film) et la milliseconde.

**Le TIMING est livre, pas seulement le compte** : chaque evenement porte son horodatage en
ms sur l horloge du match (octets 48-51, BIG-endian) et son acteur (gamertag UTF-16 decode du
film, aucune jointure DB necessaire). `VERBOSE=1 cmd/tmp_modeticks` sort la timeline complete.
Extrait verifie sur `696a9d7c` — le premier evenement du match et la rafale des trois bases :

```
t=  48.90s  equipe 0  FlyGuy8773          <- releve terrain : « 0:48 flyguy8773 capture la base B »
t=  53.56s  equipe 1  Madina97294
t=  53.56s  equipe 1  AG x GibsoN Zz
t=  72.94s  equipe 0  FlyGuy8773
t=  89.29s  equipe 0  Otti1614
t=  89.29s  equipe 0  SunburntMonk740
t=  89.96s  equipe 0  NeonKnight3166      <- releve terrain : « 1:30 trois bases »
t=  89.96s  equipe 0  FlyGuy8773
```

L horloge du footer et le `start_ms` du manifeste sont sur la MEME base : les evenements sont
donc directement replaçables sur la ligne de temps du rejeu, sans recalage.
Ce qui **reste** hors de portee : **quelle** zone (A/B/C) — le resultat negatif de l'archive
n'est pas entame par cette session.

### 4.2 CTF — REFUTE (l'identite simple ne ferme pas)

Hypothese posee **avant** la mesure (celle de l'archive : evenement de mode = interaction de
drapeau, prise/retour/capture) :

| film | evenements de mode (film) | `flag_captures + flag_grabs + flag_returns` (API) |
|---|---|---|
| `64e8adfa` | 68 | **90** |
| `530820e5` | 43 | **33** |

L'ecart change de **signe** d'un film a l'autre : aucun facteur d'echelle, aucune fenetre de
deduplication ne reconcilie les deux. Et les multisets par joueur sont disjoints
(film `64e8adfa` : 16, 12, 9, 8, 6, 6, 6, 5 — API : 28, 22, 13, 8, 7, 5, 5, 2).

**Consequence directe pour `PLAN_OBJECTIFS_TEMPS_REEL` etape 2.3** : sa consigne « ne rien
etendre au CTF sur la base des zones » etait une prudence ; elle est desormais **une mesure**.

### 4.3 KOTH — INDECIDABLE, et la raison est une donnee corrompue chez nous

Le total tombe (66 evenements de mode / 66 en somme API) mais les multisets par joueur
different (film : 13, 10, 9, 9, 8, 6, 6, 5 — API : 10, 9, 8, 8, 8, 7, 7, 6).

**Avant de conclure quoi que ce soit sur le format, la source de reference a ete regardee** :
`match_objective_stats_latest` porte **9 lignes pour un match a 8 joueurs**, dont une avec un
xuid non numerique (`bid(42.0`). La reference est donc **corrompue** pour ce match. Les deux
branches — « KOTH differe de Strongholds » et « notre ingestion est cassee » — restent ouvertes,
et aucune n'est tranchable sur ce film. **Il faut un second film KOTH.** (Defaut consigne en 8.)

---

## 5. Q2 ET Q3 — CE QUI EST ELIMINE, ET LA ROUTE QUI RESTE

### 5.1 ELIMINE — « l'octet d'etat du payload type-2 » (revendication externe, PR 752)

La source externe affirme : « l'octet 2 du payload type-2 est un compteur de periode de jeu,
**signal d'etat d'objectif universel sur les quatre modes** ; il s'incremente a chaque
transition d'objectif et vaut 0xa0/0x00 hors jeu ». Traduit dans notre vocabulaire, leur
marqueur `a0 7b 42` est le debut du payload d'un paquet type-0 (code 80) et leur octet est
`payload[5]`. Nous lisons en plus l'horodatage **reel** du paquet, la ou eux interpolent.

Outil : `cmd/tmp_modestate`. Resultat :

| film | mode | paquets marques | transitions | dont vers 0x40-0x9f |
|---|---|---|---|---|
| `000d5950` | **Slayer (aucun objectif)** | 24 899 | **2 808** | **1 429** |
| `01e1f945` | KOTH | 25 478 | 1 105 | 836 |
| `64e8adfa` | CTF | 36 274 | 1 195 | 800 |
| `530820e5` | CTF | 22 922 | 935 | 682 |
| `696a9d7c` | Strongholds | 25 392 | 1 051 | 567 |

**Le controle negatif tue la revendication** : le mode **sans aucun objectif** porte **plus**
de transitions (2 808) que le Strongholds (1 051) ou les deux CTF (935 et 1 195). Un signal
d'etat d'objectif ne peut pas etre maximal la ou il n'y a pas d'objectif.

Argument de densite, qui ferme la porte au repli « oui mais il faut filtrer » : ~1 000
transitions sur ~500 s font **une transition toutes les 0,5 s**. La coincidence avec les 3
captures d'une CTF est garantie par la densite seule : le signal ne porte aucune information
discriminante, quel que soit le filtre.

**Confirmation par une seconde ligne, et elle vient de la source elle-meme** : la PR 757 du
meme depot declare que la version precedente « contenait un algorithme fondamentalement casse »
et la remplace par un empilement de constantes reglees — `MIN_PRE_PERIOD_MS = 500 ms` dont le
commentaire dit explicitement **« filtre le bruit de byte2 »**, `CAPTURE_RECENCY_THRESHOLD_MS
= 7000 ms`, une « heuristique de divisibilite ». Ils se battent contre le bruit que nous avons
mesure.

### 5.2 ELIMINE — « le score est le compte de ticks d'evenements de mode » (PR 753/754)

Critere : **egalite exacte**, pas correlation. Deduplication a 2,5 s par equipe, exactement
l'algorithme externe.

| film | mode | dedup 2,5 s (t0-t1) | score API | verdict |
|---|---|---|---|---|
| `01e1f945` | KOTH | 25-28 | 3-2 | ECHEC |
| `64e8adfa` | CTF | 24-31 | 2-3 | ECHEC |
| `530820e5` | CTF | 24-14 | 3-0 | ECHEC |
| `696a9d7c` | Strongholds | 28-22 | 200-94 | ECHEC |

**0 sur 4.** Le cas Strongholds est definitif independamment de toute fenetre : 41 evenements
d'equipe 0 ne peuvent pas produire 200 points.

**Mais leur observation de periodicite, elle, est CONFIRMEE — et par un oracle interne** :
en KOTH la mediane des intervalles entre evenements consecutifs d'une meme equipe vaut
**5 005 ms (equipe 0) et 5 009 ms (equipe 1)** sur 64 intervalles. Une periodicite de 5,00 s,
mesuree sans aucune reference externe. Et elle est **propre au mode** : Strongholds donne
11,8 / 11,5 s de mediane avec un premier quartile a 0 (rafales multi-zones), CTF donne 15,0 /
9,9 s sans periodicite.

**La lecon utile** : l'evenement de mode est **le meme porteur** dans tous les modes, mais sa
**cadence et sa semantique changent avec le mode** — tick d'occupation a 5 s en KOTH,
prise/securisation discrete en zones, interaction de drapeau en CTF. C'est un argument **contre**
la lecture « une recette unique », et c'est mesure.

### 5.3 Q3 — pourquoi la piste « table de dispatch par event-type » est cassee

Point de depart : `filmdec/fire_events.go` affirme que le deserialiseur d'un event de type `t`
est `vtable[t] + 0x68`, table de handlers a `0x144724A90`.

**Le controle positif passe, et de facon spectaculaire** : a l'index 105,
`*(0x143D0ACA0 + 0x68) = 0x14080C1F8` — exactement le deserialiseur de degat connu du projet —
et le nom lu par la recette du depot (`vtable + 0x08` -> thunk `LEA RAX,[chaine] ; RET`) rend
**`action_weapon_fire`**. Deux confirmations independantes au meme index.

**Et pourtant la table ne se generalise pas.** Aux autres index, les noms sont incoherents avec
ce que le flux montre :

| code | occurrences mesurees | nom lu a `vtable + 0x08` |
|---|---|---|
| 105 | 832 a 3 395 | `action_weapon_fire` — **coherent, c'est le controle** |
| 80 | 24 928 a 44 058 (**71-85 % des paquets**) | `NavpointRequest` — impossible |
| 116 | 587 a 1 138 | `player_set_orbiting_camera_target` |
| 96 | 105 a 552 | `AILand` |
| 123 | 0 ou 117-249, **propre aux zones** | `biped_debug_teleport` |
| 125 | 4, une seule CTF | `vehicle_auto_turret_choose_target` — sur une CTF Catalyst **sans vehicules** |

Le desassemblage du deserialiseur du code 80 donne un mince adaptateur
(`FUN_142EF8A40 -> FUN_142EF4A98`), **pas** la boucle d'enregistrements de trame
`FUN_1406CD128` qu'un enregistrement present dans 85 % des paquets devrait etre.

**Verdict Q3, honnete** : l'index de cette table **n'est pas** notre champ 7 bits, sauf
coincidence a 105. Ce constat rejoint — et precise — le resultat L2 deja consigne dans
`KILLFEED_STATE.md` : « la table n'est PAS les classes d'event, c'est un registre EXHAUSTIF de
tout ce qui est serialisable », navigable par schema et jamais indexe par instruction. **La
recette mode -> score n'est donc pas dans cette table**, et il ne faut pas y retourner par ce
chemin.

**Ce qui reste ouvert et cadre** (la route pour la prochaine session) : le systeme de stats
(« statborg ») de `RE_EXE_GHIDRA_FINDINGS.md` §1 — 48 statlines de 0x1DF0 octets, valeur de
round a `statline + 0x38 + round*4`, pas de 0x88 par stat, dispatch de lecture a 4 types
`FUN_1406AEE98`, table d'echelle `DAT_143CE70A8` indexee par `descripteur + 0xBC`. **La question
a poser est : qui choisit le SLOT de stat selon la categorie de variante ?** L'API, elle,
tranche par `GameVariantCategory` (union discriminee, chaine externe) — ce qui **suggere** un
selecteur equivalent cote moteur, mais rien dans cette session ne l'a montre. `filmdec/statborg.go`
porte deja le deserialiseur a deux equipes ; le maillon manquant est le **binding** (les deux
index de slot que le moteur lit a `param_1 + 0x8` / `+ 0xc`).

### 5.4 Q3 — LA REPONSE, obtenue par Ghidra : l'intermediaire existe, mais ce n'est PAS du code

La chaine a ete remontee **du consommateur vers l'amont**, comme le prescrit
`METHODE_RETRO_INGENIERIE_FILM.md` §1. Elle se ferme.

**Le maillon 1 — le getter.** `FUN_1406ADA4C` (decompile) lit exactement :

```
valeur = *(int32*)( world + slot*0x88 + statline*0x1DF0 + 0x38 + round*4 )
affichee = valeur * DAT_143CE70A8[ *(byte*)(descripteur + 0xBC) ]
```

`slot` et `descripteur` ne sont pas calcules ici : ils arrivent **par parametre**
(`param_4[1]` et `*param_4`). Le `statline` vient d'une resolution par table de hachage
(FNV-1a sur l'identifiant d'equipe), pas d'un index direct.

**Le maillon 2 — la resolution.** `FUN_140C18E54` -> `FUN_140C18EA8` : un **balayage
lineaire** de la table de descripteurs a `engine + 0xDF77C`. Le compte est au premier dword,
les entrees commencent a `+8`, **le premier dword de chaque entree est l'identifiant de stat**
compare a l'argument, et le pas est de `0x30` **dwords = 0xC0 octets**. L'index trouve **EST**
le slot.

> Correction a `RE_EXE_GHIDRA_FINDINGS.md` §1, qui annonce « entrees 0x30 » : le pas est de
> 0x30 **dwords**, soit **0xC0 octets**. Le decompile est sans ambiguite (`piVar3 + 0x30` sur
> un `int*`).

**Le maillon 3 — l'appelant, et c'est lui qui tranche.** `FUN_142B7974C` est le corps natif du
binding **HavokScript** `Team_GetCurrentRoundStatValue`. Il ne fait que trois choses : prendre
la table `engine + 0xDF77C`, y resoudre **l'identifiant de stat que le SCRIPT lui passe**, et
appeler le getter.

**Le controle qui donne sa valeur au resultat** : sur tout ce chemin —
`FUN_1406ADA4C`, `FUN_140C18E54`, `FUN_140C18EA8`, `FUN_142B7974C` — il n'y a **aucun
branchement sur une categorie de variante de jeu, aucun switch de mode, aucune table indexee
par mode**. La seule chose qui varie d'un mode a l'autre est **l'identifiant de stat passe en
argument**.

Et la surface d'API scriptee, lue dans `.rdata` autour de `0x1436DEF00`, le confirme : elle est
**entierement generique**, sans une seule fonction propre a un mode —
`Team_GetCurrentRoundStatValue`, `Team_GetMatchStatValue` (+ variantes `Decimal`),
`Team_AdjustCountStat`, `Team_SetRawStat`, `Team_ConsiderNewMinMaxStatValue`. Recherches de
controle : **`Hill_` = 0 occurrence** ; `Objective_` ne rend qu'un `Objective_GetEnabled`
generique et des chaines de HUD/campagne. La presence de `temp_objective_fragments.lua`
acheve de situer la logique de mode.

**VERDICT Q3.** L'intermediaire postule par l'hypothese **existe**, mais il n'est ni dans
l'en-tete du film, ni une entite des images-cles, ni une table de dispatch du binaire — les
trois pistes que l'ordre de mission demandait de trier. C'est un **registre de stats plat,
adresse par identifiant**, partage par tous les modes, et **c'est le SCRIPT de la variante de
mode qui choisit l'identifiant**. Le moteur, lui, ne sait pas a quel mode il joue.

Autrement dit : **la recette n'est pas du code, c'est de la donnee** — elle vit dans l'asset de
variante de mode. C'est la seule lecture qui explique d'un coup les trois resultats negatifs de
cette session : pas d'octet d'etat universel (5.1), pas de comptage de ticks universel (5.2),
et un meme porteur d'evenement dont la semantique change avec le mode (5.2). Il n'y a rien a
chercher cote binaire parce qu'il n'y a rien **a** trouver cote binaire.

**Ce que ca rend decodable, et ce que ca coute** : la voie « lire le score generiquement pour
tous les modes » est **fermee cote RE statique**. Ce qui reste ouvert et vaut le detour : le
**vocabulaire des identifiants de stat** (la table `engine + 0xDF77C` est enumerable hors ligne
— compte, identifiants, echelles) ; croise avec le fait que le film serialise le statborg
(`filmdec/statborg.go` porte deja le deserialiseur a deux equipes), il donne la liste de ce qui
EST mesure par le jeu, sans avoir a deviner. C'est un lot borne pour une prochaine session.

**« Il n'y a pas de recette » est donc ECARTE, mais pas dans le sens attendu** : il y a bien un
intermediaire unique et universel — il n'est simplement pas la ou les trois pistes le
cherchaient.

---

## 6. CE QUE LA SESSION PROPOSE POUR LE MASTER PLAN §2 (a arbitrer, NON APPLIQUE)

Conformement a la consigne, ces corrections sont **proposees**, pas ecrites dans le master plan :

1. `PLAN_OBJECTIFS_TEMPS_REEL` etape 2.3 — passer de « ne rien etendre au CTF » (prudence) a
   « l'identite zones ne vaut PAS en CTF, mesure sur 2 films, ecart de signe oppose ».
2. `PLAN_OBJECTIFS_TEMPS_REEL` etape 3 — l'identite « 1 evenement de mode = 1 prise ou 1
   securisation, par joueur, a l'unite » est **etablie pour les zones** ; l'etape 3 peut donc
   livrer une timeline de zones **avec acteur et milliseconde** sans lire un seul composant
   d'objectif du flux de replication. Le decodage des 34 composants reste necessaire pour la
   **progression** d'une capture et pour l'identite de la zone, pas pour la timeline.
3. Ajouter aux impasses connues, avec leurs chiffres : l'octet d'etat type-2 (5.1) et le
   comptage de ticks (5.2). Les deux ont ete publies par une source externe credible ; sans le
   controle negatif, ils auraient coute une session.
4. Le corpus de reference du chantier doit corriger `01e1f945` : **KOTH, pas Slayer**.

---

## 6bis. Q2 — LE PORTEUR DU SCORE : LOCALISE EN SLAYER, ET UN FAUX POSITIF DEMASQUE AILLEURS

Outils : `cmd/tmp_scorefind` (recherche + leurres), `cmd/tmp_scoreread` (application d'une
regle, sans recherche), `cmd/tmp_scoreanchor` (recherche contrainte par les ancres terrain).

L'alignement d'horloge ne vient plus d'une estimation : le **manifeste de film** (en cache,
`data/cache/film_manifests/<id>.json`) donne le `start_ms` de chaque chunk. Les images-cles
sont donc datables a la milliseconde et confrontables aux evenements horodates du footer.

### 6bis.1 SLAYER — ETABLI, serie complete et leurres a zero

Oracle exact et **independant du film** : le score d'une equipe est le nombre de morts de
l'autre, et les morts sont horodatees dans le footer (th=20, 93 evenements). Cela donne une
serie attendue de **26 valeurs**, pas une valeur finale.

Regle trouvee, ancree sur la premiere occurrence du jeton 12 bits `0x7B6` :

| equipe | offset | largeur | echelle |
|---|---|---|---|
| 0 | **ancre + 28 bits** | 6 | x1 |
| 1 | **ancre + 110 bits** | 6 | x1 |

Courbes lues, `000d5950` :

```
equipe0 : 0 0 0 2 3 3 5 7 10 11 15 17 20 21 22 24 25 26 27 30 33 35 35 38 41 43
equipe1 : 0 0 0 4 7 7 9 11 13 16 17 20 23 23 24 27 30 34 36 38 40 40 43 46 49 50
```

Toutes deux monotones, fin **43-50** = le score de l'API, exactement.

**Le comptage des faux positifs, sur le flux reel** — c'est ce qui rend la mesure publiable :

| serie soumise au meme balayage | colonnes qui passent |
|---|---|
| la vraie serie (equipe 0) | 22 (le meme champ vu a des alignements redondants) |
| **leurre : serie decalee d'un cran** | **0** |
| **leurre : serie constante a la valeur finale** | **0** |

Espace balaye : deltas de -512 a +3000 bits, largeurs 6 a 16, echelles x1 a x32.

**Un mystere de l'archive tombe au passage** : `RESEARCH_THEATER_RE.md` §M-ter mesurait sur ce
meme film « byte 813 = team0 x4 ». Un champ de 6 bits qui se termine 2 bits avant la fin d'un
octet, lu comme un octet, vaut exactement 4 fois sa valeur. **L'echelle x4 etait un artefact de
lecture octet-alignee** ; en lecture bit-exacte il n'y a aucune echelle.

### 6bis.2 LES AUTRES MODES — la regle NE SE GENERALISE PAS, et c'est une mesure

`cmd/tmp_scoreread` applique la regle telle quelle, sans rien chercher. C'est une **prediction**.

| film | mode | attendu | lu par la regle Slayer | verdict |
|---|---|---|---|---|
| `000d5950` | Slayer | 43-50 | **43-50** | juste |
| `01e1f945` | KOTH | 3-2 | 3-**58** | fausse |
| `64e8adfa` | CTF | 2-3 | 1-2 | fausse |
| `530820e5` | CTF | 3-0 | 0-**54** | fausse |
| `696a9d7c` | Strongholds | 200-94 | 0-24 | fausse |

Deux observations qui ne sont **pas** du bruit et qui vont dans le sens du verdict Q3 :

- en **KOTH**, le champ a `ancre+28` finit exactement sur **3** — le score d'equipe 0 — par
  paliers propres (`0...0 1 1 1 1 2 2 2 2 2 3`), tandis que `ancre+110` finit sur 58, qui est
  le nombre de morts de l'equipe 0. **Les deux emplacements ne portent pas la meme grandeur
  selon le mode.** C'est exactement ce que Q3 predit : le bloc porte des SLOTS de stat, et
  c'est le script de la variante qui decide quelle stat y loge ;
- en **Strongholds**, un champ de 6 bits plafonne a 63 et ne peut pas porter 200 : les remises
  a zero de la courbe sont la signature d'une tranche de 6 bits d'un compteur plus large.

### 6bis.3 STRONGHOLDS — un candidat credible, DEMASQUE par les ancres terrain

Sans oracle exact, le seul critere disponible est « monotone, part de 0, finit sur le score de
l'API ». Sa **fragilite est mesuree** :

| film / equipe | valeur finale exigee | colonnes qui passent | leurres (valeur finale fausse) |
|---|---|---|---|
| `530820e5` eq. 1 | 0 | **98 544** | 1 -> 136 · 2 -> 214 |
| `64e8adfa` eq. 0 | 2 | 675 | 3 -> 79 · 4 -> 984 |
| `01e1f945` eq. 1 | 2 | 236 | 3 -> 89 · 4 -> 290 |
| `696a9d7c` eq. 0 | **200** | **30** | 201 -> **0** · 202 -> **0** |

Quand le score final est petit, le critere laisse passer des centaines de colonnes **et les
leurres autant** : il ne prouve rien. Le 200 du Strongholds, lui, est discriminant — leurres a
zero. On tenait donc 30 colonnes « arithmetiquement propres », dont `[d=+34 w=6 x4]` qui rend
**200-94 exactement, monotone**.

**Et le relevé terrain les tue.** Leur courbe vaut :

```
0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 132 136 148 160 180 180 184 188 200
```

Elle reste a **zero jusqu'a ~400 s**, alors que le relevé a l'oeil atteste **21 points a 1:30**
et **69-30 a 3:10**. Passe a `cmd/tmp_scoreanchor` avec ces ancres en contrainte, le compte
tombe a **0 colonne** pour les deux equipes.

C'est le resultat le plus important de Q2 sur le plan de la methode : **une colonne qui tombe
sur la valeur finale, monotone, avec des leurres a zero, peut etre entierement fausse.** La
regle « une mesure qui sert de score ne sert pas de filtre » n'est pas une precaution
theorique — elle a attrape un faux positif ici, en une passe.

### 6bis.4 Verdict Q2, par mode

| mode | verdict |
|---|---|
| Slayer | **ETABLI** — offsets, largeur, echelle, serie complete, leurres a zero |
| Strongholds | **ABSENT DE L'ESPACE BALAYE** (largeur fixe 6-20 bits, deltas -512..+3000, echelles x1..x32). Coherent avec l'archive, qui decrit un **varint** : un balayage a largeur fixe ne peut pas l'attraper **par construction** |
| CTF, KOTH | **INDECIDABLE par cette methode** : leurs scores (0 a 3) sont trop petits pour discriminer — des centaines de colonnes passent, les leurres aussi. Il faut une serie par image-cle, pas une valeur finale |

**Le lot suivant est donc nomme** : etendre le balayage aux encodages **a longueur variable**
(varint a continuation, le `FUN_140C18A1C` selecteur 2 bits du §2 de `RE_EXE_GHIDRA_FINDINGS`),
et pour CTF/KOTH construire une serie par image-cle a partir des evenements de mode horodates
(section 4) plutot que de la seule valeur finale.

---

## 7. Q5 — LA FAMILLE PICKUP, CARTOGRAPHIEE (pas decodee)

Outil : `cmd/tmp_pickupmap`. Ce que le lot A5 n'avait pas et qui lui manquait pour se
dimensionner : **combien de changements d'arme un film porte reellement**, par joueur et par
mode. La mesure se fait sur les evenements de tir **deja decodes par le depot**
(`filmdec.ScanFilmFireEvents`, record `action_weapon_fire`) — aucun decodage neuf.

| film | mode | evenements de tir | changements d'arme | armes distinctes cumulees |
|---|---|---|---|---|
| `000d5950` | **Super Fiesta** | 519 | 69 | **60** |
| `01e1f945` | KOTH | 2 154 | 104 | 29 |
| `64e8adfa` | CTF | 2 879 | 88 | 25 |
| `530820e5` | CTF | 1 783 | 107 | 28 |
| `696a9d7c` | Strongholds | 2 246 | 105 | 35 |

**Controle de vraisemblance interne** : le Super Fiesta, ou l'arme est tiree au sort a chaque
reapparition, porte **60** armes distinctes cumulees pour 8 joueurs (7,5 par joueur) contre 25
a 35 dans les modes a dotation fixe. La mesure se comporte comme le mode l'exige.

**Ce chiffre est une BORNE INFERIEURE, et la borne est explicite** : une arme ramassee mais
jamais tiree est invisible, et la dotation de reapparition n'est pas distinguee d'un ramassage.
Il sert a **dimensionner** A5 et a lui donner un temoin de comparaison — un decodeur de
ramassages qui rendrait MOINS que ces chiffres serait faux.

Les codes du recensement Q1 (section 3.2) compatibles en volume avec une famille de ramassage
restent **96, 97, 98, 99, 101, 114, 115, 116** — aucun n'est nomme de facon fiable, la section
5.3 dit pourquoi. **Ne pas partir de la** sans relire `KILLFEED_STATE.md` §183 : le dispatcher
`FUN_140620564` est un **apply**, pas un decode.

---

## 8. DECOUVERTES (hors perimetre, non traitees — regle 7 du contrat)

1. **Donnee corrompue** : `match_objective_stats_latest` porte **9 lignes pour le match a 8
   joueurs `01e1f945`**, dont une avec un xuid non numerique `bid(42.0`. C'est ce defaut qui
   rend Q4-KOTH indecidable. A auditer sur l'ensemble de la table (combien de matchs ont un
   nombre de lignes different de leur nombre de participants ?).
2. **Donnee gratuite jamais lue** : le paquet type-10 de 10 octets qui precede chaque trame
   (3.1). Un en-tete par trame, 1:1 exact sur 5 films, que `filmdec` traverse sans le lire.
3. `match_registry.mode_category` rend « Assassin » pour une CTF et pour un KOTH, « Other »
   pour une autre CTF et pour le Strongholds. Cette colonne ne classe rien d'utilisable ; le
   mode reel se lit dans `game_variant_name`.

---

## 9. OUTILLAGE PRODUIT (jetable, aucune dependance de prod)

| outil | ce qu'il mesure |
|---|---|
| `cmd/tmp_modescore` | recensement des paquets, histogramme des codes type-0, jeton 0x7B6 des images-cles, histogramme `type_hint` du footer |
| `cmd/tmp_modestate` | la revendication externe de l'octet d'etat type-2 (transitions + valeurs) |
| `cmd/tmp_modeticks` | evenements de mode : compte par equipe et par joueur, gamertag decode, horloge BE, deduplication, intervalles ; `VERBOSE=1` sort la TIMELINE horodatee complete (t, equipe, gamertag) |
| `cmd/tmp_scorefind` | Q2 : recherche du porteur de score, avec series leurres et comptage des faux positifs |
| `cmd/tmp_scoreread` | Q2 : applique une regle d offsets SANS rien chercher (test de prediction) |
| `cmd/tmp_scoreanchor` | Q2 : recherche contrainte par les ancres terrain (celle qui demasque les faux positifs) |
| `cmd/tmp_pickupmap` | Q5 : bornes inferieures de changements d arme par joueur et par mode |

Tous se construisent en `CGO_ENABLED=0` (aucun DuckDB). Les oracles en base passent par
`cmd/diag_q` en lecture seule.

---

## 10. CE QU'IL FAUT FAIRE ENSUITE, PAR ORDRE DE VALEUR

1. **Un second film KOTH** — c'est le seul point qui bloque un verdict Q4 complet, et il coute
   une mesure (l'outil est ecrit).
2. **Q2, la suite nommee** : etendre le balayage aux encodages a LONGUEUR VARIABLE (varint a
   continuation ; le lecteur `FUN_140C18A1C` a selecteur de 2 bits, cf. RE_EXE_GHIDRA_FINDINGS
   §2) — le Strongholds est absent de l espace a largeur fixe par construction. Et pour
   CTF/KOTH, construire une serie PAR IMAGE-CLE a partir des evenements de mode horodates
   (section 4) : leurs scores sont trop petits pour qu une valeur finale discrimine.
3. **Q3 est clos cote binaire** (5.4). Le lot qui reste, borne : enumerer hors ligne le
   vocabulaire des identifiants de stat de la table `engine + 0xDF77C` (compte, identifiants,
   echelles) et le croiser avec le statborg deja serialise dans le film — il donne la liste de
   ce que le jeu mesure, sans deviner.

---

## 11. SECONDE PASSE (2026-08-01, apres relance utilisateur) — modes a objectifs

### 11.1 Q4-KOTH : REFUTE (le doute de l'utilisateur sur la « corruption » etait fonde)

8 films KOTH neufs, tous avec exactement 8 lignes de stats (donc sans le defaut de
`01e1f945`). Comparaison des comptes d'evenements de mode par joueur a
`zone_captures + zone_secures` :

| film | multiset film | multiset API |
|---|---|---|
| `6cdec7c3` | 2 6 7 8 10 14 14 17 | 2 4 4 5 6 6 7 7 |
| `71ad4abd` | 1 7 8 11 11 12 14 19 | 3 3 4 5 6 7 8 8 |
| `75f1188f` | 2 3 5 7 8 9 10 17 | 3 4 4 6 6 7 7 7 |
| `7665e832` | 3 9 11 12 14 15 15 17 | 1 4 4 5 5 5 6 7 |
| `7f1bbf06` | 1 2 5 6 7 24 | 1 2 3 3 4 6 8 10 |
| `da2fd554` | 3 4 4 7 11 12 13 | 0 2 2 4 6 7 9 9 |
| `eeaf049b` | 2 4 4 10 11 12 15 16 | 2 2 3 4 5 6 9 9 |
| `f6091638` | 1 5 6 8 10 14 18 | 1 2 3 4 4 5 6 6 |

**8 films sur 8 en desaccord**, le film comptant systematiquement PLUS (rapport 1,4 a 2,6,
non constant). L'identite etablie pour le Strongholds **ne s'etend pas au KOTH**. Croise avec
la periodicite de 5,00 s mesuree en 5.2 : en KOTH l'evenement de mode est un **tick
d'occupation**, pas une prise. Le defaut de donnee de `01e1f945` n'etait donc pas la cause —
le verdict passe d'INDECIDABLE a **REFUTE**.

### 11.2 Q2 modes a objectifs : trois voies de plus, toutes negatives

| voie | resultat | reference de hasard mesuree |
|---|---|---|
| offset fixe consistant sur **46 films KOTH** | **0 triplet** couvre >= 60 % des films | le hasard atteint **46 %** (cible leurre score+1) |
| grammaire statborg balayee dans les images-cles, 46 films KOTH | couple (s0,s1) trouve dans **65 %** des films | **leurre permute (s1,s0) : 74 %** — la cible fait MOINS bien que le leurre |
| decodage des 16 octets de signature de la capture CE (2 716 enregistrements statborg) | **0 alignement** rend le couple 200-94 | — |

### 11.3 CE QUE LA SECONDE PASSE A TROUVE, ET QUI CHANGE LA ROUTE

**L'ARCHETYPE 6 EST LE STATBORG.** Dans `deser_table.tsv`, **les 58 composants de
l'archetype 6 pointent TOUS sur `FUN_140C18794`**, le deserialiseur de stat a deux equipes
identifie en Q3. Un composant = une paire de slots de stat, une valeur par equipe.

Et il est **replique dans le film** : la capture CE compte **2 340** dispatches `ti=6` sur la
CTF et **2 716** sur le Strongholds.

**Consequence directe : je cherchais au mauvais endroit.** Le score d'equipe n'est pas un champ
du bloc game-state des images-cles TYPE_2 — c'est un **composant replique de l'archetype 6**,
dans le flux. Tous les balayages d'images-cles de la section 6bis portaient sur le mauvais
substrat. Cela explique pourquoi le Slayer tombe (son score est AUSSI recopie dans l'entete
game-state, ou il tient sur 6 bits) et pourquoi aucun mode a objectif ne suit.

**Le lot suivant est donc precis** : porter le deserialiseur d'archetype 6 sur le flux de
trames (le depot a deja `filmdec/statborg.go` et `ReadSignedVarWidth`), sortir les 58 paires de
valeurs par image, et identifier le compIndex dont la courbe finit sur le score de l'API. Le
temoin existe pour tous les modes, et le Slayer sert de controle positif.

### 11.4 Les composants d'objectif CTF, nommes structurellement (Q4-CTF)

`ti=11` est propre a la CTF (absent du Strongholds, cf. J0). Sur `530820e5`, 5 entites, 162
dispatches. Les deux composants qui portent le trafic, decompiles :

- **`ci=6` -> `FUN_1411615F8`** : lit **UN SEUL BIT** (`FUN_1406CF008`) ecrit en
  `etat + 0x154`. 42 emissions, sur **deux entites seulement** (les deux drapeaux).
- **`ci=4` -> `FUN_140DBE170` -> `FUN_140DBE400`** : lit un **masque** en `etat + 0x104`, puis
  boucle sur **4 sous-blocs** (pas de 0x40) en lisant **4 bits** par sous-bloc marque. Soit
  **4 emplacements d'objectif, 4 bits d'etat chacun, avec masque de changement**. 108 emissions.

**Tentative de correlation, et son echec honnete** : les 42 bascules de `ci=6` ressemblaient
aux 43 evenements de mode du footer, avec un alignement de fin de match frappant
(436,6/436,5 · 443,4/443,3 · 473,2/473,4). Mise a l'epreuve avec debit et decalage libres :
**22/43 coincidences a 1,5 s** — mais le **controle** (memes instants decales de 7 s, meme
liberte de calage) en obtient **20/43**. La correspondance **n'est pas etablie** ; l'impression
venait de la densite des evenements et des deux parametres libres.

Ces deux composants restent la meilleure cible connue pour l'identite du drapeau — mais ils
demandent d'etre decodes **depuis le film**, pas depuis la capture.

---

## 12. Q2 RESOLU POUR LES MODES A OBJECTIFS — le score est un composant de l'archetype 6

Outil : `cmd/tmp_statborgfilm`. **Verdict : ETABLI**, sur les deux films a capture CE, avec
egalite EXACTE sur trois grandeurs independantes a la fois.

### 12.1 Le pont entre la capture CE et le film — arithmetique, sans balayage

La capture journalise pour chaque lecture de composant le curseur moteur `c` et 16 octets lus
au pointeur du lecteur. La recette de localisation du chantier place ces octets a
`paquet.Start + 8*floor(c/64) + 8`. En retrouvant la signature a l'offset d'octet `M` du chunk
decompresse, on inverse :

```
paquet.Start = M - 8*floor(c/64) - 8
bitpos       = 8*paquet.Start + c  =  8*M - 64 + (c mod 64)
```

Le composant commence donc dans le mot de 64 bits qui **precede** la signature, au bit
`c mod 64`. **Aucune hypothese de largeur, aucun balayage, aucun parametre libre.**

Rendement mesure : **2 708 / 2 716** lectures localisees sur le Strongholds (99,7 %) et
**2 331 / 2 340** sur la CTF (99,6 %), **zero signature absente**.

### 12.2 Ce que porte l'archetype 6 — trois composants nommes par leurs valeurs

Chaque enregistrement est decode par la grammaire statborg de Q3
(`[en-tete 5][en-tete 5][valeur][valeur]`, valeurs a longueur variable). Les entites se
separent en **deux entites d'equipe** et **huit entites de joueur**.

| composant | contenu | Strongholds `696a9d7c` | CTF `530820e5` |
|---|---|---|---|
| **0** | **SCORE DE MODE** | equipes : **200** et **94** | equipe : **3** ; joueurs : **2** et **1** |
| **1** | score personnel | equipes : **8 420** et **7 420** ; 8 joueurs : 1 675, 1 900, 2 175, 1 435, 2 385, 2 160, 1 650, 2 460 | equipes : **8 030** et **4 735** |
| **2** | frags / morts | equipes : **54/48** et **48/55** | equipes : **53/40** et **39/54** |

**Toutes ces valeurs sont exactes a l'unite contre l'API**, et elles se recoupent entre elles :
la somme des 8 scores personnels du Strongholds fait **15 840 = 8 420 + 7 420**, sans
ajustement. C'est une fermeture arithmetique gratuite, et elle tombe.

Le cas CTF « 3-0 » est instructif : l'equipe a **0 n'emet jamais** le composant 0 — une valeur
qui ne change pas n'est pas repliquee. Et les valeurs par joueur (2 et 1) sont exactement les
`flag_captures` des deux joueurs concernes. Le composant 0 porte donc **la contribution
d'objectif au niveau equipe ET au niveau joueur**.

### 12.3 La resolution temporelle — la reponse a « 5 secondes c'est trop long »

Le composant n'est reemis **que lorsqu'il change**. Chaque lecture est donc l'instant exact
d'un changement de score, pas un echantillonnage periodique :

- Strongholds, equipe qui mene : **190 emissions** du composant 0 sur 561 s, et la serie est
  **monotone croissante** ;
- CTF : **3 emissions** pour 3 captures — une par capture, a l'instant exact.

C'est structurellement plus fin que le tick de 5 s des evenements de mode du footer : on ne
depend plus d'un battement, on lit l'evenement de changement lui-meme.

### 12.4 Ce qui reste avant que ce soit universel

Ce decodage s'appuie sur la capture CE pour LOCALISER les lectures — il ne vaut donc, en
l'etat, que pour les deux films captures. **Mais il fournit desormais 5 039 positions de verite
terrain** (2 708 + 2 331) avec, pour chacune, la position de bit exacte et les valeurs
attendues. C'est exactement ce qu'il faut pour calibrer un localisateur **purement hors ligne**
et le valider sans complaisance. C'est le lot suivant, et il est borne.

### 12.5 La tentative de localisateur HORS LIGNE, et son echec mesure

Outil : `cmd/tmp_scoreoffline`. Motif cherche dans les paquets type-0 : « 10 bits d'en-tete
nuls + deux valeurs a longueur variable, la seconde nulle » — la signature exacte observee sur
la verite terrain.

**Resultat : NEGATIF, et le chiffre le dit.** Sur `696a9d7c`, le motif rend **103 560
candidats bruts** et « atteint » **les 200 valeurs** du score : il est sature de bruit. La
marche d'escalier extraite (45 a 149 s, 137 a 411 s) contredit le releve terrain (21 points a
1:30, 69 a 3:10). Le motif est trop faible — 10 bits nuls plus deux valeurs courtes se
rencontrent partout dans un flux bit-packe.

**Ce que ca dit du lot suivant** : la localisation ne s'obtiendra pas par motif, mais par
**parcours de la chaine d'enregistrements de trame** (`FUN_1406CD128`, deja porte dans
`filmdec/frame_records.go`), qui seule donne l'attribution entite/composant. C'etait deja le
mur du chantier killweapon. La nouveaute, et elle est reelle : on dispose desormais de
**5 039 positions de bit exactes avec leurs valeurs attendues** — le parcours peut enfin etre
valide enregistrement par enregistrement au lieu d'etre pilote par un gradient.

---

## 13. CE QU'ON PEUT DESSINER SUR LA LIGNE DE TEMPS D'UN REJEU — par mode, SANS dump

Relecture de tout ce qui precede sous l'angle de l'objectif reel : **rejouer le deroule du
match**. La question n'est pas « ou est le score » mais « que puis-je afficher, quand, et sur
quel film ».

### 13.1 Zones (Strongholds) — DISPONIBLE MAINTENANT, sur tous les films en cache

« Qui prend ou securise une base, et a quelle milliseconde » : **etabli et universel**.
Verifie sur **9 films** dont 8 sans aucune capture Cheat Engine :

| film | comptes film vs API (joueurs a zero exclus) |
|---|---|
| `696a9d7c` | 8/8 exact |
| `28d77409`, `353367d6` | identiques sans retouche |
| `10ed320d`, `1e26f641`, `2b2c5aa3`, `305a6b15`, `316205b8` | exacts apres exclusion des joueurs a 0 |
| `1b1e380f` | 1 joueur manquant (2 evenements) |

**8 films sur 9 exacts.** Un joueur qui ne prend aucune zone n'emet rien — d'ou les zeros
presents cote API et absents cote film. Chaque evenement porte l'horodatage en ms et le
gamertag decode du film.

### 13.2 CTF — l'INSTANT des captures est disponible, l'action ne l'est pas

- **Instants de capture** : le detecteur de rafale `tiers==6` de l'archive est deja dans le
  depot (`objectiveevents/film.go`), ms-precis, mesure a 0 manque / 0 faux positif sur 4 matchs.
- **Auteur** : par le groupe d'evenements de mode coincidents.
- **Ce qui manque** : distinguer prise / retour / capture. L'identite avec
  `flag_captures + grabs + returns` est **refutee** (section 4.2). On peut donc afficher « ce
  joueur a interagi avec le drapeau a t », pas « il l'a pris ».

### 13.3 KOTH — occupation de la colline toutes les 5 s, pas les captures

L'evenement de mode est un **tick d'occupation de 5,00 s par joueur** (mediane 5 005 ms,
64 intervalles ; refute comme prise sur 8 films, section 11.1). On peut donc tracer **qui tient
la colline au cours du temps**, a 5 s pres. Les instants de capture, eux, ne sont pas
disponibles par cette voie.

### 13.4 La COURBE DE SCORE — le seul manque, et il est cerne

| | avec capture CE | sans capture CE |
|---|---|---|
| score final | oui (API de toute facon) | oui (API) |
| **courbe de score, a l'instant de chaque changement** | **OUI** (section 12) | **NON** |

C'est le seul element du deroule qui reste hors de portee sur un film quelconque. Il est
localise (composant 0 de l'archetype 6, paquets type-0) et lu exactement sur les deux films
captures ; ce qui manque est le **parcours de la chaine d'enregistrements de trame** pour le
retrouver sans capture. Les 5 039 positions de verite terrain existent desormais pour valider
ce parcours.

---

## 14. LA CHAINE D'ENREGISTREMENTS — grammaire decodee, et LE MECANISME DE SAUT EXISTE

Attaque Ghidra du parcours de chaine, en remontant et redescendant les appels. Resultat :
**la grammaire complete de l'en-tete est etablie, et elle contient un mecanisme de saut** —
un enregistrement DECLARE quels composants il porte, on n'a donc pas a tous les parcourir.

### 14.1 La grammaire, fonction par fonction

**`FUN_1406CD128` — la boucle de trame** (deja portee en partie dans
`filmdec/frame_records.go`) :

```
boucle :
  [1 bit presence]        -> 0 = fin de la chaine
  [2 bits type]           -> 1 = NEW, 2 = DELETE, 3 = DELTA
  [identifiant d'entite]  -> FUN_1406D3140
  puis dispatch : type 1 -> FUN_141F86704 · type 3 -> FUN_141F86B58
```

Pour le type 3 (delta), le moteur valide l'entite avant de deserialiser :

```
slot = identifiant & 0x3FFFFFFF
table = *(param_1 + 0x38)          // table d'entites, pas de 0xA0
si  *(u32*)(table + slot*0xA0 + 8) == identifiant
et  *(u16*)(table + slot*0xA0 + 2) == type   -> FUN_141F86B58
```

**`FUN_1406D3140` — l'identifiant d'entite** : `[W bits valeur][2 bits]` puis
`identifiant = (2bits << 30) | (base + valeur)`. La largeur `W` et la `base` viennent de
globales (`DAT_1451F98D0` / `DAT_1451F98D4`, indexees par le parametre, gatees par
`DAT_144706104`) — **configurees au demarrage, donc pas lisibles statiquement**. C'est le seul
maillon qui reste ouvert (14.3).

**`FUN_141F86B58` — le delta** : `memcpy` de la ligne de base, puis appelle
**`FUN_14076CB60`** avec le descripteur d'archetype.

**`FUN_14076CB60` — la boucle de composants, et c'est la que tout se joue** :

```
masque = FUN_1406D7610(descripteurArchetype, lecteur)
pour i de 0 a *(int*)(descripteur + 0x4320) - 1 :        // nombre de composants
    si (masque >> i) & 1 :
        deserialiseur = (*(descripteur + i*8))[+0x28]     // le slot appele
        deserialiseur(...)
        // pose le bit sale : octet d'index a  descripteur + 0x4850 + i
```

**`FUN_1406D7610` — LE MECANISME DE SAUT** :

```
[1 bit forme]
  forme 0 : [3 bits compte N][N x 6 bits index de composant]   <- LISTE CREUSE
  forme 1 : [64 bits masque plein]
```

Autrement dit **l'enregistrement enumere les index des composants qu'il porte, sur 6 bits
chacun** (au plus 7 en forme creuse). C'est exactement l'adressage direct espere : pour
atteindre un composant on n'a pas a decoder ceux qui le precedent dans l'archetype — seulement
ceux qui sont effectivement PRESENTS et listes avant lui.

### 14.2 Validation sur la verite terrain : 1 078 / 1 090

La grammaire n'est pas supposee, elle est verifiee. En remontant depuis la **premiere lecture
de chaque entite** (1 090 cas, localises par la capture CE), on exige que les bits qui
precedent forment `[1 bit forme = 0][3 bits N][N x 6 bits]` **et que le premier index liste
soit exactement le composant observe** :

**1 078 / 1 090 = 99 %.** Repartition du nombre de composants declares :
`N=1 : 235 · N=2 : 596 · N=3 : 47 · N=4 : 32 · N=5 : 70 · N=6 : 70 · N=7 : 28`.

Un enregistrement d'entite porte donc **1 a 7 composants**, mediane 2 — et non les 58 de
l'archetype. Le cout de decodage d'un enregistrement statborg est minuscule.

### 14.3 Le maillon qui reste — et il est nomme

La largeur `W` et la `base` de l'identifiant d'entite sortent d'une globale ecrite au
demarrage. La recherche empirique par verite terrain n'a pas converge (les candidats W=9 a 14
rendent tous le meme compte, signature d'un appariement degenere) : ce n'est pas la bonne
facon de l'obtenir.

**Deux voies propres pour la prochaine session, par ordre de cout :**
1. **Statique** : remonter les ecrivains de `DAT_1451F98D0` / `DAT_144706104` par
   `get_xrefs_to` — ce sont des globales d'initialisation, leur valeur est probablement une
   constante du code de boot.
2. **Par construction** : l'en-tete complet mesure `1 + 2 + W + 2 + (4 + 6N)` bits. On connait
   deja, pour 1 078 enregistrements, la position exacte du masque ET la valeur de `N`. La
   distance entre deux enregistrements consecutifs d'une meme trame donne donc `W` par
   soustraction, sans deviner : c'est une mesure directe, pas un balayage.

Une fois `W` connu, la boucle de trame se parcourt de bout en bout **sans capture Cheat
Engine** — et la courbe de score de la section 12 devient disponible sur les 951 films du cache.

### 14.4 La largeur de l'identifiant : DERIVEE du binaire, mais NON confirmee par la mesure

La voie statique a abouti. `FUN_1406D310C` est un `ceil(log2(x))` : la largeur se deduit d'une
**borne**. Et l'initialiseur **`FUN_140D10BB0`** remplit la table des bornes en clair :

| index | base | borne | largeur deduite |
|---|---|---|---|
| 0, 1 | 0x200 | `0x1FFF - 0x200` | 13 |
| 2 | 0x200 | 0x100 | 8 |
| 3 | 0x300 | 0x100 | 8 |
| 4 | 0x200 | 0x200 | 9 |
| 5 | 0x400 | 0x100 | 8 |
| 6 | 0 | 0x200 | 9 |
| **7** *(l'appel de la boucle de trame)* | **0** | **0x1FFF** | **13** |
| 8 | 0 | 0x1FFF | 13 |

`DAT_144706100 = 0x1FFF` est lu en clair dans l'image, et `DAT_144706104 = 1` (l'initialiseur
l'arme). L'identifiant d'entite serait donc **`[13 bits slot][2 bits generation]`**, sans base.

**Mais la mesure ne le confirme pas** : en testant toutes les largeurs de 8 a 32 juste avant le
masque, **aucune** ne rend le slot attendu. La grammaire du masque, elle, est bonne a 99 % —
donc l'erreur est **entre l'identifiant et le masque**, pas apres.

**Ce qu'il faut verifier en premier a la reprise** (et ne pas re-deviner) : `FUN_1406CD128`
porte deux lectures conditionnelles gatees par `FUN_14076CEA8()` — un mot de 32 bits en tete de
chaque tour de boucle, et un autre de 8 bits sur certains types. `FUN_14076CB60` en porte une
troisieme : un mot de 32 bits **apres chaque composant**, compare a la sentinelle
`0xBCDDCBA` (« entity component corrupt »). Si ce mode est actif dans les films, il decale tout
ce qui precede le masque. La sentinelle `0xBCDDCBA` est **cherchable directement dans le
flux** : sa presence ou son absence tranche la question en une passe, sans hypothese.

C'est la seule inconnue restante entre l'etat actuel et un parcours de chaine complet hors
ligne — donc entre l'etat actuel et la courbe de score sur les 951 films du cache.
