package main

import (
	"context"
	"testing"

	"levelup/go-api/internal/i18n"
)

func TestChaqueLangueRequiseADesVoixEtDesTextesDuJeu(t *testing.T) {
	langues, err := languesDeLApp()
	if err != nil {
		t.Fatal(err)
	}
	if len(langues) != len(i18n.Locales()) {
		t.Errorf("%d langue(s) du jeu pour %d langue(s) de l'application", len(langues), len(i18n.Locales()))
	}
	vus := map[int]bool{}
	for _, l := range langues {
		if vus[l.Jeu.Blob] {
			t.Errorf("deux langues sur le même index de blob %d", l.Jeu.Blob)
		}
		vus[l.Jeu.Blob] = true
		if l.Jeu.Dossier == "" {
			t.Errorf("langue %s sans dossier de voix", l.Code)
		}
	}
}

func TestLangueDuJeuDeNeDevinePas(t *testing.T) {
	if _, ok := langueDuJeuDe([]string{"xx-XX"}); ok {
		t.Error("une étiquette inconnue ne donne aucune langue du jeu")
	}
	if j, ok := langueDuJeuDe([]string{"xx-XX", "fr-FR"}); !ok || j.Blob != 3 {
		t.Errorf("langue = %+v : la première étiquette connue l'emporte", j)
	}
}

func TestExecuterRefuseUneEtapeInconnue(t *testing.T) {
	if err := executer(context.Background(), nil, journalDeTest(t)); err == nil {
		t.Error("sans étape, la commande doit dire lesquelles existent")
	}
	if err := executer(context.Background(), []string{"inconnue"}, journalDeTest(t)); err == nil {
		t.Error("une étape inconnue doit être refusée")
	}
}
