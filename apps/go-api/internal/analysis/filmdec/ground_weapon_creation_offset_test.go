package filmdec

// ground_weapon_creation_offset_test.go — L'ORACLE DE POSITION applique a la grammaire portee
// de `ti=42` (arme au sol), et son CONTROLE POSITIF sur `ti=37`.
//
// LA QUESTION. `consumeDefaultStateTI42` (default_state_ti42.go) est un port de decompile. Rien
// ne dit qu'il soit juste : une entree fausse dans `defaultStateDeserByTI` decalerait le
// decodage de TOUS les records `ti=42` sans qu'aucune mesure ne le signale — c'est exactement
// la raison pour laquelle le lot R5 l'avait ecrit puis RETIRE.
//
// LA METHODE, ORACLE D'ABORD (transposee de equipment_creation_offset_test.go). On ne cherche
// pas l'en-tete : on cherche la POSITION. Les vies d'objet `ti=42` sont deja decodees par les
// paquets delta (`ScanFilmWorldObjects`), donc le premier point de chaque vie est un triplet de
// quanta CONNU. On balaye chaque position de bit a la recherche d'un masque de presence valide
// suivi d'une position i0 egale a ce premier point ; cette coincidence-la ne s'obtient pas par
// hasard. On regarde ENSUITE ce qui precede : si un en-tete NEW `ti=42` au bon couple
// (slot, generation) se trouve a une distance CONSTANTE, cette distance EST la largeur du
// default-state, en-tete et porte comprises.
//
// DEUX MESURES, ET LA SECONDE EST CELLE QUI TRANCHE.
//
//	LE PIC          distance dominante en-tete -> masque, comparee au CHEMIN MINIMAL
//	                (toutes portes fermees) : 24 + 60 + 1 = 85 pour ti=37, 24 + 80 + 1 = 105
//	                pour ti=42. C'est une INDICATION, pas un verdict : un record reel ouvre des
//	                portes et sa largeur monte d'autant. Un pic AU-DESSUS du minimal ne refute
//	                donc rien — mesure le 2026-08-17, ti=42 pique a 118, 150 ou 182 selon le
//	                film, et ces trois valeurs se decomposent exactement en portes de la
//	                grammaire portee (+5 la reference d'entite, +8 le prefixe de version,
//	                +13 le champ optionnel du bloc MPP, +32 un identifiant).
//	L'ATTERRISSAGE  on DEROULE le deserialiseur candidat sur les bits du record et on exige
//	                qu'il tombe AU BIT PRES sur le masque que l'oracle de position a localise.
//	                Exact, record par record, sans tolerance. C'est LE verdict.
//
// Le CONTROLE sur `ti=37` passe par le MEME code : son deserialiseur est valide par ailleurs,
// donc s'il n'atterrit pas non plus, c'est la mesure qui est en cause, pas `ti=42`.
//
// LECTURE SEULE, gardee par GW_CREATION_FILM. UN SEUL decodage filmdec par process.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GW_CREATION_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestGroundWeaponCreationOffset$' -timeout 60m -v

import (
	"fmt"
	"os"
	"testing"
)

const gwCreationFilmEnv = "GW_CREATION_FILM"

// gwOffsetPrediction rend la distance en-tete -> masque attendue pour une largeur de deser :
// l'en-tete NEW (24 bits) precede le deser, la porte du record NEW (1 bit) le suit.
func gwOffsetPrediction(deserBits int) int { return woNewHeaderBits + deserBits + 1 }

func TestGroundWeaponCreationOffset(t *testing.T) {
	dir := os.Getenv(gwCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", gwCreationFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)
	t.Logf("FILM %s · largeurs %v", dir, lay.AxisW)

	// MESURE puis CONTROLE, dans cet ordre : le controle ne sert qu'a juger la mesure.
	gwRunOffset(t, dir, GroundWeaponTypeIndex, gwOffsetPrediction(GroundWeaponDefaultStateMinBits))
	gwRunOffset(t, dir, EquipmentTypeIndex, gwOffsetPrediction(60))
}

// gwRunOffset joue l'oracle de position pour UN archetype et publie son histogramme.
func gwRunOffset(t *testing.T, dir string, ti int, predicted int) {
	t.Helper()
	arch, err := gwArchetypeForTI(dir, ti)
	if err != nil {
		t.Fatalf("archetype ti=%d : %v", ti, err)
	}
	n := CountFilmChunks(dir)
	band := worldObjectSlotBand(dir, n, ti)
	tracks, err := ScanFilmWorldObjects(dir, &equipCreationUnitRange, ti)
	if err != nil {
		t.Fatalf("trajectoires ti=%d illisibles : %v", ti, err)
	}
	land := gwNewLanding()
	pr := equipOffsetProbe{
		comps: len(arch.Components), band: band, want37: uint32(ti),
		want: map[[3]int32][]equipCreationWanted{},
		gap:  map[uint32]int{}, ti: map[uint32]int{}, hdr: map[uint32]int{},
		bits:    map[uint32]*[equipProfileBits]int{},
		onMatch: land.record,
	}
	for _, tr := range tracks {
		p := [3]float32{tr.Pts[0].X, tr.Pts[0].Y, tr.Pts[0].Z}
		c := equipCell(p)
		pr.want[c] = append(pr.want[c],
			equipCreationWanted{pos: p, life: equipCreationLifeKey{tr.Slot, tr.Gen}})
	}
	t.Logf("== ti=%d · %d composants · bande %d slots · %d vies delta · %d premiers points distincts",
		ti, len(arch.Components), len(band), len(tracks), len(pr.want))

	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeDelta {
				pr.scan(pk.Payload(data))
			}
		}
	}
	t.Logf("   CORPS TROUVES PAR L'ORACLE %d · dont precedes d'un en-tete NEW ti=%d %d",
		pr.bodies, ti, pr.withHeader)
	t.Logf("   distance en-tete -> masque, en bits (CHEMIN MINIMAL %d = 24 + deser + 1) :%s",
		predicted, equipCreationLine(pr.gap, 24))
	t.Logf("   en-tetes NEW quelconques en amont, par typeIndex lu :%s", equipCreationLine(pr.ti, 24))
	t.Logf("   VERDICT-PIC ti=%d : %s", ti, gwOffsetVerdict(pr.gap, predicted))
	land.log(t, ti)
}

// gwLanding porte L'ESSAI D'ATTERRISSAGE — la mesure qui tranche vraiment.
//
// POURQUOI LE PIC NE SUFFIT PAS. La prediction du pic est celle du CHEMIN MINIMAL (toutes les
// portes fermees) ; un record reel ouvre des portes, et sa largeur monte d'autant. Un pic
// au-dessus de la prediction ne refute donc rien par lui-meme — il faut DEROULER le
// deserialiseur sur les bits du record et voir s'il tombe AU BIT PRES sur le masque que
// l'oracle de position a localise. C'est un test exact, record par record, sans tolerance.
//
// LES TEMOINS SONT DES DESERIALISEURS FAUX passes par le MEME code. S'ils atterrissent autant
// que le porte, l'atterrissage ne mesure rien.
type gwLanding struct {
	cands []ti42Deser
	exact []int
	total int
}

func gwNewLanding() *gwLanding {
	l := &gwLanding{cands: []ti42Deser{
		{"ti42 porte (FUN_1407f0c68)", consumeDefaultStateTI42},
		{"TEMOIN ti37 (FUN_1407f105c)", consumeDefaultStateTI37},
		{"TEMOIN ti36 (FUN_1407f2224)", consumeDefaultStateTI36},
		{"TEMOIN ti38 (FUN_1408f0b48)", consumeDefaultStateTI38},
	}}
	l.exact = make([]int, len(l.cands))
	return l
}

// record deroule chaque candidat depuis la fin de l'en-tete et compte ceux qui tombent au bit
// pres sur le masque. Le masque commence 1 bit apres la fin du default-state : ce bit est la
// porte `has-components` du record NEW (cf. readCreation).
func (l *gwLanding) record(pay []byte, hdr, mask int) {
	l.total++
	for i, c := range l.cands {
		br := NewBitReader(pay)
		br.SetBitPos(hdr + woNewHeaderBits)
		c.fn(br)
		if br.BitPos()+1 == mask {
			l.exact[i]++
		}
	}
}

func (l *gwLanding) log(t *testing.T, ti int) {
	t.Helper()
	for i, c := range l.cands {
		pct := 0.0
		if l.total > 0 {
			pct = 100 * float64(l.exact[i]) / float64(l.total)
		}
		t.Logf("   ATTERRISSAGE ti=%d %-30s %4d exacts / %4d records (%5.1f %%)",
			ti, c.name, l.exact[i], l.total, pct)
	}
}

// gwOffsetVerdict rend le pic mesure, sa part, et son ecart a la prediction. Les DENOMINATEURS
// sont publies : un pic a 100 % sur trois records ne vaut pas un pic a 60 % sur trois cents.
func gwOffsetVerdict(gap map[uint32]int, predicted int) string {
	top := equipCreationTop(gap, 1)
	if len(top) == 0 {
		return "AUCUN en-tete trouve en amont d'un corps : la forme supposee du record de creation n'est pas celle du flux"
	}
	total := 0
	for _, n := range gap {
		total += n
	}
	best := top[0]
	pct := 100 * float64(best.count) / float64(total)
	if int(best.key) == predicted {
		return fmt.Sprintf("pic a %d bits = CHEMIN MINIMAL (%d/%d records, %.1f %%)",
			best.key, best.count, total, pct)
	}
	return fmt.Sprintf("pic a %d bits, chemin minimal %d, portes ouvertes %+d bits (%d/%d records, %.1f %%)",
		best.key, predicted, int(best.key)-predicted, best.count, total, pct)
}

// gwArchetypeForTI lit le registre du film (chunk_00) et rend l'archetype demande. Le pendant
// generique d'`EquipmentArchetype`, qui ne connait que `ti=37`.
func gwArchetypeForTI(dir string, ti int) (Archetype, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return Archetype{}, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return Archetype{}, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	arch, ok := reg.Archetype(ti)
	if !ok {
		return Archetype{}, fmt.Errorf("archetype %d absent du registre de %s", ti, dir)
	}
	return arch, nil
}
