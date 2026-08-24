package filmdec

// i59_rank_cross_test.go — INSTRUMENT DE MESURE de l'item 0.7 du plan
// .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md : les TAGS 0 et 2 d'`i59`
// `biped-spartan-ability-non-predicted-state`, jamais croisés à une identité.
//
// POURQUOI CETTE SONDE EXISTE, ET CE QU'ELLE N'EST PAS. La Phase 0 a rendu un négatif sur
// `i56` (item 0.4) et `i51` (item 0.5). L'arbitrage du 2026-08-25 a relevé qu'un canal
// restait vierge : l'instrument `i59_tag3_test.go` compte bien TOUS les tags (ligne 71) mais
// ne croise aux rangs QUE le tag 3 — celui du grappin, livré. Sur `00ba2e1c`, les tags 0 et
// 2 pèsent 1 572 et 1 565 lectures (3 234 lues sur 3 274 annoncées, 98,8 %), soit 12,7 fois
// le volume d'`i56` sur le même film. **Périmètre strictement instrumental, décidé par le
// superviseur : aucun code de production, aucune publication, et les Phases 1-2 ne
// s'ouvrent PAS même si le verdict est positif.**
//
// ────────────────────────────────────────────────────────────────────────────────────
// LA BARRE DE DÉCISION, ÉCRITE AVANT LA MESURE (elle ne se renégocie pas après) :
//
//	POSITIF — le tag discrimine — SI ET SEULEMENT SI ses transitions se CONCENTRENT sur les
//	vies portant le rang cible ET sont ~nulles à la fois sur les vies des AUTRES rangs et
//	sur les vies SANS identité `i48` (le témoin le plus sévère). C'est la forme qu'avait le
//	camouflage : 39 transitions sur les vies du rang cible, 0 sur 574 autres vies.
//
//	NÉGATIF — ÉTAT GÉNÉRIQUE — si le signal est présent PARTOUT à volume comparable. C'est
//	le défaut exact qui a tué `i57` (bit 0 à 1 sur 386 lectures sur 386) : un état qui ne
//	distingue personne ne date rien. Dans ce cas on CLASSE, sans négocier le seuil.
//
// ────────────────────────────────────────────────────────────────────────────────────
//
// LE CONTRÔLE DE VALIDITÉ INTERNE, ET C'EST LUI QUI REND LE VERDICT OPPOSABLE. Le tag 3 est
// mesuré AVEC les autres, alors qu'on connaît déjà sa réponse : c'est le canal du GRAPPIN,
// validé à 115/117 sur les vies de rang 20. Il doit donc ressortir concentré sur le groupe
// GRAPPIN (rangs 4 et 20). **S'il ne ressort pas, c'est la MÉTHODE qui est en cause, et le
// verdict sur les tags 0/1/2 ne vaut rien** — un négatif ne serait alors pas une propriété
// du signal mais un défaut de la mesure. Sans ce garde-fou, un « rien trouvé » est
// ininterprétable.
//
// CE QUI EST COMPTÉ. L'événement est la TRANSITION vers le tag (la lecture précédente de la
// MÊME vie portait un autre tag), pas le volume de lectures : un état répliqué tant qu'il
// dure produirait autant de « preuves » qu'il y a de paquets, ce qui favoriserait
// mécaniquement les tags les plus fréquents. Le volume brut est publié à part, comme
// dénominateur de lecture.
//
// LECTURE SEULE, gardé par I59X_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test (globaux de paquet).
// Le balayage est celui d'`i59_tag3_test.go` (`i59Scan`), réutilisé tel quel — deux lecteurs
// du même champ divergeraient.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I59X_FILM=<repo>/data/cache/film_chunks/00ba2e1c \
//	  go test ./internal/analysis/filmdec/ -run '^TestI59TagsCrossI48Rank$' -timeout 60m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i59xFilmEnv = "I59X_FILM"

// i59xTags sont les quatre valeurs du tag R(2). Le 3 est inclus comme CONTRÔLE POSITIF (cf.
// en-tête) : on connaît sa réponse, et elle valide ou invalide la méthode.
var i59xTags = []uint32{0, 1, 2, 3}

// i59xSpec construit le tableau d'un tag donné. Le groupe GRAPPIN est présent pour TOUS les
// tags, pas seulement pour le 3 : c'est ce qui rend le contrôle de validité lisible d'un
// coup d'œil, et ce qui montrerait qu'un tag « cible » se confond en réalité avec lui.
func i59xSpec(tag uint32) xrSpec {
	return xrSpec{
		event: fmt.Sprintf("TRANSITIONS vers i59 tag==%d", tag),
		groups: []xrGroup{
			{name: "RÉPULSEUR (rang 6)", ranks: []int{6}},
			{name: "PROPULSEUR (rangs 5/21)", ranks: []int{5, 21}},
			{name: "GRAPPIN (rangs 4/20) TÉMOIN+", ranks: []int{4, 20}},
		},
	}
}

func TestI59TagsCrossI48Rank(t *testing.T) {
	dir := os.Getenv(i59xFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59xFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	idx59 := s.arch.indicesOfFirst("biped-spartan-ability-non-predicted-state")
	if idx59 < 0 {
		t.Fatalf("i59 absent de l'archétype — composants : %v", s.arch.Components)
	}
	t.Logf("composant %d du registre biped : %q", idx59, s.arch.component(idx59))
	slotRanks := eaSlotRanks(t, dir)

	samples, records, with59, read, unread := i59Scan(s, idx59)
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	cov := 100 * float64(read) / float64(max(with59, 1))
	t.Logf("RECORDS delta biped %d · masque∋i59 %d (%.3f %%) · LUS %d (%.1f %% des annonces) "+
		"· illisibles %d", records, with59, 100*float64(with59)/float64(records), read, cov, unread)
	if len(samples) == 0 {
		t.Log("VERDICT : aucune lecture d'i59 sur ce film — rien à croiser")
		return
	}
	i59xLogTags(t, samples)

	reads := i59xReadsBySlot(samples)
	for _, tag := range i59xTags {
		trans := i59xTransitionsToTag(samples, tag)
		t.Logf("──────── TAG %d ────────", tag)
		xrTable(t, i59xSpec(tag), slotRanks, reads, trans)
	}
	t.Log("LECTURE DU RÉSULTAT (barre posée avant la mesure) : un tag ne discrimine que si " +
		"ses transitions se concentrent sur son rang cible ET sont ~nulles sur les autres " +
		"rangs ET sur les vies sans identité. Présent partout à volume comparable = ÉTAT " +
		"GÉNÉRIQUE, verdict négatif type i57. CONTRÔLE : le tag 3 DOIT ressortir sur le " +
		"groupe GRAPPIN — sinon c'est la méthode qui est en cause, pas les tags.")
}

// i59xLogTags publie l'histogramme des tags et le nombre de vies porteuses de chacun. Un tag
// porté par presque toutes les vies est déjà suspect d'être un état générique, avant même le
// tableau d'exclusivité.
func i59xLogTags(t *testing.T, samples []i59Sample) {
	t.Helper()
	hist := map[uint32]int{}
	slotsPerTag := map[uint32]map[uint32]bool{}
	allSlots := map[uint32]bool{}
	for _, sm := range samples {
		hist[sm.tag]++
		if slotsPerTag[sm.tag] == nil {
			slotsPerTag[sm.tag] = map[uint32]bool{}
		}
		slotsPerTag[sm.tag][sm.slot] = true
		allSlots[sm.slot] = true
	}
	t.Logf("tags R(2) : %s · %d vies portent au moins une lecture", i48RenderU32(hist), len(allSlots))
	for _, tag := range i59xTags {
		t.Logf("  tag %d : %4d lectures · %3d vies porteuses (%.0f %% des vies lues)",
			tag, hist[tag], len(slotsPerTag[tag]),
			100*float64(len(slotsPerTag[tag]))/float64(max(len(allSlots), 1)))
	}
}

// i59xReadsBySlot compte les lectures d'i59 par vie — le dénominateur du tableau.
func i59xReadsBySlot(samples []i59Sample) map[uint32]int {
	out := map[uint32]int{}
	for _, sm := range samples {
		out[sm.slot]++
	}
	return out
}

// i59xTransitionsToTag compte, par vie, les PASSAGES vers le tag demandé : la lecture
// précédente de la même vie portait un autre tag. La première lecture d'une vie n'est jamais
// une transition — on ne sait pas ce qui la précédait.
func i59xTransitionsToTag(samples []i59Sample, tag uint32) map[uint32]int {
	series := map[uint32][]i59Sample{}
	for _, sm := range samples {
		series[sm.slot] = append(series[sm.slot], sm)
	}
	out := map[uint32]int{}
	for slot, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].tsUS < ss[b].tsUS })
		for i := 1; i < len(ss); i++ {
			if ss[i].tag == tag && ss[i-1].tag != tag {
				out[slot]++
			}
		}
	}
	return out
}
