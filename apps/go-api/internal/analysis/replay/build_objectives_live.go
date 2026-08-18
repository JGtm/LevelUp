package replay

// build_objectives_live.go — LE CABLAGE du calque du DRAPEAU VIVANT : ce que l'appelant fournit,
// ce que `BuildFromFilm` decode, et ce que `BuildFromPositions` en assemble.
//
// Il vit a part de `build.go` pour la meme raison que `build_ground_weapons.go` et
// `build_score.go` : l'assemblage garde UNE ligne par calque, le detail vit a cote de la donnee
// qu'il produit. `build.go` depasse deja 500 lignes, et la regle du depot est de ne pas
// accroitre cette dette.
//
// # QUATRE LECTURES, ET UNE SEULE EST FAITE ICI
//
//	les ENREGISTREMENTS d'entite   l'appelant (il les balaye deja pour la courbe de score)
//	les BURSTS de capture          l'appelant (autre grammaire, autre parcours du film)
//	les SOCLES `flag_spawn`        l'appelant (catalogue de carte, joint par map_id)
//	le MARQUEUR de portage         ICI, dans `BuildFromFilm`, comme les autres balayages filmdec
//
// POURQUOI LES TROIS PREMIERES VIENNENT DE L'APPELANT. C'est la frontiere que ce paquet tient
// deja pour `Objectives`, `Score` et `NeutralDeaths` : `analysis/replay` est title-agnostic et
// ne connait ni le cache film du titre (les enregistrements d'entite se lisent par un manifeste
// de chunks) ni le catalogue de cartes du titre. L'appelant decode une fois et fournit — deux
// balayages du meme fait divergeraient.
//
// # LE PONT D'IDENTITE NE DEMANDE AUCUNE BASE, ET C'EST DELIBERE
//
// Le slot statborg d'un porteur se resout en xuid par les seuls INSTANTS DE MORT
// ([objectiveevents.SlotIdentityByDeaths]) : les progressions du compteur de morts du statborg
// appariees au fil des morts du film. Le pont par TOTAUX aurait exige les lignes de match, donc
// la base ; les deux ont ete confrontes a la phase 0 (8 accords / 0 desaccord la ou les deux
// repondent, et 8/8 contre 0/8 sur un film TRONQUE). Le calque du drapeau se publie donc sur un
// artefact construit SANS aucun fait de match — un CLI hors ligne le rend entier.

import (
	"log/slog"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// FlagInput est CE QUE L'APPELANT FOURNIT du drapeau, plus ce que `BuildFromFilm` y depose.
//
// LES LECTURES VOYAGENT ENSEMBLE, ET `Scanned` DIT QU'ELLES ONT ABOUTI : un calque vide sans lui
// serait indistinguable d'un film qu'on n'a pas su ouvrir (meme regle que [GroundWeaponScan]).
type FlagInput struct {
	// Scanned dit que l'appelant a bien ouvert le film. Faux : ni calque ni couverture — le
	// document ne dit alors rien du drapeau, ce qui n'est pas « il n'y en avait pas ».
	Scanned bool
	// Records sont les enregistrements d'entite du film (`objectiveevents.StatRecordsCtx`), le
	// MEME balayage que celui de la courbe de score : ils portent les evenements nommes du
	// drapeau et les progressions du compteur de morts qui identifient les slots.
	Records []objectiveevents.StatRecord
	// Bursts sont les instants des BURSTS DE CAPTURE (`objectiveevents.CaptureBurstTimes`) —
	// une autre grammaire du film, et le signal sans lequel le discriminant de mode ne tient
	// pas (un film Oddball rend 1 470 « prises » a la table du drapeau).
	Bursts []int
	// Spawns sont les socles `flag_spawn` D'EQUIPE de la carte, en coordonnees monde, lus dans
	// le catalogue versionne d'objectifs. Vides : les portages restent publies, mais tous dans
	// UN drapeau d'equipe [TeamNeutral] et sans etat `home` (sa position serait inventee).
	Spawns []FlagSpawn
	// Marks est le CONTROLE independant, depose par `BuildFromFilm` : les records de bipede
	// d'image-cle portant le marqueur de portage, et l'instant de toutes les images-cles.
	// L'appelant ne le remplit pas.
	Marks filmdec.CarrierMarkScan
}

// decodeFilmCarrierMarks balaye le marqueur de portage et JOURNALISE ce qu'il en est.
//
// IL NE BALAYE QUE LES FILMS DE CTF, et c'est une mesure de cout, pas une optimisation de
// principe. Le balayage est une marche COMPLETE des images-cles avec une fenetre glissante de
// 32 bits sur l'emprise de chaque record de bipede : sur les films mesures il pese des dizaines
// de secondes, a comparer aux ~60 s de tout le reste du decodage. Or ce qu'il produit n'est
// qu'un CONTROLE — il alimente `markerObserved` / `markerConfirmed`, jamais le calque lui-meme —
// et sur un film d'un autre mode le calque est vide de toute facon. Le payer partout serait
// payer pour rien sur la quasi-totalite des artefacts.
//
// LE VERDICT DE MODE EST DEJA LA : il se lit dans ce que l'appelant a fourni (enregistrements
// d'entite + bursts de capture), sans toucher au film. Un film non reconnu rend donc un balayage
// VIDE, ce qui est exactement ce que `buildFlagCarries` en fera.
//
// L'ECHEC N'EST PAS FATAL, et il ne se confond pas avec l'absence de marque : sans ce balayage
// le calque est publie SANS son controle independant (`markerObserved` a zero), pas ampute. Un
// silence ici laisserait croire que les images-cles ne portaient rien.
//
// HORS LIGNE — appelee par BuildFromFilm, sous LockProcessDecode.
func decodeFilmCarrierMarks(filmDir string, in FlagInput) filmdec.CarrierMarkScan {
	if !in.Scanned || !flagFilmSignalsOf(in).IsFlagFilm() {
		return filmdec.CarrierMarkScan{}
	}
	marks, err := filmdec.ScanFilmCarrierMarks(filmDir)
	if err != nil {
		slog.Warn("drapeau : marqueur de portage illisible — calque publie sans son controle",
			"err", err, "filmDir", filmDir)
		return filmdec.CarrierMarkScan{}
	}
	return marks
}

// flagFilmSignalsOf rend le verdict de mode a partir des SEULES lectures deja faites. Pur.
func flagFilmSignalsOf(in FlagInput) objectiveevents.FlagFilmSignals {
	return objectiveevents.FlagFilmSignalsFrom(in.Bursts,
		objectiveevents.NamedEventsFrom(in.Records, objectiveevents.ObjectiveTypeFlag))
}

// attachFlagCarries pose la vie des drapeaux sur le document, avec sa couverture et son journal.
//
// `own` porte le pont bipede -> joueur ET le calage d'horloge du fil des morts (filmMS = matchMS
// + DeathOffsetMS) : les evenements nommes sont dates sur l'horloge du MATCH, les images-cles et
// les positions sur celle du FILM. Sans ce calage, le controle du marqueur comparerait deux
// horloges differentes et ne confirmerait rien.
func attachFlagCarries(doc *ReplayDocument, opt Options, own OwnerReport, clock replayClock) {
	in := opt.Flag
	events := objectiveevents.NamedEventsFrom(in.Records, objectiveevents.ObjectiveTypeFlag)
	scan := FlagCarryScan{
		Scanned:  in.Scanned,
		Signals:  flagFilmSignalsOf(in),
		Events:   events,
		Identity: objectiveevents.SlotIdentityByDeaths(in.Records, deathInstantsOf(opt.Deaths)),
		Marks:    in.Marks,
		Spawns:   in.Spawns,
	}
	carries, cov := buildFlagCarries(scan, flagCarryCtx{
		origin: clock.origin, step: clock.step, frames: clock.frames,
		tracks: doc.Tracks, deaths: opt.Deaths,
		deathOffsetMS: own.DeathOffsetMS, slotXUID: own.SlotXUID,
	})
	doc.FlagCarries = carries
	if doc.Coverage != nil {
		doc.Coverage.FlagCarries = cov
	}
	logFlagCarriesCoverage(cov)
}

// deathInstantsOf traduit le fil des morts du rejeu dans la forme qu'attend le pont d'identite.
func deathInstantsOf(deaths []Death) []objectiveevents.DeathInstant {
	out := make([]objectiveevents.DeathInstant, 0, len(deaths))
	for _, d := range deaths {
		out = append(out, objectiveevents.DeathInstant{
			XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS),
		})
	}
	return out
}

// logFlagCarriesCoverage journalise ce que le calque publie — et ce qu'il ecarte.
//
// DEUX PHRASES, PARCE QUE DEUX SITUATIONS APPELLENT DEUX REPONSES : une prise sans pont ou sans
// piste est un trou de rattachement, un depassement de simultaneite ENTRE PORTAGES FERMES est
// une contradiction entre faits dates. Le second est le seul qui accuse la regle elle-meme.
func logFlagCarriesCoverage(cov *FlagCarriesCoverage) {
	if cov == nil {
		return
	}
	if !cov.FlagFilm {
		slog.Debug("rejeu : film non reconnu CTF — aucun drapeau publie",
			"bursts", cov.Bursts, "captures", cov.Captures, "vols", cov.Steals)
		return
	}
	if cov.ClosedOverlaps > 0 {
		slog.Warn("rejeu : plus de deux drapeaux portes a la fois ENTRE PORTAGES FERMES — "+
			"la lecture se contredit", "depassements", cov.ClosedOverlaps,
			"depassementsTous", cov.Overlaps, "portages", cov.Carries)
	}
	slog.Info("rejeu : vie des drapeaux",
		"prises", cov.Openings, "portages", cov.Carries, "fermes", cov.Closed,
		"ouverts", cov.Open, "sansPont", cov.NoBridge, "sansPiste", cov.NoTrack,
		"horsFenetre", cov.OutOfWindow, "marqueurConfirme", cov.MarkerConfirmed,
		"marqueurObserve", cov.MarkerObserved, "socles", cov.Spawns,
		"simultaneite", cov.Overlaps, "porteursTuesAmbigus", cov.AmbiguousCarrierKills,
		"retoursAmbigus", cov.AmbiguousReturns)
}
