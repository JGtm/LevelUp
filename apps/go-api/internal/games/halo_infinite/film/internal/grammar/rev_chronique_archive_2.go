package grammar

// rev_chronique_archive_2.go — LA CHRONIQUE DE [Rev], RANGS `.29` A `.38`.
//
// # POURQUOI UNE SECONDE ARCHIVE (2026-09-18, lot 5.1.7)
//
// La chronique ne peut que grandir : un lot, un rang, une entree. `rev_chronique.go` a atteint
// 500 lignes une seconde fois, et `rev_chronique_archive.go` — qui porte les rangs `.12` a `.28`
// — en est a 456 : y verser dix rangs de plus l aurait fait depasser le meme seuil. LA ROTATION
// SE FAIT DONC EN CHAINE, comme pour `.ai/thought_log.md` : chaque archive garde le bloc de rangs
// qu elle nomme dans son titre, et la suite VIVANTE reste dans `rev_chronique.go`. Le geste est
// ordinaire ; il se refera, et le fichier suivant s appellera `_3`.
//
// L ORDRE DE LECTURE EST CELUI DE `fichiersDeChroniqueGrammar` (`rev_test.go`) : archive,
// archive_2, puis la chronique vivante. Le gate exige que les rangs s y suivent sans trou.
//
// Comme `rev.go`, `rev_chronique.go` et la premiere archive, ce fichier est EXCLU de l ensemble
// hache par l empreinte (`fichiersHorsGrammaire`) : il DECRIT la grammaire, il n en fait pas
// partie.
//
// ENTREE `grammar-2026-09-15.29` (2026-09-18, lot 2.4.2 — RANG PROVISOIRE) : `.28` -> `.29`.
// AUCUN OCTET N EST LU AUTREMENT. LA PORTE AUX OCTETS EST UNIQUE.
//
// CE QUI DESCEND DANS LA COUCHE SOURCE (`internal/analysis/filmsource`, decision V15 (1) ;
// 2.5.a la deplacera sous `film/internal/source` par `git mv` pur) :
//
//	les quatre conventions de bord, NOMMEES    `BitsAt` (bourrage a zero du moteur), `BitAt`
//	                                           (zero des deux cotes), `BitsTolerants` (un
//	                                           lecteur qui RECULE devant un motif), `BitsTronques`
//	                                           (s arrete sans bourrer — le pied de film). Elles ne
//	                                           se fondent pas : chacune est la convention MESUREE
//	                                           d un lecteur reel, et les fondre changerait des
//	                                           valeurs decodees (D4).
//	les entiers du film                        `U16LE` / `U32LE` / `U64LE`, plus `OctetAuBit` et
//	                                           `U64LEAuBit` (offset en BITS).
//	LE marcheur de paquets                     `Paquets`. Les QUATRE copies de l en-tete de seize
//	                                           octets n en sont plus que des traductions.
//	LE decompresseur                           `Inflate` (tolerant) et `Decompresser` (strict).
//	le balayage de motif de 64 bits            `ChercherMotif64`.
//
// CE QUI CHANGE DE NOM PARCE QUE SA NATURE A CHANGE : `grammar.BitReader` / `NewBitReader`
// deviennent `Lecteur` / `LecteurSur`. Le type ne LIT plus, il DECORE — il embarque
// `*source.Bits` et n ajoute que la grammaire (profil de largeurs, capture de position,
// observateur, `ReadSignedVarWidth`). Les deux anciens noms restent nommes dans
// `archlint/no_raw_film_bytes_outside_source_test.go` : ratchet anti-resurrection.
//
// LES SIX PAQUETS QUI TRAVERSENT LA FACADE : `filmdec` (les 48 constructions de lecteur, les
// quatre sections de `chunk_00`, le second marcheur de paquets), `killsource` (le film porte
// desormais `*source.Film` et non une COPIE de ses chunks), `analysis/objectiveevents`
// (`film.go` et `statborg.go` : les trois lecteurs du pied de film disparaissent),
// `analysis/weaponv3` (`bits_word.go` — la copie DIVERGENTE de la lecture par mot — est
// supprime, `pi_resolver.go` n a plus de type `bitReader`, `timing.go` n a plus de marcheur),
// `sync/haloclient` et cinq outils `cmd/`.
//
// LES TEMOINS SONT VIVANTS, PAS TAUTOLOGIQUES. Le temoin des deux marcheurs de paquets compare
// desormais le marcheur unique a une COPIE DE REFERENCE de l ancienne grammaire de
// `grammar.WalkPackets`, sur tous les chunks de la mini-bobine (738 paquets) — il mesure donc
// vraiment les deux regles qui les separaient (arret apres CHUNK_END, refus d un en-tete
// degenere). Le differentiel du resolveur xuid -> player_index reste dans `weaponv3`, positions
// NEGATIVES comprises, et vise desormais les primitives de la source.
//
// `facts.Rev` ne bouge PAS : la SORTIE de `killsource` est identique a l octet.
// `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.30` (2026-09-16, lot 2.5.d.2 — RANG PROVISOIRE) : `.29` -> `.30`.
// DEPLACEMENT PUR, SORTIE IDENTIQUE.
//
// `internal/analysis/objectiveevents` descend sous
// `internal/games/halo_infinite/film/internal/facts/objectives` : c est la couche `facts` de l ADR 0034
// D-1 qui rentre DANS le decodeur. Le paquet change de nom (`objectiveevents` -> `objectives`)
// et de chemin, RIEN D AUTRE — aucune ligne de corps n est ajoutee ni retiree, seuls la clause
// `package`, les imports et les mentions du nom en commentaire bougent (`git diff -M` ne montre
// que des renommages).
//
// L EMPREINTE MONTE PARCE QU ELLE HACHE DES OCTETS DE SOURCE, et c est ecrit dans son en-tete :
// la clause `package` de 27 fichiers de production a change. AUCUN OCTET DE FILM N EST LU
// AUTREMENT — la grammaire du pied de film (`scanTh10Events`, `decodeTh10Block`) est identique a
// l octet, seul son LIEU a change. `racinesGrammaire` (rev_test.go) et le
// cadre herite (`film/revision/equivalence_test.go`) suivent la racine.
//
// L ORDRE DES COMMITS DE 2.5 CHANGE, ET LA MESURE LE DIT. La note de preparation (§2.7) faisait
// descendre `filmsource` en PREMIER. Mesure du 2026-09-16 : ce premier pas fait rougir D9
// (`no_title_package_in_analysis_test.go`) sur DIX-SEPT fichiers, parce que `objectiveevents` et
// `weaponv3` vivent sous `internal/analysis/` et importent `filmsource` — la facade de 2.4 etait
// nee la precisement pour l eviter (V15 (1)). Les paquets de couche quittent donc
// `internal/analysis/` AVANT la couche `source`. Deux entrees de l allowlist D9 tombent ici, par
// le deplacement lui-meme.
//
// `facts.Rev` ne bouge PAS : la SORTIE de `killsource` est identique a l octet (seul
// un commentaire y nomme desormais `objectives`). `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.31` (2026-09-16, lot 2.5.c — RANG PROVISOIRE) : `.30` -> `.31`.
// DEPLACEMENT PUR, SORTIE IDENTIQUE.
//
// LA COUCHE `grammar` PREND SON NOM. `internal/games/halo_infinite/film/filmdec` devient
// `internal/games/halo_infinite/film/internal/grammar` (554 fichiers, dont 142 de production), et
// `internal/analysis/weaponv3` descend sous `film/grammar/weaponv3` (7 fichiers). Le paquet
// change de nom (`filmdec` -> `grammar`) et de chemin, RIEN D AUTRE : aucune ligne de corps n est
// ajoutee ni retiree.
//
// POURQUOI `weaponv3` BOUGE MAINTENANT, ET PAS A SA DISSOLUTION. Il vivait sous
// `internal/analysis/` et importe la couche `source` : tant qu il y reste, faire descendre
// `filmsource` sous `film/` fait rougir D9. Sa DISSOLUTION (le resolveur dans le corps de
// `grammar`, le catalogue d armes en `games/weapons`) n est PAS un deplacement pur et reste a
// faire — l arete `weaponv3 -> analysis` est toujours dans l allowlist des couches, datee, avec
// son lot.
//
// L EMPREINTE MONTE PARCE QU ELLE HACHE LE CHEMIN RELATIF A COTE DU CONTENU, et c est ecrit dans
// son en-tete : la clause `package` de 142 fichiers a change, et `weaponv3/` entre dans l arbre
// hache. AUCUN OCTET DE FILM N EST LU AUTREMENT.
//
// CE QUI NE CHANGE PAS DE NOM, ET C EST DELIBERE : les compteurs expvar
// (`filmdec_unknown_build_*`, `filmdec_unknown_format_*`, `filmdec_keyframe_ti*`) et les prefixes
// d erreur `"filmdec: ..."`. Un compteur est un CONTRAT D EXPLOITATION (ADR 0009) ; le renommer
// serait un changement de sortie, exactement ce qu un deplacement pur s interdit. La tranche
// `"filmdec"` du registre des replis (`facts/fallback/registre.go`) est une CLE publiee du meme
// ordre. Leur renommage se decidera avec les consommateurs, pas ici.
//
// `facts.Rev` ne bouge PAS : la SORTIE de `killsource` est identique a l octet.
// `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.32` (2026-09-16, lot 2.5.a — RANG PROVISOIRE) : `.31` -> `.32`.
// DEPLACEMENT PUR, SORTIE IDENTIQUE.
//
// LA COUCHE `source` RENTRE DANS LE DECODEUR. `internal/analysis/filmsource` devient
// `internal/games/halo_infinite/film/internal/source` : le paquet change de nom (`filmsource` ->
// `source`) et de chemin, RIEN D AUTRE. Il reste ce qu il etait — une FEUILLE sans aucun import
// du depot, tenue par `archlint/filmsource_leaf_test.go`, dont la constante suit le chemin.
//
// CE QUE CE COMMIT FERME. Les SIX dernieres aretes de lieu du ratchet des couches tombent d un
// coup (`filmcache`, `grammar`, `killsource`, `replay`, `facts/objectives`, `grammar/weaponv3`
// vers `filmsource`), et la table `paquetsHorsLieuToleres` se VIDE : plus aucun paquet de couche
// ne vit hors de `film/`. Il ne reste que TROIS aretes, toutes vers `internal/analysis` RACINE,
// toutes datees « 2.5.c » : ce sont des descentes de SYMBOLES, pas des deplacements de paquet.
//
// CE QUE CE COMMIT NE FAIT PAS, ET LA MESURE LE DIT. La note de preparation §2.3 faisait remonter
// QUATORZE fichiers de `filmdec` dans la couche `source` (le lecteur canonique, les quatre
// sections de `chunk_00`, le registre, la table des joueurs). Ce n est PAS un deplacement pur :
// 76 de leurs 159 declarations sont referencees par le reste du paquet (`ReadFilmChunk` 293 fois
// dans 179 fichiers, `Lecteur` 344 fois, `WalkPackets` 217, `Archetype` 177, `Registry` 158), et
// `lecteur.go` lit `profile.Profile`, `Observation`, `FrameConfig`, `ProfilDeBalayage` — c est-a-dire que
// `source` importerait `profile` et `grammar`, deux imports VERS LE HAUT. La couche `source` du
// lot est donc `filmsource` seul ; la remontee reste a faire, et elle n est pas un `git mv`.
//
// L EMPREINTE MONTE parce que la clause `package` de cinq fichiers de production a change et que
// la racine hachee suit le paquet. AUCUN OCTET DE FILM N EST LU AUTREMENT.
// `facts.Rev` ne bouge PAS ; `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.33` (2026-09-16, lot 2.5.d.1 — RANG PROVISOIRE) : `.32` -> `.33`.
// DEPLACEMENT PUR, SORTIE IDENTIQUE.
//
// LA COUCHE `facts` EST COMPLETE. `film/killsource` devient `film/facts/killsource` et
// `film/replay/fallback` devient `film/facts/fallback` (le REGISTRE des replis, ADR 0034 D-10
// bis, qui etait classe `replay` avec son parent et se classe desormais `facts`, ou il est
// produit). Les NOMS de paquet ne changent PAS — seuls les chemins bougent : le diff ne porte que
// des lignes d import.
//
// L EMPREINTE MONTE parce qu elle hache le CHEMIN RELATIF a cote du contenu et que la racine
// `killsource` a change de place. AUCUN OCTET DE FILM N EST LU AUTREMENT ; la SORTIE de
// `killsource` est identique a l octet, donc `facts.Rev` ne bouge pas (son golden est
// refige, son ancre de racine suit le paquet). `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.34` (2026-09-16, lot 2.5.b) : `.33` -> `.34`. EXTRACTION DE LA
// COUCHE `profile`, SORTIE IDENTIQUE.
//
// CE N EST PAS UN DEPLACEMENT PUR, ET C EST LA DIFFERENCE AVEC `.31` A `.33`. La couche
// `profile` (ADR 0034 D-1) NAIT par EXTRACTION avec inversion de dependance : la DONNEE descend
// dans `film/profile`, la LECTURE reste ici. Concretement, ce qui a change dans le source de
// `grammar` :
//
//	des declarations SORTENT       les types de valeur (`I0Layout`, `MPPWidths`,
//	                               `PrecisionDescriptor`, `AxisRange` / `Vec3Range` et les cinq
//	                               plages de `DAT_143b8c6f0`, `FilmIdentity`), la table de profil
//	                               et ses invariants, la table par build et par format, le
//	                               catalogue de bornes par carte.
//	des symboles PRIVES sont       ce que `grammar` continue de lire de l autre cote de la
//	EXPORTES                       frontiere l est desormais par un nom exporte, JAMAIS par une
//	                               copie (CLAUDE.md regle 6).
//	des appelants sont             environ 350 sites qualifient desormais `profile.X`. Le CORPS
//	REQUALIFIES                    des fonctions ne change pas.
//	la RESOLUTION est INVERSEE     `ResolveProfile` reste ici — elle ouvre le `chunk_00`, lit le
//	                               registre, la version de format et la section 2 — et appelle
//	                               `profile.Resoudre` avec les cles DEJA LUES. Avant, la lecture
//	                               et la table etaient dans la meme fonction ; c est cette
//	                               couture qui aurait fait importer `grammar` par `profile`.
//
// AUCUN OCTET DE FILM N EST LU AUTREMENT : aucune largeur, aucun cadre, aucun ordre de
// composants, aucune valeur de table ne change. Les corps sont deplaces sans une ligne de
// difference, les litteraux de la table de profil sont relus a l identique (le test de
// conformite du catalogue `film_profiles.json` le prouve : il lit le SOURCE de la table par
// `go/parser`, et son contenu attendu est INCHANGE).
//
// L EMPREINTE MONTE POUR DEUX RAISONS, et les deux sont voulues : les octets de source de
// `grammar` ont change (declarations sorties, qualifieurs ajoutes), et la racine `film/profile`
// ENTRE dans l empreinte. Ce second point est le vrai gain du lot pour ce garde-rail : la
// DONNEE du decodeur — les largeurs par carte, la transposition par build, le decoupage MPP par
// format — est desormais hachee, alors qu elle aurait pu bouger sans que `grammar.Rev` monte si
// la racine n avait pas suivi.
//
// `facts.Rev` ne bouge PAS : `film/facts/killsource` ne change que ses qualifieurs,
// sa sortie est identique a l octet, et son propre ratchet d empreinte fait foi.
// `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.35` (2026-09-16, lot 2.6 volet facts + source — RANG DE FUSION) :
// `.34` -> `.35`.
// REVISIONS PAR COUCHE ET TYPES DE CONTRAT, SORTIE IDENTIQUE.
//
// TROIS GESTES, ET AUCUN NE LIT UN OCTET AUTREMENT. (1) `source.Rev` NAIT (`source/rev.go`,
// golden a historique `source/testdata/source_rev.golden`) : la couche qui porte la porte aux
// octets a desormais sa revision propre, sur le mecanisme central `film/revision`. (2)
// `facts.Rev` NAIT (`facts/rev.go`) et REPREND la valeur de `facts.Rev`
// (`killsource-2026-09-16.2`, V15 (16)) : la constante quitte `sync/killcollector`, qui la LIT,
// son empreinte couvre TOUT l arbre `facts/` (killsource, objectives, fallback) plus les VALEURS
// de `source.Rev` et de cette constante-ci (V15 (12)), et le backlog de redecodage reste un
// geste de PRODUCTION sur signal utilisateur (D6). (3) `film/types` NAIT, feuille sans aucun
// import du depot : les types de DONNEES purs qui traversent une frontiere de couche depuis
// `source` et `facts` y sont declares, et les paquets d origine gardent un ALIAS DATE le temps
// que les volets grammaire et rejeu re-pointent leurs consommateurs.
//
// L EMPREINTE MONTE parce que les racines hachees portent des fichiers neufs (`rev.go`) et,
// pour le geste (3), des alias a la place des declarations. La SORTIE est identique a l octet :
// aucune largeur, aucun cadre, aucun ordre de composants ne change, et un alias de type est le
// MEME type pour le compilateur. `SchemaVersion` reste 60 ; aucun match deja decode n est
// candidat au backlog, et c est la condition meme de V15 (16) — M2 est un jalon a ZERO
// difference de contenu.
//
// RANG DE FUSION, ET IL EST DE FUSION AU SENS STRICT : ce lot avait ete pose a `.34` sur une base
// ou l integration valait `.33`, et le lot 2.5.b a pris ce rang avant lui. Les deux entrees
// restent, dans l ordre ou les merges sont tombes — renumeroter la leur serait reecrire une
// chronique deja figee dans un golden. L empreinte de ce rang inclut la racine `film/profile`,
// entree au rang precedent.
//
// ENTREE `grammar-2026-09-15.36` (2026-09-16, lot 2.5.e-a) : `.35` -> `.36`.
// LA GRAMMAIRE RESTEE DANS `internal/analysis` DESCEND ICI. SORTIE IDENTIQUE.
//
// C est la derniere descente du pas 5 (decision V15 (4)) : trois lecteurs de film vivaient
// encore dans le paquet TITLE-AGNOSTIC, et le ratchet des lectures brutes les comptait depuis
// le 2026-09-17 comme les neuf dernieres portes aux octets a fermer.
//
//	`analysis/highlight_event_parser.go`  -> `grammar/highlight_events.go`. Le decompresseur
//	                                         local devient [source.Decompresser] + la
//	                                         sentinelle `source.ErrEnTeteZlib` (meme partage
//	                                         entre « deja clair » et « flux casse ») ; la copie
//	                                         de `readByteAtBit` devient [source.OctetAuBit], a
//	                                         la convention de bord IDENTIQUE ; les deux entiers
//	                                         deviennent `source.U32BE` et `source.U16LE`.
//	`analysis/weapon_scanner.go`          -> `grammar/weaponscan/scanner.go` (sous-paquet :
//	                                         `grammar.FireEvent` designe deja un AUTRE record).
//	                                         `binary.BigEndian.PutUint64` devient
//	                                         `filmshell.BytesFromID`.
//	`analysis/positions/`                 -> `grammar/positions/`. `bitAt` devient
//	                                         [source.BitAt] (memes zeros des deux cotes), l
//	                                         en-tete de bloc `source.U16LE` / `source.U32LE`.
//	`analysis/weapon_data.go`             -> `games/weapons/filmshell`, FEUILLE hors des racines
//	                                         hachees : ce catalogue NOMME les armes, il ne lit
//	                                         aucun octet.
//
// DEUX TYPES REMONTENT EN `domain/` LE MEME JOUR, sur le modele du lot 2.5.h : le pont
// transitoire `analysis/highlight_event_pont_film.go` est SUPPRIME (les onze fichiers de `film/`
// lisent `domain/highlightevent` en direct), et `PlayerPosition` / `TeamUnknown` naissent en
// `domain/playerposition` — leurs lecteurs sont les ports, le service de vue de match, la couche
// DuckDB et un corps HTTP, qui n ont rien a savoir d un titre.
//
// DEUX CHANGEMENTS DE FORME, ET AUCUN DE SORTIE. (1) `FireEvent.PlayerIndex` / `PlayerIndex5` et
// `FormulaAResult.PlayerIndex` deviennent `FilmIndex` / `FilmIndex5` : le ratchet
// `archlint/no_player_index_identity_test.go` interdit ce nom dans le perimetre du film, SANS
// allowlist, et le champ homologue de `grammar` s appelle deja `FilmIndex`. (2)
// `decodeFireEventAt` est extraite de `ScanFireEventsB5`, qui passait le seuil de 80 lignes une
// fois descendue — corps deplace sans une ligne de logique changee.
//
// AUCUN OCTET N EST LU AUTREMENT : chaque substitution est une fonction dont la convention de
// bord a ete verifiee identique a celle qu elle remplace, et les goldens des temps forts
// (`testdata/highlight_event_golden.txt`, 275 evenements, sha256 fige au lot 2.5.h) sont
// INCHANGES. `SchemaVersion` reste 60 ; aucun match deja decode n est candidat au backlog.
//
//
// ENTREE `grammar-2026-09-15.37` (2026-09-17, lot 2.5.e-c) : `.36` -> `.37`.
// LES QUATRE COUCHES PASSENT SOUS `film/internal/`. AUCUNE LIGNE DE GRAMMAIRE NE CHANGE.
//
// C EST LE GESTE DU PRINCIPE 11 DE L ADR 0034 (« la compilation seule garantit la frontiere ») :
// `source`, `profile`, `grammar` et `facts` vivent desormais sous
// `internal/games/halo_infinite/film/internal/`, ou le COMPILATEUR — et non un ratchet — refuse
// tout import venu de l exterieur du decodeur. Les 94 fichiers consommateurs passent par la
// facade `film/decfilm` (lot 2.5.e-b). `film/replay` reste EXPORTEE : c est la couche de
// publication, et le document de rejeu est le contrat public.
//
// POURQUOI L EMPREINTE MONTE ALORS QUE LE DEPLACEMENT EST PUR. Le cadre d empreinte hache le
// chemin RELATIF A LA RACINE (arbitrage du lot 2.6.0, pose exactement pour que ce deplacement
// ne coute rien) : les 205 chemins haches sont donc INCHANGES. Ce qui change est le CONTENU —
// chaque fichier de `grammar` et de `facts` qui importait `film/source`, `film/profile` ou
// `film/grammar` cite maintenant `film/internal/...`. Une ligne d import par fichier, et rien
// d autre.
//
// `source.Rev` NE BOUGE PAS, et c est la preuve que le geste est propre : la couche `source` est
// une FEUILLE (son seul import du depot est `film/types`, qui ne bouge pas), donc aucun de ses
// fichiers de production n a change. `SchemaVersion` reste 60 ; aucun match deja decode n est
// candidat au backlog.
//
// ENTREE `grammar-2026-09-15.38` (2026-09-17, lot 2.5.e-d) : `.37` -> `.38`.
// TROIS DOCS INVERSEES CORRIGEES, ET UNE DECISION ECRITE. COMMENTAIRES SEULS.
//
// AUCUNE LIGNE DE CODE NE CHANGE dans cette entree : ce sont des octets de commentaire, et ils
// font monter l empreinte parce que le hachage porte sur les OCTETS — c est le faux positif
// assume du mecanisme, et il coute une ligne.
//
//	`observateur.go`           SEPT champs annoncaient « Global de paquet », dont un
//	                           « UN SEUL decodage filmdec a la fois par process » — l inverse de
//	                           l en-tete du meme fichier depuis le lot 2.3, qui dit que le
//	                           LECTEUR porte l observateur et que deux films se decodent en
//	                           parallele (`TestDeuxFilmsEnParallele`). Quatre de ces phrases
//	                           etaient en outre TRONQUEES, coupees au deplacement des champs.
//	`profile_globales_test.go` son en-tete annoncait « le lot 2.1 resout le profil mais ne le
//	                           fait lire par AUCUN lecteur de bits : les globales de paquet
//	                           decident encore » — elles n existent plus depuis le lot 2.3, et
//	                           le paragraphe se contredisait avec le sien propre.
//	`player_table_control.go`  la DECISION de D3 + D8 est ecrite a l endroit qu elle concerne :
//	                           le controle du calibrage RESTE en `grammar` parce qu il DECODE
//	                           des enregistrements de slot (`decodeSlot`, `slotVacant`), donc
//	                           c est un LECTEUR. V15 (8) est perime sur sa moitie « controle ».
//
// `grammar/profile.go` et `profile/mpp_widths.go`, les deux autres docs inversees nommees par D2
// de la revue M2, avaient deja ete reecrites par le lot 2.5.b : verifiees sur pieces, elles ne
// portent plus les phrases citees.
//
// `SchemaVersion` reste 60 ; aucun match deja decode n est candidat au backlog.
//
