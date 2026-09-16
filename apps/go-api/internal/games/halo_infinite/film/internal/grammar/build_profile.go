package grammar

// build_profile.go — CE QUI VA CHERCHER LE PROFIL D UN BUILD DANS LE FILM.
//
// LA TABLE N EST PLUS ICI DEPUIS LE LOT 2.5.b : les largeurs par build et par format, leurs
// provenances ligne a ligne et leurs deux erreurs sentinelles vivent dans `profile`
// (`profile/build_profile.go`), parce que ce sont des DONNEES (ADR 0034 D-1 : « the profile
// table: per build, per map. Data derived from the game. No read logic. »).
//
// CE QUI RESTE EST EXACTEMENT CE QUI LIT OU PUBLIE :
//
//	BuildProfileFromFilm    ouvre le `chunk_00` d un film deja charge, y lit la version de
//	                        format et la section 2, et demande son profil a la table.
//	InstallFilmFormatMPP    pose le decoupage MPP du format sur le CONTEXTE du film, et rend sa
//	                        restauration.
//	UnknownBuildExpvarPairs le compteur expvar d un build refuse (D-4 : erreur typee ET
//	                        compteur). Il NOMME un compteur du service, ce qui est une
//	                        responsabilite d observabilite et non une valeur de profil.
//
// D-4 : un build absent de la table donne [profile.ErrUnknownBuild] et un compteur expvar par
// build. Il n'est JAMAIS lu au profil du build le plus proche. Les 5 films du cache sans section
// d'identification se comportent comme `HI_1_4_1` (transposition +1 600 bits mesuree sur les
// cinq) — se comporter comme n'est pas etre, et ils sont donc mis de cote.

import (
	"strings"
	"unicode"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// unknownBuildCounterName rend le nom expvar du compteur de build inconnu.
//
// NOMMAGE (ADR 0009) : `<categorie>_<sous_cle>` en snake_case. Le plan ecrit
// `grammar.unknown_build.<build>` ; la forme physique du depot est
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

// pas etre `HI_1_4_1` : leur profil se declare au REGISTRE des replis (1.9.0), pas ici.
func BuildProfileFromFilm(f *source.Film) (profile.BuildProfile, error) {
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		return profile.BuildProfile{}, profile.ErreurBuildInconnu("")
	}
	format, _ := FilmFormatVersionFromHeader(reg)
	id, err := ReadFilmIdentity(reg)
	if err != nil {
		return profile.BuildProfile{}, profile.ErreurBuildInconnu("")
	}
	return profile.BuildProfileFor(id.Build, format)
}

// InstallFilmFormatMPP installe les largeurs MPP de la VERSION DE FORMAT du film et rend leur
// restauration.
//
// ELLE NE LIT PLUS LA SECTION D IDENTIFICATION (lot 1.9.1 ter) : le decoupage du bloc MPP est
// une donnee de FORMAT, et le format se lit a `chunk_00+4`. Les cinq films du cache sans
// section d identification (format 20) traversent donc desormais cette fonction au lieu d y
// buter sur un build vide — leur largeur reste INDETERMINEE, donc rien n est installe et aucun
// bit lu ne change : c est exactement le comportement qu ils avaient, avec une cause nommee.
//
// ELLE POSE SUR LE CONTEXTE, PLUS SUR LE PROCESSUS (lot 2.3) : le decoupage voyage avec les
// lecteurs que ce contexte construit. Un film dont le FORMAT est inconnu ne change RIEN — le
// profil garde son invariant et l erreur est rendue a l appelant, qui decide (mettre le film de
// cote, ou compter un repli nomme).
func InstallFilmFormatMPP(fc *FilmContext) (func(), error) {
	format, ok := FilmFormatVersion(fc.Film())
	if !ok {
		return func() {}, profile.ErreurFormatInconnu(FilmFormatVersionUnknown)
	}
	w, ok := profile.MPPPourFormat(format)
	if !ok {
		return func() {}, profile.ErreurFormatInconnu(format)
	}
	if !w.Valid() {
		// Format CONNU dont la largeur MPP est INDETERMINEE (les formats <= 25) : on n installe
		// rien plutot qu un decoupage nul, qui ne lirait aucune identite du tout.
		return func() {}, nil
	}
	prev := fc.PoserMPP(w)
	return func() { fc.PoserMPP(prev) }, nil
}
