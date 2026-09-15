package replay

// vehicules_chassis_inventaire_test.go — L INVENTAIRE DES CHASSIS DE VEHICULE RENCONTRES,
// mesure AVANT de coder le lot 1.9.9 (plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`).
//
// CE QU IL MESURE, ET SUR QUOI. Les ARTEFACTS DE REJEU DEJA CUITS
// (`data/cache/replays/{slug}/*.json`), en LECTURE SEULE : aucun film n est redecode, aucune
// base n est ouverte, rien n est ecrit. Chaque vie de vehicule publiee y porte deja son mot
// d identite de chassis (`chassis`, hexadecimal 8 chiffres) et la famille que
// `vehicle_families.go` lui a resolue (`family`, absente quand la table ne le connait pas) : la
// question du lot — « quels chassis rencontre le constructeur, lesquels restent inconnus » — se
// repond donc SANS decodage, ce que la contrainte machine du chantier (un seul decodage a la
// fois, six executeurs en parallele) rend decisif.
//
// POURQUOI L ARTEFACT EST UNE MESURE LEGITIME ICI. Le champ `chassis` est la SORTIE DIRECTE du
// mot `MPPWord32` du record de creation `ti=40` (`vehicle_tracks.go`, `formatChassisID`) : il ne
// depend d aucune table du depot. La famille, elle, est la table — c est exactement la jointure
// que ce lot doit completer. Un artefact cuit a un schema anterieur porte donc la MEME population
// de chassis qu une cuisson d aujourd hui, et c est la population qui est demandee.
//
// CE QU IL NE MESURE PAS : les films dont aucun artefact n existe au parc. Le tableau cite le
// nombre d artefacts lus et la liste des identifiants demandes qui manquent — un negatif se
// declare, il ne se tait pas.
//
// LA BANQUE DE SONS d un chassis vient du catalogue `damagetag` embarque (`data/labels.tsv`,
// colonne `detail`, motif `vehi <chassis>`) : c est la seconde source de nommage de
// `vehicle_families.go`, celle qui a nomme le Ghost, la Banshee et le Chopper.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 \
//	PARC_ARTEFACTS=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite \
//	  go test ./internal/games/halo_infinite/film/replay/ -run TestInventaireChassisDesArtefacts -count=1 -v
//
// `PARC_IDS` (short8 separes par des virgules) borne la lecture a ces artefacts et DECLARE ceux
// qui manquent. Sans `PARC_ARTEFACTS`, le test est saute : il n a pas de donnee a lire.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

const (
	parcArtefactsEnv = "PARC_ARTEFACTS"
	parcIDsEnv       = "PARC_IDS"
)

// inventaireArtefact est le SOUS-ENSEMBLE d un artefact que cet inventaire lit. Redeclare ici
// plutot que reutilise : `ReplayDocument` tirerait tout le document en memoire pour trois
// champs, et la forme lue est celle du FICHIER, pas du type courant (les artefacts du parc sont
// cuits a des schemas anterieurs).
type inventaireArtefact struct {
	SchemaVersion int `json:"schemaVersion"`
	Vehicles      []struct {
		Chassis string `json:"chassis"`
		Family  string `json:"family"`
		Samples []struct {
			T int `json:"t"`
		} `json:"samples"`
		T0    int `json:"t0"`
		T1Max int `json:"t1max"`
	} `json:"vehicles"`
	FrameCount int `json:"frameCount"`
}

// ligneChassis agrege UN chassis sur l ensemble des artefacts lus.
type ligneChassis struct {
	chassis   string
	famille   string
	vies      int
	immobiles int // vies sans AUCUN echantillon de trajectoire
	toutLeM   int // vies dont la fenetre couvre le match entier (t0 == 0 et t1max == frameCount-1)
	films     map[string]int
}

func TestInventaireChassisDesArtefacts(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv(parcArtefactsEnv))
	if dir == "" {
		t.Skipf("%s absent : inventaire des chassis saute (aucun artefact a lire)", parcArtefactsEnv)
	}
	fichiers, manquants := inventaireFichiers(t, dir)
	par := map[string]*ligneChassis{}
	lus := 0
	for _, f := range fichiers {
		doc, err := lireArtefactInventaire(f)
		if err != nil {
			t.Errorf("artefact illisible %s : %v", filepath.Base(f), err)
			continue
		}
		lus++
		cumulerChassis(par, short8DeFichier(f), doc)
	}
	t.Logf("artefacts lus : %d ; demandes absents : %s", lus, listeOuAucun(manquants))
	t.Log("\n" + tableauChassis(par))
	if len(par) == 0 {
		t.Fatalf("aucun chassis releve sur %d artefacts : la mesure n a rien a dire", lus)
	}
}

// inventaireFichiers rend les artefacts a lire et les identifiants demandes qui manquent.
func inventaireFichiers(t *testing.T, dir string) (fichiers []string, manquants []string) {
	t.Helper()
	if ids := strings.TrimSpace(os.Getenv(parcIDsEnv)); ids != "" {
		for _, raw := range strings.Split(ids, ",") {
			id := strings.TrimSpace(raw)
			if id == "" {
				continue
			}
			p := filepath.Join(dir, id+".json")
			if _, err := os.Stat(p); err != nil {
				manquants = append(manquants, id)
				continue
			}
			fichiers = append(fichiers, p)
		}
		return fichiers, manquants
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("dossier d artefacts illisible (%s) : %v", dir, err)
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".json") || strings.HasSuffix(n, ".derived.json") {
			continue
		}
		fichiers = append(fichiers, filepath.Join(dir, n))
	}
	sort.Strings(fichiers)
	return fichiers, nil
}

func lireArtefactInventaire(path string) (inventaireArtefact, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l operateur, lecture seule
	if err != nil {
		return inventaireArtefact{}, err
	}
	var doc inventaireArtefact
	if err := json.Unmarshal(raw, &doc); err != nil {
		return inventaireArtefact{}, err
	}
	return doc, nil
}

func short8DeFichier(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".json")
}

// cumulerChassis ajoute les vies d UN artefact a l agregat.
func cumulerChassis(par map[string]*ligneChassis, film string, doc inventaireArtefact) {
	for _, v := range doc.Vehicles {
		if v.Chassis == "" {
			continue // vie sans record de creation lu : aucun mot d identite a inventorier
		}
		l := par[v.Chassis]
		if l == nil {
			l = &ligneChassis{chassis: v.Chassis, films: map[string]int{}}
			par[v.Chassis] = l
		}
		if v.Family != "" {
			l.famille = v.Family
		}
		l.vies++
		l.films[film]++
		if len(v.Samples) == 0 {
			l.immobiles++
		}
		if v.T0 == 0 && doc.FrameCount > 0 && v.T1Max >= doc.FrameCount-1 {
			l.toutLeM++
		}
	}
}

// tableauChassis rend le tableau A COLLER : une ligne par chassis, triee vies decroissantes.
func tableauChassis(par map[string]*ligneChassis) string {
	lignes := make([]*ligneChassis, 0, len(par))
	for _, l := range par {
		lignes = append(lignes, l)
	}
	sort.Slice(lignes, func(i, j int) bool {
		if lignes[i].vies != lignes[j].vies {
			return lignes[i].vies > lignes[j].vies
		}
		return lignes[i].chassis < lignes[j].chassis
	})
	var b strings.Builder
	fmt.Fprintf(&b, "%-10s %-22s %6s %10s %12s %6s  %-28s  %s\n",
		"chassis", "famille", "vies", "immobiles", "toutLeMatch", "films",
		"films (short8:vies)", "banque(s) damagetag")
	for _, l := range lignes {
		fam := l.famille
		if fam == "" {
			fam = "INCONNUE"
		}
		fmt.Fprintf(&b, "%-10s %-22s %6d %10d %12d %6d  %-28s  %s\n",
			l.chassis, fam, l.vies, l.immobiles, l.toutLeM, len(l.films),
			detailFilms(l.films), banquesDuChassis(l.chassis))
	}
	return b.String()
}

// detailFilms rend « short8:vies » par film, les plus fournis d abord, borne a cinq entrees —
// au-dela le tableau cesse d etre lisible et le compte de films dit deja l etalement.
func detailFilms(films map[string]int) string {
	type fv struct {
		film string
		n    int
	}
	out := make([]fv, 0, len(films))
	for f, n := range films {
		out = append(out, fv{f, n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].film < out[j].film
	})
	parts := make([]string, 0, len(out))
	for i, e := range out {
		if i == 5 {
			parts = append(parts, fmt.Sprintf("+%d", len(out)-5))
			break
		}
		parts = append(parts, fmt.Sprintf("%s:%d", e.film, e.n))
	}
	return listeOuAucun(parts)
}

// banquesDuChassis releve, dans le catalogue `damagetag` embarque, les banques de sons citees
// par les tags de degat dont le detail nomme `vehi <chassis>`. Vide = le catalogue ne le nomme
// pas (ce qui est le cas de la majorite des chassis, dont le Warthog).
func banquesDuChassis(chassis string) string {
	motif := "vehi " + chassis
	vues := map[string]bool{}
	for _, lbl := range damagetag.Labels() {
		i := strings.Index(lbl.Detail, motif)
		if i < 0 {
			continue
		}
		for _, mot := range strings.FieldsFunc(lbl.Detail[i:], func(r rune) bool {
			return r == ' ' || r == ',' || r == '/' || r == ')' || r == '(' || r == '\t'
		}) {
			if strings.HasPrefix(mot, "sb_") {
				vues[mot] = true
			}
		}
	}
	return listeOuAucun(triees(vues))
}

func triees(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func listeOuAucun(v []string) string {
	if len(v) == 0 {
		return "-"
	}
	return strings.Join(v, " ")
}
