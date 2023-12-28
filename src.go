package parse

import "bytes"

type Src struct {
	bytes       []byte
	linesLength []int
}

func (this *Src) Line(offset int) int {
	if this == nil {
		return 0
	}
	if this.linesLength == nil {
		lines := bytes.Split(this.bytes, []byte{'\n'})
		lenghts := make([]int, len(lines))
		for i, l := range lines {
			lenghts[i] = len(l)
		}
		this.linesLength = lenghts
	}
	for i, l := range this.linesLength {
		offset -= l
		if offset < 0 {
			return i
		}
	}
	return len(this.linesLength) // out of bounds
}
