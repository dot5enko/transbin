package codec

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dot5enko/transbin/utils"
	"log"
	"reflect"
)

const defaultBufferSize int = 512
const internalTypesCount uint16 = 26

type codec struct {
	typesCount uint16
	types      map[uint16]*structDefinition
	order      binary.ByteOrder
	typeMap    map[string]uint16

	// state
	mainBuffer      *encode_buffer
	ref             *references
	usedTypes       *DynamicArray
	encodeBuffer    *bytes.Buffer
	structureBuffer *encode_buffer

	decode_buffers *list.List
	decodeBuffer   *decode_buffer

	buffers *list.List

	dataBuffer struct {
		byte       uint8
		int32val   int32
		float32val float32
		float64val float64
		uint16val  uint16
		nameReader [255]byte
	}
}

func NewCodec() (*codec, error) {
	result := &codec{}

	// in order to not interfer with internal types

	result.typesCount = 27
	result.types = make(map[uint16]*structDefinition)
	result.order = binary.BigEndian
	result.typeMap = make(map[string]uint16)

	// state

	// main buffer and pushed one
	result.buffers = list.New()
	result.mainBuffer = NewEncodeBuffer(512, result.order)
	result.encodeBuffer = new(bytes.Buffer)

	//
	result.decode_buffers = list.New()
	result.decodeBuffer = NewDecodeBuffer(result.order)

	//
	result.usedTypes = NewDArray(10)
	result.structureBuffer = NewEncodeBuffer(256, result.order)

	var err error
	result.ref, err = NewReferencesHandler(16, result.order)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *codec) useType(t uint16) {
	c.usedTypes.Push(t)
}

func (c *codec) reset() {
	c.usedTypes.Clear()
	c.mainBuffer.Reset()
	c.ref.Reset()
	c.encodeBuffer.Reset()
	c.structureBuffer.Reset()
}

// todo reuse
func (c *codec) PushMainBuffer() {

	newBuffer := NewEncodeBuffer(defaultBufferSize, c.order)

	c.buffers.PushBack(c.mainBuffer)
	c.mainBuffer = newBuffer
}

func (c *codec) PopMainBuffer() []byte {
	result := c.mainBuffer

	prevBuffer := c.buffers.Back()
	if prevBuffer == nil {
		log.Printf("No pushed buffer were found\n")
	}

	c.mainBuffer = prevBuffer.Value.(*encode_buffer)

	return result.Bytes()
}

func (c *codec) PushDecodeBuffer(data []byte) {

	newBuffer := NewDecodeBuffer(c.order)
	newBuffer.Init(data)

	c.decode_buffers.PushBack(c.decodeBuffer)
	c.decodeBuffer = newBuffer
}

func (c *codec) PopDecodeBuffer() {
	prevBuffer := c.decode_buffers.Back()
	if prevBuffer == nil {
		log.Printf("No pushed buffer were found\n")
	}

	c.decodeBuffer = prevBuffer.Value.(*decode_buffer)
}

func (c *codec) putReference(t uint16, v reflect.Value) (reference uint16, err error) {

	var refData []byte

	if isArrayType(t) {
		at :=getArrayElementType(t)
		c.useType(at)
		refData, err = c.writeSliceFieldData(at, v)
	} else {
		switch v.Kind() {
		case reflect.String:
			refData = []byte(v.String())
		default:
			return 0, errors.New(fmt.Sprintf("Dont know how to reference such data type %s", v.Kind().String()))
		}
	}

	var refSmall uint64
	refSmall, err = c.ref.Put(refData)

	reference = uint16(refSmall)

	return

}

func (c *codec) cacheReflectionData(typeId uint16, t reflect.Type) error {
	tData, ok := c.types[typeId]
	if !ok {
		return errors.New("unable to found a type declaration in codec")
	}

	if tData.Offsets > 0 {
		return nil
	}

	for i := 0; i < int(tData.FieldCount); i++ {
		f := &tData.Fields[i]

		refField, found := t.FieldByName(f.Name)

		if !found {
			return errors.New(fmt.Sprintf("field %s not found", f.Name))
		}

		f.Offset = refField.Offset
	}

	tData.Offsets = 1
	return nil
}

func (c *codec) getTypeSize(t uint16) (int, error) {

	if isArrayType(t) {
		return 2, nil
	}

	if t > internalTypesCount {

		tref, ok := c.types[t]

		if !ok {
			return 0, utils.Error("Unable to found a size for type %d", t)
		}

		return tref.Size, nil
	} else {
		switch reflect.Kind(t) {
		case reflect.Bool, reflect.Uint8:
			return 1, nil
		case reflect.String, reflect.Slice: // reference types
			return 2, nil
		case reflect.Uint16:
			return 2, nil
		case reflect.Float32, reflect.Int32, reflect.Uint, reflect.Int, reflect.Uint32, reflect.Uintptr:
			return 4, nil
		case reflect.Float64, reflect.Int64, reflect.Uint64:
			return 8, nil
		default:
			return 0, fmt.Errorf("Unable to get a type length for type %s: %d", reflect.Kind(t), t)
		}
	}
}

func (c *codec) readArrayElement(elementType uint16, out reflect.Value) error {

	c.decodeBuffer.ReadUint16(&c.dataBuffer.uint16val)
	arrayData, length, err := c.ref.Reader.Get(uint64(c.dataBuffer.uint16val))
	if err != nil {
		return err
	}

	typeSize, err := c.getTypeSize(elementType)
	if err != nil {
		return err
	}

	var items int = int(length) / typeSize

	arrayResult := reflect.MakeSlice(out.Type(), items, items)

	c.PushDecodeBuffer(arrayData)

	fakeField := codecStructField{}
	fakeField.Type = elementType

	for i := 0; i < items; i++ {
		c.readFieldData(fakeField, arrayResult.Index(i))
	}
	c.PopDecodeBuffer()

	out.Set(arrayResult)

	return nil
}
