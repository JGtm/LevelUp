package domain

import "time"

// tactical_cellule.go — LE DETAIL D'UNE CELLULE : le lien « voir dans le rejeu » depuis
// une case de la grille tactique (Tactique S.1, lot M1 du plan d'orchestration 2026-09-07 ;
// horloge exacte, lot M1b du 2026-09-08).
//
// # CE QUE CETTE LECTURE AJOUTE A LA LECTURE DE PLACEMENT
//
// `TacticalRaster` rend une VALEUR agregee par cellule (`CelluleTactique.Valeur`/`Brut`) :
// elle ne dit pas QUELS matchs, a QUEL instant, l'ont produite. Cette lecture-ci repond a
// cette question pour UNE cellule precise, avec assez de matiere pour ouvrir le rejeu 2D du
// match a l'instant concerne (`?frame=` du lecteur, cf. playbackStore cote web).
//
// # OWNERSHIP (ADR 0029)
//
// `Contributions` ne porte QUE des matchs auxquels le joueur de la page a REELEMENT
// participe — verifie par `TacticalRepository.MatchsOuvrables`, meme nature de garde que la
// Couche B de l'ADR (`MatchViewService.IsParticipant`). Un match du perimetre demande qui
// echoue cette verification n'apparait dans AUCUNE contribution : il est COMPTE au journal du
// service (`matchs_non_ouvrables`), jamais liste ni publie — aucun lecteur web ne l'affiche. En
// usage nominal (perimetre resolu par `/filters/match-ids`, sur la base du joueur) ce compte est
// TOUJOURS zero ; la garde protege un appelant qui poserait un match_id etranger dans le corps de
// la requete — meme surface de requete que le raster (`match_ids` en liste blanche).

// Valeurs de TacticalContribution.Clock — cf. sa doc pour la regle par question.
const (
	TacticalClockMatch = "match"
	TacticalClockFilm  = "film"
)

// TacticalCelluleRequest est la demande de detail d'une cellule.
type TacticalCelluleRequest struct {
	MapID    string
	Question string
	Qui      string
	Scope    TacticalScope

	// Col, Lig : l'adresse de la cellule demandee, dans le MEME repere que
	// CelluleTactique.Col/Lig — ancre sur l'origine du monde, jamais sur les bornes de la
	// lecture agregee.
	Col, Lig int

	// PasM est le pas de la grille SUR LAQUELLE l'adresse ci-dessus a un sens, en metres :
	// celui que la lecture agregee a publie (`TacticalRaster.PasM`), renvoye tel quel par
	// le client.
	//
	// IL EST OBLIGATOIRE DEPUIS LE PAS ADAPTATIF (lot 3.2, decision D6) : une adresse de
	// cellule ne veut rien dire sans son pas — la cellule (12, 8) d'une grille de 2 m
	// couvre les cellules (48..51, 32..35) d'une grille de 0,5 m. Resolue au mauvais pas,
	// la demande rend les contributions d'un AUTRE endroit de la carte, sans rien casser.
	//
	// ZERO (ou toute valeur non finie / negative) VAUT LE PAS PAR DEFAUT, jamais une
	// erreur : c'est ce que faisaient tous les appelants avant ce lot, et un client d'une
	// version anterieure doit continuer a lire la meme chose.
	PasM float64
}

// TacticalContribution est UNE contribution a la cellule demandee.
type TacticalContribution struct {
	// MatchID est le match d'ou vient la contribution.
	MatchID string `json:"match_id"`

	// InstantMs est l'instant CONTRIBUTEUR, en millisecondes — MAIS PAS TOUJOURS SUR LA
	// MEME HORLOGE (verifie sur pieces, item 1 du lot M1). Cf. `Clock` ci-dessous pour la
	// regle par question.
	InstantMs int64 `json:"instant_ms"`

	// Clock dit sur QUELLE HORLOGE `InstantMs` est exprime — ajoute par le lot M1b
	// (2026-09-08, decision utilisateur : « corriger le decalage ») pour que le WEB puisse
	// convertir un instant exact plutot que de deviner :
	//
	//	TacticalClockFilm ("film")    temps, routes — l'horloge de l'ARTEFACT de rejeu
	//	                              (frame x FrameIntervalMs, cf. sidecar
	//	                              `PremieresEntrees`/`Routes`) — le MEME axe que le
	//	                              lecteur 2D, conversion exacte et directe.
	//	TacticalClockMatch ("match")  morts, kills, gagne, isole — l'horloge du MATCH
	//	                              (`match_kill_events.time_ms` /
	//	                              `match_death_context.time_ms`), DISTINCTE de celle du
	//	                              film : `games/halo_infinite/film/replay/lives_export.go` etablit
	//	                              `horlogeFilm = horlogeMatch + DeathOffsetMS`, un
	//	                              decalage PAR MATCH (mesure 3,6 a 50,8 s sur les films
	//	                              temoins) publie par `coverage.bridge.deathOffsetMs`
	//	                              DEPUIS LE SCHEMA 49 SEULEMENT.
	//
	// LE SERVICE NE CONVERTIT PAS LUI-MEME : il n'a pas toujours l'artefact de rejeu sous la
	// main (cuisson asynchrone, artefact absent ou trop vieux) — c'est le WEB qui convertit,
	// au moment d'ouvrir le rejeu, quand le document (et donc l'offset) est charge
	// (`lib/replay/replayLogic.resolveTacticalReplayInstant`). Decouverte du lot M1
	// (`.ai/V7.5/DECOUVERTES_TACTIQUE_2026-09-07.md`) : TRAITEE le 2026-09-08 par ce champ.
	Clock string `json:"clock"`

	// XUID est le joueur dont l'evenement a produit cette contribution (victime, tueur, ou
	// proprietaire de la piste, selon la question) — PAS necessairement le joueur de la
	// page : sous l'axe « escouade » ou « adv », une contribution appartient a un
	// coequipier ou un adversaire.
	XUID string `json:"xuid"`

	// Resultat est l'issue du match POUR L'EQUIPE DU JOUEUR DE LA PAGE, sous sa forme
	// CANONIQUE (`canonical.Outcome` : "win" / "loss" / "tie" / "dnf"). Vide quand le
	// substrat ne la connait pas (OutcomeUnknown) — jamais une valeur de repli, qui se
	// lirait comme une defaite.
	//
	// POURQUOI ELLE VOYAGE. La liste des contributions est une liste de MATCHS a rouvrir :
	// sans son issue, deux lignes de la meme journee ne se distinguent que par une heure.
	// Le LIBELLE et la COULEUR restent cote web (`useOutcomeMapping`, outcomes.toml) — on
	// ne sert jamais de texte localise ici.
	Resultat string `json:"resultat,omitempty"`

	// MatchStartedAt est la date de DEBUT DU MATCH (canonique, cf.
	// platform/duckdb.StartTimeCanonicalSQL), tiree de la MEME verification d'ouvrabilite
	// qui a admis cette contribution. La carte web l'affiche a cote de l'instant — un
	// match seul (sans date) n'identifie rien pour l'utilisateur qui a plusieurs parties
	// sur la meme carte le meme jour.
	MatchStartedAt time.Time `json:"match_started_at"`

	// ─── La mini-tuile « Rejeu » de la zone sélectionnée ───────────────────────────────
	// Chaque champ ci-dessous est ABSENT quand sa source ne le connaît pas (titre sans la
	// donnée, lecture en échec) : jamais un texte de repli.

	// Face dit ce que la contribution est pour son joueur : TacticalFace*.
	Face string `json:"face,omitempty"`
	// AutreGamertag : l'autre joueur de l'événement — le tueur d'une mort, la victime d'un frag.
	AutreGamertag string `json:"autre_gamertag,omitempty"`
	// ArmeLabel / ArmeLabelEN : le nom de l'arme de l'événement dans les deux langues, par le
	// registre d'armes du titre ; CategorieSource : la catégorie brute de la source de dégât du
	// film, publiée quand l'arme n'est pas nommée (le web la traduit, ou ne l'écrit pas).
	ArmeLabel       string `json:"arme_label,omitempty"`
	ArmeLabelEN     string `json:"arme_label_en,omitempty"`
	CategorieSource string `json:"categorie_source,omitempty"`
	// Placement : seul ou près d'un coéquipier, pour une MORT seulement.
	Placement *TacticalPlacement `json:"placement,omitempty"`
	// ModeLabel : le mode du match, dans la langue de la requête (libellé de l'app).
	ModeLabel string `json:"mode_label,omitempty"`
	// ScoreLabel : « X - Y », le camp du joueur de la page d'abord (analysis.TeamScoreLabel) ;
	// ScoreKind dit si ce sont des points ou des manches (analysis.ScoreKind*).
	ScoreLabel string `json:"score_label,omitempty"`
	ScoreKind  string `json:"score_kind,omitempty"`
	// ReplayAvailable : l'artefact de rejeu 2D du match existe — le web ne pose le bouton de
	// rejeu que là, jamais un lien vers une page vide.
	ReplayAvailable bool `json:"replay_available"`
}

// Valeurs de TacticalContribution.Face.
const (
	TacticalFaceMort         = "mort"
	TacticalFaceFrag         = "frag"
	TacticalFaceEntree       = "entree"       // temps : première entrée dans la zone
	TacticalFaceReapparition = "reapparition" // routes : la réapparition dont la route traverse la zone
)

// TacticalCelluleReponse est la reponse du detail d'une cellule.
type TacticalCelluleReponse struct {
	// Contributions est triee par date de match DECROISSANTE puis par instant croissant.
	Contributions []TacticalContribution `json:"contributions"`

	// Zone : le nom en jeu de la zone de la cellule ; absent quand aucune zone ne la nomme.
	Zone *TacticalZoneNom `json:"zone,omitempty"`
}
