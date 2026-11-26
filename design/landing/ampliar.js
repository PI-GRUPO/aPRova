const ampliacao = document.querySelector(".ampliacao");
const imagemAmpliada = new Image();
ampliacao.append(imagemAmpliada);

function abrirAmpliacao(evento) {
  evento.preventDefault();
  const miniatura = evento.currentTarget.querySelector("img");
  imagemAmpliada.src = evento.currentTarget.href;
  imagemAmpliada.alt = miniatura.alt;
  imagemAmpliada.width = Number(miniatura.getAttribute("width"));
  imagemAmpliada.height = Number(miniatura.getAttribute("height"));
  ampliacao.showModal();
}

document.querySelectorAll(".ampliar").forEach((link) => link.addEventListener("click", abrirAmpliacao));
ampliacao.addEventListener("click", () => ampliacao.close());
