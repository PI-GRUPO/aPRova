const INCLINACAO_MAXIMA_GRAUS = 6;
const podeAnimar = matchMedia("(hover: hover) and (pointer: fine) and (prefers-reduced-motion: no-preference)");

function seguirCursor(evento) {
  const cartao = evento.currentTarget;
  const caixa = cartao.getBoundingClientRect();
  const x = (evento.clientX - caixa.left) / caixa.width;
  const y = (evento.clientY - caixa.top) / caixa.height;
  cartao.style.setProperty("--giro-x", `${(0.5 - y) * 2 * INCLINACAO_MAXIMA_GRAUS}deg`);
  cartao.style.setProperty("--giro-y", `${(x - 0.5) * 2 * INCLINACAO_MAXIMA_GRAUS}deg`);
}

function voltarAoLugar(evento) {
  evento.currentTarget.style.setProperty("--giro-x", "0deg");
  evento.currentTarget.style.setProperty("--giro-y", "0deg");
}

function iluminarCursor(evento) {
  const cartao = evento.currentTarget;
  const caixa = cartao.getBoundingClientRect();
  cartao.style.setProperty("--luz-x", `${evento.clientX - caixa.left}px`);
  cartao.style.setProperty("--luz-y", `${evento.clientY - caixa.top}px`);
}

if (podeAnimar.matches) {
  for (const cartao of document.querySelectorAll(".visual-passo .cartao-vivo")) {
    cartao.addEventListener("pointermove", iluminarCursor);
  }

  for (const cartao of document.querySelectorAll("#como-funciona .fluxo .cartao-vivo")) {
    cartao.addEventListener("pointermove", seguirCursor);
    cartao.addEventListener("pointerleave", voltarAoLugar);
  }
}
