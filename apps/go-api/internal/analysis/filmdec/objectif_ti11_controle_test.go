package filmdec

// objectif_ti11_controle_test.go — LE CONTROLE QUI A RECADRE LE CHANTIER.
//
// Passer le MEME instrument sur `ti=13` — l'archetype lu juste depuis le lot C-bis — etait le
// controle qui manquait. Il a montre que le repere de « 87 a 99 % » auquel ce chantier se
// comparait n'est pas reproductible : sur ses propres modes, `ti=13` plafonne a 77 %.
//
// Le bilan complet est dans l'en-tete d'`objectif_ti11_delta_test.go`.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// TestObjectifTi11DeltaControleTi13 — LE MEME INSTRUMENT SUR L'ARCHETYPE QUI MARCHE.
//
// # POURQUOI CE CONTROLE DECIDE DE LA SUITE
//
// `ti=13` est lu juste depuis le lot C-bis : son balayage de production annonce 87 a 99 % de
// chainage, et ses valeurs sont exploitees en production. Il partage avec `ti=11` le MEME lecteur
// d'ancre (`matchWorldObjectRecord`), le meme test de fin (`worldObjectHeaderAt`), la meme boucle
// de composants. La seule chose qui change est l'archetype.
//
// Passer le MEME instrument sur `ti=13` separe donc deux mondes :
//
//	ti=13 rend ~90 %   l'instrument est bon, l'ancre est bonne, et ce qui cloche chez `ti=11`
//	                   est dans SES composants — malgre les lecteurs du jeu qui les confirment ;
//	ti=13 rend ~30 %   l'instrument n'est pas celui qui a produit le 87-99 %, et la comparaison
//	                   qui guide ce chantier depuis le debut n'a pas de sens.
//
// Sans ce controle, tout ce qui precede se compare a un chiffre venu d'ailleurs.
//
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11DeltaControleTi13 -v -timeout 40m
func TestObjectifTi11DeltaControleTi13(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11DeltaControleTi13", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	for _, cas := range []struct {
		ti       int
		observee bool
	}{
		{ObjectiveTypeIndex, false}, {ObjectiveTypeIndex, true},
		{ManagedPropertyTypeIndex, false}, {ManagedPropertyTypeIndex, true},
	} {
		ti, observee := cas.ti, cas.observee
		var b ti11DeltaBilan
		films := 0
		for _, f := range ti11Corpus {
			dir := filepath.Join(cache, "film_chunks", f.id)
			n := CountFilmChunks(dir)
			if n == 0 {
				continue
			}
			raw, err := ReadFilmChunk(dir, 0)
			if err != nil {
				continue
			}
			reg, err := ParseRegistryChunk(raw)
			if err != nil {
				continue
			}
			arch, ok := reg.Archetype(ti)
			if !ok {
				continue
			}
			band := worldObjectSlotBand(dir, n, ti)
			if observee {
				band = ti11SlotSetPour(dir, n, ti)
			}
			if len(band) == 0 {
				continue
			}
			films++
			avant, avantC := b.marches, b.chaines
			for c := 1; c <= n; c++ {
				data, err := ReadFilmChunk(dir, c)
				if err != nil {
					continue
				}
				for _, pk := range WalkPackets(data) {
					if pk.Type == PacketTypeDelta {
						ti11ControlePayload(pk.Payload(data), band, arch, ti, &b)
					}
				}
			}
			t.Logf("  ti=%-3d bande=%-9s %-9s %-26s %4d slots, %7d marche(s), %5.1f %% chainees",
				ti, ti11NomBande(observee), f.id, f.mode, len(band), b.marches-avant,
				ti11Part(b.chaines-avantC, b.marches-avant))
		}
		t.Logf("ti=%-3d bande=%-9s TOTAL %2d film(s), %7d marche(s), %5.1f %% chainees",
			ti, ti11NomBande(observee), films, b.marches, ti11Part(b.chaines, b.marches))
	}
}

// ti11ControlePayload est `ti11DeltaPayload` generalise a un archetype quelconque, sans la
// ventilation par composant (qui n'aurait pas de sens hors de `ti=11`).
func ti11ControlePayload(pay []byte, band map[uint32]bool, arch Archetype, ti int, b *ti11DeltaBilan) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || !ti11IdxDansDomaine(rec.Idx, len(arch.Components)) {
			continue
		}
		at, done := rec.After, true
		for _, id := range rec.Idx {
			name := arch.component(id)
			if name == "" || at > total {
				done = false
				break
			}
			br := NewBitReader(pay)
			br.SetBitPos(at)
			_, _, ported := consumeByName(br, name, uint32(ti), arch.Level(id))
			if !ported || br.BitPos() > total {
				done = false
				break
			}
			at = br.BitPos()
		}
		p = rec.After
		if !done {
			continue
		}
		b.marches++
		if worldObjectHeaderAt(pay, at) {
			b.chaines++
			if n := len(rec.Idx); n < len(b.tailles) {
				b.taillesChaine[n]++
			}
		}
		if n := len(rec.Idx); n < len(b.tailles) {
			b.tailles[n]++
		}
	}
}

// ti11NomBande nomme les deux bandes comparees par le controle : celle de PRODUCTION, qui comble
// tout l'intervalle des slots, et celle des slots REELLEMENT OBSERVES.
func ti11NomBande(observee bool) string {
	if observee {
		return "observee"
	}
	return "comblee"
}

// ti11SlotSetPour est `objectiveSlotSet` generalise a un archetype quelconque.
func ti11SlotSetPour(dir string, n, ti int) map[uint32]bool {
	seen, others := map[uint32]bool{}, map[uint32]bool{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI == ti {
					seen[uint32(r.Slot)] = true
					continue
				}
				others[uint32(r.Slot)] = true
			}
		}
	}
	for s := range others {
		delete(seen, s)
	}
	return seen
}
