package codec

type DynamicArray struct {
	pos  int
	size int
	data []uint16
}

func NewDArray(size int) *DynamicArray {
	result := &DynamicArray{}

	result.data = make([]uint16, size)
	result.size = size

	return result
}

func (this *DynamicArray) Clear() {
	this.pos = 0
}

func (this *DynamicArray) Push(val uint16) {
	if !this.Contains(val) {
		this.data[this.pos] = val
		this.pos++
	}
}

func (this *DynamicArray) Contains(val uint16) bool {

	if this.pos == 0 {
		return false
	}

	for i := 0; i < this.pos; i++ {
		if this.data[i] == val {
			return true
		}
	}

	return false
}

func (this *DynamicArray) Length() int {
	return this.pos
}
