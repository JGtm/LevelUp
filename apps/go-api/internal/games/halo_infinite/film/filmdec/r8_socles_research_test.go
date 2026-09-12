package filmdec

// r8_socles_research_test.go — MESURE 1 du lot R8 : les poses `deployed` de familles
// `repulsor` / `thruster` sont-elles des GESTES, ou des REAPPARITIONS SUR SOCLE ?
//
// L'HYPOTHESE A REFUTER EN PREMIER, et elle est nee d'une lecture directe de `4f77afc1` :
// trois des six poses `repulsor/deployed` de ce film tombent au meme point a 0,2 m pres,
// a 40 s d'intervalle. Un objet qui reapparait sur son socle avec un joueur vivant a
// moins de 3 m se voit attribuer ce joueur (`equipmentOwner`) et classer `deployed`
// (`equipmentOrigin`) : le classement est correct au sens de sa definition — l'objet n'est
// PAS ne de la mort du joueur — mais il ne decrit alors aucun geste.
//
// CRITERE ECRIT AVANT LA MESURE. Une pose est « au socle » s'il existe AU MOINS DEUX
// AUTRES poses du MEME identifiant de tag, dans le MEME film, a moins de
// `r8SocleRadiusM` metres en horizontal, dont l'instant differe du sien d'au moins
// `r8SocleGapFrames` frames. Deux autres et pas une : deux poses voisines peuvent etre un
// lacher et un ramassage rate ; trois poses au metre pres etalees sur la partie sont un
// point fixe de la carte. L'ecart temporel elimine la rafale (un objet qui rebondit).
//
// TEMOINS OBLIGATOIRES, mesures dans la meme passe :
//   - les poses `dropped` des memes familles (les lachers a la mort) : elles ne doivent
//     PAS se regrouper autant, sinon le critere ne mesure que la popularite d'un endroit ;
//   - les poses `deployed` des familles reellement deployables (`wall`, `sensor`, ...) :
//     un mur pose au fil de la partie ne se repose pas trois fois au metre pres.

import (
	"fmt"
	"sort"
	"testing"
)

const (
	// r8SocleRadiusM : rayon horizontal d'un socle. 1,0 m — un socle de reapparition rend
	// la MEME position a la precision de quantification pres (0,2 m observe sur 4f77afc1),
	// et 1,0 m laisse la marge sans absorber deux poses distinctes d'un couloir.
	r8SocleRadiusM = 1.0
	// r8SocleGapFrames : ecart temporel minimal, en frames du document (100 ms), entre deux
	// poses pour qu'elles comptent comme deux passages distincts. 100 frames = 10 s, en
	// dessous du cycle de reapparition mesure des socles d'arme (15,4 s median).
	r8SocleGapFrames = 100
)

// r8TargetFamilies : les deux familles du lot.
var r8TargetFamilies = map[string]bool{"repulsor": true, "thruster": true}

// r8DeployableFamilies : les familles dont un joueur DEPOSE reellement une piece sur la
// carte. Temoin negatif du critere de socle.
var r8DeployableFamilies = map[string]bool{
	"wall": true, "sensor": true, "threat_seeker": true,
	"repair_field": true, "shroud_screen": true,
}

// r8AtSocle applique le critere ecrit ci-dessus a la pose d'index `i` du film.
func r8AtSocle(pl []r8Placement, i int) bool {
	p := pl[i]
	n := 0
	for j := range pl {
		if j == i || pl[j].ID != p.ID {
			continue
		}
		if r8Dist2(p.X, p.Y, pl[j].X, pl[j].Y) > r8SocleRadiusM {
			continue
		}
		d := pl[j].T0 - p.T0
		if d < 0 {
			d = -d
		}
		if d >= r8SocleGapFrames {
			n++
		}
	}
	return n >= 2
}

// r8SocleCell est une case du tableau croise famille x origine.
type r8SocleCell struct {
	total, socle int
}

func TestR8PosesSocles(t *testing.T) {
	corpus := r8LoadCorpus(t)
	cells := map[string]*r8SocleCell{}
	perFilm := map[string]*r8SocleCell{}
	for _, a := range corpus {
		for i := range a.Placements {
			p := a.Placements[i]
			if !r8TargetFamilies[p.Family] && !r8DeployableFamilies[p.Family] {
				continue
			}
			key := p.Family + "/" + r8OriginOrUnknown(p.Origin)
			c := cells[key]
			if c == nil {
				c = &r8SocleCell{}
				cells[key] = c
			}
			c.total++
			at := r8AtSocle(a.Placements, i)
			if at {
				c.socle++
			}
			if r8TargetFamilies[p.Family] && p.Origin == "deployed" {
				f := perFilm[a.ID]
				if f == nil {
					f = &r8SocleCell{}
					perFilm[a.ID] = f
				}
				f.total++
				if at {
					f.socle++
				}
			}
		}
	}
	r8LogSocleTable(t, cells)
	r8LogSocleFilms(t, perFilm)
}

// r8OriginOrUnknown : un artefact anterieur au schema 10 ne porte pas d'origine — la
// lecture est `unknown`, JAMAIS `deployed` (cf. EquipmentPlacement.Origin).
func r8OriginOrUnknown(o string) string {
	if o == "" {
		return "unknown"
	}
	return o
}

func r8LogSocleTable(t *testing.T, cells map[string]*r8SocleCell) {
	t.Helper()
	keys := make([]string, 0, len(cells))
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("critere socle : >= 2 autres poses du meme tag a <= %.1f m et >= %d frames",
		r8SocleRadiusM, r8SocleGapFrames)
	t.Logf("%-28s %8s %8s %8s", "famille/origine", "poses", "socle", "part")
	for _, k := range keys {
		c := cells[k]
		t.Logf("%-28s %8d %8d %7.1f%%", k, c.total, c.socle,
			100*float64(c.socle)/float64(c.total))
	}
}

func r8LogSocleFilms(t *testing.T, perFilm map[string]*r8SocleCell) {
	t.Helper()
	ids := make([]string, 0, len(perFilm))
	for k := range perFilm {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	var tot, soc int
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		c := perFilm[id]
		tot += c.total
		soc += c.socle
		lines = append(lines, fmt.Sprintf("%s %d/%d", id, c.socle, c.total))
	}
	t.Logf("cibles deployed par film (socle/total) : %v", lines)
	t.Logf("CIBLES deployed, corpus : %d poses, %d au socle (%.1f%%), %d ISOLEES",
		tot, soc, 100*float64(soc)/float64(tot), tot-soc)
}
