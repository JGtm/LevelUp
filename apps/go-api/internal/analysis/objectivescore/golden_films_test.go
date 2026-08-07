package objectivescore

// golden_films_test.go — LE GOLDEN SUR FILMS RÉELS : six matchs, leurs octets, et un
// fichier figé qui dit ce que le décodeur en tire.
//
// CE QU'IL RÉPARE. Jusqu'ici la substance de ce paquet — les positions de bit
// `shOffTeam0=24`, `shOffTeam1=23`, la fenêtre `[835,912)`, le token `0x7B6`, les offsets
// KOTH — n'était contrainte par AUCUN octet réel. Les fixtures de `decode_test.go` sont
// écrites AVEC ces mêmes constantes : les décaler laisse la suite verte (mesuré par l'audit
// du 2026-08-06 : `shOffTeam0` 24 -> 23 => suite VERTE). Un décodeur dont on peut déplacer
// l'ancre sans qu'un test bronche n'est pas testé.
//
// CE QU'IL FIGE, ET DANS CET ORDRE DE PRIORITÉ :
//
//	LA POSITION DE BIT DU TOKEN, PAR CHUNK   c'est elle qui répond de `scoreToken`,
//	                                         `tokenWinLo` et `tokenWinHi`. Elle ne dépend
//	                                         d'aucune calibration.
//	LES VALEURS BRUTES, AVANT CALIBRATION    c'est le seul endroit où `shOffTeam0` et
//	                                         `shOffTeam1` répondent de quoi que ce soit.
//	                                         Voir ci-dessous : les valeurs calibrées, elles,
//	                                         n'en répondent PAS.
//	LES VALEURS PUBLIÉES                     ce que l'appelant recevrait réellement.
//
// PIÈGE STRUCTUREL, ET IL A DÉJÀ MORDU. `calibrateByFinal` remet la dernière frame
// EXACTEMENT sur le final DB. Toute assertion de la forme « la dernière frame vaut le final »
// est donc vraie par construction, quel que soit l'offset lu — c'était l'essentiel de
// l'ancien test cache-backed. Ce golden fige les brutes pour cette raison précise.
//
// CE QUE LE CORPUS RÉVÈLE, ET QUI EST UNE DONNÉE, PAS UN ÉCHEC. Le fichier figé montre noir
// sur blanc ce que l'en-tête du paquet annonce depuis la cross-validation du 2026-06-03 :
// en Strongholds la brute team0 PLAFONNE (50 sur `7344d24f` dont le final API est 193, 50
// sur `696a9d7c` dont le final API est 200) et team1 rend des valeurs quasi identiques sur
// des matchs aux finals très différents. La courbe publiée ne « retombe sur le final » que
// parce qu'on la calibre dessus. Le golden ne prétend donc pas que le décodeur est JUSTE :
// il verrouille ce qu'il LIT, pour qu'une dérive silencieuse des offsets devienne visible.
//
// EXÉCUTION : ce test exige `FILM_CACHE_ROOT` (voir filmcache_test.go) et ne tourne donc pas
// en CI. Le filet inconditionnel est `minibobine_test.go`.
//
//	FILM_CACHE_ROOT=<repo>/data/cache \
//	  go test ./internal/analysis/objectivescore/ -run TestGoldenFilmsReels [-update]

import (
	"fmt"
	"strings"
	"testing"
)

// filmReel : un film du corpus et sa vérité terrain, telle qu'elle est SOURCÉE.
//
// Le final n'est pas un chiffre choisi pour faire passer le test : c'est le score de
// l'API/registre, écrit dans la documentation de rétro-ingénierie du chantier. La source est
// citée par film — un oracle sans provenance ne vaut pas mieux qu'une fixture auto-validante.
type filmReel struct {
	Court   string // identifiant court du film dans le cache
	Variant string // game_variant_name, tel qu'il serait passé à DecodeScoreTimeline
	FinalT0 int    // score final équipe 0 (API / registre)
	FinalT1 int    // score final équipe 1
	Source  string // d'où vient ce final
}

// corpusFilmsReels : les six films à vérité terrain écrite. Strongholds et les deux
// variantes KOTH y sont représentés — les trois chemins de décodage du paquet.
var corpusFilmsReels = []filmReel{
	{"7344d24f", "Arena:Strongholds", 193, 112, "decode.go (calibration d'origine) + cache_backed_test.go"},
	{"696a9d7c", "Arena:Strongholds", 200, 94, "ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §2 (Vagabond, 561 s)"},
	{"0a247154", "Ranked:King of the Hill", 4, 2, "koth.go (le seul cas VALIDÉ EXACT, variante B)"},
	{"01e1f945", "KOTH:Arena", 3, 2, "ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §2 (Catalyst, 540 s)"},
	{"606d9844", "KOTH:Arena", 105, 8, "ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §15 (film 0-3 / API 105-8)"},
	{"8076f97f", "KOTH:Arena", 78, 105, "ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §15 (film 3-0 / API 78-105)"},
}

func TestGoldenFilmsReels(t *testing.T) {
	racine := racineCacheFilm(t)
	var b strings.Builder
	b.WriteString(enteteGoldenFilms)
	for _, f := range corpusFilmsReels {
		rendreFilmReel(&b, f, chargerChunksFilm(t, racine, f.Court))
	}
	comparerGolden(t, "films_reels", b.String())
}

const enteteGoldenFilms = `# GOLDEN objectivescore — FILMS REELS
#
# Fige ce que le decodeur LIT dans des octets que personne n a ecrits pour lui. Ne PAS
# editer a la main : regenerer avec -update, apres avoir LU le diff.
#
#   FILM_CACHE_ROOT=<repo>/data/cache \
#     go test ./internal/analysis/objectivescore/ -run TestGoldenFilmsReels -update
#
# LES BRUTES SONT LA SUBSTANCE. Les colonnes cal0/cal1 sont calibrees sur le final : leur
# derniere valeur est le final PAR CONSTRUCTION et ne repond d aucune position de bit. Ce
# sont bitpos, brut0 et brut1 qui repondent du token, de sa fenetre et des offsets.
#
# CE QUE CE FICHIER MONTRE, ET QUI EST UNE DONNEE CONNUE : en Strongholds la brute team0
# plafonne (~50) quel que soit le final reel, et team1 rend des valeurs voisines sur des
# matchs tres differents. Voir l en-tete du paquet, section cross-validation 2026-06-03 :
# la courbe Strongholds n est PAS un score per-equipe. Ce golden verrouille la lecture,
# pas sa justesse.
`

// rendreFilmReel écrit la section d'un film : l'ancrage, puis le détail par mode.
func rendreFilmReel(b *strings.Builder, f filmReel, chunks []ChunkInput) {
	mode := ClassifyMode(f.Variant)
	type2, ancres := comptesAncrage(chunks)
	fmt.Fprintf(b, "\n## %s — %s — final API %d-%d\n", f.Court, f.Variant, f.FinalT0, f.FinalT1)
	fmt.Fprintf(b, "# oracle : %s\n", f.Source)
	fmt.Fprintf(b, "chunks=%d type2=%d ancres=%d mode=%s token=0x%X fenetre=[%d,%d)\n",
		len(chunks), type2, ancres, nomMode(mode), scoreToken, tokenWinLo, tokenWinHi)
	switch mode {
	case ModeStrongholds:
		rendreStrongholds(b, f, chunks)
	case ModeKOTH:
		rendreKOTH(b, f, chunks)
	case ModeUnsupported:
		fmt.Fprintf(b, "mode non gere par ce decodeur : DecodeScoreTimeline rend nil\n")
	}
}

// rendreStrongholds : brut et calibré côte à côte, plus le plafond atteint par la brute —
// c'est lui qui rend visible que la magnitude publiée vient de la calibration.
func rendreStrongholds(b *strings.Builder, f filmReel, chunks []ChunkInput) {
	temps, raw0, raw1 := collectStrongholdsRaw(chunks)
	frames := DecodeStrongholds(chunks, f.FinalT0, f.FinalT1)
	fmt.Fprintf(b, "# instant_ms  bitpos  brut0  brut1  cal0  cal1\n")
	for i := range temps {
		fmt.Fprintf(b, "%d  %d  %d  %d  %d  %d\n",
			temps[i], bitposAt(chunks, i), raw0[i], raw1[i], frames[i].Team0, frames[i].Team1)
	}
	fmt.Fprintf(b, "brut0 %d->%d (plafond %d)  brut1 %d->%d  publie %d-%d  source=%s conf=%s\n",
		raw0[0], raw0[len(raw0)-1], maxDe(raw0), raw1[0], raw1[len(raw1)-1],
		frames[len(frames)-1].Team0, frames[len(frames)-1].Team1,
		frames[0].Source, frames[0].Confidence)
}

// rendreKOTH : les DEUX variantes sur chaque film, même si l'auto-sélection n'en retiendrait
// qu'une. La variante non retenue reste du code livré ; la figer coûte quelques lignes et
// évite qu'elle dérive sans témoin.
func rendreKOTH(b *strings.Builder, f filmReel, chunks []ChunkInput) {
	varAuto := pickKOTHVariant(f.FinalT0, f.FinalT1)
	fB, fA := DecodeKOTH(chunks, VariantB), DecodeKOTH(chunks, VariantA)
	fmt.Fprintf(b, "variante auto-selectionnee = %s (seuil %d)\n", varAuto, kothPointsThreshold)
	fmt.Fprintf(b, "# instant_ms  bitpos  octets[+12,+13,+14,+16]  B:t0-t1  A:t0-t1\n")
	for i := range fB {
		o12, o13, o14, o16 := octetsKOTH(chunks, i)
		fmt.Fprintf(b, "%d  %d  %d,%d,%d,%d  %d-%d  %d-%d\n", fB[i].TimeMS, bitposAt(chunks, i),
			o12, o13, o14, o16, fB[i].Team0, fB[i].Team1, fA[i].Team0, fA[i].Team1)
	}
	if len(fB) == 0 {
		fmt.Fprintf(b, "aucune frame ancree\n")
		return
	}
	fmt.Fprintf(b, "fin B %d-%d  fin A %d-%d  final API %d-%d\n", fB[len(fB)-1].Team0,
		fB[len(fB)-1].Team1, fA[len(fA)-1].Team0, fA[len(fA)-1].Team1, f.FinalT0, f.FinalT1)
}

// comptesAncrage : combien de chunks type-2, et combien portent le token.
func comptesAncrage(chunks []ChunkInput) (type2, ancres int) {
	for _, c := range chunks {
		if c.ChunkType != packetType2 {
			continue
		}
		type2++
		if p, tb := anchoredPayload(c); p != nil && tb >= 0 {
			ancres++
		}
	}
	return type2, ancres
}

// bitposAt : la position de bit du token dans le i-ème chunk ANCRÉ (l'ordre des frames).
func bitposAt(chunks []ChunkInput, i int) int {
	n := 0
	for _, c := range chunks {
		p, tb := anchoredPayload(c)
		if p == nil {
			continue
		}
		if n == i {
			return tb
		}
		n++
	}
	return -1
}

// octetsKOTH : les quatre octets que les deux variantes KOTH lisent, dans le i-ème chunk
// ancré. Les figer distingue « le décodeur lit autre chose » de « le film a changé ».
func octetsKOTH(chunks []ChunkInput, i int) (o12, o13, o14, o16 int) {
	n := 0
	for _, c := range chunks {
		p, tb := anchoredPayload(c)
		if p == nil {
			continue
		}
		if n == i {
			ab := tb / 8
			return byteAt(p, ab+kothBOffT0), byteAt(p, ab+kothAOffT1),
				byteAt(p, ab+kothAOffTot), byteAt(p, ab+kothBOffT1)
		}
		n++
	}
	return -1, -1, -1, -1
}

// maxDe : la plus grande valeur d'une séquence.
func maxDe(v []int) int {
	m := 0
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	return m
}

// nomMode : le libellé court d'un Mode, pour le golden.
func nomMode(m Mode) string {
	switch m {
	case ModeStrongholds:
		return "strongholds"
	case ModeKOTH:
		return "koth"
	case ModeUnsupported:
		return "non-gere"
	}
	return "?"
}
