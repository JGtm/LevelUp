package killcollector

// killsource_decoder_rev.go — LA REVISION DU DECODEUR DE SOURCE DE KILL, ET SON HISTORIQUE.
//
// # POURQUOI UN FICHIER A ELLE SEULE (revue de jalon M1, ronde 2, constat F4)
//
// DEPLACEMENT PUR depuis `collector.go` le 2026-09-16 : pas une ligne de code n a change, pas un
// commentaire n a ete reecrit. `collector.go` etait a 816 lignes — au-dessus du seuil de 500 du
// depot et en dette gelee —, et 97 de ces lignes etaient l HISTORIQUE d une constante, pas
// l enchainement telechargement / decodage / ecriture que ce fichier declare etre « TOUT ce
// qu il fait ». L historique grossit d un paragraphe a chaque lot de decodage ; le laisser la
// faisait grossir le collecteur pour une raison qui ne le concerne pas.
//
// # CE QUI N A PAS BOUGE, ET POURQUOI
//
// `metricBijAmbigue` reste dans `collector.go` : ce n est pas un bloc autonome, c est UNE ligne
// du groupe `const` des compteurs de sante, a cote des quatre autres compteurs de provenance de
// la bijection. L en sortir aurait scinde un groupe qui se lit ensemble — ce ne serait plus un
// deplacement pur.
//
// L EMPREINTE NE BOUGE PAS NON PLUS. `decoder_rev_fingerprint_test.go` hache les sources
// non-test de `internal/games/halo_infinite/film/killsource/` — jamais ce paquet-ci. Ce
// deplacement laisse donc le ratchet vert SANS regeneration du golden, et c est verifie.

// KillSourceDecoderRev — la version du decodeur, ecrite sur CHAQUE ligne produite.
//
// Elle ne sert pas a faire joli : c est elle qui permettra de savoir QUELS matchs redecoder
// apres un changement de decodage, au lieu de tout reprendre (1 325 films a 8-30 s = 3 a 11 h).
// LA FAIRE EVOLUER a chaque changement de decodage qui change les lignes produites.
//
// 2026-09-05 : `killsource-2026-07-31` -> `killsource-2026-09-05`. LE CONTRAT CI-DESSUS N AVAIT
// PAS ETE TENU : 14 commits ont touche `games/halo_infinite/film/killsource/` depuis v7.3.0 sans
// un seul bump (le seul commit qui touchait cette ligne etait un deplacement de paquet). Les
// lignes deja en base portaient donc la revision courante et etaient exclues A VIE du backlog
// (`conditionBacklog`, postsync.go) — source du degat, categorie et assistant servis avec le
// decodage d avant les vehicules. Le bump les rend a nouveau candidates.
//
// 2026-09-12 : `killsource-2026-09-05` -> `killsource-2026-09-12`. LA VERSION DU FILM EST
// DESORMAIS LUE. `loadKillFeed` passait `filmMajorVersion = 0` en dur au parseur d events, donc
// le decoupage « gamertag en tete » pour tous les films ; sur les 211 films de version 39-40 du
// cache (mars a novembre 2025) le gamertag vit 12 octets plus loin, le roster s effondrait a
// 2 noms distincts pour 24 a 27 joueurs et les portes `indice < nPlay` rejetaient les trois
// quarts des dead-states. `loadFilm` lit maintenant cette version dans l en-tete du registre du
// film (`filmdec.FilmMajorVersion`, u32 LE en tete de `chunk_00`). Couverture mesuree sur cinq
// films Big Team Battle 2025 : 15.0 -> 97.1, 11.2 -> 100.0, 6.8 -> 94.8, 5.1 -> 97.0,
// 17.3 -> 82.4 % ; temoins 2024 et 2026 inchanges au dixieme
// (.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md). Les lignes en base doivent etre redecodees :
// d ou ce bump.
//
// DECOUVERTE, NON TRAITEE : l empreinte ci-dessous ne hache que `killsource/`. Ce correctif a
// commence dans `internal/analysis/` (parseur) et dans `filmdec` — il n aurait PAS fait sonner
// le gate si `loadFilm`/`loadKillFeed` n avaient pas bouge aussi. Le gate couvre le decodeur,
// pas son amont.
//
// 2026-09-14 : `killsource-2026-09-12` -> `killsource-2026-09-14`. LA TABLE DES JOUEURS DU FILM
// DEVIENT LA SOURCE DU LIEN `indice -> joueur`, l inference le REPLI (lot 1.8,
// `killsource/film_table.go`). Les lignes produites bougent, et par deux canaux distincts :
//
//	L IDENTITE DES INDICES   la bijection etait INFEREE des votes du kill-feed (hongrois + montee
//	                         locale) ; elle est desormais LUE dans `chunk_00` partout ou la
//	                         section d identification existe. Mesure du 2026-09-14 sur 30 films et
//	                         8 builds : 314 accords sur 322 sieges, et SIX des huit ecarts sont
//	                         des joueurs que le kill-feed ne nomme pas (il ne nomme que ceux qui
//	                         tuent ou qui meurent) — l inference leur donnait le nom d un autre.
//	LA PORTE DE PUBLICATION  un film entierement lu n a plus rien d interchangeable : la marge de
//	                         bijection y est SANS OBJET et non nulle
//	                         (`Result.BijectionDetermined`). Des BTB dont les lignes etaient
//	                         refusees ligne par ligne deviennent publiables.
//
// Les lignes en base doivent etre redecodees : d ou ce bump. Le backlog lui-meme est un geste de
// PRODUCTION, reserve au pilote sur signal (decision D6 du plan du chantier).
//
// 2026-09-15 : `killsource-2026-09-14` -> `killsource-2026-09-15`. LE COUPLE (TUEUR, VICTIME)
// D UN KILL SANS MORT EN FACE EST LU AU KILL-EVENT 85, plus recolle sur le voisin (lot 1.9.3,
// `killsource/feed_couples.go`). Mesure du 2026-09-15 sur 21 films entiers (8 builds, 14 temoins
// du corpus gate) : 281 kills sans mort en face, dont 198 dont le film ECRIT le couple — et il
// ecrit EXACTEMENT celui que le recollage rendait, 198 fois sur 198, 0 contradiction. Les lignes
// bougent par deux canaux, tous deux etroits sur ce corpus :
//
//	LA VICTIME NOMMEE A LA SOURCE  quand le film nomme un BOT en victime, aucun couple n est plus
//	                               FABRIQUE sur la mort d un humain voisin (1 cas sur 21 films,
//	                               `4f77afc1`) ; la mort du voisin retourne a la population des
//	                               morts que personne ne revendique.
//	LA POPULATION DES MORTS DE BOT les couples que la lecture decide cessent d etre des candidats
//	                               a la mort de bot (mini-bobine : 17 -> 16 proposes).
//
// Les lignes en base doivent etre redecodees : d ou ce bump. Backlog = geste de PRODUCTION,
// reserve au pilote sur signal (D6).
//
// 2026-09-16 : `killsource-2026-09-15` -> `killsource-2026-09-16`. LA PORTE DE PUBLICATION LIGNE
// PAR LIGNE NE COMPTE PLUS QU UN SEUL COTE DU PROBLEME D AFFECTATION (revue de jalon M1, lentille
// L4). `BijectionDetermined` valait `Inferred <= 1` — « au plus un indice a inferer, donc une
// seule affectation possible ». La premisse est fausse DEPUIS LE LOT 1.8 lui-meme : la table du
// film ajoute au roster les joueurs que le kill-feed ne nomme pas, donc il peut rester plus de
// NOMS libres que d indices libres (`111fa685` : 25 joueurs pour 24 indices, cf. `hungarianStart`).
// Un indice pour deux noms se tranche alors par les votes du kill-feed — nuls des deux cotes pour
// un joueur qui n a ni tue ni ete tue — et rien ne le disait : `refine` n a pas deux indices a
// echanger, `bijectionMargin` rend structurellement zero, donc `BijectionDetermined` decidait
// seul et TOUT ce que la ligne porte (source du degat, credit, assistant, deux parts de degats)
// partait sur un occupant tire au sort. Le critere est desormais
// `FilmTablePinning.AffectationUnique`, qui compte les indices libres ET les noms libres.
//
// LE CHANGEMENT NE VA QUE DANS UN SENS : la porte est strictement plus fermee qu avant, donc
// aucun film ne GAGNE la publication ligne par ligne ; certains la perdent. Population non
// mesuree a l oracle `data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb` : aucune de
// ses colonnes ne porte `Inferred` ni `AddedNames` (`match_kill_events` s arrete a `publishable`),
// et sa revision la plus recente est `killsource-2026-09-05` — donc ANTERIEURE a l apparition de
// `BijectionDetermined` (2026-09-14). Le compte se mesurera a la recuisson, par
// `killsource_bijection_noms_libres_en_trop`.
// 2026-09-16, SECOND MOUVEMENT DU JOUR (fusion du lot 1.9.7) : `killsource-2026-09-16` ->
// `killsource-2026-09-16.2`. L APPARIEMENT `dead-state <-> kill-feed` SE FAIT PAR L IDENTITE DE
// PAQUET `(chunk, pidx)`, plus par une fenetre de 2,5 s (`killsource/paquet_identite.go`). Mesure du
// 2026-09-16 sur 21 films entiers : 2 899 appariements, 2 205 a identite EGALE des deux cotes, 2 a
// identite DIFFERENTE, 692 sans identite du cote feed ; l appariement par identite seule rend 2 204
// accords et ZERO desaccord — le gain est de NATURE, mais les lignes PEUVENT bouger la ou les deux
// divergent, et le decodeur choisit desormais l instant que le film ECRIT. Le suffixe `.2` : deux
// mouvements du decodeur le meme jour (la porte de publication de la revue M1, puis celui-ci),
// chacun avec son empreinte. Les lignes en base sont candidates au backlog (D6).
// 2026-09-17, LOT 2.2.a — LA REVISION NE BOUGE PAS, ET LE CHOIX EST EXPLICITE (c est ce que le
// garde-rail exige quand l empreinte change sans la revision). `killsource/` change de FORME :
// la calibration ne pose plus les deux largeurs de position dans des variables de paquet de
// `filmdec` mais les passe par `FrameConfig.Mouvement`, RESSORT son resultat
// (`calibration.Mouvement`) et le donne explicitement a `runWalk` et a `calibrateRSP` — qui en
// heritaient par effet de bord. Les LIGNES PRODUITES sont identiques a l octet : meme espace
// balaye (21 largeurs d axe x 3 largeurs d index), meme critere, meme vainqueur, memes largeurs
// pour toutes les passes qui suivent, y compris l heritage vers la cuisson du rejeu
// (`filmdec/mouvement_herite.go`). Aucun match deja decode n est candidat au backlog.
// 2026-09-17, LOT 2.3 — LA REVISION NE BOUGE TOUJOURS PAS, MEME RAISON, ET LE CHOIX EST ECRIT.
// `killsource/` change encore de FORME, plus de contenu :
//
//	`calibration.Mouvement` devient `calibration.Profil` ([filmdec.ProfilDeBalayage]) et porte
//	AUSSI le `param_4` retenu ; `resetGlobals` disparait au profit de [ProfilDeDepart], qui
//	NOMME ce que `Decode` posait dans le processus (`param_4` force a zero, generation stricte
//	levee) ; le resultat sort par [Result.ProfilCalibre], que `replaybuild` passe a
//	`replay.BuildFromFilm`.
//
// LES LIGNES PRODUITES SONT IDENTIQUES A L OCTET : meme espace balaye, meme critere, meme
// vainqueur, memes valeurs pour les passes suivantes — y compris l heritage vers la cuisson du
// rejeu, qui passe desormais par un parametre au lieu de l etat du processus. Le meme lot RETIRE
// `filmdec.LockProcessDecode` de tous ses sites d appel dans `killsource/` : un verrou ne lit
// aucun bit, et son retrait ne change pas davantage les lignes produites. Aucun match deja
// decode n est candidat au backlog.
// 2026-09-17, LOT 2.3.5 (revue adversariale) — REVISION INCHANGEE, EMPREINTE SEULE RECOPIEE.
// `killsource/decode.go` perd DEUX LIGNES DE COMMENTAIRE : la doc du verrou disparu disait
// encore « le rejeu 2D decode les memes globaux dans le meme process », ce qui contredisait
// la ligne au-dessus (doc inversee, constat P1-2 de la revue). Aucun octet n est lu
// autrement ; le choix est explicite, comme le garde-rail l exige quand l empreinte bouge
// sans la revision.
// 2026-09-18, LOT 2.4.1 — REVISION INCHANGEE, EMPREINTE SEULE RECOPIEE, ET LE CHOIX EST ECRIT
// (c est ce que ce garde-rail exige quand l empreinte bouge sans la revision).
// `killsource/` change de LECTEUR, pas de contenu : `evReader` — son type, sa boucle `bitsWide`
// et les trois primitives de position du paquet `bitAt` / `bits32` / `bitsN` — est SUPPRIME au
// profit du lecteur de bits canonique de la couche source (`filmsource.Bits`,
// `filmsource.BitsAt`, `filmsource.BitAt`). Le REFUS de lire au-dela du paquet, sans lequel une
// chaine desynchronisee lit des evenements valides apres la fin du paquet, reste entier : il
// passe du lecteur au MARCHEUR (`killsource.curseurEv`), qui teste `Remaining()` avant chaque
// lecture — `bp+n > len(pl)*8` et `Remaining() < n` sont la meme condition.
//
// LES LIGNES PRODUITES SONT IDENTIQUES A L OCTET, ET CE N EST PAS UNE DEDUCTION :
//
//	(1) appel par appel, les copies de reference des anciens lecteurs et le lecteur canonique
//	    rendent la meme valeur, la meme position de sortie et le meme drapeau sur les positions
//	    REELLES des chaines des dix bobines versionnees — 1 114 paquets a events, 109 168
//	    positions, 72 largeurs par position (`killsource/equivalence_lecteur_test.go`) ;
//	(2) de bout en bout, les triplets (code, bit de debut, bit de fin) de chaque chaine et les
//	    six champs de chaque kill-event sont identiques a un golden produit PAR LE CODE DE LA
//	    BASE, avant l absorption (`killsource/testdata/chaines_evenements.golden`).
//
// Aucun match deja decode n est candidat au backlog.
const KillSourceDecoderRev = "killsource-2026-09-16.2"

// L EMPREINTE DES SOURCES DU DECODEUR VIT DANS UN GOLDEN, A COTE DE CETTE REVISION :
// `testdata/killsource_decoder_rev.golden` porte le couple (revision, empreinte) et
// `decoder_rev_fingerprint_test.go` le compare aux sources NON-TEST de
// `internal/games/halo_infinite/film/killsource/`.
//
// POURQUOI UN GOLDEN ET PLUS UNE CONSTANTE (revue adversariale du 2026-09-12, constat P1-4).
// Tant que le test ne comparait que l EMPREINTE a une constante, remettre la revision ci-dessus a
// sa valeur d avant — en gardant la nouvelle empreinte — restait VERT : le gate ne tenait qu un
// des deux gestes qu il pretendait tenir. Le golden porte les DEUX, et le test distingue les deux
// echecs : « le decodeur a change » et « la revision a change sans le decodeur ».
