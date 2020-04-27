package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

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
	decodeBuffer    *decode_buffer
	dataBuffer      struct {
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
	result.mainBuffer = NewEncodeBuffer(512, result.order)
	result.usedTypes = NewDArray(10)
	result.encodeBuffer = new(bytes.Buffer)
	result.structureBuffer = NewEncodeBuffer(256, result.order)
	result.decodeBuffer = NewDecodeBuffer(result.order)

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

func (c *codec) putReference(v reflect.Value) (uint16, error) {

	var refData []byte

	if v.Kind() == reflect.String {
		refData = []byte(v.String())
	} else {
		return 0, errors.New(fmt.Sprintf("Dont know how to reference such data type %s", v.Kind().String()))
	}

	id, err := c.ref.Put(refData)

	return uint16(id), err

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
