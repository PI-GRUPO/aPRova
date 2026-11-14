package auth

func SenhaForte(senha string) bool {
	return len(senha) >= 6
}
