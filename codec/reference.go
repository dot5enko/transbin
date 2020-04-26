package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

type references struct {
	buff  *bytes.Buffer
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
	result.buff = new(bytes.Buffer)
	result.cap = uint64(math.Pow(2, float64(addressWidth)) - 1)

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

	binary.Write(this.buff, this.order, uint16(length))
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
}
