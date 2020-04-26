package codec

import (
	"errors"
	"fmt"
	"reflect"
)

var typeDef structDefinition
var nameReader [255]byte

func (c *codec) tryDecodeStructure() error {

	// read number of types
	nTypes, err := c.decodeBuffer.ReadByte()
	if err != nil {
		return err
	}

	for i := 0; i < int(nTypes); i++ {
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

			readed,_ := c.decodeBuffer.Read(nameReader[:sf.NameLength])
			if readed != int(sf.NameLength) {
				return errors.New("Read wrong amount of data when reading structure's field name")
			}

			sf.Name = string(nameReader[:sf.NameLength])

		}
		c.types[typeDef.Id] = typeDef
	}

	return nil

}
var typeOfElement uint16
func (c *codec) Decode(out interface{}, input []byte) error {

	c.decodeBuffer.Init(input)

	c.tryDecodeStructure()

	c.decodeBuffer.ReadUint16(&typeOfElement)

	return c.readComplexFieldData(typeOfElement, reflect.ValueOf(out))
}


func (c *codec) readFieldData(field codecStructField,out reflect.Value) error {

	if field.Type > 26 {
		return c.readComplexFieldData(field.Type, out)
	} else if field.Type == 24 {
		return c.readReferenceFieldData(field.Type, out)
	} else {
		return c.readSimpleFieldData(field.Type,out)
	}
}

var intval int32
var floatval float32

func (c *codec) readSimpleFieldData(t uint16, out reflect.Value) (err error) {

	tmpVal := reflect.Indirect(out)

	switch t {
	case uint16(reflect.Int), uint16(reflect.Int32):

		err = c.decodeBuffer.ReadInt32(&intval);
		if err != nil {
			return err
		}

		if tmpVal.CanSet() {
			tmpVal.Set(reflect.ValueOf(int(intval)))
		} else {
			return errors.New(fmt.Sprintf("Cant set value on field of type %d\n",t))
		}
	case uint16(reflect.Float32):

		c.decodeBuffer.ReadFloat32(&floatval)
		if tmpVal.CanSet() {
			tmpVal.Set(reflect.ValueOf(floatval))
		} else {
			return errors.New("unable to set float32 value of unaccessable field")
		}
	default:
		return errors.New(fmt.Sprintf("Unable to decode type %", t))
	}

	return

}
var referenceId uint16

func (c *codec) readReferenceFieldData(t uint16, out reflect.Value) error {
	switch t {
	case 24:

		c.decodeBuffer.ReadUint16(&referenceId)

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

		f := tData.Fields[i]

		fieldObj := refValue.FieldByName(f.Name)
		err = c.readFieldData(f, fieldObj)
		if err != nil {
			return
		}
	}

	return nil
}

