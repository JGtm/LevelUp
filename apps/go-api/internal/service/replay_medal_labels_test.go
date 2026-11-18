package service

// replay_medal_labels_test.go — les REGLES de la table des médailles du document, posée à la requête.
// Chaque test correspond à une décision écrite dans l'en-tête du fichier de production.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// nomsDeMedailles est un pont nom anglais -> identité : il ne connaît que les noms de sa table.
type nomsDeMedailles struct {
	table map[string]int64
	err   error
	vus   [][]string
}

func (n *nomsDeMedailles) LookupMedalMetaByName(_ context.Context, noms []string) (map[string]domain.MedalNameMeta, error) {
	n.vus = append(n.vus, noms)
	if n.err != nil {
		return nil, n.err
	}
	out := map[string]domain.MedalNameMeta{}
	for _, nom := range noms {
		if id, ok := n.table[nom]; ok {
			out[nom] = domain.MedalNameMeta{MedalNameID: id}
		}
	}
	return out, nil
}

// definitionsDeMedailles rend, par langue, les lignes du référentiel.
type definitionsDeMedailles struct {
	parLangue map[string]map[int64]port.MedalDefinitionRow
	enPanne   map[string]bool
}

func (d definitionsDeMedailles) LookupByIDs(_ context.Context, ids []int64, locale string) (map[int64]port.MedalDefinitionRow, error) {
	if d.enPanne[locale] {
		return nil, errors.New("lecture en panne")
	}
	out := map[int64]port.MedalDefinitionRow{}
	for _, id := range ids {
		if row, ok := d.parLangue[locale][id]; ok {
			out[id] = row
		}
	}
	return out, nil
}

// artefactMedailles pose un artefact minimal dont `medals` porte trois médailles : deux fois la même,
// et une dont le nom n'est pas au référentiel ; plus une sans nom (indice absent de la table du
// décodeur).
func artefactMedailles(t *testing.T, root, titleSlug, matchID string) {
	t.Helper()
	doc := `{"schemaVersion":125,"matchId":"` + matchID + `","titleSlug":"` + titleSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],` +
		`"medals":[` +
		`{"t":0,"ms":1000,"xuid":"1","id":7,"name":"Killjoy"},` +
		`{"t":0,"ms":2000,"xuid":"2","id":7,"name":"Killjoy"},` +
		`{"t":0,"ms":3000,"xuid":"1","id":99,"name":"Medaille Inconnue"},` +
		`{"t":0,"ms":4000,"xuid":"1","id":5}]}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(titleSlug, matchID), doc)
}

func definitionsKilljoy() definitionsDeMedailles {
	return definitionsDeMedailles{parLangue: map[string]map[int64]port.MedalDefinitionRow{
		"fr": {101: {MedalID: 101, Label: "Rabat-joie", Description: "Tuer un adversaire en série", Difficulty: "Heroic"}},
		"en": {101: {MedalID: 101, Label: "Killjoy", Description: "End an enemy spree", Difficulty: "Heroic"}},
	}}
}

func serviceMedailles(root string, noms port.MedalNameLookup, defs port.MedalDefinitionsRepository) port.ReplayService {
	return NewReplayService(title.DefaultSlug, root, nil, labelsDe(root), WithReplayMedals(noms, defs))
}

// TestMedalLabels_LibelleResoluEnFrEtEn : une entrée par NOM (pas par médaille gagnée), portant le
// libellé et la description de CHAQUE langue, la rareté et l'image du titre.
func TestMedalLabels_LibelleResoluEnFrEtEn(t *testing.T) {
	root := t.TempDir()
	artefactMedailles(t, root, title.DefaultSlug, "a0000003")
	noms := &nomsDeMedailles{table: map[string]int64{"Killjoy": 101}}

	doc, err := serviceMedailles(root, noms, definitionsKilljoy()).GetReplay(context.Background(), "a0000003")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if len(doc.MedalLabels) != 1 {
		t.Fatalf("%d entrées servies, attendu 1 (un nom connu) : %v", len(doc.MedalLabels), doc.MedalLabels)
	}
	got := doc.MedalLabels["Killjoy"]
	if fr, en := got.Text.Pick(i18n.Locale("fr")), got.Text.Pick(i18n.Locale("en")); fr != "Rabat-joie" || en != "Killjoy" {
		t.Errorf("libellés fr=%q en=%q, attendu Rabat-joie / Killjoy", fr, en)
	}
	if got.Description == nil || got.Description.Text.Pick(i18n.Locale("fr")) != "Tuer un adversaire en série" {
		t.Errorf("description = %+v, attendu celle de la langue fr", got.Description)
	}
	if got.Difficulty != "Heroic" {
		t.Errorf("rareté = %q, attendu Heroic", got.Difficulty)
	}
	if got.Img != "/static/medals/"+title.DefaultSlug+"/101.png" {
		t.Errorf("image = %q : elle se compose sous le dossier d'assets du titre, comme sur les tuiles de match", got.Img)
	}
	if len(noms.vus) != 1 || len(noms.vus[0]) != 2 {
		t.Errorf("le pont a reçu %v : une seule lecture, les noms uniques et non vides", noms.vus)
	}
}

// TestMedalLabels_MedailleInconnueJournaliseeSansLibelleInvente : un nom que le référentiel ne connaît
// pas n'a AUCUNE entrée et son nom est journalisé ; la médaille connue est servie quand même.
func TestMedalLabels_MedailleInconnueJournaliseeSansLibelleInvente(t *testing.T) {
	logs := capturerLogs(t)
	root := t.TempDir()
	artefactMedailles(t, root, title.DefaultSlug, "a0000003")
	noms := &nomsDeMedailles{table: map[string]int64{"Killjoy": 101}}

	doc, err := serviceMedailles(root, noms, definitionsKilljoy()).GetReplay(context.Background(), "a0000003")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if _, ok := doc.MedalLabels["Medaille Inconnue"]; ok {
		t.Error("une médaille inconnue reçoit une entrée : le lecteur garde le nom du film, rien d'inventé")
	}
	if _, ok := doc.MedalLabels[""]; ok {
		t.Error("une médaille sans nom reçoit une entrée sous la clé vide")
	}
	if !strings.Contains(logs.String(), "Medaille Inconnue") || !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("la médaille inconnue n'est pas journalisée en avertissement : %q", logs.String())
	}
	if len(doc.Medals) != 4 {
		t.Errorf("%d médailles servies, attendu 4 : la table ne retire rien de `medals`", len(doc.Medals))
	}
}

// TestMedalLabels_AucuneTableSansMedailleNiSource : pas de médaille, pas de table ; pas de source, pas de
// table non plus (et la raison est journalisée).
func TestMedalLabels_AucuneTableSansMedailleNiSource(t *testing.T) {
	root := t.TempDir()
	artefactVehicules(t, root, title.DefaultSlug, "a0000002")
	noms := &nomsDeMedailles{table: map[string]int64{"Killjoy": 101}}
	doc, err := serviceMedailles(root, noms, definitionsKilljoy()).GetReplay(context.Background(), "a0000002")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if doc.MedalLabels != nil || len(noms.vus) != 0 {
		t.Errorf("document sans médaille : table %v, lectures du pont %v", doc.MedalLabels, noms.vus)
	}

	logs := capturerLogs(t)
	artefactMedailles(t, root, title.DefaultSlug, "a0000003")
	doc, err = NewReplayService(title.DefaultSlug, root, nil, labelsDe(root)).GetReplay(context.Background(), "a0000003")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if doc.MedalLabels != nil || !strings.Contains(logs.String(), "aucune source de médailles") {
		t.Errorf("sans source : table %v, journal %q", doc.MedalLabels, logs.String())
	}
}

// TestMedalLabels_PanneDuReferentielJournaliseeEtRejeuServi : une lecture en échec ne casse pas le rejeu.
func TestMedalLabels_PanneDuReferentielJournaliseeEtRejeuServi(t *testing.T) {
	logs := capturerLogs(t)
	root := t.TempDir()
	artefactMedailles(t, root, title.DefaultSlug, "a0000003")
	noms := &nomsDeMedailles{err: errors.New("metadata illisible")}

	doc, err := serviceMedailles(root, noms, definitionsKilljoy()).GetReplay(context.Background(), "a0000003")
	if err != nil {
		t.Fatalf("une panne du référentiel ne doit pas faire échouer le rejeu : %v", err)
	}
	if doc.MedalLabels != nil || len(doc.Medals) != 4 {
		t.Errorf("table %v, %d médailles : le document part sans table, entier par ailleurs", doc.MedalLabels, len(doc.Medals))
	}
	if !strings.Contains(logs.String(), "metadata illisible") {
		t.Errorf("la panne n'est pas journalisée : %q", logs.String())
	}
}

// TestMedalLabels_LangueEnPanneAbsenteDeLaTable : une langue illisible manque à la table (journalisée),
// les autres sont servies.
func TestMedalLabels_LangueEnPanneAbsenteDeLaTable(t *testing.T) {
	logs := capturerLogs(t)
	root := t.TempDir()
	artefactMedailles(t, root, title.DefaultSlug, "a0000003")
	noms := &nomsDeMedailles{table: map[string]int64{"Killjoy": 101}}
	defs := definitionsKilljoy()
	defs.enPanne = map[string]bool{"fr": true}

	doc, err := serviceMedailles(root, noms, defs).GetReplay(context.Background(), "a0000003")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	got := doc.MedalLabels["Killjoy"]
	if got.Text.Has(i18n.Locale("fr")) || !got.Text.Has(i18n.Locale("en")) {
		t.Errorf("textes = %v, attendu la seule langue en", got.Text)
	}
	if !strings.Contains(logs.String(), "textes de médailles illisibles") {
		t.Errorf("la langue manquante n'est pas journalisée : %q", logs.String())
	}
}
