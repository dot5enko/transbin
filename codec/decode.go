package codec

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
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

	return

}
func (c *codec) tryDecodeStructure() error {

	// read number of types
	nTypes, err := c.decodeBuffer.ReadByte()
	if err != nil {
		return err
	}

	for i := 0; i < int(nTypes); i++ {
		var typeDef structDefinition
		err = c.decodeBuffer.ReadUint16(&typeDef.Id)
		if err != nil {
			return err
		}

		typeDef.FieldCount, err = c.decodeBuffer.ReadByte()

		if err != nil {
			return err
		}

		skip := false
		if _, ok := c.types[typeDef.Id]; ok {
			// skip current structure, we already have it

			for j := 0; j < int(typeDef.FieldCount); j++ {
				c.decodeBuffer.Next(2)
				nameLength, _ := c.decodeBuffer.ReadByte()
				c.decodeBuffer.Next(int(nameLength))
			}

			skip = true
		}

		if !skip {

			typeDef.Fields = make([]codecStructField, typeDef.FieldCount)

			for j := 0; j < int(typeDef.FieldCount); j++ {

				typeDef.Fields[j], _ = c.readStructFieldData()

			}
			c.types[typeDef.Id] = &typeDef
		}
	}

	return nil

}

func (c *codec) Decode(out interface{}, input []byte) error {
	return c.decodeInternal(iWrapper{out}, input)
}

func (c *codec) decodeInternal(out iWrapper, input []byte) error {

	c.decodeBuffer.Init(input)
	c.tryDecodeStructure()

	var typeOfElement uint16
	c.decodeBuffer.ReadUint16(&typeOfElement)

	// todo check if type is same in out interface and binary data given

	v := reflect.ValueOf(out.iface)
	if v.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Not addressable value (%s) provided as out parameter. it should be a pointer to structure", v.Type().Kind().String()))
	}

	indirect := reflect.Indirect(v)

	c.cacheReflectionData(typeOfElement, indirect.Type())


	var objPtr uintptr
	objWrapper := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		objPtr = objWrapper.Field(0).InterfaceData()[1]
	} else {
		objPtr = v.Pointer()
	}


	return c.readComplexType(typeOfElement, objPtr)
}

func (c *codec) readFieldData(field codecStructField, out uintptr) error {

	if field.Type > 26 {
		return c.readComplexFieldData(field, out)
	} else if field.Type == 24 {
		return c.readReferenceFieldData(field, out)
	} else {
		return c.readSimpleFieldData(field, out)
	}
}

func (c *codec) readSimpleFieldData(t codecStructField, out uintptr) error {

	// todo handle indirect

	switch t.Type {
	case uint16(reflect.Int), uint16(reflect.Int32):
		err := c.decodeBuffer.ReadInt32(&c.dataBuffer.int32val)

		if err != nil {
			return err
		}

		*(*int32)(unsafe.Pointer(out + t.Offset)) = c.dataBuffer.int32val

	case uint16(reflect.Float64):
		c.decodeBuffer.ReadFloat64(&c.dataBuffer.float64val)

		*(*float64)(unsafe.Pointer(out + t.Offset)) = c.dataBuffer.float64val
	case uint16(reflect.Float32):

		c.decodeBuffer.ReadFloat32(&c.dataBuffer.float32val)
		*(*float32)(unsafe.Pointer(out + t.Offset)) = c.dataBuffer.float32val

	default:
		return errors.New(fmt.Sprintf("Unable to decode type %", t))
	}

	return nil

}

func (c *codec) readReferenceFieldData(t codecStructField, out uintptr) error {
	switch t.Type {
	case 24:

		c.decodeBuffer.ReadUint16(&c.dataBuffer.uint16val)

		// string
		//fmt.Printf("reading refernced data %d...\n", referenceId)
	default:
		return errors.New(fmt.Sprintf("Unable to decode referenced type %d\n", t))
	}

	return nil
}

func (c *codec) readComplexType(t uint16, out uintptr) (err error) {
	tData, ok := c.types[t]
	if !ok {
		return errors.New(fmt.Sprintf("No structure data in coded on how to decode %d type", t))
	}

	for i := 0; i < int(tData.FieldCount); i++ {

		f := tData.Fields[i]

		// todo handle pointer type fields
		//fieldObj := refValue.Field(i)
		//if fieldObj.Kind() == reflect.Ptr {
		//	if fieldObj.IsNil() {
		//		fieldObj.Set(reflect.New(fieldObj.Type().Elem()))
		//	}
		//}

		err = c.readFieldData(f, out)

		if err != nil {
			return
		}
	}

	return nil
}
func (c *codec) readComplexFieldData(t codecStructField, out uintptr) (err error) {

	// todo hanlde indirect value
	return c.readComplexType(t.Type,out + t.Offset)

}
