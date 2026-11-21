const DURACAO_CONTAGEM_MS = 1400;

function desacelerar(progresso) {
  return progresso === 1 ? 1 : 1 - Math.pow(2, -10 * progresso);
}

function contarAte(elemento) {
  const valorFinal = Number(elemento.textContent);
  const inicio = performance.now();

  function quadro(agora) {
    const progresso = Math.min((agora - inicio) / DURACAO_CONTAGEM_MS, 1);
    elemento.textContent = String(Math.round(valorFinal * desacelerar(progresso)));
    if (progresso < 1) requestAnimationFrame(quadro);
  }

  elemento.textContent = "0";
  requestAnimationFrame(quadro);
}

const preferePoucoMovimento = matchMedia("(prefers-reduced-motion: reduce)").matches;

if (!preferePoucoMovimento) {
  document.querySelectorAll("[data-contar]").forEach(contarAte);
}
