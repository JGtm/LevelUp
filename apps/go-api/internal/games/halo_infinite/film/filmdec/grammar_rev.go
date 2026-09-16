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
const GrammarRev = "grammar-2026-09-15.19"
