package grammar

// pont_identite_test.go — L ETAGE UNIQUE DU PONT D IDENTITE (lot J4.3).
//
// TestPontDIdentite_ExemptionsDeTranslocationAppliquees porte la regle que le collecteur
// enfreignait (constat RA1-3) : le balayage des positions du pont recoit les exemptions que les
// teleportations LUES ouvrent au filtre de vitesse, sur la base de l appelant, sans rien y changer
// d autre. Aucune mini-bobine du depot ne porte d evenement de translocateur : les teleportations
// sont injectees par la couture `etageDuPont`, et le balayage des positions est un espion. Les
// quatre autres lectures tournent sur la vraie bobine.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : `optionsDesPositionsDuPont` qui rend la base telle
// quelle ; `lire` qui passe `opt.Balayage` au balayage des positions.

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestPontDIdentite_ExemptionsDeTranslocationAppliquees(t *testing.T) {
	film := chargerMiniBobine(t)
	sauts := []types.TranslocatorTeleport{
		{TimestampUS: 9_000_000, Slot: 7}, {TimestampUS: 3_000_000, Slot: 7},
		{TimestampUS: 5_000_000, Slot: 2},
	}
	base := DefaultScanFilmOptions()
	base.CaptureDirs = true
	var recu *ScanFilmOptions
	e := etageDuPont{
		teleportations: func(*source.Film, *profile.MapQuantEntry) []types.TranslocatorTeleport {
			return sauts
		},
		positions: func(_ *FilmContext, o ScanFilmOptions) ([]BipedPosition, error) {
			recu = &o
			return nil, nil
		},
	}
	l := e.lire(NewFilmContext(film), OptionsDuPont{Balayage: base})
	if recu == nil {
		t.Fatal("le balayage des positions n a pas ete appele par l etage")
	}
	if !reflect.DeepEqual(l.Translocations, sauts) {
		t.Errorf("teleportations rendues %v, attendu les lues %v", l.Translocations, sauts)
	}
	voulu := TeleportExemptionsOf(sauts)
	if !reflect.DeepEqual(recu.TeleportExemptions, voulu) {
		t.Fatalf("exemptions recues par le balayage des positions = %v, attendu %v : le pont "+
			"lirait une teleportation comme un saut impossible et la rejetterait (constat RA1-3)",
			recu.TeleportExemptions, voulu)
	}
	recu.TeleportExemptions = nil
	if !reflect.DeepEqual(*recu, base) {
		t.Errorf("l etage a change la base de l appelant au-dela des exemptions :\n  recu  %+v\n  base  %+v",
			*recu, base)
	}
}

// TestPontDIdentite_IndexSeulementSurUnFilDesMorts : la table d index n est lue que si le fil des
// morts a rendu au moins une mort, avec le roster que l APPELANT construit — sa politique.
func TestPontDIdentite_IndexSeulementSurUnFilDesMorts(t *testing.T) {
	film := chargerMiniBobine(t)
	var vu []types.Death
	l := ScanPontDIdentite(NewFilmContext(film), OptionsDuPont{
		Balayage: DefaultScanFilmOptions(),
		RosterDesMorts: func(d []types.Death) []uint64 {
			vu = d
			out := make([]uint64, 0, len(d))
			for _, m := range d {
				out = append(out, m.XUID)
			}
			return out
		},
	})
	if l.ErrMorts != nil || len(l.Morts) == 0 {
		t.Fatalf("fil des morts de la mini-bobine : %d morts, err %v", len(l.Morts), l.ErrMorts)
	}
	if !l.IndexLu || !reflect.DeepEqual(vu, l.Morts) {
		t.Fatalf("index lu = %v, roster construit sur %d morts pour %d lues", l.IndexLu, len(vu), len(l.Morts))
	}
	sans := ScanPontDIdentite(NewFilmContext(film), OptionsDuPont{Balayage: DefaultScanFilmOptions()})
	if sans.IndexLu {
		t.Error("sans roster fourni par l appelant, la table d index ne se lit pas")
	}
}

// TestPontDIdentite_CreationsLuesAvantLesPositions : l etage lit les CREATIONS de bipede AVANT les
// positions (lot J5.2, DT-8) — elles designent les generations vivantes que le balayage des
// positions applique — et rend l ensemble sous lequel les positions ont ete lues, celui dont la
// cuisson tire le compte du repli `repli_generation_vivante_inconnue_tag1`. MUTATION : remettre la
// lecture des creations APRES celle des positions fait rougir la premiere assertion.
func TestPontDIdentite_CreationsLuesAvantLesPositions(t *testing.T) {
	film := chargerMiniBobine(t)
	fc := NewFilmContext(film)
	creationsDejaLues := false
	e := etageDuPont{
		teleportations: func(*source.Film, *profile.MapQuantEntry) []types.TranslocatorTeleport { return nil },
		positions: func(c *FilmContext, _ ScanFilmOptions) ([]BipedPosition, error) {
			creationsDejaLues = c.vies.creationsLues
			return nil, nil
		},
	}
	l := e.lire(fc, OptionsDuPont{Balayage: DefaultScanFilmOptions()})
	if !creationsDejaLues {
		t.Fatal("le balayage des positions a tourne AVANT la lecture des creations : les generations " +
			"vivantes du handle ne sont pas connues quand les positions sont lues (constat GB-1)")
	}
	if l.Generations == nil || l.Generations != fc.GenerationsVivantes() {
		t.Fatalf("l etage rend %p, attendu les generations vivantes du contexte %p", l.Generations, fc.GenerationsVivantes())
	}
	leve := DefaultScanFilmOptions()
	leve.Generations = ToutesLesGenerations()
	if got := e.lire(NewFilmContext(film), OptionsDuPont{Balayage: leve}).Generations; got != leve.Generations {
		t.Errorf("un filtre fourni par l appelant doit etre celui que l etage rend")
	}
}
