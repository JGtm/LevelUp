package filmdec

// r11_grappin_research_test.go — LE SECOND TEMOIN POSITIF, PRE-INSCRIT : LE GRAPPIN.
//
// POURQUOI IL EST ELIMINATOIRE. La regle du chantier veut qu'un canal qui ne retrouve pas
// DEUX usages connus ne puisse pas prononcer de negatif sur un troisieme. Le premier temoin
// est le propulseur (verite terrain de l'utilisateur sur `1cd3848a` : 5 usages, 5 baisses de
// charge, 40/40 baisses appariees a une impulsion i57/i59 sur tout le film). Le second est le
// GRAPPIN, dont les instants viennent d'un canal TOTALEMENT INDEPENDANT : `grappleLines[]` de
// l'artefact de rejeu, produit par l'ancre i59 tag==3 — ni i56, ni i48, ni l'impulsion.
//
// CE QUE LA MESURE COMPARE, ET SON TEMOIN DE HASARD. Appariement a +/- 1,5 s entre les
// baisses de charge i56 et les accroches de grappin ; puis LA MEME mesure avec les accroches
// DECALEES DE +5 s, qui donne le niveau de coincidence fortuite. Sans ce decalage, deux
// signaux frequents se croisent toujours un peu.
//
// GARDES : `R9_FILMS`, `R9_ARTIFACTS`, `R8_BOUNDS`, `R11_IDS` (obligatoire). Aucune ecriture,
// aucune DuckDB, `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R9_FILMS=<repo>/data/cache/film_chunks \
//	  R9_ARTIFACTS=<repo>/data/cache/replays/halo_infinite \
//	  R8_BOUNDS=<wt>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R11_IDS=53ce4390,f2966f08 go test ./internal/analysis/filmdec/ \
//	  -run '^TestR11Grappin$' -count=1 -timeout 60m -v

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// r11GrappleLine est UNE accroche de grappin telle que l'artefact la publie (frames).
type r11GrappleLine struct {
	Slot uint32 `json:"slot"`
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
}

// r11DecalUS : le decalage du temoin de hasard. 5 s — le meme que les lots R8 et R9.
const r11DecalUS = 5_000_000

// r11LoadGrapples lit `grappleLines[]` de l'artefact. Ce canal est independant d'i56 : il
// vient de l'ancre i59 tag==3, deja en production.
func r11LoadGrapples(t *testing.T, id string) []r11GrappleLine {
	t.Helper()
	dir := os.Getenv(r9ArtifactsEnv)
	if dir == "" {
		t.Skipf("%s absent : pas de temoin grappin", r9ArtifactsEnv)
	}
	raw, err := os.ReadFile(filepath.Join(dir, id+".json")) //nolint:gosec // chemin sous garde
	if err != nil {
		t.Fatalf("artefact %s illisible : %v", id, err)
	}
	var a struct {
		Lines []r11GrappleLine `json:"grappleLines"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("artefact %s hors schema : %v", id, err)
	}
	return a.Lines
}

func TestR11Grappin(t *testing.T) {
	for _, dir := range r11FilmDirs(t) {
		r11GrappinOneFilm(t, dir)
	}
}

func r11GrappinOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r11Prepare(t, dir)
	lines := r11LoadGrapples(t, s.id)
	if len(lines) == 0 {
		t.Logf("%s : aucune accroche de grappin dans l'artefact — ce film ne peut pas servir "+
			"de temoin", s.id)
		return
	}
	rd := r11Collect(s.scan)
	uses := r11Uses(s, rd, r11Segments(s, rd.Ranks))
	t.Logf("%s : %d accroches de grappin (artefact) contre %d baisses de charge i56 "+
		"(tous equipements)", s.id, len(lines), len(uses))
	r11MatchGrapple(t, s, lines, uses, 0)
	r11MatchGrapple(t, s, lines, uses, r11DecalUS)
}

// r11MatchGrapple apparie accroches et baisses a +/- r11MatchUS, avec un decalage `shift`
// applique aux accroches (0 = la mesure, 5 s = le temoin de hasard).
func r11MatchGrapple(t *testing.T, s r11Setup, lines []r11GrappleLine, uses []r11Use, shift uint64) {
	t.Helper()
	hit := 0
	for _, l := range lines {
		at := r11FrameUS(s, l.T0) + shift
		for _, u := range uses {
			if u.Slot != l.Slot {
				continue
			}
			d := int64(u.TSUS) - int64(at)
			if d < 0 {
				d = -d
			}
			if d <= r11MatchUS {
				hit++
				break
			}
		}
	}
	back := 0
	for _, u := range uses {
		for _, l := range lines {
			if u.Slot != l.Slot {
				continue
			}
			at := r11FrameUS(s, l.T0) + shift
			d := int64(u.TSUS) - int64(at)
			if d < 0 {
				d = -d
			}
			if d <= r11MatchUS {
				back++
				break
			}
		}
	}
	label := "MESURE"
	if shift != 0 {
		label = "TEMOIN decale de 5 s"
	}
	t.Logf("  %-22s rappel %d/%d accroches appariees a une baisse ; %d/%d baisses appariees "+
		"a une accroche", label, hit, len(lines), back, len(uses))
}
