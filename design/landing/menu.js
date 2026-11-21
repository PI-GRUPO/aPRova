const raiz = document.documentElement;
const topo = document.querySelector(".topo");
const botaoTema = document.querySelector(".trocar-tema");
const sistemaClaro = matchMedia("(prefers-color-scheme: light)");
const telaLarga = matchMedia("(min-width: 1024px)");
const ROTULO_TEMA = { claro: "Mudar para o tema escuro", escuro: "Mudar para o tema claro" };

function temaAtual() {
  return raiz.dataset.tema ?? (sistemaClaro.matches ? "claro" : "escuro");
}

function atualizarRotuloTema() {
  botaoTema.setAttribute("aria-label", ROTULO_TEMA[temaAtual()]);
}

function guardarTema(tema) {
  try {
    localStorage.setItem("aprova-tema", tema);
  } catch {
    return;
  }
}

function trocarTema() {
  const novo = temaAtual() === "claro" ? "escuro" : "claro";
  raiz.dataset.tema = novo;
  guardarTema(novo);
  atualizarRotuloTema();
}

let ultimaRolagem = scrollY;

function esconderTopoAoDescer() {
  const descendo = scrollY > ultimaRolagem && scrollY > topo.offsetHeight;
  topo.classList.toggle("topo-escondido", !telaLarga.matches && descendo);
  ultimaRolagem = scrollY;
}

function marcarSecaoAtual(entradas) {
  for (const entrada of entradas) {
    if (!entrada.isIntersecting) continue;
    for (const link of document.querySelectorAll('.topo nav a[href^="#"]')) {
      if (link.hash === `#${entrada.target.id}`) link.setAttribute("aria-current", "true");
      else link.removeAttribute("aria-current");
    }
  }
}

const observador = new IntersectionObserver(marcarSecaoAtual, { rootMargin: "-45% 0px -50% 0px" });
document.querySelectorAll("main section").forEach((secao) => observador.observe(secao));

botaoTema.addEventListener("click", trocarTema);
sistemaClaro.addEventListener("change", atualizarRotuloTema);
addEventListener("scroll", esconderTopoAoDescer, { passive: true });
atualizarRotuloTema();
