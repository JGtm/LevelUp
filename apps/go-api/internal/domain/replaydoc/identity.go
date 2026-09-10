package replaydoc

// identity.go — LE REGISTRE D'IDENTITE, TEL QUE LE CLIENT LE RECOIT (schema 50, lot P2).
//
// Il dit SUR QUOI repose chaque nom que le document sert. Jusqu'ici l'artefact publiait
// `tracks[].xuid` sans jamais dire si ce nom venait d'une lecture du film, d'un pont par morts,
// d'une fermeture ou d'une deduction par elimination — quatre forces de preuve que rien ne
// distinguait. Le client peut desormais les afficher, et le gate de non-regression peut refuser
// qu'un lien `direct` redevienne `deduit`.
//
// CE QUI N'EST PAS RESOLU EST DANS LA LISTE, avec `source = "non_resolu"` et un `xuid` vide :
// une identite manquante est un fait a montrer, pas une ligne a supprimer.

// IdentitySection est le registre d'identite d'un film.
type IdentitySection struct {
	Players       []IdentityPlayer       `json:"players,omitempty"`
	BipedSlots    []IdentityBipedSlot    `json:"bipedSlots,omitempty"`
	StatborgSlots []IdentityStatborgSlot `json:"statborgSlots,omitempty"`
	Coverage      IdentityCoverage       `json:"coverage"`
}

// IdentityPlayer est une identite du film : son index, son xuid (humain) ou son `bid` (bot).
type IdentityPlayer struct {
	// FilmIndex vaut -1 quand le film ne place ce joueur nulle part (connu de la seule feuille).
	FilmIndex int `json:"filmIndex"`
	// XUID en decimal — vide pour un bot.
	XUID string `json:"xuid,omitempty"`
	// Bid est l'identifiant stable d'un bot, forme `bid(N.0)`.
	Bid string `json:"bid,omitempty"`
	// Name est le nom tel que le film l'ecrit.
	Name string `json:"name,omitempty"`
	Link Link   `json:"link"`
}

// IdentityBipedSlot est l'occupation d'un slot de bipede sur un intervalle de frames.
type IdentityBipedSlot struct {
	Slot uint32 `json:"slot"`
	// XUID de l'occupant. Vide quand rien ne l'a nomme (`link.source = "non_resolu"`).
	XUID string `json:"xuid,omitempty"`
	// Bid est l'identifiant STABLE du BOT qui occupe le corps, forme `bid(N.0)` (schema 51,
	// lot 4.3). EXCLUSIF avec `XUID` : un bot n'a pas de xuid, et lui en fabriquer un le rendrait
	// joignable avec un humain. Vide pour un humain comme pour un corps que rien ne nomme.
	Bid  string `json:"bid,omitempty"`
	Link Link   `json:"link"`
}

// IdentityStatborgSlot est l'identite d'un slot d'entite statborg POUR UNE MANCHE.
type IdentityStatborgSlot struct {
	Slot  int `json:"slot"`
	Round int `json:"round"`
	// XUID du joueur. Vide quand le pont par manche n'a pas su nommer le slot.
	XUID string `json:"xuid,omitempty"`
	Link Link   `json:"link"`
}

// Link dit d'ou vient un lien du registre, par quelle voie, sur quelle preuve, et entre quelles
// bornes il vaut.
type Link struct {
	// Source : `direct` (lu dans le film), `catalogue`, `externe` (base / feuille de match),
	// `deduit` (pont, fermeture, elimination, geometrie), `non_resolu` (rien ne l'a nomme).
	Source string `json:"source"`
	// Method nomme la voie exacte a l'interieur de la source (`PlayerIndexTable`, `bid`,
	// `pont_par_morts`, `fermeture`, `elimination_roster`, `instants_de_mort`, ...).
	Method string `json:"method,omitempty"`
	// Readings porte le denominateur de la voie quand elle en a un (lectures concordantes,
	// morts appariees, votes).
	Readings int `json:"readings,omitempty"`
	// Metric porte la grandeur continue de la voie quand elle en a une (distance, ecart).
	Metric *float64 `json:"metric,omitempty"`
	// From / To bornent la validite du lien sur l'axe de frames, bornes INCLUSIVES. Un slot
	// recycle porte PLUSIEURS liens bornes, jamais un lien aplati.
	From int `json:"from"`
	To   int `json:"to"`
}

// IdentityCoverage compte les liens par type d'entite et par provenance.
type IdentityCoverage struct {
	// FilmIndex : les liens « index de joueur du film <-> identite ».
	FilmIndex LinkCounts `json:"filmIndex"`
	// BipedSlot porte deux ventilations de plus que ses voisines : la part de `direct` obtenue
	// en PROPAGEANT le record de creation d'un corps a ses autres sejours, et la CAUSE de chaque
	// `non_resolu`. Elles n'ont de sens que pour cette famille (lot E2, 2026-09-08).
	BipedSlot    BipedLinkCounts `json:"bipedSlot"`
	StatborgSlot LinkCounts      `json:"statborgSlot"`
}

// LinkCounts est le decompte d'une famille de liens par provenance.
type LinkCounts struct {
	Direct     int `json:"direct"`
	Catalog    int `json:"catalogue"`
	External   int `json:"externe"`
	Inferred   int `json:"deduit"`
	Unresolved int `json:"non_resolu"`
}

// BipedLinkCounts est [LinkCounts] pour la famille du slot de bipede, avec ses deux ventilations
// propres. `LinkCounts` est EMBARQUE : `direct` / `deduit` / `non_resolu` restent au meme niveau
// dans le JSON servi que dans celui de ses voisines.
type BipedLinkCounts struct {
	LinkCounts
	// DirectPropagated est la PART de `Direct` obtenue en appliquant le record de creation d'un
	// corps a une autre de ses vies. Sous-compte, jamais un total a part.
	DirectPropagated int `json:"direct_propage"`
	// UnresolvedByCause ventile `Unresolved`. La somme de ses champs EGALE `Unresolved`.
	UnresolvedByCause UnresolvedCauses `json:"non_resolu_par_cause"`
}

// UnresolvedCauses ventile les liens non resolus d'un slot de bipede par cause prouvee.
type UnresolvedCauses struct {
	// IndexOutOfTable : l'index est LU, il n'est pas dans la table publiee.
	IndexOutOfTable int `json:"index_hors_table"`
	// NoCreationRecord : aucun record de creation n'a ete lu sur ce corps.
	NoCreationRecord int `json:"sans_record"`
	// DivergentReadings : deux records du meme slot portent des index differents.
	DivergentReadings int `json:"lectures_divergentes"`
}
