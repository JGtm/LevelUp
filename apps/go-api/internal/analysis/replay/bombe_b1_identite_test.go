package replay

// bombe_b1_identite_test.go — L'IDENTITÉ DE LA BOMBE D'ASSAUT DANS LE CANAL DES ARMES TENUES.
//
// # LA QUESTION
//
// Dans Halo, la bombe d'Assaut est un OBJET TENU EN MAIN, comme le crâne d'Oddball et le
// drapeau de CTF. Le dépôt décode déjà la chronologie de l'arme tenue par joueur
// (`filmdec.ScanFilmHeldWeaponChanges`, composant weapon-state-type-info), et personne ne l'a
// jamais pointée sur l'Assaut. Le crâne et le drapeau ont un tag `weap` (32 bits hauts d'un
// identifiant filmshell : `0x0017592c` et `0x2a392328`, catalogue d'icônes + manifeste
// `replay_labels.toml`), mais AUCUN des deux n'est dans le catalogue d'ARMES
// (`weaponv3.KnownWeaponHigh32`, dérivé de l'enum weapon_data.go). La bombe a forcément son
// propre tag, avec la même signature : hors catalogue d'armes, tenue alternativement par des
// joueurs des deux équipes.
//
// # PROTOCOLE, écrit avant la mesure
//
// TÉMOIN (Oddball `43716616`, le crâne y est porté sans discontinuer) :
//
//	T1  la famille 0x0017592c émet dans ScanFilmHeldWeaponChanges : >= 1 transition VERS
//	    (prise) ET >= 1 transition DEPUIS (lâcher) ;
//	T2  elle est tenue par >= 2 slots distincts (le crâne change de mains) ;
//	T3  elle est HORS catalogue d'armes.
//
// Si T1 échoue, le canal des armes tenues ne réplique pas les objets d'objectif, l'angle
// tombe, et le négatif se publie tel quel — l'instrument n'est pas « cassé », il est réfuté.
//
// CANDIDATE (9 films d'Assaut, corpus a5Explosions) — la bombe est la famille qui satisfait :
//
//	C1  HORS catalogue d'armes ;
//	C2  présente sur les 9 films (chaque match d'Assaut a sa bombe) ;
//	C3  tenue par >= 3 slots distincts par film en médiane (elle change de mains) ;
//	C4  >= 1 transition VERS et >= 1 transition DEPUIS sur chaque film.
//
// Si plusieurs familles passent C1-C4, la discrimination est l'oracle des détonations
// (chronologie contre les 28 explosions datées) — test B2, pas ici.
//
// RÉGIME : garde `ASSAUT_CACHE`. Aucune base, aucun réseau, sentinelle mémoire armée, verrou
// process filmdec (un seul décodage à la fois).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run BombeB1 -v -timeout 30m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

const (
	b1Temoin = "43716616"
	b1Crane  = uint32(0x0017592c)
)

// b1Films : les 9 films d'Assaut, mêmes identifiants que l'oracle a5Explosions.
var b1Films = []string{
	"1c01e34f", "34bb3bc8", "35b75a31", "3d58eb37", "69b16f5d",
	"9f57c612", "c75f33b8", "ce083875", "df8fcbef",
}

// b1FamStat agrège ce qu'une famille d'identifiant a fait dans UN film.
type b1FamStat struct {
	vers, depuis int             // transitions VERS la famille et DEPUIS la famille
	slots        map[uint32]bool // slots de bipède l'ayant tenue
	firstUS      uint64          // première transition VERS
	lastUS       uint64          // dernière transition (VERS ou DEPUIS)
}

// b1ScanFilm balaye UN film et rend l'agrégat par famille, plus les stats du balayage.
func b1ScanFilm(t *testing.T, cache, id string) (map[uint32]*b1FamStat, filmdec.HeldWeaponChangeStats) {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	changes, stats, err := filmdec.ScanFilmHeldWeaponChanges(dir, nil)
	if err != nil {
		t.Fatalf("%s : canal des armes tenues illisible : %v", id, err)
	}
	fams := map[uint32]*b1FamStat{}
	get := func(f uint32) *b1FamStat {
		if fams[f] == nil {
			fams[f] = &b1FamStat{slots: map[uint32]bool{}}
		}
		return fams[f]
	}
	for _, ch := range changes {
		if ch.Family != filmdec.NoWeaponVariant {
			s := get(ch.Family)
			s.vers++
			s.slots[ch.Slot] = true
			if s.firstUS == 0 {
				s.firstUS = ch.TimestampUS
			}
			s.lastUS = ch.TimestampUS
		}
		if ch.Previous != filmdec.NoWeaponVariant && ch.Previous != ch.Family {
			s := get(ch.Previous)
			s.depuis++
			s.slots[ch.Slot] = true
			s.lastUS = ch.TimestampUS
		}
	}
	return fams, stats
}

// b1Ligne formate une famille pour le journal de mesure.
func b1Ligne(f uint32, s *b1FamStat) string {
	nom, connu := weaponv3.KnownWeaponHigh32[f]
	if !connu {
		nom = "HORS CATALOGUE"
	}
	return fmt.Sprintf("0x%08x  vers=%3d depuis=%3d slots=%2d  [%s]",
		f, s.vers, s.depuis, len(s.slots), nom)
}

// b1HorsCatalogue rend les familles hors catalogue d'armes, triées par nombre de slots
// porteurs décroissant puis par volume de transitions.
func b1HorsCatalogue(fams map[uint32]*b1FamStat) []uint32 {
	var out []uint32
	for f := range fams {
		if _, connu := weaponv3.KnownWeaponHigh32[f]; !connu {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := fams[out[i]], fams[out[j]]
		if len(a.slots) != len(b.slots) {
			return len(a.slots) > len(b.slots)
		}
		if a.vers+a.depuis != b.vers+b.depuis {
			return a.vers+a.depuis > b.vers+b.depuis
		}
		return out[i] < out[j]
	})
	return out
}

// TestBombeB1Temoin applique T1-T3 au film Oddball témoin.
func TestBombeB1Temoin(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandée : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestBombeB1Temoin")()
	release := filmdec.LockProcessDecode()
	defer release()

	fams, stats := b1ScanFilm(t, cache, b1Temoin)
	t.Logf("%s : records=%d masque=%d emissions=%d repeats=%d, %d familles vues",
		b1Temoin, stats.Records, stats.WithComponent, stats.Emissions, stats.Repeats, len(fams))
	for i, f := range b1HorsCatalogue(fams) {
		if i >= 12 {
			break
		}
		t.Logf("  hors catalogue : %s", b1Ligne(f, fams[f]))
	}

	crane, ok := fams[b1Crane]
	if !ok {
		t.Errorf("T1 ÉCHOUE : le crâne 0x%08x n'émet JAMAIS dans le canal des armes tenues "+
			"— le canal ne réplique pas les objets d'objectif, l'angle tombe", b1Crane)
		return
	}
	t.Logf("CRÂNE : %s", b1Ligne(b1Crane, crane))
	if crane.vers < 1 || crane.depuis < 1 {
		t.Errorf("T1 ÉCHOUE : crâne vu mais vers=%d depuis=%d (>=1 et >=1 exigés)", crane.vers, crane.depuis)
	}
	if len(crane.slots) < 2 {
		t.Errorf("T2 ÉCHOUE : crâne tenu par %d slot(s) (>=2 exigés)", len(crane.slots))
	}
	if _, connu := weaponv3.KnownWeaponHigh32[b1Crane]; connu {
		t.Errorf("T3 ÉCHOUE : le crâne est au catalogue d'armes — le filtre C1 serait faux")
	}
}

// TestBombeB1Assaut applique C1-C4 aux neuf films d'Assaut et nomme les candidates.
func TestBombeB1Assaut(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandée : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestBombeB1Assaut")()
	release := filmdec.LockProcessDecode()
	defer release()

	// parFilm[film] = agrégat ; présence[fam] = films où la famille émet (VERS et DEPUIS).
	parFilm := map[string]map[uint32]*b1FamStat{}
	for _, id := range b1Films {
		fams, stats := b1ScanFilm(t, cache, id)
		parFilm[id] = fams
		t.Logf("%s : records=%d emissions=%d repeats=%d, %d familles",
			id, stats.Records, stats.Emissions, stats.Repeats, len(fams))
		for i, f := range b1HorsCatalogue(fams) {
			if i >= 6 {
				break
			}
			t.Logf("  hors catalogue : %s", b1Ligne(f, fams[f]))
		}
	}

	presence := map[uint32]int{}
	complets := map[uint32]int{} // films où C4 tient (>=1 vers ET >=1 depuis)
	for _, fams := range parFilm {
		for f, s := range fams {
			if _, connu := weaponv3.KnownWeaponHigh32[f]; connu {
				continue // C1
			}
			presence[f]++
			if s.vers >= 1 && s.depuis >= 1 {
				complets[f]++
			}
		}
	}

	var candidates []uint32
	for f, n := range presence {
		if n == len(b1Films) && complets[f] == len(b1Films) { // C2 + C4
			candidates = append(candidates, f)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i] < candidates[j] })

	t.Logf("=== CANDIDATES C1+C2+C4 (%d) ===", len(candidates))
	for _, f := range candidates {
		var slotsParFilm []int
		for _, id := range b1Films {
			slotsParFilm = append(slotsParFilm, len(parFilm[id][f].slots))
		}
		sort.Ints(slotsParFilm)
		mediane := slotsParFilm[len(slotsParFilm)/2]
		c3 := "C3 ÉCHOUE"
		if mediane >= 3 {
			c3 = "C3 tient"
		}
		t.Logf("  0x%08x : slots/film %v, médiane %d — %s", f, slotsParFilm, mediane, c3)
	}
	if len(candidates) == 0 {
		t.Logf("AUCUNE candidate ne passe C1+C2+C4 — le canal ne porte pas la bombe sur tout le corpus")
	}
}
