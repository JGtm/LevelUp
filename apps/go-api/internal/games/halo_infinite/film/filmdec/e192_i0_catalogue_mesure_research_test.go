package filmdec

// e192_i0_catalogue_mesure_research_test.go — LOT 1.9.2, LA MESURE AVANT DE CODER.
//
// # LA QUESTION, POSÉE AVANT TOUTE LIGNE DE PRODUCTION
//
// Le découpage d'i0 (porte, index de région attendu, largeurs d'axe) est une DONNÉE DE PROFIL :
// le catalogue de carte l'écrit (`data/titles/halo_infinite/reference/map_quant_bounds.json`,
// `MapQuantEntry.Layout`). Le chemin de cuisson l'impose déjà (`replay/build_from_film.go`) ;
// les chemins de `sync/killcollector` ne l'imposaient pas : ils partaient de
// `DefaultScanFilmOptions()` avec `Layout` nil, ce qui laisse `DetectI0LayoutOf` DÉCIDER
// (`offline_biped_band.go/bipedI0Layout`).
//
// Avant de convertir, il faut CHIFFRER, film par film :
//
//	ce que l'auto-détection rend et ce que le catalogue impose (largeurs, porte, région) ;
//	sur quelles cartes les deux DIFFÈRENT ;
//	combien d'enregistrements de position CHANGENT quand le catalogue décide — sous les deux
//	  jeux de réglages employés par `killcollector` : celui des POSITIONS (filtres de production)
//	  et celui des TOUCHES (`BuildBipedTracks` : filtres désarmés, chunks énumérés).
//
// # POURQUOI CET INSTRUMENT ET PAS UNE LECTURE DU CODE
//
// La mesure du 2026-09-03 (27 faux enregistrements sur 267 400, `film_context.go`) porte sur UN
// film et sur le chemin de CUISSON. Rien ne disait ce que la conversion change sur les témoins du
// corpus gate ni sur les huit builds — ni, surtout, ce qu'elle ne change pas. Un tableau collé
// depuis cet instrument le dit.
//
// # CE QU'IL NE FAIT PAS
//
// Aucune écriture, aucune base, aucun artefact : il charge des films en lecture seule et compte.
// Les TIRS (`ScanFilmWeaponShots`) et les DÉGÂTS (`ScanFilmWeaponDamages`) ne sont pas mesurés
// ici parce qu'ils ne lisent PAS i0 — vérifié par grep (`I0Layout` / `DetectI0` absents de
// `weapon_hits.go`), consigné au §5 du plan.
//
// GARDE `CHUNK00_FILMS` (répertoires de film absolus séparés par `;`). Les cartes viennent d'une
// table versionnée ci-dessous, recopiée de `config/replay_corpus.toml` et de
// `replay/testdata/equivalence/CORPUS.txt` ; un film absent de la table est NOMMÉ et sauté, il
// n'est jamais deviné.
//
//	CHUNK00_FILMS='C:/.../film_chunks/bcb6d393;C:/.../film_chunks/60ae07c4' \
//	  CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/filmdec/ \
//	  -run '^TestE192CatalogueContreDetection$' -v -count=1 -timeout 3600s

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// e192CarteDuFilm : la carte de chaque témoin, telle que le registre du match la nomme.
//
// SOURCE VERSIONNÉE, PAS UNE DEVINETTE : les quatorze premiers sont les témoins du corpus gate
// (`config/replay_corpus.toml`, champ `carte`), les trois derniers complètent les huit builds de
// l'échantillon court (`replay/testdata/equivalence/CORPUS.txt`). Les noms sont les `map_name`
// BRUTS ; `NormalizeMapName` retire les suffixes de variante au `Lookup`.
var e192CarteDuFilm = map[string]string{
	// Corpus gate (14 témoins).
	"bcb6d393": "Cliffhanger",
	"fb1a1a72": "Banished Narrows",
	"d9781168": "Dredge",
	"c75f33b8": "Curfew",
	"bf15f7ab": "Perilous",
	"51ebbc0f": "Banished Narrows",
	"084a804d": "Fortitude Heavies",
	"0797ce72": "Live Fire",
	"111fa685": "Command",
	"e5adf7b2": "Fragmentation",
	"60ae07c4": "Live Fire - Ranked",
	"a349fea8": "Fragmentation Heavies",
	"bfecd02b": "Snowbound",
	"4f77afc1": "Flood Gulch",
	// Complément des huit builds (échantillon court d'équivalence).
	"50247b26": "Oasis",
	"a521164d": "Fragmentation Heavies",
	"11de8353": "Thunderhead",
}

// e192Ligne : une ligne de la mesure, un film.
type e192Ligne struct {
	Film, Carte, Build string
	Version            int
	Impose             I0Layout
	Detecte            I0Layout
	DetecteErr         error
	IndexBitUns        int
	Records            int
	// PosAuto / PosCat : positions rendues sous les réglages de PRODUCTION des positions
	// (`DefaultScanFilmOptions` + bornes de la carte), découpage auto-détecté puis imposé.
	PosAuto, PosCat int
	// TrackAuto / TrackCat : idem sous les réglages des TOUCHES (`BuildBipedTracks`).
	TrackAuto, TrackCat int
	// BrutAuto / BrutCat : les ENREGISTREMENTS que la marche accepte, TOUS FILTRES DÉSARMÉS
	// (saturation comprise) — la population sur laquelle la porte de région mord, et la seule
	// qui se compare à la mesure du 2026-09-03 (« 27 faux enregistrements sur 267 400 »).
	BrutAuto, BrutCat int
	// LibreAuto / LibreCat : idem, generation de handle NON filtree (RequireTag1 desarme) — mesure
	// jouee sur les seuls films ou les deux decoupages divergent.
	LibreAuto, LibreCat int
	ErrAuto, ErrCat     error
}

// Difference dit si le catalogue et l'auto-détection donnent le même découpage.
func (l e192Ligne) Difference() bool {
	if l.DetecteErr != nil {
		return true
	}
	return l.Impose != l.Detecte
}

func TestE192CatalogueContreDetection(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	cat, err := LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	var lignes []e192Ligne
	for _, dir := range dirs {
		l, ok := e192Mesure(t, cat, dir)
		if !ok {
			continue
		}
		lignes = append(lignes, l)
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].Film < lignes[j].Film })
	e192Rapport(t, lignes)
}

// e192Mesure remplit la ligne d'un film : identité, découpages rivaux, et les quatre balayages.
func e192Mesure(t *testing.T, cat *MapQuantCatalog, dir string) (e192Ligne, bool) {
	t.Helper()
	l := e192Ligne{Film: filepath.Base(dir)}
	carte, ok := e192CarteDuFilm[l.Film]
	if !ok {
		t.Logf("  %-10s SAUTÉ : film absent de la table des cartes versionnée", l.Film)
		return l, false
	}
	l.Carte = carte
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Logf("  %-10s SAUTÉ : carte %q absente du catalogue (%v)", l.Film, carte, err)
		return l, false
	}
	l.Impose = entry.Layout()
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Logf("  %-10s SAUTÉ : film illisible (%v)", l.Film, err)
		return l, false
	}
	l.Version, l.Build = e192Identite(film)
	lay, rep, derr := DetectI0LayoutOf(film)
	l.Detecte, l.DetecteErr, l.IndexBitUns, l.Records = lay, derr, rep.IndexBitOnes, rep.Records
	e192Balayages(film, entry, &l)
	return l, true
}

// e192Identite rend la version majeure et le build du film, lus dans `chunk_00`.
func e192Identite(film *filmsource.Film) (int, string) {
	reg, ok := FilmRegistryChunk(film)
	if !ok {
		return FilmMajorVersionUnknown, "sans chunk_00"
	}
	v, _ := FilmMajorVersionFromHeader(reg)
	id, err := ReadFilmIdentity(reg)
	if err != nil {
		return v, "sans section"
	}
	return v, id.Build
}

// e192Balayages joue les QUATRE balayages : positions et pistes de touche, découpage auto-détecté
// puis imposé par le catalogue. Les réglages sont ceux des DEUX chemins de `killcollector`.
func e192Balayages(film *filmsource.Film, entry MapQuantEntry, l *e192Ligne) {
	rng := entry.Range()
	impose := l.Impose

	pos := DefaultScanFilmOptions()
	pos.WorldRange = &rng
	a, errA := ScanBipedPositions(NewFilmContext(film), pos)
	pos.Layout = &impose
	b, errB := ScanBipedPositions(NewFilmContext(film), pos)
	l.PosAuto, l.PosCat, l.ErrAuto, l.ErrCat = len(a), len(b), errA, errB

	// Réglages des TOUCHES : `BuildBipedTracks` désarme les deux filtres et énumère les chunks.
	tr := DefaultScanFilmOptions()
	tr.MaxSpeedMPS, tr.IsolationGapMS = 0, 0
	tr.WorldRange = &rng
	tr.Chunks = FilmChunkNumbers(film)
	c, _ := ScanBipedPositions(NewFilmContext(film), tr)
	tr.Layout = &impose
	d, _ := ScanBipedPositions(NewFilmContext(film), tr)
	l.TrackAuto, l.TrackCat = len(c), len(d)

	// BRUT : tous filtres désarmés, saturation comprise. C'est la population d'ENREGISTREMENTS
	// que la porte de région accepte ou écarte, sans qu'aucun post-traitement ne la retouche.
	br := ScanFilmOptions{RequireTag1: true, WorldRange: &rng}
	e, _ := ScanBipedPositions(NewFilmContext(film), br)
	br.Layout = &impose
	f, _ := ScanBipedPositions(NewFilmContext(film), br)
	l.BrutAuto, l.BrutCat = len(e), len(f)
	if !l.Difference() {
		return
	}
	// TAG LIBRE, sur les seuls films qui divergent : la génération du handle ne filtre plus, la
	// population est alors TOUT ce que la marche reconnaît. C'est la variante la plus large, celle
	// qui borne par le haut ce que la porte de région écarte.
	br.RequireTag1, br.Layout = false, nil
	g, _ := ScanBipedPositions(NewFilmContext(film), br)
	br.Layout = &impose
	h, _ := ScanBipedPositions(NewFilmContext(film), br)
	l.LibreAuto, l.LibreCat = len(g), len(h)
}

// e192Rapport colle le tableau : une ligne par film, puis les totaux et la liste des cartes où
// les deux découpages divergent.
func e192Rapport(t *testing.T, lignes []e192Ligne) {
	t.Helper()
	t.Logf("######## 1.9.2 — CATALOGUE CONTRE AUTO-DÉTECTION, %d films ########", len(lignes))
	t.Logf("  %-10s %-9s %-22s %-38s %-30s %-4s %10s %10s %8s %10s %8s %10s %8s",
		"film", "build", "carte", "catalogue", "auto-détecté", "diff",
		"pos auto", "pos cat", "Δpos", "piste cat", "Δpiste", "brut cat", "Δbrut")
	var dPos, dTrack, dBrut int
	var divergentes []string
	for _, l := range lignes {
		auto := "ERREUR"
		if l.DetecteErr == nil {
			auto = l.Detecte.String()
		}
		diff := "="
		if l.Difference() {
			diff = "OUI"
			divergentes = append(divergentes, fmt.Sprintf("%s (%s)", l.Carte, l.Film))
		}
		t.Logf("  %-10s %-9s %-22s %-38s %-30s %-4s %10d %10d %+8d %10d %+8d %10d %+8d",
			l.Film, l.Build, l.Carte, l.Impose.String(), auto, diff,
			l.PosAuto, l.PosCat, l.PosCat-l.PosAuto,
			l.TrackCat, l.TrackCat-l.TrackAuto, l.BrutCat, l.BrutCat-l.BrutAuto)
		if l.ErrAuto != nil || l.ErrCat != nil {
			t.Logf("      erreurs de balayage : auto=%v catalogue=%v", l.ErrAuto, l.ErrCat)
		}
		dPos += l.PosCat - l.PosAuto
		dTrack += l.TrackCat - l.TrackAuto
		dBrut += l.BrutCat - l.BrutAuto
	}
	t.Logf("  TOTAL : Δpositions %+d, Δpistes %+d, Δbruts %+d", dPos, dTrack, dBrut)
	t.Logf("  CARTES OÙ CATALOGUE ET DÉTECTION DIFFÈRENT : %d — %v", len(divergentes), divergentes)
	for _, l := range lignes {
		t.Logf("  détail %-10s : records i0 profilés %6d, bit d'index à 1 sur %5d ; brut auto %7d -> cat %7d",
			l.Film, l.Records, l.IndexBitUns, l.BrutAuto, l.BrutCat)
	}
	for _, l := range lignes {
		if l.LibreAuto == 0 {
			continue
		}
		t.Logf("  tag libre %-10s : auto %7d -> catalogue %7d (%+d)",
			l.Film, l.LibreAuto, l.LibreCat, l.LibreCat-l.LibreAuto)
	}
}
