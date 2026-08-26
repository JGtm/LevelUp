package replay

// inventory_delta_corpus_test.go — LA CONFRONTATION DES DEUX CANAUX D'INVENTAIRE, sur corpus.
//
// LA QUESTION QU'ELLE TRANCHE, et c'est LE verrou du chantier : les compteurs de grenades lus
// dans les paquets DELTA (filmdec.ScanFilmInventoryDeltas) disent-ils la même chose que ceux
// lus aux IMAGES-CLÉS (ScanFilmKeyframeInventory) ? Les deux canaux n'ont AUCUNE étape
// commune — motif d'ancrage et fenêtres de bits dans les images-clés, marche de composants par
// les désers de production dans les deltas. S'ils divergent, l'un des deux lit à côté et la
// fusion est morte ; s'ils concordent, les deltas rafraîchissent légitimement entre deux
// images-clés.
//
// LA RÈGLE RÉFUTABLE : pour chaque image-clé dont les grenades sont lues, la DERNIÈRE lecture
// delta antérieure du même slot doit donner le même quadruplet. Un désaccord est soit une
// erreur de lecture, soit un changement survenu entre les deux et non transmis (défaut de
// RAPPEL de l'ancre, cf. inventory_delta_rappel_test.go) — l'instrument publie l'écart de
// temps de chaque divergence, qui est ce qui les sépare.
//
// L'instrument mesure AUSSI le gain de fraîcheur, qui est la raison d'être du chantier : âge
// médian de la dernière lecture connue, images-clés seules puis deux canaux fusionnés.
//
// CE QU'IL N'EST PAS : un test de CI. Il exige les films (des gigaoctets hors dépôt) et il est
// gardé par INV_DELTA_FILMS.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 INV_DELTA_FILMS=<repo>/data/cache/film_chunks INV_DELTA_MAX=25 \
//	  go test ./internal/analysis/replay/ -run '^TestInventoryDeltaConfrontationCorpus$' -timeout 120m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	invDeltaFilmsEnv = "INV_DELTA_FILMS"
	invDeltaMaxEnv   = "INV_DELTA_MAX"
	// invDeltaConcordanceFloor est le plancher de concordance du corpus. Il n'est pas choisi
	// au doigt mouillé : l'étude du 2026-08-24 mesure 97,2 % sur le film témoin et attribue
	// l'unique divergence à un ramassage de 3,3 s non transmis — un défaut de rappel, pas de
	// justesse. En dessous de 95 %, la faisabilité multi-films est RÉFUTÉE.
	invDeltaConcordanceFloor = 95.0
)

// invCorpusFilms rend les répertoires de film du cache, triés par nom.
func invCorpusFilms(t *testing.T, root string, max int) []string {
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
		if filmdec.CountFilmChunks(dir) > 0 {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	if max > 0 && len(dirs) > max {
		dirs = dirs[:max]
	}
	return dirs
}

// invConfrontation est le bilan d'un film (ou du corpus, par sommation).
type invConfrontation struct {
	// Keyframes est le nombre d'images-clés dont les grenades sont lues — le dénominateur
	// naturel. WithPrior en est la part qu'un delta antérieur du même slot précède.
	Keyframes, WithPrior int
	// Agree est le nombre de couples concordants ; Disagree son complément.
	Agree, Disagree int
	// SelChecked / SelAgree : la même confrontation sur le rang SÉLECTIONNÉ (i47), quand les
	// deux canaux le publient.
	SelChecked, SelAgree int
	// Between est le nombre de lectures delta tombant STRICTEMENT entre deux images-clés du
	// même slot — l'information réellement neuve, celle que la fusion apporte.
	Between, Deltas int
	// MagChecked / MagAgree : la confrontation des CHARGEURS, restreinte aux couples dont
	// l'écart de temps est court (cf. invAmmoWindowUS).
	//
	// POURQUOI UNE FENÊTRE ICI ET PAS POUR LES GRENADES. Un compteur de grenades change
	// quelques fois par vie ; un chargeur change à CHAQUE tir. Confronter une lecture delta
	// vieille de 10 s au chargeur d'une image-clé mesurerait le rythme de tir du joueur, pas
	// la justesse de la lecture. La fenêtre courte est ce qui rend le test réfutable.
	MagChecked, MagAgree int
	// MagByWindow : la MEME confrontation a plusieurs ecarts de temps. C'est ce PROFIL, et non
	// un taux isole, qui dit si un desaccord est une erreur de lecture ou une consommation
	// reelle survenue entre les deux lectures. Une erreur de lecture ne s'ameliore pas quand
	// on rapproche les deux mesures ; une consommation, si.
	MagByWindow [len(invAmmoWindowsUS)]invPair
	// TransReal / TransCaught : LE RAPPEL DE L'ANCRE (lot 4.3). Une TRANSITION REELLE est un
	// changement de quadruplet de grenades entre deux images-cles consecutives du meme slot —
	// le canal de production l'atteste, independamment des deltas. La transition est CAPTUREE
	// si une lecture delta du meme slot, tombant dans l'intervalle, rend deja le nouveau
	// quadruplet. Le rapport des deux est la part des changements que l'ancre voit passer.
	TransReal, TransCaught int
	// FilmsAmmoRefused compte les films dont le canal munitions a été refusé en bloc par la
	// porte du scanner (distribution de chargeurs contaminée).
	FilmsAmmoRefused int
}

// invAmmoWindowUS borne l'écart de temps d'un couple confrontable sur les chargeurs. 1 s :
// assez court pour qu'un joueur n'ait pas vidé son chargeur entre les deux lectures, assez
// long pour que des couples existent.
const invAmmoWindowUS = 1_000_000

// invAmmoWindowsUS sont les ecarts auxquels la confrontation est refaite, du plus serre au plus
// large. Le PROFIL est le test : s'il monte quand la fenetre se resserre, les deltas lisent
// juste et le desaccord est du tir.
var invAmmoWindowsUS = [5]uint64{100_000, 250_000, 500_000, 1_000_000, 2_000_000}

// invPair est un couple (verifies, concordants).
type invPair struct{ n, ok int }

func (c *invConfrontation) add(o invConfrontation) {
	c.Keyframes += o.Keyframes
	c.WithPrior += o.WithPrior
	c.Agree += o.Agree
	c.Disagree += o.Disagree
	c.SelChecked += o.SelChecked
	c.SelAgree += o.SelAgree
	c.Between += o.Between
	c.Deltas += o.Deltas
	c.MagChecked += o.MagChecked
	c.MagAgree += o.MagAgree
	for i := range c.MagByWindow {
		c.MagByWindow[i].n += o.MagByWindow[i].n
		c.MagByWindow[i].ok += o.MagByWindow[i].ok
	}
	c.FilmsAmmoRefused += o.FilmsAmmoRefused
	c.TransReal += o.TransReal
	c.TransCaught += o.TransCaught
}

func TestInventoryDeltaConfrontationCorpus(t *testing.T) {
	root := os.Getenv(invDeltaFilmsEnv)
	if root == "" {
		t.Skipf("%s non défini : instrument de corpus sauté", invDeltaFilmsEnv)
	}
	films := invCorpusFilms(t, root, invEnvInt(invDeltaMaxEnv, 0))
	if len(films) == 0 {
		t.Fatalf("aucun film exploitable dans %s", root)
	}
	t.Logf("corpus : %d films", len(films))

	var tot invConfrontation
	var ageKFAll, ageMergedAll []float64
	for _, dir := range films {
		c, ageKF, ageMerged, err := invConfrontOneFilm(dir)
		if err != nil {
			t.Logf("%-12s  ECHEC : %v", filepath.Base(dir), err)
			continue
		}
		tot.add(c)
		ageKFAll = append(ageKFAll, ageKF...)
		ageMergedAll = append(ageMergedAll, ageMerged...)
		t.Logf("%-12s  kf=%-5d avecDeltaAnterieur=%-5d concordance=%-4d/%-4d (%5.1f %%) | "+
			"sel %d/%d | deltas=%-5d dont entre deux kf=%-5d | age median kf=%.2f s -> fusion=%.2f s",
			filepath.Base(dir), c.Keyframes, c.WithPrior, c.Agree, c.WithPrior,
			invPct(c.Agree, c.WithPrior), c.SelAgree, c.SelChecked, c.Deltas, c.Between,
			medianOf(ageKF), medianOf(ageMerged))
	}
	t.Logf("TOTAL  kf=%d | avecDeltaAnterieur=%d | CONCORDANCE %d/%d = %.2f %% | "+
		"selection %d/%d = %.2f %% | deltas=%d dont %d entre deux kf | "+
		"AGE MEDIAN images-cles seules %.2f s -> fusionne %.2f s (%.1f %%)",
		tot.Keyframes, tot.WithPrior, tot.Agree, tot.WithPrior, invPct(tot.Agree, tot.WithPrior),
		tot.SelAgree, tot.SelChecked, invPct(tot.SelAgree, tot.SelChecked),
		tot.Deltas, tot.Between,
		medianOf(ageKFAll), medianOf(ageMergedAll),
		-100*(1-medianOf(ageMergedAll)/maxF(medianOf(ageKFAll), 1e-9)))

	if tot.WithPrior == 0 {
		t.Fatal("aucun couple confrontable sur le corpus : la mesure n'a pas eu lieu")
	}
	t.Logf("MUNITIONS  chargeurs confrontables (ecart <= %.0f s) = %d | CONCORDANCE %d/%d = %.2f %% | "+
		"films au canal munitions refuse = %d / %d",
		float64(invAmmoWindowUS)/1e6, tot.MagChecked, tot.MagAgree, tot.MagChecked,
		invPct(tot.MagAgree, tot.MagChecked), tot.FilmsAmmoRefused, len(films))

	t.Logf("RAPPEL DE L'ANCRE  transitions attestees par les images-cles = %d | "+
		"capturees par un delta = %d -> RAPPEL %.2f %%",
		tot.TransReal, tot.TransCaught, invPct(tot.TransCaught, tot.TransReal))

	for i, w := range invAmmoWindowsUS {
		b := tot.MagByWindow[i]
		t.Logf("  chargeur, ecart <= %5.2f s : %d / %d = %.2f %%",
			float64(w)/1e6, b.ok, b.n, invPct(b.ok, b.n))
	}

	if p := invPct(tot.Agree, tot.WithPrior); p < invDeltaConcordanceFloor {
		t.Fatalf("concordance de corpus %.2f %% (< %.1f %%) — les deux canaux ne décrivent pas "+
			"le même inventaire : NE PAS livrer la fusion", p, invDeltaConcordanceFloor)
	}
}

// invConfrontOneFilm décode les deux canaux d'un film et les confronte.
func invConfrontOneFilm(dir string) (invConfrontation, []float64, []float64, error) {
	var c invConfrontation
	release := filmdec.LockProcessDecode()
	defer release()

	kf, _, err := ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0)
	if err != nil {
		return c, nil, nil, fmt.Errorf("images-clés : %w", err)
	}
	deltas, dst, err := filmdec.ScanFilmInventoryDeltas(dir)
	if err != nil {
		return c, nil, nil, fmt.Errorf("deltas : %w", err)
	}
	c.Deltas = len(deltas)
	if dst.AmmoRefused {
		c.FilmsAmmoRefused = 1
	}

	bySlot := map[uint32][]filmdec.InventoryDelta{}
	for _, d := range deltas {
		bySlot[d.Slot] = append(bySlot[d.Slot], d)
	}
	for s := range bySlot {
		seq := bySlot[s]
		sort.SliceStable(seq, func(i, j int) bool { return seq[i].TimestampUS < seq[j].TimestampUS })
	}

	for _, k := range kf {
		if !k.GrenadesRead {
			continue
		}
		c.Keyframes++
		prior, ok := invLastDeltaBefore(bySlot[k.Slot], k.TimestampUS)
		if !ok {
			continue
		}
		c.WithPrior++
		if invSameGrenades(prior.Grenades, k.Grenades) {
			c.Agree++
		} else {
			c.Disagree++
		}
		if prior.SelRead && prior.Sel != filmdec.InventoryDeltaNoSel && k.SelectedGrenadeRank >= 0 {
			c.SelChecked++
			if prior.Sel == k.SelectedGrenadeRank {
				c.SelAgree++
			}
		}
	}
	for _, k := range kf {
		if !k.AmmoRead {
			continue
		}
		invConfrontAmmo(&c, bySlot[k.Slot], k)
	}

	invRecall(&c, kf, bySlot)
	ageKF, ageMerged, between := invFreshness(kf, bySlot)
	c.Between = between
	return c, ageKF, ageMerged, nil
}

// invLastDeltaBefore rend la dernière lecture delta PORTEUSE DE COMPTEURS antérieure à `at`.
// Une lecture qui ne porte qu'i47 ne dit rien des compteurs : la retenir masquerait la vraie
// dernière mesure.
func invLastDeltaBefore(seq []filmdec.InventoryDelta, at uint64) (filmdec.InventoryDelta, bool) {
	var out filmdec.InventoryDelta
	found := false
	for _, d := range seq {
		if d.TimestampUS > at {
			break
		}
		if d.Grenades != nil {
			out, found = d, true
		}
	}
	return out, found
}

// invConfrontAmmo confronte, pour chaque emplacement d'arme, le dernier CHARGEUR transmis en
// delta au chargeur de l'image-clé — a la seule condition que les deux lectures soient proches
// dans le temps (cf. invAmmoWindowUS).
func invConfrontAmmo(c *invConfrontation, seq []filmdec.InventoryDelta, k KeyframeInventory) {
	for slot := 0; slot < 2 && slot < len(k.Ammo); slot++ {
		if k.Ammo[slot].Mag == nil {
			continue
		}
		var last *uint32
		var lastTS uint64
		for i := range seq {
			d := seq[i]
			if d.TimestampUS > k.TimestampUS {
				break
			}
			for _, a := range d.Ammo {
				if a.WeaponSlot == slot && a.Mag != nil {
					last, lastTS = a.Mag, d.TimestampUS
				}
			}
		}
		if last == nil {
			continue
		}
		gap := k.TimestampUS - lastTS
		same := *last == *k.Ammo[slot].Mag
		for i, w := range invAmmoWindowsUS {
			if gap <= w {
				c.MagByWindow[i].n++
				if same {
					c.MagByWindow[i].ok++
				}
			}
		}
		if gap > invAmmoWindowUS {
			continue
		}
		c.MagChecked++
		if same {
			c.MagAgree++
		}
	}
}

// invRecall mesure LE RAPPEL : sur les transitions que les images-cles attestent, combien les
// deltas en portent-ils une trace ?
//
// CE QUE CE CHIFFRE EST, ET CE QU'IL N'EST PAS. Il ne dit pas combien de records l'ancre
// manque — on ignore combien il en existe. Il dit quelque chose de plus utile pour une fiche :
// quand l'inventaire d'un joueur CHANGE reellement, le canal delta le rapporte-t-il ? Un
// rappel bas ne rend pas les lectures fausses (la concordance le mesure a part) ; il borne le
// gain de fraicheur qu'on peut promettre.
func invRecall(c *invConfrontation, kf []KeyframeInventory, bySlot map[uint32][]filmdec.InventoryDelta) {
	bySlotKF := map[uint32][]KeyframeInventory{}
	for _, k := range kf {
		if k.GrenadesRead {
			bySlotKF[k.Slot] = append(bySlotKF[k.Slot], k)
		}
	}
	for slot, seq := range bySlotKF {
		sort.Slice(seq, func(i, j int) bool { return seq[i].TimestampUS < seq[j].TimestampUS })
		for i := 1; i < len(seq); i++ {
			prev, cur := seq[i-1], seq[i]
			if prev.Grenades == cur.Grenades {
				continue // pas de transition attestee
			}
			c.TransReal++
			for _, d := range bySlot[slot] {
				if d.TimestampUS <= prev.TimestampUS || d.TimestampUS > cur.TimestampUS {
					continue
				}
				if d.Grenades != nil && invSameGrenades(d.Grenades, cur.Grenades) {
					c.TransCaught++
					break
				}
			}
		}
	}
}

func invSameGrenades(delta []uint32, kfv [4]uint32) bool {
	if len(delta) != len(kfv) {
		return false
	}
	for i := range kfv {
		if delta[i] != kfv[i] {
			return false
		}
	}
	return true
}

// invFreshness échantillonne à 1 Hz, sur la durée de vie de chaque slot, l'âge de la dernière
// lecture connue — d'abord avec les images-clés seules, puis les deux canaux fusionnés.
func invFreshness(
	kf []KeyframeInventory, bySlot map[uint32][]filmdec.InventoryDelta,
) (ageKF, ageMerged []float64, between int) {
	kfBySlot := map[uint32][]uint64{}
	for _, k := range kf {
		if k.GrenadesRead {
			kfBySlot[k.Slot] = append(kfBySlot[k.Slot], k.TimestampUS)
		}
	}
	for s, kts := range kfBySlot {
		sort.Slice(kts, func(i, j int) bool { return kts[i] < kts[j] })
		merged := append([]uint64(nil), kts...)
		for _, d := range bySlot[s] {
			if d.Grenades == nil {
				continue
			}
			merged = append(merged, d.TimestampUS)
			if d.TimestampUS > kts[0] && d.TimestampUS < kts[len(kts)-1] {
				between++
			}
		}
		sort.Slice(merged, func(i, j int) bool { return merged[i] < merged[j] })
		for t0 := kts[0]; t0 <= kts[len(kts)-1]; t0 += 1_000_000 {
			ageKF = append(ageKF, lastAgeS(kts, t0))
			ageMerged = append(ageMerged, lastAgeS(merged, t0))
		}
	}
	return ageKF, ageMerged, between
}

func invPct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func invEnvInt(name string, def int) int {
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
