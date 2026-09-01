package replayartifacts

// mvar_rattrapage.go — QUAND LA CHAINE RECUPERE UN FILM, ELLE COMBLE LE CATALOGUE DE CARTES.
//
// # Le trou que ce fichier ferme
//
// L'origine d'un ramassage (`Pickup.origin`, schema 32) se lit sur la CARTE : le catalogue
// `map_weapon_pads.json` declare les points d'apparition d'objet ramassable. Une carte ABSENTE
// de ce catalogue rend donc `spawner` impossible, et l'artefact le dit
// (`coverage.pickups.spawnPointsState == "map_absent"`). Le catalogue couvrait 72 cartes sur la
// centaine jouee : chaque carte manquante etait un rejeu ampute, et rien ne la comblait jamais.
//
// # Ou ce rattrapage vit, et pourquoi PAS ailleurs
//
// Trois endroits auraient pu l'accueillir, deux sont exclus par doctrine :
//
//	le SYNC RAPIDE          exclu. Il reste INTACT : c'est le chemin que l'utilisateur attend,
//	                        et lui ajouter un appel reseau conditionnel le ralentirait pour une
//	                        donnee dont il n'a pas besoin.
//	la CUISSON d'artefact   exclu. Elle est OFFLINE-PURE et le reste — une generation
//	                        d'artefact ne telecharge RIEN, sans exception. C'est la regle qui
//	                        garantit qu'un artefact est reproductible depuis le disque.
//	le FETCH DE FILM        ICI. Cette chaine telecharge deja les chunks du film : elle est en
//	                        ligne par nature, elle connait le match, et un `.mvar` de plus y
//	                        est un appel marginal.
//
// # Ce qu'il fait, et rien de plus
//
// Carte ABSENTE du catalogue -> un appel UGC, depot du `.mvar` au cache de donnees, ajout d'une
// CLE NEUVE au catalogue. Carte PRESENTE -> il ne fait RIEN, pas meme un appel : il ne verifie
// pas si elle a derive. Verifier couterait un appel API PAR MATCH deja connu, ce qui est
// exactement la lourdeur que ce chemin doit eviter. La derive des cartes connues se traite par
// `mapopads-build --refresh-drifted`, a la main ou en maintenance planifiee.
//
// # Best-effort STRICT
//
// Auth, reseau, parse, ecriture : TOUT echec est journalise, compte, et le fetch de film
// CONTINUE. Un film ne se perd jamais a cause d'un `.mvar`. C'est la seule promesse que ce
// fichier fait sur son propre echec, et elle est testee.

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/mapcatalog"
)

// entryFromMvarFn est la couture qui rend le CHEMIN NOMINAL testable sans `.mvar` reel.
//
// Les fixtures `.mvar` du depot vivent hors de l'arbre (`.ai/V7.5/dumps/mapvar/`) et les tests
// qui en dependent SAUTENT quand elles manquent — donc en CI. Sans cette couture, la promesse
// « carte inconnue -> entree ajoutee, existantes byte-identiques » ne serait verifiee nulle
// part la ou elle compte. Le parsing lui-meme est couvert par les tests de `mapvar`.
//
// Le code de production ne la reassigne jamais.
var entryFromMvarFn = mapcatalog.EntryFromMvar

// bilanRattrapage compte ce qu'une passe de rattrapage a fait. Publie au journal du cycle :
// sans denominateurs, « une carte ajoutee » ne se juge pas.
type bilanRattrapage struct {
	dejaLa, ajoutees, sansMapID, horsObjectifs, echecs int
}

// MvarFetcher est ce que le rattrapage exige de son fournisseur UGC — une INTERFACE, pour que
// le test puisse echouer a volonte sans reseau ni jeton.
type MvarFetcher interface {
	// FetchMvarForMap rend le contenu du `.mvar` d'une carte et le nom de fichier retenu.
	FetchMvarForMap(ctx context.Context, mapID, mvarFile string) (blob []byte, base string, err error)
}

// rattraperCartesAbsentes complete le catalogue pour les cartes du lot qui n'y sont pas.
//
// Appelee UNE fois par lot et non par match : plusieurs films d'une meme carte n'ouvrent qu'un
// seul appel, et l'ordre du lot ne change pas le resultat.
func rattraperCartesAbsentes(ctx context.Context, d Deps, work []buildWork,
	fetcher MvarFetcher,
) bilanRattrapage {
	var b bilanRattrapage
	if fetcher == nil {
		return b
	}
	catPath := title.NewPathResolver(d.RepoRoot).MapWeaponPadsPath(d.TitleSlug)
	cat, err := replay.LoadMapWeaponPads(catPath)
	if err != nil {
		slog.WarnContext(ctx, "rattrapage mvar: catalogue des cartes illisible — rattrapage "+
			"saute, les films sont recuperes normalement", "err", err, "path", catPath)
		b.echecs++
		return b
	}
	objectifs, err := replay.LoadMapObjectives(
		title.NewPathResolver(d.RepoRoot).MapObjectivesPath(d.TitleSlug))
	if err != nil {
		slog.WarnContext(ctx, "rattrapage mvar: catalogue d objectifs illisible — sans lui on "+
			"ne sait pas quel fichier de variante demander", "err", err)
		b.echecs++
		return b
	}
	vues := make(map[string]bool, len(work))
	for _, w := range work {
		mapID := w.facts.MapID
		switch {
		case mapID == "":
			b.sansMapID++
			continue
		case vues[mapID]:
			continue
		}
		vues[mapID] = true
		if _, deja := cat.Maps[mapID]; deja {
			// CARTE CONNUE : on ne touche a RIEN, et on ne verifie pas sa derive. Voir l'en-tete.
			b.dejaLa++
			continue
		}
		e, ok := objectifs.Maps[mapID]
		if !ok {
			// Sans entree au catalogue d'objectifs, on ne sait pas QUEL fichier de variante
			// demander : la carte reste absente, et ca se compte.
			b.horsObjectifs++
			continue
		}
		if ajouterCarteAuCatalogue(ctx, d, fetcher, catPath, mapID, e) {
			b.ajoutees++
		} else {
			b.echecs++
		}
	}
	if b.ajoutees > 0 || b.echecs > 0 {
		slog.InfoContext(ctx, "rattrapage mvar: cartes absentes du catalogue",
			"ajoutees", b.ajoutees, "deja_presentes", b.dejaLa,
			"sans_map_id", b.sansMapID, "hors_catalogue_objectifs", b.horsObjectifs,
			"echecs", b.echecs)
	}
	return b
}

// ajouterCarteAuCatalogue fait le travail d'UNE carte. Rend faux sur tout echec — et l'echec
// n'interrompt jamais rien.
func ajouterCarteAuCatalogue(ctx context.Context, d Deps, fetcher MvarFetcher,
	catPath, mapID string, e replay.MapObjectivesEntry,
) bool {
	blob, base, err := fetcher.FetchMvarForMap(ctx, mapID, e.MvarFile)
	if err != nil {
		slog.WarnContext(ctx, "rattrapage mvar: telechargement echoue — la carte reste absente "+
			"du catalogue, le film est recupere normalement",
			"map_id", mapID, "err", err)
		return false
	}
	if err := deposerMvar(d, mapID, base, blob); err != nil {
		// Le depot est une TRACE, pas une dependance : on continue meme s'il echoue.
		slog.WarnContext(ctx, "rattrapage mvar: depot du .mvar au cache echoue",
			"map_id", mapID, "err", err)
	}
	entry, _, _, err := entryFromMvarFn(mapID, e, blob, base)
	if err != nil {
		slog.WarnContext(ctx, "rattrapage mvar: variante illisible — la carte reste absente",
			"map_id", mapID, "fichier", base, "err", err)
		return false
	}
	err = mapcatalog.AddEntry(catPath, mapID, entry)
	switch {
	case errors.Is(err, mapcatalog.ErrEntryExists):
		// Un autre film du meme lot, ou un autre processus, l'a ajoutee entre-temps. Ce n'est
		// pas un echec : l'ajout-seul a fait exactement son travail.
		return true
	case err != nil:
		slog.WarnContext(ctx, "rattrapage mvar: ecriture du catalogue echouee",
			"map_id", mapID, "err", err)
		return false
	}
	n := 0
	if entry.SpawnPoints != nil {
		n = len(*entry.SpawnPoints)
	}
	slog.InfoContext(ctx, "rattrapage mvar: carte AJOUTEE au catalogue",
		"map_id", mapID, "carte", e.PublicName, "socles", len(entry.Pads),
		"points_apparition", n)
	return true
}

// deposerMvar ecrit le `.mvar` telecharge au cache de donnees, sous le map_id.
//
// POURQUOI LE GARDER : il rend la passe REJOUABLE hors ligne. `mapopads-build --from` relit un
// dossier de `.mvar` ; sans depot, regenerer le catalogue exigerait de re-telecharger.
func deposerMvar(d Deps, mapID, base string, blob []byte) error {
	dir := filepath.Join(d.CacheRoot, "mvar", mapID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, base), blob, 0o600)
}
