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

func (this *decode_buffer) InitStack() {
	// todo resize
	this.states = make([]BufferState, 10)
	this.statesPos = 0
}

func (this *decode_buffer) Reset() {
	this.statesPos = 0
}

func (this *decode_buffer) PushState(data []byte, at int) {

	this.states[this.statesPos] = BufferState{pos: this.pos, size: this.allocator.size, ref: this.allocator.data}

	this.allocator.data = data
	this.pos = at
	this.allocator.size = len(data)

	this.statesPos++
}

func (this *decode_buffer) PopState() {

	this.statesPos--
	prevState := this.states[this.statesPos]

	this.allocator.data = prevState.ref
	this.pos = prevState.pos
	this.allocator.size = prevState.size
}

func (this *decode_buffer) Init(b []byte) {
	this.allocator = &buff_allocator{}
	this.allocator.data = b
	this.pos = 0
}

func (this *decode_buffer) GotoPos(pos int) {
	this.pos = pos
}

func (this *decode_buffer) ReadByte() (n byte, err error) {

	n = this.allocator.data[this.pos]
	this.pos++

	return
}

func (this *decode_buffer) Read(p []byte) (n int, err error) {

	n = copy(p, this.allocator.data[this.pos:])
	this.pos += n

	return
}

func (this *decode_buffer) Next(i int) {
	this.pos += i
}

func (this *decode_buffer) ReadUint8(dest *uint8) error {

	*dest = this.allocator.data[this.pos]
	this.pos++

	return nil
}

func (this *decode_buffer) ReadUint16(dest *uint16) (err error) {
	*dest = this.order.Uint16(this.allocator.data[this.pos:])
	this.pos += 2

	// todo check bounds
	return nil
}

func (this *decode_buffer) ReadInt32(data *int32) error {
	*data = int32(this.order.Uint32(this.allocator.data[this.pos:]))
	this.pos += 4

	return nil
}

func (this *decode_buffer) ReadInt64(data *int64) error {
	*data = int64(this.order.Uint64(this.allocator.data[this.pos:]))
	this.pos += 8

	return nil
}

func (this *decode_buffer) ReadFloat32(data *float32) error {
	*data = math.Float32frombits(this.order.Uint32(this.allocator.data[this.pos:]))
	this.pos += 4

	return nil
}

func (this *decode_buffer) ReadFloat64(data *float64) error {
	*data = math.Float64frombits(this.order.Uint64(this.allocator.data[this.pos:]))
	this.pos += 8

	return nil
}
