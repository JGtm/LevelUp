package service

// replay_weapon_labels_completion_test.go — LES ARTEFACTS DÉJÀ CUITS APPRENNENT LES ARMES
// QUE LE REGISTRE VIENT D'APPRENDRE.
//
// Le Mutilator (famille 0xD7915565) était nommé dans le seed des libellés depuis avril, mais
// son identifiant n'était pas au registre filmshell : les artefacts cuits avant le 2026-09-13
// ne portent AUCUN libellé pour lui, et le rejeu comme la vue de match affichaient
// « 0xD7915565 » sur son socle. La correction du registre ne suffit pas — sans complétion à la
// requête, il aurait fallu recuire tous les artefacts.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// familleMutilator — la famille d'arme telle que le document l'écrit (8 chiffres, majuscules).
const familleMutilator = "0xD7915565"

// artefactSocleMutilator pose un artefact CUIT AVANT que le registre connaisse le Mutilator :
// son socle porte la famille, et la table de libellés ne la nomme pas. C'est l'état réel des
// artefacts locaux et de production (vérifié sur bc60b4d9, cf. plus bas).
func artefactSocleMutilator(t *testing.T, root, titleSlug, matchID string) {
	t.Helper()
	doc := `{"schemaVersion":54,"matchId":"` + matchID + `","titleSlug":"` + titleSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],` +
		`"weaponPads":[{"x":0,"y":0,"weapon":"` + familleMutilator + `","spawns":[0]}],` +
		`"weaponLabels":{"0x48C19D2D":{"en":"MA40 AR","fr":"MA40 AR","fx":"ballistic"}}}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(titleSlug, matchID), doc)
}

// TestWeaponLabels_CompleteUnArtefactCuitAvantLeRegistre : le socle d'une arme entrée au
// registre APRÈS la cuisson est nommé à la requête, et le libellé déjà cuit n'est pas touché.
func TestWeaponLabels_CompleteUnArtefactCuitAvantLeRegistre(t *testing.T) {
	root := t.TempDir()
	mappingsReels(t, root, title.DefaultSlug)
	artefactSocleMutilator(t, root, title.DefaultSlug, "match-mutilateur")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-mutilateur")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	lbl, ok := doc.WeaponLabels[familleMutilator]
	if !ok {
		t.Fatalf("%s absent de la table des libellés — le socle afficherait son hexadécimal",
			familleMutilator)
	}
	if lbl.Fr != "Mutilateur" || lbl.En != "Mutilator" {
		t.Errorf("libellé = %q / %q, attendu « Mutilator » / « Mutilateur »", lbl.En, lbl.Fr)
	}
	if lbl.Key != "hinf_mutilator" {
		t.Errorf("clé = %q, attendu hinf_mutilator", lbl.Key)
	}
	if lbl.Role != "shotgun" {
		t.Errorf("rôle = %q, attendu shotgun (registre canonique)", lbl.Role)
	}
	if lbl.Img == "" {
		t.Errorf("aucune vignette pour %s — le calque des socles dessinerait un glyphe neutre "+
			"alors que le titre sert l'icône de cette famille", familleMutilator)
	}
	// Le libellé DÉJÀ CUIT reste celui de l'artefact : la complétion comble, elle n'écrase pas.
	if got := doc.WeaponLabels["0x48C19D2D"].Fr; got != "MA40 AR" {
		t.Errorf("libellé cuit du MA40 = %q, attendu MA40 AR (intouché)", got)
	}
}

// TestWeaponLabels_CompleteUnArtefactSansAucunLibelle : le cas le plus dégradé — la table est
// ABSENTE du fichier. C'était le seul cas non réparable tant que la résolution s'arrêtait sur
// `len(WeaponLabels) == 0`.
func TestWeaponLabels_CompleteUnArtefactSansAucunLibelle(t *testing.T) {
	root := t.TempDir()
	mappingsReels(t, root, title.DefaultSlug)
	doc := `{"schemaVersion":54,"matchId":"m","titleSlug":"` + title.DefaultSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],` +
		`"weaponPads":[{"x":0,"y":0,"weapon":"` + familleMutilator + `","spawns":[0]}]}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, "match-nu"), doc)

	served, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-nu")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if got := served.WeaponLabels[familleMutilator].Fr; got != "Mutilateur" {
		t.Errorf("libellé FR = %q, attendu Mutilateur (table absente du fichier : elle se crée)", got)
	}
}

// artefactLocalMutilator — l'artefact RÉEL du poste dont le socle de Mutilator s'affichait en
// hexadécimal (match bc60b4d9-40dc-4790-a533-3197aea9060a). Les artefacts ne sont pas versionnés :
// ce test est une VÉRIFICATION DE POSTE, il se saute là où le fichier n'existe pas (CI comprise).
// Chemin surchargeable pour le rejouer depuis un autre arbre de travail.
const envArtefactLocal = "LEVELUP_REPLAY_ARTEFACT_MUTILATOR"

func cheminArtefactLocal(t *testing.T) string {
	t.Helper()
	if p := os.Getenv(envArtefactLocal); p != "" {
		return p
	}
	// 1. l'arbre de travail courant, quand il porte les caches de rejeu ;
	local := title.NewPathResolver(depotRacine(t)).
		ReplayArtifactPath(title.DefaultSlug, "bc60b4d9-40dc-4790-a533-3197aea9060a")
	if _, err := os.Stat(local); err == nil {
		return local
	}
	// 2. l'arbre principal du poste, seul à porter `data/cache/replays` (les arbres de travail
	//    secondaires n'ont que les données versionnées).
	return filepath.Join("C:\\", "Users", "Guillaume", "Downloads", "Scripts",
		"LevelUp-go-migration", "data", "cache", "replays", "halo_infinite", "bc60b4d9.json")
}

// TestWeaponLabels_ArtefactReelDuPoste : la preuve sur pièce. Cet artefact porte la famille
// 0xD7915565 sur un socle ET dans des loadouts, sans aucun libellé pour elle — c'est
// exactement ce que l'utilisateur voyait à l'écran.
func TestWeaponLabels_ArtefactReelDuPoste(t *testing.T) {
	src := cheminArtefactLocal(t)
	brut, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("artefact local absent (%s) — vérification de poste uniquement", src)
	}

	root := t.TempDir()
	mappingsReels(t, root, title.DefaultSlug)
	const matchID = "bc60b4d9-40dc-4790-a533-3197aea9060a"
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, matchID), string(brut))

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), matchID)
	if err != nil {
		t.Fatalf("lecture du rejeu réel : %v", err)
	}

	var socle bool
	for _, p := range doc.WeaponPads {
		if p.Weapon == familleMutilator {
			socle = true
		}
	}
	if !socle {
		t.Fatalf("cet artefact ne porte plus de socle %s — choisir un autre témoin", familleMutilator)
	}
	lbl, ok := doc.WeaponLabels[familleMutilator]
	if !ok {
		t.Fatalf("%s toujours sans libellé sur l'artefact réel : le socle afficherait son "+
			"hexadécimal à l'écran", familleMutilator)
	}
	if lbl.Fr != "Mutilateur" {
		t.Errorf("libellé FR servi = %q, attendu Mutilateur", lbl.Fr)
	}
	if lbl.Key != "hinf_mutilator" {
		t.Errorf("clé servie = %q, attendu hinf_mutilator", lbl.Key)
	}
}
