// Package facts PORTE LA REVISION DE LA COUCHE DES FAITS, ET RIEN D AUTRE.
//
// La couche `facts` est un ARBRE de paquets (`killsource`, `objectives`, `fallback`) : aucun
// d eux n est « la couche ». La revision, elle, en designe UNE seule — celle que chaque ligne de
// kill porte en base et qui commande le backlog de redecodage. Elle vit donc a la RACINE de
// l arbre, dans un paquet qui ne declare que cela : lui donner du code le rendrait importable
// pour autre chose, et un import de commodite finirait par ramener une dependance dans le
// paquet que tout le monde lit pour une chaine de caracteres.
package facts

// rev.go — LA REVISION DE LA COUCHE DES FAITS, ET SA CHRONIQUE.
//
// # D OU ELLE VIENT (lot 2.6.1, 2026-09-16)
//
// Elle est L HERITIERE DIRECTE de `killcollector.KillSourceDecoderRev` : meme serie, meme
// valeur, meme historique — le fichier qui suit EST celui de la constante d avant, deplace par
// `git mv`, et sa chronique n a pas ete renumerotee (decision V15 (16) du
// PLAN_DECODEUR_FILM_2026-09-13 : M2 est un jalon a ZERO difference de contenu, il n ouvre aucun
// backlog). La constante ne vit plus dans `sync/killcollector` : ce paquet la LIT et l ecrit sur
// chaque ligne produite, il ne la PORTE plus.
//
// POURQUOI CE DEPLACEMENT EST LE CORRECTIF D UN TROU, ET PAS UN RANGEMENT. Tant que la constante
// vivait hors de l arbre hache, un deplacement pur de la constante elle-meme laissait l empreinte
// verte sans regeneration (mesure du 2026-09-16) — et surtout l empreinte ne hachait que
// `killsource/`, alors que la sortie des faits depend aussi d `objectives/`, de `fallback/`, de
// la GRAMMAIRE et de la facon dont les octets sont atteints. Le defaut etait ecrit noir sur blanc
// dans l en-tete d avant (« le gate couvre le decodeur, pas son amont ») : une correction de
// grammaire qui change la sortie de `killsource` sans toucher un octet de `killsource/` ne
// faisait sonner personne, et les lignes deja en base portaient la revision courante — exclues A
// VIE du backlog.
//
// # CE QUE L EMPREINTE HACHE DESORMAIS
//
//	TOUT L ARBRE `film/facts/`   killsource, objectives, fallback ; CE fichier exclu (il DECRIT
//	                             la couche, il n en fait pas partie).
//	LA VALEUR DE `source.Rev`    la porte aux octets (V15 (12)).
//	LA VALEUR DE `grammar.Rev`   la grammaire de lecture — et elle-meme hache les valeurs de
//	                             `profile.Rev` et de `source.Rev` depuis le volet grammaire du
//	                             lot 2.6.1. Les quatre revisions se chainent : la plus basse qui
//	                             monte fait monter toutes celles du dessus.
//
// Hacher des VALEURS amont et pas leurs sources est ce qui rend la regle mecanique : une montee
// d une couche du dessous fait monter les faits sans que personne ait a y penser. C est plus
// strict qu avant, et c est le comportement voulu — un faux positif coute une ligne, un faux
// negatif coute un parc de lignes fausses en base.
//
// # CE QU UNE MONTEE COMMANDE : LE BACKLOG, ET IL PART SUR SIGNAL
//
// Chaque ligne de `match_kill_events` porte dans `decoder_rev` la revision qui l a produite. Une
// montee rend candidates au redecodage toutes les lignes portant une revision anterieure
// (`conditionBacklog`, `sync/killcollector/postsync.go`). Le redecodage du parc est un geste de
// PRODUCTION, reserve au pilote SUR SIGNAL UTILISATEUR (decision D6 du plan), JAMAIS automatique.
//
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
const Rev = "killsource-2026-09-16.6"

// L EMPREINTE DES SOURCES DE LA COUCHE VIT DANS UN GOLDEN, A COTE DE CETTE REVISION :
// `testdata/facts_rev.golden` porte le couple (revision, empreinte) avec son historique, et
// `rev_test.go` le compare aux sources NON-TEST de tout l arbre `film/facts/` et aux valeurs
// amont.
//
// POURQUOI UN GOLDEN ET PLUS UNE CONSTANTE (revue adversariale du 2026-09-12, constat P1-4).
// Tant que le test ne comparait que l EMPREINTE a une constante, remettre la revision ci-dessus a
// sa valeur d avant — en gardant la nouvelle empreinte — restait VERT : le gate ne tenait qu un
// des deux gestes qu il pretendait tenir. Le golden porte les DEUX, et le test distingue les deux
// echecs : « les sources de la couche ont change » et « la revision a change sans la couche ».
