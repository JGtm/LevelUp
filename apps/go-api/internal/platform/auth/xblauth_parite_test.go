// Package auth — xblauth_parite_test.go : parité entre les types de la brique `haloapi/xblauth`
// et ceux de LevelUp qu'ils reflètent (plan D3, lot 2). Le test vit ici parce que c'est le seul
// paquet qui importe les deux côtés (la brique n'a pas le droit d'importer `domain`).
package auth

import (
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
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

// TestDescripteurXBL_PariteAvecTitre : la façade passe à la brique exactement les valeurs du
// descripteur Halo par défaut que la chaîne lit (les tests de la brique recopient ces valeurs :
// `descripteurHaloTest`).
func TestDescripteurXBL_PariteAvecTitre(t *testing.T) {
	got := descripteurXBL(title.DefaultHaloAuthDescriptor())
	want := xblauth.Descriptor{
		XSTSAudience:    "https://prod.xsts.halowaypoint.com/",
		SpartanAudience: "urn:343:s3:services",
		SpartanTokenURL: "https://settings.svc.halowaypoint.com/spartan-token",
		ClearanceURL:    "https://settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/active",
	}
	if got != want {
		t.Errorf("descripteurXBL(défaut Halo) = %+v\nattendu %+v", got, want)
	}
}

// TestScopes_DescriptorDerivation_GoldenParity prouve que les scopes dérivés du
// descripteur (MT-02 leg 4) sont byte-identiques aux littéraux historiques, sous les deux
// formes : la liste (XboxScopes) et la forme jointe qu'envoie la brique (ClientAzure.ScopesParam,
// ex-variable `xboxScopes` de oauth_refresh.go).
func TestScopes_DescriptorDerivation_GoldenParity(t *testing.T) {
	if len(XboxScopes) != 2 || XboxScopes[0] != "Xboxlive.signin" || XboxScopes[1] != "Xboxlive.offline_access" {
		t.Errorf("XboxScopes = %v, want [Xboxlive.signin Xboxlive.offline_access]", XboxScopes)
	}
	if got := (AzureOAuthClient{}).versXBL().ScopesParam(); got != "Xboxlive.signin Xboxlive.offline_access" {
		t.Errorf("scopes joints = %q, want space-joined", got)
	}
}

// TestClientAzureXBL_Parite : le client que reçoit la brique porte l'identifiant, le secret à
// envoyer, l'autorité et les scopes de la résolution d'exploitation (aucune autre source).
func TestClientAzureXBL_Parite(t *testing.T) {
	c := AzureOAuthClient{ClientID: "id-test", rawSecret: "secret-test"}
	got := c.versXBL()
	if got.ID != "id-test" || got.Secret != c.SecretToSend() || got.Autorite != MSALAuthority {
		t.Errorf("versXBL = %v, attendu ID, secret à envoyer et autorité de la résolution", got)
	}
	if !reflect.DeepEqual(got.Scopes, XboxScopes) {
		t.Errorf("Scopes = %v, attendu XboxScopes %v", got.Scopes, XboxScopes)
	}
	if got.URLAutorisation() != MSALAuthority+"/oauth2/v2.0/authorize" {
		t.Errorf("URLAutorisation = %q", got.URLAutorisation())
	}
}
