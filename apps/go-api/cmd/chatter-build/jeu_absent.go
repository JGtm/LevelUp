//go:build !gamefiles

package main

import "errors"

// errSansTag : la commande est compilée sans l'accès au jeu.
var errSansTag = errors.New("cette étape lit l'installation du jeu : la compiler avec -tags=gamefiles " +
	"(CGO_ENABLED=1 go run -tags=gamefiles ./cmd/chatter-build <étape> ...)")

// racineDuJeu : sans le tag, aucune installation n'est cherchée.
func racineDuJeu(string) (string, error) { return "", errSansTag }

// ouvrirModule : sans le tag, aucun module n'est ouvert.
func ouvrirModule(string) (sourceModule, error) { return nil, errSansTag }
