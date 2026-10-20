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
//	le constat vieillit   passe [DureeDeVieDesConstatsSansCarte] (6 h), le match est relu : la
//	                      cause peut disparaitre SANS changement de catalogue (backfill des noms du
//	                      registre, traduction arrivee dans `asset_translations`).
//	le processus redemarre le registre n est pas persiste.
//	le match quitte le    quand un cycle a lu le backlog JUSQU A SA FIN, les entrees qu il n a pas
//	backlog               vues sont retirees ([registreSansCarte.elaguer]).
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
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
)

// DureeDeVieDesConstatsSansCarte : au-dela, un constat « sans carte » est OUBLIE et le match relu.
//
// POURQUOI UNE DUREE EN PLUS DE L EMPREINTE DU CATALOGUE. La cause d un constat peut disparaitre
// SANS que le catalogue de bornes change : un `map_name` brut (UUID) reecrit par le backfill des
// noms du registre (`BackfillRegistryNames`, action admin ou CLI), ou une traduction arrivee dans
// `asset_translations` que le resolveur du post-sync sait desormais lire. Six heures bornent ce
// retard a moins d une journee de jeu, pour un cout d une relecture de nom par match sans carte et
// par six heures (au plus 512 par cycle, cf. [PostSyncBacklogPagesMax]) — contre une a CHAQUE cycle
// avant le registre.
const DureeDeVieDesConstatsSansCarte = 6 * time.Hour

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
	// ids : l heure du CONSTAT de chaque match.
	ids map[string]time.Time
	// maintenant : l horloge (nil : time.Now) — la couture des tests de duree de vie.
	maintenant func() time.Time
}

// nouveauRegistreSansCarte : un registre vide, a l horloge du systeme.
func nouveauRegistreSansCarte() *registreSansCarte {
	return &registreSansCarte{ids: map[string]time.Time{}}
}

// heure : l heure du registre.
func (r *registreSansCarte) heure() time.Time {
	if r.maintenant != nil {
		return r.maintenant()
	}
	return time.Now()
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
		r = nouveauRegistreSansCarte()
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
	r.ids = map[string]time.Time{}
}

// filtrer rend les ids que le registre ne connait pas (ou plus : constat expire, retire), dans
// l ordre, et le nombre de sautes.
func (r *registreSansCarte) filtrer(ids []string) (inconnus []string, sautes int) {
	if r == nil {
		return ids, 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	maintenant := r.heure()
	for _, id := range ids {
		if constat, connu := r.ids[id]; connu {
			if maintenant.Sub(constat) < DureeDeVieDesConstatsSansCarte {
				sautes++
				continue
			}
			delete(r.ids, id) // constat expire : le match est relu (et reinscrit s il reste sans carte)
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
		r.ids[id] = r.heure()
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
