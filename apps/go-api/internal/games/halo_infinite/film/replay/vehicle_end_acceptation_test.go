package replay

// vehicle_end_acceptation_test.go — LE TEST D ACCEPTATION DU LOT 1.9.10, sur l artefact CUIT.
//
// POURQUOI SUR L ARTEFACT ET NON SUR LE FILM. Ce que le lot doit prouver est ce que le DOCUMENT
// publie — la fin de vie d un vehicule, sa cause et son instant. Le lire sur l artefact frais
// fait passer le test par la chaine de cuisson ENTIERE (profil, balayages, assemblage,
// serialisation), la ou une lecture directe du film re-plomberait ici la moitie du builder et
// prouverait autre chose. L artefact vient du gate de corpus, joue sous « voie libre » :
//
//	go run ./cmd/replay-corpus-gate --base=<sha> --parc-root <principal> --source-root <worktree> \
//	  --keep-work --json <rapport>
//	LOT1910_ARTEFACT=<racine de travail>/.../bfecd02b.json go test \
//	  ./internal/games/halo_infinite/film/replay/ -run Acceptation1910 -v -count=1
//
// LE TEMOIN EST `bfecd02b`, DESIGNE PAR L UTILISATEUR (2026-09-14) : son `ghost` de slot 777
// cessait d etre dessine a 287,4 s. La question que ce test tranche, et il la tranche dans les
// deux sens :
//
//	le film ECRIT la mort de cette vie  -> `end = destroyed`, `tEnd` la date, et la borne
//	                                      d affichage ne la depasse pas. L effacement du sprite
//	                                      est alors JUSTE, et il est EXPLIQUE ;
//	le film ne l ecrit PAS              -> la vie ne doit etre coupee par AUCUNE inference : sa
//	                                      borne haute est celle du recensement, sa cause est
//	                                      `unknown`, et le defaut restant est la LECTURE DES
//	                                      ECHANTILLONS d un vehicule occupe — que le test NOMME
//	                                      au lieu de la laisser passer.
//
// GARDE PAR `LOT1910_ARTEFACT` : sans artefact frais, il se saute. Il ne lit ni le cache de
// films ni le parc.

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"levelup/go-api/internal/domain/replaydoc"
)

// acceptation1910Env : l artefact CUIT sur lequel le test d acceptation porte.
const acceptation1910Env = "LOT1910_ARTEFACT"

// acceptation1910Slot : le `ghost` que l utilisateur a nomme.
const acceptation1910Slot = uint32(777)

// TestAcceptation1910GhostDeBfecd02b : la fin de vie publiee vient-elle d une LECTURE ?
func TestAcceptation1910GhostDeBfecd02b(t *testing.T) {
	path := os.Getenv(acceptation1910Env)
	if path == "" {
		t.Skipf("%s absent : test d acceptation saute (il exige un artefact CUIT)", acceptation1910Env)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l operateur du gate
	if err != nil {
		t.Fatalf("artefact illisible : %v", err)
	}
	var doc replaydoc.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("artefact non deserialisable : %v", err)
	}
	if len(doc.Vehicles) == 0 {
		t.Fatalf("l artefact %s ne publie aucun vehicule", doc.MatchID)
	}
	acceptation1910Rapport(t, doc)
	vies := acceptation1910ViesDuSlot(doc)
	if len(vies) == 0 {
		t.Fatalf("aucune vie de slot %d dans %s : le temoin a change de forme",
			acceptation1910Slot, doc.MatchID)
	}
	for _, v := range vies {
		acceptation1910VerifierUneVie(t, v, doc.FrameCount)
	}
}

// acceptation1910ViesDuSlot rend les vies publiees du slot temoin.
func acceptation1910ViesDuSlot(doc replaydoc.ReplayDocument) []replaydoc.VehicleTrack {
	var out []replaydoc.VehicleTrack
	for _, v := range doc.Vehicles {
		if v.Slot == acceptation1910Slot {
			out = append(out, v)
		}
	}
	return out
}

// acceptation1910Rapport colle la table des vies : elle est la mesure, le verdict n en est que
// la conclusion.
func acceptation1910Rapport(t *testing.T, doc replaydoc.ReplayDocument) {
	t.Helper()
	t.Logf("%s — %d frames, %d vies de vehicule", doc.MatchID, doc.FrameCount, len(doc.Vehicles))
	if c := doc.Coverage; c != nil && c.Vehicles != nil {
		v := c.Vehicles
		t.Logf("  morts lues %d (reprises %d, orphelines %d, queue inconnue %d) · fins :"+
			" destroyed %d · film_end %d · unknown %d · echantillons apres la fin %d",
			v.DeathsRead, v.DeathsMatched, v.DeathsUnmatched, v.DeathsTailDesync,
			v.EndDestroyed, v.EndFilmEnd, v.EndUnknown, v.SamplesAfterEnd)
	}
	for _, v := range doc.Vehicles {
		fin := "-"
		if v.TEnd != nil {
			fin = strconv.Itoa(*v.TEnd)
		}
		t.Logf("  slot %4d gen %d %-10s t0=%-5d t1=%-5d t1max=%-5d end=%-9s tEnd=%-5s"+
			" echantillons=%d episodes=%d",
			v.Slot, v.Gen, v.Family, v.T0, v.T1, v.T1Max, v.End, fin, len(v.Samples), len(v.Rides))
	}
}

// acceptation1910VerifierUneVie applique la disjonction du lot a UNE vie du temoin.
func acceptation1910VerifierUneVie(t *testing.T, v replaydoc.VehicleTrack, frames int) {
	t.Helper()
	switch v.End {
	case VehicleEndDestroyed:
		if v.TEnd == nil {
			t.Errorf("slot %d gen %d : end=destroyed sans tEnd — une destruction publiee sans"+
				" sa date n est pas une lecture", v.Slot, v.Gen)
			return
		}
		if *v.TEnd < v.T0 {
			t.Errorf("slot %d gen %d : tEnd=%d precede la naissance t0=%d — la mort attribuee"+
				" n appartient pas a cette vie", v.Slot, v.Gen, *v.TEnd, v.T0)
		}
		// LA BORNE D AFFICHAGE PEUT DEPASSER LA MORT ECRITE, ET LA MESURE L IMPOSE.
		//
		// CETTE ASSERTION A ETE ECRITE AVANT LA MESURE, ET LA MESURE L A REFUTEE. Elle exigeait
		// `t1max <= tEnd`. Sur `bfecd02b` le ghost 777 est ecrit DETRUIT a 274,0 s et le film
		// REPLIQUE ENCORE SA POSITION jusqu a 282,1 s — 8,1 s d epave, exactement le profil de
		// mise au repos mesure au lot V3 (13 a 36 s apres l abandon). Affirmer l absence a 274,0 s
		// contredirait des positions que le film ecrit ; le depot publie la contradiction et la
		// compte (`coverage.vehicles.samplesAfterEnd`), il ne l arbitre pas en silence.
		//
		// CE QUI RESTE INTERDIT, et c est le vrai defaut que ce lot retire : une borne d affichage
		// qui depasse A LA FOIS la mort ecrite ET la derniere position repliquee — c est-a-dire
		// une fin INFEREE. Quand `t1max` passe `tEnd`, il doit s arreter EXACTEMENT sur `t1`.
		if v.T1Max > *v.TEnd && v.T1Max != v.T1 {
			t.Errorf("slot %d gen %d : t1max=%d depasse la fin ecrite tEnd=%d SANS s arreter sur la"+
				" derniere position t1=%d — la borne d affichage est inferee, pas lue",
				v.Slot, v.Gen, v.T1Max, *v.TEnd, v.T1)
		}
	case VehicleEndFilmEnd:
		if v.T1Max != frames-1 {
			t.Errorf("slot %d gen %d : end=film_end mais t1max=%d au lieu de %d — une vie que la"+
				" derniere image-cle recense encore ne se coupe pas avant la fin du film",
				v.Slot, v.Gen, v.T1Max, frames-1)
		}
	default:
		// LE CAS QUI INSTRUIT. Le film n ecrit pas la mort de cette vie : la borne haute doit
		// venir du RECENSEMENT et de rien d autre. Si le sprite s efface quand meme avant la
		// fin du film, ce qui reste a corriger est la LECTURE DES ECHANTILLONS d un vehicule
		// occupe — et c est ce que ce message nomme, plutot que de laisser passer.
		t.Errorf("slot %d gen %d : end=%q — le film n ECRIT PAS la fin de cette vie."+
			" Bornes publiees t0=%d t1=%d t1max=%d sur %d frames, %d echantillons."+
			" A instruire : le canal des positions d un vehicule OCCUPE (replay/vehicle_rides.go,"+
			" build_vehicles.go) — la disparition n est ni datee ni expliquee",
			v.Slot, v.Gen, v.End, v.T0, v.T1, v.T1Max, frames, len(v.Samples))
	}
}
