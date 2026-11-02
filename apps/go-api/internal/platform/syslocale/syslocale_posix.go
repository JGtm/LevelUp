//go:build !windows

package syslocale

import (
	"os"
	"strings"
	"time"
)

// ReadRaw lit TZ (ou, à défaut, le nom du fuseau local de Go quand il en a un) et les variables
// de langue POSIX dans l'ordre de précédence de la norme : LC_ALL, LC_MESSAGES, LANG, puis
// LANGUAGE (liste séparée par des deux-points).
func ReadRaw() Raw {
	var r Raw
	r.IANAZone = strings.TrimPrefix(strings.TrimSpace(os.Getenv("TZ")), ":")
	if r.IANAZone == "" {
		if name := time.Local.String(); name != "Local" && name != "UTC" {
			r.IANAZone = name
		}
	}
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := strings.TrimSpace(os.Getenv(v)); val != "" {
			r.UILanguages = append(r.UILanguages, val)
		}
	}
	if val := os.Getenv("LANGUAGE"); val != "" {
		r.UILanguages = append(r.UILanguages, strings.Split(val, ":")...)
	}
	return r
}
