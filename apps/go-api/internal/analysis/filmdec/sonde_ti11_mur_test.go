package filmdec

// sonde_ti11_mur_test.go — INSTRUMENT DE MESURE DU MUR de l'archetype OBJECTIFS (ti=11),
// phase 4 du plan .ai/V7.5/replay2d/PLAN_R4_OBJECTIFS_VIVANTS_TI11.md (lot R4).
//
// POURQUOI CETTE VOIE, ET PAS CELLE DE LA PHASE 1. La phase 1 a cherche les records ti=11 dans
// les paquets DELTA, avec le reconnaisseur d'en-tete des objets du monde : REFUTE (la vraie
// bande rend moins que son fantome, et pres d'un « record » sur deux porte un index hors
// grammaire). Ici on ne cherche plus : les records d'IMAGE-CLE sont LOCALISES EXACTEMENT par
// WalkKeyframeWorld, qui rend le bit de debut de chacun. 201 records ti=11 sur 64e8adfa, sans
// aucune reconnaissance de motif — donc sans aucun faux positif possible.
//
// LA CLE DE L'ALIGNEMENT, et elle est structurelle : l'en-tete d'un record d'image-cle est
// [id:32][field:26][ti:6] (keyframe_world.go:19-22), et TraverseEntity commence precisement par
// lire R(6) typeIndex (traverse.go:1010). Le champ `ti` de l'en-tete EST le typeIndex que
// TraverseEntity attend : il suffit donc de poser le lecteur a `Bit + 58`. Aucun calage a
// deviner, aucun balayage — et le controle est immediat, puisque TraverseEntity doit relire 11.
//
// CE QU'IL MESURE :
//
//	4.1  le MASQUE reel des records ti=11 (quels composants sont PRESENTS), et donc
//	     4.2  LE MUR : le premier composant present, qui est aussi le premier non porte (0/34).
//	     Le taux d'indices hors grammaire (> i33) sert de controle de purete, comme en phase 1 :
//	     ici il doit tomber a ZERO, sans quoi l'alignement serait faux.
//
// TEMOIN DE CONTROLE : le MEME calcul sur un archetype DEJA DECODE (ti=37, equipement, 30/31
// composants portes). Si la methode est juste, elle doit rendre sur ti=37 des masques propres ET
// une traversee qui aboutit — c'est ce qui prouve que ce n'est pas la methode qui est fautive
// quand ti=11 desynchronise.
//
// IL NE MODIFIE RIEN : lecture seule du film, aucune ecriture disque. SOUS GARDE
// D'ENVIRONNEMENT (TI11_FILM), saute partout ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 TI11_FILM=<repo>/data/cache/film_chunks/64e8adfa \
//	  go test ./internal/analysis/filmdec/ -run TI11Mur -v

import (
	"sort"
	"testing"
)

// ti11WallStats agrege la traversee des records d'un archetype.
type ti11WallStats struct {
	records      int
	tiMismatch   int // TraverseEntity n'a pas relu le ti attendu : alignement faux
	outOfGrammar int // masque portant un index > dernier composant de l'archetype
	emptyMask    int
	firstPresent map[int]int
	present      map[int]int
	desyncAt     map[int]int
	completed    int // traversee aboutie (DesyncAt == -1)
	popcount     map[int]int
}

func newTI11WallStats() ti11WallStats {
	return ti11WallStats{
		firstPresent: map[int]int{}, present: map[int]int{},
		desyncAt: map[int]int{}, popcount: map[int]int{},
	}
}

// ti11WalkArchetype traverse tous les records d'image-cle d'un archetype donne et agrege ce que
// la traversee rend. HORS LIGNE (I/O disque sur tout le film).
func ti11WalkArchetype(dir string, reg *Registry, wantTI int) ti11WallStats {
	s := newTI11WallStats()
	arch, ok := reg.Archetype(wantTI)
	if !ok {
		return s
	}
	maxIdx := len(arch.Components) - 1
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, r := range WalkKeyframeWorld(pay) {
				if r.TI != wantTI {
					continue
				}
				s.records++
				br := NewBitReader(pay)
				br.SetBitPos(r.Bit + keyframeRecordTIBit)
				tr := TraverseEntity(br, reg, 0)
				if int(tr.TypeIndex) != wantTI {
					s.tiMismatch++
					continue
				}
				idx := ti11MaskIndices(tr.Mask)
				s.popcount[len(idx)]++
				if len(idx) == 0 {
					s.emptyMask++
					continue
				}
				bad := false
				for _, i := range idx {
					if i > maxIdx {
						bad = true
					}
					s.present[i]++
				}
				if bad {
					s.outOfGrammar++
				}
				s.firstPresent[idx[0]]++
				s.desyncAt[tr.DesyncAt]++
				if tr.DesyncAt == -1 {
					s.completed++
				}
			}
		}
	}
	return s
}

// ti11MaskIndices rend les index de composant presents dans un masque, croissants.
func ti11MaskIndices(mask uint64) []int {
	var out []int
	for i := 0; i < 64; i++ {
		if mask&(uint64(1)<<uint(i)) != 0 {
			out = append(out, i)
		}
	}
	return out
}

// TestTI11MurParImageCle (4.1 + 4.2) situe le mur de ti=11 sur des records LOCALISES, et se
// controle sur un archetype deja decode.
func TestTI11MurParImageCle(t *testing.T) {
	dir := ti11Dir(t)
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("lecture de chunk_00 (registre) : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("analyse du registre : %v", err)
	}

	// TEMOIN : ti=37 (equipement) est couvert 30/31 par le dispatch. La methode doit y rendre
	// des masques propres et des traversees qui aboutissent. Si elle echoue AUSSI sur ti=37,
	// c'est la methode qui est en cause, pas ti=11 — et le resultat sur ti=11 ne vaut rien.
	witness := ti11WalkArchetype(dir, reg, 37)
	ti11ReportWall(t, reg, 37, witness, "TEMOIN")

	target := ti11WalkArchetype(dir, reg, ti11TypeIndex)
	ti11ReportWall(t, reg, ti11TypeIndex, target, "CIBLE")

	if target.records == 0 {
		t.Logf("VERDICT : aucun record ti=%d sur ce film — mesure impossible ici.", ti11TypeIndex)
		return
	}
	if target.tiMismatch == target.records {
		t.Logf("VERDICT : alignement REFUTE — TraverseEntity ne relit jamais ti=%d a Bit+%d.",
			ti11TypeIndex, keyframeRecordTIBit)
		return
	}
	firsts := ti11SortedKeys(target.firstPresent)
	if len(firsts) > 0 {
		t.Logf("VERDICT — LE MUR de ti=%d est au composant i%d (%s), premier PRESENT dans %d"+
			" records sur %d (%.1f %%). Avec une couverture de dispatch de 0/34, ce premier"+
			" present est aussi le premier NON PORTE : c'est lui qu'il faut porter, et rien"+
			" avant lui.",
			ti11TypeIndex, firsts[0], ti11CompName(reg, ti11TypeIndex, firsts[0]),
			target.firstPresent[firsts[0]], target.records,
			100*float64(target.firstPresent[firsts[0]])/float64(target.records))
	}
}

// ti11CompName rend le nom du composant i d'un archetype, ou "?" hors borne.
func ti11CompName(reg *Registry, ti, i int) string {
	arch, ok := reg.Archetype(ti)
	if !ok || i < 0 || i >= len(arch.Components) {
		return "?"
	}
	return arch.Components[i]
}

// ti11ReportWall publie toutes les grandeurs d'un balayage, avec leurs denominateurs.
func ti11ReportWall(t *testing.T, reg *Registry, ti int, s ti11WallStats, role string) {
	t.Helper()
	arch, _ := reg.Archetype(ti)
	t.Logf("%s ti=%d (%d composants) — %d records d'image-cle", role, ti, len(arch.Components), s.records)
	if s.records == 0 {
		return
	}
	pct := func(n int) float64 { return 100 * float64(n) / float64(s.records) }
	t.Logf("  alignement : ti relu != attendu %d (%.1f %%) · masque vide %d (%.1f %%)"+
		" · masque HORS GRAMMAIRE %d (%.1f %%)",
		s.tiMismatch, pct(s.tiMismatch), s.emptyMask, pct(s.emptyMask),
		s.outOfGrammar, pct(s.outOfGrammar))
	t.Logf("  traversees ABOUTIES (DesyncAt == -1) : %d / %d (%.1f %%)",
		s.completed, s.records, pct(s.completed))
	t.Logf("  nombre de composants presents par record : %s", ti11Histogram(s.popcount))
	t.Logf("  PREMIER composant present (c'est le mur quand rien n'est porte) :")
	for _, i := range ti11SortedKeys(s.firstPresent) {
		t.Logf("    i%-2d  %5d records  %5.1f %%   %s", i, s.firstPresent[i], pct(s.firstPresent[i]),
			ti11CompName(reg, ti, i))
	}
	t.Logf("  presence par composant :")
	keys := ti11SortedKeys(s.present)
	sort.Ints(keys)
	for _, i := range keys {
		t.Logf("    i%-2d  %5d  %5.1f %%   %s", i, s.present[i], pct(s.present[i]),
			ti11CompName(reg, ti, i))
	}
	t.Logf("  DesyncAt : %s", ti11Histogram(s.desyncAt))
}
