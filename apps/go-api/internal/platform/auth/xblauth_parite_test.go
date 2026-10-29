// Package auth — xblauth_parite_test.go : parité entre les types de la brique `haloapi/xblauth`
// et ceux de LevelUp qu'ils reflètent (plan D3, lot 2). Le test vit ici parce que c'est le seul
// paquet qui importe les deux côtés (la brique n'a pas le droit d'importer `domain`).
package auth

import (
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/xblauth"
)

// champsDe rend, dans l'ordre, « nom type `tag` » de chaque champ d'une struct.
func champsDe(t reflect.Type) []string {
	out := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		out = append(out, fmt.Sprintf("%s %s `%s`", f.Name, f.Type, f.Tag))
	}
	return out
}

// TestXblauthTokens_PariteAvecDomainHaloTokens : `xblauth.Tokens` porte exactement les champs de
// `domain.HaloTokens` (noms, types, étiquettes, ordre). Un champ ajouté d'un seul côté serait perdu
// à la conversion sans que rien ne casse.
func TestXblauthTokens_PariteAvecDomainHaloTokens(t *testing.T) {
	brique := champsDe(reflect.TypeOf(xblauth.Tokens{}))
	levelup := champsDe(reflect.TypeOf(domain.HaloTokens{}))
	if !reflect.DeepEqual(brique, levelup) {
		t.Errorf("xblauth.Tokens = %v\ndomain.HaloTokens = %v\nles deux structures doivent rester identiques", brique, levelup)
	}
}

// TestXblauthFamilles_PariteAvecTokenFamily : les provenances de la brique ont les valeurs que le
// magasin persiste (`UserTokens.TokenClientFamily`).
func TestXblauthFamilles_PariteAvecTokenFamily(t *testing.T) {
	if xblauth.FamilleAzure != TokenFamilyAzure {
		t.Errorf("xblauth.FamilleAzure = %q, TokenFamilyAzure = %q", xblauth.FamilleAzure, TokenFamilyAzure)
	}
	if xblauth.FamilleXboxNative != TokenFamilyXboxNative {
		t.Errorf("xblauth.FamilleXboxNative = %q, TokenFamilyXboxNative = %q", xblauth.FamilleXboxNative, TokenFamilyXboxNative)
	}
}
