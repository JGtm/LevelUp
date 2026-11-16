package replaydoc

// damage.go — LES DEGATS EN POINTS, DANS LE TEMPS ET PAR ARME : UNE ESTIMATION. Jumeau de `replay.Damage`.

// Damage est UNE ligne de degat : la frame, l'auteur et la victime (xuid decimal ; absents pour le decor
// ou un bot cote auteur, pour un bot cote victime), l'effet de degat, l'arme du dernier tir de l'auteur
// dans les deux secondes, les points retires au bouclier et au corps (au dixieme pres) et `measured`
// (les points viennent de la vie lue autour du coup, pas de la table des effets). `damage[]` ne porte que
// les messages dont l'instant tient dans l'axe ; les totaux de `shotStats[]` comptent tout le film.
type Damage struct {
	T        int     `json:"t"`
	Attacker string  `json:"attacker,omitempty"`
	Victim   string  `json:"victim,omitempty"`
	Effect   string  `json:"effect"`
	Weapon   string  `json:"w,omitempty"`
	Shield   float64 `json:"shield"`
	Body     float64 `json:"body"`
	Measured bool    `json:"measured,omitempty"`
}
