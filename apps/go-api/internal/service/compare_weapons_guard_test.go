package service

// compare_weapons_guard_test.go — LE TRI « FRAGS DÉCROISSANTS, DÉPARTAGE SUR LE LIBELLÉ »
// NE S'ÉCRIT QU'UNE FOIS (D11 du plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md).
//
// # POURQUOI CE GARDE-RAIL
//
// Ce comparateur vivait inline dans `buildTopWeaponKills`, et `buildWeaponAccuracy` en porte
// déjà une variante sur sa propre métrique. Le profil d'armes du Face-à-face en aurait été la
// TROISIÈME écriture. La règle n°6 du dépôt est explicite : à la troisième copie on
// centralise ET on pose un garde-rail — une factorisation sans garde-rail re-diverge (leçon
// chiffrée du dépôt : un prédicat passé de 8 à 36 copies APRÈS centralisation).
//
// CE QUI EST EN JEU N'EST PAS COSMÉTIQUE. Le départage alphabétique est ce qui rend le
// classement DÉTERMINISTE : sans lui, deux armes à frags égaux permutent d'une réponse à
// l'autre, ce que le harnais de régression visuelle avait fini par attraper. Une copie qui
// oublierait le départage rendrait un classement instable sur UNE page seulement — le pire
// des cas, parce qu'il ne se voit qu'en comparant deux captures.
//
// # CE QU'IL CAPTE, ET CE QU'IL NE CAPTE PAS
//
// Le motif vise la FORME COMPLÈTE du comparateur (kills décroissants PUIS libellé croissant),
// pas la seule comparaison de frags : `internal/service/` porte plusieurs tris par frags
// décroissants légitimes et différents — la répartition des frags trie des rôles. Les signaler
// ferait désactiver ce garde-rail dans le mois.
//
// # ET LA VARIANTE DÉPARTAGÉE SUR L'IDENTIFIANT D'ARME
//
// Les séries temporelles ont longtemps départagé sur `WeaponID` : MÊME classement, TROISIÈME
// doctrine — à frags égaux, deux armes s'affichaient dans un ordre différent entre Séries
// temporelles et Synthèse / Face-à-face, et le motif ci-dessus ne la voyait pas (littéral
// différent). `buildTopWeapons` appelle désormais `topWeaponKillRows` comme les autres, et
// `topWeaponSortByIDInline` interdit le retour de cette variante — dans N'IMPORTE QUEL fichier
// non-test du paquet, propriétaire compris : elle n'a plus de site légitime.
//
// Elle ne capte QUE la forme sur `.Kills` : le tableau d'armes de l'escouade
// (`squad_service_v2_weapons_medals.go`) trie un `.Total` et départage sur `WeaponID` — autre
// classement (table exhaustive, grenades incluses), légitime, et couvert par le contrôle
// positif ci-dessous.
//
// Il ne capte PAS un comparateur écrit avec des variables intermédiaires
// (`a, b := rows[i], rows[j]`). Ce trou est ACCEPTÉ : le contrôle positif ci-dessous garantit
// au moins que le détecteur détecte encore, et la forme visée est celle que produirait
// mécaniquement un copier-coller de l'existant — qui est le risque réel.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// topWeaponSortOwner : le SEUL fichier autorisé à écrire ce comparateur, vu depuis la racine
// de la marche (le paquet `service`).
const topWeaponSortOwner = "synthesis_service_builders.go"

// topWeaponSortInline matche le comparateur « frags décroissants, départage sur le libellé »
// quel que soit le nom de la tranche triée. `(?s)` pour franchir les retours à la ligne, que
// gofmt impose entre les trois instructions.
var topWeaponSortInline = regexp.MustCompile(
	`(?s)\[i\]\.Kills\s*!=\s*\w+\[j\]\.Kills.{0,120}?\[i\]\.Kills\s*>\s*\w+\[j\]\.Kills` +
		`.{0,120}?\[i\]\.Label\s*<\s*\w+\[j\]\.Label`)

// topWeaponSortByIDInline matche la MÊME doctrine départagée sur l'identifiant d'arme —
// la troisième écriture retirée le 2026-09-19 (`buildTopWeapons`). Aucun propriétaire :
// ce départage n'a plus de site légitime dans le paquet.
var topWeaponSortByIDInline = regexp.MustCompile(
	`(?s)\[i\]\.Kills\s*!=\s*\w+\[j\]\.Kills.{0,120}?\[i\]\.Kills\s*>\s*\w+\[j\]\.Kills` +
		`.{0,120}?\[i\]\.WeaponID\s*<\s*\w+\[j\]\.WeaponID`)

// TestTriTopArmesEcritUneSeuleFois : hors de son propriétaire, aucun fichier Go non-test du
// paquet `service` ne réécrit le comparateur.
func TestTriTopArmesEcritUneSeuleFois(t *testing.T) {
	verifierProprietaireTriTopArmes(t)
	fautifs := scanTriTopArmes(t, ".")
	if len(fautifs) > 0 {
		t.Fatalf("comparateur du top armes RÉÉCRIT hors de %s : %v.\n"+
			"Appeler topWeaponKillRows (D11) — une copie perd le départage alphabétique et "+
			"rend le classement non déterministe", topWeaponSortOwner, fautifs)
	}
}

// TestTriTopArmesJamaisDepartageSurIdentifiant : la variante « frags décroissants, départage
// sur l'identifiant d'arme » ne revient nulle part — propriétaire compris.
func TestTriTopArmesJamaisDepartageSurIdentifiant(t *testing.T) {
	fautifs := scanMotifTri(t, ".", topWeaponSortByIDInline, "")
	if len(fautifs) > 0 {
		t.Fatalf("top armes départagé sur l'identifiant d'arme dans %v.\n"+
			"Appeler topWeaponKillRows (D11) : le départage se fait sur le LIBELLÉ, sinon "+
			"deux armes à frags égaux s'ordonnent différemment d'une page à l'autre", fautifs)
	}
}

// TestTriTopArmesDetecteUneCopie est le CONTRÔLE POSITIF. Sans lui, le test ci-dessus reste
// vert même si le motif ne matche plus rien — un détecteur mort affirme « zéro fautif » avec
// la même sérénité qu'un détecteur vivant.
//
// Le répertoire est celui de `t.TempDir()`, supprimé par le framework : rien n'est écrit dans
// l'arbre du dépôt, donc rien ne peut y rester si le test échoue en cours de route.
func TestTriTopArmesDetecteUneCopie(t *testing.T) {
	racine := t.TempDir()
	const fautif = "copie_tri.go"
	const fautifID = "copie_tri_identifiant.go"
	const licite = "tri_legitime.go"
	contenus := map[string]string{
		fautif: "package faux\n\nimport \"sort\"\n\n" +
			"func f(rows []struct {\n\tKills int\n\tLabel string\n}) {\n" +
			"\tsort.SliceStable(rows, func(i, j int) bool {\n" +
			"\t\tif rows[i].Kills != rows[j].Kills {\n" +
			"\t\t\treturn rows[i].Kills > rows[j].Kills\n\t\t}\n" +
			"\t\treturn rows[i].Label < rows[j].Label\n\t})\n}\n",
		// La TROISIÈME doctrine retirée le 2026-09-19 : même classement, départage sur
		// l'identifiant d'arme. Elle n'a plus de site légitime → elle doit être signalée.
		fautifID: "package faux\n\nimport \"sort\"\n\n" +
			"func g(rows []struct {\n\tKills int\n\tWeaponID int64\n}) {\n" +
			"\tsort.SliceStable(rows, func(i, j int) bool {\n" +
			"\t\tif rows[i].Kills != rows[j].Kills {\n" +
			"\t\t\treturn rows[i].Kills > rows[j].Kills\n\t\t}\n" +
			"\t\treturn rows[i].WeaponID < rows[j].WeaponID\n\t})\n}\n",
		// Le tableau d'armes de l'escouade : classement d'un TOTAL (table exhaustive,
		// grenades incluses), départagé sur l'identifiant. Autre classement, légitime —
		// le signaler ferait désactiver ce garde-rail.
		licite: "package faux\n\nimport \"sort\"\n\n" +
			"func h(rows []struct {\n\tTotal int\n\tWeaponID int64\n}) {\n" +
			"\tsort.SliceStable(rows, func(i, j int) bool {\n" +
			"\t\tif rows[i].Total != rows[j].Total {\n" +
			"\t\t\treturn rows[i].Total > rows[j].Total\n\t\t}\n" +
			"\t\treturn rows[i].WeaponID < rows[j].WeaponID\n\t})\n}\n",
	}
	for nom, src := range contenus {
		if err := os.WriteFile(filepath.Join(racine, nom), []byte(src), 0o600); err != nil {
			t.Fatalf("écriture de %s : %v", nom, err)
		}
	}

	vus := map[string]bool{}
	for _, f := range scanTriTopArmes(t, racine) {
		vus[f] = true
	}
	if !vus[fautif] {
		t.Errorf("copie NON DÉTECTÉE : %s — le détecteur ne détecte plus rien", fautif)
	}
	if vus[fautifID] {
		t.Errorf("%s départage sur l'identifiant : c'est l'AUTRE motif qui doit le voir, "+
			"pas celui du libellé", fautifID)
	}
	if vus[licite] {
		t.Errorf("%s trie un total : classement distinct, ne doit pas être signalé", licite)
	}

	vusID := map[string]bool{}
	for _, f := range scanMotifTri(t, racine, topWeaponSortByIDInline, "") {
		vusID[f] = true
	}
	if !vusID[fautifID] {
		t.Errorf("copie NON DÉTECTÉE : %s — le détecteur du départage sur l'identifiant "+
			"ne détecte plus rien", fautifID)
	}
	if vusID[licite] {
		t.Errorf("%s trie un total, pas des frags : ne doit pas être signalé", licite)
	}
}

// verifierProprietaireTriTopArmes échoue tout de suite si le propriétaire ne porte plus le
// comparateur : un garde-rail qui ne garde plus rien est pire qu'aucun garde-rail.
func verifierProprietaireTriTopArmes(t *testing.T) {
	t.Helper()
	src, err := os.ReadFile(filepath.Clean(topWeaponSortOwner))
	if err != nil {
		t.Fatalf("lecture du propriétaire %s : %v", topWeaponSortOwner, err)
	}
	if !strings.Contains(string(src), "func topWeaponKillRows(") {
		t.Fatalf("topWeaponKillRows a disparu de %s : le garde-rail ne vérifie plus rien",
			topWeaponSortOwner)
	}
	if !topWeaponSortInline.Match(src) {
		t.Fatalf("le comparateur a changé de forme dans %s : le motif ne le reconnaît plus, "+
			"donc il ne reconnaîtrait pas non plus une copie", topWeaponSortOwner)
	}
}

// scanTriTopArmes marche la racine donnée et rend les fichiers qui réécrivent le comparateur
// départagé sur le LIBELLÉ, propriétaire exempté. Alias de scanMotifTri.
func scanTriTopArmes(t *testing.T, racine string) []string {
	t.Helper()
	return scanMotifTri(t, racine, topWeaponSortInline, topWeaponSortOwner)
}

// scanMotifTri marche la racine donnée et rend les fichiers Go non-test que `motif` matche,
// en chemin RELATIF, hors `exempt` (chemin relatif du seul propriétaire autorisé ; "" =
// aucun). La racine est un paramètre pour que le contrôle positif lui soumette de vraies
// copies sans les écrire dans le dépôt.
func scanMotifTri(t *testing.T, racine string, motif *regexp.Regexp, exempt string) []string {
	t.Helper()
	var fautifs []string
	err := filepath.WalkDir(racine, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, racine+string(filepath.Separator)))
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			(exempt != "" && rel == exempt) {
			return nil
		}
		raw, rerr := os.ReadFile(filepath.Clean(path))
		if rerr != nil {
			t.Fatalf("lecture %s : %v", path, rerr)
		}
		if motif.Match(raw) {
			fautifs = append(fautifs, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("marche de %s : %v", racine, err)
	}
	return fautifs
}
