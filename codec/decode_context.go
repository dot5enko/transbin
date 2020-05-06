package codec

import (
	"errors"
	"fmt"
	"github.com/dot5enko/transbin/utils"
	"reflect"
	"runtime"
)

type decode_context struct {
	global *codec

	buffer     *decode_buffer
	references references_reader

	dataBuffer struct {
		byte       uint8
		int32val   int32
		float32val float32
		float64val float64
		uint16val  uint16
		nameReader [255]byte
	}
}

func NewDecodeContext(global *codec) *decode_context {

	result := &decode_context{}
	result.references = new_references_reader(global.order)
	result.buffer = global.get_free_dbuffer()
	result.global = global

	return result
}

func (ctx *decode_context) Reset() {

}

func (ctx *decode_context) readArrayElement(buffer *decode_buffer, elementType uint16, out reflect.Value) error {

	buffer.ReadUint16(&ctx.dataBuffer.uint16val)
	arrayData, length, err := ctx.references.Get(uint64(ctx.dataBuffer.uint16val))
	if err != nil {
		return err
	}

	typeSize, err := ctx.global.getTypeSize(elementType)
	if err != nil {
		return err
	}

	var items int = int(length) / typeSize

	// check if out is not a slice already
	arrayResult := reflect.MakeSlice(out.Type(), items, items)

	fakeField := codecStructField{}
	fakeField.Type = elementType

	curBuf := buffer.InitBranch(arrayData)

	for i := 0; i < items; i++ {
		ctx.readFieldData(&curBuf, fakeField, arrayResult.Index(i))
	}

	out.Set(arrayResult)

	return nil
}

func (c *decode_context) readMapField(buffer *decode_buffer, interfaceElemType uint16, keyType uint16, refBytes []byte, out reflect.Value) error {

	dataLen := len(refBytes)

	newMap := reflect.MakeMap(out.Type())

	// type of interface element
	elemSize, err := c.global.getTypeSize(interfaceElemType)
	if err != nil {
		return err
	}

	keySize, err := c.global.getTypeSize(keyType)
	elemSize += keySize

	elems := dataLen / elemSize

	values := reflect.MakeSlice(reflect.SliceOf(out.Type().Elem()), elems, elems)
	keys := reflect.MakeSlice(reflect.SliceOf(out.Type().Key()), elems, elems)

	fakeKeyField := codecStructField{}
	fakeKeyField.Type = keyType

	subBuffer := buffer.InitBranch(refBytes)

	for i := 0; i < elems; i++ {
		// read key
		fakeKeyField.Type = keyType
		err := c.readFieldData(&subBuffer, fakeKeyField, keys.Index(i))

		if err != nil {
			return err
		}

		// read value
		fakeKeyField.Type = interfaceElemType
		err = c.readFieldData(&subBuffer, fakeKeyField, values.Index(i))
		if err != nil {
			return err
		}

		newMap.SetMapIndex(keys.Index(i), values.Index(i))
	}

	out.Set(newMap)

	return err
}

func (c *decode_context) readStructFieldData() (sf codecStructField, err error) {

	err = c.buffer.ReadUint16(&sf.Type)
	if err != nil {
		return
	}

	sf.NameLength, err = c.buffer.ReadByte()
	if err != nil {
		return
	}

	readed, _ := c.buffer.Read(c.dataBuffer.nameReader[:sf.NameLength])
	if readed != int(sf.NameLength) {
		return sf, errors.New("Read wrong amount of data when reading structure's field name")
	}

	sf.Name = string(c.dataBuffer.nameReader[:sf.NameLength])

	// calc size in struct
	sf.Size, err = c.global.getTypeSize(sf.Type)

	return

}

// returns struct header size
func (c *decode_context) tryDecodeStructure() (int, error) {

	headerSize := 0

	// read number of types
	nTypes, err := c.buffer.ReadByte()
	if err != nil {
		return headerSize, err
	}

	headerSize += 1

	for i := 0; i < int(nTypes); i++ {

		var typeDef structDefinition
		err = c.buffer.ReadUint16(&typeDef.Id)

		if err != nil {
			return headerSize, err
		}
		headerSize += 2

		typeDef.FieldCount, err = c.buffer.ReadByte()
		if err != nil {
			return headerSize, err
		}
		headerSize += 1

		skip := false
		if _, ok := c.global.types[typeDef.Id]; ok {
			// skip current structure, we already have it

			for j := 0; j < int(typeDef.FieldCount); j++ {
				c.buffer.Next(2)

				nameLength, _ := c.buffer.ReadByte()
				// name and type
				headerSize += 3
				headerSize += int(nameLength)

				c.buffer.Next(int(nameLength))
			}

			skip = true
		}

		if !skip {

			typeDef.Fields = make([]codecStructField, typeDef.FieldCount)

			for j := 0; j < int(typeDef.FieldCount); j++ {

				typeDef.Fields[j], err = c.readStructFieldData()
				if err != nil {
					return headerSize, err
				}

				// type and nameLength
				headerSize += 3
				headerSize += int(typeDef.Fields[j].NameLength)

				typeDef.Size += typeDef.Fields[j].Size
			}

			c.global.types[typeDef.Id] = &typeDef
		}
	}

	return headerSize, nil

}

func (c *decode_context) Decode(out interface{}, input []byte) error {

	c.buffer.Init(input)

	headerSize, err := c.tryDecodeStructure()

	if err != nil {
		return err
	}

	var typeOfElement uint16
	c.buffer.ReadUint16(&typeOfElement)

	structSize, err := c.global.getTypeSize(typeOfElement)
	if err != nil {
		return err
	}

	refsOffset := structSize + headerSize + 2 // 2 bytes for struct type

	c.references.Init(input[refsOffset:])

	// todo check if type is same in out interface and binary data given

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Not addressable value (%s) provided as out parameter. it should be a pointer to structure", v.Type().Kind().String()))
	}

	indirect := reflect.Indirect(v)

	fakeField := codecStructField{}
	fakeField.Type = typeOfElement

	return c.readFieldData(c.buffer, fakeField, indirect)
	//c.cacheReflectionData(typeOfElement, indirect.Type())

	//return c.readComplexFieldData(typeOfElement, indirect)
}

func (c *decode_context) readFieldData(buffer *decode_buffer, field codecStructField, out reflect.Value) error {



	if isArrayType(field.Type) {
		return c.readArrayElement(buffer, getArrayElementType(field.Type), out)
	} else {
		if field.Type > internalTypesCount {
			return c.readComplexFieldData(buffer, field.Type, out)
		} else {
			switch reflect.Kind(field.Type) {
			case reflect.String, reflect.Map, reflect.Interface:
				return c.readReferenceFieldData(buffer, field.Type, out)
			default:
				return c.readSimpleFieldData(buffer, field.Type, out)
			}

		}
	}

}

func (c *decode_context) readSimpleFieldData(buffer *decode_buffer, t uint16, out reflect.Value) error {

	tmpVal := reflect.Indirect(out)

	switch reflect.Kind(t) {
	case reflect.Int, reflect.Int32:


		err := buffer.ReadInt32(&c.dataBuffer.int32val)

		if err != nil {
			return err
		}

		if tmpVal.CanSet() {
			val := int64(c.dataBuffer.int32val)

			if tmpVal.Kind() == reflect.Interface {
				tmpVal.Set(reflect.ValueOf(val))
			} else {


				tmpVal.SetInt(val)

			}
		} else {
			return utils.Error("Cant set value on field of type %d\n", t)
		}


	case reflect.Float64:
		buffer.ReadFloat64(&c.dataBuffer.float64val)
		if tmpVal.CanSet() {

			val := float64(c.dataBuffer.float64val)

			if tmpVal.Kind() == reflect.Interface {
				tmpVal.Set(reflect.ValueOf(val))
			} else {
				tmpVal.SetFloat(val)
			}
		} else {
			return errors.New("unable to set float32 value of unaccessable field")
		}
	case reflect.Float32:

		buffer.ReadFloat32(&c.dataBuffer.float32val)
		if tmpVal.CanSet() {

			if tmpVal.Kind() == reflect.Interface {
				tmpVal.Set(reflect.ValueOf(float64(c.dataBuffer.float32val)))
			} else {
				tmpVal.SetFloat(float64(c.dataBuffer.float32val))
			}
		} else {
			return errors.New("unable to set float32 value of unaccessable field")
		}
	default:
		return errors.New(fmt.Sprintf("Unable to decode type %", t))
	}

	return nil

}

func (c *decode_context) readReferenceFieldData(buffer *decode_buffer, t uint16, out reflect.Value) error {



	switch reflect.Kind(t) {
	case reflect.String:
		buffer.ReadUint16(&c.dataBuffer.uint16val)

		refBytes, _, err := c.references.Get(uint64(c.dataBuffer.uint16val))
		if err != nil {
			return err
		}

		if out.Kind() == reflect.Interface {
			out.Set(reflect.ValueOf(string(refBytes)))
		} else {

			runtime.ReadMemStats(&curms)
			out.SetString(string(refBytes))

			oldMallocs := curms.Mallocs
			runtime.ReadMemStats(&curms)
			if oldMallocs != curms.Mallocs {
				refsC += (curms.Mallocs - oldMallocs)
				fmt.Printf(" -- put simple field %d mallocs +%d when adding type %s\n", refsC, curms.Mallocs-oldMallocs, out.Type().String())
			}

		}
	case reflect.Map:

		var elType uint16
		var keyType uint16

		buffer.ReadUint16(&elType)
		buffer.ReadUint16(&keyType)

		buffer.ReadUint16(&c.dataBuffer.uint16val)

		refBytes, _, err := c.references.Get(uint64(c.dataBuffer.uint16val))
		if err != nil {
			return err
		}
		err = c.readMapField(buffer, elType, keyType, refBytes, out)
		if err != nil {
			return err
		}
	case reflect.Interface:

		var interfaceType uint16
		buffer.ReadUint16(&interfaceType)
		buffer.ReadUint16(&c.dataBuffer.uint16val)

		refBytes, _, err := c.references.Get(uint64(c.dataBuffer.uint16val))
		if err != nil {
			return err
		}
		fakeField := codecStructField{}
		fakeField.Type = interfaceType

		refBuffer := buffer.InitBranch(refBytes)

		c.readFieldData(&refBuffer, fakeField, out)
	default:
		return utils.Error("Unable to decode referenced type: %s\n", reflect.Kind(t).String())
	}

	return nil
}

func (c *decode_context) readComplexFieldData(buffer *decode_buffer, t uint16, out reflect.Value) (err error) {

	refValue := reflect.Indirect(out)

	if refValue.Kind() != reflect.Struct {
		return utils.Error("cannot decode data to %s", refValue.Kind().String())
	}

	tData, ok := c.global.types[t]
	if !ok {
		return utils.Error("No structure present in data. dont know how to decode %d type", t)
	}

	for i := 0; i < int(tData.FieldCount); i++ {

		f := tData.Fields[i]

		fieldObj := refValue.Field(i)
		if fieldObj.Kind() == reflect.Ptr {
			if fieldObj.IsNil() {
				fieldObj.Set(reflect.New(fieldObj.Type().Elem()))
			}
		}

		err = c.readFieldData(buffer, f, fieldObj)

		if err != nil {
			return
		}
	}

	return nil
}
