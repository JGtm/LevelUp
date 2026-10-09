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
//
// COMPLEMENT DU 2026-09-27 (lot J10.1, REVISION CONSTANTE : serie nee sur cette branche, jamais
// publiee) : le tri du pied de film (`scanTh10Events`) devient TOTAL (instant, puis position du
// XUID dans le chunk, DT-9). Deux evenements th=10 de la meme milliseconde ordonnent les actions
// publiees (numero `Seq` dense) et `captureScorer` garde le premier du plus grand instant : leur
// rang tenait au tri. Ecrit sur J10 sous `objectives-2026-09-26`, il rejoint ce rang a la fusion
// de J10 dans la branche de suite d audit (J8 l avait deja monte) ; golden regenere a revision
// constante. A la meme fusion, le tri de `roundStartsOfCompte` (ne de J8) s ecrit en `cmp.Or` :
// son comparateur etait deja total, aucune sortie ne change.
//
// COMPLEMENT DU 2026-09-28 (lot J8.7-bis, REVISION CONSTANTE) : les deux replis qui se declenchent a
// la LECTURE (`repli_emission_hors_domaine_jetee`, `repli_instant_sur_la_premiere_manche`) se notent
// dans un enregistreur par document ([ReplisALaConsultation]) passe aux lectures publiques ; le filtre
// des deux marches des series devient une fonction ([emissionHorsDomaine]). Aucune serie, aucune
// identite, aucune action ne change (`replay-equiv` sur les 20 films de reference : seul
// `coverage.fallbacks` bouge, et il est publie par `replay`) ; golden regenere a revision constante.
//
// COMPLEMENT DU 2026-09-28 (lot R1 du plan de suite d audit, constat C1 du rapport G-corpus J11,
// REVISION CONSTANTE : rang ne sur cette branche, jamais publie) : LE SIEGE RECYCLE PORTE UN LIEN PAR
// OCCUPATION (slotidentity_occupations.go). Un slot statborg dont les compteurs de base retombent a
// zero puis reprennent (un joueur part, un autre prend son siege) est decoupe en occupations, chacune
// nommee par les instants de mort de SON segment (meme regle que le pont plat) ou, pour la derniere,
// par le triplet de son segment ; une occupation que rien ne prouve reste vide et le pont s y
// abstient. LA SORTIE CHANGE, sur les seuls films a siege recycle (68 changements de siege sur 128
// films du parc) : `RoundIdentity.At` rend l occupant de l INSTANT — donc les actions d objectif et
// les portages de drapeau —, `identity.statborgSlots` publie une ligne bornee par occupation (non
// resolue quand elle n est pas nommee). `AtRound` et la courbe de score ne bougent pas, sauf
// qu aucune completion de manche entiere (triplet, elimination, residu) ne donne plus un occupant de
// siege recycle a un second slot. Mesure : 7 temoins objectifs + 11de8353 + 4f77afc1, seuls les films
// a siege recycle bougent, et seulement sur ces deux sorties. `bcb6d393` : le frag de 70 706 ms revient a son auteur (2535460750735339) au lieu du
// remplacant arrive 2 min 30 plus tard.
//
// COMPLEMENT DU 2026-10-02 (retrait des replis nuls, decision DU-7, REVISION CONSTANTE) : deux replis
// a compte NUL sur le parc sortent de la couche (`.ai/V7.5/MESURES_PARC_REPLIS_NULS_2026-10-02.md`, 1 227
// artefacts de la vague J11.4). `repli_instant_sur_la_premiere_manche` : un instant anterieur a toute
// manche connue n est plus range dans la premiere — [RoundIdentity.RoundAt] rend faux et
// [RoundIdentity.At] ne nomme personne ; `repli_mort_sans_xuid_ignoree` : la mort sans xuid reste hors
// du fil, son compte disparait. Les deux champs de [ComptesDesReplis] et la note d instant de
// [ReplisALaConsultation] partent avec eux. LA SORTIE NE CHANGE SUR AUCUN FILM DU PARC (les deux
// conditions y comptent zero) : les faits persistes et les calques dates `objectives-2026-09-27`
// restent frais, aucune recuisson. Golden regenere a revision constante.
//
// COMPLEMENT DU 2026-10-03 (lot 2.6 de la representation intermediaire, REVISION CONSTANTE) : LA
// LECTURE DU STATBORG, DU PIED DE FILM ET DES RAFALES DE CAPTURE DESCEND DANS LA GRAMMAIRE
// (`grammar/signaux`, ADR 0037 : D-2 amende). Ce paquet ne lit plus d octet : ses points d entree
// passent le film a la grammaire et portent ce qu elle rend (enregistrements, comptes des deux
// replis du statborg, constats). LE PERIMETRE CHANGE DE FORME : la couche entre desormais `grammar`
// par sa VALEUR (`amont grammar`), et une montee de `grammar.Rev` demande la decision de ce gate —
// golden regenere a revision constante quand la lecture des signaux ne change pas, montee sinon.
// AUCUNE SORTIE NE CHANGE (`replay-equiv` sur les 20 films de reference, faits et killsource
// identiques a l octet) ; golden regenere a revision constante.
//
// COMPLEMENT DU 2026-10-04 (vague 2 de la campagne de grammaire apres sa revue adverse : lots LU et
// LT, REVISION CONSTANTE) : `grammar.Rev` monte a `grammar-2026-10-03.5` (la chaine de tete d une
// liste d evenements refuse un masque ou un ordre que l ecrivain n ecrit pas ; largeur de la
// signature du localisateur derivee du cadre). `grammar/signaux` ne change pas, et l etape
// `objectives` de `replay-equiv` est IDENTIQUE sur les 20 films de reference (binaire de `6fa631df0`
// contre binaire de la tete) : la lecture des signaux ne change pas. Golden regenere a revision
// constante. Mesures : `campagne_grammaire_2026-10-01/vague2_tsv/revue/`.
//
// COMPLEMENT DU 2026-10-05 (lot VA de la campagne de grammaire, etape V1 : la vue A lue message par
// message par une seule lecture ; REVISION CONSTANTE) : `grammar.Rev` monte a `grammar-2026-10-06`.
// `grammar/signaux` ne change pas, et l etape `objectives` de `replay-equiv` est IDENTIQUE sur les 20
// films de reference (binaire de `87cdfa761` contre binaire du lot ; seule l etape `artifact`
// diverge, par sa chaine de revision). Golden regenere a revision constante. Mesures :
// `campagne_grammaire_2026-10-01/LOT_VA_V1.md`.
//
// COMPLEMENT DU 2026-10-06 (lot VA de la campagne de grammaire, etapes V2 et V3 : la fin de la vue A
// fixe le debut de la vue B, la variante de partie du film decide les genres 85 et 116 ; fusion de
// `feat/v75` a `fed1efed2` et corrections de la revue ; REVISION CONSTANTE) : `grammar.Rev` monte a
// `grammar-2026-10-06.4` (les rangs `.2` a `.4` que le lot portait sur sa seule branche, jamais
// fusionnes, y sont reunis). `grammar/signaux` ne change pas, et l etape `objectives` de
// `replay-equiv` est IDENTIQUE sur les 20 films de reference (binaire de `fed1efed2` contre binaire du
// lot). Golden regenere a revision constante. Mesures : `campagne_grammaire_2026-10-01/LOT_VA_V2.md`,
// `LOT_VA_V3.md` (§15).
//
// COMPLEMENT DU 2026-10-07 (branche `feat/zones-etat-initial`, REVISION CONSTANTE) : `grammar.Rev`
// monte a `grammar-2026-10-06.6` (voie image-cle de `grammar.ScanManagedProperties`, ti=13).
// `grammar/signaux` ne change pas et n appelle pas ce balayage : la lecture des signaux ne change
// pas. Golden regenere a revision constante.
//
// COMPLEMENT DU 2026-10-07 (branche `feat/zones-proprietaire`, REVISION CONSTANTE) : `grammar.Rev`
// monte a `grammar-2026-10-07` (nom `i0` des proprietes ti=13 lu aux images-cles).
// `grammar/signaux` ne change pas et n appelle pas ce balayage : la lecture des signaux ne change
// pas. Golden regenere a revision constante.
//
// COMPLEMENT DU 2026-10-07 (lot 2.7.c4 de la representation intermediaire, REVISION CONSTANTE) :
// `grammar.Rev` monte a `grammar-2026-10-07.2` (la vue A lit le message de kill sans sa queue ; la
// recherche des kill-events de killsource descend dans la grammaire en rattrapage). `grammar/signaux`
// ne change pas ; l etape `objectives` de `replay-equiv` est IDENTIQUE sur les 20 films de reference
// (binaires de `e9a64d87b` contre ceux du lot). Golden regenere a revision constante.
//
// COMPLEMENT DU 2026-10-08 (lot 2.7.d de la representation intermediaire, REVISION CONSTANTE) :
// `grammar.Rev` monte a `grammar-2026-10-08` (positions bipedes, pistes et creations des objets du
// monde lues par la marche des trames d abord) et le registre des replis gagne deux entrees.
// `grammar/signaux` ne change pas ; l etape `objectives` de `replay-equiv` est IDENTIQUE sur les 20
// films de reference (binaires de `acfe4851a` contre ceux du lot). Golden regenere a revision
// constante.
//
// COMPLEMENT DU 2026-10-09 (jalon LK du plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`,
// REVISION CONSTANTE) : `grammar.Rev` monte a `grammar-2026-10-09` (records d image-cle lus sous la
// portee de l etat complet). `grammar/signaux` ne change pas ; l etape `objectives` de `replay-equiv`
// est IDENTIQUE sur les 20 films de reference (binaires de `d429dc517` contre ceux du lot). Golden
// regenere a revision constante.
//
// COMPLEMENT DU 2026-10-09 (lot 2.7.d1 du plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`,
// REVISION CONSTANTE) : `grammar.Rev` monte a `grammar-2026-10-09.2` (etat complet du bipede aux
// images-cles lu par la grammaire, fenetres de bits derriere la lecture ; la marque de portage, lue
// par la grammaire sur les records admis, reste la configuration de la fenetre, U-2 (b)).
// `grammar/signaux` ne change pas ; les etapes `objectives` et `carrierMarks` de `replay-equiv` sont
// IDENTIQUES sur les 20 films de reference (binaires de `3aa885e37` contre ceux du lot). Golden
// regenere a revision constante.

// Rev est la revision de la sortie des objectifs.
const Rev = "objectives-2026-09-27"
