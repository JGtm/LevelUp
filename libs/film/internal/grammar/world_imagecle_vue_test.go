package grammar

// world_imagecle_vue_test.go — LA VUE D UNE LIAISON D IMAGE-CLE SE LIT, ELLE NE S ATTRIBUE PAS
// (lot 5.13.1).
//
// Les deux bits de tete d un identifiant d image-cle sont l ESPACE DE NOMS DE LA VUE (`vue + 8`,
// `FUN_142f2e174`). Ces tests fixent les trois proprietes que [World.BindImageCle] en tire, et
// qui remplacent l attribution d office de `BindWildcard` :
//
//	le PREMIER espace de noms rencontre est celui de la vue ENREGISTREE (rang 0) ;
//	un espace de noms DIFFERENT ne s attribue a aucun rang -> vue INCONNUE, qui ne rejette rien ;
//	la GENERATION reste inconnue (`GenAny`) — l image-cle ne la porte pas.

import "testing"

func TestBindImageCleLitLaVueDansLEspaceDeNoms(t *testing.T) {
	w := NewWorld(nil)
	w.BindImageCle(1, 512, BipedTypeIndex)
	s, ok := w.slots[512]
	if !ok {
		t.Fatalf("slot 512 non lie")
	}
	if s.Vue != vueDeLImageCle {
		t.Fatalf("vue = %d, attendu %d (la vue enregistree)", s.Vue, vueDeLImageCle)
	}
	if !s.GenAny {
		t.Fatalf("GenAny = false : l image-cle ne porte pas la generation du datum")
	}
	if s.TypeIndex != BipedTypeIndex {
		t.Fatalf("archetype = %d, attendu %d", s.TypeIndex, BipedTypeIndex)
	}
}

func TestBindImageCleUnSecondEspaceDeNomsNEstAttribueAAucunRang(t *testing.T) {
	w := NewWorld(nil)
	w.BindImageCle(1, 512, BipedTypeIndex) // fixe l espace de noms de la vue enregistree
	w.BindImageCle(2, 7140, BipedTypeIndex)
	s := w.slots[7140]
	if s.Vue != vueInconnue {
		t.Fatalf("vue = %d, attendu %d (espace de noms etranger a la vue enregistree)",
			s.Vue, vueInconnue)
	}
	// La vue inconnue ne rejette rien : la garde de table de vue laisse passer ce slot.
	w.PoserVueCourante(0)
	if !w.VuePossede(7140) {
		t.Fatalf("une vue INCONNUE doit laisser passer, comme une generation inconnue")
	}
}

func TestBindImageCleLaVueEnregistreeResteLaMemeEntreDeuxImagesCles(t *testing.T) {
	w := NewWorld(nil)
	w.BindImageCle(1, 512, BipedTypeIndex)
	w.BindImageCle(1, 513, BipedTypeIndex) // meme espace de noms, image-cle suivante
	if s := w.slots[513]; s.Vue != vueDeLImageCle {
		t.Fatalf("vue = %d, attendu %d", s.Vue, vueDeLImageCle)
	}
}
