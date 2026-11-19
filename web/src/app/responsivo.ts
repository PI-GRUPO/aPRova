import { useSyncExternalStore } from "react";

const consulta = "(max-width: 767px)";
function assinar(aoMudar: () => void) {
	const media = window.matchMedia(consulta);
	media.addEventListener("change", aoMudar);
	return () => media.removeEventListener("change", aoMudar);
}
export function useMobile() {
	return useSyncExternalStore(assinar, () => window.matchMedia(consulta).matches);
}
