package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dot5enko/transbin/utils"
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
	mainBuffer *encode_buffer
	ref        *references
	usedTypes  *DynamicArray

	encodeBuffer    *bytes.Buffer
	structureBuffer *encode_buffer

	decodeBuffer *decode_buffer

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
	result.mainBuffer = NewEncodeBuffer(512, result.order)
	result.encodeBuffer = new(bytes.Buffer)

	//
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

func unrollPrt(p reflect.Type) reflect.Type {
	for {
		if p.Kind() == reflect.Ptr {
			p = p.Elem()
		} else {
			break
		}
	}

	return p
}
func (c *codec) getType(p reflect.Type) (uint16, error) {

	p = unrollPrt(p)

	t := uint16(p.Kind())

	switch reflect.Kind(t) {

	case reflect.Struct:
		tref, ok := c.typeMap[getTypeCode(p)]

		if !ok {
			return 0, utils.Error("Unable to found a size for type %d", t)
		}

		return tref, nil
	case reflect.Slice:
		t, err := c.getType(p.Elem())
		if err != nil {
			return 0, err
		}
		return setArrayTypeFlag(t), nil
	default:

		return t, nil

		//return 0, utils.Error("Unable to detect internal type for %s", p.Kind().String())
	}
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
		case reflect.Float32, reflect.Int32, reflect.Uint, reflect.Int, reflect.Uint32, reflect.Uintptr, reflect.Interface:
			return 4, nil
		case reflect.Float64, reflect.Int64, reflect.Uint64:
			return 8, nil
		case reflect.Map:
			return 2 /* reference to data id*/ + 2 /*element type*/ + 2 /*elements count*/, nil
		default:
			return 0, fmt.Errorf("Unable to get a type length for type %s: %d", reflect.Kind(t).String(), t)
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
	c.decodeBuffer.PushState(arrayData, 0)

	fakeField := codecStructField{}
	fakeField.Type = elementType

	for i := 0; i < items; i++ {
		c.readFieldData(fakeField, arrayResult.Index(i))
	}
	c.decodeBuffer.PopState()

	out.Set(arrayResult)

	return nil
}

func (c *codec) readMapField(interfaceElemType uint16, keyType uint16, refBytes []byte, out reflect.Value) error {

	dataLen := len(refBytes)

	c.decodeBuffer.PushState(refBytes, 0)

	newMap := reflect.MakeMap(out.Type())

	// type of interface element
	elemSize, err := c.getTypeSize(interfaceElemType)
	if err != nil {
		return err
	}

	keySize, err := c.getTypeSize(keyType)
	elemSize += keySize

	elems := dataLen / elemSize

	values := reflect.MakeSlice(reflect.SliceOf(out.Type().Elem()), elems, elems)
	keys := reflect.MakeSlice(reflect.SliceOf(out.Type().Key()), elems, elems)

	fakeKeyField := codecStructField{}
	fakeKeyField.Type = keyType

	for i := 0; i < elems; i++ {
		// read key
		fakeKeyField.Type = keyType
		err := c.readFieldData(fakeKeyField, keys.Index(i))

		if err != nil {
			return err
		}

		// read value
		fakeKeyField.Type = interfaceElemType
		err = c.readFieldData(fakeKeyField, values.Index(i))
		if err != nil {
			return err
		}

		newMap.SetMapIndex(keys.Index(i), values.Index(i))
	}

	out.Set(newMap)
	c.decodeBuffer.PopState()

	return err
}
