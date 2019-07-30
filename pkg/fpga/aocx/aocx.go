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

package aocx

import (
	"bytes"
	"compress/gzip"
	"debug/elf"
	"io"
	"io/ioutil"
	"os"

	"github.com/intel/intel-device-plugins-for-kubernetes/pkg/fpga/gbs"
	"github.com/pkg/errors"
)

// A File represents an open GBS file.
type File struct {
	AutoDiscovery          string
	AutoDiscoveryXML       string
	Board                  string
	BoardPackage           string
	BoardSpecXML           string
	CompilationEnvironment string
	Hash                   string
	KernelArgInfoXML       string
	QuartusInputHash       string
	QuartusReport          string
	Target                 string
	Version                string
	GBS                    *gbs.File
	closer                 io.Closer
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

func setSection(f *File, section *elf.Section) error {
	name := section.SectionHeader.Name
	if name == ".acl.fpga.bin" {
		data, err := section.Data()
		if err != nil {
			return errors.Wrap(err, "unable to read .acl.fpga.bin")
		}
		f.GBS, err = parseFpgaBin(data)
		if err != nil {
			return errors.Wrap(err, "unable to parse gbs")
		}
		return nil
	}

	fieldMap := map[string]*string{
		".acl.autodiscovery":       &f.AutoDiscovery,
		".acl.autodiscovery.xml":   &f.AutoDiscoveryXML,
		".acl.board":               &f.Board,
		".acl.board_package":       &f.BoardPackage,
		".acl.board_spec.xml":      &f.BoardSpecXML,
		".acl.compilation_env":     &f.CompilationEnvironment,
		".acl.rand_hash":           &f.Hash,
		".acl.kernel_arg_info.xml": &f.KernelArgInfoXML,
		".acl.quartus_input_hash":  &f.QuartusInputHash,
		".acl.quartus_report":      &f.QuartusReport,
		".acl.target":              &f.Target,
		".acl.version":             &f.Version,
	}

	if field, ok := fieldMap[name]; ok {
		data, err := section.Data()
		if err != nil {
			return errors.Wrapf(err, "%s: unable to get section data", name)
		}
		*field = string(data)
	}
	return nil
}

// NewFile creates a new File for accessing an ELF binary in an underlying reader.
// The ELF binary is expected to start at position 0 in the ReaderAt.
func NewFile(r io.ReaderAt) (*File, error) {
	el, err := elf.NewFile(r)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read header")
	}
	f := new(File)
	for _, section := range el.Sections {
		err = setSection(f, section)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func parseFpgaBin(d []byte) (*gbs.File, error) {
	gb, err := elf.NewFile(bytes.NewReader(d))
	gz := gb.Section(".acl.gbs.gz")
	if gz == nil {
		return nil, errors.New("no .acl.gbs.gz section in .acl.fgpa.bin")
	}
	gzr, err := gzip.NewReader(gz.Open())
	if err != nil {
		return nil, errors.Wrap(err, "unable to open gzip reader for .acl.gbs.gz")
	}
	b, err := ioutil.ReadAll(gzr)
	if err != nil {
		return nil, errors.Wrap(err, "unable to uncompress .acl.gbs.gz")
	}
	return gbs.NewFile(bytes.NewReader(b))
}