package service

// replay_vehicle_labels.go — LE SPRITE DE CHAQUE FAMILLE DE CHASSIS, pose A LA REQUETE.
//
// CE QUE CA OUVRE. L artefact publie, par vie de vehicule, une FAMILLE de chassis (`warthog`,
// `ghost`, `mongoose`...) resolue du mot d identite `MPPWord32` du film. Il ne publie AUCUNE URL :
// un asset statique est servi par le serveur, et son chemin depend du TITRE. Cette table donne au
// client, pour chaque famille effectivement employee par le document, l URL du sprite vu de
// dessus et le fait qu il se teigne a la couleur de l equipe qui l occupe.
//
// POURQUOI A LA REQUETE ET NON AU BUILD. C est le patron deja pose par `mapObjectives` et par
// `WeaponLabel.Key` (cf. `replay_weapon_labels.go`), et la raison est la meme, mesuree : les
// artefacts sont cuits hors ligne et le sont deja par dizaines ; une URL figee au build les
// laisserait muets jusqu a une re-cuisson complete. Et la regle du depot le dit — on ne stocke
// jamais une resolution qui peut s ameliorer.
//
// TITLE-AGNOSTIC PAR CONSTRUCTION : l URL est composee POUR LE TITRE du service
// (`static.URL(static.KindVehicle, s.titleSlug, ...)`), sans aucune comparaison de slug. Un titre
// dont le dossier d assets ne porte pas ces sprites servira des URLs qui n aboutissent pas — le
// client garde alors son marqueur neutre, exactement comme pour une famille inconnue. Aucune clef
// de capability `replay.*` n existe : exemption assumee, documentee au plan d integration.

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/assets/static"
)

// vehicleSpriteDir est le sous-dossier des sprites de vehicule sous
// `/static/vehicles-assets/{titleSlug}/`.
//
// POURQUOI UN SOUS-DOSSIER : le dossier d assets d un vehicule n est pas reserve au rejeu 2D
// (rien n interdit d y servir demain une vignette de fiche ou une icone de medaille de vehicule),
// et ces sprites-ci ont une convention qui leur est propre — vue de dessus, nez en haut, echelle
// en mm/px documentee par `index.json`. Meme motif et meme raison que `weaponIconDir` pour les
// icones d arme extraites du jeu.
const vehicleSpriteDir = "replay/"

// resolveVehicleLabels pose `doc.VehicleLabels` : une entree par FAMILLE effectivement employee
// par les vies du document.
//
// NIL-SAFE ET SILENCIEUX EN DEGRADATION : un document sans vehicule, ou dont aucune vie ne resout
// de famille, ne recoit AUCUNE table — pas une table vide. L absence dit « rien a dessiner », une
// table vide dirait « regarde, il n y a rien », et le client ne peut pas distinguer les deux.
// Le journal, lui, compte ce qui a ete resolu et ce qui ne l a pas ete : une famille vide n est
// jamais une erreur (le chassis n est pas au catalogue), mais elle ne doit pas etre muette.
func (s *replayService) resolveVehicleLabels(ctx context.Context, doc *replay.ReplayDocument) {
	if doc == nil || len(doc.Vehicles) == 0 {
		return
	}
	families, unnamed := vehicleFamiliesUsed(doc.Vehicles)
	if len(families) == 0 {
		slog.InfoContext(ctx, "rejeu 2D : aucune famille de chassis resolue — vehicules sans sprite",
			"vies", len(doc.Vehicles), "titleSlug", s.titleSlug)
		return
	}
	labels := make(map[string]replay.VehicleLabel, len(families))
	for _, f := range families {
		url := static.URL(static.KindVehicle, s.titleSlug, vehicleSpriteDir+f, ".png")
		if url == "" {
			// Composition refusee (titre vide, Kind invalide) : la famille garde son nom et
			// perd sa vignette. Jamais l URL d un voisin.
			continue
		}
		labels[f] = replay.VehicleLabel{Img: url, Tinted: true}
	}
	if len(labels) == 0 {
		slog.WarnContext(ctx, "rejeu 2D : aucune URL de sprite de vehicule composee",
			"familles", len(families), "titleSlug", s.titleSlug)
		return
	}
	doc.VehicleLabels = labels
	if unnamed > 0 {
		slog.InfoContext(ctx, "rejeu 2D : vies de vehicule sans famille resolue",
			"sansFamille", unnamed, "familles", len(labels), "titleSlug", s.titleSlug)
	}
}

// vehicleFamiliesUsed releve les familles employees par les vies, TRIEES (l ordre d une map ne
// se publie pas), et compte les vies restees sans famille.
func vehicleFamiliesUsed(tracks []replay.VehicleTrack) ([]string, int) {
	seen := map[string]bool{}
	unnamed := 0
	for _, tr := range tracks {
		if tr.Family == "" {
			unnamed++
			continue
		}
		seen[tr.Family] = true
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, unnamed
}
