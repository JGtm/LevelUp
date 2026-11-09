// Package syslocale lit le fuseau horaire et la langue d'interface du système d'exploitation.
//
// Seule la lecture est ici (adaptateur système) : la décision de s'en servir, la validation du
// résultat et le repli appartiennent à l'appelant (internal/config). Le paquet n'importe aucun
// autre paquet de l'application.
package syslocale

// Raw porte les valeurs brutes lues du système, avant toute interprétation. Un champ vide veut
// dire « non lu » : le système ne le fournit pas, ou la lecture a échoué (l'appelant journalise
// l'erreur par ReadErr).
type Raw struct {
	// WindowsZone : clé de fuseau de Windows (« Romance Standard Time »). Vide hors Windows.
	WindowsZone string
	// IANAZone : nom de fuseau IANA déjà fourni par le système (variable TZ). Vide sous Windows.
	IANAZone string
	// UILanguages : étiquettes de langue d'interface par ordre de préférence (« fr-FR », « en_US.UTF-8 »).
	UILanguages []string
	// ReadErr : première erreur de lecture rencontrée (registre, API système), nil si tout a répondu.
	ReadErr error
}

// Timezone rend le nom de fuseau IANA que le système désigne, et false quand il n'en désigne
// aucun que ce paquet sache traduire. Un nom IANA fourni tel quel gagne ; sinon la clé Windows
// est traduite par la table windowsZones. Pur.
func (r Raw) Timezone() (string, bool) {
	if r.IANAZone != "" {
		return r.IANAZone, true
	}
	return IANAFromWindows(r.WindowsZone)
}

// IANAFromWindows traduit une clé de fuseau Windows en nom IANA (territoire « 001 » de la table
// windowsZones de CLDR). Rend false pour une clé absente ou inconnue. Pur.
func IANAFromWindows(key string) (string, bool) {
	iana, ok := windowsZones[key]
	return iana, ok
}
