package filmdec

// player_table_profile.go — LA LARGEUR DU BLOC DE PERSONNALISATION, PAR BUILD (lot 1.5.2).
//
// # POURQUOI CE FICHIER EST UNE DONNEE DE PROFIL, ET PAS UNE MESURE
//
// D-3 d'ADR 0034 : le profil est resolu une fois, immuable, et sa cle est le BUILD lu en clair
// dans la section 2 de `chunk_00`. Une largeur par build est une donnee de profil — elle ne
// s'auto-detecte pas sur le film. Le calibrage lu sur le film reste possible et il reste
// PRECIEUX, mais comme CONTROLE : `PlayerTableReport` compte les accords et les contradictions
// entre ce que le profil dit et ce que le film mesure (player_table.go). Un calibrage qui
// decide serait exactement l'inference que D-3 interdit.
//
// D-4 : un build absent de cette table donne [ErrUnknownBuild] et un compteur expvar par build.
// Il n'est JAMAIS lu au profil du build le plus proche. Les 5 films du cache sans section
// d'identification se comportent comme `HI_1_4_1` (transposition +1 600 bits mesuree sur les
// cinq) — se comporter comme n'est pas etre, et ils sont donc mis de cote.
//
// # LA MESURE QUI DONNE CES QUATRE VALEURS
//
// L'ordre d'ecriture de `FUN_1407edea8` place TOUS les champs de longueur variable AVANT le
// gamertag et tous les champs de largeur fixe APRES. Le bloc de queue `sub+0x1400` porte le
// MEME gamertag en UTF-16 petit-boutiste : il est donc localisable dans le flux sans rien
// supposer de ce qui le precede, ce qui coupe l'enregistrement en deux moities mesurables
// separement (`residus_slots_research_test.go`, `NOTE_RESIDUS_CHUNK00_2026-09-13.md` section 2).
// Resultat, sur un film par build : la moitie APRES est invariante sur les sept builds, la plus
// longue suite de bits NULS de la moitie AVANT vaut exactement le bloc de personnalisation, et
// le RESTE hors bloc nul vaut **270 bits sur les huit groupes** — c'est-a-dire qu'AUCUN autre
// champ n'a change de largeur. La transposition par build est ce bloc, et rien d'autre.

import (
	"fmt"
	"strings"
	"unicode"
)

// ErrUnknownBuild : le build du film n'est pas dans la table de profil ci-dessous.
//
// L'erreur rendue par [ReadPlayerTable] enveloppe cette sentinelle avec le nom du build, pour
// qu'un journal dise LEQUEL : `errors.Is(err, ErrUnknownBuild)` reste vrai.
const ErrUnknownBuild = chunk00Error("filmdec: build absent de la table de profil")

// persoOctetsRef : la largeur du bloc `sub+0xcc0` sur le build de l'executable desassemble.
// Elle sert de REFERENCE a la transposition : un build dont le bloc est plus court decale
// l'enregistrement d'autant, bit pour bit (cf. persoDeltaBits).
const persoOctetsRef = 1852

// personnalisationOctets rend la largeur, en octets, du bloc de personnalisation `sub+0xcc0`
// pour un build donne. UN COMMENTAIRE DE PROVENANCE PAR LIGNE : une valeur sans preuve n'est
// pas une valeur de profil (D-3, regle 1).
func personnalisationOctets(build string) (int, bool) {
	switch build {
	case "HI_1_13_0":
		// 1 852 = 0x73C, LU CHEZ L'ECRIVAIN : `FUN_1407edea8` ecrit `LEA R8,[RDI+0xcc0] ;
		// R9D=0x39e0` (14 816 bits). Fermeture independante chez le serialiseur DELTA
		// `FUN_1407ec27c`, qui ecrit la MEME structure champ par champ : `actionPose` a
		// `+0x738`, plus 4 = `0x73C`. Ghidra, 2026-09-12 — c'est le build de l'exe desassemble.
		return 1852, true
	case "HI_1_12_0":
		// 1 852 aussi, et c'est MESURE, pas suppose : transposition `mesure - predit` de 0 bit,
		// valeur unique sur les 146 films `HI_1_12_0` du cache (2026-09-14, oracle
		// `TestResidusSlotChaineCorpus`). La bascule du bloc se fait entre `HI_1_11_0` et
		// `HI_1_12_0`, au meme endroit que celle de `n2` dans la trame.
		return 1852, true
	case "HI_1_11_0":
		// 1 492 = 1 852 - 360. Transposition mesuree -2 880 bits (= -360 octets), valeur unique
		// sur les 39 films `HI_1_11_0` du cache ; la coupe autour du bloc de queue sur
		// `00ba2e1c` place ces 360 octets DANS le bloc nul, RESTE hors bloc nul inchange a
		// 270 bits. 360 = 10 x 0x24, le pas d'une attache d'armure (`FUN_1407ebf44`) — appui,
		// pas preuve : sans executable de ce build, la CAUSE reste une hypothese, la LARGEUR
		// est une mesure.
		return 1492, true
	case "HI_1_10_0":
		// 1 492, transposition -2 880 bits, valeur unique sur les 26 films `HI_1_10_0` du
		// cache ; coupe verifiee sur `084a804d` (2026-09-13, note section 2.1).
		return 1492, true
	case "HI_1_9_0":
		// 1 312 = 1 852 - 540. Transposition -4 320 bits, valeur unique sur l'unique film
		// `HI_1_9_0` du cache (`11de8353`), coupe verifiee sur ce film. 540 = 15 x 0x24.
		return 1312, true
	case "HI_1_8_0":
		// 1 312, transposition -4 320 bits, valeur unique sur les 10 films `HI_1_8_0` du
		// cache ; coupe verifiee sur `0a247154`.
		return 1312, true
	case "HI_1_4_1":
		// 2 052 = 1 852 + 200. Transposition +1 600 bits, valeur unique sur l'unique film
		// `HI_1_4_1` du cache (`a521164d`), coupe verifiee sur ce film. CE +200 NE FERME SUR
		// AUCUN PAS CONNU (ni 0x24 = 36, ni 0x58 = 88) : la largeur est mesuree, sa cause n'est
		// pas etablie, et elle est au registre des reports du plan.
		return 2052, true
	}
	return 0, false
}

// persoDeltaBits rend la transposition d'un enregistrement de slot pour une largeur de bloc de
// personnalisation donnee : la difference avec la reference, en bits. C'est la SEULE constante
// par laquelle la grammaire du slot se transpose d'un build a l'autre.
func persoDeltaBits(octets int) int { return (octets - persoOctetsRef) * 8 }

// unknownBuildCounterName rend le nom expvar du compteur de build inconnu.
//
// NOMMAGE (ADR 0009) : `<categorie>_<sous_cle>` en snake_case. Le plan ecrit
// `filmdec.unknown_build.<build>` ; la forme physique du depot est
// `filmdec_unknown_build_<build>`, parce que c'est la convention que tout `/debug/vars` de ce
// service suit deja (`killsource_*`, `replay_artifact_*`, `filmdec_keyframe_ti12_*`).
//
// Le build entre dans le nom EN MINUSCULES et purge de tout ce qui n'est ni lettre, ni chiffre,
// ni `_` : un nom de build vient du film, donc d'une source que ce processus ne controle pas, et
// un nom de compteur bati sur une chaine arbitraire salirait `/debug/vars`. Un build VIDE (film
// sans section d'identification) se compte sous `sans_section` — il y en a 5 au cache.
func unknownBuildCounterName(build string) string {
	nettoye := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case unicode.IsUpper(r) && r < unicode.MaxASCII:
			return unicode.ToLower(r)
		default:
			return '_'
		}
	}, build)
	if nettoye == "" {
		nettoye = "sans_section"
	}
	return "filmdec_unknown_build_" + nettoye
}

// UnknownBuildExpvarPairs rend le compteur a publier quand [ReadPlayerTable] refuse un film pour
// build inconnu (D-4 d'ADR 0034 : erreur typee ET compteur, le film est mis de cote).
//
// `filmdec` NOMME ses compteurs et ne depend PAS d'`internal/observability` : meme patron que
// `KillSourceHealth.ExpvarPairs` (cable par `killcollector`) et que
// `NavpointRadialScan.KeyframeExpvarPairs` (cable par `replay.decodeFilmBombReads`). C'est ce
// qui garde le decodeur sans dependance interne et cette fonction testable sans expvar.
//
// LE CABLEUR N'EXISTE PAS ENCORE, ET C'EST DATE : le lot 1.5 ne livre que des LECTEURS, sans
// consommateur ; le premier appelant de production de [ReadPlayerTable] est le registre
// d'identite du lot 1.6, et c'est lui qui publiera cette paire. Si ce lot passe sans qu'elle
// soit cablee, c'est un defaut de 1.6, pas un compteur « au cas ou » : le refus d'un film serait
// alors invisible en production, ce que D-4 interdit.
func UnknownBuildExpvarPairs(build string) []ExpvarPair {
	return []ExpvarPair{{Name: unknownBuildCounterName(build), Value: 1}}
}

// erreurBuildInconnu enveloppe [ErrUnknownBuild] avec le nom du build refuse.
func erreurBuildInconnu(build string) error {
	return fmt.Errorf("%w : %q", ErrUnknownBuild, build)
}
