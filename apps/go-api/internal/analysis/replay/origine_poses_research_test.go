package replay

// origine_poses_research_test.go — L'ORIGINE D'UNE POSE : DOTATION AU SPAWN ou OBJET DEPLOYE ?
//
// LA QUESTION, ET POURQUOI ELLE N'EST PAS COSMETIQUE. Le lot de nommage
// (PLAN_NOMMAGE_EQIP_TRANSLOCATEUR) a decouvert que `equipmentPlacements` mele deux classes
// d'objet : deux grenades IDENTIQUES a 2 cm l'une de l'autre au MEME instant plus une
// capacite, meme poseur — c'est la DOTATION du joueur, creee a sa position quand il la
// recoit, pas un objet pose sur la carte. Dessiner un arc de mur a ces positions dessine un
// mur ou personne n'en a deploye.
//
// CE QUE CET INSTRUMENT MESURE, et ce qu'il ne suppose pas. Pour chaque pose, l'ecart entre
// l'instant de creation de l'OBJET et le DEBUT DE LA VIE de son poseur, plus la distance
// entre la pose et la position du poseur a ce debut de vie. Un objet de dotation nait avec
// son porteur : ecart nul, distance nulle. Un objet deploye nait en cours de vie. Aucun
// seuil n'est applique ici — la DISTRIBUTION est publiee, et c'est elle qui dit si le
// critere du plan (2 frames / 1,5 m) separe deux modes ou coupe au milieu d'un continuum.
//
// LE POSEUR N'EST PAS RECALCULE : c'est `equipmentOwner` — LA fonction de production, celle
// qui remplit `Owner` dans l'artefact. Un second calcul du meme fait diverge (regle du
// depot) ; ici il rendrait en plus une mesure qui ne dit rien de ce qui est publie.
//
// LA VIE D'UN SLOT EST DECOUPEE COMME AILLEURS DANS CE PAQUET : trou de plus de `lifeGapUS`
// (5 s) = nouvelle vie. Le seuil n'est pas invente pour l'occasion, c'est celui de lives.go.
//
// LECTURE SEULE. Aucune base, aucun artefact ecrit. UN SEUL decodage filmdec par process
// (LockProcessDecode, comme BuildFromFilm).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ORIGINE_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  ORIGINE_MAP=Cliffhanger \
//	  go test ./internal/analysis/replay/ -run '^TestOriginePosesDistribution$' -timeout 60m -v

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/mappings"
)

const (
	origineFilmEnv   = "ORIGINE_FILM"
	origineMapEnv    = "ORIGINE_MAP"
	origineBoundsEnv = "ORIGINE_BOUNDS"
)

// originePose est une pose mesuree : ce que la distribution consomme.
//
// LES DEUX BOUTS DE LA VIE SONT MESURES, ET C'EST LE POINT. Le plan ne prevoyait que le
// DEBUT (dotation au spawn). La premiere mesure a rendu 0 pose sur 295 a moins de 2 frames
// du debut de vie, et une mediane de 12 a 31 SECONDES apres lui : l'hypothese du spawn ne
// survit pas. On mesure donc aussi l'ecart a la FIN de la vie — parce qu'en Halo un joueur
// qui MEURT laisse tomber ses grenades et son equipement, ce qui produit exactement la
// signature observee (deux grenades identiques a 2 cm plus une capacite, meme instant,
// meme poseur). Publier les deux bouts est ce qui permet de trancher au lieu de supposer.
type originePose struct {
	id     uint32
	family string
	owner  int
	// dtDebutUS / dtFinUS : t0_objet - t0_vie et t0_objet - t1_vie (signes).
	dtDebutUS, dtFinUS int64
	// distDebut / distFin : distance de la pose a la position du poseur au DEBUT puis a la
	// FIN de sa vie, en metres.
	distDebut, distFin float64
	// vieUS est la duree de la vie du poseur — le denominateur sans lequel « 20 s apres le
	// debut » ne se juge pas.
	vieUS   int64
	hasLife bool
}

// origineVie est une vie de slot : le decoupage de lives.go applique au nuage brut.
type origineVie struct {
	from, to uint64
	x, y, z  float32 // position du PREMIER echantillon de la vie
	// lx, ly, lz : position du DERNIER echantillon — la ou la vie s'arrete (la mort).
	lx, ly, lz float32
}

func TestOriginePosesDistribution(t *testing.T) {
	dir := os.Getenv(origineFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", origineFilmEnv)
	}
	entry := origineMapEntry(t)
	release := filmdec.LockProcessDecode()
	defer release()
	defer installWorldObjectPrecision(entry, dir)()

	worldRange := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &worldRange
	scan.CaptureDirs = true
	positions, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("nuage des bipedes indisponible : %v", err)
	}
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].TimestampUS < positions[j].TimestampUS
	})
	raw, st, err := filmdec.ScanFilmEquipmentPlacements(dir, &worldRange)
	if err != nil {
		t.Fatalf("balayage des poses impossible : %v", err)
	}
	if !st.Calibration.Widths.Valid() {
		t.Skipf("film non calibre (%s) : aucune pose publiable, rien a mesurer", st.Calibration)
	}
	familles := origineFamilles(t)
	vies := origineVies(positions)
	t.Logf("FILM %s · carte %q · %s · %d positions · %d slots · %d vies · %d poses",
		filepath.Base(dir), entry.Module, st.Calibration, len(positions),
		len(vies), origineNbVies(vies), len(raw))

	poses := origineMesure(raw, positions, familles, vies)
	origineDistribution(t, filepath.Base(dir), poses)
}

// origineMapEntry charge l'entree de catalogue de la carte. Sans elle, pas de METRES — et le
// seuil du plan est en metres : on echoue plutot que de mesurer dans une unite muette.
func origineMapEntry(t *testing.T) filmdec.MapQuantEntry {
	t.Helper()
	nom := os.Getenv(origineMapEnv)
	if nom == "" {
		t.Fatalf("%s absent : la carte porte les bornes de dequantification, sans elle la"+
			" distance n'est pas en metres", origineMapEnv)
	}
	chemin := os.Getenv(origineBoundsEnv)
	if chemin == "" {
		chemin = filepath.Join(repoRootForTest(t), "data", "titles", "halo_infinite",
			"reference", "map_quant_bounds.json")
	}
	cat, err := filmdec.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible (%s) : %v", chemin, err)
	}
	e, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue : %v", nom, err)
	}
	return e
}

// origineFamilles lit le manifeste du titre — la MEME table que la production.
func origineFamilles(t *testing.T) map[uint32]string {
	t.Helper()
	chemin := filepath.Join(repoRootForTest(t), "config", "titles", "halo_infinite",
		"mappings", "replay_labels.toml")
	labels, err := mappings.LoadReplayLabelsFromFile(chemin)
	if err != nil {
		t.Fatalf("replay_labels.toml : %v", err)
	}
	return labels.EquipmentObjects()
}

// origineVies decoupe le nuage en vies par slot : trou de plus de lifeGapUS = nouvelle vie.
// `positions` doit etre trie par instant.
func origineVies(positions []filmdec.BipedPosition) map[uint32][]origineVie {
	out := map[uint32][]origineVie{}
	for _, p := range positions {
		if !p.HasWorld {
			continue // sans bornes, ce n'est pas une position
		}
		v := out[p.Slot]
		if n := len(v); n > 0 && p.TimestampUS-v[n-1].to <= lifeGapUS {
			v[n-1].to = p.TimestampUS
			v[n-1].lx, v[n-1].ly, v[n-1].lz = p.X, p.Y, p.Z
			out[p.Slot] = v
			continue
		}
		out[p.Slot] = append(v, origineVie{
			from: p.TimestampUS, to: p.TimestampUS,
			x: p.X, y: p.Y, z: p.Z, lx: p.X, ly: p.Y, lz: p.Z,
		})
	}
	return out
}

func origineNbVies(vies map[uint32][]origineVie) int {
	n := 0
	for _, v := range vies {
		n += len(v)
	}
	return n
}

// origineMesure croise chaque pose avec la vie de son poseur.
func origineMesure(
	raw []filmdec.EquipmentPlacement, positions []filmdec.BipedPosition,
	familles map[uint32]string, vies map[uint32][]origineVie,
) []originePose {
	out := make([]originePose, 0, len(raw))
	for _, p := range raw {
		fam := familles[p.GlobalID]
		if fam == "" {
			fam = equipmentFamilyOther
		}
		op := originePose{id: p.GlobalID, family: fam, owner: -1}
		if slot, _, ok := equipmentOwner(positions, p); ok {
			op.owner = int(slot)
			if v, hit := origineVieDe(vies[slot], p.T0US); hit {
				op.hasLife = true
				op.dtDebutUS = int64(p.T0US) - int64(v.from)
				op.dtFinUS = int64(p.T0US) - int64(v.to)
				op.vieUS = int64(v.to) - int64(v.from)
				op.distDebut = origineDist(p, v.x, v.y, v.z)
				op.distFin = origineDist(p, v.lx, v.ly, v.lz)
			}
		}
		out = append(out, op)
	}
	return out
}

// origineVieDe rend la vie qui CONTIENT l'instant, ou a defaut la plus proche en temps : une
// pose dont le poseur a ete mesure appartient a une vie, et l'ecarter silencieusement
// biaiserait la distribution vers les cas faciles.
func origineVieDe(vs []origineVie, at uint64) (origineVie, bool) {
	if len(vs) == 0 {
		return origineVie{}, false
	}
	best, bestGap := origineVie{}, uint64(math.MaxUint64)
	for _, v := range vs {
		if at >= v.from && at <= v.to {
			return v, true
		}
		gap := v.from - at
		if at > v.to {
			gap = at - v.to
		}
		if gap < bestGap {
			best, bestGap = v, gap
		}
	}
	return best, true
}

func origineDist(p filmdec.EquipmentPlacement, x, y, z float32) float64 {
	dx, dy, dz := float64(p.X-x), float64(p.Y-y), float64(p.Z-z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// origineDistribution publie la distribution par famille, avec ses denominateurs, et une
// ligne par pose (prefixe POSE) pour l'agregation inter-films.
func origineDistribution(t *testing.T, film string, poses []originePose) {
	t.Helper()
	parFamille := map[string][]originePose{}
	for _, p := range poses {
		parFamille[p.family] = append(parFamille[p.family], p)
		t.Logf("POSE\t%s\t0x%08x\t%s\t%d\t%d\t%d\t%.3f\t%.3f\t%d\t%t",
			film, p.id, p.family, p.owner, p.dtDebutUS, p.dtFinUS,
			p.distDebut, p.distFin, p.vieUS, p.hasLife)
	}
	fams := make([]string, 0, len(parFamille))
	for f := range parFamille {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	t.Logf("=== ORIGINE · film %s — DEBUT de vie (spawn) contre FIN de vie (mort) ===", film)
	t.Logf("%-22s %5s %5s | %9s %9s %8s | %9s %9s %8s | %8s", "famille", "n", "vie",
		"dbt<=2f", "dbt p50", "d p50", "fin<=2f", "fin p50", "d p50", "vie p50")
	for _, f := range fams {
		t.Log(origineLigne(f, parFamille[f]))
	}
}

// origineLigne resume une famille aux DEUX bouts de la vie du poseur, avec ses denominateurs.
func origineLigne(fam string, ps []originePose) string {
	var dDebut, dFin, xDebut, xFin, vies []float64
	courtsDebut, courtsFin, sansVie := 0, 0, 0
	for _, p := range ps {
		if !p.hasLife {
			sansVie++
			continue
		}
		dDebut = append(dDebut, math.Abs(float64(p.dtDebutUS))/1000)
		dFin = append(dFin, math.Abs(float64(p.dtFinUS))/1000)
		xDebut = append(xDebut, p.distDebut)
		xFin = append(xFin, p.distFin)
		vies = append(vies, float64(p.vieUS)/1e6)
		if math.Abs(float64(p.dtDebutUS)) <= float64(2*origineFrameUS) {
			courtsDebut++
		}
		if math.Abs(float64(p.dtFinUS)) <= float64(2*origineFrameUS) {
			courtsFin++
		}
	}
	n, avecVie := len(ps), len(ps)-sansVie
	return fmt.Sprintf("%-22s %5d %5d | %9s %9s %8s | %9s %9s %8s | %8s",
		fam, n, avecVie,
		originePart(courtsDebut, avecVie), origineQ(dDebut, 0.5, "ms"), origineQ(xDebut, 0.5, "m"),
		originePart(courtsFin, avecVie), origineQ(dFin, 0.5, "ms"), origineQ(xFin, 0.5, "m"),
		origineQ(vies, 0.5, "s"))
}

// origineFrameUS est le pas de la grille du document, en microsecondes — l'unite dans
// laquelle le plan exprime son seuil (« 2 frames »).
const origineFrameUS = int64(DefaultFrameIntervalMS) * 1000

func origineQ(v []float64, q float64, unite string) string {
	if len(v) == 0 {
		return "-"
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(q * float64(len(s)-1))
	return fmt.Sprintf("%.2f%s", s[i], unite)
}

func originePart(k, n int) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d=%.0f%%", k, n, 100*float64(k)/float64(n))
}
