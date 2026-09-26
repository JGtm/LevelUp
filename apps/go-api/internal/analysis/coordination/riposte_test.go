package coordination

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// equipesRiposte : victime et vengeur en equipe 0, tueur en equipe 1.
func equipesRiposte() domain.EquipesParMatch {
	return domain.EquipesParMatch{"m1": {"v": 0, "ami": 0, "tueur": 1}}
}

func journalRiposte(delaiMs int64) []domain.KillEvent {
	return []domain.KillEvent{
		{MatchID: "m1", KillerXUID: "tueur", VictimXUID: "v", TimeMs: 10_000},
		{MatchID: "m1", KillerXUID: "ami", VictimXUID: "tueur", TimeMs: 10_000 + delaiMs},
	}
}

// TestRipostes_SansBorne : une riposte HORS fenetre est vue par Ripostes et ignoree par
// Echanges. C'est toute la raison d'etre de la fonction : les deux dernieres barres de
// l'histogramme du delai n'existent pas autrement.
func TestRipostes_SansBorne(t *testing.T) {
	const tardive = 40_000

	bilan := Echanges(journalRiposte(tardive), equipesRiposte())
	if bilan.NbVengees != 0 {
		t.Fatalf("Echanges compte %d vengeance(s) a %d ms, attendu 0 (fenetre %d ms)",
			bilan.NbVengees, tardive, FenetreEchangeMs)
	}

	morts := Ripostes(journalRiposte(tardive), equipesRiposte())
	trouvee := false
	for _, m := range morts {
		if m.VictimeXUID != "v" {
			continue
		}
		trouvee = true
		if !m.Vengee || m.DelaiMs != tardive || m.VengeurXUID != "ami" {
			t.Errorf("riposte = %+v, attendue {Vengee, DelaiMs %d, ami}", m, tardive)
		}
	}
	if !trouvee {
		t.Fatal("la mort de la victime doit figurer dans les ripostes")
	}
}

// TestRipostes_CoincideSousLaFenetre : sous la fenetre, les deux lectures rendent le MEME
// vengeur et le MEME delai. C'est cet invariant qui autorise l'histogramme entier (barres
// dans la fenetre COMPRISES) a se construire sur la seule lecture sans borne — s'il
// tombait, l'histogramme et le taux ne parleraient plus de la meme population.
func TestRipostes_CoincideSousLaFenetre(t *testing.T) {
	for _, delai := range []int64{0, 1, 2_500, FenetreEchangeMs - 1, FenetreEchangeMs} {
		bilan := Echanges(journalRiposte(delai), equipesRiposte())
		morts := Ripostes(journalRiposte(delai), equipesRiposte())
		if len(bilan.Morts) != len(morts) {
			t.Fatalf("delai %d : %d morts vs %d", delai, len(bilan.Morts), len(morts))
		}
		for i := range morts {
			a, b := bilan.Morts[i], morts[i]
			if a.Vengee != b.Vengee || a.DelaiMs != b.DelaiMs || a.VengeurXUID != b.VengeurXUID {
				t.Errorf("delai %d, mort %d : Echanges %+v vs Ripostes %+v", delai, i, a, b)
			}
		}
	}
}

// TestRipostes_MemeCasLimites : Ripostes ne desserre QUE la borne de temps. Un tueur mort
// de l'environnement ne riposte de rien, et une mort dont le tueur est inconnu n'est pas
// vengeable — a 40 s comme a 3 s.
func TestRipostes_MemeCasLimites(t *testing.T) {
	t.Run("tueur mort sans auteur", func(t *testing.T) {
		kills := []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "tueur", VictimXUID: "v", TimeMs: 10_000},
			{MatchID: "m1", KillerXUID: "", VictimXUID: "tueur", TimeMs: 50_000},
		}
		for _, m := range Ripostes(kills, equipesRiposte()) {
			if m.VictimeXUID == "v" && m.Vengee {
				t.Errorf("mort %+v vengee : une chute ne venge personne", m)
			}
		}
	})
	t.Run("tueur inconnu", func(t *testing.T) {
		kills := []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "", VictimXUID: "v", TimeMs: 10_000},
			{MatchID: "m1", KillerXUID: "ami", VictimXUID: "tueur", TimeMs: 12_000},
		}
		for _, m := range Ripostes(kills, equipesRiposte()) {
			if m.VictimeXUID == "v" && m.Vengeable {
				t.Errorf("mort %+v vengeable : personne ne pouvait la venger", m)
			}
		}
	})
	t.Run("riposte par un adversaire", func(t *testing.T) {
		equipes := domain.EquipesParMatch{"m1": {"v": 0, "tueur": 1, "autre": 1}}
		kills := []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "tueur", VictimXUID: "v", TimeMs: 10_000},
			{MatchID: "m1", KillerXUID: "autre", VictimXUID: "tueur", TimeMs: 20_000},
		}
		for _, m := range Ripostes(kills, equipes) {
			if m.VictimeXUID == "v" && m.Vengee {
				t.Errorf("mort %+v vengee : le vengeur doit etre un COEQUIPIER", m)
			}
		}
	})
}
