package infinite

// gamecmsfichiers.go — fichiers GameCMS servis tels quels ([CleGameCMS]) : images de médailles, de
// badges de défis et d'identité Spartan, documents JSON de progression. Deux fonctions PURES
// construisent l'URL d'un fichier, un GET unique (sans reprise, sans en-tête `Accept`) rend ses
// octets. Le choix du chemin, le classement des erreurs (404, autre statut), le type MIME rendu et
// le cache sont à l'appelant ; le délai de la requête est celui du Doer.
// Différences de FORME avec le client d'avant la brique : un refus de la garde est marqué
// [haloapi.ErrDefinitif] (`errors.Is` de la cause reste vrai) et la garde précède la construction de l'URL.

import (
	"context"
	"strings"
)

// URLFichierGameCMS rend l'URL d'un fichier GameCMS : `{hôte}/{jeu}/{chemin}`, `chemin` étant le
// chemin après le segment de jeu (par exemple `Progression/file/Metadata/Metadata.json` ou
// `waypoint/file/images/{id}.png`), posé tel quel.
func URLFichierGameCMS(e Endpoints, chemin string) string {
	return e.base(CleGameCMS) + "/" + chemin
}

// URLImageGameCMS rend l'URL de téléchargement d'une image GameCMS désignée par `brut`, ou "" si
// `brut` n'en désigne aucune (vide, ou document `.json`). Une URL absolue (`http://`, `https://`)
// est rendue telle quelle ; un chemin déjà préfixé du segment de jeu (`{jeu}/images/file/`,
// `{jeu}/progression/file/`, `{jeu}/waypoint/file/`) est posé sous l'hôte ; un chemin
// `images/file/`, `progression/file/` ou `waypoint/file/` reçoit le segment de jeu ; tout autre
// chemin reçoit `{jeu}/images/file/`. Les comparaisons ignorent la casse, le chemin garde la sienne
// (barres de tête retirées).
func URLImageGameCMS(e Endpoints, brut string) string {
	nettoye := strings.TrimSpace(brut)
	if nettoye == "" {
		return ""
	}
	minuscule := strings.ToLower(nettoye)
	if strings.HasSuffix(minuscule, ".json") {
		return ""
	}
	if strings.HasPrefix(minuscule, "http://") || strings.HasPrefix(minuscule, "https://") {
		return nettoye
	}
	hote, jeu := e.Hote(CleGameCMS), e.PrefixeJeu()
	chemin := strings.TrimLeft(nettoye, "/")
	minuscule = strings.ToLower(chemin)
	p := strings.ToLower(jeu)
	switch {
	case aUnPrefixe(minuscule, p+"/images/file/", p+"/progression/file/", p+"/waypoint/file/"):
		return hote + "/" + chemin
	case aUnPrefixe(minuscule, "images/file/", "progression/file/", "waypoint/file/"):
		return hote + "/" + jeu + "/" + chemin
	default:
		return hote + "/" + jeu + "/images/file/" + chemin
	}
}

// aUnPrefixe dit si `s` commence par l'un des `prefixes`.
func aUnPrefixe(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// TelechargerFichierGameCMS télécharge le fichier d'URL `url` (construite par [URLFichierGameCMS] ou
// [URLImageGameCMS]) en un GET unique, sans en-tête `Accept`, avec les jetons de `auth` s'ils sont
// donnés ([Auth] vide : aucun en-tête d'authentification). La garde du Doer ([haloapi.Garde]) est
// consultée d'abord : une sortie refusée rend l'erreur de la garde telle quelle, avant toute autre
// erreur (URL invalide comprise) et sans requête. Un statut non-200 rend un [*HTTPError] (404
// compris : l'appelant classe), un échec d'émission un [*ErreurTransport], un corps coupé un
// [*ErreurLecture].
func (c *Client) TelechargerFichierGameCMS(ctx context.Context, auth Auth, url string) ([]byte, error) {
	if err := c.verifierGarde(ctx); err != nil {
		return nil, err
	}
	return c.getUnique(ctx, requeteUnique{url: url, auth: auth})
}
