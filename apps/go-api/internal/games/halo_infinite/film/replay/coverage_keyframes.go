package replay

// coverage_keyframes.go — LA SANTÉ DE LA MARCHE D'IMAGE-CLÉ, PUBLIÉE (lot M3.1 de la campagne
// « retours rejeu », 2026-09-23, schéma 69).
//
// POURQUOI. Les armes portées et l'inventaire des fiches sortent tous deux de la marche de la
// table d'image-clé (`grammar.WalkKeyframeWorld`). Quand cette marche perdait des records — une
// fausse ancre de slot bas élue devant les vrais bipèdes, une fenêtre vide qui coupait la table —,
// la fiche affichait des cellules vides et RIEN dans le document ne le disait (rapport
// `fiche_armes` §6-3 : 213 images-clés touchées sur 2 868 au parc, invisibles). Ce bloc le dit.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar"

// KeyframeCoverage dit, pour le film entier, comment le balayeur d'image-clé a atteint chaque
// record, et combien de bipèdes il a manqués alors que les images-clés voisines les portaient.
//
// Les décisions se lisent ensemble : chaque record (sauf le premier d'une image-clé) est atteint
// par un voisin immédiat, un saut de largeur, un recalage sur l'en-tête exact d'un bipède ou une
// élection — cette dernière est le REPLI nommé `repli_ancre_d_image_cle_par_election`, compté
// aussi dans `coverage.fallbacks`.
type KeyframeCoverage struct {
	// Keyframes est le nombre de payloads d'image-clé marchés.
	Keyframes int `json:"keyframes"`
	// Records est le nombre de records atteints ; Bipeds ceux de l'archétype bipède.
	Records int `json:"records"`
	Bipeds  int `json:"bipeds"`
	// Neighbors / Jumps : records atteints par le voisin immédiat (slot+1, génération 1) ou par
	// le saut de largeur d'un archétype déjà vu deux fois à la même largeur.
	Neighbors int `json:"neighbors"`
	Jumps     int `json:"jumps"`
	// Resyncs : records atteints par l'en-tête EXACT d'un bipède, devant une élection.
	Resyncs int `json:"resyncs"`
	// Elections : records choisis par le repli (génération basse, slot bas).
	Elections int `json:"elections"`
	// Refutations : élus que le repli a REFUSÉS parce qu'un record prouvé par la grammaire du
	// film les contredisait (lot D-fix, 2026-09-24, `grammar/keyframe_world_preuve.go`) ;
	// l'élection a repris sans eux. Chacun est une fausse ancre qui aurait effacé les vrais
	// records qui la précédaient.
	Refutations int `json:"refutations"`
	// Slides : fenêtres de 120 000 bits SANS candidat traversées sans arrêter la marche.
	Slides int `json:"slides"`
	// FramedAbsentBipeds : couples (image-clé, bipède) où le bipède est ancré aux deux
	// images-clés voisines et manque à celle-ci — une perte de la marche, pas une absence.
	FramedAbsentBipeds int `json:"framedAbsentBipeds"`
}

// buildKeyframeCoverage projette la couverture de la marche lue dans le film. Nil quand aucune
// image-clé n'a été marchée : un film sans image-clé lisible ne publie pas une ligne de zéros.
func buildKeyframeCoverage(c grammar.KeyframeWalkCoverage) *KeyframeCoverage {
	if c.Payloads == 0 {
		return nil
	}
	return &KeyframeCoverage{
		Keyframes: c.Payloads, Records: c.Records, Bipeds: c.Bipedes,
		Neighbors: c.Voisins, Jumps: c.Sauts, Resyncs: c.Recalages, Elections: c.Elections,
		Refutations: c.Refutations, Slides: c.Glissements, FramedAbsentBipeds: c.BipedesAbsentsEncadres,
	}
}
