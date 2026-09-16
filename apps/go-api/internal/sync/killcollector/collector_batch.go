package killcollector

// collector_batch.go — LA TRADUCTION `resultat du decodeur -> lignes ecrivables`, et elle
// seule : `BuildKillSourceBatch`, `killToInsert`, `pctToU8`.
//
// Sorti de `collector.go` par deplacement pur au lot 2.7 (2026-09-16, scission des fichiers de
// plus de 500 lignes) : aucune ligne de logique n'a change. C'est la frontiere la plus nette du
// fichier d'origine — ici on ne decide rien, on ne parle ni au reseau ni a la base : on traduit.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/facts"
	"levelup/go-api/internal/games/halo_infinite/film/facts/killsource"
	"levelup/go-api/internal/persist"
)

// BuildKillSourceBatch traduit le resultat du decodeur en lignes ecrivables.
//
// C EST LA SEULE FONCTION DU LOT OU LES TROIS ETATS DE L ASSISTANT PEUVENT SE PERDRE, et elle
// est exportee pour que le backfill (session suivante) emprunte EXACTEMENT le meme chemin — une
// seconde traduction serait une seconde chance de les confondre.
//
// Les trois traductions qui ne sont pas des copies de champ :
//
//	Assist.Index == -1        -> nil (NULL). Ecrire -1 fabriquerait un indice de replication.
//	KillerDamage.Known false  -> nil (NULL). « Non mesure » n est jamais zero.
//	AssistDamage              -> nil AUSSI quand l assistant n est pas NOMME. ⚠ `Known` peut
//	                             valoir VRAI avec un `Name` VIDE (assistant REFUSE) : le champ
//	                             etait present, la part est mesuree, mais son PORTEUR est refuse
//	                             — et sans assistant nomme ce bloc porte une constante par film
//	                             qui ne veut rien dire (elle vaut 20 sur certains films). Le
//	                             persister refuse cette ligne, et il a raison.
//
// AUCUN PLAFOND A 100 sur les parts : 1,7 % des kill-events vont jusqu a 228, ce sont des
// donnees. Le seul plafond applique est celui du TYPE (uint8, 255) — et si une valeur le
// depassait, c est le type qu il faudrait elargir, pas la valeur qu il faudrait ecreter.
func BuildKillSourceBatch(matchID string, res *killsource.Result, ids MatchIdentities) persist.KillSourceBatch {
	batch := persist.KillSourceBatch{
		MatchID:     matchID,
		DecoderRev:  facts.Rev,
		Publishable: res.LineByLinePublishable(),
		Deaths:      make([]persist.KillEventInsert, 0, len(res.Kills)),
	}
	for i := range res.Kills {
		batch.Deaths = append(batch.Deaths, killToInsert(&res.Kills[i], ids))
	}
	return batch
}

// killToInsert : UNE mort. Decoupe de [BuildKillSourceBatch] pour rester sous le plafond de
// longueur du depot (80 lignes).
//
// LES TROIS NOMS PASSENT PAR [MatchIdentities.Resoudre] — victime, tueur, assistant. Aucun ne se
// resout « a la main » : le film peut donner un gamertag OU un xuid, et la regle qui les
// distingue n existe qu a un seul endroit.
func killToInsert(k *killsource.Kill, ids MatchIdentities) persist.KillEventInsert {
	victimeXUID, victimeNom := ids.Resoudre(k.Victim)
	tueurXUID, tueurNom := ids.Resoudre(k.Feed.Killer)
	d := persist.KillEventInsert{
		TimeMS:             k.TimeMS,
		VictimGamertag:     victimeNom,
		VictimXUID:         victimeXUID,
		FeedKillerGamertag: tueurNom,
		FeedKillerXUID:     tueurXUID,
		FeedPresent:        k.Feed.Present,
		AssistGamertag:     k.Assist.Name,
		AssistKnown:        k.Assist.Known,
		AssistRejected:     k.Assist.Rejected,
		AssistExtra:        k.Assist.Extra,
		SourceTag:          k.Source.Tag,
		Diverges:           k.Diverges,
		ReadPath:           string(k.Read.Path),
		ReadOrigin:         string(k.Read.Origin),
	}
	if k.Assist.Name != "" {
		d.AssistXUID, d.AssistGamertag = ids.Resoudre(k.Assist.Name)
	}
	// La categorie voyage AVEC le tag : les deux sortent de la meme lecture du dead-state, et
	// le persister refuse une demi-source. Tag nul = source NON MESUREE, donc categorie vide.
	if k.Source.Tag != 0 {
		d.SourceCategory = k.Source.Category.Name()
	} else {
		// Sans source mesuree, la divergence est INDEFINISSABLE (elle compare les deux
		// verites). Ecrire FALSE la ferait passer pour « mesure : pas de divergence ».
		d.Diverges = false
	}
	if k.Assist.Index >= 0 {
		idx := k.Assist.Index
		d.AssistIndex = &idx
	}
	if k.KillerDamage.Known {
		d.KillerDamagePct = pctToU8(k.KillerDamage.Pct)
	}
	if k.AssistDamage.Known && k.Assist.Name != "" {
		d.AssistDamagePct = pctToU8(k.AssistDamage.Pct)
	}
	return d
}

// pctToU8 : la part de degats, bornee par le TYPE et par lui seul.
//
// ⚠ `uint8` plafonne a 255 pour un maximum MESURE a 228 : la marge existe mais elle est mince.
// Si une valeur superieure apparaissait, c est le TYPE qu il faudrait elargir — pas la valeur
// qu il faudrait plafonner. Le log est la pour qu on l apprenne au lieu de le subir.
func pctToU8(pct int) *uint8 {
	if pct < 0 {
		pct = 0
	}
	if pct > 255 {
		slog.Warn("killsource: part de degats au-dela de la capacite du type UTINYINT — "+
			"ELARGIR LE TYPE, ne pas plafonner la valeur", "pct", pct)
		pct = 255
	}
	v := uint8(pct)
	return &v
}
