package replay

// fallbacks_publication.go — CE QUE L'ARTEFACT DIT DE SES PROPRES REPLIS (schema 58, D14).
//
// Un REPLI est une decision de secours prise quand la lecture du film ne tranche pas. Le
// REGISTRE (`film/replay/fallback`) les declare tous — nom stable, condition typee, date de
// pose, cible et critere de retrait — et un `fallback.Compteur` par cuisson compte ceux qui se
// declenchent. Ce fichier est la seule porte entre ce compteur et le document publie.
//
// POURQUOI UN FICHIER A PART : `build.go` pesait 530 lignes AVANT ce lot — deja au-dela du seuil
// de 500 du depot —, et la regle 5 interdit d'accroitre une dette gelee. Ni lui ni `coverage.go`
// ne grandissent donc pour un calque neuf. L'assemblage n'appelle qu'une
// ligne, en toute fin de `BuildFromPositions`.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// FallbackHit est un repli et son nombre de declenchements sur CETTE cuisson.
//
// LE TYPE EST DEFINI ICI, ET PAS DANS `fallback`, POUR NE PAS FAIRE VOYAGER UN TYPE DU DECODEUR
// JUSQU'AU DOCUMENT SERVI : `internal/domain/replaydoc` est le jumeau de ce document et ne
// depend d'aucun paquet de titre. Deux structures identiques de deux champs scalaires coutent
// moins qu'une frontiere percee.
type FallbackHit struct {
	// Name est le nom STABLE du repli, tel qu'il se relit au registre.
	Name string `json:"name"`
	// Hits est le nombre de declenchements sur cette cuisson.
	Hits int `json:"hits"`
}

// attachFallbackCoverage pose la liste des replis declenches sur le document, et la journalise.
//
// IL S'APPELLE EN DERNIER, ET C'EST OBLIGATOIRE : les calques qui se replient le plus (drapeau,
// zones, collines, bombe, vehicules) sont poses APRES `buildCoverage`. L'appeler plus haut ne
// porterait que les replis du balayage et des premiers calques.
func attachFallbackCoverage(doc *ReplayDocument, c *fallback.Compteur) {
	if doc.Coverage == nil {
		return // aucun calque de couverture : rien ou poser le compte
	}
	doc.Coverage.Fallbacks = fallbackHitsOf(c)
	// UNE LIGNE PAR REPLI DECLENCHE, et pas un total : un total ne designe aucun chantier, alors
	// qu'un nom en designe un — le registre porte sa cible de retrait en face.
	for _, h := range doc.Coverage.Fallbacks {
		slog.Info("rejeu : repli declenche",
			"match_id", doc.MatchID, "repli", h.Name, "declenchements", h.Hits)
	}
}

// fallbackHitsOf projette le rapport du compteur vers ce que le document publie. Rend nil quand
// aucun repli ne s'est declenche — `omitempty` fait alors disparaitre le champ.
func fallbackHitsOf(c *fallback.Compteur) []FallbackHit {
	rap := c.Rapport()
	if len(rap) == 0 {
		return nil
	}
	out := make([]FallbackHit, 0, len(rap))
	for _, d := range rap {
		out = append(out, FallbackHit{Name: string(d.Nom), Hits: d.Declenchements})
	}
	return out
}
