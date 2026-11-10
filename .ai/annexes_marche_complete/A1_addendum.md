# Addendum A1 : localiser les largeurs fausses (2026-11-10)

Mesures sous `C:\t\marche\A1\` (hors dépôt). Instruments : `libs/film/research/divergence/` (tag `research`, aucun code de production touché).
Corpus : 30 films, 1 051 019 paquets comparés (HI_1_12_0 : 7 films, 222 872 paquets ; HI_1_13_0 : 23 films, 828 147 paquets), dont 3 BTB (4f77afc1, 879a4dba, 8b1f15cb), modes Slayer, CTF, KOTH, Oddball, Bases, « Other ». Liste : `films.txt` (20 films de A1.1), `films_30.txt` (les 30).

## 0. Constat préalable : l'instrument de A0 ne lit pas sous catalogue de carte

L'instrument de A0 (`cmd_fermeture`, `research/divergence`) ouvre le film par `grammar.ContexteDeFilm`, qui est une « enveloppe d'instrument, hors production » (`internal/grammar/film_context.go:490`) : les largeurs de i0 sont lues dans le film et AUCUNE entrée du catalogue de carte n'est posée sur le profil. La cuisson passe par `grammar.NewFilmContextForMap` puis `PoserLaCarteEtLeDecoupage` (`replay/build_from_film.go`, `replay/world_object_precision.go:49`). Sans entrée, les tables de position de la vue A sont vides (`internal/grammar/vue_a_versions.go:107`, « vides sans carte ») et la lecture s'arrête sur les genres 5 et 6.

J'ai donc ajouté un second contexte à l'instrument (`ouvrir.go`, variable `DIVERGENCE_CARTE`, le niveau du film désigne l'entrée du catalogue) et rejoué les 30 films dans les deux contextes. Tout ce qui suit est donné dans le contexte « sous carte » (celui de la production) et, quand il diffère, dans le contexte « sans carte » (celui de A0, qui reste la référence du plan pour C1).

| contexte | HI_1_12_0 fermés | HI_1_13_0 fermés | corpus entier (127 films lus, `fermeture_carte/`) |
|---|---|---|---|
| sans carte (A0) | 96,89 % | 96,61 % (échantillon) | 96,93 % et 97,11 % |
| sous carte (production) | 97,67 % | 97,09 % (échantillon) | 97,70 % et 97,60 % |

Conséquence : C1 « aujourd'hui » vaut 97,6 à 97,7 % en production, pas 97,1 %. Les genres 5 et 6 de la vue A et une grande part de « liste d'évènements non localisée » étaient des artefacts de l'instrument (genre 5 : 208 paquets en 13_0 sans carte, 35 sous carte ; liste non localisée : 1 568 puis 498). Le reste de la liste (position par index de région, ability, type, états de create) ne change pas. Décision à prendre au user : la porte de A2 se mesure-t-elle sous `ContexteDeFilm` (comme A0) ou sous carte (comme la cuisson) ? Je recommande sous carte.

Contrôle de l'instrument (100 paquets tirés parmi ceux que les deux lectures ferment) : 0 désaccord de position sur 24 films sur 30 sous carte ; 6 films avec 1 à 3 désaccords sur 100. Ce ne sont pas des erreurs de convention de bit : ce sont les « fermetures fausses » du §2 (paquets fermés chez nous, lus de travers, 0,14 % des paquets dans chaque build sous carte), le test les nomme « désaccord » parce que sa porte exige zéro.

## 1. A1.2 : classement des premiers désaccords non rattrapés

Paquets non fermés chez nous, rangés par lecteur en cause (premier désaccord qui ne se rattrape pas). Part = part des paquets du build. Sous carte.

### HI_1_13_0 (828 147 paquets, 804 066 fermés = 97,09 %, 24 081 non fermés)

| paquets | part | lecteur en cause |
|---|---|---|
| 12 087 | 1,46 % | B : `ti=40` composant 0 (update) |
| 6 404 | 0,77 % | B : sortie par rejet (hors datum) |
| 1 799 | 0,22 % | B : type de l'entité (update) |
| 812 | 0,10 % | B : `ti=35` composant 57 (update) |
| 772 | 0,09 % | aucun désaccord de position (le témoin ne ferme pas non plus) |
| 498 | 0,06 % | A : liste d'évènements non localisée |
| 317 | 0,04 % | le témoin s'arrête sur Component(35, 0) (il ne ferme pas non plus) |
| 305+186+67 | 0,07 % | B : `ti=21` composant 16 (update), trois formes de la même largeur |
| 251 | 0,03 % | B : `ti=38` composant 0 (update) |
| 140 | 0,02 % | B : `ti=41` composant 0 (update) |
| 103 | 0,01 % | le témoin s'arrête ou reprend (arrêt « aucun ») |
| 48 | 0,01 % | B : `ti=21` composant 2 (create) |
| 41+11+34+22+22+22+5 | 0,02 % | états de create : `ti=47`, `ti=13`, `ti=10`, `ti=4`, `ti=22` comp. 0, `ti=3` comp. 0 |
| 35 | 0,00 % | A : largeur du message genre 5 |
| 22 | 0,00 % | B : `ti=37` composant 0 (update) |

### HI_1_12_0 (222 872 paquets, 217 682 fermés = 97,67 %, 5 190 non fermés)

| paquets | part | lecteur en cause |
|---|---|---|
| 4 855 | 2,18 % | B : sortie par rejet (hors datum) |
| 104 | 0,05 % | A : liste d'évènements non localisée |
| 78 | 0,03 % | A : largeur du message genre 82 |
| 34 | 0,02 % | B : `ti=35` composant 57 (update) |
| 30 | 0,01 % | A : largeur du message genre 36 |
| 26 / 21 / 6 / 5 | 0,03 % | A : largeur des genres 0 / 15 / 9 / 21 |
| 8+8 | 0,01 % | `ti=22` composant 0 (create) ; `ti=4` état de create |

Dans le contexte sans carte les tableaux sont dans `porte_tableaux.txt` et `eff_30.tsv` (13_0 : `ti=40` c.0 12 087 ; hors datum 9 125 ; type 1 774 ; liste non localisée 1 568 ; `ti=35` c.57 800 ; genre 5 208, genre 6 58 ; 12_0 : hors datum 6 113, liste 501, genre 82 87, genre 36 43, genre 5 35).

### Fermetures fausses (fermés chez nous, en désaccord avec le témoin)

HI_1_13_0 : 1 181 paquets (0,14 %) ; HI_1_12_0 : 315 (0,14 %). Par lecteur : type de l'entité 684 et 224 ; le témoin s'arrête sur `UnknownEntity` 236 (13_0) ; genre 5 : 53 (13_0), 637 sans carte ; genre 15 : 35 et 13 ; genre 82 : 29 et 27 ; `ti=40` c.0 : 78. Un paquet fermé à tort a toutes ses mesures aval (records, positions) lues de travers alors qu'il compte comme fermé : le taux réel est donc de 0,14 point inférieur à celui des tableaux.

## 2. A1.3 : la liste fermée des lecteurs à corriger

Seuil : 0,1 % des paquets des builds récents (1 051 paquets sur les deux builds ; 828 en 13_0), plus les lecteurs minimaux imposés. « Débloqués » = paquets non fermés dont ce lecteur est le premier désaccord, sous carte, 30 films ; la cascade de rejets « hors datum » est comptée à part (fiche H).

| rang | fiche | lecteur à corriger | paquets débloqués (13_0 + 12_0) | part (13_0 / 12_0) | cascade hors datum estimée |
|---|---|---|---|---|---|
| 1 | P | position quantifiée : largeurs par INDEX de région (BSP) : i0 dynamique `ti=40/41/37`, position d'objet `ti=38/42`, flock `ti=21` i16, positions de la vue A (genres 5, 6) | 13 106 + 3 (sous carte) | 1,58 % / 0,00 % | 26 (13_0) ; 2 202 + 1 651 sans carte (genres 5 et 6) |
| 2 | H | table de datums : entités nées au chunk précédent (lecture de l'image-clé) | 5 954 + 211 (rejets de ce type) | 0,72 % / 0,09 % | sans objet (c'est la cascade elle-même) |
| 3 | V | vue A : genres 82, 0, 15, 9, 36 (HI_1_12_0) et liste non localisée | 505 + 275 | 0,06 % / 0,12 % | 66 (13_0) ; 4 644 (12_0) |
| 4 | T | type de l'entité d'une mise à jour (liaison par slot sans génération) | 1 799 + 2 (+ 684 + 224 fermetures fausses) | 0,22 % / 0,00 % | non estimable |
| 5 | A | `ti=35` composant 57 (`biped-spartan-ability`, tag 3, branche `flags & 1`) | 812 + 34 | 0,10 % / 0,02 % | non estimable |
| 6 | C | états de create : `ti=3` c.0, `ti=21` c.2, `ti=47`, `ti=13`, `ti=10`, `ti=4`, `ti=22` c.0 | 246 + 19 | 0,03 % / 0,01 % | 319 (13_0) |
| 7 | K | vue C : `kind` 1 et 2 | 0 comme premier désaccord | 0 | sans objet (voir fiche K) |
| 8 | W | le témoin ne ferme pas non plus (Component(35,0), arrêt « aucun », sans désaccord) | 1 209 + 2 | 0,15 % | hors de portée du témoin |

Les fiches ci-dessous. Conventions : « nous » = `libs/film/internal/grammar` (chemin abrégé `g/`) ; « témoin » = `C:\t\oracle-filmshell\filmshell-patch\src\` (chemin abrégé `w/`). Les bits sont relatifs à la charge utile du paquet (convention de la sonde).

### Fiche P : largeurs de position par index de région

1. Notre lecteur. `FUN_14076e524` est porté une fois : `g/lecteur_position.go:152` (`lireE524Sur`) : porte R(1) ; si 0, index R(`IndexW`) ; puis trois axes aux largeurs de la ligne (`largeursDeLaLigne`, `lecteur_position.go:137`). L'en-tête du fichier cite le jeu : « index >= 0 -> ligne NIVEAU de la table PAR INDEX `DAT_1445ccbe0 + (index*0x20 + NIVEAU)*0xc` ». Mais `tablesDuProfil` (`:110`) ne porte que la ligne de la région jouée (`axesCarte` de l'entrée de catalogue) : TOUT index lu reçoit les largeurs de la région jouée. Les sites concernés : `consumeAbsoluteWithGate` (`g/components_position_i0.go:226`, i0 de `ti=35/40`), `consumeObjectPositionMonde` (`g/lecteur_position_exceptions.go:53`, `ti=36..43` composant 0), `consumeFlockPosition` (`:82`, `ti=21` i16 : index figé à 1 bit, trois axes à 6 bits), `chargeDetonation` / `chargeImpact` (`g/vue_a_charges_armes.go:32,66` : sous `tablesDeLaRegionJouee`, `transloc_events.go:272`, un autre index que la région jouée ARRÊTE la lecture).
2. Le témoin. `w/components.rs:1409` (`Position`) : 4 bits à zéro, puis `bits(box_bits)` = l'index de BSP, puis les trois champs aux largeurs de CETTE boîte (`box_widths[b]` si le niveau la liste, sinon les largeurs de la région jouée), puis 2 bits. `w/components.rs:1227` (`point`, utilisé par `ObjectPosition` l.1455, `NavpointOffset` l.2070 = `flock-position-component` l.490, et l'ability) : un drapeau (1 : boîte par défaut 3 x 22), sinon index puis largeurs de la boîte. `w/events.rs:361` (`pos_codes`) : même règle pour la vue A (genres 5, 6, 32, 64, 78, 82, 116), les largeurs diminuées de `16 - précision` bits. Les boîtes viennent de `w/game/levels_data.rs` (un catalogue par niveau : région jouée, autres BSP, `index_bits`), p. ex. `sgh_interlock` (l.53) : jouée = index 1 (12, 12, 11), autres = index 0 (12, 12, 13), 2 (13, 13, 13), 3 (19, 19, 16), `index_bits` 2.
3. La différence mesurée (trois occurrences relues dans le film, `loupe.sh`). `sgh_interlock`, i0 de `ti=40` : 59680177 1/112 (bits 1873..), 29a3ef4c 1/184 (7906..), 672641d0 1/134 (1916..) : les dix premiers bits sont `0000 11 0111` : quatre zéros, index = `11` = 3. Témoin : bloc de 62 bits (4 + 2 + 19 + 19 + 16 + 2) ; nous : 43 bits (largeurs de la région jouée 12 + 12 + 11). Écart 19 = (19-12) + (19-12) + (16-11). Constant (-19) sur les 12 087 paquets de `ti=40` sous carte et sur `ti=41` (140) et `ti=37` (22), tous dans les trois films Interlock. Flock `ti=21` i16 : 672641d0 1/176, 29a3ef4c 1/210, 59680177 1/154, bits 64.. = `0 11 011` : drapeau 0, index 3 ; témoin 57 bits (1 + 2 + 54), nous 20 (1 + 1 + 3 x 6) : écart -37 sur 305 + 186 + 67 paquets. `ti=38` composant 0 (7cc59cc5 44/88, fo09_academy, index 1 = 18/18/18 contre 15/15/17 : écart -7, 251 paquets) : témoin 59 bits, nous 52. `ti=42` (sgh_blueprint, écart -13, 4 paquets) : non expliqué, hors seuil.
4. Qui a raison. Le témoin : sur les paquets de cette fiche, `temoin_ferme` vaut « oui » dans 12 087 cas sur 12 087 (`ti=40` c.0) et 305 sur 305 (`ti=21` c.16) ; la lecture du témoin mène le paquet jusqu'à son dernier octet.
5. Correction proposée. Remplacer l'unique ligne de la région jouée par la table par index de région (index, bornes, largeurs) de la carte dans le catalogue (`map_quant_bounds.json`, produit par `cmd/mapquant-build` qui lit aujourd'hui la seule région 0 / jouée), et faire lire `largeursDeLaLigne` la ligne de l'index lu ; lever l'arrêt « autre index » de `tablesDeLaRegionJouee` ; reprendre `consumeFlockPosition` et `consumeObjectPositionMonde` sur `lireE524Sur`. Fonction du jeu à relire : `FUN_14076e524` (table par index `DAT_1445ccbe0`, remplie par `FUN_140be9a14` / `FUN_140be9b88`), déjà citée par le dépôt (`lecteur_position.go`, en-tête) ; le dépôt dit aussi, sur `consumeFlockPosition`, que la lecture « du jeu » a été essayée au lot J6.3 et qu'elle faisait BAISSER la fermeture de `ks_e5adf7b2` : c'est cette lacune de table (index lus aux largeurs de la région jouée) qui l'explique très probablement ; à vérifier au moment de la correction. Je n'ai pas relu le désassemblage (aucune instance Ghidra ouverte) : D-1 reste à tenir en A2. Source des boîtes : le catalogue du témoin (`levels_data.rs`, MIT) comme recoupement, mais la donnée doit être produite par `mapquant-build` depuis le jeu (ADR 0041, rien d'extrait du jeu dans la brique).
6. Débloqués au mieux (premier désaccord, sous carte) : 12 087 + 251 + 140 + 22 (`ti=40/38/41/37` c.0) + 558 (`ti=21` c.16) + 39 (genres 5, 6) = 13 106 en 13_0, 3 en 12_0. Les cartes concernées sont celles à plusieurs BSP : Interlock (index 3), Academy/Wetland/Forge (index 1), Live Fire. Cascade : les genres 5 et 6 (positions de la vue A hors région) sont la cause de 2 202 + 1 651 rejets hors datum sans carte, et de 26 sous carte ; estimation : paquet « débloqué » sur la ligne de la cause, comptage direct (voir fiche H).

### Fiche H : sortie par rejet « hors datum » et sa cascade

Population entière classée (pas d'échantillon : 15 238 rejets en contexte sans carte, 11 259 sous carte, ventilés par la création que le témoin a vue et par les paquets lus entre cette création et le rejet).

| build, sous carte | rejets | (a) création sautée ou paquet mal lu en amont, même chunk | (b) entité née dans un chunk précédent | (c) autre |
|---|---|---|---|---|
| HI_1_13_0 | 6 404 | 450 (7 %) | 5 954 (93 %) | 0 |
| HI_1_12_0 | 4 855 | 4 644 (96 %) | 211 (4 %) | 0 |

(a) : le premier paquet défectueux entre la création et le rejet est : vue A genres/liste (66 en 13_0, 4 644 en 12_0 : genre 82 2 459, liste 793, genre 0 704, genre 15 326, genre 9 310, genre 36 52), états de create `ti=3` c.0 « sortie 1 » (267), `ti=10` (20), 39 sans défaut identifié. Sans carte : (a) 13_0 = 3 213 dont 1 861 genre 5 et 341 genre 6, 1 651 en 12_0 pour le genre 5 (instrument, §0).
(b) : cas type 8b1f15cb : les slots 136 à 143 (type 3, `low-frequency` / `high-frequency`) sont créés au chunk 16 paquet 2188 (que nous fermons sans désaccord) ; au chunk 17 chaque mise à jour du slot 136 est rejetée « hors datum ». `TestDatums` (instrument) rend pour l'image-clé du chunk 17 : table de 555 slots, 307 ambigus, slots 136, 137, 143 ABSENTS (684 et 690 présents, `ti=35`). Notre table de datums d'un chunk vient d'un balayage à tous les décalages de bit des en-têtes d'image-clé puis de la plus longue sous-suite croissante en slot (`g/keyframe_datums.go:85,122`) ; le témoin retrouve les mêmes entrées (`w/snapshot.rs`, garde sur 108 bits) et pose les types du chunk par `seed` (`w/payload.rs:182`, appelé `payload.rs:572`). Concentration : 8b1f15cb 4 472, 4f77afc1 504, d4ebdd5d 455, 672641d0 217, 03b7bbb6 147 (BTB et grands films), types 3, 35 et 41 ; 7 entités de type 41 produisent 1 001 rejets : peu d'entités, beaucoup de paquets.
Correction proposée pour (b) : lire l'image-clé du chunk comme le témoin (garde de 108 bits par entrée, pas de sous-suite croissante) ou garder la table d'un chunk au suivant tant qu'aucun delete ne la retire ; fonctions du jeu : `FUN_142f2e174` / `FUN_142f30610` (dump de l'image-clé), citées par `keyframe_datums.go` ; non relues par moi. Au mieux 5 954 + 211 paquets ; pas de lecteur de paquet delta fautif, c'est une lecture de l'image-clé.

### Fiche V : vue A (genres 82, 0, 15, 9, 36 ; liste non localisée)

Seul le cas `HI_1_12_0` est hors artefact (0,12 %, 275 paquets directs, 4 644 en cascade). Mesure : l'écart de début de vue B est de +48 (genres 82, 0, 36, 9), +64, +84 (genre 15) ; sur 221dd082 4/1944 le témoin lit UN message (genre 75) et ferme la vue, nous en lisons cinq (`[75 105 104 0 82]`) : nous continuons à lire des messages là où le témoin lit le terminateur, donc une largeur trop courte quelque part dans la tête de la vue (le lecteur nommé « genre 82 » est le dernier message que NOUS lisons, pas la cause). Les largeurs du genre 75 comparées champ à champ sont égales (`g/vue_a_charges.go:197` contre `w/events.rs:575`) ; la poignée (`w/events.rs:388`, table `handle_kinds`) est le candidat suivant, non établi. Liste non localisée sous carte : 343 de 498 paquets sur 672641d0 (Interlock) avec un genre 5 en tête : même cause que la fiche P (position hors région jouée).

### Fiche T : type de l'entité d'une mise à jour

1. Nous : `World.ArchetypeForSlot` (`g/world.go:368`) rend le type lié au SLOT ; `rejetDeVue` (`g/frame_infer.go:156`) ne teste que « slot lié ». Les liaisons viennent des NEW lus, de la table de datums de l'image-clé et du repli par anticipation (`keyframe_anticipe.go`).
2. Témoin : `w/payload.rs:182` (`entity_type`) : type de la dernière création vue, remplacé à chaque chunk par l'instantané (`seed`), puis plages d'index, puis inférence `infer_type` (`:343`) départagée par l'instantané suivant.
3. Occurrences : 4f77afc1 19/740 slot 1179 gen 3 (nous 41, témoin 36 ; le slot a été créé en gen 2 au chunk 16 comme projectile 41, supprimé au 16/626, ré-alloué en gen 3 sans création vue) ; b5d130ad 5/2396 slot 1048 (41 / 36) ; 8b1f15cb 8/1816 slot 62 (5 / 19) ; 672641d0 11/508 slot 4 (2 / 6). Tous ont la même position de bit des deux côtés : seul le type diffère.
4. Le témoin ferme 1 773 de ces 1 776 paquets.
5. Correction : lier le type à l'eid COMPLET (slot et génération : la garde du jeu compare `*(uint*)(slot*200 + t) != eid`, citée dans `frame_infer.go` pour `FUN_1406cbaa0`) et délier à la suppression ; confirmer par `FUN_1406cbaa0` / `FUN_1406caad8`, que le dépôt cite. Cause locale précise (suppression non lue, ou naissance en gen 3 portée seulement par l'image-clé) : non établie.
6. 1 799 + 2 débloqués ; effet sur la cascade non estimable ; 908 fermetures fausses (684 + 224) disparaîtraient.

### Fiche A : `ti=35` composant 57, `biped-spartan-ability`, tag 3

1. Nous : `g/components_biped_spartan.go:357` (`consumeBipedSpartanAbility`) et `:413` (`consumeSpartanAbilityTag3`) : `FUN_142f268c4` R(2) tag ; tag 3 = `FUN_142f262d4` : a = R(1) ; si a, R(6) ; t = R(1) ; si t, queue de position (`FUN_14076e494`). Le commentaire (lot 5.13.3) affirme que `FUN_140f03dfc` met `dst[2]` à zéro et que la porte `(dst[2] & 1) == 0` est TOUJOURS vraie : la branche lue sous `flags & 1` serait inatteignable.
2. Témoin : `w/components.rs:2614` : tag 2 bits ; 3 : drapeau ; `flags = bits(6)` ; si `flags & 1` : un drapeau c ; c = 1 : `skip(11 + 2)` et, si `flags & 0x10`, un `place` ; c = 0 : `point` ; puis un drapeau et un second `point`.
3. Trois occurrences (bits 706 de 03b7bbb6 3/2230, 500 de 20fd2c23 4/174, 775 de 8b1f15cb 2/2140) : `111 000001 0...` : tag 3, a = 1, flags = `000001` (bit 0 posé), c = 0. Nous lisons 10 bits (2 + 1 + 6 + t) ; le témoin lit 87, 121 et 109 bits (2 + 1 + 6 + 1 + 2 positions de 1 + index + largeurs de la région jouée + 1 : wetland 1 + 1 + 47 = 49 par position). Écart -77, -111, -99 : constant par carte (-99 sur 447 paquets de 8b1f15cb et sur 25, 19, 14, 13, 11 paquets d autres films de Wetland, Desert et Academy).
4. Le témoin ferme 704 de ces 834 paquets (84 %) et tombe juste sur le composant 59 suivant dans les trois occurrences.
5. Correction : lire `flags` comme valeur du champ testé (la porte `dst[2] & 1` s'applique aux 6 bits lus) et lire la branche `c` puis les deux positions (par la fiche P). À confirmer sur `FUN_142f262d4` et surtout `FUN_14297ea84` : si celle-ci range ses 6 bits dans `dst[2]`, la conclusion du lot 5.13.3 (« dst[2] reste zéro ») tombe. Je n'ai pas relu ces fonctions ; la mesure contredit le commentaire du dépôt, le désassemblage tranche (D-1).
6. 812 + 34 débloqués ; dépend de P pour la largeur des positions.

### Fiche C : états de create (sous le seuil, listés par minimum)

`ti=3` c.0 (`low-frequency`, composant initial d'un create ; sortie 1) : 5 directs, 267 en cascade ; `ti=21` c.2 (create, `flock-destination`) 48 ; `ti=47` 52, `ti=13` 34, `ti=10` 22, `ti=4` 30, `ti=22` c.0 30. Témoin : `w/payload.rs:229` (liste de composants initiaux d'un create, `NO_INITIAL_COMPONENTS` = 7, 13, 14, 20, 23, 28, 29, 46, 47, 49 sans liste). Différence non isolée par fiche (aucun ne passe 0,03 %).

### Fiche K : vue C, `kind` 1 et 2

Premier désaccord de vue C : 0 paquet sur les 1 102 paquets dont la cause est « kind non porté » (13_0, 30 films) ; tous ont un désaccord en amont (vue B : `ti=40` c.0 633, type 257, `ti=21` c.16 103, ...) ou aucun désaccord et le témoin ne ferme pas non plus (50). Côté témoin, les messages `kind` 1 (enregistrements de commande, `w/components.rs:835` et `1091`) sont 0 sur 03b7bbb6 et 29a3ef4c, 6 sur 756 696 messages sur 8b1f15cb ; `kind` 2 ignoré par lui. Porter `kind` 1 et 2 ne débloque donc aucun paquet mesuré : la liste de A2.v (`kind` 1, 2) n'a pas de poids sur C1 ; seule la question de l'écart « bit de référence de 7 bits » reste à arbitrer.

### Fiche W : le témoin ne ferme pas non plus

Component(35, 0) (317 + 127 en A0) : `w/components.rs:1409` ne lit que la forme absolue de i0 (`bits(4) != 0` rend None) ; arrêt « aucun » (103) et « aucun désaccord » (772) : paquets qu'aucune lecture ne ferme ; hors de portée de la comparaison, donc hors de la liste.

## 3. Porte de A1 : 99 % par build récent ?

Calcul sur l'échantillon de 30 films (fermés + paquets expliqués), sous carte. « Prudent » = paquets directs seulement ; « au mieux » = directs + cascade hors datum attribuée par la cause ci-dessus + (b) quand la fiche H est portée.

| build | fermés | + directs de la liste (P, V, T, A, C) | prudent | + cascade (a) attribuée | + (b) fiche H | au mieux |
|---|---|---|---|---|---|---|
| HI_1_13_0 | 97,09 % | 13 106 + 505 + 1 799 + 812 + 246 = 16 468 (1,99 %) | 99,08 % | +0,05 % | +0,72 % | 99,85 % |
| HI_1_12_0 | 97,67 % | 3 + 275 + 34 + 19 + 2 = 333 (0,15 %) | 97,82 % | +2,08 % (vue A) | +0,09 % | 99,99 % |

(13_0 : 13 106 est P, 505 V, 1 799 T, 812 A, 246 C, à quelques formes voisines près.) Résiduel hors liste : 1 209 paquets en 13_0 (0,15 %, le témoin ne ferme pas non plus) et 4 en 12_0.
Sans carte (contexte de A0) : 13_0 96,61 % + 2,14 % = 98,75 % prudent, 99,84 % au mieux ; 12_0 96,89 % + 0,36 % = 97,25 % prudent, 99,90 % au mieux.
Lecture : la porte « 99 % » est atteinte au mieux sur les deux builds ; en prudent, seul HI_1_13_0 y est (99,08 %, marge de 0,08 point) ; HI_1_12_0 dépend de la cascade de la vue A (2,08 %, 4 644 rejets que je n'ai pas pu relier à une largeur précise). Les 0,14 % de fermetures fausses ne sont pas retranchés : le taux « au bit près » réel est donc inférieur d'autant. Echantillon de 30 films : le taux corpus entier sous carte est 97,70 % et 97,60 %, proche de celui de l'échantillon.

## 4. A1.4 : causes d'arrêt déjà typées, builds récents, corpus entier (A0, contexte sans carte)

HI_1_12_0 + HI_1_13_0, 119 161 paquets non fermés (`fermeture_bloquants.tsv`) :

| classe | paquets | part |
|---|---|---|
| vue B : sortie par rejet | 90 425 | 75,9 % |
| vue C : terminateur hors cadre | 12 078 | 10,1 % |
| vue A : liste d'évènements non localisée | 6 048 | 5,1 % |
| vue C : `kind` non porté | 3 400 | 2,9 % |
| vue B : fin de payload | 2 228 | 1,9 % |
| archétype hors registre (ti=36..63) | 2 044 | 1,7 % |
| composant non porté (`non_porte` 1 432, `partiel` 289, `deser_non_cable` 56) | 1 777 | 1,5 % |
| vue C : bloc 0xbc / débordement | 492 + 182 | 0,6 % |
| écrivain (invariants de l'écriture) | 487 | 0,4 % |

Composants non portés les plus fréquents : `ti=0` i18 player-roles 138, `ti=48` i0 103, `ti=0` i22 monitor-noclip 96, `ti=0` i19 total-budget 70, `ti=26` i0 60, `ti=0` i20 45. Message de vue A non porté : aucune cause typée distincte dans la carte (il apparaît comme « liste non localisée »). Lecture : 99 % de ces causes sont des SYMPTÔMES de la lecture en amont (rejet, terminateur hors cadre, `kind`) ; les causes réellement typées (composant non porté, archétype hors registre) pèsent 3,2 % des non fermés, soit 0,1 % des paquets, sous le seuil.

## 5. A1.5 : builds anciens (sans témoin)

Fermeture sans carte (A0) : HI_1_8_0 41,3 % (13 films), HI_1_9_0 32,8 % (4), HI_1_10_0 26,5 % (4), HI_1_11_0 23,3 % (3). Causes d'arrêt (paquets non fermés) :

| build | non fermés | sortie par rejet | liste non localisée | écrivain | vue C (hors cadre + kind + 0xbc) | composant non porté | archétype hors registre |
|---|---|---|---|---|---|---|---|
| HI_1_8_0 | 283 081 | 95,1 % | 4,0 % | 0,2 % | 0,5 % | 0,05 % | 0,04 % |
| HI_1_9_0 | 66 134 | 93,4 % | 5,7 % | 0,2 % | 0,6 % | 0,07 % | 0,06 % |
| HI_1_10_0 | 67 155 | 87,8 % | 10,7 % | 0,6 % | 0,6 % | 0,13 % | 0,08 % |
| HI_1_11_0 | 50 294 | 85,5 % | 13,0 % | 1,0 % | 0,4 % | 0,04 % | 0,04 % |

Ventilation des rejets (`fermeture_rejets.tsv`, état du slot au bloc de type 1 | naissance) : HI_1_8_0 : vide + naissance non lue 80,5 %, trace + naissance non lue 15,1 %, génération 0 1,2 %. HI_1_9_0 : 68,6 / 21,6 / 5,2 %. HI_1_10_0 : 58,7 / 28,4 / 8,4 %. HI_1_11_0 : 60,8 / 31,9 / 5,2 % (+ 0,3 % ré-alloué sous une autre génération). En 13_0 sans carte la même mesure donne « aucune allocation » 59,9 % : les deux étiquettes (naissance non lue, aucune allocation) disent que la création n'a pas été lue, soit ce que la fiche H nomme (a) dans la même chunk. Sur les builds anciens la création non lue est la cause quasi unique (plus de 95 % des rejets) ; je n'ai pas de témoin pour dire quelle largeur la cause.

Registre : le registre de composants (`ti`, index) de tous les lecteurs de la liste est IDENTIQUE du HI_1_8_0 au HI_1_13_0 (49 archétypes avant 12_0, 50 depuis ; relevé par `TestRegistre` sur un film par build) : `ti=35/40` i0 « object-position-dynamic-precision », `ti=36..43` i0 « object-position », `ti=21` i16 « flock-position », `ti=35` i57 « biped-spartan-ability », `ti=40` i47, `ti=41` i18 à i21.

| lecteur | HI_1_8_0 | HI_1_9_0 | HI_1_10_0 | HI_1_11_0 |
|---|---|---|---|---|
| P : position par index de région (i0, objet, flock) | candidat (composants identiques ; sans effet sur les cartes à une seule BSP) | candidat | candidat | candidat |
| P : positions de la vue A (genres 5, 6) | inconnu (table des genres propre au build, `vue_a_versions.go`) | inconnu | inconnu | inconnu |
| H : table de datums par image-clé (b) | candidat (lecture d'image-clé commune) | candidat | candidat | candidat |
| H (a) : création non lue | candidat : cause dominante (plus de 95 % des rejets) | candidat | candidat | candidat |
| V : vue A genres 82, 0, 15, 9, 36 | inconnu (largeurs par build) | inconnu | inconnu | inconnu |
| T : type par slot sans génération | candidat (même `World`) | candidat | candidat | candidat |
| A : `ti=35` i57 tag 3 | candidat (même composant ; la version du lecteur `p.version > 1` du témoin est à lire) | candidat | candidat | candidat |
| C : états de create | inconnu (registre identique, listes de composants initiaux à comparer) | inconnu | inconnu | inconnu |
| K : vue C kind 1, 2 | sans objet (0 paquet de poids) | sans objet | sans objet | sans objet |

« Candidat » = le composant existe au même (ti, index) ; D-4 demande de l'essayer tel quel et de mesurer, ce que le témoin ne permet pas de pré-confirmer sur ces builds.

## 6. Ce que je n'ai pas pu établir

- Le désassemblage (D-1) : aucune instance Ghidra ouverte ; aucune fonction `FUN_...` relue. Les fiches P, H, T, A s'appuient sur ce que le dépôt cite du jeu, sur le témoin et sur les bits relus. La fiche A contredit un commentaire du dépôt (lot 5.13.3) ; la fiche P en précise un autre (`consumeFlockPosition`, lot J6.3 : la baisse de fermeture constatée alors est probablement la même lacune de table).
- Fiche P, genres 5 et 6 : l'explication « position hors région jouée » est vérifiée sur Interlock (liste non localisée de 672641d0 ouverte par un genre 5) mais pas sur trois occurrences de bits pour les genres 5 et 6 ; sans carte ils s'expliquent par l'absence d'entrée de catalogue (instrument).
- Fiche V (HI_1_12_0, genres 82, 0, 15, 9, 36) : le lecteur fautif n'est pas isolé ; seul le symptôme (nous lisons des messages de plus que le témoin, début de vue B +48, +64, +84) l'est. Candidat : table des poignées par genre. 4 644 rejets en cascade dépendent de cette fiche.
- Fiche T : la cause locale (delete non lu, ou naissance portée seulement par l'image-clé) n'est pas isolée. Fiche C : pas de fiche par lecteur (sous 0,03 %).
- Cascade hors datum : estimation « au mieux » = tous les rejets dont le premier paquet défectueux est dans la liste, supposés disparaître ; prudente = 0. Je ne sais pas dire quelle part disparaît réellement (un rejet peut survivre à la correction de sa cause si l'entité a aussi été perdue ailleurs). Aucune des deux ne tient compte du fait que (b), 93 % de la cascade en 13_0, n'est pas un lecteur de paquet delta.
- Fiche H (b) : je n'ai pas vérifié que la table du témoin contient les slots 136 à 143 ; je sais seulement que le témoin leur donne le type 3 et que notre table de datums du chunk 17 ne les contient pas.
- Taux sur 30 films, pas sur le corpus entier : la table de fermeture sous carte du corpus entier est dans `fermeture_carte/fermeture_carte_films.tsv` (127 films lus) ; les 24 films refusés par le témoin n'ont pas de niveau dans `corpus_151.tsv`, donc pas de mesure sous carte pour les builds anciens.
- Le seuil 0,1 % : aucune cause hors liste ne l'atteint, mais `ti=42` c.0 (4 paquets, écart -13, `sgh_blueprint`) n'est pas expliqué par la fiche P.

## Fichiers

`C:\t\marche\A1\` : `films.txt`, `films_30.txt`, `divergence/` (sans carte), `divergence_carte/` (sous carte), `eff_30.tsv`, `eff_carte_30.tsv` (premier désaccord effectif par paquet), `porte_tableaux.txt`, `fermeture_carte/` (fermeture sans carte / sous carte, corpus entier), scripts `pipeline*.sh`, `loupe.sh`, `*.awk`. Dépôt : `libs/film/research/divergence/` (`ouvrir.go`, `bits_research_test.go`, `fermeture_carte_research_test.go`, `film.go`, `rapport.go`, `divergence_research_test.go`).
