package main

// cmd_backfill_killsource_test.go — LA VALIDATION DES DRAPEAUX, ET SEULEMENT ELLE.
//
// POURQUOI CE FICHIER EXISTE. Aucun test de `cmd/levelup` ne mentionnait `killsource` : la
// validation croisee `--online` / `--gamertag` n etait tenue par rien. Inverser sa condition
// aurait fait echouer TOUTE passe en ligne correctement invoquee, avec un message trompeur ;
// la supprimer aurait fait partir la passe sans tokens jusqu apres la lecture complete du
// registre.
//
// Ce que ce test NE fait pas : ouvrir une base, joindre le reseau, ou jouer une passe. Il
// s arrete a la porte — les erreurs verifiees ici sont toutes rendues AVANT le premier
// `os.Stat` sur la base partagee.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/config"
)

func TestBackfillKillSource_ValidationDesDrapeaux(t *testing.T) {
	// RepoRoot volontairement inexistant : si la validation laisse passer, l erreur suivante
	// portera sur `shared_matches introuvable` — ce que les cas ci-dessous refusent
	// explicitement, et c est ainsi qu ils distinguent « refuse a la porte » de « refuse plus
	// loin ».
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}

	cas := []struct {
		nom       string
		args      []string
		fragment  string
		pourquoiC string
	}{
		{
			nom:       "--online sans --gamertag",
			args:      []string{"--online", "--dry-run"},
			fragment:  "--online exige --gamertag",
			pourquoiC: "sans tokens la passe partirait et echouerait apres avoir lu tout le registre",
		},
		{
			nom:       "--gamertag sans --online",
			args:      []string{"--gamertag", "JGtm", "--dry-run"},
			fragment:  "n a de sens qu avec --online",
			pourquoiC: "la passe hors ligne n emet aucune requete : accepter le drapeau ferait croire l inverse",
		},
		{
			nom:       "--films-only et --credit-only",
			args:      []string{"--films-only", "--credit-only"},
			fragment:  "s excluent",
			pourquoiC: "les deux drapeaux se contredisent ; en silence, l un des deux serait ignore",
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			err := runBackfillKillSource(cfg, c.args)
			if err == nil {
				t.Fatalf("aucune erreur — %s", c.pourquoiC)
			}
			if !strings.Contains(err.Error(), c.fragment) {
				t.Errorf("erreur = %q, attendu un message contenant %q (la validation a laisse "+
					"passer et l echec vient d ailleurs)", err.Error(), c.fragment)
			}
		})
	}
}

// TestBackfillKillSource_CombinaisonValideDepasseLaPorte : le pendant POSITIF — sans lui, une
// validation inversee (`o.online && o.gamertag != ""`) passerait les cas ci-dessus tout en
// refusant toute invocation correcte.
func TestBackfillKillSource_CombinaisonValideDepasseLaPorte(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	err := runBackfillKillSource(cfg, []string{"--online", "--gamertag", "JGtm", "--dry-run"})
	if err == nil {
		t.Fatal("erreur attendue : la base partagee n existe pas sous ce repoRoot")
	}
	if strings.Contains(err.Error(), "--online") || strings.Contains(err.Error(), "--gamertag") {
		t.Errorf("une invocation VALIDE est refusee a la porte : %v", err)
	}
	if !strings.Contains(err.Error(), "shared_matches introuvable") {
		t.Errorf("erreur = %q, attendu l echec d ouverture de la base (donc porte franchie)", err.Error())
	}
}
