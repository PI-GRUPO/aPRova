import { createContext, type ReactNode, useCallback, useContext, useEffect, useEffectEvent, useMemo, useRef, useState } from "react";
import { Navigate, useLocation } from "react-router";
import { ErroHTTP } from "@/servicos/http";
import { consultarSessao, encerrarSessao, type Sessao } from "@/servicos/sessao";
import { CacheDeLeituras } from "./cache-de-leituras";

const Contexto = createContext<{
	sessao?: Sessao;
	cacheDeLeituras?: CacheDeLeituras;
	erro: string;
	carregar: () => void;
	sair: () => Promise<void>;
} | null>(null);
type EstadoDaSessao = { sessao?: Sessao; erro: string; carregando?: boolean; encerrada?: boolean };

export function ProvedorDeSessao({ children }: { children: ReactNode }) {
	const [estado, definirEstado] = useState<EstadoDaSessao>({ erro: "" });
	const { pathname } = useLocation();
	const controlador = useRef<AbortController | null>(null);
	const cancelarConsulta = useCallback(() => {
		controlador.current?.abort();
		controlador.current = null;
	}, []);
	const encerrarLocalmente = useCallback(() => {
		cancelarConsulta();
		definirEstado({ sessao: { autenticado: false, login_disponivel: true }, erro: "", encerrada: true });
	}, [cancelarConsulta]);

	const carregar = useCallback(() => {
		if (controlador.current) return;
		const c = new AbortController();
		controlador.current = c;
		definirEstado((anterior) => (anterior.erro ? { ...anterior, carregando: true, erro: "" } : anterior));
		consultarSessao(c.signal)
			.then((sessao) => {
				if (c.signal.aborted) return;
				if (sessao.autenticado && Date.parse(sessao.expira_em) <= Date.now()) {
					encerrarLocalmente();
					return;
				}
				definirEstado((anterior) => {
					if (anterior.encerrada) return anterior;
					return { sessao, erro: "", encerrada: anterior.sessao?.autenticado && !sessao.autenticado };
				});
			})
			.catch((e) => {
				if (c.signal.aborted) return;
				if (e instanceof ErroHTTP && e.status === 401) {
					encerrarLocalmente();
					return;
				}
				definirEstado((anterior) => {
					if (anterior.encerrada) return anterior;
					return {
						...anterior,
						carregando: false,
						erro: e instanceof ErroHTTP ? e.message : "Não foi possível consultar a sessão.",
					};
				});
			})
			.finally(() => {
				if (controlador.current === c) controlador.current = null;
			});
	}, [encerrarLocalmente]);
	const revalidarSessao = useEffectEvent(() => {
		if (!estado.encerrada && (pathname !== "/entrar" || estado.sessao?.autenticado)) carregar();
	});
	useEffect(() => {
		carregar();
		const canal = typeof BroadcastChannel === "function" ? new BroadcastChannel("aprova:sessao") : null;
		if (canal)
			canal.onmessage = (evento) => {
				if (evento.data === "sair") encerrarLocalmente();
			};
		const aoRecuperarFoco = () => revalidarSessao();
		const aoRestaurarPagina = (evento: PageTransitionEvent) => {
			if (evento.persisted) revalidarSessao();
		};
		window.addEventListener("focus", aoRecuperarFoco);
		window.addEventListener("pageshow", aoRestaurarPagina);
		window.addEventListener("aprova:sessao-expirada", encerrarLocalmente);
		return () => {
			cancelarConsulta();
			canal?.close();
			window.removeEventListener("focus", aoRecuperarFoco);
			window.removeEventListener("pageshow", aoRestaurarPagina);
			window.removeEventListener("aprova:sessao-expirada", encerrarLocalmente);
		};
	}, [cancelarConsulta, carregar, encerrarLocalmente]);
	useEffect(() => {
		if (!estado.sessao?.autenticado) return;
		const timer = setTimeout(encerrarLocalmente, Math.max(0, Date.parse(estado.sessao.expira_em) - Date.now()));
		return () => {
			clearTimeout(timer);
		};
	}, [estado.sessao, encerrarLocalmente]);

	const { erro } = estado;
	const sessao = estado.carregando ?? erro ? undefined : estado.sessao;
	const escopoDaSessao = sessao?.autenticado
		? JSON.stringify([sessao.usuario.id, sessao.csrf_token, sessao.expira_em, sessao.instalacoes.map((i) => i.id).sort((a, b) => a - b)])
		: "";
	const cacheDeLeituras = useMemo(() => (escopoDaSessao ? new CacheDeLeituras() : undefined), [escopoDaSessao]);
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
		encerrarLocalmente();
		if (typeof BroadcastChannel === "function") {
			const canal = new BroadcastChannel("aprova:sessao");
			canal.postMessage("sair");
			canal.close();
		}
	}
	if (estado.encerrada && pathname !== "/entrar") return <Navigate to="/entrar" replace />;
	return <Contexto value={{ sessao, cacheDeLeituras, erro, carregar, sair }}>{children}</Contexto>;
}
export function useSessao() {
	const v = useContext(Contexto);
	if (!v) throw new Error("Provedor de sessão ausente");
	return v;
}
