const colunaFixa = document.querySelector(".caso-visuais");
const visuais = [...document.querySelectorAll(".visual-passo")];
const lugarOriginal = new Map(visuais.map((visual) => [visual, { passo: visual.parentElement, depois: visual.nextElementSibling }]));
const telaComColunaFixa = matchMedia("(min-width: 1100px)");

function mostrarVisual(numero) {
  for (const visual of visuais) visual.classList.toggle("ativo", visual.dataset.visual === numero);
}

function posicionarVisuais() {
  for (const visual of visuais) {
    const { passo, depois } = lugarOriginal.get(visual);
    if (telaComColunaFixa.matches) colunaFixa.append(visual);
    else passo.insertBefore(visual, depois);
  }
  colunaFixa.classList.toggle("em-uso", telaComColunaFixa.matches);
}

function acompanharPasso(entradas) {
  for (const entrada of entradas) {
    if (!entrada.isIntersecting) continue;
    const visual = visuais.find((v) => lugarOriginal.get(v).passo === entrada.target);
    if (visual) mostrarVisual(visual.dataset.visual);
  }
}

const observadorDePassos = new IntersectionObserver(acompanharPasso, { rootMargin: "-40% 0px -55% 0px" });
document.querySelectorAll(".passos > li").forEach((passo) => observadorDePassos.observe(passo));

posicionarVisuais();
mostrarVisual("1");
telaComColunaFixa.addEventListener("change", posicionarVisuais);
