package main

// recensement.go — la SORTIE du recensement : un tableau Markdown, trie, destine au
// document de mesure du chantier. Sortie vide = stdout.

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EcrisRecensement rend le tableau du corpus par carte.
func EcrisRecensement(c *Corpus, sortie string) error {
	w := io.Writer(os.Stdout)
	if sortie != "" {
		if err := os.MkdirAll(filepath.Dir(sortie), 0o755); err != nil {
			return fmt.Errorf("creation du dossier de sortie : %w", err)
		}
		f, err := os.Create(sortie)
		if err != nil {
			return fmt.Errorf("creation du fichier de recensement : %w", err)
		}
		defer func() {
			if errFerme := f.Close(); errFerme != nil {
				slog.Error("mappower: fermeture du recensement", "err", errFerme, "sortie", sortie)
			}
		}()
		w = f
	}
	return ecrisTableau(w, c)
}

// ecrisTableau formate le corpus.
func ecrisTableau(w io.Writer, c *Corpus) error {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- mappower-build --recensement, titre %s, %s -->\n\n",
		c.TitleSlug, time.Now().UTC().Format("2006-01-02"))
	fmt.Fprintf(&b, "Matchs du registre retenus : %d ; ecartes faute de nom de carte : %d ;"+
		" artefacts de rejeu sans match au registre : %d.\n\n",
		len(c.Matchs), c.SansNomDeCarte, c.SansCorrespondance)
	fmt.Fprintln(&b, "| # | Carte | Axe | Matchs avec positions | Kills | Matchs joues | Artefacts de rejeu | Categories |")
	fmt.Fprintln(&b, "|---|---|---|---|---|---|---|---|")
	for i, e := range c.Cartes {
		if e.MatchsKills == 0 && e.Artefacts == 0 {
			continue
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %d | %d | %d | %d | %s |\n",
			i+1, e.Carte, e.Axe, e.MatchsKills, e.Kills, e.MatchsTotal, e.Artefacts,
			strings.Join(e.Categories, ", "))
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("ecriture du recensement : %w", err)
	}
	return nil
}
