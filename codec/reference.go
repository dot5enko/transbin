package codec

import (
	"encoding/binary"
	"errors"
	"github.com/dot5enko/transbin/utils"
	"math"
)

type references_reader struct {
	buffer     *decode_buffer
	offsets    map[uint64]uint64
	dataLength uint16
	refsCount  uint64
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

	return this.buffer.data[posStart : posStart+int(length)], length, nil
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

		if this.buffer.pos == dLen {
			break
		}
	}

	this.buffer.GotoPos(0)
}

type references struct {
	buff   *encode_buffer
	Reader references_reader

	count uint64
	cap   uint64
	order binary.ByteOrder
}

func NewReferencesHandler(addressWidth int, order binary.ByteOrder) (*references, error) {
	result := &references{}

	if addressWidth%8 != 0 {
		return nil, errors.New("Adress width should be a multiply of 8")
	}

	result.order = order
	result.buff = NewEncodeBuffer(512, order)
	result.cap = uint64(math.Pow(2, float64(addressWidth)) - 1)

	// reader init
	result.Reader.buffer = NewDecodeBuffer(order)
	result.Reader.offsets = make(map[uint64]uint64)

	result.Reset()

	return result, nil
}

func (this *references) Put(data []byte) (uint64, error) {

	curId := this.count

	if this.count == this.cap {
		return 0, errors.New("You reached limit of references. addressation overflow")
	}

	length := len(data)
	if length > 65535 {
		return 0, errors.New("Length overflow")
	}

	this.buff.PutUint16(uint16(length))

	actualLen, _ := this.buff.Write(data)

	if actualLen != length {
		return 0, errors.New("Unable to write whole data")
	}

	this.count++

	return curId, nil
}

func (this *references) Reset() {
	this.buff.Reset()
	this.count = 1
	this.Reader.refsCount = 0
}
