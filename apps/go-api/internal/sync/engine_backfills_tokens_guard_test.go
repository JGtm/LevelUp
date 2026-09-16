package sync

import (
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// Revue adversariale du 2026-09-16 (P0) : un moteur construit par le pool porte des tokens
// vides et un client personnalisé ; la garde « tokens Halo absents » de RunBackfillCSR /
// RunBackfillSharedCSR le rejetait AVANT de regarder le client, cassant les passes CSR pour
// tous les joueurs. Ce test échoue si la garde redevient aveugle au client personnalisé.
func TestRequireTokensUnlessCustomClient_ClientPooleSuffit(t *testing.T) {
	e := &SyncEngine{tokens: &domain.HaloTokens{}, customClient: &mockHaloClient{}}
	if err := e.requireTokensUnlessCustomClient("RunBackfillCSR"); err != nil {
		t.Fatalf("client personnalisé posé, tokens vides : attendu nil, obtenu %v", err)
	}
}

func TestRequireTokensUnlessCustomClient_SansRienRefuse(t *testing.T) {
	for name, tokens := range map[string]*domain.HaloTokens{"nil": nil, "vides": {}} {
		e := &SyncEngine{tokens: tokens}
		err := e.requireTokensUnlessCustomClient("RunBackfillCSR")
		if err == nil {
			t.Fatalf("[%s] aucun client ni token : attendu une erreur", name)
		}
		if strings.Contains(err.Error(), "re-login") {
			t.Fatalf("[%s] le message ne doit pas pousser à une re-capture (ADR 0023) : %q", name, err)
		}
	}
}

func TestRequireTokensUnlessCustomClient_TokensDuJoueurSuffisent(t *testing.T) {
	e := &SyncEngine{tokens: &domain.HaloTokens{SpartanToken: "spartan"}}
	if err := e.requireTokensUnlessCustomClient("RunBackfillSharedCSR"); err != nil {
		t.Fatalf("tokens du joueur présents : attendu nil, obtenu %v", err)
	}
}
