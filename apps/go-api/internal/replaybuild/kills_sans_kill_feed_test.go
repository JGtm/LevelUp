package replaybuild

// kills_sans_kill_feed_test.go — UN FILM SANS KILL-FEED DONNE QUAND MEME SES BOTS A LA CUISSON.
//
// `decfilm.Decode` rend `ErrNoKillFeed` pour un match sans aucun frag ; le resultat nil d avant ne
// portait aucun bot, et les vies des bots que BOT_METADATA declare restaient sans identite. Le film
// est ici un seul paquet BOT_METADATA (type 12) en memoire : un bot, au slot 8.

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/replayidentity"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
)

// filmUnBotDeclare fabrique un film dont l unique paquet est un BOT_METADATA declarant `nom` au `slot`,
// de `bid` : `nbBots` en tete, puis l entree (slot et bid en gros-boutiste, nom en UTF-16BE a 0x74 du
// slot), sous l en-tete de paquet de 16 octets.
func filmUnBotDeclare(t *testing.T, slot, bid int, nom string) *decfilm.Film {
	t.Helper()
	pl := make([]byte, 4+2076+256)
	binary.BigEndian.PutUint32(pl, 1)
	binary.BigEndian.PutUint32(pl[4:], uint32(slot)) //nolint:gosec // slot de test borne
	binary.BigEndian.PutUint32(pl[8:], uint32(bid))  //nolint:gosec // bid de test borne
	for i, c := range []byte(nom) {
		pl[4+0x74+2*i+1] = c
	}
	chunk := make([]byte, 16+len(pl))
	binary.LittleEndian.PutUint16(chunk, 12)                  // BOT_METADATA
	binary.LittleEndian.PutUint32(chunk[4:], uint32(len(pl))) //nolint:gosec // payload de test borne
	binary.LittleEndian.PutUint64(chunk[8:], 5_306_466_741)
	copy(chunk[16:], pl)
	film, err := decfilm.Load(decfilm.MemoryChunks([][]byte{chunk}), nil)
	if err != nil {
		t.Fatalf("film synthetique : %v", err)
	}
	return film
}

func TestBotsSansKillFeedAlimententLeRegistreDIdentite(t *testing.T) {
	film := filmUnBotDeclare(t, 8, 59, "343 Hollis")
	refus := fmt.Errorf("killsource %s: %w", "5da6fd30", decfilm.ErrNoKillFeed)

	res := botsSansKillFeed(context.Background(), "5da6fd30", film, refus)
	if res == nil {
		t.Fatal("resultat nil : un bot est declare, le refus est celui du kill-feed absent")
	}
	if res.LineByLinePublishable() {
		t.Error("le resultat minimal ne porte ni mort ni bijection : l attribution ligne par ligne reste fermee")
	}
	bots := replayidentity.BotIdentities(res)
	if len(bots) != 1 || bots[0].FilmIndex != 8 || bots[0].BotID != 59 || !bots[0].BotIDLu ||
		bots[0].Name != "343 Hollis"+decfilm.BotSuffix {
		t.Fatalf("identites de bot %+v : attendu le seul bot 59 du slot 8, nomme par BOT_METADATA", bots)
	}
	if p := profilDeBalayageDeLaCuisson(res); p == nil || !reflect.DeepEqual(*p, decfilm.ProfilDeDepart()) {
		t.Errorf("profil %+v : le resultat minimal porte le profil de depart, comme le resultat nil d avant", p)
	}
}

func TestBotsSansKillFeedNeRepondentQueAuKillFeedAbsent(t *testing.T) {
	film := filmUnBotDeclare(t, 8, 59, "343 Hollis")
	if res := botsSansKillFeed(context.Background(), "5da6fd30", film, errors.New("autre refus")); res != nil {
		t.Errorf("un autre refus de decodage rend %+v, attendu nil", res)
	}
	if res := botsSansKillFeed(context.Background(), "5da6fd30", nil, decfilm.ErrNoKillFeed); res != nil {
		t.Errorf("film nil : %+v, attendu nil", res)
	}
}
