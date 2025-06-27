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
		lengths := make([]int, len(lines))
		for i, l := range lines {
			lengths[i] = len(l)
		}
		for i := 0; i < len(lengths)-1; i++ {
			lengths[i]++ // account for '\n'
		}
		this.linesLength = lengths
	}
	line := 1
	for _, l := range this.linesLength {
		if offset < l {
			return line
		}
		offset -= l
		line++
	}
	return line // out of bounds
}
