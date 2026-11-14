package frete

func Calcular(pesoKg float64) float64 {
	if pesoKg <= 1 {
		return 10
	}
	return 10 + (pesoKg-1)*2.5
}
