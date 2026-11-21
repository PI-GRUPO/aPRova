const INCLINACAO_MAXIMA_GRAUS = 6;
const podeAnimar = matchMedia("(hover: hover) and (pointer: fine) and (prefers-reduced-motion: no-preference)");

function seguirCursor(evento) {
  const cartao = evento.currentTarget;
  const caixa = cartao.getBoundingClientRect();
  const x = (evento.clientX - caixa.left) / caixa.width;
  const y = (evento.clientY - caixa.top) / caixa.height;
  cartao.style.setProperty("--luz-x", `${x * 100}%`);
  cartao.style.setProperty("--luz-y", `${y * 100}%`);
  cartao.style.setProperty("--giro-x", `${(0.5 - y) * 2 * INCLINACAO_MAXIMA_GRAUS}deg`);
  cartao.style.setProperty("--giro-y", `${(x - 0.5) * 2 * INCLINACAO_MAXIMA_GRAUS}deg`);
}

function voltarAoLugar(evento) {
  evento.currentTarget.style.setProperty("--giro-x", "0deg");
  evento.currentTarget.style.setProperty("--giro-y", "0deg");
}

if (podeAnimar.matches) {
  for (const cartao of document.querySelectorAll(".cartao-vivo")) {
    cartao.addEventListener("pointermove", seguirCursor);
    cartao.addEventListener("pointerleave", voltarAoLugar);
  }
}
