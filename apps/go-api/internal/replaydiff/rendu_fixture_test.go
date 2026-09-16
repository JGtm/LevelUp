package replaydiff

// rendu_fixture_test.go — LES DEUX RENDUS REJOUES SUR UN RAPPORT DE PAIRE REEL
// (lot 2.8.5, 2026-09-17).
//
// # POURQUOI UNE FIXTURE REELLE, ET PAS UN RAPPORT FABRIQUE
//
// Tous les autres tests de ce paquet construisent leur rapport a la main : trois mesures, deux
// axes, un sens par cas. C'est ce qu'il faut pour prouver une REGLE (un compteur d'echec
// s'inverse, un groupe conserve est une reattribution), et c'est insuffisant pour prouver un
// RENDU. Un rapport fabrique a toujours la forme que son auteur imaginait : des metriques
// courtes, un seul axe en ecart, jamais de cle a rallonge par joueur, jamais les trois
// categories a la fois.
//
// `testdata/paire_bcb6d393_54_60.json` est un rapport MESURE : la sortie de
//
//	replay-diff -ancien <artefact base> -nouveau <artefact tete> -json
//
// sur les deux artefacts CONSERVES (`--keep-work`) du corpus gate de la cloture M1
// (2026-09-16 / 17, temoin `bcb6d393`, famille ctf_mono_manche, schema 54 -> 60). Il porte les
// TROIS categories a la fois, ce qu'aucun rapport fabrique du depot ne faisait :
//
//	60 gains       30 « gain » + 30 « apparu »
//	13 pertes       9 « perte » +  4 « disparu »
//	 2 changements  la reattribution d'une vie (`tracks/par-xuid/2535429985869093` 6 -> 5 et sa
//	                jumelle `tracks/vies-par-xuid/...`), temoin des 12 index pour 8 occupants
//	                (lots 1.9.13 / 1.9.14)
//
// Ces comptes recoupent EXACTEMENT les colonnes que le corpus gate avait ecrites pour ce temoin
// (60 / 13 / 2). Les chemins de fichier ont ete remis en chemins RELATIFS a la racine de
// travail : une fixture ne porte pas le disque de celui qui l'a produite.
//
// Aucune cuisson n'est refaite ici : cette fixture EST la mesure.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixturePaireReelle : le chemin de la fixture, nomme une fois.
const fixturePaireReelle = "paire_bcb6d393_54_60.json"

// chargerPaireReelle lit la fixture. Une fixture illisible est une PANNE, pas un test a sauter.
func chargerPaireReelle(t *testing.T) Rapport {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", fixturePaireReelle))
	if err != nil {
		t.Fatalf("fixture illisible : %v", err)
	}
	var r Rapport
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("fixture invalide : %v", err)
	}
	return r
}

// TestFixturePaireReellePorteLesTroisCategories — LE PLANCHER DE LA FIXTURE. Si elle derive (un
// rapport re-genere sur un autre temoin, un fichier tronque), les deux tests de rendu
// ci-dessous verifieraient un cas qui n'est plus celui qu'on croit. Les comptes sont ECRITS EN
// CLAIR, jamais relus depuis la fixture qu'ils verifient.
func TestFixturePaireReellePorteLesTroisCategories(t *testing.T) {
	r := chargerPaireReelle(t)
	if r.SchemaAncien != 54 || r.SchemaNouveau != 60 {
		t.Fatalf("schemas = %d -> %d, 54 -> 60 attendus", r.SchemaAncien, r.SchemaNouveau)
	}
	parSens := map[string]int{}
	for _, d := range r.Differences {
		parSens[d.Sens]++
	}
	attendu := map[string]int{SensGain: 30, SensApparu: 30, SensPerte: 9, SensDisparu: 4, SensChangement: 2}
	for sens, n := range attendu {
		if parSens[sens] != n {
			t.Errorf("%d ecart(s) de sens %q, %d attendu(s) — la fixture a derive", parSens[sens], sens, n)
		}
	}
	var gains, pertes, changements int
	for _, b := range r.Bilans {
		gains, pertes, changements = gains+b.Gains, pertes+b.Pertes, changements+b.Changements
	}
	if gains != 60 || pertes != 13 || changements != 2 {
		t.Errorf("bilans = %d gains / %d pertes / %d changements, 60 / 13 / 2 attendus "+
			"(les colonnes ecrites par le corpus gate pour ce temoin)", gains, pertes, changements)
	}
}

// TestAfficherTableauSurUnePaireReelle — LE RENDU LISIBLE. Ce que le tableau doit contenir n'est
// pas « quelque chose » : l'en-tete de match avec les deux schemas, le bilan de CHAQUE axe en
// ecart, le groupement par axe, et le detail des trois categories — y compris les cles a
// rallonge par joueur, que seul un rapport reel porte.
func TestAfficherTableauSurUnePaireReelle(t *testing.T) {
	var b strings.Builder
	AfficherTableau(&b, chargerPaireReelle(t), false)
	out := b.String()
	t.Logf("tableau rendu (extrait de 24 lignes) :\n%s", extraitLignes(out, 24))

	for _, attendu := range []string{
		"match bcb6d393 : schema 54 -> 60",
		"75 ecarts sur 761 mesures",
		// Les bilans par axe, pertes en tete (afficherBilans trie dessus).
		"couverture         pertes=13   gains=36   changements=0",
		"pistes             pertes=0    gains=19   changements=2",
		// Le groupement par axe.
		"[couverture]", "[pistes]", "[roster]",
		// Un ecart de CHAQUE categorie, nomme.
		"disparu", "perte", "gain", "apparu", "changement",
		// La cle a rallonge par joueur, tronquee a 58 colonnes par `tronquer`.
		"tracks/par-xuid/2535429985869093",
		// Une valeur absente d'un cote s'affiche par le tiret cadratin, jamais par du vide.
		"—",
	} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le tableau doit contenir %q :\n%s", attendu, extraitLignes(out, 40))
		}
	}
	// `tout=false` : le compte des identiques ne s'affiche PAS en pied.
	if strings.Contains(out, "mesures identiques non listees") {
		t.Errorf("tout=false ne doit pas imprimer le pied des identiques :\n%s", extraitLignes(out, 40))
	}
}

// TestAfficherTableauToutAjouteLePiedDesIdentiques — `-tout` : le meme rapport, plus le compte
// des mesures identiques. C'est la SEULE difference entre les deux regimes.
func TestAfficherTableauToutAjouteLePiedDesIdentiques(t *testing.T) {
	r := chargerPaireReelle(t)
	var court, complet strings.Builder
	AfficherTableau(&court, r, false)
	AfficherTableau(&complet, r, true)
	if !strings.Contains(complet.String(), "(686 mesures identiques non listees)") {
		t.Errorf("le pied doit nommer les 686 mesures identiques :\n%s", extraitLignes(complet.String(), 8))
	}
	if !strings.HasPrefix(complet.String(), court.String()) {
		t.Error("`-tout` doit AJOUTER au rendu court, pas le reecrire")
	}
}

// TestEcrireJSONSurUnePaireReelleEstRelisible — LE RENDU MACHINE. Un rapport ecrit puis relu
// doit rendre les MEMES comptes : c'est ce qui autorise un balayage a agreger des centaines de
// paires sans jamais rouvrir un artefact.
func TestEcrireJSONSurUnePaireReelleEstRelisible(t *testing.T) {
	r := chargerPaireReelle(t)
	path := filepath.Join(t.TempDir(), "paire.json")
	if err := EcrireJSON(path, r); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin construit par le test
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	var relu Rapport
	if err := json.Unmarshal(raw, &relu); err != nil {
		t.Fatalf("le JSON ecrit n'est pas relisible : %v", err)
	}
	if len(relu.Differences) != len(r.Differences) {
		t.Errorf("%d ecart(s) relus, %d ecrits", len(relu.Differences), len(r.Differences))
	}
	if relu.Identiques != r.Identiques || relu.MatchID != r.MatchID {
		t.Errorf("identiques/matchId perdus a l'ecriture : %d/%q contre %d/%q",
			relu.Identiques, relu.MatchID, r.Identiques, r.MatchID)
	}
	for axe, b := range r.Bilans {
		if relu.Bilans[axe] != b {
			t.Errorf("bilan de l'axe %q perdu a l'ecriture : %+v contre %+v", axe, relu.Bilans[axe], b)
		}
	}
	// LES TROIS CATEGORIES SURVIVENT A L'ALLER-RETOUR — c'est la garantie que le balayage
	// agrege les changements et pas seulement les gains et les pertes.
	if !strings.Contains(string(raw), `"sens": "changement"`) {
		t.Error("le JSON ecrit ne porte aucun ecart de sens `changement`")
	}
}

// extraitLignes rend les `n` premieres lignes d'une sortie, pour que l'echec d'un test affiche
// de quoi comprendre sans noyer le journal sous 75 ecarts.
func extraitLignes(s string, n int) string {
	lignes := strings.Split(s, "\n")
	if len(lignes) <= n {
		return s
	}
	return strings.Join(lignes[:n], "\n") + "\n  ... (" + itoa(len(lignes)-n) + " ligne(s) de plus)"
}

// itoa evite d'importer strconv pour un seul appel dans un message de test.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var chiffres []byte
	for n > 0 {
		chiffres = append([]byte{byte('0' + n%10)}, chiffres...)
		n /= 10
	}
	return string(chiffres)
}
