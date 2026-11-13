import { type ClassValue, clsx } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

const mesclar = extendTailwindMerge({
	extend: {
		classGroups: {
			"font-size": [{ text: ["12", "14", "16", "20", "24", "32"] }],
			rounded: [{ rounded: ["selo", "controle", "painel"] }],
		},
	},
});

export function cn(...entradas: ClassValue[]) {
	return mesclar(clsx(entradas));
}
