package filmdec

// keyframe_entity_queue_test.go — INSTRUMENT DE MESURE du lot R6
// (cf. `.ai/V7.5/replay2d/PLAN_R6_FILE_PAR_ENTITE.md`).
//
// CE QU'IL MESURE :
//
//	TestKFQReconcile (C0) — le tampon capture en RAM en juillet
//	  (`.ai/V7.5/dumps/keyframe_buffer_live.bin`, etiquete « keyframe ») correspond a QUEL
//	  objet du film ? Coincidence exacte d'abord, sinon coincidence de PREFIXE — et la
//	  reponse est publiee avec son denominateur (nombre de films balayes).
//
//	TestKFQFirstDelta (C1, C2) — la boucle de records PORTEE traverse-t-elle le PREMIER
//	  paquet de type 0 d'une session, et quels archetypes lie-t-elle ? La largeur d'id est
//	  BALAYEE (valeur de runtime absente du film) et la retenue est publiee.
//
//	TestKFQAnchorShape (H2) — la table type-2 est-elle un bitstream de records ou une table
//	  d'octets ? Mesure de l'alignement des ancres et des ecarts modulo 8.
//
// Il ne PUBLIE rien dans l'artefact : il rend des taux et leurs denominateurs.
//
// LECTURE SEULE, garde par KFQ_FILM (et KFQ_ROOT pour C0), saute partout ailleurs (CI
// comprise). UN SEUL decodage filmdec par process : un seul test a la fois.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KFQ_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestKFQ' -timeout 30m -v

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	kfqFilmEnv = "KFQ_FILM"
	kfqRootEnv = "KFQ_ROOT"
	kfqDumpEnv = "KFQ_DUMP"
)

// kfqFilmDir rend le repertoire de film de la garde, ou saute le test.
func kfqFilmDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(kfqFilmEnv)
	if dir == "" {
		t.Skipf("%s non defini : instrument de mesure, saute", kfqFilmEnv)
	}
	return dir
}

// kfqRegistry charge le registre d'archetypes du film.
func kfqRegistry(t *testing.T, dir string) *Registry {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible dans %s : %v", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible dans %s : %v", dir, err)
	}
	return reg
}

// TestKFQReconcile (C0) reconcilie un tampon capture avec un paquet de film.
func TestKFQReconcile(t *testing.T) {
	root := os.Getenv(kfqRootEnv)
	dump := os.Getenv(kfqDumpEnv)
	if root == "" || dump == "" {
		t.Skipf("%s / %s non definis : instrument de mesure, saute", kfqRootEnv, kfqDumpEnv)
	}
	want, err := os.ReadFile(dump)
	if err != nil {
		t.Fatalf("dump illisible %s : %v", dump, err)
	}
	films, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("racine illisible %s : %v", root, err)
	}
	t.Logf("dump %s : %d octets ; denominateur : %d entrees sous %s",
		filepath.Base(dump), len(want), len(films), root)

	const prefixLen = 16
	if len(want) < prefixLen {
		t.Fatalf("dump trop court pour un prefixe de %d octets", prefixLen)
	}
	res, err := FindPackets(root, []func([]byte) bool{
		KFQEqual(want), KFQPrefix(want[:prefixLen]),
	})
	if err != nil {
		t.Fatalf("balayage : %v", err)
	}
	exact, pre := res[0], res[1]

	t.Logf("coincidence EXACTE : %d paquet(s)", len(exact))
	for _, r := range exact {
		t.Logf("  film=%s chunk=%d paquet=#%d type=%d taille=%d",
			r.Film, r.Chunk, r.Packet.Index, r.Packet.Type, r.Packet.Size)
	}

	byType := map[uint16]int{}
	byIndex := map[int]int{}
	for _, r := range pre {
		byType[r.Packet.Type]++
		byIndex[r.Packet.Index]++
	}
	t.Logf("coincidence de PREFIXE (%d octets) : %d paquet(s) ; par type %v ; par rang %v",
		prefixLen, len(pre), byType, byIndex)
}

// TestKFQFirstDelta (C1, C2) mesure la traversee du premier paquet type-0 d'une session.
func TestKFQFirstDelta(t *testing.T) {
	dir := kfqFilmDir(t)
	reg := kfqRegistry(t, dir)

	chunk, pk, cn, err := FirstPacketOfType(dir, PacketTypeDelta)
	if err != nil {
		t.Fatalf("premier paquet type-0 introuvable : %v", err)
	}
	pay := pk.Payload(chunk)
	t.Logf("premier paquet type-0 : chunk_%02d paquet #%d taille=%d octets (%d bits)",
		cn, pk.Index, pk.Size, pk.Size*8)

	vs := KFQFrameVariants(10, 14)
	best, all := BestVariant(reg, pay, vs, nil)
	for _, w := range all {
		if w.Records < 3 || w.Overrun {
			continue // bruit : moins de 3 records, ou marche qui deborde du tampon
		}
		t.Logf("  idLow=%2d amorce=%d extra=%-5v : records=%4d NEW=%4d (propres %4d) "+
			"DEL=%3d DELTA=%4d bits %6d/%6d (%.1f%%) arret=%q",
			w.Variant.IDLowBits, w.Variant.Preamble, w.Variant.ExtraFields,
			w.Records, w.New, w.CleanNew, w.Del, w.Delta,
			w.EndBit, w.TotalBits, 100*w.Coverage(), w.Stop)
	}
	t.Logf("RETENU idLow=%d amorce=%d extra=%v : couverture %.2f%% (%d/%d bits), %d NEW propres, %d combinaisons probees",
		best.Variant.IDLowBits, best.Variant.Preamble, best.Variant.ExtraFields,
		100*best.Coverage(), best.EndBit, best.TotalBits, best.CleanNew, len(vs))
	for _, ti := range best.SortedTIs() {
		t.Logf("    ti=%-3d x%d", ti, best.ByTI[uint32(ti)])
	}

	// Contre-liste INDEPENDANTE : les archetypes que le balayeur de la table type-2 rend
	// sur le MEME film. Les deux chaines ne partagent aucun deserialiseur.
	kchunks, kpkts, kidx := AllPacketsOfType(dir, PacketTypeKeyframe)
	var seedPay []byte
	for i, kpk := range kpkts {
		kByTI := map[int]int{}
		recs := WalkKeyframeWorld(kpk.Payload(kchunks[i]))
		for _, r := range recs {
			kByTI[r.TI]++
		}
		t.Logf("contre-liste type-2 #%d (chunk_%02d, %d o) : %d records, %d archetypes, "+
			"ti=11 x%d ti=37 x%d ti=42 x%d",
			kpk.Index, kidx[i], kpk.Size, len(recs), len(kByTI),
			kByTI[11], kByTI[37], kByTI[42])
		if len(recs) > len(seedPay) {
			seedPay = kpk.Payload(kchunks[i])
		}
	}

	// H4 : le premier paquet type-0 suppose-t-il un World deja peuple ? On rejoue la meme
	// marche avec le World issu de la table type-2 (chaine disjointe).
	if seedPay == nil {
		t.Logf("H4 sans objet : aucun paquet type-2")
		return
	}
	bestSeed, _ := BestVariant(reg, pay, vs,
		func() *World { return WorldFromKeyframe(reg, seedPay) })
	t.Logf("H4 RETENU idLow=%d amorce=%d extra=%v : couverture %.2f%% avec World pre-peuple, "+
		"contre %.2f%% sans", bestSeed.Variant.IDLowBits, bestSeed.Variant.Preamble,
		bestSeed.Variant.ExtraFields, 100*bestSeed.Coverage(), 100*best.Coverage())
}

// TestKFQAnchorShape (H2) mesure la forme des ancres de la table type-2.
func TestKFQAnchorShape(t *testing.T) {
	dir := kfqFilmDir(t)
	chunk, pk, cn, err := FirstPacketOfType(dir, PacketTypeKeyframe)
	if err != nil {
		t.Fatalf("paquet type-2 introuvable : %v", err)
	}
	sh := MeasureKeyframeAnchors(pk.Payload(chunk))
	t.Logf("table type-2 : chunk_%02d paquet #%d taille=%d octets ; %d ancres ; "+
		"%d alignees sur l'octet", cn, pk.Index, pk.Size, sh.Records, sh.BitAligned)
	t.Logf("  ecarts modulo 8 : %v", sh.GapMod8)
	t.Logf("  ecarts distincts : %d valeurs sur %d ecarts",
		len(sh.GapValues), max(0, sh.Records-1))
}
