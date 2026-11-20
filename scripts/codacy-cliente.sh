#!/usr/bin/env bash
set -euo pipefail

API_DO_CODACY="https://api.codacy.com/2.0"

uso() {
	echo "uso: $0 <subcomando> [argumentos]" >&2
	echo "  baixar <nome> <versao> <sha256>  baixa o conversor codacy/<nome> e confere o SHA-256" >&2
	echo "  converter <nome> <entrada> <saida>  converte a saída da ferramenta para o formato do Codacy" >&2
	echo "  enviar <arquivo>                  envia resultados convertidos para o commit" >&2
	echo "  finalizar                         avisa o Codacy de que os resultados do commit terminaram" >&2
	echo "variáveis: CODACY_PROJECT_TOKEN, COMMIT, PASTA_CODACY" >&2
	exit 2
}

exigir_variavel() {
	if [ -z "${!1:-}" ]; then
		echo "$1 não definida" >&2
		exit 2
	fi
}

baixar() {
	local nome="$1" versao="$2" sha256="$3"
	local destino="${PASTA_CODACY}/${nome}"
	mkdir -p "$PASTA_CODACY"
	curl -sSfL -o "$destino" "https://github.com/codacy/${nome}/releases/download/${versao}/${nome}-${versao}"
	echo "${sha256}  ${destino}" | sha256sum --check --strict
	chmod +x "$destino"
}

converter() {
	local nome="$1" entrada="$2" saida="$3"
	jq --ascii-output --compact-output . "$entrada" | "${PASTA_CODACY}/${nome}" >"$saida"
}

postar() {
	local rota="$1"
	shift
	curl -sSfL -X POST \
		-H "project-token: ${CODACY_PROJECT_TOKEN}" \
		-H "Content-type: application/json" \
		"$@" \
		"${API_DO_CODACY}/commit/${COMMIT}/${rota}"
	echo
}

enviar() {
	postar issuesRemoteResults --data-binary "@$1"
}

finalizar() {
	postar resultsFinal
}

[ $# -ge 1 ] || uso
subcomando="$1"
shift

case "$subcomando" in
baixar)
	[ $# -eq 3 ] || uso
	exigir_variavel PASTA_CODACY
	baixar "$@"
	;;
converter)
	[ $# -eq 3 ] || uso
	exigir_variavel PASTA_CODACY
	converter "$@"
	;;
enviar)
	[ $# -eq 1 ] || uso
	exigir_variavel CODACY_PROJECT_TOKEN
	exigir_variavel COMMIT
	enviar "$1"
	;;
finalizar)
	[ $# -eq 0 ] || uso
	exigir_variavel CODACY_PROJECT_TOKEN
	exigir_variavel COMMIT
	finalizar
	;;
*)
	uso
	;;
esac
