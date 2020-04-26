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
	types      map[uint16]structDefinition
	order      binary.ByteOrder
	typeMap    map[string]uint16

	// state
	mainBuffer      *encode_buffer
	ref             *references
	usedTypes       *DynamicArray
	encodeBuffer    *bytes.Buffer
	structureBuffer *encode_buffer
	decodeBuffer 	*decode_buffer
}


func NewCodec() (*codec, error) {
	result := &codec{}

	// in order to not interfer with internal types

	result.typesCount = 27
	result.types = make(map[uint16]structDefinition)
	result.order = binary.BigEndian
	result.typeMap = make(map[string]uint16)

	// state
	result.mainBuffer = NewEncodeBuffer(512,result.order)
	result.usedTypes = NewDArray(10)
	result.encodeBuffer = new(bytes.Buffer)
	result.structureBuffer = NewEncodeBuffer(256,result.order)
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
