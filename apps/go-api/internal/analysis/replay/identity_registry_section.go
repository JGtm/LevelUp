package replay

// identity_registry_section.go — LA SECTION `identity` DE L'ARTEFACT.
//
// # ELLE EST LE TYPE RENDU, PAS UNE PROJECTION (contrainte 4 de l'inventaire P1, axe (j))
//
// Le registre CONSTRUIT cette section ; personne ne la reconstruit ailleurs. Deux ecritures du
// meme fait divergeraient au premier ajustement — c'est la lecon `flag_assign` vs
// `assembleFlagLives` (plan v2, R3).
//
// # CE QU'ELLE PUBLIE, ET POURQUOI CHAQUE LIGNE PORTE SES BORNES
//
// Trois familles de liens, celles que le lot P2 couvre : l'index de joueur, le slot de bipede
// dans le temps, le slot d'entite statborg par manche. Chacune porte sa PROVENANCE
// (`canonical.Link`) et ses BORNES : un slot de bipede est reattribue a chaque reapparition, un
// slot de statborg d'une manche a l'autre. Un lien aplati crediterait le premier occupant, et
// c'est exactement le defaut P0-2.
//
// # CE QUI N'EST PAS RESOLU EST PUBLIE, ET COMPTE
//
// Une vie sans nom, un slot de statborg qui emet sans etre nomme, un joueur du roster que le
// film ne place nulle part : tous entrent avec `source = non_resolu`. Les jeter ferait de la
// couverture un compte de RESCAPES, et le rapport se lirait 100 % sur un registre partiel.
// C'est la doctrine §0.2 du plan v2, servie jusqu'au contrat.

import (
	"strconv"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/canonical"
)

// IdentitySection est la section `identity` du document de rejeu.
type IdentitySection struct {
	// Players : une entree par identite connue du film — humain ou bot.
	Players []IdentityPlayer `json:"players,omitempty"`
	// BipedSlots : le slot de bipede -> identite, BORNE dans le temps (index de frame).
	BipedSlots []IdentityBipedSlot `json:"bipedSlots,omitempty"`
	// StatborgSlots : le slot d'entite statborg -> identite, PAR MANCHE.
	StatborgSlots []IdentityStatborgSlot `json:"statborgSlots,omitempty"`
	// Coverage : le recapitulatif par type de lien et par provenance. C'est LUI que le gate
	// corpus compare : il echoue si un compte `direct` baisse, ou si un `deduit` monte a
	// `direct` constant.
	Coverage IdentityCoverage `json:"coverage"`
}

// Empty dit que la section n'a rien a publier (pas d'axe de frames, ou aucun lien).
func (s IdentitySection) Empty() bool {
	return len(s.Players) == 0 && len(s.BipedSlots) == 0 && len(s.StatborgSlots) == 0
}

// IdentityPlayer est une identite du film : son index, son xuid (humain) ou son `bid` (bot).
type IdentityPlayer struct {
	// FilmIndex est l'index du joueur DANS CE FILM. -1 quand le film ne le place nulle part
	// (joueur connu de la seule feuille de match).
	FilmIndex int `json:"filmIndex"`
	// XUID en decimal — vide pour un bot.
	XUID string `json:"xuid,omitempty"`
	// Bid est l'identifiant STABLE d'un bot, forme `bid(N.0)` — la meme que la base emploie.
	//
	// IL EST LU DEPUIS TOUJOURS ET N'ETAIT PAS PUBLIE (inventaire P1, E9) : le paquet
	// BOT_METADATA declare `{Slot, BotID, Name}` et `BotID` EST le N de `bid(N.0)`. Faute de
	// le publier, la jointure web des bots se faisait sur le NOM NU, et deux bots homonymes
	// fusionnaient. C'est le gain le moins cher du lot P : ni decodage neuf, ni mesure.
	Bid string `json:"bid,omitempty"`
	// Name est le nom tel que le film l'ecrit (gamertag humain, ou nom du bot).
	Name string `json:"name,omitempty"`
	// Link dit d'ou vient ce lien.
	Link canonical.Link `json:"link"`
}

// IdentityBipedSlot est l'occupation d'un slot de bipede sur un intervalle de frames.
type IdentityBipedSlot struct {
	Slot uint32 `json:"slot"`
	// XUID de l'occupant. VIDE quand rien ne l'a nomme — la ligne est publiee quand meme,
	// avec `link.source = non_resolu`.
	XUID string         `json:"xuid,omitempty"`
	Link canonical.Link `json:"link"`
}

// IdentityStatborgSlot est l'identite d'un slot d'entite statborg POUR UNE MANCHE.
type IdentityStatborgSlot struct {
	Slot  int `json:"slot"`
	Round int `json:"round"`
	// XUID du joueur. VIDE quand le pont par manche n'a pas su nommer le slot.
	XUID string         `json:"xuid,omitempty"`
	Link canonical.Link `json:"link"`
}

// IdentityCoverage est le recapitulatif des liens PAR TYPE D'ENTITE et par provenance.
type IdentityCoverage struct {
	// FilmIndex : les liens « index de joueur du film <-> identite ». Nomme d apres
	// `RosterEntry.FilmIndex` — un index est un ORDRE LOCAL, valable dans CE film seulement, et
	// jamais une identite (ratchet `no_player_index_identity_test.go`).
	FilmIndex    canonical.LinkCounts `json:"filmIndex"`
	BipedSlot    canonical.LinkCounts `json:"bipedSlot"`
	StatborgSlot canonical.LinkCounts `json:"statborgSlot"`
}

// Total agrege les trois familles — le chiffre unique du journal, jamais celui du gate (qui
// compare famille par famille : une famille qui monte ne doit pas masquer celle qui baisse).
func (c IdentityCoverage) Total() canonical.LinkCounts {
	return canonical.LinkCounts{
		Direct:     c.FilmIndex.Direct + c.BipedSlot.Direct + c.StatborgSlot.Direct,
		Catalog:    c.FilmIndex.Catalog + c.BipedSlot.Catalog + c.StatborgSlot.Catalog,
		External:   c.FilmIndex.External + c.BipedSlot.External + c.StatborgSlot.External,
		Inferred:   c.FilmIndex.Inferred + c.BipedSlot.Inferred + c.StatborgSlot.Inferred,
		Unresolved: c.FilmIndex.Unresolved + c.BipedSlot.Unresolved + c.StatborgSlot.Unresolved,
	}
}

// buildIdentitySection assemble la section publiee. Rend une section VIDE quand l'appelant n'a
// pas d'axe de frames (le collecteur de sync) : un lien sans bornes exprimables ne se publie pas.
func buildIdentitySection(r IdentityRegistry, in IdentityInput) IdentitySection {
	if in.Clock.StepUS == 0 {
		return IdentitySection{}
	}
	s := IdentitySection{
		Players:       identityPlayers(in),
		BipedSlots:    identityBipedSlots(r, in.Clock),
		StatborgSlots: identityStatborgSlots(in),
	}
	for _, p := range s.Players {
		s.Coverage.FilmIndex.Add(p.Link.Source)
	}
	for _, b := range s.BipedSlots {
		s.Coverage.BipedSlot.Add(b.Link.Source)
	}
	for _, t := range s.StatborgSlots {
		s.Coverage.StatborgSlot.Add(t.Link.Source)
	}
	return s
}

// identityPlayers publie les identites du film : les humains que la table d'index NOMME
// (lien DIRECT), les bots que BOT_METADATA declare (lien DIRECT par `bid`), et les joueurs que
// seule la feuille connait (lien EXTERNE, index -1).
func identityPlayers(in IdentityInput) []IdentityPlayer {
	noms := gamertagsOf(in.Deaths)
	out := make([]IdentityPlayer, 0, len(in.PlayerIndices.ByXUID)+len(in.Bots))
	humains := map[uint64]bool{}
	for _, x := range triesParXUID(in.PlayerIndices.ByXUID) {
		humains[x] = true
		out = append(out, IdentityPlayer{
			FilmIndex: in.PlayerIndices.ByXUID[x],
			XUID:      strconv.FormatUint(x, 10),
			Name:      noms[x],
			Link: canonical.Link{Source: canonical.LinkDirect,
				Method: canonical.MethodPlayerIndexTable, Readings: in.PlayerIndices.Readings,
				From: 0, To: in.Clock.lastFrame()},
		})
	}
	for _, b := range in.Bots {
		out = append(out, IdentityPlayer{
			FilmIndex: b.FilmIndex, Bid: b.Bid(), Name: b.Name,
			Link: canonical.Link{Source: canonical.LinkDirect, Method: canonical.MethodBotID,
				From: 0, To: in.Clock.lastFrame()},
		})
	}
	// LE ROSTER DE LA BASE EN DERNIER, et il n'est ni direct ni deduit : le film ne le porte
	// pas (un joueur qui ne meurt jamais n'entre dans aucune table d'index), la base le sait.
	for _, x := range in.RosterXUIDs {
		if x == 0 || humains[x] {
			continue
		}
		humains[x] = true
		out = append(out, IdentityPlayer{
			FilmIndex: -1, XUID: strconv.FormatUint(x, 10), Name: noms[x],
			Link: canonical.Link{Source: canonical.LinkExternal, From: 0,
				To: in.Clock.lastFrame()},
		})
	}
	return out
}

// identityBipedSlots publie une ligne PAR VIE — jamais une par slot. C'est ce qui interdit de
// rejouer le defaut P0-2 sur un slot recycle.
func identityBipedSlots(r IdentityRegistry, c IdentityClock) []IdentityBipedSlot {
	lives := r.Vies()
	out := make([]IdentityBipedSlot, 0, len(lives))
	for i, l := range lives {
		lien := canonical.Link{From: c.frameOf(l.from), To: c.frameOf(l.to)}
		var xuid string
		switch {
		case l.xuid == 0:
			lien.Source, lien.Method = canonical.LinkUnresolved, canonical.MethodNone
		default:
			xuid = strconv.FormatUint(l.xuid, 10)
			lien.Source = canonical.LinkInferred
			lien.Method = methodeDeNommage(l.nomPar)
			if r.deducedLives[i] {
				lien.Method = canonical.MethodRosterElimination
			}
		}
		out = append(out, IdentityBipedSlot{Slot: l.slot, XUID: xuid, Link: lien})
	}
	return out
}

// methodeDeNommage traduit l'axe `nomPar` d'une vie en voie canonique. Un `nomPar` inconnu
// rend [canonical.MethodNone] : le producteur ne doit pas inventer une voie qu'il ne connait pas.
func methodeDeNommage(nomPar string) canonical.LinkMethod {
	switch nomPar {
	case NomParMort:
		return canonical.MethodDeathBridge
	case NomParFermeture:
		return canonical.MethodClosure
	case NomParElimination:
		return canonical.MethodRosterElimination
	}
	return canonical.MethodNone
}

// identityStatborgSlots publie une ligne par couple (manche, slot EMETTEUR) — nomme ou non.
//
// LE DENOMINATEUR EST L'EMISSION, PAS LE NOMMAGE : un slot qui parle et qu'on ne sait pas
// nommer est exactement ce que la couverture doit montrer. Le compter hors du total ferait
// disparaitre le trou que le lot R4 a mesure (8 couples perdus sur 3 films).
func identityStatborgSlots(in IdentityInput) []IdentityStatborgSlot {
	id := in.Statborg.Identity
	if !id.Resolved() || len(in.Statborg.Records) == 0 {
		return nil
	}
	var out []IdentityStatborgSlot
	for _, round := range id.Rounds() {
		for _, slot := range objectiveevents.EmittingPlayerSlots(in.Statborg.Records, round) {
			xuid := id.AtRound(round, slot)
			lien := canonical.Link{From: 0, To: in.Clock.lastFrame()}
			if xuid == "" {
				lien.Source, lien.Method = canonical.LinkUnresolved, canonical.MethodNone
			} else {
				lien.Source = canonical.LinkInferred
				lien.Method = methodeStatborg(id.Origin(round, slot))
			}
			out = append(out, IdentityStatborgSlot{Slot: slot, Round: round, XUID: xuid, Link: lien})
		}
	}
	return out
}

// methodeStatborg traduit la voie de `objectiveevents` en voie canonique.
func methodeStatborg(origin string) canonical.LinkMethod {
	switch origin {
	case objectiveevents.OriginDeathInstants:
		return canonical.MethodDeathInstants
	case objectiveevents.OriginSheetTriplet:
		return canonical.MethodSheetTriplet
	case objectiveevents.OriginElimination:
		return canonical.MethodRosterElimination
	}
	return canonical.MethodNone
}

// frameOf place un instant du film (microsecondes) sur l'axe de frames, borne aux extremes.
// Les bornes d'un lien decrivent une OCCUPATION, pas un evenement : les rogner a l'axe est
// exact, les jeter perdrait la ligne.
func (c IdentityClock) frameOf(tUS int64) int {
	if c.StepUS == 0 {
		return 0
	}
	f := (tUS - int64(c.OriginUS)) / int64(c.StepUS)
	if f < 0 {
		return 0
	}
	if last := int64(c.lastFrame()); f > last {
		return int(last)
	}
	return int(f)
}

// lastFrame rend l'index de la derniere frame publiee (bornes INCLUSIVES).
func (c IdentityClock) lastFrame() int {
	if c.FrameCount <= 0 {
		return 0
	}
	return c.FrameCount - 1
}

// triesParXUID rend les xuids d'une table en ordre croissant : l'artefact doit etre
// reproductible a l'octet, et l'ordre d'iteration d'une map Go ne l'est pas.
func triesParXUID(byXUID map[uint64]int) []uint64 {
	out := make([]uint64, 0, len(byXUID))
	for x := range byXUID {
		out = append(out, x)
	}
	trierUint64(out)
	return out
}
