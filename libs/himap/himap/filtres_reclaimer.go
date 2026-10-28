// Package himap — filtres_reclaimer.go : LES FILTRES DE VISIBILITE DECLARES PAR LE JEU,
// et jamais lus jusqu'ici.
//
// CE FICHIER NE FILTRE RIEN. Il ne fait que LIRE et EXPOSER trois declarations du format,
// pour qu'elles soient MESUREES avant d'etre appliquees. Brancher un filtre sur la cuisson
// sans en connaitre le rendement est le piege que ce chantier paie deja ailleurs : un
// critere plausible qui efface la moitie de l'arene ne se voit que sur l'image, trop tard.
//
// LES TROIS DECLARATIONS, et ou elles vivent :
//
//	niveau INSTANCE (tag sbsp, bloc `instanced geometry instances`, champ `flags` @0x78)
//	  bit 12 « exclude from intel map » — le jeu dit lui-meme que cette instance n'a pas sa
//	  place sur une carte vue de dessus. C'est EXACTEMENT ce que nous produisons.
//
//	niveau SECTION (tag rtgo/mode, bloc `meshes`, champ `mesh flags` @+20 du pas de 60)
//	  bit 3 « mesh is custom shadow caster » — Reclaimer ecarte la section
//	  (HaloInfiniteCommon.cs:182). Nous n'ecartons ce bit qu'au niveau INSTANCE, sur un
//	  AUTRE champ (`mesh flags override` @0x110 du sbsp) : une section d'ombre portee par un
//	  modele partage passe donc a travers.
//
//	niveau LOD (bloc `LOD render data`, pas de 148)
//	  `lod flags` @0x8C : masque des niveaux de detail auxquels l'enregistrement appartient.
//	  `lod render flags` @0x8E : bit 0 « LOD has shadow proxies ».
//
// D'OU VIENNENT LES OFFSETS DE LOD, et pourquoi ils ne sont pas deduits du plugin. Le walk
// du plugin rend 138 pour `LOD render data` la ou le pas REEL est 148 : le type `_39`
// (tableau fixe) y est tabule a 32 octets alors que `vertex buffer indices` en declare 19
// entrees de 2 octets, soit 38. En corrigeant ce seul ecart la suite tombe juste —
// `index buffer index` @138 (0x8A), la valeur DEJA cablee et validee par le critere T1 du
// handoff — puis `lod flags` @140 et `lod render flags` @142.
//
// Le report n'est pas cru sur parole : `TestTemoinOffsetsLOD` le tranche sur les octets du
// jeu. Un champ a UN SEUL drapeau declare ne peut prendre que 0 ou 1 ; les offsets voisins,
// eux, etalent leurs valeurs. C'est ce contraste qui fait la preuve, pas la plausibilite.
package himap

// FlagExcludeFromIntelMap : bit 12 du champ `flags` (@0x78) d'une instance de geometrie.
//
// « intel map » est le nom que le jeu donne a la vue de dessus. Le drapeau designe donc la
// geometrie que ses propres auteurs ont exclue d'un fond de carte. Il est decode depuis
// l'origine (`Instance.Flags`) et n'a jamais ete lu.
const FlagExcludeFromIntelMap = 1 << 12

// FlagLODHasShadowProxies : bit 0 du champ `lod render flags` d'un enregistrement de LOD.
const FlagLODHasShadowProxies = 1 << 0

// Offsets, dans un enregistrement `meshes` (60 o) et `LOD render data` (148 o), des champs
// de filtrage. Cf. l'en-tete de fichier pour leur etablissement.
const (
	sectionOffMeshFlags  = 20   // `mesh flags`, u16
	lodOffLodFlags       = 0x8C // `lod flags`, u16
	lodOffLodRenderFlags = 0x8E // `lod render flags`, u16
)

// ExclueDeCarteIntel dit si le jeu exclut lui-meme cette instance d'une vue de dessus.
func (in Instance) ExclueDeCarteIntel() bool { return in.Flags&FlagExcludeFromIntelMap != 0 }

// FiltreSection est ce que le tag de geometrie declare, pour UN maillage, sur les trois
// criteres que Reclaimer applique.
type FiltreSection struct {
	// MeshFlags est le champ `mesh flags` de la section (@+20 du pas de 60).
	MeshFlags uint16
	// LodFlags / LodRenderFlags sont ceux du LOD RETENU par la regle de finesse
	// (`chooseLOD`) — pas du LOD 0, qui n'est pas forcement celui qu'on dessine.
	LodFlags       uint16
	LodRenderFlags uint16
	// Triangles est le nombre de triangles du LOD retenu, lu au descripteur d'indices. Il
	// sert a PONDERER : une section ecartee qui ne porte rien ne change pas l'image.
	Triangles int
}

// ProjecteurOmbre : la section est un volume d'ombre (bit 3 de `mesh flags`). Meme bit et
// meme sens que `Instance.ProjecteurOmbre`, mais porte par le MODELE et non par la pose.
func (f FiltreSection) ProjecteurOmbre() bool { return f.MeshFlags&FlagMeshCustomShadowCaster != 0 }

// ProxiesOmbre : le LOD retenu porte des mandataires d'ombre (`LOD has shadow proxies`).
func (f FiltreSection) ProxiesOmbre() bool { return f.LodRenderFlags&FlagLODHasShadowProxies != 0 }

// HorsLOD dit que le LOD retenu n'appartient PAS au niveau de detail demande.
//
// `lod flags` a zero est traite comme « appartient a tous » : un enregistrement qui ne
// declare aucun niveau n'est pas une section a jeter, c'est une declaration absente. Le
// distinguer evite de compter comme filtrage ce qui n'est qu'un champ vide.
func (f FiltreSection) HorsLOD(lod int) bool {
	if f.LodFlags == 0 || lod < 0 || lod > 15 {
		return false
	}
	return f.LodFlags&(1<<uint(lod)) == 0
}

// FiltreDuMaillage lit les trois declarations pour un maillage d'un tag `rtgo` ou `mode`.
//
// Rend false quand la section elle-meme est illisible — cas normal (un tag peut porter des
// maillages sans bloc de sections exploitable), signale a l'appelant plutot qu'avale.
func (a *RuntimeGeoAsset) FiltreDuMaillage(meshIndex int) (FiltreSection, bool) {
	var f FiltreSection
	if a.geo.SectionsTarget < 0 {
		return f, false
	}
	absS, sizeS := a.info.blockAbs(a.geo.SectionsTarget)
	if absS < 0 || sizeS <= 0 || sizeS%sectionStride != 0 {
		return f, false
	}
	if meshIndex < 0 || meshIndex >= sizeS/sectionStride {
		return f, false
	}
	o := absS + meshIndex*sectionStride + sectionOffMeshFlags
	if o+2 > len(a.info.tag) {
		return f, false
	}
	f.MeshFlags = u16(a.info.tag, o)
	l, ok := a.chooseLOD(meshIndex)
	if !ok || l.abs < 0 || l.abs+lodStride > len(a.info.tag) {
		return f, true
	}
	f.LodFlags = u16(a.info.tag, l.abs+lodOffLodFlags)
	f.LodRenderFlags = u16(a.info.tag, l.abs+lodOffLodRenderFlags)
	if l.indexBuf < len(a.indexDescs) {
		f.Triangles = a.indexDescs[l.indexBuf].Count / 3
	}
	return f, true
}

// TrianglesDuMaillage rend le nombre de triangles du LOD retenu, SANS decoder le maillage.
//
// C'est ce qui rend une mesure de masse tenable : decoder les sommets d'un module entier
// demande des giga-octets, alors que la ponderation par les triangles se lit dans les
// descripteurs du tag.
func (a *RuntimeGeoAsset) TrianglesDuMaillage(meshIndex int) (int, bool) {
	l, ok := a.chooseLOD(meshIndex)
	if !ok || l.indexBuf >= len(a.indexDescs) {
		return 0, false
	}
	return a.indexDescs[l.indexBuf].Count / 3, true
}

// LODsDuMaillage rend les couples (`lod flags`, `lod render flags`) de TOUS les niveaux de
// detail d'un maillage — pas seulement de celui que la regle de finesse retient.
func (a *RuntimeGeoAsset) LODsDuMaillage(meshIndex int) [][2]uint16 {
	var out [][2]uint16
	for _, l := range a.lods(meshIndex) {
		if l.abs < 0 || l.abs+lodStride > len(a.info.tag) {
			continue
		}
		out = append(out, [2]uint16{
			u16(a.info.tag, l.abs+lodOffLodFlags),
			u16(a.info.tag, l.abs+lodOffLodRenderFlags),
		})
	}
	return out
}

// U16DuLOD lit un u16 a un offset ARBITRAIRE du k-ieme enregistrement de LOD d'un maillage.
//
// Reservee au temoin d'offsets : elle n'a de sens que pour comparer des offsets CONCURRENTS
// sur les memes octets — meme role que `DequantSigne` pour la dequantification. Ne jamais
// s'en servir pour produire une carte.
func (a *RuntimeGeoAsset) U16DuLOD(meshIndex, k, offset int) (uint16, bool) {
	lods := a.lods(meshIndex)
	if k < 0 || k >= len(lods) || lods[k].abs < 0 {
		return 0, false
	}
	o := lods[k].abs + offset
	if o < 0 || o+2 > len(a.info.tag) {
		return 0, false
	}
	return u16(a.info.tag, o), true
}

// NbLODsDuMaillage rend le nombre d'enregistrements de LOD d'un maillage.
func (a *RuntimeGeoAsset) NbLODsDuMaillage(meshIndex int) int { return len(a.lods(meshIndex)) }

// PorteLeLOD dit si un maillage possede AU MOINS UN enregistrement appartenant au niveau de
// detail demande.
//
// C'est la question que pose vraiment le filtre de LOD, et elle ne se confond pas avec
// `FiltreSection.HorsLOD` : celle-la juge le niveau que NOUS avons retenu, celle-ci juge la
// section entiere. Reclaimer ne rabat pas sur un autre niveau — une section dont aucun
// enregistrement ne correspond au LOD demande est SAUTEE.
//
// Une section sans enregistrement de LOD, ou dont tous les `lod flags` sont a zero, est
// tenue pour PRESENTE a tous les niveaux : un champ vide n'est pas un refus.
func (a *RuntimeGeoAsset) PorteLeLOD(meshIndex, lod int) bool {
	lods := a.LODsDuMaillage(meshIndex)
	if len(lods) == 0 || lod < 0 || lod > 15 {
		return true
	}
	for _, l := range lods {
		if l[0] == 0 || l[0]&(1<<uint(lod)) != 0 {
			return true
		}
	}
	return false
}
