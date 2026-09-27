package objectives

// rev.go — LA REVISION DE LA SORTIE DES OBJECTIFS, ET SA CHRONIQUE.
//
// # POURQUOI ELLE NAIT
//
// Jusqu au lot J3.3 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-2 (c)) une
// seule revision datait tout l arbre `facts/` : `facts.Rev`, celle que chaque ligne de kill porte
// en base. Une correction de ce paquet — le statborg, les actions d objectif, les manches, le
// drapeau — la faisait monter, donc rouvrait le backlog killsource pour une sortie que
// `killsource` ne produit pas (il n importe pas ce paquet ; mesure de l audit du 2026-09-24).
// Il y a desormais UNE REVISION PAR CONSOMMATEUR DE FAITS : [killsource.Rev] pour le kill-feed,
// celle-ci pour les objectifs.
//
// # CE QU ELLE DATE, ET CE QU UNE MONTEE COMMANDE
//
// Les calques du document qui sortent de ce paquet (`identity`, `objectives`, `scoreTimeline`, le
// drapeau, la couronne, le crane, l armement de la bombe — table `couchesDesCalques` de
// `film/replay/layers.go`) et les faits persistes, dont l en-tete la porte. Une montee les rend
// PERIMES : le verdict de recuisson dit `redecoder`, et les faits sont refuses a la relecture. Elle
// n ouvre AUCUN backlog killsource — c est tout l objet de sa naissance.
//
// # CE QUE L EMPREINTE HACHE
//
// La fermeture des imports de production du paquet (lot J3.2), figee par
// `testdata/objectives_perimetre.golden`, et la VALEUR de `source.Rev`, la seule couche revisee
// qu il importe. Ce fichier est EXCLU : il DECRIT la couche.
//
// # LA FORME, ET LA CHRONIQUE
//
// `objectives-AAAA-MM-JJ[.N]`, `N >= 2` ; le premier rang du jour s ecrit sans suffixe.
//
// ENTREE `objectives-2026-09-26` (2026-09-26, lot J3.3) : NAISSANCE DE LA REVISION DES OBJECTIFS.
// AUCUNE SORTIE NE CHANGE : ce rang pose la constante, son golden et son gate. Les calques qu elle
// date portaient jusqu ici la valeur de `facts.Rev` (`killsource-2026-09-24`) dans `layers` : ils
// portent desormais la sienne, et c est ce que la montee de schema du meme lot publie.
//
// ENTREE `objectives-2026-09-27` (2026-09-27, lot J8.5 du plan de suite d audit, constat FO-3) :
// LE PONT PAR INSTANTS DE MORT DEROULE LA SERIE PUBLIEE DU COMPTEUR DE MORTS. `deathProgressions`
// et sa version par manche appliquaient leurs propres gardes (slot de joueur, valeur dans
// [0, 1000]) ; ils deroulent desormais `SeriesTotal` / `SeriesByRound` de `DeathsComponent` — la
// manche confrontee au temps, les manches fantomes ecartees, la plus longue sous-suite non
// decroissante, la borne par pas. LA SORTIE CHANGE : l identite statborg par instants de mort
// (donc `identity.statborgSlots`, les calques d objectif, `scoreTimeline.players`) cesse de se
// taire sur un slot qu une emission aberrante noyait, et le pont PLAT d un film multi-manche (hors
// production : outils `statnames-sweep`, `zone-attribution`) voit toutes ses manches. Les faits
// persistes de la couche deviennent `redecoder` ; aucun backlog killsource ne s ouvre. Le MEME
// rang porte le lot J8.6 (constat FO-4), neutre : les deux gardes `len(kept) == 0` de
// `cumulateRounds` et de `SeriesByRound`, inatteignables, sont retirees.

// Rev est la revision de la sortie des objectifs.
const Rev = "objectives-2026-09-27"
