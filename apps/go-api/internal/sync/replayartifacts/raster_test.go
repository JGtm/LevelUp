package replayartifacts

// raster_test.go — LA PROJECTION D'UN ARTEFACT VERS SON SIDECAR D'OCCUPATION, SANS BASE.
//
// Aucune base n'entre ici et c'est le fond du lot : le sidecar est un fichier a cote de son
// artefact. Les comptes sont EXACTS et se verifient a la main — 100 ms par frame, 250 ms de
// pas, donc 2 s de presence = 8 echantillons.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// artefactImmobile pose un artefact minimal : deux joueurs qui ne bougent pas, de la frame
// 0 a la frame 20 (soit 2 s a 100 ms par frame).
const artefactImmobile = `{
  "schemaVersion": 39,
  "matchId": "000d5950-1234-4abc-9def-0123456789ab",
  "frameCount": 21,
  "frameIntervalMs": 100,
  "tracks": [
    {"slot":1,"team":-1,"xuid":"111","startFrame":0,"endFrame":20,
     "points":[{"t":0,"x":0.25,"y":0.25},{"t":20,"x":0.25,"y":0.25}]},
    {"slot":2,"team":-1,"xuid":"222","startFrame":0,"endFrame":20,
     "points":[{"t":0,"x":10.25,"y":0.25},{"t":20,"x":10.25,"y":0.25}]}
  ]
}`

// TestProjeterRasterTactique_ComptesExacts — la projection nominale.
func TestProjeterRasterTactique_ComptesExacts(t *testing.T) {
	s, err := ProjeterRasterTactique(ecrireFichier(t, "artefact.json", artefactImmobile))
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if s.SchemaVersion != domain.TacticalRasterSchemaVersion {
		t.Fatalf("schema_version = %d", s.SchemaVersion)
	}
	if s.MatchID != "000d5950-1234-4abc-9def-0123456789ab" {
		t.Fatalf("match_id = %q", s.MatchID)
	}
	if s.ShortID != "000d5950" {
		t.Fatalf("short_id = %q, attendu la cle courte de l'artefact", s.ShortID)
	}
	if s.ArtifactSchemaVersion != 39 {
		t.Fatalf("artifact_schema_version = %d, attendu celui de l'artefact PROJETE", s.ArtifactSchemaVersion)
	}
	if s.PasM != tactical.PasParDefautM {
		t.Fatalf("pas_m = %v, attendu %v", s.PasM, tactical.PasParDefautM)
	}
	if s.FrameIntervalMs != 100 || s.PasEchantillonMs != tactical.PasOccupationMs {
		t.Fatalf("echelles = %d ms/frame, pas %d ms", s.FrameIntervalMs, s.PasEchantillonMs)
	}
	if len(s.Joueurs) != 2 {
		t.Fatalf("joueurs = %d, attendu 2", len(s.Joueurs))
	}
	// Tri par xuid : 111 puis 222.
	if s.Joueurs[0].XUID != "111" || s.Joueurs[1].XUID != "222" {
		t.Fatalf("joueurs non tries par xuid : %q, %q", s.Joueurs[0].XUID, s.Joueurs[1].XUID)
	}
	j := s.Joueurs[0]
	if len(j.Cellules) != 1 || j.Cellules[0].Echantillons != 8 {
		t.Fatalf("cellules de 111 = %+v, attendu une seule a 8 echantillons (2 s / 250 ms)", j.Cellules)
	}
	// x = 0,25 -> col 0 ; x = 10,25 -> col 20. Les indices sont ancres sur l'ORIGINE DU
	// MONDE : c'est ce qui rend deux rasters de matchs differents sommables.
	if j.Cellules[0].Col != 0 || j.Cellules[0].Lig != 0 {
		t.Fatalf("cellule de 111 = (%d,%d), attendu (0,0)", j.Cellules[0].Col, j.Cellules[0].Lig)
	}
	if s.Joueurs[1].Cellules[0].Col != 20 {
		t.Fatalf("cellule de 222 en col %d, attendu 20", s.Joueurs[1].Cellules[0].Col)
	}
	if len(j.Spawns) != 1 || j.Spawns[0].Frame != 0 || j.Spawns[0].X != 0.25 {
		t.Fatalf("spawns de 111 = %+v", j.Spawns)
	}
	if len(j.PremieresEntrees) != 1 || j.PremieresEntrees[0].Frame != 0 {
		t.Fatalf("premieres entrees de 111 = %+v", j.PremieresEntrees)
	}
}

// TestProjeterRasterTactique_SchemaAncien — LE POINT QUI REND LE RATTRAPAGE UTILE AVANT
// UNE RE-CUISSON DU PARC : la projection ne lit que `matchId` et les points des pistes.
// Un artefact de schema 20 — sans `frameIntervalMs`, sans aucun des champs montes depuis —
// se projette exactement comme un artefact courant, a ceci pres que l'echelle de temps
// prend la valeur par defaut du decodeur.
func TestProjeterRasterTactique_SchemaAncien(t *testing.T) {
	const v20 = `{
      "schemaVersion": 20,
      "matchId": "aabbccdd-0000-0000-0000-000000000000",
      "frameCount": 21,
      "tracks": [
        {"slot":1,"team":-1,"xuid":"111",
         "points":[{"t":0,"x":0.25,"y":0.25},{"t":20,"x":0.25,"y":0.25}]}
      ]
    }`
	s, err := ProjeterRasterTactique(ecrireFichier(t, "v20.json", v20))
	if err != nil {
		t.Fatalf("projection d'un artefact de schema 20: %v", err)
	}
	if s.ArtifactSchemaVersion != 20 {
		t.Fatalf("artifact_schema_version = %d, attendu 20", s.ArtifactSchemaVersion)
	}
	if s.FrameIntervalMs != 100 {
		t.Fatalf("frame_interval_ms = %d, attendu la valeur par defaut du decodeur (100)", s.FrameIntervalMs)
	}
	if len(s.Joueurs) != 1 || len(s.Joueurs[0].Cellules) != 1 ||
		s.Joueurs[0].Cellules[0].Echantillons != 8 {
		t.Fatalf("occupation = %+v, attendu 8 echantillons — sans bornes de vie declarees, "+
			"ce sont les points qui bornent", s.Joueurs)
	}
}

// TestProjeterRasterTactique_SansPisteNommee — UN SIDECAR VIDE MAIS PRESENT. « Mesure a
// zero » et « non mesure » sont deux etats differents, et la lecture d'occupation les
// distingue par la PRESENCE du fichier : la liste doit donc etre `[]`, jamais `null`.
func TestProjeterRasterTactique_SansPisteNommee(t *testing.T) {
	const anonyme = `{"schemaVersion":39,"matchId":"abc","frameCount":21,"frameIntervalMs":100,
      "tracks":[{"slot":1,"team":-1,"points":[{"t":0,"x":1,"y":1},{"t":20,"x":1,"y":1}]}]}`
	s, err := ProjeterRasterTactique(ecrireFichier(t, "anonyme.json", anonyme))
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if len(s.Joueurs) != 0 {
		t.Fatalf("joueurs = %+v, attendu aucun", s.Joueurs)
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("serialisation: %v", err)
	}
	var relu map[string]json.RawMessage
	if err := json.Unmarshal(raw, &relu); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if string(relu["joueurs"]) != "[]" {
		t.Fatalf("joueurs serialise en %s, attendu [] — `null` se lirait comme un champ absent",
			string(relu["joueurs"]))
	}
}

// TestProjeterRasterTactique_Refus — les deux entrees qui ne peuvent rien produire, et qui
// le DISENT (elles comptent en echec, jamais en silence).
func TestProjeterRasterTactique_Refus(t *testing.T) {
	if _, err := ProjeterRasterTactique(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("artefact absent : attendu une erreur")
	}
	if _, err := ProjeterRasterTactique(ecrireFichier(t, "casse.json", `{"schemaVersion":`)); err == nil {
		t.Fatal("artefact illisible : attendu une erreur")
	}
	// Sans matchId, le raster n'a pas de cle : le plancher de rarete se compte en matchs
	// DISTINCTS, et un raster anonyme ne pourrait jamais y entrer.
	if _, err := ProjeterRasterTactique(ecrireFichier(t, "sansid.json", `{"schemaVersion":39,"tracks":[]}`)); err == nil {
		t.Fatal("artefact sans matchId : attendu une erreur")
	}
}

// TestEcrireSidecarRaster_CreeLeDossierEtRelit — le rattrapage tourne sur des postes ou
// aucun sidecar n'a jamais ete ecrit : le dossier doit naitre, et le fichier se relire.
func TestEcrireSidecarRaster_CreeLeDossierEtRelit(t *testing.T) {
	s, err := ProjeterRasterTactique(ecrireFichier(t, "artefact.json", artefactImmobile))
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	root := t.TempDir()
	path := titlePkg.NewPathResolver(root).TacticalRasterPath(titlePkg.DefaultSlug, s.MatchID)
	if err := EcrireSidecarRaster(path, s); err != nil {
		t.Fatalf("ecriture: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	var relu domain.TacticalRasterSidecar
	if err := json.Unmarshal(raw, &relu); err != nil {
		t.Fatalf("deserialisation: %v", err)
	}
	if relu.MatchID != s.MatchID || len(relu.Joueurs) != 2 ||
		relu.Joueurs[0].Cellules[0].Echantillons != 8 {
		t.Fatalf("sidecar relu different de l'ecrit : %+v", relu)
	}
	// Le sidecar est dans le SOUS-dossier `rasters/`, pas au premier niveau du dossier
	// d'artefacts — sans quoi AvailableSet le compterait pour un match et la purge le
	// supprimerait.
	if filepath.Base(filepath.Dir(path)) != "rasters" {
		t.Fatalf("sidecar hors du sous-dossier rasters : %s", path)
	}
}

// TestProjeterRastersTactiques_LotDuCycle — l'etape du fil de l'eau : elle depose ce
// qu'elle peut, et un match en echec n'empeche NI les suivants NI le reste de la cuisson.
func TestProjeterRastersTactiques_LotDuCycle(t *testing.T) {
	root := t.TempDir()
	pr := titlePkg.NewPathResolver(root)
	bon := ecrireFichier(t, "bon.json", artefactImmobile)
	casse := ecrireFichier(t, "casse.json", `{"schemaVersion":`)
	d := Deps{RepoRoot: root, TitleSlug: titlePkg.DefaultSlug, Gamertag: "TestGT"}
	projeterRastersTactiques(context.Background(), d, []artefactCuit{
		{matchID: "aaaaaaaa-0000-0000-0000-000000000000", path: casse},
		{matchID: "000d5950-1234-4abc-9def-0123456789ab", path: bon},
	})
	if _, err := os.Stat(pr.TacticalRasterPath(titlePkg.DefaultSlug, "000d5950")); err != nil {
		t.Fatalf("le sidecar du match sain n'a pas ete depose alors qu'un autre a echoue : %v", err)
	}
	if _, err := os.Stat(pr.TacticalRasterPath(titlePkg.DefaultSlug, "aaaaaaaa")); !os.IsNotExist(err) {
		t.Fatalf("un sidecar a ete depose pour un artefact illisible (err = %v)", err)
	}
}

// TestProjeterRastersTactiques_LotVide — aucun artefact cuit : rien a faire, et surtout
// aucun dossier cree pour rien.
func TestProjeterRastersTactiques_LotVide(t *testing.T) {
	root := t.TempDir()
	projeterRastersTactiques(context.Background(),
		Deps{RepoRoot: root, TitleSlug: titlePkg.DefaultSlug}, nil)
	if _, err := os.Stat(titlePkg.NewPathResolver(root).TacticalRasterDir(titlePkg.DefaultSlug)); !os.IsNotExist(err) {
		t.Fatalf("dossier des rasters cree sans aucun artefact a projeter (err = %v)", err)
	}
}

// ─── LE TEMPS EN VEHICULE, DE BOUT EN BOUT ─────────────────────────────────────

// artefactAvecVehicule : un joueur dont le bipede s'arrete a la frame 10, puis qui roule
// de la frame 10 a la frame 50 (4 s a 100 ms/frame) dans un Warthog qui traverse la carte.
//
// C'EST LA FORME REELLE DU DEFAUT : le trou de 4 s du bipede est ici volontairement court
// (une vie unique) pour que le cas reste lisible ; en production ces trous durent 13 a 36 s
// et coupent la vie en deux, ce qui rend le calque des episodes INDISPENSABLE — sans lui,
// aucune de ces secondes n'est mesuree.
const artefactAvecVehicule = `{
  "schemaVersion": 39,
  "matchId": "cafe0001-0000-0000-0000-000000000000",
  "frameCount": 51,
  "frameIntervalMs": 100,
  "tracks": [
    {"slot":1,"team":-1,"xuid":"111","startFrame":0,"endFrame":10,
     "points":[{"t":0,"x":0.25,"y":0.25},{"t":10,"x":0.25,"y":0.25}]}
  ],
  "vehicles": [
    {"slot":771,"gen":1,"family":"warthog","t0":0,"t1":50,"t1max":50,"end":"unknown",
     "samples":[{"t":10,"x":30.25,"y":0.25},{"t":30,"x":60.25,"y":0.25}],
     "rides":[{"t0":10,"t1":50,"slot":1,"xuid":"111","src":"event"}]}
  ]
}`

// TestProjeterRasterTactique_TempsEnVehicule — LE CONSTAT C1 DE LA REVUE.
//
// Sans le calque des episodes, ce match sortait avec les seuls echantillons du bipede : le
// trajet en vehicule etait perdu en silence, et le match comptait pourtant comme MESURE.
func TestProjeterRasterTactique_TempsEnVehicule(t *testing.T) {
	s, err := ProjeterRasterTactique(ecrireFichier(t, "vehicule.json", artefactAvecVehicule))
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if len(s.Joueurs) != 1 {
		t.Fatalf("joueurs = %+v, attendu 1", s.Joueurs)
	}
	total := 0
	parCellule := map[[2]int]int{}
	for _, c := range s.Joueurs[0].Cellules {
		total += c.Echantillons
		parCellule[[2]int{c.Col, c.Lig}] = c.Echantillons
	}
	// Le bipede couvre [0, 1000 ms) = 4 echantillons en cellule (0,0) ; l'episode couvre
	// [1000, 5000 ms) = 16 echantillons sur les cellules du VEHICULE.
	if total != 20 {
		t.Fatalf("echantillons = %d, attendu 20 (4 de bipede + 16 de vehicule) : le temps "+
			"en vehicule est perdu", total)
	}
	if parCellule[[2]int{0, 0}] != 4 {
		t.Fatalf("cellule du bipede = %d echantillons, attendu 4", parCellule[[2]int{0, 0}])
	}
	// x=30,25 -> col 60 ; x=60,25 -> col 120.
	if parCellule[[2]int{60, 0}] == 0 || parCellule[[2]int{120, 0}] == 0 {
		t.Fatalf("les cellules du vehicule ne sont pas peintes : %+v", s.Joueurs[0].Cellules)
	}
	// UN EMBARQUEMENT NE CREE AUCUN SPAWN : il ne doit y en avoir qu'un, celui de la vie.
	if len(s.Joueurs[0].Spawns) != 1 || s.Joueurs[0].Spawns[0].Frame != 0 {
		t.Fatalf("spawns = %+v, attendu le seul spawn de la vie", s.Joueurs[0].Spawns)
	}
}

// TestProjeterRasterTactique_VehiculeSansOccupantNomme — un episode dont le film n'a pas
// nomme l'occupant n'attribue ce temps a personne : le vehicule EST occupe, son occupant
// est inconnu.
func TestProjeterRasterTactique_VehiculeSansOccupantNomme(t *testing.T) {
	const anonyme = `{"schemaVersion":39,"matchId":"abc","frameCount":51,"frameIntervalMs":100,
      "tracks":[],
      "vehicles":[{"slot":771,"gen":1,"t0":0,"t1":50,"t1max":50,"end":"unknown",
        "samples":[{"t":10,"x":30.25,"y":0.25}],
        "rides":[{"t0":10,"t1":50,"slot":1,"src":"gap"}]}]}`
	s, err := ProjeterRasterTactique(ecrireFichier(t, "anon.json", anonyme))
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if len(s.Joueurs) != 0 {
		t.Fatalf("joueurs = %+v, attendu aucun", s.Joueurs)
	}
}

// TestProjeterRasterTactique_PointsIgnoresEstStructurellementNul — C9, ET CE QUE LA
// MESURE A REVELE.
//
// Le sidecar TRANSPORTE `points_ignores` (les positions non finies ecartees), parce que la
// lecture ne peut pas le recalculer : elle somme des comptes deja groupes PAR CELLULE, et
// un point ecarte n'a jamais eu de cellule.
//
// MAIS IL VAUT TOUJOURS 0 AUJOURD'HUI, ET C'EST STRUCTUREL : `replay.Point.X/Y` et
// `replay.VehicleSample.X/Y` sont des `float32`, et JSON ne peut exprimer aucune valeur non
// finie — une coordonnee qui deborde float32 fait echouer la DESERIALISATION DE TOUT
// L'ARTEFACT (mesure : `json: cannot unmarshal number 1e39 into ... float32`), donc la
// projection entiere echoue avant d'ecarter quoi que ce soit. Le garde de
// `Grille.Cellule` et ce transport restent : ils tiennent le contrat du champ, et le jour
// ou un producteur autre que ce JSON alimenterait le sidecar, la mesure serait vraie sans
// rien changer. Ce test-ci FIGE le fait, pour qu'il ne soit pas redecouvert comme un bug.
func TestProjeterRasterTactique_PointsIgnoresEstStructurellementNul(t *testing.T) {
	s, err := ProjeterRasterTactique(ecrireFichier(t, "artefact.json", artefactImmobile))
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if s.PointsIgnores != 0 {
		t.Fatalf("points_ignores = %d : un artefact JSON ne peut porter aucune position "+
			"non finie (float32), ce compte ne peut pas etre non nul", s.PointsIgnores)
	}
	// LA PREUVE DE L'IMPOSSIBILITE, jouee : une coordonnee non finie n'est pas « un point
	// ecarte », c'est un artefact ILLISIBLE — et cela se compte en echec, pas en silence.
	const horsBornes = `{"schemaVersion":39,"matchId":"abc","frameCount":21,"frameIntervalMs":100,
      "tracks":[{"slot":1,"team":-1,"xuid":"111","startFrame":0,"endFrame":20,
        "points":[{"t":0,"x":1e39,"y":0.25}]}]}`
	if _, err := ProjeterRasterTactique(ecrireFichier(t, "hb.json", horsBornes)); err == nil {
		t.Fatal("une coordonnee hors bornes float32 doit faire echouer la projection entiere")
	}
}

// TestSidecarSchemaVersion_SuitLaFormule — la version du FORMAT couvre la FORMULE : le
// calque des embarquements change ce que `echantillons` compte, donc tout sidecar
// anterieur est perime et doit etre reecrit par le rattrapage. La laisser a 1 aurait
// laisse coexister deux mesures differentes sous le meme nom.
func TestSidecarSchemaVersion_SuitLaFormule(t *testing.T) {
	if domain.TacticalRasterSchemaVersion < 2 {
		t.Fatalf("schema du sidecar = %d : l'ajout du temps en vehicule change la formule, "+
			"la version doit avoir bouge", domain.TacticalRasterSchemaVersion)
	}
	s, err := ProjeterRasterTactique(ecrireFichier(t, "artefact.json", artefactImmobile))
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if s.SchemaVersion != domain.TacticalRasterSchemaVersion {
		t.Fatalf("schema_version ecrit = %d", s.SchemaVersion)
	}
}
