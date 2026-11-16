import { contaSlackMock } from "@/mocks/conta";
// Este adapter não chama OAuth, n8n ou API de identidade.
export async function simularVinculoSlack(conectado: boolean) {
	return { ...contaSlackMock, conectado };
}
