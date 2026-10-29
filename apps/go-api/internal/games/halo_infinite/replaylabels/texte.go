package replaylabels

import (
	"levelup/go-api/internal/i18n"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

// texte.go — LE PASSAGE D'UN LIBELLÉ PAR LANGUE ENTRE LEVELUP ET LE DÉCODEUR DE FILM (plan D2,
// item 2.9 ; ADR 0040).
//
// Le décodeur transporte ses libellés dans son propre type (`replay.Text`, code de langue ->
// libellé) et ne connaît aucune liste de langues : il écrit sur le fil les langues que porte le
// texte. LevelUp, qui tient la liste (`config/locales.toml`), fait les deux conversions ICI, et nulle
// part ailleurs : tout producteur d'un libellé du rejeu passe par TexteVersFilm, tout lecteur d'un
// libellé du rejeu par TexteDepuisFilm. Le fil qui en résulte est, octet pour octet, celui de
// `i18n.MarshalFlat` : `fil_jumeau_test.go` le prouve pour les cinq dispositions du document.

// TexteVersFilm convertit un libellé de LevelUp en libellé du rejeu : CHAQUE langue de la liste de
// l'application, vide comprise quand le texte ne la porte pas, et aucune autre. C'est ce que
// `i18n.MarshalFlat` écrivait (toutes les langues de la liste pour une disposition sans
// `OmitEmpty`, aucune clé hors liste) : le décodeur, qui écrit les clés du texte, rend donc le même
// fil. Un texte nil donne toutes les langues vides, comme l'original l'écrivait.
func TexteVersFilm(t i18n.Text) replay.Text {
	codes := i18n.FlatCodes()
	out := make(replay.Text, len(codes))
	for _, code := range codes {
		out[string(code)] = t[code]
	}
	return out
}

// TexteDepuisFilm convertit un libellé du rejeu en libellé de LevelUp : les langues de la liste de
// l'application que le texte porte (une valeur vide reste une langue « connue, vide »), et aucune
// autre. C'est ce que `i18n.UnmarshalFlat` retenait d'un fil : la lecture de LevelUp est inchangée.
// Un texte sans aucune langue de la liste donne nil.
func TexteDepuisFilm(t replay.Text) i18n.Text {
	var out i18n.Text
	for _, code := range i18n.FlatCodes() {
		if v, ok := t[string(code)]; ok {
			out.Set(code, v)
		}
	}
	return out
}
