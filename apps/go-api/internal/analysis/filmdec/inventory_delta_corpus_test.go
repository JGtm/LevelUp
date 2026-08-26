package filmdec

// inventory_delta_corpus_test.go — L'INSTRUMENT DE CORPUS du suivi delta de l'inventaire.
//
// LA QUESTION QU'IL TRANCHE : la faisabilité du suivi delta a été établie sur UN SEUL FILM
// (000d5950). C'est la réserve nommée par l'étude du 2026-08-24 §2.6. Cet instrument rejoue
// ScanFilmInventoryDeltas sur N films du cache et publie, film par film, le taux de lectures
// PLAUSIBLES d'i22 et la conformité masque/sélection d'i47 — les deux tests réfutables que le
// scanner porte.
//
// CE QU'IL N'EST PAS : un test de CI. Il exige les films (des gigaoctets hors dépôt) et il
// est gardé par INV_DELTA_FILMS. Les garde-rails qui valent en CI sont dans
// inventory_delta_test.go.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 INV_DELTA_FILMS=<repo>/data/cache/film_chunks INV_DELTA_MAX=15 \
//	  go test ./internal/analysis/filmdec/ -run '^TestInventoryDeltaCorpus$' -timeout 120m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	invDeltaFilmsEnv = "INV_DELTA_FILMS"
	invDeltaMaxEnv   = "INV_DELTA_MAX"
)

// invDeltaCorpusFilms rend les répertoires de film du cache, triés par nom (l'ordre est
// stable d'une exécution à l'autre : une mesure qui change de corpus ne se compare pas).
func invDeltaCorpusFilms(t *testing.T, root string, max int) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("lecture du cache de films %s : %v", root, err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if CountFilmChunks(dir) > 0 {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	if max > 0 && len(dirs) > max {
		dirs = dirs[:max]
	}
	return dirs
}

func TestInventoryDeltaCorpus(t *testing.T) {
	root := os.Getenv(invDeltaFilmsEnv)
	if root == "" {
		t.Skipf("%s non défini : instrument de corpus sauté", invDeltaFilmsEnv)
	}
	release := LockProcessDecode()
	defer release()

	films := invDeltaCorpusFilms(t, root, invDeltaEnvInt(invDeltaMaxEnv, 0))
	if len(films) == 0 {
		t.Fatalf("aucun film exploitable dans %s", root)
	}
	t.Logf("corpus : %d films", len(films))

	var tot InventoryDeltaStats
	var totEmitted int
	var mags, reserves []uint32
	var magBySlot, resBySlot [invDeltaWeaponSlots][]uint32
	filmsAmmoRefused := 0
	for _, dir := range films {
		out, st, err := ScanFilmInventoryDeltas(dir)
		if err != nil {
			t.Logf("%-12s  ECHEC : %v", filepath.Base(dir), err)
			continue
		}
		tot.Records += st.Records
		tot.WithI22 += st.WithI22
		tot.WithI47 += st.WithI47
		tot.I22Read += st.I22Read
		tot.I22Unread += st.I22Unread
		tot.I47Read += st.I47Read
		tot.I47Unread += st.I47Unread
		tot.Implausible += st.Implausible
		tot.NoSelection += st.NoSelection
		tot.MaskEmpty += st.MaskEmpty
		tot.SelOutsideMask += st.SelOutsideMask
		tot.AccordChecked += st.AccordChecked
		tot.Accord += st.Accord
		tot.WithAmmo += st.WithAmmo
		tot.WithRounds += st.WithRounds
		tot.AmmoRead += st.AmmoRead
		tot.RoundsRead += st.RoundsRead
		tot.MagRead += st.MagRead
		tot.MagOutOfEnvelope += st.MagOutOfEnvelope
		tot.MagCorroborated += st.MagCorroborated
		tot.MagOutOfEnvelopeCorroborated += st.MagOutOfEnvelopeCorroborated
		tot.ResOutOfEnvelope += st.ResOutOfEnvelope
		if st.AmmoRefused {
			filmsAmmoRefused++
		}
		var filmMags []uint32
		for _, r := range out {
			for _, a := range r.Ammo {
				if a.Mag != nil {
					mags = append(mags, *a.Mag)
					filmMags = append(filmMags, *a.Mag)
					magBySlot[a.WeaponSlot] = append(magBySlot[a.WeaponSlot], *a.Mag)
				}
				if a.Res != nil {
					reserves = append(reserves, *a.Res)
					resBySlot[a.WeaponSlot] = append(resBySlot[a.WeaponSlot], *a.Res)
				}
			}
		}
		t.Logf("%-12s  CHARGEUR %s | horsEnveloppe %d/%d = %.2f %%%s",
			filepath.Base(dir), invDeltaQuantiles(filmMags),
			st.MagOutOfEnvelope, st.MagRead, invDeltaPct(st.MagOutOfEnvelope, st.MagRead),
			invRefusalMark(st.AmmoRefused))
		totEmitted += len(out)
		t.Logf("%-12s  records=%-8d | i22 masque=%-5d lues=%-5d implausibles=%-4d (%5.1f %%) | "+
			"i47 masque=%-5d sansSel=%-4d masqueVide=%-4d horsMasque=%-4d | accord i22^i47 %d/%d | publiees=%d",
			filepath.Base(dir), st.Records, st.WithI22, st.I22Read, st.Implausible,
			invDeltaPct(st.I22Read-st.Implausible, st.I22Read),
			st.WithI47, st.NoSelection, st.MaskEmpty, st.SelOutsideMask,
			st.Accord, st.AccordChecked, len(out))
	}
	t.Logf("TOTAL  records=%d | i22 lues=%d implausibles=%d -> %.2f %% plausibles | "+
		"i47 lues=%d masqueVide=%d horsMasque=%d -> %.2f %% conformes | "+
		"ACCORD i22^i47 %d/%d = %.2f %% | publiees=%d",
		tot.Records, tot.I22Read, tot.Implausible,
		invDeltaPct(tot.I22Read-tot.Implausible, tot.I22Read),
		tot.I47Read, tot.MaskEmpty, tot.SelOutsideMask,
		invDeltaPct(tot.I47Read-tot.MaskEmpty-tot.SelOutsideMask, tot.I47Read-tot.MaskEmpty),
		tot.Accord, tot.AccordChecked, invDeltaPct(tot.Accord, tot.AccordChecked),
		totEmitted)

	t.Logf("MUNITIONS  masqueAmmo=%d lues=%d dont chargeur ecrit=%d horsEnveloppe=%d | "+
		"masqueRounds=%d lues=%d horsEnveloppe=%d",
		tot.WithAmmo, tot.AmmoRead, tot.MagRead, tot.MagOutOfEnvelope,
		tot.WithRounds, tot.RoundsRead, tot.ResOutOfEnvelope)
	t.Logf("CORROBORATION  chargeurs lus=%d dont corrobores par i22=%d ; hors enveloppe=%d dont "+
		"corrobores=%d -> taux corrobore %.2f %% contre non corrobore %.2f %%",
		tot.MagRead, tot.MagCorroborated, tot.MagOutOfEnvelope, tot.MagOutOfEnvelopeCorroborated,
		invDeltaPct(tot.MagOutOfEnvelopeCorroborated, tot.MagCorroborated),
		invDeltaPct(tot.MagOutOfEnvelope-tot.MagOutOfEnvelopeCorroborated, tot.MagRead-tot.MagCorroborated))
	t.Logf("chargeur R(8)  : %s", invDeltaQuantiles(mags))
	t.Logf("reserve  R(11) : %s", invDeltaQuantiles(reserves))
	for k := 0; k < invDeltaWeaponSlots; k++ {
		t.Logf("  emplacement %d  chargeur %s", k, invDeltaQuantiles(magBySlot[k]))
		t.Logf("  emplacement %d  reserve  %s", k, invDeltaQuantiles(resBySlot[k]))
	}

	// LA CALIBRATION EST LE TEST : un curseur mal placé produirait une loi UNIFORME sur toute
	// la plage du champ. On vérifie donc que la distribution N'EST PAS uniforme — la médiane
	// doit rester loin sous le milieu du champ (128 pour un R(8), 1024 pour un R(11)).
	if len(mags) > 100 {
		if m := invDeltaMedian(mags); m >= 128 {
			t.Fatalf("chargeur : médiane %d >= 128 — la distribution remplit le champ R(8), "+
				"signature d'un curseur perdu", m)
		}
	}
	if len(reserves) > 50 {
		if m := invDeltaMedian(reserves); m >= 1024 {
			t.Fatalf("réserve : médiane %d >= 1024 — la distribution remplit le champ R(11)", m)
		}
	}
	// LE CANAL MUNITIONS SE JUGE PAR FILM, PAS EN MOYENNE — c'est la découverte du lot 4.2.
	// `mags` ne contient QUE les films retenus (un film refusé voit ses munitions retirées de
	// la sortie), donc son maximum est borné par l'enveloppe par construction. Ce qui se juge
	// ici, c'est la PART DU CORPUS que la porte laisse passer : si elle s'effondre, le canal
	// munitions n'est pas exploitable et il ne faut pas le fusionner.
	t.Logf("PORTE MUNITIONS  %d films sur %d refusés en bloc (distribution de chargeurs "+
		"contaminée) -> canal munitions exploitable sur %.0f %% du corpus",
		filmsAmmoRefused, len(films), invDeltaPct(len(films)-filmsAmmoRefused, len(films)))
	if len(films) > 0 && filmsAmmoRefused*2 > len(films) {
		t.Fatalf("canal munitions refusé sur %d films sur %d (plus de la moitié) — "+
			"NE PAS le fusionner", filmsAmmoRefused, len(films))
	}
	if len(mags) > 0 {
		if m := invDeltaQuantiles(mags); invDeltaMedian(mags) == 0 {
			t.Fatalf("films retenus : distribution de chargeurs dégénérée (%s)", m)
		}
	}
	if tot.RoundsRead > 0 {
		if p := invDeltaPct(tot.ResOutOfEnvelope, tot.RoundsRead); p > 1.0 {
			t.Fatalf("réserve : %.2f %% de lectures hors enveloppe (> 1 %%)", p)
		}
	}

	// LES SEUILS SONT CEUX DU PLAN (lot 1 : « 100 % plausibles »), assouplis d'un point pour
	// laisser passer le bruit attendu d'un balayage bit à bit sur un corpus large. En dessous,
	// la faisabilité multi-films est RÉFUTÉE et la fusion ne doit pas être livrée.
	if tot.I22Read > 0 {
		if p := invDeltaPct(tot.I22Read-tot.Implausible, tot.I22Read); p < 99.0 {
			t.Fatalf("i22 : %.2f %% de lectures plausibles sur le corpus (< 99 %%) — "+
				"le curseur n'est pas à sa place sur une part significative des films", p)
		}
	}
	if n := tot.I47Read - tot.MaskEmpty; n > 0 {
		if p := invDeltaPct(n-tot.SelOutsideMask, n); p < 99.0 {
			t.Fatalf("i47 : %.2f %% de sélections dans un masque non vide (< 99 %%)", p)
		}
	}
	// LE CONTRÔLE CROISÉ : deux désers, deux positions, une seule réponse attendue.
	if tot.AccordChecked > 0 {
		if p := invDeltaPct(tot.Accord, tot.AccordChecked); p < 99.0 {
			t.Fatalf("accord i22^i47 : %.2f %% (< 99 %%) — les deux composants ne décrivent "+
				"pas le même inventaire, donc au moins un curseur est mal placé", p)
		}
	}
}

func invDeltaPct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

func invDeltaEnvInt(name string, def int) int {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// invDeltaQuantiles décrit une distribution — c'est ce résumé, et non une valeur isolée, qui
// dit si un curseur est à sa place.
func invDeltaQuantiles(v []uint32) string {
	if len(v) == 0 {
		return "(vide)"
	}
	c := append([]uint32(nil), v...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	q := func(f float64) uint32 { return c[int(f*float64(len(c)-1))] }
	distinct := map[uint32]bool{}
	for _, x := range c {
		distinct[x] = true
	}
	return fmt.Sprintf("n=%d min=%d p10=%d p50=%d p90=%d max=%d distinctes=%d",
		len(c), c[0], q(0.1), q(0.5), q(0.9), c[len(c)-1], len(distinct))
}

func invDeltaMedian(v []uint32) uint32 {
	if len(v) == 0 {
		return 0
	}
	c := append([]uint32(nil), v...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[len(c)/2]
}

// invRefusalMark rend visible, sur la ligne du film, le refus en bloc du canal munitions.
func invRefusalMark(refused bool) string {
	if refused {
		return "  <- CANAL MUNITIONS REFUSE"
	}
	return ""
}
