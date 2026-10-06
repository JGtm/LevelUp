package squademprise

import (
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

// entreeEquipement — le témoin de la maquette (illustration, MESURES §9) posé sur un seul match lu :
// mur moi 52 · 0 · 32 (23 pris), reste du camp 146 · 7 · 151 ; capteur moi 6 · 1 · 54 (12 pris),
// reste 16 · 4 · 183 ; grappin 84 et propulseur 65 lâchés par moi. Plus de quoi tester les
// exclusions : un adversaire, un match au camp inconnu, des poses de capteur (qui ne sont pas un
// usage) et un répulseur lâché (aucune ligne).
func entreeEquipement() Input {
	in := entreeSolo()
	in.Current = []Match{{MatchID: "m1", StartTime: t0}, {MatchID: "ffa", StartTime: t0}}
	in.Film.Films["m1"] = sessionusage.FilmRow{MatchID: "m1", DurationMS: 600_000}
	in.Film.Films["ffa"] = sessionusage.FilmRow{MatchID: "ffa", DurationMS: 600_000}
	in.Film.Participants = participants("m1") // « ffa » n'a aucun participant : camp inconnu
	in.Film.Players = []sessionusage.PlayerRow{
		{MatchID: "m1", XUID: "P",
			TakenByFamily:    map[string]int{"wall": 23, "sensor": 12},
			DeployedByFamily: map[string]int{"wall": 52, "sensor": 99},
			SpentByFamily:    map[string]int{"sensor": 6},
			KeptByFamily:     map[string]int{"sensor": 1},
			DroppedByFamily:  map[string]int{"wall": 32, "sensor": 54, "grapple": 84, "thruster": 65, "repulsor": 9}},
		{MatchID: "m1", XUID: "R",
			DeployedByFamily: map[string]int{"wall": 146},
			KeptByFamily:     map[string]int{"wall": 7, "sensor": 4},
			DroppedByFamily:  map[string]int{"wall": 151, "sensor": 183, "grapple": 40},
			SpentByFamily:    map[string]int{"sensor": 16}},
		{MatchID: "m1", XUID: "E1", DeployedByFamily: map[string]int{"wall": 500}, DroppedByFamily: map[string]int{"sensor": 500}},
		{MatchID: "ffa", XUID: "P", DeployedByFamily: map[string]int{"wall": 1000}, DroppedByFamily: map[string]int{"grapple": 1000}},
	}
	return in
}

func familleDe(e *domain.EmpriseEquipment, f string) *domain.EmpriseEquipmentFamily {
	for i := range e.Families {
		if e.Families[i].Family == f {
			return &e.Families[i]
		}
	}
	return nil
}

func TestBuildEquipment_TemoinDeLaMaquette(t *testing.T) {
	e := BuildEquipment(entreeEquipement())
	if e == nil || e.MatchesMeasured != 1 {
		t.Fatalf("équipement = %+v, attendu 1 match mesuré (le match au camp inconnu sort)", e)
	}
	mur := familleDe(e, "wall")
	if mur == nil || !mur.Measured || *mur.Me != (domain.EmpriseEquipmentOutcomes{Taken: 23, Used: 52, Kept: 0, Dropped: 32}) ||
		*mur.Rest != (domain.EmpriseEquipmentOutcomes{Used: 146, Kept: 7, Dropped: 151}) {
		t.Errorf("mur = %+v / %+v, attendu moi 23 pris, 52 · 0 · 32 ; reste 146 · 7 · 151", mur.Me, mur.Rest)
	}
	capteur := familleDe(e, "sensor")
	if capteur == nil || *capteur.Me != (domain.EmpriseEquipmentOutcomes{Taken: 12, Used: 6, Kept: 1, Dropped: 54}) ||
		*capteur.Rest != (domain.EmpriseEquipmentOutcomes{Used: 16, Kept: 4, Dropped: 183}) {
		t.Errorf("capteur = %+v / %+v, attendu moi 12 pris, 6 · 1 · 54 (servi = charge consommée, pas les poses) ; reste 16 · 4 · 183",
			capteur.Me, capteur.Rest)
	}
	grappin, propulseur := familleDe(e, "grapple"), familleDe(e, "thruster")
	if grappin == nil || grappin.Measured || grappin.DroppedMe != 84 || grappin.Me != nil {
		t.Errorf("grappin = %+v, attendu non mesuré, 84 lâchés par moi (le reste du camp n'y compte pas)", grappin)
	}
	if propulseur == nil || propulseur.Measured || propulseur.DroppedMe != 65 {
		t.Errorf("propulseur = %+v, attendu non mesuré, 65 lâchés", propulseur)
	}
}

func TestBuildEquipment_FamillesEtOrdre(t *testing.T) {
	e := BuildEquipment(entreeEquipement())
	var ordre []string
	for _, f := range e.Families {
		ordre = append(ordre, f.Family)
	}
	attendu := []string{"grapple", "wall", "sensor", "translocator_beacon", "shroud_screen", "threat_seeker", "repair_field", "thruster"}
	if len(ordre) != len(attendu) {
		t.Fatalf("familles = %v, attendu %v (ni bonus ni répulseur)", ordre, attendu)
	}
	for i := range attendu {
		if ordre[i] != attendu[i] {
			t.Fatalf("familles = %v, attendu %v", ordre, attendu)
		}
	}
	if champ := familleDe(e, "repair_field"); champ == nil || *champ.Me != (domain.EmpriseEquipmentOutcomes{}) {
		t.Errorf("champ de réparation = %+v, attendu une ligne à zéro (famille du bilan jamais touchée)", champ)
	}
}

// Le LOBBY : tous les joueurs des matchs mesurés — mon camp, l'adversaire et les joueurs sans camp
// connu. Une famille tenue par le seul adversaire (lanceur de traque, par E1) ou par un joueur sans
// camp (X) est dans le lobby, à zéro pour moi et le reste de mon camp ; une famille que personne
// n'a tenue (champ de réparation) a un lobby à zéro. Les matchs non mesurés (« ffa ») n'y comptent
// pas, comme pour moi.
func TestBuildEquipment_Lobby(t *testing.T) {
	in := entreeEquipement()
	in.Film.Players = append(in.Film.Players,
		sessionusage.PlayerRow{MatchID: "m1", XUID: "E1", SpentByFamily: map[string]int{"threat_seeker": 3},
			DroppedByFamily: map[string]int{"threat_seeker": 2, "thruster": 5}},
		sessionusage.PlayerRow{MatchID: "m1", XUID: "X", KeptByFamily: map[string]int{"shroud_screen": 1},
			DroppedByFamily: map[string]int{"grapple": 7}},
		sessionusage.PlayerRow{MatchID: "ffa", XUID: "E9", SpentByFamily: map[string]int{"repair_field": 50}},
	)
	e := BuildEquipment(in)
	mur := familleDe(e, "wall")
	if mur.Lobby == nil || *mur.Lobby != (domain.EmpriseEquipmentOutcomes{Taken: 23, Used: 52 + 146 + 500, Kept: 7, Dropped: 32 + 151}) {
		t.Errorf("lobby du mur = %+v, attendu moi + reste du camp + adversaire (23 pris, 698 · 7 · 183)", mur.Lobby)
	}
	traque := familleDe(e, "threat_seeker")
	if traque.Lobby == nil || *traque.Lobby != (domain.EmpriseEquipmentOutcomes{Used: 3, Dropped: 2}) ||
		*traque.Me != (domain.EmpriseEquipmentOutcomes{}) || *traque.Rest != (domain.EmpriseEquipmentOutcomes{}) {
		t.Errorf("lanceur de traque = moi %+v, reste %+v, lobby %+v ; attendu lobby 3 · 0 · 2 (l'adversaire seul), moi et reste à zéro",
			traque.Me, traque.Rest, traque.Lobby)
	}
	if ecran := familleDe(e, "shroud_screen"); ecran.Lobby == nil || ecran.Lobby.Kept != 1 || *ecran.Rest != (domain.EmpriseEquipmentOutcomes{}) {
		t.Errorf("écran occultant = reste %+v, lobby %+v ; attendu le gardé du joueur sans camp dans le lobby seul", ecran.Rest, ecran.Lobby)
	}
	if champ := familleDe(e, "repair_field"); champ.Lobby == nil || *champ.Lobby != (domain.EmpriseEquipmentOutcomes{}) {
		t.Errorf("champ de réparation = lobby %+v, attendu zéro (personne ne l'a tenu sur un match mesuré)", champ.Lobby)
	}
	if g := familleDe(e, "grapple"); g.DroppedLobby != 84+40+7 || g.DroppedMe != 84 {
		t.Errorf("grappin = %d lâchés au lobby, %d par moi ; attendu 131 (moi, reste du camp, joueur sans camp) et 84", g.DroppedLobby, g.DroppedMe)
	}
	if p := familleDe(e, "thruster"); p.DroppedLobby != 65+5 {
		t.Errorf("propulseur = %d lâchés au lobby, attendu 70 (moi et l'adversaire)", p.DroppedLobby)
	}
}

func TestBuildEquipment_SansFilm(t *testing.T) {
	in := entreeEquipement()
	in.Film = nil
	if e := BuildEquipment(in); e != nil {
		t.Errorf("équipement sans film = %+v, attendu nil", e)
	}
}
