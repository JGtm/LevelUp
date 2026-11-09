//go:build windows

package syslocale

import (
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// timeZoneInformationKey : clé du registre qui porte le fuseau actif du poste.
const timeZoneInformationKey = `SYSTEM\CurrentControlSet\Control\TimeZoneInformation`

// ReadRaw lit le fuseau actif (valeur TimeZoneKeyName du registre, lecture seule) et les langues
// d'interface de l'utilisateur courant (GetUserPreferredUILanguages, étiquettes BCP-47).
func ReadRaw() Raw {
	var r Raw
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, timeZoneInformationKey, registry.QUERY_VALUE)
	if err != nil {
		r.ReadErr = fmt.Errorf("registre %s: %w", timeZoneInformationKey, err)
	} else {
		defer k.Close()
		if zone, _, err := k.GetStringValue("TimeZoneKeyName"); err != nil {
			r.ReadErr = fmt.Errorf("registre TimeZoneKeyName: %w", err)
		} else {
			r.WindowsZone = zone
		}
	}
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil {
		if r.ReadErr == nil {
			r.ReadErr = fmt.Errorf("GetUserPreferredUILanguages: %w", err)
		}
		return r
	}
	r.UILanguages = langs
	return r
}
