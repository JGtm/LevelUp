package replay

// killpos_bridge.go — LE PONT SLOT -> XUID, EXPOSÉ À UN CONSOMMATEUR HORS DU DOCUMENT DE REJEU.
//
// # POURQUOI CE FICHIER EXISTE
//
// `killpos.go` (BuildKillPositions) a besoin d'un `slotXUID map[uint32]uint64` pour poser une
// mort sur les bonnes trajectoires. Ce pont existe déjà, LU et pas voté, dans `owners.go`
// (buildOwners/OwnerReport.SlotXUID) — mais `buildOwners` est non-exportée, parce que jusqu'ici
// son SEUL appelant était `BuildFromPositions` (build.go), qui assemble un document de rejeu
// complet.
//
// `sync/killcollector` (producteur `shared.kill_positions` pour Halo Infinite, G.2bis) n'a besoin
// QUE de ce pont, pas d'un document de rejeu : construire un `replaybuild.Builder` complet pour
// en extraire une seule table serait disproportionné, et romprait le contrat « replaybuild n'ouvre
// aucune base » pour rien (killcollector, lui, EST déjà dans un contexte base + lease).
//
// # POURQUOI UN EXPORT ET PAS UNE RÉÉCRITURE
//
// « Deux décodeurs du même fait divergeraient » est la règle qui gouverne tout ce paquet
// (cf. l'en-tête de killpos.go, deaths_source.go, identity.go — la même phrase y revient trois
// fois). Elle vaut aussi pour deux CONSOMMATEURS d'un seul pont : réimplémenter
// buildLifeSpans/bestDeathOffset/nameLivesByDeaths/ownersFromLives dans killcollector serait une
// SECONDE lecture du même fil des morts, avec sa propre chance de diverger de celle mesurée et
// éprouvée ici (cf. lives.go — mesures McNemar, plateau de bestDeathOffset, etc.). Ce fichier
// n'ajoute donc AUCUNE logique : il compose deux fonctions déjà existantes et les publie.
//
// # CE QUE CE PONT NE FAIT PAS ICI
//
// PAS DE FERMETURE PAR TIR (`fire = nil`). Les fermetures (closures.go) déduisent un slot restant
// anonyme à partir d'un événement de tir dont l'auteur est connu — killcollector n'a et ne calcule
// aucun flux de tirs pour cet usage (la ventilation par arme, shots.go, résout des INDICES de
// réplication, pas des positions). La version SANS fermeture est PLUS CONSERVATRICE (moins de
// slots nommés, jamais un slot nommé à tort) : exactement ce que la règle de prudence de
// killpos.go demande — une position absente reste absente plutôt qu'approchée.

import (
	"levelup/go-api/internal/analysis/filmdec"
)

// ResolveSlotXUID construit le pont slot -> xuid à partir des SEULES lectures nécessaires,
// pour un appelant qui n'assemble pas de document de rejeu complet.
//
//   - pos : positions bipeds DÉJÀ décodées (filmdec.ScanFilmBipedPositions) — pas redécodées ici.
//   - deaths : le fil des morts du film (ScanFilmDeaths) — nomme chaque vie par sa victime.
//   - idx : l'index de joueur du film (ScanFilmPlayerIndices) — second maillon du pont.
//
// Rend le pont ET le rapport complet (OwnerReport) : un appelant qui écrit des données en base
// doit pouvoir journaliser sur quoi elles reposent, exactement comme build.go le fait pour le
// document de rejeu (cf. son log "slots"/"viesNommees"/"desaccordsIndex").
func ResolveSlotXUID(pos []filmdec.BipedPosition, deaths []Death, idx PlayerIndexTable) (map[uint32]uint64, OwnerReport) {
	rep := buildOwners(indexBySlot(pos), deaths, idx, nil)
	return rep.SlotXUID, rep
}
