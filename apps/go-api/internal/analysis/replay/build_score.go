package replay

import (
	"log/slog"

	"levelup/go-api/internal/analysis/objectiveevents"
)

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
// `deaths` est le fil des morts deja scanne par `BuildFromPositions` (opt.Deaths) : il sert a
// l'identite PAR MANCHE des joueurs en multi-manche (le slot d'entite est reattribue d'une manche
// a l'autre), exactement comme la couronne VIP et le drapeau le consomment. En mono-manche il
// n'est pas lu — le chemin plat par totaux est conserve a l'octet.
func attachScoreTimeline(doc *ReplayDocument, in *ScoreInput, deaths []Death, c scoreClock, matchID string) *ScoreCoverage {
	tl, cov := buildScoreTimeline(in, deaths, c)
	doc.ScoreTimeline = tl
	logScoreCoverage(matchID, cov, tl)
	logRoundWindows(matchID, in, cov)
	return cov
}

// logRoundWindows publie ce que la confrontation de la MANCHE DECLAREE AU TEMPS a ecarte (cf.
// objectiveevents/round_windows.go). Un compte NON NUL est le nominal d'un film multi-manche
// (5 a 27 enregistrements sur les temoins mesures le 2026-09-06).
//
// LE SILENCE SUR UN FILM A PLUSIEURS MANCHES EST LUI AUSSI UN SIGNAL, et c'est pour cela qu'il
// est ecrit : il dit qu'AUCUNE borne n'a pu etre posee, donc que le numero de manche du film ne
// suit pas l'horloge (trois films du parc : `fb1a1a72`, `72b0a25e`, `a4083bd2`). C'est la
// condition dans laquelle le controle de chronologie du total peut encore avoir a mordre.
// Rien n'est publie sur un film mono-manche : il n'a pas de borne de manche.
func logRoundWindows(matchID string, in *ScoreInput, cov *ScoreCoverage) {
	if in == nil || len(in.Records) == 0 || cov == nil || cov.Rounds < 2 {
		return
	}
	if n := objectiveevents.ResolveRoundWindows(in.Records).Outliers(in.Records); n > 0 {
		slog.Info("rejeu : enregistrements hors de la fenetre de leur manche declaree, ecartes",
			"match_id", matchID, "ecartes", n, "enregistrements", len(in.Records),
			"manches", cov.Rounds)
		return
	}
	slog.Warn("rejeu : AUCUNE borne de manche posee sur un film a plusieurs manches — le numero "+
		"de manche ne suit pas l'horloge, les compteurs restent ceux d'avant",
		"match_id", matchID, "manches", cov.Rounds, "enregistrements", len(in.Records))
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
