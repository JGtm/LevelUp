package replay

// instruments_partages_test.go — symboles des instruments de recherche (socle de power-up
// `ps*`, bombe, socles d'armes, chronologie) dont se servent les gardes du build par defaut
// (`powerup_socle_oracle_test.go`, `powerup_socle_temoin_test.go`, gates bombe et Assaut).
// Deplaces tels quels au J12.7 bis depuis les fichiers tagues `research` (decision DU-5 : le
// tag cache les instruments, jamais une garde) ; chaque declaration garde le corps et le
// commentaire de son fichier d'origine.

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// amArmeSentinelle arme le plafond memoire de MESURE pour un balayage de corpus, et rend la
// fonction de desarmement (a differer par l'appelant).
//
// POURQUOI CHAQUE INSTRUMENT DE CE FICHIER L'APPELLE (leçon du 2026-08-31). Ces balayages
// enchainent jusqu'a 65 films DANS UN SEUL PROCESSUS. Le decodage du statborg est borne par
// `statMaxRecordsPerFilm` et les pics mesures restent sous le dixieme de gibioctet — mais c'est
// une PROPRIETE OBSERVEE, pas une garantie, et la doctrine du depot ne fait pas d'exception :
// tout processus qui enchaine des films arme sa sentinelle (cf. `internal/filmproc`).
func amArmeSentinelle(t *testing.T, nom string) func() {
	t.Helper()
	g := filmproc.Arm(nom, filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — balayage interrompu pour proteger la machine",
			float64(peak)/(1<<30))
	})
	return func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio (plafond souple %d Gio)",
			float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}
}

// b3MecheMS rend la mèche mesurée du film (Husky Raid a la sienne).
func b3MecheMS(id string) int {
	if id == "1c01e34f" {
		return 5100
	}
	return b2MecheMS
}

func lastAgeS(ts []uint64, at uint64) float64 {
	var last uint64
	var got bool
	for _, t := range ts {
		if t <= at {
			last, got = t, true
		}
	}
	if !got {
		return 0
	}
	return float64(at-last) / 1e6
}

func medianOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}

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

// psCentreDesSocles calcule le centre de la carte a partir des socles d'arme PUBLIES.
//
// LA REGLE, ecrite avant la mesure (plan, section 3) : l'axe de symetrie en y est la moyenne
// des milieux des paires MIROIR (deux socles de meme x a moins de 0,5 m, de y opposes) ; le x
// du centre est le milieu de l'etendue des socles PORTES PAR CET AXE. Aucune des deux moities
// n'est devinee : chacune sort d'une symetrie observee.
func psCentreDesSocles(pads []psPoint) psCentre {
	var ct psCentre
	var somme float64
	for i := range pads {
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

// chronoEpisodes : les periodes zoomees de Nilton410, en SECONDES d'horloge du feed.
var chronoEpisodes = [][2]float64{
	{41, 46.3}, {49, 52}, {61, 61.8}, {68, 68.8}, {71, 73}, {85, 86},
}

// psArtEnv porte le repertoire des artefacts cuits. Defaut : le chemin du depot.
const psArtEnv = "OBJ_FILM_ART"

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
