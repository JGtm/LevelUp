package service

// replay_shield_cycles.go — LES BRIS ET LES RECHARGES DE BOUCLIER DE CHAQUE JOUEUR, POSES A LA REQUETE.
//
// La definition d'un bris et d'une recharge est celle de `analysis/shieldcycle`, lue par l'adaptateur
// `replayshield` : le resume d'usage et la fiche du rejeu n'ont pas deux calculs. Le document servi porte
// les INSTANTS (millisecondes sur l'axe du document), pas des totaux : la fiche compte ceux qui precedent
// la tete de lecture, et le total du match est la longueur de la liste.
//
// UN BOT N'A PAS DE XUID : il est servi sous la cle `bot:<nom>`, celle que le web donne a son entree de
// roster (`botKey`, lib/replay/rosterLogic.ts) ; deux bots homonymes se confondent, comme dans le roster.
//
// Un joueur que le film ne mesure pas n'a aucune entree (« non mesure ») ; un joueur mesure sans
// evenement a deux listes vides. Un document sans `frameIntervalMs` n'a pas d'horloge en millisecondes :
// aucune entree non plus, la fiche ne montre alors pas de bouclier plutot qu'un instant devine.

import (
	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"levelup/go-api/internal/analysis/shieldcycle"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/replayshield"
)

// botKeyPrefix prefixe le nom d'un bot pour en faire la cle de son entree (cf. l'en-tete).
const botKeyPrefix = "bot:"

// shieldCyclesOf rend, par xuid, les instants des bris et des recharges du document stocke ; nil quand
// aucun joueur n'est mesure ou que le document n'a pas de pas d'image.
func shieldCyclesOf(doc *replay.ReplayDocument) map[string]replaydoc.ShieldCycle {
	if doc == nil || doc.FrameIntervalMS <= 0 {
		return nil
	}
	lives := replayshield.Lives(doc)
	for i := range doc.Tracks {
		if lives[i].XUID == "" && doc.Tracks[i].Bot != "" {
			lives[i].XUID = botKeyPrefix + doc.Tracks[i].Bot
		}
	}
	events := shieldcycle.Cycles(lives)
	if len(events) == 0 {
		return nil
	}
	out := make(map[string]replaydoc.ShieldCycle, len(events))
	for xuid, evs := range events {
		c := replaydoc.ShieldCycle{Breaks: []int{}, Recharges: []int{}}
		for _, e := range evs {
			ms := replayshield.FrameMS(e.Frame, doc.FrameIntervalMS)
			switch e.Kind {
			case shieldcycle.Break:
				c.Breaks = append(c.Breaks, ms)
			case shieldcycle.Recharge:
				c.Recharges = append(c.Recharges, ms)
			}
		}
		out[xuid] = c
	}
	return out
}
