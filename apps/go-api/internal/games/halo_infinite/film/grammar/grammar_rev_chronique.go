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
// LES RANGS `.12` A `.20` VIVENT DANS `grammar_rev_chronique_archive.go` : la chronique se
// ROTATIONNE quand ce fichier atteint 500 lignes, comme `.ai/thought_log.md`. Ce qui suit est
// la suite VIVANTE, a partir du `.21`.
//
// ENTREE `grammar-2026-09-15.21` (2026-09-17, lot 2.2.a — RANG DE FUSION) : `.20` -> `.21`.
// AUCUN OCTET N EST LU AUTREMENT.
//
// LES SIX RANGS DU LOT 2.2 ONT ETE DECALES DE +1 (`.20`-`.25` -> `.21`-`.26`) : ils avaient ete
// poses a titre PROVISOIRE sur une base ou l integration valait `.19`, et celle-ci est passee a
// `.20` (fusion 2.7g + 2.7p + correctif lint). Ce sont donc les rangs de FUSION. Le decalage ne
// touche AUCUNE source hachee : `grammar_rev.go` est hors de l ensemble d empreinte depuis la
// revue R1 (P2-3), et l empreinte des six rangs est inchangee a l octet.
//
// LES LECTEURS DU CHEMIN DE POSITION PRENNENT LEUR VALEUR AU PROFIL. Les cinq valeurs de la
// famille 2.2.a — le descripteur de quantification de TRAVERSEE, la largeur d axe des chemins
// ABSOLUS, et les trois drapeaux de contexte (pleine precision, queue de poignee du delta,
// saut calibre) — etaient des VARIABLES DE PAQUET de `filmdec`. Elles voyagent desormais avec
// le LECTEUR DE BITS (`Lecteur.mv`, pose par [Lecteur.poserMouvement]), seul objet deja
// passe a tous les deserialiseurs : une copie par lecteur construit, jamais une lecture par bit
// lu (budget 0.A.5). Leur source est [mouvementDuProfil], la MEME fonction que [ResolveProfile]
// emploie — il n y a plus deux tables de valeurs.
//
// LE SEUL ECRIVAIN DE PRODUCTION PASSE PAR LE CADRE. La calibration de `killsource`
// (`calibrate.go`) balayait 63 configurations en ECRIVANT dans le processus a chaque essai ;
// elle les passe maintenant par `FrameConfig.Mouvement`, que chaque porte de balayage pose EN
// TETE sur son lecteur. Son resultat est RENDU (`calibration.Mouvement`) et passe explicitement
// a `runWalk` et a `calibrateRSP`, qui en heritaient jusqu ici par effet de bord.
//
// UNE VARIABLE DE PAQUET SURVIT, ET ELLE EST NOMMEE : `herite`
// (`mouvement_herite.go`), le profil qu une passe laisse a la suivante DANS LE MEME PROCESSUS.
// Ce n est pas un reglage mais un FAIT DE PRODUCTION mesure sur pieces : `replaybuild` decode
// `killsource` PUIS appelle `replay.BuildFromFilm`, et `killsource` ne restaure pas les
// largeurs qu il a calibrees — la cuisson du rejeu decode donc deja aux largeurs du kill-feed.
// Le retirer changerait la largeur de chaque i0 du rejeu, ce que D4 interdit. Il porte donc sa
// date de bascule, sa cible de retrait (lot 2.3, au plus tard 2.5) et son critere mesurable.
// Bilan du ratchet : 94 -> 90 variables de paquet.
//
// `KillSourceDecoderRev` NE BOUGE PAS : `killsource/` change de forme (la calibration rend son
// resultat au lieu de l ecrire dans le processus) mais les lignes PRODUITES sont identiques a
// l octet — meme espace balaye, meme critere, meme vainqueur, memes largeurs pour les passes
// qui suivent. `SchemaVersion` reste 60 : aucun champ publie ne bouge.
// ENTREE `grammar-2026-09-15.22` (2026-09-17, lot 2.2.b — RANG DE FUSION) : `.21` -> `.22`.
// AUCUN OCTET N EST LU AUTREMENT.
//
// LES LECTEURS DES OBJETS DU MONDE PRENNENT LEUR VALEUR AU PROFIL. Quatre variables de paquet
// de plus le rejoignent : le descripteur world-object (largeurs d axe, largeur d index de
// region, region attendue), la range de dequantification absolue, le quantum du chemin delta
// et la largeur d axe du chemin delta axis-width. Trois n avaient plus d ecrivain depuis le
// lot E ; la quatrieme est posee PAR CARTE, et son installateur
// (`replay/world_object_precision.go`) ecrit desormais le profil HERITE au lieu d une globale
// propre. Aucune variable neuve : le canal du lot 2.2.a suffisait.
//
// POURQUOI L HERITAGE ET PAS LE PROFIL DU FILM, ICI AUSSI. Les largeurs de la carte doivent
// atteindre les quarante balayages de `replay.BuildFromFilm`, dont aucun ne recoit le profil et
// dont chacun construit ses propres lecteurs. Tant que le profil ne descend pas jusqu a eux
// (lot 2.5), l installateur reste le seul canal — c est la meme dette, pas une nouvelle.
//
// TROIS LECTEURS N ONT PAS DE LECTEUR DE BITS (`projectiles.go` : `projGateBits`, `projPosBits`,
// `decodeWorldObjectPos`). Ils lisent par DECALAGE D OCTET sur un payload, pour LOCALISER un
// record avant de le decoder, et prennent donc leurs largeurs a l heritage par une fonction
// NOMMEE au lieu d une variable. Le garde-rail d allowlist les voit : il est re-cle sur les
// TROIS formes d acces, plus sur un seul nom.
//
// `KillSourceDecoderRev` ne bouge PAS (`killsource/` intact) ; `SchemaVersion` reste 60.
// ENTREE `grammar-2026-09-15.23` (2026-09-17, lot 2.2.c — RANG DE FUSION) : `.22` -> `.23`.
// AUCUN OCTET N EST LU AUTREMENT.
//
// LES MARCHES D IMAGE-CLE PRENNENT LEUR CADRE AU PROFIL. `walkKeyframeFullState`,
// `consumeFullStateDefaultBlock` et la lecture d equipe du pied de record lisaient les DEUX
// constantes du paquet (en-tete de 108 bits, mot de taille de 32) ; elles les prennent
// desormais au lecteur, qui les porte depuis une source unique — `cadreDuProfil`, la meme
// fonction que [ResolveProfile] emploie pour poser `Profile.Keyframe`. Les deux constantes
// n ont plus qu UN lecteur dans le paquet : cette fonction.
//
// POURQUOI CELA COMPTE ALORS QUE LE RATCHET NE BOUGE PAS. La famille des images-cles ne portait
// AUCUNE variable de paquet : `keyframeBodyVariants` avait deja quitte la production a la revue
// de jalon M1 (constat C4, avec `walkKeyframeBody`), et les largeurs etaient des `const`. Le
// compte reste donc a 86 — et le gain n est pas la : il est qu une valeur faussee DANS LE
// PROFIL rougit desormais la lecture (`TestKeyframeClosureRatchet`, sur les sept bobines), au
// lieu de ne rougir qu une constante que personne ne relie au profil.
//
// LE BOUTON DES TEMOINS NEGATIFS SURVIT, ET IL EST INTACT : quand il est pose, il remplace le
// cadre du profil — c est sa seule raison d etre (mesure du plancher de faux positifs, regle 4
// de METHODE_RETRO_INGENIERIE_FILM). Il n est pas un reglage de production : non exporte, sans
// appelant hors instruments, et son propre garde-rail interdit meme de le NOMMER ici.
//
// `KillSourceDecoderRev` ne bouge PAS ; `SchemaVersion` reste 60.
// ENTREE `grammar-2026-09-15.24` (2026-09-17, lot 2.2.d — RANG DE FUSION) : `.23` -> `.24`.
// AUCUN OCTET N EST LU AUTREMENT.
//
// LA VERSION DU FILM VIENT DU PROFIL DU CONTEXTE. La phase de balayage des positions du rejeu
// rouvrait le registre pour son propre compte (`FilmMajorVersion(s.film)`) alors que le
// contexte avait deja resolu le profil a sa construction (D1). Elle prend desormais
// `Profile.Highlight()` — MEME valeur par le MEME chemin (les deux composent depuis
// `FilmMajorVersionFromHeader`), une localisation de `chunk_00` en moins par cuisson, et le
// drapeau `Lue` a la place d un second booleen qui disait la meme chose.
//
// CE QUE LA FAMILLE NE PEUT PAS FAIRE, ET POURQUOI C EST ECRIT ICI. La BRANCHE qui applique
// l implantation du gamertag (`analysis.decodeEventBytes`, `version <= 38 || version >= 41`) et
// les offsets du PIED (octets 36, 37, 47, 48 dans `analysis/objectiveevents`) vivent sous
// `internal/analysis/`, a qui le ratchet `no_title_package_in_analysis_test.go` interdit
// d importer un paquet de titre — allowlist a une seule entree, et ce n est pas celle-la. Les y
// faire lire le profil exigerait ce que la decision V5 et le pas 5 tranchent : faire descendre
// la source du film sous `film/internal/source`. La deuxieme copie de la regle reste donc en
// place, gardee par `TestProfilHighlightEgaleLeParseur` — le garde-rail qui interdit aux deux
// de diverger. Statue `[!]` a l item 2.2.d, consigne en §4 du plan.
//
// L EMPREINTE NE BOUGE PAS, ET LA REVISION SI : c est le seul lot de la serie dont le
// changement vit ENTIEREMENT hors des deux paquets haches (`filmdec/` et `killsource/`) — il
// est dans `replay/film_scan.go`. La revision monte quand meme, parce qu elle nomme la
// GRAMMAIRE SOUS LAQUELLE UN ARTEFACT A ETE CUIT et qu un lecteur de cette chaine a change ;
// le golden porte donc le meme sha sur deux rangs, ce qui se lit et ne se devine pas.
//
// `KillSourceDecoderRev` ne bouge PAS (`killsource/` intact) ; `SchemaVersion` reste 60.
// ENTREE `grammar-2026-09-15.25` (2026-09-17, lot 2.2.e — RANG DE FUSION) : `.24` -> `.25`.
// AUCUN OCTET N EST LU AUTREMENT.
//
// EQUIPEMENT ET MOBILITE AU PROFIL, ET L HERITAGE DEVIENT UNE STRUCTURE. Cinq valeurs de plus
// quittent les variables de paquet pour le profil que le lecteur porte : les DEUX largeurs du
// bloc `object-multiplayer-properties` (la lecture de tout l etat par defaut en depend), le
// `param_4` qu un harnais force et son drapeau, et les bits supplementaires d une action de
// mobilite. Deux listes de candidats de la calibration MPP redeviennent des FONCTIONS — c etaient
// des tables de grammaire deguisees en `var`, que rien n ecrivait.
//
// L HERITAGE PORTE DESORMAIS TOUT CE QU UNE PASSE LAISSE A LA SUIVANTE, en UNE structure
// (`profil_herite.go`, renomme depuis `mouvement_herite.go`) : mouvement, largeurs MPP,
// `param_4` force. Onze variables de paquet regroupees en une depuis le lot 2.2.a, avec une
// seule date de bascule, une seule cible de retrait et un seul critere.
//
// SA REMISE A ZERO RESTE BORNEE AU MOUVEMENT, et c est mesure : `killsource.resetGlobals` ne
// remettait pas les largeurs MPP avant ce lot (elles sont posees et RESTAUREES par leur
// installateur, donc equilibrees) ni le `param_4` (remis juste apres par son propre reglage).
// L elargir serait un changement de comportement, pas un nettoyage.
//
// DEPLACEMENT PUR EN PRIME : la migration des deux largeurs a fait passer `default_state.go` a
// 503 lignes ; le decoupage du bloc MPP en sort dans `mpp_widths.go`, sans qu une ligne de
// logique change. Ratchet des variables de paquet : 86 -> 79.
//
// LA REVUE ADVERSARIALE DU LOT (2.3.5) N A CHANGE AUCUN OCTET LU : 27 constats, 3 P1 et 19 P2
// confirmes, 5 refuses, zero P0. Cote `filmdec` — ratchet des variables resserre de 22 a 21 (le
// compte REEL ; 22 laissait une place libre), six blocs de doc INVERSEE reecrits, godoc de trois
// symboles disparus retirees, et `Lecteur.Observation` / `Lecteur.Contexte`, ajoutes par ce
// lot sans aucun appelant hors test, SUPPRIMES (regle 7). Cote `replay`, la pose du profil puis
// de la carte devient `poserProfilPuisCarte`, qu epingle
// `TestRouteDuProfilCalibreJusquAuContexte` : la route de D1 n avait aucun test qui rougisse, ni
// sur des options ignorees, ni sur l ordre inverse — qui effacait les largeurs de la carte en
// silence, `PoserProfilDeBalayage` remplacant le profil ENTIER.
//
// `KillSourceDecoderRev` ne bouge PAS : `killsource/` est INTACT — le balayage de `param_4`
// passe toujours par `SetRecordStateParam`, meme espace, meme critere, memes lignes produites.
// `SchemaVersion` reste 60.
// ENTREE `grammar-2026-09-15.26` (2026-09-17, lot 2.2.f — RANG DE FUSION) : `.25` -> `.26`.
// AUCUN OCTET N EST LU AUTREMENT.
//
// L OBSERVATEUR : TRENTE-SEPT VARIABLES DE PAQUET DEVIENNENT LES CHAMPS D UN SEUL OBJET. Les
// VINGT-NEUF crochets de deserialiseur (un par famille de composants, plus la capture de
// position, le masque de record et la sonde de references d unite) et les HUIT compteurs de
// l inference de chaine — dont `compWidthObs`, « la table sans verrou » que l en-tete de
// `decode_gate.go` nomme comme l une des deux raisons du verrou de processus — sont desormais
// `grammar.Observation`. Ratchet : 79 -> 43.
//
// UN OBSERVATEUR NE CHANGE AUCUNE CONSOMMATION DE BITS, et c est la propriete qui le distingue
// du PROFIL : le profil DECIDE des largeurs, l observateur ne fait que recevoir ce que le
// deserialiseur a deja lu. Elle est ecrite en tete de `observateur.go` — un champ qui changerait
// un compte de bits serait une valeur de profil mal rangee.
//
// LA FORME « PASSE EN PARAMETRE » EST OUVERTE, ET SEULEMENT LA OU ELLE NE MENT PAS.
// `FrameConfig.Obs` sert la famille de l inference de chaine, dont les compteurs sont ecrits
// dans des fonctions qui tiennent DEJA leur cadre : un instrument y passe SON observateur et lit
// ses compteurs sans jamais ecrire dans le processus. Les vingt-neuf crochets de deserialiseur,
// eux, sont publies par des feuilles que seul le lecteur de bits atteint, et le lecteur ne
// recevra son observateur qu au pas 5, avec le profil ; leur donner un parametre que les
// feuilles ne liraient pas serait un mensonge, pas une etape. C est ecrit tel quel dans
// `observateur.go`, avec la date de bascule, la cible de retrait et le critere.
//
// LES ANCRES `fichier:ligne` DE LA TABLE ECS SUIVENT (85 lignes recalees) : le retrait des
// declarations a deplace des fonctions, et le garde-rail G1 lit ces ancres sur pieces.
//
// `KillSourceDecoderRev` ne bouge PAS (`killsource/` intact) ; `SchemaVersion` reste 60.
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
// ENTREE `grammar-2026-09-15.34` (2026-09-16, lot 2.6 volet facts + source) : `.33` -> `.34`.
// REVISIONS PAR COUCHE ET TYPES DE CONTRAT, SORTIE IDENTIQUE.
//
// TROIS GESTES, ET AUCUN NE LIT UN OCTET AUTREMENT. (1) `source.Rev` NAIT (`source/rev.go`,
// golden a historique `source/testdata/source_rev.golden`) : la couche qui porte la porte aux
// octets a desormais sa revision propre, sur le mecanisme central `film/revision`. (2)
// `facts.Rev` NAIT (`facts/rev.go`) et REPREND la valeur de `KillSourceDecoderRev`
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
