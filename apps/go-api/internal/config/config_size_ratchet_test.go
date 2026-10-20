package config

// config_size_ratchet_test.go — gel de la taille de config.go (revue adversariale du lot B5,
// constat C5 ; lot B-C4 du backlog 2026-09-26).
//
// config.go dépasse le seuil de 500 lignes (règle n°5 de CLAUDE.md) : dette préexistante,
// gelée à 629 lignes, sa taille avant le lot B5. B5 l'avait portée à 640 ; les ajouts du
// mode démo vivent dans config_demo.go. Ce test interdit de l'accroître à nouveau : le seuil
// ne se relève pas, il ne peut que baisser.

import (
	"os"
	"strings"
	"testing"
)

const configGoMaxLines = 629

func TestConfigGo_TailleGelee(t *testing.T) {
	data, err := os.ReadFile("config.go")
	if err != nil {
		t.Fatalf("lecture de config.go : %v", err)
	}
	n := strings.Count(string(data), "\n")
	if n > configGoMaxLines {
		t.Errorf("config.go fait %d lignes, gel à %d (règle n°5, dette préexistante) : "+
			"les ajouts vont dans un fichier dédié (ex. config_demo.go)", n, configGoMaxLines)
	}
}
