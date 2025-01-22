package models

import "crypto/sha1"

func HashKey(key string, numTasks int) int {
	// Use SHA-1 to hash the key
	hash := sha1.New()
	hash.Write([]byte(key))
	hashBytes := hash.Sum(nil)

	// Convert the hash to an integer
	hashInt := int(hashBytes[0])<<24 | int(hashBytes[1])<<16 | int(hashBytes[2])<<8 | int(hashBytes[3])

	// Take the modulus by the number of tasks to get a consistent result
	return hashInt % numTasks
}
