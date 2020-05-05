package codec

import (
	"encoding/binary"
	"github.com/dot5enko/transbin/utils"
)

type references_reader struct {
	buffer     *decode_buffer
	offsets    map[uint64]uint64
	dataLength uint16
	refsCount  uint64
}

func new_references_reader(order binary.ByteOrder) references_reader {
	result := references_reader{}

	result.buffer = NewDecodeBuffer(order)
	result.offsets = make(map[uint64]uint64)

	return result
}

func (this *references_reader) Get(id uint64) ([]byte, uint16, error) {

	if this.refsCount == 0 || id > this.refsCount {
		return nil, 0, utils.Error("No such reference. refCount = %d", this.refsCount)
	}

	pos := int(this.offsets[id])
	var length uint16

	this.buffer.GotoPos(pos)
	this.buffer.ReadUint16(&length)

	posStart := pos + 2

	return this.buffer.allocator.data[posStart : posStart+int(length)], length, nil
}

func (this *references_reader) Init(data []byte) {

	dLen := len(data)

	this.buffer.Init(data)
	this.buffer.GotoPos(0)

	for {

		this.refsCount++

		this.offsets[this.refsCount] = uint64(this.buffer.pos)
		this.buffer.ReadUint16(&this.dataLength)

		this.buffer.Next(int(this.dataLength))

		if this.buffer.pos >= dLen {
			break
		}
	}

	this.buffer.GotoPos(0)
}

func (this *references_reader) Reset() {
	this.buffer.Reset()
	this.refsCount = 0
	this.dataLength = 0
}
