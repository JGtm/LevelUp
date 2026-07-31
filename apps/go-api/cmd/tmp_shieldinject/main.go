// tmp_shieldinject — ajoute le BOUCLIER decode (Point.Sh de l'artefact) au bloc de donnees
// de la page de demonstration, sans la regenerer.
//
// La page a ete EDITEE A LA MAIN. Ce binaire ne la reecrit pas : il remplace uniquement le
// contenu de <script id="replay-data">, et a l'interieur uniquement :
//   - chaque point de trajectoire, auquel il AJOUTE un 6e element (bouclier en pourcentage
//     entier, ou null quand le film ne l'a pas replique a cet instant) ;
//   - la cle "stats", enrichie de la couverture du bouclier.
//
// Tout le reste — bornes, callouts, objectifs, structure, tirs, roster — est recopie
// OCTET POUR OCTET (json.RawMessage) : ni rearrondi, ni reformate. Les cinq premiers
// elements de chaque point le sont aussi.
//
// CE QUI N'EST PAS INJECTE, ET POURQUOI : la sante (i4) et le compteur de reapparition
// (ti=5 i1). Leurs temoins n'ont pas passe la barre (couverture 0,6 % pour la sante ;
// 1,75x seulement contre le hasard, sur 13 observations, pour la reapparition). Cf.
// etat_valeurs.md. Une barre de vie fausse est pire qu'absente.
//
// Usage : go run ./cmd/tmp_shieldinject <artefact.json> <page.html>
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay"
)

const dataOpen = `<script id="replay-data" type="application/json">`
const dataClose = `</script>`

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_shieldinject <artefact.json> <page.html>")
		os.Exit(2)
	}
	shield, err := loadShield(os.Args[1])
	if err != nil {
		fmt.Println("artefact:", err)
		os.Exit(1)
	}
	page, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Println("page:", err)
		os.Exit(1)
	}
	html := string(page)
	i := strings.Index(html, dataOpen)
	if i < 0 {
		fmt.Println("bloc de donnees introuvable dans la page")
		os.Exit(1)
	}
	start := i + len(dataOpen)
	rel := strings.Index(html[start:], dataClose)
	if rel < 0 {
		fmt.Println("fin du bloc de donnees introuvable")
		os.Exit(1)
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(html[start:start+rel]), &fields); err != nil {
		fmt.Println("donnees de la page:", err)
		os.Exit(1)
	}
	tracks, withSh, points, err := injectTracks(fields["tracks"], shield)
	if err != nil {
		fmt.Println("tracks:", err)
		os.Exit(1)
	}
	fields["tracks"] = tracks
	if err := addStat(fields, "shieldPts", withSh); err != nil {
		fmt.Println("stats:", err)
		os.Exit(1)
	}
	out, err := json.Marshal(fields)
	if err != nil {
		fmt.Println("serialisation:", err)
		os.Exit(1)
	}
	page = []byte(html[:start] + string(out) + html[start+rel:])
	if err := os.WriteFile(os.Args[2], page, 0o644); err != nil {
		fmt.Println("ecriture:", err)
		os.Exit(1)
	}
	fmt.Printf("bouclier injecte : %d points sur %d en portent un (%.1f %%) ; page %d octets\n",
		withSh, points, 100*float64(withSh)/float64(points), len(page))
}

// loadShield indexe le bouclier de l'artefact par (slot, frame). Absent de la table = le
// film n'a pas replique le bouclier a cet instant ; le point recevra null, PAS zero — un
// bouclier a zero est une mesure, une absence n'en est pas une.
func loadShield(path string) (map[uint64]int, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		return nil, err
	}
	out := map[uint64]int{}
	for _, tr := range doc.Tracks {
		for _, p := range tr.Points {
			if p.Sh == nil {
				continue
			}
			v := int(*p.Sh*100 + 0.5)
			if v < 0 {
				v = 0
			}
			if v > 100 {
				v = 100
			}
			out[uint64(tr.Slot)<<32|uint64(uint32(p.T))] = v
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("l'artefact ne porte aucun bouclier (champ sh)")
	}
	return out, nil
}

// injectTracks reecrit le tableau des trajectoires en ajoutant un 6e element a chaque
// point. Les elements existants sont recopies tels quels (RawMessage) : aucun nombre n'est
// re-serialise, donc aucun arrondi ne bouge.
func injectTracks(raw json.RawMessage, shield map[uint64]int) (json.RawMessage, int, int, error) {
	var tracks []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tracks); err != nil {
		return nil, 0, 0, err
	}
	withSh, points := 0, 0
	for _, tr := range tracks {
		var slot uint32
		if err := json.Unmarshal(tr["slot"], &slot); err != nil {
			return nil, 0, 0, fmt.Errorf("slot: %w", err)
		}
		var pts []json.RawMessage
		if err := json.Unmarshal(tr["p"], &pts); err != nil {
			return nil, 0, 0, fmt.Errorf("points du slot %d: %w", slot, err)
		}
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, ptRaw := range pts {
			var elems []json.RawMessage
			if err := json.Unmarshal(ptRaw, &elems); err != nil {
				return nil, 0, 0, fmt.Errorf("point %d du slot %d: %w", i, slot, err)
			}
			if len(elems) < 5 {
				return nil, 0, 0, fmt.Errorf("point %d du slot %d : %d elements, 5 attendus", i, slot, len(elems))
			}
			if len(elems) > 5 { // deja injecte : on repart des 5 premiers, l'outil est idempotent
				elems = elems[:5]
			}
			var t int
			if err := json.Unmarshal(elems[0], &t); err != nil {
				return nil, 0, 0, err
			}
			points++
			sh := "null"
			if v, ok := shield[uint64(slot)<<32|uint64(uint32(t))]; ok {
				sh = strconv.Itoa(v)
				withSh++
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('[')
			for k, e := range elems {
				if k > 0 {
					buf.WriteByte(',')
				}
				buf.Write(e)
			}
			buf.WriteByte(',')
			buf.WriteString(sh)
			buf.WriteByte(']')
		}
		buf.WriteByte(']')
		tr["p"] = json.RawMessage(buf.Bytes())
	}
	out, err := json.Marshal(tracks)
	return out, withSh, points, err
}

// addStat ajoute une entree au bloc "stats" sans toucher aux autres.
func addStat(fields map[string]json.RawMessage, key string, val int) error {
	stats := map[string]json.RawMessage{}
	if raw, ok := fields["stats"]; ok {
		if err := json.Unmarshal(raw, &stats); err != nil {
			return err
		}
	}
	stats[key] = json.RawMessage(strconv.Itoa(val))
	out, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	fields["stats"] = out
	return nil
}
