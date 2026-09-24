package replay

// build_pistes.go — LES SIX PREMIERES PASSES DE L ASSEMBLAGE : les pistes et leur identite, les
// equipes et le roster, les calques du combat (tirs, projectiles, grenades), le score et les
// objectifs, les episodes d equipement actif, puis la composition de la couverture.
//
// DEPLACEMENT PUR depuis le corps de `BuildFromPositions` (lot 2.7 volet publication,
// 2026-09-16) : chaque passe est le bloc d instructions qu elle etait, dans le meme ordre, avec
// ses commentaires de mesure ; seule la designation des variables a change (`doc` -> `a.doc`).
// Voir `build.go` pour l ordre des passes et ce qu il protege.

import "log/slog"

// poserLesPistes construit le registre d identite, publie les trajectoires et les nomme.
//
// PREMIERE PASSE APRES L OUVERTURE, et tout le reste en depend : un calque ne publie que ce qui
// a une trajectoire publiee, et le nommage d une vie est ce qui relie un slot a un joueur.
func (a *assemblage) poserLesPistes() {
	// Les tirs sont rattachés sur les positions NON décimées (le rattachement se joue à
	// ~120 ms, la grille du rejeu est à 100 ms : décimer d'abord perdrait des tireurs).
	// LE PONT slot -> joueur vient du seul fil des morts (cf. owners.go). Il conditionne les
	// tirs ET les lancers : le construire une seule fois, et le partager.
	// Les TIRS entrent dans la construction du pont — non pour désigner un tireur (l'événement
	// porte déjà son auteur), mais parce que la fermeture A a besoin de savoir QUAND un joueur
	// agit sans avoir de corps nommé. Cf. closures.go.
	refs := fireRefs(a.fire)
	// LE REGISTRE D'IDENTITE EST LE SEUL PRODUCTEUR DE LIENS (lot P2, 2026-09-08). Il compose
	// les lectures directes (index de joueur, `bid`), le pont par morts et l'elimination sur le
	// roster, publie la provenance de chaque lien, et expose des accesseurs qui portent DEJA
	// leurs gardes — aucun calque ne reconstruit son propre pont (garde-rail `archlint`).
	a.reg = BuildIdentityRegistry(IdentityInput{
		Positions: a.sorted, BipedCreations: a.opt.BipedCreations,
		Deaths: a.opt.Deaths, PlayerIndices: a.opt.PlayerIndices, FilmTable: a.opt.FilmTable,
		Bots: a.opt.Bots, Fire: refs, RosterXUIDs: a.opt.RosterXUIDs,
		Participants: a.opt.Participants,
		Statborg: StatborgIdentityInput{
			Identity: a.opt.StatborgIdentity, Records: scoreRecordsOf(a.opt.Score)},
		Clock:     IdentityClock{OriginUS: a.origin, StepUS: a.step, FrameCount: a.doc.FrameCount},
		MatchID:   a.matchID,
		Fallbacks: a.opt.Fallbacks,
	})
	if !a.reg.Section.Empty() {
		a.doc.Identity = &a.reg.Section
	}
	// LA POSE DES TRACES ET DES BORNES vit dans `tracks_publication.go` (lot 1.9.13) : la
	// découpe vient des VIES du registre, et ce fichier-ci est déjà au-delà du seuil de 500 L.
	a.trackCov = poserLesTraces(&a.doc, a.sorted, decoupeDesTraces{
		origin: a.origin, step: a.step, minPoints: a.opt.minPoints(), scoped: a.opt.Scoped,
		vies: a.reg.Vies(), fb: a.opt.Fallbacks,
	})
	// CE QUE LA PORTE DES POSITIONS A ECARTE se publie avec la couverture des traces : un point
	// ecarte est un point que le seuil de publication n a jamais vu (cf. positions_porte.go).
	a.porte.poserSur(&a.trackCov, a.matchID)
	// L'IDENTITÉ se pose sur les traces dès que le pont existe : sans elle, un client ne peut
	// ni nommer un joueur, ni regrouper ses vies, ni colorer une équipe. Le nommage se fait
	// PAR VIE depuis le 2026-09-02 — un slot recyclé porte une identité par occupant.
	nameTracksByLives(a.doc.Tracks, a.reg.Vies(), a.origin, a.step, a.opt.Fallbacks)
	// LES BOTS ENTRENT APRÈS LES HUMAINS : une vie nommée par un xuid n'est jamais écrasée,
	// et seuls les slots que le pont attribue à un index de bot prennent son nom.
	nameBotTracks(a.doc.Tracks, a.reg.IndexParSlot(), a.opt.Bots)
	// LES RELAIS EN DERNIER : le remplaçant hérite des vies restées anonymes après tout ce
	// que la lecture et les fermetures savaient nommer (cf. successions.go).
	attributeSuccessions(a.doc.Tracks, a.opt.Successions, a.origin, a.step,
		a.reg.DeathOffsetMS(), a.reg.DeathOffsetMatches(), refs)
	// LE NOMMAGE FINAL, ET IL EST LA CONSEQUENCE D'UNE DECISION PRODUIT (2026-09-07) : « les vies
	// anonymes n'existent pas ; une vie est un humain ou un bot, point ». Ce qui reste sans nom
	// apres les quatre passes ci-dessus est un DEFAUT du pont, pas une categorie de donnee : il
	// se repare par l'OCCUPATION DU SLOT DANS LE TEMPS, et le residu se compte et s'alarme
	// (cf. unnamed_lives.go).
	a.unnamed = nameRemainingLives(a.doc.Tracks, a.reg, a.origin, a.step)
	// LES VIES QUE LE REGISTRE A DEDUITES rejoignent le residu de nommage : leur identite
	// etablit qu'un joueur etait la, jamais qu'un AUTRE n'y etait pas. Les lecteurs qui prouvent
	// une ABSENCE (le gate de presence des portages) doivent pouvoir s'en abstenir.
	for i := range a.reg.TracesDeduites(a.doc.Tracks, a.origin, a.step) {
		a.unnamed.deduced[i] = true
	}
	logUnnamedLives(a.matchID, a.doc.Tracks, a.unnamed)
}

// poserLesEquipesEtLeRoster pose l equipe du film sur les vies et sur le roster, puis les
// sieges. APRES le nommage : le xuid d une vie est ce qui la relie a son index de joueur.
func (a *assemblage) poserLesEquipesEtLeRoster() {
	// LE ROSTER VIENT DE LA TABLE EFFECTIVE DU REGISTRE, PAS DES OPTIONS (lot 1.6.2) : la table du
	// film y a deja pose ses sieges, et ses gamertags nomment les joueurs a ZERO MORT, que le fil
	// des morts ne peut pas nommer. Relire `opt.PlayerIndices` ici republierait la table d'AVANT la
	// composition — deux tables du meme film, ce que le registre existe pour empecher.
	// L'EQUIPE VIENT DU FILM, ET DE LUI SEUL (lot 1.7, decision utilisateur V4). Elle se pose
	// sur les vies ET sur le roster ; la base n'entre que dans `coverage.teams` comme CONTROLE.
	// Posee APRES le nommage : le xuid d'une vie est ce qui la relie a son index de joueur.
	a.equipes = newTeamPublication(a.reg, a.opt.PlayerTeams, a.opt.TeamScan, a.opt.ScoreboardTeams)
	a.viesTotal, a.viesNommees, a.viesSlotAmbigu = a.equipes.poserSurLesTraces(a.doc.Tracks)
	a.doc.Roster = buildRoster(a.reg.TableDIndex(), nomsDesJoueurs(a.reg, a.opt.Deaths), a.opt.Bots, a.equipes)
	// LE SIEGE APRES LE ROSTER ET APRES LES TRACES, parce qu'il a besoin des deux : l'index lu
	// pour le siege, les vies publiees pour savoir qui libere et qui arrive (cf. sieges.go).
	a.siegeCov = poserLesSieges(a.doc.Roster, a.doc.Tracks, a.opt.FilmTable, a.doc.FrameCount, a.opt.Fallbacks)
	// L'ORIGINE se publie APRÈS le pont : son témoin (le calage du fil des morts) en sort.
	a.doc.OriginMs = resolveOriginMs(a.origin, a.opt.FilmClockOriginUS, a.reg.DeathOffsetMS(), a.reg.DeathOffsetMatches())
	a.reg.logRegistry(a.matchID)
}

// poserTirsProjectilesEtGrenades publie les trois calques du combat, chacun avec sa couverture.
func (a *assemblage) poserTirsProjectilesEtGrenades() {
	// Chaque calque rend sa COUVERTURE en même temps que son contenu. Le filtrage par
	// trajectoire publiée qui suit est lui aussi compté, sous une catégorie distincte.
	var shots []Shot
	shots, a.shotOrphans, a.shotCov = buildShots(a.sorted, a.fire, a.origin, a.step, a.reg.IndexParSlot())
	a.doc.Shots = keepShotsOfPublishedTracks(shots, a.doc.Tracks)
	a.shotCov.Unpublished = countUnpublished(len(shots), len(a.doc.Shots))
	a.shotCov.Attached = len(a.doc.Shots)
	a.shotCov.warnIfLossy("tirs")

	a.doc.Loadouts = keepLoadoutsOfPublishedTracks(buildLoadouts(a.opt.Loadouts, a.origin, a.step), a.doc.Tracks)

	// Les projectiles se construisent AVANT les lancers : le lancer publie son lien vers le
	// projectile né de lui (Grenade.Proj), qui pointe un index de la tranche PUBLIÉE.
	var pubProjByRaw map[int]int
	var projTronquees int
	a.doc.Projectiles, pubProjByRaw, projTronquees = buildProjectiles(a.opt.Projectiles, a.origin, a.step)
	// LA COUVERTURE DES PROJECTILES EST CONSTRUITE ICI mais POSEE plus bas : `doc.Coverage`
	// n'existe qu'apres `buildCoverage`. Nil quand aucune piste n'est fournie — un film sans
	// projectile ne publie pas trois zeros.
	if len(a.opt.Projectiles) > 0 {
		a.projCov = &ProjectileCoverage{
			Tracks: len(a.opt.Projectiles), Published: len(a.doc.Projectiles), Truncated: projTronquees,
		}
		if projTronquees > 0 {
			// JOURNALISE, JAMAIS AVALE (regle n°3) : une coupure protege le rendu, elle ne
			// repare pas la dequantification qui la cause.
			slog.Info("rejeu : trajectoires de projectile coupees a un pas impossible",
				"match_id", a.matchID, "tronquees", projTronquees, "pistes", len(a.opt.Projectiles),
				"seuil_m", projectileMaxStepM)
		}
	}

	var gren []Grenade
	gren, a.grenCov = buildGrenades(a.sorted, a.opt.Grenades, a.origin, a.step, a.reg.IndexParSlot(), a.opt.Projectiles, pubProjByRaw)
	a.doc.Grenades = keepGrenadesOfPublishedTracks(gren, a.doc.Tracks)
	a.grenCov.Unpublished = countUnpublished(len(gren), len(a.doc.Grenades))
	a.grenCov.Attached = len(a.doc.Grenades)
	a.grenCov.warnIfLossy("grenades")
}

// poserScoreEtObjectifs mesure la couverture des equipes, etablit l horloge du score et pose les
// actions d objectif puis la courbe de score.
func (a *assemblage) poserScoreEtObjectifs() {
	// LA COUVERTURE DES EQUIPES SE CONSTRUIT ICI mais SE POSE plus bas, avec les autres :
	// `doc.Coverage` n'existe qu'a partir de `buildCoverage`. Elle a besoin du roster, qui est
	// son denominateur.
	a.teamCov = a.equipes.couverture(a.viesTotal, a.viesNommees, a.viesSlotAmbigu, a.doc.Roster)
	logTeamCoverage(a.matchID, a.teamCov)
	a.clock = replayScoreClock(&a.doc, a.interval, a.matchID)
	a.objCov = attachObjectiveActions(&a.doc, a.opt, a.reg, a.clock)
	a.scoreCov = attachScoreTimeline(&a.doc, a.opt, a.clock, a.matchID)
}

// poserEpisodesDEquipement publie les episodes d etat actif (camo, surbouclier) et les frags
// qu ils portent. AVANT la couverture, qui publie leur compte.
func (a *assemblage) poserEpisodesDEquipement() {
	// L'ETAT ACTIF des deux familles mesurees (camo, surbouclier) : episodes dates par
	// vie, fermes a la mort quand rien n'a mesure la fin (cf. equipment_episodes.go).
	// Le surbouclier se lit dans les positions NON decimees : la decimation garde un
	// echantillon par frame et perdrait des transitions. Construit AVANT la couverture,
	// qui publie son compte.
	// LA BORNE DES EPISODES EST LA MORT, PAS LE NOM (correctif E2-bis) : le registre dit quelles
	// vies une mort LUE termine, les seules frontieres que la couture d'un silence de replication
	// ne franchit pas (cf. equipment_episodes.trackFrameWindows).
	a.clotureesParMort = a.reg.TracesCloturesParMort(a.doc.Tracks, a.origin, a.step)
	var camoNonBinary int
	a.doc.EquipmentEpisodes, camoNonBinary = buildEquipmentEpisodes(a.sorted, a.opt.CamoStates, a.origin, a.step,
		a.doc.Tracks, a.clotureesParMort)
	if camoNonBinary > 0 {
		slog.Warn("rejeu : lectures camo NON BINAIRES ignorees — l'interrupteur mesure ne connait que 0 et 4095",
			"lectures", camoNonBinary)
	}
	// Les FRAGS SOUS EFFET ACTIF : jointure des episodes avec les kills resolus par
	// l'appelant (cf. equipment_episode_kills.go). AVANT la couverture, qui publie
	// killsRead a cote des compteurs.
	// LES ETATS DE MOUVEMENT (schema 65) : le MEME pliage d intervalles que les episodes
	// ci-dessus — memes fenetres de vie, meme cloture a la mort — sur les transitions lues
	// d `i29`, `i62` et `i54`. Rien de devine : le SPRINT est refute et le SAUT n est pas
	// prouve (cf. `document_stances.go`).
	a.doc.Stances, a.stanceCov = buildStances(stanceInputs{
		reads: a.opt.MovementStates, stats: a.opt.MovementStateStats,
		origin: a.origin, step: a.step, tracks: a.doc.Tracks,
		closedByDeath: a.clotureesParMort,
	})
	slog.Info("rejeu : etats de mouvement",
		"balaye", a.stanceCov.Scanned, "absent", a.stanceCov.Absent,
		"records", a.stanceCov.Records, "desyncs", a.stanceCov.Desyncs,
		"lectures", a.stanceCov.Reads, "intervalles", a.stanceCov.Intervals,
		"parGenre", a.stanceCov.ByKind, "vies", a.stanceCov.Lives,
		"viesPubliees", a.stanceCov.TracksTotal, "ecartees", a.stanceCov.Dropped,
		"largeursCarte", a.stanceCov.MapWidths)
	a.killsRead = attachAllEquipmentKills(a.doc.EquipmentEpisodes, a.opt.Kills, occupantParFrame(a.reg, replayClock{origin: a.origin, step: a.step, fb: a.opt.Fallbacks}), a.doc.OriginMs, a.interval)
}

// composerLaCouverture assemble `doc.Coverage` et y pose les mesures faites par les passes
// precedentes, puis le coup d envoi. C EST ICI QUE `doc.Coverage` NAIT : toute passe qui mesure
// avant elle garde sa mesure et la pose apres.
func (a *assemblage) composerLaCouverture() {
	a.doc.Coverage = buildCoverage(a.shotCov, a.grenCov, a.objCov, a.reg, a.doc.OriginMs != nil, a.scoreCov)
	a.doc.Coverage.Projectiles = a.projCov
	// CE QUE LE SEUIL DE PUBLICATION A REFUSE (schema 55) : mesure faite en tete de fonction,
	// posee ici. Sans elle, un artefact publiant 90 traces la ou le film en porte 95 etait
	// indistinguable d'un film a 90 vies.
	a.doc.Coverage.Tracks = &a.trackCov
	// CE QUE LE FILM DIT DES EQUIPES, et ce que la base en pense (lot 1.7) : mesure faite
	// ci-dessus, posee ici.
	a.doc.Coverage.Teams = &a.teamCov
	// LES ETATS DE MOUVEMENT (schema 65) : mesure faite au pliage des pistes, posee ici.
	a.doc.Coverage.Stances = &a.stanceCov
	// CE QUE LA POSE DES SIEGES A LU ET APPARIE (lot 1.9.14) : mesure faite ci-dessus, posee ici.
	a.doc.Coverage.Seats = &a.siegeCov
	// La version du film est une DIMENSION du décodage : elle voyage avec l'artefact plutôt que
	// d'exiger une relecture du film pour la retrouver (cf. Coverage.FilmMajorVersion).
	a.doc.Coverage.FilmMajorVersion = a.opt.FilmMajorVersion
	// SOUS QUELLES REVISIONS CET ARTEFACT A ETE CUIT (schema 61, lot 2.6.3). Pose ICI et pas dans
	// `BuildFromFilm` : tout ce que ce code cuit doit porter le bloc, y compris un document
	// assemble depuis des positions sans film — l ABSENCE du bloc est reservee aux artefacts
	// anterieurs au schema 61, et lui donner un second sens rouvrirait l ambiguite que D-7
	// interdit.
	a.doc.Coverage.Decoder = couvertureDuDecodeur(a.opt.FilmIdentity)
	// LE RESIDU DE NOMMAGE SE PUBLIE AVEC LE PONT : un artefact qui porte des vies sans identite
	// doit le DIRE, sans quoi le defaut ne se voit que dans les journaux du jour de la cuisson.
	a.doc.Coverage.Bridge.NamedByPreviousLife = a.unnamed.byPrevious
	a.doc.Coverage.Bridge.NamedByNextLife = a.unnamed.byNext
	a.doc.Coverage.Bridge.NamedBySlotBridge = a.unnamed.byBridge
	a.doc.Coverage.Bridge.UnnamedLives = a.unnamed.remaining
	a.doc.Coverage.Bridge.UnnamedLivesContested = a.unnamed.contested
	// LE VERDICT DU FIL DES MORTS (schema 69, lot M5.2) : vide et illisible ne sont pas la meme
	// chose, et le document le dit (cf. BridgeHealth.DeathsFeed).
	a.doc.Coverage.Bridge.DeathsFeed = deathsFeedPublie(a.opt.DeathsFeed, len(a.opt.Deaths))
	// La couverture des episodes d'equipement se publie AVEC eux : « N episodes » sans
	// « sur M vies » se lirait comme une exhaustivite.
	a.doc.Coverage.Equipment = equipmentCoverage(a.doc.EquipmentEpisodes, a.doc.Tracks, a.clotureesParMort)
	a.doc.Coverage.Equipment.KillsRead = a.killsRead
	// CE QUE CHAQUE VOIE DE LECTURE DES MORTS A PROPOSE (schema 62, lot 4.2.1-b) : la mesure vient
	// de l APPELANT, qui a decode les morts, et elle se pose ICI parce qu elle qualifie le meme
	// decodage que `KillsRead` juste au-dessus. nil reste nil : le bloc absent dit « non lu ».
	a.doc.Coverage.DeathsPaths = a.opt.Kills.Paths
	// LE COUP D'ENVOI, date par le premier mouvement des pistes (cf. t0_film.go). Il se pose
	// APRES la couverture et non a cote d'`OriginMs` (l. 528) pour deux raisons : son verdict
	// vit dans `doc.Coverage`, qui n'existe qu'ici, et il se calcule sur les pistes PUBLIEES,
	// posees juste au-dessus. SANS ORIGINE, PAS DE COUP D'ENVOI : le resultat est un instant
	// sur l'horloge du fil, et sans origine cette horloge n'est pas etablie — publier une
	// mesure calee sur zero la rendrait fausse de 3,6 s a 50,8 s selon le match.
	if a.doc.OriginMs != nil {
		a.doc.T0FilmMs, a.doc.Coverage.T0Film = DetectT0Film(
			t0FilmTracksOf(a.doc.Tracks), a.interval, *a.doc.OriginMs, a.matchID)
	}
}
