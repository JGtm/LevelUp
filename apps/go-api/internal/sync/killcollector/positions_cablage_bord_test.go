package killcollector

// positions_cablage_bord_test.go — LE CABLAGE DU BORD DANS LA PASSE DES POSITIONS (revue adversariale
// E10.1, constat C8). Les tests de `replay` donnent le bord a la main ; aucun ne tombait si le
// collecteur cessait de le passer. Ici, le materiau d'une passe (bord partage, origine d'horloge,
// compteur de replis) traverse les DEUX constructeurs d'entree, et une victime silencieuse n'a de lieu
// que par la tenue du bord : sans le cablage, `kill_positions` perd la victime et le contexte de mort
// disparait.

import (
	"testing"

	"levelup/go-api/internal/persist"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
	"github.com/JGtm/LevelUp/libs/film/replay"
)

// materiauDuSilence : la victime 111 (slot 1) est immobile en (5, 0) jusqu'a 9,2 s puis se tait ;
// 222 (slot 2) est en (20, 0) pendant 20 s. 222 tue 111 a 10 s.
func materiauDuSilence() (materiauDIsolement, MatchIdentities, []persist.KillEventInsert) {
	var positions []decfilm.BipedPosition
	for t := int64(0); t <= 20_000; t += 100 {
		if t <= 9_200 {
			positions = append(positions, bipedAt(1, t, 5, 0, 0))
		}
		positions = append(positions, bipedAt(2, t, 20, 0, 0))
	}
	mat := materiauDIsolement{
		registre:  registreSynthetique(positions, map[uint32]uint64{1: 111, 2: 222}),
		positions: positions,
		origineUS: -80_000,
		bord:      replay.LectureDuBord{Lue: true}.Partagee(),
	}
	ids := MatchIdentities{Equipes: map[string]int{"111": 0, "222": 1}, replis: decfilm.NouveauCompteur()}
	deaths := []persist.KillEventInsert{{TimeMS: 10_000, VictimGamertag: "V", VictimXUID: "111",
		FeedKillerGamertag: "K", FeedKillerXUID: "222", FeedPresent: true}}
	return mat, ids, deaths
}

func TestEntreeDesPositions_PorteLeMateriauDeLaPasse(t *testing.T) {
	mat, ids, deaths := materiauDuSilence()
	e := entreeDesPositions(mat, deaths, ids)
	if !e.Bord.Lue || e.OrigineUS != mat.origineUS || e.Fallbacks != ids.replis {
		t.Fatalf("bord lu %v, origine %d (attendu %d), compteur partage %v : le materiau ne traverse pas",
			e.Bord.Lue, e.OrigineUS, mat.origineUS, e.Fallbacks == ids.replis)
	}
	if len(e.Kills) != 1 || len(e.Morts) != 1 || len(e.Positions) != len(mat.positions) {
		t.Fatalf("couples %d, morts %d, positions %d : le journal ou les positions manquent",
			len(e.Kills), len(e.Morts), len(e.Positions))
	}
	c := entreeContexteMorts(mat, e.Morts, equipesNumeriques(ids.Equipes), ids)
	if !c.Bord.Lue || c.OrigineDuFilmUS != mat.origineUS || c.Fallbacks != ids.replis {
		t.Fatalf("contexte : bord lu %v, origine %d, compteur partage %v", c.Bord.Lue, c.OrigineDuFilmUS, c.Fallbacks == ids.replis)
	}
}

// TestPasseDesPositions_LaVictimeSilencieuseEstTenueParLeBord : la tenue place la victime dans
// `kill_positions` ET dans le contexte de mort, au meme lieu.
func TestPasseDesPositions_LaVictimeSilencieuseEstTenueParLeBord(t *testing.T) {
	mat, ids, deaths := materiauDuSilence()
	pass := composerPassePositions(entreeDesPositions(mat, deaths, ids), "m1")
	if len(pass.rows) != 1 || pass.rows[0].VictimX == nil || pass.rows[0].KillerX == nil {
		t.Fatalf("kill_positions : %+v, attendu les deux cotes (victime tenue)", pass.rows)
	}
	if x := *pass.rows[0].VictimX; x < 5-toleranceX || x > 5+toleranceX {
		t.Errorf("victime en x=%v, attendu 5 (sa tenue)", x)
	}
	ctxs, _, ecarts := toDeathContextRows(mat, ids, deaths)
	if len(ctxs) != 1 || ecarts.sansLieu != 0 {
		t.Fatalf("contextes %+v (sans lieu %d) : la victime tenue devait avoir un contexte", ctxs, ecarts.sansLieu)
	}
}

// TestPasseDesPositions_SansBordLaVictimeSilencieuseResteSansLieu : le temoin du test precedent —
// sans la lecture du bord, ni `kill_positions` ni le contexte ne la placent.
func TestPasseDesPositions_SansBordLaVictimeSilencieuseResteSansLieu(t *testing.T) {
	mat, ids, deaths := materiauDuSilence()
	mat.bord = replay.LectureDuBord{}
	pass := composerPassePositions(entreeDesPositions(mat, deaths, ids), "m1")
	if len(pass.rows) != 1 || pass.rows[0].VictimX != nil {
		t.Fatalf("kill_positions : %+v, attendu la victime sans lieu", pass.rows)
	}
	if ctxs, _, ecarts := toDeathContextRows(mat, ids, deaths); len(ctxs) != 0 || ecarts.sansLieu != 1 {
		t.Fatalf("contextes %+v (sans lieu %d), attendu aucun", ctxs, ecarts.sansLieu)
	}
}
