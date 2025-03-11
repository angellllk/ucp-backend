package model

func ErrorMsg(s string) string {
	switch s {
	case "invalid email address":
		return "Adresa de email folosita este invalida."
	case "username can only contain letters and digits":
		return "Numele contului nu poate contine spatii si caractere speciale."
	case "name cannot be empty":
		return "Numele caracterului trebuie completat."
	case "origin cannot be empty":
		return "Originea caracterului trebuie completata."
	case "origin contains wrong characters":
		return "Originea caracterului trebuie sa contina doar litere."
	case "age must be between 12 and 80 years old":
		return "Varsta caracterului trebuie sa fie intre 12 si 80 de ani."
	case "invalid character name":
		return "Numele caracterului trebuie sa fie de forma Prenume_Nume."
	case "invalid length for character origin":
		return "Originrea caracterului trebuie aiba minim 4 caractere."
	default:
		return "Parola trebuie sa aiba minim 8 caractere (litere, cifre si caractere speciale)"
	}
}
