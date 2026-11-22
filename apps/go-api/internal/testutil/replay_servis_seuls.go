package testutil

// replay_servis_seuls.go — LES CHAMPS ET LES TYPES DU DOCUMENT DE REJEU SERVI QUI N'ONT AUCUNE SOURCE
// DANS LE DOCUMENT STOCKE.
//
// Le document stocke (`replay.ReplayDocument`, format de cuisson) et le document servi
// (`domain/replaydoc.ReplayDocument`, forme de fil) sont jumeaux, hors ces champs : des calques RESOLUS
// A LA REQUETE depuis le referentiel du titre, que la cuisson n'ecrit jamais et que le decodeur de film
// ne porte pas. Deux gardes de paquets differents confrontent les deux documents et lisent CETTE liste,
// la seule : `service/replayview/parity_test.go` (champ par champ, et copie de la projection) et
// `internal/filmcorpus/document_shape_jumeaux_test.go` (forme serialisee). Une liste par garde
// diverge ; une exception ne s'ajoute donc qu'ici.
//
// Ce fichier n'est lie qu'aux binaires de test : aucun code de production n'importe `testutil`.
//
// UNE ENTREE SE JUSTIFIE PAR UNE DATE ET UN MOTIF (`2026-11-17 : ...`), et ne survit pas a son champ :
// `ReplayServisSeulsIncoherents` rend, pour ce que le garde a VU du document servi, chaque entree sans
// date, sans motif, ou qui ne designe plus rien.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ReplayChampsServisSeuls : les champs du document SERVI sans source stockee, par `Type.Champ`, avec la
// date de la decision et son motif.
var ReplayChampsServisSeuls = map[string]string{
	"ReplayDocument.MedalLabels": "2026-11-17 : libelle, description, image et rarete de chaque medaille de `medals`, " +
		"poses a la requete par service/replay_medal_labels.go depuis le referentiel du titre.",
	"ReplayDocument.ShieldCycles": "2026-11-22 : instants des bris et des recharges de bouclier de chaque joueur mesure, calcules a la requete " +
		"par service/replay_shield_cycles.go depuis les pistes du document stocke avec analysis/shieldcycle (une seule definition).",
}

// ReplayTypesServisSeuls : les TYPES du document servi qui n'existent que pour un champ de
// `ReplayChampsServisSeuls`, par nom de type, avec la date de la decision. Les gardes les visitent sans leur
// chercher de jumeau stocke.
var ReplayTypesServisSeuls = map[string]string{
	"MedalLabel":       "2026-11-17 : valeur de `ReplayDocument.MedalLabels`.",
	"MedalDescription": "2026-11-17 : description d'un `MedalLabel`.",
	"MedalSprite":      "2026-11-17 : decoupe de feuille de sprites d'un `MedalLabel`.",
	"ShieldCycle":      "2026-11-22 : valeur de `ReplayDocument.ShieldCycles`.",
}

// motifDate : un motif commence par la date de la decision.
var motifDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\b`)

// ReplayServisSeulsIncoherents confronte les deux listes a ce qu'un garde a VU du document servi
// (`champsVus` par `Type.Champ`, `typesVus` par nom de type) et rend, triees, les raisons de refuser :
// une entree sans motif ou dont le motif ne commence pas par une date, une entree qui ne designe plus
// ni champ ni type du document servi. Vide quand tout est coherent.
func ReplayServisSeulsIncoherents(champsVus, typesVus map[string]bool) []string {
	var out []string
	verifier := func(liste string, entrees map[string]string, vus map[string]bool) {
		for cle, motif := range entrees {
			switch {
			case strings.TrimSpace(motif) == "":
				out = append(out, fmt.Sprintf("%s : l'entree %q n'a AUCUNE justification", liste, cle))
			case !motifDate.MatchString(motif):
				out = append(out, fmt.Sprintf("%s : le motif de %q ne commence pas par une date (AAAA-MM-JJ)", liste, cle))
			}
			if !vus[cle] {
				out = append(out, fmt.Sprintf("%s : l'entree %q ne designe plus rien dans le document servi — "+
					"une exception qui survit a son champ finit par en couvrir un autre", liste, cle))
			}
		}
	}
	verifier("ReplayChampsServisSeuls", ReplayChampsServisSeuls, champsVus)
	verifier("ReplayTypesServisSeuls", ReplayTypesServisSeuls, typesVus)
	sort.Strings(out)
	return out
}
