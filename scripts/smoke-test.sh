#!/usr/bin/env bash
# Confere, logo depois de um deploy (ou com a API local), se as rotas do aPRova
# respondem. Sai com código diferente de zero se qualquer conferência falhar.
#
# uso: scripts/smoke-test.sh <url_base>
#   SMOKE_RECIBO_ID      id de um recibo para conferir /receipts/{id}/verify
#   N8N_SEGREDO_ACOES    segredo do n8n, para as rotas assinadas
#   SMOKE_INSTALLATION_ID installation_id das rotas assinadas
#                        (padrão: APROVA_CHAT_INSTALLATION_ID)
set -Eeuo pipefail

uso() {
	echo "uso: $0 <url_base>   ex.: $0 http://localhost:8080" >&2
	exit 2
}

[ "$#" -eq 1 ] || uso
BASE="${1%/}"
[[ "$BASE" =~ ^https?:// ]] || uso

RECIBO="${SMOKE_RECIBO_ID:-}"
SEGREDO="${N8N_SEGREDO_ACOES:-}"
INSTALACAO="${SMOKE_INSTALLATION_ID:-${APROVA_CHAT_INSTALLATION_ID:-}}"
TEMPO_LIMITE=10
FALHAS=0
CORPO="$(mktemp)"
trap 'rm -f "$CORPO"' EXIT

ok() { echo "  ✔ $1"; }
falhou() {
	echo "  ✘ $1" >&2
	FALHAS=$((FALHAS + 1))
}

# pedir <caminho> [cabeçalho...]: grava o corpo em $CORPO e imprime o status.
pedir() {
	local caminho="$1"
	shift
	local cabecalhos=()
	for cabecalho in "$@"; do
		cabecalhos+=(-H "$cabecalho")
	done
	curl -sS -o "$CORPO" -w '%{http_code}' --max-time "$TEMPO_LIMITE" "${cabecalhos[@]}" "$BASE$caminho" || echo "000"
}

conferir() {
	local descricao="$1" esperado_status="$2" esperado_texto="$3" status="$4"
	if [ "$status" != "$esperado_status" ]; then
		falhou "$descricao: esperava $esperado_status, veio $status"
	elif [ -n "$esperado_texto" ] && ! grep -q -- "$esperado_texto" "$CORPO"; then
		falhou "$descricao: a resposta não contém $esperado_texto"
	else
		ok "$descricao"
	fi
}

# assinar <conteúdo>: imprime os cabeçalhos X-Aprova-* do mesmo jeito que o n8n.
assinar() {
	local timestamp assinatura
	timestamp="$(date +%s)"
	assinatura="$(printf '%s.%s' "$timestamp" "$1" | openssl dgst -sha256 -hmac "$SEGREDO" -r | cut -d' ' -f1)"
	printf 'X-Aprova-Timestamp: %s\nX-Aprova-Assinatura: sha256=%s\n' "$timestamp" "$assinatura"
}

echo "smoke test em $BASE"

echo "rotas públicas"
conferir "GET /health" 200 '"status"' "$(pedir /health)"
conferir "GET /ready (banco e redis)" 200 '"status"' "$(pedir /ready)"
conferir "GET /receipts/keys" 200 '"chaves"' "$(pedir /receipts/keys)"

if [ -n "$RECIBO" ]; then
	conferir "GET /receipts/$RECIBO/verify" 200 '"valido":true' "$(pedir "/receipts/$RECIBO/verify")"
else
	echo "  - /receipts/{id}/verify pulado: defina SMOKE_RECIBO_ID"
fi

echo "rotas assinadas (n8n)"
if [ -n "$SEGREDO" ] && [ -n "$INSTALACAO" ]; then
	consulta="installation_id=$INSTALACAO"
	mapfile -t cabecalhos < <(assinar "$consulta")
	conferir "GET /reports/weekly" 200 '"total_execucoes"' "$(pedir "/reports/weekly?$consulta" "${cabecalhos[@]}")"

	mapfile -t cabecalhos < <(assinar "$(printf 'GET\n/api/analises\n%s' "$consulta")")
	conferir "GET /api/analises" 200 '' "$(pedir "/api/analises?$consulta" "${cabecalhos[@]}")"

	conferir "GET /reports/weekly sem assinatura é recusado" 401 '' "$(pedir "/reports/weekly?$consulta")"
else
	echo "  - puladas: defina N8N_SEGREDO_ACOES e SMOKE_INSTALLATION_ID"
fi

if [ "$FALHAS" -gt 0 ]; then
	echo "smoke test reprovado: $FALHAS conferência(s) falharam" >&2
	exit 1
fi
echo "smoke test aprovado"
