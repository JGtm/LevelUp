package filmdec

// vitesse_filtre_artefact_research_test.go — lecture de l'ARTEFACT PUBLIÉ pour
// TestVitesseFiltre (lot R3) : chargement du document de rejeu, appariement de piste,
// mesure du retard de publication au lieu d'arrivée et des trous de frames. La question
// de recherche et l'usage sont documentés en tête de vitesse_filtre_research_test.go.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// vitfArt est la part de l'artefact de rejeu que la mesure lit (document.go fait foi sur
// les champs ; points {t,x,y} datés en frames : US_film = (originMs + t*frameIntervalMs)*1000).
type vitfArt struct {
	FrameIntervalMS int            `json:"frameIntervalMs"`
	OriginMs        *int64         `json:"originMs"`
	Tracks          []vitfArtTrack `json:"tracks"`
}

type vitfArtTrack struct {
	Slot   uint32         `json:"slot"`
	Points []vitfArtPoint `json:"points"`
}

type vitfArtPoint struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// vitfChargerArtefact lit VITESSE_ARTEFACT ; absent = pas de mesure artefact (le dire).
func vitfChargerArtefact(t *testing.T) *vitfArt {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(vitfArtefactEnv))
	if path == "" {
		t.Logf("%s absent : pas de mesure sur l'artefact publié", vitfArtefactEnv)
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s=%q illisible : %v", vitfArtefactEnv, path, err)
	}
	var art vitfArt
	if err := json.Unmarshal(raw, &art); err != nil {
		t.Fatalf("artefact %s : JSON illisible : %v", path, err)
	}
	if art.OriginMs == nil || art.FrameIntervalMS <= 0 {
		t.Fatalf("artefact %s sans originMs/frameIntervalMs : conversion en frames impossible", path)
	}
	return &art
}

// vitfPiste rend la piste du slot dont les points couvrent la frame (une piste = une vie).
func vitfPiste(art *vitfArt, slot uint32, frame int) *vitfArtTrack {
	for i := range art.Tracks {
		tr := &art.Tracks[i]
		if tr.Slot != slot || len(tr.Points) == 0 {
			continue
		}
		if tr.Points[0].T <= frame && tr.Points[len(tr.Points)-1].T >= frame+1 {
			return tr
		}
	}
	return nil
}

// vitfMesureArtefact mesure dans l'ARTEFACT PUBLIÉ : le retard entre la frame de
// l'événement et la première position publiée au lieu d'arrivée (<= vitfArriveeM en 2D,
// comme translocArriveM), le plus grand pas de position dans [frame, frame+5], les trous
// de frames à ±10, et une CALIBRATION (distance piste/échantillon film d'avant-saut : si
// elle est grande, l'instrument et l'artefact ne parlent pas des mêmes coordonnées et la
// mesure serait fausse).
func vitfMesureArtefact(t *testing.T, art *vitfArt, m *vitfMesure, origine uint64) {
	t.Helper()
	_ = origine // le filmMS de m est déjà sur l'horloge film
	m.frameEv = int((m.filmMS - *art.OriginMs) / int64(art.FrameIntervalMS))
	tr := vitfPiste(art, m.ev.slot, m.frameEv)
	if tr == nil {
		t.Logf("  [@%d ms slot %d] artefact : AUCUNE piste ne couvre la frame %d", m.filmMS, m.ev.slot, m.frameEv)
		return
	}
	m.artOK = true
	arr := [2]float32{m.posArr[0], m.posArr[1]}
	var avant *vitfArtPoint
	for k := range tr.Points {
		pt := &tr.Points[k]
		if pt.T <= m.frameEv {
			avant = pt
		}
		if k > 0 {
			a, b := tr.Points[k-1], tr.Points[k]
			if b.T >= m.frameEv && b.T <= m.frameEv+5 {
				if d := vitfDist2D(a.X, a.Y, b.X, b.Y); d > m.sautArtM {
					m.sautArtM, m.retardSautFr = d, b.T-m.frameEv
				}
			}
			for f := a.T + 1; f < b.T; f++ {
				if f >= m.frameEv-10 && f <= m.frameEv+10 {
					m.trous++
					m.trousListe = append(m.trousListe, f)
				}
			}
		}
		if m.retardArrFr < 0 && pt.T >= m.frameEv && vitfDist2D(pt.X, pt.Y, arr[0], arr[1]) <= vitfArriveeM {
			m.retardArrFr = pt.T - m.frameEv
			m.deplArtM = vitfDist2D(pt.X, pt.Y, arr[0], arr[1])
		}
	}
	if avant != nil {
		m.calibM = vitfDist2D(avant.X, avant.Y, m.posAvant[0], m.posAvant[1])
	}
}

func vitfDist2D(ax, ay, bx, by float32) float64 {
	return translocDist([3]float32{ax, ay, 0}, [3]float32{bx, by, 0})
}
