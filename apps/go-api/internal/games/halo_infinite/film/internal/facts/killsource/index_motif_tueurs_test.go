package killsource

// index_motif_tueurs_test.go — LE MOTIF DU XUID CHERCHE AUSSI LES JOUEURS QUI TUENT SANS MOURIR
// (lot J7.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-3).
//
// Le lien par motif (`index_motif.go`) cherche les xuids que le FILM nomme : les sieges de
// `chunk_00` et le kill-feed. Du kill-feed, seul le xuid des MORTS etait retenu (`buildFeed`
// remplissait `xuidDe` sur l evenement `death` seulement) : un remplacant qui tue sans jamais
// mourir — donc absent de la table, ecrite a l ouverture — n etait cherche nulle part, et son
// indice restait a l inference.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : ne plus retenir le xuid sur l evenement `kill` dans
// [buildFeed].

import (
	"testing"

	"levelup/go-api/internal/domain/highlightevent"
)

func TestLeMotifChercheAussiLesTueursQuiNeMeurentPas(t *testing.T) {
	kf := buildFeed([]highlightevent.HighlightEvent{
		{EventType: highlightevent.EventTypeKill, XUID: 111, Gamertag: "Remplacant", TimeMS: 1000},
		{EventType: highlightevent.EventTypeDeath, XUID: 222, Gamertag: "Victime", TimeMS: 1000},
	})
	nom, xuids := xuidsNommesParLeFilm(nil, kf)
	if nom[111] != "Remplacant" {
		t.Errorf("xuid 111 -> %q, attendu \"Remplacant\" : un joueur qui tue sans mourir n est pas "+
			"cherche au motif", nom[111])
	}
	if nom[222] != "Victime" {
		t.Errorf("xuid 222 -> %q, attendu \"Victime\"", nom[222])
	}
	if len(xuids) != 2 || xuids[0] != 111 || xuids[1] != 222 {
		t.Errorf("xuids cherches = %v, attendu [111 222] (ordre stable)", xuids)
	}
}
