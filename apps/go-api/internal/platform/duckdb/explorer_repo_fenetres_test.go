package duckdb

// explorer_repo_fenetres_test.go — LES LECTURES DE L'EXPLORER (onglet « Joueur ») SONT PAYÉES À
// LEUR PÉRIMÈTRE (ADR 0036 I1 et I2).
//
//  1. Frags échangés (GetKillerVictimBetween), assistances (GetRelationAssists) et duels
//     (GetRivalTimeline sur les matchs communs) bornent chaque fenêtre `_latest` du kill-feed aux
//     matchs de la lecture : rouge si une fenêtre voit une ligne d'un match où le joueur n'a pas
//     joué (le défaut ne change aucun chiffre ; seul le nombre de lignes vues le montre).
//  2. La recherche d'un joueur par son nom (ResolveXUIDByGamertag) rend le xuid que rendrait la vue
//     des noms, et ne l'évalue pas quand un alias ou un gamertag de participant porte ce nom.

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestExplorerRepo_LecturesDuJoueur_FenetresBornees : quatre matchs à moi sur dix ; chaque
// fenêtre des trois lectures ne voit que les lignes de ces quatre-là, et les nombres sont ceux du
// corpus.
func TestExplorerRepo_LecturesDuJoueur_FenetresBornees(t *testing.T) {
	b := newBaseNotee(t)
	mesMatchs := seedFenetresTactiques(t, b.pdb, matchsDuJoueurFenetres)
	seedMatchsDesAutres(t, b.pdb, matchsDesAutresFenetres)
	ctx := context.Background()
	borne := matchsDuJoueurFenetres * mortsParMatchFenetres
	b.carnet.vider()

	kv, err := NewExplorerRepo(b.pdb, tacXUIDMoi).GetKillerVictimBetween(ctx, tacXUIDMoi, tacXUIDAdv)
	if err != nil {
		t.Fatalf("GetKillerVictimBetween: %v", err)
	}
	// Par match : moi → adv, adv → moi, moi → adv.
	if kv.KillsDealt != 2*matchsDuJoueurFenetres || kv.DeathsSuffered != matchsDuJoueurFenetres {
		t.Fatalf("frags échangés = %+v, want %d infligés / %d subis", kv, 2*matchsDuJoueurFenetres, matchsDuJoueurFenetres)
	}
	exigerFenetresBornees(t, b, "GetKillerVictimBetween", borne, 1)

	// Parité avec la requête libre, y compris pour un joueur que le kill-feed nomme sans qu'il
	// figure parmi les participants (le cas existe sur la base réelle).
	tacKill(t, b.pdb, mesMatchs[0], tacXUIDMoi, "x_hors_participants", 9000, true)
	for _, autre := range []string{tacXUIDAdv, "x_hors_participants"} {
		borneLue, err := NewExplorerRepo(b.pdb, tacXUIDMoi).GetKillerVictimBetween(ctx, tacXUIDMoi, autre)
		if err != nil {
			t.Fatalf("GetKillerVictimBetween(%s): %v", autre, err)
		}
		var libre struct{ dealt, suffered int }
		if err := b.brute.QueryRowContext(ctx, QKillsBetweenPlayers, tacXUIDMoi, autre, autre, tacXUIDMoi).
			Scan(&libre.dealt, &libre.suffered); err != nil {
			t.Fatalf("QKillsBetweenPlayers(%s): %v", autre, err)
		}
		if borneLue.KillsDealt != libre.dealt || borneLue.DeathsSuffered != libre.suffered {
			t.Fatalf("frags échangés avec %s : bornés %+v, libres %+v", autre, borneLue, libre)
		}
	}
	b.carnet.vider()
	borne++ // la mort hors participants posée ci-dessus est dans un de mes matchs

	repo := NewCareerRepo(b.pdb)
	if _, err := repo.GetRelationAssists(ctx, nil); err != nil {
		t.Fatalf("GetRelationAssists: %v", err)
	}
	exigerFenetresBornees(t, b, "GetRelationAssists", borne, 1)

	duels, err := repo.GetRivalTimeline(ctx, tacXUIDAdv, mesMatchs, 20)
	if err != nil {
		t.Fatalf("GetRivalTimeline: %v", err)
	}
	if len(duels) != matchsDuJoueurFenetres {
		t.Fatalf("duels = %d, want %d (un par match commun en ennemi)", len(duels), matchsDuJoueurFenetres)
	}
	exigerFenetresBornees(t, b, "GetRivalTimeline (matchs communs)", borne, 1)
}

// TestExplorerRepo_ResolveXUIDByGamertag_CommeLaVue : pour chaque niveau de la cascade des noms,
// la résolution rend le xuid de la vue ; la vue n'est lue que pour les niveaux que ni l'alias ni
// le participant ne portent.
func TestExplorerRepo_ResolveXUIDByGamertag_CommeLaVue(t *testing.T) {
	b := newBaseNotee(t)
	ctx := context.Background()
	tacMatch(t, b.pdb, "n01", tacCarteA, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))
	for _, s := range []string{
		// Alias : le nom que la vue retient, même si un participant porte un autre nom.
		`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('x_alias', 'Alpha Nom'), ('x_renomme', 'Nouveau Nom'), ('x_vide', '')`,
		`INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome) VALUES
			('n01', 'x_alias', 'Alpha Ancien', 0, 2),
			('n01', 'x_renomme', 'Ancien Nom', 0, 2),
			('n01', 'x_vide', 'Bravo Nom', 1, 3),
			('n01', 'x_part', 'Charlie Nom', 1, 3),
			('n01', 'bid(42.0)', 'Bot Nom', 1, 3)`,
	} {
		tacExec(t, b.pdb, s)
	}
	// Un joueur que seul le kill-feed nomme.
	tacKillNomme(t, b.pdb, "n01", "x_feed", "Delta Nom", "x_part", "Charlie Nom", 1000, nil, nil)

	repo := NewExplorerRepo(b.pdb, tacXUIDMoi)
	cas := []struct {
		nom, motif, want string
		vueLue           bool
	}{
		{"alias", "alpha nom", "x_alias", false},
		{"alias vide, participant", "bravo nom", "x_vide", false},
		{"participant seul", "Charlie Nom", "x_part", false},
		{"nom périmé (alias plus récent)", "Ancien Nom", "", true},
		{"kill-feed seul", "Delta Nom", "x_feed", true},
		{"bot", "Bot Nom", "", true},
		{"inconnu", "Zoulou", "", true},
	}
	for _, c := range cas {
		b.carnet.vider()
		got, err := repo.ResolveXUIDByGamertag(ctx, c.motif)
		if c.want == "" {
			if !errors.Is(err, sql.ErrNoRows) {
				t.Errorf("%s : (%q, %v), want sql.ErrNoRows", c.nom, got, err)
			}
		} else if err != nil || got != c.want {
			t.Errorf("%s : (%q, %v), want %q", c.nom, got, err, c.want)
		}
		if vue := vueDesNomsLue(b.carnet.vider()); vue != c.vueLue {
			t.Errorf("%s : vue des noms lue = %v, want %v", c.nom, vue, c.vueLue)
		}
		var parLaVue string
		errVue := b.brute.QueryRowContext(ctx, `SELECT xuid FROM v_gamertag_lookup
			WHERE gamertag ILIKE ? AND xuid NOT LIKE 'bid(%' LIMIT 1`, c.motif).Scan(&parLaVue)
		if (errVue == nil) != (c.want != "") || parLaVue != c.want {
			t.Errorf("%s : la vue rend (%q, %v), le cas attend %q — le corpus ne teste plus la parité", c.nom, parLaVue, errVue, c.want)
		}
	}
}

// vueDesNomsLue : une des requêtes notées lit v_gamertag_lookup.
func vueDesNomsLue(requetes []requeteNotee) bool {
	for _, r := range requetes {
		if strings.Contains(r.sql, "v_gamertag_lookup") {
			return true
		}
	}
	return false
}
