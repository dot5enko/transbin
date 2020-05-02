package codec

import "sort"

type DynamicArray struct {
	pos  int
	size int
	data []int
}

func NewDArray(size int) *DynamicArray {
	result := &DynamicArray{}

	result.data = make([]int, size)
	result.size = size

	return result
}

func (this *DynamicArray) Clear() {
	this.pos = 0
}

func (this *DynamicArray) Push(val uint16) {
	if !this.Contains(val) {
		this.data[this.pos] = int(val)
		this.pos++
	}
}

func (this *DynamicArray) Contains(val uint16) bool {

	if this.pos == 0 {
		return false
	}

	for i := 0; i < this.pos; i++ {
		if this.data[i] == int(val) {
			return true
		}
	}

	return false
}

func (this *DynamicArray) Length() int {
	return this.pos
}

func (this *DynamicArray) Sort() {
	sort.Ints(this.data[:this.pos])
}
