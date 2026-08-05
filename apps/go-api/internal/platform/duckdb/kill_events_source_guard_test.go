package duckdb

// kill_events_source_guard_test.go — LES TROIS GARDE-RAILS DE LA BASCULE DU 2026-08-03.
//
// Ils gardent trois régressions DISTINCTES, chacune constatée ou mesurée :
//
//  1. un lecteur de production qui repartirait sur `killer_victim_pairs` (46,5 % de doublons —
//     c'est la régression que toute la bascule existe pour empêcher) ;
//  2. une seconde copie de la requête « frags entre deux joueurs » (elle en avait déjà deux, et
//     deux copies finissent par afficher deux nombres différents pour le même duel) ;
//  3. un `COALESCE(xuid, '')` sur une lecture du kill-feed, qui fusionnerait toutes les morts
//     de bot en UN acteur de chaîne vide.
//
// Chacun a été vu ROUGE avant d'être commité (témoin de détection : réintroduction du motif
// interdit dans le fichier concerné, test rejoué, motif retiré).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sourcesDuPaquet lit les .go non-test du répertoire, plus le sous-paquet halo5 qui porte le
// lecteur de kill-feed Halo 5. Retourne (chemin relatif -> contenu).
func sourcesDuPaquet(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, dir := range []string{".", "halo5"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("lecture de %s : %v", dir, err)
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			p := filepath.Join(dir, name)
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("lecture de %s : %v", p, err)
			}
			out[filepath.ToSlash(p)] = string(b)
		}
	}
	if len(out) == 0 {
		t.Fatal("aucune source lue — le garde-rail ne garderait rien")
	}
	return out
}

// TestAucunLecteurNeLitLAncienneTable : plus aucun SELECT de ce paquet ne part de
// `killer_victim_pairs`. La table RESTE en base (elle est la base crédit des producteurs, cf.
// `PLAN_BRANCHEMENT_KILLSOURCE.md` §2.4) — c'est sa LECTURE PAR LE PRODUIT qui est terminée.
//
// Le paquet `duckdb` est la couche de lecture : aucune exception n'y est légitime. Le jour où
// il en faudrait une, elle s'écrit ici avec sa date et son motif, jamais en silence.
func TestAucunLecteurNeLitLAncienneTable(t *testing.T) {
	for path, src := range sourcesDuPaquet(t) {
		for _, motif := range []string{"FROM killer_victim_pairs", "JOIN killer_victim_pairs"} {
			if strings.Contains(src, motif) {
				t.Errorf("%s contient %q.\n"+
					"La couche de lecture ne lit plus `killer_victim_pairs` depuis le 2026-08-03 : "+
					"elle porte 46,5 %% de doublons exacts et gonflait les agrégats carrière d'un "+
					"facteur 1,879 (101 frags affichés pour 29 réels sur le duel de contrôle). "+
					"Lire `%s` — traduction des colonnes dans kill_events_source.go.",
					path, motif, KillEventsCanonicalTable)
			}
		}
	}
}

// TestUneSeuleRequeteDeFragsEntreDeuxJoueurs : le SQL de [QKillsBetweenPlayers] n'existe qu'une
// fois. Q19b (Explorer) et CompareRepo.GetEncounterStats en portaient deux copies textuelles.
func TestUneSeuleRequeteDeFragsEntreDeuxJoueurs(t *testing.T) {
	const signature = "AND victim_xuid = ?) AS kills_dealt"
	var porteurs []string
	for path, src := range sourcesDuPaquet(t) {
		if strings.Contains(src, signature) {
			porteurs = append(porteurs, path)
		}
	}
	if len(porteurs) != 1 || porteurs[0] != "kill_events_source.go" {
		t.Errorf("la requête « frags entre deux joueurs » est portée par %v, attendu "+
			"[kill_events_source.go] et rien d'autre.\n"+
			"Deux copies d'un même duel affichent tôt ou tard deux nombres différents sur deux "+
			"pages : passer par QKillsBetweenPlayers.", porteurs)
	}
}

// TestPasDeXuidNormaliseEnChaineVide : aucune lecture du kill-feed ne remplace un xuid absent
// par une chaîne vide.
//
// LE CAS N'EST PAS THÉORIQUE, ET IL A DEUX FACES. Sur Infinite, `killer_victim_pairs` ne portait
// AUCUN xuid NULL : les quatre `COALESCE(xuid, ”)` des lecteurs étaient des NO-OP, donc jamais
// observés. La canonique, elle, porte 973 lignes de bot en NULL — le COALESCE cesserait d'être
// inerte et les fondrait en un acteur unique qui frappe et meurt en permanence. Sur Halo 5,
// l'ancienne table code DÉJÀ ses bots en chaîne vide, et ce fantôme entrait dans le top-10
// némésis / souffre-douleur sous un libellé masqué (161 frags, 127 morts sur le joueur le plus
// actif) : la bascule le retire, ce garde-rail empêche de le faire revenir.
func TestPasDeXuidNormaliseEnChaineVide(t *testing.T) {
	interdits := []string{
		"COALESCE(killer_xuid, '')",
		"COALESCE(victim_xuid, '')",
		"COALESCE(feed_killer_xuid, '')",
		"COALESCE(kv.killer_xuid, '')",
		"COALESCE(kv.victim_xuid, '')",
		"COALESCE(kv.feed_killer_xuid, '')",
	}
	for path, src := range sourcesDuPaquet(t) {
		for _, motif := range interdits {
			if strings.Contains(src, motif) {
				t.Errorf("%s contient %q.\n"+
					"Un xuid absent désigne un BOT : le normaliser en chaîne vide fusionne toutes "+
					"les morts de bot en un seul acteur fantôme. Écarter la ligne "+
					"(`AND <colonne> IS NOT NULL`), ou la servir telle quelle si le lecteur sait "+
					"représenter l'absence.", path, motif)
			}
		}
	}
}
