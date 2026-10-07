package dataset

import (
	"crypto/sha1"
	"fmt"
)

// namespaceURL is the RFC 9562 namespace for URL names.
var namespaceURL = [16]byte{0x6b, 0xa7, 0xb8, 0x11, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

var projectNamespace = uuidV5(namespaceURL, "https://github.com/yeremi777/nihongo-foundation")

// rowID is the deterministic id of the row with the given natural-key name.
func rowID(name string) string {
	u := uuidV5(projectNamespace, name)
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

func uuidV5(namespace [16]byte, name string) [16]byte {
	h := sha1.New()
	h.Write(namespace[:])
	h.Write([]byte(name))
	var u [16]byte
	copy(u[:], h.Sum(nil))
	u[6] = u[6]&0x0f | 0x50
	u[8] = u[8]&0x3f | 0x80
	return u
}
