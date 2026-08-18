package replaybuild

// matchfacts.go — LE DEUXIEME DECODAGE DU FILM, ET CE QU'IL ALIMENTE.
//
// # CE QUE CE FICHIER FAIT
//
// Il lit UNE fois les enregistrements d'entite du film (le « statborg ») et en tire les deux
// calques qui en dependent :
//
//	la COURBE DE SCORE     les deux camps et les compteurs vivants de chaque joueur ;
//	les ACTIONS D'OBJECTIF nommees (capture, retour, prise de zone) et attribuees a un xuid.
//
// UN SEUL DECODAGE POUR LES DEUX, et c'est la raison d'etre du fichier : les fonctions de
// facade d'`objectiveevents` (`NamedEvents`, `SlotIdentity`) re-decodent le film
// a chaque appel. Les enchainer coûterait trois balayages complets la ou un seul suffit — sur
// une machine qui paie deja le decodage des positions, ce n'est pas un detail (0,6 a 2,4 s et
// jusqu'a 21 Mo par film, mesure du corpus de 22).
//
// # POURQUOI C'EST ICI ET PAS DANS `analysis/replay`
//
// Meme frontiere que pour les morts sans revendication : `analysis/` est title-agnostic et pur,
// ce paquet est la couche d'ASSEMBLAGE. C'est lui qui sait ou vit le cache film du titre, et
// c'est lui qui recoit de l'appelant les faits de base (`port.MatchFacts`) que ni l'un ni
// l'autre ne va chercher en base.
//
// # TOUTE DEGRADATION EST JOURNALISEE, JAMAIS AVALEE
//
// Film sans manifeste, faits absents, mode sans famille d'objectif : chacun de ces cas rend un
// artefact PARFAITEMENT VALIDE, seulement plus pauvre. Le taire laisserait croire que le film
// ne portait rien.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/port"
)

// filmStats est ce que le second decodage rend au constructeur.
type filmStats struct {
	score      *replay.ScoreInput
	objectives []objectiveevents.IdentifiedEvent
	// flag porte les lectures du DRAPEAU VIVANT que seul cet etage peut faire : les
	// enregistrements d'entite (les memes que la courbe de score) et les bursts de capture. Les
	// SOCLES s'y ajoutent chez l'appelant (ils viennent du catalogue de carte, pas du film).
	flag replay.FlagInput
}

// readFilmStats decode les enregistrements d'entite et assemble les entrees des deux calques.
//
// Rend un filmStats VIDE (score nil) quand le film n'est pas lisible par cette porte : le
// document sort alors sans courbe de score ET sans couverture de score, ce qui dit « rien n'a
// ete lu » plutot que « rien n'existait ».
func readFilmStats(ctx context.Context, matchID, filmDir string, facts port.MatchFacts) filmStats {
	src, found, err := filmcache.OpenChunkDir(filmDir)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "replaybuild: manifeste de film illisible — rejeu sans courbe de score",
			"err", err, "match_id", matchID, "filmDir", filmDir)
		return filmStats{}
	case !found:
		slog.InfoContext(ctx, "replaybuild: film sans manifeste au cache — rejeu sans courbe de score",
			"match_id", matchID, "filmDir", filmDir)
		return filmStats{}
	}
	recs, truncated := objectiveevents.StatRecordsCtx(ctx, src, matchID)
	if len(recs) == 0 {
		slog.InfoContext(ctx, "replaybuild: aucun enregistrement d'entite dans le film — courbe de score vide",
			"match_id", matchID)
	}
	lines := playerLines(facts)
	if facts.Empty() {
		slog.WarnContext(ctx, "replaybuild: aucun fait de match fourni — pas de compteurs de joueur "+
			"ni d'actions d'objectif, et l'identite des camps retombe sur les frags",
			"match_id", matchID, "enregistrements", len(recs))
	}
	return filmStats{
		score: &replay.ScoreInput{
			Records:    recs,
			Lines:      lines,
			TeamByXUID: teamByXUID(facts),
			TeamScores: facts.TeamScores,
			Truncated:  truncated,
		},
		objectives: identifiedEvents(ctx, matchID, recs, lines, facts.GameVariantName),
		flag:       flagInput(recs, src),
	}
}

// flagInput assemble ce que le calque du DRAPEAU VIVANT lit dans le film.
//
// DEUX GRAMMAIRES, DEUX PARCOURS, ET LE SECOND EST INEVITABLE. Les enregistrements d'entite sont
// deja la (ils portent les evenements nommes du drapeau et les progressions du compteur de
// morts) ; les BURSTS DE CAPTURE, eux, sont des evenements de score et se lisent ailleurs dans
// le film. Sans eux le discriminant de mode ne tient pas : la table d'emplacements du drapeau,
// appliquee a un film Oddball, rend 1 470 « prises » et 994 « vols ». Le cout est un parcours de
// plus, sur une chaine qui en fait deja une dizaine pour le meme film.
//
// AUCUN FAIT DE MATCH N'ENTRE ICI, et c'est ce qui rend le calque publiable hors ligne : le
// porteur se nomme par les INSTANTS DE MORT, jamais par les lignes de match.
func flagInput(recs []objectiveevents.StatRecord, src objectiveevents.FilmSource) replay.FlagInput {
	return replay.FlagInput{
		Scanned: true,
		Records: recs,
		Bursts:  objectiveevents.CaptureBurstTimes(src),
	}
}

// identifiedEvents nomme les actions d'objectif du film et les attribue a un xuid.
//
// DEUX CONDITIONS, ET LES DEUX SONT DES REFUS EXPLICITES : sans famille d'objectif (mode sans
// table nommee, ou variante inconnue) aucun nom n'est possible ; sans lignes de match, aucun
// slot ne peut etre apparie — et poser une action sur un slot arbitraire est precisement
// l'erreur que le pont d'identite existe pour eviter.
func identifiedEvents(ctx context.Context, matchID string, recs []objectiveevents.StatRecord,
	lines []objectiveevents.PlayerLine, variant string) []objectiveevents.IdentifiedEvent {
	objType := objectiveevents.ObjectiveTypeOf(variant)
	if objType == "" || len(lines) == 0 {
		return nil
	}
	named := objectiveevents.NamedEventsFrom(recs, objType)
	identity := objectiveevents.SlotIdentityFrom(recs, lines)
	out := objectiveevents.IdentifyNamedEvents(named, identity)
	slog.InfoContext(ctx, "replaybuild: actions d'objectif identifiees",
		"match_id", matchID, "famille", objType, "nommees", len(named),
		"identifiees", len(out), "slotsApparies", len(identity))
	return out
}

// playerLines traduit les faits de match en lignes d'appariement.
func playerLines(facts port.MatchFacts) []objectiveevents.PlayerLine {
	if len(facts.Players) == 0 {
		return nil
	}
	out := make([]objectiveevents.PlayerLine, 0, len(facts.Players))
	for _, p := range facts.Players {
		out = append(out, objectiveevents.PlayerLine{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists,
		})
	}
	return out
}

// teamByXUID rend le camp de chaque joueur. Un camp inconnu (-1) n'entre PAS dans la table :
// il ferait entrer un faux camp dans la somme des frags qui identifie les slots d'equipe.
func teamByXUID(facts port.MatchFacts) map[string]int {
	out := make(map[string]int, len(facts.Players))
	for _, p := range facts.Players {
		if p.TeamID < 0 {
			continue
		}
		out[p.XUID] = p.TeamID
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
