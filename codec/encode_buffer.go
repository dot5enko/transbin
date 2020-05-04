package codec

import (
	"encoding/binary"
	"math"
)

type encode_buffer struct {
	pop_buff
	order binary.ByteOrder
}

func NewEncodeBuffer(size int, order binary.ByteOrder) *encode_buffer {

	result := &encode_buffer{}

	result.data = make([]byte, size)
	result.pos = 0
	result.size = size
	result.order = order

	result.InitStack();

	return result
}

func (this *encode_buffer) Reset() {
	this.pos = 0
}

func (this *encode_buffer) ReadByte() (n byte, err error) {

	n = this.data[this.pos]
	this.pos++

	return
}

func (this *encode_buffer) Write(p []byte) (n int, err error) {
	n = copy(this.data[this.pos:], p)
	this.pos += n

	return
}

func (this *encode_buffer) Next(i int) {
	this.pos += i
}

func (this *encode_buffer) Bytes() []byte {
	return this.data[:this.pos]
}

func (this *encode_buffer) PutInt32(v int32) {
	this.order.PutUint32(this.data[this.pos:], uint32(v))
	this.pos += 4
}

func (this *encode_buffer) PutFloat32(v float32) {
	this.order.PutUint32(this.data[this.pos:], math.Float32bits(v))
	this.pos += 4
}

func (this *encode_buffer) PutUint16(v uint16) {
	this.order.PutUint16(this.data[this.pos:], v)
	this.pos += 2
}

func (this *encode_buffer) WriteByte(u uint8) {
	this.data[this.pos] = u
	this.pos++
}

func (this *encode_buffer) PutFloat64(f float64) {
	this.order.PutUint64(this.data[this.pos:], math.Float64bits(f))
	this.pos += 8
}
