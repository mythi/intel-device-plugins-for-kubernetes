// Copyright 2019 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"log"

	// "io/ioutil"

	"strings"

	"github.com/pkg/errors"

	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/fpga/aocx"
	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/fpga/gbs"
	fpga "github.com/intel/intel-device-plugins-for-kubernetes/pkg/fpga/linux"
)

func main() {
	var err error

	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Please provide filename")
	}

	fname := flag.Arg(0)

	switch {
	case strings.HasSuffix(fname, ".gbs"):
		gbsInfo(fname)
	case strings.HasSuffix(fname, ".aocx"):
		aocxInfo(fname)
	case strings.HasPrefix(fname, "/dev/dfl-fme."), strings.HasPrefix(fname, "/dev/intel-fpga-fme."):
		fmeInfo(fname)
	case strings.HasPrefix(fname, "/dev/dfl-port."), strings.HasPrefix(fname, "/dev/intel-fpga-port."):
		portInfo(fname)
	case fname == "pr":
		if flag.NArg() < 3 {
			log.Fatal("pr fme gbs")
		}
		fme := flag.Arg(1)
		bs := flag.Arg(2)
		doPR(fme, bs)

	default:
		err = errors.Errorf("unknown arguments %+v", flag.Args())

	}
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

func gbsInfo(fname string) {
	m, err := gbs.Open(fname)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer m.Close()

	// fmt.Printf("Return:\n%+v\n", m)
	fmt.Printf("GBS InterfaceID: %s AFU UUID: %+v Size: %d\n", m.InterfaceUUID(), m.AcceleratorTypeUUID(), m.Bitstream.Size)

	// if m.Bitstream != nil {
	// 	fmt.Printf("Bitstream: %+v\n", m.Bitstream)
	// 	// r, err := m.Bitstream.Data()
	// 	// if err != nil {
	// 	// 	log.Fatalf("%+v", err)
	// 	// }
	// 	// ioutil.WriteFile(flag.Arg(0)+".rbf", r, 0644)
	// 	// f, err := os.OpenFile(fname+".rbf", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	// 	// if err != nil {
	// 	// 	log.Fatalf("%+v", err)
	// 	// }
	// 	// defer f.Close()
	// 	// wr, err := io.Copy(f, m.Bitstream.Open())
	// 	// if err != nil {
	// 	// 	log.Fatalf("%+v", err)
	// 	// }
	// 	// fmt.Printf("Written %d bytes\n", wr)
	// }
}

func aocxInfo(fname string) {
	m, err := aocx.Open(fname)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer m.Close()
	// fmt.Printf("Return:\n%+v\n", m)
	fmt.Printf("AOCX Info:\nBoard: %s\nTarget: %s\nHash: %s\nVersion: %s\n", m.Board, m.Target, m.Hash, m.Version)
	if m.GBS != nil {
		// fmt.Printf("GBS: %+v\n", m.GBS)
		fmt.Printf("GBS InterfaceID: %s AFU UUID: %+v Hash: %s Size: %d\n", m.GBS.InterfaceUUID(), m.GBS.AcceleratorTypeUUID(), m.Hash, m.GBS.Bitstream.Size)
	}
}

func fmeInfo(fname string) {
	var f fpga.FpgaFME
	var err error
	switch {
	case strings.HasPrefix(fname, "/dev/dfl-fme."):
		f, err = fpga.NewDflFME(fname)
	case strings.HasPrefix(fname, "/dev/intel-fpga-fme."):
		f, err = fpga.NewIntelFpgaFME(fname)
	}
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer f.Close()
	fmt.Print("API:")
	fmt.Println(f.GetAPIVersion())
	fmt.Print("CheckExtension:")
	fmt.Println(f.CheckExtension())
}

func portInfo(fname string) {
	var f fpga.FpgaPort
	var err error
	switch {
	case strings.HasPrefix(fname, "/dev/dfl-port."):
		f, err = fpga.NewDflPort(fname)
	case strings.HasPrefix(fname, "/dev/intel-fpga-port."):
		f, err = fpga.NewIntelFpgaPort(fname)
	}
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer f.Close()
	fmt.Print("API:")
	fmt.Println(f.GetAPIVersion())
	fmt.Print("CheckExtension:")
	fmt.Println(f.CheckExtension())
	fmt.Print("Reset:")
	fmt.Println(f.PortReset())
	fmt.Print("PortGetInfo:")
	fmt.Println(f.PortGetInfo())
	pi, err := f.PortGetInfo()
	if err == nil {
		for idx := 0; uint32(idx) < pi.Regions; idx++ {
			fmt.Printf("PortGetRegionInfo %d\n", idx)
			fmt.Println(f.PortGetRegionInfo(uint32(idx)))
		}
	}
}

func doPR(fme, bs string) {
	var f fpga.FpgaFME
	var err error
	switch {
	case strings.HasPrefix(fme, "/dev/dfl-fme."):
		f, err = fpga.NewDflFME(fme)
	case strings.HasPrefix(fme, "/dev/intel-fpga-fme."):
		f, err = fpga.NewIntelFpgaFME(fme)
	default:
		log.Fatalf("unknown FME")
	}
	fmt.Printf("Trying to program %s to port 0 of %s", bs, fme)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer f.Close()
	fmt.Print("API:")
	fmt.Println(f.GetAPIVersion())
	m, err := gbs.Open(bs)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer m.Close()
	// fmt.Printf("Return:\n%+v\n", m)
	if m.Bitstream != nil {
		rawBistream, err := m.Bitstream.Data()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		fmt.Print("Trying to PR, brace yourself! :")
		fmt.Println(f.PortPR(0, rawBistream))
	}
}
