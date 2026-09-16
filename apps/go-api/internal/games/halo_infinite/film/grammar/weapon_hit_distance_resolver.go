package grammar

// weapon_hit_distance_resolver.go — LE RESOLVEUR DE DISTANCE tireur<->victime, productionise
// (Lot 3 du plan PRECISION_ARME_DISTANCE). Il fabrique la WeaponHitDistanceFunc que
// PairWeaponHits injecte : pour un damage_aftermath apparie, il rend la distance monde entre le
// RESPONSABLE (attaquant) et le BLESSE (victime) au ts du degat.
//
// # D OU VIENT LA DISTANCE
//
// Les deux refs d en-tete d un damage_aftermath (VictimIdx, ResponsibleIdx) sont des index BRUTS
// domaine-1 ; additionnes a une BASE (bande bipede, ~512), ils designent le slot bipede dont
// offline_biped.go decode la position monde. La base juste n est pas devinee : ScanFilmWeaponDamages
// rend deja un argmax d atterrissage, mais on la CALIBRE ici par balayage (ResolveHitDistanceBase),
// en prenant la base qui resout le plus de couples — c est la meme mesure que l instrument de
// recherche (sondeBaseSweep), une seule regle.
//
// # BEST-EFFORT PAR CONSTRUCTION
//
// Sans bornes de carte (WorldRange nil), aucune coordonnee monde n existe : la fonction rend
// alors (0, false) pour toute touche — la touche reste COMPTEE par PairWeaponHits, seule sa
// distance manque (reserve #3/#5 du plan). Idem si l une des deux positions ne se resout pas au
// ts du degat dans la tolerance temporelle.
//
// LIEN AVEC L INSTRUMENT DE RECHERCHE : lot1_attrib_arme_tir_research_test (attribM3) mesurait la
// meme distance via des helpers `sonde*` (lot1_sonde_precision_helpers_test.go). Ce fichier est la
// version DE PRODUCTION de cette resolution ; les deux copies (production ici, instrument la-bas)
// restent dans la limite de CLAUDE.md (<= 2), l instrument gardant en plus ses mesures de recherche.

import (
	"fmt"
	"math"
	"sort"
)

// WeaponHitPosToleranceUS est l ecart temporel maximal entre le ts d un degat et l echantillon de
// position retenu (120 ms). MEME valeur que la sonde precision (sondePosTolUS) et que
// replay/shots.go shotPosToleranceUS : le degat est horodate a l impact, la position echantillonnee
// a une cadence propre, 120 ms cadre le decalage sans confondre deux instants distincts.
const WeaponHitPosToleranceUS uint64 = 120_000

// hitPosSample : une position monde horodatee d un slot bipede.
type hitPosSample struct {
	ts      uint64
	x, y, z float32
}

// BuildBipedTracks decode les positions monde des bipeds du film de dir et les indexe par slot,
// triees par ts. Filtres teleport/isolation DESACTIVES : on maximise la couverture (savoir combien
// de degats sont resolubles), on ne lisse pas une trajectoire. n borne le nombre de chunks balayes.
//
// L ENTREE DE CATALOGUE EST LE PARAMETRE, ET C EST LE LOT 1.9.2. Elle porte les DEUX donnees de
// profil de la carte, indissociables : les bornes monde (`Range`) et le DECOUPAGE d i0
// (`Layout` — porte, index de region attendu, largeurs d axe). Ce balayage ne passait que les
// bornes ; `ScanFilmOptions.Layout` restait nil, donc `DetectI0LayoutOf` DECIDAIT du decoupage
// sur le film. Sur une carte a plus de deux regions de compression l auto-detection ne sait pas
// voir l index de region (i0_layout.go) : sur Live Fire elle rend `gate=5 region=0 13/12/11` la
// ou le catalogue dit `gate=6 region=1 12/12/11`, et sa porte laissait passer des
// enregistrements d une AUTRE region — 26 sur 267 400 (`60ae07c4`, mesure du lot 1.9.2).
// D-3 d ADR 0034 : le decoupage d axe est une DONNEE DE PROFIL, jamais une detection.
//
// UNE ENTREE SANS DECOUPAGE EXPLOITABLE EST UNE ERREUR TYPEE (D-4), pas un repli silencieux sur
// la detection : sans largeurs d axe, aucune coordonnee monde ne peut etre produite juste.
func BuildBipedTracks(dir string, entry MapQuantEntry, n int) (map[uint32][]hitPosSample, error) {
	lay := entry.Layout()
	if !lay.Valid() {
		return nil, fmt.Errorf("%w (film %s) : l entree de catalogue ne porte pas de decoupage d axe exploitable",
			ErrUnknownMapBounds, dir)
	}
	wr := entry.Range()
	opt := DefaultScanFilmOptions()
	opt.MaxSpeedMPS = 0
	opt.IsolationGapMS = 0
	opt.WorldRange = &wr
	opt.Layout = &lay
	opt.Chunks = make([]int, 0, n)
	for c := 1; c <= n; c++ {
		opt.Chunks = append(opt.Chunks, c)
	}
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		return nil, fmt.Errorf("balayage biped (%s) : %w", dir, err)
	}
	tr := map[uint32][]hitPosSample{}
	for _, p := range pos {
		tr[p.Slot] = append(tr[p.Slot], hitPosSample{ts: p.TimestampUS, x: p.X, y: p.Y, z: p.Z})
	}
	for s := range tr {
		ss := tr[s]
		sort.Slice(ss, func(i, j int) bool { return ss[i].ts < ss[j].ts })
		tr[s] = ss
	}
	return tr, nil
}

// ResolveHitDistanceBase choisit la BASE (decalage index brut -> slot) qui resout le plus de
// couples (victime, attaquant) parmi les bases candidates de la bande bipede. Meme mesure que
// l instrument de recherche (sondeBaseSweep) : la base la plus resolvante l emporte. Rend 512 par
// defaut si aucun degat n a ses deux positions (le film est alors non resolvant, distances vides).
func ResolveHitDistanceBase(damages []WeaponDamage, tracks map[uint32][]hitPosSample) int {
	peakBase, peakRes := 512, -1
	for _, b := range lot1chBases {
		if b < 400 {
			continue // seules les bases de la bande bipede sont pertinentes
		}
		res := 0
		for _, d := range damages {
			if d.VictimIdx < 0 || d.ResponsibleIdx < 0 {
				continue
			}
			_, okv := nearestSample(tracks[uint32(b+d.VictimIdx)], d.TimestampUS)
			_, oka := nearestSample(tracks[uint32(b+d.ResponsibleIdx)], d.TimestampUS)
			if okv && oka {
				res++
			}
		}
		if res > peakRes {
			peakBase, peakRes = b, res
		}
	}
	return peakBase
}

// NewWeaponHitDistanceFunc fabrique la WeaponHitDistanceFunc a injecter dans PairWeaponHits :
// pour un degat apparie, elle rend la distance monde attaquant<->victime au ts du degat, ok=false
// si l une des deux positions ne se resout pas (touche comptee, distance non). tracks/base viennent
// de BuildBipedTracks + ResolveHitDistanceBase.
func NewWeaponHitDistanceFunc(tracks map[uint32][]hitPosSample, base int) WeaponHitDistanceFunc {
	return func(d WeaponDamage) (float64, bool) {
		if d.VictimIdx < 0 || d.ResponsibleIdx < 0 {
			return 0, false
		}
		pv, okv := nearestSample(tracks[uint32(base+d.VictimIdx)], d.TimestampUS)
		pa, oka := nearestSample(tracks[uint32(base+d.ResponsibleIdx)], d.TimestampUS)
		if !okv || !oka {
			return 0, false
		}
		return sampleDist(pa, pv), true
	}
}

// FilmWeaponHitDistance est le raccourci de production : decode les tracks, calibre la base et rend
// la WeaponHitDistanceFunc prete a injecter, plus la base retenue. Une entree de catalogue sans
// decoupage exploitable rend (nil, 0, err) : l appelant traite l absence de distance comme un cas
// normal (hits comptes sans distance), il ne fait pas echouer la passe.
func FilmWeaponHitDistance(dir string, entry MapQuantEntry, damages []WeaponDamage, n int) (WeaponHitDistanceFunc, int, error) {
	tracks, err := BuildBipedTracks(dir, entry, n)
	if err != nil {
		return nil, 0, err
	}
	base := ResolveHitDistanceBase(damages, tracks)
	return NewWeaponHitDistanceFunc(tracks, base), base, nil
}

// LA CARTE NE SE DEVINE PLUS ICI — `DetectFilmMapEntry` A ETE SUPPRIMEE AU LOT 1.9.4 (2026-09-15).
//
// Elle resolvait l ENTREE DE CATALOGUE d un film par la SIGNATURE de ses largeurs d axe : lire le
// decoupage d i0 dans le film ([DetectI0Layout]) puis retenir l entree du catalogue dont les
// `axisWidths` coincident, a condition qu il n y en ait qu UNE. Son unique appelant,
// `sync/killcollector/hits.go`, lui passait un `mapNameOverride` VIDE alors que le meme
// collecteur resolvait deja le nom de carte du match par la base.
//
// LA MESURE DU LOT (§5 du plan) : les largeurs valent `min(26, ceilLog2(ceil(60*etendue)))` par
// axe, une grandeur si grossiere que 68 des 79 cartes du catalogue tombent dans 5 classes de meme
// signature — dont une de 59 cartes. Sur les 17 films mesures : 2 accords, 13 ambiguites, et
// 2 DESACCORDS (les deux films Live Fire, ou la signature designe `aquarius` avec un seul
// candidat). Elle n a donc pas ete rendue optionnelle ni retrogradee en repli : une mecanique
// juste 2 fois sur 17 ne protege de rien et fabrique des distances fausses.
//
// La carte vient desormais du NOM DE MATCH (`killcollector/map_identity.go`) ; une carte sans nom
// ou hors catalogue est une erreur typee COMPTEE (D-4 d ADR 0034). [DetectI0Layout] n a plus
// aucun appelant de PRODUCTION : elle reste l enveloppe D2 des instruments de recherche, sous la
// garde de `archlint/no_recomputed_film_context_test.go` (allowlist) et de
// `archlint/no_film_reread_test.go` (enveloppes interdites en production).

// nearestSample rend la position du slot la plus proche de T dans la fenetre
// [T-WeaponHitPosToleranceUS, T+WeaponHitPosToleranceUS], et sa validite. La tolerance est la
// constante MESUREE du chantier — elle etait un parametre que tous les appelants passaient a
// l'identique (unparam, ratchet CI) ; re-parametrer le jour ou une mesure exigera une autre fenetre.
func nearestSample(track []hitPosSample, T uint64) (hitPosSample, bool) {
	const tol = WeaponHitPosToleranceUS
	if len(track) == 0 {
		return hitPosSample{}, false
	}
	i := sort.Search(len(track), func(i int) bool { return track[i].ts >= T })
	best, ok := hitPosSample{}, false
	var bd uint64 = math.MaxUint64
	pick := func(s hitPosSample) {
		d := T - s.ts
		if s.ts > T {
			d = s.ts - T
		}
		if d <= tol && d < bd {
			best, ok, bd = s, true, d
		}
	}
	if i-1 >= 0 {
		pick(track[i-1])
	}
	if i < len(track) {
		pick(track[i])
	}
	return best, ok
}

// sampleDist rend la distance euclidienne monde entre deux positions.
func sampleDist(a, b hitPosSample) float64 {
	dx, dy, dz := float64(b.x-a.x), float64(b.y-a.y), float64(b.z-a.z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
