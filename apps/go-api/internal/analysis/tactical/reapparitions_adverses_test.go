package tactical

// reapparitions_adverses_test.go — la lecture « reapparitions adverses » : la regle « zone tenue » (au
// moins deux vivants de l'equipe dans le volume), le rangement par zone nommee, les planchers.
// Geometrie synthetique, utilise les helpers `carre` et `zone` de zones_test.go.

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

const equipeMoi = 0

// vivant pose un joueur vivant de l'equipe `eq` en (x, y, z).
func vivant(eq int, x, y, z float64) Vivant { return Vivant{XUID: "x", Equipe: eq, X: x, Y: y, Z: z} }

// La zone choisie : un carre de 0 a 10 m, tranche de -1 a 3 m.
func zoneChoisie() ZoneNommee { return zone("Hall", 1, carre(0, 0, 10, 10), -1, 3) }

func TestZoneTenue_DeuxVivantsDeLEquipeDansLeVolume(t *testing.T) {
	z := zoneChoisie()
	cas := []struct {
		nom        string
		instantane []Vivant
		attendu    bool
	}{
		{"deux dedans", []Vivant{vivant(equipeMoi, 2, 2, 0), vivant(equipeMoi, 8, 8, 1)}, true},
		{"un seul dedans", []Vivant{vivant(equipeMoi, 2, 2, 0), vivant(equipeMoi, 20, 20, 0)}, false},
		{"un adversaire ne compte pas", []Vivant{vivant(equipeMoi, 2, 2, 0), vivant(1, 3, 3, 0)}, false},
		{"mauvais etage : la tranche compte", []Vivant{vivant(equipeMoi, 2, 2, 0), vivant(equipeMoi, 3, 3, 9)}, false},
		{"marge de tranche", []Vivant{vivant(equipeMoi, 2, 2, 3.2), vivant(equipeMoi, 3, 3, -1.2)}, true},
		{"trois dedans", []Vivant{vivant(equipeMoi, 1, 1, 0), vivant(equipeMoi, 2, 2, 0), vivant(equipeMoi, 3, 3, 0)}, true},
		{"vie non nommee compte", []Vivant{{Equipe: equipeMoi, X: 1, Y: 1}, {Equipe: equipeMoi, X: 2, Y: 2}}, true},
		{"instantane vide", nil, false},
	}
	for _, c := range cas {
		if got := ZoneTenue(z, equipeMoi, c.instantane); got != c.attendu {
			t.Errorf("%s : ZoneTenue = %v, attendu %v", c.nom, got, c.attendu)
		}
	}
	if ZoneTenue(ZoneNommee{VolumeIndex: 9, ZHaut: 3}, equipeMoi, []Vivant{vivant(0, 1, 1, 0), vivant(0, 2, 2, 0)}) {
		t.Error("une zone sans forme ne contient personne")
	}
}

// Une zone qui en contient une autre tient pour les joueurs de l'une comme de l'autre : l'appartenance est
// le contenu du volume choisi, pas le nom en jeu du point.
func TestZoneTenue_LaZoneEnglobanteContientLesPointsDeLaZoneImbriquee(t *testing.T) {
	grande := zone("Grande", 1, carre(0, 0, 20, 20), -1, 6)
	if !ZoneTenue(grande, equipeMoi, []Vivant{vivant(equipeMoi, 2, 2, 0), vivant(equipeMoi, 15, 15, 4)}) {
		t.Error("la grande zone contient les deux joueurs, quel que soit le nom de leur point")
	}
}

// obsEn pose une reapparition adverse en (x, y, z) du match `m`, avec `n` equipiers dans la zone choisie.
func obsEn(m string, x, y, z float64, n int) ObservationAdverse {
	var inst []Vivant
	for i := 0; i < n; i++ {
		inst = append(inst, vivant(equipeMoi, 1+float64(i), 1, 0))
	}
	inst = append(inst, vivant(1, 50, 50, 0))
	return ObservationAdverse{MatchID: m, X: x, Y: y, Z: z, EquipeActive: equipeMoi, Instantane: inst}
}

func TestLireReapparitionsAdverses_ReferenceEtTenue(t *testing.T) {
	zones := []ZoneNommee{
		zoneChoisie(),
		zone("Sortie", 2, carre(20, 0, 30, 10), -1, 3),
		zone("Cour", 3, carre(40, 0, 50, 10), -1, 3),
	}
	var obs []ObservationAdverse
	// Cinq matchs. Par match : 4 reapparitions a Sortie dont 3 quand la zone est tenue, 2 a Cour
	// sans qu'elle le soit, et 1 hors de toute zone avec la zone tenue.
	for _, m := range []string{"m1", "m2", "m3", "m4", "m5"} {
		for i := 0; i < 3; i++ {
			obs = append(obs, obsEn(m, 25, 5, 0, 2))
		}
		obs = append(obs, obsEn(m, 25, 5, 0, 1))
		obs = append(obs, obsEn(m, 45, 5, 0, 1), obsEn(m, 45, 5, 0, 0))
		obs = append(obs, obsEn(m, 90, 90, 0, 3))
	}
	ref, tenue := LireReapparitionsAdverses(i18n.Default, obs, zoneChoisie(), zones)

	if ref.Reapparitions != 35 || ref.Matchs != 5 || !ref.Suffisante {
		t.Fatalf("reference = %+v, attendu 35 reapparitions sur 5 matchs, suffisante", ref)
	}
	if tenue.Reapparitions != 20 || tenue.Matchs != 5 || !tenue.Suffisante {
		t.Fatalf("tenue = %+v, attendu 20 reapparitions (3 a Sortie + 1 hors zone, par match) sur 5 matchs", tenue)
	}
	if len(tenue.Zones) != 1 || tenue.Zones[0].VolumeIndex != 2 || tenue.Zones[0].Reapparitions != 15 || tenue.HorsZone() != 5 {
		t.Fatalf("zones tenues = %+v, sans zone = %d : attendu Sortie x15 et 5 hors zone", tenue.Zones, tenue.HorsZone())
	}
	if p := tenue.Zones[0].Part; p != 15.0/20.0 {
		t.Errorf("part = %v, attendu 0,75", p)
	}
	// Reference : Sortie 20, Cour 10, hors zone 5 ; Sortie d'abord.
	if len(ref.Zones) != 2 || ref.Zones[0].VolumeIndex != 2 || ref.Zones[0].Reapparitions != 20 ||
		ref.Zones[1].VolumeIndex != 3 || ref.Zones[1].Reapparitions != 10 || ref.HorsZone() != 5 {
		t.Fatalf("zones de reference = %+v (sans zone %d)", ref.Zones, ref.HorsZone())
	}
	if ref.Zones[0].Nom != "Sortie" {
		t.Errorf("nom = %q, attendu Sortie", ref.Zones[0].Nom)
	}
	somme := ref.HorsZone()
	for _, z := range ref.Zones {
		somme += z.Reapparitions
	}
	if somme != ref.Reapparitions {
		t.Errorf("la somme des zones et du hors-zone (%d) doit valoir les reapparitions (%d)", somme, ref.Reapparitions)
	}
}

// Les planchers : sous trois matchs distincts, ou sous vingt reapparitions, les zones ne sont pas servies et
// la raison dit lequel manque. Les effectifs, eux, sont servis.
func TestLireReapparitionsAdverses_Planchers(t *testing.T) {
	zones := []ZoneNommee{zoneChoisie(), zone("Sortie", 2, carre(20, 0, 30, 10), -1, 3)}
	repeter := func(matchs, parMatch int) []ObservationAdverse {
		var obs []ObservationAdverse
		for m := 0; m < matchs; m++ {
			for i := 0; i < parMatch; i++ {
				obs = append(obs, obsEn(fmt.Sprintf("m%d", m), 25, 5, 0, 2))
			}
		}
		return obs
	}
	cas := []struct {
		nom               string
		matchs, parMatch  int
		suffisante        bool
		raison            string
		reapparitionsAtt  int
		matchsAttendus    int
		zonesServiesAtten int
	}{
		{"deux matchs, beaucoup de reapparitions", 2, 30, false, domain.TacticalRaisonSousPlancherMatchs, 60, 2, 0},
		{"trois matchs, vingt reapparitions moins une", 3, 6, false, domain.TacticalRaisonSousPlancherReapparitions, 18, 3, 0},
		{"pile aux planchers", 3, 7, true, "", 21, 3, 1},
		{"aucune observation", 0, 0, false, domain.TacticalRaisonSousPlancherMatchs, 0, 0, 0},
	}
	for _, c := range cas {
		ref, tenue := LireReapparitionsAdverses(i18n.Default, repeter(c.matchs, c.parMatch), zoneChoisie(), zones)
		for nom, r := range map[string]domain.TacticalRepartitionZones{"reference": ref, "tenue": tenue} {
			if r.Suffisante != c.suffisante || r.Raison != c.raison || r.Reapparitions != c.reapparitionsAtt ||
				r.Matchs != c.matchsAttendus || len(r.Zones) != c.zonesServiesAtten {
				t.Errorf("%s / %s : %+v", c.nom, nom, r)
			}
			if r.Zones == nil {
				t.Errorf("%s / %s : Zones doit etre une liste vide, pas nil (contrat JSON)", c.nom, nom)
			}
		}
	}
}

// Une reapparition que la zone tenue ne retient pas reste dans la reference : les deux
// repartitions n'ont pas le meme effectif.
func TestLireReapparitionsAdverses_LaReferenceNEstPasFiltree(t *testing.T) {
	zones := []ZoneNommee{zoneChoisie(), zone("Sortie", 2, carre(20, 0, 30, 10), -1, 3)}
	var obs []ObservationAdverse
	for _, m := range []string{"a", "b", "c"} {
		for i := 0; i < 8; i++ {
			obs = append(obs, obsEn(m, 25, 5, 0, i%2*2)) // une fois sur deux, deux equipiers dans la zone
		}
	}
	ref, tenue := LireReapparitionsAdverses(i18n.Default, obs, zoneChoisie(), zones)
	if ref.Reapparitions != 24 || tenue.Reapparitions != 12 {
		t.Fatalf("reference %d, tenue %d : attendu 24 et 12", ref.Reapparitions, tenue.Reapparitions)
	}
}
