package main

// page.go — LA PAGE DE NOMMAGE EST RÉÉCRITE PAR L'EXTRACTION, PAS ENTRETENUE À LA MAIN.
//
// POURQUOI. Les tables `NOMS_JEU` / `NOMS_KILLFEED` de `NOMMAGE_ICONES.html` sont une COPIE
// d'`index.json`. Recopiées à la main, elles ont vieilli en silence et la page affichait moins
// d'étiquettes que ce que l'extraction savait — un défaut invisible, puisque rien ne compare
// les deux. La copie est donc générée : chaque `weapon-icons-build` réécrit les blocs entre
// leurs marqueurs, et une table périmée redevient impossible.
//
// Ce qui est réécrit se limite aux DONNÉES. La prose, la mise en page et le tableau du registre
// restent au fichier — le générateur ne touche jamais qu'entre `<<NOM>>` et `<</NOM>>`.

import (
	"fmt"
	"os"
	"path/filepath"

	"sort"
	"strings"
)

const (
	pageRelPath  = ".ai/V7.5/icones/NOMMAGE_ICONES.html"
	docBaseName  = "ETAT_DE_L_ART_ICONES.md"
	pageLineWrap = 92
)

// writePageTables réécrit les tables de données de la page de nommage ET celles de l'état de
// l'art, qui portent la même copie sous une autre forme.
func writePageTables(path string, entries []iconEntry) error {
	if err := rewrite(path, "// ", []marked{
		{"NB", pageCounts(entries)},
		{"RESOLU", pageMap("RESOLU", entries, "", func(e iconEntry) string { return e.WeaponKey })},
		{"NOMS_JEU", pageMap("NOMS_JEU", entries, "contour", pageDisplayName)},
		{"NOMS_KILLFEED", pageMap("NOMS_KILLFEED", entries, "killfeed", pageDisplayName)},
	}); err != nil {
		return err
	}
	doc := filepath.Join(filepath.Dir(path), docBaseName)
	return rewrite(doc, "<!-- ", []marked{
		{"MD_WEAP", mdWeaponTable(entries)},
		{"MD_KF", mdGrid(entries, "killfeed")},
	})
}

type marked struct{ name, body string }

func rewrite(path, prefix string, blocks []marked) error {
	raw, err := os.ReadFile(path) //nolint:gosec // chemins dérivés de -page
	if err != nil {
		return err
	}
	out := string(raw)
	for _, b := range blocks {
		if out, err = replaceMarked(out, path, prefix, b); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(out), 0o600)
}

// replaceMarked remplace le contenu entre `<<nom>>` et `<</nom>>`. L'absence d'un marqueur est
// une ERREUR : elle signifierait que le fichier a divergé du générateur, donc qu'une table est
// en train de vieillir en silence — exactement ce que ce fichier empêche.
func replaceMarked(doc, path, prefix string, b marked) (string, error) {
	suffix := ""
	if strings.HasPrefix(prefix, "<!--") {
		suffix = " -->"
	}
	open := prefix + "<<" + b.name + ">>" + suffix
	end := prefix + "<</" + b.name + ">>" + suffix
	i := strings.Index(doc, open)
	j := strings.Index(doc, end)
	if i < 0 || j < i {
		return "", fmt.Errorf("marqueur %s absent de %s", b.name, filepath.Base(path))
	}
	return doc[:i+len(open)] + "\n" + b.body + doc[j:], nil
}

// mdWeaponTable : une ligne par index de l'atlas d'armes — clé du registre et nom craqué.
func mdWeaponTable(entries []iconEntry) string {
	var sb strings.Builder
	sb.WriteString("| # | weapon_key (registre) | nom interne craque |\n|---|---|---|\n")
	for _, e := range entries {
		if e.Style != "contour" {
			continue
		}
		fmt.Fprintf(&sb, "| %02d | %s | %s |\n", e.Index, mdCell(e.WeaponKey), mdCell(pageDisplayName(e)))
	}
	return sb.String()
}

// mdGrid : grille compacte à 4 couples (index, nom) par ligne.
func mdGrid(entries []iconEntry, style string) string {
	names := map[int]string{}
	max := -1
	for _, e := range entries {
		if e.Style != style {
			continue
		}
		names[e.Index] = pageDisplayName(e)
		if e.Index > max {
			max = e.Index
		}
	}
	var sb strings.Builder
	sb.WriteString("| # | nom | # | nom | # | nom | # | nom |\n|---|---|---|---|---|---|---|---|\n")
	for i := 0; i <= max; i++ {
		fmt.Fprintf(&sb, "| %02d | %s ", i, mdCell(names[i]))
		if i%4 == 3 {
			sb.WriteString("|\n")
		}
	}
	if max%4 != 3 {
		for i := max % 4; i < 3; i++ {
			sb.WriteString("| | ")
		}
		sb.WriteString("|\n")
	}
	return sb.String()
}

// mdCell : une cellule Markdown. Le « | » qui sépare deux noms concurrents DOIT être échappé,
// sinon il casse les colonnes du tableau (défaut vécu sur la ligne `bandit | bandit_evo`).
func mdCell(s string) string {
	if s == "" {
		return "—"
	}
	return strings.ReplaceAll(s, "|", "\\|")
}

func pageCounts(entries []iconEntry) string {
	n := map[string]int{}
	for _, e := range entries {
		n[e.Style]++
	}
	return fmt.Sprintf("const NB_WEAP = %d, NB_KF = %d;\n", n["contour"], n["killfeed"])
}

// pageDisplayName rend le nom craqué, suffixé « ? » quand sa provenance n'est pas canonique —
// la page s'en sert pour le donner À VÉRIFIER plutôt que comme un fait.
func pageDisplayName(e iconEntry) string {
	if e.GameName != "" && e.NameUnverified {
		return e.GameName + " ?"
	}
	return e.GameName
}

// pageMap sérialise `index: "valeur"` pour un style donné (style vide = tous), en lignes courtes.
func pageMap(name string, entries []iconEntry, style string, val func(iconEntry) string) string {
	type kv struct {
		idx int
		v   string
	}
	var rows []kv
	seen := map[int]bool{}
	for _, e := range entries {
		if (style != "" && e.Style != style) || seen[e.Index] {
			continue
		}
		if v := val(e); v != "" {
			seen[e.Index] = true
			rows = append(rows, kv{e.Index, v})
		}
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].idx < rows[b].idx })

	var sb strings.Builder
	sb.WriteString("const " + name + " = {\n")
	line := " "
	for _, r := range rows {
		tok := fmt.Sprintf(` %d:%q,`, r.idx, r.v)
		if len(line)+len(tok) > pageLineWrap {
			sb.WriteString(line + "\n")
			line = " "
		}
		line += tok
	}
	if strings.TrimSpace(line) != "" {
		sb.WriteString(line + "\n")
	}
	sb.WriteString("};\n")
	return sb.String()
}
