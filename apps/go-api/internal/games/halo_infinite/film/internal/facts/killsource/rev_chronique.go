package killsource

// rev_chronique.go — LA CHRONIQUE DE LA REVISION DES FAITS, ET RIEN D AUTRE.
//
// SORTIE DE `rev.go` PAR DEPLACEMENT PUR le 2026-09-21 (lot 5.7) : le fichier passait 500
// lignes au moment ou l entree `.3` s y ajoutait, et le ratchet de taille
// (`archlint/film_file_size_test.go`) refuse de grandir. AUCUN OCTET DE CODE N EST TOUCHE — le
// fichier ne porte que des commentaires, exactement comme `grammar/rev_chronique.go`, dont ce
// decoupage reprend la forme. La regle de la chronique est INCHANGEE : une entree par rang,
// jamais reecrite, et la revision se decide AVANT le golden. Les rangs anterieurs a
// `killsource-2026-09-26` vivent dans `rev_chronique_archive.go` puis `rev_chronique_archive_2.go`.

// # LA CHRONIQUE — UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// ENTREE `killsource-2026-09-26` (2026-09-26, jalon J7 du PLAN_SUITE_AUDIT_DECODEUR_FILM) : LA
// REVISION MONTE UNE FOIS POUR LES SEPT CONSTATS FK-1 A FK-7, ET LA SORTIE DU KILL-FEED CHANGE.
//
//	FK-2 (J7.1)  un nom de remplissage (`?`, `?N`) n est plus jamais publie : un assistant pose
//	             par l inference sur un remplissage sort REJETE `hors-roster` (colonne
//	             `assist_gamertag` NULL, `assist_rejected` = `hors-roster`) ; les temps 4 et 5 ne
//	             publient pas une ligne dont le nom pris au roster serait un remplissage
//	             (`Stats.NomsDeRemplissageRefuses`). Predicat unique `estNomDeRemplissage`.
//	FK-1 (J7.2)  l espace des humains est le NOMBRE DE SIEGES de la table du film, et non plus le
//	             nombre de noms du kill-feed : un remplacant ne desepingle plus le bot de relais. Un
//	             bot n est desepingle que si son slot tombe sous cette borne ET sur un siege que la
//	             table NOMME (revue adverse : un siege VACANT intercale ne desepingle rien). Sans
//	             table, aucun bot n est desepingle. Les bots non epingles sont publies
//	             (`Coverage.BotsNonEpingles`), journalises par film, et comptes par
//	             `replayidentity` (`killsource_bots_non_epingles`).
//	FK-3 (J7.3)  le lien par motif du xuid cherche aussi les joueurs qui tuent sans mourir. Ces
//	             candidats ne font que COMPLETER (revue adverse) : une lecture qui se contredit ou
//	             tombe sur un indice deja retenu les ecarte SEULS (`MotifTueursEcartes`), sans faire
//	             tomber l epinglage par motif du film.
//	FK-4 (J7.4)  le temps 4 ne reecrit plus un instant publie (`Stats.CollisionsDeMortDeBot`),
//	             SAUF une ligne du temps 3 posee sur un couple RECOLLE : elle ne confirme pas le
//	             tueur du couple, et cede a la mort de bot verifiee au meme instant (revue adverse,
//	             `Stats.AutoInfligeesSurCoupleFabriqueRemplacees`). Un couple recolle n est fantome
//	             que si sa mort de bot est PUBLIEE, ce qui tient `Covered <= RealPairs`. Revue ronde
//	             2 : « au meme instant » est a la milliseconde — le dead-state de la mort de bot doit
//	             tomber a l instant du kill-feed que la ligne porte ; une mort de bot prise par la
//	             fenetre a un kill VOISIN n est plus publiee a une fausse date et ne prive plus ce
//	             voisin de sa ligne. Un remplacement retire la provenance de la ligne remplacee :
//	             `Stats.Appariement` compte une provenance PAR LIGNE PUBLIEE (plus de `Fenetre`
//	             gonfle d une ligne disparue).
//	FK-5 (J7.5)  le numerateur de sante ne compte que des candidats, une fois : le temps 3 laisse
//	             les indices de bot aux temps de bot, et les inexpliques a indice de bot se
//	             comptent sur la population (plus sur le scan entier).
//	FK-6 (J7.6)  un kill-event lu deux fois dans le meme paquet (champs identiques) n est garde
//	             qu une fois, au bit le plus bas (`AssistStats.Doublons`) : plus de couple fabrique
//	             pour un kill orphelin voisin, plus de multi-attachement par doublon.
//	FK-7 (J7.7)  une carte fournie egale a l invariant (Cliffhanger) est une carte APPLIQUEE : plus
//	             de faux repli `repli_carte_absente_largeurs_par_defaut` ni de faux avertissement.
//
// LES LIGNES DE KILL DEJA EN BASE DEVIENNENT CANDIDATES AU BACKLOG DE REDECODAGE
// (`conditionBacklog`, `sync/killcollector/postsync.go`) : il est traite a la vague unique J11,
// geste de PRODUCTION pris par le pilote SUR SIGNAL UTILISATEUR (D6), jamais automatique.
// `SchemaVersion` ne monte pas (la forme du document de rejeu ne change pas) ; le codec des faits
// ne monte pas : la section 5 gagne des champs JSON, et un fichier ecrit sous la revision
// anterieure est refuse sur son EN-TETE, qui porte `killsource.Rev`.
//
// COMPLEMENT DU 2026-09-27, MEME RANG, MEME LOT NON PUBLIE (correctif J7 « carte obligatoire »,
// enquete ENQUETE_MARCHE_KILLSOURCE_2026-09-27). La revision NE MONTE PAS : `killsource-2026-09-26`
// n est pas publiee (la serie en base est `killsource-2026-09-24`), et deux changements d un meme
// lot partagent un rang — en ouvrir un second ferait redecoder le parc deux fois. L empreinte, elle,
// change (golden regenere par sa porte). Ce que la sortie gagne :
//
//	CARTE OBLIGATOIRE  `Decode` sans entree de catalogue portant des largeurs rend
//	                   `ErrCarteAbsente` ; plus aucun decodage aux largeurs de l invariant
//	                   (Cliffhanger) en production. Le collecteur met le film de cote
//	                   (`ecarte-carte-non-resolue`, `killsource_ecartes_carte_non_resolue`, aucun
//	                   marqueur de registre : il reste au backlog et sera decode quand sa carte se
//	                   resoudra). Le repli `repli_carte_absente_largeurs_par_defaut` est RETIRE du
//	                   registre ; seuls les instruments de recherche decodent sans carte, et le
//	                   DECLARENT (`Options.RechercheSansCarte`, interdit en production par ratchet).
//	CARTE LUE          la presence de la carte se lit sur l ENTREE (`carteApplicable`), plus sur une
//	                   difference de largeurs — acheve FK-7 sur l entree commise de Cliffhanger.
//
// LA MARCHE REVIENT SUR LE BANC, PAS EN PRODUCTION : le banc `TestGoldenFilms` decodait sans carte
// depuis `f3a2f00eb` ; sous leur carte, les quatre films rendent marche 356/369 et scan 7/8 au
// cumul (000d5950 94/91, 9b191a7f 84/80, 78919882 98/94, fccc61cd 93/91). En production, seuls les
// matchs SANS CARTE RESOLUE changent : ils etaient publies par le scan aux largeurs d une autre carte ;
// les NOUVELLES passes ne les publient plus (le match est mis de cote, il reste au backlog). LES
// LIGNES DEJA EN BASE RESTENT : celles d un match sans carte decode a `killsource-2026-09-24` (aux
// largeurs de Cliffhanger) demeurent dans `match_kill_events`, et aucune passe ne les remplace —
// le match n est plus jamais decode tant que sa carte ne se resout pas. Leur sort (purge ou non)
// est une decision renvoyee a J11 ; ce lot ne touche a aucune donnee.
//
// CORRECTIONS DE REVUE DU MEME JOUR, SANS EFFET SUR LA SORTIE (l empreinte ne bouge pas) : un match
// dont le registre ne porte ni `map_id` ni `map_name` (`port.ErrMatchMapUnknown`) est une carte NON
// RESOLUE, plus une panne retentee et telechargee a chaque cycle ; le post-sync ne relit plus la
// carte d un match deja constate sans elle sous le meme catalogue de bornes, pendant six heures au
// plus (registre en memoire, jauge `killsource_postsync_backlog_sans_carte`) et ne compte plus le
// backlog qu une fois par cycle. Le resolveur de carte du post-sync EMPRUNTE les metadonnees que le
// processus tient : un `map_name` reste UUID brut au registre est traduit comme au backfill hors
// ligne, au lieu d etre ecarte pour toujours (regression de la carte obligatoire, fermee le meme
// jour). Residu accepte : un film expire SANS carte n est jamais telecharge, donc jamais marque
// `MBitFilmAbsent`, et reste au backlog.

// ENTREE `killsource-2026-09-27` (2026-09-27, lot J5.5 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25) :
// LA REVISION MONTE DERRIERE LA GRAMMAIRE, PAR LA RECETTE DU SENS UNIQUE (ADR 0034 D-6 (2) et (3)).
//
// AUCUN OCTET DU PERIMETRE DE `killsource` N EST TOUCHE PAR LE JALON J5 hors `film/types`
// (renommage `EquipmentLifeKey` -> `LifeKey` du lot J5.1, deja recopie a revision constante). Ce
// qui monte est la VALEUR de `grammar.Rev` (`grammar-2026-09-24` -> `grammar-2026-09-27` : filtre
// de generation vivante GB-1, chaines d equipement par vie, slot de capture des records NEW), que
// la fermeture des imports de cette couche rencontre et hache. La chaine est mecanique : une montee
// de la couche du dessous remonte jusqu au backlog killsource, sans qu on ait a la plaider.
//
// CE QUE CE BACKLOG RAPPORTERAIT POUR LE KILL-FEED : RIEN ATTENDU, et c est argumente, non mesure
// par ce lot. `killsource` ne lit ni les positions bipedes, ni les huit canaux delta, ni les
// chaines d equipement que J5 change, et n installe aucun crochet de capture (le slot de capture
// pose sur un record NEW ne sert qu aux crochets et a l accumulateur, sans installateur en
// production). La preuve se lit au gate du superviseur (equivalence `killsource` sur `replay-equiv`
// et corpus gate).
//
// LES LIGNES DE `match_kill_events` DEJA EN BASE DEVIENNENT CANDIDATES au redecodage
// (`conditionBacklog`, `sync/killcollector/postsync.go`) : geste de PRODUCTION, pris par le pilote
// SUR SIGNAL UTILISATEUR (D6), jamais automatique. La meme vague (J11) rejoue de toute facon les
// faits d isolement, dont `IsolationDecoderRev` monte au meme lot.
//
// COMPLEMENT DU 2026-09-27 (lot J10.1 du meme plan, REVISION CONSTANTE) : la serie n a jamais ete
// publiee, son golden est regenere a revision constante par la recette. Ce qui change dans le
// perimetre : la VALEUR de `grammar.Rev` (`grammar-2026-09-27.3`) et cinq tris de cette couche
// rendus TOTAUX (DT-9) — les kill-events (instant, chunk, paquet, bit : `pickAssistHit` garde le
// premier porteur), les bots d un meme slot (ordre de decouverte, stable : l epinglage au
// kill-feed), les paquets de replication (instant, chunk, rang), les dead-states de la marche et les
// images-cles (ordre du film, stable), et l ordre des candidats de la bijection (le contenu du
// dead-state departage deux candidats sans position). Le kill-feed ne peut changer que sur des ex
// aequo que l ancien tri departageait au hasard.
//
// PORTE AUSSI J7 (fusion de `feat/suite-audit-decodeur-j7` du 2026-09-27) : `killsource-2026-09-26`
// n a JAMAIS ETE PUBLIEE SEULE — la serie en base reste `killsource-2026-09-24`, et ce rang
// `killsource-2026-09-27` est le premier publie qui contient les sept constats FK-1 a FK-7 et la carte
// obligatoire de l entree precedente, en plus de J5.5 et de J10. Un seul backlog de redecodage (J11)
// pour les deux entrees. A la fusion, les comptes de replis du lot J8.7 suivent les regles de J7 :
// le compte `repli_carte_absente_largeurs_par_defaut` disparait avec le repli, et celui de
// `repli_roster_indice_hors_bijection` juge le nom publie par le predicat unique
// `estNomDeRemplissage`.
//
// COMPLEMENT DU 2026-10-02 (retrait des replis nuls, decision DU-7, REVISION CONSTANTE) : deux replis
// a compte NUL sur le parc sortent de la couche (`.ai/V7.5/MESURES_PARC_REPLIS_NULS_2026-10-02.md`, passe
// killsource de la vague J11.4, 1 222 films). `repli_gamertag_par_xuid_brut` : un event du kill-feed
// dont le xuid n a aucun gamertag dans le bloc n entre plus dans le fil (il y entrait sous un nom
// `xuid:<N>` fabrique) ; `repli_roster_indice_hors_bijection` : son compte (noms « ? » publies, deja
// refuses par `pass.nomPubliable` depuis J7) et ses champs de `ReplisDuDecodage` disparaissent. LA
// SORTIE NE CHANGE SUR AUCUN FILM DU PARC : les deux conditions y comptent zero, donc chaque ligne de
// `match_kill_events` deja ecrite sous `killsource-2026-09-27` est celle que ce code ecrirait. La
// revision reste, AUCUN backlog n est ouvert ; golden regenere a revision constante. Seule la FORME du
// resultat observe change (`Stats.Replis` perd deux champs nuls).
//
// COMPLEMENT DU 2026-10-03 (vague 1 de la campagne de grammaire : lots L8, L3a, L4a et corrections
// de la revue de la vague, `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`, REVISION CONSTANTE) : la
// VALEUR de `grammar.Rev` monte jusqu a `grammar-2026-10-03.2`, donc l empreinte ; aucune source de
// la couche ne change. Sortie `cmd/killsource json` sur les 19 temoins de
// `config/replay_corpus.toml`, binaire de `67c379fc1` contre binaire de la vague : AUCUNE mort,
// aucune valeur, aucune voie ne change. Ne changent que des sorties NON PERSISTEES : le diagnostic
// d ORACLE `Result.Calibration` (lu par `cmd/killsource` seul) et, sur `111fa685`, deux compteurs de
// sante publies en expvar par `Health.ExpvarPairs` (`killsource_candidates_total` 226 -> 227,
// `killsource_unexplained_pair` 24 -> 25). Chaque ligne de `match_kill_events` deja ecrite sous
// `killsource-2026-09-27` est celle que ce code ecrirait : la revision reste et aucun backlog n est
// ouvert. Une montee rendrait tout le parc candidat (`conditionBacklog`), et le hook post-sync,
// installe par defaut (`sync/engine_options.go`), le redecoderait de lui-meme, huit films par cycle,
// pour reecrire des lignes identiques. Golden regenere a revision constante. Mesures :
// `campagne_grammaire_2026-10-01/vague1_tsv/`.
//
// COMPLEMENT DU 2026-10-04 (vague 2 de la campagne de grammaire apres sa revue adverse : lots LU et
// LT ; REVISION CONSTANTE) : une source de la couche change (`walk.go` appelle le localisateur
// unique de `grammar`, lot LU) et la VALEUR de `grammar.Rev` monte a `grammar-2026-10-03.5`, donc
// l empreinte. Sortie `cmd/killsource json` sur les 19 temoins de `config/replay_corpus.toml` et
// `1c4c63c2` (carte Refuge), binaire de `6fa631df0` contre binaire de la tete : IDENTIQUE A L OCTET
// sur les 20 films. Le lot LS, qui faisait monter la revision (voie de 229 morts du balayage a la
// marche), est retire de la vague : la revision reste et aucun backlog n est ouvert. Golden regenere
// a revision constante. Mesures : `campagne_grammaire_2026-10-01/vague2_tsv/revue/`.
//
// COMPLEMENT DU 2026-10-05 (lot VA de la campagne de grammaire, etape V1 : la vue A lue message par
// message par une seule lecture ; REVISION CONSTANTE) : aucune source de la couche ne change ; la
// VALEUR de `grammar.Rev` monte a `grammar-2026-10-06` et celle de `profile.Rev` a
// `profile-2026-10-06`, donc l empreinte. Sortie `cmd/killsource json` sur les 19 temoins de
// `config/replay_corpus.toml` et `1c4c63c2` (carte Refuge), binaire de `87cdfa761` contre binaire du
// lot : IDENTIQUE A L OCTET sur les 20 films. La marche de killsource ne lit pas la vue A
// (`DecodeFrameRecords` saute l amorce ; debut par `LocaliserBoucleDeRecords`, inchange). Golden
// regenere a revision constante. Mesures :
// `campagne_grammaire_2026-10-01/LOT_VA_V1.md`.
//
// COMPLEMENT DU 2026-10-06 (lot VA de la campagne de grammaire, etapes V2 et V3 : la fin de la vue A
// fixe le debut de la vue B, la variante de partie du film decide les genres 85 et 116 ; fusion de
// `feat/v75` a `fed1efed2` puis `b033d30f0`, corrections de la revue et decisions du pilote du
// 2026-10-06 ; REVISION CONSTANTE, DECISION D23 DU PILOTE DU 2026-10-06, MESUREE SUR 20 FILMS ET
// ESTIMEE SUR LE PARC) : une source de la couche change (`walk.go` part de [grammar.DebutDeLaVueB] ;
// la calibration porte la grammaire de vue A du film sous la carte du match,
// [grammar.VueADuFilmSousCarte]) et les VALEURS de `grammar.Rev` (`grammar-2026-10-06.4`) et de
// `profile.Rev` (`profile-2026-10-06.3`) montent, donc l empreinte. Sur la branche du lot seule, les
// etapes V2 et V3 avaient porte `grammar-2026-10-06.2` a `.4` et `profile-2026-10-07` : rangs jamais
// fusionnes, reunis a la fusion. La marche part de la fin de la vue A quand elle decide (film de
// classe EGALE : table native sous la version majeure 0x29 que le jeu joue ; film de classe PREFIXE,
// table prefixe ou table egale sous une autre majeure : si la marche depuis elle ferme le paquet),
// sinon du localisateur, inchange.
//
// MESURE : `cmd/killsource json` sur les 19 temoins de `config/replay_corpus.toml` et `1c4c63c2`
// (carte Refuge), binaire de `fed1efed2` contre binaire du lot, puis binaire de `b033d30f0` contre
// binaire du lot apres les decisions du pilote : aucune mort, aucune valeur, aucune voie ne change ;
// 19 films identiques a l octet, et sur `c75f33b8` le seul compteur de diagnostic
// `concordance.enregistrements_lus_par_les_deux_voies` (`Stats.Redundant`, non persiste) passe de 5 a
// 6. CE QUI N EST PAS MESURE : l entree de la marche change bel et bien (debuts de vue B, compteur
// ci-dessus), et le parc local compte, sur 1 657 films a section d identification, 1 401 films
// HI_1_13_0 de classe EGALE et 147 HI_1_12_0 de classe PREFIXE (table egale, majeure 0x28) ; que
// chaque ligne de `match_kill_events` deja ecrite sous `killsource-2026-09-27` soit celle que ce code
// ecrirait est MESURE sur 20 films et ESTIME sur le parc. Les lignes en base ne portent que
// `decoder_rev = killsource.Rev` : une ligne que ce lot changerait ne serait jamais recalculee. La
// revision reste et aucun backlog n est ouvert : c est la decision D23 du pilote du 2026-10-06, prise
// en connaissance de ce risque. Golden regenere a revision constante. Mesures :
// `campagne_grammaire_2026-10-01/LOT_VA_V2.md`, `LOT_VA_V3.md` (§15, §16).
//
// COMPLEMENT DU 2026-10-06 (lot « Rejeu : toute entree du roster a l equipe que le film ecrit »,
// `.ai/V7.5/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`, REVISION CONSTANTE) : `botmeta_equipe.go` lit
// l EQUIPE de chaque bot dans son entree BOT_METADATA (octet `+ 0xCE5` du bloc de 44 octets, par la
// grammaire de l ecrivain `FUN_14299bda0`) et la publie en `BotEntry.Team`, avec son bilan
// (`Roster.BotEquipes`) et deux diagnostics (`killsource.equipes_de_bots`, en erreur pour un bot
// sans equipe ; `killsource.entrees_de_bots`). L AGREGAT QUI EPINGLE LE ROSTER DU KILL-FEED EST LE
// MEME A L OCTET (memes bots, meme ordre, meme `NBots`) et aucune ligne de kill ne lit l equipe :
// sur les quatre films de reference et la mini-bobine, seules changent la ligne des bots
// (`343 Aloysius` et `343 PardonMy` : equipe 1) et la ligne neuve du bilan ; le cumul ne bouge pas.
// Aucune ligne de `match_kill_events` ne change : la revision reste et AUCUN BACKLOG n est ouvert,
// le meme choix qu au lot M2.1 (`BotEntry.Declarations`, suite du rang `killsource-2026-09-22.2`).
// Les faits portent l equipe en section 5 : c est `replay.SchemaDesFaits` qui monte (5 -> 6, apres
// la vue A), et la re-extraction des faits est un geste de `backfill-replay`. Golden regenere a
// revision constante.
//
// COMPLEMENT DU 2026-10-07 (branche `feat/zones-etat-initial`, REVISION CONSTANTE) : aucune source
// de la couche ne change ; la VALEUR de `grammar.Rev` monte a `grammar-2026-10-06.6` (voie image-cle
// de `grammar.ScanManagedProperties`, ti=13), donc l empreinte. Ni la marche ni la calibration de
// killsource n appellent ce balayage : sortie inchangee par construction. Golden regenere a revision
// constante.
//
// COMPLEMENT DU 2026-10-07 (branche `feat/zones-proprietaire`, REVISION CONSTANTE) : aucune source
// de la couche ne change ; la VALEUR de `grammar.Rev` monte a `grammar-2026-10-07` (nom `i0` des
// proprietes ti=13 lu aux images-cles par `grammar.ScanManagedProperties`), donc l empreinte. Ni la
// marche ni la calibration de killsource n appellent ce balayage : sortie inchangee par
// construction. Golden regenere a revision constante.
//
// COMPLEMENT DU 2026-10-07 (lot 2.7.c1 de la representation intermediaire, REVISION CONSTANTE) :
// les lectures de killsource hors de sa marche (table des joueurs, fil des kills, motif des xuid,
// BOT_METADATA, gabarit du dead-state) descendent dans la grammaire a l identique ;
// `KILLSOURCE_FIXTURES` identique sur les quatre films de reference.
//
// ENTREE `killsource-2026-10-07` (2026-10-07, lots 2.7.c2 a 2.7.c4 de la representation
// intermediaire) : KILLSOURCE EST UN CANAL DE LA MARCHE DES TRAMES.
//
// Ce qui change, contre `killsource-2026-09-27` :
//   - les dead-states viennent de la marche des trames ([grammar.LireLaMarcheDeKillsource]) : son
//     monde, ses debuts de vue B, et les listes qu elle ne localise pas, recuperees par le canal des
//     morts. La timeline et le filtre de la bande bipede sont retires (l archetype que la marche lie
//     au slot les remplace ; `repli_deadstate_hors_bande_bipede` sort du registre) ;
//   - le critere de la calibration se compte sous le monde des preliminaires de la marche : les
//     scores publies changent, la decision du mot de poignee reste l invariant ;
//   - 2.7.c3 : le decoupage du bloc MPP est celui que la grammaire resout pour le film (8/3 sur les
//     formats anciens), comme la cuisson ; un decoupage non resolu se dit
//     (`killsource.decoupage_mpp_non_resolu`) ;
//   - 2.7.c4 : les kill-events viennent de la vue A lue (le message de kill sans sa queue,
//     `grammar.Rev` `grammar-2026-10-07.2`), et la recherche bit a bit descend dans la grammaire en
//     rattrapage compte, borne aux trames dont la lecture de la vue A n est pas etablie, avant la vue
//     B (`repli_kill_rattrape_hors_vue_a`). killsource ne lit plus aucun octet : l origine des
//     instants est le plus petit horodatage des paquets de replication (`chunks.go`).
// PREUVE (2026-10-07) : 2.7.c2-c3, contre `killsource-2026-09-27` : sur les 19 temoins, contenu des
// 2 747 morts publiees identique, 411 changent de voie technique ; sur les 20 films d equivalence,
// seules `killsource`, `killRefs` et la couverture de l artefact bougent. 2.7.c4, contre c2-c3 :
// contenu identique sauf deux morts qui gagnent un kill-event ; 56 morts passent du balayage a la
// marche, 20 l inverse. Les lignes de `match_kill_events` deviennent candidates au redecodage :
// backlog sur signal de l utilisateur (D6).
//
// ENTREE `killsource-2026-10-07.2` (2026-10-07, fusion de `feat/v75` dans le lot des arrets de la
// vue B, `grammar` `.3` a `.8`) : aucune source de la couche ne change. Les cinq composants que la
// marche des trames lit desormais (`ti=43`, `ti=12` `i16` et `i18`, `ti=45` `i0`, `ti=10` `i2` a
// `i17`) prolongent la vue B de paquets ou killsource cherche ses dead-states : trois morts des 19
// temoins (`e5adf7b2` 05:29, `1c4c63c2` 08:15 et 13:38) passent du balayage a la marche
// (`read_path` `scan` -> `marche`), contenu publie identique ; les 17 autres temoins ne changent que
// par le diagnostic `calibration`. `read_path` est persiste : la revision monte (D23), backlog sur
// signal de l utilisateur (D6).
//
// COMPLEMENT DU 2026-10-08 (lot 2.7.d de la representation intermediaire, REVISION CONSTANTE) :
// `grammar.Rev` monte a `grammar-2026-10-08` (positions et objets du monde derriere la marche).
// `cmd/killsource json` sur les 19 temoins, binaire de `acfe4851a` contre binaire du lot :
// sorties identiques a l octet. Golden regenere a revision constante.
//
// ENTREE `killsource-2026-10-08` (2026-10-08, lot `feat/aj-film`, point 18 des ajustements) : LES
// MORTS QUI TOUCHENT UN BOT. NUMERO PROVISOIRE, a renumeroter a la fusion dans `feat/v75`.
//
// Ce qui change, contre `killsource-2026-10-07.2` :
//   - les morts DE bot et PAR un bot gardent leur premiere passe (premier candidat de la fenetre), puis
//     un COMPLEMENT reprend les seuls kills qu elle laisse sans ligne faute de dead-state libre, avec
//     les seuls dead-states qu aucun temps n a servis (`bot_affectation.go`) : il ajoute des lignes, il
//     n en retire aucune ;
//   - une mort de bot dont le dead-state designe le bot lui-meme (chute, source globale) se publie au
//     credit du feed, divergence levee — le temps 3 de l hybride, pour les bots ;
//   - le recollage d un kill ne prend que la PREMIERE mort voisine, et pas quand elle est deja
//     consommee, qu un kill-event 85 l ecrit d un autre tueur, ou que les dead-states la donnent a un
//     autre tueur ; il ne va jamais chercher la mort suivante (`repli_couple_recolle_sur_le_voisin`) ;
//   - le nom d un bot publie se lit a l instant de la ligne : sur un indice que plusieurs bots tiennent
//     l un apres l autre, celui que BOT_METADATA declare a cet instant (`hybrid_bots.go`).
// PREUVE (2026-10-08, `cmd/killsource json`, binaire de `6375eaf3c` contre binaire du lot) : sur
// `0a08d2f2`, 119 -> 124 lignes, +4 morts de bot (trois de SuSpec7c0br4, une de JGtm, dont une par
// la source du bot) et +1 mort infligee par un bot (Madina97294) ; `ca684191` 79 -> 80 (une mort
// infligee par un bot que le recollage prenait) ; `760fb768` une victime renommee a l instant
// (343 Cliffton -> 343 Chilies) ; `b1ad85eb`, `4f77afc1`, `50f16538` identiques. Aucune ligne
// existante ne change de tueur ni de source. Les lignes de `match_kill_events` deviennent candidates
// au redecodage : backlog sur signal de l utilisateur (D6).
