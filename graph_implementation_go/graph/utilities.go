package graph

import (
	"hash/crc32"
	"hash/crc64"
	"log"
	"time"
)

var crcTable *crc64.Table = crc64.MakeTable(crc64.ISO)

func hashStringId(stringID string) uint64 {
	hashed := crc64.Checksum([]byte(stringID), crcTable)
	return hashed
}

func hashPackageName(packageName string) uint32 {
	hashed := crc32.ChecksumIEEE([]byte(packageName))
	return hashed
}

func LookupVersions(packageName string, versionMap map[uint32][]string) []string {
	hash := hashPackageName(packageName)
	return versionMap[hash]
}

func LookupByStringId(stringId string, hashTable map[uint64]int64) int64 {
	hash := hashStringId(stringId)
	goId := hashTable[hash]
	return goId
}

func findNode(hashMap map[uint64]int64, idToNodeInfo map[int64]NodeInfo, stringId string) (int64, bool) {
	var nodeId int64
	var correctOk bool
	if info, ok := idToNodeInfo[LookupByStringId(stringId, hashMap)]; ok {
		nodeId = info.id
		correctOk = true
	} else {
		log.Printf("String id %s was not found \n", stringId)
		correctOk = false
	}
	return nodeId, correctOk
}

func CreateHashedVersionMap(pi *[]PackageInfo) map[uint32][]string {
	result := make(map[uint32][]string, len(*pi))
	for _, pkg := range *pi {
		hashedName := hashPackageName(pkg.Name)
		result[hashedName] = make([]string, 0, len(pkg.Versions))
		for ver := range pkg.Versions {
			result[hashedName] = append(result[hashedName], ver)
		}
	}
	return result
}

// InInterval returns true when time t lies in the interval [begin, end], false otherwise
func InInterval(t, begin, end time.Time) bool {
	return t.Equal(begin) || t.Equal(end) || t.After(begin) && t.Before(end)
}
