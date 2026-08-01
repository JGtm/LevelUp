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
| Q2 | Porteur autoritaire du score d'equipe | **NON CONCLU — 2 candidats ELIMINES** | L'octet d'etat externe et le comptage de ticks tombent tous deux sur controle negatif |
| Q3 | La recette mode -> score | **NON CONCLU — route affinee** | La piste « table de dispatch par event-type » est cassee (voir 5.3) ; la piste statborg + categorie de variante reste ouverte et est desormais cadree |
| Q4 | Evenements d'objectif nommes | **ETABLI pour les zones · REFUTE pour CTF · INDECIDABLE pour KOTH** | Zones : 3 chaines independantes, egalite EXACTE 8/8 par joueur |
| Q5 | Famille pickup (sous-produit) | **CONSIGNE, non creuse** | Section 7 |

**Le resultat de la session, en une phrase** : en mode a zones, un evenement de mode du footer
type-3 **EST** une prise ou une securisation de zone, par joueur, a l'unite — total 77 = 77 et
**8 joueurs sur 8** en egalite exacte, avec l'ancre terrain qui tombe sur l'acteur ET sur la
seconde ; et l'hypothese d'un intermediaire universel mode -> score n'a **aucun** support
mesure a ce stade, les deux porteurs candidats testes ayant ete elimines par controle negatif.

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

**« Il n'y a pas de recette » reste recevable** et n'est pas ecarte : la section 5.2 montre au
contraire que le meme porteur d'evenement change de semantique selon le mode.

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

## 7. Q5 — LA FAMILLE PICKUP, CONSIGNEE SANS ETRE CREUSEE

Ce que cette session peut deposer pour la piste A5 (ramassages), sans avoir decode quoi que ce
soit : l'histogramme complet du premier enregistrement de trame par mode (3.2) fournit la
**liste des codes candidats et leurs volumes**, ce qui manquait pour cibler. Les codes presents
dans les cinq modes avec des volumes compatibles avec des ramassages (quelques centaines par
film) sont **96, 97, 98, 99, 101, 114, 115, 116**. Aucun n'est nomme de facon fiable — la
section 5.3 explique pourquoi la voie du nommage par table est fermee.

**Ne pas partir de la** sans avoir d'abord relu `KILLFEED_STATE.md` §183 : le dispatcher
`FUN_140620564` (codes 0x02-0x3c) est un **apply**, pas un decode, et il consomme une structure
deja desérialisée.

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
| `cmd/tmp_modeticks` | evenements de mode : compte par equipe et par joueur, gamertag decode, horloge BE, deduplication, intervalles |

Tous se construisent en `CGO_ENABLED=0` (aucun DuckDB). Les oracles en base passent par
`cmd/diag_q` en lecture seule.

---

## 10. CE QU'IL FAUT FAIRE ENSUITE, PAR ORDRE DE VALEUR

1. **Un second film KOTH** — c'est le seul point qui bloque un verdict Q4 complet, et il coute
   une mesure (l'outil est ecrit).
2. **Q2 non traite** : la localisation du porteur de score par mode n'a **pas** ete faite. Elle
   demande le balayage de colonnes ancre sur `0x7B6` avec **comptage publie des faux positifs**
   (combien de colonnes passent par hasard) — sans ce comptage, la mesure n'est pas publiable.
   Les temoins sont prets : Slayer = la timeline de frags, Strongholds = 21 puis 69-30 aux
   ancres puis 200-94, CTF = `FlagCaptures`.
3. **Q3** : reprendre par le binding statborg (5.3), pas par la table de dispatch.
