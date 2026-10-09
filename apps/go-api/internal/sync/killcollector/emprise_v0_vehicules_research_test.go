//go:build research

package killcollector

// emprise_v0_vehicules_research_test.go — LOT V0.1 DU PLAN `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md` :
// LE JOUEUR EN VEHICULE N'EST PAS SITUE.
//
// LA QUESTION. La cause `unplaced` (« en vehicule ou position non lue ») ne couvre le vehicule que
// si, pendant un episode d'occupation, le collecteur ne SITUE pas l'occupant. Le seuil du plan,
// ecrit avant la mesure : sur les episodes LUS (`src = film`) du calque vehicules, au moins 95 %
// du temps a bord sans position de moins d'une seconde dans `ScanBipedPositions`.
//
// LES DEUX HORLOGES, ET LA CONVERSION EMPLOYEE (ecrite ici parce que c'est le piege du lot) :
//
//	episode du rejeu   frames du document : film µs = origine + f × pas, ou `origine` est le
//	                   PREMIER paquet de position de la cuisson (`assemblage.ouvrir`) et `pas` =
//	                   `frameIntervalMs` × 1000. L'instrument de cuisson pose `origineUs` EXACT
//	                   (minimum des positions balayees par la cuisson) ; a defaut, `originMs` +
//	                   horloge du film (`ScanClockOrigin`), a la milliseconde.
//	collecteur         horloge du MATCH : film µs / 1000 − `DeathOffsetMS()` du registre de la
//	                   passe (`death_context.go`, `indexerParXUID`) — celle de `visibleA`.
//
// Chaque frame [T0, T1] d'un episode lu (bornes incluses, pas de 100 ms) est une SONDE, posee a
// l'instant du MATCH ci-dessus. Un occupant est « situe » a une sonde si `visibleA` rend une
// position de moins d'une seconde — la fonction de production, atteinte par
// `replay.ContextesDesMorts` (cf. l'en-tete de emprise_v0_passe_research_test.go).

import (
	"os"
	"strconv"
	"testing"

	"levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// v0Horloge : ce que l'instrument de cuisson pose du document de rejeu (sous-ensemble lu ici).
type v0Horloge struct {
	MatchID         string   `json:"matchId"`
	FrameIntervalMS int      `json:"frameIntervalMs"`
	OriginMs        *int64   `json:"originMs"`
	FilmClockUS     uint64   `json:"filmClockUs"`
	OrigineUS       uint64   `json:"origineUs"`
	Rides           []v0Ride `json:"rides"`
}

// v0Ride : un episode d'occupation du calque vehicules.
type v0Ride struct {
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
	Slot uint32 `json:"slot"`
	XUID string `json:"xuid"`
	Src  string `json:"src"`
}

// v0SrcLue : la source d'un episode LU dans le film (la constante `VehicleRideSrcFilm` de
// `film/replay`, recopiee en litteral pour ne pas elargir la surface compagnon du paquet hors de
// `film/`).
const v0SrcLue = "film"

// v0Vehicules : le resultat V0.1 d'un film.
type v0Vehicules struct {
	Episodes          int    `json:"episodes"`
	EpisodesProximite int    `json:"episodesProximite"`
	EpisodesSansXUID  int    `json:"episodesSansXuid"`
	OccupantsSansVie  int    `json:"episodesOccupantSansVieCollecteur"`
	Sondes            int    `json:"sondes"`
	NonSitues         int    `json:"nonSitues"`
	Sondes1s          int    `json:"sondesPremiereSeconde"`
	NonSitues1s       int    `json:"nonSituesPremiereSeconde"`
	EcartOrigineUS    int64  `json:"ecartOrigineUs"`
	OrigineRetenueUS  uint64 `json:"origineRetenueUs"`
}

// v0PremierePosition rend le premier paquet de position que le COLLECTEUR a balaye (controle de
// l'origine du document).
func v0PremierePosition(mat materiauDIsolement) uint64 {
	var min uint64
	for i := range mat.positions {
		ts := mat.positions[i].TimestampUS
		if min == 0 || ts < min {
			min = ts
		}
	}
	return min
}

// v0Sonde : un instant a bord et l'episode auquel il appartient.
type v0Sonde struct {
	premiereSeconde bool
}

// v0MesurerV01 mesure la part du temps a bord sans position de moins d'une seconde.
func v0MesurerV01(t *testing.T, p *v0Passe, mat materiauDIsolement) {
	t.Helper()
	var h v0Horloge
	court := title.FilmShortMatchID(p.MatchID)
	if !v0Lire(t, v0DirDuLot(), "ref", court, &h) {
		p.V01Refus = "document de rejeu absent (instrument de cuisson non joue)"
		return
	}
	if !mat.registre.PontPubliable() {
		p.V01Refus = "pont du collecteur non publiable : aucun fait d'isolement ecrit pour ce match"
		return
	}
	res := &v0Vehicules{}
	origine, step := h.OrigineUS, uint64(h.FrameIntervalMS)*1000
	if origine == 0 && h.OriginMs != nil {
		origine = h.FilmClockUS + uint64(*h.OriginMs)*1000
	}
	res.OrigineRetenueUS = origine
	res.EcartOrigineUS = int64(p.OrigineUS) - int64(origine)
	vivants := map[uint64]bool{}
	for _, v := range mat.registre.ViesNommees() {
		vivants[v.XUID] = true
	}
	calage := mat.registre.DeathOffsetMS()
	var journal []replay.MortDuJournal
	var sondes []v0Sonde
	for _, r := range h.Rides {
		if r.Src != v0SrcLue {
			res.EpisodesProximite++
			continue
		}
		res.Episodes++
		x, err := strconv.ParseUint(r.XUID, 10, 64)
		if err != nil || x == 0 {
			res.EpisodesSansXUID++
			continue
		}
		if !vivants[x] {
			res.OccupantsSansVie++
		}
		debut := int64((origine+uint64(r.T0)*step)/1000) - calage
		for f := r.T0; f <= r.T1; f++ {
			tMS := int64((origine+uint64(f)*step)/1000) - calage
			journal = append(journal, replay.MortDuJournal{VictimeXUID: x, TempsMS: tMS})
			sondes = append(sondes, v0Sonde{premiereSeconde: tMS-debut < 1000})
		}
	}
	v0Sonder(res, journal, sondes, mat)
	p.V01 = res
}

// v0Sonder passe TOUTES les sondes du film par `replay.ContextesDesMorts` en un seul appel : un
// contexte sort pour chaque sonde dont l'occupant est visible (`visibleA`), dans l'ordre du
// journal — d'ou l'appariement a deux curseurs.
func v0Sonder(res *v0Vehicules, journal []replay.MortDuJournal, sondes []v0Sonde,
	mat materiauDIsolement) {
	// UN CAMP PAR OCCUPANT : chaque sonde est seule de son camp, donc aucun coequipier n est
	// examine — le contexte ne depend que de la visibilite de la victime.
	equipes := map[uint64]int{}
	for _, m := range journal {
		if _, ok := equipes[m.VictimeXUID]; !ok {
			equipes[m.VictimeXUID] = len(equipes)
		}
	}
	ctxs := replay.ContextesDesMorts(replay.EntreeContexteMorts{
		Positions: mat.positions, Registre: mat.registre, Journal: journal, Equipes: equipes,
	})
	j := 0
	for i, m := range journal {
		vu := j < len(ctxs) && ctxs[j].VictimeXUID == m.VictimeXUID && ctxs[j].TempsMS == m.TempsMS
		if vu {
			j++
		}
		res.Sondes++
		if sondes[i].premiereSeconde {
			res.Sondes1s++
		}
		if !vu {
			res.NonSitues++
			if sondes[i].premiereSeconde {
				res.NonSitues1s++
			}
		}
	}
}

// v0DirDuLot rend le repertoire du lot (variable deja exigee par la garde).
func v0DirDuLot() string { return os.Getenv("EMPRISE_V0_DIR") }

// v0JournalV01 ecrit la ligne V0.1 d'un film.
func v0JournalV01(t *testing.T, court string, p v0Passe) {
	t.Helper()
	if p.V01 == nil {
		t.Logf("%s  V0.1 non mesure : %s", court, p.V01Refus)
		return
	}
	r := p.V01
	part := 0.0
	if r.Sondes > 0 {
		part = 100 * float64(r.NonSitues) / float64(r.Sondes)
	}
	t.Logf("%s  V0.1 episodes lus=%d (proximite ecartes=%d, sans xuid=%d, occupant sans vie=%d) "+
		"sondes=%d non situes=%d (%.2f %%) ; 1re seconde %d/%d ; ecart d'origine %d µs",
		court, r.Episodes, r.EpisodesProximite, r.EpisodesSansXUID, r.OccupantsSansVie,
		r.Sondes, r.NonSitues, part, r.NonSitues1s, r.Sondes1s, r.EcartOrigineUS)
}
