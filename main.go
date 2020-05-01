package main

import (
	"encoding/json"
	"fmt"
	"github.com/dot5enko/transbin/codec"
	"testing"
	"time"
)

type NStruct struct {
	Nint    int
	Nstring int
	N3      int
	N5      int
	Floa    float64
}

type TestStruct struct {
	Id           int
	Value        float32
	NestedStruct NStruct
}

func PrintBenchmark(label string, result testing.BenchmarkResult) {

	memPerRun := float64(float64(result.MemBytes) / float64(result.N))
	allocsPerRun := float64(float64(result.MemAllocs) / float64(result.N))
	timePerRun := result.T / time.Duration(result.N)

	fmt.Printf("%8d time %8s. memory %f, memallocs %f, size %f [%s]\n", result.N, timePerRun, memPerRun, allocsPerRun, result.Extra["encoded_size"], label)
}

func main() {

	var toEncode TestStruct

	toEncode.Id = 49
	toEncode.Value = 32720.2383

	//toEncode.NestedStruct = &NStruct{}

	toEncode.NestedStruct.Nint = 99
	toEncode.NestedStruct.Nstring = 38
	toEncode.NestedStruct.N3 = 33
	toEncode.NestedStruct.N5 = 55
	toEncode.NestedStruct.Floa = 28973892.3833

	var encodedFull []byte
	c, _ := codec.NewCodec()

	encodedResult := c.EncodeFull(toEncode)

	decodedBack := TestStruct{}

	//codec.Reporting = true

	c.Decode(&decodedBack, encodedResult)
	//return

	//return
	PrintBenchmark("binary full encode", testing.Benchmark(func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			encodedFull = c.EncodeFull(toEncode)
		}

		b.ReportAllocs()
		b.ReportMetric(float64(len(encodedFull)), "encoded_size")
	}))

	binpro, _ := codec.NewCodec()
	PrintBenchmark("binary full decode", testing.Benchmark(func(b *testing.B) {

		var x TestStruct
		for i := 0; i < b.N; i++ {
			err := binpro.Decode(&x, encodedFull)
			if err != nil {
				panic(err)
			}
		}

		b.ReportAllocs()
		b.ReportMetric(float64(x.NestedStruct.Nint), "encoded_size")
	}))

	fmt.Println("")

	var encoded []byte
	bin, _ := codec.NewCodec()

	PrintBenchmark("binary data encode", testing.Benchmark(func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			encoded = bin.Encode(toEncode)
		}

		b.ReportAllocs()
		b.ReportMetric(float64(len(encoded)), "encoded_size")
	}))

	binpro1, _ := codec.NewCodec()

	PrintBenchmark("binary data decode", testing.Benchmark(func(b *testing.B) {

		var x TestStruct

		// read structure
		binpro1.Decode(&x, encodedFull)

		for i := 0; i < b.N; i++ {
			err := binpro1.Decode(&x, encoded)
			if err != nil {
				panic(err)
			}
		}

		b.ReportAllocs()
		b.ReportMetric(float64(x.NestedStruct.Nint), "encoded_size")
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
		b.ReportMetric(float64(x.NestedStruct.Nint), "encoded_size")
	}))

}
