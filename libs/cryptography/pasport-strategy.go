package cryptography

import (
	"strings"
	"time"

	"golang.org/x/exp/rand"
)

func VigenereEncrypt(plaintext string, key []int) string {
	encrypted := make([]byte, len(plaintext))

	for i, char := range plaintext {
		shift := key[i%len(key)]
		encrypted[i] = byte((int(char) + shift) % 128)
	}

	return string(encrypted)
}

func VigenereDecrypt(ciphertext string, key []int) string {
	decrypted := make([]byte, len(ciphertext))

	for i, char := range ciphertext {
		shift := key[i%len(key)]
		decrypted[i] = byte((int(char) - shift + 128) % 128)
	}

	return string(decrypted)
}

func PolyalphabetEncrypt(plaintext string, key string) string {
	var domData []rune
	var cipher strings.Builder

	lastKey := ""
	for _, letter := range key {
		if !strings.ContainsRune(lastKey, letter) {
			lastKey += string(letter)
		}
	}

	var lastAlphabetLetter []rune
	for c := 32; c <= 126; c++ {
		letter := rune(c)
		if !strings.ContainsRune(lastKey, letter) {
			lastAlphabetLetter = append(lastAlphabetLetter, letter)
		}
	}

	for _, letter := range lastKey {
		domData = append(domData, rune(letter))
	}

	for _, letter := range lastAlphabetLetter {
		domData = append(domData, letter)
	}

	for i, letter := range plaintext {
		indexPlaintext := -1
		for j, e := range domData {
			if e == letter {
				indexPlaintext = j
				break
			}
		}

		if indexPlaintext != -1 {
			shift := i % len(key)
			newIndex := (indexPlaintext + shift) % len(domData)
			cipher.WriteRune(domData[newIndex])
		} else {
			cipher.WriteRune(letter)
		}
	}

	return cipher.String()
}

func PolyalphabetDecrypt(ciphertext string, key string) string {
	var domData []rune
	var plaintext strings.Builder

	lastKey := ""
	for _, letter := range key {
		if !strings.ContainsRune(lastKey, letter) {
			lastKey += string(letter)
		}
	}

	var lastAlphabetLetter []rune
	for c := 32; c <= 126; c++ {
		letter := rune(c)
		if !strings.ContainsRune(lastKey, letter) {
			lastAlphabetLetter = append(lastAlphabetLetter, letter)
		}
	}

	for _, letter := range lastKey {
		domData = append(domData, rune(letter))
	}

	for _, letter := range lastAlphabetLetter {
		domData = append(domData, letter)
	}

	for i, letter := range ciphertext {
		indexCiphertext := -1
		for j, e := range domData {
			if e == letter {
				indexCiphertext = j
				break
			}
		}

		if indexCiphertext != -1 {
			// Menggunakan kunci untuk membalikkan perubahan
			shift := i % len(key)
			newIndex := (indexCiphertext - shift + len(domData)) % len(domData)
			plaintext.WriteRune(domData[newIndex])
		} else {
			// Jika karakter tidak ditemukan dalam domData, masukkan langsung
			plaintext.WriteRune(letter)
		}
	}

	return plaintext.String()
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
