import { createContext, type ReactNode, useCallback, useContext, useEffect, useRef, useState } from "react";
import { ErroHTTP } from "@/servicos/http";

import { consultarSessao, encerrarSessao, type Sessao } from "@/servicos/sessao";

const Contexto = createContext<{ sessao?: Sessao; erro: string; carregar: () => void; sair: () => Promise<void> } | null>(null);
export function ProvedorDeSessao({ children }: { children: ReactNode }) {
	const [sessao, setSessao] = useState<Sessao>();
	const [erro, setErro] = useState("");
	const controlador = useRef<AbortController | null>(null);
	const carregar = useCallback(() => {
		controlador.current?.abort();
		const c = new AbortController();
		controlador.current = c;
		setSessao(undefined);
		setErro("");
		consultarSessao(c.signal)
			.then((v) => {
				if (!c.signal.aborted) setSessao(v);
			})
			.catch((e) => {
				if (!c.signal.aborted) {
					if (e instanceof ErroHTTP && e.status === 401) setSessao({ autenticado: false, login_disponivel: true });
					else setErro(e instanceof ErroHTTP ? e.message : "Não foi possível consultar a sessão.");
				}
			});
	}, []);
	useEffect(() => {
		carregar();
		const expirar = () => {
			controlador.current?.abort();
			setSessao({ autenticado: false, login_disponivel: true });
		};
		const canal = typeof BroadcastChannel === "function" ? new BroadcastChannel("aprova:sessao") : null;
		if (canal) canal.onmessage = expirar;
		window.addEventListener("focus", carregar);
		window.addEventListener("pageshow", carregar);
		window.addEventListener("aprova:sessao-expirada", expirar);
		return () => {
			controlador.current?.abort();
			canal?.close();
			window.removeEventListener("focus", carregar);
			window.removeEventListener("pageshow", carregar);
			window.removeEventListener("aprova:sessao-expirada", expirar);
		};
	}, [carregar]);
	useEffect(() => {
		if (!sessao?.autenticado) return;
		const timer = setTimeout(
			() => setSessao({ autenticado: false, login_disponivel: true }),
			Math.max(0, Date.parse(sessao.expira_em) - Date.now()),
		);
		return () => clearTimeout(timer);
	}, [sessao]);
	const logout = useRef<Promise<void> | null>(null);
	function sair() {
		logout.current ??= encerrar().finally(() => {
			logout.current = null;
		});
		return logout.current;
	}
	async function encerrar() {
		if (!sessao?.autenticado) return;
		try {
			await encerrarSessao(sessao.csrf_token);
		} catch (e) {
			if (!(e instanceof ErroHTTP && e.status === 401)) throw e;
		}
		controlador.current?.abort();
		setSessao({ autenticado: false, login_disponivel: true });
		if (typeof BroadcastChannel === "function") {
			const canal = new BroadcastChannel("aprova:sessao");
			canal.postMessage("sair");
			canal.close();
		}
	}
	return <Contexto value={{ sessao, erro, carregar, sair }}>{children}</Contexto>;
}
export function useSessao() {
	const v = useContext(Contexto);
	if (!v) throw new Error("Provedor de sessão ausente");
	return v;
}
