package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"runtime"
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
	decodeBuffer    *decode_buffer
	dataBuffer      struct {
		int32val   int32
		float32val float32
		uint16val  uint16
	}
}

var res, firstRun runtime.MemStats
var reports int

const allocsPerRun int = 3
const statFormat string = "%4d %8d %25s\n"

//var prevMallocs, curMallocs,mallocs uint64

func ReportAllocs(label string) {

	runtime.ReadMemStats(&res)

	if reports == 0 {
		firstRun = res
	}

	//prevMallocs = prev.Mallocs - uint64(allocsPerRun*(reports-1))
	//
	//if prevMallocs < 0 {
	//	prevMallocs = 0
	//} else if prevMallocs > 1000000 {
	//	prevMallocs = 0
	//}
	//
	//curMallocs = res.Mallocs - uint64(allocsPerRun*reports)
	//mallocs = curMallocs - prevMallocs

	fmt.Printf(statFormat, res.Mallocs-uint64(allocsPerRun*reports)-firstRun.Mallocs, res.Alloc-firstRun.Alloc, label)

	reports++
}

func NewCodec() (*codec, error) {
	result := &codec{}

	// in order to not interfer with internal types

	result.typesCount = 27
	result.types = make(map[uint16]structDefinition)
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
