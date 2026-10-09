package killcollector

// origin_replication_test.go — LE VERROU D'ÉGALITÉ entre les origines de film nommées par la
// réplication (décodeur) et leurs chaînes partagées (`domain/killscope`). Même montage que
// category_headshot_test.go : ce paquet est le seul qui importe les deux.
//
// CE QUE LA DÉRIVE COÛTERAIT : le lecteur des paires tueur -> victime de la Vue match
// (`platform/duckdb`, Q20) écarte ces origines dans une passe non publiable. Une chaîne qui
// divergerait d'un caractère ne reconnaîtrait plus aucune ligne de bot : les identités de
// réplication d'une passe ambiguë redeviendraient lisibles ligne à ligne, sans erreur ni compteur.

import (
	"testing"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

func TestOriginesDeReplicationEgalesAuDecodeur(t *testing.T) {
	cas := []struct {
		nom, partage, decodeur string
	}{
		{"OriginFilmBotVictim", killscope.OriginFilmBotVictim, string(decfilm.OriginBot)},
		{"OriginFilmBotKiller", killscope.OriginFilmBotKiller, string(decfilm.OriginBotKiller)},
	}
	for _, c := range cas {
		if c.partage != c.decodeur {
			t.Errorf("killscope.%s = %q, decodeur = %q — le filtre des identites de replication "+
				"cesserait de reconnaitre ces lignes, sans erreur ni compteur", c.nom, c.partage, c.decodeur)
		}
	}
}
