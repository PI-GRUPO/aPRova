import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/500.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";
import "./estilo.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import favicon from "../../marca/png/icone-app-fundo-escuro-32.png";
import { App } from "./App";

const icone = document.createElement("link");
icone.rel = "icon";
icone.type = "image/png";
icone.href = favicon;
document.head.append(icone);

const raiz = document.getElementById("raiz");
if (raiz) {
	createRoot(raiz).render(
		<StrictMode>
			<App />
		</StrictMode>,
	);
}
