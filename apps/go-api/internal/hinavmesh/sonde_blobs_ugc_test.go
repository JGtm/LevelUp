package hinavmesh

// SONDE (2026-08-27) — LES DEUX AUTRES BLOBS DE L'ASSET UGC PORTENT-ILS DES LIEUX NOMMES ?
//
// Question de l'utilisateur : le jeu affiche des callouts sur TOUTES les cartes, Forge
// comprises, alors que notre catalogue n'en connait que pour les 22 cartes natives. Un
// asset UGC de carte Forge publie NEUF fichiers ; nous n'en lisons que deux (`map.mvar` et
// `navmesh.blob`). Restent `audioocclusion.blob` et `lightprobes.blob`.
//
// L'occlusion audio, en particulier, est souvent un maillage de PIECES — et une piece
// nommee EST un callout. Cette sonde ouvre les deux blobs et dit, par la mesure, s'ils
// portent la moindre chaine de caractere et la moindre zone nommee.
//
// LES BLOBS NE SONT PAS VERSIONNES (`lightprobes.blob` d'Isolation pese 9,1 Mo). La sonde
// les lit dans le dossier designe par LEVELUP_UGC_BLOBS et se declare ABSENTE sinon —
// jamais verte par defaut. Pour les rapatrier (le stockage blob est anonyme, seule la
// resolution de l'asset demande un jeton) :
//
//	P=https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/map/<assetId>/<versionId>
//	curl -o $DIR/audioocclusion.blob $P/audioocclusion.blob
//	curl -o $DIR/lightprobes.blob    $P/lightprobes.blob
//
// Le couple (assetId, versionId) est le champ `Files.Prefix` de l'asset — voir
// cmd/mapobj-build/fetch.go.
//
// La sonde ne conclut pas : elle donne les nombres qui tranchent.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// dossierBlobsUGC est la variable d'environnement qui designe le depot local de blobs.
const dossierBlobsUGC = "LEVELUP_UGC_BLOBS"

// longueurChaineMin : en dessous de 5 caracteres, un balayage de chaines sur du binaire
// rend surtout du bruit. Les noms de classes Havok et les noms de lieux depassent tous
// ce seuil.
const longueurChaineMin = 5

// chainesImprimables rend les suites de caracteres ASCII imprimables d'au moins
// longueurChaineMin, avec leur nombre d'occurrences.
func chainesImprimables(buf []byte) map[string]int {
	out := map[string]int{}
	debut := -1
	for i := 0; i <= len(buf); i++ {
		imprimable := i < len(buf) && buf[i] >= 0x20 && buf[i] < 0x7F
		switch {
		case imprimable && debut < 0:
			debut = i
		case !imprimable && debut >= 0:
			if i-debut >= longueurChaineMin {
				out[string(buf[debut:i])]++
			}
			debut = -1
		}
	}
	return out
}

// regionsOuCharge decoupe la charge comme un `navmesh.blob` quand le preambule s'y prete,
// et rend la charge entiere comme region unique sinon. La distinction EST une mesure : un
// blob qui ne porte pas le preambule des 5 regions n'a pas la meme structure.
func regionsOuCharge(charge []byte) ([][]byte, bool) {
	if d, err := regions(charge); err == nil {
		return d, true
	}
	return [][]byte{charge}, false
}

// TestSondeBlobsUGCChaines — LE TABLEAU QUI TRANCHE : que portent les trois blobs ?
func TestSondeBlobsUGCChaines(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv(dossierBlobsUGC))
	if dir == "" {
		t.Skipf("depot de blobs absent : definir %s (voir l'en-tete du fichier)", dossierBlobsUGC)
	}
	for _, nom := range []string{"audioocclusion.blob", "lightprobes.blob", "navmesh.blob"} {
		chemin := filepath.Join(dir, nom)
		blob, err := os.ReadFile(chemin)
		if err != nil {
			t.Logf("%-22s ABSENT (%v)", nom, err)
			continue
		}
		charge, err := decompresse(blob)
		if err != nil {
			t.Errorf("%-22s deballage KO : %v", nom, err)
			continue
		}
		decoupe, preambule := regionsOuCharge(charge)
		t.Logf("=== %s : %d o comprimes -> %d o inflates, preambule 5 regions : %v, %d region(s)",
			nom, len(blob), len(charge), preambule, len(decoupe))
		for i, region := range decoupe {
			decritRegion(t, i, region)
		}
	}
}

// decritRegion dit ce qu'une region porte : un fichier-tag Havok (et alors sa classe
// racine et ses types), ou du binaire brut (et alors ses chaines).
func decritRegion(t *testing.T, i int, region []byte) {
	t.Helper()
	entete := ""
	if len(region) >= 8 {
		entete = fmt.Sprintf("%q", region[:8])
	}
	if len(region) < 8 || string(region[4:8]) != sectionTAG0 {
		ch := chainesImprimables(region)
		t.Logf("  region %d : %8d o  BRUTE  entete=%s  %d chaine(s) distincte(s)", i+1, len(region), entete, len(ch))
		journaliseChaines(t, ch, 25)
		return
	}
	f, err := lireFichierTag(region)
	if err != nil {
		t.Logf("  region %d : %8d o  TAG0 illisible : %v", i+1, len(region), err)
		return
	}
	racine, err := f.racine()
	classe := "?"
	if err == nil {
		classe = f.nomType(racine.Type)
	}
	ch := chainesImprimables(region)
	t.Logf("  region %d : %8d o  TAG0 classe racine=%s  %d chaine(s) distincte(s)",
		i+1, len(region), classe, len(ch))
	journaliseChaines(t, ch, 25)
}

// journaliseChaines affiche les chaines les plus frequentes, ET separement celles qui ne
// ressemblent PAS a de la reflexion Havok : c'est la-dedans qu'un nom de lieu se cacherait.
func journaliseChaines(t *testing.T, ch map[string]int, max int) {
	t.Helper()
	if len(ch) == 0 {
		t.Logf("      aucune chaine de %d caracteres ou plus", longueurChaineMin)
		return
	}
	cles := make([]string, 0, len(ch))
	var horsHavok []string
	for s := range ch {
		cles = append(cles, s)
		if !ressembleAHavok(s) {
			horsHavok = append(horsHavok, s)
		}
	}
	sort.Slice(cles, func(a, b int) bool {
		if ch[cles[a]] != ch[cles[b]] {
			return ch[cles[a]] > ch[cles[b]]
		}
		return cles[a] < cles[b]
	})
	for i, s := range cles {
		if i >= max {
			t.Logf("      ... (%d chaines distinctes au total)", len(cles))
			break
		}
		t.Logf("      x%-5d %q", ch[s], s)
	}
	sort.Strings(horsHavok)
	t.Logf("      HORS REFLEXION HAVOK : %d chaine(s)", len(horsHavok))
	for i, s := range horsHavok {
		if i >= max {
			t.Logf("      ... (%d hors reflexion au total)", len(horsHavok))
			break
		}
		t.Logf("      >>> %q", s)
	}
}

// ressembleAHavok reconnait le vocabulaire de la reflexion d'un fichier-tag : noms de
// classes (`hk...`), de membres, et les jetons de section. Ce sont les chaines qu'on
// s'attend a trouver ; tout le reste merite d'etre regarde.
func ressembleAHavok(s string) bool {
	if strings.HasPrefix(s, "hk") || strings.HasPrefix(s, "Hk") {
		return true
	}
	switch s {
	case sectionTAG0, "SDKV", "20220100", "DATA", "TYPE", "TPTR", "TST1", "TNA1", "FST1",
		"TBDY", "THSH", "TPAD", "INDX", "ITEM", "PTCH", "TSHA", "TSEQ":
		return true
	}
	return false
}

// motPlausible distingue un MOT d'un artefact de flottants. Le balayage brut d'un champ
// de nombres rend des dizaines de milliers de fausses chaines (« H~BFI », « BOI}BDI ») :
// compter ces chaines-la ne repond a aucune question. Un nom de lieu, lui, est fait de
// lettres, il porte une voyelle, et il n'est pas noye de ponctuation.
func motPlausible(s string) bool {
	if len(s) < 6 {
		return false
	}
	lettres, voyelles, autres := 0, 0, 0
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			lettres++
			if strings.ContainsRune("aeiouyAEIOUY", r) {
				voyelles++
			}
		case r == ' ', r == '_', r == '-', r >= '0' && r <= '9':
		default:
			autres++
		}
	}
	return voyelles > 0 && autres == 0 && lettres*5 >= len(s)*4
}

// profilOctets caracterise une region binaire : part d'octets nuls et nombre de valeurs
// distinctes. Un champ dense de flottants et une grille creuse ne rendent pas les memes
// nombres, et c'est ce qui dit de quelle NATURE est la donnee.
func profilOctets(buf []byte) (partNulle float64, distincts int) {
	var histo [256]int
	for _, b := range buf {
		histo[b]++
	}
	for _, n := range histo {
		if n > 0 {
			distincts++
		}
	}
	if len(buf) > 0 {
		partNulle = float64(histo[0]) / float64(len(buf))
	}
	return partNulle, distincts
}

// TestSondeBlobsUGCMots — LA QUESTION POSEE, SANS LE BRUIT : un nom de lieu, c'est un MOT.
//
// Le comptage brut de chaines ne tranche rien sur un champ de flottants. Ce test ne garde
// que les suites qui ressemblent a du langage, et affiche le profil binaire de chaque
// blob : c'est lui qui dit si l'on a affaire a une grille de nombres ou a une structure
// nommee.
func TestSondeBlobsUGCMots(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv(dossierBlobsUGC))
	if dir == "" {
		t.Skipf("depot de blobs absent : definir %s (voir l'en-tete du fichier)", dossierBlobsUGC)
	}
	for _, nom := range []string{"audioocclusion.blob", "lightprobes.blob", "navmesh.blob"} {
		blob, err := os.ReadFile(filepath.Join(dir, nom))
		if err != nil {
			t.Logf("%-22s ABSENT (%v)", nom, err)
			continue
		}
		charge, err := decompresse(blob)
		if err != nil {
			t.Errorf("%-22s deballage KO : %v", nom, err)
			continue
		}
		partNulle, distincts := profilOctets(charge)
		var mots []string
		for s := range chainesImprimables(charge) {
			if motPlausible(s) {
				mots = append(mots, s)
			}
		}
		sort.Strings(mots)
		t.Logf("=== %-22s inflate %9d o  octets nuls %5.1f %%  valeurs distinctes %3d  MOTS %d",
			nom, len(charge), 100*partNulle, distincts, len(mots))
		t.Logf("    32 premiers octets : % x", charge[:min(32, len(charge))])
		for i, s := range mots {
			if i >= 60 {
				t.Logf("    ... (%d mots au total)", len(mots))
				break
			}
			t.Logf("    MOT %q", s)
		}
	}
}

// vocabulaireLieu : les jetons que porterait FORCEMENT une structure de lieux nommes.
// « named location » est le prefixe de conception que les 816 zones natives portent toutes
// (callouts.go, calloutNamePrefix) ; les autres sont les mots que le moteur emploie autour
// d'une zone. Chercher des SOUS-CHAINES echappe au bruit du balayage de chaines : une
// occurrence, meme noyee dans des flottants, se voit.
var vocabulaireLieu = []string{
	"named location", "location", "callout", "zone", "room", "portal", "cluster",
	"region", "sector", "area_", "poi_", "label",
}

// TestSondeBlobsUGCVocabulaire — LE TEST QUI TRANCHE SANS DEPENDRE DU BRUIT.
//
// Le balayage de chaines d'un champ de flottants rend des milliers de faux mots : compter
// ces mots-la ne repond a rien. Chercher un VOCABULAIRE, si. Si les 19,5 Mo d'occlusion
// audio et les 17 Mo de sondes de lumiere ne portent pas une seule occurrence de
// « location », de « room » ou de « callout », alors ces blobs ne portent pas de lieux
// nommes — et la question est close pour eux.
func TestSondeBlobsUGCVocabulaire(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv(dossierBlobsUGC))
	if dir == "" {
		t.Skipf("depot de blobs absent : definir %s (voir l'en-tete du fichier)", dossierBlobsUGC)
	}
	for _, nom := range []string{"audioocclusion.blob", "lightprobes.blob", "navmesh.blob"} {
		blob, err := os.ReadFile(filepath.Join(dir, nom))
		if err != nil {
			t.Logf("%-22s ABSENT (%v)", nom, err)
			continue
		}
		charge, err := decompresse(blob)
		if err != nil {
			t.Errorf("%-22s deballage KO : %v", nom, err)
			continue
		}
		bas := bytes.ToLower(charge)
		var trouves []string
		for _, mot := range vocabulaireLieu {
			if n := bytes.Count(bas, []byte(mot)); n > 0 {
				trouves = append(trouves, fmt.Sprintf("%s x%d", mot, n))
			}
		}
		if len(trouves) == 0 {
			t.Logf("=== %-22s %9d o inflates : AUCUN jeton de lieu (%d jetons cherches)",
				nom, len(charge), len(vocabulaireLieu))
			continue
		}
		t.Logf("=== %-22s %9d o inflates : %s", nom, len(charge), strings.Join(trouves, ", "))
		// Le contexte est ce qui distingue un nom de lieu d'un nom de MEMBRE de classe
		// Havok : on montre les 48 octets autour de la premiere occurrence de chaque jeton.
		for _, mot := range vocabulaireLieu {
			if i := bytes.Index(bas, []byte(mot)); i >= 0 {
				d, f := max(0, i-16), min(len(charge), i+len(mot)+32)
				t.Logf("    %-16s @%-9d %q", mot, i, charge[d:f])
			}
		}
	}
}
