package filmdec

// grammar_rev_chronique_archive.go — LA CHRONIQUE DE [GrammarRev], RANGS `.12` A `.20`.
//
// # POURQUOI UNE ARCHIVE (2026-09-18, lot 2.4.2)
//
// La chronique est ne pouvoir que grandir : un lot, un rang, une entree. Sortie de
// `grammar_rev.go` au lot 2.4.1 parce que ce fichier avait atteint 500 lignes, elle a atteint le
// meme seuil deux rangs plus loin. ELLE SE ROTATIONNE DONC, comme `.ai/thought_log.md` : les
// rangs anciens passent ici, `grammar_rev_chronique.go` ne garde que la suite VIVANTE. Le geste
// se refait quand ce dernier repasse 500 lignes — c est un geste ordinaire, pas un incident.
//
// Comme `grammar_rev.go` et `grammar_rev_chronique.go`, ce fichier est EXCLU de l ensemble hache
// par l empreinte (`fichiersHorsGrammaire`) : il DECRIT la grammaire, il n en fait pas partie.
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
