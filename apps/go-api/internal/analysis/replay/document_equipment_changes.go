package replay

// document_equipment_changes.go — LES RAMASSAGES ET LES CONSOMMATIONS D'ÉQUIPEMENT, datés à
// la milliseconde.
//
// CE QUE C'EST. `abilities[]` dit ce qu'un joueur PORTE, lecture par lecture. Ce calque-ci dit
// ce qui lui ARRIVE : il a ramassé un équipement, ou il vient de le consommer. Même matière
// (i48), deux questions différentes — c'est le rapport qu'entretiennent `inventory[]` et
// `weaponChanges[]` côté armes.
//
// CE QUI N'EST PAS PUBLIÉ, ET POURQUOI. Les annonces de RÉAPPARITION. Sur les modes où les
// joueurs renaissent équipés, la première émission d'une vie tombe dans le même paquet que sa
// première position : ce n'est pas un ramassage, et le publier comme tel doublerait le
// décompte. Ces émissions restent comptées dans la couverture (`spawned`), et ce que le joueur
// porte à sa naissance est déjà dans `abilities[]`.
//
// CE QUE LA COUVERTURE APPORTE ICI ET NULLE PART AILLEURS : un TÉMOIN DE COMPLÉTUDE. Le
// compteur de rotation d'i48 avance de 1 à chaque émission ; un pas plus grand dénonce des
// émissions manquées et les compte. `missedEstimate` n'est donc pas une estimation au doigt
// mouillé — c'est une lecture. Mesuré sur trois films : environ 16 émissions manquées pour
// 319 vues. Aucun autre calque du rejeu ne sait dire cela de lui-même.

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// EquipmentChangeKind qualifie un changement d'équipement, tel que le document le publie.
type EquipmentChangeKind string

const (
	// EquipmentTaken : le joueur porte désormais cet équipement — un RAMASSAGE sur la carte.
	EquipmentTaken EquipmentChangeKind = "taken"
	// EquipmentSpent : le joueur n'en porte plus. Il l'a CONSOMMÉ, et la mesure l'établit :
	// sur les 17 émissions à porte ouverte du corpus, aucune ne tombe dans la dernière
	// seconde de la vie — ce n'est jamais la mort qui vide l'emplacement.
	EquipmentSpent EquipmentChangeKind = "spent"
)

// NoAbilityRank est la valeur des champs de rang quand il n'y a PAS d'équipement (sur un
// `spent`) ou quand le rang précédent n'est pas lisible. C'est la même sentinelle que celle
// du décodeur ; elle est publiée telle quelle plutôt qu'omise, pour qu'un client n'ait pas à
// distinguer « champ absent » de « rang zéro » — le rang 0 existe.
const NoAbilityRank = filmdec.AbilitySetNoRank

// EquipmentChange est UN changement d'équipement porté.
type EquipmentChange struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot désigne la Track concernée — donc une VIE, pas un joueur.
	Slot uint32 `json:"slot"`
	// Kind qualifie le changement.
	Kind EquipmentChangeKind `json:"kind"`
	// R est le RANG de palette de l'équipement désormais porté, même convention que
	// `AbilityRead.R` — c'est `AbilityLabels` qui le nomme, ou pas. Vaut NoAbilityRank sur
	// un `spent`.
	R int `json:"r"`
	// From est le rang précédent sur cette vie, ou NoAbilityRank quand il n'est pas lisible
	// (première émission de la vie : le film ne dit pas ce qui la précédait).
	From int `json:"from"`
}

// EquipmentChangeCoverage dit ce que le calque a vu, ce qu'il a écarté, et — seul de tous les
// calques du rejeu — ce qu'il a MANQUÉ.
type EquipmentChangeCoverage struct {
	// Decoded est le nombre de changements rendus par le décodeur.
	Decoded int `json:"decoded"`
	// Published est le nombre publié dans le document.
	Published int `json:"published"`
	// Taken / Spent ventilent les publiés.
	Taken int `json:"taken"`
	Spent int `json:"spent"`
	// Spawned compte les annonces de réapparition — écartées (cf. l'en-tête du fichier).
	Spawned int `json:"spawned"`
	// BeforeOrigin compte les changements antérieurs à la première frame — écartés.
	BeforeOrigin int `json:"beforeOrigin"`
	// Lives est le nombre de vies ayant émis au moins une fois.
	Lives int `json:"lives"`
	// MissedEstimate est le nombre d'émissions que le compteur de rotation dit MANQUÉES, et
	// CounterJumps le nombre d'endroits où il l'a dit. Un lecteur qui veut juger la
	// couverture de ce calque n'a besoin de rien d'autre.
	MissedEstimate int `json:"missedEstimate"`
	CounterJumps   int `json:"counterJumps"`
	// LivesFirstOffSpec compte les vies dont la première émission n'a pas le compteur
	// attendu : des émissions ont été manquées en amont, ou le slot a été mal ancré.
	LivesFirstOffSpec int `json:"livesFirstOffSpec"`
	// Repeats compte les émissions qui ne font pas avancer le compteur. Une valeur non nulle
	// contredirait la propriété qui fonde le calque — le composant ne devrait entrer au
	// masque QUE sur changement.
	Repeats int `json:"repeats"`
}

// buildEquipmentChanges projette les changements lus dans le film sur l'axe de frames du
// document. Les annonces de réapparition et les événements antérieurs à l'origine sont
// écartés — un rejeu ne montre pas ce qui précède sa première frame.
func buildEquipmentChanges(
	changes []filmdec.EquipmentChange, st filmdec.EquipmentChangeStats, origin, step uint64,
) ([]EquipmentChange, EquipmentChangeCoverage) {
	cov := EquipmentChangeCoverage{
		Decoded: len(changes), Lives: st.Lives, MissedEstimate: st.MissedEstimate,
		CounterJumps: st.CounterJumps, LivesFirstOffSpec: st.LivesFirstOffSpec,
		Repeats: st.Repeats,
	}
	if len(changes) == 0 || step == 0 {
		return nil, cov
	}
	out := make([]EquipmentChange, 0, len(changes))
	for _, c := range changes {
		if c.Kind == filmdec.EquipmentSpawned {
			cov.Spawned++
			continue
		}
		if c.TimestampUS < origin {
			cov.BeforeOrigin++
			continue
		}
		e := EquipmentChange{
			T: int((c.TimestampUS - origin) / step), Slot: c.Slot,
			R: c.Rank, From: c.Previous,
		}
		if c.Kind == filmdec.EquipmentSpent {
			e.Kind, cov.Spent = EquipmentSpent, cov.Spent+1
		} else {
			e.Kind, cov.Taken = EquipmentTaken, cov.Taken+1
		}
		out = append(out, e)
		cov.Published++
	}
	if len(out) == 0 {
		return nil, cov
	}
	return out, cov
}

// keepEquipmentChangesOfPublishedTracks écarte les changements dont le slot n'a pas de
// trajectoire publiée : le client n'aurait aucune fiche où les poser (même règle que
// `keepAbilitiesOfPublishedTracks`).
func keepEquipmentChangesOfPublishedTracks(
	changes []EquipmentChange, tracks []Track,
) []EquipmentChange {
	return keepOfPublishedTracks(changes, tracks,
		func(c EquipmentChange, published map[uint32]bool) bool { return published[c.Slot] })
}

// birthOfLives construit le témoin de naissance exigé par `ScanFilmEquipmentChanges` : le
// premier échantillon de position de chaque vie, sur l'horloge du film.
//
// C'EST LUI QUI DISTINGUE un ramassage d'une annonce de réapparition, et le balayage ne sait
// pas le faire sans. Le nuage employé est celui des positions BRUTES, avant décimation : une
// naissance décimée décalerait le témoin de plusieurs dixièmes de seconde, juste assez pour
// classer un ramassage précoce en réapparition.
func birthOfLives(positions []filmdec.BipedPosition) func(uint32) (uint64, bool) {
	if len(positions) == 0 {
		return nil
	}
	first := make(map[uint32]uint64, 64)
	for _, p := range positions {
		if at, ok := first[p.Slot]; !ok || p.TimestampUS < at {
			first[p.Slot] = p.TimestampUS
		}
	}
	return func(slot uint32) (uint64, bool) {
		at, ok := first[slot]
		return at, ok
	}
}

// logEquipmentChangeCoverage sort la couverture du calque, TÉMOIN DE COMPLÉTUDE COMPRIS : un
// journal qui ne dirait que les publiés laisserait croire qu'ils sont tous là.
func logEquipmentChangeCoverage(cov EquipmentChangeCoverage) {
	slog.Info("rejeu : ramassages et consommations d equipement",
		"decodes", cov.Decoded, "publies", cov.Published,
		"ramassages", cov.Taken, "consommations", cov.Spent,
		"reapparitions", cov.Spawned, "avantOrigine", cov.BeforeOrigin,
		"vies", cov.Lives, "manqueesEstimees", cov.MissedEstimate,
		"sautsCompteur", cov.CounterJumps, "premiereHorsNorme", cov.LivesFirstOffSpec,
		"repetitions", cov.Repeats)
}
