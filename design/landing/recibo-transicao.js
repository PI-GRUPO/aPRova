const origemRecibo = document.querySelector("[data-recibo-origem]");
const destinoRecibo = document.querySelector("#recibo .recibo-cartao");
const movimentoReduzidoRecibo = matchMedia("(prefers-reduced-motion: reduce)");
const telaDesktopRecibo = matchMedia("(min-width: 1100px)");

const imagemRecibo = origemRecibo.cloneNode(true);
imagemRecibo.removeAttribute("data-recibo-origem");
imagemRecibo.classList.add("recibo-transicao");
imagemRecibo.setAttribute("aria-hidden", "true");
imagemRecibo.inert = true;
imagemRecibo.hidden = true;
document.body.append(imagemRecibo);

let quadroRecibo = 0;

function atualizarTransicaoRecibo() {
  quadroRecibo = 0;
  const origem = origemRecibo.getBoundingClientRect();
  const destino = destinoRecibo.getBoundingClientRect();
  const inicio = origem.top + scrollY + origem.height / 2 - innerHeight / 2;
  const fim = destino.top + scrollY + destino.height / 2 - innerHeight / 2;
  const progresso = Math.max(0, Math.min(1, (scrollY - inicio) / Math.max(1, fim - inicio)));
  const emFoco = origemRecibo.contains(document.activeElement) || destinoRecibo.contains(document.activeElement);
  const animar = telaDesktopRecibo.matches && !movimentoReduzidoRecibo.matches && !emFoco && fim > inicio;
  const emTransicao = animar && scrollY > inicio + 1 && scrollY < fim - 1;

  imagemRecibo.hidden = !emTransicao;
  origemRecibo.style.opacity = emTransicao ? "0" : "";
  origemRecibo.style.pointerEvents = emTransicao ? "none" : "";

  if (!emTransicao) {
    destinoRecibo.style.opacity = "";
    return;
  }

  // A mesma rolagem controla o percurso e a troca da imagem pelo recibo.
  const percurso = progresso * progresso * (3 - 2 * progresso);
  const mistura = Math.max(0, Math.min(1, (progresso - 0.5) / 0.5));
  const revelacao = mistura * mistura * (3 - 2 * mistura);
  const largura = origem.width + (destino.width - origem.width) * percurso;
  const altura = origem.height + (destino.height - origem.height) * percurso;
  const esquerda = origem.left + (destino.left - origem.left) * percurso;
  const topo = origem.top + (destino.top - origem.top) * percurso;

  imagemRecibo.style.width = `${largura}px`;
  imagemRecibo.style.height = `${altura}px`;
  imagemRecibo.style.transform = `translate3d(${esquerda}px, ${topo}px, 0)`;
  imagemRecibo.style.opacity = String(1 - revelacao);
  destinoRecibo.style.opacity = String(revelacao);
}

function agendarTransicaoRecibo() {
  if (!quadroRecibo) quadroRecibo = requestAnimationFrame(atualizarTransicaoRecibo);
}

addEventListener("scroll", agendarTransicaoRecibo, { passive: true });
addEventListener("resize", agendarTransicaoRecibo);
addEventListener("load", agendarTransicaoRecibo);
document.addEventListener("focusin", agendarTransicaoRecibo);
document.addEventListener("focusout", agendarTransicaoRecibo);
movimentoReduzidoRecibo.addEventListener("change", agendarTransicaoRecibo);
telaDesktopRecibo.addEventListener("change", agendarTransicaoRecibo);

const tamanhoRecibo = new ResizeObserver(agendarTransicaoRecibo);
tamanhoRecibo.observe(document.querySelector("main"));
agendarTransicaoRecibo();
