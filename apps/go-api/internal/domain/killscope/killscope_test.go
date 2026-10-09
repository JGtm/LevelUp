package killscope

import "testing"

// TestValeursDeFilEpinglees — CES VALEURS SONT SUR DISQUE, EN PRODUCTION.
//
// Ce ne sont pas des identifiants internes : elles sont écrites en clair dans `read_path` /
// `read_origin` de `shared.match_kill_events`, une table append-only dont on ne réécrit pas
// l'historique. Changer une constante ici sans migrer les lignes déjà écrites ne casserait RIEN
// visiblement — la préséance cesserait simplement de reconnaître les anciennes passes, et un
// producteur crédit réécrirait par-dessus une passe de film. Sans erreur, sans compteur.
//
// D'où ce test : il n'a pas d'autre rôle que de rendre ce changement IMPOSSIBLE en silence.
func TestValeursDeFilEpinglees(t *testing.T) {
	cas := []struct {
		nom, got, want string
	}{
		{"ReadPathLiveFeed", ReadPathLiveFeed, "kill-feed"},
		{"ReadPathCreditBackfill", ReadPathCreditBackfill, "highlight-events"},
		{"OriginCreditOnly", OriginCreditOnly, "credit-seul"},
	}
	for _, c := range cas {
		if c.got != c.want {
			t.Errorf("%s = %q, attendu %q — cette valeur est écrite sur disque en production. "+
				"La changer orpheline les lignes déjà écrites : la préséance ne les reconnaît "+
				"plus et un producteur crédit peut écraser une passe de film, sans symptôme.",
				c.nom, c.got, c.want)
		}
	}
}

// TestLesDeuxVoiesRestentDistinctes — la préséance se décide sur `read_path`.
//
// Deux voies devenues égales (copier-coller, renommage hâtif) rendraient les deux producteurs
// crédit indiscernables : le filtre de préséance ne pourrait plus dire lequel a écrit la passe
// courante. Le test coûte une ligne et ferme un mode de panne muet.
func TestLesDeuxVoiesRestentDistinctes(t *testing.T) {
	if ReadPathLiveFeed == ReadPathCreditBackfill {
		t.Fatalf("les deux voies crédit valent toutes deux %q — elles ne sont plus discernables, "+
			"et la préséance se décide sur cette colonne", ReadPathLiveFeed)
	}
}
