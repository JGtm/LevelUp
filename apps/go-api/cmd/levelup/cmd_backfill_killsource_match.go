package main

// cmd_backfill_killsource_match.go — `--match` : LA PASSE HORS LIGNE BORNEE A QUELQUES MATCHS.
//
// Plan Emprise vies, V5.2a (2026-09-30). La passe entiere du parc local (~1 300 films, ~1 min par
// film et par ouvrier) tient le serveur arrete des heures ; `--limit` prend les films les MOINS
// CHERS, pas ceux d une soiree. `--match` nomme les matchs voulus : la SELECTION hors ligne
// (`filmsACollecter`) ne regarde plus qu eux. Le reste de la selection ne change pas — `matchsAJour`
// s applique toujours (sauf `--force`), le tri par cout aussi, `--limit` s applique apres le filtre.
//
// UN PREFIXE COURT N EST ACCEPTE QUE S IL EST UNIVOQUE : 8 caracteres et plus (la forme courte
// d un film, celle que les journaux et le cache affichent), designant UN SEUL match du registre.
// Un prefixe ambigu ou inconnu est REFUSE avec la liste des candidats — jamais ignore en silence,
// jamais elargi : une passe qui decoderait autre chose que ce qu on a nomme est pire qu une erreur.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// longueurMiniPrefixe : la longueur en dessous de laquelle un fragment n est pas un identifiant.
const longueurMiniPrefixe = 8

// nbCandidatsCites : combien de matchs un refus d ambiguite nomme.
const nbCandidatsCites = 5

// validerMatch : les incompatibilites de `--match`, AVANT d ouvrir quoi que ce soit.
//
// `--online` choisit ses films dans l historique du JOUEUR (API), pas dans le registre : le filtre
// n y aurait aucun sens et serait ignore. `--credit-only` ne joue que la passe SQL -> SQL, qui ne
// lit pas cette selection : meme raison — un drapeau accepte puis ignore ferait croire a une
// passe bornee.
func validerMatch(o killsourceOptions) error {
	if o.match == "" {
		return nil
	}
	if o.online {
		return fmt.Errorf("--match n a de sens que HORS LIGNE : --online choisit ses films dans " +
			"l historique du joueur, pas dans le registre")
	}
	if o.creditOnly {
		return fmt.Errorf("--match n a pas de sens avec --credit-only : la passe credit est SQL -> " +
			"SQL sur tout le registre, elle ne lit pas la selection des films")
	}
	if len(decouperMatchs(o.match)) == 0 {
		return fmt.Errorf("--match %q : aucun identifiant (liste separee par des virgules)", o.match)
	}
	return nil
}

// decouperMatchs : la liste separee par des virgules, sans blancs ni vides, sans doublons.
func decouperMatchs(brut string) []string {
	var out []string
	vus := map[string]bool{}
	for _, p := range strings.Split(brut, ",") {
		p = strings.TrimSpace(p)
		if p == "" || vus[strings.ToLower(p)] {
			continue
		}
		vus[strings.ToLower(p)] = true
		out = append(out, p)
	}
	return out
}

// resoudreMatchs : chaque fragment de `brut` designe UN match du registre, ou la fonction refuse.
//
// Un identifiant EXACT du registre gagne toujours ; sinon le fragment est un prefixe (casse
// ignoree) d au moins [longueurMiniPrefixe] caracteres qui doit designer exactement un match.
func resoudreMatchs(registre []string, brut string) (map[string]bool, error) {
	exact := make(map[string]string, len(registre))
	for _, id := range registre {
		exact[strings.ToLower(id)] = id
	}
	out := map[string]bool{}
	for _, frag := range decouperMatchs(brut) {
		bas := strings.ToLower(frag)
		if id, ok := exact[bas]; ok {
			out[id] = true
			continue
		}
		if len(frag) < longueurMiniPrefixe {
			return nil, fmt.Errorf("--match %q : %d caracteres, il en faut au moins %d "+
				"(identifiant complet ou prefixe court univoque)", frag, len(frag), longueurMiniPrefixe)
		}
		var candidats []string
		for _, id := range registre {
			if strings.HasPrefix(strings.ToLower(id), bas) {
				candidats = append(candidats, id)
			}
		}
		switch len(candidats) {
		case 0:
			return nil, fmt.Errorf("--match %q : aucun match du registre ne porte ce prefixe", frag)
		case 1:
			out[candidats[0]] = true
		default:
			return nil, fmt.Errorf("--match %q : prefixe AMBIGU (%d matchs : %s) — donner plus de caracteres",
				frag, len(candidats), citerCandidats(candidats))
		}
	}
	return out, nil
}

// citerCandidats : les premiers candidats d un prefixe ambigu, pour que le refus se suffise.
func citerCandidats(candidats []string) string {
	if len(candidats) <= nbCandidatsCites {
		return strings.Join(candidats, ", ")
	}
	return strings.Join(candidats[:nbCandidatsCites], ", ") + ", ..."
}

// registreDeLaPasse : les matchs du registre que la passe hors ligne considere — tous, ou ceux
// que `--match` nomme (dans l ordre stable du registre).
func registreDeLaPasse(ctx context.Context, db *sql.DB, o killsourceOptions) ([]string, error) {
	return registreBorne(ctx, db, o.match)
}

// registreBorne : la resolution de `--match` contre le registre, PARTAGEE par toutes les
// sous-commandes qui l offrent (backfill-killsource, backfill-vehicle-takes). Vide = tout le
// registre ; sinon chaque fragment designe UN match du registre ou la fonction refuse AVANT que
// l appelant ecrive quoi que ce soit (RV8 : une valeur prise telle quelle ecrivait en table
// append-only un identifiant qui n existe pas).
func registreBorne(ctx context.Context, db *sql.DB, match string) ([]string, error) {
	registre, err := matchsDuRegistre(ctx, db, 0)
	if err != nil || match == "" {
		return registre, err
	}
	if len(decouperMatchs(match)) == 0 {
		return nil, fmt.Errorf("--match %q : aucun identifiant (liste separee par des virgules)", match)
	}
	voulus, err := resoudreMatchs(registre, match)
	if err != nil {
		return nil, err
	}
	restreint := make([]string, 0, len(voulus))
	for _, id := range registre {
		if voulus[id] {
			restreint = append(restreint, id)
		}
	}
	return restreint, nil
}
