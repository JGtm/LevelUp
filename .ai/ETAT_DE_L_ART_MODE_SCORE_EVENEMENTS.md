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

### 14.5 L'identifiant EST trouve — par la mesure, pas par l'hypothese

La recherche « quel champ est CONSTANT par entite et DIFFERENT entre entites » tranche sans
rien supposer : **purete 100 %, 10 valeurs distinctes pour 10 entites** (2 equipes + 8 joueurs),
a **2 bits avant le masque, sur 13 bits** — exactement la largeur derivee du binaire en 14.4.

Ma verification precedente echouait pour une raison simple : **la valeur du flux n'est pas
l'identifiant runtime de la capture.** La correspondance mesuree est propre :

| eid de la capture | slot dans le flux |
|---|---|
| 1073741827 (equipe A) | **6** |
| 1073741828 (equipe B) | **8** |
| 1073741829 a 1073741836 (les 8 joueurs) | 10, 12, 14, 16, 18, 20, 22, 24 |

soit `slot_flux = 2 x (eid - 0x40000000)`. La grammaire d'en-tete est donc **complete** :

```
[1 bit presence][2 bits type][13 bits slot][2 bits generation][1 bit forme][3 bits N][N x 6 bits index]
```

### 14.6 L'ancrage direct hors ligne : tente, INSUFFISANT en l'etat

`cmd/tmp_scorechain` ancre sur ce motif et decode les composants listes. Resultat sur le
Strongholds, slot 6, composant 0 : **506 ancrages**, valeurs incoherentes (1 815 043 216,
-19 383...), la ou la verite terrain en compte **190** et une courbe monotone jusqu'a 200.

**Le diagnostic est net, et il faut le dire tel quel** : 13 bits de slot ne suffisent pas a
discriminer dans un paquet de ~2 000 bits, et mes contraintes de cadrage
(bit de presence, 2 bits de type) ne sont pas encore correctes — avec elles, 5 ancrages ; sans
elles, 506 de bruit. Le motif est trop faible **seul**.

**Ce qui manque exactement, et c'est un lot, pas une enigme** : caler le cadrage
presence/type. `FUN_1406CD128` lit le bit de presence puis 2 bits de type, mais avec deux
lectures conditionnelles gatees par `FUN_14076CEA8` (32 bits en tete de tour, 8 bits selon le
type) qui decalent le tout si le mode est actif. La verite terrain le donne **par soustraction
directe** : on connait 1 078 positions de masque exactes ; l'ecart entre le masque et le debut
reel de l'enregistrement se lit, il ne se devine pas. C'est une mesure de quelques lignes, pas
une recherche.

Une fois ce calage pose, l'ancrage devient sur-contraint (presence + type + 13 bits de slot +
forme + compte + index croissants bornes = plus de 30 bits de contrainte dure) et la courbe de
score tombe hors ligne sur les 951 films.

### 14.7 Le cadrage, MESURE par soustraction (et non devine)

Sur les 1 078 en-tetes de masque localises, lecture des bits qui precedent le champ de 13 bits :

| 3 bits avant le slot | occurrences |
|---|---|
| `010` | 926 |
| `110` | 151 |
| `000` | 1 |

Les **deux bits adjacents au slot valent `10` dans 1 077 cas sur 1 078**, et le bit encore
avant varie (0 ou 1). C'est une constante de cadrage, pas du bruit.

**Et elle revele une incoherence a lever avant d'aller plus loin** : lus MSB-first, ces deux
bits valent **2**, alors que le type DELTA vaut **3** dans `FUN_1406CD128`. Deux lectures
possibles, et il faut les departager par la mesure, pas par preference :

1. **le champ d'identite fait 14 bits, pas 13** — l'exploration empirique donnait 100 % de
   purete pour les largeurs 11 a 14 a ce meme decalage, et la correspondance mesuree
   `slot_flux = 2 x (eid - 0x40000000)` **suggere exactement cela** : un bit de poids faible
   supplementaire capte par une largeur de 14. Dans ce cas les bits `10` que je lis
   chevauchent le champ, et le type est ailleurs ;
2. le cadrage porte un champ intermediaire non identifie entre le type et le slot.

**Le controle qui tranche, et il est immediat** : relancer la recherche de purete en fixant la
largeur a 14 et en verifiant que les 3 bits qui precedent deviennent alors `[1 bit presence]
[2 bits = 11]`. Si `11` apparait, l'hypothese 1 est etablie et l'ancrage devient sur-contraint
(presence + type + 14 bits de slot + forme + compte + index croissants).

C'est le point de reprise exact : une mesure, pas une recherche.

---

## 15. LA COURBE DE SCORE EST LUE HORS LIGNE — le chantier est ferme (2026-08-01, 3e passe)

> Ce qu'il fallait obtenir : **le deroule d'un match a objectifs — la progression du score et
> qui prend quoi, a l'instant pres, sur les films deja en cache et sans aucun dump**. C'est
> fait, et c'est verifie par confrontation, pas par vraisemblance.

### 15.1 Le controle de 14.7 est passe — mais il a corrige l'hypothese, pas seulement confirme

Le controle prescrit (fixer la largeur a 14 et regarder les bits precedents) a ete joue sur les
1 078 en-tetes de verite terrain (`tmp_chainhdr`, mode `FRAMEBITS`). Il rend trois mesures :

| largeur du slot | bit precedent | 2 bits precedents | purete du slot |
|---|---|---|---|
| 13 | 1 dans 151/1 078 | `10` dans 1 077/1 078 | 100 % |
| **14** | **1 dans 1 077/1 078** | varie | **100 %** |

et, a largeur 14, les **2 bits qui suivent le slot valent `10` dans 1 077/1 078**, tandis que
les slots observes sont exactement `6, 8` (equipes) et `10, 12, 14, 16, 18, 20, 22, 24`.

L'hypothese 1 de 14.7 est donc **etablie sur la largeur** (14 bits) mais **corrigee sur le
type** : les 2 bits qui precedent le bit de presence sont **independants l'un de l'autre**
(22 co-occurrences observees contre 21,6 attendues sous independance). Ce ne sont pas un champ
de type : c'est **la queue de l'enregistrement precedent**. On ne les contraint pas.

Cadrage retenu, entierement mesure :

```
[1 bit = 1][14 bits slot][2 bits = 10][1 bit forme = 0][3 bits N][N x 6 bits index croissants]
puis, par index : [5 bits = 0][5 bits = 0][valeur A][valeur B][2 drapeaux][conditionnelles]
```

### 15.2 Les deux en-tetes de 5 bits du composant valent ZERO — 10 bits de contrainte de plus

Sans eux, l'ancrage rend **434 lectures pour 284 reelles** sur le Strongholds : 151 faux
positifs. La mesure des champs de l'enregistrement statborg separe les deux populations sans
appel au jugement :

| champ | lectures reelles | faux positifs |
|---|---|---|
| en-tete A = 0 | 283 / 283 | 11 / 151 |
| en-tete B = 0 | 283 / 283 | 98 / 151 |

Exiger `A == 0 && B == 0` est donc une contrainte **lue**, pas un seuil choisi.

### 15.3 Le rendement, confronte position par position a la capture Cheat Engine

`cmd/tmp_scoreverify` compare l'ancrage hors ligne et la capture CE **bit de depart par bit de
depart**, et publie les deux chiffres qui comptent :

| film | verite terrain | ancrages | retrouves (position ET valeur) | faux positifs | manques |
|---|---|---|---|---|---|
| `696a9d7c` Strongholds | 284 | 285 | **283 (99,6 %)** | **2 (0,7 %)** | 1 |
| `530820e5` CTF | 6 | 6 | **6 (100 %)** | **0** | 0 |

**Zero valeur fausse a bonne position** dans les deux cas. Les 2 residus tombent sur la seule
monotonie du score, sans seuil arbitraire (`keepMonotone`).

### 15.4 Les DEUX horloges sont la meme, a la milliseconde — mesure, pas hypothese

`cmd/tmp_filmclock` compare le `TimestampUS` du premier paquet de chaque chunk au `start_ms`
du manifeste : **ecart compris entre -4 et 0 ms sur 573 s de film**, 30 chunks. La courbe de
score horodatee par paquet et les evenements d'objectif du footer sont donc **directement
superposables, sans recalage**.

Piege paye au passage : prendre pour origine le premier paquet *ou l'on trouve quelque chose*
au lieu du `start_ms` du manifeste decale toute la courbe (36 s sur le Strongholds, 140 s sur
le CTF). L'origine se prend **par chunk, sur le manifeste**.

### 15.5 La preuve croisee CTF : deux decodeurs independants, la meme milliseconde

En CTF le score d'equipe n'avance **que** sur une capture. Les deux sources sont decodees par
des chemins qui n'ont rien en commun (bursts a 6 tiers du footer d'un cote, composant 0 de
l'archetype 6 dans les paquets delta de l'autre) :

```
140.072  OBJ    capture         2535469190789936
140.073  SCORE  equipe slot 8  -> 1
186.218  OBJ    capture         2533274823110022
186.219  SCORE  equipe slot 8  -> 2
473.211  OBJ    capture         2533274858283686
473.212  SCORE  equipe slot 8  -> 3
```

**3 captures sur 3, chacune suivie de son point a +1 ms.** Score final 3-0, conforme a l'API.

### 15.6 Le releve terrain Strongholds : quatre ancres sur quatre

Le releve a ete ecrit a l'oeil le 2026-07-31, **avant** tout decodage de score.

| releve terrain | ce que rend le decodage |
|---|---|
| 0:48 — flyguy8773 capture la base B | `zone_capture` a **48,901 s** |
| 1:30 — une equipe controle les trois bases, **21 pour l'autre** | rafale de 4 captures a 89,29/89,96 s ; **slot 8 = 21** a t=90 s |
| 3:10 — **score 69 - 30** | a t=190 s : **slot 6 = 69, slot 8 = 30** |
| 5:34 — controle des trois bases par l'equipe de flyguy8773 | slot 6 a 1 pt/s sans interruption, slot 8 gele |

Le recit se lit d'un bloc : premiere capture a 48,9 s (equipe 0) ; l'equipe 1 prend deux zones a
53,556 s et **commence a marquer 1 s plus tard, a 54,552 s** ; elle plafonne a 21 quand
l'equipe 0 prend les trois bases a 89,96 s et **demarre a 90,288 s**. Score final 200-94,
conforme a l'API.

### 15.7 Ce que ca livre, et ce qui reste

**Livre, sur les films en cache, sans aucun dump ni capture** : la ligne de temps fusionnee
« evenement d'objectif + progression du score », a la milliseconde. Dans le depot :
`internal/analysis/objectiveevents.ScoreCurve`, avec deux tests de verite terrain.
Outil de lecture : `cmd/tmp_timeline <cacheDir> <filmID> <gameVariantName>`.

**Ce que porte le composant depend du mode, et c'est mesure** : en Strongholds il n'est emis
QUE par les 2 entites d'equipe (284 lectures, toutes sur les slots 6 et 8) ; en CTF les 8
entites de joueur l'emettent aussi, ou il vaut leur compte de captures. Les composants 1
(score personnel) et 2 (frags/morts) sont per-joueur dans tous les modes et le meme ancrage
les atteint — rendement mesure 374/381 et 385/397 — mais ils ne sont pas livres : les
atteindre oblige a chainer les largeurs des composants qui les precedent dans la liste creuse.

**Reste hors de portee, inchange par cette passe** : **quelle** zone (A/B/C) est prise — le
resultat negatif de l'archive tient toujours ; et l'identite fine des actions CTF
(prise / retour), refutee en 4.2.

### 15.8 Le balayage des 951 films du cache — la generalite, chiffree

Le decodeur a ete passe sur **tout le cache**, sans oracle externe, en ne regardant que ce
qu'un film seul ne montre pas : la distribution des scores maximaux doit reproduire les
plafonds canoniques des modes, et les valeurs aberrantes doivent rester rares.

| resultat | compte |
|---|---|
| films decodes proprement | **944 / 951 (99,3 %)** |
| films a valeur aberrante (> 1 000) | **5** |
| films sans aucune emission | 2 |

Distribution du score maximal par film — ce sont les plafonds de Halo, retrouves sans qu'on
les ait dits au decodeur :

| score max | 50 | 3 | 200 | 100 | 2 | 5 | 49 | 1 | 80 |
|---|---|---|---|---|---|---|---|---|---|
| films | **576** | **159** | **48** | 44 | 35 | 18 | 17 | 14 | 4 |

(50 = Slayer · 3 et 2 = CTF · 200 = Strongholds · 5 = KOTH classe.)

**Deux contraintes ont ete ajoutees a partir de ce balayage, et chacune est mesuree** :

1. **selecteur de largeur != 2.** Aucune lecture reelle n'encode sa valeur sur 32 bits ; les
   ancrages a valeur aberrante, si. Passe de 24 a 10 films aberrants.
2. **plus longue suite croissante, au lieu d'un filtre glouton.** Un parasite a valeur
   enorme arrivant EN PREMIER masquait toute la vraie courbe derriere lui. Le critere de
   longueur est sans parametre — les emissions reelles ecrasent en nombre les parasites
   (283 contre 2 sur le film de reference). Passe de 10 a **5**.

---

## 16. NOMMER LES EVENEMENTS — bareme du score personnel, identite des slots, medailles

> Demande de l'utilisateur : « le score personnel ne m'interesse pas comme chiffre, mais
> comme DESCRIPTION de l'evenement ». Cette section rend trois resultats : le bareme des
> actions, l'identite nominative des entites, et une table de 88 medailles nommees.
>
> Garde posee par l'utilisateur et respectee : **tout evenement de score n'est pas une
> medaille**. Les deux canaux sont mesures separement et le lien n'est jamais suppose.

### 16.1 Le bareme des actions, mesure par coincidence temporelle

Le composant 1 porte le score personnel dans sa **valeur B** (la valeur A est constamment
nulle). Ses increments sont confrontes a deux decodages independants : les morts de
`killsource` (horloge du kill-feed) et les evenements d'objectif d'`objectiveevents`.

Strongholds `696a9d7c` — 217 increments, fenetre d'appariement 1,5 s :

| bareme | n | SON frag | SON objectif | SON assistance | inexplique |
|---|---|---|---|---|---|
| **+100** | 95 | **95** | — | 0 | **0** |
| +50 | 100 | 4 | — | 46 | 50 |
| **+25** | 16 | 9 | **16/16 (100 %)** | 1 | — |

CTF `530820e5` — 152 increments, ou le contraste est plus net :

| bareme | n | SON frag | SON assistance | inexplique par le combat |
|---|---|---|---|---|
| **+100** | 77 | **75** | 1 | 1 |
| **+50** | 31 | 0 | **29** | 2 |
| +25 | 18 | 2 | 0 | **16** |
| **+300** | **3** | 1 | 0 | 2 |

**+100 = un frag** et **+50 = une assistance** sont etablis. **+300 apparait exactement
3 fois pour 3 captures de drapeau** (score final 3-0). +25 est un bareme d'objectif : en
Strongholds il coincide a 100 % avec un evenement d'objectif.

**Ce que ce resultat referme cote depot** : `internal/analysis/temporal/engagement_score.go`
pose `100*kills + 50*assists` et `ObjectivePointsPerCapture = 25.0` avec la mention « a
calibrer en post-validation ». Les trois valeurs sont desormais **mesurees sur le film**.

**La limite, et il faut la dire** : les increments ne sont pas atomiques. Le CTF porte des
increments de 125 (= 100 + 25), 60 (= 50 + 10) et 225 : plusieurs actions tombent dans le
meme paquet et se somment. Un bareme compose ne se lit pas par sa seule valeur.

### 16.2 L'IDENTITE des slots d'entite — resolue, et par deux voies qui se confirment

En appariant les increments de +100 d'un slot aux instants de frag de chaque joueur, chaque
slot s'attribue a **un** joueur, sans collision — 95 increments sur 95 expliques, le second
candidat toujours loin derriere (0 a 4) :

| slot | 10 | 12 | 14 | 16 | 18 | 20 | 22 | 24 |
|---|---|---|---|---|---|---|---|---|
| joueur | FlyGuy8773 | SunburntMonk740 | AG x GibsoN Zz | Destruk2107 | Otti1614 | Madina97294 | JGtm | NeonKnight3166 |
| frags expliques | 6/8 | 14/15 | 15/15 | 8/9 | 14/15 | 14/15 | 8/9 | 16/16 |

La meme operation avec les evenements d'objectif attribue a chaque slot un **xuid**, ce qui
referme le pont gamertag <-> xuid **sans aucune jointure en base**. Les deux voies se
confirment sur les deux joueurs dont l'identite etait etablie ailleurs (§4.1) :
slot 24 -> NeonKnight3166 -> xuid `2535458126310341` (16 evenements) et
slot 22 -> JGtm -> `2533274823110022` (9 evenements).

**Consequence** : toutes les courbes par joueur deviennent nominatives.

### 16.3 Les MEDAILLES du film, nommees — 88 couples, controle sur moities disjointes

Piste ouverte par l'utilisateur : plutot que d'inferer, lire une bibliotheque. Le depot en a
une, `medal_definitions` (167 medailles, nom et description FR/EN).

**Ce que le bloc d'evenement NE porte PAS** : l'identifiant de medaille. Balayage de tous
les decalages de 32 bits des 60 octets, dans les deux boutismes — **aucun** ne rend un
identifiant du catalogue. La colonne `medal_definitions.personal_score` existe mais n'a
jamais ete peuplee (167 lignes a 0).

**Ce qu'il porte** : un couple `(type_hint, medal_type)` et le xuid. Le nommage se fait donc
par les COMPTES et se prouve par l'UNICITE : pour un film donne, le vecteur « combien de
fois par joueur » d'un couple doit etre exactement celui d'une medaille de `medals_earned`.
Un seul film laisse des ambiguites ; **l'intersection sur 948 films les leve**.

| resultat | valeur |
|---|---|
| films lus | 948 |
| couples distincts | 95 |
| **couples resolus a UNE medaille** | **88** |
| ambigus | 7 (tous vus sur **un seul** film — medailles de vehicule) |

**Le controle qui donne sa valeur au resultat** : la table est ajustee separement sur les
films pairs (474) et sur les films impairs (474), qui ne partagent **aucun** film. Les deux
tables nomment 78 et 81 couples, en commun 72, et **zero desaccord**.

Extrait : `(50,26)` Abats joie · `(50,71)` Tirailleur · `(50,74)` Boxeur · `(50,76)`
Flingueur · `(50,101)` Kong · `(50,127)` Frag d'outre-tombe · `(100,0)` Double frag ·
`(100,9)` Folie meurtriere · `(150,1)` Triple frag · `(150,10)` Massacre · `(200,44)`
Extermination · `(225,3)` Quelle tuerie.

**Ce que la table NE resout pas, et c'est mesure** : appliquee film par film, elle rend
27,3 % de triplets (medaille, joueur) exacts sur la moitie non utilisee pour l'ajustement,
avec **11 440 sur-comptes pour 3 730 sous-comptes**. Le maillon faible n'est pas le nommage
— prouve par le controle sur moities disjointes — mais le **scan d'evenements du footer**,
qui n'est exact sur la totalite d'un film que dans ~37 % des cas. C'est lui qu'il faut
durcir avant d'exposer des medailles horodatees.

**Et la garde de l'utilisateur tient** : les actions d'objectif pures (prise, retour,
capture de drapeau, capture de zone) **ne sont pas des medailles**. Les medailles liees aux
objectifs sont des faits d'armes (« Bataille de drapeaux », « Interception », « Garde de
colline », « Zonard »...). Les deux canaux sont complementaires, pas redondants.

### 16.4 Les autres modes a objectifs — ce qui passe et ce qui differe

Courbe de score confrontee au score de `match_registry` sur les films caches de KOTH,
Total Control et Oddball :

| resultat | valeur |
|---|---|
| films disponibles | 61 (135 matchs sans film cache) |
| **score final EXACT** | **46 (75 %)**, dont **aucun** 0-0 trivial |
| ecarts | 15 |

Les 15 ecarts ne sont pas du bruit, ils sont **structures** — le film et le registre ne
comptent pas la meme chose :

- **Total Control** : le film compte les points fins, le registre compte les SETS.
  `a521164d` film 96-0 / API 3-0 · `a349fea8` film 32-64 / API 1-2 — soit **un facteur 32**.
  Le film est donc plus FIN que la reference, pas faux.
- **KOTH** : `606d9844` film 0-3 / API 105-8 ; `8076f97f` film 3-0 / API 78-105. Le film
  porte un compte de collines, le registre un total de points.
- **Oddball** : `24dbb67d` film 100-78 / API 200-121 ; plusieurs films plafonnent a 80 ou
  100. A instruire (manches ?).

**Conclusion** : le decodeur ne casse pas hors des zones et du CTF ; c'est la SEMANTIQUE du
composant 0 qui depend du mode. Ce qu'il faut, avant d'exposer ces modes, c'est nommer la
quantite mode par mode — pas re-decoder.

### 16.5 LA BIBLIOTHEQUE EXISTE DEJA — `personal_score_awards`, et elle ferme le nommage

L'utilisateur demandait s'il existe un texte pour les evenements de score qui ne sont PAS
des medailles. **Oui, et dans le depot** : la table `personal_score_awards` des bases
joueur porte `award_name`, `award_category`, `award_count` et `award_score`. Le bareme s'en
deduit directement (`award_score / award_count`) :

| valeur unitaire | noms possibles |
|---|---|
| **300** | `flag_captured` |
| **100** | `killed_player` · `flag_capture_assist` · `hill_scored` · `destroyed_scorpion` · `destroyed_wraith` |
| **75 / 50** | `zone_captured` (variable) |
| **50** | `kill_assist` · `ball_control` · `emp_assist` · `driver_assist` · destructions de vehicules legers |
| **25** | `flag_stolen` · `flag_returned` · `zone_secured` · `hill_control` · `runner_stopped` · les `hijacked_*` |
| **10** | `flag_taken` · `ball_taken` · `sensor_assist` · `mark_assist` |
| **-100** | `self_destruction` · `betrayed_player` |

Les valeurs mesurees sur le film en 16.1 (100 = frag, 50 = assistance, 25 = objectif,
300 = capture) sont donc **confirmees par une source qui ne doit rien au film**.

**Consequence de code, et c'etait un defaut** : le score personnel **n'est pas monotone**
(-100 existe). `PersonalScoreCurve` n'applique donc AUCUN filtre de monotonie — celui de
`ScoreCurve` ne vaut que pour le score de MODE, qui, lui, ne recule jamais.

**La reconciliation, sur un match reel** — JGtm, film `1bc77d2e` (CTF), slot 18 :

| recompense de l'API | compte API | ce que le film porte |
|---|---|---|
| `killed_player` | 24 | 22 x (+100) + 2 x (+125) = **24** |
| `flag_captured` | 1 | 1 x (+300) = **1** |
| `kill_assist` | 2 | 2 x (+50) = **2** |
| `flag_stolen` 4 + `flag_returned` 2 + `runner_stopped` 2 | 8 | 6 x (+25) + les 2 x (+25) inclus dans les +125 = **8** |
| `flag_taken` | 1 | 1 x (+10) = **1** |

Somme : 22x100 + 6x25 + 2x50 + 2x125 + 300 + 10 = **3 010**, exactement le total de l'API.
**Aucune recompense n'est perdue, aucune n'est inventee**, et les increments composes se
decomposent exactement.

**Ce qui reste a faire pour etiqueter, et c'est mecanique** : la valeur seule ne suffit pas
a nommer un +25 (six noms possibles). Trois contraintes le font, et elles sont toutes
disponibles : la **valeur**, le **quota** exact par joueur (`personal_score_awards`), et la
**coincidence temporelle** (`runner_stopped` tombe sur un frag du joueur, `flag_returned` sur
un evenement de drapeau). C'est le meme genre d'appariement contraint que celui qui a nomme
les 88 medailles.

### 16.6 L'ETIQUETAGE — la ligne de temps nommee, livree

`objectiveevents.LabelPersonalScore(points, quotas)` rapproche les increments dates du film
et le quota `personal_score_awards` du joueur. La resolution se fait a la VALEUR, ce qui est
exact et sans arbitrage :

- une valeur portee par **une seule** recompense du quota -> l'evenement est **nomme** ;
- une valeur partagee -> l'evenement porte **la liste des candidates**, jamais un nom choisi
  au hasard ;
- un increment **compose** est decompose **si et seulement si** la decomposition est unique
  a nombre de parts minimal. Le cas ambigu est reel : `zone_captured` valant tantot 50
  tantot 75, un +125 admet `25+100` ET `50+75` — l'increment reste alors brut. Se taire vaut
  mieux qu'un faux nom.

**Bout en bout sur un vrai film** (`1bc77d2e`, CTF, 3 joueurs suivis, `cmd/tmp_awards`) :
100 evenements etiquetes, **56 nommes sans ambiguite**, 8 issus d'un increment compose,
**0 sans nom ni candidate**. Extrait :

```
    60.324  JGtm                +100  killed_player
    60.324  Chocoboflor          +50  kill_assist
    69.866  JGtm                 +25  l'un de : flag_returned, flag_stolen, runner_stopped
   141.388  Madina97294          +10  flag_taken
```

Les 44 evenements non nommes le sont pour une raison **lisible et non pour un echec** :
`killed_player` et `flag_capture_assist` valent tous deux 100, donc un joueur qui a les deux
au compteur ne peut pas etre departage par la seule valeur ; idem pour les six recompenses a
25 points. **JGtm, qui n'a pas de `flag_capture_assist`, voit ses 24 frags nommes 24/24.**

**Ce qui leverait le reste, et c'est le prochain lot** : la **coincidence temporelle**.
`runner_stopped` doit tomber sur un frag du joueur, `flag_capture_assist` sur une capture de
son equipe, `flag_returned` sur un evenement de drapeau. Ces trois regles sont des
HYPOTHESES a mettre a l'epreuve avec un controle negatif, pas a coder d'emblee — le
chantier a paye deux fois pour l'avoir oublie (§5, regle 4).

---

## 17. L'IDENTIFIANT UNIQUE : ce n'est pas la valeur du score, c'est LE COMPOSANT

> Objection de l'utilisateur, et elle est juste : « l'un de : flag_returned, flag_stolen,
> runner_stopped » est inexploitable. Il faut un identifiant unique. **Il existe, et le film
> le porte** — la §16.6 lisait la mauvaise chose.

### 17.1 Le registre du film nomme le schema, et il dit 28 emplacements de statistique

`chunk_00` (type 1) est le registre ECS : un bloc par archetype, chaque bloc portant la liste
ORDONNEE des noms de composants — exactement l'ordre que la boucle de composants itere, donc
l'index de la liste creuse. `filmdec.ParseRegistryChunk` le lit deja.

L'archetype **6** (le statborg) rend :

| index | nom du composant |
|---|---|
| 0 a 27 | `statborg-current-round-value-stat-component` (**28 emplacements**) |
| 28 a 55 | `statborg-finalized-rounds-values-stat-component` (28, valeurs de fin de manche) |
| 56 | `statborg-round-outcomes-component` |
| 57 | `statborg-entry-index-and-type-component` |

Le nom ne distingue donc pas les 28 : **c'est l'INDEX qui est l'identite de la statistique**.
Ce que confirme le getter natif releve par l'archive
(`Team_GetCurrentRoundStatValue` @ `0x142C6B118`) :
`value_raw = *(int32*)(world + statSlot*0x88 + teamIdx*0x1DF0 + 0x38 + round*4)` — `statSlot`
est bien un rang dans une table de statistiques.

### 17.2 Chaque recompense a SON emplacement — verifie nominativement

Confrontation des valeurs finales d'un slot d'entite aux recompenses `personal_score_awards`
du joueur correspondant. **JGtm, film `696a9d7c` (Strongholds), slot 22** :

| recompense de l'API | compte | composant du film |
|---|---|---|
| `killed_player` | 9 | **comp 2, valeur A** = 9 |
| `kill_assist` | 7 | **comp 3, valeur A** = 7 |
| `zone_captured` | 7 | **comp 20, valeur B** = 7 |
| `zone_secured` | 2 | **comp 21, valeur A** = 2 |
| (score personnel) | 1 650 | **comp 1, valeur B** = 1 650 |

Controle d'ensemble : sur ce film, comp 20 B totalise **61** et comp 21 A **16** ; leur somme
vaut **77**, exactement le total `zone_captures + zone_secures` de l'API.

**Et le cas exact qui bloquait**, JGtm sur `1bc77d2e` (CTF), slot 18 :

| recompense | compte | composant |
|---|---|---|
| `killed_player` | 24 | comp 2 A = 24 |
| `flag_stolen` | **4** | **comp 24 A = 4** |
| `runner_stopped` | 2 | comp 21 B ou comp 23 A (les deux valent 2) |
| `flag_returned` | 2 | l'autre des deux |
| `kill_assist` | 2 | comp 3 A = 2 |
| (score personnel) | 3 010 | comp 1 B = 3 010 |

Les trois recompenses a 25 points **vivent dans trois composants distincts**. L'ambiguite de
la §16.6 venait de la methode (lire la VALEUR de l'increment), pas du film. `flag_stolen` est
deja separe sur ce seul film ; `runner_stopped` et `flag_returned` occupent deux composants
identifies, il ne reste qu'a dire lequel est lequel — un second film ou leurs comptes
different suffit.

### 17.3 La consequence de conception

**On ne lit pas un increment de score, on lit QUEL composant a bouge.** Chaque emission d'un
composant de statistique est un evenement uniquement identifie, horodate a la milliseconde,
et attribuable au joueur par le slot d'entite. Plus de quota, plus de liste de candidates,
plus d'heuristique temporelle.

`LabelPersonalScore` (§16.6) reste utile comme repli quand le quota est connu mais pas le
film ; il n'est plus le chemin principal.

### 17.4 Ce qu'il reste, et c'est un balayage, pas une enigme

Nommer les 28 emplacements une fois pour toutes : pour chaque film, confronter la valeur
finale de chaque composant a chaque slot au compte de la recompense correspondante dans
`personal_score_awards`, et **intersecter sur les films**. C'est exactement le solveur qui a
nomme 88 medailles (§16.3), avec le meme controle par moities disjointes.

Deux notes de faisabilite :
- l'oracle vit dans les **bases joueur** (`personal_score_awards`), qui ne sont pas tenues par
  le serveur de dev — le balayage ne depend pas de `shared_matches_v2.duckdb` ;
- il faut la correspondance slot -> joueur par film, qui s'obtient deja par les instants de
  frag (§16.2).

**Ghidra n'a pas ete necessaire** : le film porte son propre schema. Il le redeviendrait si le
balayage laissait des emplacements sans nom — le getter `0x142C6B118` et la table qu'il indexe
sont alors la cible.

### 17.5 Le balayage : les emplacements nommes, et le piege qu'il a revele

Solveur `cmd/tmp_statnames` : pour chaque couple (film, joueur suivi), la valeur finale de
chaque composant est confrontee au compte de chaque recompense de `personal_score_awards`,
puis les candidates sont intersectees sur les films. L'identite du slot vient du compte de
frags (`killed_player` = comp 2 A) — **cette ancre est donc circulaire pour comp 2 A seul**,
et son nommage ne compte pas comme resultat.

**Le premier balayage, tous modes confondus, etait FAUX** — et c'est lui qui a appris la
regle. Il rendait `comp 21 A = flag_captured` quand la verification nominative sur le
Strongholds disait `zone_secured`. Cause : **le sens d'un emplacement depend du MODE**.
Intersecter des films de modes differents force un nom unique sur une case qui en porte
plusieurs. Le mode se lit sans base — les recompenses le trahissent (`flag_*`, `zone_*`,
`hill_*`, `ball_*`).

Partitionne par mode, tout redevient coherent.

**Modes a zones** (56 films, 76 couples) :

| composant | recompense | observations |
|---|---|---|
| comp 20 B | **`zone_captured`** | 73 |
| comp 21 A | **`zone_secured`** | 54 |
| comp 3 A · comp 12 B | `kill_assist` | 69 · 68 |
| comp 2 A · comp 12 A | `killed_player` (ancre) | 76 |

**CTF** (147 films, 145 couples) — et c'est la reponse a l'objection :

| composant | recompense | observations |
|---|---|---|
| comp 23 A | **`flag_returned`** | 80 |
| comp 24 A | **`flag_stolen`** | 105 |
| comp 21 B · comp 23 B | **`runner_stopped`** | 75 |
| comp 22 A | `flag_taken` | 67 |
| comp 20 B | `flag_capture_assist` | 47 |
| comp 0 A · comp 21 A · comp 5 B | `flag_captured` | 54 · 54 · 74 |

**Les trois recompenses a 25 points occupent trois composants distincts.** L'ambiguite de la
§16.6 est levee : on lit le composant, pas la valeur.

**Ce qui n'est pas ferme, et il faut le dire** : plusieurs cles portent le meme nom. Une
partie est reelle (le statborg duplique des statistiques — `comp 12 A` reproduit `comp 2 A`
sur 76/76 observations), une autre est une coincidence de comptes que seul un controle par
moities disjointes trancherait. Table figee : `.ai/refs/TABLE_STATS_STATBORG.tsv`.

**Angle mort assume** : l'oracle ne couvre que les **4 joueurs suivis** (les bases joueur),
d'ou 588 couples ecartes sur 1 531. Un oracle sur les 8 joueurs viendrait de
`match_objective_stats_latest` — dans la base partagee, tenue en ecriture par le serveur de
dev pendant cette session.

### 17.6 LE CONTROLE SUR MOITIES DISJOINTES — il recale la table de 17.5

La table de 17.5 sur-affirmait, et le controle le dit. Ajustee separement sur les films de
rang pair et de rang impair (aucun film partage), puis comparee :

| mode | cles nommees des deux cotes | **desaccords** |
|---|---|---|
| CTF | 19 | **8** |
| zones | 7 | **1** |

**Ce qui SURVIT** — et ce sont exactement les cles a forte observation, les revendications
utiles :

| mode | composant | recompense |
|---|---|---|
| CTF | comp 23 A | **`flag_returned`** |
| CTF | comp 24 A | **`flag_stolen`** |
| CTF | comp 21 B | **`runner_stopped`** |
| CTF | comp 22 A | `flag_taken` |
| CTF | comp 20 B | `flag_capture_assist` |
| CTF | comp 0 A · comp 21 A | `flag_captured` |
| zones | comp 20 B | **`zone_captured`** |
| zones | comp 21 A | **`zone_secured`** |
| les deux | comp 2 A · comp 12 A | `killed_player` (ancre, circulaire) |
| les deux | comp 3 A · comp 12 B | `kill_assist` |

**Ce qui TOMBE** : `5 B`, `6 A`, `8 A`, `8 B`, `9 A`, `22 B`, `23 B`, `25 A` en CTF et `5 B`
en zones — exactement les cles a faible observation que 17.5 signalait comme suspectes. Elles
etaient nommees par coincidence de comptes, pas par une correspondance reelle.

`comp 21 A` vaut `zone_secured` en zones et `flag_captured` en CTF : la dependance au mode
est desormais adossee au controle, plus seulement a une observation.

**La table figee (`.ai/refs/TABLE_STATS_STATBORG.tsv`) ne garde que les cles confirmees.**

---

## 18. LA LECTURE PAR COMPOSANT EST CODEE ET RECETTEE (2026-08-02)

> Lot 1 du handoff `HANDOFF_EVENEMENTS_NOMMES_2026-08-01.md` §4. `NamedEvents(src, mode)`
> vit dans `internal/analysis/objectiveevents/named.go`, adossee a la table du §17.6.
> Elle remplace `LabelPersonalScore` comme chemin principal ; celle-ci reste le repli quand
> le quota est connu mais pas le film.

### 18.1 Le mecanisme, et le seul choix de conception qu'il demande

Une emission d'un emplacement de statistique EST un evenement. Le compteur d'un slot passe
de n a n+1 : c'est une occurrence, datee de l'emission, attribuee au joueur par son slot.
Les increments de plus d'une unite sont convertis en autant d'evenements de meme instant.

Deux filtres, et tous deux ont ete payes par un echec :

1. **La plus longue sous-suite NON DECROISSANTE** (et non strictement croissante, comme
   pour le score de mode) : un composant porte deux valeurs et il est reemis des que l'UNE
   des deux bouge, donc la meme valeur revient legitimement. `longestRun(pts, strict)`
   sert les deux usages.
2. **Les emissions negatives sont jetees AVANT le choix de la sous-suite.** Les jeter
   apres ne suffit pas : sur la suite (1, -115, 1), la sous-suite retenue devenait
   (-115, 1) et l'evenement etait date de la DERNIERE emission au lieu de la premiere.
   Sans aucun filtre, cette meme suite rendait **116** evenements au lieu d'un
   (`1bc77d2e`, slot 24, comp 0 A).

### 18.2 La recette — 30 confrontations exactes sur 30

Confrontation des comptes decodes a `personal_score_awards`, sur les joueurs suivis
(oracle regenere le 2026-08-02, il reproduit a l'identique le balayage de la veille) :

| mode | film | resultat |
|---|---|---|
| zones | `696a9d7c` | slot 22 = JGtm **4/4** · slot 20 = Madina97294 **4/4** (zero inclus) |
| CTF | `1bc77d2e` | JGtm, Chocoboflor, Madina97294 — **exact sur tout sauf `flag_taken`** |
| CTF | 6 films (recette `cmd/tmp_namedcheck`) | **EXACT partout**, un seul ecart : `flag_taken` |

Controle d'ensemble sur les HUIT joueurs, qui ne depend d'aucune correspondance
slot -> joueur : sur `696a9d7c`, `zone_captured` totalise **61** et `zone_secured` **16**,
somme **77** — exactement le total de l'API, retrouve independamment du §17.2.

**Ce que la lecture par valeur ne pouvait pas faire** : `zone_captured` et `zone_secured`
valent tous deux 25 points ; `flag_returned`, `flag_stolen` et `runner_stopped` aussi. Ils
se separent ici parfaitement, parce qu'on ne lit plus la valeur mais l'emplacement.

### 18.3 `flag_taken` — le film est PLUS FIN que l'API, et c'est mesure

Seul ecart de toute la recette, et il a **un seul sens** : le film compte parfois plus,
**jamais moins**. Sur 8 couples (film, joueur) : 4 exacts, 4 au-dessus, ecart total +11,
pire ecart +5, **zero contre-exemple**.

L'explication vient de l'utilisateur et elle colle a la mesure : ramasser le drapeau au sol
pendant sa course se compte a chaque fois, mais ne se **recompense** pas a chaque fois.
Madina97294, qui joue en lancant et rattrapant le drapeau, est a 16 contre 4 ; JGtm, qui
prefere l'attraper et courir, est a 3 contre 1.

C'est le meme precedent qu'en §16.4 (Total Control : facteur 32) — **le film est plus fin
que la reference, pas faux**. Le test encode donc une INEGALITE (`film >= API`) et non une
egalite : un « film moins » serait fatal a cette lecture, on ne peut pas rater une action
qu'on recompense.

### 18.4 Le controle croise interne, gratuit et sans oracle

Le statborg duplique certaines statistiques : `comp 12 A` reproduit `comp 2 A` (frags),
`comp 12 B` reproduit `comp 3 A` (assistances), `comp 0 A` reproduit `comp 21 A`
(captures). Les emplacements redondants n'emettent pas — sinon chaque frag compterait
double — mais `CrossCheckNamedEvents` les confronte a leur canonique et signale tout
desaccord. **Aucun desaccord sur les deux films de reference**, une fois le filtre des
negatives pose au bon endroit ; c'est d'ailleurs ce controle qui a demasque le parasite
a -115.

### 18.5 Ce que ce lot NE fait pas

- **KOTH et Oddball ne sont toujours pas nommes.** `KnownAwards("hill")` et
  `KnownAwards("ball")` rendent un inventaire VIDE, et `NamedEvents` y rend nil plutot
  qu'un nom invente. Le balayage est le meme, c'est le corpus qui manque.
- **L'oracle reste celui des 4 joueurs suivis.** `match_objective_stats_latest` (8 joueurs,
  426 matchs) est disponible — la base partagee s'est liberee — mais n'a pas ete branchee.
- **Aucun cablage cote API ni cote rejeu 2D** : `NamedEvents` est une fonction pure, encore
  sans appelant de production.

### 18.6 AVERTISSEMENT D'EXPLOITATION — le balayage corpus est une bombe

`StatRecords` ancre par balayage BIT A BIT et alloue une map par enregistrement retenu. Sur
un film sain le pic est de 16 a 55 Mo, mais **un balayage corpus (56 a 147 films) a rendu
la machine de l'utilisateur inutilisable deux fois le 2026-08-02**, jusqu'a redemarrage
force — aggrave par un `go build` lance en parallele.

Regle pour la suite, non negociable : balayage **au premier plan uniquement**, un seul
processus, borne `LIMIT=n` explicite, **plafond memoire surveille qui tue le processus**, et
prevenir l'utilisateur avant tout balayage large. C'est sa machine qui paie.

---

## 19. LE BINAIRE NOMME LES STATS — 123 noms lus, sans un seul film balaye (2026-08-02)

> **Objection de l'utilisateur, et elle redresse le chantier** : « en quoi c'est necessaire de
> balayer autant de matchs ? on cherche a comprendre comment c'est exploite dans le jeu et le
> film, c'est de la retro-ingenierie pas de l'exploration de films ».
>
> Elle est juste. Le §17.4 le disait deja — « Ghidra le redeviendrait si le balayage laissait
> des emplacements sans nom » — et le balayage laissait `hill` et `ball` sans nom. La voie
> statistique etait le mauvais outil, et elle coutait la machine de l'utilisateur (§18.6).

### 19.1 Acces : le bridge MCP est casse, le plugin repond quand meme

Le bridge Python de `ghidra-mcp` echoue sous Windows (`module 'socket' has no attribute
'AF_UNIX'`) : la decouverte se fait par socket UNIX, que CPython Windows n'expose pas. Un
redemarrage cote Ghidra n'y change rien, c'est le CLIENT qui est en cause.

**Contournement, et il suffit** : le plugin GhidraMCP expose une API HTTP sur `127.0.0.1:8089`.
`curl` la joint directement — `/get_current_program_info`, `/search_strings?search_term=`,
`/decompile_function?address=`, `/get_xrefs_to?address=`, `/read_memory?address=&length=`.
Programme charge : `HaloInfinite.exe`, image base `0x140000000`, 311 104 fonctions.

### 19.2 Les noms des stats sont EN CLAIR dans le binaire

Ce que l'API expose sous `personal_score_awards.award_name` (`flag_captured`, `zone_secured`...)
**n'existe pas** dans l'executable : ces noms-la sont cote serveur. Les noms INTERNES, eux, y
sont, en `<Famille>Stats_<Nom>` — **218 chaines, 123 noms de stats, 10 familles** :

| famille | n | contenu |
|---|---:|---|
| `CoreStats_` | 48 | Score, PersonalScore, Kills, Deaths, Assists, KDA, Accuracy... |
| `InfectionStats_` | 12 | AlphasKilled, SpartansInfected... |
| `ElimStats_` · `CtfStats_` | 11 · 11 | |
| `VipStats_` · `BombStats_` | 9 · 9 | |
| `StrongholdsStats_` · `StockpileStats_` · `OddballStats_` | 6 · 6 · 6 | |
| `ExtractionStats_` | 5 | |

Table figee : **`.ai/refs/TABLE_STATS_BINAIRE.tsv`** (famille, rang, nom, adresse de la chaine).

**CTF** : FlagCaptures · FlagReturns · FlagSteals · FlagCaptureAssists · FlagCarriersKilled ·
**FlagGrabs** · FlagReturnersKilled · FlagSecures · KillsAsFlagCarrier · KillsAsFlagReturner ·
TimeAsFlagCarrier.

**Oddball** (le `ball` que le balayage n'avait pas nomme) : TimeAsSkullCarrier ·
SkullCarriersKilled · KillsAsSkullCarrier · LongestTimeAsSkullCarrier · SkullGrabs ·
SkullScoringTicks.

### 19.3 KOTH N'A PAS de famille de stats — et c'est une reponse, pas un echec

Aucune chaine `*Stats_*Hill*`, `KothStats_` ni `KingOfTheHill` cote stats. Ce n'est pas un trou
de recherche : c'est **coherent avec `match_objective_stats`**, qui porte des colonnes `flag_*`,
`zone_*`, `skull_*`, `power_seed_*`, `extraction_*`, `vip_*` et **aucune colonne `hill_*`**.

Consequence pour le lot 2 du handoff : « nommer `hill` » n'a probablement pas d'objet tel qu'il
etait formule. `hill_control` / `hill_scored` de `personal_score_awards` sont des RECOMPENSES DE
SCORE, pas des stats de boxscore ; les collines sont vraisemblablement comptees par
`StrongholdsStats_*` (une colline est une zone). **A confirmer sur un film KOTH — non fait.**

### 19.4 `CtfStats_FlagGrabs` confirme l'ecart de `flag_taken`, et par une source independante

Le §18.3 mesurait sur le film que `flag_taken` compte parfois plus que l'API, jamais moins, et
l'utilisateur l'expliquait par le style de jeu (lancer et rattraper le drapeau pendant sa
course). Le binaire tranche : le jeu compte des **`FlagGrabs`** — des RAMASSAGES. L'API
recompense des `flag_taken`. Deux quantites differentes, et le film porte la plus fine.

**La mesure et le binaire disent la meme chose sans rien se devoir.**

### 19.5 La corroboration croisee, et c'est le resultat le plus fort de la section

`CoreStats_` dans l'ordre de ses chaines commence par :

| rang | nom binaire | ce que la MESURE sur film disait, etablie independamment |
|---:|---|---|
| 0 | **Score** | `comp 0 A` = score de MODE (283/284 contre capture CE) |
| 1 | **PersonalScore** | `comp 1 B` = score PERSONNEL |
| 5 · 6 | **Kills** · **Deaths** | `comp 2` = frags ET morts (score.go le notait deja) |
| 7 | **Assists** | `comp 3 A` = `kill_assist` |

Les deux premiers emplacements du statborg portent exactement les deux premieres stats du
binaire. Le decodage du film et l'ordre du binaire se confirment mutuellement.

### 19.6 Ce qui reste ouvert — l'index, pas le nom

La correspondance **nom -> index d'emplacement** n'est PAS etablie, et il faut le dire net.
L'identifiant de stat est attribue A L'EXECUTION : `FUN_1403a77e0` appelle
`FUN_140748a74(PTR_s_CtfStats_FlagCaptures_1443d10c0, ...)` et range le retour dans des globales
CONSECUTIVES (`_DAT_1451a28a0`, `+4`, `+8` pour Captures/Returns/Steals). L'ordre d'enregistrement
fait l'index ; il n'est pas ecrit en dur.

Ce qui EST etabli sur la structure :
- une **table de descripteurs** a `0x1443d10c0`, **stride `0x50`** (10 pointeurs), dont le
  **premier champ est le pointeur vers le nom** — verifie sur Captures (`+0x00`), Returns
  (`+0x50`), Steals (`+0xA0`) ;
- la statline monde vit a `world + statSlot*0x88 + teamIdx*0x1DF0 + 0x38 + round*4`, et
  `FUN_140807ebc` boucle sur **56 stats** de stride `0x88` — soit exactement les 28 emplacements
  `current-round` + 28 `finalized-rounds` du registre du film. **`statSlot` est donc bien
  l'index de composant** ;
- une table indexee par stat, **stride `0xC0`**, a `base + 0xdf78c` (nombre de stats a
  `+0xdf77c`), dont un champ est teste `!= 4` — un TYPE de stat. Sa base est allouee au
  runtime (`DAT_145121d28 + 0x28 + (DAT_1445c5838 & 0xffff) * 0x1134f0`), donc **illisible en
  statique**.

**Les deux routes pour fermer l'index**, aucune tentee :
1. lire les globales d'ID en memoire, jeu lance (Cheat Engine ou le debogueur Ghidra) — direct
   et definitif ;
2. reconstruire l'ordre d'enregistrement statiquement, en listant tous les appelants de
   `FUN_140748a74` et l'ordre des initialiseurs — plus fragile.

En attendant, la table du §17.6 reste la source pour CTF et zones : elle est MESUREE, et le §18.2
la recette a 30 confrontations exactes sur 30.

---

## 20. LE PONT VERS LE REJEU 2D — l'identite, pas le numero de slot (2026-08-02)

> Lot 4 du handoff §4. **Livre : le pont.** Non livre : l'integration dans le document de
> rejeu (cf. §20.4).

### 20.1 Le piege qu'il fallait voir avant de coller quoi que ce soit

Les evenements nommes portent un slot d'entite STATBORG (10..24 pairs). Le rejeu 2D indexe
ses trajectoires par slot de BIPED et identifie ses vies par XUID. **Ce sont deux espaces de
slots differents.** Les confondre aurait colle les evenements sur les mauvais joueurs — et sur
une carte, l'erreur serait invisible et credible.

Le pont ne peut donc pas etre le numero. C'est le XUID.

### 20.2 La methode : un triplet exact, pas une fenetre temporelle

`SlotIdentity(src, lines)` apparie chaque slot a une ligne de `match_participants` par le
triplet **(frags, morts, assistances)**.

Ce que cela remplace : l'appariement par coincidence temporelle du §16.2 (apparier les
increments de +100 aux instants de frag). Celui-ci marchait, mais exigeait une fenetre de
tolerance et une source externe d'instants de frag. **Le triplet compare trois entiers
exacts** — ni fenetre, ni seuil, ni parametre.

**Mesure** : appariement UNIQUE sur les **8 joueurs des deux films de reference**, et les
slots rendus concordent avec les identites que le §16.2 avait etablies par l'autre chemin
(slot 18 -> JGtm `2533274823110022` sur `1bc77d2e`). Deux methodes independantes, meme
resultat.

**Prudence codee** : un slot dont le triplet ne designe pas UNE seule ligne n'est pas
apparie, et un xuid que deux slots se disputent n'est attribue a aucun des deux.
`IdentifyNamedEvents` ECARTE les evenements des slots non apparies plutot que de les
attribuer par defaut — la perte est observable par difference avec `NamedEvents`.

### 20.3 Et il confirme trois emplacements de plus, nominativement

L'appariement a exige de verifier `comp 2 B`. Contre `match_participants` sur `696a9d7c` :

| emplacement | ce qu'il porte | verification |
|---|---|---|
| `comp 2 A` | frags | 8/8 exacts |
| **`comp 2 B`** | **morts** | **8/8 exacts** |
| `comp 3 A` | assistances | 8/8 exacts |
| `comp 1 B` | score personnel | slot 22 = **1 650**, exactement le §17.2 |

Le binaire dit la meme chose : `CoreStats_` declare `Score`, `PersonalScore`, ..., `Kills`,
`Deaths`, `Assists` (§19.5). **La mesure sur film et l'ordre du binaire se confirment.**

Consequence pratique : l'appariement ne depend d'AUCUN mode — frags, morts et assistances
sont repliques quel que soit le type de partie. Il fonctionne donc aussi en Slayer, KOTH et
Oddball, ou aucun emplacement d'objectif n'est nomme.

### 20.4 Ce qui N'EST PAS fait, et il faut le lire avant de croire le lot clos

Le document de rejeu (`internal/analysis/replay.ReplayDocument`) **ne porte pas encore ces
evenements**. Il manque, et ce n'est pas trivial :

1. un champ optionnel (`Events []IdentifiedEvent`, `omitempty`) et sa ligne de `Coverage` —
   publier des evenements attribues sans dire combien ne l'ont pas ete laisserait croire a
   l'exhaustivite ;
2. la conversion `TimeMS` -> index de frame `T` via `FrameIntervalMS`, avec la question de
   l'arrondi (un evenement tombe entre deux frames) ;
3. le cablage dans `cmd/replay-build`, qui est HORS LIGNE et n'ouvre aucune base — les
   `PlayerLine` doivent donc lui etre fournies en entree, comme `Extract` recoit deja son
   `Roster` ;
4. le rendu cote client.

L'horloge, elle, ne pose pas de probleme : le `TimeMS` des evenements est sur l'horloge du
manifeste, la meme que les positions — superposable sans recalage (§15).

---

## 21. KOTH : AUCUN COMPTEUR DE COLLINE N'EST REPLIQUE — mesure, plus deduction (2026-08-02)

> Lot 2 du handoff §4. Le §19.3 le DEDUISAIT du binaire ; cette section le MESURE sur film,
> ce qui n'est pas la meme chose.

**Protocole** : balayage des 28 emplacements (les deux cotes) sur **deux films KOTH**,
`d2b74083` et `cae8471d`. Slots identifies par le couple (frags, assistances), unique dans
les deux cas.

**Le bareme valide l'identite avant toute conclusion** — `comp 1 B` (score personnel) :

| film | joueur | detail | comp 1 B |
|---|---|---|---|
| `d2b74083` | JGtm | 6x100 + 1x50 + 8x25 + 1x100 | **950** |
| `d2b74083` | Madina97294 | 8x100 + 2x50 + 2x25 + 1x100 | **1 050** |
| `cae8471d` | JGtm | 10x100 + 5x50 + 13x25 + 2x100 | **1 775** |
| `cae8471d` | Madina97294 | 14x100 + 5x50 + 6x25 + 1x100 | **1 900** |

Quatre egalites exactes : les slots sont les bons, et `hill_control` = 25 pts,
`hill_scored` = 100 pts sont confirmes.

**Le resultat** : **aucun** des 56 emplacements ne porte `hill_control` (8 et 2 sur le
premier film, 13 et 6 sur le second) ni `hill_scored`. Verifie emplacement par emplacement.

**Une hypothese posee et REFUTEE** : `comp 22 B` valait 551 / 135 sur le premier film, soit
un rapport 4,08 contre 4,00 pour `hill_control` — de quoi croire a un temps de controle dont
la recompense derive. Le second film la tue : 1 025 / 666 = **1,54** quand `hill_control`
donne **2,17**. C'etait une coincidence, et c'est le controle sur un second film qui l'a dit.

**Trois sources concordantes** : la mesure sur film (ci-dessus), le binaire (aucune famille
`*Stats_*Hill*`, §19.2) et la base (`match_objective_stats` n'a aucune colonne `hill_*`).

**Conclusion du lot 2** : `ball` est nomme (§19.2, famille `OddballStats_`), `hill` **n'a pas
d'objet** — il n'y a rien a nommer. `KnownStats("hill")` rend un inventaire vide et
`NamedEvents` y rend nil, ce qui est le comportement juste et non une lacune.

---

## 22. L'ORACLE A HUIT JOUEURS — il double la recette ET corrige un NOM FAUX (2026-08-02)

> Lot 3 du handoff §4. La base partagee s'etant liberee,
> `match_objective_stats_latest` (426 matchs x 8 joueurs) a ete confronte au decodage.

### 22.1 Ce que l'oracle a huit joueurs change

`personal_score_awards` ne couvre que les 4 joueurs suivis ; `match_objective_stats_latest`
couvre les 8. La recette passe de **30 confrontations a 64**, et surtout elle change de
nature : un decodage qui n'aurait marche que sur les joueurs suivis n'y survivrait pas.

| film | mode | confrontations | resultat |
|---|---|---|---|
| `696a9d7c` | zones | 16 (8 joueurs x 2 compteurs) | **toutes exactes** |
| `1bc77d2e` | CTF | 48 (8 joueurs x 6 compteurs) | **toutes exactes** |

### 22.2 LE NOM ETAIT FAUX, et c'est le gain principal du lot

`comp 22 A` etait nomme **`flag_taken`** d'apres `personal_score_awards`. Le §18.3 avait
mesure qu'il comptait systematiquement PLUS que cette recompense, jamais moins, et l'avait
mis au compte d'un « film plus fin que l'API ». **C'etait une erreur de nom, pas une
finesse.**

L'oracle a huit joueurs porte une colonne `flag_grabs`, et les valeurs de `comp 22 A` en sont
la copie exacte, slot par slot : **16 pour Madina97294** la ou la recompense `flag_taken`
dit 4, 13 pour un autre joueur, 3 pour JGtm. Le binaire disait la meme chose depuis le
debut : `CtfStats_FlagGrabs` (§19.2).

**La cause de fond, et elle valait d'etre corrigee** : le statborg replique des
**STATISTIQUES**, pas les **RECOMPENSES DE SCORE** que le serveur en derive. Pour la plupart
les deux coincident numeriquement (`flag_returns` / `flag_returned`), ce qui masquait la
confusion. Ramasser le drapeau au sol se COMPTE a chaque fois mais ne se RECOMPENSE pas a
chaque fois — d'ou l'ecart, et l'explication de l'utilisateur (« Madina lance et ramasse le
drapeau pendant sa course ») etait la bonne des le depart.

**Consequence de code** : les constantes du paquet passent de `Award*` a `Stat*` et portent
desormais les noms de `match_objective_stats` (`flag_grabs`, `flag_captures`, `kills`,
`assists`...). Le champ `NamedEvent.Award` devient `NamedEvent.Stat`.

### 22.3 Ce que le lot 3 n'a PAS fait

Le handoff proposait de **rejouer le balayage** avec cet oracle pour lever les doublons. Ce
balayage n'a pas ete rejoue : il n'est plus necessaire pour nommer (le binaire le fait,
§19), et il reste ce qui a rendu la machine inutilisable (§18.6). L'oracle a servi a
**recetter et corriger**, ce qui etait sa valeur reelle.

Les doublons connus (`12 A` = `2 A`, `12 B` = `3 A`, `0 A` = `21 A`) restent des doublons
REELS, verifies par le controle croise interne — il n'y a rien a departager.
