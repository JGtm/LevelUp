package filmreplis

import "github.com/JGtm/halo-infinite-film-decoder/decfilm"

// Les fichiers de l'hote qui portent plusieurs sites.
const (
	fichierHits         = "internal/sync/killcollector/hits.go"
	fichierShots        = "internal/sync/killcollector/shots.go"
	fichierReplaybuild  = "internal/replaybuild/replaybuild.go"
	fichierRegleDuDecor = "internal/service/replay_vehicle_scenery_rule.go"
)

// sitesDeLHote : les sites, dans go-api, des replis que le registre de la brique marque
// `CompleteeParLHote`. Fichier relatif a la racine du module `apps/go-api`, ancre exacte comme dans
// le registre de la brique.
var sitesDeLHote = map[decfilm.Nom][]decfilm.Site{
	decfilm.NomDistancesDeToucheDesactivees: {
		{Fichier: fichierHits, Ancre: "replisDeLaPasse(ctx).Declenche(decfilm.NomDistancesDeToucheDesactivees)"},
		{Fichier: fichierHits, Ancre: "carte hors catalogue de bornes, distances desactivees"},
	},
	decfilm.NomCartePremierNomResolu: {
		{Fichier: "internal/sync/killcollector/map_identity.go", Ancre: "replisDeLaPasse(ctx).Declenche(decfilm.NomCartePremierNomResolu)"},
		{Fichier: fichierHits, Ancre: "premierNomSansArbitrage(ctx, noms, err)"},
		{Fichier: "internal/sync/killcollector/map_identity.go", Ancre: "if entry, err := c.mapBounds.Lookup(name); err == nil {"},
	},
	decfilm.NomXuidVidePourNomInconnu: {
		{Fichier: "internal/sync/killcollector/identities.go", Ancre: "m.replis.Declenche(decfilm.NomXuidVidePourNomInconnu)"},
		{Fichier: "internal/sync/killcollector/identities.go", Ancre: "return m.ParNom[nom], nom"},
	},
	decfilm.NomIndiceEnCollisionJete: {
		{Fichier: fichierShots, Ancre: "fb.DeclencheN(decfilm.NomIndiceEnCollisionJete, collisions)"},
		{Fichier: fichierShots, Ancre: "out[pi] = \"\""},
	},
	decfilm.NomPremiereOccurrenceSansConcordance: {
		{Fichier: fichierShots, Ancre: "*discordances++"},
		{Fichier: fichierShots, Ancre: "fb.DeclencheN(decfilm.NomPremiereOccurrenceSansConcordance, discordances)"},
		{Fichier: fichierShots, Ancre: "continue // premiere occurrence gagnante"},
	},
	decfilm.NomPrecisionParArmePasseSautee: {
		{Fichier: fichierHits, Ancre: "replisDeLaPasse(ctx).Declenche(decfilm.NomPrecisionParArmePasseSautee)"},
		{Fichier: fichierHits, Ancre: "numerateur non configure (chemin live sans cache disque)"},
	},
	decfilm.NomCoequipiersPartisConstanteNulle: {
		{Fichier: "internal/sync/killcollector/isolation_facts.go", Ancre: "ids.replis.DeclencheN(decfilm.NomCoequipiersPartisConstanteNulle, len(out))"},
		{Fichier: "internal/sync/killcollector/isolation_facts.go", Ancre: "TeammatesLeft:  0,"},
	},
	"repli_enregistrement_statborg_abandonne": {
		{Fichier: "internal/replaybuild/matchfacts.go", Ancre: "sb.Replis.Plus(pont.Identite().ComptesDesReplis())"},
	},
	decfilm.NomAssistantNonResoluAbandonne: {
		{Fichier: "internal/replaybuild/kills.go", Ancre: "r.assistantsNonResolus++"},
		{Fichier: "internal/replaybuild/kills.go", Ancre: "fb.DeclencheN(decfilm.NomAssistantNonResoluAbandonne, r.assistantsNonResolus)"},
		{Fichier: "internal/replaybuild/kills.go", Ancre: "ref.AssistXUID, ref.AssistKnown = aXUID, true"},
	},
	decfilm.NomRepereNeutreGeneriqueConserve: {
		{Fichier: fichierReplaybuild, Ancre: "fb.Declenche(decfilm.NomRepereNeutreGeneriqueConserve)"},
		{Fichier: fichierReplaybuild, Ancre: "kind, img, ok := adapter.NeutralDeathIcon(d.Source.Tag)"},
	},
	decfilm.NomRelaisDeBotAbandonne: {
		{Fichier: fichierReplaybuild, Ancre: "fb.Declenche(decfilm.NomRelaisDeBotAbandonne)"},
		{Fichier: fichierReplaybuild, Ancre: "if _, err := fmt.Sscanf(p.XUID, \"bid(%d.0)\", &id); err != nil {"},
	},
	decfilm.NomCatalogueDeZonesAbsent: {
		{Fichier: "internal/replaybuild/zones.go", Ancre: "fb.Declenche(decfilm.NomCatalogueDeZonesAbsent)"},
		{Fichier: "internal/replaybuild/zones.go", Ancre: "titre sans table d'objectifs"},
	},
	decfilm.NomFraicheurDesDerivationsParTaille: {
		{Fichier: "internal/sync/replayartifacts/derivations_backlog.go", Ancre: "fb.Declenche(decfilm.NomFraicheurDesDerivationsParTaille)"},
		{Fichier: "internal/replaybuild/derivations_index.go", Ancre: "return m.Rev == DerivationsRev && int64(m.ArtifactBytes) == st.Size()"},
	},
	"repli_temps_forts_dernier_numero": {
		{Fichier: "internal/replaybuild/filmfacts_cuisson.go", Ancre: "func jugerFilmSansManifeste("},
	},
	"repli_coequipier_hors_de_vue_par_defaut": {
		{Fichier: "internal/sync/killcollector/positions_entrees.go", Ancre: "Fallbacks: ids.replis, OrigineDuFilmUS"},
	},
	"repli_decor_sous_le_sol_foule_du_match": {
		{Fichier: fichierRegleDuDecor, Ancre: "case floorKnown && s.Z < floor-sceneryFloorToleranceM:"},
	},
	"repli_decor_carte_sans_zone_affiche": {
		{Fichier: fichierRegleDuDecor, Ancre: "// repli_decor_carte_sans_zone_affiche : aucune zone, rien de masque, compte."},
	},
	"repli_decor_tenu_en_l_air_a_vide": {
		{Fichier: fichierRegleDuDecor, Ancre: "// repli_decor_tenu_en_l_air_a_vide : compte dans `hidden` sous sa raison."},
	},
}

// appelantsDeLHote : les repertoires de go-api (relatifs a la racine du module) ou un outil hors
// production appelle le symbole d'un repli que le registre de la brique marque `CompleteeParLHote`.
var appelantsDeLHote = map[decfilm.Nom][]string{
	"repli_famille_objectif_vide": {"cmd/diag_weapons_v3"},
}
