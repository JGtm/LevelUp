# Mesures C.1, C.2, C.3 : gestes des joueurs (décodeur de film, v1.6.0)

Date : 2026-11-23. Aucun code de production modifié, aucun commit. Sondes jetables non versionnées (`internal/grammar/mesc_research_test.go`,
`mesc2_research_test.go`, `research/cmd_mescmap/` dans la branche de recherche), scripts d'analyse et sorties dans ce dossier (hors dépôt).
Corpus : 22 films, builds HI_1_4_1 à HI_1_13_0 (`builds.tsv`). Un film se désigne par son identifiant court. Aucun identifiant de joueur ici.
Marques : **vérifié** = rejoué sur les films ; **déduit** = raisonné.

## Suppositions et limites (à lire d'abord)

- La vérité demandée en C.1 (les 7 rejeux du parc qui publient accroupi et glissade par les composants) est inutilisable.
  **Vérifié** : les trois films du corpus où ces composants vivent (a349fea8, a521164d, 50247b26 ; builds anciennes) ne rendent
  AUCUNE entrée de contrôle ; aucune comparaison bit à bit n'est possible. Ailleurs, ces composants ne donnent que l'état de
  naissance du corps (quelques dizaines de lectures par film). Les 7 rejeux du parc n'ont pas pu être identifiés sans ouvrir une
  base du dossier principal : mesure sur le corpus seul.
- La vérité de remplacement est PHYSIQUE (vitesse et trajectoire verticale lues dans le film) : nécessaire, pas suffisante.
- Le lien manette-corps vient du document de rejeu (indice de joueur, corps, frames de vie) ; horloge approchée. **Vérifié** : sur
  4 films, l'accord « manette poussée / corps en mouvement » dépasse 80 % pour 98 % des vies évaluées.
- Seuils de vitesse fixés sur un film (000d5950), appliqués aux autres : accroupi <= 1,65 ; debout 1,8 à 3,4 (unités du décodeur).

## C.1 Accroupissement et glissade

Lecture : valeur 1 (bit d'indice 0) des drapeaux de l'entrée de contrôle.

Accord physique sur instants stables (état inchangé depuis 0,5 s, corps au sol, manette poussée à fond), tous builds :

| Mesure | Résultat |
|---|---|
| Bit posé, vitesse compatible (<= 1,65, ou >= 3,4 pour la glissade) | 91 829 / 96 269 = **95,39 %** |
| Vitesse debout normale (1,8 à 3,4), bit nul | 1 857 709 / 1 861 655 = **99,79 %** |
| Par film, bit posé | de 84,7 % (51101d1d, 601 instants) à 97,8 % (03b7bbb6) ; un seul film sous 90 % : 51101d1d |
| Par build, bit posé | HI_1_8 95,65 % ; HI_1_9 95,09 % ; HI_1_10 92,38 % ; HI_1_11 96,17 % ; HI_1_12 95,21 % ; HI_1_13 95,73 % |

La vitesse lente sans bit n'est pas une erreur : marche arrière, lunette ou mur ralentissent aussi (**déduit**).
**La hauteur ne distingue pas l'accroupi** : variation de hauteur à +0,4 s d'un front montant : 0,000 m en médiane (34 fronts,
000d5950). **Vérifié.**

Ce qui distingue accroupi et glissade (1 710 séries du bit, HI_1_12 et HI_1_13) :

| | accroupi | glissade |
|---|---|---|
| Vitesse au début de la série | < 2,5 (médiane 1,40) | >= 2,5 (pic médian 3,6) |
| Durée p10 / médiane / p90 | 0,10 / 0,62 / 4,51 s | 0,13 / 0,47 / 1,22 s |
| Vitesse maximale de la série | <= 1,7 dans 51 % ; >= 3,3 dans 2,4 % | >= 3,3 dans 74 % ; jamais <= 1,7 |
| Sprint actif au début | 3 / 793 | 1 / 893 |

Même bit pour les deux : la glissade est ce bit posé à vitesse de sprint ; le sprint cesse dès qu'elle commence. **Vérifié.**

**Verdict C.1.** Accroupi : **publiable** (95,4 % et 99,8 %, builds HI_1_8 à HI_1_13), réserve : vérité physique, pas celle du plan.
Glissade : **non publiable à 90 %** (74 % des séries rapides atteignent le pic, aucune vérité indépendante).
Correction de lecture : aucune (le bit est déjà lu). Fichiers visés : `internal/grammar/commandes_de_controle.go` (publier le
bit de valeur 1) et `replay/document_stances.go` (le commentaire « jamais lus sur les versions récentes » est vrai des
composants, faux de la vue de contrôle).

## C.2 Bouton de saut

Bit de valeur 4 (indice 2). **Désaccord tranché : le contributeur a raison** (valeur 4 = indice 2 = saut ; valeur 1 = indice 0 =
accroupi). Le commentaire « bit de poids 4 » de `commandes_de_controle.go` est exact si « poids 4 » désigne la valeur 4. Les autres
indices et les autres champs du bloc : 0 % de concordance avec un départ du sol. **Vérifié.**

Vérité : départ du sol = vitesse verticale qui passe de |vz| <= 0,6 à [2,5 ; 3,5] (impulsion mesurée à 3,03 au front du bit, 351
fronts) ; la hauteur confirme (+0,78 m à +0,5 s, identique avec et sans bit). Accord = départs à couverture continue (une entrée à
chaque pas sur +-100 ms) avec bit posé sur +-100 ms.

| Build | Films | Cadence des entrées | Départs | **Accord** |
|---|---|---|---|---|
| HI_1_8_0 | 1 | 60 Hz | 218 | 93,1 % |
| HI_1_9_0 | 1 (11de8353) | 30 Hz | 124 | **61,3 %** |
| HI_1_10_0 | 1 (1c4c63c2) | 60 Hz | 441 | 95,5 % |
| HI_1_10_0 | 2 (084a804d, 111fa685) | 30 Hz | 366 | **51,9 %** |
| HI_1_11_0 | 1 (e5adf7b2) | 30 Hz | 159 | **51,6 %** |
| HI_1_12_0 | 2 | 60 Hz | 591 | 98,0 % |
| HI_1_13_0 | 11 | 60 Hz | 4 172 | 98,5 % |

Tous films à 60 Hz : 5 314 / 5 422 = **98,01 %** (plus bas film : fb1a1a72, 92,7 %).

**Cause des 50 à 61 % : la cadence, pas la disposition.** Ces quatre films sont lus à une entrée toutes les 33 ms, les autres toutes les
16,7 ms (**vérifié**). À 60 Hz, 81,9 % des appuis durent un pas (sur 5 845 séries) : un appui d'un pas est vu une fois sur deux à
30 Hz, rappel attendu **59 %** (calcul) contre 47 à 61 % mesuré. Les paquets non fermés de ces builds sortent par « rejet de la
vue B » (21 693 sur 28 500 sur 084a804d, **vérifié**), pas par la vue de contrôle ; l'essai de largeurs alternatives n'a pas
reproduit la fermeture de référence : indice seulement (**déduit**).
Le bit est « bouton enfoncé », pas « saut effectué » : 34 à 78 % des séries portent un départ du sol (appui en l'air ou bloqué).

**Verdict C.2.** Tous builds : **non publiable à 90 %** (`[!]`), car HI_1_9 à HI_1_11 contiennent des films à 30 Hz plafonnés vers
59 %. **Publiable pour les films à 60 Hz** (98,0 %), si le pas des entrées est exposé en couverture ; un film à 30 Hz ne publie pas
le bouton. Correction de lecture : aucune sur la disposition ; ajouter le pas des entrées (fichiers visés :
`internal/grammar/tir_continu.go`, `replay/document_stances.go`).

## C.3 Sens des quatre booléens d'un rechargement

9 162 rechargements distincts ; chargeur lu avant pour 5 748 ; lecture après dans les 4 s.

| Booléen (rang de lecture) | À 1 | Hypothèse testée | Taux | Verdict |
|---|---|---|---|---|
| 1er (valeur 1) | 55,9 % | rechargement interrompu (chargeur n'augmente pas) | 83,6 % ; à 1 seuls 15 % aboutissent, délai médian 3,0 s contre 1,0 s, messages rapprochés (1,5 s) | **sans nom** (< 90 %) |
| 2e (valeur 2) | 99,9 % | toutes | constant | **sans nom** |
| 3e (valeur 4) | 0,0 % | toutes | constant | **sans nom** |
| 4e (valeur 8) | 11,8 % | copie ou répétition (écart 0,1 s au message précédent, gain de chargeur 1 contre 8) | non chiffré | **sans nom** |
| tous | | chargeur vide avant | non testable : 0 cas de chargeur à zéro lu avant | sans preuve |
| tous | | arme secondaire | 47 à 53 % (hasard) | **réfutée** |

Aucun booléen nommé. Le premier, à 1, marque des rechargements qui n'aboutissent pas ou répétés (**déduit**).

## En bref

- C.1 : accroupi à 95,4 % / 99,8 % d'accord physique, publiable sous réserve (vérité du plan inaccessible) ; glissade non publiable (74 %).
- C.2 : le contributeur a raison ; 98,0 % à 60 Hz mais 52 à 61 % à 30 Hz (cadence des entrées) : non publiable sur tous les builds.
- C.3 : aucun booléen nommé ; deux constants, un lié à l'échec du rechargement (83,6 %).
