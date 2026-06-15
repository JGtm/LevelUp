//go:build cgo

package main

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
)

// renderDiagnosticText rend le rapport en table texte déterministe (sans padding
// sur des mots accentués, pour un golden stable). Read-only.
func renderDiagnosticText(rep *domain.TitleDiagnostic) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Diagnostic du titre : %s\n\n", rep.TitleSlug)

	b.WriteString("Mappings TOML :\n")
	for _, cf := range rep.ConfigFiles {
		state := "manquant"
		if cf.Present {
			state = "present"
		}
		req := "optionnel"
		if cf.Required {
			req = "requis"
		}
		fmt.Fprintf(&b, "  %s : %s (%s)\n", cf.Name, state, req)
	}

	b.WriteString("\nBases de donnees :\n")
	for _, db := range rep.Databases {
		state := "absente"
		if db.Exists {
			state = "presente"
		}
		fmt.Fprintf(&b, "  %s : %s\n", db.Name, state)
		if db.Error != "" {
			fmt.Fprintf(&b, "    erreur : %s\n", db.Error)
		}
		for _, tb := range db.Tables {
			ts := "absente"
			if tb.Exists {
				ts = fmt.Sprintf("%d lignes", tb.Rows)
			}
			fmt.Fprintf(&b, "    %s : %s\n", tb.Name, ts)
		}
	}
	return b.String()
}
