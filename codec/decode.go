package codec

import (
	"errors"
	"fmt"
	"reflect"
)

var nameReader [255]byte

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

		if _, ok := c.types[typeDef.Id]; ok {
			// skip current structure, we already have it

			for j := 0; j < int(typeDef.FieldCount); j++ {
				c.decodeBuffer.Next(2)
				nameLength, _ := c.decodeBuffer.ReadByte()
				c.decodeBuffer.Next(int(nameLength))
			}

			continue
		}

		typeDef.Fields = make([]codecStructField, typeDef.FieldCount)

		for j := 0; j < int(typeDef.FieldCount); j++ {

			sf := &typeDef.Fields[j]

			err = c.decodeBuffer.ReadUint16(&sf.Type)
			if err != nil {
				return err
			}

			sf.NameLength, err = c.decodeBuffer.ReadByte()
			if err != nil {
				return err
			}

			readed, _ := c.decodeBuffer.Read(nameReader[:sf.NameLength])
			if readed != int(sf.NameLength) {
				return errors.New("Read wrong amount of data when reading structure's field name")
			}

			sf.Name = string(nameReader[:sf.NameLength])

		}
		c.types[typeDef.Id] = &typeDef
	}

	return nil

}

var typeOfElement uint16

func (c *codec) Decode(out interface{}, input []byte) error {

	c.decodeBuffer.Init(input)
	ReportAllocs(" - decode structure ")
	c.tryDecodeStructure()
	c.decodeBuffer.ReadUint16(&typeOfElement)

	// todo check if type is same in out interface and binary data given

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("Not addressable value (%s) provided as out parameter. it should be a pointer to structure", v.Type().Kind().String()))
	}

	indirect := reflect.Indirect(v);

	//c.cacheReflectionData(typeOfElement, indirect.Type())

	return c.readComplexFieldData(typeOfElement, indirect)
}

func (c *codec) readFieldData(field codecStructField, out reflect.Value) error {

	if field.Type > 26 {
		return c.readComplexFieldData(field.Type, out)
	} else if field.Type == 24 {
		return c.readReferenceFieldData(field.Type, out)
	} else {
		return c.readSimpleFieldData(field.Type, out)
	}
}

func (c *codec) readSimpleFieldData(t uint16, out reflect.Value) error {

	ReportAllocs(" - readSimpleFieldData")

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
	case uint16(reflect.Float32):
		ReportAllocs(" - read int32  ")

		c.decodeBuffer.ReadFloat32(&c.dataBuffer.float32val)
		if tmpVal.CanSet() {
			tmpVal.SetFloat(float64(c.dataBuffer.float32val))
			ReportAllocs("  + read field float32  ")
		} else {
			return errors.New("unable to set float32 value of unaccessable field")
		}
	default:
		return errors.New(fmt.Sprintf("Unable to decode type %", t))
	}

	ReportAllocs(" + readSimpleFieldData")

	return nil

}

func (c *codec) readReferenceFieldData(t uint16, out reflect.Value) error {
	switch t {
	case 24:

		c.decodeBuffer.ReadUint16(&c.dataBuffer.uint16val)

		// string
		//fmt.Printf("reading refernced data %d...\n", referenceId)
	default:
		return errors.New(fmt.Sprintf("Unable to decode referenced type %d\n", t))
	}

	return nil
}

func (c *codec) readComplexFieldData(t uint16, out reflect.Value) (err error) {

	refValue := reflect.Indirect(out)

	if refValue.Kind() != reflect.Struct {
		return errors.New("cannot decode data to " + refValue.Kind().String())
	}

	tData, ok := c.types[t]
	if !ok {
		return errors.New(fmt.Sprintf("No structure data in coded on how to decode %d type", t))
	}

	for i := 0; i < int(tData.FieldCount); i++ {

		ReportAllocs("------read complex field ")
		f := tData.Fields[i]

		fieldObj := refValue.Field(i)
		ReportAllocs("++++++read complex field")
		err = c.readFieldData(f, fieldObj)

		if err != nil {
			return
		}
	}

	return nil
}
