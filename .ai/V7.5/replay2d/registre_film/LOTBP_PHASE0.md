# Lots B + P — phase 0 (MESURER) : ce que le film dit du MOTEUR DE PARTIE et du JOUEUR

> Executeur, 2026-08-18, worktree `LevelUp-wt-joueur-moteur`, branche `wt/joueur-moteur`.
> Perimetre ferme : B.0.1-B.0.5 et P.0.1-P.0.5 du plan
> `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`. Lecture seule, aucune publication,
> aucun champ du document de rejeu. Seuils ECRITS AVANT la mesure (D13), jamais rebaisses.

## 0. Le resultat en une phrase

**L'entite MOTEUR DE PARTIE (ti=0) n'est pas dans le film** : 1 record sur 22 films et
1 269 000 records certains, sur un archetype de 27 composants pourtant declare au registre.
Tout le lot B en depend — horloge officielle, mort subite, etats, manches, conditions de fin :
les cinq canaux sont **NEGATIFS, faute de porteur**. **L'entite JOUEUR (ti=5) existe et se lit**
(27 829 records certains, 8 slots par match, 22 films), mais son **debit est trop faible d'un
a deux ordres de grandeur** pour les seuils de la phase 0 : 1 a 12 lectures par film pour le
chargement de depart, la visee de controle ou le choix de reapparition.

**Gate 0 lot B : NON ATTEINT** (B.0.2 non tenu, B.0.4 non tenu). **Gate 0 lot P : NON ATTEINT**
(P.0.3 non mesurable, P.0.2 non tenu, aucun etat de P.0.5 tenu).

---

## 1. L'instrument

| Piece | Fichier | Role |
|---|---|---|
| Scanner par bande | `filmdec/game_entities_scan_test.go` + `_bands_test.go` + `_walk_test.go` | ancrage sur bande de slots (ti=0, ti=5, temoin ti=4, deux bandes de controle) |
| Chasse au slot | `filmdec/game_entities_hunt_test.go` | `HuntArchetypeSlots` (par grammaire), `HuntGameEngineClock` (par signature d horloge) |
| Voie sequentielle | `filmdec/game_entities_chain_test.go` | `ScanFilmGameEntitiesChain` : records CERTAINS via `DecodeFrameRecords` + confirmation par image-cle |
| Instrument ti=0 / ti=5 | `filmdec/game_state_measure_test.go` + `_bands` + `_players` + `_dump` (garde `GAME_FILM`) | B.0.1/B.0.2/B.0.3/B.0.5, P.0.1, P.0.2, P.0.5 |
| Instrument des canaux joueur | `filmdec/player_bridge_measure_test.go` + `player_bridge_channels_test.go` (garde `BRIDGE_FILM`) | P.0.3, P.0.4, fenetres actives de B.0.4 |
| Instrument du fil des morts | `replay/player_state_measure_test.go` (garde `PLAYER_FILM`) | B.0.4 : delai mort -> reapparition, temps mort cumule |

> **Tout est INSTRUMENT, rien n est PRODUCTION** (regle « 0 code mort », arbitrage du superviseur
> le 2026-08-18) : aucune de ces pieces n a de consommateur de production puisque la phase 0 ne
> publie rien, donc toutes vivent dans des fichiers `_test.go`. Consequence assumee : un fichier
> de test n est pas visible depuis un autre paquet, donc les mesures qui consomment la chaine ont
> SUIVI la chaine dans `filmdec`, et seules celles qui consomment le fil des morts sont restees
> dans `replay`. Le partage est fait par DEPENDANCE, et aucun decodage n est en double. Le seul
> prix : P.0.3 compare desormais l arme du tir a une lecture trouvee sur n importe quel slot au
> lieu du seul slot du tireur — une BORNE SUPERIEURE de la couverture, donc un negatif plus
> difficile a atteindre, qui tient quand meme (0 tir couvert sur 5 219).

| Pilotes machine | `.ai/V7.5/replay2d/registre_film/lotBP/run_one.ps1`, `run_corpus.ps1` | UN film par processus, avant-plan, plafond 3 Go surveille (D17) |

### 1.1 Les deux voies de lecture, et pourquoi il en a fallu deux

**Voie A — ancrage par bande de slots** (modele `equipment_state.go` / `projectiles.go`). Elle
lit TOUS les paquets, mais c'est un reconnaisseur probabiliste : l'en-tete delta ne contraint
que 21 bits. Mesure sur `000d5950` : la bande de ti=5 (32 slots) rend **301 en-tetes par slot
contre 283 sur une bande de slots VIDES** — rapport **x1,07**. La purete de la bande ti=5
(masque tenant dans les 27 composants de l'archetype) va de **29,2 % a 62,9 %** selon le film.
**Cette voie ne peut PAS trancher un seuil sur ti=5.**

**Voie B — chaine sequentielle** (`DecodeFrameRecords` + World amorce par image-cle). Elle ne
devine rien : un record dont la traversee aboutit est certain. Son prix, mesure par le lot A et
retrouve ici : **31 a 53 % des paquets delta sont propres** (mediane ~37 %). C'est ELLE qui
tranche les seuils ; la voie A ne fait que la corroborer.

**Le durcissement qui a change le resultat.** Sans confirmation, la voie B rendait
**9 405 « records de ti=0 »** et un slot de BIPEDE (516) creditе de 558 « records de ti=5 ».
Deux artefacts, tous deux documentes dans `DecodeFrameRecords` lui-meme : (a) un record de
SUPPRESSION n'a pas de trace, donc son `TypeIndex` vaut zero par defaut — le moteur de partie
heritait de toutes les suppressions du film ; (b) un record NEW qui suit une fausse-propre est
desaligne et son `R(6)` de tete lie le slot a un archetype FAUX. **Retenir un record seulement
quand une image-cle a lie CE slot a CET archetype supprime les deux d'un coup** — et fait
tomber ti=0 de 9 405 a 0.

### 1.2 Temoins d'ancrage (B.0.1)

| Temoin | Attendu (lot C) | Mesure (22 films) | Verdict |
|---|---|---|---|
| Purete ti=4 (1 slot, 1 composant) | 98,7-99,8 % | **93,4 % a 99,8 %**, mediane 98,9 % | **La methode reproduit l'etalon** |
| Rapport reel/fantome ti=4 | — | x64,9 (voisinage) / x108,8 (vide) sur en-tetes bruts ; **x229 / x567** sur composant vise | ancrage sain sur un archetype bavard |
| Rapport reel/fantome ti=5 | — | **x0,74 a x1,39** (voisinage), **x0,85 a x5,33** (vide) | **la bande ti=5 ne se detache pas du bruit** |
| Purete ti=5 | — | 29,2 % a 62,9 % | idem |
| Comblement de plage | — | **0 slot ajoute** sur les 22 films | choix de la bande OBSERVEE confirme |
| Slots ambigus ecartes | — | 0 sur les 22 films | — |

Le temoin ti=4 est le point important : il prouve que **l'instrument fonctionne**, et que le
resultat sur ti=5 est une propriete du CANAL, pas un defaut de la methode.

---

## 2. Lot B — l'horloge officielle, les etats, les temps morts

### B.0.1 — le scanner : `[x]` fait

`ScanFilmGameEntities` (`game_entities_scan_test.go`) : bandes ti=0 et ti=5 etablies par
`WalkKeyframeWorld`, ancrage par en-tete delta, marche `consumeByNameCapturing` composant par
composant (le round-timer i5 et le respawn i1 restent sur la couche de CAPTURE, jamais
dupliques). Temoins publies ci-dessus.

**LA DECOUVERTE DE L'ITEM : la bande de ti=0 est VIDE sur les 22 films.** Le recensement des
images-cles de `000d5950` porte 28 archetypes — `ti=2:1 ti=4:1 ti=5:32 ti=6:48 ti=9:8 ti=12:3
ti=13:8 ti=14:32 ti=15:1 ti=17:33 ti=18:1 ti=19:1 ti=20:7 ti=21:8 ti=22:1 ti=25:1 ti=26:1
ti=27:1 ti=29:1 ti=34:1 ti=35:90 ti=37:278 ti=38:139 ti=41:19 ti=42:178 ti=43:2 ti=45:1
ti=47:8` — et **pas ti=0**. Les premiers slots sont `0->[22] 1->[34] 2->[2] 3..50->[6]`.

Trois chemins independants ont ete essayes avant de conclure :

1. **Chasse par grammaire** (`HuntArchetypeSlots`) : compter par slot les en-tetes dont le
   masque tient dans les 27 composants de ti=0 et annonce le round-timer. **Non discriminant** :
   les 15 premiers slots sont tous dans la plage 512-620, c'est-a-dire les BIPEDES, dont la
   densite de records fabrique assez d'en-tetes compatibles pour ecraser un archetype a une
   seule entite (slot 569 : 842 annonces ; mediane par slot : 0).
2. **Chasse par signature d'horloge** (`HuntGameEngineClock`) : marcher ces records et retenir
   la valeur capturee. **Aucun slot ne porte d'horloge** : monotonie 0,42-0,50 partout (le
   hasard vaut 0,50), amplitude A de 0 a 35 900 s, pentes de -956 a +127 s/s. Sur 3 396 slots,
   pas un seul decompte.
3. **Voie sequentielle confirmee** : **0 record de ti=0** sur 21 des 22 films, **1** sur
   `0a247154`. Total : **1 record sur 1 269 000 records certains**.

**Oracle exterieur concordant** : la capture Cheat Engine du lot C
(`.ai/V7.5/film_re/RELEVE_TERRAIN_CAPTURES_2026-07-31.md:68-70,101-102`) ventile
**1 205 704 records** (Strongholds) et **988 752** (CTF) par archetype dispatche — **ti=0 est
absent des deux listes**. La ventilation de ma voie sequentielle retrouve d'ailleurs les memes
archetypes vivants (ti=35, ti=4, ti=21, ti=5, ti=6, ti=2, ti=42, ti=38...).

**Conclusion B.0.1 : l'entite moteur de partie n'est pas repliquee dans le film.** Le statut
`porte` de ses composants a la table ECS est un statut de PORTAGE (le deserialiseur existe,
issu de la RE), pas un statut d'OBSERVATION. Les cinq desers de `components_game_engine.go`
sont donc corrects et inutilisables : ils n'ont pas de flux.

### B.0.2 — round-timer : `[!]` NEGATIF, denominateur nul

| Mesure | Seuil ecrit | Resultat | Denominateur |
|---|---|---|---|
| Pente vs horloge du film | \|pente + 1\| <= 2 % sur >= 90 % des films | **NON MESURABLE** | **0 lecture** sur 22 films |
| Valeur initiale vs `regulation.toml` | +/- 1 s sur >= 90 % | **NON MESURABLE** | 0 lecture |
| Instant du premier decompte vs `originMs` | ecart median et p90, mesure publiee (D4) | **NON MESURABLE** | 0 lecture |

Aucune lecture d'horloge de manche n'a ete obtenue, ni par bande, ni par chasse, ni par chaine.
L'`originMs` publie n'a donc rien a confronter — il reste ce qu'il est, et **D4 est respecte
sans effort** : ce lot ne touche pas a l'origine.

### B.0.3 — mort subite / prolongation : `[!]` NEGATIF, avec l'oracle construit

L'oracle a ete construit et il est utilisable tel quel : **un seul export en lecture seule**
(`registre_film/oracle_export_lotB.sql` -> `oracle_lotB_overtime.tsv`, 707 matchs AYANT UN FILM
dans le cache local et une variante presente dans `regulation.toml`). Le flag reproduit le
chemin de production a l'identique : `elapsed = MEDIAN(time_played_seconds)` des participants
presents au debut ET a la fin (repli MAX), seuil `regulation + 40` (`OvertimeMarginSeconds`).

**Exactement 10 matchs du cache sont flagues `IsOvertime`** :

| short8 | variante | carte | elapsed | depassement | film |
|---|---|---|---|---|---|
| `eb665a8f` | CTF:Arena | Origin | 976 s | +256 s | mesure |
| `4a93f0e2` | CTF:Arena | Dynasty | 966 s | +246 s | **non ajoute** |
| `521c5c38` | CTF:Arena | Forest | 946 s | +226 s | **non ajoute** |
| `8b512df2` | CTF:Arena | Forest | 937 s | +217 s | **non ajoute** |
| `eba1e63f` | CTF:Arena | Aquarius | 895 s | +175 s | mesure |
| `19ef6b04` | CTF:Arena | Aquarius | 877 s | +157 s | **non ajoute** |
| `44e14331` | CTF:Arena | Shiro | 860 s | +140 s | mesure |
| `aaaf6c76` | Strongholds:Arena | Kaiketsu | 835 s | +115 s | mesure |
| `64e8adfa` | CTF:Arena | Catalyst | 809 s | +89 s | mesure (deja au corpus) |
| `7ff4271a` | CTF:Arena | Illusion | 774 s | +54 s | mesure |

**6 des 10 ont ete mesures** (la consigne d'execution bornait l'ajout a 5 films ; les 4 restants
sont nommes ci-dessus pour le superviseur). **Aucun ne porte de lecture d'i6** : 0 record de
ti=0 sur les six. Cote temoins non flagues : **16 films du corpus** (4 Slayer, 3 CTF, 4
Strongholds, 4 KOTH, 1 Oddball ; + 1 BTB hors table de reglement) — **i6 nul partout, 0 faux
positif**, mais ce 0 est acquis trivialement puisque le canal ne parle jamais.

**Verdict** : la mort subite du film **ne peut pas** etre la source exacte de la prolongation.
La phase 3 du lot B (branchement persist) tombe. Les 2 prolongations courtes connues (+19 s,
+24 s) n'ont pas ete cherchees : sans porteur, la question ne se pose plus.

### B.0.4 — respawn : `[!]` NON TENU

Le compte a rebours de reapparition (ti=5 i1) EXISTE, lui, et se lit — mais sa fenetre ACTIVE
n'est presque jamais transmise.

| Film | lectures du compte a rebours | dont ACTIVES | morts du fil | couverture | verdict (seuil 90 %) |
|---|---|---|---|---|---|
| `000d5950` | 310 | **4** | 93 | 4,30 % | NON TENU |
| `530820e5` | 472 | **3** | 94 | 3,19 % | NON TENU |
| `7344d24f` | 591 | **1** | 117 | 0,85 % | NON TENU |
| 22 films (total) | 11 417 | **111** | — | ~1 % des lectures | NON TENU |

Les valeurs dominantes sont statiques (`false/4/8`, `false/5/8`, `false/6/8`, `false/7/8` =
92,6 % des lectures de `000d5950`) : le canal transmet une CONFIGURATION, pas un decompte
vivant.

**Ce que le lot livre quand meme, et qui est solide** : le temps mort se mesure tres bien SANS
ti=5, par les trajectoires deja decodees (fin de vie -> debut de la vie suivante du MEME joueur,
via le pont `lives.go`).

| Film | intervalles mesurables | mediane | total | moyenne |
|---|---|---|---|---|
| `000d5950` | 91 | 8,06 s | 865,3 s | 9,51 s |
| `530820e5` | 88 | 10,16 s | 932,8 s | 10,60 s |
| `7344d24f` | 111 | 10,06 s | 1 135,5 s | 10,23 s |

L'EQUIPE n'est pas resolue (le film ne la porte pas de facon fiable, decision de 2026-06) : le
total est rendu par joueur et somme, jamais devine par equipe.

### B.0.5 — etats et manches : `[!]` NEGATIF, denominateur nul

i2 (etat), i4 (manche), i8 (conditions de fin), i7 (grace) sont des composants de ti=0 :
**0 lecture confirmee sur 22 films**. La borne de manche d'`24dbb67d` (Oddball 2 manches) ne
peut donc pas etre confrontee au moment ou le score de manche 1 se fige (~290 s, lot A) :
`24dbb67d` rend 0 record de ti=0 comme les autres.

Les 47 « lectures d'i2 » et l'unique « lecture d'i6 » vues AVANT le durcissement de la chaine
etaient des artefacts de liaison NEW desalignee — elles disparaissent toutes avec la
confirmation par image-cle. **C'est exactement le piege que le temoin fantome devait attraper,
et il l'a attrape.**

### Gate 0 lot B : **NON ATTEINT**

B.0.2 non tenu (denominateur nul) ET B.0.4 non tenu (0,85-4,30 % contre 90 %). Publication
limitee a ce qui tient : **rien de ti=0**. Le seul acquis publiable du lot B est le temps mort
par joueur, qui ne vient pas de ti=0 et n'avait pas besoin de ce lot pour exister.

---

## 3. Lot P — l'entite JOUEUR (ti=5) et l'inventaire

### P.0.1 — volumes : `[x]` fait

**ti=5 EXISTE et se lit.** 22 films, voie sequentielle confirmee :

| Grandeur | Mediane | Etendue |
|---|---|---|
| Records ti=5 certains par film | 1 305 | 397 (`606d9844`, 235 s) a 2 258 (`eb665a8f`) |
| Slots ti=5 ayant parle | 25 | 16 a 32 |
| Part des 8 slots les plus bavards | **96,5 %** | 43,4 % (`06dfe6d9`, BTB 26 joueurs) a 99,3 % |

La ventilation de `000d5950` est sans ambiguite : slots 52-59 a 138/134/107/100/85/75/74/68
records, puis une queue de 1 a 5 sur les slots 60-83. **Le moteur declare 32 entites joueur
(la capacite du serveur) et 8 seulement parlent** — c'est le denominateur honnete, et c'est
aussi pourquoi la bande de 32 slots divisait le signal par quatre dans la voie A.

**Controle croise avec l'oracle exterieur** : la capture Cheat Engine du lot C compte
**ti=5 : 2 908** records (Strongholds) et **2 495** (CTF) par film. Ma voie sequentielle en rend
1 274-1 335 sur les Strongholds du corpus, soit **~45 % du trafic reel** — coherent avec les
37 % de paquets propres. **Le plafond n'est donc pas mon decodage : meme a 100 %, ti=5 resterait
un canal a ~2 700 records par film pour 27 composants.**

Annonces par composant (22 films, records certains) :

| Composant | Lectures / 22 films | Par film |
|---|---|---|
| i20 `malleable-properties` | **9 026** | ~410 |
| i1 `respawn-timer` (capture) | **11 417** | ~519 |
| i3 `target-tracking` | 336 | ~15 |
| i2 `soft-kill-timer` | 282 | ~13 |
| i14 `lives-remaining` | 206 | ~9 |
| i18 `active-in-game` | 163 | ~7 |
| i17 `control-aiming` | 150 | ~7 |
| i6 `desired-respawn-player` | 128 | ~6 |
| i11 `engine-loadout` | **117** | **~5** |
| i12 `desired-respawn-location` | 115 | ~5 |
| i15 `last-betrayer` | 109 | ~5 |
| i19 `pending-join` | 105 | ~5 |

**Deux canaux bavards, dix canaux a une poignee de lectures par film.** C'est ce tableau qui
condamne P.0.2, P.0.4 et P.0.5, avant meme d'ouvrir une valeur.

### P.0.2 — chargement de depart (i11) : `[!]` NON TENU

Seuil ecrit : bijection stable octet -> famille sur **>= 90 % des vies** de 3 films, MEME table
sur les 3.

| Film | vies (trajectoires) | lectures d'i11 certaines | couverture |
|---|---|---|---|
| `000d5950` | 105 | **7** | **6,7 %** |
| `530820e5` | 98 | **5** | **5,1 %** |
| `7344d24f` | 124 | **10** | **8,1 %** |

Le seuil de 90 % est hors d'atteinte d'un facteur 11 a 18, et l'ecart ne vient pas du croisement
mais du DENOMINATEUR. Le contenu confirme : sur les 7 lectures certaines de `000d5950`, **7
valeurs distinctes** sur 8 octets. L'hypothese « indices de palette du mode » (un meme
chargement revenant a chaque reapparition) predisait des REPETITIONS ; on n'en observe aucune.

> Nuance a consigner, parce qu'elle n'est pas nulle : la voie A (ancrage, bruit compris) voyait
> sur `000d5950` la valeur `144,150,194,160,223,248,45,103` **37 fois** sur 118 lectures, et
> `0,255,255,255,255,255,255,255` **8 fois**. Le hasard ne repete pas un octuplet 37 fois : il y
> a bien un motif structure quelque part dans ce canal. Mais il n'apparait pas dans les lectures
> CERTAINES, et une mesure faite sur une bande a 29-63 % de purete ne peut pas fonder un seuil.
> **Ecrit comme une piste, pas comme un resultat.**

Consequence sur le plan : la sonde **F1 devient `[~]`** (couverte par cette mesure, negative).

### P.0.3 — arme en main image par image : `[!]` NON MESURABLE — **et la cause est identifiee**

Seuil ecrit : au frame de chaque tir, arme en main du tireur == famille du tir, **>= 90 %**.

| Film | tirs | records de BIPEDE certains | lectures d'arme en main | tirs couverts |
|---|---|---|---|---|
| `000d5950` | 519 | 18 041 | **0** | **0 (0,0 %)** |
| `530820e5` | 1 783 | 17 828 | **4** | **0 (0,0 %)** |
| `7344d24f` | 2 917 | 21 787 | **1** | **0 (0,0 %)** |

**La cause n'est pas le decodeur : c'est le canal.** Recensement des annonces au masque sur les
records de bipede CERTAINS (le composant le plus annonce l'est 17 500 a 21 700 fois) :

| Film | i42 `desired-weapon-set` | i43 | i44 | i45 | i46 (`weapon-state-type-info`) |
|---|---|---|---|---|---|
| `000d5950` | 11 | 1 | 0 | 1 | 0 |
| `530820e5` | 41 | 0 | 4 | 1 | 0 |
| `7344d24f` | 29 | 2 | 2 | 2 | 0 |

**Les quatre composants d'identite d'arme sont annonces 0 a 4 fois par film**, contre ~20 000
pour le composant dominant. L'arme en main n'est **pas** transmise en delta ; `World.HeldWeapon`
est alimente par un canal qui ne parle pratiquement jamais. La latence de changement d'arme n'a
donc pas de sens a mesurer (1 transition relevee sur les 3 films).

**Consequence directe sur la ligne « Decouvertes » du plan** : `World.HeldWeapon` (0 appelant)
n'est pas seulement inutilise, **il est inutilisable par cette voie**. La decision de le
supprimer ou de le garder revient au plan item 4 (canal n°1), avec ce chiffre en main.

Les munitions en delta (i30/i31/i33/i34) n'ont PAS ete instrumentees : le controle de bornes
n'a de sens qu'une fois l'arme portee connue, et elle ne l'est jamais. **Reporte, condition de
reprise ecrite au §6.**

### P.0.4 — seconde source de visee (i17) : `[!]` NON MESURABLE

Seuil ecrit : \|delta cap\| <= 5° sur **>= 90 %** des paires a <= 100 ms.

| Film | pont ti=5 -> bipede | lectures i17 | paires a <= 100 ms | points sans cap de corps |
|---|---|---|---|---|
| `000d5950` | 4 entites (4/4 fenetres appariees) | 12 | **0** | 81 491 / 171 842 (47,4 %) |
| `530820e5` | 3 entites (3/3) | 3 | **0** | 71 824 / 148 907 (48,2 %) |
| `7344d24f` | 1 entite (1/1) | 5 | **0** | 75 926 / 188 997 (40,2 %) |

Deux verrous cumules, chacun suffisant : **(a)** i17 rend 1 a 27 lectures par film ; **(b)** le
pont ti=5 -> bipede ne couvre que 1 a 4 entites joueur sur 8, parce qu'il se construit sur les
fenetres ACTIVES du respawn (1 a 12 par film, cf. B.0.4). **Couverture ajoutee sur les points
sans cap : 0,00 %** — contre 40-48 % de points qui en auraient besoin.

Le pont lui-meme est propre la ou il existe (4/4, 3/3, 1/1 fenetres appariees a +/- 5 s d'une
fin de vie), mais un pont de 3 coincidences n'est pas un pont, et il est publie comme tel.

### P.0.5 — etats du joueur : `[!]` AUCUN ETAT TENU

| Canal | Seuil ecrit | Lectures certaines (22 films) | Ce que les valeurs disent | Verdict |
|---|---|---|---|---|
| i2 hors-limites | fenetres datees vs morts neutres | 282 (~13/film) | 9 lectures / 9 valeurs distinctes sur `000d5950` | **NON TENU** (denominateur) |
| i3 repere | distribution, transitions | 336 (~15/film) | **4 valeurs sur 2 bits**, distribution `0,0`(20) `1,0`(8) `0,1`(3) `1,1`(2) — **non uniforme, donc reelle** | **NON TENU** (denominateur), mais canal REEL |
| i12 choix de reapparition | distance <= 3 m sur >= 80 % des vies | 115 (~5/film), **dont 6 sur 7 porte FERMEE** sur `000d5950` | 1 seule valeur exploitable par film | **NON TENU** |
| i14 vies | modes a vies | 206 (~9/film) | 15 valeurs distinctes sur 16 lectures (aucun mode a vies au corpus) | **NON TENU** |
| i15 trahison | accord >= 90 % avec les kills amis | 109 (~5/film) | 7 valeurs distinctes sur 8 lectures | **NON TENU** |
| i18 / i19 arrivees-departs | +/- 5 s de la premiere/derniere trajectoire | 163 / 105 | **2 valeurs (0/1)**, distribution 6/2 et 4/1 | **NON TENU** (denominateur) |
| i20 modificateurs | valeurs distinctes | **9 026** (~410/film) | **10 valeurs distinctes ; les 2 premieres font 248/256 = 96,9 %** — deux tuples de 24 entrees, structures et repetes | **CANAL REEL, non exploite** |

**Le seul canal de ti=5 qui porte un signal massif et propre est i20** (`malleable-properties`,
3 x R(1) + 6 x [R(1) -> R(12)] + 9 x R(1)). Sa forme mesuree sur `000d5950` :
`1,1,1,1,478,1,797,1,1434,1,1106,1,1106,0,...` — six valeurs de 12 bits sous porte, quasi
constantes sur toute la partie, deux variantes dominantes. **C'est une table de modificateurs de
joueur, pas un etat vivant** : elle ne repond a aucune des questions de P.0.5, mais elle est le
seul gisement dense de l'entite joueur et elle est ecrite ici pour qui voudra la reprendre.

### Gate 0 lot P : **NON ATTEINT**

Clause par clause : P.0.3 >= 90 % -> **non mesurable (0 tir couvert sur 5 219)** ; P.0.2 tenu ->
**non (5,1-8,1 % de couverture)** ; deux etats de P.0.5 tenus -> **zero**. Negatif global ecrit,
**lot P clos `[!]`** pour la phase 0.

---

## 4. Cout machine (D17)

| Instrument | Executions | Duree | Pic memoire | Plafond |
|---|---|---|---|---|
| `GAME_FILM` (2 voies + chasse) | 27 | 872 s au total, **mediane 32 s/film** (min 9 s, max 91 s) | **31 Mo max** | 3 072 Mo, jamais approche |
| `PLAYER_FILM` (positions + tirs + chaine) | 5 | 315 s, **moyenne 63 s/film** | **36 Mo max** | idem |
| `CLOCK_HUNT` (chasse par signature) | 1 | +50 s sur le film | 29 Mo | mise sous garde d'environnement apres mesure |

Regles tenues : **un film par processus** (la boucle est dans `run_corpus.ps1`, jamais dans
`go test`), **avant-plan**, **pic surveille par echantillonnage** (`PeakWorkingSet64` toutes les
250 ms, kill au-dela de 3 Go — jamais declenche), **cout mesure sur 3 films avant les autres**,
**une commande `go` a la fois**. Le film `1b1e380f` n'a pas ete touche. Journal :
`lotBP/cout_machine.tsv`.

---

## 5. Corpus mesure (22 films)

Slayer `000d5950` `00162144` `02784ce1` `0215fe6b` · CTF `64e8adfa` `530820e5` `53ce4390` ·
KOTH `0a247154` `01e1f945` `606d9844` `8076f97f` · Strongholds `7344d24f` `696a9d7c` `10ed320d`
`1e26f641` · Oddball `24dbb67d` · BTB `06dfe6d9` · **ajoutes pour B.0.3** (prolongations
flaguees) `aaaf6c76` `eb665a8f` `7ff4271a` `44e14331` `eba1e63f`.

`06dfe6d9` (BTB, 26 joueurs, build de juillet 2025) se comporte comme les autres sur ti=0
(0 record) ; sa seule difference mesuree est la part des 8 slots les plus bavards, qui tombe a
**43,4 %** au lieu de 96,5 % — normal avec 26 joueurs, et coherent avec le routage des hooks
PAR NOM etabli au lot 0.

Sorties versionnees : `lotBP/synthese_films.tsv` (une ligne par film, 22 colonnes),
`lotBP/<short8>_chaine.tsv.gz` (les records CERTAINS, compresses : 1,8 Mo bruts),
`lotBP/<short8>_ti0.tsv`, `lotBP/<short8>_arme_en_main.tsv`,
`lotBP/<short8>.GAME_FILM.log` et `.PLAYER_FILM.log`, `lotBP/cout_machine.tsv`,
`oracle_lotB_overtime.tsv` (+ `oracle_export_lotB.sql`), `LOTBP_gates.log`.

NON versionne, et pourquoi : les vidages `_ti5.tsv` de la voie A (6,1 Mo) sont le melange
signal/bruit que la voie B remplace — les garder inviterait a les relire comme une mesure. Les
oracles du lot A (`oracle_lotA*.tsv`) ont ete LUS depuis `../LevelUp-wt-score-film` et non
recopies ici : ils appartiennent a leur lot, et une seconde copie divergerait.

---

## 6. Reports et decouvertes (hors perimetre — notes, NON traitees)

1. **Munitions en delta (i30/i31/i33/i34, P.0.3)** — non instrumentees. *Condition de reprise* :
   qu'un canal d'arme portee existe ; il n'existe pas par la voie delta (0-4 annonces/film).
2. **4 matchs flagues `IsOvertime` non mesures** — `4a93f0e2`, `521c5c38`, `8b512df2`,
   `19ef6b04` (films presents au cache). Sans objet tant que ti=0 est absent ; nommes pour que
   le denominateur de 10 soit reconstituable.
3. **i20 `malleable-properties`** : 9 026 lectures certaines, 2 tuples dominants a 96,9 %, six
   valeurs de 12 bits sous porte. Seul gisement dense de ti=5, **jamais mesure par personne**.
   Candidat naturel a une sonde (lot F) : Fiesta/mutations, vitesse, degats, sauts.
4. **Motif d'i11 vu par la voie A** : `144,150,194,160,223,248,45,103` x37 et
   `0,255,...,255` x8 sur `000d5950`. Non retrouve dans les lectures certaines. A ne pas
   confondre avec un resultat.
5. **`World.HeldWeapon`** : le canal qui l'alimente (i43-i46) est annonce 0-4 fois par film.
   L'arbitrage garder/supprimer appartient au plan item 4, ce chiffre en main.
6. **Le respawn i1 transmet une CONFIGURATION** (`false/4/8`, `false/5/8`, `false/6/8`,
   `false/7/8` = 92,6 % des lectures), pas un decompte vivant. L'unite des mots T0/T1 reste
   non calibree (`vitality.go:144`).
7. **`ti=8`, `ti=32`, `ti=17`** apparaissent dans la ventilation sequentielle avec des volumes
   non negligeables (2 102, 70, 135 sur `000d5950` apres confirmation) et ne sont inventories
   nulle part.
8. **`WalkKeyframeWorld` et les archetypes absents** : ti=0 n'est dans aucune image-cle. Le
   marcheur est valide a 249/250 entites — l'entite manquante de cette validation n'a jamais ete
   nommee. Non investigue (hors perimetre).

## 7. Ce qui n'a PAS ete fait, et pourquoi

- **Aucune publication, aucun champ du document de rejeu** : c'est le contrat de la phase 0.
- **Aucune ecriture DuckDB** : un seul export en LECTURE SEULE, DB ouverte `READ_ONLY` avec le
  serveur local en cours d'execution (aucun conflit : la lecture seule inter-process est le
  chemin prevu).
- **`traverse.go` et les desers existants : pas touches.** Les hooks du lot 0 ont ete consommes
  tels quels, la couche de capture aussi. Aucune largeur de bit n'a bouge.
- **La calibration de l'unite du respawn (T0/T1)** : hors perimetre de la phase 0.
- **`24dbb67d` borne de manche vs figeage du score de manche 1** : impossible, ti=0 absent.
