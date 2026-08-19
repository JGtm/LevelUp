// Package archlint — no_raw_kill_scope_literal_test.go : ratchet J4R-3.
//
// Interdit tout littéral brut de PORTÉE de `shared.match_kill_events` (`"kill-feed"`,
// `"credit-seul"`, `"highlight-events"` en position de valeur `read_path` / `read_origin`) hors
// de son propriétaire `internal/domain/killscope`.
//
// POURQUOI CE TEST EXISTE — il répare une promesse non tenue. Le commentaire de
// `migration/steps_shared_kill_events_from_pairs.go` annonçait que sa copie privée des deux
// valeurs était « verrouillée par un test d'égalité — cf.
// steps_shared_kill_events_from_pairs_test.go ». Ce fichier n'a jamais existé (constat J4R-3,
// relevé par DEUX relecteurs indépendants). La copie était donc libre de dériver.
//
// CE QUE LA DÉRIVE COÛTERAIT, ET POURQUOI ELLE SERAIT MUETTE : c'est sur `read_path` que se
// décide la PRÉSÉANCE entre producteurs (le film gagne, toujours). Une copie qui diverge d'un
// caractère ne lève aucune erreur — le filtre de préséance cesse simplement de reconnaître les
// passes de l'autre écrivain, et un producteur crédit réécrit par-dessus une passe de film. Le
// nombre de lignes ne bouge pas, aucun nom ne change : seule la source du dégât fatal disparaît
// de la lecture.
//
// Ce ratchet couvre AUSSI le retour en arrière que `killcollector.covertParUnFilm` documente
// (J4R-4-bis) : la préséance se testait autrefois EN NÉGATIF (`read_path <> 'highlight-events'`),
// donc toute voie autre que la sienne passait pour un film. Écrire une telle comparaison exige
// un littéral — que ce test refuse.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// killScopeAllowlist : UNE entrée, un FAUX POSITIF, et elle doit disparaître.
//
// Contrairement aux ratchets de dette (allowlist décroissante), il n'y a rien à migrer ici : les
// quatre écrivains lisent déjà `domain/killscope`. Une entrée ajoutée ici serait une seconde
// source de vérité pour une valeur qui décide de la préséance — la justifier par écrit ET par
// une date, ou ne pas l'ajouter.
//
// `cmd/variant-probe/main.go` (ajouté le 2026-08-20) : ce n'est PAS une seconde source de
// vérité. Le fichier ne touche ni `match_kill_events`, ni `read_path`, ni la préséance : son
// unique occurrence est le NOM d'un drapeau CLI — `flag.String("scan", ...)` — qui déclare le
// mode hors ligne d'une sonde réseau ponctuelle écrite par une AUTRE session. La collision est
// purement textuelle avec le mot `scan`, entré au ratchet le 2026-08-03 comme voie de film. Le
// ratchet lui-même ne bouge pas : ni le motif, ni les propriétaires, ni la marche du walk — seul
// ce fichier-là sort du périmètre, et le self-check ci-dessous interdit que l'entrée survive à
// sa cause.
// CONDITION DE REPRISE (à router vers la session variant-probe) : suppression de l'outil (la
// voie API qu'il sondait est fermée depuis le 2026-08-19) ou renommage du drapeau — dans les
// deux cas l'entrée DISPARAÎT.
var killScopeAllowlist = map[string]bool{
	"cmd/variant-probe/main.go": true,
}

// killScopeRE matche les valeurs de portée en littéral Go OU SQL (simples quotes).
//
// Les chaînes crédit sont distinctives : elles n'apparaissent pas dans du texte courant. La prose
// française du dépôt écrit « kill-feed » sans guillemets droits (commentaires) — non matchée,
// puisque les lignes de commentaire sont sautées avant le test.
//
// LES DEUX VOIES DE FILM (`marche`, `scan`) SONT ENTRÉES DANS LE RATCHET LE 2026-08-03, avec
// l'inversion de préséance (dette H3 refermée). Elles étaient hors périmètre au motif que « ce
// sont des mots courants, un grep produirait du bruit » — c'est vrai d'un grep, pas de ce test :
// il exige le littéral ENTIER entre quotes, donc `"scan %s: %w"` ou `"marche arrière"` ne
// matchent pas. Le propriétaire typé reste `killsource.Path` ; `killscope` en porte la seule
// autre copie, verrouillée par `killcollector.TestFilmReadPathsEgalesAuDecodeur`.
var killScopeRE = regexp.MustCompile(
	`["']((kill-feed)|(credit-seul)|(highlight-events)|(marche)|(scan))["']`)

// killScopeOwners : les deux paquets qui ont le droit d'écrire ces valeurs.
//
// `killscope` est le propriétaire des chaînes nues ; `killsource` est celui du type `Path` (le
// décodeur nomme ses propres voies, et il est title-specific — `persist` et `migration` ne
// peuvent pas l'importer, c'est toute la raison d'être de `killscope`). Leur ÉGALITÉ est
// verrouillée par un test, pas par ce ratchet.
var killScopeOwners = []string{
	"internal/domain/killscope/",
	"internal/games/halo_infinite/film/killsource/",
}

func TestNoRawKillScopeLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal
	apiRoot := filepath.Dir(internalRoot)                // .../apps/go-api

	var violations []string
	for _, racine := range []string{internalRoot, filepath.Join(apiRoot, "cmd")} {
		err := filepath.WalkDir(racine, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, _ := filepath.Rel(apiRoot, path)
			rel = filepath.ToSlash(rel)
			// Les propriétaires des valeurs, et eux seuls, ont le droit de les écrire.
			for _, owner := range killScopeOwners {
				if strings.HasPrefix(rel, owner) {
					return nil
				}
			}
			if killScopeAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				// `--` : les DDL du dépôt sont des chaînes brutes Go dont les commentaires SQL
				// citent les valeurs pour documenter la colonne. C'est de la prose, pas une
				// seconde source de vérité — même traitement que `//`.
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") ||
					strings.HasPrefix(trimmed, "--") {
					continue
				}
				if killScopeRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", racine, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("littéral brut de portée `match_kill_events` interdit (J4R-3) — utiliser "+
			"`killscope.ReadPathLiveFeed` / `killscope.ReadPathCreditBackfill` / "+
			"`killscope.OriginCreditOnly` (paquet `internal/domain/killscope`, feuille sans "+
			"import : importable depuis persist, migration, killcollector et ops sans cycle). "+
			"Une copie qui dérive d'un caractère rend la préséance film aveugle, SANS erreur "+
			"ni compteur :\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestKillScopeAllowlistEntriesStayJustified (self-check, même leçon V4d/VF-6 que
// `TestHalowaypointAllowlistEntriesPointToExistingFiles`) — chaque clé de killScopeAllowlist
// doit désigner un fichier EXISTANT qui matche RÉELLEMENT le motif. Une entrée dont le fichier
// a disparu, ou dont le littéral a été retiré, est un trou latent : un fichier recréé à ce
// chemin pourrait écrire une vraie valeur de portée sans déclencher le ratchet. C'est ce test
// qui garantit que l'exception du 2026-08-20 s'efface d'elle-même le jour où sa cause s'en va.
func TestKillScopeAllowlistEntriesStayJustified(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	for rel := range killScopeAllowlist {
		data, err := os.ReadFile(filepath.Join(apiRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("killScopeAllowlist : entrée %q pointe un fichier inexistant (%v) — sa cause "+
				"a disparu, retirer l'entrée (un fichier recréé à ce chemin échapperait au "+
				"ratchet).", rel, err)
			continue
		}
		if !killScopeRE.Match(data) {
			t.Errorf("killScopeAllowlist : entrée %q ne matche plus le motif de portée — le "+
				"littéral a été retiré ou renommé, retirer l'entrée (allowlist décroissante, "+
				"cible : VIDE).", rel)
		}
	}
}
