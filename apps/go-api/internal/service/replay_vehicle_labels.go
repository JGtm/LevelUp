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
//
// DEPUIS LE LOT 1.9.9 (2026-09-16), CETTE TABLE PORTE AUSSI CE QUE LA FAMILLE *EST*. Une famille
// de chassis peut ne pas etre un vehicule : la TOURELLE AUTOMATIQUE BANNIE (`tourelle_auto_bannie`,
// chassis `0x038df01a`) est un ELEMENT DE CARTE — decision utilisateur du 2026-09-14. Le manifeste
// du titre (`replay_labels.toml`, `[[vehicle_families]]`) la QUALIFIE : nature (`kind`), libelle
// bilingue, et si un asset est servi pour elle. Les libelles ne sont JAMAIS ecrits en Go (regle 1
// du depot) ; la nature traverse jusqu au client, qui lui reserve un pictogramme dedie au lieu du
// marqueur neutre des chassis non resolus.

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
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
	qualifiees := s.vehicleFamiliesQualifiees(ctx)
	labels := make(map[string]replay.VehicleLabel, len(families))
	for _, f := range families {
		lbl, ok := vehicleLabelOf(f, qualifiees[f], s.titleSlug)
		if !ok {
			continue
		}
		labels[f] = lbl
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

// vehicleLabelOf compose le libelle d UNE famille : son URL de sprite quand un asset est servi,
// et ce que le TITRE dit d elle quand il la qualifie (nature, libelle bilingue).
//
// Rend faux quand il n y a RIEN a publier pour cette famille : ni URL composable, ni
// qualification. Une entree vide dans la table dirait au client « cette famille existe et n a
// rien », ce qui est indistinguable d une famille qu il ne connait pas.
//
// L ORDRE DES DEUX REFUS COMPTE. Une famille QUALIFIEE SANS asset (`Sprite` faux) est publiee
// avec sa nature et son libelle, SANS URL : c est exactement ce qui permet au client de dessiner
// son pictogramme dedie plutot que le marqueur neutre des chassis non resolus. Composer une URL
// morte a la place ferait un 404 par match et laisserait le client incapable de distinguer
// « pas encore charge » de « aucun asset a ce jour ».
func vehicleLabelOf(famille string, info replay.VehicleFamilyInfo, titleSlug string) (replay.VehicleLabel, bool) {
	lbl := replay.VehicleLabel{Kind: info.Kind, En: info.En, Fr: info.Fr}
	// Une famille NON qualifiee par le titre (le cas de dix-huit sur dix-neuf : les noms propres
	// du jeu) a toujours son asset servi — c est le regime d avant le lot 1.9.9, inchange.
	if info.Kind == "" || info.Sprite {
		url := static.URL(static.KindVehicle, titleSlug, vehicleSpriteDir+famille, ".png")
		if url == "" {
			// Composition refusee (titre vide, Kind invalide) : la famille garde son nom et
			// perd sa vignette. Jamais l URL d un voisin.
			return lbl, lbl.Kind != ""
		}
		lbl.Img, lbl.Tinted = url, true
	}
	return lbl, lbl.Img != "" || lbl.Kind != ""
}

// vehicleFamiliesQualifiees rend ce que le TITRE declare des familles de chassis (nature,
// libelle bilingue, asset servi ou non), ou nil.
//
// MEME PATRON ET MEME RAISON QUE `resolveWeaponLabels` : le catalogue est charge POUR LE TITRE du
// service, a la requete, parce qu une qualification qui peut s ameliorer ne se fige pas dans un
// artefact deja cuit. Best-effort ET DIT : un catalogue illisible est une erreur de
// configuration, pas un document sans vehicules.
func (s *replayService) vehicleFamiliesQualifiees(ctx context.Context) map[string]replay.VehicleFamilyInfo {
	cat, err := replaylabels.Load(s.repoRoot, s.titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : manifeste du titre illisible — familles de chassis non"+
			" qualifiees (ni nature, ni libelle)", "err", err, "titleSlug", s.titleSlug)
		return nil
	}
	return cat.VehicleFamilies
}

// vehicleFamiliesUsed releve les familles employees par les vies, TRIEES (l ordre d une map ne
// se publie pas), et compte les vies restees sans famille.
//
// LA VARIANTE COMPTE COMME UNE FAMILLE EMPLOYEE (schema 69) : c est elle qui nomme le sprite d un
// Rockethog ou d un Gungoose (`VehicleTrack.Variant`), et son URL se compose exactement comme celle
// d une famille — le lot A sert un sprite par variante.
func vehicleFamiliesUsed(tracks []replay.VehicleTrack) ([]string, int) {
	seen := map[string]bool{}
	unnamed := 0
	for _, tr := range tracks {
		if tr.Variant != "" {
			seen[tr.Variant] = true
		}
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
