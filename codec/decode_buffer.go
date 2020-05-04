package codec

import (
	"encoding/binary"
	"math"
)

type decode_buffer struct {
	pop_buff
	order binary.ByteOrder
}

func NewDecodeBuffer(order binary.ByteOrder) *decode_buffer {

	result := &decode_buffer{order: order}
	result.InitStack()

	return result
}

func (this *decode_buffer) Init(b []byte) {
	this.data = b
	this.pos = 0
}

func (this *decode_buffer) GotoPos(pos int) {
	this.pos = pos
}

func (this *decode_buffer) ReadByte() (n byte, err error) {

	n = this.data[this.pos]
	this.pos++

	return
}

func (this *decode_buffer) Read(p []byte) (n int, err error) {

	n = copy(p, this.data[this.pos:])
	this.pos += n

	return
}

func (this *decode_buffer) Next(i int) {
	this.pos += i
}

func (this *decode_buffer) ReadUint8(dest *uint8) error {

	*dest = this.data[this.pos]
	this.pos++

	return nil
}

func (this *decode_buffer) ReadUint16(dest *uint16) (err error) {
	*dest = this.order.Uint16(this.data[this.pos:])
	this.pos += 2

	// todo check bounds
	return nil
}

func (this *decode_buffer) ReadInt32(data *int32) error {
	*data = int32(this.order.Uint32(this.data[this.pos:]))
	this.pos += 4

	return nil
}

func (this *decode_buffer) ReadFloat32(data *float32) error {
	*data = math.Float32frombits(this.order.Uint32(this.data[this.pos:]))
	this.pos += 4

	return nil
}

func (this *decode_buffer) ReadFloat64(data *float64) error {
	*data = math.Float64frombits(this.order.Uint64(this.data[this.pos:]))
	this.pos += 8

	return nil
}
