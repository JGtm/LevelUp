package filmdec

// ti13_etat_report_test.go — LOT C-bis PHASE 1, item CB.1.2 : les volets (c), (d), (e) de la
// mesure d'etat, et le verdict temporel partage. Suite de `ti13_etat_test.go`, coupe la pour
// tenir le seuil de 500 lignes par fichier.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// ti13VerdictTemporel confronte une liste d'INSTANTS (sommets de rampe, changements d'etat...)
// aux captures de l'oracle, avec les trois chiffres sans lesquels un taux ne se juge pas : le
// taux reel, le temoin decale de +20 s, et le NIVEAU DU HASARD.
//
// POURQUOI LE NIVEAU DU HASARD EST PUBLIE. Avec N instants et une fenetre de +/- w sur une duree
// T, une capture tombe pres d'un instant par pur hasard avec une probabilite d'environ
// `N x 2w / T`. Le lot C a mesure un temoin a 36,6 % pour un seuil de 20 % — et ce temoin etait
// SOUS le hasard (46,1 %) : le seuil etait inatteignable par construction pour un canal bavard.
// Le verdict se prononce donc sur le seuil ECRIT, mais le hasard est publie a cote pour que le
// lecteur sache ce que le temoin vaut.
func ti13VerdictTemporel(t *testing.T, sb *strings.Builder, short, volet, quoi string,
	instants []int, o zcOracle, col *ti13Col,
) {
	t.Helper()
	if len(o.times) == 0 {
		t.Logf("  volet (%s) : ORACLE ABSENT sur ce film (famille %q) — aucune capture nommee,"+
			" le volet n'est pas jugeable ici", volet, o.family)
		fmt.Fprintf(sb, "# (%s) %s : oracle absent (famille %s)\n", volet, short, o.family)
		return
	}
	sort.Ints(instants)
	reel := ti13Couvertes(o.times, instants, 0)
	temoin := ti13Couvertes(o.times, instants, ti13DecalageMS)
	n := len(o.times)
	duree := ti13Duree(col)
	hasard := 0.0
	if duree > 0 {
		hasard = float64(len(instants)) * 2 * ti13FenetreMS / float64(duree)
		if hasard > 1 {
			hasard = 1
		}
	}
	partReel := float64(reel) / float64(n)
	partTem := float64(temoin) / float64(n)

	tenu := partReel >= ti13SeuilCaptures && partTem < partReel/2
	v := "NON TENU"
	if tenu {
		v = "TENU"
	}
	t.Logf("  captures couvertes par un %s dans +/- %d ms : %d/%d = %.1f %% (seuil %.0f %%)",
		quoi, ti13FenetreMS, reel, n, 100*partReel, 100*ti13SeuilCaptures)
	t.Logf("  temoin decale (+%d ms) : %d/%d = %.1f %% (doit etre sous la MOITIE du reel,"+
		" soit %.1f %%)", ti13DecalageMS, temoin, n, 100*partTem, 100*partReel/2)
	t.Logf("  NIVEAU DU HASARD : %.1f %% (%d instants, fenetre +/- %d ms, duree %d ms)",
		100*hasard, len(instants), ti13FenetreMS, duree)
	t.Logf("  VERDICT (%s) : %s", volet, v)
	fmt.Fprintf(sb, "# (%s) %s : reel %d/%d (%.1f %%), temoin %.1f %%, hasard %.1f %%, verdict %s\n",
		volet, short, reel, n, 100*partReel, 100*partTem, 100*hasard, v)
}

// ti13Couvertes compte les captures ayant un instant dans la fenetre, apres decalage du temoin.
func ti13Couvertes(captures, instants []int, decalage int) int {
	n := 0
	for _, c := range captures {
		cible := c + decalage
		i := sort.SearchInts(instants, cible-ti13FenetreMS)
		if i < len(instants) && instants[i] <= cible+ti13FenetreMS {
			n++
		}
	}
	return n
}

// ti13Duree rend l'etendue temporelle couverte par les echantillons.
func ti13Duree(col *ti13Col) int {
	lo, hi, ok := 0, 0, false
	for _, g := range [][]ti13Ech{col.scal, col.joue} {
		for _, e := range g {
			if !ok || e.tMS < lo {
				lo, ok = e.tMS, true
			}
			if e.tMS > hi {
				hi = e.tMS
			}
		}
	}
	return hi - lo
}

// -------------------------------------------------------------------------------------
// (c) TAG 4 et ENUMERES — lequel est l'ETAT ?
// -------------------------------------------------------------------------------------

// ti13RapportEtat teste les deux candidats a « l'etat de la zone » : le R(32) du tag 4 et les
// enumeres (tag 1 en mode A). Le critere d'ENUMERABILITE vient du gate : un etat a peu de
// valeurs distinctes par slot.
func ti13RapportEtat(t *testing.T, sb *strings.Builder, short string, col *ti13Col, o zcOracle) {
	t.Helper()
	for _, cand := range []struct {
		nom string
		tag int
	}{
		{"TAG 4 (R32)", ManagedPropertyTagU32},
		{"TAG 1 (enumere)", ManagedPropertyTagEnum},
	} {
		ti13EtatCandidat(t, sb, short, col, o, cand.nom, cand.tag)
	}
}

// ti13EtatCandidat mesure un candidat d'etat : enumerabilite par slot, puis coincidence avec
// les captures.
func ti13EtatCandidat(t *testing.T, sb *strings.Builder, short string, col *ti13Col,
	o zcOracle, nom string, tag int,
) {
	t.Helper()
	parSlot := map[uint32][]ti13Ech{}
	for _, e := range col.scal {
		if e.tag == tag && e.hasPay {
			parSlot[e.slot] = append(parSlot[e.slot], e)
		}
	}
	t.Logf("")
	t.Logf("=== (c) %s — %d valeurs sur %d slots", nom, ti13CountTag(col.scal, tag), len(parSlot))
	if len(parSlot) == 0 {
		t.Logf("  aucune valeur : candidat SANS OBJET sur ce film")
		fmt.Fprintf(sb, "# (c) %s %s : aucune valeur\n", nom, short)
		return
	}
	enumerables, juges := 0, 0
	var changements []int
	for _, s := range ti13SlotsTries(parSlot) {
		ss := parSlot[s]
		sort.Slice(ss, func(i, j int) bool { return ss[i].tMS < ss[j].tMS })
		distinctes := map[uint64]bool{}
		for k, e := range ss {
			distinctes[e.pay] = true
			if k > 0 && ss[k-1].pay != e.pay {
				changements = append(changements, e.tMS)
			}
		}
		if len(ss) >= ti13MinParSlot {
			juges++
			if len(distinctes) <= ti13MaxValeursEtat {
				enumerables++
			}
		}
		t.Logf("  slot %d : %d valeurs, %d distinctes (seuil <= %d)", s, len(ss), len(distinctes),
			ti13MaxValeursEtat)
		fmt.Fprintf(sb, "c_etat\t%s\t%s\t%d\t%d\t%d\n", short, nom, s, len(ss), len(distinctes))
	}
	ti13VerdictEnumerable(t, sb, short, nom, enumerables, juges)
	t.Logf("  %d changements de valeur detectes", len(changements))
	ti13VerdictTemporel(t, sb, short, "c/"+nom, "changement de valeur", changements, o, col)
}

// ti13VerdictEnumerable rend le verdict d'enumerabilite.
func ti13VerdictEnumerable(t *testing.T, sb *strings.Builder, short, nom string, ok, juges int) {
	t.Helper()
	if juges == 0 {
		t.Logf("  ENUMERABILITE : aucun slot n'atteint %d valeurs — non jugeable", ti13MinParSlot)
		return
	}
	part := float64(ok) / float64(juges)
	v := "NON TENUE"
	if part >= ti13SeuilCoherence {
		v = "TENUE"
	}
	t.Logf("  ENUMERABILITE : %d/%d slots juges ont <= %d valeurs distinctes = %.1f %% : %s",
		ok, juges, ti13MaxValeursEtat, 100*part, v)
	fmt.Fprintf(sb, "# (c) %s %s : enumerabilite %d/%d (%.1f %%), verdict %s\n", nom, short,
		ok, juges, 100*part, v)
}

// -------------------------------------------------------------------------------------
// (d) UNICITE — une seule zone « active » a la fois ?
// -------------------------------------------------------------------------------------

// ti13RapportUnicite mesure, sur les instants ou au moins un slot porte une rampe en cours, la
// part du temps ou UN SEUL slot la porte. C'est la clause KOTH, reformulee au niveau du SLOT
// ti=13 (objet) et non du marqueur — la correction que le lot C avait demandee.
func ti13RapportUnicite(t *testing.T, sb *strings.Builder, short string, col *ti13Col) {
	t.Helper()
	series := map[uint32][]ti13Ech{}
	for _, e := range col.scal {
		if e.tag == ManagedPropertyTagQuant && e.hasPay {
			series[e.slot] = append(series[e.slot], e)
		}
	}
	var toutes []ti13Ramp
	for s, ss := range series {
		sort.Slice(ss, func(i, j int) bool { return ss[i].tMS < ss[j].tMS })
		toutes = append(toutes, ti13FindRamps(s, ss)...)
	}
	t.Logf("")
	t.Logf("=== (d) UNICITE : une seule zone active a la fois ? — %d rampes sur %d slots",
		len(toutes), len(series))
	if len(toutes) == 0 {
		t.Logf("  aucune rampe : volet (d) SANS OBJET sur ce film")
		fmt.Fprintf(sb, "# (d) %s : aucune rampe\n", short)
		return
	}
	seule, total := ti13TempsSeul(toutes)
	if total == 0 {
		t.Logf("  duree cumulee nulle : non jugeable")
		return
	}
	part := float64(seule) / float64(total)
	v := "NON TENUE"
	if part >= ti13SeuilUnique {
		v = "TENUE"
	}
	t.Logf("  temps couvert par au moins une rampe : %d ms, dont %d ms par UNE SEULE = %.1f %%"+
		" (seuil %.0f %%) : %s", total, seule, 100*part, 100*ti13SeuilUnique, v)
	fmt.Fprintf(sb, "# (d) %s : unicite %d/%d ms (%.1f %%), seuil %.0f %%, verdict %s\n", short,
		seule, total, 100*part, 100*ti13SeuilUnique, v)
}

// ti13TempsSeul rend (temps ou une seule rampe court, temps ou au moins une court), en ms.
func ti13TempsSeul(rs []ti13Ramp) (seule, total int) {
	type borne struct {
		t, d int
	}
	bs := make([]borne, 0, 2*len(rs))
	for _, r := range rs {
		bs = append(bs, borne{r.t0, +1}, borne{r.tMax, -1})
	}
	sort.Slice(bs, func(i, j int) bool {
		if bs[i].t != bs[j].t {
			return bs[i].t < bs[j].t
		}
		return bs[i].d < bs[j].d
	})
	n := 0
	for i := 0; i+1 < len(bs); i++ {
		n += bs[i].d
		dt := bs[i+1].t - bs[i].t
		if n >= 1 {
			total += dt
		}
		if n == 1 {
			seule += dt
		}
	}
	return seule, total
}

// -------------------------------------------------------------------------------------
// (d bis) LES VALEURS PAR JOUEUR, et (e) le volet CTF
// -------------------------------------------------------------------------------------

// ti13RapportParJoueur decrit ce que portent les composants i2..i33 : quels index de joueur
// parlent, quels tags, et combien de valeurs distinctes par joueur.
func ti13RapportParJoueur(t *testing.T, sb *strings.Builder, short string, col *ti13Col) {
	t.Helper()
	t.Logf("")
	t.Logf("=== (d bis / e) VALEURS PAR JOUEUR (i2..i33) — %d echantillons", len(col.joue))
	if len(col.joue) == 0 {
		t.Logf("  aucun echantillon : volet SANS OBJET sur ce film")
		fmt.Fprintf(sb, "# (d bis) %s : aucun echantillon par joueur\n", short)
		return
	}
	parTag := map[int]int{}
	parJoueur := map[int]map[uint64]int{}
	for _, e := range col.joue {
		parTag[e.tag]++
		j := ManagedPropertyPlayerIndex(e.idx)
		if j < 0 || !e.hasPay {
			continue
		}
		if parJoueur[j] == nil {
			parJoueur[j] = map[uint64]int{}
		}
		parJoueur[j][e.pay]++
	}
	t.Logf("  distribution des tags : %s", ti13TagsTries(parTag, len(col.joue)))
	js := make([]int, 0, len(parJoueur))
	for j := range parJoueur {
		js = append(js, j)
	}
	sort.Ints(js)
	for _, j := range js {
		tot := 0
		for _, n := range parJoueur[j] {
			tot += n
		}
		t.Logf("  joueur %d : %d valeurs, %d distinctes", j, tot, len(parJoueur[j]))
		fmt.Fprintf(sb, "d_joueur\t%s\t%d\t%d\t%d\n", short, j, tot, len(parJoueur[j]))
	}
	t.Logf("  RESERVE : la phase 0 a mesure qu'en Strongholds le trafic apparent de ces" +
		" composants est de la CONTAMINATION d'ancrage (0 %% de chainage). Les chiffres" +
		" ci-dessus ne valent que sur les films ou le chainage les valide (KOTH).")
	fmt.Fprintf(sb, "# (d bis) %s : %d echantillons par joueur, %d index de joueur\n", short,
		len(col.joue), len(parJoueur))
}

// ti13TagsTries rend la distribution des tags, les plus frequents d'abord.
func ti13TagsTries(m map[int]int, total int) string {
	type kv struct{ tag, n int }
	l := make([]kv, 0, len(m))
	for tag, n := range m {
		l = append(l, kv{tag, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	var sb strings.Builder
	for i, e := range l {
		if i >= 6 {
			break
		}
		fmt.Fprintf(&sb, "t%d:%d (%.1f %%) ", e.tag, e.n, 100*float64(e.n)/float64(total))
	}
	return strings.TrimSpace(sb.String())
}
