package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dot5enko/transbin/utils"
	"math"
	"reflect"
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

		if this.buffer.pos >= dLen {
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

func (this *references) GetId() uint64 {

	cur := this.count

	this.count++
	return cur
}
func (this *references) Put(data []byte) error {

	if this.count == this.cap {
		return errors.New("You reached limit of references. addressation overflow")
	}

	length := len(data)
	if length > 65535 {
		return errors.New("Length overflow")
	}

	this.buff.PutUint16(uint16(length))

	actualLen, _ := this.buff.Write(data)

	if actualLen != length {
		return errors.New("Unable to write whole data")
	}

	return nil
}

func (this *references) Reset() {
	this.buff.Reset()
	this.count = 1
	this.Reader.refsCount = 0
}

func (c *codec) writeArrayLikeData(v reflect.Value, cb func(n int, v reflect.Value, buffer *encode_buffer) error) (err error) {
	sliceLength := v.Len()

	t, err := c.getType(v.Type().Elem())
	if err != nil {
		return err
	}

	sizeOfElement, err := c.getTypeSize(t)
	if err != nil {
		return err
	}
	allocate := (sliceLength * sizeOfElement)

	// ref size
	c.ref.buff.PutUint16(uint16(allocate))
	curPos := c.ref.buff.pos

	// keeping allocated bytes for writing
	c.ref.buff.Next(allocate)

	c.mainBuffer.PushState(c.ref.buff.data[curPos:], 0)
	if sliceLength > 0 {
		err = cb(sliceLength, v, c.mainBuffer)
	} else {

	}
	c.mainBuffer.PopState()

	return err
}

func (c *codec) putReference(t uint16, v reflect.Value) (reference uint16, err error) {

	reference = uint16(c.ref.GetId())

	if isArrayType(t) {
		at := getArrayElementType(t)
		c.useType(at)
		err = c.writeArrayLikeData(v, func(n int, v0 reflect.Value, buffer *encode_buffer) error {
			var fakeField codecStructField
			fakeField.Type = at

			for i := 0; i < n; i++ {
				err = c.writeFieldData(buffer, fakeField, v0.Index(i))
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return
		}
	} else {

		switch v.Kind() {
		case reflect.String:
			err = c.ref.Put([]byte(v.String()))
		case reflect.Interface:
			var tCode uint16
			tCode,err = c.getType(v.Elem().Type())
			if (err != nil) {
				return
			}

			allocate,_ := c.getTypeSize(tCode)

			refSize := uint16(allocate + 2)
			c.ref.buff.PutUint16(refSize)

			// put type of referenced object
			c.ref.buff.PutUint16(tCode)
			oldPos := c.ref.buff.pos
			c.ref.buff.Next(allocate)

			c.mainBuffer.PushState(c.ref.buff.data[oldPos:],0)
			// put referenced object
			_, err = c.encodeToBuffer(c.mainBuffer, v.Elem())
			c.mainBuffer.PopState()

		case reflect.Map:
			err = c.writeArrayLikeData(v, func(n int, v0 reflect.Value, buffer *encode_buffer) error {
				typeOfMap, err := c.getType(v.Type().Elem())
				if err != nil {
					return err
				}

				// put value type
				buffer.PutUint16(typeOfMap)

				iter := v0.MapRange()

				for iter.Next() {
					name := iter.Key().String()

					// name field lenth and bytes
					buffer.WriteByte(uint8(len(name)))
					buffer.Write([]byte(name))

					_, err := c.encodeToBuffer(buffer, iter.Value())
					if err != nil {
						return err
					}
				}

				return nil

			})
		default:
			return 0, errors.New(fmt.Sprintf("Dont know how to reference such data type %s", v.Kind().String()))
		}
	}

	return

}
