package teammates

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Tests des « Outils de destruction » de l'Escouade (décision D8 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : chaque frag nommé, sans plafond.

const (
	toolJ, toolC, toolM    = "JGtm", "Chocoboflor", "Madina97294"
	toolXJ, toolXC, toolXM = "x_jgtm", "x_choco", "x_madina"
	keyToolCoilPlasma      = "hinf_" + "coil_plasma"
	keyToolCoilKinetic     = "hinf_" + "coil_kinetic"
	keyToolEnvironment     = "hinf_" + "environment"
	keyToolFrag            = "hinf_" + "frag_grenade"
)

// toolsLoader : doublure du chargeur, avec l'extension OPTIONNELLE des catégories.
type toolsLoader struct {
	*fakeSquadLoader
	rows       []port.WeaponKillRow
	categories []port.KillSourceCategoryRow
}

func (l *toolsLoader) LoadWeaponKills(
	_ context.Context, _ string, _ port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	return l.rows, nil
}

func (l *toolsLoader) LoadKillSourceCategories(
	_ context.Context, _ string, _ port.WeaponKillFilters,
) ([]port.KillSourceCategoryRow, error) {
	return l.categories, nil
}

// filmRow : une ligne telle que le lecteur adossé au film la rend (clé, nom FR d'abord,
// nom EN d'abord, classe du registre).
func filmRow(xuid, key, label, labelEN, class string, kills int) port.WeaponKillRow {
	return port.WeaponKillRow{
		XUID: xuid, WeaponKey: key, Label: label, LabelEN: labelEN, Class: class,
		Kills: kills, FromDamageSource: true,
	}
}

// soiree2209Rows : la soirée du 22/09 (maquette C3EW, « Proposition »), telle que le lecteur
// du film la rend — une ligne par (joueur, clé), les bobines et la chute comprises, puis
// les deux sentinelles de la feuille (grenade, mêlée), que le builder ignore.
func soiree2209Rows() []port.WeaponKillRow {
	type w struct {
		key, fr, en, class string
		j, c, m            int
	}
	weapons := []w{
		{"hinf_br75", "BR75", "BR75", "shoulder", 22, 22, 35},
		{"hinf_sidekick", "MK50 Sidekick", "Mk50 Sidekick", "sidearm", 13, 17, 34},
		{"hinf_m41_spnkr", "M41 SPNKr", "M41 SPNKr", "heavy", 11, 4, 5},
		{"hinf_s7_sniper", "S7 Sniper", "S7 Sniper", "heavy", 0, 0, 11},
		{"hinf_ma40_ar", "MA40 AR", "MA40 AR", "shoulder", 1, 2, 6},
		{keyToolFrag, "Grenade frag", "Frag Grenade", "grenade", 2, 1, 4},
		{"hinf_bandit", "Bandit EVO", "M392 Bandit", "shoulder", 2, 0, 4},
		{"hinf_mangler", "Déchiqueteur", "Mangler", "sidearm", 0, 2, 3},
		{"hinf_plasma_grenade", "Grenade plasma", "Plasma Grenade", "grenade", 2, 1, 0},
		{"hinf_dynamo_grenade", "Grenade dynamo", "Dynamo Grenade", "grenade", 1, 0, 0},
		{"hinf_vestige_carbine", "Carabine Vestige", "Vestige Carbine", "shoulder", 1, 0, 0},
		{"hinf_mutilator", "Mutilateur", "Mutilator", "shoulder", 1, 0, 0},
		{"hinf_vk78_commando", "VK78 Commando", "VK78 Commando", "shoulder", 1, 0, 0},
		// Bobines (une par type porte une clé) et chute : aussi rendues comme lignes par
		// arme, de classe environnement — les catégories les reprennent.
		{keyToolCoilPlasma, "Bobine à plasma", "Plasma Coil", "environmental", 1, 0, 0},
		{keyToolCoilKinetic, "Bobine à fusion UNSC", "UNSC Fusion Coil", "environmental", 0, 0, 1},
		{keyToolEnvironment, "Chute et environnement", "Environment", "environmental", 0, 1, 1},
	}
	var out []port.WeaponKillRow
	for _, x := range weapons {
		for _, pk := range []struct {
			xuid  string
			kills int
		}{{toolXJ, x.j}, {toolXC, x.c}, {toolXM, x.m}} {
			if pk.kills > 0 {
				out = append(out, filmRow(pk.xuid, x.key, x.fr, x.en, x.class, pk.kills))
			}
		}
	}
	// Sentinelles de la feuille de match : ignorées (la feuille arrive par la série).
	out = append(out,
		port.WeaponKillRow{XUID: toolXM, WeaponID: 0, Kills: 3, IsGrenadeMelee: true},
		port.WeaponKillRow{XUID: toolXM, WeaponID: 1, Kills: 16, IsGrenadeMelee: true},
	)
	return out
}

// soiree2209Categories : objets explosifs (2 / 1 / 2, dont une bobine avec clé chez JGtm et
// Madina97294) et chute (0 / 1 / 1), par catégorie de source du film.
func soiree2209Categories() []port.KillSourceCategoryRow {
	return []port.KillSourceCategoryRow{
		{XUID: toolXJ, Category: domain.KillSourceCategoryExplosiveObject, WeaponKey: keyToolCoilPlasma, Kills: 1},
		{XUID: toolXJ, Category: domain.KillSourceCategoryExplosiveObject, Kills: 1},
		{XUID: toolXC, Category: domain.KillSourceCategoryExplosiveObject, Kills: 1},
		{XUID: toolXM, Category: domain.KillSourceCategoryExplosiveObject, WeaponKey: keyToolCoilKinetic, Kills: 1},
		{XUID: toolXM, Category: domain.KillSourceCategoryExplosiveObject, Kills: 1},
		{XUID: toolXC, Category: domain.KillSourceCategoryEnvironment, WeaponKey: keyToolEnvironment, Kills: 1},
		{XUID: toolXM, Category: domain.KillSourceCategoryEnvironment, WeaponKey: keyToolEnvironment, Kills: 1},
	}
}

// sheetPoint : la feuille de match d'un joueur sur la soirée (total, mêlée).
func sheetPoint(total, melee int) []domain.SquadPerformanceSeriesPoint {
	m := melee
	return []domain.SquadPerformanceSeriesPoint{{MatchID: "m1", Kills: total, MeleeKills: &m}}
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

func wantKills(t *testing.T, l domain.SquadWeaponToolLine, name string, j, c, m int) {
	t.Helper()
	if l.KillsByPlayer[toolJ] != j || l.KillsByPlayer[toolC] != c || l.KillsByPlayer[toolM] != m {
		t.Errorf("%s = %d / %d / %d, want %d / %d / %d", name,
			l.KillsByPlayer[toolJ], l.KillsByPlayer[toolC], l.KillsByPlayer[toolM], j, c, m)
	}
	if l.TotalSquad != j+c+m {
		t.Errorf("%s total = %d, want %d", name, l.TotalSquad, j+c+m)
	}
}

// TestSquadWeaponTools_Soiree2209 : L2.4 — les chiffres témoins de la maquette C3EW
// retrouvés en passant par buildSquadWeaponKills (chargeur, catégories, feuille de match).
func TestSquadWeaponTools_Soiree2209(t *testing.T) {
	loader := &toolsLoader{
		fakeSquadLoader: &fakeSquadLoader{},
		rows:            soiree2209Rows(),
		categories:      soiree2209Categories(),
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", squadLoader: loader}
	perf := map[string][]domain.SquadPerformanceSeriesPoint{
		toolJ: sheetPoint(65, 6), toolC: sheetPoint(63, 13), toolM: sheetPoint(121, 16),
	}
	teammates := []domain.TeammateRow{teammateWithXUID(toolC, toolXC), teammateWithXUID(toolM, toolXM)}

	tools, _ := svc.buildSquadWeaponKills(context.Background(),
		[]domain.SquadMatchRow{{MatchID: "m1"}}, toolJ, toolXJ, teammates, perf)
	if tools == nil {
		t.Fatal("outils absents")
	}
	if len(tools.Players) != 3 || tools.Players[0] != toolJ {
		t.Errorf("Players = %v, want [JGtm Chocoboflor Madina97294]", tools.Players)
	}
	lines := toolLineByName(t, tools)

	wantKills(t, lines["BR75"], "BR75", 22, 22, 35)
	wantKills(t, lines["Mutilateur"], "Mutilateur", 1, 0, 0)
	wantKills(t, lines["VK78 Commando"], "VK78 Commando", 1, 0, 0)
	frag := lines["Grenade frag"]
	wantKills(t, frag, "grenade à fragmentation", 2, 1, 4)
	if frag.WeaponKey != keyToolFrag || frag.Class != domain.FragClassGrenade || frag.LabelEN != "Frag Grenade" {
		t.Errorf("grenade frag = %+v, want clé %s, classe grenade, nom EN", frag, keyToolFrag)
	}
	wantKills(t, lines["Grenade plasma"], "grenade plasma", 2, 1, 0)
	wantKills(t, lines["Grenade dynamo"], "grenade dynamo", 1, 0, 0)
	// Mêlée depuis la feuille de match, pas depuis la sentinelle ni le film.
	wantKills(t, lines[domain.SquadToolKindMelee], "mêlée", 6, 13, 16)
	// Objets explosifs et chute par catégorie de source ; les lignes de bobine et
	// d'environnement qu'elles recouvrent s'effacent.
	wantKills(t, lines[domain.SquadToolKindExplosiveObject], "objet explosif", 2, 1, 2)
	wantKills(t, lines[domain.SquadToolKindEnvironment], "chute, environnement", 0, 1, 1)
	for _, gone := range []string{"Bobine à plasma", "Bobine à fusion UNSC", "Chute et environnement"} {
		if _, ok := lines[gone]; ok {
			t.Errorf("ligne %q présente : la catégorie aurait dû la reprendre", gone)
		}
	}
	// Feuille = lignes pour JGtm et Madina97294 ; Chocoboflor a une ligne de plus au film
	// qu'à la feuille (reliquat négatif) : aucune ligne « Non attribué ».
	if _, ok := lines[domain.SquadToolKindUnattributed]; ok {
		t.Errorf("« Non attribué » inattendu : %+v", lines[domain.SquadToolKindUnattributed])
	}
	// Plus de plafond : les 17 lignes de la maquette moins « Sans ligne au film ».
	if len(tools.Lines) != 16 {
		t.Errorf("%d lignes, want 16", len(tools.Lines))
	}
	if tools.Lines[0].Label != "BR75" {
		t.Errorf("première ligne = %+v, want BR75 (plus gros total)", tools.Lines[0])
	}
}

// TestBuildSquadWeaponTools_ReliquatEnDernier : le reliquat feuille − lignes forme la
// ligne « Non attribué », toujours dernière, même devant un plus gros total.
func TestBuildSquadWeaponTools_ReliquatEnDernier(t *testing.T) {
	tools := buildSquadWeaponTools(squadToolInputs{
		rows:           []port.WeaponKillRow{filmRow(toolXJ, "hinf_br75", "BR75", "BR75", "shoulder", 2)},
		playersOrdered: []string{toolJ},
		gtByXUID:       map[string]string{toolXJ: toolJ},
		sheet:          map[string]domain.FragKillTypeCounts{toolJ: {Total: 10, Melee: 1}},
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

// TestBuildSquadWeaponTools_MecaniquesNatives : sur un titre aux mécaniques natives
// (capability), les frags de mêlée attribués à l'arme tenue quittent la ligne de l'arme ;
// assassinats et capacités spartanes ont leur ligne ; une arme sans nom rejoint le reliquat.
func TestBuildSquadWeaponTools_MecaniquesNatives(t *testing.T) {
	tools := buildSquadWeaponTools(squadToolInputs{
		rows: []port.WeaponKillRow{
			{XUID: toolXJ, WeaponID: 10, WeaponKey: "h5_br85", Label: "BR85", Class: "shoulder", Kills: 12, MechanicKills: 4},
			{XUID: toolXJ, WeaponID: 99, Kills: 3}, // arme hors registre, sans nom
		},
		playersOrdered: []string{toolJ},
		gtByXUID:       map[string]string{toolXJ: toolJ},
		sheet: map[string]domain.FragKillTypeCounts{
			toolJ: {Total: 25, Melee: 2, Assassination: 3, GroundPound: 1, ShoulderBash: 1},
		},
		hasMechanics: true,
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

// TestBuildSquadWeaponTools_SansMecaniques : capability absente → ni assassinat ni
// capacité spartane, même si la feuille en porte (compteurs à zéro sur ces titres).
func TestBuildSquadWeaponTools_SansMecaniques(t *testing.T) {
	tools := buildSquadWeaponTools(squadToolInputs{
		playersOrdered: []string{toolJ},
		gtByXUID:       map[string]string{toolXJ: toolJ},
		sheet:          map[string]domain.FragKillTypeCounts{toolJ: {Total: 3, Melee: 3, Assassination: 2}},
	})
	if tools == nil || len(tools.Lines) != 1 || tools.Lines[0].Kind != domain.SquadToolKindMelee {
		t.Errorf("lignes = %+v, want la seule mêlée", tools)
	}
}

// TestBuildSquadWeaponTools_Vide : aucun joueur, ou aucun frag → nil.
func TestBuildSquadWeaponTools_Vide(t *testing.T) {
	if got := buildSquadWeaponTools(squadToolInputs{}); got != nil {
		t.Errorf("sans joueur : %+v, want nil", got)
	}
	got := buildSquadWeaponTools(squadToolInputs{
		playersOrdered: []string{toolJ},
		sheet:          map[string]domain.FragKillTypeCounts{toolJ: {}},
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
	if got := playersAboveSheet(tools, sheet); got != 1 {
		t.Errorf("playersAboveSheet = %d, want 1", got)
	}
}
