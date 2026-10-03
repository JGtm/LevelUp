package killcollector

// positions_pont_test.go — LOT J4.4 : CE QUE L ETAGE UNIQUE DU PONT CHANGE AUX LECTURES DU
// COLLECTEUR, ET RIEN D AUTRE.
//
// Le lot J4.3 a remplace la sequence recopiee du collecteur (positions, horloge, fil des morts,
// index, creations) par `decfilm.ScanPontDIdentite`, le meme etage que la cuisson. Le changement
// DECLARE est UN SEUL : le balayage des positions recoit desormais les exemptions de translocation
// (decision D2) — c est ce qui fait monter [IsolationDecoderRev]. Ce test fige le « rien d autre » :
// sur une bobine du depot SANS evenement de translocateur, les cinq lectures que le collecteur tire
// de l etage sont celles de l ANCIENNE sequence, recopiee ici a l identique (roster de la feuille,
// aucune capture de direction). Le « celui-la » (les exemptions atteignent le balayage des
// positions) est tenu au plus pres de l etage : `grammar/pont_identite_test.go`,
// `TestPontDIdentite_ExemptionsDeTranslocationAppliquees`.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

// bobineDuPont : la mini-bobine killsource du film de reference `000d5950` (Cliffhanger), prefixe
// CONTIGU de ses chunks 00 a 05 — la seule bobine du depot dont les positions bipedes se lisent
// (les huit bobines par build du rejeu n ont aucun bipede dans leurs images-cles). Sans
// evenement de translocateur. Relative a CE paquet.
const bobineDuPont = "../../games/halo_infinite/film/internal/facts/killsource/testdata/minibobine_000d5950"

// sequenceAvantJ43 rejoue les lectures du collecteur TELLES QU ELLES ETAIENT avant le lot J4.3.
func sequenceAvantJ43(t *testing.T, film *decfilm.Film, entry decfilm.MapQuantEntry, roster []uint64) (lecturesDuFilm, uint64) {
	t.Helper()
	fc := decfilm.NewFilmContextForMap(film, &entry, nil)
	positions, err := decfilm.ScanBipedPositions(fc, optionsDeBalayageDesPositions(fc, entry))
	if err != nil {
		t.Fatalf("ancienne sequence, positions : %v", err)
	}
	origin, err := decfilm.ScanClockOrigin(film)
	if err != nil {
		t.Fatalf("ancienne sequence, horloge : %v", err)
	}
	deaths, err := decfilm.ScanDeaths(film)
	if err != nil {
		t.Fatalf("ancienne sequence, fil des morts : %v", err)
	}
	idx, err := decfilm.ScanPlayerIndices(film, roster)
	if err != nil {
		t.Fatalf("ancienne sequence, index : %v", err)
	}
	creations, _, err := decfilm.ScanBipedCreations(fc)
	if err != nil {
		creations = nil
	}
	return lecturesDuFilm{positions: positions, creations: creations, deaths: deaths, idx: idx}, origin
}

func TestPontDuCollecteur_SeuleLExemptionChange(t *testing.T) {
	film, err := decfilm.LoadDir(bobineDuPont, nil)
	if err != nil {
		t.Fatalf("bobine %s : %v", bobineDuPont, err)
	}
	entry, err := catalogueDeBornesVersionne(t).Lookup("Cliffhanger")
	if err != nil {
		t.Fatalf("entree de catalogue : %v", err)
	}
	deaths, err := decfilm.ScanDeaths(film)
	if err != nil || len(deaths) == 0 {
		t.Fatalf("fil des morts de la bobine : %d morts, err %v", len(deaths), err)
	}
	ids := MatchIdentities{}
	var roster []uint64
	vus := map[uint64]bool{}
	for _, d := range deaths {
		if !vus[d.XUID] {
			vus[d.XUID] = true
			ids.XUIDs = append(ids.XUIDs, strconv.FormatUint(d.XUID, 10))
			roster = append(roster, d.XUID)
		}
	}

	apres, origineApres, err := lireLePontDuCollecteur(context.Background(), film, entry, ids, "000d5950")
	if err != nil {
		t.Fatalf("etage du pont, politiques du collecteur : %v", err)
	}
	avant, origineAvant := sequenceAvantJ43(t, film, entry, roster)
	if len(avant.positions) == 0 {
		t.Fatal("aucune position sur la bobine : la comparaison ne prouverait rien")
	}
	if tp := decfilm.ScanPontDIdentite(decfilm.NewFilmContextForMap(film, &entry, nil),
		decfilm.OptionsDuPont{Balayage: decfilm.DefaultScanFilmOptions(), Carte: &entry}).Translocations; len(tp) != 0 {
		t.Fatalf("%d teleportation(s) sur la bobine : ce temoin doit en etre SANS", len(tp))
	}
	if origineApres != origineAvant {
		t.Errorf("origine d horloge %d, attendu %d", origineApres, origineAvant)
	}
	for _, c := range []struct {
		nom          string
		apres, avant any
	}{
		{"positions", apres.positions, avant.positions},
		{"creations", apres.creations, avant.creations},
		{"fil des morts", apres.deaths, avant.deaths},
		{"table d index", apres.idx, avant.idx},
	} {
		if !reflect.DeepEqual(c.apres, c.avant) {
			t.Errorf("%s : l etage du pont rend autre chose que l ancienne sequence du collecteur "+
				"sur un film SANS teleportation — seule l exemption de translocation devait changer", c.nom)
		}
	}
}
