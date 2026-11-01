package xboxlive

// Endpoints : l'hôte des succès Xbox Live. Un champ vide retombe sur l'adresse par défaut
// ([EndpointsParDefaut]).
type Endpoints struct {
	// Succes : hôte des succès (achievements v2).
	Succes string
}

// EndpointsParDefaut rend l'hôte officiel. L'UNIQUE endroit de la brique qui nomme un hôte Xbox
// Live (une fonction, pas une variable : aucune valeur de paquet mutable).
func EndpointsParDefaut() Endpoints {
	return Endpoints{Succes: "https://achievements.xboxlive.com"}
}

// hoteSucces rend l'hôte des succès, l'adresse par défaut s'il est vide.
func (e Endpoints) hoteSucces() string {
	if e.Succes != "" {
		return e.Succes
	}
	return EndpointsParDefaut().Succes
}
