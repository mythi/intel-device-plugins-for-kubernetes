// Copyright 2020 Intel Corporation. All Rights Reserved.
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
	"os"
	"path/filepath"
	"time"

	"github.com/google/gousb"

	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/debug"
	dpapi "github.com/intel/intel-device-plugins-for-kubernetes/pkg/deviceplugin"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	// Movidius MyriadX Vendor ID
	vendorID = 0x03e7
	// Movidius MyriadX Product ID
	productID = 0xf63b
	// Device plugin settings.
	namespace        = "vpu.intel.com"
	deviceType       = "hddl"
	daemonDeviceType = "hddldaemon"

	hddlContainerPath = "/var/tmp"
	hddlServiceSock   = "hddl_service.sock"
	hddlServiceReady  = "hddl_service_ready.mutex"
	hddlServiceAlive  = "hddl_service_alive.mutex"
	ionDevNode        = "/dev/ion"
	myriadDevNode     = "/dev/myriad"
)

var (
	isdebug = flag.Int("debug", 0, "debug level (0..1)")
)

type gousbContext interface {
	OpenDevices(opener func(desc *gousb.DeviceDesc) bool) ([]*gousb.Device, error)
}

type devicePlugin struct {
	usbContext   gousbContext
	vendorID     int
	productID    int
	sharedDevNum int
	hddlHostPath string
}

func newDevicePlugin(usbContext gousbContext, vendorID int, productID int, sharedDevNum int, hddlHostPath string) *devicePlugin {
	return &devicePlugin{
		usbContext:   usbContext,
		vendorID:     vendorID,
		productID:    productID,
		sharedDevNum: sharedDevNum,
		hddlHostPath: hddlHostPath,
	}
}

func (dp *devicePlugin) Scan(notifier dpapi.Notifier) error {
	for {
		devTree, err := dp.scan()
		if err != nil {
			return err
		}

		notifier.Notify(devTree)

		time.Sleep(5 * time.Second)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err == nil && info != nil {
		return !info.IsDir()
	}
	// regard all other case as abnormal
	return false
}

func (dp *devicePlugin) scan() (dpapi.DeviceTree, error) {
	var nUsb int
	needsDaemon := false
	devTree := dpapi.NewDeviceTree()

	// first check if HDDL sock is there
	if !fileExists(filepath.Join(dp.hddlHostPath, hddlServiceSock)) {
		needsDaemon = true
	}

	devs, err := dp.usbContext.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		thisVendor := desc.Vendor
		thisProduct := desc.Product
		debug.Printf("checking %04x,%04x vs %s,%s", dp.vendorID, dp.productID, thisVendor.String(), thisProduct.String())
		if (gousb.ID(dp.vendorID) == thisVendor) && (gousb.ID(dp.productID) == thisProduct) {
			nUsb++
		}
		return false
	})
	defer func() {
		for _, d := range devs {
			// TODO(mythi): close in parallel?
			d.Close()
		}
	}()

	if err != nil {
		debug.Printf("list usb device %s", err)
	}
	if nUsb == 0 {
		return devTree, nil
	}

	debug.Printf("found %d devices, needsDaemon: %+v", nUsb, needsDaemon)

	if !needsDaemon {
		for i := 0; i < nUsb*dp.sharedDevNum; i++ {
			devID := fmt.Sprintf("hddl_service-%d", i)
			// HDDL use a unix socket as service provider to manage /dev/myriad[n]
			// Here we only expose an ION device to be allocated for HDDL client in containers
			nodes := []pluginapi.DeviceSpec{
				{
					HostPath:      ionDevNode,
					ContainerPath: ionDevNode,
					Permissions:   "rw",
				},
			}

			mounts := []pluginapi.Mount{
				{
					HostPath:      filepath.Join(dp.hddlHostPath, hddlServiceSock),
					ContainerPath: filepath.Join(hddlContainerPath, hddlServiceSock),
				},
				{
					HostPath:      filepath.Join(dp.hddlHostPath, hddlServiceAlive),
					ContainerPath: filepath.Join(hddlContainerPath, hddlServiceAlive),
				},
				{
					HostPath:      filepath.Join(dp.hddlHostPath, hddlServiceReady),
					ContainerPath: filepath.Join(hddlContainerPath, hddlServiceReady),
				},
			}
			devTree.AddDevice(deviceType, devID, dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))
		}
	} else {
		var nodes []pluginapi.DeviceSpec
		for i := 0; i < nUsb; i++ {
			nodes = append(nodes, pluginapi.DeviceSpec{
				HostPath:      fmt.Sprintf("%s%d", myriadDevNode, i),
				ContainerPath: fmt.Sprintf("%s%d", myriadDevNode, i),
				Permissions:   "rw",
			})
		}
		// TODO(mythi): hddldaemon may want to run bsl_reset?
		nodes = append(nodes, pluginapi.DeviceSpec{
			HostPath:      ionDevNode,
			ContainerPath: ionDevNode,
			Permissions:   "rw",
		})
		mounts := []pluginapi.Mount{
			{
				HostPath:      "/dev/bus/usb",
				ContainerPath: "/dev/bus/usb",
			},
			{
				HostPath:      dp.hddlHostPath,
				ContainerPath: hddlContainerPath,
			},
		}
		devTree.AddDevice(daemonDeviceType, "hddldaemon_service-0", dpapi.NewDeviceInfo(pluginapi.Healthy, nodes, mounts, nil))
	}

	return devTree, nil
}

func main() {
	var sharedDevNum int
	var autobootStartupDelay string
	var hddlHostPath string

	flag.IntVar(&sharedDevNum, "shared-dev-num", 1, "number of containers sharing the same VPU device")
	flag.StringVar(&autobootStartupDelay, "autoboot-startup-delay", "1s", "startup delay to wait autoboot device reset")
	// the deployment can set this to any path on the host.
	flag.StringVar(&hddlHostPath, "hddl-host-path", hddlContainerPath, "")

	flag.Parse()

	if *isdebug > 0 {
		debug.Activate()
		debug.Printf("isdebug is on")
	}

	if sharedDevNum < 1 {
		fmt.Println("The number of containers sharing the same VPU must greater than zero")
		os.Exit(1)
	}

	delay, err := time.ParseDuration(autobootStartupDelay)
	if err != nil {
		fmt.Println("error")
		os.Exit(1)
	}
	time.Sleep(delay)
	fmt.Println("VPU device plugin started")

	// add lsusb here
	ctx := gousb.NewContext()
	defer ctx.Close()
	ctx.Debug(*isdebug)

	plugin := newDevicePlugin(ctx, vendorID, productID, sharedDevNum, hddlHostPath)
	manager := dpapi.NewManager(namespace, plugin)
	manager.Run()
}
