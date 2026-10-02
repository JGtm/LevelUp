# Rapport — Campagne de recherche sur la grammaire du jeu, phase 1 (2026-10-01, révisé le 2026-10-02)

> Plan : `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (étape 5). Worktree
> `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`, base `feat/v75` = `69564ef7d`.
> **Rien n'est commité.** Aucune sortie de production n'a changé : `grammar.Rev` est inchangé
> (`grammar-2026-09-27.3`, empreinte régénérée à révision constante), aucun film cuit, aucune base
> ouverte. Les A/B de composants des mesures bis passent par une surcouche de recherche
> (`go test -overlay`, copies dans `campagne_grammaire_2026-10-01/mesures_bis2_overlay/`) : aucun
> fichier de production n'est modifié sur disque.
>
> **Révision du 2026-10-02.** Ce rapport est corrigé d'après la critique de complétude
> (`campagne_grammaire_2026-10-01/CRITIQUE_COMPLETUDE_1.md`, 39 points) et les mesures bis 1 à 3,
> puis d'après la critique n° 2 (`CRITIQUE_COMPLETUDE_2.md`, points N1 à N21, même jour). La liste
> des corrections, avec l'ancienne valeur, est au §7.
>
> Pièces :
> - la carte v2 : `CARTE_FERMETURE_V2_2026-10-01.md` et les TSV de `carte_fermeture_v2_2026-10-01/` ;
> - les huit notes de recherche `campagne_grammaire_2026-10-01/T1_*.md` à `T8_*.md` ;
> - les verdicts des deux vérificateurs adverses de chaque constat, extraits du journal du workflow :
>   `campagne_grammaire_2026-10-01/VERIFICATIONS_ADVERSES.md` (43 constats : 41 retenus, T1-6
>   contesté, T6-C5 réfuté) ;
> - les mesures ciblées `MESURES_CIBLEES.md`, à lire avec sa section de corrections en tête ;
> - les mesures bis : `MESURES_BIS_1.md`, `MESURES_BIS_2.md`, `MESURES_BIS_3_POPULATIONS.md` ;
> - les recoupements avec J12 : `J12_RECOUPEMENTS_2026-10-01.md`.
>
> Corpus : 20 films, soit les 19 témoins de `config/replay_corpus.toml` plus `1c4c63c2`, sur
> 9 builds. Le témoin utilisateur `81c02726` est mesuré à part. Ghidra : `HaloInfinite.exe` =
> HI_1_13_0 (`269225.26.04.08.1618`), seul exécutable disponible.
>
> Convention :
> - **établi** = lu dans le jeu ET confirmé par une mesure ;
> - **probable** = l'un des deux seulement, avec sa réserve écrite ;
> - **mesuré** = compté sur les films ;
> - **estimé** = dérivé d'une mesure par une hypothèse écrite ;
> - **borne** = gain maximal d'UN mécanisme, mesuré par un oracle ; ce n'est ni un gain acquis, ni
>   la borne de tout correctif ;
> - **sain** = paquet fermé qui ne contredit aucun des trois invariants instrumentés de l'écrivain
>   (ordre, masque, vue C). Sain ne veut pas dire juste.

---

## 0. En bref, pour l'utilisateur

1. **Ce qu'on sait de la cause n°1.** La cause n°1, « vue C : terminateur hors cadre », bloque
   264 757 paquets. Sur HI_1_13_0, elle porte 92,7 % des records utiles non fermés avec la table
   `product_use` corrigée (92,5 % avec l'ancienne table, CARTE §2).
   - Dans 95,3 % des cas, la vue B s'arrête par le **rejet hors datum** d'un DELTA dont l'entité
     n'existe pas dans notre monde.
   - **Établi chez l'écrivain** (lu dans Ghidra et confirmé sur les paquets fermés) : le jeu écrit
     le NEW d'une entité dans un paquet antérieur à son premier DELTA (`FUN_142f2e174`,
     `FUN_142f2f8f0`).
   - **Mesuré** : 84,4 % des paquets hors cadre rejetés visent un eid dont la naissance est attestée
     par le bloc de type 1 du chunk suivant.
   - **Probable seulement** : que tous nos rejets soient des naissances ratées. Trois points restent
     ouverts : la file de records différés `vue+0x1b320` (D-35), les 4 598 paquets fermés au bit
     près après un rejet (D-2), et la région (iii'), contraire à l'ordre de l'écrivain.
2. **Où sont les naissances ratées** (mesuré par la sonde M1, à ≤ 3 paquets du premier rejet,
   contre un témoin : 30,2 % de « trouvé » pour les eid rejetés, 1,4 % pour le témoin) :
   - surtout dans la tête des paquets à événements que le localisateur saute (région (i)) ;
   - et dans les paquets à événements non localisés (région (ii)).

   Ce chiffre a trois limites :
   - la région (i) n'est pas séparée de la vue A dans les paquets à événements ; le témoin borne ce
     bruit à 41 eid ;
   - au-delà de 3 paquets, la recherche ne discrimine plus (6 724 eid, 61 858 paquets hors cadre :
     recherche préalable R-P6) ;
   - 2 856 eid sont introuvables, mais ce sont surtout l'image-clé incomplète et des décalages de
     curseur, pas des naissances.
3. **Les lire fermerait beaucoup, mais pas tout** (oracle limité aux régions (i) et (ii)) :
   - corpus : +28 168 paquets nets, +360 182 records utiles ;
   - HI_1_13_0, sur un dénominateur fixe (§3) : records utiles fermés de **78,4 % à 85,9 %** ;
     dans les paquets sains, de 78,1 % à 85,6 %.

   Lier les régions (iii) et (iii') par l'oracle est **nuisible** : −65 et −522 paquets nets.
   L'oracle n'est la borne que d'un mécanisme : sur HI_1_12_0, la variante `tete-bloc` le dépasse.
4. **Les mesures bis ajoutent quatre leviers mesurés**, tous hors de la borne de L1 :
   - **`ti=3`** (`low-frequency` non porté, et `high-frequency` homonyme mal routé) : +30 618 / −10
     paquets, +234 454 utiles, sur trois films HI_1_13_0. La mesure ferme et le témoin `ti=4`
     s'effondre. **Établi pour `i0`** (`low-frequency`, lecteur lu dans le jeu). **Probable pour
     `i1`** : sa grammaire (26 bits) est lue, mais l'appartenance de sa table `0x143d07af0` à `ti=3`
     est déduite, pas lue (BIS_3 §8, PLAN L8).
   - **`ti=43`**, grammaire T7 : +18 105 / −12 paquets.
     - HI_1_12_0 passe de 5 835 à 17 132 paquets fermés.
     - Sur HI_1_13_0, le « hors cadre » baisse de 5 872 paquets : ce lot agit donc AUSSI sur la
       cause n°1.
     - Le témoin `81c02726` gagne +3 580 paquets.
     - Mais sur HI_1_10_0, 382 des 401 gains sont factices.
     - Sur HI_1_12_0, le « hors cadre » MONTE (3 249 → 4 029) : 780 paquets que `ti=43` arrêtait
       avancent jusqu'à la vue C. C'est un déplacement de cause, pas une perte (PLAN D-64).
   - **Live Fire, lecture par index de plage** : +3 269 / −3 et +1 806 / −49.
   - **Marche d'image-clé toutes générations** (oracle) : +1 784 / −1.
5. **Ce qui est écarté.** Ces pistes n'expliquent rien de mesurable dans la cause n°1 :
   - le décalage de masque (T2), le bourrage (T8) et les kinds 1/2 de la vue C (T5) ;
   - la garde par eid complet (T1-6) et les NEW refusés (T1-4) ;
   - le bloc `0xbc` : 6 et 9 paquets fermés sous ses deux formes, contre 11 et 10 pour les témoins
     décalés d'un bit, donc au niveau du hasard ; le lot L5 est sorti ;
   - `IDLowBits` (D-24) : 13 bits sur tout le corpus.
6. **Une mauvaise nouvelle de mesure : les fermetures factices.**
   - 8 388 paquets que nous comptons « fermés » (2,9 %) contredisent une règle de l'écrivain, dont
     22 % des fermés de HI_1_10_0 : le taux de fermeture de ce build est surestimé.
   - Les gains de certaines variantes sont aussi factices : 70 % de ceux de `tete-bloc` sur
     HI_1_10_0, et 382 des 401 gains de `ti=43` sur ce build.
7. **Le déclencheur de la représentation intermédiaire n'est atteint sur aucun build**, même sous
   l'oracle (§3).
   - La moitié « entrées » n'a qu'une borne basse : ≥ 45,0 % sur HI_1_13_0.
   - Fait nouveau : un lot qui lit plus loin fait monter le dénominateur lui-même. Sous la grammaire
     `ti=3`, HI_1_13_0 lit +255 477 records utiles de plus. Le nombre de records réellement écrits
     par le jeu reste inconnu.
8. **J12 : des recoupements réels avec les lots** (§6 et PLAN §6.0) ; « aucun lot ne touche les
   fichiers de J12 » était faux.
   - La branche J12 est visible localement comme référence distante :
     `origin/feat/suite-audit-decodeur-j12` = `bc0e2511a`, 29 commits au-dessus de `8b894a677`.
     L'inventaire est fait par `git diff`, fichier par fichier.
   - Les fichiers de production des lots touchés par J12 sont surtout `object_deaths_march.go`,
     `movement_states.go`, `frame_infer.go`, les `dispatch_*.go`, les lecteurs de position,
     `type1_datums.go`, `source/bits.go`, `film_context.go` et `registry.go`, plus `world.go`,
     `diagnostics.go` et `equivalence_lecteur_test.go` côté killsource.
   - **État des conflits (corrigé le 2026-10-02)** : aucun essai de fusion n'a été joué.
     - Attendus mécaniques par lecture du diff : tris, `go fix`, `errors.Is`, documentation.
     - Contraintes de STRUCTURE : J12.3 retire `slog` de `film_context.go` (L6a) ; J12.4 réécrit
       `registry.go` (L8). Ces lots écrivent sur la forme post-J12.
     - Un essai `git merge-tree` est un item à faire avant tout développement de phase 2 (PLAN §6.0).
   - Une montée de `grammar.Rev` périme le parc que J11.4 recuit en ce moment.
   - Une décision de l'utilisateur est demandée sur l'ordre des lots par rapport à la
     représentation intermédiaire (PLAN §6.3, D-RI : deux vagues de la campagne AVANT l'étape 1).

## 1. La cause n°1, ventilée (mesuré, carte v2)

« Vue C : terminateur hors cadre » = 264 757 paquets sur 20 films ; 3 142 830 records utiles en jeu
(table `product_use` corrigée).

| Ventilation | Valeur |
|---|---|
| Sortie de la vue B par **rejet hors datum** | 252 262 (95,3 %) |
| Sortie par terminateur | 12 495 (4,7 %) |
| Rejet de vue / autre sortie | 0 / 0 |
| Vue C vide (un seul bit 0) | 254 306 (96,1 %) |
| Reste derrière la vue C ≥ 64 bits | 264 321 (99,8 %) |
| Croisement « rejet, vue C vide, reste ≥ 64 » | 249 931 (94,4 %) |

Quand la vue B sort par rejet, le curseur reste après l'en-tête rejeté. La vue C lit alors le bit
suivant comme son terminateur : c'est le sélecteur de base du DELTA rejeté, nul dans 99,99 % des
DELTA.

Le jeu fait de même : `FUN_142987460` ignore le code de retour de `vtable[0x40]`. Sur ce point, le
Go est fidèle au jeu ; c'est le **monde** Go qui diverge (T3 C2, T8-C5).

Part du rejet hors datum dans le hors cadre, par build :

| Build | Part |
|---|---|
| HI_1_13_0 | 97,6 % |
| HI_1_10_0 | 99,0 % |
| HI_1_8_0 | 96,7 % |
| HI_1_11_0 | 97,2 % |
| HI_1_9_0 | 97,7 % |
| HI_1_12_0 | 98,5 % |
| version-33 | 86,7 % |
| version-31 | 77,5 % |
| HI_1_4_1 | 76,1 % |

Plausibilité de l'eid rejeté (mesuré, paquets hors cadre rejetés) :

| Classe | Paquets | Part |
|---|---|---|
| Naissance attestée par le bloc de type 1 du chunk suivant | 213 033 | 84,4 % |
| eid attesté par un autre bloc du film (même tête) | 15 898 | 6,3 % |
| Indéterminé | 16 344 | 6,5 % |
| Slot au-delà du plafond du film | 6 987 | 2,8 % |

« Naissance attestée » repose sur un classement de l'instrument qui a deux erreurs connues, à
corriger par le lot L0 :
- une naissance de génération 0 est rangée en « réallouée » : 12 854 paquets
  (`MESURES_BIS_3_POPULATIONS.md` §3.1) ;
- un NEW lu mais désynchronisé est rangé en « naissance non lue » : 3 560 paquets sur `81c02726`
  (`MESURES_BIS_2.md` §7.2). Ce film est HORS des 20 films : **l'ampleur de cette erreur dans les
  213 033 du corpus n'est pas mesurée** (au moins les 13 978 paquets de la région (iii) sur
  HI_1_13_0). Item ouvert, avec L0 (PLAN D-63, L0.5).

Le 84,4 % vient de la CARTE v2, qui publie ces deux classements sans correction ; elle porte depuis
le 2026-10-02 un bandeau qui le signale.

« Slot au-delà du plafond » n'est pas une question de largeur d'identifiant : c'est un curseur
décalé (§4).

La cascade est concentrée : 4 704 eid portent 94,2 % des paquets hors cadre rejetés (sur HI_1_13_0,
2 028 eid en portent 95,6 %). Une naissance ratée produit une cascade de rejets jusqu'à la fin du
chunk.

Où vivent les NEW ratés (mesuré à ≤ 3 paquets du premier rejet). Le témoin est un eid de même tête
sur un slot jamais alloué.

| Région | eid liés | témoins | paquets hors cadre portés | oracle limité à la région (net) |
|---|---|---|---|---|
| (i) tête d'un paquet à événements, sautée par le localisateur | 1 371 | 41 | 62 787 | **+16 383** |
| (ii) paquet à événements non localisé (perdu entier) | 1 910 | 95 | 66 788 | **+11 540** |
| (iii') après l'arrêt par rejet d'un paquet antérieur | 1 289 | 90 | 29 090 | **−522** |
| (iii') après un arrêt « ouverte » (désynchronisation) | 89 | 13 | 2 381 | +19 |
| (iii) NEW `ti=3` lu mais désynchronisé (tous HI_1_13_0) | 511 | 0 | 13 978 | **−65** |
| (iv) NEW lu mais refusé | 0 | 0 | — | — |
| **total lié** | **5 170** | **239** | | oracle-NEW +27 611 |
| introuvables | 2 856 eid | | 15 192 paquets (13 274 hors cadre) | — |
| trouvés à plus de 3 paquets (non liés) | 6 724 eid | | 61 858 | recherche R-P6 |

Notes sur ce tableau (mesuré, `MESURES_BIS_1.md` §4 et §6, `MESURES_BIS_3_POPULATIONS.md`) :
- **Région (iii)**.
  - Les 511 eid sont tous sur HI_1_13_0 (`fb1a1a72` 340, `51ebbc0f` 167, `c75f33b8` 4), et tous des
    `ti=3 i0`. Les autres builds n'ont de (iii) qu'à plus de 3 paquets.
  - La population « `ti=3 i0` » est plus large : 1 134 eid et 27 763 paquets. 623 de ces eid ont une
    occurrence que M1 préfère, ou un NEW désynchronisé à plus de 3 paquets.
  - La cause est la grammaire `ti=3`, que l'oracle ne corrige pas : porter cette grammaire ferme
    +30 618 / −10 paquets.
- **Liaisons fausses.** Les ≈ 4,7 % de liaisons fausses (environ 240 sur 5 170) sont une
  **estimation** tirée du seul taux du témoin. L'oracle lie avec le `R(6)` trouvé par M1, or ce
  `R(6)` contredit le pont masque -> archétype dans 144 cas sur 266 (D-7, non tranché). La fiabilité
  réelle des liaisons est donc inconnue au-delà de cette estimation.
- **Interactions.** Les régions interagissent peu. La somme des oracles par région vaut +27 355 ;
  l'oracle (i)+(ii) seul vaut +28 168.

## 2. État des huit pistes

Verdicts adverses de chaque constat : `campagne_grammaire_2026-10-01/VERIFICATIONS_ADVERSES.md`.

| Piste | État | Ce qui est établi (lu ET mesuré) | Probable, ouvert ou réfuté | Pièces |
|---|---|---|---|---|
| **T1** naissances non lues | **établi chez l'écrivain ; probable dans le flux** | Le jeu ne collecte qu'un mot par entité et par paquet (`FUN_142f2e174`). L'état 3, qui autorise un DELTA, est posé à l'écriture du NEW pour la vue `0x20` du film (`FUN_142f2f8f0`). La vue B est écrite dans l'ordre NEW*, DELTA*, DEL*, `000` (`FUN_14076b9c8`), slots croissants. Mesure : 84,4 % des paquets hors cadre rejetés ont une naissance attestée. | « Tout rejet est une naissance ratée » : probable, avec trois réserves (D-35, D-2, (iii')). Le chemin de LECTURE des régions (i) et (ii) n'est pas écrit (R-L1 c). | T1 ; MESURES §T1-1 ; BIS_1 §4 |
| T1-1 « rejet jamais légitime » | établi chez l'écrivain ; **nuancé par la mesure** | Le jeu n'écrit pas ce cas (deux lecteurs). | 4 598 paquets ferment au bit près après un rejet (3 756 sur HI_1_10_0) : pied de trame d'une vue vide ou fermeture factice, non tranché. La moitié est un `DEL ti=0` suivi d'un en-tête nul (BIS_3 §3.4). | MESURES §T1-1 |
| T1-2 ordre de la vue B | établi | Tient sur les paquets fermés : 97 violations seules, 0,03 %. | Comme oracle dur, il ne détecte presque rien dans la cause n°1. | T1, T3 C4 |
| T1-3 allocateur par pool | **établi pour la tête sur HI_1_13_0 ; indice faible sur HI_1_12_0 (rang 0 seulement : pool 1 à 13/15, pool 4 à 10/18) ; non établi pour le slot ; réfuté comme règle des vieux builds** | Table statique `0x143cefd78`, curseurs = mots de queue du bloc de type 1, tête `(gen+1)&3`. La tête est prédite pour 97,7 % des naissances ratées de HI_1_13_0. | Le RANG dans le pool n'est pas prédit (rang 0 : pool 4 à 44 %, pool 0 à 4/145, pool 2 à 12/41). A/B « tête par le bloc » : perte brute de 7 751 paquets sur le corpus, net **+11 354** ; perte nette en paquets sur HI_1_8_0, HI_1_10_0 et HI_1_11_0 ; HI_1_9_0 est net positif en paquets mais négatif en utiles. Le filtre d'invariants réduit ces pertes sans les annuler (BIS_1 §2). | T1 ; MESURES corr. 1, 10 ; BIS_1 §2 |
| T1-4 / T3 C5 NEW sur slot occupé | établi (mécanisme), **effet négligeable** | Le jeu évince l'occupant (`FUN_1408f18d0` rend toujours 1) ; le Go refuse si l'archétype diffère. | 1 228 refus : 7 créations perdues, 408 lectures fausses, ≤ 137 paquets touchés. | T1, T3 |
| T1-5 régions | mesuré contre témoin | (i) et (ii) dominent. | (iii') contraire à l'ordre de l'écrivain, non tranché (R-L1 a). (iii) = grammaire `ti=3`. | MESURES §T1-5, corr. 4-5 |
| T1-6 garde par eid complet | **contesté, contestation confirmée** | Le jeu compare l'eid complet (`FUN_1406cbaa0`), le Go compare le slot. | 1 154 DELTA sur 5,96 M, ≤ 325 paquets hors cadre : mise en conformité sans gain. | MESURES §T1-6 |
| **T2** décalage de masque | **réfuté comme cause** | Le Go indexe le masque comme l'écrivain (`FUN_142e2da44`, registre copié tel quel). La sortie 2 ne joue sur aucun des 20 films. | Le commentaire `frame_infer.go:122-125` est faux (D-13). | T2 |
| T2-4 bit hors archétype = témoin | établi pour les DELTA | Tient sur les DELTA fermés (0,027 %). | **Contredit pour les NEW** : 35 % des NEW de paquets fermés ont un masque impossible, ce qui donne les fermetures factices (§4). | T2, MESURES §T2-4 |
| **T3** fin de vue B | **établi** | Après un code 2/3, le lecteur du film continue ; le lecteur réseau abandonne (`FUN_14076b47c`). Aucune vue ne porte de longueur. | Le correctif « ne pas lire la vue C après un rejet » est écarté : 0 paquet gagné, jusqu'à 4 598 perdus. | T3 |
| **T4** positions | **mesuré par A/B** ; T4-C3 **établi sur Live Fire** | T4-C3 : le jeu lit la ligne de l'index lu (`FUN_14076e524`), le Go la plage cataloguée. Lire par index ferme +3 269 / −3 paquets sur `0797ce72` et +1 806 / −49 sur `60ae07c4`, contexte de production. | Par site : `flock-position` +416 / −10 et `tacmap-displayasset` +77 / −16 ; `world-object-i0` +12 338 records d'image-clé mais +31 paquets net ; `ti38-i18` et `unit-actor-state` en perte. T4-C2 contredit : `d9781168` 34:336 est perdu sous toutes les variantes. Ordre des plages des cartes à deux sbsp non lu. Un seul exécutable, donc rien n'est lu pour les vieux builds. | T4 ; BIS_2 §3 |
| **T5** vue C | **établi** | La vue C d'un film ne porte que des kind 0, au plus un par joueur, index croissants, ≤ 32 (`FUN_142f2c3b0`, `FUN_14076b0e8`). Mesure : 279 paquets fermés sur 284 704 sont contredits par la vue C (0,1 % ; « 284 425 » était le nombre de fermés NON contredits), dont 119 le sont aussi par le masque (fermetures factices). | Bloc `0xbc` : grammaire lue, mais **jamais lu au bon bit** sur le corpus (les deux formes ferment autant que les témoins). La fourche `+0x74` n'est pas tranchable par la fermeture. La borne « ≤ 953 paquets » est réfutée. | T5 ; BIS_2 §2 |
| **T6** véhicules `ti=40` | **établi** (porte, type de physique) ; T6-C5 réfuté | Porte `+0x818` = (type de physique du tag vehi == 6) (`FUN_14058c2ec`, `FUN_1408b44fc`). Types lus dans les tags installés : Falcon et Wasp sont de type 6, toutes les familles connues sont cohérentes. 0 contradiction en delta. | Image-clé : la porte levée ferme 136 records sur 7 059, la porte posée 26. 98 % restent non fermés : une autre largeur `ti=40` est fausse, non identifiée. Les châssis `77ef810a`, `4118381d` et `d0b40d0a` sont absents des modules installés et non identifiés (probablement des lectures fausses de NEW : hypothèse). | T6 ; BIS_2 §4 |
| **T7** dispositifs et moteur | **établi pour `ti=43`** (grammaire lue, lecteur et écrivain ; fermeture mesurée) ; **T7-5 et T7-6 réfutés sur HI_1_13_0** | Corpus +18 105 / −12 paquets. HI_1_12_0 passe de 5 835 à 17 132 fermés ; son hors cadre monte de 3 249 à 4 029 par déplacement de cause (780 paquets que `ti=43` arrêtait atteignent la vue C, D-64). Le hors cadre de HI_1_13_0 baisse de 86 921 à 81 049. `81c02726` gagne +3 580 paquets (le NEW désynchronise sur `i19`, pas sur `i20`/`i21`/`i22`). | HI_1_10_0 : 382 gains factices sur 401. `ti=2`/`ti=0` : grammaire lue, gain de portage non mesuré. La forge `ti=0 i18-i26` n'est pas relevée. | T7 ; BIS_2 §5 |
| **T8** bourrage | **réfuté comme cause** | Le jeu rend des zéros au-delà du tampon (`FUN_1406d6c7c`) et déclare l'échec au-delà de 8 × taille (`FUN_14298816c`). | 740 paquets débordent ; effet indirect : 24 paquets. | T8 |

## 3. Le déclencheur de la représentation intermédiaire, recalculé

Définition (spec) : ≥ 95 % des records utiles fermés **par build**, et la même part pour les entrées
de contrôle utiles.

**Quel dénominateur.** Le pourcentage publié jusqu'ici divise par les records utiles lus PAR LA
MARCHE (dénominateur variable). Or ce nombre change d'une variante à l'autre : une marche qui lit
moins semble fermer plus (`MESURES_BIS_1.md` §7). Les colonnes ci-dessous utilisent donc :
- le dénominateur **fixe maximum** : par film, le maximum des records utiles lus sur les 14 marches
  de bis 1, le même pour toutes les variantes ;
- un numérateur compté en paquets **sains**.

Pour la phase 2, la règle est complétée (PLAN §6.0, « Pourcentages ») : le fixe est le maximum sur
{référence, lots déjà fusionnés, oracles mesurés}, recalculé à chaque vague, et le variable est
publié à côté.

| Build | Référence (variable) | Référence (fixe max) | Oracle (i)+(ii) (fixe max) | Sains, référence → oracle (i)+(ii) (fixe max) |
|---|---|---|---|---|
| HI_1_13_0 | 80,6 % | 78,4 % | **85,9 %** | 78,1 % → **85,6 %** |
| HI_1_8_0 | 31,0 % | 29,1 % | 45,4 % | 28,9 % → 45,2 % (*) |
| HI_1_9_0 | 26,4 % | 25,7 % | 36,8 % | 25,6 % → 36,7 % (*) |
| HI_1_11_0 | 27,7 % | 26,9 % | 35,0 % | 26,7 % → 34,8 % (*) |
| HI_1_12_0 | 28,9 % | 28,1 % | 30,7 % | 28,1 % → 30,7 % (*) |
| HI_1_10_0 | 19,4 % | 18,7 % | 22,7 % | 18,3 % → 22,1 % (*) |
| version-33 / HI_1_4_1 / version-31 | ≤ 0,8 % | ≤ 0,7 % | ≤ 0,8 % | ≤ 0,8 % |
| corpus | 43,4 % | 42,0 % | 47,8 % | 41,8 % → 47,5 % |

Sources : `mb_agg_variantes_par_build.tsv`, colonne `pct_max`, et `MESURES_BIS_1.md` §7.
(*) Pour ces builds, la valeur est recalculée ici avec `mb_variantes.tsv` (somme par film du maximum
des 14 marches) et `mb_agg_fermes_sains_par_build.tsv`. L'écart est ≤ 0,1 point avec bis 1 là où
bis 1 donne la valeur.

- **Moitié « records » : non atteinte sur aucun build**, même sous l'oracle et quel que soit le
  dénominateur.
- **Le dénominateur fixe ne compte pas les records écrits par le jeu.** Ce n'en est qu'une borne
  basse, et elle monte dès qu'un lot lit plus loin.
  - Sous la grammaire `ti=3` (mesuré, `mb3_ti3.tsv`), HI_1_13_0 lit 2 759 700 records utiles, contre
    2 504 223 en référence et 2 574 513 au maximum des marches de bis 1.
  - Sur ce dénominateur, la part tombe à 73,1 % pour la référence et 81,6 % sous `ti=3`. L'oracle
    (i)+(ii) tombe à ≤ 80,1 % (arithmétique sur les TSV, estimé : l'oracle et `ti=3` ne sont pas
    mesurés ensemble).
  - De même sous `ti=43`, HI_1_12_0 lit 139 947 records utiles contre 121 628. Sa part sur ce
    dénominateur passe de 25,1 % à 82,3 %.
- **Moitié « entrées »** (item 1.3, `MESURES_BIS_1.md` §1).
  - L'estimateur de la CARTE §5 est **tautologique**, prouvé et mesuré : il redonne la part des
    paquets fermés (66,9 % sur HI_1_13_0).
  - Le dénominateur indépendant suit la règle de l'écrivain : au plus une entrée par joueur (T5). Il
    donne des **bornes basses** : sur HI_1_13_0, ≥ 45,0 % (sièges ∪ joueurs vus), ≥ 49,7 % (table du
    film), ≥ 49,3 % (entités `ti=9`) ; sur le corpus, ≥ 24,2 %.
  - Ce ne sont que des bornes basses : sur HI_1_13_0, seuls 19 % des paquets fermés portent une
    entrée par siège.
  - Sous l'oracle (i)+(ii), HI_1_13_0 est ≥ 48,8 % (≥ 48,5 % sous l'oracle-NEW ; `MESURES_BIS_1.md`
    §7). La moitié « entrées » **ne peut pas être déclarée atteinte**.
  - Le dénominateur exact demande de lire dans Ghidra l'appelant de `FUN_14076b0e8` (statut proposé :
    `[~]` borne basse, `[!]` exact).
- **Conséquence (estimé) : la phase 2 seule ne fera probablement pas atteindre le déclencheur.**
  C'est une extrapolation : les leviers sont mesurés SÉPARÉMENT (L1, L8, L2, L9), et leur combinaison
  sur HI_1_13_0 n'est ni mesurée ni estimée. La recherche R-COMB (PLAN §6.0) la mesure, en copie de
  recherche. La décision d'ouvrir
  l'étape 1 de la représentation intermédiaire est DÉJÀ PRISE par l'utilisateur le 2026-10-01 : elle
  démarre après la fusion de J12 (ANALYSE §7 (2)). La question qui reste : faut-il réviser le seuil
  de 95 % ou sa définition (dénominateur) ? (PLAN §6.3, D1.)

## 4. Ce que la phase 1 a appris en plus (non prévu)

1. **Fermetures factices** (mesuré).
   - 8 388 paquets fermés (2,9 %) contredisent un invariant de l'écrivain, surtout par un NEW à
     masque impossible dans un paquet d'un seul record.
   - Il y en a 6 410 sur HI_1_10_0 (22 % de ses fermés) et 1 487 sur HI_1_13_0.
   - `vueCFermee` est nécessaire, pas suffisant.
   - Filtrer le localisateur de production par les invariants (`bande+inv`) retire 7 314 de ces
     fermetures factices et ne perd que 29 paquets sains.
2. **Les naissances ratées sont probablement des entités éphémères.**
   - Mesuré : 5 714 des 6 033 eid « naissance non lue » ont un masque vide au bloc suivant.
   - « Née et morte dans le chunk » est une **interprétation** : aucune mesure ne sépare une entrée
     libérée d'une entrée vivante sans composant.
   - Dans les deux cas, la table anticipée ne les rattrape pas.
3. **Image-clé incomplète** (87 eid, 13 643 paquets hors cadre).
   - L'image-clé porte ces entités (en-tête exact présent, 87 sur 87) ; c'est la marche du Go qui les
     perd.
   - Elle traite la génération 0 comme nulle (`kfAnchorFromID`) et limite le voisin et le recalage à
     la génération 1.
   - Contrôle : la génération 0 n'est jamais déclarée par la marche (0 sur 191), la génération 1
     toujours (373 612 sur 373 612).
   - Oracle : +1 784 / −1 paquets → lot L9.
4. **La génération 0 est valide dans le jeu**, et le Go la traite comme nulle en au moins deux
   endroits : la marche d'image-clé, et la classe de naissance des instruments. 99,6 % de la classe
   « réalloué » (12 906 paquets) sont des naissances de génération 0.
5. **Deux composants portent le même nom (`high-frequency`) avec deux grammaires.** Le dispatch Go
   les route par nom (`dispatch_item.go:122`), ce qui est faux pour `ti=3`. D'autres homonymes peuvent
   exister.
6. **« Aucune allocation » et « au-delà du plafond » sont des décalages de curseur, pas des
   entités.**
   - Les têtes des eid rejetés sont réparties au hasard sur des slots jamais alloués.
   - Coupables prouvés par A/B : la lecture par index de plage sur Live Fire (−81 % et −91 %), et
     `flock-position` (−77 % de ce qu'il précédait).
   - Coupables non attribués : `ti=20 i1` (694 paquets), `DEL ti=0` (658), `ti=14 i1` (241),
     `ti=41 i2` (161) → R-P3.
7. **`IDLowBits` (D-24)** : la largeur vaut `ceil(log2(cardinal))`, et le cardinal vaut 8 191 sur
   chaque bloc du corpus. Ce n'est pas une cause. `1c4c63c2` alloue déjà le slot 8 190.
8. **Région (iii')** : 1 289 NEW trouvés derrière un rejet antérieur, ce que l'ordre de l'écrivain
   interdit derrière un DELTA lu. Non tranché ; l'oracle sur cette région est négatif.
9. **Pont masque -> archétype (D-7)** : le `R(6)` trouvé contredit le pont dans 144 cas sur 266, et
   l'oracle-NEW lie avec ce `R(6)`.
10. **Allocateur** : la table de pools ne prédit rien avant HI_1_12_0 (autre table, ou
    `DAT_144706104` à 0).
11. Liste complète : PLAN §5 (D-1 à D-66).

## 5. Ce qui est établi, probable ou ouvert, en une phrase par point

- **Établi** (lu ET mesuré) :
  - l'écrivain écrit un NEW avant tout DELTA de l'entité, et la vue B dans l'ordre
    NEW*/DELTA*/DEL* ;
  - la vue C d'un film ne porte que des kind 0 ;
  - la porte `+0x818` est le type de physique 6 (vtol) ;
  - la grammaire `ti=3 i0` (`FUN_142ed4aec`) et la grammaire `ti=43` (T7) ;
  - la lecture par index de plage sur Live Fire ;
  - la largeur d'identifiant de 13 bits sur le corpus ;
  - T2, T8, les kinds 1/2, la garde par eid et le bloc `0xbc` ne sont pas des leviers mesurables.
- **Mesuré, sans lecture du jeu qui l'explique** :
  - les fermetures factices ;
  - la borne de L1 limitée aux régions (i) et (ii) ;
  - les A/B des sites de position.
- **Probable** :
  - la grammaire `ti=3 i1` (`FUN_142ed4880`, 26 bits) : lue, mais l'appartenance de sa table
    `0x143d07af0` à `ti=3` est déduite (voisinage, champs, mesure), pas lue ;
  - la tête prédite par l'allocateur sur HI_1_12_0 (indice faible : rang 0 seulement) ;
  - que nos rejets hors cadre soient des naissances ratées dans ≥ 84 % des paquets. Réserves :
    D-35, D-2, (iii') ; ce classement a aussi deux erreurs d'instrument (§1), dont l'ampleur de la
    seconde (D-44) sur le corpus n'est pas mesurée ;
  - que les naissances ratées soient nées et mortes dans le chunk ;
  - que les trois châssis inconnus soient des lectures fausses.
- **Supposé** :
  - que les régions (i) et (ii) se lisent par grammaire (vue A complète) plutôt que par recherche
    d'en-tête (R-L1 c, lot L1) ;
  - que les grammaires lues sur l'exe HI_1_13_0 vaillent pour les vieux builds (positions, `ti=2`).
- **Ouvert** :
  - région (iii') ;
  - les 4 598 paquets fermés après un rejet ;
  - D-35 ;
  - D-7 ;
  - la fourche `+0x74` (rôle de `FUN_1404f293c` en relecture Theater) ;
  - la largeur `ti=40` fausse en image-clé ;
  - l'ordre des plages des cartes à deux sbsp ;
  - le contexte état complet de `ti38-i18` ;
  - les naissances à plus de 3 paquets (R-P6) ;
  - le dénominateur exact des entrées ;
  - la combinaison des leviers (R-COMB) ;
  - la part de D-44 dans les naissances attestées du corpus.

## 6. Suite

Plan de la phase 2 : `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` §6. Il contient les lots
reclassés, les recherches préalables, les gates complétés, l'inventaire des recoupements avec J12 et
les décisions demandées (§6.3).

En résumé, trois points structurent la suite :
- **J12**. La chaîne qui bloque sa fusion : la vague J11.4 tourne sur l'autre PC de l'utilisateur
  (fin estimée vers 5-6 h le 2026-10-02, chiffre transmis, non vérifié ici), puis vient J11.5. J11.6
  est fait (`8b894a677`). La fusion de J12 est visée en fin de matinée du 2026-10-02 et sera suivie
  de la revue adversariale finale du chantier.
- **Les gates de phase 2 tournent sur CE PC** : aucune concurrence de RAM ou de CPU avec la vague.
- **`grammar.Rev`** : sur la branche, chaque lot qui change une sortie le monte ; au parc, UNE
  recuisson par vague fusionnée. La première périme le parc que J11.4 recuit : backlog
  `redecoder` / killsource (ADR 0034). Tout lot qui change une sortie, composants compris, passe le
  gate killsource.
- **Une décision de l'utilisateur est demandée (D-RI, recommandation remplacée le 2026-10-02)** :
  - vague 1 = lots de composants (L2, L8, L9, L6a, L6b, puis L4 et L3 si leurs prérequis sont
    levés), fusionnés ensemble, une recuisson ;
  - références re-figées ;
  - vague 2 = unification à zéro différence des localisateurs jumeaux (LU), puis L1 et L7 posés une
    fois, deuxième recuisson ;
  - ensuite seulement l'étape 1 de la représentation intermédiaire, puis 2.x.

  C'est un changement par rapport à l'ordre « marcheur avant la campagne » de l'ANALYSE §6 : L1
  atterrit une fois grâce à l'unification, et les plus gros gains arrivent plus tôt.

## 7. Corrections de cette révision (points de la critique)

| Point | Avant (2026-10-01) | Après |
|---|---|---|
| 1 | « aucun lot ne touche les fichiers de J12 » | Inventaire par `git diff` (PLAN §6.0) ; conflits mécaniques réels. |
| 2-4 | Dépendance à J12 non justifiée ; chaîne absente ; conflit avec la recuisson absent | PLAN §6.0 : chaîne J11.4 → J11.5 → J12, option « prouver maintenant », backlog, gates sur ce PC. |
| 5 | « seul L0 n'a aucune dépendance » | Le ratchet J12.7 accepte les fichiers de la campagne. `frame_closure_detail*.go` restent de la production non taguée, appelée par l'instrument seul, comme `FrameClosure`. |
| 6-8 | D1 « ouvrir l'étape 1 ? » | D1 est déjà tranchée (ANALYSE §7 (2)) et reformulée en « réviser le seuil ? ». Ajouts : D-RI (place des lots par rapport à la représentation intermédiaire) et D-VEH (ANALYSE §7 (6), restée ouverte). |
| 9 | « perte nette … (−7 751 sur le corpus) » | Perte brute 7 751, net +11 354 ; perte nette en paquets sur HI_1_8_0, HI_1_10_0 et HI_1_11_0 seulement. |
| 10 | Oracle = « borne », pourcentages sur dénominateur variable | Borne d'UN mécanisme ; dénominateur fixe maximum et paquets sains (§3). |
| 11 | `oracle-bloc+tete-bloc` non publiée | MESURES corr. 6. |
| 12-13 | (iii) « presque tout `ti=3 i0` » ; ligne (iii') ouverte omise ; témoins ne somment pas | Table du §1 : 5 170 liaisons, 239 témoins, (iii) tous `ti=3 i0`, tous HI_1_13_0. |
| 14 | « moteur `ti=2` (4 331) » ; « 12 112 sur un film » ; 92,7 % non expliqué | `ti=2` 4 208 + `ti=0` 123 = 4 331 ; `ti=43` 12 127 paquets sur HI_1_12_0, dont `i35` 12 112 ; 92,7 % (table corrigée) contre 92,5 % (ancienne table). |
| 15 | « relue par deux vérificateurs », sans pièce | `VERIFICATIONS_ADVERSES.md`. |
| 16 | « Établi : le jeu n'écrit jamais un DELTA non déclaré ; nos rejets sont donc des naissances » | Établi chez l'écrivain ; probable dans le flux, avec trois réserves. |
| 17 | T1-3 « établi sur HI_1_13_0 » | Établi pour la tête ; non établi pour le slot. |
| 18 | « masque vide = né et mort » présenté comme mesure | Interprétation (§4.2). |
| 19-20 | D-7 et la limite de M1 absents | §1 et §0.2. |
| 21 | Gains non passés aux invariants | §0.6 et MESURES corr. 9. |
| 22, 38 | Entrées « supposé » | §3 : estimateur tautologique, bornes basses mesurées. |
| 23-29 | Mesures déclarées impossibles ; populations sans lot | Intégrées (§0.4-0.5, §2, §4) ; lots L8 et L9, recherches R-P3 et R-P6, L5 sorti. |
| 24 | Bloc `0xbc` : « borne ≤ 953 » | Réfutée. |
| 26 | `81c02726` non mesuré | Mesuré : +3 580 paquets, blocage sur `i19`. |
| 30 | L3 non instruit sur les vieux builds | R-L3 créée, puis dotée d'un porteur, d'une méthode et d'un gate (PLAN §6.2, 2026-10-02). La cause reste non élucidée. |
| 31 | Gate killsource absent | PLAN §6.0 gate 3 : équivalence ou delta déclaré, `killsource.Rev`, backfill déclaré ; `walk.go` dans les fichiers de L1. Étendu aux lots de composants (N1). |
| 32 | Pas de gate de performance | PLAN §6.0 gate 4 ; plafond recommandé +10 % de durée et de pic mémoire (D11, N8). |
| 33 | Gate par build | PLAN §6.0 gate 2 : « aucune baisse sur AUCUN FILM », en paquets et en utiles sains. |
| 34 | Garde-fou de la recopie sans paquet à événements | D-61 (branche non localisée prouvée, `listes_non_localisees=27`) ; gate 5 et L0.3 pour la branche localisée. |
| 35 | Prérequis de L4 sans porteur | PLAN L4 « Prérequis, avec porteur et méthode » ; D5. |
| 36 | L5 et L6 rangés en « gain mesuré » | L5 sorti (mesuré, BIS_2 §2) ; L6a et L6b mesurés en A/B ; L3 et L4 marqués « estimé ». |
| 37, 38 | Aucun statut au PLAN §2 ; item 1.3 sans `[!]` | Statuts proposés au PLAN §6.4 ; leur report au §2 et l'en-tête du plan relèvent du superviseur (non fait dans ce rapport). |
| 39 | Gate 1 sans trace | PLAN §4 (journal), BIS_1 §10, BIS_2 §8, BIS_3 §10. Il manque `archlint` (N2). |

**Critique n° 2 (2026-10-02, `CRITIQUE_COMPLETUDE_2.md`)**

| Point | Avant | Après |
|---|---|---|
| N1 | Gate killsource réservé aux lots de marche | PLAN §6.0 gate 3 : TOUT lot qui change une sortie (`walk.go:69` appelle `grammar.DecodeFrameRecords`) ; L8 cite le point 3. |
| N2 | `archlint` jamais joué ; deux sondes > 500 L | Sondes découpées le 2026-10-02 (plus grand fichier de la campagne : 480 L, `wc -l`) ; `go test ./internal/archlint/` reste à jouer et consigner (PLAN D-66, §6.4). |
| N3 | Surcouche hors CI, non protégée contre J12 | D10 complétée : mesure seulement, hors CI, copies re-synchronisées à la fusion de J12 (item du PLAN §6.0), supprimées à la fusion du lot. |
| N4 | D-RI : L1 « avec ou après 2.7 » (cycle avec ANALYSE §7 (2)) | D-RI remplacée : L1 en vague 2, avant l'étape 1 ; 2.7 attend L1. |
| N5 | Composants fusionnés « pendant l'étape 1 » | Aucun lot de comportement pendant l'étape 1 ; références re-figées après chaque vague. |
| N6 | Gate 1 « une montée par lot » contre D7 « une par vague » | Règle unique : une montée par lot sur la branche, une recuisson par vague au parc (gate 1, D7). |
| N7 | Dénominateur fixe sans ensemble défini | Maximum sur {référence, lots fusionnés, oracles mesurés}, recalculé à chaque vague ; fixe et variable publiés (PLAN §6.0, D1). |
| N8 | Plafond du gate 4 absent des décisions | D11 : +10 % de durée et de pic mémoire au plus, trois témoins et un BTB. |
| N9 | « Tous les conflits sont mécaniques » | §0.8 et PLAN §6.0 : mécaniques attendus (tris, `go fix`, `errors.Is`) ; contraintes de structure J12.3 (`film_context.go`, L6a) et J12.4 (`registry.go`, L8) ; essai `git merge-tree` avant la phase 2. |
| N10 | « La phase 2 seule ne fera pas atteindre le déclencheur » | §3 : marqué « estimé » ; combinaison non mesurée ; recherche R-COMB. |
| N11 | Grammaire `ti=3` « établie » | §0.4 et §5 : établie pour `i0`, probable pour `i1` (table déduite). |
| N12 | Tête « établie sur HI_1_12_0 » | §2 T1-3 et §5 : indice faible sur HI_1_12_0 (rang 0 : 13/15, 10/18). |
| N13 | CARTE v2 sans bandeau | Bandeau « Corrections du 2026-10-02 » en tête de la CARTE (estimateur tautologique, D-43, D-44). |
| N14 | Part de D-44 sur le corpus non mesurée | §1 et §5 ; item ouvert L0.5 (PLAN D-63). |
| N15 | « Estimateur CARTE §5 » de BIS_1 ≠ CARTE | BIS_1 §1 : estimateur au niveau du build agrégé, la CARTE somme par film ; HI_1_4_1 indéfini (PLAN D-65). |
| N16 | 279 / 284 425 | 279 / 284 704 (§2 T5 ; MESURES, correction 22). |
| N17 | `DEL ti=0` 622 | 658 (`mb3_tables.tsv`) ; le 622 recopiait `flock-position` (§4.6, BIS_3, PLAN D-56). |
| N18 | `81c02726` hors cadre 4 149 / 4 150 | 4 149 après un rejet + 1 après un terminateur = 4 150 (BIS_2 §5.1). |
| N19 | D-5 « 13 644 hors cadre » ; « ≥ 48,5 % sous l'oracle » ; 598 contre 623 eid | 13 644 paquets dont 13 643 hors cadre ; 48,8 % sous (i)+(ii) (48,5 % sous l'oracle-NEW) ; 623 = 598 (retenus en (iii'), (ii), (i)) + 25 (iii) à plus de 3 paquets (BIS_1 §9). |
| N20 | J12R : « NON poussée » et « Vérifié par diff » | Section « Faits transmis » marquée périmée sur la poussée. |
| N21 | HI_1_12_0 : hors cadre en hausse non commenté | Déplacement de cause mesuré : 780 paquets que `ti=43` arrêtait atteignent la vue C (§0.4, §2 T7, PLAN D-64, BIS_2 §5.4). |
| 1 (reste) | `world.go`, `diagnostics.go`, `equivalence_lecteur_test.go` absents de l'inventaire | Ajoutés au PLAN §6.0. |
| 3 (reste) | ANALYSE §6 et SUITE non réconciliés | PLAN §6.0 : la chaîne suit les faits transmis ; le schéma de l'ANALYSE §6 est périmé sur ce point. |
| 5 (reste) | « 0 code mort » non traité | PLAN D-62 et D9 : question posée (déplacer sous `film/research/` ou justifier), non tranchée. |

## 8. Recherches préalables du 2026-10-02

Ajouté le 2026-10-02 ; le reste de ce rapport n'est pas réécrit. Le détail, les verdicts adverses et
les renvois aux notes sont au PLAN §6.5 ; les découvertes neuves au PLAN §5 (D-68 à D-103).

**Ce qui a été fait.** Six chantiers de recherche (workflow `wf_9088d8bd-e43`, worktrees temporaires
sur `fe18bf67c`, aucun fichier de production), chacun relu par un vérificateur adverse qui a
recalculé les chiffres sur les TSV et relu Ghidra en lecture seule. Notes : `R_COMB.md`, `R_LOC.md`,
`R_NAIS.md`, `R_COMP.md`, `R_VEH.md`, `R_FUSION.md` (sous `campagne_grammaire_2026-10-01/`), intégrées
le même jour (176 fichiers).

**En bref**

| Question | Réponse | Statut |
|---|---|---|
| Les leviers ensemble (R-COMB) | HI_1_13_0 : 86,0 à 88,0 % des records utiles en paquets sains (dénominateur fixe de R-COMB), 83,7 à 85,7 % sur le fixe consolidé ; corpus 50,6 à 51,3 % (46,4 à 47,1 %). Les gains se recouvrent (HI_1_13_0 : somme des seuls +547 609 utiles sains, combinaison +465 529 à +525 771) | mesuré |
| « La phase 2 seule n'atteindra pas 95 % » (§3) | Vraie pour les six leviers mesurés. Au-delà, l'estimation de R-COMB n'est pas une borne (vérificateur), et LS, LP, LM n'étaient pas dans la combinaison : non décidable sans R-COMB-2 pour HI_1_13_0, HI_1_12_0 et les formats 24-25 | partiel |
| Signature du localisateur figée sur le slot 123 (R-LS, D-67) | Confirmée sur pièces. Sur la cuisson, l'ordre « 123 → fermeture par NEW de tête → signature high-frequency » gagne +18 119 paquets (+18 086 sains), 0 film en baisse ; l'ordre de l'enquête fait baisser un film. Killsource : 1 080 morts rendues à la marche, valeurs inchangées, mais la voie est publiée (backfill dû) | établi |
| Naissances (R-L1) | (a) les « NEW » trouvés derrière un DELTA ne sont pas des records (l'écrivain ne l'écrit jamais) ; (b) les 4 598 fermés après un rejet sont factices ; (c) la vue A ne se lit qu'en portant la charge de 41 genres de messages ; (d) condition par film pour L1a : +15 070 sains, 0 film en baisse, seuil choisi sur le corpus | établi ; (c) partiel |
| Naissances lointaines (R-P6) | Pas de règle générale ; pas de lot | partiel |
| Moteur `ti=2` (R-L3) | Portage HI_1_13_0 : +3 575 sains, 0 film en baisse. Vieux builds : un bit de trop, mais sa position (`i4` à `i9`) n'est pas discriminée | établi / NON CONFIRMÉ (position) |
| Homonymes (R-HOM) | Un seul homonyme de grammaire (`high-frequency`) ; la grammaire de `ti=3 i1` est établie | établi |
| Décalages résiduels (R-P3) | Lecteurs justes ; liaisons fausses de slot ; règle « désaveu d'une déclaration d'image-clé que le bloc dit non vivante » : +694 sains, 0 perdu ; site `ti=41 i0` à ajouter à L6a | partiel |
| Véhicules `ti=40` (R-L4) | Trois lectures communes d'image-clé (portée, `i0` écrivain, MPP 8/3) ferment 3 058 / 3 058 records ; delta : +1 436 / −0. Le MPP 8/3 seul rapporte +80 979 paquets sur 6 films des formats 24-25 (pertes non jugées) | établi (MPP : mesuré) |
| Plages des cartes (R-L6) | Ordre établi ; « index 1 sur une carte à une plage » réfuté ; L6a hors Live Fire réfuté (−15 sains) | établi / réfuté |
| Fusion de J12 | Un conflit (empreinte de révision), quatre sondes à mettre à l'accesseur de J12.4, 11 ratchets J12 verts ; vraie fusion faite (`f28a4a816`), carte v2 identique après J12 | établi |

**Effet sur les lots (proposé, décisions D12 à D17 au PLAN §6.3)**
- Vague 1 : L8, L2, L3a (portage, mesuré), L4a (delta, mesuré), L6a (Live Fire + site `ti=41 i0`),
  L6b, L9 ; candidats neufs LM (MPP 8/3) et LK (image-clé sous portée), puis L4b.
- Vague 2 : LU avec un paramètre d'ordre, LS, L1a sous condition par film, LP (neuf), L7.
- Sortis : L1b (devient un chantier de grammaire des messages de la vue A), L1c (confirmé), L6a hors
  Live Fire, tout lot P6.
- L0 : quatre invariants ou témoins de plus (sortie de vue B par rejet, masque au-delà de
  l'archétype, mot de DEL non nul, témoins à 2-8 bits).

**Points non confirmés par les vérificateurs** : position du bit de trop des vieux builds (`i4`) ;
« lecture fausse » de deux châssis sur trois ; « critère de bascule de `PorteeBaseline` rempli » ;
explication de l'ancienne contradiction du MPP ; partie estimée de la phrase des 95 %.

**Outillage** : les surcouches de mesure des chantiers sont d'avant J12 (sauf celle de la fusion) et
ne compilent plus ensemble sous le tag commun (`go vet` rouge avec quatre sur cinq) : une surcouche
unique post-J12 est à faire avant toute mesure de la vague 1 (D17). `archlint` était rouge sur
toutes les têtes du 2026-10-02 (`TestNoExpiredTODO`, échéance du 2026-10-01 hors campagne) ; soldé
par `feat/v75` (`2f1e7b98b`), vert sur la tête de la campagne.
