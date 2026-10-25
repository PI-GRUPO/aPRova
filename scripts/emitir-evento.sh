#!/usr/bin/env bash
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PASTA_EXEMPLOS="${RAIZ}/workflows/exemplos"
ARQUIVO_ENV="${RAIZ}/.env"
TEMPO_LIMITE_SEGUNDOS=15
URL_PUBLICA_PROVISORIA="https://aprova.example.com"

uso() {
	echo "uso: $0 <tipo> <url> [opções]" >&2
	echo "  tipo: um dos exemplos em workflows/exemplos: $(tipos_conhecidos)" >&2
	echo "  url: endereço do webhook que recebe o evento, ex.: http://localhost:5978/webhook/aprova" >&2
	echo "opções:" >&2
	echo "  --repositorio dono/nome   sobrescreve dados.repositorio" >&2
	echo "  --numero-pr N             sobrescreve dados.numero_pr" >&2
	echo "  --nivel-risco NIVEL       sobrescreve dados.nivel_risco (baixo, medio ou alto)" >&2
	echo "  --dado campo=valor        sobrescreve qualquer campo de dados; valor JSON vale como JSON, o resto como texto" >&2
	echo "  --imprimir                mostra corpo e cabeçalhos sem enviar" >&2
	echo "ambiente:" >&2
	echo "  N8N_SEGREDO_EVENTOS       segredo da assinatura; se vazio, vem do .env" >&2
	echo "  APROVA_URL_PUBLICA        base da url_recibo; se vazia, vem do .env ou fica ${URL_PUBLICA_PROVISORIA}" >&2
	exit 2
}

abortar() {
	echo "$1" >&2
	exit 1
}

tipos_conhecidos() {
	local arquivo nomes=()
	for arquivo in "${PASTA_EXEMPLOS}"/*.json; do
		[ -e "$arquivo" ] && nomes+=("$(basename "$arquivo" .json)")
	done
	echo "${nomes[*]:-nenhum}"
}

exigir_dependencias() {
	local ferramenta faltando=()
	for ferramenta in curl jq openssl; do
		command -v "$ferramenta" >/dev/null 2>&1 || faltando+=("$ferramenta")
	done
	[ "${#faltando[@]}" -eq 0 ] || abortar "dependências ausentes: ${faltando[*]}. Instale e rode de novo."
}

ler_configuracao() {
	local nome="$1"
	if [ -n "${!nome:-}" ]; then
		printf '%s' "${!nome}"
		return 0
	fi
	[ -f "$ARQUIVO_ENV" ] || return 0
	local linha valor
	linha="$(grep -E "^[[:space:]]*${nome}=" "$ARQUIVO_ENV" | tail -n 1 || true)"
	valor="${linha#*=}"
	valor="${valor%$'\r'}"
	valor="${valor#\"}"
	valor="${valor%\"}"
	valor="${valor#\'}"
	valor="${valor%\'}"
	printf '%s' "$valor"
}

novo_uuid() {
	local aleatorio variante
	aleatorio="$(openssl rand -hex 16)"
	variante="$(printf '%x' $(((16#${aleatorio:16:1} & 3) | 8)))"
	printf '%s-%s-4%s-%s%s-%s' "${aleatorio:0:8}" "${aleatorio:8:4}" "${aleatorio:13:3}" \
		"$variante" "${aleatorio:17:3}" "${aleatorio:20:12}"
}

hmac_sha256() {
	local segredo="$1" conteudo="$2"
	printf '%s' "$conteudo" | openssl dgst -sha256 -hmac "$segredo" -r | awk '{print $1}'
}

[ "$#" -ge 2 ] || uso
TIPO="$1"
URL="$2"
shift 2

IMPRIMIR=false
RECOMPOR_URL_PR=false
SOBRESCRITAS='{}'

exigir_dependencias

exigir_nome_de_campo() {
	[[ "$1" =~ ^[a-z_]+$ ]] || abortar "nome de campo inválido: $1"
}

sobrescrever_com_texto() {
	exigir_nome_de_campo "$1"
	SOBRESCRITAS="$(jq -c --arg campo "$1" --arg valor "$2" '.[$campo] = $valor' <<<"$SOBRESCRITAS")"
}

sobrescrever_com_json() {
	exigir_nome_de_campo "$1"
	SOBRESCRITAS="$(jq -c --arg campo "$1" --argjson valor "$2" '.[$campo] = $valor' <<<"$SOBRESCRITAS")"
}

sobrescrever() {
	if jq -n --argjson valor "$2" 'true' >/dev/null 2>&1; then
		sobrescrever_com_json "$1" "$2"
	else
		sobrescrever_com_texto "$1" "$2"
	fi
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--imprimir)
		IMPRIMIR=true
		shift
		;;
	--repositorio)
		[ "$#" -ge 2 ] || uso
		sobrescrever_com_texto repositorio "$2"
		RECOMPOR_URL_PR=true
		shift 2
		;;
	--numero-pr)
		[ "$#" -ge 2 ] || uso
		[[ "$2" =~ ^[1-9][0-9]*$ ]] || abortar "--numero-pr precisa ser um inteiro positivo: $2"
		sobrescrever_com_json numero_pr "$2"
		RECOMPOR_URL_PR=true
		shift 2
		;;
	--nivel-risco)
		[ "$#" -ge 2 ] || uso
		sobrescrever_com_texto nivel_risco "$2"
		shift 2
		;;
	--dado)
		[ "$#" -ge 2 ] && [[ "$2" == *=* ]] || uso
		sobrescrever "${2%%=*}" "${2#*=}"
		shift 2
		;;
	*)
		uso
		;;
	esac
done

[[ "$TIPO" =~ ^[a-z_]+$ ]] && [ -f "${PASTA_EXEMPLOS}/${TIPO}.json" ] ||
	abortar "tipo de evento desconhecido: ${TIPO}. Tipos conhecidos: $(tipos_conhecidos)"

CAMPOS_AUSENTES="$(jq -r --argjson sobrescritas "$SOBRESCRITAS" \
	'.dados as $dados | [$sobrescritas | keys[] | select(. as $campo | $dados | has($campo) | not)] | join(", ")' \
	"${PASTA_EXEMPLOS}/${TIPO}.json")"
[ -z "$CAMPOS_AUSENTES" ] || abortar "o tipo ${TIPO} não tem o(s) campo(s) ${CAMPOS_AUSENTES} em dados"

SEGREDO="$(ler_configuracao N8N_SEGREDO_EVENTOS)"
[ -n "$SEGREDO" ] || abortar "N8N_SEGREDO_EVENTOS não definido no ambiente nem em ${ARQUIVO_ENV}"

URL_PUBLICA="$(ler_configuracao APROVA_URL_PUBLICA)"
URL_PUBLICA="${URL_PUBLICA:-$URL_PUBLICA_PROVISORIA}"
[[ "$URL_PUBLICA" =~ ^https?://[^/[:space:]]+(/[^[:space:]]*)?$ ]] ||
	abortar "APROVA_URL_PUBLICA precisa começar com http:// ou https://: ${URL_PUBLICA}"

TIMESTAMP="$(date +%s)"
CORPO="$(jq -c \
	--arg id_evento "$(novo_uuid)" \
	--arg ocorrido_em "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
	--argjson sobrescritas "$SOBRESCRITAS" \
	--argjson recompor_url_pr "$RECOMPOR_URL_PR" \
	--arg url_publica "${URL_PUBLICA%/}" \
	'.id_evento = $id_evento
	| .ocorrido_em = $ocorrido_em
	| .dados += $sobrescritas
	| if $recompor_url_pr and (.dados | has("url_pr")) and ($sobrescritas | has("url_pr") | not) then
		.dados.url_pr = "https://github.com/\(.dados.repositorio)/pull/\(.dados.numero_pr // (.dados.url_pr | split("/") | last))"
	  else . end
	| if (.dados | has("url_recibo")) and ($sobrescritas | has("url_recibo") | not) then
		.dados.url_recibo = "\($url_publica)/recibos/\(.dados.url_recibo | split("/recibos/") | last)"
	  else . end' \
	"${PASTA_EXEMPLOS}/${TIPO}.json")"
ASSINATURA="sha256=$(hmac_sha256 "$SEGREDO" "${TIMESTAMP}.${CORPO}")"

if [ "$IMPRIMIR" = true ]; then
	printf 'POST %s\n' "$URL"
	printf 'Content-Type: application/json\n'
	printf 'X-Aprova-Timestamp: %s\n' "$TIMESTAMP"
	printf 'X-Aprova-Assinatura: %s\n' "$ASSINATURA"
	printf '\n%s\n' "$CORPO"
	exit 0
fi

ARQUIVO_CORPO="$(mktemp)"
ARQUIVO_RESPOSTA="$(mktemp)"
trap 'rm -f "$ARQUIVO_CORPO" "$ARQUIVO_RESPOSTA"' EXIT
printf '%s' "$CORPO" >"$ARQUIVO_CORPO"

if ! CODIGO="$(curl -sS --max-time "$TEMPO_LIMITE_SEGUNDOS" -o "$ARQUIVO_RESPOSTA" -w '%{http_code}' \
	-X POST \
	-H 'Content-Type: application/json' \
	-H "X-Aprova-Timestamp: ${TIMESTAMP}" \
	-H "X-Aprova-Assinatura: ${ASSINATURA}" \
	--data-binary "@${ARQUIVO_CORPO}" \
	"$URL")"; then
	abortar "não foi possível enviar ${TIPO} para ${URL}"
fi

echo "${TIPO} enviado para ${URL}: HTTP ${CODIGO}"
head -c 2000 "$ARQUIVO_RESPOSTA"
echo
[[ "$CODIGO" =~ ^2[0-9][0-9]$ ]] || abortar "a resposta não foi 2xx"
