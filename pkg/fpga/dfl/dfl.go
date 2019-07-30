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

package dfl

const (
	// Constants from <uapi/linux/fpga-dfl.h>
	// #define DFL_FPGA_API_VERSION 0

	// ioctl calls for DFL
	// #define DFL_FPGA_MAGIC 0xB6
	// #define DFL_FPGA_BASE 0
	// #define DFL_PORT_BASE 0x40
	// #define DFL_FME_BASE 0x80

	// * Common IOCTLs for both FME and AFU file descriptor *

	// FPGAGetAPIVersion IOCTL
	// * DFL_FPGA_GET_API_VERSION - _IO(DFL_FPGA_MAGIC, DFL_FPGA_BASE + 0)
	// *
	// * Report the version of the driver API.
	// * Return: Driver API Version.
	FPGAGetAPIVersion = 0xB600

	// FPGACheckExtension IOCTL
	// * DFL_FPGA_CHECK_EXTENSION - _IO(DFL_FPGA_MAGIC, DFL_FPGA_BASE + 1)
	// *
	// * Check whether an extension is supported.
	// * Return: 0 if not supported, otherwise the extension is supported.
	FPGACheckExtension = 0xB601

	// IOCTLs for AFU file descriptor

	// FPGAPortReset IOCTL
	// * DFL_FPGA_PORT_RESET - _IO(DFL_FPGA_MAGIC, DFL_PORT_BASE + 0)
	// *
	// * Reset the FPGA Port and its AFU. No parameters are supported.
	// * Userspace can do Port reset at any time, e.g. during DMA or PR. But
	// * it should never cause any system level issue, only functional failure
	// * (e.g. DMA or PR operation failure) and be recoverable from the failure.
	// * Return: 0 on success, -errno of failure
	FPGAPortReset = 0xB640

	// FPGAPortGetInfo IOCTL
	// * DFL_FPGA_PORT_GET_INFO - _IOR(DFL_FPGA_MAGIC, DFL_PORT_BASE + 1,
	// *						struct dfl_fpga_port_info)
	// *
	// * Retrieve information about the fpga port.
	// * Driver fills the info in provided struct dfl_fpga_port_info.
	// * Return: 0 on success, -errno on failure.
	FPGAPortGetInfo = 0xB641

	// FPGAPortGetRegionInfo IOCTL
	// * FPGA_PORT_GET_REGION_INFO - _IOWR(FPGA_MAGIC, PORT_BASE + 2,
	// *					struct dfl_fpga_port_region_info)
	// *
	// * Retrieve information about a device memory region.
	// * Caller provides struct dfl_fpga_port_region_info with index value set.
	// * Driver returns the region info in other fields.
	// * Return: 0 on success, -errno on failure.
	FPGAPortGetRegionInfo = 0xB642

	// FPGAPortDMAMap IOCTL
	// * DFL_FPGA_PORT_DMA_MAP - _IOWR(DFL_FPGA_MAGIC, DFL_PORT_BASE + 3,
	// *						struct dfl_fpga_port_dma_map)
	// *
	// * Map the dma memory per user_addr and length which are provided by caller.
	// * Driver fills the iova in provided struct afu_port_dma_map.
	// * This interface only accepts page-size aligned user memory for dma mapping.
	// * Return: 0 on success, -errno on failure.
	FPGAPortDMAMap = 0xB643

	// FPGAPortDMAUnmap IOCTL
	// * DFL_FPGA_PORT_DMA_UNMAP - _IOW(FPGA_MAGIC, PORT_BASE + 4,
	// *						struct dfl_fpga_port_dma_unmap)
	// *
	// * Unmap the dma memory per iova provided by caller.
	// * Return: 0 on success, -errno on failure.
	FPGAPortDMAUnmap = 0xB644

	// IOCTLs for FME file descriptor

	// FPGAFMEPortPR IOCTL
	//  * DFL_FPGA_FME_PORT_PR - _IOW(DFL_FPGA_MAGIC, DFL_FME_BASE + 0,
	//  *						struct dfl_fpga_fme_port_pr)
	//  *
	//  * Driver does Partial Reconfiguration based on Port ID and Buffer (Image)
	//  * provided by caller.
	//  * Return: 0 on success, -errno on failure.
	//  * If DFL_FPGA_FME_PORT_PR returns -EIO, that indicates the HW has detected
	//  * some errors during PR, under this case, the user can fetch HW error info
	//  * from the status of FME's fpga manager.
	FPGAFMEPortPR = 0xB680

	// Flags in dflFPGAPortRegionInfo
	dflPortRegionFlagRead  = (1 << 0) // Region is readable
	dflPortRegionFlagWrite = (1 << 1) // Region is writable
	dflPortRegionFlagMmap  = (1 << 2) // Can be mmaped to userspace
	// Index in dflFPGAPortRegionInfo
	dflPortRegionIndexAFU = 0 // AFU
	dflPortRegionIndexSTP = 1 // Signal Tap

)

type dflFPGAPortInfo struct {
	Argsz      uint32 // Input: Structure length
	Flags      uint32 // Output: Zero for now
	NumRegions uint32 // Output: The number of supported regions
	NumUmsgs   uint32 // Output: The number of allocated umsgs
}

type dflFPGAPortRegionInfo struct {
	Argsz   uint32 // Input: Structure length
	Flags   uint32 // Output: Access permission
	Index   uint32 // Input: Region index
	Padding uint32
	Size    uint64 // Output: Region size (bytes)
	Offset  uint64 // Output: Region offset from start of device fd
}

type dflFPGAPortDMAMap struct {
	Argsz    uint32 // Input: Structure length
	Flags    uint32 // Input: Zero for now
	UserAddr uint64 // Input: Process virtual address
	Length   uint64 // Input: Length of mapping (bytes)
	IOVA     uint64 // Output: IO virtual address
}

type dflFPGAPortDMAUnmap struct {
	Argsz uint32 // Input: Structure length
	Flags uint32 // Input: Zero for now
	IOVA  uint64 // Input: IO virtual address
}

type dflFPGAFMEPortPR struct {
	Argsz         uint32 // Input: Structure length
	Flags         uint32 // Input: Zero for now
	PortID        uint32 // Input
	BufferSize    uint32 // Input
	BufferAddress uint64 // Userspace address to the buffer for PR
}
