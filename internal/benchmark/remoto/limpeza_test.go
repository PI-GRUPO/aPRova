package remoto

import (
	"slices"
	"testing"
)

func TestUltimoCommitDaBranchTrazOSHAEAArvore(t *testing.T) {
	cliente, _ := clienteDeTeste(t)
	base, err := cliente.PublicarCommit(t.Context(), "dono/bench-go", "main", Commit{Mensagem: "base", Arquivos: map[string][]byte{"README.md": []byte("oi")}})
	if err != nil {
		t.Fatalf("publicar base: %v", err)
	}

	ultimo, err := cliente.UltimoCommit(t.Context(), "dono/bench-go", "main")

	if err != nil || ultimo != base {
		t.Errorf("último commit da main = %+v, %v; esperava %+v", ultimo, err, base)
	}
}

func TestFecharPRMudaOEstadoParaClosed(t *testing.T) {
	cliente, falso := clienteDeTeste(t)
	pr, err := cliente.AbrirPR(t.Context(), "dono/bench-go", "e2e-1", "main", "título", "corpo")
	if err != nil {
		t.Fatalf("abrir PR: %v", err)
	}

	if err := cliente.FecharPR(t.Context(), "dono/bench-go", pr.Numero); err != nil {
		t.Fatalf("fechar PR: %v", err)
	}

	if estado := texto(falso.prs[0], "state"); estado != "closed" {
		t.Errorf("estado do PR = %q, esperava closed", estado)
	}
}

func TestApagarBranchRemoveARef(t *testing.T) {
	cliente, falso := clienteDeTeste(t)
	if _, err := cliente.PublicarCommit(t.Context(), "dono/bench-go", "e2e-1", Commit{Mensagem: "x", Arquivos: map[string][]byte{"a.txt": []byte("a")}}); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	if err := cliente.ApagarBranch(t.Context(), "dono/bench-go", "e2e-1"); err != nil {
		t.Fatalf("apagar branch: %v", err)
	}

	if _, existe := falso.refs["refs/heads/e2e-1"]; existe {
		t.Error("a ref da branch continua lá")
	}
}

func TestBranchesComPrefixoListaSoAsQueComecamComEle(t *testing.T) {
	cliente, _ := clienteDeTeste(t)
	for _, branch := range []string{"e2e-100", "e2e-200", "main"} {
		if _, err := cliente.PublicarCommit(t.Context(), "dono/bench-go", branch, Commit{Mensagem: branch, Arquivos: map[string][]byte{"a.txt": []byte(branch)}}); err != nil {
			t.Fatalf("publicar %s: %v", branch, err)
		}
	}

	branches, err := cliente.BranchesComPrefixo(t.Context(), "dono/bench-go", "e2e-")
	slices.Sort(branches)

	if err != nil || !slices.Equal(branches, []string{"e2e-100", "e2e-200"}) {
		t.Errorf("branches = %v, %v", branches, err)
	}
}

func TestComentariosDevolveOCorpoDeCadaComentarioDoPR(t *testing.T) {
	cliente, falso := clienteDeTeste(t)
	falso.comentarios[7] = []string{"primeiro", "<!-- aprova:resultado-da-analise -->\nrisco baixo"}

	corpos, err := cliente.Comentarios(t.Context(), "dono/bench-go", 7)

	if err != nil || !slices.Equal(corpos, falso.comentarios[7]) {
		t.Errorf("comentários = %v, %v", corpos, err)
	}
}
