package tactical

// reapparitions_comptes.go — LA JOINTURE « COMPTE A REBOURS -> JOUEUR ».
//
// Le compte a rebours de reapparition est publie par le film avec l'INDEX FILM du joueur, pas
// avec son xuid ; la correspondance se lit dans l'identite des joueurs du document. C'est la
// DEUXIEME copie de cette jointure (la premiere est la preuve de fin de vie du rejeu web) : une
// troisieme impose de la centraliser. Elle se retire quand le decodeur publiera le xuid sur le
// compte a rebours.

// CompteBrut est un compte a rebours tel que le document le publie : l'index film du joueur
// qui attend, et la frame ou le compte COMMENCE (la frame de sa mort).
type CompteBrut struct {
	IndexFilm int
	Debut     int
}

// JoindreDebutsDeCompte rattache chaque compte a rebours au xuid de son joueur par
// `xuidParIndexFilm` (index film -> xuid). Un compte dont le joueur n'a pas de xuid (index
// inconnu, bot) est ecarte et COMPTE dans le second retour : il ne peut prouver la mort de
// personne.
func JoindreDebutsDeCompte(comptes []CompteBrut, xuidParIndexFilm map[int]string) ([]DebutCompte, int) {
	out := make([]DebutCompte, 0, len(comptes))
	sansXUID := 0
	for _, c := range comptes {
		xuid := xuidParIndexFilm[c.IndexFilm]
		if xuid == "" {
			sansXUID++
			continue
		}
		out = append(out, DebutCompte{XUID: xuid, Frame: c.Debut})
	}
	return out, sansXUID
}
