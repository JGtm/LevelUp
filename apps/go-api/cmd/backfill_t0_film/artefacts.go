package main

// artefacts.go — LA LECTURE MINIMALE DU CORPUS D'ARTEFACTS.
//
// # POURQUOI « MINIMALE » EST UNE CONTRAINTE ET PAS UN GOUT
//
// Le corpus local pese ~240 Mo pour une centaine d'artefacts, et un artefact porte bien plus
// que des positions : geometrie de carte, calques d'objectifs, fil des eliminations, courbe de
// score, couvertures. Desirialiser `replay.ReplayDocument` en entier pour n'en lire que les
// pistes ferait naitre des dizaines de milliers de structures par fichier, dont aucune ne
// servirait. La structure ci-dessous ne declare QUE les cinq champs dont le detecteur a besoin ;
// `encoding/json` saute tout le reste sans rien allouer. Precedent du depot :
// `cmd/zone-attribution/census.go` (lecture de la seule version de schema).
//
// # LE DETECTEUR EST CELUI DE LA PRODUCTION, PAS UNE COPIE
//
// `replay.DetectT0Film` est appele ICI sur les pistes PUBLIEES de l'artefact — exactement les
// memes que celles que `build.go` lui donne au moment de la cuisson (le cablage A5 le place
// APRES `buildCoverage`, donc sur les pistes du document). Recalculer depuis l'artefact rend
// donc, par construction, la valeur que la cuisson aurait publiee : c'est ce qui permet de
// reparer l'historique SANS recuire un seul film (decision D4 du plan).
//
// # LES TROIS REFUS QUI NE VIENNENT PAS DU DETECTEUR
//
// Un fichier illisible, un artefact sans `matchId` et un artefact sans `originMs` ne sont pas
// des refus de mesure : ce sont des artefacts inexploitables. Ils sont nommes a part pour que
// le compte rendu ne les melange jamais aux quatre refus de `t0_film.go`.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay"
)

const (
	// raisonIllisible : le fichier ne s'ouvre pas ou n'est pas un JSON valide.
	raisonIllisible = "artefactIllisible"
	// raisonSansMatchID : l'artefact ne revendique aucun match — rien a joindre au registre.
	raisonSansMatchID = "artefactSansMatchId"
	// raisonSansOrigine : sans `originMs`, la frame 0 n'est posee sur aucune horloge et le
	// resultat du detecteur ne serait rattachable a rien (meme garde que le cablage A5).
	raisonSansOrigine = "artefactSansOrigine"
)

// pointMinimal / pisteMinimal / artefactMinimal : la part de l'artefact reellement lue.
// Les noms de champs JSON suivent `replay.Point` / `replay.Track` / `replay.ReplayDocument`.
type pointMinimal struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type pisteMinimale struct {
	XUID   string         `json:"xuid"`
	Points []pointMinimal `json:"points"`
}

type artefactMinimal struct {
	SchemaVersion   int             `json:"schemaVersion"`
	MatchID         string          `json:"matchId"`
	FrameIntervalMS int             `json:"frameIntervalMs"`
	OriginMs        *int64          `json:"originMs"`
	Tracks          []pisteMinimale `json:"tracks"`
}

// verdictArtefact : ce qu'un artefact a rendu, mesure ou refus.
type verdictArtefact struct {
	fichier string
	matchID string
	schema  int
	// t0FilmMs n'a de sens que si `detecte`. Il est sur l'axe du MATCH (millisecondes depuis
	// `start_time`), comme `header.t0_ms` — c'est ce qui autorise l'addition au start canonique.
	t0FilmMs int64
	detecte  bool
	// raison nomme le refus : l'une des trois constantes ci-dessus, ou l'un des quatre refus
	// publies par `replay.T0FilmCoverage.Reason`.
	raison string
	// rafale / pistes / pistesEnMouvement : les temoins du detecteur, republies pour que le
	// compte rendu d'un refus dise POURQUOI sans rouvrir le fichier.
	rafale, pistes, pistesEnMouvement int
}

// scannerArtefacts lit tout le dossier des artefacts d'un titre et rend un verdict par fichier,
// trie par nom — deux passages se comparent alors ligne a ligne.
//
// Un fichier a la fois : les octets d'un artefact et ses pistes converties ne coexistent que le
// temps d'un tour de boucle. Le corpus entier ne tient jamais en memoire.
func scannerArtefacts(dir string) ([]verdictArtefact, error) {
	entrees, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("dossier des artefacts %s: %w", dir, err)
	}
	out := make([]verdictArtefact, 0, len(entrees))
	for _, e := range entrees {
		// Suffixe EXACT : le dossier porte des sauvegardes manuelles (`.json.ancien-11juillet`)
		// qui ne sont pas des artefacts courants et ne doivent rien reparer.
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		out = append(out, lireUnArtefact(filepath.Join(dir, e.Name())))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].fichier < out[j].fichier })
	return out, nil
}

// lireUnArtefact decode un artefact et lui applique le detecteur de production.
func lireUnArtefact(path string) verdictArtefact {
	v := verdictArtefact{fichier: filepath.Base(path)}
	doc, err := decoderMinimal(path)
	if err != nil {
		v.raison = raisonIllisible
		return v
	}
	v.matchID, v.schema = doc.MatchID, doc.SchemaVersion
	switch {
	case v.matchID == "":
		v.raison = raisonSansMatchID
		return v
	case doc.OriginMs == nil:
		v.raison = raisonSansOrigine
		return v
	}
	t0, cov := replay.DetectT0Film(pistesDe(doc), doc.FrameIntervalMS, *doc.OriginMs, v.matchID)
	v.pistes, v.pistesEnMouvement, v.rafale = cov.Tracks, cov.Moving, cov.Burst
	if t0 == nil {
		v.raison = cov.Reason
		return v
	}
	v.detecte, v.t0FilmMs = true, *t0
	return v
}

// decoderMinimal ouvre le fichier en flux : les octets ne sont jamais tous en memoire en plus
// de la structure decodee.
func decoderMinimal(path string) (artefactMinimal, error) {
	f, err := os.Open(path)
	if err != nil {
		return artefactMinimal{}, err
	}
	defer func() { _ = f.Close() }()
	var doc artefactMinimal
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return artefactMinimal{}, err
	}
	return doc, nil
}

// pistesDe ramene les pistes lues a l'entree NEUTRE du detecteur (`replay.T0FilmTrack`), celle
// qui existe precisement pour que cette commande n'ait pas a reconstruire un `replay.Track`
// complet — ni, pire, a recopier la logique de detection.
func pistesDe(doc artefactMinimal) []replay.T0FilmTrack {
	out := make([]replay.T0FilmTrack, 0, len(doc.Tracks))
	for i := range doc.Tracks {
		pts := make([]replay.T0FilmPoint, len(doc.Tracks[i].Points))
		for j, p := range doc.Tracks[i].Points {
			pts[j] = replay.T0FilmPoint{T: p.T, X: p.X, Y: p.Y, Z: p.Z}
		}
		out = append(out, replay.T0FilmTrack{XUID: doc.Tracks[i].XUID, Points: pts})
	}
	return out
}
