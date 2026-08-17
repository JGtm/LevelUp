package filmdec

// default_state_ti42_oracle_test.go — L'ORACLE DE LARGEUR, jamais consomme jusqu'ici, applique
// a la grammaire portee de `ti=42`.
//
// CE QU'EST L'ORACLE. `.ai/V7.5/dumps/kf_capture_sample.txt` porte 400 FRONTIERES DE RECORDS
// EXACTES relevees au point d'entree du dispatcheur de record du jeu (`FUN_1406cbaa0`, hook
// Cheat Engine non bloquant) : par ligne, le TYPE de record (1 = NEW, 3 = DELTA), l'IDENTIFIANT
// d'entite, la POSITION EN BITS du reader (`reader+0x2c`) et l'adresse du tampon. Le tampon est
// `kf_slot0_live.bin` (7 286 octets). Deux frontieres consecutives donnent la LARGEUR EXACTE du
// record intercale : c'est une verite terrain qui ne depend d'aucun film, d'aucune carte, et
// d'aucune hypothese du depot.
//
// POURQUOI IL TRANCHE LA OU UN FILM NE TRANCHE PAS. Un balayage bit a bit d'un film accepte un
// record quand son corps « se deroule sans absurdite » — critere negatif. Ici la largeur est
// DONNEE : un deserialiseur juste atterrit AU BIT PRES sur la frontiere suivante, un
// deserialiseur faux n'y atterrit pas. Aucune tolerance, aucun ajustement.
//
// LE TEMOIN EST OBLIGATOIRE. Un taux d'atterrissage exact ne veut rien dire seul : on rejoue le
// MEME calcul avec des deserialiseurs FAUX (celui de ti=37, celui de ti=36, et un saut fixe a la
// largeur predite). Si les faux atterrissent autant que le vrai, la mesure ne mesure rien.
//
// LECTURE SEULE, aucune base, aucun film. SOUS GARDE D'ENVIRONNEMENT (TI42_CAPTURE).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 TI42_CAPTURE=<repo>/.ai/V7.5/dumps/kf_capture_sample.txt \
//	  TI42_BUFFER=<repo>/.ai/V7.5/dumps/kf_slot0_live.bin \
//	  go test ./internal/analysis/filmdec/ -run TI42WidthOracle -timeout 10m -v

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	ti42CaptureEnv = "TI42_CAPTURE" // kf_capture_sample.txt : les frontieres exactes
	ti42BufferEnv  = "TI42_BUFFER"  // kf_slot0_live.bin : le tampon correspondant
)

// ti42CaptureRecord est une frontiere de record relevee en live.
type ti42CaptureRecord struct {
	kind   int    // 1 = NEW, 2 = DEL, 3 = DELTA (valeur de RCX au dispatch)
	id     uint32 // identifiant d'entite (tag sur les bits 30-31, slot sur les 30 bits bas)
	bitPos int    // position du reader APRES l'en-tete type+id, la ou R(6) typeIndex se lit
}

// ti42HeaderWidths : les deux en-tetes possibles avant `bitPos`. NEW/DEL passent par
// `R(1)=0 ; R(2)=type` (3 bits) + identifiant (13 + 2) = 18 ; un DELTA emprunte le raccourci
// `R(1)=1` (1 bit) + identifiant = 16. On ne CHOISIT pas : on reconcilie l'en-tete avec
// l'identifiant relevé en live, et un record dont aucun en-tete ne se reconcilie est compte a
// part au lieu d'etre suppose.
var ti42HeaderWidths = []int{18, 16}

func TestTI42WidthOracle(t *testing.T) {
	capPath, bufPath := os.Getenv(ti42CaptureEnv), os.Getenv(ti42BufferEnv)
	if capPath == "" || bufPath == "" {
		t.Skipf("%s/%s absents : oracle de largeur saute", ti42CaptureEnv, ti42BufferEnv)
	}
	buf, err := os.ReadFile(bufPath)
	if err != nil {
		t.Fatalf("tampon %s illisible : %v", bufPath, err)
	}
	recs, err := ti42ReadCapture(capPath)
	if err != nil {
		t.Fatalf("capture %s illisible : %v", capPath, err)
	}
	t.Logf("CAPTURE — %d frontieres, tampon %d octets (%d bits)", len(recs), len(buf), len(buf)*8)

	hdr, unresolved := ti42ResolveHeaders(buf, recs)
	t.Logf("EN-TETES — reconcilies %d / %d (non reconcilies %d)",
		len(recs)-unresolved, len(recs), unresolved)

	widths, tis := ti42BodyWidths(buf, recs, hdr)
	ti42LogWidthsByTI(t, recs, widths, tis)
	ti42LogLanding(t, buf, recs, widths, tis)
}

// ti42ReadCapture lit le releve : `<type> <id hexa> <bit> <tampon>` par ligne.
func ti42ReadCapture(path string) ([]ti42CaptureRecord, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []ti42CaptureRecord
	for _, line := range strings.Split(string(blob), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		kind, e1 := strconv.Atoi(f[0])
		id, e2 := strconv.ParseUint(f[1], 16, 64)
		bp, e3 := strconv.Atoi(f[2])
		if e1 != nil || e2 != nil || e3 != nil {
			continue
		}
		out = append(out, ti42CaptureRecord{kind: kind, id: uint32(id), bitPos: bp})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].bitPos < out[j].bitPos })
	return out, nil
}

// ti42ResolveHeaders rend, pour chaque record, la largeur de son en-tete — celle dont les bits
// qui precedent `bitPos` REDONNENT l'identifiant et le type releves en live. 0 = non reconcilie.
func ti42ResolveHeaders(buf []byte, recs []ti42CaptureRecord) (hdr []int, unresolved int) {
	hdr = make([]int, len(recs))
	for i, r := range recs {
		for _, h := range ti42HeaderWidths {
			if r.bitPos-h < 0 || !ti42HeaderMatches(buf, r, h) {
				continue
			}
			hdr[i] = h
			break
		}
		if hdr[i] == 0 {
			unresolved++
		}
	}
	return hdr, unresolved
}

// ti42HeaderMatches dit si l'en-tete de largeur h qui precede `bitPos` porte bien le type et
// l'identifiant releves. L'identifiant se lit `R(13)` bas puis `R(2)` de tag (FUN_1406d3140).
func ti42HeaderMatches(buf []byte, r ti42CaptureRecord, h int) bool {
	p := r.bitPos - h
	switch h {
	case 18:
		if PeekBits(buf, p, 1) != 0 || int(PeekBits(buf, p+1, 2)) != r.kind {
			return false
		}
	case 16:
		if PeekBits(buf, p, 1) != 1 || r.kind != 3 {
			return false
		}
	default:
		return false
	}
	low := uint32(PeekBits(buf, r.bitPos-15, 13))
	tag := uint32(PeekBits(buf, r.bitPos-2, 2))
	return low == (r.id&0x3fffffff)&0x1fff && tag == r.id>>30
}

// ti42BodyWidths rend la largeur EXACTE du corps de chaque record (de `bitPos` a l'en-tete du
// suivant) et son typeIndex quand le record est un NEW. -1 = indeterminable.
func ti42BodyWidths(buf []byte, recs []ti42CaptureRecord, hdr []int) (widths, tis []int) {
	widths = make([]int, len(recs))
	tis = make([]int, len(recs))
	for i := range recs {
		widths[i], tis[i] = -1, -1
		if i+1 < len(recs) && hdr[i+1] > 0 {
			widths[i] = recs[i+1].bitPos - hdr[i+1] - recs[i].bitPos
		}
		if recs[i].kind == 1 {
			tis[i] = int(PeekBits(buf, recs[i].bitPos, 6))
		}
	}
	return widths, tis
}

// ti42LogWidthsByTI publie la distribution des largeurs de corps par archetype : c'est la piece
// justificative de l'oracle, et elle dit du meme coup si `ti=42` figure dans la capture.
func ti42LogWidthsByTI(t *testing.T, recs []ti42CaptureRecord, widths, tis []int) {
	t.Helper()
	byTI := map[int][]int{}
	for i := range recs {
		if tis[i] < 0 || widths[i] < 0 {
			continue
		}
		byTI[tis[i]] = append(byTI[tis[i]], widths[i])
	}
	keys := make([]int, 0, len(byTI))
	for k := range byTI {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		w := byTI[k]
		sort.Ints(w)
		t.Logf("   ti=%-3d records NEW %3d · largeurs de corps min %5d / mediane %5d / max %5d%s",
			k, len(w), w[0], w[len(w)/2], w[len(w)-1], ti42Sample(w))
	}
}

// ti42Sample rend les premieres largeurs distinctes, pour lire la forme sans noyer le journal.
func ti42Sample(w []int) string {
	var out []string
	for i, v := range w {
		if i > 0 && v == w[i-1] {
			continue
		}
		if len(out) >= 8 {
			out = append(out, "...")
			break
		}
		out = append(out, strconv.Itoa(v))
	}
	return " · distinctes " + strings.Join(out, ",")
}

// ti42Deser porte un candidat de deserialiseur et son nom, pour la mesure et pour ses temoins.
type ti42Deser struct {
	name string
	fn   func(*BitReader)
}

// ti42LogLanding est LA mesure : sur les records NEW de la capture, le deserialiseur porte
// atterrit-il AU BIT PRES sur la frontiere suivante ? On mesure archetype par archetype, avec
// trois temoins FAUX qui doivent s'effondrer.
func ti42LogLanding(t *testing.T, buf []byte, recs []ti42CaptureRecord, widths, tis []int) {
	t.Helper()
	cands := []ti42Deser{
		{"ti42 porte (FUN_1407f0c68)", consumeDefaultStateTI42},
		{"TEMOIN ti37 (FUN_1407f105c)", consumeDefaultStateTI37},
		{"TEMOIN ti36 (FUN_1407f2224)", consumeDefaultStateTI36},
		{"TEMOIN saut fixe 80 bits", func(br *BitReader) { br.Skip(GroundWeaponDefaultStateMinBits) }},
	}
	for _, c := range cands {
		exact, total := ti42CountLandings(buf, recs, widths, tis, c.fn)
		t.Logf("ATTERRISSAGE %-30s — ti=42 : %d exacts sur %d records", c.name, exact, total)
	}
	// TEMOIN CROISE : le meme calcul sur ti=37, dont le deserialiseur est deja valide par
	// ailleurs. S'il n'atterrit pas non plus, c'est la MESURE qui est en cause, pas ti=42.
	for _, c := range cands[:2] {
		exact, total := ti42CountLandingsTI(buf, recs, widths, tis, c.fn, EquipmentTypeIndex)
		t.Logf("CONTROLE ti=37  %-30s — %d exacts sur %d records", c.name, exact, total)
	}
}

func ti42CountLandings(
	buf []byte, recs []ti42CaptureRecord, widths, tis []int, fn func(*BitReader),
) (exact, total int) {
	return ti42CountLandingsTI(buf, recs, widths, tis, fn, GroundWeaponTypeIndex)
}

// ti42CountLandingsTI compte les records de l'archetype `want` dont le corps se deroule
// EXACTEMENT jusqu'a la frontiere suivante : R(6) typeIndex, l'etat par defaut candidat, la
// porte du record NEW, puis — seulement si la porte est fermee — plus rien. Les records a porte
// ouverte portent un masque et des composants dont les largeurs dependent de la carte de la
// capture, inconnue : ils sont exclus du denominateur au lieu d'etre devines.
func ti42CountLandingsTI(
	buf []byte, recs []ti42CaptureRecord, widths, tis []int, fn func(*BitReader), want int,
) (exact, total int) {
	for i := range recs {
		if tis[i] != want || widths[i] < 0 {
			continue
		}
		br := NewBitReader(buf)
		br.SetBitPos(recs[i].bitPos + 6) // R(6) typeIndex deja lu
		fn(br)
		if br.BitPos() > len(buf)*8 {
			continue
		}
		gateAt := br.BitPos()
		if PeekBits(buf, gateAt, 1) != 0 {
			continue // porte ouverte : masque + composants, largeurs dependantes de la carte
		}
		total++
		if gateAt+1-recs[i].bitPos == widths[i] {
			exact++
		}
	}
	return exact, total
}
