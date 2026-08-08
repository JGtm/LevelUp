// Commande tmp_pjcnt — INSTRUMENT DE RECHERCHE (piste E, precision projectiles).
//
// Verdict et chiffres : `.ai/V7.5/VERDICT_PRECISION_PROJECTILES.md`.
//
// Quatre mesures, chacune derriere son drapeau :
//
//	-align   gate de reproduction : P(pas = +1) du compteur 7 bits a -1 / 0 / +1 bit
//	-hdr     ventilation des records type 105 par classe d en-tete et par suffixe d arme
//	-arme    par arme : taux de porteur, pas moyen du compteur, CADENCE inter-record
//	-fit     deconvolution du taux de touche par arme contre les touches API du match
//
// Rien ici n est du code de production : aucun fichier livre n est touche, la sortie
// est un CSV de mesure. Le corpus film est lu en LECTURE SEULE, toujours plafonne
// par -limit (le balayage non borne du corpus est une bombe RAM documentee).
//
// Repere de bits — le record type 105 dans la numerotation PAYLOAD de filmdec :
//
//	bits 0..6    (7)  type d event = 105          -> pay[0]>>1
//	bit  7       (1)  variante (0 = long, porte l arme)
//	bits 26..32  (7)  COMPTEUR DE TIR             <- 7ter.80 `eventStart+22`, eventStart = bit 4
//	bits 35..39  (5)  indice de tireur            <- GUIDE_WEAPON_SHOTS §6 (5 bits, pas 4)
//	bits 44..75  (32) arme, moitie haute
//	bits 76..107 (32) arme, moitie basse
//
// L equivalence `eventStart = payload bit 4` se lit sur l arme : l ancien scanneur la
// prend a `eventStart+40`, filmdec a `payload 44`.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

// Offsets, en bits depuis le debut du payload du record type 105.
const (
	recType     = 105
	bitCounter  = 26 // 7ter.80 : eventStart+22, eventStart = payload bit 4
	widthCount  = 7
	bitShooter  = 35 // 5 bits — largeur reelle (GUIDE_WEAPON_SHOTS §6)
	widthShoot  = 5
	bitWeaponHi = 44
	bitWeaponLo = 76
	widthWeapon = 32
	// minBitsHead : le compteur et l indice tiennent avant le bit 40.
	minBitsHead = 40
	// minBitsWeapon : l arme complete tient avant le bit 108.
	minBitsWeapon = 108
	// bitCountersNull : drapeau « compteurs nuls » du record (bit 110 du payload).
	bitCountersNull = 110
	counterMod      = 1 << widthCount
)

// rec est un record type 105 lu dans le flux, dans l ordre du film.
type rec struct {
	shooter int
	counter int
	weapon  uint64
	long    bool
	hasWeap bool
	// porteur : le record applique reellement du degat. Le bit 110 du payload est le
	// drapeau « compteurs nuls » du record type 105 (filmdec/fire_events.go, chemin
	// « record vide ») ; porteur = ce drapeau A ZERO.
	porteur bool
	// tsUS est l horodatage moteur du paquet porteur du record (meme horloge que les
	// positions). Il sert a mesurer la CADENCE inter-record, qui dit si un record est
	// un TIR (cadence = temps de cycle de l arme) ou une TOUCHE (cadence irreguliere).
	tsUS uint64
	// hdr porte les bits 8..11 du payload. L ancien scanneur bit-a-bit les EXIGEAIT a
	// 0b0110 (ils font partie de son marqueur 11 bits) ; le balayage par paquets ne les
	// contraint pas. C est la seule difference de population entre les deux instruments.
	hdr int
}

// playerAgg agrege les records d un indice de tireur dans un film.
type playerAgg struct {
	records   int            // records longs (ceux qui portent une arme)
	shortRecs int            // records courts
	gapSum    int            // somme des pas du compteur entre records longs consecutifs
	gapSumAll int            // idem en incluant les records courts dans la chaine
	steps     map[int]int    // distribution des pas (records longs)
	byWeapon  map[uint64]int // records longs par arme
	gapByWeap map[uint64]int // pas attribues a une arme (les deux bornes portent la meme)
	gapAmbig  int            // pas dont les deux bornes portent des armes differentes
}

func newPlayerAgg() *playerAgg {
	return &playerAgg{
		steps:     map[int]int{},
		byWeapon:  map[uint64]int{},
		gapByWeap: map[uint64]int{},
	}
}

// apiRow porte la reference API d un joueur (jamais dans le critere de decodage).
type apiRow struct {
	shotsFired, shotsHit int
}

type matchRef struct {
	matchID  string
	pairName string
	players  []apiRow
}

func main() {
	var (
		filmsDir = flag.String("films", "", "racine du cache de films (lecture seule)")
		refCSV   = flag.String("ref", "", "CSV de reference API (pfx,match_id,pair_name,xuid,gamertag,shots_fired,shots_hit,...)")
		famille  = flag.String("famille", "", "filtre de famille de mode : FIESTA|TACTICAL|BTB|STANDARD (vide = tout)")
		limit    = flag.Int("limit", 20, "nombre maximum de films traites (plafond memoire/CPU)")
		outCSV   = flag.String("out", "", "CSV de sortie par match")
		outPlr   = flag.String("outplayer", "", "CSV de sortie par (match, indice de tireur)")
		align    = flag.Bool("align", false, "controle d alignement : mesure P(pas=+1) a -1/0/+1 bit")
		hdrStats = flag.Bool("hdr", false, "ventile les records par classe d en-tete et par suffixe d arme")
		perWeap  = flag.Bool("arme", false, "pas moyen du compteur par arme (test de la piste E)")
		doFit    = flag.Bool("fit", false, "deconvolution du taux de touche par arme contre les touches API du match")
		minShots = flag.Float64("minshots", 3000, "tirs minimum pour qu une arme entre dans le systeme")
		normVis  = flag.Bool("norm", false, "normalise les tirs decodes de chaque match par sa visibilite (total API)")
		weapCSV  = flag.String("armes", "", "CSV weapon_id,name_en exporte de metadata.duckdb")
	)
	flag.Parse()

	if *filmsDir == "" || *refCSV == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_pjcnt -films <dir> -ref <ref.csv> [-famille FIESTA] [-limit N]")
		os.Exit(2)
	}

	refs, err := loadRef(*refCSV)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reference illisible: %v\n", err)
		os.Exit(1)
	}

	pfxs := selectFilms(*filmsDir, refs, *famille, *limit)
	fmt.Fprintf(os.Stderr, "films retenus: %d (famille=%q)\n", len(pfxs), *famille)

	if *align {
		runAlign(*filmsDir, pfxs)
		return
	}
	if *hdrStats {
		runHdr(*filmsDir, pfxs)
		return
	}
	if *perWeap {
		runWeapon(*filmsDir, pfxs, loadWeaponNames(*weapCSV), *outCSV)
		return
	}
	if *doFit {
		runFit(*filmsDir, pfxs, refs, loadWeaponNames(*weapCSV), *minShots, *outCSV, *normVis)
		return
	}
	runMain(*filmsDir, pfxs, refs, *outCSV, *outPlr)
}

// ─────────────────────────────────────────────────────────────────────────────
//  Lecture des films
// ─────────────────────────────────────────────────────────────────────────────

// scanRecords lit tous les records type 105 d un film, DANS L ORDRE DU FLUX.
// Un chunk a la fois : le corpus entier ne tient pas en memoire.
func scanRecords(dir string, counterBit int) []rec {
	n := filmdec.CountFilmChunks(dir)
	var out []rec
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != recType {
				continue
			}
			bits := len(pay) * 8
			if bits < minBitsHead {
				continue
			}
			r := rec{
				shooter: int(readBits(pay, bitShooter, widthShoot)),
				counter: int(readBits(pay, counterBit, widthCount)),
				long:    pay[0]&1 == 0,
				hdr:     int(readBits(pay, 8, 4)),
				tsUS:    p.TimestampUS,
			}
			if r.long && bits >= minBitsWeapon {
				r.weapon = readBits(pay, bitWeaponHi, widthWeapon)<<32 | readBits(pay, bitWeaponLo, widthWeapon)
				r.hasWeap = true
			}
			if r.long && bits > bitCountersNull {
				r.porteur = readBits(pay, bitCountersNull, 1) == 0
			}
			out = append(out, r)
		}
	}
	return out
}

// readBits lit n bits (n <= 64) a partir de bitPos, MSB d abord.
func readBits(data []byte, bitPos, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		idx := (bitPos + i) / 8
		if idx >= len(data) {
			return v << uint(n-i)
		}
		bit := (data[idx] >> uint(7-(bitPos+i)%8)) & 1
		v = v<<1 | uint64(bit)
	}
	return v
}

// aggregate regroupe les records par indice de tireur et calcule les pas du compteur.
func aggregate(recs []rec) map[int]*playerAgg {
	byPlayer := map[int][]rec{}
	for _, r := range recs {
		byPlayer[r.shooter] = append(byPlayer[r.shooter], r)
	}
	out := map[int]*playerAgg{}
	for pi, rs := range byPlayer {
		a := newPlayerAgg()
		var prevLong *rec
		var prevAny *rec
		for i := range rs {
			r := rs[i]
			if isFire(r) {
				a.records++
				a.byWeapon[r.weapon]++
				if prevLong != nil {
					step := (r.counter - prevLong.counter + counterMod) % counterMod
					a.gapSum += step
					a.steps[step]++
					switch {
					case prevLong.hasWeap && r.hasWeap && prevLong.weapon == r.weapon:
						a.gapByWeap[r.weapon] += step
					default:
						a.gapAmbig += step
					}
				}
				prevLong = &rs[i]
			} else {
				a.shortRecs++
			}
			if prevAny != nil {
				a.gapSumAll += (r.counter - prevAny.counter + counterMod) % counterMod
			}
			prevAny = &rs[i]
		}
		out[pi] = a
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
//  Reference API
// ─────────────────────────────────────────────────────────────────────────────

func loadRef(path string) (map[string]*matchRef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("reference vide")
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[h] = i
	}
	need := []string{"pfx", "match_id", "pair_name", "shots_fired", "shots_hit"}
	for _, k := range need {
		if _, ok := col[k]; !ok {
			return nil, fmt.Errorf("colonne %q absente", k)
		}
	}
	out := map[string]*matchRef{}
	for _, row := range rows[1:] {
		pfx := row[col["pfx"]]
		m := out[pfx]
		if m == nil {
			m = &matchRef{matchID: row[col["match_id"]], pairName: row[col["pair_name"]]}
			out[pfx] = m
		}
		m.players = append(m.players, apiRow{
			shotsFired: atoi(row[col["shots_fired"]]),
			shotsHit:   atoi(row[col["shots_hit"]]),
		})
	}
	return out, nil
}

func atoi(s string) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return v
}

// familyOf classe un pair_name dans une famille d arsenal.
func familyOf(pairName string) string {
	p := strings.ToLower(pairName)
	switch {
	case strings.Contains(p, "fiesta"):
		return "FIESTA"
	case strings.Contains(p, "tactical"):
		return "TACTICAL"
	case strings.HasPrefix(p, "btb"):
		return "BTB"
	default:
		return "STANDARD"
	}
}

// selectFilms rend les prefixes de films presents sur disque ET dans la reference,
// filtres par famille, tries pour etre deterministes, plafonnes a limit.
func selectFilms(root string, refs map[string]*matchRef, famille string, limit int) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cache de films illisible: %v\n", err)
		os.Exit(1)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := refs[e.Name()]
		if m == nil {
			continue
		}
		if famille != "" && familyOf(m.pairName) != famille {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// quantile rend le quantile q d un echantillon (copie triee ; 0 si vide).
func quantile(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	i := int(q * float64(len(cp)-1))
	return cp[i]
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func sortedKeys(m map[int]*playerAgg) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func openCSV(path string, header []string) (*csv.Writer, func()) {
	if path == "" {
		return nil, func() {}
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sortie CSV impossible: %v\n", err)
		os.Exit(1)
	}
	w := csv.NewWriter(f)
	_ = w.Write(header)
	return w, func() { w.Flush(); _ = f.Close() }
}

func writeRow(w *csv.Writer, vals ...any) {
	if w == nil {
		return
	}
	row := make([]string, len(vals))
	for i, v := range vals {
		switch t := v.(type) {
		case string:
			row[i] = t
		case int:
			row[i] = strconv.Itoa(t)
		default:
			row[i] = fmt.Sprint(t)
		}
	}
	_ = w.Write(row)
}
