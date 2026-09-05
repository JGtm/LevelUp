package replay

// bomb_stats_document.go — LES CINQ STATISTIQUES D'OBJECTIF DE L'ASSAUT, POSÉES SUR LE DOCUMENT.
//
// # POURQUOI ELLES SE CALCULENT ICI, ET PAS CHEZ LEUR CONSOMMATEUR
//
// Le crochet post-sync (`sync/replayartifacts/bombstats.go`) ne fait QUE lire l'artefact RANGÉ
// et persister ce qu'il y trouve — la doctrine de `usage.go`. Il ne peut PAS appeler
// `BuildBombStats` lui-même : le noyau attend des entrées que le document NE PORTE PAS sous la
// forme voulue, et les re-dériver en ferait un SECOND décodeur du même fait, moins précis —
// l'anti-pattern que l'en-tête de bomb_stats.go condamne. MESURÉ, entrée par entrée :
//
//	Objectives            PORTÉ  `doc.Objectives` a bien Stat / XUID / TimeMS ;
//	Armings               PORTÉ  `doc.BombArmings`, MÊME type, MÊME horloge (celle du film) ;
//	Carry                 NON    `doc.BombCarries` est une projection sur la grille de FRAMES
//	                             (100 ms), qui ÉCARTE les périodes non pontées, ne distingue
//	                             pas un lâcher d'une mort, et ne publie pas `CarryMSByXUID` ;
//	FilmToMatchOffsetMS   NON    ni `originMs` ni `t0FilmMs` ne l'expriment ;
//	Kills                 NON    la paire tueur/victime n'est nulle part dans le document.
//
// Les quatre premières existent EN PLEINE FIDÉLITÉ à l'endroit exact où ce fichier travaille —
// `BuildFromPositions`, après `attachBombCarries` (qui rend la chronologie en millisecondes) et
// après `attachBombArmings` (qui publie les armements). Le calcul se fait donc ICI, UNE fois, et
// le résultat voyage dans l'artefact.
//
// AUCUNE ÉTAPE OBSERVÉE N'EST AJOUTÉE : ce fichier ne balaie rien, il projette des sources déjà
// décodées — comme `attachVehicleShots` ou les couvertures, que `BuildFromFilmSteps` ne couvre
// pas davantage. C'est le digest `artifact` du harnais qui les porte.
//
// # `bomb_carriers_killed` RESTE ABSENT, ET C'EST ÉCRIT PLUTÔT QUE COMBLÉ
//
// La statistique demande `[]KillRef` — une mort appariée à son TUEUR ET à sa VICTIME, datée sur
// l'horloge du MATCH. Cette forme n'existe nulle part dans la chaîne de cuisson : `opt.Kills` est
// un `[]EquipmentKillRef` (tueur + instant, PAS de victime, et sur l'horloge du FILM), et le seul
// producteur de `KillRef` du dépôt est `sync/killcollector/positions.go`, qui les lit dans
// `match_kill_events` — une table de BASE, que ce paquet n'ouvre jamais. `KillsRead` reste donc
// FAUX et le champ sort à `nil` chez TOUS les joueurs : « on n'a pas regardé », jamais un zéro
// qui se lirait comme une mesure. Report inscrit à `.ai/V7.5/REGISTRE_REPORTS.md`.

import "log/slog"

// attachBombStats calcule les cinq statistiques d'Assaut et les faits datés, et les pose sur le
// document.
//
// GARDE DE MODE : `opt.Bomb.CarryScanned`, posée par l'appelant (`replaybuild.isBombVariant`)
// sur TOUTE la famille bomb, One Bomb comprise. Hors de la famille : ni calque, ni couverture —
// la même règle que les autres calques d'objectif.
func attachBombStats(doc *ReplayDocument, opt Options, own OwnerReport, carry HeldObjectCarry) {
	if !opt.Bomb.CarryScanned {
		return
	}
	stats, events := BuildBombStats(BombStatsInput{
		// Le statborg a été décodé — `opt.Score` est l'entrée qui le porte, et l'appelant ne
		// la renseigne que quand le film a rendu des enregistrements d'entité — ET le mode est
		// de la famille bomb (la garde ci-dessus).
		DetonationsRead: opt.Score != nil,
		Objectives:      opt.Objectives,
		// Sans pont slot -> xuid, `attachBombCarries` ne reconstruit AUCUNE période : publier
		// des zéros affirmerait une mesure qui n'a pas eu lieu.
		CarryRead: len(own.SlotXUID) > 0,
		Carry:     carry,
		// ArmingsRead suit la CONFRONTATION LOCALE : un calque retenu à la source (garde 2,
		// tout-ou-rien) n'est pas « zéro armement », c'est une absence de lecture.
		ArmingsRead: bombArmingsRead(doc),
		Armings:     doc.BombArmings,
		// LE RECALAGE, exactement la dérivation écrite en tête de bomb_arms.go :
		// horlogeMatch = horlogeFilm + premierPaquetDuFilmUS/1000 − deathOffsetMS.
		FilmToMatchOffsetMS: int(int64(opt.FilmClockOriginUS)/1000 - own.DeathOffsetMS),
	})
	doc.BombStats = &stats
	doc.BombEvents = events
	logBombStats(doc.MatchID, stats.Coverage)
}

// bombArmingsRead dit si le calque des armements a été LU ET PUBLIÉ — balayage armé ET
// confrontation locale tenue. La couverture est l'autorité : un calque `Suppressed` a bien été
// lu, mais son contenu a été retenu tout entier, et l'attribution ne doit rien en tirer.
func bombArmingsRead(doc *ReplayDocument) bool {
	if doc.Coverage == nil || doc.Coverage.BombArmings == nil {
		return false
	}
	return doc.Coverage.BombArmings.Scanned && !doc.Coverage.BombArmings.Suppressed
}

// logBombStats publie la couverture au journal, témoins de lecture COMPRIS : sans eux, « 0
// armement attribué » ne se distingue pas de « le canal n'a pas été lu ».
func logBombStats(matchID string, c BombStatsCoverage) {
	slog.Info("rejeu : statistiques d'objectif de l'Assaut",
		"match_id", matchID, "joueurs", c.Players, "explosions", c.Detonations,
		"armements", c.Armings, "attribues", c.ArmingsAttributed,
		"parLacher", c.ArmingsByDrop, "parRepli", c.ArmingsByActiveCarry,
		"sansPorteur", c.ArmingsNoCarrier, "sansPont", c.ArmingsNoBridge,
		"ambigus", c.ArmingsAmbiguous, "periodes", c.Periods,
		"periodesSansPont", c.PeriodsNoBridge, "periodesOuvertes", c.PeriodsOpen,
		"lu_explosions", c.DetonationsRead, "lu_portage", c.CarryRead,
		"lu_armements", c.ArmingsRead, "lu_kills", c.KillsRead)
}
