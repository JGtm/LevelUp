package filmdec

// r8_artefact_research_test.go — SOCLE COMMUN des instruments du lot R8 (usage du
// repulseur et du propulseur). Lecture SEULE des artefacts de rejeu deja cuits
// (`data/cache/replays/halo_infinite/*.json`) : aucun film n'est redecode ici, aucune
// DuckDB n'est ouverte, rien n'est ecrit.
//
// POURQUOI L'ARTEFACT ET PAS LE FILM pour la piste A : les poses `deployed`, les pistes de
// bipedes et le canal i48 y sont DEJA decodes et sur le MEME axe de temps (frames de
// `frameIntervalMs`). Mesurer la physique sur l'artefact ne suppose rien de neuf sur le
// format ; si la piste A rendait un positif, il faudrait le rejouer sur le film.
//
// Gardes : `R8_ARTIFACTS` (dossier des artefacts) — absent, tous les tests R8 sont sautes.
// `R8_IDS` (liste d'identifiants a 8 hex, separes par des virgules) borne le balayage.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_ARTIFACTS=<repo>/data/cache/replays/halo_infinite \
//	  go test ./internal/analysis/filmdec/ -run '^TestR8' -count=1 -timeout 30m -v

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	r8ArtifactsEnv = "R8_ARTIFACTS"
	r8IDsEnv       = "R8_IDS"
)

// r8Point est un point publie d'une piste (axe `t` en frames du document).
type r8Point struct {
	T int     `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// r8Track est une VIE publiee (le slot migre aux reapparitions).
type r8Track struct {
	Slot     uint32    `json:"slot"`
	Team     int       `json:"team"`
	XUID     string    `json:"xuid"`
	EndFrame int       `json:"endFrame"`
	Points   []r8Point `json:"points"`
}

// r8Placement est une pose d'equipement (cf. replay.EquipmentPlacement).
type r8Placement struct {
	T0     int     `json:"t0"`
	T1     int     `json:"t1"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
	Family string  `json:"family"`
	ID     string  `json:"id"`
	Owner  int     `json:"owner"`
	Origin string  `json:"origin"`
}

// r8Ability est une lecture du canal i48 (rang de capacite portee).
type r8Ability struct {
	T    int    `json:"t"`
	Slot uint32 `json:"slot"`
	R    int    `json:"r"`
	Src  string `json:"src"`
}

// r8Change est une transition d'equipement publiee (schema >= 21) : `taken` / `spent`.
type r8Change struct {
	T    int    `json:"t"`
	Slot uint32 `json:"slot"`
	Kind string `json:"kind"`
	R    int    `json:"r"`
	From int    `json:"from"`
}

// r8Grapple est une ligne de grappin publiee : un instant d'usage CERTAIN d'un
// equipement de mobilite, lu d'un canal propre. C'est le temoin POSITIF du lot.
type r8Grapple struct {
	Slot uint32  `json:"slot"`
	T0   int     `json:"t0"`
	T1   int     `json:"t1"`
	AX   float64 `json:"ax"`
	AY   float64 `json:"ay"`
	AZ   float64 `json:"az"`
}

// r8Label est un libelle de rang de capacite tel que l'artefact le publie.
type r8Label struct {
	EN string `json:"en"`
	FR string `json:"fr"`
}

// r8Artifact est la vue MINIMALE dont R8 a besoin. Volontairement partielle : les schemas
// du parc vont de 20 a 34 et un champ absent doit rester un zero, jamais une erreur.
type r8Artifact struct {
	ID              string
	MatchID         string             `json:"matchId"`
	SchemaVersion   int                `json:"schemaVersion"`
	FrameCount      int                `json:"frameCount"`
	FrameIntervalMs int                `json:"frameIntervalMs"`
	Tracks          []r8Track          `json:"tracks"`
	Placements      []r8Placement      `json:"equipmentPlacements"`
	Abilities       []r8Ability        `json:"abilities"`
	Changes         []r8Change         `json:"equipmentChanges"`
	Grapples        []r8Grapple        `json:"grappleLines"`
	AbilityLabels   map[string]r8Label `json:"abilityLabels"`
}

// r8LoadCorpus charge les artefacts du dossier `R8_ARTIFACTS`, bornes par `R8_IDS`.
// Saute le test si la garde est absente.
func r8LoadCorpus(t *testing.T) []*r8Artifact {
	t.Helper()
	dir := os.Getenv(r8ArtifactsEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", r8ArtifactsEnv)
	}
	only := map[string]bool{}
	for _, s := range strings.Split(os.Getenv(r8IDsEnv), ",") {
		if s = strings.TrimSpace(s); s != "" {
			only[s] = true
		}
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("glob %s : %v", dir, err)
	}
	out := make([]*r8Artifact, 0, len(paths))
	for _, p := range paths {
		id := strings.TrimSuffix(filepath.Base(p), ".json")
		if len(only) > 0 && !only[id] {
			continue
		}
		a := r8ReadArtifact(t, p, id)
		if a != nil {
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		t.Fatalf("aucun artefact lisible dans %s", dir)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// r8ReadArtifact lit UN artefact. Rend nil (sans echouer) sur un fichier hors schema :
// le parc porte des copies datees (`*.ancien-*`) qu'un balayage ne doit pas faire tomber.
func r8ReadArtifact(t *testing.T, path, id string) *r8Artifact {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par la garde d'environnement
	if err != nil {
		t.Logf("artefact illisible %s : %v", path, err)
		return nil
	}
	var a r8Artifact
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Logf("artefact hors schema %s : %v", path, err)
		return nil
	}
	if a.FrameIntervalMs <= 0 || len(a.Tracks) == 0 {
		return nil
	}
	a.ID = id
	return &a
}

// r8Dist2 est la distance HORIZONTALE (x, y) en metres — z est l'altitude dans le
// document (cf. EquipmentPlacement.Z). Deux poses d'un meme socle se comparent au sol.
func r8Dist2(ax, ay, bx, by float64) float64 {
	dx, dy := ax-bx, ay-by
	return math.Hypot(dx, dy)
}

// r8Speeds rend la vitesse HORIZONTALE instantanee d'une piste, indexee par frame de
// depart : v[f] = distance(P(f), P(f+1)) / pas. Les trous de la piste (frames non
// consecutives) sont EXCLUS — une teleportation d'echantillonnage n'est pas une vitesse.
func r8Speeds(tr r8Track, stepMs int) map[int]float64 {
	out := make(map[int]float64, len(tr.Points))
	step := float64(stepMs) / 1000.0
	for i := 1; i < len(tr.Points); i++ {
		a, b := tr.Points[i-1], tr.Points[i]
		if b.T != a.T+1 {
			continue
		}
		out[a.T] = r8Dist2(a.X, a.Y, b.X, b.Y) / step
	}
	return out
}

// r8Median rend la mediane d'un echantillon (copie triee, l'entree n'est pas modifiee).
func r8Median(v []float64) float64 {
	return r8Quantile(v, 0.5)
}

// r8Quantile rend le quantile `q` (0..1) d'un echantillon, par interpolation basse.
func r8Quantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(q * float64(len(s)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(s) {
		i = len(s) - 1
	}
	return s[i]
}

// r8PeakSpeed rend la vitesse MAXIMALE de `sp` sur la fenetre [t0+lo, t0+hi] (bornes
// incluses) et le nombre de frames effectivement lues — sans quoi un maximum sur une
// fenetre vide se lirait comme un zero mesure.
func r8PeakSpeed(sp map[int]float64, t0, lo, hi int) (peak float64, seen int) {
	for f := t0 + lo; f <= t0+hi; f++ {
		v, ok := sp[f]
		if !ok {
			continue
		}
		seen++
		if v > peak {
			peak = v
		}
	}
	return peak, seen
}

// r8TrackAt rend l'index de la piste dont le slot vaut `slot` ET dont la fenetre couvre
// `frame`. Le slot MIGRE aux reapparitions : chercher par slot seul attraperait une autre
// vie du meme joueur.
func r8TrackAt(tracks []r8Track, slot uint32, frame int) int {
	for i := range tracks {
		if tracks[i].Slot != slot || len(tracks[i].Points) == 0 {
			continue
		}
		if frame >= tracks[i].Points[0].T && frame <= tracks[i].EndFrame {
			return i
		}
	}
	return -1
}

// r8RankOfFamily rend le rang de capacite que l'artefact nomme comme la famille demandee
// (`thruster` -> « Thruster », `repulsor` -> « Repulsor »), et -1 quand la palette du film
// ne le nomme pas. AUCUNE constante de rang en dur : la palette varie d'un film a l'autre
// (famille A : 5 / 6 ; famille B : 21 / non nomme).
func r8RankOfFamily(a *r8Artifact, family string) int {
	want := map[string]string{"thruster": "thruster", "repulsor": "repulsor"}[family]
	if want == "" {
		return -1
	}
	for k, lab := range a.AbilityLabels {
		if !strings.EqualFold(lab.EN, want) {
			continue
		}
		n := 0
		for _, c := range k {
			if c < '0' || c > '9' {
				return -1
			}
			n = n*10 + int(c-'0')
		}
		return n
	}
	return -1
}
