import "@fontsource-variable/inter";
import "@fontsource/ibm-plex-mono/400.css";
import { Provider as ProvedorDeBaloes } from "@radix-ui/react-tooltip";
import "./estilos/tokens.css";
import "./estilos/aplicacao.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app/App";
import { FronteiraDeErro } from "./app/FronteiraDeErro";

const raiz = document.getElementById("raiz");
if (raiz)
	createRoot(raiz).render(
		<StrictMode>
			<FronteiraDeErro>
				<ProvedorDeBaloes delayDuration={180}>
					<App />
				</ProvedorDeBaloes>
			</FronteiraDeErro>
		</StrictMode>,
	);
