# Mesures ciblées — campagne de grammaire, étape 4 (2026-10-01)

> Plan : `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`, étape 4 (item 4.1). Worktree
> `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`, base `feat/v75` = `69564ef7d`,
> rien de commité. Entrées : carte v2 (`CARTE_FERMETURE_V2_2026-10-01.md`, TSV
> `carte_fermeture_v2_2026-10-01/`) et les constats retenus/contestés de l'étape 3 (T1 à T8).
> Sorties brutes de la sonde : `mesures_ciblees_tsv/` (41 TSV, une ligne par film et par clé).
>
> Convention : **mesuré** = compté par la sonde ou la carte v2 sur les 20 films ; **estimé** =
> dérivé d'une mesure par une hypothèse écrite ; **non mesuré** = la mesure exige de toucher un
> lecteur de production ou un portage (interdit à cette étape), la raison est donnée.

## Corrections du 2026-10-01

> Ajoutées après la critique de complétude (`CRITIQUE_COMPLETUDE_1.md`, 39 points) et les trois
> séries de mesures bis (`MESURES_BIS_1.md`, `MESURES_BIS_2.md`, `MESURES_BIS_3_POPULATIONS.md`).
> Le texte des §1 à §5 ci-dessous est conservé tel qu'écrit à l'étape 4 ; là où il diverge de cette
> section, **cette section fait foi**. Verdicts des vérificateurs adverses (étape 3), constat par
> constat : `VERIFICATIONS_ADVERSES.md` (43 constats : 41 retenus, T1-6 contesté, T6-C5 réfuté).

**Chiffres et libellés (mesurés, sauf mention)**

1. **§2 et §T1-3, « −7 751 »** : c'est la perte BRUTE de `tete-bloc` sur le corpus (gagnés +19 105,
   perdus 7 751). Le solde NET du corpus est **+11 354 paquets**. Soldes nets par build
   (`mc_variantes.tsv`) : HI_1_13_0 +12 193, HI_1_12_0 +695, HI_1_9_0 **+73**, HI_1_11_0 −115,
   HI_1_10_0 −486, HI_1_8_0 −1 060, builds anciens +54. La « perte nette » ne vaut donc que pour
   HI_1_8_0, HI_1_10_0 et HI_1_11_0 en paquets ; HI_1_9_0 est net positif en paquets et ne baisse
   qu'en records utiles (26,4 % → 26,2 %).
2. **« 2 836 eid introuvables »** (§T1-1, §C1, §5) : coquille. Le TSV `mc_m1_regions.tsv` et la
   mesure bis 3 donnent **2 856 eid** (15 192 paquets, dont 13 274 hors cadre).
3. **« Aucune allocation »** : 6 455 paquets (compté par eid sur ses paquets, §T1-5 et bis 3) ;
   la CARTE §4 en donne 6 451 (compté rejet par rejet). Les deux sont justes, l'unité diffère.
4. **Table du §T1-5** : les liaisons de l'oracle-NEW somment (i) 1 371 + (ii) 1 910 + (iii') après
   rejet 1 289 + (iii') « ouverte » 89 + (iii) 511 = **5 170** ; les témoins à ≤ 3 paquets en région
   non lue 41 + 95 + 90 + 13 + 0 = **239** (1,4 % de 16 882). Une (iii') « après terminateur »
   (26 eid) n'est pas une région non lue : ni liée ni comptée (`MESURES_BIS_1.md` §6).
5. **Région (iii) contre `ti=3 i0`** (§T1-5, §T7-5) : deux populations différentes.
   - (iii) = eid dont M1 retient un NEW lu désynchronisé à ≤ 3 paquets. Elle compte 511 eid et
     13 978 paquets hors cadre, **tous sur HI_1_13_0** (`fb1a1a72` 340, `51ebbc0f` 167,
     `c75f33b8` 4) et **tous `ti=3 i0`**.
   - `ti=3 i0` = eid précédés d'un NEW `ti=3` désynchronisé, quelle que soit l'occurrence que M1
     retient. Elle compte 1 134 eid et 27 763 paquets, sur les trois mêmes films.
   - La phrase « les autres builds suivent la même hiérarchie » ne vaut que pour (i) et (ii).
   - La cause de (iii) est établie depuis (lue dans Ghidra et mesurée, `MESURES_BIS_3_POPULATIONS.md` §6) :
     `ti=3 i0 low-frequency` n'est pas porté (`FUN_142ed4aec`), et `ti=3 i1 high-frequency` est lu en
     `R(8)` (le composant homonyme de `ti=4`) au lieu de 26 bits (`FUN_142ed4880`). Porter les deux
     ferme **+30 618 / −10 paquets, +234 454 records utiles**. « 412 utiles » sous-estimait donc la
     population : les DELTA `ti=3` lus faux pèsent bien plus que les NEW.
6. **Variante `oracle-bloc+tete-bloc`**, annoncée au §1 et non publiée : +20 809 / −7 535, net
   +13 274 ; HI_1_13_0 +15 595 / −3 055 ; 16,1 % de gains factices ; perte nette sur HI_1_8_0 (−774)
   et HI_1_11_0 (−58) (`MESURES_BIS_1.md` §5).

**Qualifications à revoir**

7. **L'oracle-NEW n'est pas une borne de tout correctif de naissance**, mais celle d'UN mécanisme :
   lier une naissance retrouvée APRÈS son paquet. Sur HI_1_12_0, `tete-bloc` le dépasse sur tous les
   dénominateurs (33,1 % contre 30,7 %, dénominateur fixe maximum).
   - Le pourcentage du §2 est calculé sur les records utiles lus PAR LA MARCHE (dénominateur
     variable). Ce dénominateur monte ou baisse avec la variante : HI_1_10_0 25,9 % (variable)
     contre 22,4 % (fixe maximum) ; HI_1_13_0 86,2 % contre 85,4 %.
   - Lecture honnête, à retenir : dénominateur fixe maximum, numérateur compté en paquets sains
     (`MESURES_BIS_1.md` §7). Sur HI_1_13_0, cela donne 78,1 % pour la référence et 85,6 % sous
     l'oracle (i)+(ii).
8. **La borne de L1 se ventile** (`MESURES_BIS_1.md` §4) : (i) +16 383 nets, (ii) +11 540, (iii')
   après rejet **−522**, (iii') ouverte +19, (iii) **−65**. Lier (iii) et (iii') par l'oracle est
   nuisible (gains factices à 100 % et 40,5 %). La borne utile est (i)+(ii) : **+28 168 nets,
   +360 182 utiles**, supérieure à l'oracle complet (+27 611 / +344 129).
9. **Gains passés aux invariants de l'écrivain** (absent de la version d'origine) : 461 des 29 769
   gains de l'oracle-NEW sont des fermetures factices (1,5 % ; 5,4 % sur HI_1_10_0). Pour
   `tete-bloc`, c'est 17,4 % (70 % sur HI_1_10_0) (`MESURES_BIS_1.md` §3).
10. **§T1-3 « tête prédite pour 97,7 % »** : ce chiffre porte sur la tête `(gen+1)&3` seule, pas sur
    le slot. Le rang dans le pool n'est pas prédit (rang 0 sur HI_1_13_0 : pool 4 à 44 %, pool 0 à
    4/145, pool 2 à 12/41). « Établi sur HI_1_13_0 » vaut pour la tête ; pour le slot, c'est
    **non établi**.
11. **§2 « masque vide au bloc suivant : l'entité est déjà détruite »** : c'est une **interprétation**.
    Aucune mesure ne sépare une entrée libérée d'une entrée vivante sans composant. Ce qui est mesuré :
    5 714 des 6 033 eid « naissance non lue » ont un masque vide au bloc suivant.
12. **Pont masque -> archétype (découverte 5, D-7)** : le `R(6)` trouvé par M1 contredit le pont dans
    144 des 266 cas, et l'oracle-NEW lie avec ce `R(6)`. La fiabilité des liaisons est donc moins bonne
    que les ≈ 4,7 % de liaisons fausses estimés depuis le seul taux du témoin. Ce point n'est pas
    tranché.
13. **Classement de l'instrument, deux erreurs** (portées par le lot L0) :
    - (a) La classe « réalloué sous une autre génération » contient à 99,6 % des naissances de
      génération 0 (`(3+1)&3`), déjà libérées au bloc suivant (`MESURES_BIS_3_POPULATIONS.md` §3.1).
    - (b) Une naissance lue mais désynchronisée est rangée en « naissance non lue » : 3 560 paquets
      sur `81c02726` (`MESURES_BIS_2.md` §7.2).
    - Les chiffres des régions et de l'oracle-NEW dépendent de ces deux classements.

**Constats dont le verdict change (mesures bis)**

14. **T4-C1, T4-C4 à T4-C6 « non mesurés »** : ils sont mesurés depuis, par une surcouche de
    recherche (`go test -overlay`, aucun fichier de production modifié sur disque).
    - Sur le corpus, `flock-position` donne +416 / −10 paquets et `tacmap-displayasset` +77 / −16.
    - `world-object-i0` ferme +12 338 records d'image-clé, mais seulement +31 paquets net, et −364 en
      image-clé sur version-31.
    - `ti38-i18` et `unit-actor-state` sont en perte.
    - Détail : `MESURES_BIS_2.md` §3.1.
15. **T4-C2** : la queue de `waypointstate` ne conserve pas `d9781168` 34:336, sous aucune variante.
    La prédiction de T4 est contredite. Pour `flock-destination`, lire la queue sans condition donne
    le même résultat par construction (niveau 2 partout).
16. **T4-C3 « non concluant »** : il est **établi sur Live Fire**, lu dans le jeu (`FUN_14076e524`)
    et mesuré sous le contexte de production. Lire les largeurs de la plage de l'index lu donne
    +3 269 / −3 paquets sur `0797ce72` et +1 806 / −49 sur `60ae07c4`. Sur les autres cartes, ce
    n'est pas tranché (`MESURES_BIS_2.md` §3.3). D-6 (`0797ce72`) en découle : la lecture par index
    retire 81 % des rejets « aucune allocation » de ce film (`MESURES_BIS_3_POPULATIONS.md` §3.2).
17. **T5-3, « borne estimée ≤ 953 paquets »** : cette borne est **réfutée**. Porté sous ses deux
    formes, le bloc 0xbc ferme 6 et 9 paquets ; les témoins décalés d'un bit en ferment 11 et 10.
    Sur le corpus, le bloc n'est jamais lu au bon bit, et la fourche `+0x74` n'est pas tranchable par
    la fermeture (`MESURES_BIS_2.md` §2).
18. **T6-C1, « compatible avec type 6 = VTOL »** : c'est **établi**. Le type de physique est lu dans
    les tags `vehi` installés : Falcon `0000254b` et Wasp `b65b3b4a` sont de type 6, et toutes les
    familles connues sont cohérentes. Les trois châssis `77ef810a`, `4118381d` et `d0b40d0a` sont
    absents des modules installés et ne sont pas identifiés (`MESURES_BIS_2.md` §4).
19. **T6-C2, « non mesuré »** : c'est mesuré. Sur 7 059 records `ti=40` d'image-clé, 14 sont fermés
    en référence, 26 porte posée, 136 porte levée, 137 porte par châssis. 98 % restent non fermés : une
    autre largeur `ti=40` d'image-clé est fausse et n'est pas identifiée.
20. **T7-5 et T7-6 sont réfutés sur HI_1_13_0.** La grammaire T7 portée en copie de recherche donne :
    - corpus : +18 105 / −12 paquets ;
    - HI_1_13_0 : « hors cadre » de 86 921 à 81 049 ;
    - HI_1_12_0 : de 5 835 à 17 132 paquets fermés ;
    - `81c02726` : +3 580 paquets ; le NEW désynchronise sur `i19`, pas sur `i20`/`i21`/`i22`.

    La mesure « 3 eid » de §T7-5 cherchait des NEW absents des records rendus par la marche. Sur
    HI_1_10_0, 382 des 401 paquets gagnés sont factices (`MESURES_BIS_2.md` §5).
21. **Populations « sans NEW manquant » du §T1-5** (`MESURES_BIS_3_POPULATIONS.md`) :
    - **Image-clé incomplète (87 eid)** : l'en-tête de l'entité est présent dans l'image-clé dans
      87 cas sur 87. La marche du Go la rejette : génération 0 traitée comme un identifiant nul,
      voisin et recalage limités à la génération 1. Oracle : +1 784 / −1 paquets.
    - **Aucune allocation** et **au-delà du plafond** : ce sont des décalages de curseur, pas des
      entités.
    - **Eid introuvables** : ils relèvent à 82,8 % de l'image-clé incomplète.
    - **D-24 (`IDLowBits`)** n'est pas une cause : chaque bloc de type 1 a 8 191 entrées, soit
      13 bits partout.

**Corrections du 2026-10-02 (critique de complétude n° 2, `CRITIQUE_COMPLETUDE_2.md`)**

22. **§T5-1, « 279 / 284 425 »** : mauvais dénominateur. 284 425 = 284 704 − 279 est le nombre de
    fermés NON contredits. Lire **279 / 284 704** (0,1 %).
23. **§T1-3, HI_1_12_0** : « mécanisme confirmé sur HI_1_12_0/HI_1_13_0 » vaut pour la tête sur
    HI_1_13_0 (97,7 %). Sur HI_1_12_0, seul le rang 0 est mesuré (pool 1 à 13/15, pool 4 à 10/18) :
    c'est un **indice faible**, pas un établissement.
24. **Correction 13 (b)** : le « 3 560 » vient de `81c02726`, HORS des 20 films. La part de cette
    erreur dans les 213 033 « naissances attestées » du corpus n'est pas mesurée (PLAN D-63, L0.5).

**Ce qui reste non mesuré** (le §5 « Limites » est à lire avec cette liste) :
- l'oracle d'ordre dans `decodeInferLoop` (T1-2) ;
- la lecture RÉELLE des régions (i) et (ii) : seuls l'oracle et le localisateur `tete-bloc+inv`
  sont mesurés ;
- les gains de portage de `ti=0`/`ti=2` (T7-4) et des composants `ti=40` en delta (T6-C4) ;
- tout effet sur le document publié, car aucune cuisson n'a été faite ;
- la combinaison des leviers L1 + L8 + L2 + L9 sur HI_1_13_0 (recherche R-COMB, PLAN §6.0).

## 1. Protocole

**Sonde** : six fichiers de test `research` dans `film/internal/grammar/` (aucun fichier de
production touché, `grammar.Rev` inchangé, l'empreinte ne hache que les sources non-test) :

| Fichier | Rôle |
|---|---|
| `campagne_marche_research_test.go` | la marche de la carte v2 (même pilotage que `FrameClosureDetaillee`, mêmes lecteurs), qui garde en plus les records de la vue B, les NEW refusés, la chaîne de tête ; deux variantes de mesure (oracle de liaisons, localisateur de tête) |
| `campagne_blocs_research_test.go` | blocs de type 1 par chunk, déclarations des images-clés, pont masque -> archétype (NOTE 5.21 §3.2), prédiction de l'allocateur (pools par plage de slots, curseurs = mots de queue, T1 §2.2) |
| `campagne_naissances_research_test.go` | ventilation des rejets (ordre R1/R2, naissance, cascade), NEW refusés, NEW bordés, allocateur des NEW lus |
| `campagne_m1_research_test.go` | sonde M1 (recherche du NEW manquant, témoin, région, distance) et les deux oracles |
| `campagne_invariants_research_test.go` | invariants de l'écrivain : ordre de la vue B, masques, vue C, bourrage, têtes, véhicules |
| `campagne_mesures_research_test.go` | entrée, cinq marches par film, registres |

Commande (depuis `apps/go-api`, `GOCACHE` dédié, CGO) :
`CAMPAGNE_RACINE=<LevelUp>/data/cache/film_chunks CAMPAGNE_FILMS=<20 ids> CAMPAGNE_SORTIE=<scratchpad>
go test -tags=research -count=1 -timeout 120m -run '^TestCampagneMesuresCiblees$' ./internal/games/halo_infinite/film/internal/grammar/`.
20 films (les 19 témoins + `1c4c63c2`), un à la fois, sentinelle `filmproc` 4 Gio, pics de 79 à
343 Mio, 509 s. Deux passes identiques (la seconde ajoute deux tables) ; `mc_variantes.tsv` est
identique à l'octet entre les deux.

**Contrôle** : la marche « référence » rend, film par film, exactement la carte v2 (paquets
fermés, records utiles fermés/lus, 264 757 hors cadre) — vérifié sur `bfecd02b` (26 403 / 31 232,
226 580 / 254 963, 3 154) et sur les totaux par build.

**Les cinq marches par film** (`mc_variantes.tsv`) :

- `reference` : la carte v2.
- `oracle-NEW` : chaque eid rejeté dont la sonde M1 retrouve l'en-tête NEW dans une région non lue,
  au plus 3 paquets avant le premier rejet, est lié (`BindFull`, `R(6)` trouvé) APRÈS le paquet où
  l'en-tête a été trouvé — comme si son NEW avait été lu. Borne mesurée du correctif T1.
- `oracle-bloc` : chaque « naissance non lue » dont le pont masque -> archétype du bloc suivant
  donne l'archétype est liée avant son premier paquet rejeté (aucune recherche).
- `tete-bloc` : A/B de T1-3 : les candidats NEW de tête (`candidatsDeTete`) sont admis par la
  bande de production OU par l'allocation de leur eid au bloc de type 1 (né dans le chunk, ou
  vivant à son début). Reste identique.
- `oracle-bloc+tete-bloc` : les deux.

Pour chaque variante : paquets gagnés (fermés ici, non fermés en référence) et perdus.

**La sonde M1** : pour chaque PREMIER rejet hors datum d'un eid dans un chunk (17 097 couples
(chunk, eid) sur le corpus, 259 692 paquets rejetés), toutes les occurrences d'un en-tête
`0 01 slot(13) tête(2)` de cet eid suivi d'un `R(6)` < 50 sont relevées dans les paquets delta du
chunk antérieurs au premier rejet, puis rangées en région : (i) tête d'un paquet à événements avant
le début localisé ; (ii) paquet à événements non localisé ; (iii) NEW lu mais désynchronisé ; (iv)
NEW lu mais refusé ; (iii') après l'arrêt de la vue B d'un paquet antérieur (par sortie) ; et les
régions de faux positif (vue A, intérieur d'un record lu). La plus plausible est retenue (région non
lue, traversée propre, la plus proche du rejet). **Témoin** : un eid de même tête sur un slot jamais
alloué ni déclaré (16 882 témoins), même recherche. Le discriminant décisif est la **distance** :

| Distance au premier rejet (région non lue) | eid rejetés | témoins |
|---|---|---|
| paquet précédent | 3 220 | 111 |
| 2-3 paquets avant | 1 950 | 128 |
| 4-10 paquets avant | 1 208 | 311 |
| plus de 10 paquets avant | 5 516 | 7 126 |

À ≤ 3 paquets, le taux de « trouvé » du témoin est **1,4 %** (239 / 16 882) contre **30,2 %**
(5 170 / 17 097) pour les eid rejetés. Faux positifs **estimés** parmi les 5 170 liaisons de
l'oracle-NEW : ≈ 240 (4,7 %). Au-delà de 3 paquets la sonde ne discrimine plus (témoin >
rejeté) : les régions comptées au-delà sont du bruit et ne sont pas utilisées.

## 2. Résultat transverse : ce que la lecture des naissances fermerait

Mesuré, par build (paquets fermés, records utiles fermés / lus) :

| Build | Référence | oracle-NEW | tete-bloc | Gagnés / perdus (oracle-NEW) | Gagnés / perdus (tete-bloc) |
|---|---|---|---|---|---|
| HI_1_13_0 (10 films) | 224 818 (66,9 %) ; utiles 80,6 % | **241 377 (71,8 %) ; utiles 86,2 %** | 237 011 (70,5 %) ; utiles 83,2 % | +16 918 / −359 | +15 248 / −3 055 |
| HI_1_12_0 | 5 835 ; 28,9 % | 6 247 ; 31,0 % | 6 530 ; 33,1 % | +412 / 0 | +695 / 0 |
| HI_1_11_0 | 4 285 ; 27,7 % | 5 239 ; 35,0 % | 4 170 ; 25,8 % | +983 / −29 | +221 / −336 |
| HI_1_10_0 (3) | 28 741 ; 19,4 % | 30 931 ; 25,9 % | 28 255 ; 17,7 % | +3 942 / −1 752 | +2 231 / −2 717 |
| HI_1_9_0 | 5 739 ; 26,4 % | 7 021 ; 36,9 % | 5 812 ; 26,2 % | +1 291 / −9 | +337 / −264 |
| HI_1_8_0 | 13 969 ; 31,0 % | 20 180 ; 45,7 % | 12 909 ; 29,0 % | +6 219 / −8 | +319 / −1 379 |
| HI_1_4_1, version-31, version-33 | 1 317 ; ≈ 0 % | 1 320 ; ≈ 0 % | 1 371 ; ≈ 0 % | +4 / −1 | +54 / 0 |
| **corpus** | **284 704 ; 43,4 %** | **312 315 ; 49,6 %** | 296 058 ; 45,0 % | **+29 769 / −2 158** | +19 105 / −7 751 |

- `oracle-NEW` : 5 170 liaisons font passer les utiles fermés de 2 588 167 à **2 932 296**
  (+344 129) et « hors cadre » de 264 757 à **196 556** paquets ; sur HI_1_13_0, 1 711 liaisons,
  utiles fermés 2 017 679 -> 2 197 646, hors cadre 86 921 -> 58 517. C'est une **borne mesurée**
  du correctif T1 sous trois réserves écrites : environ 4,7 % de liaisons fausses (estimé), les
  naissances retrouvées à plus de 3 paquets ou introuvables ne sont pas liées, et le record NEW
  lui-même n'est pas relu dans son paquet. Le déclencheur (95 % par build) n'est atteint nulle
  part, même sous l'oracle.
- `tete-bloc` : gain net mesuré sur HI_1_13_0 (+12 193 paquets) et HI_1_12_0, **perte nette** sur
  HI_1_10_0, HI_1_11_0 et HI_1_8_0 (utiles fermés en baisse : 19,4 -> 17,7 %, 27,7 -> 25,8 %,
  31,0 -> 29,0 %). Élargir les candidats de tête fait accepter de fausses chaînes sur les vieux
  builds : un filtre plus large n'est pas un gain sans un oracle de plus.
- `oracle-bloc` : seulement 266 liaisons possibles, gain +2 087 paquets. Sur 6 033 eid
  « naissance non lue », **5 714 ont un masque de composants VIDE au bloc suivant** (l'entité est
  déjà détruite) : les naissances non lues sont des entités nées ET mortes dans le chunk, que
  l'image-clé ne verra jamais — c'est pourquoi la table anticipée ne les rattrape pas.
- Paquets perdus par l'oracle-NEW : 2 158, dont 1 752 sur HI_1_10_0 ; c'est la population des
  « paquets fermés après un rejet » (§3, C2) et des fermetures factices (§3, T2-4).

## 3. Constat par constat

### T1-1 — rejet hors datum = naissance antérieure non lue (retenu)

- Mesuré (carte v2) : 252 262 des 264 757 paquets hors cadre (95,3 %) sortent par un rejet hors
  datum ; 0 rejet de vue.
- Mesuré (sonde) : ventilation de plausibilité demandée par la correction 3, en paquets hors cadre
  rejetés : naissance attestée par le bloc suivant 213 033 (84,4 %), eid attesté par un autre bloc
  du film (même tête) 15 898 (6,3 %), indéterminé 16 344 (6,5 %), slot au-delà du plafond du film
  6 987 (2,8 %). Le « cadrage faux » (slot hors plafond) pèse 2,8 % des paquets (5 175 eid), contre
  526 / 632 slots sur `bfecd02b` cités par le vérificateur : beaucoup d'eid, peu de paquets.
- Mesuré (M1, ≤ 3 paquets, région non lue) : le NEW de l'eid est retrouvé pour 5 170 des 17 097
  premiers rejets (30,2 %, témoin 1,4 %) ; ces eid portent la majorité des paquets hors cadre
  (§T1-5). Introuvables : 2 836 eid (15 192 paquets).
- Verdict : **compatible et majoritaire, pas universel**. 4 598 paquets FERMENT au bit près après
  une sortie par rejet (1,8 % des rejets), dont 3 863 sans aucun DELTA lu avant l'en-tête rejeté ;
  3 756 des 4 598 sont sur HI_1_10_0 : soit le « pied de trame » d'une vue vide (lot 5.11.6), soit des fermetures
  factices (cf. T2-4). « Un rejet n'est jamais légitime » n'est donc pas observé tel quel.
- Effet du correctif (mesuré, borne) : oracle-NEW, §2 (+27 611 paquets nets, +344 129 utiles).

### T1-2 — ordre NEW*, DELTA*, DEL*, slots croissants (retenu)

- Mesuré : paquets FERMÉS dont la vue B viole l'ordre : **1 072 / 284 704 (0,38 %)** ; HI_1_13_0
  266 / 224 818 (0,12 %), HI_1_10_0 734 / 28 741 (2,6 %). 975 de ces 1 072 portent AUSSI un masque
  impossible (T2-4) : ce sont des fermetures factices. Violations d'ordre sans autre contradiction :
  97 paquets fermés (0,03 %).
- Mesuré : paquets hors cadre avec violation : 1 759 / 264 757 (0,66 %) — la vue B lue jusqu'au
  rejet respecte l'ordre dans 99,3 % des cas.
- Mesuré : chaînes de tête acceptées par `debutParChaine` : 5 587, dont 45 violent l'ordre (0,8 %,
  dont 32 « DEL dans la chaîne »).
- Verdict : **l'ordre tient sur les paquets proprement fermés** (attendu 0 : 97 résiduels, tous
  builds) ; comme oracle dur il ne détecterait presque rien sur la cause n°1 (la vue B est déjà
  ordonnée jusqu'au rejet). Son gain est de filtrer les candidats de tête : 45 chaînes seulement
  aujourd'hui. Non mesuré : le gain en fermeture d'un oracle d'ordre dans `decodeInferLoop`
  (correctif de production).

### T1-3 — slot et tête d'une naissance prédictibles par l'allocateur (retenu)

- Mesuré (M2, premier NEW propre de chaque pool par chunk, rang 0 ET tête prédite) : HI_1_13_0
  pool 1 222/283 (78 %), pool 3 193/273 (71 %), pool 4 136/306 (44 %), pool 2 12/41, pool 0 4/145 ;
  HI_1_12_0 pool 1 13/15, pool 4 10/18 ; **tous les builds antérieurs ≤ 15 %** (HI_1_10_0 pool 1
  0/74, pool 4 4/147). La table de pools par plage de slots ne vaut que pour HI_1_12_0+.
- Mesuré (eid « naissance non lue ») : tête prédite (`(gen+1)&3`) dans 97,7 % des cas sur
  HI_1_13_0 (1 987 / 2 034) ; mais rang 0 seulement pour 142 eid, rang 1-3 pour 219, rang 4-63 pour
  1 626 : le rang est l'ordre de naissance dans le chunk, il grandit avec la densité ; un « k petit »
  sans suivre les NEW lus ne prédit pas.
- Mesuré (M4, A/B `tete-bloc`, filtre bande OU bloc) : §2 — net positif sur HI_1_13_0 (+12 193)
  et HI_1_12_0, net négatif sur HI_1_8_0 à HI_1_11_0.
- Verdict : **mécanisme confirmé sur HI_1_12_0/HI_1_13_0, réfuté comme règle générale des vieux
  builds** ; l'élargissement du filtre a un coût mesuré (7 751 paquets perdus sur le corpus).

### T1-4 — réutilisation de slot sans DEL ; refus Go (retenu)

- Mesuré (M3) : 1 228 NEW refusés sur 20 films. Verdict de l'image-clé suivante : lecture fausse
  408, **création perdue 7**, indécis 813. Pools : 1 160 en pools 0/4, **68 en pools 1-3** (la
  restriction « pools 0 et 4 seulement » ne tient pas sur le corpus). Prédits par l'allocateur
  (rang < 64) : **5 des 7 créations perdues, 2 des 408 lectures fausses**, 35 des 813 indécis :
  « lier seulement un NEW prédit » séparerait bien les deux populations, sur 7 cas.
- Mesuré : paquets suivants du chunk dont le premier record fautif porte le slot refusé : 137 au
  total.
- Verdict : **effet négligeable** (≤ 137 paquets, contre 264 757 hors cadre) ; le refus protège
  surtout contre des lectures fausses. Correctif de faible priorité.

### T1-5 — où vivent les NEW non lus (retenu)

Mesuré, régions à ≤ 3 paquets du premier rejet (eid rejetés / témoins ; paquets hors cadre et
utiles en jeu des eid rejetés) :

| Région | eid | témoins | paquets hors cadre | utiles en jeu |
|---|---|---|---|---|
| (i) tête d'un paquet à événements | 1 371 | 41 | 62 787 | 825 619 |
| (ii) paquet à événements non localisé | 1 910 | 95 | 66 788 | 926 209 |
| (iii') après l'arrêt par rejet d'un paquet antérieur | 1 289 | 90 | 29 090 | 403 241 |
| (iii') après l'arrêt « ouverte » (désynchronisation) | 89 | 13 | 2 381 | 34 127 |
| (iii) NEW lu désynchronisé | 511 | 0 | 13 978 | 412 |
| (iv) NEW lu refusé | 0 (1 à 4-10 paquets) | 0 | 4 | 0 |

- HI_1_13_0 : (i) 17 366, (ii) 16 072, (iii') rejet 10 129, (iii) 13 978 paquets ; témoin 32 /
  5 296. Les autres builds suivent la même hiérarchie, (i) et (ii) en tête.
- (iii) est presque entièrement `ti=3 i0 low-frequency` (1 134 eid précédés d'un NEW ti=3
  désynchronisé, 27 763 paquets, 719 utiles) : beaucoup de paquets, presque aucun utile.
- **Découverte non tranchée** : 1 289 naissances sont trouvées APRÈS le point de rejet d'un paquet
  antérieur (témoin 90). L'ordre NEW-avant-DELTA de l'écrivain les interdit derrière un DELTA lu :
  soit l'en-tête rejeté de ce paquet n'était pas un DELTA de la vue B (décalage amont), soit le
  groupe NEW n'est pas toujours en tête. À départager (Ghidra ou sonde dédiée).
- Classes sans NEW manquant (correction 4) : vivant au bloc du chunk 14 723 paquets (91 eid ; 87
  eid, 13 644 paquets, **non liés au début du chunk ET non déclarés par l'image-clé** : image-clé
  incomplète, pas une naissance) ; réalloué sous une autre génération 12 906 ; aucune allocation
  6 455 ; non mesurable 4 920.
- Verdict : la prédiction « (i) et (ii) dominent » est **confirmée** ; (iii) pèse en paquets mais
  pas en utiles ; (iii') est une troisième région réelle, non prévue.

### T1-6 — garde sur l'eid complet (contesté)

- Mesuré : DELTA dont le slot est vivant au bloc du chunk et dont la tête diffère de la génération
  du bloc : **1 154 sur 5 964 380 (0,02 %)**, dont 325 dans des paquets hors cadre (104 sur
  HI_1_13_0), 417 dans des paquets fermés ; 1 seul expliqué par une réallocation au bloc suivant.
- Mesuré : conflits d'archétype des images-clés par film 0 à 20 (`1c4c63c2` 20, `111fa685` 14,
  `e5adf7b2` 12) ; têtes déclarées ≠ 1 : `1c4c63c2` 296, `a349fea8` 18, `084a804d` 2, 3 films 1.
- Verdict : **la contestation est confirmée** : borne marginale (≤ 325 paquets hors cadre). Le
  mécanisme existe (têtes variables sur HI_1_10_0) mais ne pèse pas sur la cause n°1.

### T2-1 — le masque est indexé comme l'écrivain (retenu)

- Mesuré (vérification croisée par T2-4) : bits hors archétype sur les DELTA des paquets fermés :
  889 / 3 377 076 (0,026 %).
- Verdict : **confirmé** ; aucun paquet hors cadre expliqué. Effet nul.

### T2-2 — registre = copie de desc+0x200 (retenu)

- Mesuré : idem T2-1 (0,026 % sur les DELTA fermés, tous builds ; HI_1_13_0 89 / 2,67 M).
- Verdict : **confirmé** sur les DELTA ; aucun effet.

### T2-3 — sortie 2 des décalés limitée à ti=24 (retenu)

- Mesuré : aucune entrée `ti=24` de niveau ≠ 1 dans les registres des 20 films (9 builds).
- Verdict : **confirmé** ; la sortie 2 ne peut jouer sur aucun film du corpus. Effet nul.

### T2-4 — un bit de masque hors archétype est un témoin de décadrage (retenu)

- Mesuré (a), DELTA des paquets fermés : 889 bits hors archétype + 28 épars non croissants sur
  3 377 076 (0,027 %).
- Mesuré (b), dernier record DELTA des paquets hors cadre : sortie par **rejet** 1 262 / 249 049
  (0,51 % ; archétypes n ≤ 10 : 2,5 % contre 0,12 % fermés) ; sortie par **terminateur** 723 /
  10 231 (7,1 % ; n ≤ 10 : **27,8 %**). Records antérieurs : 450 / 3,55 M.
- Mesuré : paquets hors cadre sans aucune violation relue : 249 748 / 252 262 sorties par rejet
  (99,0 %) contre 10 726 / 12 495 sorties par terminateur (85,8 %).
- **Mesure qui contredit l'attendu (a) pour les NEW** : NEW des paquets fermés à masque impossible
  8 303 / 23 784 (35 %) — HI_1_10_0 6 308 / 9 770, HI_1_13_0 1 515 / 13 150. Bits au-delà
  aléatoires (`ti=38 n=20 -> 0x800`, `0x884856fcf94`…), paquets surtout d'un seul record
  (« NEW · terminateur · record 1 sur 1 »). Au total **8 388 paquets fermés (2,9 %) contredisent un
  invariant de l'écrivain** (masque 8 150, ordre seul 97, vue C seule 141), dont 6 410 sur HI_1_10_0
  (22 % de ses fermés) et 1 487 sur HI_1_13_0 (0,66 %) ; 4 766 sont des paquets d'un seul record.
- Verdict : l'invariant tient pour les DELTA ; **la sortie par rejet arrive sur une lecture
  alignée** (le désalignement naît au rejet, pas avant), la sortie par terminateur des hors cadre
  porte un désalignement antérieur. `vueCFermee` admet des fermetures factices mesurables : la
  mesure de fermeture de HI_1_10_0 est gonflée.

### T2-5 — niveau de la première homonyme (retenu)

- Mesuré : 0 groupe d'homonymes à niveaux multiples sur les 20 films (25 groupes sur 9 films, 29
  sur 11).
- Verdict : **confirmé** ; effet nul ; le ratchet n'est qu'une garde.

### C1 — une sortie par rejet signale un NEW perdu (R1) ou un décadrage (R2) (retenu)

- Mesuré (ordre, C4) : rejets des paquets hors cadre R1 250 837 (99,4 %), R2 1 425 (0,6 %) ; R1 et
  naissance attestée : 213 015.
- Mesuré (cascade) : 4 704 eid (≥ 11 paquets chacun) portent 237 717 des 252 262 paquets hors
  cadre rejetés (94,2 %) ; HI_1_13_0 : 2 028 eid -> 81 134 / 84 825 (95,6 %) ; `bfecd02b` : 28 eid
  -> 3 122 / 3 145. « Quelques centaines d'eid par film expliquent l'essentiel » : **confirmé**.
- Mesuré (M1) : voir T1-5 ; (a) 1 371, (b) 1 910, (c) 511, (d) 0, (e) 2 836 eid.
- Verdict : **confirmé** sur l'ampleur et la ventilation, avec la réserve T1-1 (4 598 rejets dans
  des paquets fermés).

### C2 — après un rejet, la vue C est lue au mauvais endroit (retenu)

- Mesuré (carte v2) : croisement « rejet hors datum, vue C vide, reste ≥ 64 » 249 931 paquets
  (94,4 % des hors cadre). Prédiction « vue C vide ≥ 99,9 % après rejet » : 96,1 % de vues C vides
  sur l'ensemble des hors cadre.
- Mesuré : arrêts de vue C autres que « hors cadre », par sortie de vue B : kind non porté
  2 856 après terminateur / 2 115 après rejet ; bloc 0xbc 953 / 617 ; débordement 47 / 100. Le
  classement par sortie doit précéder TOUTES les causes de vue C (correction 3) : 2 832 paquets
  sont concernés.
- Effet de la phase 2 « ne pas lire la vue C sur rejet » : **0 paquet fermé en plus** (mesuré par
  construction), et **4 598 paquets fermés perdus** (fermés après un rejet, §T1-1) — à statuer
  avant de l'écrire.
- Verdict : mécanisme confirmé, effet du correctif d'instrument = renommage ; phase 2 dangereuse
  telle quelle.

### C3 — aucune longueur de vue dans le format (retenu)

- Rien à mesurer (constat de lecture). Indirect : `t8_terminateurs` montre que 296 419 vues B
  terminées le sont dans le payload ; la reprise ne peut venir que d'un oracle.

### C4 — oracle d'ordre R1/R2 (retenu)

- Mesuré : R1 99,4 % des rejets hors cadre ; paquets fermés qui violent l'ordre 0,38 % (dont
  91 % aussi contredits par le masque). Composant précédant la première violation : non ventilé
  par composant (les violations des hors cadre portent sur le dernier record dans 61 %).
- Verdict : **l'oracle sépare peu** : R2 = 0,6 % ; la réserve du vérificateur (un en-tête décadré
  donne souvent un slot > dernier DELTA) n'est pas levée, mais la naissance attestée (84,4 %) et
  M1 tranchent mieux que l'ordre.

### C5 — le jeu évince l'occupant ; le Go refuse (retenu)

- Mesuré : identique à T1-4 (1 228 refus, 7 créations perdues, 408 lectures fausses, 813
  indécis). Ordre du NEW refusé : 281 des 408 lectures fausses (69 %) violent l'ordre (NEW après
  DELTA/DEL) ; les 7 créations perdues respectent l'ordre.
- Verdict : **effet négligeable sur la cause n°1** ; l'oracle d'ordre départage les lectures
  fausses dans 69 % des cas seulement : évincer sans condition reste dangereux.

### T4-C1 — sites de position sans grammaire dépendante du build (retenu)

- Non mesuré : l'oracle de stabilité par entité et le retrait des exceptions exigent de modifier
  `lecteur_position_exceptions.go` (code de production, sans bascule).
- Indirect (carte v2) : `object-position` comme dernier composant lu avant un rejet n'est pas dans
  les dix premières lignes de `fermeture_hors_cadre_dernier.tsv`.

### T4-C2 — queue de waypointstate et flock-destination (retenu)

- Mesuré : niveau du registre = **2** pour `ti=21 i2..i11 flock-destination` et `ti=34 i7
  tacmap-waypointstate` sur les 20 films (9 builds).
- Verdict : la « seule dépendance de contenu légitime » (registre 0 ou 1) **n'existe sur aucun
  film**. Le gain du correctif (`d9781168 34:336`) n'est pas mesuré ici (lecteur de production).

### T4-C3 — index de plage != plage cataloguée (retenu)

- Mesuré (`Observation.IndexAbsolus`, contexte des instruments, `Region = 0` sur tous les films) :
  index 1 de 0,05 % (`f75e7053` 65 / 156 596) à 2,7 % (`0797ce72` 4 241 / 154 410) ; Live Fire
  (`0797ce72`, `60ae07c4`) : index 0 dominant (149 947 et 280 256).
- Verdict : **non concluant** : le contexte des instruments ne pose pas la région du catalogue, et
  l'histogramme mélange les sites ; il contredit en apparence « 59 376 / 59 377 i0 d'index 1 sur
  Live Fire » (autre population). À remesurer sous le contexte de production.

### T4-C4, T4-C5, T4-C6 — exceptions de position (retenus)

- Non mesuré : les A/B (retrait de `world-object-i0`, `flock-position`, `ti38-i18`) exigent de
  modifier un lecteur de production. Les gains/pertes de J6.3 restent à remesurer à la tête.

### T5-1 — la vue C d'un film ne porte que des kind 0 (retenu)

- Mesuré (M1) : paquets FERMÉS contredits par la vue C : 279 / 284 425 (0,1 %) — kind 3 152, index
  non croissant 94, en-tête cdc04 posé 31, code 63 2 (premier critère) ; HI_1_10_0 234, HI_1_13_0
  23. 119 de ces paquets sont aussi contredits par le masque.
- Mesuré (M2) : « kind non porté » par sortie de vue B : 2 856 après terminateur, 2 115 après
  rejet (43 % après rejet) ; HI_1_13_0 898 / 437 (67 % après rejet) ; version-31 et version-33 :
  884 et 850 après terminateur.
- Verdict : **attendu « quasi-totalité après un rejet » réfuté** sur le corpus (vrai aux deux tiers
  sur HI_1_13_0). Les kind 1/2/3 restent des marqueurs de désalignement : 1 469 paquets hors cadre
  portent un kind 3.

### T5-2 — terminateur de vue C exact ; a = b = 0 écrivable (retenu)

- Mesuré : entrées a = b = 0 dans des vues C fermées : **112** (sur 2 301 194) ; hors cadre 2 010.
- Verdict : a = b = 0 apparaît dans des paquets fermés (compatible « écrivable ») ; ne doit pas
  servir d'oracle. Effet de fermeture nul.

### T5-3 — grammaire du bloc 0xbc (retenu)

- Mesuré : paquets « bloc 0xbc non porté » 1 570, dont 953 après un terminateur de vue B
  (candidats à fermer après portage) et 617 après un rejet ; HI_1_13_0 148 / 240 ; version-31 548
  après terminateur.
- Non mesuré : M4 (deux formes de +0x74) exige un portage. Borne estimée : ≤ 953 paquets, ≤ 19 378
  utiles en jeu.

### T5-4 — valeurs impossibles de la vue C (retenu)

- Mesuré : code analogique 63 dans des vues C fermées : 13 paquets (2 en premier critère) ; dans
  les hors cadre : 7. En-tête cdc04 posé : 205 fermés, 845 hors cadre.
- Mesuré : hors cadre en trois classes : vide (sans entrée jugeable) 254 306, non contredit 8 168,
  impossible 2 283.
- Verdict : l'oracle « code 63 » ne vaut pas 0 sur les fermés (13), cohérent avec les fermetures
  factices ; il ne départage presque rien (96 % des hors cadre ont une vue C vide).

### T5-5 — grammaire des kind 1/2 (retenu)

- Couvert par T5-1 : 4 971 paquets « autre non fermé » s'arrêtent sur un kind 1/2 ; aucun paquet
  fermé n'en porte (par construction : l'arrêt empêche la fermeture). Effet nul.

### T6-C1 — la porte +0x818 se lit dans MPPWord32 (retenu)

- Mesuré (M1) : DELTA ti=40 qui annoncent i34, par châssis (dernière création lue du slot) :
  `0000254b` (Falcon, `vehicle_families.go`) 14 171 (HI_1_13_0) ; `77ef810a` 457 et `4118381d` 373
  (HI_1_10_0) ; `d0b40d0a` 2 523 (version-31) ; châssis inconnu 15 737. **Aucun châssis terrestre
  connu** (Warthog, Mongoose, Ghost, Scorpion, Wraith, Chopper) n'annonce i33/i34.
- Mesuré : 43 NEW ti=40 annoncent i33/i34 avec des châssis absents de la table (`70aff72a`,
  `baff2205`…) : ce sont des NEW à masque impossible (T2-4), donc des lectures fausses.
- Verdict : **compatible avec « type 6 = VTOL »** pour les châssis identifiés ; trois châssis
  (`77ef810a`, `4118381d`, `d0b40d0a`) restent à identifier et 47 % des annonces ont un châssis
  inconnu (véhicules nés avant le film). Effet de fermeture : nul (requalification).

### T6-C2 — i33/i34/i41/i42 en image-clé (retenu)

- Non mesuré : M2 exige de porter i30-i47 dans la marche d'image-clé.

### T6-C3 — en delta, l'annonce de i33/i34 prouve la porte (retenu)

- Mesuré (M3) : annonces de i30 : 1 DELTA sur `038df01a` (tourelle automatique bannie, qui a une
  tourelle), 3 sur châssis inconnu, 24 NEW sur des châssis non identifiés (lectures fausses
  probables) ; i33/i34 sur châssis non VTOL identifié : **0**. Masque complet : 0 record.
- Verdict : **0 contradiction mesurée** sur les châssis identifiés ; renommage « supposée -> loi »
  sans gain de fermeture.

### T6-C4 — grammaires ti=40 non portées (retenu)

- Mesuré (carte v2, paquets bloqués / utiles en jeu) : i31 808 / 4 827, i38 530 / 11 215, i33 81,
  i37 60, i45 46+8, i47 45, i46 28+3, i30 28, i44 13 ; total ti=40 1 666 paquets, dont HI_1_13_0
  938 (8 108 utiles), HI_1_10_0 262, version-33 146.
- Verdict : borne ≤ 1 666 paquets (0,26 % des paquets), à mesurer après portage.

### T7-1, T7-2 — dispositifs ti=43 (retenus)

- Mesuré (carte v2) : ti=43 bloque 12 939 paquets (80 353 utiles en jeu), dont HI_1_12_0 12 127
  (67 679) ; i35 12 114 paquets dont **12 112 sur `bcb6d393`** ; HI_1_13_0 469 paquets (6 368
  utiles) ; HI_1_8_0 4.
- Verdict : borne mesurée ≤ 12 939 paquets ; concentration sur un film confirmée.

### T7-3 — registres ti=43 / ti=0 / ti=2 identiques entre builds (retenu)

- Mesuré (empreinte noms + niveaux) : ti=2 **identique sur les 20 films** ; ti=0 identique sur 19
  (27 entrées), version-31 diffère (26) ; ti=43 : trois familles — HI_1_10_0 à HI_1_13_0 (41
  entrées, 15 films), **HI_1_8_0 = HI_1_9_0** (41 entrées, empreinte différente), HI_1_4_1 =
  version-31 = version-33 (40 entrées).
- Verdict : la correction du vérificateur est confirmée et précisée : l'identité ti=43 ne vaut que
  de HI_1_10_0 à HI_1_13_0 ; **HI_1_9_0 diffère aussi** (non signalé par le vérificateur).

### T7-4 — moteur ti=0 / ti=2 (retenu)

- Mesuré (carte v2) : ti=2 bloque 4 208 paquets (i15 3 996 ; HI_1_13_0 2 260 paquets ti=2) ; ti=0
  123 paquets dont forge-engine (`ti=0 i18`, `i19`) 10. ti=0 compte 27 entrées (26 sur version-31).
- Verdict : borne ≤ 4 331 paquets, utiles en jeu faibles (ti=2 HI_1_13_0 229).

### T7-5 — NEW ti=43 désynchronisé -> rejets (retenu)

- Mesuré : rejets précédés, dans le chunk, d'un NEW désynchronisé du MÊME eid : ti=43 **3 eid**
  (6 paquets hors cadre au plus) ; l'essentiel est `ti=3 i0` (1 134 eid, 27 763 paquets, 719 utiles)
  et `ti=10 i2` (12 eid, 202 paquets).
- Verdict : **le lien ti=43 -> hors cadre est mesuré et négligeable** ; le mécanisme existe pour ti=3.

### T7-6 — T7 n'est pas le levier de la cause n°1 (retenu)

- Mesuré (carte v2) : causes T7 (ti 0, 2, 43) sur HI_1_13_0 : 2 778 paquets, 6 793 utiles en jeu
  (contre 450 803 utiles hors cadre) ; corpus 17 270 paquets. Lien indirect (T7-5) : 3 eid.
- Verdict : **confirmé**.

### T8-C1 — zéros au-delà du tampon (retenu)

- Rien à mesurer (équivalence de lecture). Effet nul.

### T8-C2 — le moteur déclare en échec un paquet qui déborde (retenu)

- Mesuré : paquets dont une lecture dépasse 8 x taille : **740** (738 par la fin ou un en-tête de
  vue B, 2 par la vue C).
- Verdict : instrument ; aucune fermeture en plus.

### T8-C3 — grammaire de l'écrivain du paquet (retenu)

- Mesuré (carte v2) : reste derrière la vue C des hors cadre : ≥ 64 bits 99,8 %, négatif 0 —
  sous-consommation. Sur les paquets fermés le reste est dans [0, 7] par définition de
  `vueCFermee` (tautologie, pas une mesure de l'écrivain).
- Verdict : le bourrage n'explique aucun hors cadre ; `vueCFermee` est nécessaire, pas suffisant
  (8 388 fermetures contredites, T2-4).

### T8-C4 — records qui finissent dans la queue minimale (retenu)

- Mesuré (queueMin = 4, `HasExtraFields` faux sur les 20 films) : (a) terminateurs de vue B lus
  au-delà du payload 32, sans place pour la vue C 6 ; (b) records débordants 658 (440 DELTA, 218
  NEW) ; (c) records finissant dans la queue minimale 42 ; (d) NEW liés bordés 157, paquets suivants
  du chunk dont le premier record fautif porte leur slot : **24**.
- Verdict : **effet indirect mesuré négligeable** (24 paquets) ; ≈ 740 paquets changeraient de
  cause, 0 de statut.

### T8-C5 — rejet = monde Go divergent (retenu)

- Mesuré : voir T1-1 / T1-5 / §2 : sur HI_1_13_0, les eid dont la naissance est retrouvée à ≤ 3
  paquets portent 58 339 des 84 825 paquets hors cadre rejetés (68,8 %, témoin 0,6 %) ; l'oracle
  qui les lie ferme +16 918 paquets.
- Verdict : **confirmé et chiffré**, avec la réserve des 4 598 paquets fermés après rejet.

## 4. Découvertes (consignées, non traitées)

1. **Fermetures factices** : 8 388 paquets fermés (2,9 %) contredisent un invariant de l'écrivain
   (surtout un NEW à masque impossible dans un paquet d'un seul record) ; 6 410 sur HI_1_10_0 (22 %
   de ses fermés). La part de fermeture de HI_1_10_0 est surestimée.
2. **Naissances après le point de rejet** (T1-5, (iii')) : 1 289 eid (témoin 90) dont le NEW est
   trouvé derrière un rejet d'un paquet antérieur, contre l'ordre NEW-avant-DELTA.
3. **Naissances non lues = entités nées et mortes dans le chunk** : 5 714 des 6 033 eid ont un
   masque de composants vide au bloc suivant.
4. **Image-clé incomplète** : 87 eid vivants au bloc du chunk, ni liés au début du chunk ni
   déclarés par l'image-clé, portent 13 644 paquets hors cadre (découverte 2 de l'étape 1 chiffrée).
5. **Pont masque -> archétype** : sur les 266 naissances où il résout, le `R(6)` trouvé par M1 le
   contredit 144 fois (54 %) ; non tranché (collision de masques de jeunes entités, ou M1).
6. **Allocateur** : la table de pools par plage de slots ne prédit rien sur les builds antérieurs
   à HI_1_12_0 (≤ 15 % au rang 0) : autre table ou `DAT_144706104` à 0 sur ces films.
7. **T7-3** : HI_1_9_0 partage le registre ti=43 de HI_1_8_0, pas celui de HI_1_10_0+.
8. **T4-C3** : `IndexAbsolus` sous le contexte des instruments contredit en apparence la mesure Live
   Fire de `map_bounds.go` : à remesurer sous le contexte de production.

## 5. Limites

- M1 ne relit pas le corps des NEW trouvés au-delà de leur traversée ; la région (i) n'est pas
  séparée de la vue A dans les paquets à événements (le témoin borne le bruit à 41 eid).
- L'oracle-NEW lie l'eid APRÈS le paquet de la naissance : il ne relit pas ce paquet ; il ne fait
  rien pour les 2 836 eid introuvables ni pour ceux trouvés à plus de 3 paquets.
- Les variantes ne mesurent que des fermetures (paquets, records utiles) ; aucune sortie publiée.
- Non mesurés (production ou portage requis) : T1-2 oracle d'ordre dans `decodeInferLoop`, T4-C1,
  T4-C4 à T4-C6, T5-3 M4, T6-C2, gains de portage T6-C4/T7.
