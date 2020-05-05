package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dot5enko/transbin/utils"
	"math"
	"reflect"
	"runtime"
	"unsafe"
)

type references_writer struct {
	buff encode_buffer

	count uint64
	cap   uint64
	order binary.ByteOrder
}

func NewReferencesWriter(addressWidth int, order binary.ByteOrder) (*references_writer, error) {
	result := &references_writer{}

	if addressWidth%8 != 0 {
		return nil, utils.Error("Adress width should be a multiply of 8")
	}

	result.order = order
	result.buff = NewEncodeBuffer(512, order)
	result.cap = uint64(math.Pow(2, float64(addressWidth)) - 1)

	result.Reset()

	return result, nil
}

func (this *references_writer) GetId() uint64 {

	cur := this.count

	this.count++
	return cur
}
func (this *references_writer) Put(data []byte) error {

	if this.count == this.cap {
		return errors.New("You reached limit of references_writer. addressation overflow")
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

func (this *references_writer) Reset() {
	this.buff.Reset()
	this.count = 1
}

func (c *encode_context) writeArrayLikeData(v reflect.Value, parent_buf encode_buffer, cb func(n int, v reflect.Value, b encode_buffer) error) (sliceLength int, err error) {
	sliceLength = v.Len()

	var t uint16
	t, err = c.global.getType(v.Type().Elem())
	if err != nil {
		return
	}

	var sizeOfElement int

	sizeOfElement, _ = c.global.getTypeSize(t)

	if v.Kind() == reflect.Map {
		keyType, err := c.global.getType(v.Type().Key())
		if err != nil {
			return 0, err
		}
		sizeOfKey, _ := c.global.getTypeSize(keyType)
		sizeOfElement += sizeOfKey

	}

	allocate := (sliceLength * sizeOfElement)

	// ref size
	c.ref.buff.PutUint16(uint16(allocate))

	// keeping allocated bytes for writing
	if sliceLength > 0 {
		err = cb(sliceLength, v, c.ref.buff.Branch(allocate))
	} else {

	}

	return
}

var curms runtime.MemStats
var refsC uint64 = 0

func (c *encode_context) putReference(buffer encode_buffer, t uint16, v reflect.Value) (reference uint16, err error) {

	reference = uint16(c.ref.GetId())

	if isArrayType(t) {
		at := getArrayElementType(t)
		c.useType(at)
		_, err = c.writeArrayLikeData(v, buffer, func(n int, v0 reflect.Value, b encode_buffer) error {
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

			var vStr []byte

			if v.CanAddr() {
				vStr = *(*[]byte)(unsafe.Pointer(v.UnsafeAddr()))
			} else {
				vStr = []byte(v.String())
			}

			err = c.ref.Put([]byte(vStr))

		case reflect.Interface:
			// [type of ref data;2b;][ref id; 2b]

			interfaceActualData := v.Elem()

			var tCode uint16
			tCode, err = c.global.getType(interfaceActualData.Type())
			if err != nil {
				return
			}

			// put type of referenced object
			buffer.PutUint16(tCode)

			// allocated size in references_writer for actual data
			allocate, _ := c.global.getTypeSize(tCode)
			c.ref.buff.PutUint16(uint16(allocate))

			// use allocated data
			// put referenced object
			_, err = c.encodeElementToBuffer(c.ref.buff.Branch(allocate), interfaceActualData)
			// use buffer's data
		case reflect.Map:

			// [element type;2b][key type;2b][reference id; 2b] ... [name len;N;1b;][name bytes;Nb][fieldData;Xb]

			typeOfMap, err := c.global.getType(v.Type().Elem())
			typeOfMapKey, err := c.global.getType(v.Type().Key())

			if err != nil {
				return 0, err
			}

			// type of element
			buffer.PutUint16(typeOfMap)

			// type of key
			buffer.PutUint16(typeOfMapKey)

			_, err = c.writeArrayLikeData(v, buffer, func(n int, v0 reflect.Value, b encode_buffer) error {

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
