package replaybuild

// kills.go — LES FRAGS SOUS EFFET ACTIF (camo, surbouclier) : décodage killsource partagé
// avec neutralDeaths, résolution d'identité HORS LIGNE, et construction des EquipmentKillRef que
// `games/halo_infinite/film/replay` joint aux épisodes d'équipement (cf. equipment_episode_kills.go).
//
// PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F, sous-lot F.1. Décision utilisateur 8a/8b,
// DEC-7 (révisée) : GO à petite population — cf. le journal du plan pour le détail des
// marges. VUE MATCH UNIQUEMENT, aucun agrégat (couverture d'artefacts insuffisante).

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// decodeKillSource décode killsource UNE SEULE FOIS par match. neutralDeaths ET killRefs en
// dérivent tous les deux — avant le lot F.1, seul neutralDeaths décodait ; lui ajouter un
// second appel aurait payé une DEUXIÈME fois le décodage du film pour le même fait (le verrou
// de paquet qui le rendait coûteux a disparu au lot 2.3 ; le décodage, lui, coûte toujours).
// nil = décodage impossible (film absent ou source non décodable), déjà journalisé ici :
// les deux appelants n'ont qu'à tester le nil.
//
// LE FILM EST CELUI QUE `BuildBytes` A DÉJÀ CHARGÉ (lot 1, PLAN_CUISSON_PERF item 1.4) : ce
// décodage ouvrait et redécompressait le film ENTIER pour son propre compte, en plus des
// balayages. `film` nil (chunks illisibles, déjà journalisé par `chargerFilm`) n'est plus une
// lecture ratée ici mais un refus en amont — `decfilm.Decode` rend alors `ErrNoChunk`, et le
// journal en Info ci-dessous reste la SEULE trace côté cuisson, au même niveau qu'avant.
func (b *Builder) decodeKillSource(ctx context.Context, matchID string, entry decfilm.MapQuantEntry, film *decfilm.Film) *decfilm.Result {
	opts := decfilm.DefaultOptions()
	// LA CARTE DU MATCH DESCEND DANS LE DECODAGE DES MORTS (lot 3.4.1). Les largeurs d'axe du
	// chemin absolu de position sont une constante PAR CARTE ; jusqu'à ce lot `killsource` les
	// INFÉRAIT par balayage, faute de recevoir la moindre entrée de catalogue — il était le seul
	// chemin de décodage dans ce cas. L'inférence est devenue ORACLE (V17, M3-Q8 : « la valeur
	// LUE prime sur la valeur mesurée »), et la valeur lue arrive ici.
	//
	// C'EST L'ENTRÉE QUE `BuildBytes` A DÉJÀ RÉSOLUE, ET IL N'Y EN A PAS D'AUTRE (2026-09-27) : une
	// carte hors catalogue y est refusée AVANT toute lecture (`ErrMapNotInCatalog`), donc ce
	// décodage ne voit jamais un match sans carte. La seconde résolution qu'il faisait, et sa
	// branche « largeurs par défaut », ont disparu avec le repli — « le flux du film est la seule
	// source fiable. Pas de repli. »
	opts.Carte = &entry
	res, err := decfilm.Decode(ctx, matchID, film, &opts)
	if res != nil {
		replay.JournaliserDiagnostics(ctx, res.Diagnostics) // lot J12.3, ADR 0034 D-4
	}
	if err != nil {
		slog.InfoContext(ctx, "replaybuild: source de dégât non décodée — morts neutres et frags sous effet non décodés",
			"err", err, "match_id", matchID)
		return nil
	}
	return res
}

// profilDeBalayageDeLaCuisson rend le PROFIL que la cuisson du rejeu doit porter apres le
// decodage du kill-feed.
//
// DEUX CAS, ET LES DEUX SONT LE COMPORTEMENT DE PRODUCTION D AVANT LE LOT 2.3, rendu explicite :
//
//	DECODAGE ABOUTI  le profil CALIBRE sur ce film (descripteur de traversee, largeur d axe
//	                 absolue, `param_4`). C est l heritage que la decouverte D1 du lot 2.2.a a
//	                 nomme : reel, voulu (la grammaire mesuree prime sur le defaut), mais qui
//	                 passait par l etat du processus.
//	DECODAGE REFUSE  le profil DE DEPART de `killsource` ([decfilm.ProfilDeDepart]) —
//	                 l invariant plus le `param_4` force a zero. `decfilm.Decode` posait ce
//	                 zero AVANT de lire quoi que ce soit, et ne le retirait jamais : un film
//	                 dont le kill-feed ne se decode pas laissait donc lui aussi sa trace sur la
//	                 cuisson. Le reproduire ici est ce qui rend le pas STRUCTUREL (zero
//	                 difference d octet, critere D4 du jalon).
func profilDeBalayageDeLaCuisson(res *decfilm.Result) *decfilm.ProfilDeBalayage {
	if res != nil {
		p := res.ProfilCalibre
		return &p
	}
	p := decfilm.ProfilDeDepart()
	return &p
}

// killRefs résout, pour chaque frag publié par killsource, l'identité du TUEUR, de l'ASSISTANT
// et de la VICTIME en XUID — les deux jointures de `games/halo_infinite/film/replay` (épisodes d'équipement et
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
// contrat que neutralDeaths et que le reste de replaybuild). `decfilm.Kill.Feed.Killer` et
// `decfilm.Kill.Victim` portent un GAMERTAG ; le pont gamertag -> xuid vient du fil des morts DU FILM : chaque
// mort y porte le xuid ET le gamertag de sa victime dans le MÊME enregistrement — aucune table
// externe à charger.
//
// L'HORLOGE DES COUPLES EST CELLE DU MATCH, sans conversion : `decfilm.Kill.TimeMS` et
// `types.Death.TimeMS` sont le MÊME champ du MÊME enregistrement du chunk highlight. La
// dérivation et son contrôle vivent en tête de `replay.MatchKillsInput` — c'est là que la règle
// doit être lue, pas ici, parce que c'est là qu'elle est consommée.
//
// LA LECTURE EST PARTAGÉE DEPUIS LE LOT 1 (2026-09-02) : ce fichier et `matchfacts.go`
// ouvraient et reparsaient chacun le chunk highlight, pour en tirer le même fil. Ils reçoivent
// désormais le MÊME résultat, lu une fois par `BuildBytes` — mêmes valeurs, mêmes refus
// journalisés, une décompression et un parse de moins par cuisson.
//
// `fb` recoit le repli de la resolution (assistant non resolu) — le compteur de la CONSTRUCTION
// (lot J8.7), que l assemblage verse ; nil ne compte rien. Un gamertag que le fil porte sous deux
// xuids ne resout rien, et le journal le dit.
func (b *Builder) killRefs(ctx context.Context, matchID string, deaths filmDeaths, res *decfilm.Result, fb *decfilm.Compteur) (replay.KillsInput, replay.MatchKillsInput) {
	if res == nil {
		return replay.KillsInput{}, replay.MatchKillsInput{}
	}
	if !res.LineByLinePublishable() {
		slog.InfoContext(ctx, "replaybuild: attribution ligne par ligne refusée — frags sous effet actif et porteurs tués non mesurés",
			"match_id", matchID, "kills", len(res.Kills))
		return replay.KillsInput{}, replay.MatchKillsInput{}
	}
	if deaths.err != nil {
		slog.InfoContext(ctx, "replaybuild: fil des morts illisible — frags sous effet actif et porteurs tués non mesurés",
			"err", deaths.err, "match_id", matchID)
		return replay.KillsInput{}, replay.MatchKillsInput{}
	}
	parGamertag, ambigus := gamertagXUIDIndex(deaths.list)
	if ambigus > 0 {
		slog.WarnContext(ctx, "replaybuild: gamertag porte par deux xuids dans le fil des morts — ses frags ne se resolvent pas",
			"match_id", matchID, "gamertags_ambigus", ambigus)
	}
	r := resolveKills(res.Kills, parGamertag)
	fb.DeclencheN(decfilm.NomAssistantNonResoluAbandonne, r.assistantsNonResolus)
	r.log(ctx, matchID, len(res.Kills))
	return replay.KillsInput{Read: true, Kills: r.refs, Paths: voiesDesMorts(res)},
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
	// assistantsNonResolus : assistants nommés par le kill-feed que la résolution ne sait pas
	// traduire en xuid — le frag est publié sans eux (`repli_assistant_non_resolu_abandonne`,
	// lot J8.7).
	assistantsNonResolus int
}

// resolveKills résout tueur, assistant et victime en xuid, EN UNE PASSE. Pure : aucune I/O,
// testable sans film (kills_test.go).
func resolveKills(kills []decfilm.Kill, byGamertag map[string]uint64) killResolution {
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
			} else {
				r.assistantsNonResolus++
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
func (r killResolution) log(ctx context.Context, matchID string, total int) {
	if r.killerUnresolved > 0 {
		slog.WarnContext(ctx, "replaybuild: tueur non résolu en xuid — frag omis de la jointure équipement",
			"match_id", matchID, "non_resolus", r.killerUnresolved, "total", total)
	}
	if r.victimUnresolved > 0 {
		slog.InfoContext(ctx, "replaybuild: victime non résolue en xuid — couple omis de la jointure porteurs tués",
			"match_id", matchID, "non_resolues", r.victimUnresolved, "couples", len(r.pairs),
			"total", total)
	}
}

// gamertagXUIDIndex construit gamertag -> xuid depuis le fil des morts du film — le MÊME
// enregistrement porte les deux pour la victime (cf. types.Death).
//
// UN GAMERTAG VU SOUS DEUX XUIDS N'ENTRE PAS DANS LA TABLE : aucun des deux n'est plus sûr que
// l'autre, et ses frags restent non résolus. Le second rendu compte ces gamertags ambigus.
func gamertagXUIDIndex(deaths []types.Death) (map[string]uint64, int) {
	out := make(map[string]uint64, len(deaths))
	ambigus := map[string]bool{}
	for _, d := range deaths {
		if d.Gamertag == "" || ambigus[d.Gamertag] {
			continue
		}
		if premier, seen := out[d.Gamertag]; seen && premier != d.XUID {
			ambigus[d.Gamertag] = true
			delete(out, d.Gamertag)
			continue
		}
		out[d.Gamertag] = d.XUID
	}
	return out, len(ambigus)
}

// resolveKillIdentity résout un gamertag killsource en xuid, contre le fil des morts DU FILM — la
// seule source disponible hors ligne : ce paquet n'ouvre aucune base (contrat de replaybuild).
func resolveKillIdentity(name string, byGamertag map[string]uint64) (uint64, bool) {
	xuid, ok := byGamertag[name]
	return xuid, ok
}

// voiesDesMorts traduit ce que les DEUX VOIES de lecture des morts ont propose, apparie et publie
// en bloc de couverture de l artefact (schema 62, lot 4.2.1-b).
//
// # POURQUOI ICI, ET PAS DANS LE DECODEUR
//
// Les deux voies existent dans le type publie de `killsource` depuis toujours et leurs
// denominateurs sont mesures par le decodage (`Result.Stats.Walk` / `.Scan`) ; ce qui manquait
// est le PONT jusqu a l artefact — aucun compte par voie ne l atteignait, donc aucun
// consommateur ne pouvait ponderer une ligne de mort par la precision de la voie qui l a lue.
// Le decodeur, lui, n a pas a connaitre la forme de l artefact : il rend ses statistiques, et
// c est la cuisson qui decide de les publier (meme frontiere que `neutralDeaths` juste au-dessus).
//
// L APPEL EST GARDE PAR SON APPELANT : il n est atteint que sur le chemin ou `KillsInput.Read`
// vaut vrai, donc un bloc present veut dire « les morts ont ete lues », jamais « elles ont ete
// lues et n ont rien rendu ».
func voiesDesMorts(res *decfilm.Result) *replay.DeathsPathsCoverage {
	if res == nil {
		return nil
	}
	return &replay.DeathsPathsCoverage{
		Walk: tallyDeVoie(res.Stats.Walk),
		Scan: tallyDeVoie(res.Stats.Scan),
	}
}

// tallyDeVoie recopie les trois denominateurs d une voie. Les noms ne changent pas en route :
// une traduction de vocabulaire entre le decodeur et l artefact rendrait la jointure des deux
// mesures impossible a verifier.
func tallyDeVoie(s decfilm.PathStats) replay.DeathsPathTally {
	return replay.DeathsPathTally{Population: s.Population, Matched: s.Matched, Published: s.Published}
}
