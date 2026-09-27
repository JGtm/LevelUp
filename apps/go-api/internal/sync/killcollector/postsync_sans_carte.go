package killcollector

// postsync_sans_carte.go — LE REGISTRE DES MATCHS DEJA CONSTATES SANS CARTE (revue du correctif
// J7, 2026-09-27).
//
// # LE COUT QU IL FERME
//
// Un match sans carte resolue ne quitte jamais le backlog (aucun marqueur terminal : il redevient
// decodable le jour ou le catalogue de bornes connait sa carte). A l etat stable — des dizaines de
// matchs Forge en tete du backlog —, chaque cycle post-sync relisait le nom de carte de chacun
// (jusqu a 8 x 64 = 512 lectures, sous le verrou de passe) et rejournalisait le meme WARN, sans
// fin. Le registre retient, EN MEMOIRE et par titre, les matchs deja constates sans carte : un
// cycle suivant les saute sans relire leur carte ni journaliser (un compteur les dit).
//
// # QUAND IL S OUBLIE
//
//	le catalogue change   le registre est tenu pour UNE empreinte du catalogue de bornes (hachage
//	                      des entrees chargees, [empreinteDuCatalogue]) ; le catalogue est relu a
//	                      chaque cycle, et une autre empreinte vide le registre : chaque match est
//	                      relu une fois sous le nouveau catalogue.
//	le processus redemarre le registre n est pas persiste.
//	le match quitte le    quand un cycle a lu le backlog JUSQU A SA FIN, les entrees qu il n a pas
//	backlog               vues sont retirees ([registreSansCarte.elaguer]).
//
// RESIDU ACCEPTE : un nom de carte qui arriverait en base apres le constat (registre des matchs
// complete plus tard) n est relu qu au prochain changement de catalogue ou redemarrage.
//
// # LA JAUGE
//
// [CompteurPostSyncSansCarte] publie la taille du registre (ou tout le backlog quand la carte
// n est pas cablee ce cycle) : `killsource_postsync_backlog_restant` moins elle est le retard que
// le decodeur PEUT resorber.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sync"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
)

const (
	// CompteurPostSyncSansCarte : la JAUGE des matchs du backlog constates sans carte resolue.
	CompteurPostSyncSansCarte = "killsource_postsync_backlog_sans_carte"
	// metricSansCarteDejaConstates : matchs SAUTES par un cycle parce que deja constates sans carte
	// sous le catalogue courant — ni relecture de carte, ni journal. Il compte des sauts, pas des
	// matchs distincts.
	metricSansCarteDejaConstates = "killsource_postsync_sans_carte_deja_constates"
)

// registreSansCarte : les matchs constates sans carte sous UNE empreinte du catalogue de bornes.
type registreSansCarte struct {
	mu        sync.Mutex
	empreinte string
	ids       map[string]struct{}
}

var (
	registresMu        sync.Mutex
	registresSansCarte = map[string]*registreSansCarte{}
)

// registreDuTitre rend le registre du titre, cree au premier appel.
func registreDuTitre(slug string) *registreSansCarte {
	registresMu.Lock()
	defer registresMu.Unlock()
	r, ok := registresSansCarte[slug]
	if !ok {
		r = &registreSansCarte{ids: map[string]struct{}{}}
		registresSansCarte[slug] = r
	}
	return r
}

// registreDuCycle rend le registre du titre ACCORDE au catalogue du cycle, ou nil quand la carte
// n est pas cablee (sans catalogue, aucun constat n a de sens : tout match est ecarte).
func registreDuCycle(ctx context.Context, slug string, deps DepsCapture) *registreSansCarte {
	if !deps.Cablee() {
		return nil
	}
	empreinte := empreinteDuCatalogue(ctx, deps.Bounds)
	if empreinte == "" {
		return nil
	}
	r := registreDuTitre(slug)
	r.accorder(ctx, empreinte)
	return r
}

// empreinteDuCatalogue : le SHA-256 des entrees chargees, serialisees par `encoding/json` (cles
// de map triees, donc stable). "" quand la serialisation echoue — le registre est alors inutilise
// pour ce cycle, jamais tenu sous une empreinte fausse.
func empreinteDuCatalogue(ctx context.Context, cat *decfilm.MapQuantCatalog) string {
	blob, err := json.Marshal(cat)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: killsource — empreinte du catalogue de bornes impossible ; "+
			"registre des matchs sans carte inutilise ce cycle", "err", err)
		return ""
	}
	somme := sha256.Sum256(blob)
	return hex.EncodeToString(somme[:])
}

// accorder vide le registre si l empreinte du catalogue a change.
func (r *registreSansCarte) accorder(ctx context.Context, empreinte string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.empreinte == empreinte {
		return
	}
	if len(r.ids) > 0 {
		slog.InfoContext(ctx, "post-sync: killsource — catalogue de bornes change, les matchs "+
			"constates sans carte seront relus", "constates", len(r.ids))
	}
	r.empreinte = empreinte
	r.ids = map[string]struct{}{}
}

// filtrer rend les ids que le registre ne connait pas, dans l ordre, et le nombre de sautes.
func (r *registreSansCarte) filtrer(ids []string) (inconnus []string, sautes int) {
	if r == nil {
		return ids, 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		if _, connu := r.ids[id]; connu {
			sautes++
			continue
		}
		inconnus = append(inconnus, id)
	}
	return inconnus, sautes
}

// noter inscrit des matchs constates sans carte.
func (r *registreSansCarte) noter(ids []string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		r.ids[id] = struct{}{}
	}
}

// elaguer ne garde que les ids LUS au backlog : a appeler seulement quand le cycle l a lu jusqu a
// sa fin — un match absent de cette lecture l a quitte.
func (r *registreSansCarte) elaguer(lus map[string]bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range r.ids {
		if !lus[id] {
			delete(r.ids, id)
		}
	}
}

// taille : le nombre de matchs constates sans carte.
func (r *registreSansCarte) taille() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.ids)
}

// publierSansCarte publie la jauge : la taille du registre, ou tout le backlog quand la carte
// n est pas cablee ce cycle (aucun match ne peut alors etre decode).
func publierSansCarte(reg *registreSansCarte, total int) {
	n := total
	if reg != nil {
		n = min(reg.taille(), total)
	}
	observability.SetInt(CompteurPostSyncSansCarte, int64(n))
}
