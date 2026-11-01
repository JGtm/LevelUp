package infinite

// ugcmaps.go — cartes du Discovery UGC ([CleDiscoveryUGC], segment `Maps`) et fichiers de leur
// stockage blob. Deux GET uniques, sans reprise : le Discovery rend le DTO d'une carte (avec ou sans
// version, authentifié, identité de l'appelant), le stockage blob rend des octets (sans jeton).
// Distinct de [Client.FetchAsset] : autre segment (`Maps`), version facultative, DTO réduit,
// identité (`User-Agent`, `Origin`) fournie par l'appelant. Le choix du fichier de variante, la
// validation du DTO, les journaux et le délai de la requête (celui du Doer) sont à l'appelant.
// Différences de FORME avec le client d'avant la brique : un refus de la garde est marqué
// [haloapi.ErrDefinitif] (`errors.Is` de la cause reste vrai) et la garde précède la construction de l'URL.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// lectureMaxUGC : octets lus au plus d'une réponse du Discovery ou du stockage blob (garde-fou : une
// variante de carte pèse une centaine de Ko).
const lectureMaxUGC = 64 << 20

// enteteUserAgent et enteteOrigin : les en-têtes d'identité d'un GET unique de l'appelant.
const (
	enteteUserAgent = "User-Agent"
	enteteOrigin    = "Origin"
)

// Identite : l'identité que l'appelant présente aux services UGC (`User-Agent`, `Origin`). Valeur de
// l'application, jamais une constante de la brique. Une valeur vide : l'en-tête n'est pas posé.
type Identite struct {
	UserAgent string
	Origin    string
}

// CarteUGC : le DTO Discovery d'une carte (champs utilisés seulement, le reste du document est
// ignoré).
type CarteUGC struct {
	AssetID    string `json:"AssetId"`
	VersionID  string `json:"VersionId"`
	PublicName string `json:"PublicName"`
	Files      struct {
		Prefix            string   `json:"Prefix"`
		FileRelativePaths []string `json:"FileRelativePaths"`
	} `json:"Files"`
	CustomData struct {
		NumOfObjectsOnMap int   `json:"NumOfObjectsOnMap"`
		TagLevelID        int64 `json:"TagLevelId"`
		IsBaked           bool  `json:"IsBaked"`
	} `json:"CustomData"`
}

// FetchCarteUGC résout une carte via le Discovery UGC : `{hôte}/{jeu}/Maps/{assetID}` puis
// `/versions/{versionID}` si `versionID` n'est pas vide (dernière version publiée sinon). Les jetons
// sont obligatoires : sans jeton Spartan, aucune requête n'est envoyée. Un statut non-200 rend un
// [*HTTPError], un échec d'émission un [*ErreurTransport].
func (c *Client) FetchCarteUGC(ctx context.Context, auth Auth, assetID, versionID string, id Identite) (*CarteUGC, error) {
	// La garde d'abord : une sortie refusée (mode démo) l'est avant toute autre erreur.
	if err := c.verifierGarde(ctx); err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/Maps/%s", c.endpoints.base(CleDiscoveryUGC), url.PathEscape(assetID))
	if versionID != "" {
		endpoint += "/versions/" + url.PathEscape(versionID)
	}
	if auth.Spartan == "" {
		return nil, fmt.Errorf("aucun jeton Spartan disponible pour %s", endpoint)
	}
	body, err := c.getUnique(ctx, requeteUnique{
		url: endpoint, auth: auth, accept: typeJSON, lectureMax: lectureMaxUGC,
		entetes: entetesIdentite(id),
	})
	if err != nil {
		return nil, err
	}
	var carte CarteUGC
	if err := json.Unmarshal(body, &carte); err != nil {
		return nil, fmt.Errorf("décoder l'asset %s: %w", assetID, err)
	}
	return &carte, nil
}

// TelechargerFichierUGC télécharge un fichier du stockage blob d'une carte : `prefix` (sans sa barre
// finale) puis `/` puis `relPath`, NON échappé. Aucun jeton n'est envoyé (le stockage blob répond
// sans) ; seule l'identité `User-Agent` l'est. La garde du Doer ([haloapi.Garde]) est consultée
// d'abord : une sortie refusée rend l'erreur de la garde telle quelle, avant toute autre erreur (URL
// invalide comprise) et sans requête. Même erreurs que [Client.FetchCarteUGC].
func (c *Client) TelechargerFichierUGC(ctx context.Context, prefix, relPath string, id Identite) ([]byte, error) {
	if err := c.verifierGarde(ctx); err != nil {
		return nil, err
	}
	id.Origin = "" // le stockage blob ne reçoit que le User-Agent
	body, err := c.getUnique(ctx, requeteUnique{
		url: strings.TrimSuffix(prefix, "/") + "/" + relPath, accept: typeJSON, lectureMax: lectureMaxUGC,
		entetes: entetesIdentite(id),
	})
	return body, err
}

// entetesIdentite : les en-têtes d'identité non vides, `User-Agent` d'abord.
func entetesIdentite(id Identite) []entete {
	var out []entete
	if id.UserAgent != "" {
		out = append(out, entete{enteteUserAgent, id.UserAgent})
	}
	if id.Origin != "" {
		out = append(out, entete{enteteOrigin, id.Origin})
	}
	return out
}
