package weapontier

import (
	"testing"
)

// Familles d'arme du témoin, écrites comme le film les écrit.
const (
	ar     = "0x48C19D2D" // MA40 — arme de départ
	pistol = "0xF408190F" // Sidekick — arme de départ
	sniper = "0x9D6AAED2" // S7 — socle de puissance
	hydra  = "0xB619D84A" // Hydra — râtelier, malgré son rôle `power`
	bonus  = "powerup_overshield"
)

// temoin — quatre socles : un râtelier, un socle de puissance, un bonus, et un socle
// qu'aucun emplacement ne confirme.
func temoin() ([]Pad, []Spot) {
	pads := []Pad{
		{Weapon: hydra},  // 0 — râtelier
		{Weapon: sniper}, // 1 — puissance
		{Weapon: bonus},  // 2 — bonus
		{Weapon: ar},     // 3 — NON confirmé
	}
	cross := []Spot{
		{Pad: 0, Family: "rack"},
		{Pad: 1, Family: "power"},
		{Pad: 2, Family: "powerup"},
	}
	return pads, cross
}

// vies fabrique un canal `loadouts` : n vies portant `w`, chacune sur son propre slot, avec
// une SECONDE émission par vie (le canal ré-émet après un changement d'arme) pour que les
// tests prouvent que seule la première compte.
func vies(depart int, w []string, apres []string) []Spawn {
	var out []Spawn
	for i := 0; i < depart; i++ {
		slot := uint32(512 + i)
		out = append(out, Spawn{Slot: slot, Weapons: w})
		if apres != nil {
			out = append(out, Spawn{Slot: slot, Weapons: apres})
		}
	}
	return out
}

func TestTierOf_LesQuatreNiveaux(t *testing.T) {
	pads, cross := temoin()
	m := NewMatch(pads, cross, vies(20, []string{ar, pistol}, []string{sniper, pistol}), false)

	cas := []struct {
		nom  string
		pad  int
		arme string
		want Tier
	}{
		{"râtelier", 0, hydra, TierGround},
		{"socle de puissance", 1, sniper, TierPower},
		{"socle de bonus", 2, bonus, TierPowerup},
		{"socle non confirmé, arme de base", 3, ar, TierBase},
	}
	for _, c := range cas {
		if got := m.TierOf(c.pad, c.arme); got != c.want {
			t.Errorf("%s : niveau = %q, attendu %q", c.nom, got, c.want)
		}
	}
	if m.Lives() != 20 {
		t.Errorf("vies lues = %d, attendu 20", m.Lives())
	}
}

// TestTierOf_NonClasseQuandRienNeConfirme — D4 : un socle sans emplacement confirmé reste
// non classé et visible ; il ne tombe JAMAIS dans « terrain » par défaut.
func TestTierOf_NonClasseQuandRienNeConfirme(t *testing.T) {
	pads, _ := temoin()
	m := NewMatch(pads, nil, vies(20, []string{ar, pistol}, nil), false)
	if got := m.TierOf(1, sniper); got != TierUnclassified {
		t.Errorf("sans référence de carte : niveau = %q, attendu %q", got, TierUnclassified)
	}
	// Et un index de socle hors bornes ne pioche pas le voisin.
	if got := m.TierOf(99, sniper); got != TierUnclassified {
		t.Errorf("index hors bornes : niveau = %q, attendu %q", got, TierUnclassified)
	}
}

// TestTierOf_BasePrimeSurLEmplacement — l'ordre du plan : base > terrain > puissance.
func TestTierOf_BasePrimeSurLEmplacement(t *testing.T) {
	pads := []Pad{{Weapon: ar}}
	cross := []Spot{{Pad: 0, Family: "rack"}}
	m := NewMatch(pads, cross, vies(20, []string{ar, pistol}, nil), false)
	if got := m.TierOf(0, ar); got != TierBase {
		t.Errorf("arme de départ sur râtelier : niveau = %q, attendu %q", got, TierBase)
	}
}

// TestBaseWeapons_SeulePremiereEmission — le piège mesuré à l'étape 0 : le canal ré-émet.
func TestBaseWeapons_SeulePremiereEmission(t *testing.T) {
	pads := []Pad{{Weapon: sniper}}
	cross := []Spot{{Pad: 0, Family: "power"}}
	// Les vingt vies démarrent AR+Sidekick puis ramassent toutes le sniper : s'il comptait,
	// le sniper serait « base » et le niveau « puissance » disparaîtrait du match.
	m := NewMatch(pads, cross, vies(20, []string{ar, pistol}, []string{sniper, pistol}), false)
	if got := m.TierOf(0, sniper); got != TierPower {
		t.Errorf("arme ramassée en cours de vie : niveau = %q, attendu %q", got, TierPower)
	}
}

// TestBaseWeapons_SeuilDeQueue — la queue de 5,6 % mesurée à l'étape 0 ne promeut rien.
func TestBaseWeapons_SeuilDeQueue(t *testing.T) {
	pads := []Pad{{Weapon: sniper}}
	cross := []Spot{{Pad: 0, Family: "power"}}
	l := vies(40, []string{ar, pistol}, nil)
	// Une seule vie sur 41 démarre sniper en main : 2,4 %, sous BaseShareMin.
	l = append(l, Spawn{Slot: 999, Weapons: []string{sniper, pistol}})
	m := NewMatch(pads, cross, l, false)
	if got := m.TierOf(0, sniper); got != TierPower {
		t.Errorf("arme de la queue : niveau = %q, attendu %q", got, TierPower)
	}
	// Douze vies sur 52 (23 %) suffisent en revanche.
	for i := 0; i < 11; i++ {
		l = append(l, Spawn{Slot: uint32(900 + i), Weapons: []string{sniper, pistol}})
	}
	if got := NewMatch(pads, cross, l, false).TierOf(0, sniper); got != TierBase {
		t.Errorf("arme de départ franche : niveau = %q, attendu %q", got, TierBase)
	}
}

// TestTierOf_DepartsAleatoires — Fiesta : aucun niveau « base », les deux autres restent.
func TestTierOf_DepartsAleatoires(t *testing.T) {
	pads, cross := temoin()
	// En Fiesta l'AR est bien distribué au départ, mais le niveau ne doit pas exister.
	m := NewMatch(pads, cross, vies(20, []string{ar, pistol}, nil), true)
	if !m.RandomStarts() {
		t.Fatal("RandomStarts devrait être vrai")
	}
	if got := m.TierOf(3, ar); got != TierUnclassified {
		t.Errorf("départs aléatoires, socle non confirmé : niveau = %q, attendu %q", got, TierUnclassified)
	}
	if got := m.TierOf(0, hydra); got != TierGround {
		t.Errorf("départs aléatoires, râtelier : niveau = %q, attendu %q", got, TierGround)
	}
	if got := m.TierOf(1, sniper); got != TierPower {
		t.Errorf("départs aléatoires, puissance : niveau = %q, attendu %q", got, TierPower)
	}
	// Les vies restent comptées : l'écran doit pouvoir dire sur quoi il s'appuie.
	if m.Lives() != 20 {
		t.Errorf("vies lues = %d, attendu 20", m.Lives())
	}
}

// TestMatchZero_Utilisable — le zéro classe tout en non classé sans paniquer.
func TestMatchZero_Utilisable(t *testing.T) {
	var m Match
	if got := m.TierOf(0, ar); got != TierUnclassified {
		t.Errorf("Match zéro : niveau = %q, attendu %q", got, TierUnclassified)
	}
	if m.Lives() != 0 || m.RandomStarts() {
		t.Errorf("Match zéro : vies=%d aléatoire=%v", m.Lives(), m.RandomStarts())
	}
}

func TestCrossCheck_UnSeulSens(t *testing.T) {
	pads, cross := temoin()
	roles := map[string]string{hydra: "power", sniper: "sniper", ar: "automatic", pistol: "sidearm"}
	roleOf := func(w string) string { return roles[w] }

	m := NewMatch(pads, cross, nil, false)
	c := m.RunCrossCheck(pads, roleOf)
	// L'Hydra (rôle `power`) sur râtelier NE COMPTE PAS : c'est le cas nominal à 10,5 %.
	if c.LightOnPower != 0 {
		t.Errorf("aucun rôle léger sur puissance attendu, obtenu %d (%v)", c.LightOnPower, c.Weapons)
	}
	if c.Pads != 4 {
		t.Errorf("socles examinés = %d, attendu 4", c.Pads)
	}
	if c.Alert() {
		t.Error("alerte levée sur un match sain")
	}

	// Un AR (rôle `automatic`) sur le socle de puissance : c'est le sens suspect.
	pads[1] = Pad{Weapon: ar}
	c = m.RunCrossCheck(pads, roleOf)
	if c.LightOnPower != 1 || c.Weapons[ar] != 1 {
		t.Fatalf("inversion non comptée : %+v", c)
	}
	// 1 sur 4 = 25 %, très au-dessus des 2 %.
	if !c.Alert() {
		t.Error("alerte non levée à 25 % de socles inversés")
	}
}

// TestCrossCheck_ShotgunNestPasLeger — le fusil à pompe EST une arme de socle de puissance.
func TestCrossCheck_ShotgunNestPasLeger(t *testing.T) {
	pads := []Pad{{Weapon: "0xSHOT"}}
	cross := []Spot{{Pad: 0, Family: "power"}}
	m := NewMatch(pads, cross, nil, false)
	c := m.RunCrossCheck(pads, func(string) string { return "shotgun" })
	if c.LightOnPower != 0 || c.Alert() {
		t.Errorf("le fusil à pompe ne doit pas alerter : %+v", c)
	}
}
