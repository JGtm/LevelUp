package replayartifacts

// Tests — LA PROJECTION DES NIVEAUX D'ARMES, et les portes qui la gouvernent.
//
// CE QU'ILS VERROUILLENT, dans l'ordre des pieges du domaine :
//   - le niveau vient de la CARTE (l'emplacement Forge qui confirme le socle), jamais du nom
//     ni du role de l'arme ;
//   - un artefact anterieur au schema 30 ne produit RIEN : aucune prise n'y est nommee, et une
//     passe de zeros affirmerait que personne n'a rien pris ;
//   - un joueur du roster qui n'a rien pris a une ligne `aucune_prise` — le ZERO MESURE, sans
//     lequel il serait indistinguable d'un joueur d'un match sans film ;
//   - une carte hors reference produit quand meme une passe, `pads_confirmed = 0` : « des
//     socles, mais pas de niveaux » est une mesure ;
//   - les modes a departs ALEATOIRES ne produisent aucun niveau `base` ;
//   - LA PORTE DE CAPABILITY EST CABLEE : la debrancher fait rougir ce fichier (lecon des
//     prises nettes, constats C1/M6/M7).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/persist"
)

// Familles d'arme du temoin, ecrites comme le film les ecrit.
const (
	tAR     = "0x48C19D2D"
	tPistol = "0xF408190F"
	tSniper = "0x9D6AAED2"
	tHydra  = "0xB619D84A"
)

// refTemoin : une carte de reference a trois emplacements — un ratelier, un socle de
// puissance, un socle de bonus.
func refTemoin() *ReferenceEmplacements {
	pos := func(x float64) replay.MapWeaponPadSpot {
		return replay.MapWeaponPadSpot{
			Pos: mapvarVec(x), TypeID: "0x6253CFC0", Family: "rack", Objects: 1,
		}
	}
	rack := pos(0)
	power := pos(10)
	power.Family = "power"
	power.TypeID = "0x5F379533"
	return &ReferenceEmplacements{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion,
		TitleSlug:     "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{
			"carte-1": {MapID: "carte-1", MvarFile: "m.mvar", Pads: []replay.MapWeaponPadSpot{rack, power}},
		},
	}
}

// docTemoin : deux socles (un a chaque emplacement), deux joueurs, deux prises nommees, et un
// troisieme joueur au roster qui n'a rien pris.
func docTemoin(schema int, loadouts []replay.Loadout) *replay.ReplayDocument {
	a, b := "a1", "b1"
	return &replay.ReplayDocument{
		SchemaVersion: schema,
		MatchID:       "m1",
		Roster: []replay.RosterEntry{
			{XUID: "a1", Name: "Alpha"}, {XUID: "b1", Name: "Bravo"},
			{XUID: "c1", Name: "Charlie"},
			{XUID: "", Name: "BotZero", Bot: true},
		},
		WeaponPads: []replay.WeaponPad{
			{X: 0, Y: 0, Z: 0, Weapon: tHydra},
			{X: 10, Y: 0, Z: 0, Weapon: tSniper},
		},
		PadPickups: []replay.PadPickup{
			{Pad: 0, TLow: 5, THigh: 15, XUID: &a},
			{Pad: 1, TLow: 20, THigh: 30, XUID: &b},
			{Pad: 1, TLow: 40, THigh: 50, XUID: nil}, // sans ramasseur : jamais attribuee
		},
		Loadouts: loadouts,
	}
}

// viesDepart : n vies partant avec `w`, chacune sur son slot.
func viesDepart(n int, w []string) []replay.Loadout {
	out := make([]replay.Loadout, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, replay.Loadout{T: 74, Slot: uint32(512 + i), W: w})
	}
	return out
}

// ligne rend la ligne (xuid, niveau) de la passe, ou une chaine vide.
func ligne(b persist.PadTiersBatch, xuid, tier string) (persist.PadTierRow, bool) {
	for _, r := range b.Rows {
		if r.XUID == xuid && r.Tier == tier {
			return r, true
		}
	}
	return persist.PadTierRow{}, false
}

func TestProjeterNiveaux_LeNiveauVientDeLaCarte(t *testing.T) {
	b := ProjeterNiveauxDArmes("m1", docTemoin(30, viesDepart(20, []string{tAR, tPistol})),
		refTemoin(), IdentiteMatchNiveaux{MapID: "carte-1", PairName: "Arena:Slayer"}, false)
	if b.MatchID != "m1" {
		t.Fatalf("passe vide : %+v", b)
	}
	if b.PadsTotal != 2 || b.PadsConfirmed != 2 {
		t.Errorf("socles = %d/%d, attendu 2/2", b.PadsConfirmed, b.PadsTotal)
	}
	// L'HYDRA A UN ROLE « power » AU REGISTRE et se trouve sur un RATELIER : elle est de
	// TERRAIN. Si le niveau se lisait sur l'arme, cette ligne rendrait « puissance ».
	if r, ok := ligne(b, "a1", persist.PadTierGround); !ok || r.Pickups != 1 {
		t.Errorf("Alpha devrait avoir 1 prise de terrain : %+v", b.Rows)
	}
	if r, ok := ligne(b, "b1", persist.PadTierPower); !ok || r.Pickups != 1 {
		t.Errorf("Bravo devrait avoir 1 prise de puissance : %+v", b.Rows)
	}
	// LE ZERO MESURE : Charlie n'a rien pris, mais il a une ligne.
	if r, ok := ligne(b, "c1", persist.PadTierNoPickup); !ok || r.Pickups != 0 || r.WeaponFamily != "" {
		t.Errorf("Charlie devrait avoir une ligne %s a zero : %+v", persist.PadTierNoPickup, b.Rows)
	}
	// LE BOT N'EN A PAS : il n'a pas de xuid, et la table est clef par xuid.
	for _, r := range b.Rows {
		if r.XUID == "" {
			t.Errorf("une ligne sans xuid est partie en base : %+v", r)
		}
	}
	// La passe doit passer la validation du persister — sinon elle serait refusee en prod.
	if err := validerPasse(b); err != nil {
		t.Errorf("passe refusee par le persister : %v", err)
	}
}

func TestProjeterNiveaux_SchemaTropAncien(t *testing.T) {
	// Schema 29 : `padPickups[].xuid` n'existe pas encore. Une passe de zeros affirmerait que
	// personne n'a pris de socle.
	b := ProjeterNiveauxDArmes("m1", docTemoin(PadTiersMinSchema-1, nil),
		refTemoin(), IdentiteMatchNiveaux{MapID: "carte-1"}, false)
	if b.MatchID != "" {
		t.Fatalf("passe produite sur un artefact trop ancien : %+v", b)
	}
}

func TestProjeterNiveaux_CarteHorsReference(t *testing.T) {
	b := ProjeterNiveauxDArmes("m1", docTemoin(30, viesDepart(20, []string{tAR, tPistol})),
		refTemoin(), IdentiteMatchNiveaux{MapID: "carte-inconnue"}, false)
	if b.MatchID == "" {
		t.Fatal("la passe doit etre PRODUITE : « des socles, mais pas de niveaux » est une mesure")
	}
	if b.PadsConfirmed != 0 || b.PadsTotal != 2 {
		t.Errorf("socles = %d/%d, attendu 0/2", b.PadsConfirmed, b.PadsTotal)
	}
	// Tout tombe en non classe — et surtout PAS en terrain par defaut.
	if _, ok := ligne(b, "a1", persist.PadTierUnclassified); !ok {
		t.Errorf("Alpha devrait etre non classe : %+v", b.Rows)
	}
	if _, ok := ligne(b, "a1", persist.PadTierGround); ok {
		t.Errorf("aucun niveau « terrain » ne doit sortir d'une carte hors reference : %+v", b.Rows)
	}
}

func TestProjeterNiveaux_ArmeDeBase(t *testing.T) {
	// L'AR est l'arme de depart des vingt vies, et le socle 0 la porte : elle passe en `base`
	// malgre son ratelier (ordre base > terrain > puissance).
	doc := docTemoin(30, viesDepart(20, []string{tAR, tPistol}))
	doc.WeaponPads[0].Weapon = tAR
	b := ProjeterNiveauxDArmes("m1", doc, refTemoin(),
		IdentiteMatchNiveaux{MapID: "carte-1", PairName: "Arena:Slayer"}, false)
	if _, ok := ligne(b, "a1", persist.PadTierBase); !ok {
		t.Errorf("Alpha devrait avoir une prise de base : %+v", b.Rows)
	}
}

func TestProjeterNiveaux_DepartsAleatoires(t *testing.T) {
	doc := docTemoin(30, viesDepart(20, []string{tAR, tPistol}))
	doc.WeaponPads[0].Weapon = tAR
	b := ProjeterNiveauxDArmes("m1", doc, refTemoin(),
		IdentiteMatchNiveaux{MapID: "carte-1", PairName: "Super Fiesta:Slayer"}, true)
	if !b.RandomStarts {
		t.Error("RandomStarts devrait etre vrai")
	}
	if _, ok := ligne(b, "a1", persist.PadTierBase); ok {
		t.Errorf("aucun niveau « base » ne doit sortir d'un mode a departs aleatoires : %+v", b.Rows)
	}
	// Les deux autres niveaux restent lisibles.
	if _, ok := ligne(b, "a1", persist.PadTierGround); !ok {
		t.Errorf("le niveau « terrain » doit rester : %+v", b.Rows)
	}
}

// TestPorteCapabiliteNiveaux_EstCABLEE — LE GARDE-RAIL DE LA PORTE.
//
// Il ne teste pas une fonction : il teste que le CABLAGE existe. Debrancher
// `capabiliteNiveauxArmee` de `persisterNiveauxDArmes`, ou la remplacer par un `true` en dur,
// fait rougir ce test — c'est exactement le defaut que la revue des prises nettes a releve
// (constats C1/M6/M7 : une porte ecrite mais jamais franchie).
func TestPorteCapabiliteNiveaux_EstCABLEE(t *testing.T) {
	src, err := os.ReadFile("padtiers.go")
	if err != nil {
		t.Fatalf("lecture de padtiers.go : %v", err)
	}
	txt := string(src)
	for _, attendu := range []string{
		// la porte est declaree...
		"porteCapability(ctx, d, games.CapFilmWeaponTiers,",
		// ...et elle est FRANCHIE avant toute lecture de reference ou ecriture.
		"armee, incident := capabiliteNiveauxArmee(ctx, d)",
		"if !armee {",
	} {
		if !strings.Contains(txt, attendu) {
			t.Errorf("la porte de capability des niveaux d'armes n'est plus cablee : %q absent "+
				"de padtiers.go — la projection ecrirait sur un titre qui ne la declare pas", attendu)
		}
	}
	// La capability doit etre au vocabulaire canonique, sinon le TOML ne peut pas l'armer.
	if !games.IsKnownCapabilityKey(games.CapFilmWeaponTiers) {
		t.Error("film.weapon_tiers absente de AllCapabilityKeys() : le TOML du titre ne peut pas l'armer")
	}
	// Et la derivation doit etre appelee par l'orchestrateur.
	der, err := os.ReadFile("derivations.go")
	if err != nil {
		t.Fatalf("lecture de derivations.go : %v", err)
	}
	if !strings.Contains(string(der), "persisterNiveauxDArmes(ctx, d, b, lus)") {
		t.Error("persisterNiveauxDArmes n'est plus appelee par Deriver : la grandeur ne serait " +
			"JAMAIS produite au fil de l'eau, et rien d'autre ne rougirait")
	}
}

// TestRegleDepartsAleatoires_LueDuTitre — la seconde porte vient du TOML, jamais du Go.
func TestRegleDepartsAleatoires_LueDuTitre(t *testing.T) {
	reg, err := mappings.LoadRegulationFromFile(
		filepath.Join("..", "..", "..", "..", "..", "config", "titles", "halo_infinite",
			"mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("regulation.toml illisible : %v", err)
	}
	if len(reg.RandomStartModePrefixes()) == 0 {
		t.Fatal("le titre ne declare aucun mode a departs aleatoires")
	}
	for _, cas := range []struct {
		pair string
		want bool
	}{
		{"Super Fiesta:Slayer", true},
		{"Fiesta:CTF", true},
		{"Husky Raid", true},
		{"Arena:Slayer", false},
		{"BTB:Total Control", false},
		{"", false},
		// Le prefixe se coupe sur `:` : « Fiestaval » n'est pas « Fiesta ».
		{"Fiestaval:Slayer", false},
	} {
		if got := reg.HasRandomStarts(cas.pair); got != cas.want {
			t.Errorf("HasRandomStarts(%q) = %v, attendu %v", cas.pair, got, cas.want)
		}
	}
}

// validerPasse rejoue la validation du persister sans base : c'est la seule facon de prouver
// ici qu'une passe projetee serait ACCEPTEE en production.
func validerPasse(b persist.PadTiersBatch) error {
	blob, err := json.Marshal(b)
	if err != nil {
		return err
	}
	var relu persist.PadTiersBatch
	if err := json.Unmarshal(blob, &relu); err != nil {
		return err
	}
	return persist.ValidatePadTiersBatch(relu)
}

// mapvarVec : une position d emplacement sur l axe X, les deux autres a zero.
func mapvarVec(x float64) mapvar.Vec3 { return mapvar.Vec3{X: x} }
