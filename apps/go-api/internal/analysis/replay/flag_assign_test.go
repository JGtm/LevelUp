package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_assign_test.go — A QUEL DRAPEAU UN PORTAGE APPARTIENT, sans film.
//
// Les quatre tests figent les trois regles de `flag_assign.go` ET leur ordre. La geometrie est
// celle de `bcb6d393` (CTF:Arena, 3-0), reduite a l'echelle du banc : le socle du camp qui SUBIT
// en (0,0), celui du camp qui MARQUE en (100,100), et des porteurs qui font le trajet de l'un a
// l'autre. Le calque ne connait pas les equipes (`Track.Team` n'est pas dans le film) : c'est
// l'ETIQUETTE du socle, donc `FlagCarry.Team`, qui porte le verdict.

// flagAssignSpawns rend les deux socles du banc, etiquetes comme la carte Cliffhanger : le
// drapeau PORTE est celui de l'equipe 1, et celui que ses porteurs ne doivent JAMAIS prendre est
// celui de l'equipe 0.
func flagAssignSpawns() []FlagSpawn {
	return []FlagSpawn{{Team: 1, X: 0, Y: 0}, {Team: 0, X: 100, Y: 100}}
}

// flagTestPath fabrique une piste qui BOUGE : un porteur prend le drapeau a un endroit et le
// lache a un autre — c'est exactement l'ecart que l'attribution doit suivre.
func flagTestPath(slot uint32, xuid string, from, to, bascule int, x0, y0, x1, y1 float32) Track {
	tr := Track{Slot: slot, Team: TeamNeutral, XUID: xuid, StartFrame: from, EndFrame: to}
	for t := from; t <= to; t++ {
		x, y := x0, y0
		if t >= bascule {
			x, y = x1, y1
		}
		tr.Points = append(tr.Points, Point{T: t, X: x, Y: y})
	}
	return tr
}

// porteursDe rend, dans l'ordre, les porteurs des intervalles portes d'un drapeau.
func porteursDe(f FlagCarry) []string {
	var out []string
	for _, s := range f.Spans {
		if s.State != FlagStateCarried && s.State != FlagStateCarriedOpen {
			continue
		}
		if s.XUID == nil {
			out = append(out, "")
			continue
		}
		out = append(out, *s.XUID)
	}
	return out
}

// assertPorteurs compare la suite des porteurs d'un drapeau a celle attendue.
func assertPorteurs(t *testing.T, f FlagCarry, want []string) {
	t.Helper()
	got := porteursDe(f)
	if len(got) != len(want) {
		t.Fatalf("drapeau d'equipe %d porte par %v, attendu %v", f.Team, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("drapeau d'equipe %d porte par %v, attendu %v", f.Team, got, want)
		}
	}
}

// TestFlagAssignLeSolSuitLeTempsEtNonLOrdreDesPrises — LA CAUSE PREMIERE du fait `bcb6d393`.
//
// L'etat « ou git chaque drapeau » se tenait a jour en parcourant les PRISES : la position de
// LACHER d'un portage y etait inscrite des son attribution, donc AVANT d'avoir eu lieu. Deux
// portages qui se recouvrent suffisent alors a chasser du sol une position qui y est encore.
//
// Ici le portage de « 2 » s'ouvre a 3 000 ms et ne se ferme qu'a 9 000 ms — apres la prise de
// « 3 » a 5 000 ms — et il lache loin, en (5,5). Sans l'ordre du temps, ce lacher-la efface la
// position (92,92) ou « 1 » vient de poser le drapeau, et la prise de « 3 », ramassee a 0 m de
// cette position, retombe sur le socle LE PLUS PROCHE : celui de son propre camp.
//
// LES DEUX DRAPEAUX SONT DEHORS, ET C'EST VOLONTAIRE : « 4 » tient celui de l'equipe 0 du debut
// a la fin. La regle du « seul drapeau en jeu » est donc MUETTE, et seule la position au sol,
// tenue dans l'ordre du temps, peut repondre — c'est ce que ce test isole.
func TestFlagAssignLeSolSuitLeTempsEtNonLOrdreDesPrises(t *testing.T) {
	tracks := []Track{
		flagTestPath(12, "1", 0, 48, 48, 2, 2, 92, 92),   // vole a (2,2), meurt en (92,92)
		flagTestPath(14, "2", 0, 99, 90, 30, 30, 5, 5),   // portage qui RECOUVRE, lache loin
		flagTestTrack(16, "3", 0, 99, 92, 92),            // ramasse la ou « 1 » a lache
		flagTestPath(18, "4", 0, 99, 60, 98, 98, 99, 20), // tient l'autre drapeau tout du long
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 500, Slot: 18, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 14, Stat: objectiveevents.StatFlagGrabs},
			{TimeMS: 5000, Slot: 16, Stat: objectiveevents.StatFlagGrabs},
			{TimeMS: 7000, Slot: 16, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(
			map[int]string{12: "1", 14: "2", 16: "3", 18: "4"}),
		Spawns: flagAssignSpawns(),
	}
	deaths := []Death{{XUID: 1, TimeMS: 4800}, {XUID: 2, TimeMS: 9000}}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.Carries != 4 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 4 portages attendus, invariant tenu", *cov)
	}
	assertPorteurs(t, flagOfTeam(t, got, 1), []string{"1", "2", "3"})
	// LE CONTROLE QUI FAIT TOMBER LE DEFAUT : le drapeau du camp qui marque n'est porte que par
	// « 4 », l'adversaire. La prise de « 3 » n'a rien a y faire.
	assertPorteurs(t, flagOfTeam(t, got, 0), []string{"4"})
	// Et la CAPTURE se publie sur le drapeau ADVERSE, a la frame qui suit son instant.
	if !flagHomeAt(flagOfTeam(t, got, 1), 71) {
		t.Errorf("aucun retour a la base a la frame 71 sur le drapeau adverse : la capture de "+
			"7 000 ms doit y renvoyer le drapeau, etats %v", flagStatesOf(flagOfTeam(t, got, 1)))
	}
}

// flagHomeAt dit qu'un drapeau est rentre a sa base a la frame donnee.
func flagHomeAt(f FlagCarry, frame int) bool {
	for _, s := range f.Spans {
		if s.State == FlagStateHome && s.T0 == frame {
			return true
		}
	}
	return false
}

// flagStatesOf rend la suite des etats d'un drapeau, pour les messages d'echec.
func flagStatesOf(f FlagCarry) []string {
	out := make([]string, 0, len(f.Spans))
	for _, s := range f.Spans {
		out = append(out, s.State)
	}
	return out
}

// TestFlagAssignPriseVaAuSeulDrapeauEnJeu — LA REGLE QUI MANQUAIT.
//
// Une PRISE (`flag_grabs`) ramasse un drapeau DEJA EN JEU : elle ne peut pas porter sur un
// drapeau reste a son socle. Quand le sol ne rattache rien — le drapeau est dans une main, et le
// portage qui le tient n'est pas encore ferme — le repli sur le socle le plus proche est FAUX :
// il designe justement le socle du camp qui court vers sa base. Ici la prise se fait en (88,88),
// a 17 m du socle de l'equipe 0 et a 124 m de celui de l'equipe 1.
func TestFlagAssignPriseVaAuSeulDrapeauEnJeu(t *testing.T) {
	tracks := []Track{
		flagTestTrack(12, "1", 0, 99, 2, 2), // vole et ne lache jamais : le sol reste vide
		flagTestTrack(14, "2", 0, 99, 88, 88),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 14, Stat: objectiveevents.StatFlagGrabs},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   flagAssignSpawns(),
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Carries != 2 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 2 portages attendus, invariant tenu", *cov)
	}
	if cov.AssignedByPlay != 1 {
		t.Errorf("assignedByPlay = %d, attendu 1 : la troisieme regle a tranche, elle se publie",
			cov.AssignedByPlay)
	}
	assertPorteurs(t, flagOfTeam(t, got, 1), []string{"1", "2"})
	assertPorteurs(t, flagOfTeam(t, got, 0), nil)
}

// TestFlagAssignADeuxDrapeauxEnJeuLaRegleSeTait — LA CONTRE-EPREUVE.
//
// La regle ci-dessus n'est pas un fourre-tout : elle ne repond que lorsqu'UN SEUL drapeau est en
// jeu. Les deux dehors, rien ne les departage et le socle le plus proche reprend la main —
// exactement le comportement d'avant. On retrecit le repli, on ne le supprime pas.
func TestFlagAssignADeuxDrapeauxEnJeuLaRegleSeTait(t *testing.T) {
	tracks := []Track{
		flagTestTrack(12, "1", 0, 99, 2, 2),   // vole le drapeau de l'equipe 1
		flagTestTrack(14, "2", 0, 99, 98, 98), // vole celui de l'equipe 0
		flagTestTrack(16, "3", 0, 99, 88, 88), // ramasse loin de tout drapeau au sol
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 2000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 16, Stat: objectiveevents.StatFlagGrabs},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2", 16: "3"}),
		Spawns:   flagAssignSpawns(),
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Carries != 3 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 3 portages attendus, invariant tenu", *cov)
	}
	if cov.AssignedByPlay != 0 {
		t.Errorf("assignedByPlay = %d, attendu 0 : a deux drapeaux dehors la regle se TAIT",
			cov.AssignedByPlay)
	}
	assertPorteurs(t, flagOfTeam(t, got, 1), []string{"1"})
	assertPorteurs(t, flagOfTeam(t, got, 0), []string{"2", "3"})
}

// TestFlagAssignLeVolResteAuSocle — L'ORDRE DES REGLES, fige.
//
// Un VOL se fait AU SOCLE par definition : il prend le drapeau de ce socle, meme si l'AUTRE
// drapeau git a ses pieds. Ici « 1 » lache le drapeau de l'equipe 1 en (98,98), au pied du socle
// de l'equipe 0, et « 2 » y vole aussitot : c'est le drapeau de l'equipe 0 qu'il prend.
func TestFlagAssignLeVolResteAuSocle(t *testing.T) {
	tracks := []Track{
		flagTestPath(12, "1", 0, 20, 20, 2, 2, 98, 98),
		flagTestTrack(14, "2", 0, 99, 98, 98),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   flagAssignSpawns(),
	}
	deaths := []Death{{XUID: 1, TimeMS: 2000}}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, deaths, 100))
	if cov.Carries != 2 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 2 portages attendus, invariant tenu", *cov)
	}
	assertPorteurs(t, flagOfTeam(t, got, 1), []string{"1"})
	assertPorteurs(t, flagOfTeam(t, got, 0), []string{"2"})
}
