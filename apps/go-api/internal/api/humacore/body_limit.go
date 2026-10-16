// Package humacore — body_limit.go : plafond de corps par défaut des opérations.
//
// Huma ne pose son propre plafond (1 Mio) que pour une entrée à champ Body ; une
// entrée à champ RawBody est lue intégralement en mémoire, sans limite. Les trois
// fabriques d'API (NewAPI, NewAPIWithConfig, NewSubrouterAPI) posent donc
// DefaultMaxBodyBytes sur toute opération enregistrée sans MaxBody explicite.
package humacore

import "github.com/danielgtaylor/huma/v2"

// DefaultMaxBodyBytes est le plafond de corps (1 Mio) de toute opération qui ne
// déclare pas MaxBody. Une exception se déclare avec MaxBody, à côté de Op, et se
// justifie par la constante du domaine qui la fonde.
//
// Le garde-rail TestEveryOperationHasBodyLimit (paquet api) exige MaxBodyBytes > 0
// sur chaque opération du document.
const DefaultMaxBodyBytes int64 = 1 << 20

// applyDefaultMaxBody pose DefaultMaxBodyBytes sur une opération sans plafond de
// corps déclaré. Un plafond explicite, y compris supérieur au défaut, est conservé.
func applyDefaultMaxBody(op *huma.Operation) {
	if op.MaxBodyBytes == 0 {
		op.MaxBodyBytes = DefaultMaxBodyBytes
	}
}

// DocumentOperation implémente huma.OperationDocumenter. huma.Register l'appelle
// sur l'opération ORIGINALE, celle que le handler relit à chaque requête, avant
// que le document ou un groupe n'en fasse des copies : c'est le seul point où le
// plafond par défaut atteint à la fois l'exécution et le document.
func (a *apiWithAdapter) DocumentOperation(op *huma.Operation) {
	applyDefaultMaxBody(op)
	if !op.Hidden {
		a.OpenAPI().AddOperation(op)
	}
}

// DocumentOperation pose le plafond par défaut sur l'opération originale puis
// laisse le groupe en faire ses copies préfixées.
func (s *subrouterAPI) DocumentOperation(op *huma.Operation) {
	applyDefaultMaxBody(op)
	s.Group.DocumentOperation(op)
}
