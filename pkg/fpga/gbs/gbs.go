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

package gbs

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
)

const (
	bitstreamGUID1   uint64 = 0x414750466e6f6558
	bitstreamGUID2   uint64 = 0x31303076534247b7
	fileHeaderLength        = 20
)

// BitstreamHeader represents header struct of the GBS file
type BitstreamHeader struct {
	GUID1          uint64
	GUID2          uint64
	MetadataLength uint32
}

// A File represents an open GBS file.
type File struct {
	BitstreamHeader
	Metadata  BitstreamMetadata
	Bitstream *Bitstream
	closer    io.Closer
}

// BitstreamMetadata represents parsed JSON metadata of GBS file
type BitstreamMetadata struct {
	Version      int    `json:"version"`
	PlatformName string `json:"platform-name,omitempty"`
	AfuImage     struct {
		MagicNo         int    `json:"magic-no,omitempty"`
		InterfaceUUID   string `json:"interface-uuid,omitempty"`
		AfuTopInterface struct {
			Class       string `json:"class"`
			ModulePorts []struct {
				Params struct {
					Clock string `json:"clock,omitempty"`
				} `json:"params"`
				Optional bool   `json:"optional,omitempty"`
				Class    string `json:"class,omitempty"`
			} `json:"module-ports,omitempty"`
		} `json:"afu-top-interface"`
		Power               int         `json:"power"`
		ClockFrequencyHigh  interface{} `json:"clock-frequency-high,omitempty"`
		ClockFrequencyLow   interface{} `json:"clock-frequency-low,omitempty"`
		AcceleratorClusters []struct {
			AcceleratorTypeUUID string `json:"accelerator-type-uuid"`
			Name                string `json:"name"`
			TotalContexts       int    `json:"total-contexts"`
		} `json:"accelerator-clusters"`
	} `json:"afu-image"`
}

// A Bitstream represents a raw bitsream data (RBF) in the GBS binary
type Bitstream struct {
	Size uint64
	// Embed ReaderAt for ReadAt method.
	// Do not embed SectionReader directly
	// to avoid having Read and Seek.
	// If a client wants Read and Seek it must use
	// Open() to avoid fighting over the seek offset
	// with other clients.
	io.ReaderAt
	sr *io.SectionReader
}

// Open returns a new ReadSeeker reading the bitsream body.
func (b *Bitstream) Open() io.ReadSeeker { return io.NewSectionReader(b.sr, 0, 1<<63-1) }

// Data reads and returns the contents of the bitstream.
func (b *Bitstream) Data() ([]byte, error) {
	dat := make([]byte, b.Size)
	n, err := io.ReadFull(b.Open(), dat)
	return dat[0:n], err
}

// Open opens the named file using os.Open and prepares it for use as GBS.
func Open(name string) (*File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ff, err := NewFile(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	ff.closer = f
	return ff, nil
}

// Close closes the File.
// If the File was created using NewFile directly instead of Open,
// Close has no effect.
func (f *File) Close() error {
	var err error
	if f.closer != nil {
		err = f.closer.Close()
		f.closer = nil
	}
	return err
}

// We need both Seek and ReadAt
type bitstreamReader interface {
	io.ReadSeeker
	io.ReaderAt
}

// NewFile creates a new File for accessing an ELF binary in an underlying reader.
// The ELF binary is expected to start at position 0 in the ReaderAt.
func NewFile(r bitstreamReader) (*File, error) {
	sr := io.NewSectionReader(r, 0, 1<<63-1)

	f := new(File)
	// TODO:
	// 1. Read file header
	sr.Seek(0, io.SeekStart)
	if err := binary.Read(sr, binary.LittleEndian, &f.BitstreamHeader); err != nil {
		return nil, errors.Wrap(err, "unable to read header")
	}
	// 2. Validate Magic/GUIDs
	if f.GUID1 != bitstreamGUID1 || f.GUID2 != bitstreamGUID2 {
		return nil, errors.Errorf("Wrong magic in GBS file: %#x %#x Expected %#x %#x", f.GUID1, f.GUID2, bitstreamGUID1, bitstreamGUID2)
	}
	// 3. Read/unmarshal metadata JSON
	if f.MetadataLength == 0 || f.MetadataLength >= 4096 {
		return nil, errors.Errorf("Incorrect length of GBS metadata %d", f.MetadataLength)
	}
	dec := json.NewDecoder(io.NewSectionReader(r, fileHeaderLength, int64(f.MetadataLength)))
	if err := dec.Decode(&f.Metadata); err != nil {
		return nil, errors.Wrap(err, "unable to parse GBS metadata")
	}
	// 4. Create bitsream struct
	b := new(Bitstream)
	// 4.1. calculate offest/size
	last, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine file size")
	}
	b.Size = uint64(last - fileHeaderLength - int64(f.MetadataLength))
	// 4.2. assign internal sr
	b.sr = io.NewSectionReader(r, int64(fileHeaderLength+f.MetadataLength), int64(b.Size))
	b.ReaderAt = b.sr
	f.Bitstream = b
	return f, nil
}
