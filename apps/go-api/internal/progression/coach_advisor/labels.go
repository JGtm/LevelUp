package coach_advisor

import (
	"levelup/go-api/internal/i18n"
)

// labels.go — titres et descriptions narratifs des arcs dynamiques composés par arc_composer
// (cf. ADR 0028 §"Cohérence narrative"). Les textes vivent dans le catalogue des messages
// (`internal/i18n/messages/coach.toml`) : une valeur par langue de l'application.
//
// Templates statiques par radar_axis. Pas de génération LLM — déterministe
// et indépendant de tout service externe. Pour un titre futur sans
// correspondance, fallback générique.

// arcTitleForAxis retourne le titre d'un arc paramétré par son axis pivot, dans toutes les
// langues. ADR 0028 §"Cohérence narrative" :
//
//	combat     : Combat Mastery       / Domination en combat
//	survival   : Resilient Comeback   / Reprise solide
//	support    : Team Pillar          / Pilier d'équipe
//	objective  : Objective First      / Objectif d'abord
//	score      : Elite Performance    / Performance d'élite
//	impact     : Impact Player        / Joueur d'impact
func arcTitleForAxis(axis string) i18n.Text {
	switch axis {
	case "combat":
		return i18n.MsgText("coach.arc.title.combat", nil)
	case "survival":
		return i18n.MsgText("coach.arc.title.survival", nil)
	case "support":
		return i18n.MsgText("coach.arc.title.support", nil)
	case "objective":
		return i18n.MsgText("coach.arc.title.objective", nil)
	case "score":
		return i18n.MsgText("coach.arc.title.score", nil)
	case "impact":
		return i18n.MsgText("coach.arc.title.impact", nil)
	}
	// Fallback générique — utilisé si un futur titre introduit un axis inconnu.
	return i18n.MsgText("coach.arc.title.default", nil)
}

// arcDescriptionForAxis retourne une description courte, dans toutes les langues, paramétrée
// par l'axis pivot et le nombre d'étapes. Tone neutre, factuel — pas de promesse marketing.
func arcDescriptionForAxis(axis string, stepCount int) i18n.Text {
	return i18n.MsgTextWith("coach.arc.description", func(l i18n.Locale) i18n.Params {
		return i18n.Params{"steps": stepCount, "axis": axisHuman(l, axis)}
	})
}

// axisHuman retourne le nom de l'axis pivot dans la langue `l`, tel qu'inséré dans la phrase de
// description de l'arc. Axis inconnu : l'identifiant lui-même (précédé de « sur » en français).
func axisHuman(l i18n.Locale, axis string) string {
	switch axis {
	case "combat":
		return i18n.Msg(l, "coach.axis.combat", nil)
	case "survival":
		return i18n.Msg(l, "coach.axis.survival", nil)
	case "support":
		return i18n.Msg(l, "coach.axis.support", nil)
	case "objective":
		return i18n.Msg(l, "coach.axis.objective", nil)
	case "score":
		return i18n.Msg(l, "coach.axis.score", nil)
	case "impact":
		return i18n.Msg(l, "coach.axis.impact", nil)
	}
	return i18n.Msg(l, "coach.axis.other", i18n.Params{"axis": axis})
}

// reasonKeys retourne la clé d'i18n du motif d'une proposition pour chaque langue de l'application :
// la clé de base suivie de « . » et du code de la langue (`coach.proposal.arc.combat.fr`).
func reasonKeys(base string) i18n.Text {
	keys := make(i18n.Text, len(i18n.Locales()))
	for _, l := range i18n.Locales() {
		keys[l] = base + "." + string(l)
	}
	return keys
}

// Bases des libellés que Proposal.ReasonParams stocke une fois par langue (`label_fr`,
// `label_en`... : voir i18n.ParamKey). Un défi proposé porte `label` ; un arc proposé porte
// `title` et `description`.
const (
	ReasonParamLabel       = "label"
	ReasonParamTitle       = "title"
	ReasonParamDescription = "description"
)

var reasonParamBases = []string{ReasonParamLabel, ReasonParamTitle, ReasonParamDescription}

// ResolveReasonParams retourne ReasonParams pour la langue `l` : chaque famille de libellés
// stockée par langue devient une clé unique (`label`, `title`, `description`). Les lignes déjà
// stockées (`label_fr`, `label_en`) se lisent ainsi sans migration ; une langue absente d'une
// ligne replie sur l'anglais. Un JSON illisible rend l'erreur, l'appelant la journalise.
func ResolveReasonParams(raw string, l i18n.Locale) (string, error) {
	return i18n.ResolveParamsJSON(raw, reasonParamBases, l)
}
