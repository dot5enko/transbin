package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/dot5enko/transbin/codec"
	"testing"
	"time"
)

import _ "net/http/pprof"

type ProductVal struct {
	Name  string
	Price float64
}

type MapValStruct struct {
	Int  int
	Name string
}

type NStruct struct {
	Nint    int
	Nstring int
	N3      int
	N5      int
	Floa    float64
	Fl2     float64
	Product ProductVal
}

type TestStruct struct {
	Id           int
	Value        float32
	NestedStruct []NStruct
	MapVal       MapValStruct
	StrVal       string
}

func PrintBenchmark(label string, result testing.BenchmarkResult) {

	memPerRun := float64(float64(result.MemBytes) / float64(result.N))
	allocsPerRun := float64(float64(result.MemAllocs) / float64(result.N))
	timePerRun := result.T / time.Duration(result.N)

	fmt.Printf("%8d time %8s. memory %f, memallocs %f, size %f [%s]\n", result.N, timePerRun, memPerRun, allocsPerRun, result.Extra["encoded_size"], label)
}

func main() {

	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()

	var toEncode TestStruct

	toEncode.Id = 49
	toEncode.Value = 32720.2383
	toEncode.StrVal = "holaAmigo grande!"
	//toEncode.Ids = []int32{99, 88, 77, 66, 55, 44, 33, 22, 11}

	toEncode.MapVal.Int = 5
	toEncode.MapVal.Name = "serhii"

	ids := 10
	toEncode.NestedStruct = make([]NStruct, ids)

	for i := 0; i < ids; i++ {
		toEncode.NestedStruct[i].Nint = 99
		toEncode.NestedStruct[i].Nstring = 38
		toEncode.NestedStruct[i].N3 = 33
		toEncode.NestedStruct[i].N5 = 55
		toEncode.NestedStruct[i].Floa = 28973892.3833
		toEncode.NestedStruct[i].Fl2 = 99.98765432

		toEncode.NestedStruct[i].Product.Price = 10.95
		toEncode.NestedStruct[i].Product.Name = "json binary self describing proto"

	}
	//
	//toEncode := make(map[string]interface{})
	//
	//toEncode["name"] = "serhii"
	//toEncode["price"] = 500.99

	jb0, _ := json.Marshal(toEncode)
	fmt.Printf("Bef a result : %s[%d]\n", jb0, len(jb0))

	var encodedFull []byte
	c, _ := codec.NewCodec(binary.LittleEndian)

	ctx := codec.NewEncodeContext(c)

	encodedResult, err := ctx.EncodeFull(toEncode)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("Got encoded data %d bytes length\n", len(encodedResult))

	//decodedBack := make(map[string]interface{})
	decodedBack := TestStruct{}
	decodeCtx := codec.NewDecodeContext(c)

	//codec.Reporting = true
	err = decodeCtx.Decode(&decodedBack, encodedResult)
	if err != nil {
		panic(err)
	}

	jb, _ := json.Marshal(decodedBack)
	fmt.Printf("Got a result : %s\n", jb)

	time.Sleep(time.Microsecond)
	//return

	////return
	PrintBenchmark("binary full encode", testing.Benchmark(func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			encodedFull, _ = ctx.EncodeFull(toEncode)
		}

		b.ReportAllocs()
		b.ReportMetric(float64(len(encodedFull)), "encoded_size")
	}))

	binpro, _ := codec.NewCodec(binary.LittleEndian)
	binproDec := codec.NewDecodeContext(binpro)
	PrintBenchmark("binary full decode", testing.Benchmark(func(b *testing.B) {

		var x TestStruct
		for i := 0; i < b.N; i++ {
			err := binproDec.Decode(&x, encodedFull)
			if err != nil {
				panic(err)
			}
		}

		b.ReportAllocs()
		b.ReportMetric(float64(x.NestedStruct[0].Floa), "encoded_size")
	}))

	fmt.Println("")

	var encoded []byte
	bin, _ := codec.NewCodec(binary.LittleEndian)
	binCtx := codec.NewEncodeContext(bin)

	PrintBenchmark("binary data encode", testing.Benchmark(func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			encoded, _ = binCtx.Encode(toEncode)
		}

		b.ReportAllocs()
		b.ReportMetric(float64(len(encoded)), "encoded_size")
	}))

	binpro1, _ := codec.NewCodec(binary.LittleEndian)
	binpro1Ctx := codec.NewDecodeContext(binpro1)

	PrintBenchmark("binary data decode", testing.Benchmark(func(b *testing.B) {

		var x TestStruct

		// read structure
		binpro1Ctx.Decode(&x, encodedFull)

		for i := 0; i < b.N; i++ {
			err := binpro1Ctx.Decode(&x, encoded)
			if err != nil {
				panic(err)
			}
		}

		b.ReportAllocs()
		b.ReportMetric(float64(x.NestedStruct[0].Floa), "encoded_size")
	}))

	fmt.Println("")

	var encodedJson []byte

	PrintBenchmark("json encode", testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encodedJson, _ = json.Marshal(toEncode)
		}

		b.ReportAllocs()
		b.ReportMetric(float64(len(encodedJson)), "encoded_size")
	}))

	PrintBenchmark("json decode", testing.Benchmark(func(b *testing.B) {

		var x TestStruct
		for i := 0; i < b.N; i++ {
			err := json.Unmarshal(encodedJson, &x)
			if err != nil {
				fmt.Printf("Got decoding error: %s\n", err.Error())
			}
		}

		b.ReportAllocs()
		b.ReportMetric(float64(x.NestedStruct[0].Floa), "encoded_size")
	}))

}
