package replaybuild

// kills.go — LES FRAGS SOUS EFFET ACTIF (camo, surbouclier) : décodage killsource partagé
// avec neutralDeaths, résolution d'identité HORS LIGNE, et construction des EquipmentKillRef que
// `analysis/replay` joint aux épisodes d'équipement (cf. equipment_episode_kills.go).
//
// PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F, sous-lot F.1. Décision utilisateur 8a/8b,
// DEC-7 (révisée) : GO à petite population — cf. le journal du plan pour le détail des
// marges. VUE MATCH UNIQUEMENT, aucun agrégat (couverture d'artefacts insuffisante).

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

// decodeKillSource décode killsource UNE SEULE FOIS par match. neutralDeaths ET killRefs en
// dérivent tous les deux — avant le lot F.1, seul neutralDeaths décodait ; lui ajouter un
// second appel aurait payé une DEUXIÈME fois le verrou filmdec partagé pour le même fait.
// nil = décodage impossible (film absent ou source non décodable), déjà journalisé ici :
// les deux appelants n'ont qu'à tester le nil.
//
// LE FILM EST CELUI QUE `BuildBytes` A DÉJÀ CHARGÉ (lot 1, PLAN_CUISSON_PERF item 1.4) : ce
// décodage ouvrait et redécompressait le film ENTIER pour son propre compte, en plus des
// balayages. `film` nil (chunks illisibles, déjà journalisé par `chargerFilm`) n'est plus une
// lecture ratée ici mais un refus en amont — `killsource.Decode` rend alors `ErrNoChunk`, et le
// journal en Info ci-dessous reste la SEULE trace côté cuisson, au même niveau qu'avant.
func (b *Builder) decodeKillSource(matchID string, film *filmsource.Film) *killsource.Result {
	res, err := killsource.Decode(context.Background(), matchID, film, nil)
	if err != nil {
		slog.Info("replaybuild: source de dégât non décodée — morts neutres et frags sous effet non décodés",
			"err", err, "match_id", matchID)
		return nil
	}
	return res
}

// killRefs résout, pour chaque frag publié par killsource, l'identité du TUEUR, de l'ASSISTANT
// et de la VICTIME en XUID — les deux jointures de `analysis/replay` (épisodes d'équipement et
// `bomb_carriers_killed`) ne consomment plus que des identités déjà résolues.
//
// DEUX SORTIES, UNE SEULE PASSE DE RÉSOLUTION (lot G.6, 2026-09-05). La victime a été ajoutée
// ici plutôt que dans un second producteur : c'est LA MÊME table gamertag -> xuid et LE MÊME
// enregistrement killsource, et une seconde résolution en aurait fait une copie du même fait —
// la règle des 2 copies du dépôt. Elles restent DEUX types parce que les deux jointures n'ont
// pas la même population : la première crédite un TUEUR (un frag sur un bot y compte), la
// seconde exige les DEUX identités et écarte le couple sinon.
//
// MÊME PORTE QUE LES MORTS SANS REVENDICATION (`Result.LineByLinePublishable`) : porte
// fermée = les deux entrées à `Read=false`, jamais un champ à zéro qui se lirait comme une
// mesure — c'est exactement ce que `Coverage.Equipment.KillsRead` et
// `BombStatsCoverage.KillsRead` existent pour distinguer.
//
// LA RÉSOLUTION EST HORS LIGNE, ENTIÈREMENT FILM-NATIVE : ce paquet n'ouvre AUCUNE base (même
// contrat que neutralDeaths et que le reste de replaybuild). `killsource.Kill.Feed.Killer` et
// `killsource.Kill.Victim` portent un GAMERTAG (ou `xuid:<N>` en repli, cf.
// killsource.XUIDNamePrefix) ; le pont gamertag -> xuid vient du fil des morts DU FILM : chaque
// mort y porte le xuid ET le gamertag de sa victime dans le MÊME enregistrement — aucune table
// externe à charger.
//
// L'HORLOGE DES COUPLES EST CELLE DU MATCH, sans conversion : `killsource.Kill.TimeMS` et
// `replay.Death.TimeMS` sont le MÊME champ du MÊME enregistrement du chunk highlight. La
// dérivation et son contrôle vivent en tête de `replay.MatchKillsInput` — c'est là que la règle
// doit être lue, pas ici, parce que c'est là qu'elle est consommée.
//
// LA LECTURE EST PARTAGÉE DEPUIS LE LOT 1 (2026-09-02) : ce fichier et `matchfacts.go`
// ouvraient et reparsaient chacun le chunk highlight, pour en tirer le même fil. Ils reçoivent
// désormais le MÊME résultat, lu une fois par `BuildBytes` — mêmes valeurs, mêmes refus
// journalisés, une décompression et un parse de moins par cuisson.
func (b *Builder) killRefs(matchID string, deaths filmDeaths, res *killsource.Result) (replay.KillsInput, replay.MatchKillsInput) {
	if res == nil {
		return replay.KillsInput{}, replay.MatchKillsInput{}
	}
	if !res.LineByLinePublishable() {
		slog.Info("replaybuild: attribution ligne par ligne refusée — frags sous effet actif et porteurs tués non mesurés",
			"match_id", matchID, "kills", len(res.Kills))
		return replay.KillsInput{}, replay.MatchKillsInput{}
	}
	if deaths.err != nil {
		slog.Info("replaybuild: fil des morts illisible — frags sous effet actif et porteurs tués non mesurés",
			"err", deaths.err, "match_id", matchID)
		return replay.KillsInput{}, replay.MatchKillsInput{}
	}
	r := resolveKills(res.Kills, gamertagXUIDIndex(deaths.list))
	r.log(matchID, len(res.Kills))
	return replay.KillsInput{Read: true, Kills: r.refs},
		replay.MatchKillsInput{Read: true, Kills: r.pairs, Dropped: len(res.Kills) - len(r.pairs)}
}

// killResolution est ce qu'UNE passe de résolution rend, POUR LES DEUX JOINTURES — plus ce
// qu'elle a perdu, ventilé par CAUSE. Sans cette ventilation, « 12 couples écartés » ne
// distinguerait pas un roster de bots (attendu, majoritaire) d'un pont d'identité cassé.
type killResolution struct {
	refs  []replay.EquipmentKillRef
	pairs []replay.KillRef
	// killerUnresolved : frags dont le TUEUR n'a pas d'identité — ils manquent aux DEUX sorties.
	killerUnresolved int
	// victimUnresolved : frags dont le tueur est résolu mais PAS la victime — ils manquent à la
	// seule sortie des couples. Cas nominal et attendu : une victime BOT n'a pas de xuid, sa
	// mort n'est dans aucun enregistrement du fil, et aucune période de portage ne lui est
	// pontée non plus — l'écarter ne perd donc rien de mesurable.
	victimUnresolved int
}

// resolveKills résout tueur, assistant et victime en xuid, EN UNE PASSE. Pure : aucune I/O,
// testable sans film (kills_test.go).
func resolveKills(kills []killsource.Kill, byGamertag map[string]uint64) killResolution {
	r := killResolution{
		refs:  make([]replay.EquipmentKillRef, 0, len(kills)),
		pairs: make([]replay.KillRef, 0, len(kills)),
	}
	for _, k := range kills {
		killer, ok := resolveKillIdentity(k.Feed.Killer, byGamertag)
		if !ok {
			r.killerUnresolved++
			continue // aucune identité résolue : ce frag ne rencontrera aucun slot, l'omettre est sans perte
		}
		ref := replay.EquipmentKillRef{XUID: killer, TimeMS: k.TimeMS}
		if k.Assist.Known && k.Assist.Name != "" {
			if aXUID, ok := resolveKillIdentity(k.Assist.Name, byGamertag); ok {
				ref.AssistXUID, ref.AssistKnown = aXUID, true
			}
		}
		r.refs = append(r.refs, ref)
		victim, ok := resolveKillIdentity(k.Victim, byGamertag)
		if !ok {
			r.victimUnresolved++
			continue
		}
		r.pairs = append(r.pairs, replay.KillRef{
			KillerXUID: killer, VictimXUID: victim, TimeMS: int64(k.TimeMS),
		})
	}
	return r
}

// log publie ce que la passe a perdu. DEUX NIVEAUX, parce que les deux faits n'ont pas la même
// gravité : un tueur non résolu reste une anomalie du pont d'identité (WARN, message inchangé
// depuis le lot F.1), une victime non résolue est le cas NOMINAL des morts de bot (INFO). Aucun
// des deux n'est tu : un producteur qui tait ses trous laisse croire à l'exhaustivité.
func (r killResolution) log(matchID string, total int) {
	if r.killerUnresolved > 0 {
		slog.Warn("replaybuild: tueur non résolu en xuid — frag omis de la jointure équipement",
			"match_id", matchID, "non_resolus", r.killerUnresolved, "total", total)
	}
	if r.victimUnresolved > 0 {
		slog.Info("replaybuild: victime non résolue en xuid — couple omis de la jointure porteurs tués",
			"match_id", matchID, "non_resolues", r.victimUnresolved, "couples", len(r.pairs),
			"total", total)
	}
}

// gamertagXUIDIndex construit gamertag -> xuid depuis le fil des morts du film — le MÊME
// enregistrement porte les deux pour la victime (cf. replay.Death). EN CAS DE DIVERGENCE, LE
// PREMIER GAGNE — même règle que replay.gamertagsOf (identity.go), pour la même raison : les
// 32 octets d'un même xuid ne varient pas d'un enregistrement à l'autre à l'intérieur d'un
// film, donc rien à arbitrer.
func gamertagXUIDIndex(deaths []replay.Death) map[string]uint64 {
	out := make(map[string]uint64, len(deaths))
	for _, d := range deaths {
		if d.Gamertag == "" {
			continue
		}
		if _, seen := out[d.Gamertag]; !seen {
			out[d.Gamertag] = d.XUID
		}
	}
	return out
}

// resolveKillIdentity résout un nom killsource (gamertag, ou repli `xuid:<N>`) en xuid.
//
// MÊME RÈGLE À DEUX CAS QUE `killcollector.MatchIdentities.Resoudre`
// (internal/sync/killcollector/identities.go) — une copie DÉLIBÉRÉE, pas une divergence : ce
// paquet n'ouvre aucune base (contrat de replaybuild) et ne peut donc pas construire le
// `MatchIdentities` porté par `v_gamertag_lookup` que le collecteur EN LIGNE utilise ;
// celui-ci résout contre le fil des morts DU FILM, la seule source disponible hors ligne. Les
// deux s'accordent sur LA RÈGLE (repli `xuid:` d'abord, gamertag ensuite), jamais sur LA
// SOURCE — si un troisième lecteur de cette règle apparaît, centraliser (règle du dépôt sur
// les copies, CLAUDE.md n° 6).
func resolveKillIdentity(name string, byGamertag map[string]uint64) (uint64, bool) {
	if reste, ok := strings.CutPrefix(name, killsource.XUIDNamePrefix); ok {
		xuid, err := strconv.ParseUint(reste, 10, 64)
		if err != nil {
			return 0, false
		}
		return xuid, true
	}
	xuid, ok := byGamertag[name]
	return xuid, ok
}
