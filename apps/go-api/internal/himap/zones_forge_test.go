package himap

// Temoins du lecteur de zones de callout Forge. Ils tiennent sur des objets construits a la
// main : le decodage du fichier est deja couvert par `mapvar`, ce qui se joue ici est la
// GEOMETRIE (orientation) et le TRI (rateliers).

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// zoneTest fabrique un objet « zone nommee » : boite de cotes pleins a, b, orientee par f.
func zoneTest(sid uint32, x, y float64, a, b float64, f mapvar.Vec3) mapvar.Object {
	return mapvar.Object{
		TypeID:     TypeIDZoneNommee,
		LocationID: sid,
		Pos:        mapvar.Vec3{X: x, Y: y},
		Forward:    f,
		ShapeRaw: &mapvar.ShapeRaw{
			Family: 3, // boite
			S5:     int64(a * 65536),
			S6:     int64(b * 65536),
			S7:     int64(2 * 65536),
			S8:     int64(1 * 65536),
		},
	}
}

// TestZoneOrienteeSuitSonVecteurAvant — une zone tournee de 90 degres echange ses cotes. Sans
// orientation, le masque declarerait « dedans » les coins qui sont dehors : la chaine des
// objectifs a paye 31 pour cent de faux positifs sur une zone tournee de 20 degres.
func TestZoneOrienteeSuitSonVecteurAvant(t *testing.T) {
	droite := ZonesNommeesForge([]mapvar.Object{
		zoneTest(1, 0, 0, 10, 4, mapvar.Vec3{X: 1}),
	})
	tournee := ZonesNommeesForge([]mapvar.Object{
		zoneTest(1, 0, 0, 10, 4, mapvar.Vec3{Y: 1}),
	})
	if len(droite) != 1 || len(tournee) != 1 {
		t.Fatalf("zones lues : %d et %d, attendu 1 et 1", len(droite), len(tournee))
	}
	if g := gabarit(droite[0].Contour); math.Abs(g[0]-10) > 0.01 || math.Abs(g[1]-4) > 0.01 {
		t.Fatalf("zone alignee : gabarit %v, attendu 10x4", g)
	}
	if g := gabarit(tournee[0].Contour); math.Abs(g[0]-4) > 0.01 || math.Abs(g[1]-10) > 0.01 {
		t.Fatalf("zone tournee : gabarit %v, attendu 4x10 (les cotes echanges)", g)
	}
}

// TestZoneSansAvantRetombeSurLesAxes — un vecteur avant nul ne doit pas produire un polygone
// degenere : quelques zones du dump n'en portent pas.
func TestZoneSansAvantRetombeSurLesAxes(t *testing.T) {
	zs := ZonesNommeesForge([]mapvar.Object{zoneTest(1, 5, 5, 6, 2, mapvar.Vec3{})})
	if len(zs) != 1 {
		t.Fatalf("zones lues : %d, attendu 1", len(zs))
	}
	if g := gabarit(zs[0].Contour); math.Abs(g[0]-6) > 0.01 || math.Abs(g[1]-2) > 0.01 {
		t.Fatalf("gabarit %v, attendu 6x2 sur les axes du monde", g)
	}
}

// TestRatelierEcarte — la palette d'objets non poses ne doit JAMAIS servir de masque. Mesure du
// 2026-08-27 : `illusion` aligne 57 boites identiques a abscisse constante, `forbidden` 33 sur
// une grille. Rogner la carte sur une droite l'effacerait entierement.
func TestRatelierEcarte(t *testing.T) {
	var palette []mapvar.Object
	for i := 0; i < 20; i++ {
		palette = append(palette, zoneTest(uint32(100+i), -13.66, float64(i)*0.6, 0.5, 0.5, mapvar.Vec3{X: 1}))
	}
	if zs := ZonesNommeesForge(palette); zs != nil {
		t.Fatalf("ratelier de %d boites identiques accepte comme zones (%d rendues)", len(palette), len(zs))
	}
	// Une carte, elle, donne a chaque lieu la taille de son lieu : elle passe.
	var carte []mapvar.Object
	for i := 0; i < 20; i++ {
		carte = append(carte, zoneTest(uint32(200+i), float64(i)*7, float64(i%3)*9, 4+float64(i), 3+float64(i%5), mapvar.Vec3{X: 1}))
	}
	if zs := ZonesNommeesForge(carte); len(zs) != 20 {
		t.Fatalf("carte a zones variees : %d zones retenues, attendu 20", len(zs))
	}
}

// TestZonesIgnoreLesAutresObjets — seuls les objets du bon type ET porteurs d'un StringId
// comptent. Un objet Forge ordinaire n'est pas une zone, et un objet du bon type sans StringId
// non plus.
func TestZonesIgnoreLesAutresObjets(t *testing.T) {
	ordinaire := zoneTest(7, 0, 0, 5, 5, mapvar.Vec3{X: 1})
	ordinaire.TypeID = 42
	muet := zoneTest(0, 1, 1, 5, 5, mapvar.Vec3{X: 1})
	sansForme := zoneTest(9, 2, 2, 5, 5, mapvar.Vec3{X: 1})
	sansForme.ShapeRaw = nil
	zs := ZonesNommeesForge([]mapvar.Object{ordinaire, muet, sansForme, zoneTest(3, 9, 9, 5, 5, mapvar.Vec3{X: 1})})
	if len(zs) != 1 || zs[0].StringID != 3 {
		t.Fatalf("zones retenues : %+v, attendu la seule zone 3", zs)
	}
}
