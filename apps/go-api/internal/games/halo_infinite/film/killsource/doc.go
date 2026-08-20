// Package killsource decode, a partir des chunks d un film Theater Halo Infinite et de RIEN
// D AUTRE, la SOURCE DU DEGAT FATAL de chaque mort — et il la publie A COTE de ce que le
// kill-feed du jeu affiche, jamais a la place.
//
// # LA DOCTRINE, NON NEGOCIABLE
//
// L arme d un kill est la SOURCE DU DEGAT. Jamais l arme TENUE par le tueur au moment du kill,
// jamais le CREDIT que le jeu attribue. Le champ lu est `tag+0x00` du composant dead-state i11
// de la VICTIME (RE_LOG 7ter.35, confirme 9/9 en mode Theater) : c est un tag de groupe `jpt!`
// (damage_effect) des archives du jeu, etabli 59/59 sur quatre films sans une exception.
//
// # LES DEUX VERITES — C EST LA DECISION D ARCHITECTURE DE CE PAQUET
//
// Le kill-feed et le dead-state ne repondent PAS a la meme question :
//
//	KILL-FEED    QUI RECOIT LE CREDIT. Le jeu credite le dernier joueur ayant inflige des
//	             degats, MEME quand la mort vient d une source appartenant a la victime.
//	             C est ce que le joueur voit a l ecran, et ce sur quoi ses statistiques
//	             officielles sont baties.
//	DEAD-STATE   QUELLE EST LA SOURCE DU DEGAT fatal. Roquette tiree trop pres, baril lance
//	             trop pres, chute : la source appartient alors a la VICTIME.
//
// Les deux sont VRAIS. Quand ils divergent — 8 cas mesures sur les quatre films de reference,
// confirmes 8/8 en Theater (RE_LOG 7ter.63, 7ter.66) — LA DIVERGENCE EST ELLE-MEME UNE
// INFORMATION. Exemple, dans les mots de l utilisateur : << le kill feed attribue le kill a
// HizaroMne4262 alors que c est MOI qui me tue en tirant une roquette sur un mur >>.
//
// DONC, dans [Kill] : [Kill.Feed] et [Kill.Source] sont au MEME NIVEAU, [Kill.Diverges] signale
// leur desaccord, et AUCUNE des deux n est presentee comme corrigeant l autre. Un consommateur
// qui ne veut que le kill-feed lit [Kill.Feed] et ignore tout le reste ; un consommateur qui ne
// veut que la source lit [Kill.Source] et ignore [Kill.Feed]. Les deux sont toujours remplies.
//
// # CE QUE LE BRANCHEUR A A ECRIRE
//
// Une seule fonction publique, [Decode]. Le cablage complet tient en quelques lignes :
//
//	src := killsource.MemoryChunks(chunks)              // les chunks deja telecharges
//	res, err := killsource.Decode(ctx, matchID, src, nil) // nil = la configuration GELEE
//	if err != nil {
//	    return err                                       // errors.Is(err, killsource.ErrNoKillFeed), ...
//	}
//	for _, k := range res.Kills {
//	    // k.Victim            : la victime
//	    // k.Feed.Killer       : le joueur credite par le JEU        <- VERITE KILL-FEED
//	    // k.Source.Display    : l etiquette de la source du degat   <- VERITE SOURCE
//	    // k.Source.Tag        : l identifiant brut, independant de toute table de nommage
//	    // k.Diverges          : true => la source appartenait a la VICTIME
//	    // k.Read.Origin       : killsource.OriginBotKiller => LE TUEUR EST UN BOT et il vient du
//	    //                       roster de replication, pas du kill-feed. A EXCLURE si le
//	    //                       consommateur ne veut que ce que le kill-feed nomme lui-meme.
//	    // k.Read.Path         : killsource.PathWalk (98.2 %) ou PathScan (78.4 %) — a ponderer
//	    // k.Assist.Known      : FAUX => ON NE SAIT PAS. Ne JAMAIS le lire comme « pas d assistant »
//	    // k.Assist.Name       : l assistant ; vide AVEC Known => « pas d assistant », MESURE
//	    // k.Assist.Extra      : assistants en surplus sur cette mort — 0 attendu, a surveiller
//	    // k.KillerDamage      : part de degats du tueur     ; .Known = false => NON MESURE
//	    // k.AssistDamage      : part de degats de l assistant ; .Known = false => NON MESURE
//	}
//	for _, d := range res.UnclaimedDeaths {
//	    // LES MORTS QUE PERSONNE NE REVENDIQUE — liste SEPAREE, jamais melangee aux kills :
//	    // il n y a pas de credit a porter, et en inventer un serait un mensonge.
//	    // d.Victim / d.VictimXUID : la victime (le xuid est la seule cle de jointure fiable)
//	    // d.Source.Class          : DEGAT_GLOBAL (chute, environnement, hors-limites) ou la
//	    //                          classe de sa PROPRE source — c est ce qui distingue une
//	    //                          chute d une roquette tiree trop pres d un mur
//	    // Population RARE et mesuree : 0 sur les quatre films de reference, 1 a 2 par match
//	    // ailleurs (mesure du 2026-08-14).
//	}
//	for _, p := range res.Health.ExpvarPairs() {
//	    observability.AddInt(p.Name, p.Value)            // ADR 0009, compteurs entiers
//	}
//
// `killsource.DirChunks(dir)` remplace `MemoryChunks` pour rejouer un film depuis le disque.
// Avant de publier ligne par ligne, tester `res.LineByLinePublishable()` : il refuse quand la
// bijection indice -> joueur n a pas de marge (BTB) ou quand la sante est en ALERTE. CETTE PORTE
// VAUT AUSSI POUR L ASSISTANT ET LES DEUX PARTS DE DEGATS : ils sont nommes par la MEME bijection
// que le reste, il n existe donc volontairement aucune porte separee.
//
// # TROIS ETATS, JAMAIS DEUX — LA SEMANTIQUE LA PLUS FACILE A MAL LIRE DU PAQUET
//
// L assistant et les deux parts de degats ne se lisent PAS comme des champs nullables ordinaires.
// << ON NE SAIT PAS >> et << IL N Y EN A PAS >> sont deux faits differents, et les confondre
// fabrique des << 0 assist >> qui n ont jamais ete observes :
//
//	k.Assist.Known == false            ON NE SAIT PAS — aucun kill-event attache a cette mort.
//	                                   `k.KillerDamage` et `k.AssistDamage` sont alors NON MESURES.
//	Known == true, Name == ""          PAS D ASSISTANT, et c est une MESURE. `k.AssistDamage` reste
//	                                   NON MESURE : sans assistant, ce bloc du film porte une
//	                                   constante PAR FILM (149, 70, 20, 197 selon le film) — 74 %
//	                                   des lignes attachees sont dans ce cas, et 20 se lit comme un
//	                                   pourcentage parfaitement credible.
//	Known == true, Name renseigne      un assistant NOMME ; `k.AssistDamage` est mesuree.
//	Known == true, Rejected non vide   le champ designait quelqu un qu on REFUSE de nommer
//	                                   (`Assist.Name` vide). La part EST mesuree, son porteur ne
//	                                   l est pas : ne pas l ecrire en base sans nom.
//
// AUCUNE DES DEUX PARTS N EST PLAFONNEE A 100 : 1.7 % des kill-events attaches a de vraies morts
// nommees depassent 100, jusqu a 228. Une valeur > 100 est une DONNEE REELLE, pas une lecture
// ratee — son interpretation (degat excedentaire) n est PAS etablie.
//
// CE PAQUET EST IMPORTE PAR L APPLICATION : `internal/replaybuild/replaybuild.go`
// (`neutralDeaths`, via `killsource.DirChunks` + `killsource.Decode`) l utilise pour typer
// les lignes de mort neutres de l artefact de rejeu 2D — brique partagee par
// `cmd/replay-build`, `levelup backfill-replay`, l action admin replay-build et l etape
// post-sync locale (cf. l en-tete de `replaybuild`). Le reste (`cmd/killsource`) est un
// outil. Il ne touche ni la base, ni le reseau, ni les fichiers du jeu : il decode des
// chunks deja mis en cache sur disque par un producteur anterieur. Une ecriture per-match
// issue de ce decodage devra passer par `internal/persist/BatchBuilder` (regle anti-ART,
// ADR 0019/0030).
//
// # OFFLINE PUR — ET C EST PROUVE, PAS AFFIRME
//
// Le catalogue des 468 ids `jpt!` et la table de nommage sont EMBARQUES par `go:embed` dans
// `internal/games/halo_infinite/film/damagetag` (63 Ko). Preuve faite en renommant sur le disque
// les deux racines du jeu (`Content/deploy`, `Content/Sound`) : sorties identiques au bit, meme
// SHA-256 (RE_LOG 7ter.68). Le serveur de production n a pas Halo installe, et n en a pas besoin.
//
// # PORTEE DES CHIFFRES — LES DENOMINATEURS SE NOMMENT
//
// Mesure du 2026-07-27 (RE_LOG 7ter.70, verifie 7ter.71 ; hybride 7ter.72, verifie 7ter.73), sur
// QUATRE films a 8 joueurs :
//
//	371 / 371 couples REELS                                     = 100.0 %
//	371 / 372 couples reconstruits par la reconstruction de feed =  99.7 %
//	371 / 375 morts du KILL-FEED (aucune mort de bot : le chunk HIGHLIGHT est humain-seul)
//	                                                             =  98.9 %
//	380 / 380 morts de l API (+5 morts DE bot, +4 morts PAR un bot) = 100.0 %
//
// NE JAMAIS ECRIRE << X % DES MORTS >> SANS DIRE LEQUEL. [Coverage] les porte tous les quatre.
//
// UN SEUL DE CES QUATRE TAUX A CHANGE AVEC RE_LOG 7ter.79, ET C EST SON NUMERATEUR, PAS SON
// DENOMINATEUR : les morts de l API valaient **376 / 380 = 98.9 %**, elles valent **380 / 380 =
// 100.0 %**. Les 4 qui manquaient sont les morts INFLIGEES PAR un bot ([OriginBotKiller]), que
// personne ne decodait. LES TROIS AUTRES TAUX SONT INCHANGES, NUMERATEUR COMPRIS — la nouvelle
// population n entre dans AUCUN d eux, et [Coverage.Covered] ne la compte pas.
//
// TROIS POPULATIONS DISJOINTES, ET LA COMPTABILITE SE FERME FILM PAR FILM :
//
//	humain tue par un humain  le feed porte le kill ET la mort   -> [Coverage.Covered]
//	BOT tue par un humain     le feed porte le kill, pas la mort -> [Coverage.BotDeaths]
//	humain tue par un BOT     le feed porte la mort, pas le kill -> [Coverage.BotKillerDeaths]
//
// Le cout se lit PAR VOIE, et il est publie ligne par ligne dans [Provenance] :
// gate (b) 98.2 % (334/340) pour la MARCHE, 78.4 % (29/37) pour le RATTRAPAGE DU SCAN. Un
// consommateur qui pondere doit regarder [Provenance.Path].
//
// RESERVE SUR CE SPLIT, NON LEVEE : la bijection indice -> joueur est ajustee sur l UNION des
// deux voies, dont la marche fournit 91 % des candidats. Un objectif maximise CONJOINTEMENT
// n est pas neutre — le 98.2 % peut etre flatte et le 78.4 % penalise par cet effet seul
// (RE_LOG 7ter.73 (5)). LIRE CE COUPLE COMME UNE VENTILATION DU COUT, jamais comme deux
// precisions directement comparables. Le controle qui leverait la reserve est nomme et non
// execute : re-ajuster la bijection sur les seuls enregistrements du rattrapage.
//
// # DECISION DE PERIMETRE — POURQUOI LA MARCHE EST PORTEE ALORS QUE LE SCAN SUFFIRAIT
//
// C est une DECISION, pas une consequence subie, et elle merite d etre lue avant d alleger le
// paquet. Le SCAN DIRECT (RE_LOG 7ter.60) a rendu la marche, la calibration et le localisateur
// slot-123 FACULTATIFS pour la question << quelle source a tue >>. Porter la marche coute donc
// tout ce que le scan avait rendu inutile : `walk.go`, `calibrate.go`, `world.go`, et la
// contrainte d execution serialisee qui va avec.
//
//	CE QUE CE COUT N ACHETE PAS  ni couverture ni precision globale. Le gate (b) global ne bouge
//	                             pas d un centieme : `|marche| = redondants` sur les quatre films
//	                             (346/346), la marche est un SOUS-ENSEMBLE POSITIONNEL TOTAL du
//	                             scan, donc l hybride consulte le MEME ensemble (7ter.73 (2)).
//	CE QU IL ACHETE              (1) la RECUPERATION sous catalogue perime — ablation d un tag
//	                             reel : 20 lignes perdues au lieu de 224, facteur 11.2 ; (2) la
//	                             DETECTION de ce meme catalogue perime, que seule la marche peut
//	                             voir puisqu elle n a pas de porte de catalogue ; (3) l etiquette
//	                             d ORIGINE de chaque ligne, sans laquelle le cout ne serait pas
//	                             ventilable.
//	POURQUOI ON PAIE             le catalogue est EMBARQUE (le serveur n a pas le jeu) et il
//	                             VIEILLIRA. La detection seule ne recupere rien : sans la marche,
//	                             on saurait que la table est perimee sans pouvoir publier les
//	                             lignes concernees. Et l identite ci-dessus TOMBE des que le
//	                             catalogue est incomplet : +0.70 point mesure en faveur de
//	                             l hybride sur catalogue ampute.
//	CE QUE LA DECISION NE COUVRE le gain de 11.2x vaut en LIGNES PUBLIEES, PAS EN ANCRES de
//	PAS                          verite terrain : 5 des 30 ancres sont servies par le SCAN SEUL et
//	                             tombent malgre l hybride (7ter.73 (4b)).
//
// # CE QUI EST HORS PERIMETRE, ET POURQUOI
//
//   - LE BTB (24 slots / 36 participants). Le decodage y tient en AGREGAT (224/293 = 76.5 %)
//     mais la bijection indice -> joueur y a une marge NULLE (RE_LOG 7ter.53) : aucune
//     attribution ligne par ligne n y est publiable. [Result.LineByLinePublishable] le dit.
//     Decision utilisateur : perimetre 4v4. Le code ne casse pas en BTB, il ne l optimise pas.
//     L ASSISTANT NE FAIT PAS EXCEPTION, et la mesure est nette : 62 assistants nommes pour 122 a
//     l API en BTB (51 %), contre 17/17 et 29/29 en Arena.
//   - LE NOMMAGE DES SOURCES SANS NOM. 206 des 468 tags ne remontent a aucune source ; ils
//     s affichent [LabelOther]. Un encodage existe quelque part, ce n est pas la priorite.
//   - LA PEREMPTION DU CATALOGUE. Risque MINEUR (les ajouts d armes sont rares). La DETECTION
//     reste posee et testee ([Health]), elle n est pas un entretien regulier.
//
// # CONTRAINTE D EXECUTION — UN SEUL DECODAGE A LA FOIS DANS UN PROCESS
//
// Les parametres de replication de `filmdec` sont des GLOBAUX DE PAQUET. Enchainer deux films
// dans le meme process contamine la calibration du second (mesure : le score de fccc61cd passe
// de 1111 a 1214 selon l ordre d appel, RE_LOG 7ter.52). [Decode] serialise donc les passes par
// un verrou de paquet et remet les globaux a leur valeur d origine a chaque entree. Le cout est
// de **8 a 30 secondes par film** (mesure : 8.2 / 11.2 / 28.6 / 11.1 s sur les quatre films de
// reference). Le << ~22 ms >> qui figurait ici venait d un autre chantier et portait sur une passe
// bien plus courte : il justifiait a tort de negliger la serialisation des decodages.
package killsource
