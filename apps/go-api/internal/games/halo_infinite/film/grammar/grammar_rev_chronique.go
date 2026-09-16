package grammar

// grammar_rev_chronique.go — LA CHRONIQUE DE [GrammarRev], UNE ENTREE PAR RANG.
//
// # POURQUOI CE FICHIER EXISTE (2026-09-18, lot 2.4.1)
//
// La chronique vivait dans le godoc de [GrammarRev]. Au rang `.28` ce fichier passait 500
// lignes, et le ratchet de taille (`archlint/film_file_size_test.go`) le refusait — a juste
// titre : une chronique qui ne peut plus grandir cesse d etre tenue, et c est exactement le
// defaut F5 que `TestChroniqueCouvreLaRevisionCourante` a ete ecrit pour fermer. Elle vit donc
// ici, ou elle peut grandir, et `grammar_rev.go` ne garde que la regle et la constante.
//
// # CE FICHIER N EST PAS DE LA GRAMMAIRE
//
// Comme `grammar_rev.go`, il est EXCLU de l ensemble hache par l empreinte (cf.
// `fichiersHorsGrammaire`) : il DECRIT la grammaire, il n en fait pas partie. Sans cette
// exclusion, ecrire une entree changerait l empreinte, et la branche « la revision a change
// sans que la grammaire bouge » redeviendrait du code mort (revue R1, P2-3).
//
// # LA CHRONIQUE, A PARTIR DU `.11` : UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// LES TROIS DERNIERES ENTREES ONT ETE REECRITES LE 2026-09-16 (revue de jalon M1, RONDE 2,
// constat F5). Elles etaient EMPILEES et toutes trois annoncaient « `.11` -> `.12` », suivies de
// deux lignes « FUSION ... au rang suivant » qui racontaient une renumerotation que l integration
// n a jamais faite : la chronique s arretait donc a `.12` pendant que la constante valait `.14`,
// et les changements de COMPORTEMENT portes par `.13` et `.14` n avaient AUCUNE entree. Relevee
// commit par commit sur l integration (`git log --first-parent`), la suite reelle est celle-ci —
// un lot, un rang, dans l ordre ou les merges sont tombes.
//
// LES RANGS `.12` A `.26` VIVENT DANS `grammar_rev_chronique_archive.go` : la chronique se
// ROTATIONNE quand ce fichier atteint 500 lignes, comme `.ai/thought_log.md`. Le geste a ete
// refait le 2026-09-16 (lot 2.5.b), sur les rangs `.21` a `.26` (les six lots de la famille
// 2.2) : c est le geste ordinaire que l en-tete de l archive annonce, pas un incident. Ce qui
// suit est la suite VIVANTE, a partir du `.27`.
//
// ENTREE `grammar-2026-09-15.27` (2026-09-17, lot 2.3 — RANG PROVISOIRE) : `.26` -> `.27`.
// AUCUN OCTET N EST LU AUTREMENT.
//
// LE PROFIL DE BALAYAGE REMPLACE L HERITAGE PAR L ETAT DU PROCESSUS. La variable `herite`
// (`profil_herite.go`, lots 2.2.a/b/e) — traversee, largeur d axe absolue, largeurs des objets
// du monde, decoupage MPP, `param_4` force — disparait. Ce qu elle portait devient
// [ProfilDeBalayage], une VALEUR : le lecteur en tient une copie (`Lecteur.p`), le contexte du
// film celle du decodage courant, et `FrameConfig.Profil` la passe aux portes de balayage.
//
// LA CALIBRATION DE `killsource` VOYAGE DESORMAIS PAR LES OPTIONS (condition D1 du pilote).
// `replaybuild.BuildBytes` decode `killsource` PUIS appelle `replay.BuildFromFilm` dans le MEME
// processus ; jusqu ici la cuisson heritait des largeurs calibrees par l ETAT DU PROCESSUS —
// heritage REEL et VOULU, mais invisible et incompatible avec deux decodages en parallele. Le
// chemin est explicite : `killsource.Result.ProfilCalibre` -> `replay.Options.ProfilDeBalayage`
// -> `FilmContext`. Kill-feed non decode : `killsource.ProfilDeDepart()` — l invariant plus le
// `param_4` force a zero, ce que `Decode` laissait derriere lui meme en echec. Pas STRUCTUREL.
//
// LA DOUBLE ECRITURE DATEE EST RETIREE A SA DATE CIBLE : `replay.doubleEcritureGlobales`
// (bascule 2026-09-17, cible « lot 2.3 », critere « 0 variable mutable ») disparait avec la
// variable qu elle alimentait ; `installWorldObjectPrecision` pose sur le CONTEXTE.
//
// LES ENVELOPPES D2 (`ScanFilm*(dir)`) POSENT LE DECOUPAGE LU DANS LE FILM sur leur propre
// contexte — le geste que chaque instrument repetait a la main, et dont l oubli « desalignait les
// desers sans lever d erreur ». La CUISSON prend les largeurs du CATALOGUE, jamais l auto-detection.
//
// LES DOUZE BASCULES DE GRAMMAIRE SUIVENT LE MEME CHEMIN (famille 2 du lot) : les A/B de
// retro-ingenierie — corruption per-composant, queue d un record NEW, deser d etat par
// archetype, `simulation-state` complet, portee baseline, grammaire d ECRIVAIN du chemin absolu
// d i0, corps i54 et i59, inference de chaine, generation stricte, et les DEUX tables de
// largeurs — deviennent [GrammaireBalayage], un champ du profil. Ratchet : 42 -> 30.
//
// LA GENERATION STRICTE ETAIT LE SECOND HERITAGE SILENCIEUX, desormais ecrit :
// `killsource.resetGlobals` levait `SetStrictGeneration(true)` pour tout le PROCESSUS sans
// jamais le rabaisser. [killsource.ProfilDeDepart] le porte, `replaybuild` le passe.
//
// UN CADRE DE TRAME SE PREND AU CONTEXTE ([FilmContext.CadreDeBalayage]), plus a
// `DefaultFrameConfig()` seul, qui rend l INVARIANT : s en contenter decoderait aux largeurs
// d une autre carte — ce que l heritage masquait (`ScanObjectDeaths`).
//
// LA CAPTURE DE POSITION SUIT (famille 3 du lot). Six variables de paquet decrivaient UN record
// en cours de decodage — ou le composant i0 a commence, son slot, le monde d accumulation et son
// slot, le repli d absolue : elles deviennent `captureDePosition`, un champ du LECTEUR. La
// septieme, l histogramme des index de plage absolus, est un COMPTEUR et rejoint `Observation`.
// `lastRepVersion` / `LastRepVersion()` : SUPPRIMES (aucun appelant). Ratchet 30 -> 23, dont
// UNE SEULE encore ecrite (`observateur`).
//
// L OBSERVATEUR EST LA DERNIERE A PARTIR (famille 4 du lot), avec les VINGT-HUIT reglages
// publics qui l ecrivaient. Chaque balayage construit le SIEN et le pose AVEC son profil par un
// porteur unique — `ContexteDeLecture{Profil, Obs}` — dont les deux champs ont une nature
// OPPOSEE : le profil DECIDE des largeurs, l observateur ne fait que RECEVOIR. Les onze
// `publishXxx` et les compteurs d issue de chaine deviennent des methodes nil-safe
// d `Observation` ; `ChainStats`, `ResetChainStats`, `ChainRepairedCount`, `InferResyncCount`
// (accesseurs DE PROCESSUS sans appelant) sont supprimes.
//
// RATCHET : 23 -> 22 puis 21 (resserre a la revue), et surtout **ZERO variable de paquet
// ECRITE** : quatre erreurs sentinelles, seize tables de grammaire, un dedoublonneur.
//
// LE VERROU DE PROCESSUS DISPARAIT (famille 5, item 2.3.1) : `LockProcessDecode` et son fichier
// `decode_gate.go` sont SUPPRIMES, avec les 373 sites d appel (276 fichiers) qui le prenaient.
// Son en-tete nommait DEUX raisons d exister — l etat de paquet du decodeur, et « la table sans
// verrou » des largeurs de bouchon ; les quatre familles precedentes ont retire l une (devenue
// VALEUR du lecteur) et l autre (CHAMP de l observation). Un verrou prive de ses deux raisons
// n est plus une protection mais une serialisation. Le verrou INTER-PROCESSUS
// `filmproc.AcquireSolo` n est PAS concerne : il garde la RAM de la machine, et il reste.
//
// DEUX RATCHETS FIGENT LE RESULTAT (item 2.3.2). `TestAucunVarDePaquetEcriteDansFilmdec` refuse
// par AST toute ecriture visant une variable de paquet de `filmdec`.
// `TestAucunVerrouDeDecodageDePaquet` remplace l ancien ratchet INVERSE qui EXIGEAIT le verrou :
// il interdit `LockProcessDecode`, `processDecodeMu` et un fichier nomme `decode_gate.go`.
//
// DEUX FILMS SE DECODENT EN PARALLELE (item 2.3.3) : `TestDeuxFilmsEnParallele` decode
// `a521164d` (HI_1_4_1) et `fb1a1a72` (HI_1_13_0) en serie puis dans deux goroutines ;
// empreintes identiques a l octet, sous `-race`.
//
// `KillSourceDecoderRev` ne bouge PAS : `killsource/` change de FORME (la calibration rend un
// profil, `resetGlobals` disparait) mais les lignes PRODUITES sont identiques a l octet. Son
// golden est regenere pour refiger le couple (revision, empreinte). `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.28` (2026-09-18, lot 2.4.1 — RANG PROVISOIRE) : `.27` -> `.28`.
// AUCUN OCTET N EST LU AUTREMENT, et c est PROUVE bit a bit, pas suppose.
//
// LE LECTEUR DE BITS DESCEND DANS LA COUCHE SOURCE. `source.Bits` est desormais LE lecteur
// du depot : MSB-first big-endian, bourrage a zero au-dela du tampon, lecture par mot de 64
// bits. [Lecteur] ne porte plus ni tampon ni position — il EMBARQUE `*source.Bits` et n y
// ajoute que ce qui appartient a la grammaire : le profil de largeurs, la capture de position,
// l observateur, et le codec [Lecteur.ReadSignedVarWidth]. `filmdec/bits_word.go` disparait :
// sa lecture par mot est [source.BitsAt], et ses trois derniers appelants directs
// (`PeekBits`, `kfReadBits`, `readBitsAt`) y passent.
//
// `killsource.evReader` EST ABSORBE (item 2.4.1). Le deuxieme des sept lecteurs de bits du
// depot — son type, sa boucle `bitsWide`, et les trois primitives de position du paquet
// `bitAt` / `bits32` / `bitsN` — est SUPPRIME ; ses 23 sites passent par [source.BitAt] et
// [source.BitsAt]. `bits32` lisait CINQ octets puis decalait, ce qui n est pas la boucle de
// la lecture par mot : son equivalence est prouvee comme les autres.
//
// LE DRAPEAU DE DEBORDEMENT RESTE AU MARCHEUR, PAS AU LECTEUR (arbitrage V15 (3)). Le lecteur
// canonique garde la semantique du MOTEUR (bourrage a zero) ; la MEFIANCE de la chaine
// d evenements — sans laquelle une chaine desynchronisee lit des evenements valides apres la
// fin du paquet — vit dans `killsource.curseurEv`, qui teste `Remaining()` avant chaque lecture.
// `bp+n > len(pl)*8` et `Remaining() < n` sont la MEME condition : aucune valeur lue ne change,
// et `Skip` reste borne exactement la ou `evReader.skip` le bornait (chez le marcheur).
//
// LES DEUX PREUVES. (1) Appel par appel, sur les positions REELLES des chaines des dix bobines
// versionnees — 1 114 paquets a events, 109 168 positions, 72 largeurs par position — les copies
// de reference des anciens lecteurs et le lecteur canonique rendent la meme valeur, la meme
// position de sortie et le meme drapeau (`killsource/equivalence_lecteur_test.go`). (2) De bout
// en bout, les triplets (code, bit de debut, bit de fin) de chaque chaine, plus les six champs
// de chaque kill-event, sont IDENTIQUES a un golden produit par le code de la BASE, avant
// l absorption (`killsource/testdata/chaines_evenements.golden`, sans porte `-update`).
//
// L EMPREINTE HACHE DESORMAIS QUATRE RACINES : `internal/analysis/filmsource` rejoint `filmdec`,
// `killsource` et `objectiveevents`. Sans cela ce lot aurait OUVERT UN TROU — la lecture de bits
// qui vient d y descendre aurait pu changer sans que `GrammarRev` bouge.
//
// `KillSourceDecoderRev` ne bouge PAS : la SORTIE de `killsource` est identique a l octet (c est
// exactement ce que les deux preuves ci-dessus etablissent), seule sa source change. Son golden
// est regenere pour refiger le couple (revision, empreinte). `SchemaVersion` reste 60.
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
// `KillSourceDecoderRev` ne bouge PAS : la SORTIE de `killsource` est identique a l octet.
// `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.30` (2026-09-16, lot 2.5.d.2 — RANG PROVISOIRE) : `.29` -> `.30`.
// DEPLACEMENT PUR, SORTIE IDENTIQUE.
//
// `internal/analysis/objectiveevents` descend sous
// `internal/games/halo_infinite/film/facts/objectives` : c est la couche `facts` de l ADR 0034
// D-1 qui rentre DANS le decodeur. Le paquet change de nom (`objectiveevents` -> `objectives`)
// et de chemin, RIEN D AUTRE — aucune ligne de corps n est ajoutee ni retiree, seuls la clause
// `package`, les imports et les mentions du nom en commentaire bougent (`git diff -M` ne montre
// que des renommages).
//
// L EMPREINTE MONTE PARCE QU ELLE HACHE DES OCTETS DE SOURCE, et c est ecrit dans son en-tete :
// la clause `package` de 27 fichiers de production a change. AUCUN OCTET DE FILM N EST LU
// AUTREMENT — la grammaire du pied de film (`scanTh10Events`, `decodeTh10Block`) est identique a
// l octet, seul son LIEU a change. `racinesGrammaire` (grammar_rev_fingerprint_test.go) et le
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
// `KillSourceDecoderRev` ne bouge PAS : la SORTIE de `killsource` est identique a l octet (seul
// un commentaire y nomme desormais `objectives`). `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.31` (2026-09-16, lot 2.5.c — RANG PROVISOIRE) : `.30` -> `.31`.
// DEPLACEMENT PUR, SORTIE IDENTIQUE.
//
// LA COUCHE `grammar` PREND SON NOM. `internal/games/halo_infinite/film/filmdec` devient
// `internal/games/halo_infinite/film/grammar` (554 fichiers, dont 142 de production), et
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
// `KillSourceDecoderRev` ne bouge PAS : la SORTIE de `killsource` est identique a l octet.
// `SchemaVersion` reste 60.
//
// ENTREE `grammar-2026-09-15.32` (2026-09-16, lot 2.5.a — RANG PROVISOIRE) : `.31` -> `.32`.
// DEPLACEMENT PUR, SORTIE IDENTIQUE.
//
// LA COUCHE `source` RENTRE DANS LE DECODEUR. `internal/analysis/filmsource` devient
// `internal/games/halo_infinite/film/source` : le paquet change de nom (`filmsource` ->
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
// `lecteur.go` lit `Profile`, `Observation`, `FrameConfig`, `ProfilDeBalayage` — c est-a-dire que
// `source` importerait `profile` et `grammar`, deux imports VERS LE HAUT. La couche `source` du
// lot est donc `filmsource` seul ; la remontee reste a faire, et elle n est pas un `git mv`.
//
// L EMPREINTE MONTE parce que la clause `package` de cinq fichiers de production a change et que
// la racine hachee suit le paquet. AUCUN OCTET DE FILM N EST LU AUTREMENT.
// `KillSourceDecoderRev` ne bouge PAS ; `SchemaVersion` reste 60.
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
// `killsource` est identique a l octet, donc `KillSourceDecoderRev` ne bouge pas (son golden est
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
// format — est desormais hachee, alors qu elle aurait pu bouger sans que `GrammarRev` monte si
// la racine n avait pas suivi.
//
// `KillSourceDecoderRev` ne bouge PAS : `film/facts/killsource` ne change que ses qualifieurs,
// sa sortie est identique a l octet, et son propre ratchet d empreinte fait foi.
// `SchemaVersion` reste 60.
