package replay

import "log/slog"

// build_score.go — LE CABLAGE DU CALQUE DE SCORE DANS L'ASSEMBLAGE.
//
// POURQUOI UN FICHIER A PART (revue R1, 2026-08-18) : `build.go` depasse deja 500 lignes, et la
// regle du depot est de ne pas accroitre cette dette. Le modele est celui de
// `build_ground_weapons.go` — l'assemblage garde UNE ligne par calque, le detail vit a cote de
// la donnee qu'il produit. `BuildFromPositions` n'y gagne pas seulement en longueur : les trois
// pieces du score (horloge, calque, couverture) tiennent desormais ensemble.

// replayScoreClock construit l'horloge qui pose sur la grille de frames tout ce qui est date
// depuis l'HORLOGE DU FILM — les actions d'objectif et la courbe de score.
//
// C'est ici que l'origine est retranchee (report `:123` du registre) : les evenements sont dates
// depuis le PREMIER PAQUET DU FILM, la grille compte depuis le premier paquet de POSITION, et
// l'ecart entre les deux zeros est exactement `originMs`. Quand l'origine n'est pas etablie, la
// soustraction se fait avec zero et `coverage.originResolved` le DIT (cf. origin.go).
func replayScoreClock(doc *ReplayDocument, intervalMS int, matchID string) scoreClock {
	return scoreClock{
		intervalMS: intervalMS,
		frames:     doc.FrameCount,
		originMS:   originMSOf(doc.OriginMs, matchID),
	}
}

// attachScoreTimeline pose la courbe de score sur le document et rend sa couverture.
//
// LA COUVERTURE VOYAGE AVEC LE CALQUE, et son ABSENCE dit autre chose que des compteurs a zero :
// « l'appelant n'a rien fourni a lire » (CLI hors ligne sans faits de match) n'est pas « le film
// n'a rien livre ». C'est pourquoi elle est rendue plutot que posee ici : `buildCoverage`
// l'assemble avec les autres, et un nil reste un nil.
func attachScoreTimeline(doc *ReplayDocument, in *ScoreInput, c scoreClock, matchID string) *ScoreCoverage {
	tl, cov := buildScoreTimeline(in, c)
	doc.ScoreTimeline = tl
	logScoreCoverage(matchID, cov, tl)
	return cov
}

// logScoreCoverage journalise ce que le calque a publie — et ce qu'il n'a pas resolu.
func logScoreCoverage(matchID string, cov *ScoreCoverage, tl *ScoreTimeline) {
	if cov == nil {
		return
	}
	teams, players := 0, 0
	if tl != nil {
		teams, players = len(tl.Teams), len(tl.Players)
	}
	if cov.TeamIdentity == ScoreIdentityUnresolved {
		slog.Warn("rejeu : identite des camps NON RESOLUE — courbes de score publiees sans equipe",
			"match_id", matchID, "equipes", teams, "joueurs", players, "manches", cov.Rounds)
	}
	if cov.Truncated {
		slog.Warn("rejeu : lecture des enregistrements TRONQUEE — courbes de score incompletes",
			"match_id", matchID, "points", cov.Points)
	}
	slog.Info("rejeu : courbe de score",
		"match_id", matchID, "identiteEquipes", cov.TeamIdentity, "manches", cov.Rounds,
		"modePorte", cov.ModeSupported, "equipes", teams, "joueurs", players, "points", cov.Points)
}
