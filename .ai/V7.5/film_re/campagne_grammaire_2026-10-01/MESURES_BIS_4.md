# Mesures bis 4 — item 1.3 (dénominateur des entrées de contrôle) et reste du point 25 (2026-10-02)

> Répond à l'item **1.3** du plan (dénominateur EXACT des entrées de contrôle utiles) et au **reste
> du point 25** de `CRITIQUE_COMPLETUDE_1.md` (le juge des invariants sur les A/B de position de
> `MESURES_BIS_2.md` §3). Worktree `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`,
> rien de commité. Aucun fichier de production modifié, `grammar.Rev` inchangé
> (`grammar-2026-09-27.3`). Aucune base, aucune cuisson, aucun backfill. Films en lecture seule : les
> 20 films du corpus (19 témoins de `config/replay_corpus.toml` + `1c4c63c2`). Ghidra en lecture
> seule (`decompile_function`, `read_memory`, `get_xrefs_to`, `search_instructions`) ; extraits dans
> `ghidra_13/`.
>
> Convention : **mesuré** = compté par une sonde sur les films ; **lu** = lu dans le binaire
> (adresse donnée) ; **estimé** = dérivé d'une mesure par une hypothèse écrite. « Sain » = paquet
> fermé qui ne contredit aucun des trois invariants de l'écrivain (juge de `MESURES_BIS_1.md` §0).
> Sain n'est pas juste.

## 0. Résumé

| Point | Verdict | Preuve principale |
|---|---|---|
| 1.3 dénominateur exact | **`[!]` : ne se déduit pas du film.** Une entrée du joueur k existe dans la vue C d'une trame si et seulement si l'ordonnanceur d'AU MOINS UN PAIR qui envoie un paquet dans cette trame tente le candidat (k, kind 0). La décision d'envoyer dépend de l'horloge murale, de la cadence et du débit estimé du pair ; la liste des candidats dépend d'une priorité (position, millisecondes depuis le dernier envoi à ce pair) et de plafonds réglés à l'exécution. Aucune de ces grandeurs n'est dans le film. | §1 (lu, adresses) ; corroboré au §2 (mesuré : 1 124 cas joueur × triplet sain où la vue C du paquet du milieu est entièrement vide alors que le joueur a une entrée juste avant et juste après ; à écart d'horodatage égal, les deux issues) |
| 25 juge sur les A/B de position | **16 A/B jugés sur 20 films**. Gains et pertes SAINS : `flock-position` +398 / −7 (net +391, +3 300 records utiles) ; `tacmap-displayasset` +64 / 0 (HI_1_13_0 seul, aucune perte saine sur aucun film) ; `world-object-i0` +164 / −41 (net +123 ; 490 des 654 gains sont contredits ; trois films en baisse saine) ; `unit-actor-state` +11 / −28 (net −17) ; `ti38-i18` +1 / −8 (net −7 ; 155 des 156 gains contredits) ; `i0-bipede-prechigh` +1 / −2 ; neuf sites à **0 / 0** sain : tous leurs gains sont contredits et toutes leurs pertes retirent une fermeture factice. Les treize ensemble : +672 / −58 (net +614). | §3 (mesuré, contrôles tenus à l'identique de BIS_2) |

## 1. Item 1.3 — quand une entrée est-elle écrite pour un joueur ? (lu)

### 1.1 La chaîne, de l'enregistreur au tick de jeu

Tout est lu dans `HaloInfinite.exe` (base `0x140000000`) ; décompilations dans `ghidra_13/FUN_<adresse>.c`,
vtables et désassemblage manuel dans `ghidra_13/vtables_et_desassemblage.txt`.

1. **L'enregistreur** `FUN_142f2c3b0` (déjà lu par T5) recopie, pour k = 0..31, le tampon PAR JOUEUR
   `t = *(DAT_145178b58 + 0x480 + k*0x4c8)` si `(t+0x20) - 1 < 2` (état 1 ou 2) et si le tampon
   n'est pas vide (`(FUN_14076b9b0() + 7) / 8 > 0`), puis écrit le terminateur.
2. **Remise à zéro par trame.** `FUN_14076bb10` remet les 32 tampons par joueur à l'état 0
   (`*(lVar5 + 0x20) = 0`, boucle `k < 0x20`). Ordre d'une trame, `FUN_140514828` (et `FUN_142d5f008`,
   même ordre) : `FUN_14076bb10` → `FUN_140518b0c` (envoi aux pairs, point 5) → `FUN_140514114`
   (non lu) → `FUN_142f2c3b0` si `FUN_1405f6254() == 4`. Le film ne voit donc que les entrées
   écrites PENDANT la trame.
3. **Le seul écrivain des tampons** est `FUN_14076b0e8` (slot `+0x18` de la vtable de la vue C
   `0x1436a8770` ; ses seules références sont des DONNÉES : `0x1436a8788` et deux entrées de table
   d'exceptions, `get_xrefs_to`). Pour (k, kind 0) : si l'état du tampon de k n'est ni 1 ni 2, il le
   remet à zéro, le passe à 1 et y écrit l'entrée ; puis il recopie le tampon dans le paquet du pair
   et compare sa taille au budget du pair `param_2[9]`. **Le tampon est écrit AVANT le contrôle de
   budget** : une entrée qui ne tient pas dans le paquet du pair (`(int)param_2[9] < taille` →
   recul de l'écrivain, retour nul) reste quand même dans le tampon, donc dans le film. Sur
   succès seulement : effacement du bit k de `vue+0x2550` et `vue+0x2558`
   (`14076b27a` / `14076b284`) et `vue+0x2570 + k*4 = FUN_1405f50b8()` (heure de l'envoi, en ms).
4. **L'appelant de l'écrivain** est l'ordonnanceur des vues `FUN_14076aca4`, slot `+0x28` de la
   vtable du canal de simulation `0x1436a86b8` (construite par `FUN_140b87664`). Il est appelé par
   la construction du paquet d'un pair `FUN_1405167fc` (`140516b81 CALL qword ptr [RAX + 0x28]`,
   RCX = le canal, EDX = numéro de paquet, R8 = l'écrivain de bits). Il :
   - calcule le budget en bits du paquet (`taille*8 - réserve(+0x170) - param_5 - bits déjà écrits`) ;
   - prend la liste de candidats PRÉCONSTRUITE de la trame (`+0x188`) si `FUN_14051a14c()`
     (= réglage nommé `PrebuildOutgoingReplicationRequests`, `DAT_144de4c10`, `FUN_141123e24`,
     ET `DAT_144eadd40`, le drapeau d'enregistrement que teste aussi l'enregistreur), sinon la
     reconstruit (`FUN_14076c008`) ;
   - appelle `vue->vtbl[0x18]` (donc `FUN_14076b0e8` pour la vue C) pour CHAQUE candidat de l'index
     de reprise (`+0x194`) jusqu'au bout de la liste, sans s'arrêter au premier refus ; le premier
     refus devient l'index de reprise.
5. **La liste de candidats** (`FUN_14076c008`) : pour chacune des trois vues, si
   `vtbl[0x08]` (vue C : `0x14076c250`, désassemblé à la main : vrai si `vue+0x19` et l'un des
   masques `0x1f44`, `0x2564`, `0x2550`, `0x2558` est non nul) alors `vtbl[0x10]` (vue C :
   `FUN_1405f4478`) ajoute ses candidats ; tri par priorité décroissante (`FUN_14076c304`, bits
   `0x7fe000`) ; troncature à `0x100` ; si `DAT_14498bdd8 != 0`, plafond
   `DAT_14498bdd8 / *(FUN_140514044() + 0xd0)` (un nombre de pairs, lecture non poursuivie).
   `FUN_1405f4478` crée le candidat (k, kind 0) si le bit k est posé dans `0x1f44 | 0x2550 | 0x2558`,
   avec une priorité tirée de la position de l'unité du joueur k (`FUN_140493720`) passée à
   `FUN_141fd95ec` avec le contexte du pair (non lue : « distance » est une lecture probable, pas
   établie) et du temps écoulé depuis le dernier envoi de k À CE PAIR
   (`FUN_1405f5008(vue+0x2570[k])`, comparé à `DAT_14498bdb0`).
6. **L'envoi par pair** (`FUN_140518b0c`) : en mode film avec préconstruction, la liste de chaque
   pair d'état > 3 est reconstruite à chaque trame (`FUN_142fd11c4` → `FUN_14076c008(canal, 0)`) ;
   puis chaque pair, à tour de rôle (`*(DAT_144e61a00 + 0x908)` décale le premier), passe par
   `FUN_140516fa0`, qui n'envoie QUE si `FUN_1405185b0` le décide (au plus deux paquets par appel).
   `FUN_1405185b0` décide sur une cadence propre au pair `r` (paquets par seconde, tirée de
   `DAT_144de55ec + pair*0x130` ou forcée à -1) : `r < 0` → envoi ; `r = 0` → pas d'envoi ;
   `0 < r <= 30` (`DAT_143cd8394` = 30.0f) → envoi si le temps écoulé depuis le dernier envoi à ce
   pair (horloge en ms lue dans la TLS `+0x2a0`, `+0x18`, divisée par 1000) atteint
   `round(1000 / r) - 1` (`DAT_143cd8348` = 1000.0f) ; `r > 30` → envoi à chaque trame. Quand le
   pair est sous contrôle de débit (son bit dans `DAT_144de7be0`), la taille du paquet vient de
   `FUN_141ff6cc8`, plafonnée à `0x480` octets, et l'envoi est annulé si elle est sous
   `DAT_14498bf0c` (0 dans l'image, posé à l'exécution), sauf cadence forcée (`r = -1`).
7. **Le bit en attente** `vue+0x2550` (bloc `0x68`, le « a » de l'entrée) est posé, pour CHAQUE pair
   (sauf ceux que `FUN_1405f0adc(&DAT_144de3ea0, pair+0x2c)` exclut), par le relais du serveur
   `FUN_141f85a84` → `FUN_14076ac14` à chaque tick de jeu (`FUN_1422d1c5c`, `FUN_1423b456a`, branche
   serveur de `FUN_14059d26c`), pour chaque joueur k du masque de la mise à jour du tick, à condition
   que `FUN_1406d16b8` soit vrai : **le joueur a une unité** (`*(joueur + 0x2e0) != -1`, troisième
   argument nul). La copie du bloc et la pose du bit sont INCONDITIONNELLES : aucune comparaison avec
   l'état précédent. Le masque de la mise à jour (`FUN_1404975a0`) a le bit k si
   `FUN_1406b09c0` rend vrai : entrée fraîche reçue, ou dernière entrée MAINTENUE tant que son âge
   est sous `DAT_144989f60` (posé à l'exécution, `FUN_140a9d678`), et valide (`FUN_1406d4db8` :
   flottants finis et bornés, pas un test « entrée non neutre »).
   Le bit `vue+0x2558` (bloc `0xbc`, le « b ») n'est posé que par le chemin du joueur LOCAL
   (`FUN_14076a2f4` → `FUN_14076d11c`) et par l'autre mode (`FUN_142f2a40c`, `FUN_142f2abb8`,
   `FUN_14048ee34()` vrai) ; le relais du serveur ne le pose jamais.

### 1.2 La condition, écrite

Le joueur k a une entrée dans la vue C du paquet de la trame f **si et seulement si**, pendant f, au
moins un pair p :

- (a) **envoie** au moins un paquet (`FUN_1405185b0` vrai : horloge murale, cadence du pair,
  débit estimé, taille minimale) ;
- (b) a (k, kind 0) dans sa liste de candidats à partir de l'index de reprise, après tri par
  priorité et plafond (`0x100`, `DAT_14498bdd8`) ;
- (c) a le bit k en attente : posé à chaque tick où k a un contrôle valide (frais ou maintenu) ET
  une unité, effacé seulement par une écriture de k vers p qui a tenu dans le budget de p.

Seul (c) est une propriété du jeu (un joueur vivant, au sens « a une unité ») ; il est déductible du
film en principe (non fait ici). **(a) et (b) dépendent de l'horloge murale, du débit estimé, du
nombre de pairs et des réglages chargés à l'exécution** : `DAT_14498bdb0`, `DAT_14498bdb4`,
`DAT_14498bdac`, `DAT_14498bdd8` valent 0 dans l'image (`read_memory`), reçoivent des défauts dans
`FUN_140a9fd64` (`DAT_14498bdb0 = 1000`) puis sont remplacés par `FUN_140be7310` depuis une
configuration chargée (`FUN_140be60d0` / `FUN_140be7aac`). Aucune de ces grandeurs n'est écrite dans
le film.

**Conséquence pour l'item 1.3** : le nombre d'entrées d'un paquet NON fermé n'est fonction d'aucune
donnée du film ; le dénominateur exact ne se déduit pas. La borne haute par paquet la plus serrée
qu'autorise la lecture est « joueurs ayant une unité » (plus serrée que « sièges »), et la vraie
valeur peut descendre à 0 (trame sans envoi). L'hypothèse « une entrée seulement quand l'état de
contrôle change » est réfutée par la lecture (point 7 : copie et pose sans comparaison).

### 1.3 Statut proposé

**1.3 : `[!]`**, raison prouvée : la présence d'une entrée dépend de (a) et (b) du §1.2, qui sont
des états d'exécution du réseau (horloge murale, cadence et débit par pair, nombre de pairs,
réglages chargés) absents du film. Le dénominateur exact ne se calcule pas ; restent la borne basse
indépendante de BIS_1 §1 (HI_1_13_0 ≥ 45,0 %, corpus ≥ 24,2 %) et, comme borne haute par paquet plus
serrée que « sièges », « joueurs ayant une unité » (§1.2 (c)), non mesurée ici.

## 2. Corroboration sur les films (mesuré)

### 2.1 Sonde et contrôles

Sonde : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/campagne_bis4_controle_research_test.go`
(neuve, tag `research`), `TestCampagneBis4Controle`. Marche de référence des instruments
([`cmMarcher`], contexte [`cmOuvrir`]) avec le juge de la référence ([`cmJuge`]) en tête du diffuseur.
Elle ne regarde que les paquets FERMÉS (vue C lue au bit prouvé). Joué le 2026-10-02, un film à la
fois, sentinelle `filmproc` 4 Gio, pics 74 à 311 Mio, 103 s. Sorties :
`mesures_bis4_tsv/mb4_controle.tsv` (par film) et `mb4_controle_par_build.tsv` (par build et corpus).

Contrôles (mesurés, tenus) : 629 142 paquets delta, 284 704 fermés, 2 301 194 entrées fermées dont
2 301 082 utiles (= carte v2, CARTE §5, BIS_1 §0) ; fermés − sains = 284 704 − 276 316 = **8 388**,
le nombre de fermés contredits de la référence (MESURES_CIBLEES §T2-4) ; le test vérifie film par film
que paquets et fermés égalent ceux du rapport de la marche.

Définitions : un **triplet** = trois paquets delta consécutifs d'un chunk, p−1, p, p+1, tous fermés ;
on regarde chaque index k présent dans p−1 ET p+1 (le joueur a une entrée juste avant et juste
après) et on note s'il est présent dans p. Un triplet est **sain** si ses trois paquets le sont.

### 2.2 Résultats par build (triplets et vues vides : paquets SAINS)

| Build | Paquets delta | Écart au précédent 15-18 ms / 31-35 ms | Fermés / sains | Vue C vide (sains) | Triplets sains : k présent dans p | k absent, vue C de p non vide | k absent, vue C de p vide |
|---|---|---|---|---|---|---|---|
| HI_1_13_0 | 335 960 | 294 790 / 32 442 | 224 818 / 223 331 | 13 439 (6,0 %) | 1 282 748 | 1 036 | 666 |
| HI_1_12_0 | 21 864 | 21 440 / 1 | 5 835 / 5 830 | 716 (12,3 %) | 26 848 | 0 | 0 |
| HI_1_11_0 | 16 824 | 12 / 15 767 | 4 285 / 4 121 | 562 (13,6 %) | 50 943 | 0 | 0 |
| HI_1_10_0 | 129 369 | 70 540 / 47 515 | 28 741 / 22 331 | 5 406 (24,2 %) | 262 519 | 424 | 458 |
| HI_1_9_0 | 17 629 | 4 / 16 328 | 5 739 / 5 613 | 2 027 (36,1 %) | 58 760 | 0 | 0 |
| HI_1_8_0 | 49 696 | 48 801 / 2 | 13 969 / 13 828 | 777 (5,6 %) | 66 945 | 2 | 0 |
| HI_1_4_1 | 11 130 | 8 / 9 747 | 701 / 692 | 692 (100 %) | 0 | 0 | 0 |
| version-33 | 28 751 | 34 / 26 072 | 463 / 427 | 426 (99,8 %) | 0 | 0 | 0 |
| version-31 | 17 919 | 2 / 16 735 | 153 / 143 | 142 (99,3 %) | 0 | 0 | 0 |
| **corpus** | 629 142 | 435 631 / 164 609 (autres 28 216) | 284 704 / 276 316 | 24 187 (8,8 %) | **1 748 763** | **1 462** | **1 124** |

Sur TOUS les fermés (sains ou non) : 1 796 212 / 4 924 / 3 089.

Trous entre deux présences d'un même index dans une suite de paquets fermés consécutifs (tous
fermés) : 8 523 trous, dont 8 013 d'un paquet, 335 de deux, 151 de 3 à 5, 19 de 6 à 29, 4 de 30 à
299, 1 de 300 ou plus ; en durée : 8 361 sous 0,1 s, 157 de 0,1 à 1 s, 4 de 1 à 3 s, 1 au-delà.

Croisement « vue C vide » × écart d'horodatage au paquet précédent (paquets sains, corpus) :

| Écart | Vue C non vide | Vue C vide | Part vide |
|---|---|---|---|
| < 12 ms | 836 | 72 | 7,9 % |
| 12-20 ms | 216 573 | 15 628 | 6,7 % |
| 20-40 ms | 34 023 | 7 975 | 19,0 % |
| ≥ 40 ms | 314 | 486 | 60,8 % |
| premier du chunk | 383 | 26 | 6,4 % |

Les 1 462 absences « vue C non vide » des triplets sains tombent à 1 080 après un écart de 20-40 ms
(HI_1_13_0 : 1 002 sur 1 036) ; les 1 124 absences « vue C vide », 513 après 12-20 ms et 611 après
20-40 ms.

### 2.3 Lecture (estimé, sur la base du §1)

- Dans 99,85 % des triplets sains (1 748 763 sur 1 751 349), un joueur présent juste avant et juste
  après l'est aussi dans p : conforme à (c), le bit étant reposé à chaque tick pour un joueur qui a
  une unité, et contraire à l'hypothèse « entrée seulement au changement d'état ».
- **1 124 cas (joueur × triplet sain) où la vue C de p est entièrement vide** alors que ce joueur a
  une entrée en p−1 et en p+1 : une trame où AUCUN tampon par joueur n'a été écrit, alors qu'à
  ±1/60 s ces joueurs avaient une unité. C'est la signature de (a) (aucun pair n'a envoyé dans la
  trame) ou d'une trame sans tick qui reposerait les bits ; ni l'un ni l'autre ne se lit dans le film.
- L'écart d'horodatage n'est PAS la règle cachée : à écart égal (12-20 ms), 15 628 paquets sains ont
  une vue C vide et 216 573 non. Il est associé (un écart de deux trames multiplie par 2,8 la part de
  vues vides, 19,0 % contre 6,7 %, et porte 74 % des absences isolées, 1 080 sur 1 462), sans décider.
- **1 462 absences isolées** (cas joueur × triplet sain) dans une vue C non vide : un joueur absent
  pendant une seule trame alors que d'autres sont écrits. Aucun chemin lu ne retire l'unité d'un
  joueur pendant 1/60 s ; c'est la forme attendue de (b) (candidat coupé par priorité / plafond, ou
  bit déjà effacé pour les pairs qui envoient). Attribution non prouvée.
- Les trous longs (≥ 1 s : 5 sur 8 523) sont presque absents : les morts n'apparaissent pas comme
  des trous à l'intérieur d'une suite de paquets fermés (hypothèse : une mort casse la suite fermée).
- HI_1_4_1, version-31 et version-33 ne ferment presque que des vues C vides (99,3 à 100 % des
  fermés sains) : sur ces builds, la vue C non vide ne ferme pas (cohérent avec BIS_1 §1, ≈ 0 entrée
  utile).

## 3. Reste du point 25 — le juge sur les A/B de position (mesuré)

### 3.1 Sonde, méthode et contrôles

Sonde : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/campagne_bis4_juge_positions_research_test.go`
(neuve, tags `research && campagne_overlay`), `TestCampagneBis4JugePositions`. Elle reprend les
variantes ([`b2pVariantes`]) et la marche ([`b2pMesure`]) de `campagne_bis2_positions_research_test.go`,
contexte des instruments, sous la surcouche `mesures_bis2_overlay/overlay.json`. Par film : la
référence est marchée sous son juge (qui note, paquet par paquet, si sa fermeture contredit un
invariant) ; chaque variante sous un juge neuf qui hérite de ces notes (règle de
`campagne_bis1_research_test.go`). « Gagné sain » = paquet gagné qui ne contredit aucun invariant ;
« perdu sain » = paquet perdu dont la fermeture de référence n'en contredisait aucun ; une perte
« réf. contredite » retire une fermeture factice (D2).

Les « 17 A/B » sont les 17 marches par film de BIS_2 §1.3 : la référence et 16 variantes (13 sites
seuls, `tacmap-waypointstate` en position seule et en queue seule, et « tous »). Les A/B « lecture par
index de plage » de BIS_2 §3.3 (contexte de production, Live Fire) ne sont PAS rejugés ici ; leurs
gains contredits y sont déjà publiés (3 sur 3 269 et 4 sur 1 806).

Joué le 2026-10-02, 20 films, un à la fois, sentinelle 4 Gio, pics 104 à 345 Mio, 1 486 s.
Sorties : `mesures_bis4_tsv/mb4_juge_positions.tsv` (film × variante) et
`mb4_juge_positions_par_build.tsv` (variante × build, et corpus).

Contrôles (mesurés, tenus) :
- fermés, gagnés et perdus de chaque (film, variante) identiques à `mesures_bis2_tsv/mb2_positions.tsv`
  sur les 340 lignes (`diff` vide) ;
- référence : 284 704 fermés dont 8 388 contredits (= MESURES_CIBLEES §T2-4) ;
- pour chaque ligne : gagnés du juge = gagnés du comparateur, perdus idem, gagnés = sains +
  contredits, perdus = sains + réf. contredits (colonne `ecart_controle` à 0 sur les 340 lignes).

### 3.2 Table par site (corpus, 20 films)

| Site (lecture du jeu) | Gagnés (sains / contredits) | Perdus (sains / réf. contredits) | **Net sain** | Utiles sains (+ / −) | Conservés sain→contredit / contredit→sain | Par build, net sain (gagnés sains / perdus sains) |
|---|---|---|---|---|---|---|
| `flock-position` | 416 (398 / 18) | 10 (7 / 3) | **+391** | +3 300 / −6 | 1 / 6 | HI_1_11_0 +292 (292/0), HI_1_4_1 +64 (64/0), HI_1_12_0 +24 (24/0), version-33 +10 (10/0), HI_1_8_0 +1 (8/7) ; HI_1_13_0, HI_1_10_0, HI_1_9_0 : 0 |
| `tacmap-displayasset` | 77 (64 / 13) | 16 (0 / 16) | **+64** | +1 855 / 0 | 1 / 1 | HI_1_13_0 +64 (64/0) ; les autres builds 0 |
| `world-object-i0` | 654 (164 / 490) | 623 (41 / 582) | **+123** | +3 214 / −243 | 78 / 74 | HI_1_9_0 +163 (163/0) ; HI_1_8_0 −30 (0/30), HI_1_11_0 −6 (0/6), HI_1_10_0 −4 (1/5) ; HI_1_13_0 0 |
| `tacmap-areaofinterest` | 12 (0 / 12) | 2 (0 / 2) | 0 | 0 / 0 | 2 / 1 | aucun |
| `respawn-location` | 10 (0 / 10) | 9 (0 / 9) | 0 | 0 / 0 | 0 / 0 | aucun |
| `i0-bipede-prechigh` | 2 (1 / 1) | 3 (2 / 1) | **−1** | +1 / −10 | 0 / 2 | HI_1_13_0 0 (1/1), HI_1_10_0 −1 (0/1) |
| `tacmap-cooptetherarea` | 0 | 1 (0 / 1) | 0 | 0 / 0 | 0 / 0 | aucun |
| `crew-order` | 3 (0 / 3) | 5 (0 / 5) | 0 | 0 / 0 | 0 / 0 | aucun |
| `tacmap-waypointstate` (position + queue) | 0 | 2 (0 / 2) | 0 | 0 / 0 | 0 / 0 | aucun |
| `waypoint-position-seule` | 2 (0 / 2) | 2 (0 / 2) | 0 | 0 / 0 | 2 / 0 | aucun |
| `waypoint-queue-seule` | 1 (0 / 1) | 2 (0 / 2) | 0 | 0 / 0 | 0 / 0 | aucun |
| `tacmap-poiicon` | 3 (0 / 3) | 6 (0 / 6) | 0 | 0 / 0 | 1 / 2 | aucun |
| `flock-destination` | 12 (0 / 12) | 16 (0 / 16) | 0 | 0 / 0 | 1 / 2 | aucun |
| `ti38-i18` | 156 (1 / 155) | 170 (8 / 162) | **−7** | 0 / 0 | 21 / 22 | HI_1_10_0 −7 (1/8) |
| `unit-actor-state` | 18 (11 / 7) | 33 (28 / 5) | **−17** | +11 / −219 | 8 / 0 | HI_1_10_0 −13 (10/23), HI_1_13_0 −4 (1/5) |
| **tous** (13 sites + queue) | 1 315 (672 / 643) | 802 (58 / 744) | **+614** | +8 410 / −440 | 100 / 102 | HI_1_11_0 +290 (296/6), HI_1_9_0 +164 (164/0), HI_1_13_0 +70 (77/7), HI_1_4_1 +68 (68/0), HI_1_12_0 +32 (32/0), version-33 +13 (13/0), HI_1_8_0 +1 (8/7), **HI_1_10_0 −24 (14/38)** ; version-31 0 |

« Aucun » = ni gain ni perte sains sur aucun build. La matrice complète site × build est dans
`mb4_juge_positions_par_build.tsv`.

### 3.3 Films en perte SAINE (critère du gate 2, par film)

| Site | Film (build) | Paquets sains + / − (net) | Records utiles sains + / − (net) |
|---|---|---|---|
| `flock-position` | `60ae07c4` (HI_1_8_0) | +8 / −7 (+1) | 0 / −6 (−6) |
| `i0-bipede-prechigh` | `0797ce72` (HI_1_13_0) | 0 / −1 (−1) | 0 / −9 (−9) |
| | `084a804d` (HI_1_10_0) | 0 / −1 (−1) | 0 / −1 (−1) |
| `ti38-i18` | `1c4c63c2` (HI_1_10_0) | +1 / −8 (−7) | 0 / 0 |
| `unit-actor-state` | `1c4c63c2` (HI_1_10_0) | +4 / −23 (−19) | +4 / −111 (−107) |
| | `4f77afc1` (HI_1_13_0) | 0 / −5 (−5) | 0 / −108 (−108) |
| `world-object-i0` | `1c4c63c2` (HI_1_10_0) | +1 / −5 (−4) | 0 / −3 (−3) |
| | `60ae07c4` (HI_1_8_0) | 0 / −30 (−30) | 0 / −40 (−40) |
| | `e5adf7b2` (HI_1_11_0) | 0 / −6 (−6) | 0 / −200 (−200) |
| tous | `0797ce72` (HI_1_13_0) | 0 / −1 (−1) | 0 / −9 (−9) |
| | `084a804d` (HI_1_10_0) | +5 / −1 (+4) | +5 / −1 (+4) |
| | `1c4c63c2` (HI_1_10_0) | +8 / −37 (−29) | +4 / −115 (−111) |
| | `4f77afc1` (HI_1_13_0) | +65 / −6 (+59) | +1 856 / −109 (+1 747) |
| | `60ae07c4` (HI_1_8_0) | +8 / −7 (+1) | 0 / −6 (−6) |
| | `e5adf7b2` (HI_1_11_0) | +296 / −6 (+290) | +3 076 / −200 (+2 876) |

Aucun autre (site, film) n'a de perte saine.

### 3.4 Ce que le juge change aux verdicts de BIS_2 §3.4

- **`tacmap-displayasset`** : le seul site sans AUCUNE perte saine sur aucun film, avec un gain sain
  (+64 paquets, +1 855 records utiles, tous sur HI_1_13_0). Ses 16 pertes, dont les −10 de
  `51ebbc0f` déjà connues (R3), retirent toutes des fermetures factices.
- **`flock-position`** : +398 sains (96 % de ses gains), mais une baisse saine en records utiles sur
  `60ae07c4` (−6) ; ses 4 gains sur HI_1_13_0 sont tous contredits.
- **Neuf sites sont neutres une fois jugés** (`tacmap-areaofinterest`, `respawn-location`,
  `tacmap-cooptetherarea`, `crew-order`, `tacmap-waypointstate` et ses deux moitiés,
  `tacmap-poiicon`, `flock-destination`) : leurs gains bruts sont tous des fermetures contredites,
  leurs pertes toutes des fermetures factices retirées. Les « pertes nettes » que BIS_2 §3.4 leur
  attribuait (`crew-order`, `tacmap-waypointstate`, `tacmap-cooptetherarea`, `flock-destination`,
  `tacmap-poiicon`) ne sont pas des pertes saines.
- **`world-object-i0`** : net sain +123 (contre +31 brut), porté par `11de8353` (HI_1_9_0, +163) ;
  75 % de ses gains bruts sont contredits ; trois films en baisse saine (`60ae07c4` −30, `e5adf7b2`
  −6, `1c4c63c2` −4). Son gain de +12 338 records d'image-clé n'est pas jugé (les invariants portent
  sur les paquets delta).
- **`ti38-i18`** : 155 de ses 156 gains sont contredits ; net sain −7, tout sur `1c4c63c2`.
- **`unit-actor-state`** et **`i0-bipede-prechigh`** restent en perte nette SAINE (−17 et −1), en
  paquets et en records utiles.
- Les treize ensemble : net sain +614 (contre +513 brut), mais HI_1_10_0 en baisse saine (−24, dont
  `1c4c63c2` −29).

Critère de retrait écrit (« aucune baisse sur aucun film », en paquets sains ET en records utiles
sains, exception D2 : fermeture factice retirée) appliqué site par site : le remplissent
`tacmap-displayasset` (avec gain) et les neuf sites neutres (sans gain). Ne le remplissent pas :
`flock-position` (records utiles de `60ae07c4`), `world-object-i0`, `ti38-i18`,
`unit-actor-state`, `i0-bipede-prechigh`.

## 4. Limites

- §1 : `FUN_140514114` (deuxième moitié de la trame), `FUN_141fd95ec` (priorité), le plafond
  `*(FUN_140514044() + 0xd0)` et le rôle exact de la machine qui enregistre (serveur dédié ou hôte)
  ne sont pas lus. Le verdict n'en dépend pas : (a) suffit à rendre la présence d'une entrée
  dépendante de l'exécution.
- §2 : sonde sur les paquets fermés seulement ; « sain » n'est pas « juste » ; l'attribution des
  1 462 absences isolées à (b) est une estimation.
- §3 : contexte des instruments seulement (comme BIS_2 §3.1) ; pas de profil `killsource` ; la
  surcouche reste celle de la tête `69564ef7d` (copies à re-synchroniser après J12, PLAN §6.0).

## 5. Découvertes (consignées, non traitées)

- D-B4-1 : le relais du serveur (`FUN_141f85a84` → `FUN_14076ac14`) ne pose jamais le bit
  `vue+0x2558` ; seul le chemin du joueur LOCAL (`FUN_14076a2f4` → `FUN_14076d11c`) et l'autre mode le
  posent. Une entrée relayée porte donc `b = 0` : le bloc `0xbc` ne peut venir que d'un joueur local
  de la machine qui enregistre. Cohérent avec BIS_2 §2 (bloc `0xbc` seulement dans des lectures
  désalignées) ; à verser au dossier du correctif C-b de T5 (porter le bloc `0xbc`).
- D-B4-2 : le tampon par joueur est écrit AVANT le contrôle de budget (`FUN_14076b0e8`) : le film
  garde une entrée même quand aucun pair n'avait la place de l'envoyer.
- D-B4-3 : 156 entrées en double dans des vues C fermées (138 sur `1c4c63c2`, 8 sur `084a804d`,
  6 sur `4f77afc1`, 3 sur `111fa685`, 1 sur `e5adf7b2`) : impossibles chez l'écrivain (T5 §6),
  déjà comptées par l'invariant « index non croissant » du juge.
- D-B4-4 : deux cadences d'enregistrement des paquets delta (écart dominant ≈ 16,7 ms sur HI_1_8_0,
  HI_1_10_0, HI_1_12_0, HI_1_13_0 ; ≈ 33 ms sur HI_1_4_1, HI_1_9_0, HI_1_11_0, version-31,
  version-33) ; sur les builds à 60 Hz, une partie des écarts vaut deux trames (HI_1_13_0 : 32 442 sur
  335 960 ; HI_1_10_0 : 47 515 sur 129 369).
- D-B4-5 : sur HI_1_4_1, version-31 et version-33, 99,3 à 100 % des fermés sains ont une vue C vide.

## 6. Gate de cette étape (sorties relevées le 2026-10-02)

Depuis `apps/go-api`, `GOCACHE` dédié (`.../scratchpad/gocache-item13`), `PATH` avec
`C:/msys64/ucrt64/bin`, CGO, une commande `go` à la fois :

- `gofmt -l` sur les deux sondes neuves : sortie vide.
- `go vet -tags=research ./internal/games/halo_infinite/film/internal/grammar/` : aucun message, code 0.
- `go vet -tags=research,campagne_overlay -overlay=<chemin absolu>/mesures_bis2_overlay/overlay.json
  ./internal/games/halo_infinite/film/internal/grammar/` : aucun message, code 0.
- `go test -count=1 ./internal/archlint/` : `ok levelup/go-api/internal/archlint 26.760s`.
- Passes de mesure : `TestCampagneBis4Controle` PASS (103 s) ; `TestCampagneBis4JugePositions`
  PASS (1 486 s).
- `git status` : les seuls fichiers suivis modifiés sont ceux des étapes précédentes (dates du
  2026-10-01 18:42 à 18:47, et `.ai/thought_log.md`, où une entrée est ajoutée en fin de fichier) ;
  `grammar.Rev` = `grammar-2026-09-27.3`. Fichiers neufs : les deux sondes, ce rapport,
  `mesures_bis4_tsv/` (4 TSV), `ghidra_13/` (extraits).
