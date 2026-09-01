// Package duckdb — temoin_bascule_couverture_test.go : les trois mesures qui rendent le
// temoin de bascule LISIBLE plutot que seulement spectaculaire.
//
//  1. La COUVERTURE de l'ancienne chaine sur le lot. Un residu « Non attribue » de 77 %
//     ne dit pas la meme chose selon que `weapon_kills` porte 80 lignes par match ou 14 :
//     dans le premier cas la correlation echoue, dans le second elle n'a simplement pas
//     tourne. Sans ce nombre, la comparaison serait un proces d'intention.
//
//  2. La SYNTHESE PAR CLASSE des tags qui rendent une image sans rendre de cle (A0.4). Le
//     detail tag par tag est necessaire pour le garde-rail A1.9 ; la synthese est
//     necessaire pour DECIDER, parce qu'elle separe ce qui est un trou (vehicules) de ce
//     qui est un choix (melee et grenade, servies par les compteurs API).
//
//  3. L'ARBITRAGE DE LA MELEE. La decision D4 ecarte les cles de classe registre `melee`
//     au motif que les compteurs API les servent deja. Encore faut-il que ce soit vrai :
//     si le compteur `melee_kills` de l'API ne compte QUE les coups de crosse, alors
//     ecarter l'epee et le marteau ne les rend pas aux compteurs — ca les jette dans
//     « Non attribue ». Cette mesure tranche, et elle ne peut etre faite qu'ici.
package duckdb

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

// ecrireCouvertureAncienne compte ce que `weapon_kills` porte reellement sur le lot.
func ecrireCouvertureAncienne(r *temoinRapport, t *testing.T, b *temoinBase, s temoinScope,
	total int,
) {
	t.Helper()
	args := make([]any, 0, len(s.matchIDs))
	for _, id := range s.matchIDs {
		args = append(args, id)
	}
	var lignes, matchs, horsSentinelles, sansID int
	err := b.pdb.Shared.QueryRow(context.Background(), `
SELECT COUNT(*)::INTEGER,
       COUNT(DISTINCT wk.match_id)::INTEGER,
       COUNT(*) FILTER (WHERE wk.effective_weapon_id NOT IN (0,1,2))::INTEGER,
       COUNT(*) FILTER (WHERE wk.effective_weapon_id IS NULL)::INTEGER
FROM v_weapon_kills wk
WHERE wk.match_id IN (`+Placeholders(len(s.matchIDs))+`)`, args...).
		Scan(&lignes, &matchs, &horsSentinelles, &sansID)
	if err != nil {
		t.Fatalf("couverture weapon_kills : %v", err)
	}
	r.ligne("## Couverture de l'ancienne chaine sur le lot")
	r.ligne("")
	r.ligne("| grandeur | valeur |")
	r.ligne("|---|---:|")
	r.ligne("| lignes `weapon_kills` | %d |", lignes)
	r.ligne("| matchs portant au moins une ligne | %d / %d |", matchs, len(s.matchIDs))
	r.ligne("| lignes hors sentinelles (0/1/2) | %d |", horsSentinelles)
	r.ligne("| lignes sans identifiant d'arme | %d |", sansID)
	r.ligne("| lignes par match | %.1f |", moyenne(lignes, len(s.matchIDs)))
	r.ligne("| lignes rapportees au total de frags API | %.1f %% |", pourcent(lignes, total))
	r.ligne("")
}

// moyenne : quotient protege, 0 si le diviseur est nul.
func moyenne(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / float64(d)
}

// ecrireSyntheseDivergence regroupe par classe `damagetag` les tags qui rendent une image
// sans rendre de cle. C'est la lecture qui separe le trou du choix.
func ecrireSyntheseDivergence(r *temoinRapport, tags []temoinTag) {
	parClasse := map[damagetag.Class][2]int{} // classe -> {tags, morts}
	for _, e := range tags {
		if e.image == "" || e.cle != "" {
			continue
		}
		v := parClasse[e.classe]
		parClasse[e.classe] = [2]int{v[0] + 1, v[1] + e.morts}
	}
	r.ligne("### Synthese par classe des tags « image sans cle »")
	r.ligne("")
	if len(parClasse) == 0 {
		r.ligne("_aucun._")
		r.ligne("")
		return
	}
	classes := make([]damagetag.Class, 0, len(parClasse))
	for c := range parClasse {
		classes = append(classes, c)
	}
	sort.Slice(classes, func(i, j int) bool {
		return parClasse[classes[i]][1] > parClasse[classes[j]][1]
	})
	r.ligne("| classe damagetag | tags | morts | lecture |")
	r.ligne("|---|---:|---:|---|")
	for _, c := range classes {
		r.ligne("| %s | %d | %d | %s |", c, parClasse[c][0], parClasse[c][1], lectureClasse(c))
	}
	r.ligne("")
}

// lectureClasse dit, en une phrase, si l'absence de cle est un trou ou un choix.
func lectureClasse(c damagetag.Class) string {
	switch c {
	case damagetag.ClassMelee:
		return "CHOIX — servie par le compteur API `melee_kills` (D4)"
	case damagetag.ClassGrenade:
		return "CHOIX — servie par le compteur API `grenade_kills` (D4)"
	case damagetag.ClassVehicule:
		return "TROU — le rejeu affiche l'engin, le graphe dirait « Non attribue » (D13)"
	case damagetag.ClassArme:
		return "TROU — arme nommee par le film, absente du registre (D13)"
	default:
		return "a instruire"
	}
}

// ecrireArbitrageMelee confronte le compteur API `melee_kills` aux deux populations de
// morts que la source de degat sait distinguer : la melee NUE (classe damagetag MELEE,
// aucune cle de registre) et la melee D'ARME (cle de registre de classe `melee` — epee,
// marteau).
//
// Lecture : si `melee_kills` est du meme ordre que la melee NUE seule, alors le compteur
// API n'inclut PAS l'epee et le marteau, et la decision D4 — appliquee telle quelle a
// TOUTES les cles de classe registre `melee` — jetterait ces morts dans « Non attribue »
// au lieu de les rendre aux compteurs.
func ecrireArbitrageMelee(r *temoinRapport, tags []temoinTag, reg map[string][2]string,
	counts domain.FragKillTypeCounts,
) {
	var meleeNue, meleeArme int
	parArme := map[string]int{}
	for _, e := range tags {
		if e.cle == "" {
			if e.classe == damagetag.ClassMelee {
				meleeNue += e.morts
			}
			continue
		}
		if m, ok := reg[e.cle]; ok && m[0] == domain.FragClassMelee {
			meleeArme += e.morts
			parArme[e.cle] += e.morts
		}
	}
	r.ligne("## Arbitrage de la melee — la decision D4 tient-elle pour l'epee et le marteau ?")
	r.ligne("")
	r.ligne("| population | morts |")
	r.ligne("|---|---:|")
	r.ligne("| compteur API `melee_kills` (autoritatif) | %d |", counts.Melee)
	r.ligne("| melee NUE vue par la source (classe MELEE, aucune cle) | %d |", meleeNue)
	r.ligne("| melee D'ARME vue par la source (cle de classe registre `melee`) | %d |", meleeArme)
	cles := make([]string, 0, len(parArme))
	for k := range parArme {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return parArme[cles[i]] > parArme[cles[j]] })
	for _, k := range cles {
		r.ligne("| dont %s | %d |", k, parArme[k])
	}
	r.ligne("")
	r.ligne("Ecart `melee_kills` moins melee NUE : %+d. Ecart `melee_kills` moins (nue + arme) : %+d.",
		counts.Melee-meleeNue, counts.Melee-meleeNue-meleeArme)
	r.ligne("")
}

// ecrireVentilationServie publie la ventilation telle que les six surfaces la servent
// APRES la bascule : lecteur adosse a la source de degat, passe au builder de production.
//
// Elle differe de la colonne « source de degat » de la comparaison : celle-ci est une
// mesure brute des tags, celle-la traverse `fragdist` et subit donc ses regles — totaux
// API pour melee et grenade, ventilation par role pour les armes a feu, residu calcule.
func ecrireVentilationServie(r *temoinRapport, d domain.FragDistribution,
	counts domain.FragKillTypeCounts,
) {
	r.ligne("## Ventilation SERVIE par les six surfaces apres la bascule")
	r.ligne("")
	r.ligne("| classe | frags | autoritatif | niveau 2 |")
	r.ligne("|---|---:|---|---|")
	for _, c := range d.Classes {
		r.ligne("| %s | %d | %t | %s |", c.Class, c.Kills, c.Authoritative, resumeRoles(c.Roles))
	}
	somme := 0
	for _, c := range d.Classes {
		somme += c.Kills
	}
	r.ligne("| **total** | %d | | |", somme)
	r.ligne("")
	r.ligne("Invariant : la somme des classes vaut le total de frags API (%d) — %t.",
		counts.Total, somme == counts.Total)
	r.ligne("")
}

// resumeRoles rend le niveau 2 d'une classe en une cellule lisible.
func resumeRoles(roles []domain.FragRoleEntry) string {
	if len(roles) == 0 {
		return "_feuille_"
	}
	parts := make([]string, 0, len(roles))
	for _, e := range roles {
		nom := e.Label
		if nom == "" {
			nom = e.Role
		}
		parts = append(parts, fmt.Sprintf("%s %d", nom, e.Kills))
	}
	return strings.Join(parts, ", ")
}
