package replayverite

// scores.go — LES ORACLES : K/D/A du statborg, morts par les vies, scores des camps, equipes.

import (
	"strconv"

	"levelup/go-api/internal/domain"
)

// methodeCirculaire : la methode de nommage d'un slot statborg qui apparie sur le triplet K/D/A de
// la feuille — le slot est nomme PARCE QUE son K/D/A egale l'API (`objectives.SlotIdentityFrom`).
const methodeCirculaire = "triplet_feuille"

// Les methodes de rattachement des camps qui apparient sur le SCORE FINAL du registre
// (`score_team_identity.go` : `a` score final, `a0` score final d'un match a sens unique).
var identitesDeCampCirculaires = map[string]bool{"a": true, "a0": true}

// statCaptureDrapeau : l action d objectif qui marque un point en CTF.
const statCaptureDrapeau = "flag_captures"

// noterKDA : O-K1..3, les K/D/A finaux du statborg contre la feuille de match.
func noterKDA(d *Document, faits domain.MatchFacts, scores map[string]Score) {
	circ := xuidsCirculaires(d)
	pub := map[string]ScoreJoueur{}
	if d.ScoreTimeline != nil {
		for _, p := range d.ScoreTimeline.Players {
			pub[p.XUID] = p
		}
	}
	var k, m, a Score
	off := map[string]bool{}
	for _, f := range faits.Players {
		off[f.XUID] = true
		if circ[f.XUID] {
			k.Exclus, m.Exclus, a.Exclus = k.Exclus+1, m.Exclus+1, a.Exclus+1
			continue
		}
		p := pub[f.XUID] // absent = courbes vides = 0 publie
		k.noter(f.XUID, p.Kills.Finale(), f.Kills)
		m.noter(f.XUID, p.Deaths.Finale(), f.Deaths)
		a.noter(f.XUID, p.Assists.Finale(), f.Assists)
	}
	// Un xuid publie que la feuille ne connait pas : tout ce qu'il porte est en trop.
	for x, p := range pub {
		if !off[x] {
			k.noter(x, p.Kills.Finale(), 0)
			m.noter(x, p.Deaths.Finale(), 0)
			a.noter(x, p.Assists.Finale(), 0)
		}
	}
	scores[ScoreKills], scores[ScoreMorts], scores[ScoreAssists] = k, m, a
}

// xuidsCirculaires : les xuid dont TOUS les slots statborg nommes l'ont ete par le triplet.
func xuidsCirculaires(d *Document) map[string]bool {
	out := map[string]bool{}
	if d.Identity == nil {
		return out
	}
	autre := map[string]bool{}
	for _, s := range d.Identity.StatborgSlots {
		if s.XUID == "" {
			continue
		}
		if s.Link.Method == methodeCirculaire {
			out[s.XUID] = true
		} else {
			autre[s.XUID] = true
		}
	}
	for x := range autre {
		delete(out, x)
	}
	return out
}

// noterVies : O-V1 (morts deduites des vies, par joueur) et O-V2 (fins de vie, sans identite).
func noterVies(d *Document, faits domain.MatchFacts, scores map[string]Score) {
	vies := map[string]int{}
	finsDeVie := 0
	for _, t := range d.Tracks {
		if t.XUID != "" {
			vies[t.XUID]++
		}
		if t.EndFrame < d.FrameCount-1-margeFinDeFilmImages {
			finsDeVie++
		}
	}
	var v1 Score
	off := map[string]bool{}
	mortsAPI := 0
	for _, f := range faits.Players {
		off[f.XUID] = true
		mortsAPI += f.Deaths
		v1.noter(f.XUID, max(0, vies[f.XUID]-1), f.Deaths)
	}
	for x, n := range vies {
		if !off[x] {
			v1.noter(x, max(0, n-1), 0)
		}
	}
	var v2 Score
	v2.noter("match", finsDeVie, mortsAPI)
	scores[ScoreMortsVies], scores[ScoreFinsDeVie] = v1, v2
}

// noterCamps : O-S1 (score par camp), O-S2 (actions de marque par camp), O-T1 (equipes).
func noterCamps(d *Document, faits domain.MatchFacts, scores map[string]Score) {
	if c := d.Coverage.Teams; c != nil {
		scores[ScoreEquipes] = Score{VP: c.Accord, FP: c.Contradiction, FN: c.Silence}
	}
	if faits.TeamScores == nil {
		return
	}
	officiel := *faits.TeamScores
	scores[ScoreCamps] = scoreDesCamps(d, officiel)
	if s, ok := scoreDesActionsDeMarque(d, faits, officiel); ok {
		scores[ScoreMarque] = s
	}
}

// scoreDesCamps : le score final publie de chaque camp contre le registre — NON NOTE quand les
// camps ont ete rattaches par ce score meme.
func scoreDesCamps(d *Document, officiel [2]int) Score {
	var s Score
	if d.Coverage.Score != nil && identitesDeCampCirculaires[d.Coverage.Score.TeamIdentity] {
		return Score{NonNote: true, Exclus: len(officiel)}
	}
	pub := map[int]int{}
	if d.ScoreTimeline != nil {
		for _, t := range d.ScoreTimeline.Teams {
			if t.TeamID != nil {
				pub[*t.TeamID] = derniere(t.Total)
			}
		}
	}
	for camp, v := range officiel {
		s.noter("camp "+strconv.Itoa(camp), pub[camp], v)
	}
	return s
}

// scoreDesActionsDeMarque : O-S2. La famille se lit dans les CALQUES publies, jamais dans un nom de
// mode : des portages de drapeau => le score est la somme des captures ; des stats de bombe => la
// somme des detonations. Sans l'un ni l'autre, le score n'est pas applicable (faux).
func scoreDesActionsDeMarque(d *Document, faits domain.MatchFacts, officiel [2]int) (Score, bool) {
	camp := map[string]int{}
	for _, p := range faits.Players {
		camp[p.XUID] = p.TeamID
	}
	var parCamp [2]int
	switch {
	case len(d.FlagCarries) > 0:
		for _, o := range d.Objectives {
			if o.Stat == statCaptureDrapeau {
				ajouterAuCamp(&parCamp, camp, o.XUID, 1)
			}
		}
	case d.BombStats != nil:
		for _, p := range d.BombStats.Players {
			if p.Detonations != nil {
				ajouterAuCamp(&parCamp, camp, p.XUID, *p.Detonations)
			}
		}
	default:
		return Score{}, false
	}
	var s Score
	for c, v := range officiel {
		s.noter("camp "+strconv.Itoa(c), parCamp[c], v)
	}
	return s, true
}

// ajouterAuCamp compte `n` au camp officiel du joueur ; un joueur sans camp connu n'est compte nulle
// part (il ne peut pas etre un faux de camp, et n'est pas un vrai non plus).
func ajouterAuCamp(parCamp *[2]int, camp map[string]int, xuid string, n int) {
	c, ok := camp[xuid]
	if !ok || c < 0 || c >= len(parCamp) {
		return
	}
	parCamp[c] += n
}
