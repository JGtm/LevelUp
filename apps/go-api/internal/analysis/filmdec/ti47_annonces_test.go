package filmdec

// ti47_annonces_test.go — L'ENTREE ET LES RAPPORTS de l'instrument « ti=47 i2 personal-ai-data »
// (plan `.ai/V7.5/replay2d/PLAN_TI47_ANNONCES_ZONE.md`). Le moteur est dans
// `ti47_annonces_scan_test.go` ; ici on ne fait que lire ce qu'il a recolte et l'ecrire.
//
// PHASE 0 (toujours) : recensement de la bande, purete, et HISTOGRAMME DE CHAINAGE — la largeur
// de i2 mesuree sur les octets, contre son fantome et contre le temoin positif `i1` (R(24) connu).
// PHASE 1 (variable `TI47_WIDTH` posee au gate 0) : lecture des valeurs et confrontation aux
// oracles hors film (`ti47_annonces_oracle_test.go`).
//
// AUCUN CODE DE PRODUCTION N'EST MODIFIE par ce lot : ni `traverse.go`, ni un `components_*.go`,
// ni un hook, ni un schema d'artefact. L'instrument lit, mesure, ecrit des TSV.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// TestTI47Annonces execute l'instrument sur UN film.
func TestTI47Annonces(t *testing.T) {
	dir := os.Getenv(ti47FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument ti=47 saute", ti47FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("lecture de chunk_00 (registre) : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("analyse du registre : %v", err)
	}
	court := filepath.Base(dir)
	t.Logf("FILM %s · registre : %d archetypes · empreinte %016x",
		court, len(reg.Archetypes), RegistryFingerprint(reg))

	n := CountFilmChunks(dir)
	debut := time.Now()
	census := probeRecenseKF(dir, n)
	t.Logf("COUT — recensement des images-cle : %s · %d slots vus tous archetypes confondus",
		time.Since(debut).Round(time.Millisecond), len(census.tous))

	b := ti47Resout(t, reg, census)
	if b == nil {
		t.Fatalf("ti=47 introuvable : rien a mesurer sur ce film")
	}
	ti47JournaliseGrammaire(t, b, census)

	width := ti47Largeur(t)
	champLo, champHi := ti47Champ(t, maxI(width, 1))
	hor := ti47Horloge(t)
	debut = time.Now()
	m := ti47Balaye(dir, n, b, hor, width, ti47Candidates(t), champLo, champHi)
	t.Logf("COUT — passe delta : %s · %d chunks · %d paquets delta",
		time.Since(debut).Round(time.Millisecond), m.chunks, m.deltaPaquets)

	ti47JournaliseRecords(t, m)
	ti47JournaliseChainage(t, m)
	ti47JournaliseEcarts(t, m)
	ti47JournaliseRuns(t, m)
	ti47EcrisChainageTSV(t, court, m)
	if width == 0 {
		t.Logf("PHASE 1 SAUTEE : %s non pose. Le gate 0 doit d'abord trancher la largeur.",
			ti47WidthEnv)
		return
	}
	ti47JournaliseValeurs(t, m, width)
	ti47OracleRapport(t, court, m)
}

// ti47Horloge charge le manifeste du film : sans lui, les emissions n'ont pas d'instant de match
// et aucun oracle n'est confrontable.
func ti47Horloge(t *testing.T) probeHorloge {
	t.Helper()
	root, short := os.Getenv(ti47CacheEnv), os.Getenv(ti47ShortEnv)
	if root == "" || short == "" {
		t.Logf("HORLOGE : %s ou %s absent — emissions sans instant de match.",
			ti47CacheEnv, ti47ShortEnv)
		return probeHorloge{}
	}
	src, ok, err := filmcache.Open(root, short)
	if err != nil || !ok {
		t.Logf("HORLOGE : manifeste indisponible (%v) — emissions sans instant de match.", err)
		return probeHorloge{}
	}
	hor := probeHorloge{startMS: map[int]int{}}
	for _, c := range src.Meta() {
		hor.startMS[c.Index] = c.StartMS
	}
	return hor
}

// ti47JournaliseGrammaire publie la grammaire de l'archetype et sa bande. Sans elle, un « i2 » se
// lirait comme une constante alors que c'est une position dans le registre DE CE FILM.
func ti47JournaliseGrammaire(t *testing.T, b *ti47Bande, c probeCensus) {
	t.Helper()
	t.Logf("ARCHETYPE ti=%d · %d composants · i%d = %s", b.ti, len(b.arch.Components),
		b.iPersonal, compPersonalAIData)
	for i, nom := range b.arch.Components {
		marque := " "
		switch i {
		case b.iPersonal:
			marque = "*"
		case b.iStatic, b.iDynamic:
			marque = "+"
		}
		t.Logf("   %s i%-2d %s", marque, i, nom)
	}
	t.Logf("BANDE : %d slots vus en image-cle pour ti=%d · fantome %d slots (jamais vus le porter)",
		len(c.parTI[b.ti]), b.ti, len(b.ghost))
}

// ti47JournaliseRecords publie le recensement, reel contre fantome.
func ti47JournaliseRecords(t *testing.T, m *ti47Moisson) {
	t.Helper()
	r, f := m.reel, m.fantome
	t.Logf("RECORDS DELTA (denominateur : %d paquets) : reel %d · fantome %d · rapport %s",
		m.deltaPaquets, r.records, f.records, probeRapport(r.records, f.records))
	t.Logf("   hors grammaire : reel %d (%.2f %%) · fantome %d (%.2f %%)",
		r.horsGrammaire, 100*float64(r.horsGrammaire)/float64(maxI(1, r.records)),
		f.horsGrammaire, 100*float64(f.horsGrammaire)/float64(maxI(1, f.records)))
	t.Logf("   i%d annonce dans %d records (%.2f %%) · fantome %d (%.2f %%)",
		m.b.iPersonal, r.parIndex[m.b.iPersonal],
		100*float64(r.parIndex[m.b.iPersonal])/float64(maxI(1, r.records)),
		f.parIndex[m.b.iPersonal],
		100*float64(f.parIndex[m.b.iPersonal])/float64(maxI(1, f.records)))
	t.Logf("   indices les plus annonces (reel) : %s", ti47TopIndices(r.parIndex, m.b, 6))
	t.Logf("   masques SINGLETON (reel) : %s", ti47TopIndices(r.singletons, m.b, 6))
	vies := r.viesConfirmees()
	t.Logf("   vies : %d paires (slot,gen) vues · %d CONFIRMEES (>= %d records) · fantome %d/%d",
		len(r.parVie), len(vies), ti47VieMin, len(f.viesConfirmees()), len(f.parVie))
	t.Logf("   slots reels les plus actifs : %s", ti47TopSlots(r.parSlot, 6))
}

// ti47TopIndices rend les index les plus comptes, avec leur nom.
func ti47TopIndices(par map[int]int, b *ti47Bande, top int) string {
	idx := make([]int, 0, len(par))
	for i := range par {
		idx = append(idx, i)
	}
	sort.Slice(idx, func(x, y int) bool { return par[idx[x]] > par[idx[y]] })
	var parts []string
	for i, id := range idx {
		if i >= top {
			break
		}
		if id < 0 { // convention de `ti47Singleton` : masque a plusieurs composants
			parts = append(parts, fmt.Sprintf("masque multiple=%d", par[id]))
			continue
		}
		nom := b.arch.component(id)
		if nom == "" {
			nom = "HORS GRAMMAIRE"
		}
		parts = append(parts, fmt.Sprintf("i%d=%d (%s)", id, par[id], nom))
	}
	return strings.Join(parts, " · ")
}

// ti47TopSlots rend les slots les plus actifs.
func ti47TopSlots(par map[uint32]int, top int) string {
	s := make([]uint32, 0, len(par))
	for k := range par {
		s = append(s, k)
	}
	sort.Slice(s, func(x, y int) bool { return par[s[x]] > par[s[y]] })
	var parts []string
	for i, k := range s {
		if i >= top {
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", k, par[k]))
	}
	return strings.Join(parts, " · ")
}

// ti47JournaliseChainage publie les quatre histogrammes et le verdict de largeur.
//
// LE PIC SE JUGE CONTRE LE PLANCHER MESURE DU MEME FILM, jamais contre un pourcentage ecrit
// d'avance : le taux de hasard depend du nombre de slots recenses, qui varie d'un film a l'autre.
func ti47JournaliseChainage(t *testing.T, m *ti47Moisson) {
	t.Helper()
	t.Logf("CHAINAGE — un record correctement dimensionne finit la ou le suivant commence.")
	for _, c := range []struct {
		nom string
		ch  *ti47Chain
	}{
		{fmt.Sprintf("i%d personal-ai · cible TOUS slots", m.b.iPersonal), &m.chainPersonal},
		{fmt.Sprintf("i%d personal-ai · cible BANDE ti=47", m.b.iPersonal), &m.chainPersonalBande},
		{fmt.Sprintf("i%d personal-ai · FANTOME", m.b.iPersonal), &m.chainPersonalFantome},
		{fmt.Sprintf("i%d dynamic R(24) · cible TOUS", m.b.iDynamic), &m.chainDynamique},
		{fmt.Sprintf("i%d dynamic R(24) · cible BANDE", m.b.iDynamic), &m.chainDynamiqueBande},
		{fmt.Sprintf("i%d static · cible TOUS", m.b.iStatic), &m.chainStatique},
	} {
		n := c.ch.denom[1]
		if n == 0 {
			t.Logf("  %-40s AUCUN record singleton", c.nom)
			continue
		}
		pl := c.ch.plancher()
		var parts []string
		for _, d := range c.ch.sommets(5) {
			parts = append(parts, fmt.Sprintf("d=%d : %.1f %% (%.1fx)",
				d, 100*c.ch.taux(d), ti47Exces(c.ch.taux(d), pl)))
		}
		t.Logf("  %-40s %6d records · plancher median %.2f %% · sommets : %s",
			c.nom, n, 100*pl, strings.Join(parts, " · "))
	}
}

// ti47JournaliseEcarts publie la distance entre debuts de records consecutifs de la bande, par
// masque singleton du PREMIER. La largeur se lit directement : distance − en-tete − index.
func ti47JournaliseEcarts(t *testing.T, m *ti47Moisson) {
	t.Helper()
	entete := worldObjectHeaderBits + worldObjectIndexBits
	idx := make([]int, 0, len(m.ecarts))
	for i := range m.ecarts {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	t.Logf("DISTANCES entre debuts de records consecutifs de la bande (en-tete+index = %d bits ;"+
		" largeur = distance − %d) :", entete, entete)
	for _, i := range idx {
		h := m.ecarts[i]
		total := 0
		for _, n := range h {
			total += n
		}
		ordre := make([]int, 0, 32)
		for d, n := range h {
			if n > 0 {
				ordre = append(ordre, d)
			}
		}
		sort.Slice(ordre, func(x, y int) bool { return h[ordre[x]] > h[ordre[y]] })
		var parts []string
		for k, d := range ordre {
			if k >= 5 {
				break
			}
			parts = append(parts, fmt.Sprintf("%d bits (W=%d) : %.1f %%", d, d-entete,
				100*float64(h[d])/float64(maxI(1, total))))
		}
		nom := m.b.arch.component(i)
		t.Logf("  apres un singleton {i%d %s} : %d distances (+%d hors plafond) · %s",
			i, nom, total, m.ecartsHors[i], strings.Join(parts, " · "))
		t.Logf("      masque du record suivant : %s", ti47TopIndices(m.suivants[i], m.b, 4))
	}
}

// ti47JournaliseRuns publie, pour chaque largeur candidate, la distribution des longueurs de
// chaine. Une largeur juste enchaine ; une fausse largeur meurt au premier pas.
func ti47JournaliseRuns(t *testing.T, m *ti47Moisson) {
	t.Helper()
	if len(m.runs) == 0 {
		return
	}
	ws := make([]int, 0, len(m.runs))
	for w := range m.runs {
		ws = append(ws, w)
	}
	sort.Ints(ws)
	t.Logf("LONGUEUR DE CHAINE par largeur candidate (records SINGLETON {i%d} de la bande,"+
		" enchainement exige sur la BANDE ti=%d) :", m.b.iPersonal, m.b.ti)
	for _, w := range ws {
		h := m.runs[w]
		var total, somme, atteint2, atteint4, atteint8, max int
		for k, n := range h {
			total += n
			somme += k * n
			if k >= 2 {
				atteint2 += n
			}
			if k >= 4 {
				atteint4 += n
			}
			if k >= 8 {
				atteint8 += n
			}
			if n > 0 && k > max {
				max = k
			}
		}
		d := float64(maxI(1, total))
		t.Logf("  W=%-4d moyenne %.2f · >=2 pas : %.1f %% · >=4 : %.1f %% · >=8 : %.1f %%"+
			" · max %d · (0 pas : %.1f %%)", w, float64(somme)/d,
			100*float64(atteint2)/d, 100*float64(atteint4)/d, 100*float64(atteint8)/d, max,
			100*float64(h[0])/d)
	}
}

// ti47Exces rend le rapport pic / plancher, ou 0 quand le plancher est nul.
func ti47Exces(taux, plancher float64) float64 {
	if plancher <= 0 {
		return 0
	}
	return taux / plancher
}

// ti47EcrisChainageTSV depose les histogrammes complets : le rapport doit etre reproductible
// decalage par decalage, pas seulement sur les cinq sommets journalises.
func ti47EcrisChainageTSV(t *testing.T, court string, m *ti47Moisson) {
	t.Helper()
	out := os.Getenv(ti47TSVEnv)
	if out == "" {
		return
	}
	var b strings.Builder
	b.WriteString("decalage\treel_tous\treel_bande\tfantome\tdyn_tous\tdyn_bande\tstat_tous" +
		"\treel_denom\tdyn_denom\n")
	for d := 1; d <= ti47MaxDecalage; d++ {
		fmt.Fprintf(&b, "%d\t%.6f\t%.6f\t%.6f\t%.6f\t%.6f\t%.6f\t%d\t%d\n", d,
			m.chainPersonal.taux(d), m.chainPersonalBande.taux(d), m.chainPersonalFantome.taux(d),
			m.chainDynamique.taux(d), m.chainDynamiqueBande.taux(d), m.chainStatique.taux(d),
			m.chainPersonal.denom[d], m.chainDynamique.denom[d])
	}
	path := filepath.Join(out, court+"_ti47_chainage.tsv")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture TSV : %v", err)
	}
	t.Logf("  TSV : %s", path)
}
