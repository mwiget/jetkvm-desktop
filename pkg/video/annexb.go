package video

import "bytes"

// splitAnnexB splits an Annex B byte stream into NAL units without start codes.
func splitAnnexB(data []byte) [][]byte {
	var nals [][]byte
	start := -1
	for i := 0; i+2 < len(data); {
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 {
			if start >= 0 {
				nals = appendNAL(nals, data[start:i])
			}
			i += 3
			start = i
			continue
		}
		i++
	}
	if start >= 0 {
		nals = appendNAL(nals, data[start:])
	}
	return nals
}

// appendNAL drops trailing zero bytes, which belong to a following four-byte
// start code (a NAL unit never ends in a zero byte).
func appendNAL(nals [][]byte, nal []byte) [][]byte {
	nal = bytes.TrimRight(nal, "\x00")
	if len(nal) == 0 {
		return nals
	}
	return append(nals, nal)
}

const (
	nalTypeSlice = 1
	nalTypeIDR   = 5
	nalTypeSPS   = 7
	nalTypePPS   = 8
)

func nalType(nal []byte) byte {
	return nal[0] & 0x1f
}
