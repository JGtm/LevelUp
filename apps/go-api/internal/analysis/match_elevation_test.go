package analysis

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

const (
	elevMe    = "xuid(1)"
	elevOther = "xuid(2)"
	elevThird = "xuid(3)"
)

// rawElev fabrique une ligne mesurée : tueur, victime, dénivelé PHYSIQUE.
func rawElev(killer, victim string, dz float64, timeMS int64) domain.MatchElevationKillRaw {
	return domain.MatchElevationKillRaw{
		KillerXUID: killer, KillerGamertag: "GT-" + killer,
		VictimXUID: victim, VictimGamertag: "GT-" + victim,
		Weapon: "Fusil", WeaponEN: "Rifle",
		TimeMS: timeMS, DistanceM: 12, DeltaZ: dz,
	}
}

// LE TEST QUI COMPTE : le signe suit le JOUEUR CONSULTÉ des deux côtés. Un frag porté d'en
// haut (+3) et une mort subie face à un tueur d'en haut (dz physique +3, donc -3 pour moi)
// ne doivent PAS sortir avec le même signe — c'est toute la question posée par la carte.
func TestBuildMatchElevation_SigneCotePersonnel(t *testing.T) {
	blk := BuildMatchElevation([]domain.MatchElevationKillRaw{
		rawElev(elevMe, elevOther, 3, 1000),
		rawElev(elevOther, elevMe, 3, 2000),
	}, elevMe, 7)
	if blk == nil {
		t.Fatal("bloc nil alors que le joueur a un frag et une mort mesurés")
	}
	if len(blk.Kills) != 2 {
		t.Fatalf("Kills = %d, attendu 2", len(blk.Kills))
	}
	frag, mort := blk.Kills[0], blk.Kills[1]
	if frag.Side != domain.MatchElevationSideKill || frag.DeltaZM != 3 {
		t.Errorf("frag : side=%q dz=%v, attendu kill/+3", frag.Side, frag.DeltaZM)
	}
	if mort.Side != domain.MatchElevationSideDeath || mort.DeltaZM != -3 {
		t.Errorf("mort : side=%q dz=%v, attendu death/-3 (le tueur était AU-DESSUS, donc moi en dessous)",
			mort.Side, mort.DeltaZM)
	}
	if frag.Opponent != "GT-"+elevOther || mort.Opponent != "GT-"+elevOther {
		t.Errorf("adversaire = %q / %q, attendu le gamertag de l'autre des deux côtés",
			frag.Opponent, mort.Opponent)
	}
	if blk.MeasuredKills != 1 || blk.TotalKills != 7 {
		t.Errorf("couverture = %d/%d, attendu 1/7 (seuls les FRAGS comptent au numérateur)",
			blk.MeasuredKills, blk.TotalKills)
	}
}

// Le lobby ne porte QUE les frags des autres, côté tueur, et sa médiane est interpolée.
func TestBuildMatchElevation_LobbyEtMediane(t *testing.T) {
	blk := BuildMatchElevation([]domain.MatchElevationKillRaw{
		rawElev(elevMe, elevOther, 5, 1000),
		rawElev(elevOther, elevThird, 1, 2000),
		rawElev(elevThird, elevOther, 3, 3000),
	}, elevMe, 1)
	if len(blk.Kills) != 1 {
		t.Fatalf("Kills = %d, attendu 1 (le joueur n'a qu'un frag)", len(blk.Kills))
	}
	if len(blk.Lobby) != 2 {
		t.Fatalf("Lobby = %d, attendu 2 (les frags des autres, sans les miens)", len(blk.Lobby))
	}
	for _, k := range blk.Lobby {
		if k.Side != domain.MatchElevationSideKill {
			t.Errorf("ligne de lobby côté %q, attendu toujours le côté frag", k.Side)
		}
	}
	if math.Abs(blk.LobbyMedianDeltaZM-2) > 1e-9 {
		t.Errorf("médiane du lobby = %v, attendu 2 (interpolée entre 1 et 3)", blk.LobbyMedianDeltaZM)
	}
}

// Une victime SANS xuid (bot) n'est jamais confondue avec un joueur consulté sans xuid :
// le bloc n'existe pas sans viewer, et une ligne de bot reste au lobby.
func TestBuildMatchElevation_DegradationsPropres(t *testing.T) {
	if BuildMatchElevation(nil, elevMe, 3) != nil {
		t.Error("aucune ligne mesurée : le bloc doit être absent, pas vide")
	}
	if BuildMatchElevation([]domain.MatchElevationKillRaw{rawElev(elevOther, "", 1, 10)}, "", 0) != nil {
		t.Error("sans joueur consulté, aucun point de vue : le bloc doit être absent")
	}
	blk := BuildMatchElevation([]domain.MatchElevationKillRaw{rawElev(elevOther, "", 1, 10)}, elevMe, 0)
	if blk == nil || len(blk.Kills) != 0 || len(blk.Lobby) != 1 {
		t.Fatalf("une mort de bot doit rester au lobby, pas devenir une mort du joueur : %+v", blk)
	}
}

// L'ordre publié est celui du MATCH (l'instant), quel que soit l'ordre de lecture.
func TestBuildMatchElevation_OrdreDuMatch(t *testing.T) {
	blk := BuildMatchElevation([]domain.MatchElevationKillRaw{
		rawElev(elevMe, elevOther, 1, 9000),
		rawElev(elevMe, elevOther, 1, 1000),
	}, elevMe, 2)
	if blk.Kills[0].TimeMS != 1000 || blk.Kills[1].TimeMS != 9000 {
		t.Errorf("ordre = %d, %d — attendu l'ordre du match", blk.Kills[0].TimeMS, blk.Kills[1].TimeMS)
	}
}
