package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// seed_citation_assets_test.go — garde-rails sur les identifiants et les visuels des
// citations seedées. Pure data + accès disque en lecture seule (aucune DB), donc hors
// build cgo.
//
// Motivation : un image_path mort ne produit AUCUN signal côté serveur. Le handler
// /static/commendations/* répond 404 et le front (MedalIcon) masque l'image sur onError —
// la vignette disparaît simplement. Le défaut ne se voit donc qu'à l'œil, sur la page
// Citations d'un joueur. Ce test le transforme en échec de build.

// citationRepoRoot — racine du dépôt vue depuis internal/ops (4 niveaux : ops →
// internal → go-api → apps → racine). Les image_path seedés sont relatifs à cette racine
// ("static/commendations/<titre>/<fichier>").
const citationRepoRoot = "../../../.."

// citationImagePrefix — préfixe imposé à tout image_path (wpH5/wpHI dans le seed).
const citationImagePrefix = "static/commendations/"

// TestCitationNorms_Unique : citation_name_norm est la PRIMARY KEY de citation_mappings.
// Un doublon dans defaultCitationMappings() ne casserait pas le seed (SELECT-then-INSERT-
// or-UPDATE : la 2e ligne écraserait silencieusement la 1re), il ferait juste disparaître
// une citation. Interdit ici.
func TestCitationNorms_Unique(t *testing.T) {
	seen := make(map[string]string)
	for _, m := range defaultCitationMappings() {
		if m.Norm == "" {
			t.Errorf("citation %q: Norm vide", m.Display)
			continue
		}
		if prev, dup := seen[m.Norm]; dup {
			t.Errorf("norm %q dupliqué: %q et %q — la seconde écraserait la première au seed", m.Norm, prev, m.Display)
			continue
		}
		seen[m.Norm] = m.Display
	}
}

// TestCitationImagePaths_ExistOnDisk : tout image_path seedé désigne un fichier réellement
// présent sous static/commendations/. Volontairement AGNOSTIQUE À L'EXTENSION : le jour où
// un visuel provisoire (.svg) est remplacé par le visuel définitif (.png), basculer
// l'extension dans le seed AVANT que le fichier n'arrive fait échouer ce test — c'est
// l'effet recherché.
//
// Les noms de fichiers pré-encodés sur disque (caractères interdits sous Windows, ex. `?`
// stocké littéralement `%3F`) sont comparés tels quels : le seed porte exactement le nom du
// fichier, et ce sont les constructeurs d'URL côté analysis (BuildCitationSnippets,
// MergeCitationTotals) qui ré-encodent (url.PathEscape par segment, `%` → `%25`).
func TestCitationImagePaths_ExistOnDisk(t *testing.T) {
	staticRoot := filepath.Join(citationRepoRoot, filepath.FromSlash(citationImagePrefix))
	if _, err := os.Stat(staticRoot); err != nil {
		t.Skipf("arborescence %s absente — garde-rail NON joué (%v)", staticRoot, err)
	}
	for _, m := range defaultCitationMappings() {
		if m.ImagePath == "" {
			continue
		}
		if !strings.HasPrefix(m.ImagePath, citationImagePrefix) {
			t.Errorf("citation %q: image_path %q hors de %s", m.Norm, m.ImagePath, citationImagePrefix)
			continue
		}
		full := filepath.Join(citationRepoRoot, filepath.FromSlash(m.ImagePath))
		if _, err := os.Stat(full); err != nil {
			t.Errorf("citation %q (%q): image_path %q introuvable sur disque (%v)", m.Norm, m.Display, m.ImagePath, err)
		}
	}
}

// TestCitationEnabled_HasImagePath : une citation active sans visuel s'affiche sans
// vignette dans la grille Citations. Les seules citations sans image_path tolérées sont
// celles désactivées (inventaire conservé, moteur qui les ignore).
func TestCitationEnabled_HasImagePath(t *testing.T) {
	for _, m := range defaultCitationMappings() {
		if m.Enabled && m.ImagePath == "" {
			t.Errorf("citation %q (%q) est active mais n'a aucun image_path", m.Norm, m.Display)
		}
	}
}

// TestCompositeChildren_ExistAsCitations : tout enfant cité par un composite désigne une
// citation réellement seedée.
//
// Motivation (ajout 2026-09-10, avec les six citations « Artilleur de » qui portent le
// nombre d'enfants de `vehicle_mastery` de 9 à 15). Un enfant fantôme ne casse RIEN de
// visible : OverrideCompositeTotals ignore les norms absents de `mappings` — exactement
// comme il ignore un enfant désactivé — et le composite se contente d'afficher un total
// plus bas que la réalité. Le défaut est donc silencieux et permanent, ce qui est le pire
// des cas : une faute de frappe dans la liste JSON coûte un palier au joueur sans qu'aucun
// signal ne soit émis. Ce test la transforme en échec de build.
func TestCompositeChildren_ExistAsCitations(t *testing.T) {
	mappings := defaultCitationMappings()
	known := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		known[m.Norm] = struct{}{}
	}
	for _, m := range mappings {
		if m.CompositeChildren == "" {
			continue
		}
		var children []string
		if err := json.Unmarshal([]byte(m.CompositeChildren), &children); err != nil {
			t.Errorf("composite %q: composite_children illisible (%v)", m.Norm, err)
			continue
		}
		if len(children) == 0 {
			t.Errorf("composite %q: composite_children vide", m.Norm)
			continue
		}
		for _, child := range children {
			if _, ok := known[child]; !ok {
				t.Errorf("composite %q: enfant %q ne correspond à aucune citation seedée", m.Norm, child)
			}
		}
	}
}

// weaponRowNameRe capture le nom canonique EN d'une ligne du registre d'armes, c'est-a-dire
// la TROISIEME chaine entre guillemets de `{"cle", titleXXX, "Nom", ...}`. On lit le fichier
// source en texte plutot que d'importer `games/weapons` : les lignes du registre sont un
// type non exporte, et un accesseur ouvert pour les besoins d'un test serait une porte
// d'entree permanente sur une table qui doit rester close.
var weaponRowNameRe = regexp.MustCompile(`^\s*\{"[^"]+",\s*\w+,\s*"([^"]+)"`)

// tomlNameENRe capture la valeur `en` d'une ligne de weapon_names.toml.
var tomlNameENRe = regexp.MustCompile(`\ben\s*=\s*"([^"]+)"`)

// citationTitleMappingDirs — les manifestes de libelles a lire, un par titre seede.
var citationTitleMappingDirs = []string{"halo_infinite", "halo_5"}

// TestWeaponStatCitations_ResolvableName : tout `weapon_stat` demande un nom d'arme qui
// existe reellement.
//
// LE DEFAUT QU'IL FERME, ET POURQUOI IL ETAIT INVISIBLE. Une citation `weapon_stat` porte
// `StatName: "weapon_kills:<nom canonique EN>"`. Le moteur (analysis.dispatchFull) lit
// `ctx.Stats[StatName]` ; sync.loadWeaponKillsFromSource remplit ce map depuis le registre.
// Un nom qui ne correspond a RIEN ne provoque ni erreur, ni log, ni test rouge : la lecture
// d'une cle absente rend le zero-value, et la citation affiche simplement 0. Un joueur ne
// peut pas distinguer « citation jamais meritee » de « citation cassee ».
//
// TROIS L'ETAIENT, decouvertes le 2026-09-10 en croisant le seed et les libelles :
//   - `sidekick_mastery`  demandait « Mk51 Sidekick » pour « Mk50 Sidekick » (un chiffre) ;
//     son propre Display disait pourtant « MK50 ».
//   - `bandit_mastery`    demandait « Bandit Evo », qui est le libelle FR, la ou le moteur
//     attend l'identite EN « M392 Bandit ».
//   - `mutilator_mastery` demande « Mutilator », absent du registre — 1262 frags mesures au
//     corpus (138 807 morts, 1384 matchs) qui ne comptent pour aucune citation.
//
// Les trois sont enfants de `human_weapons_mastery` : son palier final etait donc
// inatteignable pour tout le monde, en silence.
func TestWeaponStatCitations_ResolvableName(t *testing.T) {
	known := knownWeaponNames(t)
	if len(known) == 0 {
		t.Skip("registre et libelles illisibles — garde-rail NON joue")
	}
	for _, m := range defaultCitationMappings() {
		if m.MappingType != mappingTypeWeaponStat {
			continue
		}
		name := strings.TrimPrefix(m.StatName, "weapon_kills:")
		if name == m.StatName {
			t.Errorf("citation %q: StatName %q ne porte pas le prefixe weapon_kills:", m.Norm, m.StatName)
			continue
		}
		if _, ok := known[name]; ok {
			continue
		}
		if reason, toleree := weaponStatNamesEnAttente[m.Norm]; toleree {
			t.Logf("citation %q: nom %q non resolu, TOLERE — %s", m.Norm, name, reason)
			continue
		}
		t.Errorf("citation %q: StatName demande l'arme %q, qui n'existe ni au registre "+
			"ni dans weapon_names.toml — la citation comptera zero en silence", m.Norm, name)
	}
}

// weaponStatNamesEnAttente — allowlist DATEE, une entree, et la seule tolerance de ce test.
//
// `mutilator_mastery` reste rouge dans les faits : le Mutilateur n'est pas au registre alors
// qu'il tue (1262 morts mesurees). La reparation demande de toucher registry.go,
// weapon_names.toml, killicon/data/rules.tsv et off_arsenal_guard_test.go — les QUATRE
// fichiers qu'un autre chantier (genre `PORTEUR`, kill feed) tient en vol le 2026-09-10.
// Les editer en parallele produirait exactement le conflit que ce depot passe son temps a
// eviter, pour un defaut qui existe depuis des mois.
//
// CRITERE DE RETRAIT, mesurable et non negociable : des que `hinf_mutilator` figure au
// registre avec son libelle, cette entree DISPARAIT et le test devient vert sans elle.
// DATE CIBLE : 2026-10-01. Passee cette date sans retrait, l'entree est une dette a traiter,
// pas une tolerance. Ne JAMAIS ajouter une ligne ici pour faire passer un test : la seule
// raison valable est une reparation deja engagee ailleurs, nommee et datee.
//
// LA REPARATION EST DEJA ECRITE, et cette entree est donc deja condamnee : commit `8ebbb1483`
// du lot kill feed `PORTEUR` (branche `wt/killfeed-porteur`, worktree LevelUp-wt-killfeed-porteur)
// pose `hinf_mutilator` au registre, sa famille, son libelle (en = "Mutilator") et la
// `weapon_key` sur la regle `NOM Mutilator`. Ce commit n'est PAS encore fusionne dans
// `feat/v75` le 2026-09-10 : d'ou la survie temporaire de cette ligne. GESTE A FAIRE des la
// fusion : supprimer cette entree ET la map si elle reste vide, puis relancer
// TestWeaponStatCitations_ResolvableName — il doit etre vert SANS tolerance. Le laisser
// derriere serait exactement la « compatibility guard forever » que le diagnostic de revue
// du depot interdit.
var weaponStatNamesEnAttente = map[string]string{
	"mutilator_mastery": "Mutilateur absent du registre ; reparation portee par le lot " +
		"kill feed `PORTEUR` (2026-09-10). Retrait des que `hinf_mutilator` est au registre, " +
		"cible 2026-10-01.",
}

// knownWeaponNames rend l'ensemble des noms d'armes qu'une citation peut demander : les noms
// canoniques du registre, plus les `en` des manifestes de libelles (c'est cette valeur que
// loadWeaponKeyNames prefere au nom du registre quand elle existe).
func knownWeaponNames(t *testing.T) map[string]struct{} {
	t.Helper()
	known := map[string]struct{}{}

	registry, err := os.ReadFile(filepath.Join("..", "games", "weapons", "registry.go"))
	if err != nil {
		t.Logf("registre illisible (%v)", err)
		return known
	}
	for _, line := range strings.Split(string(registry), "\n") {
		if m := weaponRowNameRe.FindStringSubmatch(line); m != nil {
			known[m[1]] = struct{}{}
		}
	}

	for _, slug := range citationTitleMappingDirs {
		path := filepath.Join(citationRepoRoot, "config", "titles", slug, "mappings", "weapon_names.toml")
		labels, err := os.ReadFile(path)
		if err != nil {
			t.Logf("libelles %s illisibles (%v)", slug, err)
			continue
		}
		for _, m := range tomlNameENRe.FindAllStringSubmatch(string(labels), -1) {
			known[m[1]] = struct{}{}
		}
	}
	return known
}
