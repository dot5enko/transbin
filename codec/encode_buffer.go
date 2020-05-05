package codec

import (
	"encoding/binary"
	"fmt"
	"math"
)

type encode_buffer struct {
	pop_buff
	order binary.ByteOrder
}

func NewEncodeBuffer(size int, order binary.ByteOrder) *encode_buffer {

	result := &encode_buffer{}
	result.buff_allocator = &buff_allocator{}

	result.data = make([]byte, size)
	result.pos = 0
	result.size = size
	result.order = order

	return result
}

func (this *encode_buffer) Reset() {
	this.pos = 0
}

func (this *encode_buffer) ReadByte() (n byte, err error) {

	n = this.allocator.data[this.pos]
	this.pos++

	return
}

func (this *encode_buffer) grow(atLeast int) {

	newSize := this.allocator.size * 2
	if atLeast > newSize {
		newSize += atLeast
	}

	newBuf := make([]byte, newSize)

	copy(newBuf, this.allocator.data[:this.pos])
	this.allocator.data = newBuf
	this.allocator.size = newSize
}

func (this *encode_buffer) tryGrow(n int) {

	//curPos := this.pos

	//fmt.Printf("trying to grow. now %d at %d -> %d\n",this.allocator.size,curPos,this.pos + n)
	if (this.pos + n) >= this.allocator.size {
		this.grow(n)
	}
}

func (this *encode_buffer) Write(p []byte) (n int, err error) {

	oldl := len(p)
	this.tryGrow(oldl)

	n = copy(this.allocator.data[this.pos:], p)

	if oldl != n {
		panic("holly shit!")
	}

	if n > 100 {
		fmt.Printf(" written holy shit of %d bytes \n", n)
	}

	this.pos += n

	return
}

func (this *encode_buffer) Next(i int) {
	this.tryGrow(i)
	this.pos += i
}

func (this *encode_buffer) Bytes() []byte {
	return this.allocator.data[:this.pos]
}

func (this *encode_buffer) PutInt32(v int32) {
	this.tryGrow(4)
	this.order.PutUint32(this.allocator.data[this.pos:], uint32(v))
	this.pos += 4
}

func (this *encode_buffer) PutFloat32(v float32) {
	this.tryGrow(4)
	this.order.PutUint32(this.allocator.data[this.pos:], math.Float32bits(v))
	this.pos += 4
}

func (this *encode_buffer) PutUint16(v uint16) {
	this.tryGrow(2)
	this.order.PutUint16(this.allocator.data[this.pos:], v)
	this.pos += 2
}

func (this *encode_buffer) WriteByte(u uint8) {
	this.tryGrow(1)
	this.allocator.data[this.pos] = u
	this.pos++
}

func (this *encode_buffer) PutFloat64(f float64) {
	this.tryGrow(8)
	this.order.PutUint64(this.allocator.data[this.pos:], math.Float64bits(f))
	this.pos += 8
}
