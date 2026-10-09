package squadagg

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Tests purs du builder des « Outils de destruction » (décision D8 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : chaque frag nommé, sans plafond. Le témoin de bout
// en bout de l'Escouade (chargeur, catégories, feuille de match) reste dans
// service/teammates/teammates_squad_weapon_tools_test.go.

const (
	toolJ, toolC   = "JGtm", "Chocoboflor"
	toolXJ, toolXC = "x_jgtm", "x_choco"
	keyToolFrag    = "hinf_" + "frag_grenade"
	// Clés Halo 5 nommées plutôt qu'écrites en clair dans les littéraux de ligne : le détecteur
	// de secrets prend « WeaponKey: "..." » pour une clé d'API (même parade que
	// fragdist_halo5_golden_test.go).
	keyToolH5BR       = "h5_" + "br85"
	keyToolH5Frag     = "h5_" + "frag_grenade"
	keyToolH5Splinter = "h5_" + "splinter_grenade"
)

// filmRow : une ligne telle que le lecteur adossé au film la rend (clé, nom FR d'abord,
// nom EN d'abord, classe du registre).
func filmRow(xuid, key, label, labelEN, class string, kills int) port.WeaponKillRow {
	return port.WeaponKillRow{
		XUID: xuid, WeaponKey: key, Label: label, LabelEN: labelEN, Class: class,
		Kills: kills, FromDamageSource: true,
	}
}

func toolLineByName(t *testing.T, tools *domain.SquadWeaponTools) map[string]domain.SquadWeaponToolLine {
	t.Helper()
	out := map[string]domain.SquadWeaponToolLine{}
	for _, l := range tools.Lines {
		name := l.Kind
		if l.Kind == domain.SquadToolKindWeapon {
			name = l.Label
		}
		if _, dup := out[name]; dup {
			t.Errorf("ligne %q en double", name)
		}
		out[name] = l
	}
	return out
}

func wantKills(t *testing.T, l domain.SquadWeaponToolLine, name string, j, c int) {
	t.Helper()
	if l.KillsByPlayer[toolJ] != j || l.KillsByPlayer[toolC] != c {
		t.Errorf("%s = %d / %d, want %d / %d", name, l.KillsByPlayer[toolJ], l.KillsByPlayer[toolC], j, c)
	}
	if l.TotalSquad != j+c {
		t.Errorf("%s total = %d, want %d", name, l.TotalSquad, j+c)
	}
}

// TestBuildWeaponTools_ReliquatEnDernier : le reliquat feuille − lignes forme la
// ligne « Non attribué », toujours dernière, même devant un plus gros total.
func TestBuildWeaponTools_ReliquatEnDernier(t *testing.T) {
	tools := BuildWeaponTools(WeaponToolInputs{
		Rows:           []port.WeaponKillRow{filmRow(toolXJ, "hinf_br75", "BR75", "BR75", "shoulder", 2)},
		PlayersOrdered: []string{toolJ},
		GtByXUID:       map[string]string{toolXJ: toolJ},
		Sheet:          map[string]domain.FragKillTypeCounts{toolJ: {Total: 10, Melee: 1}},
	})
	if tools == nil || len(tools.Lines) != 3 {
		t.Fatalf("lignes = %+v, want BR75, mêlée, non attribué", tools)
	}
	last := tools.Lines[2]
	if last.Kind != domain.SquadToolKindUnattributed || last.KillsByPlayer[toolJ] != 7 {
		t.Errorf("dernière ligne = %+v, want non attribué 7", last)
	}
	if last.Class != domain.FragClassUnattributed {
		t.Errorf("classe = %q, want unattributed", last.Class)
	}
}

// TestBuildWeaponTools_MecaniquesNatives : sur un titre aux mécaniques natives
// (capability), les frags de mêlée attribués à l'arme tenue quittent la ligne de l'arme ;
// assassinats et capacités spartanes ont leur ligne ; une arme sans nom rejoint le reliquat.
func TestBuildWeaponTools_MecaniquesNatives(t *testing.T) {
	tools := BuildWeaponTools(WeaponToolInputs{
		Rows: []port.WeaponKillRow{
			{XUID: toolXJ, WeaponID: 10, WeaponKey: keyToolH5BR, Label: "BR85", Class: "shoulder", Kills: 12, MechanicKills: 4},
			{XUID: toolXJ, WeaponID: 99, Kills: 3}, // arme hors registre, sans nom
		},
		PlayersOrdered: []string{toolJ},
		GtByXUID:       map[string]string{toolXJ: toolJ},
		Sheet: map[string]domain.FragKillTypeCounts{
			toolJ: {Total: 25, Melee: 2, Assassination: 3, GroundPound: 1, ShoulderBash: 1},
		},
		HasMechanics: true,
	})
	lines := map[string]domain.SquadWeaponToolLine{}
	for _, l := range tools.Lines {
		lines[l.Kind+"|"+l.Label] = l
	}
	check := func(id string, want int) {
		t.Helper()
		if got := lines[id].KillsByPlayer[toolJ]; got != want {
			t.Errorf("%s = %d, want %d", id, got, want)
		}
	}
	check("weapon|BR85", 8)
	check("melee|", 2)
	check("assassination|", 3)
	check("ground_pound|", 1)
	check("shoulder_bash|", 1)
	check("unattributed|", 25-8-2-3-1-1)
	if lines["ground_pound|"].Class != domain.FragClassSpartanAbility {
		t.Errorf("classe coup au sol = %q, want spartan_ability", lines["ground_pound|"].Class)
	}
}

// TestBuildWeaponTools_SansMecaniques : capability absente → ni assassinat ni
// capacité spartane, même si la feuille en porte (compteurs à zéro sur ces titres).
func TestBuildWeaponTools_SansMecaniques(t *testing.T) {
	tools := BuildWeaponTools(WeaponToolInputs{
		PlayersOrdered: []string{toolJ},
		GtByXUID:       map[string]string{toolXJ: toolJ},
		Sheet:          map[string]domain.FragKillTypeCounts{toolJ: {Total: 3, Melee: 3, Assassination: 2}},
	})
	if tools == nil || len(tools.Lines) != 1 || tools.Lines[0].Kind != domain.SquadToolKindMelee {
		t.Errorf("lignes = %+v, want la seule mêlée", tools)
	}
}

// TestBuildWeaponTools_Vide : aucun joueur, ou aucun frag → nil.
func TestBuildWeaponTools_Vide(t *testing.T) {
	if got := BuildWeaponTools(WeaponToolInputs{}); got != nil {
		t.Errorf("sans joueur : %+v, want nil", got)
	}
	got := BuildWeaponTools(WeaponToolInputs{
		PlayersOrdered: []string{toolJ},
		Sheet:          map[string]domain.FragKillTypeCounts{toolJ: {}},
	})
	if got != nil {
		t.Errorf("sans frag : %+v, want nil", got)
	}
}

// TestPlayersAboveSheet : l'écart film > feuille est compté pour la trace de l'appelant.
func TestPlayersAboveSheet(t *testing.T) {
	tools := &domain.SquadWeaponTools{
		Players: []string{toolJ, toolC},
		Lines: []domain.SquadWeaponToolLine{
			{Kind: domain.SquadToolKindWeapon, KillsByPlayer: map[string]int{toolJ: 5, toolC: 4}},
		},
	}
	sheet := map[string]domain.FragKillTypeCounts{toolJ: {Total: 5}, toolC: {Total: 3}}
	if got := PlayersAboveSheet(tools, sheet); got != 1 {
		t.Errorf("PlayersAboveSheet = %d, want 1", got)
	}
}

// TestBuildWeaponTools_Halo5GrenadesSansFilm : un titre sans film (Halo 5) rend des
// lignes de grenade TYPÉES depuis sa table native — elles ne donnent pas le détail (seul le
// film le donne) ; les grenades du joueur forment UNE ligne « Grenade » au total de la
// feuille, et rien ne tombe dans « Non attribué ».
func TestBuildWeaponTools_Halo5GrenadesSansFilm(t *testing.T) {
	tools := BuildWeaponTools(WeaponToolInputs{
		Rows: []port.WeaponKillRow{
			{XUID: toolXJ, WeaponID: 1001, WeaponKey: keyToolH5BR, Label: "BR85", Class: "shoulder", Kills: 40},
			{XUID: toolXJ, WeaponID: 1007, WeaponKey: keyToolH5Frag, Label: "Grenade frag", Class: "grenade", Kills: 8},
			{XUID: toolXJ, WeaponID: 1008, WeaponKey: keyToolH5Splinter, Label: "Grenade à fragments", Class: "grenade", Kills: 4},
			{XUID: toolXJ, WeaponID: 0, Kills: 14, IsGrenadeMelee: true},
		},
		PlayersOrdered: []string{toolJ},
		GtByXUID:       map[string]string{toolXJ: toolJ},
		Sheet:          map[string]domain.FragKillTypeCounts{toolJ: {Total: 54, Grenade: 14}},
	})
	if tools == nil {
		t.Fatal("outils absents")
	}
	var grenade *domain.SquadWeaponToolLine
	for i, l := range tools.Lines {
		if l.Kind == domain.SquadToolKindGrenade {
			grenade = &tools.Lines[i]
		}
		if l.Class == domain.FragClassGrenade && l.Kind == domain.SquadToolKindWeapon {
			t.Errorf("ligne de grenade typée sans film : %+v", l)
		}
		if l.Kind == domain.SquadToolKindUnattributed {
			t.Errorf("« Non attribué » inattendu : %+v", l)
		}
	}
	if grenade == nil || grenade.KillsByPlayer[toolJ] != 14 || grenade.Class != domain.FragClassGrenade {
		t.Errorf("ligne Grenade = %+v, want 14 frags de classe grenade", grenade)
	}
}

// TestBuildWeaponTools_GrenadesParJoueur : avec film, le détail typé remplace la ligne
// « Grenade » POUR LE JOUEUR qui l'a ; un joueur sans grenade typée au film garde la ligne
// de la feuille.
func TestBuildWeaponTools_GrenadesParJoueur(t *testing.T) {
	tools := BuildWeaponTools(WeaponToolInputs{
		Rows: []port.WeaponKillRow{
			filmRow(toolXJ, keyToolFrag, "Grenade frag", "Frag Grenade", "grenade", 2),
		},
		PlayersOrdered: []string{toolJ, toolC},
		GtByXUID:       map[string]string{toolXJ: toolJ, toolXC: toolC},
		Sheet: map[string]domain.FragKillTypeCounts{
			toolJ: {Total: 2, Grenade: 2},
			toolC: {Total: 1, Grenade: 1},
		},
	})
	lines := toolLineByName(t, tools)
	wantKills(t, lines["Grenade frag"], "grenade frag", 2, 0)
	wantKills(t, lines[domain.SquadToolKindGrenade], "grenade (feuille)", 0, 1)
}
