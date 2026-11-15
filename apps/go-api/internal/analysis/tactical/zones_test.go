package tactical

// zones_test.go — le nom en jeu d'une zone (zones.go) : les quatre temps de la règle, la marge
// de tranche, les égalités, z inconnu, la forme (contour, parties, trous). Géométrie synthétique ;
// les témoins sur le catalogue réel vivent côté service (projection du catalogue).

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"levelup/go-api/internal/i18n"
)

func carre(x0, y0, x1, y1 float64) [][2]float64 {
	return [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

func zone(nom string, vol int, contour [][2]float64, zb, zh float64) ZoneNommee {
	return ZoneNommee{Nom: i18n.FromColumns(nom, nom), Polygone: contour, ZBas: zb, ZHaut: zh, VolumeIndex: vol}
}

func attendreZone(t *testing.T, x, y float64, zs []float64, zones []ZoneNommee, nom, regle string) {
	t.Helper()
	got, ok := NommerZone(x, y, zs, zones)
	if nom == "" {
		if ok {
			t.Fatalf("(%v, %v) z=%v : attendu aucun nom, obtenu %q (%s)", x, y, zs, got.Zone.Nom[i18n.LocaleFR], got.Regle)
		}
		return
	}
	if !ok {
		t.Fatalf("(%v, %v) z=%v : attendu %q, obtenu aucun nom", x, y, zs, nom)
	}
	if got.Zone.Nom[i18n.LocaleFR] != nom || got.Regle != regle {
		t.Fatalf("(%v, %v) z=%v : obtenu %q par %q, attendu %q par %q", x, y, zs, got.Zone.Nom[i18n.LocaleFR], got.Regle, nom, regle)
	}
}

// (a) un seul polygone contient le centre avec une tranche compatible.
func TestNommerZone_UnPolygoneUneTranche(t *testing.T) {
	zones := []ZoneNommee{
		zone("Bas", 1, carre(0, 0, 10, 10), 0, 3),
		zone("Haut", 2, carre(0, 0, 10, 10), 2, 8),
	}
	attendreZone(t, 5, 5, []float64{1, 1, 1}, zones, "Bas", RegleZonePolygone)
	attendreZone(t, 5, 5, []float64{5, 6, 7}, zones, "Haut", RegleZonePolygone)
}

// La marge de tranche vaut 0,25 m, bornes comprises.
func TestNommerZone_MargeDeTranche(t *testing.T) {
	zones := []ZoneNommee{zone("Bas", 1, carre(0, 0, 10, 10), 0, 3)}
	attendreZone(t, 5, 5, []float64{3.24}, zones, "Bas", RegleZonePolygone)
	// Hors de la tranche : la règle (c), au second passage (polygone contenant, distance 0).
	attendreZone(t, 5, 5, []float64{3.26}, zones, "Bas", RegleZoneProche)
	attendreZone(t, 5, 5, []float64{-0.24}, zones, "Bas", RegleZonePolygone)
}

// (b) plusieurs candidats : la tranche la plus étroite PARMI celles qui contiennent la majorité.
func TestNommerZone_EmpileesMajoritePuisPlusEtroite(t *testing.T) {
	zones := []ZoneNommee{
		zone("Etroite", 1, carre(0, 0, 10, 10), 0, 3),
		zone("Large", 2, carre(0, 0, 10, 10), 2, 8),
	}
	// Les deux contiennent les trois événements : la plus étroite.
	attendreZone(t, 5, 5, []float64{2.5, 2.6, 2.7}, zones, "Etroite", RegleZoneEmpilee)

	// La plus étroite contient le z médian mais PAS la majorité : la large l'emporte.
	zones = []ZoneNommee{
		zone("Etroite", 1, carre(0, 0, 10, 10), 0, 3),
		zone("Profonde", 2, carre(0, 0, 10, 10), -10, 2),
	}
	attendreZone(t, 5, 5, []float64{-5, -4, 1, 9, 9}, zones, "Profonde", RegleZoneEmpilee)

	// Exactement la moitié n'est pas la majorité : la large (4 sur 4) l'emporte sur l'étroite
	// (2 sur 4), qui contient pourtant le z médian (1,5).
	zones = []ZoneNommee{
		zone("Etroite", 1, carre(0, 0, 10, 10), 0, 3),
		zone("Large", 2, carre(0, 0, 10, 10), -10, 10),
	}
	attendreZone(t, 5, 5, []float64{-5, 1, 2, 9}, zones, "Large", RegleZoneEmpilee)
}

// (b) sans majorité pour aucun candidat : la plus peuplée, puis la plus étroite, puis l'index de
// volume.
func TestNommerZone_EmpileesSansMajorite(t *testing.T) {
	zones := []ZoneNommee{
		zone("Trois", 1, carre(0, 0, 10, 10), 0, 3),
		zone("Fine", 2, carre(0, 0, 10, 10), -0.2, 1.2),
	}
	attendreZone(t, 5, 5, []float64{-5, 1, 9}, zones, "Fine", RegleZoneEmpilee)
	// Comptes INÉGAUX, aucune majorité (3 et 2 sur 7) : la plus peuplée l'emporte, même plus large.
	zones = []ZoneNommee{
		zone("Etroite", 1, carre(0, 0, 10, 10), 1, 1.2),
		zone("Peuplee", 2, carre(0, 0, 10, 10), 0, 3.5),
	}
	attendreZone(t, 5, 5, []float64{-9, -8, 1, 1.1, 3, 9, 9.5}, zones, "Peuplee", RegleZoneEmpilee)
	zones = []ZoneNommee{
		zone("Second", 7, carre(0, 0, 10, 10), 0, 3),
		zone("Premier", 3, carre(0, 0, 10, 10), 0, 3),
	}
	attendreZone(t, 5, 5, []float64{1}, zones, "Premier", RegleZoneEmpilee)
}

// (c) le polygone le plus proche à MOINS de 2 m, tranches compatibles d'abord.
func TestNommerZone_ProcheTranchesCompatiblesDAbord(t *testing.T) {
	zones := []ZoneNommee{
		zone("Compatible", 1, carre(0, 0, 10, 10), 0, 3),
		zone("PlusPres", 2, carre(0, 0, 10.5, 10), 2, 8),
	}
	// PlusPres est à 1 m mais sa tranche exclut z = 1 ; Compatible est à 1,5 m.
	attendreZone(t, 11.5, 5, []float64{1}, zones, "Compatible", RegleZoneProche)
	// Aucune tranche compatible : la plus proche de toutes.
	attendreZone(t, 11.5, 5, []float64{20}, zones, "PlusPres", RegleZoneProche)
}

// (c) la borne de 2 m est stricte.
func TestNommerZone_DeuxMetresStricts(t *testing.T) {
	zones := []ZoneNommee{zone("Salle", 1, carre(0, 0, 10, 10), 0, 3)}
	attendreZone(t, 11.99, 5, []float64{1}, zones, "Salle", RegleZoneProche)
	attendreZone(t, 12, 5, []float64{1}, zones, "", "")
}

// (d) rien à moins de 2 m ; une zone sans forme ne nomme rien.
func TestNommerZone_SansNom(t *testing.T) {
	zones := []ZoneNommee{
		zone("Loin", 1, carre(0, 0, 10, 10), 0, 3),
		{Nom: i18n.FromColumns("SansForme", "SansForme"), X: 50, Y: 50, ZBas: 0, ZHaut: 3, VolumeIndex: 2},
	}
	attendreZone(t, 50, 50, []float64{1}, zones, "", "")
	attendreZone(t, 50, 50, nil, zones, "", "")
}

// z inconnu (lecture d'artefact, ou z tous absents) : un seul polygone contenant le centre le
// nomme ; plusieurs polygones empilés ne nomment rien ; aucun : le plus proche à moins de 2 m.
func TestNommerZone_ZInconnu(t *testing.T) {
	zones := []ZoneNommee{
		zone("Bas", 1, carre(0, 0, 10, 10), 0, 3),
		zone("Haut", 2, carre(0, 0, 10, 10), 2, 8),
		zone("Seule", 3, carre(20, 0, 30, 10), 0, 3),
	}
	attendreZone(t, 5, 5, nil, zones, "", "")
	attendreZone(t, 25, 5, nil, zones, "Seule", RegleZonePolygone)
	attendreZone(t, 31, 5, nil, zones, "Seule", RegleZoneProche)
	attendreZone(t, 15, 5, nil, zones, "", "")
}

// La forme : un trou n'appartient pas à la zone (il en est le bord), une partie lui appartient.
func TestNommerZone_TrousEtParties(t *testing.T) {
	z := zone("Cour", 1, carre(40, 0, 50, 10), 0, 3)
	z.Trous = [][][2]float64{carre(44, 4, 46, 6)}
	z.Parties = [][][2]float64{carre(60, 0, 62, 2)}
	zones := []ZoneNommee{z}
	// Au centre du trou : hors de la zone, à 1 m de son bord → règle (c).
	attendreZone(t, 45, 5, []float64{1}, zones, "Cour", RegleZoneProche)
	// Dans la partie détachée : dans la zone → règle (a).
	attendreZone(t, 61, 1, []float64{1}, zones, "Cour", RegleZonePolygone)
	// Entre le contour et le trou : dans la zone.
	attendreZone(t, 41, 1, []float64{1}, zones, "Cour", RegleZonePolygone)
}

// Le z médian d'un nombre pair d'événements est la moyenne des deux du milieu.
func TestNommerZone_MedianePaire(t *testing.T) {
	zones := []ZoneNommee{zone("Bas", 1, carre(0, 0, 10, 10), 0, 3)}
	// Médiane (3 + 3,4) / 2 = 3,2 : dans la tranche avec la marge.
	attendreZone(t, 5, 5, []float64{3, 3.4}, zones, "Bas", RegleZonePolygone)
}

// jeuDeCasNommage : le chemin du jeu partagé, relatif au paquet. Le paquet testfixtures n'est PAS importé :
// il tire le décodeur de film dans la fermeture d'internal/analysis (archlint, constat G).
const jeuDeCasNommage = "../../testfixtures/testdata/cartes/nommage_zone_cas.json"

// casNommageJSON est un cas du jeu partagé avec le port web (nommage_zone_cas.json) : les zones au
// format du contrat des callouts, le point, le z (nil = inconnu) et la zone attendue (nil = aucune).
type casNommageJSON struct {
	Nom   string `json:"nom"`
	Zones []struct {
		Name        string         `json:"name"`
		VolumeIndex int            `json:"volume_index"`
		Polygon     [][2]float64   `json:"polygon"`
		Parts       [][][2]float64 `json:"parts"`
		Holes       [][][2]float64 `json:"holes"`
		ZBottom     float64        `json:"z_bottom"`
		ZTop        float64        `json:"z_top"`
	} `json:"zones"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	Z         *float64 `json:"z"`
	Attendu   *string  `json:"attendu"`
	Regle     string   `json:"regle"`
	DistanceM float64  `json:"distance_m"`
}

// Le jeu de cas PARTAGÉ avec le port web (features/match-replay/model/killerZone.test.ts) : les deux
// côtés lisent le même fichier, c'est lui qui prouve que les deux ports nomment pareil.
func TestNommerZone_JeuDeCasPartage(t *testing.T) {
	data, err := os.ReadFile(jeuDeCasNommage)
	if err != nil {
		t.Fatalf("lecture du jeu de cas : %v", err)
	}
	var jeu struct {
		Cas []casNommageJSON `json:"cas"`
	}
	if err := json.Unmarshal(data, &jeu); err != nil {
		t.Fatalf("décodage du jeu de cas : %v", err)
	}
	if len(jeu.Cas) == 0 {
		t.Fatal("jeu de cas vide : le test ne prouverait rien")
	}
	for _, c := range jeu.Cas {
		t.Run(c.Nom, func(t *testing.T) {
			zones := make([]ZoneNommee, 0, len(c.Zones))
			for _, z := range c.Zones {
				zones = append(zones, ZoneNommee{
					Nom: i18n.FromColumns(z.Name, z.Name), Polygone: z.Polygon, Parties: z.Parts, Trous: z.Holes,
					ZBas: z.ZBottom, ZHaut: z.ZTop, VolumeIndex: z.VolumeIndex,
				})
			}
			var zs []float64
			if c.Z != nil {
				zs = []float64{*c.Z}
			}
			got, ok := NommerZone(c.X, c.Y, zs, zones)
			if c.Attendu == nil {
				if ok {
					t.Fatalf("attendu aucun nom, obtenu %q (%s)", got.Zone.Nom[i18n.LocaleFR], got.Regle)
				}
				return
			}
			if !ok {
				t.Fatalf("attendu %q, obtenu aucun nom", *c.Attendu)
			}
			if got.Zone.Nom[i18n.LocaleFR] != *c.Attendu || got.Regle != c.Regle || math.Abs(got.DistanceM-c.DistanceM) > 1e-9 {
				t.Fatalf("obtenu %q par %q à %v m, attendu %q par %q à %v m",
					got.Zone.Nom[i18n.LocaleFR], got.Regle, got.DistanceM, *c.Attendu, c.Regle, c.DistanceM)
			}
		})
	}
}
