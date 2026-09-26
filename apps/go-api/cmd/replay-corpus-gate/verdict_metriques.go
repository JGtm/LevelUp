package main

// verdict_metriques.go — CE QUI COMPTE DANS LE VERDICT, ET CE QUI NE COMPTE PAS (lot 3.3.3).
//
// # LE DEFAUT QUE CE FICHIER FERME, MESURE LE 2026-09-17 (D5 et D6 (3.3.1) du plan)
//
// Le gate du lot 3.3 est sorti en code 1 sur un lot dont la mesure ne montrait AUCUNE perte. Deux
// causes, toutes deux dans la CLASSIFICATION, aucune dans la mesure :
//
//	LA TELEMETRIE. Depuis le schema 61 le document publie `coverage.decoder.{sourceRev,
//	profileRev, grammarRev, factsRev}` (`killsourceRev` et `objectivesRev` depuis le schema 72),
//	`coverage.decoder.build` et `coverage.decoder.registry.fingerprint`. Ces feuilles disent
//	QUELLE VERSION du decodeur a cuit l artefact — pas ce que le match contient. Au lot 2.6 elles etaient NEUVES, donc
//	comptees en GAINS, et le gate sortait 0 ; depuis, leur valeur BOUGE a chaque lot qui fait
//	monter une revision, `replaydiff` les classe `changement`, et `estBloquant` refusait le run.
//	Mesure du lot 3.3 : 51 des 59 changements des 17 temoins etaient ces trois chaines.
//
//	LES COMPTEURS DE REJET. `replaydiff/polarite.go` les lit deja A L ENVERS — une baisse est un
//	gain — et c est juste. Ce qu il ne peut pas voir, c est leur DENOMINATEUR : sur les huit
//	temoins anciens du lot 3.3, `coverage.grenades.noSlot` passe de 0 a 54 UNIQUEMENT parce que
//	`coverage.grenades.available` passe de 0 a 289. Un compteur de rejet qui monte d un
//	denominateur nul ne dit rien d une regression : il dit qu il y a desormais de la matiere a
//	rejeter.
//
// # LA REGLE, TRANCHEE PAR LE PILOTE LE 2026-09-17
//
//	(a) TELEMETRIE  affichee TOUJOURS (section « TELEMETRIE »), comptee NULLE PART. Une revision
//	    qui monte est attendue a chaque lot ; la voir est utile, en faire un verdict ne l est pas.
//	(b) REJET       une hausse n est une PERTE que si le RAPPORT au denominateur se degrade, avec
//	    un denominateur NON NUL des deux cotes. Sinon c est un CHANGEMENT : affiche, instruit par
//	    le pilote, toujours bloquant.
//	(c) LE RESTE    inchange. Une perte reelle et un changement hors telemetrie bloquent.
//
// # CE QUE CE FICHIER N EST PAS
//
// Il ne touche PAS `replaydiff` : la MESURE et la polarite restent ou elles sont, c est le
// VERDICT du gate de corpus qui se raffine ici. Un autre consommateur de `replaydiff` (le
// balayage de parc, `replay-diff`) garde sa lecture.

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"levelup/go-api/internal/replaydiff"
)

// sensTelemetrie : le quatrieme sens, LOCAL A CE GATE. `replaydiff` n en connait que cinq
// (perte, gain, changement, apparu, disparu) et n a pas a connaitre celui-ci : c est une
// decision de VERDICT, pas de mesure.
const sensTelemetrie = "telemetrie"

// metriquesDeTelemetrie — LA LISTE FERMEE DES FEUILLES QUI DISENT LA VERSION DU DECODEUR.
//
// Relevee sur pieces dans `replay/coverage_decoder.go` (types `DecoderCoverage` et
// `RegistryCoverage`) le 2026-09-17. Elles ne comptent ni en gain, ni en perte, ni en changement.
//
// `registry.status`, `registry.blocks` et `registry.namedSlots` n Y SONT PAS, et c est un choix :
// `status` dit si le depot RECONNAIT la grammaire de composants du film, `blocks` et
// `namedSlots` sont des grandeurs du registre lu. Les trois decrivent LE FILM, pas la version de
// notre code — une bascule y est une decouverte a instruire.
var metriquesDeTelemetrie = map[string]bool{
	"coverage.decoder.sourceRev":  true,
	"coverage.decoder.profileRev": true,
	"coverage.decoder.grammarRev": true,
	// `factsRev` : les artefacts ANTERIEURS au schema 72 la portent encore ; la garder ici fait que
	// sa disparition (scission par consommateur de faits, lot J3.3) n est pas lue comme une perte.
	"coverage.decoder.factsRev":             true,
	"coverage.decoder.killsourceRev":        true,
	"coverage.decoder.objectivesRev":        true,
	"coverage.decoder.build":                true,
	"coverage.decoder.registry.fingerprint": true,
}

// LES TROIS LITTERAUX QUE LA TABLE REPETE, NOMMES UNE FOIS (CLAUDE.md n 6). Une table de
// donnees repete par nature ses suffixes ; les nommer garde la table lisible ET tient le seuil
// des trois copies.
const (
	// blocPont : le bloc de sante du pont d identite, qui porte quatre familles de rejet.
	blocPont = "coverage.bridge."
	// rejetHorsFenetre : la prise ou la periode tombe hors de l axe de temps publie.
	rejetHorsFenetre = "outOfWindow"
	// rejetSansPont : le slot statborg n a pas ete resolu en xuid.
	rejetSansPont = "noBridge"
)

// familleDeRejet declare, pour UN bloc de couverture, ses compteurs de rejet et LE denominateur
// qui les rapporte a une population.
type familleDeRejet struct {
	// Blocs : les prefixes de bloc auxquels cette famille s applique, point final compris.
	Blocs []string
	// Denominateur : le suffixe du denominateur DANS LE MEME bloc. VIDE = aucun denominateur
	// n est publie a cote du compteur ; la lecture reste alors celle de `replaydiff`.
	Denominateur string
	// Rejets : les suffixes des compteurs de rejet du bloc.
	Rejets []string
	// Preuve : ou la mesure a ete lue, pour qu une entree ne se recopie pas de memoire.
	Preuve string
}

// rejetsDeCouverture — LA TABLE, RELEVEE SUR PIECES LE 2026-09-17.
//
// Elle n invente aucun denominateur : chaque ligne cite le fichier ou le champ le declare, et
// plusieurs blocs le NOMMENT en toutes lettres (« le denominateur de tout ce qui suit »). Une
// famille sans denominateur publie est declaree QUAND MEME, avec sa case vide : c est ce qui
// distingue « pas de denominateur » de « pas encore inventorie ».
//
//nolint:funlen // une table de donnees : la decouper masquerait la lecture ligne a ligne
func rejetsDeCouverture() []familleDeRejet {
	return []familleDeRejet{
		{
			Blocs:        []string{"coverage.shots.", "coverage.grenades.", "coverage.objectives."},
			Denominateur: "available",
			Rejets: []string{"noSlot", "ambiguous", rejetHorsFenetre, "unpublished",
				"refusedByRoster"},
			Preuve: "replay/coverage.go, LayerCoverage : `Available` est « le denominateur sans " +
				"lequel un compte de rattaches ne se juge pas », et Balanced() somme les cinq rejets",
		},
		{
			Blocs:        []string{blocPont},
			Denominateur: "livesTotal",
			Rejets:       []string{"unnamedLives", "unnamedLivesContested"},
			Preuve:       "replay/coverage_bridge.go, BridgeHealth : `LivesNamed` / `LivesTotal`",
		},
		{
			Blocs:        []string{blocPont},
			Denominateur: "indexReadings",
			Rejets:       []string{"indexDisagreements"},
			Preuve:       "replay/coverage_bridge.go : les desaccords se comptent sur les lectures d index",
		},
		{
			Blocs:        []string{blocPont},
			Denominateur: "slots",
			Rejets:       []string{"slotCollisions"},
			Preuve:       "replay/coverage_bridge.go : une collision porte sur un slot",
		},
		{
			Blocs:        []string{blocPont},
			Denominateur: "",
			Rejets:       []string{"closedContested", "closedRefused"},
			Preuve: "replay/coverage_bridge.go : les deux comptent des DEDUCTIONS abandonnees ou " +
				"rejetees, dont la population (`closedByShot` + `closedByRespawn` + les deux " +
				"refus) n est publiee par AUCUN champ — aucun denominateur a rapporter",
		},
		{
			Blocs:        []string{"coverage.projectiles."},
			Denominateur: "tracks",
			Rejets:       []string{"truncated"},
			Preuve:       "replay/coverage_decoder.go, ProjectileCoverage : `Tracks` puis `Published` / `Truncated`",
		},
		{
			Blocs:        []string{"coverage.teams."},
			Denominateur: "records",
			Rejets:       []string{"rejected", "divergences"},
			Preuve:       "replay/player_teams.go, TeamCoverage : « Records est le denominateur »",
		},
		{
			Blocs:        []string{"coverage.teams."},
			Denominateur: "",
			Rejets:       []string{"unread", "tracksSlotAmbiguous"},
			Preuve: "replay/player_teams.go : `Unread` compte les joueurs du ROSTER que le film " +
				"ne nomme pas, et l effectif du roster n est publie par aucun champ de ce bloc",
		},
		{
			Blocs:        []string{"coverage.abilityImpulses.scan."},
			Denominateur: "records",
			Rejets:       []string{"unread"},
			Preuve: "replay/document_ability_impulses.go, AbilityImpulseScanCoverage : `Read` et " +
				"`Unread` partagent les `Records` de la marche",
		},
		{
			Blocs:        []string{"coverage.flagCarries."},
			Denominateur: "openings",
			Rejets:       []string{rejetSansPont, "noTrack", rejetHorsFenetre},
			Preuve: "replay/document_objectives_live.go : « Openings … le denominateur de tout ce " +
				"qui suit »",
		},
		{
			Blocs:        []string{"coverage.flagCarries."},
			Denominateur: "",
			Rejets:       []string{"ambiguousSlot", "ambiguousReturns", "ambiguousCarrierKills"},
			Preuve: "replay/document_objectives_live.go : « CE N EST PAS UNE PARTITION DES " +
				"PRISES » — ces trois comptent de la matiere ecartee EN AMONT de toute prise",
		},
		{
			Blocs:        []string{"coverage.bombCarries."},
			Denominateur: "periods",
			Rejets:       []string{rejetSansPont, rejetHorsFenetre, "carrierAbsent"},
			Preuve:       "replay/document_bomb_carries.go : `Periods` puis `Carries` / `Closed` / `Open`",
		},
		{
			Blocs:        []string{"coverage.skullCarries."},
			Denominateur: "grabs",
			Rejets:       []string{rejetSansPont, rejetHorsFenetre},
			Preuve:       "replay/document_skull_carries.go : `Grabs` est la population des prises",
		},
		{
			Blocs:        []string{"coverage.vipCrown."},
			Denominateur: "selections",
			Rejets:       []string{rejetSansPont, rejetHorsFenetre},
			Preuve: "replay/document_vip_crown.go : « Selections … le denominateur de tout ce qui " +
				"suit »",
		},
		{
			Blocs:        []string{"coverage.vehicles."},
			Denominateur: "shots",
			Rejets:       []string{"shotsNoRide", "shotsAmbiguous", "shotsUnplaced"},
			Preuve:       "replay/document_vehicles.go : les trois se rapportent aux `Shots` du calque",
		},
		{
			Blocs:        []string{"bombStats.coverage."},
			Denominateur: "periods",
			Rejets:       []string{"periodsNoBridge", "periodsOpen"},
			Preuve:       "replay/bomb_stats.go : `Periods` puis ses ventilations",
		},
		{
			Blocs:        []string{"bombStats.coverage."},
			Denominateur: "armings",
			Rejets:       []string{"armingsNoCarrier", "armingsNoBridge", "armingsAmbiguous"},
			Preuve:       "replay/bomb_stats.go : `Armings` puis ses ventilations",
		},
	}
}

// denominateurDe rend le chemin COMPLET du denominateur d un compteur de rejet declare, et si la
// metrique est un rejet declare. Un rejet sans denominateur publie rend une chaine vide ET vrai :
// il est connu, il n a simplement rien a quoi se rapporter.
func denominateurDe(metrique string) (string, bool) {
	for _, f := range rejetsDeCouverture() {
		for _, bloc := range f.Blocs {
			if !strings.HasPrefix(metrique, bloc) {
				continue
			}
			suffixe := metrique[len(bloc):]
			for _, r := range f.Rejets {
				if suffixe != r {
					continue
				}
				if f.Denominateur == "" {
					return "", true
				}
				return bloc + f.Denominateur, true
			}
		}
	}
	return "", false
}

// classerPourLeVerdict rend le sens que LE GATE retient pour cet ecart, a partir du sens que
// `replaydiff` a mesure et de l etat du denominateur.
//
// `parMetrique` est l index des ecarts du MEME temoin : un denominateur ABSENT de cet index est
// un denominateur INCHANGE (une mesure identique des deux cotes n entre pas dans les ecarts).
// Inchange et non nul, le rapport se degrade des que le rejet monte — la lecture d avant ce lot,
// et elle reste juste.
func classerPourLeVerdict(d replaydiff.Difference,
	parMetrique map[string]replaydiff.Difference) string {
	if metriquesDeTelemetrie[d.Metrique] {
		return sensTelemetrie
	}
	if d.Sens != replaydiff.SensPerte && d.Sens != replaydiff.SensDisparu {
		return d.Sens
	}
	cleDenom, estRejet := denominateurDe(d.Metrique)
	if !estRejet || cleDenom == "" {
		return d.Sens
	}
	denom, connu := parMetrique[cleDenom]
	if !connu {
		return d.Sens // denominateur inchange : le rapport suit le rejet
	}
	rejetAncien, okRA := nombreDEcart(d.Ancien)
	rejetNouveau, okRN := nombreDEcart(d.Nouveau)
	denomAncien, okDA := nombreDEcart(denom.Ancien)
	denomNouveau, okDN := nombreDEcart(denom.Nouveau)
	if !okRA || !okRN || !okDA || !okDN {
		return d.Sens // une valeur non numerique : on ne devine pas, on garde la mesure
	}
	if denomAncien <= 0 || denomNouveau <= 0 {
		// Denominateur nul d un cote : il n y a pas de rapport a comparer. Un rejet qui monte de
		// zero parce que la population passe de 0 a N est un CHANGEMENT, pas une perte.
		return replaydiff.SensChangement
	}
	if rejetNouveau/denomNouveau > rejetAncien/denomAncien {
		return replaydiff.SensPerte
	}
	return replaydiff.SensChangement
}

// nombreDEcart analyse une valeur d ecart telle que `replaydiff` la formate (entier, ou decimal a
// quatre chiffres). Une chaine vide (mesure absente) vaut zero : c est ce que `classer` dit deja
// d une mesure absente face a un zero.
func nombreDEcart(s string) (float64, bool) {
	if s == "" {
		return 0, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// imprimerTelemetrie ecrit la section des feuilles de telemetrie qui ont bouge — celles qui ne
// comptent nulle part dans le verdict, et qu il faut voir quand meme.
//
// ELLE EST GROUPEE PAR VALEUR ET NON PAR TEMOIN, et c est le point : une revision monte pour TOUS
// les temoins a la fois, et dix-sept lignes identiques n apprennent rien de plus qu une.
func imprimerTelemetrie(w io.Writer, lignes []ligneRapport) {
	var ordre []string
	vues := map[string]map[string]bool{}
	temoins := map[string]int{}
	for _, l := range lignes {
		for _, d := range l.TelemetrieDetail {
			cle := d.Metrique + "\x00" + vide(d.Ancien) + "\x00" + vide(d.Nouveau)
			if vues[cle] == nil {
				vues[cle] = map[string]bool{}
				ordre = append(ordre, cle)
			}
			if !vues[cle][l.Temoin.ID] {
				vues[cle][l.Temoin.ID] = true
				temoins[cle]++
			}
		}
	}
	if len(ordre) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\nTELEMETRIE (%d valeur(s) — affichee, jamais comptee au verdict) :\n",
		len(ordre))
	for _, cle := range ordre {
		p := strings.Split(cle, "\x00")
		_, _ = fmt.Fprintf(w, "    %-46s %26s -> %-26s  %d temoin(s)\n",
			p[0], p[1], p[2], temoins[cle])
	}
}
