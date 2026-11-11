//go:build research

// Package duree : mesure jetable (plan marche complete, A0.3) de la duree de decodage d un film
// et de la publication de son rejeu, un film a la fois sous memguard.
//
// Entrees (le test est SAUTE sans elles) :
//
//	DUREE_FILMS      fichier des films, une ligne par film : `id<TAB>build<TAB>mode<TAB>niveau...`
//	                 (format de temoins_duree.txt ; lignes `#` ignorees)
//	DUREE_RACINE     racine des films (`<parc>/data/cache/film_chunks`), lecture seule
//	DUREE_CATALOGUE  map_quant_bounds.json (entree de carte retrouvee par module = colonne niveau)
//	DUREE_SORTIE     dossier de sortie, HORS de data/ : `duree.tsv` et un rejeu `<id>.replay.json` par film
//
//	cd libs/film && DUREE_FILMS=... DUREE_RACINE=... DUREE_CATALOGUE=... DUREE_SORTIE=C:\t\marche\A0\duree \
//	  go test -tags=research ./research/duree/ -run TestDureeDesTemoins -v -timeout 30m
//
// CE QUI EST MESURE, PAR LES SEULS POINTS D ENTREE PUBLICS : `replay.BuildFromFilmAvecFaits` rend
// le document ET les faits ; son observateur (`Options.Observe`) date la fin du balayage (etape
// `clockOrigin`, la derniere de `scanFilmInputs`), ce qui separe
//
//	charge_ms   chargement du film : lecture des chunks, decompression, decoupage en paquets
//	decode_ms   decodage film -> faits : du debut de l appel a la fin du balayage
//	pub_ms      publication faits -> rejeu, observee dans le meme appel : fin du balayage -> retour
//	pub_faits_ms  le meme travail par `replay.BuildFromFacts` sur les faits rendus (aucun octet de film)
//
// LIMITE : les options sont minimales (carte seule) ; killsource, statborg, objectifs, catalogues
// de zones et geometrie n y sont pas, donc la publication est plus courte que celle de la cuisson
// de production, et le decodage n inclut ni le fil des morts ni killsource. Le pic memoire est
// celui de la sentinelle sur tout le film (chargement, decodage, publication, serialisation).
package duree

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/memguard"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

const (
	plafondGiB = 6
	nomOutil   = "film_re/duree"
)

type temoin struct{ id, build, niveau string }

func lireTemoins(chemin string) ([]temoin, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin de l operateur
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []temoin
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		c := strings.Split(l, "\t")
		if len(c) < 4 {
			return nil, fmt.Errorf("ligne sans niveau : %q", l)
		}
		out = append(out, temoin{id: c[0], build: c[1], niveau: c[3]})
	}
	return out, sc.Err()
}

// entreeDeCarte rend l entree du catalogue dont le module est le niveau du film.
func entreeDeCarte(cat *decfilm.MapQuantCatalog, niveau string) (decfilm.MapQuantEntry, error) {
	for _, e := range cat.Maps {
		if e.Module == niveau {
			return e, nil
		}
	}
	return decfilm.MapQuantEntry{}, fmt.Errorf("aucune carte de module %q au catalogue", niveau)
}

func mio(o uint64) float64 { return float64(o) / (1 << 20) }

func TestDureeDesTemoins(t *testing.T) {
	fichier, racine := os.Getenv("DUREE_FILMS"), os.Getenv("DUREE_RACINE")
	catalogue, sortie := os.Getenv("DUREE_CATALOGUE"), os.Getenv("DUREE_SORTIE")
	if fichier == "" || racine == "" || catalogue == "" || sortie == "" {
		t.Skip("DUREE_FILMS, DUREE_RACINE, DUREE_CATALOGUE et DUREE_SORTIE sont requis (mesure sur le parc local)")
	}
	for seg := range strings.SplitSeq(filepath.ToSlash(filepath.Clean(sortie)), "/") {
		if strings.EqualFold(seg, "data") {
			t.Fatalf("DUREE_SORTIE sous data/ refuse : %s", sortie)
		}
	}
	temoins, err := lireTemoins(fichier)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := decfilm.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatal(err)
	}
	tsv, err := os.Create(filepath.Join(sortie, "duree.tsv")) //nolint:gosec // dossier de l operateur
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tsv.Close() }()
	fmt.Fprintln(tsv, "film\tbuild\tdecode_ms\tpub_ms\tpic_mio\trejeu_octets\tcharge_ms\tpub_faits_ms\tniveau")
	for _, tm := range temoins {
		entree, err := entreeDeCarte(cat, tm.niveau)
		if err != nil {
			t.Fatalf("%s : %v", tm.id, err)
		}
		ligne := mesurer(t, racine, sortie, tm, entree)
		fmt.Fprintln(tsv, ligne)
		t.Log(ligne)
		runtime.GC()
	}
}

// mesurer traite UN film sous sa propre sentinelle et rend sa ligne de TSV.
func mesurer(t *testing.T, racine, sortie string, tm temoin, entree decfilm.MapQuantEntry) string {
	t.Helper()
	ctx := context.Background()
	garde := memguard.Arm(ctx, nomOutil, plafondGiB, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets) - arret\n", tm.id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()

	dir := filepath.Join(racine, tm.id)
	debutCharge := time.Now()
	var meta []decfilm.ChunkMeta
	if src, found, err := filmcache.OpenChunkDir(dir); err == nil && found {
		meta = src.Meta()
	}
	film, err := decfilm.LoadDir(dir, meta)
	if err != nil {
		t.Fatalf("%s : chargement : %v", tm.id, err)
	}
	charge := time.Since(debutCharge)

	var finBalayage time.Time
	opt := replay.Options{MapQuant: &entree}
	opt.Observe = func(step string, _ any) {
		if step == "clockOrigin" && finBalayage.IsZero() {
			finBalayage = time.Now()
		}
	}
	debut := time.Now()
	doc, faits, err := replay.BuildFromFilmAvecFaits(ctx, tm.id, "halo_infinite", film, opt)
	fin := time.Now()
	if err != nil {
		t.Fatalf("%s : cuisson : %v", tm.id, err)
	}
	if faits == nil || finBalayage.IsZero() {
		t.Fatalf("%s : faits nil ou fin de balayage non observee", tm.id)
	}
	decode, pub := finBalayage.Sub(debut), fin.Sub(finBalayage)

	debutFaits := time.Now()
	_ = replay.BuildFromFacts(ctx, tm.id, "halo_infinite", faits, replay.Options{MapQuant: &entree})
	pubFaits := time.Since(debutFaits)

	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("%s : serialisation : %v", tm.id, err)
	}
	if err := os.WriteFile(filepath.Join(sortie, tm.id+".replay.json"), blob, 0o600); err != nil {
		t.Fatal(err)
	}
	build := tm.build
	if doc.Coverage != nil && doc.Coverage.Decoder != nil && doc.Coverage.Decoder.Build != "" {
		build = doc.Coverage.Decoder.Build
	}
	return fmt.Sprintf("%s\t%s\t%d\t%d\t%.0f\t%d\t%d\t%d\t%s", tm.id, build, decode.Milliseconds(),
		pub.Milliseconds(), mio(garde.Peak()), len(blob), charge.Milliseconds(), pubFaits.Milliseconds(), tm.niveau)
}
