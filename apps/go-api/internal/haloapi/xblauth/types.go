package xblauth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/haloapi"
)

// Tokens : les jetons Halo que rend la chaîne d'échange. Mêmes champs que `domain.HaloTokens`
// (parité tenue par un test de réflexion dans `platform/auth`, qui convertit).
type Tokens struct {
	SpartanToken     string
	ClearanceToken   string
	SpartanExpiresAt time.Time
}

// Descriptor : les paramètres de titre que la chaîne d'échange lit (audience XSTS, audience et
// adresse du Spartan, adresse de la Clearance). Une ClearanceURL vide saute la jambe Clearance
// (titre sans clearance).
type Descriptor struct {
	XSTSAudience    string
	SpartanAudience string
	SpartanTokenURL string
	ClearanceURL    string
}

// ClientAzure : le client OAuth v2 de l'application Azure (rafraîchissement, code
// d'autorisation, adresse d'autorisation). Résolu par l'appelant ; la brique ne lit aucune
// variable d'environnement.
type ClientAzure struct {
	// ID : client_id de l'application.
	ID string
	// Secret : client_secret à envoyer ("" = flux client public). Jamais journalisé : [ClientAzure.String] et
	// [ClientAzure.LogValue] le masquent.
	Secret string
	// Autorite : base des adresses OAuth v2, sans barre finale (ex. « .../consumers »).
	Autorite string
	// Scopes : scopes demandés ; joints par un espace dans les formulaires.
	Scopes []string
}

// ScopesParam rend les scopes joints par un espace, forme des corps form-encoded.
func (c ClientAzure) ScopesParam() string { return strings.Join(c.Scopes, " ") }

// URLAutorisation rend l'adresse /authorize de l'autorité.
func (c ClientAzure) URLAutorisation() string { return c.Autorite + "/oauth2/v2.0/authorize" }

// String masque le secret (un `%v` ou `%+v` ne le révèle jamais).
func (c ClientAzure) String() string {
	return "ClientAzure{ID:" + c.ID + " Autorite:" + c.Autorite + " Secret:" + masque(c.Secret) + "}"
}

// LogValue masque le secret dans un journal `slog`.
func (c ClientAzure) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", c.ID),
		slog.String("autorite", c.Autorite),
		slog.String("secret", masque(c.Secret)),
	)
}

func masque(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}

// Provenances d'un jeton d'accès Microsoft : le client OAuth qui l'a émis, qui décide du préfixe
// RpsTicket présenté à user.auth.xboxlive.com. Mêmes valeurs que `platform/auth.TokenFamily*`
// (parité tenue par un test dans `platform/auth`).
const (
	// FamilleAzure : application Azure (SSO web, rafraîchissement v2) -> « d= ».
	FamilleAzure = "azure"
	// FamilleXboxNative : client Xbox natif (device-flow, rafraîchissement MSA) -> « t= ».
	FamilleXboxNative = "xbox_native"
)

// Chaine : ce que la chaîne XBL reçoit de l'appelant pour un échange.
type Chaine struct {
	// Doer émet les requêtes (obligatoire).
	Doer haloapi.Doer
	// Creneau borne la concurrence vers user.auth.xboxlive.com (l'endpoint limite les requêtes
	// SIMULTANÉES) ; il rend la fonction de libération. nil = aucune borne.
	Creneau func(ctx context.Context) (liberer func(), err error)
	// Famille : provenance connue du jeton d'accès ("" = inconnue : « d= » puis « t= »).
	Famille string
	// Observe reçoit la provenance MESURÉE (le préfixe que l'endpoint a accepté). nil = ignorée.
	Observe func(famille string)
}
