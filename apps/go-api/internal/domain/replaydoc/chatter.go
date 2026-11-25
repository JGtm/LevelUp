package replaydoc

// chatter.go — LES REPLIQUES DES SPARTANS, jumeau servi de `film/replay/document_repliques.go` : une réplique
// prononcée par un joueur, nommée par le décodeur.
//
// Le client en tire des bulles sur la carte, la voix, et des lignes du fil. Il ne lit que quatre champs : l'instant, le
// joueur qui parle, l'identifiant du son (clé des textes et des voix versés, `static/sounds/halo_infinite/repliques/`)
// et le nom court, qui dit la famille de la réplique. La priorité, l'indice de réplique et les compteurs de couverture du
// décodeur restent dans le décodeur (`testutil.ReplayChampsNonServis`).

// Chatter est UNE réplique de Spartan, sur l'axe de frames du document.
//
// Player est le xuid du joueur qui parle, en décimal comme `Track.XUID` ; il est absent quand l'unité qui parle n'a pas
// de piste d'humain à cette frame (un bot, ou un locuteur sans piste). Sound est l'identifiant du son, en hexadécimal sur
// huit chiffres. Name est le nom court de la réplique, absent quand le build du film n'a pas de disposition mesurée : le
// client n'affiche que les répliques nommées.
type Chatter struct {
	T      int    `json:"t"`
	Player string `json:"player,omitempty"`
	Sound  string `json:"sound,omitempty"`
	Name   string `json:"name,omitempty"`
}
