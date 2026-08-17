package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTempStructure(t *testing.T, ms MapStructure) string {
	t.Helper()
	blob, err := json.Marshal(ms)
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	p := filepath.Join(t.TempDir(), "ridgeline.json")
	if err := os.WriteFile(p, blob, 0o644); err != nil {
		t.Fatalf("écriture: %v", err)
	}
	return p
}

func TestLoadMapStructure(t *testing.T) {
	p := writeTempStructure(t, MapStructure{
		SchemaVersion: MapStructureSchemaVersion,
		Module:        "ridgeline",
		Surfaces:      []Surface{{X0: -1, Y0: -2, X1: 3, Y1: 4, Z: 1.5, ZB: 0}},
	})
	ms, err := LoadMapStructure(p)
	if err != nil {
		t.Fatalf("LoadMapStructure: %v", err)
	}
	if ms.Module != "ridgeline" || len(ms.Surfaces) != 1 {
		t.Fatalf("document inattendu: %+v", ms)
	}
	if got := ms.Surfaces[0].Area(); got != 24 {
		t.Fatalf("aire = %v, attendu 24", got)
	}
}

// Une face inférieure à exactement 0 doit SURVIVRE à l'aller-retour JSON : un omitempty sur
// ZB la relirait comme 0 par défaut — invisible ici, mais il déplacerait d'un étage toute
// surface dont la face haute serait, elle, omise. Le test fige l'absence d'omitempty.
func TestSurfaceZeroAltitudeSurvivesJSON(t *testing.T) {
	blob, err := json.Marshal(Surface{X0: 1, Y0: 1, X1: 2, Y1: 2, Z: 0, ZB: 0})
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("désérialisation: %v", err)
	}
	for _, k := range []string{"z", "zb"} {
		if _, ok := back[k]; !ok {
			t.Fatalf("champ %q omis du JSON (%s) — une altitude nulle serait perdue", k, blob)
		}
	}
}

func TestLoadMapStructureRejectsOtherSchemaVersion(t *testing.T) {
	p := writeTempStructure(t, MapStructure{SchemaVersion: MapStructureSchemaVersion + 1})
	if _, err := LoadMapStructure(p); err == nil {
		t.Fatal("une version de schéma inconnue doit être refusée")
	}
}

func TestSurfaceBounds(t *testing.T) {
	if surfaceBounds(nil) != nil {
		t.Fatal("aucune surface -> pas de bornes")
	}
	b := surfaceBounds([]Surface{
		{X0: 0, Y0: 0, X1: 2, Y1: 2},
		{X0: -5, Y0: 1, X1: -1, Y1: 9},
	})
	if b == nil || b.MinX != -5 || b.MinY != 0 || b.MaxX != 2 || b.MaxY != 9 {
		t.Fatalf("bornes inattendues: %+v", b)
	}
}

// L'ajout de la structure ne doit PAS incrémenter SchemaVersion : les champs sont
// optionnels (omitempty), un client existant qui les ignore reste correct.
func TestStructureIsOptionalInDocument(t *testing.T) {
	doc := ReplayDocument{SchemaVersion: SchemaVersion}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("désérialisation: %v", err)
	}
	for _, k := range []string{"structure", "structureBounds"} {
		if _, ok := back[k]; ok {
			t.Fatalf("champ %q présent sur un document sans structure — omitempty manquant", k)
		}
	}
	// LA RÈGLE TENUE ICI : un champ OPTIONNEL de plus n'incrémente pas la version — sans
	// quoi chaque enrichissement casserait les clients. La version ne bouge que sur un
	// changement de FORME, et chaque incrément doit avoir sa raison écrite :
	//   v1 -> v2 (2026-08-02, lot 3.1/3.2) : les tables de libellés passent de `string`
	//   à `{en, fr}` et le type d'un lancer de grenade devient son RANG. Deux formes
	//   changées, donc une version — la structure, elle, n'y est pour rien.
	//   v2 -> v3 (2026-08-13, plan parité lot 2) : Grenade.proj (lien lancer -> projectile)
	//   et killEffects. Champs omitempty, mais la version monte parce qu'elle est la CLÉ DE
	//   REPRISE du backfill (lot 6) : les effets de repos de grenade n'existent que sur un
	//   artefact qui porte le lien, un v2 doit se lire « à re-cuire », pas « à jour ».
	//   v3 -> v4 (2026-08-14, plan parité lot 7.2) : originMs, l'instant de la frame 0 sur
	//   l'horloge du fil des éliminations. Champ omitempty, même raison de monter : c'est la
	//   CLÉ DE REPRISE du backfill, et sans lui le client ne peut caler le fil que par
	//   appariement statistique — un v3 doit se lire « à re-cuire », pas « à jour ».
	//   v4 -> v5 (2026-08-14, plan parité lot 7.1) : neutralDeaths, le TYPE des morts que
	//   personne ne revendique (chute / hors-limites, ou sa propre source de dégât). Champ
	//   omitempty, même raison de monter : c'est la CLÉ DE REPRISE du backfill, et sans lui
	//   le fil ne peut poser sur ces lignes qu'un repère générique — un v4 doit se lire
	//   « à re-cuire », pas « à jour ».
	//   v5 -> v6 (2026-08-14, plan PLAN_RANG_CAPACITE_I48 étape 1.2) : la capacité d'armure
	//   CHANGE DE GRANDEUR. `Inventory.a` portait `rang − 16` — l'ancre du canal d'image-clé
	//   se termine par `010`, les bits de poids fort du rang, et ce canal ne voit donc que la
	//   fenêtre 16..23. Le document publie désormais `abilities`, le RANG complet, alimenté
	//   par deux canaux. Le champ `a` est RETIRÉ, pas réinterprété : republier une autre
	//   grandeur sous la même clé aurait laissé tout client non mis à jour lire un nombre qui
	//   ne veut plus dire la même chose. Ce n'est donc PAS un champ optionnel de plus — c'est
	//   un retrait plus un changement de sens, et `abilityLabels` change de clé avec.
	//   v6 -> v7 (2026-08-16, plan PLAN_EQUIPEMENT_TI37 phase 1) : equipmentEpisodes, l'état
	//   ACTIF du camouflage et du surbouclier en épisodes datés par vie. Champ omitempty,
	//   même raison de monter que v3/v4/v5 : c'est la CLÉ DE REPRISE du backfill — l'effet
	//   plein-fiche et les sons d'équipement n'existent que sur un artefact qui porte les
	//   épisodes, un v6 doit se lire « à re-cuire », pas « à jour ».
	//   v7 -> v8 (2026-08-16, plan PLAN_GRAPPIN_LIGNE phase 1) : grappleLines, les tractions
	//   de grappin datées par vie avec leur point d'accroche en coordonnées monde. Champ
	//   omitempty, même raison de monter que v3/v4/v5/v7 : c'est la CLÉ DE REPRISE du
	//   backfill — la ligne joueur -> ancre n'existe que sur un artefact qui porte les
	//   tractions, un v7 doit se lire « à re-cuire », pas « à jour ».
	//   v8 -> v9 (2026-08-18, plan PLAN_POSES_EQUIPEMENT_PUBLICATION phase 2) :
	//   equipmentPlacements, les POSES d'objets d'équipement (mur, capteur, et les objets du
	//   monde qui partagent l'archétype) avec leur position monde, leur fenêtre [t0, t1],
	//   l'identifiant `eqip` du jeu, le poseur mesuré et son cap de visée. Champ omitempty,
	//   même raison de monter que v3/v4/v5/v7/v8 : c'est la CLÉ DE REPRISE du backfill — les
	//   marqueurs d'équipement posé n'existent que sur un artefact qui porte les poses, un v8
	//   doit se lire « à re-cuire », pas « à jour ». La couverture monte avec le champ
	//   (`coverage.placements`), et elle porte le découpage de bloc CALIBRÉ sur le film :
	//   sans elle, zéro pose par absence d'équipement et zéro pose par calibration refusée
	//   seraient indistinguables.
	//   v9 -> v10 (2026-08-18, plan PLAN_ORIGINE_POSES_ET_FAMILLES phase G) : chaque pose porte
	//   son ORIGINE MESURÉE (`origin` : `deployed` / `dropped` / `unknown`), et
	//   `coverage.placements` la croise avec la famille. UN CHAMP S'AJOUTE À UN SOUS-OBJET, ce
	//   qui d'ordinaire ne monte pas la version — ici c'est le SENS DU CALQUE ENTIER qui change,
	//   et c'est la raison. Mesure sur les 11 films calibrés : 3 242 des 3 661 poses à poseur
	//   mesuré (88,6 %) naissent dans les 2 frames ET les 1,5 m du DERNIER POINT de leur poseur
	//   — ce sont les objets qu'il PORTAIT, relâchés quand sa vie s'achève, pas des poses sur la
	//   carte. Un client v9 dessine donc un mur là où personne n'en a déployé, et sans montée de
	//   version il continuerait : la reprise du backfill se fait par SchemaVersion.
	//   L'hypothèse inverse — une DOTATION AU SPAWN, écrite avant mesure — est RÉFUTÉE : 4 poses
	//   sur 3 661 (0,1 %), et les 4 sont des vies de 0,13 à 1,49 s où début et fin se confondent.
	//   v10 -> v11 (2026-08-17, plan PLAN_ARMES_AU_SOL_2E_LECTURE phase 3) : `weaponPads` et
	//   `padPickups`, les SOCLES D'ARME du match — position, famille, apparitions, intervalles de
	//   présence bornés par le recensement des images-clés, cycle de réapparition quand il est
	//   ÉTABLI, et les occupations qui se sont achevées. Champs omitempty, même raison de monter
	//   que v3/v4/v5/v7/v8/v9 : c'est la CLÉ DE REPRISE du backfill — le calque des socles
	//   n'existe que sur un artefact qui les porte, un v10 doit se lire « à re-cuire », pas
	//   « à jour ». La couverture monte avec les champs (`coverage.groundWeapons`) : un film sans
	//   socle, un film dont toutes les armes sont des lâchers (Super Fiesta : 82,3 % de lâchers,
	//   zéro socle) et un film qu'on n'a pas su balayer rendent tous trois zéro socle, et seuls
	//   ces compteurs les distinguent. Le RAMASSEUR n'est PAS publié (`padPickups[].xuid` vaut
	//   `null`) : l'oracle des loadouts donne 88,1 % par slot de vie et 79,7 % par joueur, contre
	//   >= 90 % exigé, et le seuil n'a pas été rebaissé.
	if SchemaVersion != 11 {
		t.Fatalf("SchemaVersion = %d, attendu 11 : incrémenter exige une raison écrite ci-dessus "+
			"(un champ optionnel de plus n'en est pas une)", SchemaVersion)
	}
}
