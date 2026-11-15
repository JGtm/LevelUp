package service

// replay_destruction_causes_test.go — la CLÉ D'ARME de la cause d'une destruction de véhicule, posée
// à la requête. Chaque test correspond à une décision écrite dans l'en-tête du fichier de production :
// résolue, cause absente, cause inconnue = vide, jamais la clé d'une voisine, jamais une clé cuite.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/killicon"
)

const (
	causeConnue   uint32 = 1001 // résolue par le bouchon : arme avec clé et image
	causeSansCle  uint32 = 1002 // image sans clé de registre (grenade, mêlée)
	causeInconnue uint32 = 36   // le bouchon n'identifie pas cette source
	causeSansImg  uint32 = 1003 // clé sans image : le pictogramme n'existerait pas
)

// bouchonSources : un résolveur de source de dégât à table fixe, sans aucun titre.
type bouchonSources map[uint32]canonical.KillSourceIcon

func (b bouchonSources) KillSourceIcon(tag uint32) (canonical.KillSourceIcon, bool) {
	ic, ok := b[tag]
	return ic, ok
}

func sourcesDeTest() bouchonSources {
	return bouchonSources{
		causeConnue:  {WeaponKey: "cle_laser", ImageURL: "/static/laser.png", Tinted: true},
		causeSansCle: {ImageURL: "/static/grenade.png", Tinted: true},
		causeSansImg: {WeaponKey: "cle_sans_image"},
	}
}

// artefactDestructions pose cinq vies détruites : une cause résolue, une cause sans clé, une cause
// inconnue, une destruction sans cause portant une clé périmée dans l'artefact, et une vie sans bloc.
func artefactDestructions(t *testing.T, root, matchID string) {
	t.Helper()
	doc := `{"schemaVersion":29,"matchId":"` + matchID + `","titleSlug":"` + title.DefaultSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],` +
		`"vehicles":[` +
		`{"slot":1,"gen":0,"family":"ghost","t0":0,"t1":1,"t1max":1,"end":"destroyed","tEnd":1,"destruction":{"cause":1001}},` +
		`{"slot":2,"gen":0,"family":"ghost","t0":0,"t1":1,"t1max":1,"end":"destroyed","tEnd":1,"destruction":{"cause":1002}},` +
		`{"slot":3,"gen":0,"family":"ghost","t0":0,"t1":1,"t1max":1,"end":"destroyed","tEnd":1,"destruction":{"cause":36}},` +
		`{"slot":4,"gen":0,"family":"ghost","t0":0,"t1":1,"t1max":1,"end":"destroyed","tEnd":1,"destruction":{"causeKey":"cle_perimee"}},` +
		`{"slot":5,"gen":0,"family":"ghost","t0":0,"t1":1,"t1max":1,"end":"inconnue"},` +
		`{"slot":6,"gen":0,"family":"ghost","t0":0,"t1":1,"t1max":1,"end":"destroyed","tEnd":1,"destruction":{"cause":1003}}]}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, matchID), doc)
}

func clesDesCauses(t *testing.T, svc port.ReplayService, matchID string) []string {
	t.Helper()
	doc, err := svc.GetReplay(context.Background(), matchID)
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	out := make([]string, 0, len(doc.Vehicles))
	for _, v := range doc.Vehicles {
		if v.Destruction == nil {
			out = append(out, "<sans bloc>")
			continue
		}
		out = append(out, v.Destruction.CauseKey)
	}
	return out
}

func TestDestructionCauses_PoseLaCleDeLArmeResolue(t *testing.T) {
	root := t.TempDir()
	artefactDestructions(t, root, "a0000010")
	svc := NewReplayService(title.DefaultSlug, root, nil, nil, WithReplayKillSourceIcons(sourcesDeTest()))

	got := clesDesCauses(t, svc, "a0000010")
	want := []string{
		"cle_laser", // cause résolue
		"",          // image sans clé : pas de clé, jamais celle d'une voisine
		"",          // cause inconnue du titre (36)
		"",          // pas de cause : la clé périmée de l'artefact ne traverse pas
		"<sans bloc>",
		"", // clé sans image : le pictogramme n'existerait pas
	}
	if len(got) != len(want) {
		t.Fatalf("%d vies servies, attendu %d : %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("vie %d : causeKey = %q, attendu %q", i+1, got[i], want[i])
		}
	}
}

func TestDestructionCauses_SansResolveurAucuneCle(t *testing.T) {
	root := t.TempDir()
	artefactDestructions(t, root, "a0000011")
	for nom, svc := range map[string]port.ReplayService{
		"sans option":   NewReplayService(title.DefaultSlug, root, nil, nil),
		"résolveur nul": NewReplayService(title.DefaultSlug, root, nil, nil, WithReplayKillSourceIcons(nil)),
		"table vide":    NewReplayService(title.DefaultSlug, root, nil, nil, WithReplayKillSourceIcons(bouchonSources{})),
	} {
		for i, k := range clesDesCauses(t, svc, "a0000011") {
			if k != "" && k != "<sans bloc>" {
				t.Errorf("%s : vie %d porte la clé %q, attendu aucune", nom, i+1, k)
			}
		}
	}
}

// TestDestructionCauses_AdapterDuTitre : avec l'adapter d'assets réel du titre, la clé posée est
// exactement celle que les lignes d'élimination publient pour le même tag ; une source que le titre ne
// sait pas nommer ne reçoit rien.
func TestDestructionCauses_AdapterDuTitre(t *testing.T) {
	var tag uint32
	var cle string
	for _, id := range killicon.ResolvedTags() {
		if ic, _ := killicon.Lookup(id); ic.WeaponKey != "" {
			tag, cle = id, ic.WeaponKey
			break
		}
	}
	if cle == "" {
		t.Fatal("aucune règle du dictionnaire embarqué ne porte de clé d'arme : le test ne peut rien prouver")
	}
	s := &replayService{titleSlug: title.DefaultSlug, killIcons: halo_infinite.NewAssetURLAdapter()}
	if got := s.weaponKeyOfCause(tag); got != cle {
		t.Errorf("clé de la cause %d = %q, attendu %q (celle des lignes d'élimination)", tag, got, cle)
	}
	if got := s.weaponKeyOfCause(causeInconnue); got != "" {
		t.Errorf("clé de la cause inconnue %d = %q, attendu vide", causeInconnue, got)
	}
}
