package revision

// cadre.go — LE CADRE HISTORIQUE DU HACHAGE : le chemin du module n est pas de la grammaire
// (decision D2-F du plan `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D2_2026-10-28.md`).
//
// Deux choses de l empreinte nomment l emplacement de la brique dans son module : les chaines
// d import que le flux de jetons garde ([jetonsDe]), et les dossiers des paquets importes, qui
// prefixent leurs fichiers dans le hachage et s ecrivent dans les goldens de perimetre
// ([Module.Fermeture]). Toutes deux entrent dans le hachage sous leur nom HISTORIQUE — celui du
// temps ou la brique vivait sous `apps/go-api/internal/games/halo_infinite/film` — quel que soit
// le module qui la porte aujourd hui. Tant que la brique vit a cet endroit, la traduction est
// l identite ; quand elle devient le module `libs/film`, aucune empreinte ne bouge.
//
// LE SENS DE LA TRADUCTION. Les declarations (racines de [CouchesRevisees], goldens de perimetre)
// parlent en noms historiques ; seul l acces au disque ([Module.SurDisque]) et la lecture des
// imports ([Module.importsDuPaquet]) passent par le nom courant.

import (
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

// moduleHistorique : le module qui portait la brique quand le cadre a ete fige.
const moduleHistorique = "levelup/go-api"

// dossierHistoriqueDeLaBrique : la racine de la brique, relative a [moduleHistorique].
const dossierHistoriqueDeLaBrique = "internal/games/halo_infinite/film"

// cadreHistorique : le chemin d import historique de la racine de la brique — le prefixe sous
// lequel toute chaine d import de la brique entre dans le hachage.
const cadreHistorique = moduleHistorique + "/" + dossierHistoriqueDeLaBrique

// cheminModuleCourant : le chemin d import COURANT de la racine de la brique.
//
// LU ET NON ECRIT : c est le chemin d import de ce paquet, dont on retire `/revision`. Une
// constante devrait etre changee a la main le jour ou la brique change de module, et l oubli
// rendrait des empreintes fausses ; le chemin lu suit le paquet. Le contrat que cette lecture
// suppose — `revision` vit directement sous la racine de la brique — est tenu par
// `TestCheminModuleCourantEstLaRacineDeLaBrique`.
var cheminModuleCourant = path.Dir(reflect.TypeFor[Resultat]().PkgPath())

// litteralDansLeCadre rend le litteral d un jeton chaine tel qu il entre dans le hachage : s il
// commence par le chemin d import courant de la brique (`brique`), ce prefixe est remplace par
// [cadreHistorique] ; sinon il est rendu tel quel.
//
// Le prefixe est reconnu A LA FRONTIERE D UN CHEMIN : suivi de `/` ou de la fin du litteral,
// jamais d un autre caractere (`.../filmprofile` n est pas sous `.../film`). Les deux formes de
// chaine du langage sont lues, interpretee et brute.
func litteralDansLeCadre(lit, brique string) string {
	if len(lit) < 2 || (lit[0] != '"' && lit[0] != '`') {
		return lit
	}
	guillemet := lit[:1]
	reste, ok := strings.CutPrefix(lit[1:], brique)
	if !ok || (reste != guillemet && !strings.HasPrefix(reste, "/")) {
		return lit
	}
	return guillemet + cadreHistorique + reste
}

// cheminBrique rend le chemin d import courant de la brique pour ce module : [cheminModuleCourant],
// sauf quand un test en a pose un autre pour simuler le module de demain.
func (m Module) cheminBrique() string {
	if m.brique != "" {
		return m.brique
	}
	return cheminModuleCourant
}

// dossierDeLaBrique rend la racine de la brique, relative au module (vide quand le module EST la
// brique), et faux quand la brique ne vit pas dans ce module — un module de fixture, ou rien ne
// se traduit.
func (m Module) dossierDeLaBrique() (string, bool) {
	brique := m.cheminBrique()
	if brique == m.Chemin {
		return "", true
	}
	return strings.CutPrefix(brique, m.Chemin+"/")
}

// historique traduit un dossier relatif au module courant vers son nom historique : sous la
// racine de la brique, la racine courante est remplacee par [dossierHistoriqueDeLaBrique] ;
// ailleurs, le dossier est rendu tel quel.
func (m Module) historique(rel string) string {
	courant, ok := m.dossierDeLaBrique()
	if !ok {
		return rel
	}
	if reste, dedans := sousDossier(rel, courant); dedans {
		return path.Join(dossierHistoriqueDeLaBrique, reste)
	}
	return rel
}

// courant est la traduction inverse de [Module.historique] : un nom historique vers le dossier
// relatif au module courant.
func (m Module) courant(nom string) string {
	courant, ok := m.dossierDeLaBrique()
	if !ok {
		return nom
	}
	if reste, dedans := sousDossier(nom, dossierHistoriqueDeLaBrique); dedans {
		return path.Join(courant, reste)
	}
	return nom
}

// SurDisque rend le dossier, sur le disque, du dossier de nom historique `nom` (une racine de
// [Couche], une ligne `paquet` d un golden de perimetre) dans le module courant.
func (m Module) SurDisque(nom string) string {
	return filepath.Join(m.Racine, filepath.FromSlash(m.courant(nom)))
}

// sousDossier dit si `rel` est `dossier` ou un de ses descendants, et rend la partie qui suit.
// Un `dossier` vide contient tout.
func sousDossier(rel, dossier string) (string, bool) {
	if dossier == "" || rel == dossier {
		return strings.TrimPrefix(rel, dossier), true
	}
	return strings.CutPrefix(rel, dossier+"/")
}
