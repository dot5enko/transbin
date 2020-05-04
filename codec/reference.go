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
		return nil, utils.Error("Adress width should be a multiply of 8")
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
		return utils.Error("Unable to write whole data. expected % bytes. written %d",length,actualLen)
	}

	return nil
}

func (this *references) Reset() {
	this.buff.Reset()
	this.count = 1
	this.Reader.refsCount = 0
	this.Reader.Reset()
}

func (c *codec) writeArrayLikeData(v reflect.Value, parent_buf *encode_buffer, cb func(n int, v reflect.Value, b *encode_buffer) error) (sliceLength int, err error) {
	sliceLength = v.Len()

	var t uint16
	t, err = c.getType(v.Type().Elem())
	if err != nil {
		return
	}

	var sizeOfElement int

	sizeOfElement, _ = c.getTypeSize(t)

	if v.Kind() == reflect.Map {
		keyType, err := c.getType(v.Type().Key())
		if err != nil {
			return 0, err
		}
		sizeOfKey, _ := c.getTypeSize(keyType)
		sizeOfElement += sizeOfKey

	}

	allocate := (sliceLength * sizeOfElement)

	// ref size
	c.ref.buff.PutUint16(uint16(allocate))

	// keeping allocated bytes for writing
	arrayArea := parent_buf.Branch(allocate)

	if sliceLength > 0 {
		err = cb(sliceLength, v, arrayArea)
	}

	return
}

func (c *codec) putReference(buffer *encode_buffer, t uint16, v reflect.Value) (reference uint16, err error) {

	reference = uint16(c.ref.GetId())

	if isArrayType(t) {
		at := getArrayElementType(t)
		c.useType(at)
		_, err = c.writeArrayLikeData(v, buffer, func(n int, v0 reflect.Value, b *encode_buffer) error {
			var fakeField codecStructField
			fakeField.Type = at

			for i := 0; i < n; i++ {
				err = c.writeFieldData(b, fakeField, v0.Index(i))
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
			fmt.Printf("current ref pos : %d out of %d\n",c.ref.buff.pos,c.ref.buff.size)
			err = c.ref.Put([]byte(v.String()))
		case reflect.Interface:
			// [type of ref data;2b;][ref id; 2b]

			interfaceActualData := v.Elem()

			var tCode uint16
			tCode, err = c.getType(interfaceActualData.Type())
			if err != nil {
				return
			}

			// put type of referenced object
			buffer.PutUint16(tCode)

			// allocated size in references for actual data
			allocate, _ := c.getTypeSize(tCode)
			c.ref.buff.PutUint16(uint16(allocate))

			// allocate data in ref
			interfaceDataBuffer := c.ref.buff.Branch(allocate)

			// put referenced object
			_, err = c.encodeElementToBuffer(interfaceDataBuffer, interfaceActualData)
			// use buffer's data

		case reflect.Map:

			// [element type;2b][key type;2b][reference id; 2b] ... [name len;N;1b;][name bytes;Nb][fieldData;Xb]

			typeOfMap, err := c.getType(v.Type().Elem())
			typeOfMapKey, err := c.getType(v.Type().Key())

			if err != nil {
				return 0, err
			}

			// type of element
			buffer.PutUint16(typeOfMap)

			// type of key
			buffer.PutUint16(typeOfMapKey)

			_, err = c.writeArrayLikeData(v, buffer, func(n int, v0 reflect.Value, b *encode_buffer) error {

				iter := v0.MapRange()

				for iter.Next() {
					_, err = c.encodeElementToBuffer(b, iter.Key())
					if err != nil {
						return err
					}
					_, err = c.encodeElementToBuffer(b, iter.Value())
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
