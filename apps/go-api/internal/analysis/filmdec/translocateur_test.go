package filmdec

// translocateur_test.go — LE TRANSLOCATEUR QUANTIQUE : sa balise, et son RETOUR.
//
// CE QUE LA STRUCTURE DU JEU A DONNÉ (2026-08-18, PLAN_NOMMAGE_EQIP_TRANSLOCATEUR gate 0) :
// la BALISE est un objet d'équipement `ti=37` d'identifiant `eqip` 0x730dc70f — le `sofa`
// 0x8f1be870 qui le référence porte l'identifiant de chaîne 0x1f7c6a15, dont le murmur3 rend
// `quantum_translocator`, au rang 11 de la palette de la famille A (le rang que la
// RECETTE_LOADOUT §13 nommait déjà translocateur par un autre chemin). La pose se publie donc
// déjà, par sa famille. Ce qui manque, c'est le RETOUR.
//
// LE RETOUR N'EST PAS UN ÉVÉNEMENT DU FILM, C'EST UNE DISCONTINUITÉ DE POSITION. Aucun canal
// connu ne l'annonce ; ce qu'on peut mesurer, c'est un SAUT : le porteur disparaît d'un point
// et réapparaît à un autre en une image, sans mourir.
//
// LES SEUILS SONT ÉCRITS AVANT LA MESURE (plan, décision n°3) : saut > 4 m en 100 ms — aucun
// déplacement à pied, en sprint ni au grappin ne le fait — et arrivée à moins de 2 m d'une
// balise VIVANTE. Le témoin est obligatoire : les mêmes sauts confrontés aux poses d'un AUTRE
// équipement vivant au même instant.
//
// LE PIÈGE QUI REND CETTE MESURE POSSIBLE — ET QUI EXPLIQUE POURQUOI PERSONNE NE L'AVAIT VUE.
// `DefaultScanFilmOptions` porte `MaxSpeedMPS = 100` : une position dont la vitesse depuis la
// précédente dépasse 100 m/s est REJETÉE comme faux positif du balayage bit à bit. Un retour de
// translocateur est précisément une téléportation — 20 à 40 m en une image, soit 200 à 400 m/s.
// LE DÉCODEUR DE PRODUCTION LE JETTE. L'instrument met donc `MaxSpeedMPS = 0`, et il paie le
// prix : sans ce filtre, les aberrations du balayage reviennent. C'est exactement ce que le
// témoin sert à mesurer.
//
// LECTURE SEULE, gardé par TRANSLOC_FILM. UN SEUL décodage filmdec par process.
//
// USAGE (depuis apps/go-api) — les bornes de la carte sont OBLIGATOIRES, les seuils sont en
// mètres et un quantum n'est pas une distance :
//
//	CGO_ENABLED=0 TRANSLOC_FILM=<repo>/data/cache/film_chunks/06dfe6d9 \
//	  TRANSLOC_BOUNDS=-50.16,-51.7,163.84,50.52,51.67,186.82 \
//	  go test ./internal/analysis/filmdec/ -run '^TestTranslocateur$' -timeout 90m -v

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	translocFilmEnv   = "TRANSLOC_FILM"
	translocBoundsEnv = "TRANSLOC_BOUNDS"
	translocBeaconEnv = "TRANSLOC_BEACON"
	// translocChunksEnv borne le balayage a une liste de chunks. NECESSAIRE sur les gros films :
	// avec `MaxSpeedMPS = 0` le balayage bit a bit ne rejette plus rien, le nuage explose et le
	// processus est tue par l'OS avant d'avoir rien publie (mesure du 2026-08-18 : `06dfe6d9` et
	// `83ee3f9f` meurent sans sortie). Restreindre aux chunks qui portent la balise est la seule
	// facon de mesurer sans reintroduire le filtre qu'on veut precisement desactiver.
	translocChunksEnv = "TRANSLOC_CHUNKS"
)

// translocBeaconDefault : l'identifiant `eqip` de la balise, établi par la chaîne
// `sofd -> sofa -> eqip` (cf. l'en-tête). Surchargeable pour rejouer la mesure sur un autre
// objet — c'est ce qui permet au TÉMOIN d'être la même mesure sur autre chose.
const translocBeaconDefault = 0x730dc70f

// Les trois seuils du plan, écrits avant la mesure.
const (
	// translocJumpWindowUS : la fenêtre d'une « image » de réplication. Les positions arrivent
	// sur une grille de 100 ms ; 150 ms tolère un échantillon manqué sans franchir une frontière
	// de vie (les trous entre deux vies d'un même slot se comptent en secondes).
	translocJumpWindowUS = 150_000
	// translocJumpMinM : au-delà, ce n'est plus un déplacement. 4 m en 100 ms = 40 m/s, quand le
	// Spartan le plus rapide (grappin, véhicule) reste sous 35 m/s.
	translocJumpMinM = 4.0
	// translocArriveM : distance à la balise en deçà de laquelle l'arrivée est « sur » la balise.
	translocArriveM = 2.0
)

func TestTranslocateur(t *testing.T) {
	dir := os.Getenv(translocFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", translocFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	wr, err := translocBounds()
	if err != nil {
		t.Fatalf("%s : %v", translocBoundsEnv, err)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)

	pl, st, err := ScanFilmEquipmentPlacements(dir, &wr)
	if err != nil {
		t.Fatalf("balayage des poses impossible : %v", err)
	}
	beacon := translocBeaconID(t)
	t.Logf("FILM %s · calibration %s · %d poses", dir, st.Calibration, st.Placements)
	for id, n := range st.ByID {
		marque := ""
		if id == beacon {
			marque = "  <== BALISE"
		}
		t.Logf("    eqip 0x%08x : %d poses%s", id, n, marque)
	}
	balises, autres := translocSplit(pl, beacon)
	t.Logf("== BALISES : %d · AUTRES POSES (témoin) : %d ==", len(balises), len(autres))

	sauts := translocJumps(t, dir, lay, wr)
	translocRapport(t, sauts, balises, autres)
}

// translocBounds lit les bornes de la carte. Elles sont OBLIGATOIRES : sans elles les seuils
// en mètres n'ont pas de sens, et un instrument qui les devinerait rendrait des chiffres faux
// plutôt qu'une absence.
func translocBounds() (Vec3Range, error) {
	var wr Vec3Range
	raw := strings.TrimSpace(os.Getenv(translocBoundsEnv))
	if raw == "" {
		return wr, fmt.Errorf("bornes absentes (attendu minX,minY,minZ,maxX,maxY,maxZ en mètres —" +
			" le champ `bounds` de l'artefact du rejeu les porte)")
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 6 {
		return wr, fmt.Errorf("6 nombres attendus, %d reçus", len(parts))
	}
	var v [6]float32
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return wr, fmt.Errorf("nombre %d illisible (%q) : %w", i, p, err)
		}
		v[i] = float32(f)
	}
	for a := 0; a < 3; a++ {
		wr[a].Min, wr[a].Max = v[a], v[a+3]
		if wr[a].Max <= wr[a].Min {
			return wr, fmt.Errorf("axe %d : borne haute (%g) sous la borne basse (%g)", a, wr[a].Max, wr[a].Min)
		}
	}
	return wr, nil
}

func translocBeaconID(t *testing.T) uint32 {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(translocBeaconEnv))
	if raw == "" {
		return translocBeaconDefault
	}
	v, err := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 32)
	if err != nil {
		t.Fatalf("%s=%q illisible (GlobalID hexadécimal attendu) : %v", translocBeaconEnv, raw, err)
	}
	return uint32(v)
}

func translocSplit(pl []EquipmentPlacement, beacon uint32) (balises, autres []EquipmentPlacement) {
	for _, p := range pl {
		if p.GlobalID == beacon {
			balises = append(balises, p)
			continue
		}
		autres = append(autres, p)
	}
	return balises, autres
}

// translocSaut est une discontinuité de position dans UNE vie de bipède.
type translocSaut struct {
	slot     uint32
	t0, t1   uint64
	from, to [3]float32
	dist     float64
}

// translocJumps rend les sauts de position, filtre de vitesse DÉSACTIVÉ.
//
// `MaxSpeedMPS = 0` est le cœur de la mesure : le défaut (100 m/s) rejette précisément les
// téléportations qu'on cherche. `IsolationGapMS = 0` de même — une arrivée isolée dans le temps
// est exactement ce qu'un retour produirait.
func translocJumps(t *testing.T, dir string, lay I0Layout, wr Vec3Range) []translocSaut {
	t.Helper()
	opt := DefaultScanFilmOptions()
	opt.MaxSpeedMPS = 0
	opt.IsolationGapMS = 0
	opt.WorldRange = &wr
	opt.Layout = &lay
	opt.Chunks = translocChunks(t)
	raw, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("nuage des bipèdes indisponible : %v", err)
	}
	// Indices plutot que copies : sur un gros film le nuage non filtre pese plus que le film.
	bySlot := map[uint32][]int{}
	for i, p := range raw {
		if p.HasWorld {
			bySlot[p.Slot] = append(bySlot[p.Slot], i)
		}
	}
	var out []translocSaut
	pas := 0
	for slot, idx := range bySlot {
		sort.Slice(idx, func(i, j int) bool {
			return raw[idx[i]].TimestampUS < raw[idx[j]].TimestampUS
		})
		for i := 1; i < len(idx); i++ {
			p0, p1 := raw[idx[i-1]], raw[idx[i]]
			dt := p1.TimestampUS - p0.TimestampUS
			if dt == 0 || dt > translocJumpWindowUS {
				continue
			}
			pas++
			a := [3]float32{p0.X, p0.Y, p0.Z}
			b := [3]float32{p1.X, p1.Y, p1.Z}
			if d := translocDist(a, b); d > translocJumpMinM {
				out = append(out, translocSaut{
					slot: slot, t0: p0.TimestampUS, t1: p1.TimestampUS, from: a, to: b, dist: d,
				})
			}
		}
	}
	t.Logf("== %d positions · %d vies · %d transitions contemporaines (<= %d ms) · %d SAUTS > %.0f m ==",
		len(raw), len(bySlot), pas, translocJumpWindowUS/1000, len(out), translocJumpMinM)
	sort.Slice(out, func(i, j int) bool { return out[i].dist > out[j].dist })
	return out
}

// translocChunks lit la liste de chunks a balayer, ou nil pour tous.
func translocChunks(t *testing.T) []int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(translocChunksEnv))
	if raw == "" {
		return nil
	}
	var out []int
	for _, p := range strings.Split(raw, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			t.Fatalf("%s=%q : %q n'est pas un numero de chunk", translocChunksEnv, raw, p)
		}
		out = append(out, n)
	}
	return out
}

func translocDist(a, b [3]float32) float64 {
	var s float64
	for i := 0; i < 3; i++ {
		d := float64(a[i]) - float64(b[i])
		s += d * d
	}
	return math.Sqrt(s)
}

// translocRapport confronte les sauts aux balises, et le même test aux AUTRES poses.
//
// Le témoin n'est pas décoratif : les poses d'équipement sont nombreuses et réparties sur la
// carte, donc « une arrivée près d'une pose » arrive par hasard. C'est l'ÉCART entre les deux
// taux qui dirait quelque chose, jamais le taux seul.
func translocRapport(t *testing.T, sauts []translocSaut, balises, autres []EquipmentPlacement) {
	t.Helper()
	if len(sauts) == 0 {
		t.Log("AUCUN saut : rien à confronter")
		return
	}
	surBalise := translocProches(sauts, balises)
	surAutre := translocProches(sauts, autres)
	t.Logf("== MESURE : %d sauts sur %d arrivent à moins de %.0f m d'une BALISE vivante (%.1f %%) ==",
		len(surBalise), len(sauts), translocArriveM, 100*float64(len(surBalise))/float64(len(sauts)))
	t.Logf("== TÉMOIN : %d sauts sur %d arrivent à moins de %.0f m d'une AUTRE pose vivante (%.1f %%) ==",
		len(surAutre), len(sauts), translocArriveM, 100*float64(len(surAutre))/float64(len(sauts)))
	for i, s := range sauts {
		if i == 12 {
			t.Logf("    ... %d sauts de plus", len(sauts)-i)
			break
		}
		t.Logf("    slot %5d  %.1f m en %d ms  (%.1f,%.1f,%.1f) -> (%.1f,%.1f,%.1f)",
			s.slot, s.dist, (s.t1-s.t0)/1000,
			s.from[0], s.from[1], s.from[2], s.to[0], s.to[1], s.to[2])
	}
	for i, b := range balises {
		var proches int
		for _, s := range sauts {
			if s.t1 >= b.T0US && s.t1 <= b.T1US &&
				translocDist(s.to, [3]float32{b.X, b.Y, b.Z}) <= translocArriveM {
				proches++
			}
		}
		t.Logf("    BALISE %d : slot %d · vie %d -> %d ms · (%.1f,%.1f,%.1f) · %d saut(s) y arrivent",
			i, b.Life.Slot, b.T0US/1000, b.T1US/1000, b.X, b.Y, b.Z, proches)
	}
}

// translocProches rend les sauts dont l'ARRIVÉE tombe à moins de translocArriveM d'une pose
// VIVANTE à cet instant.
func translocProches(sauts []translocSaut, poses []EquipmentPlacement) []translocSaut {
	var out []translocSaut
	for _, s := range sauts {
		for _, p := range poses {
			if s.t1 < p.T0US || s.t1 > p.T1US {
				continue
			}
			if translocDist(s.to, [3]float32{p.X, p.Y, p.Z}) <= translocArriveM {
				out = append(out, s)
				break
			}
		}
	}
	return out
}
