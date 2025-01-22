package config

import (
	"log"

	"github.com/spaolacci/murmur3"
)

func GetServerIdInRing(serverAddr string) int {

	serverRingId := GetMbitsOfHash(serverAddr)
	log.Println("Server in Ring is", serverRingId)
	return serverRingId

}

func GetFileHashId(fileName string) int {

	fileRingId := GetMbitsOfHash(fileName)
	//log.Println("File Id ", fileRingId)
	return fileRingId

}

func GetMbitsOfHash(data string) int {

	hash32 := murmur3.Sum32([]byte(data))
	first11Bits := hash32 & 0x7FF

	// hasher := sha256.New()
	// hasher.Write([]byte(data))

	// hashSum := hasher.Sum(nil)

	// hashString := fmt.Sprintf("%x", hashSum)

	// log.Printf("\nHash value of string with String %s is %d", data, hashString)

	// firstByte := hashSum[0]
	// secondByte := hashSum[1]

	// // Combine these bits into an integer
	// // Shift the first byte by 3 to make room for 3 bits from the second byte
	// truncatedBits := (uint16(firstByte) << 3) | (uint16(secondByte) >> 5)

	// // Print the result as a binary string and as an integer
	// log.Printf("First 11 bits (integer): %d\n", truncatedBits)

	return int(first11Bits)
}
