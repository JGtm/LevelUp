package coordination

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// kill fabrique un kill du journal : killer tue victim a t.
func kill(t int64, killer, killerNom, victim, victimNom string) domain.KillDuJournal {
	return domain.KillDuJournal{
		MatchID: "m1", TimeMs: t, KillerXUID: killer, KillerGamertag: killerNom,
		VictimXUID: victim, VictimGamertag: victimNom,
	}
}

// equipeDeLaVictime : 111 (la victime) et ses coequipiers 222 et 333.
func equipeDeLaVictime(xuid string) bool { return xuid == "222" || xuid == "333" }

func TestTueurAbattu_BornesDeLaFenetre(t *testing.T) {
	mort := kill(10_000, "900", "Tueur", "111", "Victime")
	cas := []struct {
		nom   string
		delai int64
		want  bool
	}{
		{"delai 0 (echange simultane)", 0, true},
		{"borne haute incluse (3 000 ms)", 3000, true},
		{"juste au-dela (3 001 ms)", 3001, false},
		{"avant la mort (-1 ms)", -1, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			journal := []domain.KillDuJournal{mort, kill(10_000+c.delai, "222", "Allie", "900", "Tueur")}
			r, ok := TueurAbattu(journal, mort, equipeDeLaVictime)
			if ok != c.want {
				t.Fatalf("ok = %v, want %v", ok, c.want)
			}
			if ok && (r.DelaiMs != c.delai || r.Type != domain.TacticalRiposteTueurAbattu || r.ParXUID != "222") {
				t.Fatalf("riposte = %+v", r)
			}
		})
	}
}

func TestTueurAbattu_PlusieursVengeurs_LePlusTotPuisLeGamertag(t *testing.T) {
	mort := kill(10_000, "900", "Tueur", "111", "Victime")
	journal := []domain.KillDuJournal{
		mort,
		kill(11_500, "222", "Zed", "900", "Tueur"),
		kill(11_000, "333", "Yan", "900", "Tueur"),
		kill(11_000, "222", "Abel", "900", "Tueur"),
	}
	r, ok := TueurAbattu(journal, mort, equipeDeLaVictime)
	if !ok || r.ParGamertag != "Abel" || r.DelaiMs != 1000 {
		t.Fatalf("riposte = %+v ok=%v, attendu Abel a 1 000 ms (egalite de temps departagee par le gamertag)", r, ok)
	}
}

func TestTueurAbattu_NeCompteQuUnCoequipierDeLaVictime(t *testing.T) {
	mort := kill(10_000, "900", "Tueur", "111", "Victime")
	// un adversaire (950) et un bot (sans xuid) abattent le tueur : ni l'un ni l'autre n'est coequipier.
	journal := []domain.KillDuJournal{mort, kill(10_500, "950", "Autre", "900", "Tueur"), kill(10_600, "", "Bot", "900", "Tueur")}
	if r, ok := TueurAbattu(journal, mort, equipeDeLaVictime); ok {
		t.Fatalf("riposte %+v : un non-coequipier ne venge pas la victime", r)
	}
}

func TestTueurAbattu_TueurBot_ParGamertag(t *testing.T) {
	mort := kill(10_000, "", "Bot Elite", "111", "Victime")
	journal := []domain.KillDuJournal{mort, kill(10_800, "222", "Allie", "", "Bot Elite")}
	r, ok := TueurAbattu(journal, mort, equipeDeLaVictime)
	if !ok || r.DelaiMs != 800 {
		t.Fatalf("riposte = %+v ok=%v : un tueur bot s'identifie par son gamertag", r, ok)
	}
	// un bot homonyme d'un HUMAIN ne se confond pas avec lui.
	journal = []domain.KillDuJournal{mort, kill(10_800, "222", "Allie", "777", "Bot Elite")}
	if _, ok := TueurAbattu(journal, mort, equipeDeLaVictime); ok {
		t.Fatal("un humain de meme nom qu'un bot n'est pas ce bot")
	}
}

func TestTueurAbattu_TueurInconnu(t *testing.T) {
	mort := kill(10_000, "", "", "111", "Victime")
	if TueurConnu(mort) {
		t.Fatal("tueur sans identite : non connu")
	}
	if _, ok := TueurAbattu([]domain.KillDuJournal{mort, kill(10_100, "222", "Allie", "", "")}, mort, equipeDeLaVictime); ok {
		t.Fatal("un tueur inconnu ne peut pas etre rapproche d'un kill")
	}
}

func TestAbattuEnRetour_BornesEtParQui(t *testing.T) {
	// 222 frag a 10 000 ; il est tue 3 000 ms plus tard par un bot : retour, par n'importe qui.
	journal := []domain.KillDuJournal{
		kill(10_000, "222", "Allie", "900", "Cible"),
		kill(13_000, "", "Bot Elite", "222", "Allie"),
	}
	r, ok := AbattuEnRetour(journal, "m1", "222", 10_000)
	if !ok || r.DelaiMs != 3000 || r.ParGamertag != "Bot Elite" || r.Type != domain.TacticalRiposteAbattuEnRetour {
		t.Fatalf("riposte = %+v ok=%v", r, ok)
	}
	journal[1].TimeMs = 13_001
	if _, ok := AbattuEnRetour(journal, "m1", "222", 10_000); ok {
		t.Fatal("3 001 ms : hors fenetre")
	}
	if _, ok := AbattuEnRetour(journal, "m1", "", 10_000); ok {
		t.Fatal("sans xuid : rien a chercher")
	}
}
