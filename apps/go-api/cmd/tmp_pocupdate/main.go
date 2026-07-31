// cmd/tmp_pocupdate — ETAPE 4.2 du plan : injecter dans le POC l'artefact de rejeu a jour
// et sa COUVERTURE.
//
// LE POC S'EDITE, IL NE SE REGENERE PAS (decision n°5 du plan). Ce programme ne reecrit donc
// pas la page : il remplace UNIQUEMENT les entrees du bloc de donnees que le nouveau pipeline
// change — les tirs, les lancers, les compteurs — et il en AJOUTE une, la couverture. Tout le
// reste de la page, y compris les calques que ce chantier ne touche pas, est recopie a
// l'identique.
//
// POURQUOI UN PROGRAMME PLUTOT QU'UNE EDITION A LA MAIN : le bloc de donnees fait 2,9 Mo sur
// une seule ligne. Une edition manuelle y est invérifiable, et une erreur de virgule casse la
// page entiere sans rien dire.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/weaponv3"
)

// dataBlock isole le bloc JSON de la page. Le motif est ancre sur l'identifiant du script,
// pas sur une position : la page evolue, l'identifiant non.
var dataBlock = regexp.MustCompile(`(?s)(<script id="replay-data" type="application/json">)(.*?)(</script>)`)

func main() {
	poc := flag.String("poc", "", "chemin du POC (replay_demo.html)")
	art := flag.String("artifact", "", "chemin de l'artefact de rejeu (JSON)")
	out := flag.String("out", "", "chemin de sortie (defaut : ecrase le POC)")
	flag.Parse()
	if *poc == "" || *art == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_pocupdate -poc <html> -artifact <json> [-out <html>]")
		os.Exit(2)
	}
	if *out == "" {
		*out = *poc
	}
	page, err := os.ReadFile(*poc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lecture du POC:", err)
		os.Exit(1)
	}
	// FindSubmatchIndex plutot que FindSubmatch : on RECONSTRUIT la page par tranches, sans
	// jamais passer la charge utile a ReplaceAll — un identifiant d'arme en hexadecimal
	// contient des `$`, que le remplacement interpreterait comme une reference de groupe et
	// qui tronquerait la page en silence.
	idx := dataBlock.FindSubmatchIndex(page)
	if idx == nil {
		fmt.Fprintln(os.Stderr, "bloc de donnees introuvable dans le POC")
		os.Exit(1)
	}
	var data map[string]any
	if err := json.Unmarshal(page[idx[4]:idx[5]], &data); err != nil {
		fmt.Fprintln(os.Stderr, "bloc de donnees illisible:", err)
		os.Exit(1)
	}
	var doc replay.ReplayDocument
	raw, err := os.ReadFile(*art)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lecture de l'artefact:", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Fprintln(os.Stderr, "artefact illisible:", err)
		os.Exit(1)
	}
	before := counts(data)
	applyShots(data, doc)
	applyGrenades(data, doc)
	applyCoverage(data, doc)
	after := counts(data)

	enc, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "encodage:", err)
		os.Exit(1)
	}
	// Reconstruction par tranches : tout ce qui precede le bloc, le bloc neuf, tout ce qui
	// suit. Aucune autre partie de la page n'est touchee.
	rebuilt := make([]byte, 0, len(page)+len(enc))
	rebuilt = append(rebuilt, page[:idx[4]]...)
	rebuilt = append(rebuilt, enc...)
	rebuilt = append(rebuilt, page[idx[5]:]...)
	if err := os.WriteFile(*out, rebuilt, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ecriture:", err)
		os.Exit(1)
	}
	fmt.Printf("POC mis a jour : %s\n", *out)
	fmt.Printf("  tirs      %d -> %d\n", before.shots, after.shots)
	fmt.Printf("  lancers   %d -> %d\n", before.grenades, after.grenades)
	if doc.Coverage != nil {
		fmt.Printf("  couverture des tirs : %d / %d — verdict %q\n",
			doc.Coverage.Shots.Attached, doc.Coverage.Shots.Available, doc.Coverage.Verdict["shots"])
	}
}

type tally struct{ shots, grenades int }

func counts(d map[string]any) tally {
	t := tally{}
	if v, ok := d["shots"].([]any); ok {
		t.shots = len(v)
	}
	if v, ok := d["grenades"].([]any); ok {
		t.grenades = len(v)
	}
	return t
}

// applyShots remplace les tirs par ceux de l'artefact, au FORMAT POSITIONNEL de la page :
// [image, slot, x, y, cap|null, identifiant hexadecimal, libelle].
//
// LE LIBELLE N'EST PAS INVENTE : il vient du catalogue de familles, et un identifiant hors
// catalogue garde son hexadecimal plutot qu'un nom approchant — la regle du chantier voisin,
// « on ne stocke jamais une resolution qui peut s'ameliorer », vaut aussi pour l'affichage.
func applyShots(d map[string]any, doc replay.ReplayDocument) {
	labels := labelsFromPage(d)
	out := make([]any, 0, len(doc.Shots))
	for _, s := range doc.Shots {
		var head any
		if s.H != 0 {
			head = s.H
		}
		out = append(out, []any{s.T, s.Slot, s.X, s.Y, head, s.Weapon, labelOf(s.Weapon, labels)})
	}
	d["shots"] = out
}

// labelsFromPage recupere les libelles deja presents dans la page : ils ont ete valides a
// l'ecran, et les reprendre evite qu'un tir change de nom d'une version a l'autre.
func labelsFromPage(d map[string]any) map[string]string {
	out := map[string]string{}
	v, ok := d["shots"].([]any)
	if !ok {
		return out
	}
	for _, row := range v {
		r, ok := row.([]any)
		if !ok || len(r) < 7 {
			continue
		}
		id, ok1 := r[5].(string)
		lb, ok2 := r[6].(string)
		if ok1 && ok2 && lb != "" {
			out[id] = lb
		}
	}
	return out
}

// labelOf nomme une arme : d'abord le libelle deja affiche, sinon le catalogue de familles,
// sinon rien — jamais une approximation.
func labelOf(id string, known map[string]string) string {
	if lb, ok := known[id]; ok {
		return lb
	}
	if len(id) == 18 {
		var v uint64
		if _, err := fmt.Sscanf(id[2:], "%X", &v); err == nil {
			if n, ok := weaponv3.KnownWeaponHigh32[uint32(v>>32)]; ok {
				return n
			}
		}
	}
	return ""
}

// applyGrenades remplace les lancers, au format positionnel de la page.
func applyGrenades(d map[string]any, doc replay.ReplayDocument) {
	prev, _ := d["grenades"].([]any)
	if len(doc.Grenades) == 0 {
		return
	}
	width := 5
	if len(prev) > 0 {
		if r, ok := prev[0].([]any); ok {
			width = len(r)
		}
	}
	out := make([]any, 0, len(doc.Grenades))
	for _, g := range doc.Grenades {
		row := []any{g.T, g.Slot, g.X, g.Y, g.Kind}
		for len(row) < width {
			row = append(row, nil)
		}
		out = append(out, row[:width])
	}
	d["grenades"] = out
}

// applyCoverage ecrit la couverture dans le bloc de donnees et met a jour les compteurs du
// bandeau. C'est l'objet meme de l'etape 4 : publier RATTACHES SUR DISPONIBLES, la ou le
// resultat se lit.
func applyCoverage(d map[string]any, doc replay.ReplayDocument) {
	if doc.Coverage == nil {
		return
	}
	enc, err := json.Marshal(doc.Coverage)
	if err != nil {
		return
	}
	var cov map[string]any
	if err := json.Unmarshal(enc, &cov); err != nil {
		return
	}
	cov["_note"] = "rattaches / disponibles par calque, avec la cause de chaque rejet. " +
		"Le denominateur est le nombre de records QUE LE FILM PORTE, pas le nombre de tirs du match."
	d["coverage"] = cov

	st, _ := d["stats"].(map[string]any)
	if st == nil {
		st = map[string]any{}
	}
	st["shots"] = doc.Coverage.Shots.Attached
	st["shotsAvailable"] = doc.Coverage.Shots.Available
	st["grenadesAvailable"] = doc.Coverage.Grenades.Available
	d["stats"] = st
}
