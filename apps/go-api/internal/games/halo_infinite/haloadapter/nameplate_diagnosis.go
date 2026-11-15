// Package haloadapter — nameplate_diagnosis.go : diagnostic structuré de
// la résolution d'apparence Spartan (bannière/nameplate, emblème, backdrop,
// service tag) pour la surface admin « diagnostic apparence ».
//
// Ce fichier N'AJOUTE AUCUNE logique de fetch : il expose le POURQUOI de la
// résolution existante. DiagnoseNameplate partage la fonction interne
// resolveNameplate avec ResolveNameplateURL (aucune duplication du fetch
// mapping/CMS) ; DiagnoseCustomizationImage réutilise resolveCustomizationImageURL.
//
// La sémantique de LECTURE de l'apparence ne bouge PAS d'un octet : chaque
// composant est diagnostiqué INDÉPENDAMMENT, aucune condition croisée
// bannière↔emblème (verrouillé par les tests du 2026-07-08). Un verdict non-ok
// signifie SEULEMENT « le live n'a rien résolu » — la dernière valeur connue
// reste servie par le read-path (directive « jamais vide »).
//
// Note frontière hosts (garde-rail archlint des littéraux d'hôte) : aucun host
// en dur ici — les URLs sont produites par resolveNameplate et
// resolveCustomizationImageURL, qui passent par le resolver d'endpoints
// (adresses de la brique construites depuis le résolveur de titre).
package haloadapter

import (
	"context"

	"levelup/go-api/internal/domain"
)

// servedFromForVerdict dérive ServedFrom du verdict : ok signifie que le live a
// résolu (live) ; tout autre verdict signifie que la valeur servie est la
// dernière connue reportée (carry).
func servedFromForVerdict(v domain.AppearanceVerdict) domain.AppearanceServedFrom {
	if v == domain.AppearanceVerdictOK {
		return domain.AppearanceServedFromLive
	}
	return domain.AppearanceServedFromCarry
}

// DiagnoseNameplate diagnostique la résolution de la BANNIÈRE dérivée de
// l'emblème (fallback nameplate — voir ResolveNameplateURL). Il s'appuie sur la
// MÊME fonction interne resolveNameplate : aucune duplication du fetch mapping/CMS.
// Verdicts possibles : ok / upstream_missing / transient.
func DiagnoseNameplate(
	ctx context.Context,
	emblemPath string,
	cfg int64,
	spartanToken, clearanceToken string,
) domain.AppearanceDiagnosis {
	url, verdict, detail := resolveNameplate(ctx, emblemPath, cfg, spartanToken, clearanceToken)
	return domain.AppearanceDiagnosis{
		ServedFrom:  servedFromForVerdict(verdict),
		ResolvedURL: url,
		Verdict:     verdict,
		Detail:      detail,
	}
}

// DiagnoseCustomizationImage diagnostique la résolution d'une image de
// customisation (emblème ou backdrop) via resolveCustomizationImageURL — le MÊME
// chemin que GetSpartanCustomization. Succès → ok ; tout échec (HTTP / parse /
// media path absent) → transient (indéterminé, se répare au prochain refresh).
func (c *Client) DiagnoseCustomizationImage(ctx context.Context, inventoryPath string) domain.AppearanceDiagnosis {
	url, err := c.resolveCustomizationImageURL(ctx, inventoryPath)
	if err != nil || url == "" {
		return domain.AppearanceDiagnosis{
			ServedFrom: domain.AppearanceServedFromCarry,
			Verdict:    domain.AppearanceVerdictTransient,
			Detail:     domain.AppearanceDetailImageUnresolved,
		}
	}
	return domain.AppearanceDiagnosis{
		ServedFrom:  domain.AppearanceServedFromLive,
		ResolvedURL: url,
		Verdict:     domain.AppearanceVerdictOK,
		Detail:      domain.AppearanceDetailImageResolved,
	}
}
