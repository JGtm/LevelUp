// Package analysis — mmr_known.go : règle unique « un MMR à 0 est une valeur absente ».
//
// L'API de compétence rend un MMR à 0 sur un match classé abandonné (tout le monde est parti) :
// ce 0 n'est pas un niveau, c'est l'absence de mesure. Il ne doit entrer ni dans une moyenne,
// ni dans un écart (équipe - adversaire), ni dans un pic, ni dans un régresseur. Cette règle
// vit ICI et nulle part ailleurs : un MMR est connu s'il est strictement positif (un niveau
// réel ne l'est jamais autrement). Les lectures SQL passent par [SQLKnownMMR], les valeurs Go
// par [MMRKnown] / [KnownMMR] ; le garde-rail archlint/no_raw_mmr_known_test.go interdit de
// réécrire le test à la main.
//
// La donnée stockée n'est pas réécrite : une ligne déjà écrite avec 0 est neutralisée à la
// lecture, et l'écriture (persist.SanitizeParticipantRow) ne produit plus de 0.
package analysis

// MMRKnown dit si v est un MMR réel : strictement positif et fini.
func MMRKnown(v float64) bool {
	return v > 0 && !IsBadFloat(v)
}

// KnownMMR rend p quand il porte un MMR réel, nil sinon (nil, 0, négatif, NaN, Inf).
func KnownMMR(p *float64) *float64 {
	if p == nil || !MMRKnown(*p) {
		return nil
	}
	return p
}

// KnownMMRValue rend un pointeur sur v quand v est un MMR réel, nil sinon : pour les lectures
// dont la colonne est non nullable (le 0 y tient lieu d'« inconnu »).
func KnownMMRValue(v float64) *float64 {
	if !MMRKnown(v) {
		return nil
	}
	return &v
}

// SQLKnownMMR rend l'expression SQL qui ne garde de la colonne que son MMR réel (NULL sinon).
// col porte son préfixe d'alias éventuel ("p.team_mmr", "mp.enemy_mmr"). Même règle que
// [MMRKnown] : strictement positif.
func SQLKnownMMR(col string) string {
	return "CASE WHEN " + col + " > 0 THEN " + col + " END"
}
