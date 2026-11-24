package domain

// AdminReplaySchema — le décompte des rejeux rangés par rapport au schéma courant, par titre (monitoring admin).
// Un rejeu rangé porte la version de schéma qui l'a produit ; une mise à jour du décodeur laisse les rejeux déjà
// rangés « à l'ancien schéma » : ils restent lisibles mais ne portent pas ce que la nouvelle version sait y mettre.
type AdminReplaySchema struct {
	GeneratedAt string `json:"generated_at"`
	// CurrentSchema est le schéma que produit le serveur qui répond.
	CurrentSchema int                      `json:"current_schema"`
	Titles        []AdminReplaySchemaTitle `json:"titles"`
}

// AdminReplaySchemaTitle — le décompte d'un titre. Total = Current + Behind + Unreadable.
type AdminReplaySchemaTitle struct {
	TitleSlug string `json:"title_slug"`
	// Total : les rejeux rangés du titre.
	Total int `json:"total"`
	// Current : rejeux dont le schéma est le schéma courant.
	Current int `json:"current"`
	// Behind : rejeux d'un schéma antérieur au schéma courant.
	Behind int `json:"behind"`
	// Unreadable : rejeux dont l'en-tête ne se lit pas.
	Unreadable int `json:"unreadable"`
}
