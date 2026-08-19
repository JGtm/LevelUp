package replay

// powerup_socle_archetypes_test.go — PHASE 2 : QUI est au socle, archetype par archetype.
//
// CE QUE CETTE PHASE CHERCHE, ET AVEC QUELLE CIBLE. La phase 1 a MESURE le socle :
// (0,393 ; -0,012), altitude ~21,6 m, quatre ramassages a `T0 - 15` images. La question n'est
// donc plus « ou chercher » mais « quel objet du monde vit la, et de quel archetype ».
//
// LES SEPT ARCHETYPES SONT BALAYES, PAS UN SEUL. `ti=37` (equipement) est le candidat naturel,
// mais c'est exactement l'hypothese qu'il faut pouvoir REFUTER : 36 et 39 sont inconnus, 38 est
// le corps rigide, 42 l'arme au sol, 40 le vehicule. Un balayage qui ne regarde que 37 ne peut
// rendre que 37. `ti=41` (projectile) sert de TEMOIN : il traverse le centre en permanence, et
// une regle qui l'attrape n'a pas isole un socle.
//
// LE NEGATIF SE CHIFFRE. « Aucune vie a la cible » ne dit rien sans la distance a laquelle la
// plus proche est passee : `D3Min` la rend, en 3D (l'altitude compte — le milieu de Catalyst
// a deux etages, et 5 m les separent).
//
// CE QUE CETTE PHASE NE PEUT PAS VOIR, ET C'EST LE POINT DE LA PHASE 3 : un objet qui n'emet
// AUCUNE position delta. `ScanFilmWorldObjects` ne lit que les paquets delta ; un objet pose
// qui ne bouge jamais n'y figure pas. L'absence mesuree ici est donc une PREMIERE moitie de
// reponse, pas la reponse.
//
// LECTURE SEULE. Garde `OBJ_FILM` (racine du cache film) + `OBJ_FILM_ART`.
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache \
//	  go test ./internal/analysis/replay/ -run '^TestPowerupSocleArchetypes$' -timeout 60m -v

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// psTIBalayes — les archetypes d'objet du monde du balayage, temoin compris.
var psTIBalayes = []int{36, 37, 38, 39, 40, 41, 42}

// psNomTI rend l'etiquette de l'archetype, celle de `filmdec/testdata/ecs_table.tsv`.
func psNomTI(ti int) string {
	switch ti {
	case 36, 39:
		return "inconnu"
	case 37:
		return "equipement"
	case 38:
		return "corps rigide"
	case 40:
		return "vehicule"
	case 41:
		return "projectile (temoin)"
	case 42:
		return "arme au sol"
	}
	return "?"
}

// Les trois seuils de la phase, ECRITS AVANT LA MESURE (plan, section 3).
const (
	// psBoiteLarge / psBoiteSerree : rayons XY autour de la cible, en metres.
	psBoiteLarge  = 6.0
	psBoiteSerree = 3.0
	// psDepartUS : « present des le depart » = premier point dans les 5 premieres secondes.
	psDepartUS = 5_000_000
	// psImmobileM : une vie est IMMOBILE si son premier et son dernier point sont a moins
	// d'un metre — la signature d'un objet pose (meme rayon que `gwPadRadiusM`).
	psImmobileM = 1.0
)

// psVieBoite est UNE vie d'objet du monde qui passe par la boite, resumee.
type psVieBoite struct {
	TI         int
	Slot, Gen  uint32
	T0US, T1US uint64
	// P0 / P1 : premiere et derniere position ; DMin la distance XY MINIMALE a la cible ;
	// D3 la distance 3D minimale ; Etendue le deplacement total (un objet pose ne bouge pas).
	P0, P1  psPoint
	Z0, Z1  float32
	DMin    float64
	D3      float64
	Etendue float64
	Points  int
}

// psDist3 rend la distance 3D d'un point d'objet a la cible (position + altitude).
func psDist3(x, y, z float32, cible psPoint, cz float32) float64 {
	return math.Sqrt(math.Pow(float64(x-cible.X), 2) +
		math.Pow(float64(y-cible.Y), 2) + math.Pow(float64(z-cz), 2))
}

// psVieDansLaBoite resume une vie et dit si elle passe par la boite large autour de `cible`.
func psVieDansLaBoite(ti int, tr filmdec.ProjectileTrack, cible psPoint, cz float32) (psVieBoite, bool) {
	if len(tr.Pts) == 0 {
		return psVieBoite{}, false
	}
	last := tr.Pts[len(tr.Pts)-1]
	v := psVieBoite{TI: ti, Slot: tr.Slot, Gen: tr.Gen, Points: len(tr.Pts)}
	v.T0US, v.T1US = tr.Pts[0].TimestampUS, last.TimestampUS
	v.P0, v.P1 = psPoint{X: tr.Pts[0].X, Y: tr.Pts[0].Y}, psPoint{X: last.X, Y: last.Y}
	v.Z0, v.Z1 = tr.Pts[0].Z, last.Z
	v.Etendue = psDist(v.P0, v.P1)
	v.DMin, v.D3 = math.Inf(1), math.Inf(1)
	for _, p := range tr.Pts {
		if d := psDist(psPoint{X: p.X, Y: p.Y}, cible); d < v.DMin {
			v.DMin = d
		}
		if d := psDist3(p.X, p.Y, p.Z, cible, cz); d < v.D3 {
			v.D3 = d
		}
	}
	return v, v.DMin <= psBoiteLarge
}

// psStatTI compte, par archetype, ce que le balayage a rencontre — les denominateurs sans
// lesquels « N vies dans la boite » ne se juge pas.
type psStatTI struct {
	Vies, Boite, Serree, ADepart int
	// Immobiles : vies de la boite serree qui n'ont pas bouge d'un metre.
	Immobiles int
	// D3Min : la distance 3D la plus courte jamais atteinte par cet archetype. C'est ce
	// chiffre qui fait d'une absence un NEGATIF plutot qu'un silence.
	D3Min float64
	Err   error
}

// psBalayeTI balaye un archetype et rend les vies retenues, plus les compteurs.
func psBalayeTI(dir string, wr *filmdec.Vec3Range, ti int, c psCible) ([]psVieBoite, psStatTI) {
	st := psStatTI{D3Min: math.Inf(1)}
	tracks, err := filmdec.ScanFilmWorldObjects(dir, wr, ti)
	if err != nil {
		st.Err = err
		return nil, st
	}
	st.Vies = len(tracks)
	var out []psVieBoite
	for _, tr := range tracks {
		v, ok := psVieDansLaBoite(ti, tr, c.P, c.Z)
		if v.D3 < st.D3Min {
			st.D3Min = v.D3
		}
		if !ok {
			continue
		}
		st.Boite++
		if v.DMin > psBoiteSerree {
			continue
		}
		st.Serree++
		if v.Etendue <= psImmobileM {
			st.Immobiles++
		}
		if v.T0US-c.T0Film <= psDepartUS {
			st.ADepart++
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].T0US < out[j].T0US })
	return out, st
}

// psCible porte la cible de la phase 2 : le socle mesure, son altitude, et l'origine de
// l'axe des secondes du film (regle des 5 parametres).
type psCible struct {
	P      psPoint
	Z      float32
	T0Film uint64
}

// psSecondes rend un instant du film en secondes depuis le premier paquet lu.
func psSecondes(us, origine uint64) float64 { return float64(us-origine) / 1e6 }

// TestPowerupSocleArchetypes — 2.1 a 2.3 du plan, sur les quatre films Catalyst.
func TestPowerupSocleArchetypes(t *testing.T) {
	root := objRequireRoot(t)
	entry := psEntreeCarte(t)
	socle, socleZ := psSocleMesure(t)

	for _, f := range psFilmsCatalyst {
		t.Run(f.ID+"_"+f.Mode, func(t *testing.T) {
			dir := filepath.Join(root, "film_chunks", f.ID)
			if filmdec.CountFilmChunks(dir) == 0 {
				t.Skipf("aucun chunk dans %s", dir)
			}
			release := filmdec.LockProcessDecode()
			defer release()
			defer installWorldObjectPrecision(entry, dir)()
			wr := entry.Range()
			c := psCible{P: socle, Z: socleZ, T0Film: psPremierPaquetUS(dir)}

			t.Logf("=== 2.1 ARCHETYPES (cible %s z %.2f | boite %.0f/%.0f m) ===",
				psFmtPoint(socle), socleZ, psBoiteLarge, psBoiteSerree)
			for _, ti := range psTIBalayes {
				vies, st := psBalayeTI(dir, &wr, ti, c)
				if st.Err != nil {
					t.Logf("  ti=%2d %-20s : %v", ti, psNomTI(ti), st.Err)
					continue
				}
				t.Logf("  ti=%2d %-20s : %5d vies | boite %3d | serree %3d | immobiles %3d"+
					" | des le depart %2d | plus proche 3D %.2f m",
					ti, psNomTI(ti), st.Vies, st.Boite, st.Serree, st.Immobiles,
					st.ADepart, st.D3Min)
				psDetailVies(t, vies, c)
			}
		})
	}
}

// psDetailVies detaille les vies IMMOBILES de la boite serree — les seules qui puissent etre
// un objet pose. Les vies mobiles sont comptees plus haut et ne sont pas listees : ce sont
// des grenades, des projectiles et des armes qui roulent, et les lister noierait le signal.
func psDetailVies(t *testing.T, vies []psVieBoite, c psCible) {
	t.Helper()
	n := 0
	for _, v := range vies {
		if v.Etendue > psImmobileM {
			continue
		}
		n++
		if n > 20 {
			t.Logf("      (... et d'autres vies immobiles)")
			return
		}
		t.Logf("      slot %4d gen %d | %7.1f -> %7.1f s | %s z %6.2f | XY %.2f m | 3D %.2f m"+
			" | %d pts",
			v.Slot, v.Gen, psSecondes(v.T0US, c.T0Film), psSecondes(v.T1US, c.T0Film),
			psFmtPoint(v.P0), v.Z0, v.DMin, v.D3, v.Points)
	}
}

// psFmtPoint : ecriture commune d'un point, pour que deux etapes ne l'ecrivent pas autrement.
func psFmtPoint(p psPoint) string { return fmt.Sprintf("(%.3f ; %.3f)", p.X, p.Y) }

// psPremierPaquetUS rend l'horodatage du PREMIER paquet du film — l'origine de l'axe des
// secondes de cet instrument. Ce n'est PAS l'origine de la grille du document (qui compte
// depuis le premier paquet de POSITION) : les deux axes ne se comparent pas ici.
func psPremierPaquetUS(dir string) uint64 {
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			return pk.TimestampUS
		}
	}
	return 0
}
