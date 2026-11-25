package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-files-reader/himodule"
)

// moduleFab : un module fabriqué (aucun fichier du jeu). Les tags se retrouvent par (groupe, identifiant).
type moduleFab struct {
	fichiers []himodule.File
	tags     map[string][]byte
	blobs    map[string][]byte
}

func newModuleFab() *moduleFab {
	return &moduleFab{tags: map[string][]byte{}, blobs: map[string][]byte{}}
}

func cleFab(groupe string, id uint32) string { return groupe + ":" + hex8(id) }

func (m *moduleFab) ajouter(groupe string, id uint32, tag, blob []byte) {
	m.fichiers = append(m.fichiers, himodule.File{Index: len(m.fichiers), Group: groupe, GlobalID: id})
	m.tags[cleFab(groupe, id)] = tag
	m.blobs[cleFab(groupe, id)] = blob
}

func (m *moduleFab) Files(group string) []himodule.File {
	var out []himodule.File
	for _, f := range m.fichiers {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

func (m *moduleFab) Extract(f himodule.File) ([]byte, error) {
	return m.tags[cleFab(f.Group, f.GlobalID)], nil
}

func (m *moduleFab) ResourceBlob(f himodule.File) ([]byte, error) {
	return m.blobs[cleFab(f.Group, f.GlobalID)], nil
}

func (m *moduleFab) Close() error { return nil }

const (
	cleUn, cleDeux = 0xAAAA0001, 0xAAAA0002
	idSon          = 0x10
	idBanque       = 0xB0
	idListe        = 0x77
)

// blobListeFab : une liste de chaînes à quatre langues (0 anglais, 3 français), comme celle du jeu.
func blobListeFab() []byte {
	vide := ucshFab(nil)
	en := ucshFab(map[uint32]string{cleUn: "Enemy there.", cleDeux: "Shields down."})
	fr := ucshFab(map[uint32]string{cleUn: "Ennemi\taperçu !\n", cleDeux: "Boucliers à zéro !"})
	var b []byte
	for _, p := range [][]byte{en, vide, vide, fr} {
		b = append(b, p...)
	}
	return b
}

func moduleDesTags() *moduleFab {
	m := newModuleFab()
	m.ajouter(groupeListe, idListe, nil, blobListeFab())
	// deux variantes : (évènement 0xE1, clé 1) et (évènement 0xE2, clé 2)
	m.ajouter(groupeSon, idSon, sndFab(idBanque, idListe, 0xE1, 0, cleUn, 0xE2, 0, cleDeux), nil)
	m.ajouter(groupeSon, 0x11, sndFab(idBanque, idListe, 0xE1, 0, cleUn), nil) // un son non demandé
	return m
}

func TestExtraireTextesLitLesVariantesDesSonsDemandes(t *testing.T) {
	langues := languesDeTest(t)
	got, err := extraireTextes(moduleDesTags(), map[uint32]bool{idSon: true}, langues, journalDeTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Son != "00000010" || got[0].Banque != hex8(idBanque) || got[0].Liste != hex8(idListe) {
		t.Fatalf("sons = %+v : un seul son demandé, avec sa banque et sa liste", got)
	}
	v := got[0].Variantes
	if len(v) != 2 || v[0].Evenement != "000000e1" || v[0].Cle != hex8(cleUn) || v[1].Evenement != "000000e2" {
		t.Fatalf("variantes = %+v", v)
	}
	if v[0].Textes["fr"] != "Ennemi aperçu !" {
		t.Errorf("texte français = %q : la tabulation et le retour à la ligne se ramènent à une espace, les blancs de bord partent", v[0].Textes["fr"])
	}
	if v[0].Textes["en"] != "Enemy there." || v[1].Textes["fr"] != "Boucliers à zéro !" {
		t.Errorf("textes = %+v", v)
	}
}

func TestNettoyerTexteGardeLEspaceInsecable(t *testing.T) {
	if got := nettoyerTexte(" Ennemi\u00a0!\n"); got != "Ennemi\u00a0!" {
		t.Errorf("texte = %q : l'espace insécable avant la ponctuation est de la typographie, pas du blanc à retirer", got)
	}
}

// banqueDuModule : la banque d'un son, en français et en anglais ; l'évènement 0xE1 joue le média 6000.
func banqueDuModule(media uint32) []byte {
	son := append(make([]byte, 5), u32le(media)...)
	hirc := hircFab(
		objetFab(hircEvenement, 0xE1, append([]byte{1}, u32le(0xA1)...)),
		objetFab(hircAction, 0xA1, append(make([]byte, 2), u32le(0x51)...)),
		objetFab(hircSon, 0x51, son),
	)
	d := banqueFab(fnv1("French(France)"), hirc, nil)
	return append(d, banqueFab(fnv1("English(US)"), hirc, nil)...)
}

// dossierDeVoix pose un paquet de voix français qui contient les médias a et b.
func dossierDeVoix(t *testing.T, a, b []byte) string {
	t.Helper()
	racine := t.TempDir()
	fr := filepath.Join(racine, "French(France)")
	if err := os.MkdirAll(fr, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(racine, "English(US)"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fr, "voix.pck"), pckFab(a, b), 0o644); err != nil {
		t.Fatal(err)
	}
	return racine
}

func TestExtraireBanquesRetrouveLesMediasParLangue(t *testing.T) {
	langues := languesDeTest(t)
	m := newModuleFab()
	m.ajouter(groupeBanque, idBanque, banqueDuModule(0xAAAA), nil)
	sons := []SonTextes{{Son: "00000010", Banque: hex8(idBanque), Variantes: []VarianteTexte{{Evenement: "000000e1", Cle: "k"}}}}
	racine := dossierDeVoix(t, []byte("voix"), []byte("autre"))
	got, err := extraireBanques(m, sons, langues, racine, journalDeTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Absente || len(got[0].Evenements) != 1 {
		t.Fatalf("banques = %+v", got)
	}
	l := got[0].Evenements[0].Langues
	if len(l["fr"]) != 1 || l["fr"][0].ID != "0000aaaa" || l["fr"][0].Paquet != "voix.pck" {
		t.Errorf("français = %+v : le média doit se retrouver dans le paquet de voix", l["fr"])
	}
	if len(l["en"]) != 1 || l["en"][0].ID != "0000aaaa" || l["en"][0].Paquet != "" || l["en"][0].Embarque {
		t.Errorf("anglais = %+v : média nommé par la banque, sans paquet installé ni média embarqué", l["en"])
	}
}

func TestExtraireBanquesSansBanqueDuSonLaSignale(t *testing.T) {
	sons := []SonTextes{{Son: "00000010", Banque: hex8(0xDEAD)}}
	got, err := extraireBanques(newModuleFab(), sons, languesDeTest(t), dossierDeVoix(t, nil, nil), journalDeTest(t))
	if err != nil || len(got) != 1 || !got[0].Absente {
		t.Fatalf("banque absente : %+v, %v", got, err)
	}
}

func TestExtrairePaquetsEcritLesMediasEtRendLesAbsents(t *testing.T) {
	langues := languesDeTest(t)
	racine := dossierDeVoix(t, []byte("la voix"), []byte("l'autre"))
	choix := []Choix{
		{Son: "s1", Voix: map[string]MediaVoix{"fr": {ID: "0000aaaa", Paquet: "voix.pck"}, "en": {ID: "0000aaaa"}}},
		{Son: "s2", Banque: hex8(idBanque), Voix: map[string]MediaVoix{"fr": {ID: "0000bbbb", Embarque: true}}},
	}
	dest := t.TempDir()
	emb, manquants, err := extrairePaquets(choix, langues, racine, dest, journalDeTest(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dest, "fr", "0000aaaa.wem"))
	if err != nil || string(b) != "la voix" {
		t.Errorf("média extrait = %q, %v", b, err)
	}
	if len(emb) != 1 || emb[0].Son != "s2" || manquants != 1 {
		t.Errorf("embarqués = %+v, manquants = %d : un média embarqué se lit dans la banque, un média sans fichier se compte", emb, manquants)
	}
}

func TestExtraireEmbarquesLitLeMediaDeLaBanque(t *testing.T) {
	langues := languesDeTest(t)
	hirc := hircFab(objetFab(hircSon, 1, append(make([]byte, 5), u32le(2)...)))
	banque := banqueFab(fnv1("French(France)"), hirc, map[uint32][]byte{2: []byte("embarque")})
	m := newModuleFab()
	m.ajouter(groupeBanque, idBanque, banque, nil)
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "fr"), 0o755); err != nil {
		t.Fatal(err)
	}
	fr := langues[0]
	if fr.Code != "fr" {
		t.Fatalf("première langue = %q, fr attendue", fr.Code)
	}
	n, err := extraireEmbarques(m, []embarque{{Langue: fr, Son: "s", Banque: hex8(idBanque), Media: MediaVoix{ID: hex8(2), Embarque: true}}}, dest, journalDeTest(t))
	if err != nil || n != 1 {
		t.Fatalf("extraits = %d, %v", n, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "fr", hex8(2)+".wem")); string(b) != "embarque" {
		t.Errorf("média = %q", b)
	}
}
