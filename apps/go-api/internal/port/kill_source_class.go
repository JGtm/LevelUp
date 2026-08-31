// Package port — kill_source_class.go : les kills dont la SOURCE DE DEGAT est connue
// mais que l'attribution arme-a-feu ne peut pas voir.
//
// POURQUOI CE PORT EXISTE, ET POURQUOI IL N'EST PAS `WeaponKillRow`. L'attribution
// arme-a-feu (weapon_kills) reconstruit l'arme depuis les records de degat `0xd2` emis
// par le TIREUR. Trois familles de morts n'en emettent aucun : le repulseur, les bobines
// (objets explosifs) et la chute / l'environnement. Elles ne sont donc pas MAL attribuees
// aujourd'hui — elles ne le sont PAS DU TOUT, et tombent dans le residu « Non attribue »
// du sunburst. Mesure du 2026-08-29 sur la base de production (1 365 matchs decodes,
// 74 569 sources de degat mesurees) : 547 kills a la bobine, 403 par chute /
// environnement, 1 au repulseur.
//
// Leur source, elle, est connue : le decodeur de film ecrit `match_kill_events.source_tag`
// (identifiant `jpt!` 32 bits) dans le dead-state de la VICTIME. C'est une SECONDE voie de
// mesure, independante de la premiere, et c'est pour ca qu'elle a son propre port : y
// verser des lignes `weapon_kills` demanderait des `weapon_id` numeriques synthetiques
// pour des objets qui n'en ont pas. Le contrat de cette table-la est « l'arme a feu par
// kill » ; on ne le tord pas.
//
// CE QUI EST COMPTE : le CREDIT du kill-feed (`feed_killer_xuid`), pas la victime. C'est
// la meme convention que partout ailleurs dans l'app — les statistiques d'un joueur sont
// bâties sur ce que le jeu lui credite.
//
// LES PASSES NON PUBLIABLES SONT COMPTEES, ET C'EST VOULU. Une passe de decodage marquee
// `publishable = FALSE` porte des lignes « justes en AGREGAT et fausses individuellement »
// (cf. l'en-tete de migration/steps_shared_kill_events.go) : la fragilite est la
// resolution des NOMS de victimes par bijection, alors que le tueur vient du kill-feed
// AVEC son xuid. Compter par (tueur, classe) est precisement l'usage agrege que ces
// lignes autorisent. Les EXCLURE ferait perdre 40 % de la mesure sans rien gagner en
// justesse.
package port

import (
	"context"
	"errors"
)

// KillSourceClassFilters parametre la lecture agregee des kills par source de degat.
//
// Garde-fou identique a WeaponKillFilters : la requete agrege sur une table partagee
// couvrant tous les matchs et tous les joueurs. Sans MatchIDs ni XUIDs, c'est un scan
// complet — refuse par Validate().
type KillSourceClassFilters struct {
	// MatchIDs restreint aux matchs de ce lot. Obligatoire.
	MatchIDs []string

	// XUIDs restreint aux kills CREDITES a ces joueurs. Obligatoire.
	XUIDs []string
}

// ErrKillSourceClassFiltersTooBroad est retournee par Validate() quand les filtres
// laisseraient passer un scan complet de la table.
var ErrKillSourceClassFiltersTooBroad = errors.New(
	"port: KillSourceClassFilters too broad (provide MatchIDs and XUIDs)")

// Validate refuse les combinaisons qui degenerent en scan complet.
func (f KillSourceClassFilters) Validate() error {
	if len(f.MatchIDs) == 0 || len(f.XUIDs) == 0 {
		return ErrKillSourceClassFiltersTooBroad
	}
	return nil
}

// KillSourceClassRow est une ligne agregee (xuid, weapon_key, total_kills).
//
// L'agregation est faite cote DuckDB. Class et Label viennent du registre d'armes
// (metadata), resolus dans la meme passe : le lecteur n'a rien a recroiser.
type KillSourceClassRow struct {
	// XUID du joueur CREDITE du kill (feed_killer_xuid).
	XUID string `json:"xuid"`
	// WeaponKey : cle canonique du registre resolue depuis la source de degat
	// ("hinf_repulsor", "hinf_coil_plasma", "hinf_environment"). Jamais vide : une
	// source qui ne resout vers aucune cle n'est pas remontee du tout (elle reste dans
	// « Non attribue », cf. decision D6 du plan).
	WeaponKey string `json:"weapon_key"`
	// Class : classe du registre ("equipment", "environmental"). Niveau 1 du sunburst.
	Class string `json:"class"`
	// Label : nom d'affichage FR>EN resolu depuis weapon_name_labels. Niveau 2 du
	// sunburst. Vide si la metadata n'est pas seedee (repli au weapon_key cote appelant).
	Label string `json:"label,omitempty"`
	// LabelEN : meme libelle que Label mais EN-first (repli FR), resolu dans la MEME passe
	// (resolveOffArsenalKeys). Ajoute le 2026-08-29 (V2.1) pour porter le nom EN de l'objet
	// (bobines, repulseur, environnement) jusqu'a FragRoleEntry.LabelEN. Vide si la metadata
	// n'est pas seedee.
	LabelEN string `json:"label_en,omitempty"`
	// Kills : nombre de morts creditees a ce joueur pour cette source.
	Kills int `json:"kills"`
	// NonPublishableKills : sous-ensemble de Kills venant d'une passe de decodage non
	// publiable ligne a ligne. Compte dans Kills (cf. en-tete) — expose SEULEMENT pour
	// que la surface puisse dire au lecteur d'ou vient sa mesure, jamais pour filtrer.
	NonPublishableKills int `json:"non_publishable_kills,omitempty"`
}

// KillSourceClassifier traduit une SOURCE DE DEGAT du film en cle du registre d'armes.
//
// C'est la seule chose que le repo a besoin de savoir, et il ne peut pas la savoir seul :
// la table qui traduit un `jpt!` en objet est propre au titre (Halo Infinite :
// `film/killicon` adossee a `film/damagetag`). L'injecter au cablage garde
// `platform/duckdb` title-agnostic — meme motif que la ModeTaxonomy injectee dans
// MatchViewRepo.
//
// Second retour faux = cette source ne designe aucune entree de registre. Le kill n'est
// alors PAS remonte : il reste dans « Non attribue ». C'est la decision D6 du plan — on
// ne devine pas, on ne proratise pas.
type KillSourceClassifier interface {
	// KillSourceRegistryKey rend la weapon_key du registre pour une source de degat.
	KillSourceRegistryKey(sourceTag uint32) (string, bool)
}

// KillSourceClassRepository expose le loader agrege des kills par source de degat.
//
// Capability gating : retourne games.ErrCapabilityNotSupported si le titre n'a pas la
// capability "film.kill_source" (Halo 5 : autre format de film, pas de decodeur).
type KillSourceClassRepository interface {
	// LoadKillSourceClassesAggregated charge les kills agreges par (xuid, weapon_key).
	// L'appelant doit avoir appele filters.Validate() avant.
	LoadKillSourceClassesAggregated(
		ctx context.Context,
		slug string,
		filters KillSourceClassFilters,
	) ([]KillSourceClassRow, error)
}
