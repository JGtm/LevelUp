package replay

// document_knockbacks.go — LES POUSSÉES D'ÉQUIPEMENT (le répulseur), datées et attribuées par la
// liste d'événements du film (schéma 94).
//
// CE QUE C'EST. Le film écrit une poussée comme une PAIRE de messages de la vue A
// (grammar/vue_a_poussees.go) : `EquipmentObjectKnockedBack` (105) nomme l'objet poussé et l'unité
// qui pousse, puis `EquipmentKnockbackPlayer` (104), à sa suite dans la même liste, nomme le joueur
// poussé et porte l'impulsion qui lui est appliquée. Ce calque publie la paire sur l'axe de frames :
// QUI est poussé, PAR QUI, QUAND, et la vitesse imposée. Un 104 sans son 105 n'est pas publié
// (`coverage.knockbacks.unpaired`).
//
// CE QUE ÇA NE DONNE PAS. Un usage du répulseur qui ne pousse aucun joueur (dans le vide, ou sur
// un objet seul) n'écrit pas de 104 : il n'est pas ici. Une liste dont la lecture s'arrête avant
// ses messages de poussée ne les rend pas non plus.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Knockback est UNE poussée, sur l'axe de frames du document.
type Knockback struct {
	// T est l'index de frame du paquet qui écrit la paire, sur le même axe que Point.T. La piste
	// du joueur poussé accélère dans les trois frames qui suivent.
	T int `json:"t"`
	// Slot désigne la Track du joueur poussé.
	Slot uint32 `json:"slot"`
	// By désigne la Track de l'unité qui pousse ; absent quand le 105 ne la nomme pas ou que sa
	// piste n'est pas publiée (`coverage.knockbacks.withoutAuthor`).
	By *uint32 `json:"by,omitempty"`
	// VX/VY/VZ est la vitesse imposée au joueur poussé, en m/s, mêmes axes que Point.X/Y/Z.
	// SOLIDAIRES : absents ensemble quand le 104 ne porte pas d'impulsion.
	VX *float32 `json:"vx,omitempty"`
	VY *float32 `json:"vy,omitempty"`
	VZ *float32 `json:"vz,omitempty"`
}

// KnockbackCoverage dit ce que le calque a lu et ce qu'il a écarté.
type KnockbackCoverage struct {
	// Read104 et Read105 : messages des deux genres lus dans les vues A du film.
	Read104 int `json:"read104"`
	Read105 int `json:"read105"`
	// Paired : les 104 précédés de leur 105 — les poussées lues ; Unpaired : les autres.
	Paired   int `json:"paired"`
	Unpaired int `json:"unpaired"`
	// Published : les poussées publiées dans le document.
	Published int `json:"published"`
	// BeforeOrigin : poussées antérieures à la première frame — écartées.
	BeforeOrigin int `json:"beforeOrigin"`
	// NoIdentity : poussées dont le joueur poussé n'a pas de piste publiée — écartées.
	NoIdentity int `json:"noIdentity"`
	// WithoutAuthor : poussées publiées sans `by` (auteur non nommé, ou sans piste publiée).
	WithoutAuthor int `json:"withoutAuthor"`
}

// buildKnockbacks projette les poussées lues sur l'axe de frames : rien avant l'origine, rien dont
// le joueur poussé n'a pas de piste publiée.
func buildKnockbacks(evts []types.Knockback, st types.KnockbackStats, tracks []Track, origin, step uint64,
) ([]Knockback, KnockbackCoverage) {
	cov := KnockbackCoverage{Read104: st.Read104, Read105: st.Read105, Paired: st.Paired, Unpaired: st.Unpaired}
	if len(evts) == 0 || step == 0 {
		return nil, cov
	}
	published := publishedSlots(tracks)
	var out []Knockback
	for _, e := range evts {
		if e.TimestampUS < origin {
			cov.BeforeOrigin++
			continue
		}
		if !published[e.Pushed.Slot] {
			cov.NoIdentity++
			continue
		}
		k := Knockback{T: int((e.TimestampUS - origin) / step), Slot: e.Pushed.Slot}
		if e.HasAuthor && published[e.Author.Slot] {
			by := e.Author.Slot
			k.By = &by
		} else {
			cov.WithoutAuthor++
		}
		if e.HasImpulse {
			k.VX, k.VY, k.VZ = coordPtr(e.Impulse[0]), coordPtr(e.Impulse[1]), coordPtr(e.Impulse[2])
		}
		out = append(out, k)
		cov.Published++
	}
	return out, cov
}

// logKnockbackCoverage sort la couverture du calque.
func logKnockbackCoverage(ctx context.Context, cov KnockbackCoverage) {
	slog.InfoContext(ctx, "rejeu : poussees d equipement",
		"lus104", cov.Read104, "lus105", cov.Read105, "appariees", cov.Paired, "nonAppariees", cov.Unpaired,
		"publiees", cov.Published, "avantOrigine", cov.BeforeOrigin, "sansIdentite", cov.NoIdentity,
		"sansAuteur", cov.WithoutAuthor)
}
