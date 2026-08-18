package replay

// powerup_socle_research_test.go — LE POWER-UP DE SOCLE AU CENTRE DE CATALYST.
//
// CE QUE CET INSTRUMENT CHERCHE. L'utilisateur decrit (2026-08-18) un power-up pose sur un
// socle au centre de Catalyst, sur un pont — camouflage ou surbouclier selon le sous-mode.
// La chaine de production ne le voit NULLE PART : `equipmentPlacements` des artefacts cuits
// de Catalyst ne porte qu'un seul `powerup_overshield`, `dropped`, avec porteur.
//
// POURQUOI CE NEGATIF NE PROUVE RIEN, et c'est le point de depart du lot. `ScanFilmEquipment
// Placements` ne retient un record de creation `ti=37` que si sa position retombe sur le
// PREMIER POINT d'une vie decodee des paquets delta (`confirmPlacements` ->
// `MatchEquipmentLife`). Or un objet POSE cesse d'emettre sa position. Un objet de socle qui
// ne bouge JAMAIS n'a donc aucune vie delta, donc aucun record confirmable : il est invisible
// a cette chaine PAR CONSTRUCTION. La chaine `ti=42` (armes au sol), elle, retient les
// creations SANS vie delta et filtre par IDENTITE — c'est l'asymetrie que ce lot exploite.
//
// LECTURE SEULE, aucune base, aucune ecriture. Plan : `.ai/V7.5/replay2d/
// PLAN_POWERUP_SOCLE_CATALYST.md` (hypotheses et seuils ECRITS AVANT la mesure).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 OBJ_FILM_ART=<depot>/data/cache/replays/halo_infinite \
//	  go test ./internal/analysis/replay/ -run '^TestPowerupSocleAncrage$' -v
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache/film_chunks/01e1f945 OBJ_FILM_MAP=Catalyst \
//	  go test ./internal/analysis/replay/ -run '^TestPowerupSocle' -timeout 60m -v

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// LA GARDE DU FILM EST CELLE DE TOUT LE PAQUET : `OBJ_FILM` porte la RACINE du cache film
// (le repertoire qui contient `film_chunks/`), et `objRequireRoot` la lit. Ce fichier ne la
// redeclare pas — une seconde definition de la meme garde divergerait au premier correctif.

// psArtEnv porte le repertoire des artefacts cuits. Defaut : le chemin du depot.
const psArtEnv = "OBJ_FILM_ART"

// psFilmsCatalyst : les quatre films Catalyst du lot, avec leur sous-mode. Les artefacts
// cuits n'existent que pour les trois premiers (`75f1188f` n'a jamais ete cuit).
var psFilmsCatalyst = []struct {
	ID, Mode string
	Cuit     bool
}{
	{"64e8adfa", "CTF", true},
	{"530820e5", "CTF", true},
	{"01e1f945", "KOTH", true},
	{"75f1188f", "KOTH", false},
}

// psPoint est un point du plan, en coordonnees monde (metres).
type psPoint struct{ X, Y float32 }

// psDist rend la distance XY entre deux points, en metres.
func psDist(a, b psPoint) float64 {
	dx, dy := float64(a.X-b.X), float64(a.Y-b.Y)
	return math.Hypot(dx, dy)
}

// psArtDir rend le repertoire des artefacts cuits, ou saute l'etape.
func psArtDir(t *testing.T) string {
	t.Helper()
	if v := os.Getenv(psArtEnv); v != "" {
		return v
	}
	dir := filepath.Join(repoRootForTest(t), "data", "cache", "replays", "halo_infinite")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("%s absent et %s introuvable : instrument saute", psArtEnv, dir)
	}
	return dir
}

// psLoadDoc lit UN artefact cuit. Rend ok=false quand il n'existe pas : l'absence d'un
// artefact est un fait du corpus, pas une panne de l'instrument.
func psLoadDoc(t *testing.T, dir, id string) (ReplayDocument, bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return ReplayDocument{}, false
	}
	var doc ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("artefact %s illisible : %v", id, err)
	}
	return doc, true
}

// psCentre est le centre de la carte, tel que les socles publies le donnent, avec les pieces
// qui l'etablissent.
type psCentre struct {
	// C est le centre retenu.
	C psPoint
	// Paires est le nombre de paires miroir (x proches, y opposes) qui fondent l'axe.
	Paires int
	// YAxe est la moyenne des milieux des paires miroir — l'axe de symetrie en y.
	YAxe float64
	// SurAxe est le nombre de socles portes par l'axe (|y - YAxe| <= 1 m) ; XMin / XMax leur
	// etendue, dont le milieu donne le x du centre.
	SurAxe     int
	XMin, XMax float64
}

// psCentreDesSocles calcule le centre de la carte a partir des socles d'arme PUBLIES.
//
// LA REGLE, ecrite avant la mesure (plan, section 3) : l'axe de symetrie en y est la moyenne
// des milieux des paires MIROIR (deux socles de meme x a moins de 0,5 m, de y opposes) ; le x
// du centre est le milieu de l'etendue des socles PORTES PAR CET AXE. Aucune des deux moities
// n'est devinee : chacune sort d'une symetrie observee.
func psCentreDesSocles(pads []psPoint) psCentre {
	var ct psCentre
	var somme float64
	for i := 0; i < len(pads); i++ {
		for j := i + 1; j < len(pads); j++ {
			a, b := pads[i], pads[j]
			if math.Abs(float64(a.X-b.X)) > 0.5 || math.Abs(float64(a.Y+b.Y)) > 0.5 {
				continue
			}
			if math.Abs(float64(a.Y)) < 1 { // deux socles de l'axe ne font pas une paire
				continue
			}
			ct.Paires++
			somme += float64(a.Y+b.Y) / 2
		}
	}
	if ct.Paires > 0 {
		ct.YAxe = somme / float64(ct.Paires)
	}
	ct.XMin, ct.XMax = math.Inf(1), math.Inf(-1)
	for _, p := range pads {
		if math.Abs(float64(p.Y)-ct.YAxe) > 1 {
			continue
		}
		ct.SurAxe++
		ct.XMin = math.Min(ct.XMin, float64(p.X))
		ct.XMax = math.Max(ct.XMax, float64(p.X))
	}
	if ct.SurAxe > 0 {
		ct.C = psPoint{X: float32((ct.XMin + ct.XMax) / 2), Y: float32(ct.YAxe)}
	}
	return ct
}

// psSoclesUniques rassemble les socles des artefacts fournis, dedupliques a 0,5 m pres : le
// socle appartient a la CARTE, l'arme qui y apparait appartient au match (acquis du lot des
// armes au sol) — deux films de la meme carte publient donc les memes positions.
func psSoclesUniques(docs []ReplayDocument) []psPoint {
	var out []psPoint
	for _, d := range docs {
		for _, p := range d.WeaponPads {
			q := psPoint{X: p.X, Y: p.Y}
			vu := false
			for _, r := range out {
				if psDist(q, r) <= 0.5 {
					vu = true
					break
				}
			}
			if !vu {
				out = append(out, q)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		return out[i].Y < out[j].Y
	})
	return out
}

// TestPowerupSocleAncrage — PHASE 0 du plan : le centre de Catalyst, et l'etat du corpus.
//
// Ne decode AUCUN film : il ne lit que les artefacts cuits. C'est ce qui en fait l'ancrage —
// la cible spatiale des phases suivantes se pose avant qu'un seul bit de film soit lu.
func TestPowerupSocleAncrage(t *testing.T) {
	dir := psArtDir(t)
	var docs []ReplayDocument
	t.Log("=== 0.1 CORPUS ===")
	for _, f := range psFilmsCatalyst {
		doc, ok := psLoadDoc(t, dir, f.ID)
		if !ok {
			t.Logf("  %s (%s) : artefact ABSENT", f.ID, f.Mode)
			continue
		}
		docs = append(docs, doc)
		t.Logf("  %s (%s) : %d images, %d socles d'arme, %d poses, %d episodes"+
			" | surbouclier %d vies / %d episodes, camo %d vies",
			f.ID, f.Mode, doc.FrameCount, len(doc.WeaponPads), len(doc.EquipmentPlacements),
			len(doc.EquipmentEpisodes), doc.Coverage.Equipment.OvershieldLives,
			doc.Coverage.Equipment.OvershieldEpisodes, doc.Coverage.Equipment.CamoLives)
	}
	if len(docs) == 0 {
		t.Skipf("aucun artefact Catalyst dans %s : instrument saute", dir)
	}

	pads := psSoclesUniques(docs)
	ct := psCentreDesSocles(pads)
	t.Log("=== 0.1 CENTRE DE CATALYST (socles publies) ===")
	for _, p := range pads {
		t.Logf("  socle (%.3f ; %.3f)", p.X, p.Y)
	}
	t.Logf("  %d socles uniques | %d paires miroir -> axe y = %.4f", len(pads), ct.Paires, ct.YAxe)
	t.Logf("  %d socles sur l'axe, x de %.3f a %.3f -> milieu %.4f",
		ct.SurAxe, ct.XMin, ct.XMax, float64(ct.C.X))
	t.Logf("  CENTRE RETENU = (%.3f ; %.3f)", ct.C.X, ct.C.Y)

	t.Log("=== 0.1 CONTROLE : milieu des bornes des positions JOUEES ===")
	for i, d := range docs {
		mx := (d.Bounds.MinX + d.Bounds.MaxX) / 2
		my := (d.Bounds.MinY + d.Bounds.MaxY) / 2
		t.Logf("  film %d : milieu (%.3f ; %.3f), ecart au centre retenu %.3f m",
			i, mx, my, psDist(psPoint{X: mx, Y: my}, ct.C))
	}
	if ct.Paires < 2 || ct.SurAxe < 2 {
		t.Fatalf("centre non etabli : %d paires miroir, %d socles sur l'axe", ct.Paires, ct.SurAxe)
	}
}

// psMapEnv porte le nom de carte ; psBoundsEnv le catalogue de bornes. Les deux ont un
// defaut : les quatre films du lot sont tous Catalyst, et le catalogue vit dans le depot.
const (
	psMapEnv    = "OBJ_FILM_MAP"
	psBoundsEnv = "OBJ_FILM_BOUNDS"
	psCarte     = "Catalyst"
)

// psEntreeCarte rend les bornes de dequantification de la carte du lot. Elle passe par le
// MEME helper que l'instrument des socles d'arme (`mapQuantEntryFromEnv`) : une seconde
// lecture du catalogue divergerait au premier correctif.
func psEntreeCarte(t *testing.T) filmdec.MapQuantEntry {
	t.Helper()
	if os.Getenv(psMapEnv) == "" {
		t.Setenv(psMapEnv, psCarte)
	}
	return mapQuantEntryFromEnv(t, psMapEnv, psBoundsEnv)
}
