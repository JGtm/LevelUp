// Package halo_infinite — killsource_registry.go : la SOURCE DE DEGAT d'une mort ->
// l'entree du registre d'armes qui la represente.
//
// CE QUE CE FICHIER EST, ET CE QU'IL N'EST PAS. Il repond a « quelle entree de registre
// cette source designe », pas a « quelle image l'illustre ». La deuxieme question a deja
// sa reponse (`AssetURLAdapter.KillSourceIcon`, adossee a `film/killicon`) et elle ne
// couvre pas la premiere : une source peut meriter une ligne de statistiques SANS meriter
// une vignette. C'est exactement le cas de la chute et de l'environnement — voir plus bas.
//
// DEUX VOIES, DANS CET ORDRE.
//
//  1. LA TABLE killicon. Elle porte deja une colonne `weapon_key` et resout le repulseur
//     (`NOM Repulsor` -> `hinf_repulsor`) et les quatre bobines (`BANQUE
//     exp_single_small_*` -> `hinf_coil_*`). Rien a dupliquer : on lui demande.
//
//  2. LA CLASSE damagetag, pour le seul cas que killicon ne peut pas porter. Les 9 tags
//     `DEGAT_GLOBAL` (chute, environnement, hors-limites) sont indiscernables entre eux,
//     et `killicon.validate()` refuse a la lecture toute regle sans vignette — a juste
//     titre : l'atlas propose `killfeed-52 Fall` ET `killfeed-55 environment`, et en
//     choisir une pour les neuf poserait une icone fausse sur l'autre moitie des cas.
//     Une icone absente est un repli, une icone fausse est un mensonge. La CLASSE, elle,
//     est certaine — elle suffit a compter.
//
// TITLE-AGNOSTIC : ce type est Halo Infinite, il est injecte au cablage derriere
// `port.KillSourceClassifier`. Un titre sans decodeur de film n'en fournit aucun, le repo
// ne remonte alors aucune ligne, et le sunburst est byte-identique a ce qu'il etait.
package halo_infinite

import (
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/killicon"
)

// keyEnvironment est l'entree de registre qui porte la chute et l'environnement. Une
// SEULE cle pour les 9 tags `DEGAT_GLOBAL` : le film ne les distingue pas, donc on ne
// pretend pas le faire.
const keyEnvironment = "hinf_environment"

// KillSourceRegistry traduit une source de degat en cle de registre.
//
// Sans etat : les deux tables consultees sont embarquees et immuables. Le type existe
// pour porter l'interface, pas pour porter des donnees.
type KillSourceRegistry struct{}

// NewKillSourceRegistry cree le classificateur des sources de degat Halo Infinite.
func NewKillSourceRegistry() KillSourceRegistry { return KillSourceRegistry{} }

// KillSourceRegistryKey rend la weapon_key du registre pour une source de degat.
//
// Second retour faux = cette source ne designe aucune entree de registre (arme a feu
// ordinaire deja servie par l'attribution `0xd2`, melee et grenade servies par les
// compteurs API, source inconnue). Le kill reste alors dans « Non attribue ».
func (KillSourceRegistry) KillSourceRegistryKey(sourceTag uint32) (string, bool) {
	// Voie 1 : la table des vignettes porte deja la cle de registre.
	if ic, ok := killicon.Lookup(sourceTag); ok && ic.WeaponKey != "" {
		return ic.WeaponKey, true
	}
	// Voie 2 : la classe suffit pour la chute et l'environnement.
	if l, ok := damagetag.Lookup(sourceTag); ok && l.Class == damagetag.ClassGlobal {
		return keyEnvironment, true
	}
	return "", false
}

// KillSourceRegistryKey sur l'adapter d'assets : MEME table, autre question.
//
// POURQUOI ICI. L'adapter d'assets porte deja `KillSourceIcon`, qui interroge les memes
// deux tables embarquees pour repondre « quelle image ». Cette methode-ci repond « quelle
// entree de registre ». Les poser cote a cote evite d'ouvrir un troisieme type d'adapter
// dans le resolver de titres pour une seule fonction, et le cablage n'a alors RIEN a
// savoir du titre : il demande l'adapter d'assets qu'il a deja, et regarde s'il satisfait
// `port.KillSourceClassifier`. Un titre qui ne l'implemente pas n'a pas de classificateur,
// et le repo rend zero ligne — l'etat nominal d'un titre sans decodeur de film.
func (a *AssetURLAdapter) KillSourceRegistryKey(sourceTag uint32) (string, bool) {
	return KillSourceRegistry{}.KillSourceRegistryKey(sourceTag)
}

// KillSourceClassName nomme la CLASSE d'une source de degat — pour la JOURNALISATION
// seule (port.KillSourceDescriber).
//
// POURQUOI CETTE METHODE EXISTE. Un kill que le rejeu 2D sait nommer mais que le graphe
// classe « Non attribue » ne doit pas disparaitre en silence (decision D13 du plan du
// 2026-09-01). Le lecteur qui l'ecarte a besoin de citer la classe de la source pour que la
// ligne de journal soit exploitable — « 159 morts de classe VEHICULE ecartees » se traite,
// « 159 morts ecartees » ne se traite pas. Aucune decision de comportement n'en depend :
// c'est du texte pour un humain.
func (KillSourceRegistry) KillSourceClassName(sourceTag uint32) (string, bool) {
	l, ok := damagetag.Lookup(sourceTag)
	if !ok {
		return "", false
	}
	return string(l.Class), true
}

// KillSourceClassName sur l'adapter d'assets : MEME table, meme reponse. Le cablage ne
// connait que l'adapter d'assets ; c'est par lui qu'il decouvre les deux interfaces
// optionnelles de la source de degat.
func (a *AssetURLAdapter) KillSourceClassName(sourceTag uint32) (string, bool) {
	return KillSourceRegistry{}.KillSourceClassName(sourceTag)
}
