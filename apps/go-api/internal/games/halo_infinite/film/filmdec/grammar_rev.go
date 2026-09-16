package filmdec

// grammar_rev.go — LA REVISION DE LA GRAMMAIRE DU FILM.
//
// # LA REGLE A TROIS ETAGES, ET CE QUE CHACUN PROTEGE
//
//	GrammarRev              monte a TOUT changement de grammaire — une largeur, un cadre, un
//	                        ordre de composants, un lecteur neuf. C'est la revision de CE qui lit
//	                        les octets du film.
//	KillSourceDecoderRev    monte quand la SORTIE de `killsource` peut changer : les lignes de
//	                        kill deja en base sont alors candidates au backlog de redecodage.
//	SchemaVersion           monte quand le CONTENU CUIT change : `backfill-replay` re-cuit tout
//	                        artefact anterieur.
//
// Les trois sont INDEPENDANTES et ne se remplacent pas. Une largeur corrigee dans un composant
// que personne ne consomme encore fait monter `GrammarRev` SEULE. La meme largeur, une fois
// branchee sur le kill feed, fait monter `KillSourceDecoderRev` aussi. Si l'artefact publie s'en
// trouve change, `SchemaVersion` monte a son tour. Confondre les trois, c'est soit re-cuire le
// parc pour un commentaire, soit laisser en base des lignes decodees par une grammaire morte.
//
// # POURQUOI UNE CONSTANTE, ET PAS UN COMMENTAIRE
//
// `KillSourceDecoderRev` a porte pendant des mois la consigne « la faire evoluer a chaque
// changement de decodage » : mesure du 2026-09-05, 14 commits sur le decodeur, ZERO bump. Une
// consigne ecrite dans un commentaire ne se tient pas toute seule. Le garde-rail qui rend
// celle-ci executoire est `grammar_rev_fingerprint_test.go` : il hache les sources de `filmdec`
// ET de `killsource`, et rougit des qu'une d'elles bouge sans que cette constante monte.

// GrammarRev est la revision de la grammaire de lecture du film.
//
// FORME : `grammar-AAAA-MM-JJ`, la date du jour ou la grammaire a change, suivie d'un `.N`
// quand un SECOND lot la change le MEME jour. Ce qui doit rester separable est le LOT, pas le
// commit : deux changements d'un meme lot partagent la revision (lot 1.1.5, 2026-09-14), deux
// LOTS ne la partagent pas.
//
// LE SUFFIXE EST NE AU LOT 1.2 (2026-09-14), et il corrige une regle qui se retournait contre
// son objet. La regle disait « deux changements le meme jour partagent la meme revision » ; le
// lot 1.2 (le registre lu a l'octet 8) tombait le meme jour que le lot 1.1 (l'equipe a l'octet
// 37 du pied). La partager aurait voulu dire regenerer le golden sur la branche « revision
// inchangee, empreinte differente » — c'est-a-dire faire taire le ratchet dans le cas precis
// pour lequel il existe : une grammaire qui change. La forme admet donc un rang, et la revision
// continue de nommer ce qu'elle nomme.
// LOT 1.9.2 (2026-09-15) : `grammar-2026-09-15.1` -> `grammar-2026-09-15.2`. Le résolveur de
// distance de touche (`weapon_hit_distance_resolver.go`) prend désormais l'ENTRÉE DE CATALOGUE de
// la carte au lieu de ses seules bornes, et en impose le DÉCOUPAGE d'i0 au balayage des positions
// — là où `ScanFilmOptions.Layout` restait nil, donc où `DetectI0LayoutOf` décidait. Aucune
// grammaire d'octets n'est réécrite ; ce qui change est QUELLE grammaire s'applique, et c'est
// exactement ce que cette révision doit nommer.
// FUSION (2026-09-16) : l integration portait `.3` (lots 1.9.2 et 1.9.3) et la branche du lot
// 1.9.1 bis `.6` (gardes n1/n2, quatre sites de ti=37, prefixe objet relu, profil par build) ;
// les deux grammaires sont reunies ici, au rang suivant.
// LOT 1.9.4 (2026-09-15) : `grammar-2026-09-15.7` -> `.8`. `DetectFilmMapEntry` est SUPPRIMEE de
// `weapon_hit_distance_resolver.go` — elle identifiait la CARTE d un film par la signature de ses
// largeurs d axe (`DetectI0Layout`), alors que son unique appelant tenait deja le nom de carte du
// match. AUCUN BIT LU NE CHANGE, et l empreinte monte quand meme parce qu elle hache des octets
// de source (c est ecrit dans son en-tete) : ce qui change est QUELLE carte, donc quelles bornes
// et quel decoupage, s appliquent a un film — le meme genre de changement que la revision `.2`
// nommait au lot 1.9.2. `KillSourceDecoderRev` ne bouge PAS (`film/killsource/` n a pas bouge,
// et son propre ratchet d empreinte fait foi) ; `SchemaVersion` non plus (le chemin de cuisson
// n appelait pas cette fonction — verifie le 2026-09-15 : equivalence 10/10 identiques,
// corpus gate 14 temoins a 0 gain / 0 perte / 0 changement).
// LOT 1.9.1 ter (2026-09-15) : `.7` -> `.8`. AUCUN BIT LU NE CHANGE, mais la CLE de la
// grammaire, si — et c'est exactement ce que cette revision doit nommer. Le decoupage du bloc
// `object-multiplayer-properties` etait keye par le NOM DE BUILD ; il l'est desormais par la
// VERSION DE FORMAT de `chunk_00` (`chunk_00+4`), qui est la valeur que le LECTEUR du jeu
// consulte (`FUN_14299ab50` : la largeur du registre et celle de la table par type en
// derivent ; `FUN_1428e1c0c` : c'est elle que les six branches de version de l'executable
// lisent). Sur les 1 351 films du cache les deux cles donnent le MEME decoupage — le
// changement est verifiable et neutre — mais la nouvelle couvre les cinq films sans section
// d'identification, que l'ancienne ne pouvait pas nommer.
// LOT 1.9.1 ter (2026-09-15, second commit) : `.8` -> `.9`. AUCUN BIT LU NE CHANGE ICI NON
// PLUS, et la revision monte pour la meme raison qu au `.8` : le CADRE. La condition du repli
// `repli_largeurs_mpp_calibrees_sur_le_film` est renommee `build_sans_profil_relu` ->
// `format_sans_profil_relu` (elle nommait une cle qui n existe plus), et le declenchement du
// repli sur une version de format INCONNUE est desormais COMPTE
// (`filmdec.UnknownFormatExpvarPairs` -> `filmdec_unknown_format_<n>`, cable dans
// `replay/mpp_format_inconnu.go`) et signale par un avertissement par film. Un consommateur qui
// decide de redecoder doit voir que la condition du repli a change de nom ; un exploitant doit
// voir qu un patch du jeu a change le format. `SchemaVersion` reste 59.
// FUSION (2026-09-16) : l integration portait `.8` (lot 1.9.4) et la branche du lot 1.9.1 ter
// `.9` (cle = version de format, repli compte) ; les deux sont reunies ici, au rang suivant.
// LOT 1.9.13 (2026-09-15) : `.7` -> `.8`. AUCUNE grammaire d octets ne change, et aucun bit lu
// n est lu autrement : `objectiveevents.RoundBounds.Starts` est un ACCESSEUR de lecture sur des
// bornes de manche deja mesurees, que la decoupe des vies du rejeu consomme. L empreinte hache
// les OCTETS des trois paquets (cf. grammar_rev_fingerprint_test.go, « il ne distingue pas un
// changement de grammaire d une reformulation de commentaire ») : le faux positif coute cette
// ligne, et c est le marche assume du garde-rail. `KillSourceDecoderRev` ne bouge PAS —
// `killsource/` n est pas touche.
// FUSION (2026-09-16) : l integration portait `.10` et la branche du lot 1.9.13 `.8` (accesseur
// neuf dans objectiveevents, faux positif d empreinte) ; reunies ici, au rang suivant.
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
// ENTREE `grammar-2026-09-15.12` (2026-09-16, revue de jalon M1 lentille L4, merge `99644996e`) :
// `.11` -> `.12`. AUCUNE grammaire d octets ne change. Ce qui change est la PORTE qui decide si
// les attributions ligne par ligne de `killsource` sont publiables : `BijectionDetermined` valait
// « au plus un indice a inferer », il vaut desormais « une seule affectation possible »
// (`FilmTablePinning.AffectationUnique`, indices libres ET noms libres). L empreinte hache les
// octets des trois paquets, dont `killsource/` : elle monte donc, et la revision avec elle.
// `KillSourceDecoderRev` MONTE aussi (`killsource-2026-09-16`) parce que la sortie persistee
// change — la porte ne fait que se fermer, aucun film ne gagne la publication ligne par ligne.
// `SchemaVersion` reste 59 : le document du rejeu ne porte pas cette porte.
//
// LE MEME RANG PORTE AUSSI LA CORRECTION DE DOC de `mppWidthsPourFormat` (son bloc finissait par
// « la cle est le BUILD » alors que la fonction commute sur le FORMAT depuis le lot 1.9.1 ter) :
// l empreinte hache les OCTETS des trois paquets, commentaires compris, donc une reformulation la
// fait bouger. Un LOT partage sa revision (regle de la forme `.N` ci-dessus) — ces deux
// changements sont le meme lot de revue, ils partagent donc `.12`.
//
// ENTREE `grammar-2026-09-15.13` (2026-09-16, revue de jalon M1 lentille D13 constat 2, merge
// `1f478d5c3`) : `.12` -> `.13`. AUCUNE grammaire d octets n est reecrite ; ce qui change est
// QUELLE grammaire s applique, et c est exactement ce que cette revision doit nommer (meme genre
// de changement que `.2` et `.8`). Les DEUX sites qui installent le decoupage du bloc
// `object-multiplayer-properties` — `ScanEquipmentPlacements` et `replay.gwWidthsForFilm` — le
// resolvaient par `BuildProfileFromFilm`, donc par la table des SEPT builds en dur, alors que le
// registre des replis declare la cle VERSION DE FORMAT (`format_sans_profil_relu`, lot 1.9.1
// ter). Un film au format 27 dont le build est hors table prenait les largeurs CALIBREES devant
// une largeur RELUE, sans compteur ni avertissement. Les deux sites passent desormais par
// `MPPWidthsForFilm`, PORTE UNIQUE — c est le changement de comportement de ce rang.
//
// MESURE QUI BORNE L EFFET (cache, 657 films au 2026-09-15, `TestMPPResolutionCorpus`) : 6 films
// portent un build hors table — 5 sans section d identification (format 20) et 1 `HI_1_5_1`
// (format 23) —, AUCUN a un format dont la largeur est relue. Zero octet cuit ne change sur ce
// cache ; le gain porte sur le parc NEUF. `SchemaVersion` reste 59.
//
// `KillSourceDecoderRev` NE BOUGE PAS A CE RANG, et la confusion vaut d etre nommee : la porte de
// publication de `killsource` (`AffectationUnique`) releve de CETTE constante-la, et elle a ete
// traitee au rang PRECEDENT (`.12`). `.13` ne touche que la porte MPP de `filmdec`/`replay` ;
// `killsource/` n y est pas modifie, et son propre ratchet d empreinte fait foi.
//
// ENTREE `grammar-2026-09-15.14` (2026-09-16, revue de jalon M1 lentille L3, merge `29c5d6c85`) :
// `.13` -> `.14`. AUCUNE grammaire d octets ne change, et aucun bit n est lu autrement. Trois
// gestes, tous de SURFACE :
//
//	(1) `DetectI0Layout(dir)` et `ScanFilmEquipmentSpawnEvents(dir)`, deux enveloppes `dir` de
//	    PRODUCTION sans aucun appelant de production (48 appels de test pour la premiere, 1 pour
//	    la seconde — greps colles au compte rendu du lot), sortent du binaire : la premiere
//	    devient `detectI0Layout` dans un fichier de test du paquet, la seconde disparait au
//	    profit de sa forme film chez son unique appelant. Regle 7 du depot (« 0 code mort ») ;
//	(2) `walkKeyframeBody`, la boucle de corps d image-cle que le lot 1.4 avait deja unexportee
//	    faute d appelant de production, et sa table `keyframeBodyVariants`, passent dans un
//	    fichier `_test.go` du meme paquet — leurs cinq appelants sont des instruments ;
//	(3) deux en-tetes corriges (`film_major_version.go` nomme le second u32 — la VERSION DE
//	    FORMAT — et renvoie a `film_format_version.go`, dont l imparfait fautif tombe).
//
// LES TROIS FORMES LUES EN PRODUCTION SONT INCHANGEES, A L OCTET : `DetectI0LayoutOf`,
// `ScanEquipmentSpawnEvents`, `WalkKeyframeFullState`. L empreinte hache les OCTETS des trois
// paquets (cf. `grammar_rev_fingerprint_test.go`, « il ne distingue pas un changement de
// grammaire d une reformulation de commentaire ») : ce faux positif COUTE ce rang, et c est la
// seule raison pour laquelle `.14` existe. `KillSourceDecoderRev` ne bouge PAS (`killsource/`
// intact) ; `SchemaVersion` non plus.
//
// TOUTE ENTREE NEUVE OUVRE SUR LE MOT `ENTREE` SUIVI DE LA REVISION entre accents graves, et le
// golden `testdata/grammar_rev.golden` porte la sienne en regard :
// `TestChroniqueCouvreLaRevisionCourante` exige les DEUX pour la valeur ci-dessous, et c est ce
// qui empeche la chronique de s arreter a un rang que la constante a depasse (constat F5). Les
// entrees anterieures au 2026-09-16 n ont pas cette forme — elles ne sont pas relues par le
// ratchet, qui ne mord que sur la valeur COURANTE.
//
// ENTREE `grammar-2026-09-15.15` (2026-09-16, lot 1.9.10, merge `3b5e1b465`) : UN LECTEUR NEUF
// ENTRE DANS LE PAQUET — la MARCHE des morts d objet (`object_deaths*.go`) deroule la boucle de
// records des paquets delta et lit le composant `object-dead-state` (ti=40) la ou aucun
// balayage ancre ne l atteint ; le verrou `DesyncAt == -1` qui jetait des morts lues est leve
// (accepter si `DesyncAt == -1` ou `DesyncAt > index(dead-state)`). Aucune grammaire d octets
// existante n est reecrite ; ce qui change est CE QUE LE PAQUET SAIT LIRE. La calibration du
// cadre (IDLowBits) ne balaye plus l amorce (propriete du format) et son cadre par defaut est
// un repli nomme et compte. `KillSourceDecoderRev` ne bouge PAS ; `SchemaVersion` reste 59 en
// attendant la montee unique de la vague (les champs `vehicles[].end/tEnd` et
// `coverage.vehicles.*` arrivent avec elle).
//
// ENTREE `grammar-2026-09-15.16` (2026-09-16, lot 1.9.7, fusion) : L APPARIEMENT `dead-state <->
// kill-feed` DE `killsource` SE FAIT PAR L IDENTITE DE PAQUET `(chunk, pidx)` que le film ecrit,
// et non plus par une fenetre de 2,5 s (`killsource/paquet_identite.go`, six sites convertis, la
// fenetre devient le repli nomme et compte `repli_appariement_par_fenetre_temporelle`). Aucune
// grammaire d octets n est reecrite : ce qui change est QUEL enregistrement lu se rattache a quel
// instant — et l empreinte de cette revision couvre `killsource/`, donc elle monte avec lui.
// `KillSourceDecoderRev` monte au meme geste (`killsource-2026-09-16.2`) ; `SchemaVersion` reste 59.
//
// ENTREE `grammar-2026-09-15.17` (2026-09-16, lot 1.9.11, fusion) : LE DESIGNATEUR DE MANCHE EST PUBLIE TEL QU ECRIT ET LA GARDE D ORDRE
// devient une CONTRADICTION publiee (coverage.score.roundsWritten / roundsContradicted / roundsDecreed) ; le decret de la manche 0 est un repli nomme et compte ; ResolveRounds rend le verdict complet (objectiveevents, hache par l empreinte). Aucun octet lu autrement ; SchemaVersion 59 (montee de vague).
// ENTREE `grammar-2026-09-15.18` (2026-09-17, lot 2.1, « le profil, resolu une fois, encore
// recopie ») : `.17` -> `.18`. AUCUNE grammaire d octets n est reecrite, et AUCUN bit n est lu
// autrement — c est la promesse meme du jalon M2 (D4 : un pas structurel est clos a ZERO
// difference d equivalence). L empreinte hache les OCTETS des trois paquets, commentaires
// compris : elle monte parce que la SOURCE change, et la revision avec elle.
//
// CE QUI CHANGE, ET C EST DE LA STRUCTURE :
//
//	`filmdec.Profile` NAIT (`profile.go`, `profile_table.go`). Il porte ce qui ne se lit pas
//	    dans le flux — identite, carte, implantation du gamertag, cadre d image-cle, mouvement,
//	    slots, MPP — resolu UNE fois a partir des TROIS cles que le film ECRIT (version de
//	    format, build, version majeure) et de l entree de catalogue de la carte. Champs prives,
//	    accesseurs par valeur, table par type clonee : immuable, et prouve tel.
//	LE CONTEXTE LE RESOUT A LA CONSTRUCTION (D1) sur le chemin de la cuisson, et il s ouvre
//	    desormais dans `replay.BuildFromFilm` au lieu de `scanFilmInputs` — pour qu il n y ait
//	    qu UNE resolution par cuisson. Les trois derivations memorisees restent paresseuses,
//	    donc calculees a la meme date qu avant, et l horloge des etapes demarre au meme endroit.
//	L INSTALLATEUR DES LARGEURS D AXE DE LA CARTE (`replay/world_object_precision.go`) LIT LE
//	    PROFIL et ecrit ENCORE la globale de paquet : double ecriture
//	    datee (`doubleEcritureGlobales`, bascule 2026-09-17, retrait cible lot 2.3, critere
//	    « 0 variable de paquet mutable dans filmdec »).
//	LES TROIS SITES DE `ParseHighlightEvents` QUI LISENT LEUR VERSION DANS LE FILM
//	    (`killsource/chunks.go`, `replay/deaths_source.go`, `cmd/levelup` par `ops`) la prennent
//	    a `HighlightProfileOfFilm` / `HighlightProfileFromHeader` : MEME u32, MEME valeur, source
//	    unique et implantation NOMMEE.
//
// `KillSourceDecoderRev` NE BOUGE PAS, et le choix est EXPLICITE comme son ratchet l exige :
// `killsource/` change de deux lignes — la source de la version majeure et le commentaire qui la
// nomme — et les lignes PRODUITES sont identiques a l octet, donc aucun match deja decode n est
// candidat au backlog. `SchemaVersion` reste 59 : le document publie ne gagne ni ne perd un champ.
// ENTREE `grammar-2026-09-15.19` (2026-09-16, lot 2.7 volet grammaire) : `.18` -> `.19`. AUCUN
// OCTET N EST LU AUTREMENT. Scission par DEPLACEMENT PUR des cinq fichiers de `filmdec` qui
// depassaient 500 lignes — `traverse.go` (1 388), `unit_weaponstate.go` (956),
// `frame_records.go` (793), `components_biped_ability.go` (699), `components_movement.go`
// (554). Le `switch` de 815 lignes et 194 arms de `consumeByName` devient une CHAINE de sept
// maillons relies par leur branche `default` (`dispatch_object.go` porte l explication et
// l exemption de longueur) : un arm qui rend `ported=false` rend depuis son propre maillon, le
// dernier maillon rend le `default` d origine mot pour mot, et l ordre des arms — celui des
// lots de portage — est conserve. Deux extractions seulement, toutes deux un bloc recopie
// desindente d une tabulation : `consumePredictedAbsolute` (FUN_140f7ea14, la branche
// `predFlag == 1` d i0, qui est une fonction du moteur a part entiere) et
// `skipCalibratedPosition` (le banc de calibration, garde par un drapeau).
// CONTROLE DE DEPLACEMENT PUR, colle au compte rendu du lot : le multi-ensemble des lignes de
// chaque fichier d origine est INCLUS dans celui de ses fichiers d arrivee — zero ligne perdue,
// les seules lignes neuves sont les en-tetes de fichier, les signatures des maillons et les six
// `return` de chainage. L empreinte hache les OCTETS des trois paquets : ce faux positif coute
// ce rang, meme nature que `.14` et `.13` de la revue M1.
// `KillSourceDecoderRev` ne bouge PAS (`killsource/` intact) ; `SchemaVersion` non plus (aucun
// octet cuit ne change, et `replay-equiv` doit rendre ZERO difference — une difference serait
// une regression, pas une divergence).
//
// ENTREE `grammar-2026-09-15.20` (2026-09-17, fusion des volets 2.7g et 2.7p du lot 2.7 + correctif
// lint) : `.19` -> `.20`. AUCUN octet n est lu autrement et AUCUNE grammaire n est reecrite : la
// source de `filmdec/` change de deux commentaires `//nolint:unparam` dates (consume140c1e9d4 : w
// toujours 12 ; consumeDynPrecVec3 : mag toujours 19 — largeurs de grammaire ecrites au site d appel,
// que le lot 2.2 porte au profil), sites sortis de la baseline lint par la scission 2.7g. L empreinte
// hache les octets, commentaires compris : elle monte, la revision avec elle. Le volet 2.7p (paquet
// `replay`, hors empreinte) est fusionne au meme geste ; `SchemaVersion` 60 et `KillSourceDecoderRev`
// inchangees.
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
// le LECTEUR DE BITS (`BitReader.mv`, pose par [BitReader.poserMouvement]), seul objet deja
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
// `filmdec.Observation`. Ratchet : 79 -> 43.
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
const GrammarRev = "grammar-2026-09-15.26"
