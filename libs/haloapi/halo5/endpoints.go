package halo5

// Endpoints : les deux hôtes des services Halo 5. Un champ vide retombe sur l'adresse par défaut
// ([EndpointsParDefaut]).
type Endpoints struct {
	// Stats : hôte de l'historique, des service records, du carnage et des événements.
	Stats string
	// Profils : hôte de l'apparence et des rendus PNG (Spartan, emblème).
	Profils string
}

// EndpointsParDefaut rend les hôtes officiels. L'UNIQUE endroit de la brique qui nomme un hôte Halo 5
// (une fonction, pas une variable : aucune valeur de paquet mutable). Les mêmes chaînes figurent
// dans `config/titles/halo_5/constants.toml` (`[endpoints]`), que la brique ne lit pas.
func EndpointsParDefaut() Endpoints {
	return Endpoints{
		Stats:   "https://spartanstats.svc.halowaypoint.com",
		Profils: "https://haloplayer.svc.halowaypoint.com",
	}
}

// hoteStats rend l'hôte des stats, l'adresse par défaut s'il est vide.
func (e Endpoints) hoteStats() string {
	if e.Stats != "" {
		return e.Stats
	}
	return EndpointsParDefaut().Stats
}

// hoteProfils rend l'hôte des profils, l'adresse par défaut s'il est vide.
func (e Endpoints) hoteProfils() string {
	if e.Profils != "" {
		return e.Profils
	}
	return EndpointsParDefaut().Profils
}
