package killsource

// neutral_death_research_test.go — INSTRUMENT DE MESURE (lot 7.1 du rejeu 2D, 2026-08-14).
//
// # LA QUESTION
//
// Le fil du rejeu 2D affiche une LIGNE NEUTRE pour les morts qu'aucun kill ne revendique
// (suicide, chute, sortie de zone). L'utilisateur demande d'y porter l'ICONE DU TYPE DE MORT.
// La donnee qui distinguerait ces types est la NATURE de la source du degat fatal
// (`damagetag.Class` : ARME / MELEE / GRENADE / VEHICULE / OBJET_EXPLOSIF / DEGAT_GLOBAL).
//
// AVANT DE CABLER QUOI QUE CE SOIT : COMBIEN DE CES MORTS PORTENT UNE NATURE ? Le guide
// killsource previent que la population est RARE (0 sur les quatre films de reference, 1 sur le
// BTB) et que sur le BTB « la population ne ferme pas » (7 orphelines, 0 appariee, 0 publiee).
// Ce fichier CHIFFRE, film par film, au lieu de supposer.
//
// # CE QU'IL MESURE, ET SUR QUELLE POPULATION
//
// La population est `feed.orphD` : les morts du kill-feed que la reconstruction de couples n'a
// JAMAIS consommees — le feed porte la MORT, il ne porte AUCUN kill en face. C'est exactement ce
// que le client appelle une « mort neutre », et c'est la population que `resolveBotKillerDeaths`
// consulte deja au temps 5 (elle y sert a nommer un TUEUR BOT, jamais a nommer une source).
//
// Pour chacune, l'instrument cherche les candidats dead-state dans la fenetre ±tolMS dont
// l'INDICE VICTIME rend le gamertag mort, et publie leur tag, leur nature, leur statut, ainsi
// que l'indice tueur (egal a la victime = mort auto-infligee).
//
// # LECTURE SEULE, AUCUNE ECRITURE, AUCUNE BASE
//
//	NEUTRAL_DEATH_FILMS="<dir1>,<dir2>" go test ./internal/games/halo_infinite/film/killsource/ \
//	    -run TestNeutralDeathNatureCoverage -timeout 30m -v
//
// Sans la variable : SKIP (la CI n'a pas les films, ils ne sont pas versionnes — meme regime que
// [TestGoldenFilms]).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

const neutralDeathFilmsEnv = "NEUTRAL_DEATH_FILMS"

// TestNeutralDeathNatureCoverage : la couverture des morts sans tueur par la nature du degat.
func TestNeutralDeathNatureCoverage(t *testing.T) {
	spec := os.Getenv(neutralDeathFilmsEnv)
	if spec == "" {
		t.Skipf("mesure non demandee : %s vide", neutralDeathFilmsEnv)
	}
	for _, item := range strings.Split(spec, ",") {
		dir := strings.TrimSpace(item)
		if dir == "" {
			continue
		}
		t.Run(filepath.Base(dir), func(t *testing.T) {
			var b strings.Builder
			neutralDeathReport(t, &b, dir)
			t.Logf("\n%s", b.String())
		})
	}
}

func neutralDeathReport(t *testing.T, b *strings.Builder, dir string) {
	t.Helper()
	src, err := DirChunks(dir)
	if err != nil {
		t.Fatalf("chunks %s : %v", dir, err)
	}
	// LE CHEMIN PUBLIC, celui que le producteur d'artefacts consomme : mesurer par une voie
	// interne mesurerait autre chose que ce qui est servi.
	res, err := Decode(context.Background(), filepath.Base(dir), src, nil)
	if err != nil {
		t.Fatalf("Decode %s : %v", dir, err)
	}
	fmt.Fprintf(b, "kill-feed : %d kills, %d morts ; couples reels %d ; "+
		"morts de bot %d, morts PAR un bot %d\n",
		res.Coverage.FeedKills, res.Coverage.FeedDeaths, res.Coverage.RealPairs,
		res.Coverage.BotDeaths, res.Coverage.BotKillerDeaths)
	fmt.Fprintf(b, "morts orphelines du feed : %d ; expliquees par un tueur BOT %d ; "+
		"SANS REVENDICATION publiees %d ; publiable ligne par ligne = %v\n",
		res.Stats.Unclaimed.Population, res.Stats.BotKiller.Published,
		res.Stats.Unclaimed.Published, res.LineByLinePublishable())
	for _, d := range res.UnclaimedDeaths {
		lab, known := damagetag.Lookup(d.Source.Tag)
		fmt.Fprintf(b, "  %s  %-20s xuid=%d  tag=%08x connu=%v  nature=%-14s statut=%-12s "+
			"nom=%-24q affichage=%-24q voie=%s\n",
			msClock(d.TimeMS), d.Victim, d.VictimXUID, d.Source.Tag, known, d.Source.Class,
			d.Source.Status, lab.Name, d.Source.Display, d.Read.Path)
	}
	inexplique := res.Stats.Unclaimed.Population - res.Stats.BotKiller.Published - res.Stats.Unclaimed.Published
	fmt.Fprintf(b, "BILAN : %d/%d morts orphelines publiees sans revendication, %d inexpliquees\n",
		res.Stats.Unclaimed.Published, res.Stats.Unclaimed.Population, inexplique)
}

func msClock(ms int) string { return fmt.Sprintf("%02d:%02d", ms/60000, (ms/1000)%60) }
