import { useTema } from "@/app/tema";

const arquivos = import.meta.glob("../ativos/*.svg", { eager: true, query: "?url", import: "default" }) as Record<string, string>;
export type NomeDoAtivo =
	| "activity"
	| "list"
	| "receipt"
	| "folder"
	| "scale"
	| "history"
	| "settings"
	| "lightbulb"
	| "link"
	| "clock"
	| "marca"
	| "voltar"
	| "tema"
	| "fio"
	| "vertical"
	| "etiqueta"
	| "fechar"
	| "concluido"
	| "falhou"
	| "risco-baixo"
	| "risco-medio"
	| "risco-alto"
	| "risco-nao_avaliado"
	| "github"
	| "ajuda"
	| "serrilhado";
export function Ativo({ nome, className = "", ativo = false }: { nome: NomeDoAtivo; className?: string; ativo?: boolean }) {
	const { tema } = useTema();
	const arquivo = nome;
	const url = arquivos[`../ativos/${tema}-${arquivo}.svg`];
	if (ativo) return <span aria-hidden="true" className="icone-ativo" style={{ maskImage: `url("${url}")` }} />;
	return <img className={className} src={arquivos[`../ativos/${tema}-${arquivo}.svg`]} alt={nome === "marca" ? "aPRova" : ""} />;
}
