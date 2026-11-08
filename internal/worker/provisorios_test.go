package worker

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
)

func taskRunProvisorio() domain.TaskRun {
	return domain.TaskRun{ID: "c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a", TenantID: 58291734, Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 14}
}

func TestEmissorDeReciboProvisorioSoRegistraEDevolveSempreOMesmoRecibo(t *testing.T) {
	var saida bytes.Buffer
	emissor := NovoEmissorReciboProvisorio(slog.New(slog.NewJSONHandler(&saida, nil)))

	primeiro, err := emissor.Emitir(t.Context(), PedidoRecibo{TaskRun: taskRunProvisorio(), Gate: domain.GateDeRisco{Nivel: domain.RiscoAlto}})
	if err != nil {
		t.Fatalf("emitir: %v", err)
	}
	segundo, _ := emissor.Emitir(t.Context(), PedidoRecibo{TaskRun: taskRunProvisorio()})

	if len(primeiro.Hash) != 32 || primeiro.ID != taskRunProvisorio().ID {
		t.Errorf("recibo provisório = %+v", primeiro)
	}
	if !bytes.Equal(primeiro.Hash, segundo.Hash) || primeiro.ID != segundo.ID {
		t.Error("o mesmo task_run precisa devolver o mesmo recibo, como o contrato exige")
	}
	if registro := saida.String(); !strings.Contains(registro, "provisório") || !strings.Contains(registro, taskRunProvisorio().ID) {
		t.Errorf("o emissor provisório registra no log o que faria: %s", registro)
	}
}

func TestPublicadorProvisorioSoRegistra(t *testing.T) {
	var saida bytes.Buffer
	publicador := NovoPublicadorPRProvisorio(slog.New(slog.NewJSONHandler(&saida, nil)))

	err := publicador.Publicar(t.Context(), Publicacao{TaskRun: taskRunProvisorio(), Gate: domain.GateDeRisco{Nivel: domain.RiscoMedio}})

	if err != nil {
		t.Fatalf("publicar: %v", err)
	}
	if registro := saida.String(); !strings.Contains(registro, "provisório") || !strings.Contains(registro, taskRunProvisorio().ID) || !strings.Contains(registro, "medio") {
		t.Errorf("o publicador provisório registra o que publicaria: %s", registro)
	}
}
