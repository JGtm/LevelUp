package facts

// rev_chronique.go — LA CHRONIQUE DE LA REVISION DES FAITS, ET RIEN D AUTRE.
//
// SORTIE DE `rev.go` PAR DEPLACEMENT PUR le 2026-09-21 (lot 5.7) : le fichier passait 500
// lignes au moment ou l entree `.3` s y ajoutait, et le ratchet de taille
// (`archlint/film_file_size_test.go`) refuse de grandir. AUCUN OCTET DE CODE N EST TOUCHE — le
// fichier ne porte que des commentaires, exactement comme `grammar/rev_chronique.go`, dont ce
// decoupage reprend la forme. La regle de la chronique est INCHANGEE : une entree par rang,
// jamais reecrite, et la revision se decide AVANT le golden.

// # LA CHRONIQUE — UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// ENTREE `killsource-2026-09-16.2` (2026-09-16, lot 2.6.1) : LA CONSTANTE DESCEND DANS LA COUCHE,
// SA VALEUR NE BOUGE PAS. Le rang est celui de la fusion du lot 1.9.7 (l appariement par identite
// de paquet) et il est repris TEL QUEL : rien dans le decodage n a change, il n y a rien a
// redecoder, et aucun backlog ne s ouvre. Ce qui change est le PERIMETRE de l empreinte (tout
// l arbre des faits, plus les valeurs de `source.Rev` et de `GrammarRev`) et le LIEU de la
// constante. La regle « montee de `facts.Rev` = backlog killsource » vaudra a partir du premier
// changement de SORTIE qui suivra.
//
// ENTREE `killsource-2026-09-16.3` (2026-09-16, lot 2.5.e-a) : L AMONT A BOUGE, LA SORTIE NON —
// AUCUN BACKLOG.
//
// C est la PREMIERE montee mecanique de cette revision, et elle fait exactement ce pour quoi le
// perimetre a ete elargi au lot 2.6.1 : `source.Rev` (`source-2026-09-16` -> `.2`) et
// `GrammarRev` (`.35` -> `.36`) montent avec la descente de la grammaire restee dans
// `internal/analysis` (decision V15 (4)), et comme leurs VALEURS sont hachees ici (V15 (12)),
// les faits montent SANS que personne ait eu a y penser. C est le comportement voulu.
//
// L arbre `facts/` lui-meme change d un seul geste : `killsource/feed.go` lit
// `domain/highlightevent` et `grammar.ParseHighlightEvents` en direct, la ou il passait par le
// pont transitoire `internal/analysis/highlight_event_pont_film.go` pose au lot 2.5.h. Le pont
// etait cinq renvois sans logique ; il est supprime, et le ratchet qui annoncait sa peremption
// avec lui.
//
// LA SORTIE EST IDENTIQUE A L OCTET : le lecteur des temps forts rend les memes evenements (son
// golden versionne est inchange), et aucune largeur ni aucun ordre de la couche ne bouge. AUCUN
// BACKLOG DE REDECODAGE N EST OUVERT — la regle « montee de `facts.Rev` = backlog killsource »
// vaut a partir du premier changement de SORTIE, et ce n en est pas un.
//
// ENTREE `killsource-2026-09-16.4` (2026-09-17, lot 2.5.e-c) : LA COUCHE DEMENAGE SOUS
// `film/internal/`, LA SORTIE NE BOUGE PAS.
//
// Les quatre couches passent sous `internal/games/halo_infinite/film/internal/` (ADR 0034
// principe 11 : le compilateur, et non un ratchet, refuse desormais tout import venu de
// l exterieur du decodeur). L empreinte monte par DEUX canaux, tous deux de forme : les fichiers
// de `facts/` qui importaient `film/source`, `film/profile` ou `film/grammar` citent maintenant
// `film/internal/...`, et la VALEUR amont de `GrammarRev` passe a `grammar-2026-09-15.37` pour la
// meme raison. Le cadre d empreinte hache le chemin RELATIF A LA RACINE, donc les chemins haches
// sont INCHANGES — c est l arbitrage du lot 2.6.0, pose pour que ce deplacement ne coute rien.
//
// AUCUN BACKLOG : une ligne d import par fichier ne change pas un octet de la sortie.
//
// ENTREE `killsource-2026-09-16.5` (2026-09-17, lot 2.5.e-d) : L AMONT MONTE POUR DES
// COMMENTAIRES, LA SORTIE NE BOUGE PAS.
//
// `GrammarRev` passe a `grammar-2026-09-15.38` pour trois corrections de doc inversee et une
// decision ecrite (commentaires seuls, cf. la chronique de la grammaire), et cette valeur est
// hachee ici depuis le lot 2.6.1. Rien de la couche des faits ne change. AUCUN BACKLOG.
//
// ENTREE `killsource-2026-09-16.6` (2026-09-17, lot 2.6.1, volet grammaire) : L AMONT CHANGE DE
// NOM ET DE PERIMETRE, LA SORTIE NE BOUGE PAS. AUCUN BACKLOG.
//
// La derniere des quatre couches herite du mecanisme central : `GrammarRev` devient
// `grammar.Rev`, et son empreinte cesse de hacher les octets de `source/`, de `profile/` et de
// `facts/` pour ne hacher que les siens PLUS les valeurs de `profile.Rev` et de `source.Rev`.
// Sa valeur passe donc de `grammar-2026-09-15.38` a `grammar-2026-09-15.39` — deux quantites
// differentes ne se figent pas sous la meme ligne, la raison est ecrite dans l entree `.39` de
// `grammar/rev_chronique.go`. Cette valeur est hachee ICI (V15 (12)), d ou ce rang.
//
// RIEN DE LA COUCHE DES FAITS NE CHANGE : ni un octet de `killsource/`, `objectives/` ou
// `fallback/`, ni une largeur, ni un ordre, ni une borne. Le decoupage de l empreinte de la
// grammaire ne touche aucun bit lu — il ne change QUE le diagnostic qu un gate rouge rend.
// AUCUN BACKLOG DE REDECODAGE : la regle « montee de `facts.Rev` = backlog killsource » vaut a
// partir du premier changement de SORTIE, et ce n en est pas un.
//
// # L HISTORIQUE DE LA SERIE `killsource-...`, REPRIS SANS RENUMEROTATION
//
// Ce qui suit est la chronique telle qu elle a ete ecrite rang par rang, du temps ou la constante
// s appelait `KillSourceDecoderRev` et vivait dans `sync/killcollector`. Elle n a pas ete
// reecrite : une chronique reecrite ne dit plus ce qui s est passe.
// 2026-09-05 : `killsource-2026-07-31` -> `killsource-2026-09-05`. LE CONTRAT CI-DESSUS N AVAIT
// PAS ETE TENU : 14 commits ont touche `games/halo_infinite/film/internal/facts/killsource/` depuis v7.3.0 sans
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
// film (`grammar.FilmMajorVersion`, u32 LE en tete de `chunk_00`). Couverture mesuree sur cinq
// films Big Team Battle 2025 : 15.0 -> 97.1, 11.2 -> 100.0, 6.8 -> 94.8, 5.1 -> 97.0,
// 17.3 -> 82.4 % ; temoins 2024 et 2026 inchanges au dixieme
// (.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md). Les lignes en base doivent etre redecodees :
// d ou ce bump.
//
// DECOUVERTE, NON TRAITEE : l empreinte ci-dessous ne hache que `killsource/`. Ce correctif a
// commence dans `internal/analysis/` (parseur) et dans `grammar` — il n aurait PAS fait sonner
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
// `grammar` mais les passe par `FrameConfig.Mouvement`, RESSORT son resultat
// (`calibration.Mouvement`) et le donne explicitement a `runWalk` et a `calibrateRSP` — qui en
// heritaient par effet de bord. Les LIGNES PRODUITES sont identiques a l octet : meme espace
// balaye (21 largeurs d axe x 3 largeurs d index), meme critere, meme vainqueur, memes largeurs
// pour toutes les passes qui suivent, y compris l heritage vers la cuisson du rejeu
// (`filmdec/mouvement_herite.go`). Aucun match deja decode n est candidat au backlog.
// 2026-09-17, LOT 2.3 — LA REVISION NE BOUGE TOUJOURS PAS, MEME RAISON, ET LE CHOIX EST ECRIT.
// `killsource/` change encore de FORME, plus de contenu :
//
//	`calibration.Mouvement` devient `calibration.Profil` ([grammar.ProfilDeBalayage]) et porte
//	AUSSI le `param_4` retenu ; `resetGlobals` disparait au profit de [ProfilDeDepart], qui
//	NOMME ce que `Decode` posait dans le processus (`param_4` force a zero, generation stricte
//	levee) ; le resultat sort par [Result.ProfilCalibre], que `replaybuild` passe a
//	`replay.BuildFromFilm`.
//
// LES LIGNES PRODUITES SONT IDENTIQUES A L OCTET : meme espace balaye, meme critere, meme
// vainqueur, memes valeurs pour les passes suivantes — y compris l heritage vers la cuisson du
// rejeu, qui passe desormais par un parametre au lieu de l etat du processus. Le meme lot RETIRE
// `grammar.LockProcessDecode` de tous ses sites d appel dans `killsource/` : un verrou ne lit
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
// profit du lecteur de bits canonique de la couche source (`source.Bits`,
// `source.BitsAt`, `source.BitAt`). Le REFUS de lire au-dela du paquet, sans lequel une
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
// 2026-09-18, LOT 2.4.2 — REVISION INCHANGEE, EMPREINTE SEULE RECOPIEE, ET LE CHOIX EST ECRIT.
// `killsource/` cesse de garder une COPIE des chunks du film : le type `film` portait
// `chunks [][]byte`, il porte desormais le `*source.Film` lui-meme et lit par
// `src.Chunk(i)`. MEME tranche d octets — `source.Film.Chunk` rend la tranche interne, sans
// copie, et c est deja elle que la copie recopiait. `walk.go` construit son lecteur par
// `grammar.LecteurSur` (l ancien `NewBitReader`, renomme parce que le type ne lit plus, il
// decore). Aucune largeur, aucun ordre de bits, aucune borne ne change : les lignes produites
// sont identiques a l octet, et aucun match deja decode n est candidat au backlog.
// 2026-09-17, LOT 2.6.2 (volet grammaire / rejeu) — REVISION INCHANGEE, EMPREINTE SEULE
// RECOPIEE, ET LE CHOIX EST ECRIT. Les DEUX alias dates de l arbre des faits
// (`killsource/types_alias.go`, `objectives/types_alias.go`, poses au volet facts du meme lot)
// sont SUPPRIMES : les onze types de contrat se nomment `types.X` partout, y compris dans la
// couche qui les produit. Un alias de type est LE MEME type pour le compilateur — aucune lecture,
// aucune largeur, aucun appariement ne change, les lignes produites sont identiques a l octet et
// aucun match deja decode n est candidat au backlog. Le critere de retrait ecrit dans les deux
// fichiers supprimes est tenu, et il etait mesurable au `grep`.
// ENTREE `killsource-2026-09-16.7` (2026-09-17, lot 3.3.1) : LA REVISION MONTE DERRIERE LA
// GRAMMAIRE, ET LA SORTIE DU DECODEUR CHANGE — LES LANCERS DE GRENADE DES BUILDS ANCIENS.
// Aucun octet de `facts/` n est ecrit autrement ; ce qui monte
// est la VALEUR de `grammar.Rev` (elle-meme derriere `profile.Rev`), que cette revision hache —
// le sens unique des quatre couches joue exactement comme il est ecrit.
//
// CE QUI A CHANGE EN AMONT : l amorce du record de creation de projectile devient une donnee de
// PROFIL, keyee par les neuf clefs du depot (sept builds, deux versions majeures sans section
// d identification). Le balayage comparait 24 bits sur TOUS les films ; sur les builds anterieurs
// a `HI_1_12_0` son vingt-quatrieme bit est le bit de poids fort de l identifiant, d ou ZERO
// lancer publie sur cinq temoins du corpus. Mesure du volet recherche et de ce lot : 1 282
// lancers sur ces cinq temoins, plus `a521164d` (105), `11de8353` (145) et `50247b26` (95).
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE. Les lignes de
// `match_kill_events` deja en base portent la revision anterieure et deviennent CANDIDATES au
// redecodage (`conditionBacklog`, `sync/killcollector/postsync.go`) ; le redecodage du parc reste
// un geste de PRODUCTION, pris par le pilote. Les lancers de grenade ne sont pas des kill-events
// — ce qui change reellement pour `killsource` est la revision qu il estampille, pas ses lignes.
//
// `SchemaVersion` NE MONTE PAS : la FORME du document de rejeu est inchangee (aucun champ ajoute,
// la couverture du balayage est journalisee et ne voyage pas dans l artefact). Ce qui change est
// le CONTENU des artefacts des films anciens, et c est ce que le corpus gate mesure.
// ENTREE `killsource-2026-09-17` (2026-09-17, lot 3.4.1) : LA REVISION MONTE, `.7` -> le rang
// du jour.
// LA SORTIE DES FAITS CHANGE, ET C EST LE BUT DU LOT.
//
// TROIS CAUSES, chacune mesurable :
//
//	LA GRAMMAIRE   `grammar.Rev` passe au `.40` (le chemin absolu d i0 lit les largeurs et les
//	               bornes de la table PAR INDEX de la carte au lieu d une largeur UNIFORME de
//	               14 bits ; correctif D1 (3.4) sur la regle d emission). `facts.Rev` hache sa
//	               VALEUR : elle monterait meme si rien de `facts/` n avait bouge.
//	LA CALIBRATION `killsource/calibrate.go` : l inference des largeurs NE DECIDE PLUS. Les
//	               largeurs viennent du profil — c est-a-dire du catalogue de la carte, dont la
//	               loi est verifiee 79 cartes sur 79 — et le balayage devient un ORACLE qui
//	               compte les desaccords (`calibration.Desaccords`, publie dans
//	               `Result.Calibration`, qui ne sort pas de la CLI). Arbitrage utilisateur V17,
//	               M3-Q8 : « la valeur LUE prime sur la valeur mesuree ».
//	LA CARTE      `killsource.Decode` recoit desormais l ENTREE DE CATALOGUE de la carte du
//	              match (`Options.Carte`), depuis `replaybuild.BuildBytes` et depuis
//	              `sync/killcollector`. Elle DECIDE les largeurs d axe du chemin absolu de
//	              position, la ou ce paquet etait le seul chemin de decodage du depot a ne
//	              recevoir aucun catalogue et a devoir les inferer. Mesure sur `e5adf7b2`
//	              (Fragmentation, 17/17/15) : la voie MARCHE passe de 13 lignes appariees sur
//	              16 a 167 sur 169, la voie SCAN de 176 a 22, et les 191 morts publiees sur
//	              197 couples reels sont les MEMES des deux cotes — zero perte. Sans carte, le
//	              repli `repli_carte_absente_largeurs_par_defaut` est pose, compte et AVERTI
//	              par film : le decodeur lit alors les largeurs d UNE autre carte, et le dit.
//
// LES LIGNES DE KILL DEJA EN BASE DEVIENNENT CANDIDATES AU BACKLOG DE REDECODAGE, et ce
// backlog part sur SIGNAL UTILISATEUR (D6), JAMAIS automatiquement : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote.
// ENTREE `killsource-2026-09-17.2` (2026-09-17, lot 3.4.2) : LA REVISION MONTE PARCE QUE LE
// BALAYAGE CESSE DE DECIDER CE QU IL NE MESURE PAS.
//
// LA SORTIE DES FAITS CHANGE, et le changement est un RETRAIT DE BRUIT, pas un gain de lecture.
//
// CE QUE `replay-equiv` A MESURE (20 films, sans `-update`, §5 du plan) : le lot 3.4.1 faisait
// bouger `abilityImpulses` sur 7 films, `grappleReads.stats` sur 9 et `pads` sur 3 — 19 ecarts
// hors liste sur 14 films, dont QUATRE lectures publiees perdues. Aucune de ces etapes n aurait
// du bouger.
//
// LA CAUSE, INSTRUITE SUR DEUX FILMS EN LECTURE SEULE. `infererLargeurs` balayait ENSEMBLE la
// largeur d axe et la largeur du mot de poignee, et retenait le COUPLE de meilleur score sous
// une largeur d axe UNIFORME — celle que la production a CESSE de lire au lot 3.4.1, quand les
// largeurs sont passees au triplet de la carte. Le `iw` retenu etait donc l argmax dans un monde
// que le decodeur n habite plus. Et le garde-fou ne pouvait pas le voir : `flatRatio` teste la
// nettete de la largeur d AXE, puis le code prenait le `iw` du MEME gagnant sans verifier qu il
// fut discrimine — une seule mesure, deux grandeurs, un seul garde.
//
//	a521164d  au TRIPLET LU [17 17 15]   iw=1 272 · iw=2 272 · iw=3 272   AVEUGLE
//	          sous l UNIFORME            iw=1 226 · iw=2 226 · iw=3 230   4 records sur 226
//	64e8adfa  au TRIPLET LU [15 15 15]   61 · 61 · 61                     EGALITE PARFAITE
//
// Le critere est AVEUGLE a la grandeur qu il decidait : la valeur publiee roulait sur un ex aequo
// tranche par un `sort.Slice` INSTABLE, et elle voyageait jusqu au rejeu
// (`profilDeBalayageDeLaCuisson`, `replaybuild/kills.go`) ou tous les lecteurs derriere i0 en
// heritaient.
//
// CE QUE CE LOT FAIT : DEUX MESURES SEPAREES. L oracle d axe balaie ses 21 largeurs a mot de
// poignee FIGE et n ecrit rien ; la decision du mot de poignee score ses 3 candidats AU TRIPLET
// LU et ne les retient QUE s ils dominent la mediane d un facteur `flatRatio`. Sur les deux
// films instruits la mesure ne discrimine pas : l invariant 1 tient, sous le repli
// `repli_largeur_mot_de_poignee_inferee` re-motive au registre. Les deux tris sont rendus
// DETERMINISTES (score decroissant, puis largeur croissante).
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote.
//
// `SchemaVersion` NE MONTE PAS : aucun champ n est ajoute au document. `grammar.Rev` et
// `profile.Rev` NE MONTENT PAS : aucun octet de ces deux couches n est touche.
// ENTREE `killsource-2026-09-18` (2026-09-18, lot 5.1.1) : LA REVISION MONTE MECANIQUEMENT,
// `.2` -> le premier rang du 18. AUCUNE SOURCE DE `film/facts/` N EST TOUCHEE PAR CE LOT.
//
// CE QUI LA FAIT MONTER : l empreinte de cette couche hache les VALEURS de `source.Rev` et de
// `grammar.Rev`, et `grammar.Rev` monte au lot 5.1.1 (`grammar-2026-09-18` : l archetype
// `managed-navpoint` ti=12 est lu de `i1` au minuteur manuel, douze lecteurs neufs). La chaine
// est voulue : une grammaire qui change date les lignes deja decodees, meme quand le fait
// publie ne bouge pas encore.
//
// CE QUE LA SORTIE FAIT AUJOURD HUI : rien de plus. Aucun composant porte par 5.1.1 n alimente
// `killsource` — les douze lecteurs servent `ti=12`, que la chaine des morts ne marche pas.
// LE BACKLOG QU ELLE OUVRE EST DONC UN BACKLOG DE DATATION, pas de correction.
//
// C EST L UNIQUE MONTEE DE CETTE CONSTANTE POUR TOUT LE LOT 5.1, ET C EST DELIBERE : le volet
// 5.1.4 (l attribution de la fin de vie des vehicules) CHANGERA vraiment la sortie des faits, et
// il partagera ce rang — deux changements d un meme lot partagent la revision. Ouvrir deux
// backlogs pour un seul lot ferait redecoder le parc deux fois.
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote. UN BACKFILL
// `killsource-2026-09-17.2` TOURNAIT AU MOMENT DE CE LOT : la montee le rend candidat a son
// tour, ce que l utilisateur a accepte en ouvrant le lot (V26).
//
// `SchemaVersion` reste 62 ; `profile.Rev` ne monte pas (aucun octet de `profile/` touche).
// ENTREE `killsource-2026-09-20` (2026-09-20, lot 5.2b.1) : LE ROSTER DU DECODEUR VOIT LES
// REMPLACANTS, ET UN PARTICIPANT NON COMPTE N ETEINT PLUS LE MATCH.
//
// DEUX SOURCES POUR CETTE MONTEE, et elles vont dans le meme sens.
//
//	`facts/killsource/` CHANGE      une TROISIEME lecture d identite entre dans le roster
//	                                (`index_motif.go`) : les cinq bits qui precedent le motif du
//	                                xuid dans les chunks de replication, c est-a-dire ce que le
//	                                rejeu publie sous le nom `PlayerIndexTable`, par le MEME
//	                                resolveur (`weaponv3.ResolveXuidToPI`). Elle voit les joueurs
//	                                qui REMPLACENT un partant en cours de match, que la table de
//	                                `chunk_00` — ecrite a l ouverture du film — ignore.
//	`grammar.Rev` MONTE             `grammar-2026-09-20`, et cette couche hache sa valeur.
//
// CE QUE LA MESURE DIT, SUR `b1ad85eb` (Domicile, HI_1_13_0, 2026-09-20) : la table de
// `chunk_00` nomme HUIT sieges (0..7), BOT_METADATA tient le 8, et le kill-feed nomme un
// NEUVIEME humain — `Claudors` — que rien ne pouvait placer. Le motif du xuid le lit a l indice
// 10, UNANIME sur 22 chunks de replication sur 27, et il CONFIRME les huit sieges de la table
// (`MotifAgree = 8`, zero contradiction). Les huit dead-states hors roster disparaissent, la
// publication ligne par ligne s ouvre, 77 lignes sortent dont 63 a source NOMMEE.
//
// LA TABLE DE `chunk_00` GARDE LA MAIN quand les deux lectures se contredisent : elle est la
// plus eprouvee (314 accords sur 322 sieges, 30 films). Une contradiction se COMPTE
// (`FilmTablePinning.MotifContradict`), elle ne deplace rien — meme doctrine que le controle par
// les votes du kill-feed (D14 b).
//
// TROISIEME CHANGEMENT DE SORTIE, MESURE AU MEME ENDROIT : plusieurs bots declares sur un MEME
// slot ajoutaient chacun un nom au roster pour un seul indice, et les perdants restaient des
// NOMS LIBRES — de la matiere a inference. Deux noms de bot fantomes suffisaient a rendre
// `FilmTablePinning.AffectationUnique` faux des qu un indice se liberait, donc a refermer la
// publication que l epinglage du remplacant venait d ouvrir. Le vainqueur du slot ne change pas
// (le dernier declare) ; la succession REMPLACE le nom en place et se compte
// (`Roster.BotsSuccedes`).
//
// BACKLOG KILLSOURCE SUR SIGNAL UTILISATEUR (D6), JAMAIS AUTOMATIQUE : chaque ligne de
// `match_kill_events` porte cette revision dans `decoder_rev`, `conditionBacklog`
// (`sync/killcollector/postsync.go`) rend candidate toute ligne qui en porte une anterieure, et
// le redecodage du parc reste un geste de PRODUCTION pris par le pilote. CELUI-CI EST UN
// BACKLOG DE CORRECTION, pas de datation : les matchs a remplacement changent de verdict de
// publication.
//
// `SchemaVersion` NE MONTE PAS : aucun champ n est ajoute au document.

// ENTREE `killsource-2026-09-21` (2026-09-21, lot 5.3.3-a) : LA REVISION MONTE MECANIQUEMENT
// DERRIERE LA GRAMMAIRE — `i60` EST DECLARE COMPLET QUAND LA CARTE EST LA.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21` :
// `SimStateComplet` ne se pose plus a la main, il SUIT les largeurs d axe de la carte du match
// (chronique de `grammar`, entree du meme jour). La traversee du bipede va donc plus loin sur
// tout film dont la carte est cataloguee — 38 desynchronisations d `i60` en moins sur le seul
// `bfecd02b`. Cette constante hache la VALEUR de la revision de grammaire : elle monte
// mecaniquement, et les lignes de `match_kill_events` anterieures deviennent candidates au
// backlog de redecodage (D6, SUR SIGNAL UTILISATEUR, jamais automatiquement).
//
// CE QUE CE BACKLOG RAPPORTERAIT, MESURE AVANT DE L OUVRIR : RIEN. A/B par `replay-build` sur
// `000d5950` et `bcb6d393`, bascule levee puis abaissee, cache de faits vide a chaque passe :
// artefact BIT A BIT IDENTIQUE. Le `replay-equiv` du meme film ne deplace que le digest de
// l etape `killsource`, et ce digest porte la VALEUR du profil calibre — compte et octets du
// kill-feed inchanges. Le pilote n a donc aucune raison de declencher ce backlog pour cette
// revision-ci.
//
// `SchemaVersion` NE MONTE PAS : aucun champ neuf au document, et aucun octet cuit ne change.

// ENTREE `killsource-2026-09-21.2` (2026-09-21, lot 5.3.6) : LA REVISION MONTE DERRIERE UN
// BALAYAGE NEUF DE LA COUCHE GRAMMAIRE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.2` : la couche
// rend une valeur de plus — les ETATS DE MOUVEMENT du Spartan (accroupi, glissade, action de
// mobilite), publies en `stances[]` au schema 65. Cette constante hache la VALEUR de la revision
// de grammaire : elle monte, et les lignes de `match_kill_events` anterieures deviennent
// candidates au backlog de redecodage (D6, SUR SIGNAL UTILISATEUR, jamais automatiquement).
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED : rien de neuf. Le calque des etats est
// ADDITIF — un balayage de plus, sur un canal que le kill-feed ne lit pas. C est le CONTENU CUIT
// qui change (`SchemaVersion` 64 -> 65), et c est `backfill-replay` qui le re-cuit, pas ce
// backlog-ci.

// ENTREE `killsource-2026-09-21.3` (2026-09-21, lot 5.7) : LA REVISION MONTE DERRIERE UNE
// CORRECTION DE LARGEUR DE LA COUCHE GRAMMAIRE, ET CELLE-CI DEPLACE DES BITS.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.3` : les
// QUATRE CHARGES de `ti=35 i55 biped-posture-physics-component` sont portees (la glose « 0 bit
// lu » de son repartiteur `FUN_141fd997c` etait fausse : c est l union discriminee de l etat
// physique du bipede, de 15 a plus de cent bits selon le tag). Cette constante hache la VALEUR
// de la revision de grammaire : elle monte.
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED, ET C EST A PRENDRE AU SERIEUX CETTE FOIS :
// contrairement aux deux entrees precedentes, LES RECORDS DU BIPEDE NE FERMENT PLUS AUX MEMES
// BITS. Sur `bfecd02b` la marche rend 97 345 records `ti=35` au lieu de 97 447 et 6
// desynchronisations au lieu de 3, a etalon de contenu INCHANGE (`i0` 85,5 %, `i1` 77,5 %,
// `i21` 65,3 %, `i25` 97,1 %). Un ecart de un pour mille sur la population de records peut
// deplacer une attribution de kill. LE REDECODAGE RESTE UN GESTE DE PRODUCTION SUR SIGNAL
// UTILISATEUR (D6), jamais automatique — mais pour cette revision-ci il n est PAS sans objet, et
// c est dit.
//
// `SchemaVersion` NE MONTE PAS : aucun champ neuf au document. Le CONTENU CUIT change, lui, et
// ce sont les fixtures de contrat et `backfill-replay` qui en repondent.

// ENTREE `killsource-2026-09-21.4` (2026-09-21, lot 5.7.4) : LA REVISION MONTE DERRIERE UNE
// CORRECTION DE PUBLICATION DE LA COUCHE GRAMMAIRE, ET LE KILL-FEED N EST PAS CONCERNE.
//
// AUCUN OCTET DE `facts/` N EST TOUCHE. `grammar.Rev` passe a `grammar-2026-09-21.4` : la porte
// des etats de mouvement s inscrit dans la neutralisation des lectures speculatives, qu elle
// avait oubliee au lot 5.3.6 — elle publiait les essais d alignement de la marche, dans un
// rapport de 14 a 152 pour un (chronique de `grammar`, entree du meme jour). Cette constante
// hache la VALEUR de la revision de grammaire : elle monte.
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED : RIEN, et c est prouve cette fois-ci plutot
// qu argumente. Le correctif n eteint qu UN crochet, `EtatMouvementHook` ; `PosCaptureHook` et
// `UnitRefHook` etaient DEJA inscrits dans la neutralisation depuis le lot 2.2, donc aucune
// position, aucune vitesse, aucune reference d unite ne change. `replay-equiv` sur `bcb6d393`
// le mesure : 3 etapes deplacees sur 57 — `movementStates`, `movementStates.stats` et
// `artifact` —, et `killsource` en fait partie des 54 IDENTIQUES a l octet.
//
// `SchemaVersion` NE MONTE PAS : la FORME du document ne change pas. Son CONTENU change
// (`stances[]` perd les intervalles qu il tenait d essais jetes), et c est `backfill-replay` qui
// en repond.
