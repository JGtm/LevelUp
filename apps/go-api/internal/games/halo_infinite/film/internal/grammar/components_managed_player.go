package grammar

// components_managed_player.go — LES COMPOSANTS DE L ARCHETYPE `managed-player` (ti=9), LOT 3.6.a.
//
// # CE QUE CE FICHIER FERME
//
// `ti=9` portait 8 composants sur 10, et sa fermeture d image-cle valait 0/1 717 sur les sept
// bobines du ratchet 0.A.3 — UN SEUL composant en cause, `i4`, nomme par le golden comme le
// bloquant des sept bobines. Dans une image-cle il n y a pas de masque de presence : tous les
// composants de l archetype sont la, dans l ordre du registre, donc un composant sans lecteur
// bloque tout ce qui le suit. `i4` et `i9` sont les deux derniers, et ils sont ici.
//
// # LES GRAMMAIRES SONT RELEVEES CHEZ L ECRIVAIN, PAS MESUREES
//
// Un composant porte est une fonction pure (profil, bits) : ce fichier ne lit aucune table du
// jeu, aucun etat de processus, et n a AUCUNE entree de profil — les grammaires portees ici sont
// des largeurs LITTERALES d instruction. Sources, sous Ghidra en lecture seule :
//
//	i4  `.ai/V7.5/film_re/NOTE_3_6_TI9_2026-09-16.md` §2 — ecrivain `FUN_142ed5bc8`
//	    (descripteur `0x143d08840`, ecrivain a `descripteur + 0x40` = `0x143d08880`).

// Etiquettes de registre des composants de ti=9 portes ici.
const (
	compManagedPlayerForgeWeather = "managed-player-forge-weather-effect-overrides-component"
)

// consumeManagedPlayerForgeWeatherOverrides porte ti=9 `i4
// managed-player-forge-weather-effect-overrides-component` — ecrivain `FUN_142ed5bc8`.
//
// GRAMMAIRE : `R(32)` ; `R(32)`. Deux champs de 32 bits dans cet ordre, 64 bits
// INCONDITIONNELS : aucune porte, aucun compteur, aucune boucle, aucun appel virtuel. Relevee
// sur les deux `ADD dword ptr [<reg> + 0x2c], 0x20` du desassemblage (`142ed5c70` et
// `142ed5d20`, plus leurs jumeaux de chemin rapide `142ed5bf9` et `142ed5cb6`) et sur les deux
// seules ecritures de sortie (`142ed5ca1 MOV dword ptr [RDI + 0x3c]`, `142ed5d5d MOV dword ptr
// [RDI + 0x40]`).
//
// AUCUNE DEPENDANCE DE CONFIGURATION : rien de la largeur ne depend d une table du jeu, d une
// carte ni d un build. C est le meme patron que `i5
// managed-player-active-mission-name-component` (`FUN_142ed5ab0`), deja porte.
func consumeManagedPlayerForgeWeatherOverrides(br *Lecteur) {
	br.ReadBits(32)
	br.ReadBits(32)
}
