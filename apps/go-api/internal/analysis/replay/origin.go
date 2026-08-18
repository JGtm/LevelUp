package replay

import (
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// origin.go — L'ORIGINE DE LA FRAME 0, PUBLIEE SUR L'HORLOGE DU FIL.
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// `build.go` cale sa frame 0 sur le PREMIER PAQUET DE POSITION (`origin =
// sorted[0].TimestampUS`). Ce referentiel n'existe nulle part ailleurs : le fil des
// eliminations vit sur l'horloge des highlight events, que le client reconstruit par
// `event_time_ms + t0_ms` (le T0 reel, `match_registry.real_start_time`, est deja servi par
// la Match View). Poser les deux sur le meme axe sans rien publier decalait le fil de 3,7 s
// a 39,8 s selon le match — mesure ci-dessous.
//
// # POURQUOI L'ORIGINE EST PUBLIEE SUR L'HORLOGE DU FILM, ET PAS BRUTE
//
// L'horodatage de paquet N'EST PAS relatif au debut du film : c'est une horloge MOTEUR, et
// elle vaut 4 521 s / 3 896 s / 2 259 s / 8 583 s sur les quatre films temoins — un temps
// depuis le demarrage du jeu. Publier ce nombre tel quel ne donnerait au client aucun zero
// commun. Le calage sur `start_time_utc` du match, lui, ferait entrer une horloge murale
// (et sa dette de fuseau connue) dans un artefact qui n'ouvre aucune base.
//
// Ce qui est publie est donc la DIFFERENCE de deux lectures du meme film :
//
//	originMs = (premier paquet de position − premier paquet du chunk 1) / 1000
//
// Le premier paquet du chunk 1 est le debut du film (le manifeste lui donne
// `start_ms = 0`), c'est-a-dire exactement le zero des `event_time_ms`. Aucun appariement,
// aucune estimation : deux entiers lus, une soustraction.
//
// # LA MESURE, ET SON TEMOIN INDEPENDANT (2026-08-14, `origin_research_test.go`)
//
//	film       originMs (lecture)   temoin par le fil des morts   ecart
//	000d5950            3 604 ms                     3 660 ms     56 ms
//	64e8adfa           10 516 ms                    10 573 ms     57 ms
//	e94163af           39 772 ms                    39 788 ms     16 ms
//	606d9844            6 890 ms                     6 971 ms     81 ms
//
// Le temoin est `bestDeathOffset` (deja en production pour nommer les vies) : il resout le
// decalage entre l'horloge du fil des morts et celle du film par plateau, au pas de 10 ms.
// Les deux chemins ne partagent aucune piece — l'un lit deux en-tetes de paquet, l'autre
// apparie 90 a 122 morts a des fins de vie. Leur accord a moins de 100 ms est ce qui fonde
// la publication.
//
// Rappel de ce que le client mesurait AVANT, par appariement statistique cote navigateur :
// +3 678 / +10 589 / +39 856 ms sur les trois premiers films — memes valeurs, a 20-70 ms.
//
// # LE REFUS EST EXPLICITE
//
// Un chunk 1 illisible (cache tronque) rend l'origine incalculable : le document ne publie
// alors AUCUNE origine, et le client retombe sur son appariement (cf. killFeedLogic.ts).
// Un temoin qui contredit la lecture de plus de `originControlToleranceMS` fait de meme :
// on ne sert jamais une origine douteuse en silence.

// originControlToleranceMS borne l'ecart accepte entre l'origine LUE et le temoin issu du
// fil des morts. Les quatre films temoins tiennent en 16-81 ms ; le temoin est lui-meme
// quantifie au pas de 10 ms et apparie dans une fenetre de 150 ms (`deathMatchWindowMS`).
// Une seconde est donc un ordre de grandeur au-dessus du bruit des deux methodes, et bien
// en deca du plus petit ecart qu'une erreur de referentiel produirait (3,6 s).
const originControlToleranceMS = 1000

// originControlMinMatches est le nombre minimal de morts appariees pour qu'un temoin ait
// valeur de contradiction. En deca, un desaccord dit seulement que le fil des morts est
// pauvre — il ne dit rien de la lecture, et faire taire l'origine sur cette base perdrait
// l'information sur les films les plus fragiles.
const originControlMinMatches = 5

// ScanFilmClockOrigin rend l'horodatage moteur du PREMIER PAQUET du film, c'est-a-dire le
// zero de l'horloge sur laquelle les highlight events sont dates.
//
// HORS LIGNE (I/O disque) — jamais depuis un chemin de requete.
func ScanFilmClockOrigin(filmDir string) (uint64, error) {
	raw, err := filmdec.ReadFilmChunk(filmDir, 1)
	if err != nil {
		return 0, fmt.Errorf("chunk 1 (origine d'horloge) : %w", err)
	}
	packets := filmdec.WalkPackets(raw)
	if len(packets) == 0 {
		return 0, fmt.Errorf("chunk 1 (origine d'horloge) : aucun paquet lisible dans %s", filmDir)
	}
	return packets[0].TimestampUS, nil
}

// resolveOriginMs rend l'origine de la frame 0 sur l'horloge du fil, ou nil quand elle
// n'est pas etablie.
//
// firstPosUS est l'horodatage du premier paquet de position (l'origine des frames) ;
// filmClockUS celui du premier paquet du film. Le temoin (deathOffsetMS, matched) vient de
// `bestDeathOffset` : `horlogeFilm = horlogeFil + deathOffsetMS`, d'ou une seconde
// expression de la meme origine, `firstPosUS/1000 − deathOffsetMS`.
func resolveOriginMs(firstPosUS, filmClockUS uint64, deathOffsetMS int64, matched int) *int64 {
	if filmClockUS == 0 || firstPosUS < filmClockUS {
		slog.Warn("rejeu : origine d'horloge non etablie — le client retombera sur l'appariement",
			"premierPaquetPositionUS", firstPosUS, "premierPaquetFilmUS", filmClockUS)
		return nil
	}
	read := int64(firstPosUS-filmClockUS) / 1000
	if matched >= originControlMinMatches {
		control := int64(firstPosUS)/1000 - deathOffsetMS
		if ecart := control - read; ecart > originControlToleranceMS || ecart < -originControlToleranceMS {
			slog.Warn("rejeu : origine LUE contredite par le fil des morts — aucune origine publiee",
				"origineLueMs", read, "origineTemoinMs", control, "ecartMs", ecart, "mortsAppariees", matched)
			return nil
		}
	}
	return &read
}

// originMSOf rend l'origine de la frame 0 en millisecondes, ou 0 quand elle n'est pas etablie.
//
// ZERO N'EST PAS UNE ORIGINE NEUTRE, C'EST UN REPLI, et il se journalise. Les calques dates sur
// l'horloge du FILM (actions d'objectif, courbe de score) se posent sur la grille de frames par
// soustraction de cette origine ; sans elle, ils restent decales de l'ecart reel entre le
// premier paquet du film et le premier paquet de position — 3,6 s a 50,8 s selon le match.
// Le REFUS d'origine est deja journalise par resolveOriginMs ; ce qui se journalise ici est sa
// CONSEQUENCE sur les calques, qui n'est pas la meme information.
func originMSOf(origin *int64, matchID string) int {
	if origin == nil {
		slog.Warn("rejeu : aucune origine etablie — calques dates sur l'horloge du film NON recales",
			"match_id", matchID)
		return 0
	}
	return int(*origin)
}
