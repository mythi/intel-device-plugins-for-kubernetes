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
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	// "io/ioutil"

	"strings"

	"github.com/pkg/errors"

	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/fpga/aocx"
	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/fpga/gbs"
	fpga "github.com/intel/intel-device-plugins-for-kubernetes/pkg/fpga/linux"
)

const (
	fpgaBitStreamDirectory = "/srv/intel.com/fpga"
)

func main() {
	var err error
	var bitstream string
	var device string
	var dryRun bool
	flag.StringVar(&bitstream, "b", "", "Path to bitstream file (GBS or AOCX)")
	flag.StringVar(&device, "d", "", "Path to device node (FME or Port)")
	flag.BoolVar(&dryRun, "dry-run", false, "Don't write/program, just validate and log")

	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Please provide command: info, fpgainfo, fmeinfo, portinfo, install, pr")
	}

	cmd := flag.Arg(0)
	err = validateFlags(cmd, bitstream, device)
	if err != nil {
		log.Fatalf("Invalid arguments: %+v", err)
	}

	// fmt.Printf("Cmd: %q\nBitstream: %q\nDevice: %q\n", cmd, bitstream, device)
	switch cmd {
	case "info":
		err = printBitstreamInfo(bitstream)
	case "pr":
		err = doPR(device, bitstream, dryRun)
	case "fpgainfo":
		err = fpgaInfo(device)
	case "fmeinfo":
		err = fmeInfo(device)
	case "portinfo":
		err = portInfo(device)
	case "install":
		err = installBitstream(bitstream, dryRun)
	default:
		err = errors.Errorf("unknown command %+v", flag.Args())

	}
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

func validateFlags(cmd, bitstream, device string) error {
	switch cmd {
	case "info", "install":
		// bitstream must not be empty
		if bitstream == "" {
			return errors.Errorf("bitstream filename is missing")
		}
	case "fpgainfo", "fmeinfo", "portinfo":
		// device must not be empty
		if device == "" {
			return errors.Errorf("FPGA device name is missing")
		}
	case "pr":
		// device and bitstream can't be empty
		if bitstream == "" {
			return errors.Errorf("bitstream filename is missing")
		}
		if device == "" {
			return errors.Errorf("FPGA device name is missing")
		}
	}
	return nil
}

type bitstreamInfo struct {
	InterfaceUUID       string
	AcceleratorTypeUUID string
	InstallPath         string
	Extra               map[string]string
}

func installBitstream(fname string, dryRun bool) (err error) {
	var info *bitstreamInfo
	switch filepath.Ext(fname) {
	case ".gbs":
		info, err = gbsInfo(fname)
	case ".aocx":
		info, err = aocxInfo(fname)
	default:
		err = errors.Errorf("unknown file format of file %s", fname)
	}
	if err != nil {
		return
	}
	fmt.Printf("Installing bitstream %q as %q\n", fname, info.InstallPath)
	if dryRun {
		fmt.Println("Dry-run: no copying performed")
		return
	}
	err = os.MkdirAll(filepath.Dir(info.InstallPath), 0755)
	if err != nil {
		return errors.Wrap(err, "unable to create destination directory")
	}
	src, err := os.Open(fname)
	if err != nil {
		return errors.Wrap(err, "can't open bitstream file")
	}
	defer src.Close()
	dst, err := os.OpenFile(info.InstallPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "can't create destination file")
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return
}

func fpgaInfo(fname string) error {
	switch {
	case strings.HasPrefix(fname, "/dev/dfl-fme."), strings.HasPrefix(fname, "/dev/intel-fpga-fme."):
		return fmeInfo(fname)
	case strings.HasPrefix(fname, "/dev/dfl-port."), strings.HasPrefix(fname, "/dev/intel-fpga-port."):
		return portInfo(fname)
	}
	return errors.Errorf("unknown FPGA device file %s", fname)
}

func printBitstreamInfo(fname string) (err error) {
	var info *bitstreamInfo
	switch filepath.Ext(fname) {
	case ".gbs":
		info, err = gbsInfo(fname)
	case ".aocx":
		info, err = aocxInfo(fname)
	default:
		err = errors.Errorf("unknown file format of file %s", fname)
	}
	if err != nil {
		return
	}
	fmt.Printf("Bitstream file        : %q\n", fname)
	fmt.Printf("Interface UUID        : %q\n", info.InterfaceUUID)
	fmt.Printf("Accelerator Type UUID : %q\n", info.AcceleratorTypeUUID)
	fmt.Printf("Installation Path     : %q\n", info.InstallPath)
	if len(info.Extra) > 0 {
		fmt.Println("Extra:")
		for k, v := range info.Extra {
			fmt.Printf("\t%s : %q\n", k, v)
		}
	}
	return
}

func gbsInfo(fname string) (info *bitstreamInfo, err error) {
	m, err := gbs.Open(fname)
	if err != nil {
		return
	}
	defer m.Close()

	info = &bitstreamInfo{
		InterfaceUUID:       m.InterfaceUUID(),
		AcceleratorTypeUUID: m.AcceleratorTypeUUID(),
		InstallPath:         filepath.Join(fpgaBitStreamDirectory, m.InterfaceUUID(), m.AcceleratorTypeUUID()+filepath.Ext(fname)),
		Extra:               map[string]string{"Size": strconv.FormatUint(m.Bitstream.Size, 10)},
	}
	return
}

func aocxInfo(fname string) (info *bitstreamInfo, err error) {
	m, err := aocx.Open(fname)
	if err != nil {
		return
	}
	defer m.Close()

	if m.GBS != nil {
		info = &bitstreamInfo{
			InterfaceUUID:       m.GBS.InterfaceUUID(),
			AcceleratorTypeUUID: m.GBS.AcceleratorTypeUUID(),
			InstallPath:         filepath.Join(fpgaBitStreamDirectory, m.GBS.InterfaceUUID(), m.Hash+filepath.Ext(fname)),
			Extra: map[string]string{
				"Board":   m.Board,
				"Target":  m.Target,
				"Hash":    m.Hash,
				"Version": m.Version,
				"Size":    strconv.FormatUint(m.GBS.Bitstream.Size, 10),
			},
		}
		return
	}
	return nil, errors.Errorf("can't read GBS from AOCX file")
}

func fmeInfo(fname string) error {
	var f fpga.FpgaFME
	var err error
	switch {
	case strings.HasPrefix(fname, "/dev/dfl-fme."):
		f, err = fpga.NewDflFME(fname)
	case strings.HasPrefix(fname, "/dev/intel-fpga-fme."):
		f, err = fpga.NewIntelFpgaFME(fname)
	default:
		return errors.Errorf("unknow type of FME %s", fname)
	}
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Print("API:")
	fmt.Println(f.GetAPIVersion())
	fmt.Print("CheckExtension:")
	fmt.Println(f.CheckExtension())
	return nil
}

func portInfo(fname string) error {
	var f fpga.FpgaPort
	var err error
	switch {
	case strings.HasPrefix(fname, "/dev/dfl-port."):
		f, err = fpga.NewDflPort(fname)
	case strings.HasPrefix(fname, "/dev/intel-fpga-port."):
		f, err = fpga.NewIntelFpgaPort(fname)
	default:
		err = errors.Errorf("unknown type of port %s", fname)
	}
	if err != nil {
		return err
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
	return nil
}

func doPR(fme, bs string, dryRun bool) error {
	var f fpga.FpgaFME
	var err error
	switch {
	case strings.HasPrefix(fme, "/dev/dfl-fme."):
		f, err = fpga.NewDflFME(fme)
	case strings.HasPrefix(fme, "/dev/intel-fpga-fme."):
		f, err = fpga.NewIntelFpgaFME(fme)
	default:
		return errors.Errorf("unknown FME %s", fme)
	}
	fmt.Printf("Trying to program %s to port 0 of %s", bs, fme)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Print("API:")
	fmt.Println(f.GetAPIVersion())
	m, err := gbs.Open(bs)
	if err != nil {
		return err
	}
	defer m.Close()
	// fmt.Printf("Return:\n%+v\n", m)
	if m.Bitstream != nil {
		rawBistream, err := m.Bitstream.Data()
		if err != nil {
			return err
		}
		if dryRun {
			fmt.Println("Dry-Run: Skipping actual programming")
			return nil
		}
		fmt.Print("Trying to PR, brace yourself! :")
		fmt.Println(f.PortPR(0, rawBistream))
	}
	return nil
}
