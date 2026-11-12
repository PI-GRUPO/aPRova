import { useCallback, useSyncExternalStore } from "react";

export type Tema = "escuro" | "claro";

const chave = "aprova-tema";
const ouvintes = new Set<() => void>();

function temaAtual(): Tema {
	return document.documentElement.dataset.tema === "claro" ? "claro" : "escuro";
}

function assinar(ouvinte: () => void) {
	ouvintes.add(ouvinte);
	return () => ouvintes.delete(ouvinte);
}

function aplicar(tema: Tema) {
	document.documentElement.dataset.tema = tema;
	try {
		localStorage.setItem(chave, tema);
	} catch {}
	for (const ouvinte of ouvintes) ouvinte();
}

export function useTema() {
	const tema = useSyncExternalStore(assinar, temaAtual);
	const alternar = useCallback(() => aplicar(temaAtual() === "escuro" ? "claro" : "escuro"), []);
	return { tema, alternar };
}
