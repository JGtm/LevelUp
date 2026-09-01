// Package halo_infinite — killsource_concordance_test.go : LE GARDE-RAIL DE CONCORDANCE
// entre le kill feed du rejeu 2D et le graphe de répartition des frags (décision D13 du
// plan .ai/V7.5/PLAN_SOURCE_UNIQUE_ARME_2026-09-01.md).
//
// # Le problème qu'il tient
//
// Les deux surfaces lisent le MÊME dictionnaire embarqué (`killicon` adossé à `damagetag`)
// mais lui posent des questions différentes :
//
//   - le kill feed appelle `KillSourceIcon` — il réussit dès que `killicon.Lookup` connaît
//     le tag, donc dès qu'une règle lui donne une vignette ;
//   - le graphe appelle `KillSourceRegistryKey` — il exige en plus une clé de registre
//     (`Icon.WeaponKey`), et se rabat sinon sur la classe `DEGAT_GLOBAL`.
//
// Une règle ajoutée avec une vignette mais SANS clé produit donc un kill que le rejeu sait
// nommer et que le graphe range dans « Non attribué ». Mesuré le 2026-09-01 sur la base
// réelle : 28 tags dans ce cas, 1 786 morts sur un lot de 200 matchs.
//
// # Ce que le garde-rail exige
//
// Que l'ensemble des règles rendant une IMAGE et l'ensemble des règles rendant une CLÉ
// soient identiques — à l'exception d'une allowlist EXPLICITE, DATÉE, justifiée entrée par
// entrée. Toute règle ajoutée plus tard sans clé fera rougir ce test. Modèle :
// `internal/sync/no_art_patterns_test.go`.
//
// L'allowlist est aussi un RATCHET DANS L'AUTRE SENS : une entrée qui a gagné sa clé doit
// en sortir. Sans quoi elle deviendrait une dispense permanente, ce qu'aucune allowlist ne
// doit être.
package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/killicon"
)

// dispenseConcordance : une règle autorisée à donner une image sans donner de clé.
type dispenseConcordance struct {
	genre  killicon.Genre
	cle    string // le champ `cle` de la règle, verbatim
	depuis string // date d'entrée dans l'allowlist
	raison string // pourquoi le graphe n'a pas de clé à lui donner
}

// dispensesConcordance — l'allowlist, mesurée le 2026-09-01 (étape A0.4 du plan).
//
// ELLE EST PASSÉE DE 19 À 5 ENTRÉES à la clôture de l'étape A6 (2026-09-01). Les
// quatorze règles BANQUE — châssis et tourelles — portent désormais leur clé de registre
// et sont donc SORTIES d'ici : c'est le ratchet inverse, une dispense qui survit à sa
// raison d'être couvrirait un jour une régression qu'elle n'a jamais eu à couvrir.
//
// Les cinq qui restent ne sont pas un retard, ce sont cinq refus motivés : trois objets
// que le registre ne porte pas (et ne DOIT pas porter sans décision de catalogue), une
// quatrième grenade que le titre ne distingue pas, et le repli de classe de la mêlée dont
// le total vient du compteur API.
var dispensesConcordance = []dispenseConcordance{
	{killicon.GenreNom, "Infected Energy Sword", "2026-09-01",
		"variante infectée de l'épée : le registre ne porte que l'arme canonique, et la " +
			"distinguer inventerait une entrée que le jeu ne distingue pas non plus en statistiques"},
	{killicon.GenreNom, "Mythic Sandwich / Sandwich", "2026-09-01",
		"objet de mode événementiel, absent de l'arsenal : aucune entrée de registre à lui donner"},
	{killicon.GenreNom, "Mutilator", "2026-09-01",
		"arme nommée par le film mais absente du registre — mesure, pas oubli (116 morts " +
			"sur le lot du 2026-09-01) ; l'ajouter est une décision de catalogue, hors de ce lot"},
	{killicon.GenreGGGL, "3", "2026-09-01",
		"entrée 3/4 de la liste des grenades (`kineticbanished`, grenade à pointes) : le " +
			"registre Halo Infinite ne porte que frag, plasma et dynamo"},
	{killicon.GenreClasse, "MELEE", "2026-09-01",
		"repli de classe : la mêlée est servie par le compteur API `melee_kills`, dont le " +
			"total est autoritatif — le graphe n'a pas besoin d'une clé pour la compter (D4)"},
}

// TestConcordanceImageEtCle : toute règle qui donne une IMAGE doit donner une CLÉ, sauf
// dispense explicite.
func TestConcordanceImageEtCle(t *testing.T) {
	dispensees := map[[2]string]bool{}
	for _, d := range dispensesConcordance {
		if d.depuis == "" || d.raison == "" {
			t.Fatalf("dispense %s/%s sans date ou sans raison : une allowlist non justifiée "+
				"est une dispense permanente", d.genre, d.cle)
		}
		dispensees[[2]string{string(d.genre), d.cle}] = true
	}
	for _, r := range killicon.Rules() {
		if r.WeaponKey != "" {
			continue
		}
		if !dispensees[[2]string{string(r.Genre), r.Key}] {
			t.Errorf("règle %s %q : donne l'image %q mais AUCUNE clé de registre.\n"+
				"Le rejeu 2D saura nommer ce kill, le graphe le rangera dans « Non attribué ».\n"+
				"Renseigner la colonne weapon_key de rules.tsv, ou ajouter une dispense datée "+
				"et justifiée à dispensesConcordance.", r.Genre, r.Key, r.Sprite)
		}
	}
}

// TestConcordanceAllowlistSansEntreePerimee : le ratchet dans l'autre sens. Une dispense
// dont la règle a gagné sa clé doit SORTIR de l'allowlist — sinon elle survivrait à sa
// raison d'être et couvrirait, un jour, une régression qu'elle n'a jamais eu à couvrir.
func TestConcordanceAllowlistSansEntreePerimee(t *testing.T) {
	sansCle := map[[2]string]bool{}
	for _, r := range killicon.Rules() {
		if r.WeaponKey == "" {
			sansCle[[2]string{string(r.Genre), r.Key}] = true
		}
	}
	for _, d := range dispensesConcordance {
		if !sansCle[[2]string{string(d.genre), d.cle}] {
			t.Errorf("dispense %s %q PÉRIMÉE : la règle porte désormais une clé de registre "+
				"(ou a disparu). La retirer de dispensesConcordance.", d.genre, d.cle)
		}
	}
}

// TestConcordanceTagsSansCleSontConnus : la vérification côté TAGS, celle qui compte pour
// le lecteur. Un tag qui obtient une image sans obtenir de clé doit appartenir à une classe
// `damagetag` pour laquelle on sait dire pourquoi.
//
// Trois classes seulement sont acceptables sans clé depuis la clôture de l'étape A6
// (2026-09-01) : MELEE et GRENADE (servies par les compteurs API, décision D4) et ARME
// (trois objets que le registre ne porte pas : Mutilator, Mythic Sandwich, épée infectée).
// VEHICULE en est SORTIE — les quatorze châssis et tourelles ont désormais leur clé.
// Une classe INCONNUE ou OBJET_EXPLOSIF dans cette liste serait une régression.
func TestConcordanceTagsSansCleSontConnus(t *testing.T) {
	acceptables := map[damagetag.Class]bool{
		damagetag.ClassMelee:   true, // compteur API
		damagetag.ClassGrenade: true, // compteur API
		damagetag.ClassArme:    true, // Mutilator, Sandwich, épée infectée
		// ClassVehicule EST SORTIE le 2026-09-01 : l'étape A6 a donné une clé de registre
		// aux quatorze châssis et tourelles. Un tag VEHICULE sans clé est désormais une
		// RÉGRESSION, plus une dette.
	}
	reg := NewKillSourceRegistry()
	for _, tag := range killicon.ResolvedTags() {
		if _, ok := reg.KillSourceRegistryKey(tag); ok {
			continue
		}
		l, known := damagetag.Lookup(tag)
		if !known {
			t.Errorf("tag 0x%08x : image sans clé, et inconnu de damagetag — incohérence "+
				"entre les deux tables embarquées", tag)
			continue
		}
		if !acceptables[l.Class] {
			t.Errorf("tag 0x%08x (%s, %q) : image sans clé dans une classe non prévue. "+
				"Soit la règle doit porter une clé, soit la classe doit rejoindre la liste "+
				"des acceptables avec sa justification.", tag, l.Class, l.Name)
		}
	}
}
