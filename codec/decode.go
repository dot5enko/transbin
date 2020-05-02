package codec

import (
	"errors"
	"fmt"
	"github.com/dot5enko/transbin/utils"
	"reflect"
)

func (c *codec) readStructFieldData() (sf codecStructField, err error) {

	err = c.decodeBuffer.ReadUint16(&sf.Type)
	if err != nil {
		return
	}

	sf.NameLength, err = c.decodeBuffer.ReadByte()
	if err != nil {
		return
	}

	readed, _ := c.decodeBuffer.Read(c.dataBuffer.nameReader[:sf.NameLength])
	if readed != int(sf.NameLength) {
		return sf, errors.New("Read wrong amount of data when reading structure's field name")
	}

	sf.Name = string(c.dataBuffer.nameReader[:sf.NameLength])

	// calc size in struct
	sf.Size, err = c.getTypeSize(sf.Type)

	return

}

// returns struct header size
func (c *codec) tryDecodeStructure() (int, error) {

	headerSize := 0

	// read number of types
	nTypes, err := c.decodeBuffer.ReadByte()
	if err != nil {
		return headerSize, err
	}

	headerSize += 1

	for i := 0; i < int(nTypes); i++ {

		var typeDef structDefinition
		err = c.decodeBuffer.ReadUint16(&typeDef.Id)

		if err != nil {
			return headerSize, err
		}
		headerSize += 2

		typeDef.FieldCount, err = c.decodeBuffer.ReadByte()
		if err != nil {
			return headerSize, err
		}
		headerSize += 1

		skip := false
		if _, ok := c.types[typeDef.Id]; ok {
			// skip current structure, we already have it

			for j := 0; j < int(typeDef.FieldCount); j++ {
				c.decodeBuffer.Next(2)

				nameLength, _ := c.decodeBuffer.ReadByte()
				// name and type
				headerSize += 3
				headerSize += int(nameLength)

				c.decodeBuffer.Next(int(nameLength))
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

			c.types[typeDef.Id] = &typeDef
		}
	}

	return headerSize, nil

}

func (c *codec) Decode(out interface{}, input []byte) error {

	c.decodeBuffer.Init(input)

	headerSize, err := c.tryDecodeStructure()

	if err != nil {
		return err
	}

	var typeOfElement uint16
	c.decodeBuffer.ReadUint16(&typeOfElement)

	structSize, err := c.getTypeSize(typeOfElement)
	if err != nil {
		return err
	}

	refsOffset := structSize + headerSize + 2 // 2 bytes for struct type

	c.ref.Reader.Init(input[refsOffset:])

	// todo check if type is same in out interface and binary data given

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Not addressable value (%s) provided as out parameter. it should be a pointer to structure", v.Type().Kind().String()))
	}

	indirect := reflect.Indirect(v)

	//c.cacheReflectionData(typeOfElement, indirect.Type())

	return c.readComplexFieldData(typeOfElement, indirect)
}

func (c *codec) readFieldData(field codecStructField, out reflect.Value) error {

	if isArrayType(field.Type) {
		return c.readArrayElement(getArrayElementType(field.Type), out)
	} else {
		if field.Type > internalTypesCount {
			return c.readComplexFieldData(field.Type, out)
		} else {
			switch reflect.Kind(field.Type) {
			case reflect.String:
				return c.readReferenceFieldData(field.Type, out)
			default:
				return c.readSimpleFieldData(field.Type, out)
			}

		}
	}
}

func (c *codec) readSimpleFieldData(t uint16, out reflect.Value) error {

	tmpVal := reflect.Indirect(out)

	switch t {
	case uint16(reflect.Int), uint16(reflect.Int32):
		err := c.decodeBuffer.ReadInt32(&c.dataBuffer.int32val)

		if err != nil {
			return err
		}

		if tmpVal.CanSet() {
			tmpVal.SetInt(int64(c.dataBuffer.int32val))
		} else {
			return errors.New(fmt.Sprintf("Cant set value on field of type %d\n", t))
		}
	case uint16(reflect.Float64):
		c.decodeBuffer.ReadFloat64(&c.dataBuffer.float64val)
		if tmpVal.CanSet() {
			tmpVal.SetFloat(float64(c.dataBuffer.float64val))
		} else {
			return errors.New("unable to set float32 value of unaccessable field")
		}
	case uint16(reflect.Float32):

		c.decodeBuffer.ReadFloat32(&c.dataBuffer.float32val)
		if tmpVal.CanSet() {
			tmpVal.SetFloat(float64(c.dataBuffer.float32val))
		} else {
			return errors.New("unable to set float32 value of unaccessable field")
		}
	default:
		return errors.New(fmt.Sprintf("Unable to decode type %", t))
	}

	return nil

}

func (c *codec) readReferenceFieldData(t uint16, out reflect.Value) error {

	c.decodeBuffer.ReadUint16(&c.dataBuffer.uint16val)

	refBytes, length, err := c.ref.Reader.Get(uint64(c.dataBuffer.uint16val))
	if err != nil {
		return err
	}

	fmt.Printf("got ref bytes: = %s:%d. type = %d\n", refBytes, length, t)

	switch t {
	case uint16(reflect.String):
	case uint16(reflect.Slice):

		// string
	default:
		return errors.New(fmt.Sprintf("Unable to decode referenced type %d\n", t))
	}

	return nil
}

func (c *codec) readComplexFieldData(t uint16, out reflect.Value) (err error) {

	refValue := reflect.Indirect(out)

	if refValue.Kind() != reflect.Struct {
		return utils.Error("cannot decode data to %s", refValue.Kind().String())
	}

	tData, ok := c.types[t]
	if !ok {
		return errors.New(fmt.Sprintf("No structure data in coded on how to decode %d type", t))
	}

	for i := 0; i < int(tData.FieldCount); i++ {

		f := tData.Fields[i]

		fieldObj := refValue.Field(i)
		if fieldObj.Kind() == reflect.Ptr {
			if fieldObj.IsNil() {
				fieldObj.Set(reflect.New(fieldObj.Type().Elem()))
			}
		}

		err = c.readFieldData(f, fieldObj)

		if err != nil {
			return
		}
	}

	return nil
}
