import type { Consulta } from "@/servicos/fonte";

const validade = 10 * 60_000;
const limitePorConsulta = 40;
type Entrada<T> = { dados: T; venceEm: number };

export class CacheDeLeituras {
	private consultas = new Map<Consulta<unknown>, Map<string, Entrada<unknown>>>();
	geracao = 0;

	buscar<T>(consulta: Consulta<T>, chave: string) {
		const entrada = this.consultas.get(consulta)?.get(chave);
		return entrada && entrada.venceEm > Date.now() ? (entrada as Entrada<T>) : undefined;
	}

	guardar<T>(consulta: Consulta<T>, chave: string, dados: T) {
		let entradas = this.consultas.get(consulta);
		if (!entradas) {
			entradas = new Map();
			this.consultas.set(consulta, entradas);
		}
		entradas.delete(chave);
		entradas.set(chave, { dados, venceEm: Date.now() + validade });
		if (entradas.size > limitePorConsulta) {
			const primeira = entradas.keys().next();
			if (!primeira.done) entradas.delete(primeira.value);
		}
	}

	descartar(consulta: Consulta<unknown>, chave: string) {
		this.consultas.get(consulta)?.delete(chave);
	}

	limpar() {
		this.geracao++;
		this.consultas.clear();
	}
}
