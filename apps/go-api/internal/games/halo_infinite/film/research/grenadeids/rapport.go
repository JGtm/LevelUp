//go:build research

package grenadeids

// rapport.go — LA SORTIE TEXTE. Elle est faite pour etre RECOPIEE dans la note : une ligne par
// fait, chiffres colles, aucune interpretation. Les verdicts appartiennent a la note, pas a
// l instrument.

import (
	"fmt"
	"io"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// EcrireEntete publie ce que le film dit de lui-meme, avant toute mesure.
func EcrireEntete(w io.Writer, b *Bobine) {
	build := b.Build
	if build == "" {
		build = "sans-section"
	}
	ti := "absent"
	if b.TiProjectile >= 0 {
		ti = fmt.Sprintf("%d (%d/%d noms)", b.TiProjectile, b.NomsProjectileVus, len(composantsProjectile))
	}
	accord := "="
	if b.Marqueur != b.MarqueurProd {
		accord = "DIFFERENT"
	}
	fmt.Fprintf(w, "== %s  version=%d  build=%s  archetypes=%d  ti_projectile=%s\n",
		b.ID, b.Version, build, b.Archetypes, ti)
	fmt.Fprintf(w, "   marqueur_production=0x%06X  marqueur_registre=0x%06X  %s  ambiguites=%v\n",
		b.MarqueurProd, b.Marqueur, accord, b.Ambiguites)
}

// EcrireReleve publie les passes A, B et C d un film.
func EcrireReleve(w io.Writer, r *Releve, top int) {
	fmt.Fprintf(w, "   paquets_delta=%d octets_delta=%d marqueurs_production=%d marqueurs_registre=%d\n",
		r.PaquetsDelta, r.OctetsDelta, r.Marqueurs[MarqueurProduction], r.Marqueurs[MarqueurRegistre])
	fmt.Fprintf(w, "   marqueurs_impairs=%d\n", r.Marqueurs[MarqueurImpair])
	ecrirePasseA(w, r, MarqueurProduction, "production")
	ecrirePasseA(w, r, MarqueurRegistre, "registre")
	ecrirePasseA(w, r, MarqueurImpair, "impair")
	ecrirePasseB(w, r)
	ecrirePasseC(w, "passe C  apres le marqueur de production", r.Famille, top)
	ecrirePasseC(w, "passe C  apres le marqueur du registre", r.FamilleRegistre, top)
	ecrirePasseC(w, "passe C  apres le marqueur IMPAIR (identifiant lu a +23)", r.FamilleImpair, top)
	ecrirePasseCParArchetype(w, r, top)
}

// ecrirePasseCParArchetype separe les DEUX populations que le marqueur confond (decouverte D2
// (3.3r)) : sur le marqueur de production, `ti=41` (projectile) et `ti=9` (`managed-player`)
// portent les MEMES vingt-quatre bits. Ce qui suit une naissance de `managed-player` n a rien a
// faire dans la famille des grenades, et jusqu ici rien ne les separait.
func ecrirePasseCParArchetype(w io.Writer, r *Releve, top int) {
	if len(r.ParTypeIndex) == 0 && r.TiIndetermines == 0 {
		return
	}
	fmt.Fprint(w, "   passe C  marqueurs par ARCHETYPE reel (sixieme bit d index lu a marqueur-1) :")
	for _, ti := range clesTriees(r.ParTypeIndex) {
		fmt.Fprintf(w, " ti=%d:%d", ti, r.ParTypeIndex[ti])
	}
	fmt.Fprintf(w, " indetermines=%d\n", r.TiIndetermines)
	for _, ti := range clesTriees(r.FamilleParTi) {
		ecrirePasseC(w, fmt.Sprintf("passe C  ti=%d SEUL", ti), r.FamilleParTi[ti], top)
	}
}

// ecrirePasseA publie, decalage par decalage, les identifiants actuels reconnus.
func ecrirePasseA(w io.Writer, r *Releve, src SourceMarqueur, nom string) {
	parSource := r.ParDecalage[src]
	if len(parSource) == 0 {
		fmt.Fprintf(w, "   passe A  marqueur %s : AUCUN identifiant actuel dans la fenetre\n", nom)
		return
	}
	fmt.Fprintf(w, "   passe A  marqueur %s : %d decalage(s) portant un identifiant actuel\n",
		nom, len(parSource))
	for _, d := range clesTriees(parSource) {
		for _, id := range idsTries(parSource[d]) {
			rang, _ := rangConnu(id)
			fmt.Fprintf(w, "     decalage=%+d  0x%08X  n=%d  rang_actuel=%d\n",
				d, id, parSource[d][id], rang)
		}
	}
}

// ecrirePasseB publie le balayage absolu et le rattachement au marqueur le plus proche.
//
// LE HASARD ATTENDU EST PUBLIE AVEC LA MESURE, et c est ce qui rend la passe refutable : quatre
// valeurs de 32 bits cherchees a chaque position de bit d un flux de N bits tombent par hasard
// `N x 4 / 2^32` fois. Sous ce chiffre, un compte ne dit rien.
func ecrirePasseB(w io.Writer, r *Releve) {
	fmt.Fprintf(w, "   passe B  hasard attendu sur ce flux = %.2f occurrence(s) (4 valeurs sur 2^32, %d positions)\n",
		hasardAttendu(r.OctetsDelta), r.OctetsDelta*8)
	if len(r.Absolus) == 0 {
		fmt.Fprintln(w, "   passe B  balayage absolu : AUCUNE occurrence des quatre identifiants actuels")
		return
	}
	for _, id := range idsTries(r.Absolus) {
		rang, _ := rangConnu(id)
		fmt.Fprintf(w, "   passe B  0x%08X rang_actuel=%d total=%d hors_marqueur=%d\n",
			id, rang, r.Absolus[id], r.HorsMarqueur[id])
		for _, d := range clesTriees(r.DistanceAuMarqueur[id]) {
			fmt.Fprintf(w, "     ecart_au_marqueur=%+d n=%d\n", d, r.DistanceAuMarqueur[id][d])
		}
	}
}

// ecrirePasseC publie l histogramme sans liste blanche.
func ecrirePasseC(w io.Writer, titre string, fam []Candidat, top int) {
	if len(fam) == 0 {
		return
	}
	fmt.Fprintf(w, "   %s : %d identifiant(s) distinct(s)\n", titre, len(fam))
	for i, c := range fam {
		if top > 0 && i >= top {
			fmt.Fprintf(w, "     ... %d de plus\n", len(fam)-top)
			break
		}
		etiquette := "inconnu"
		if c.Connu {
			etiquette = fmt.Sprintf("rang=%d", c.Rang)
		}
		fmt.Fprintf(w, "     0x%08X n=%d %s index103[%d..%d] hors=%d  index102[%d..%d] hors=%d\n",
			c.ID, c.N, etiquette, c.IndexMin, c.IndexMax, c.IndexHors8,
			c.AltMin, c.AltMax, c.AltHors8)
	}
}

// EcrireAppariement publie la passe D.
func EcrireAppariement(w io.Writer, st StatsI22, app []Appariement, top int) {
	fmt.Fprintf(w, "   passe D  i22 : records=%d lectures=%d i22_lues=%d i22_non_lues=%d "+
		"implausibles=%d porteurs=%d decrements=%d\n",
		st.Records, st.Lectures, st.I22Read, st.I22Unread, st.Implausible, st.Porteurs, st.Decrements)
	fmt.Fprintf(w, "     decrements par rang : 0=%d 1=%d 2=%d 3=%d\n",
		st.ParRang[0], st.ParRang[1], st.ParRang[2], st.ParRang[3])
	for i, a := range app {
		if top > 0 && i >= top {
			fmt.Fprintf(w, "     ... %d de plus\n", len(app)-top)
			break
		}
		fmt.Fprintf(w, "     0x%08X n=%d apparies=%d ambigus=%d rangs[%d,%d,%d,%d] "+
			"index_apparies=%d fonctionnel=%s\n",
			a.ID, a.N, a.Apparies, a.Ambigus,
			a.ParRang[0], a.ParRang[1], a.ParRang[2], a.ParRang[3],
			len(a.Couples), ouiNon(a.Fonctionnel()))
	}
}

// hasardAttendu rend le nombre d occurrences que les quatre identifiants produiraient par pur
// hasard sur un flux de `octets` octets balaye bit a bit.
func hasardAttendu(octets int) float64 {
	return float64(octets) * 8 * float64(len(grammar.GrenadeTypeIDsByRank)) / 4294967296.0
}

// ouiNon rend un booleen en clair.
func ouiNon(v bool) string {
	if v {
		return "oui"
	}
	return "non"
}

// rangConnu rend le rang actuel d un identifiant, et s il appartient a la liste blanche.
func rangConnu(id uint32) (int, bool) { return grammar.GrenadeRankOf(id) }

// clesTriees rend les cles entieres d une carte, croissantes.
func clesTriees[V any](m map[int]V) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// idsTries rend les identifiants d une carte, croissants.
func idsTries[V any](m map[uint32]V) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
