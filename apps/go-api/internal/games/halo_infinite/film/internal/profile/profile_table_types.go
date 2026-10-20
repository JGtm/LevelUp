package profile

// profile_table_types.go — LES TYPES DE LA TABLE DU PROFIL (`profile_table.go`) : la provenance et
// la ligne. Les CONSTANTES de provenance restent a cote de la table, ou le lecteur du catalogue
// commis les evalue.

// Provenance dit d ou vient une valeur de profil. Cf. l en-tete de `profile_table.go`.
type Provenance string

// LigneProfil est UNE ligne de la table : une valeur, sa cle, sa provenance et sa preuve.
type LigneProfil struct {
	// Cle est la cle ECRITE qui selectionne cette ligne : `format=27`, `build=HI_1_13_0`,
	// `majeure>=41`, ou `toutes` quand la valeur ne depend d aucune cle connue.
	Cle string
	// Champ nomme ce que la ligne pose, dans le vocabulaire du [Profile].
	Champ string
	// Valeur est la valeur posee, ecrite pour etre lue par un humain.
	Valeur string
	// Source est la provenance (cf. l en-tete).
	Source Provenance
	// Preuve est la fonction Ghidra (provenance relue), ou le film temoin et son oracle
	// (provenance mesuree), ou ce qui manque (provenance presumee).
	Preuve string
	// Date est le jour ou cette provenance a ete etablie, `AAAA-MM-JJ`.
	Date string
}
