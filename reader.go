// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

// Slightly edited, see original repo: https://github.com/kelindar/iostream

package main

import (
	"bytes"
	"io"
)

type source interface {
	io.Reader
	io.ByteReader
	Slice(n int) (buffer []byte, err error)
	GetOffset() int64
	SetOffset(offset int64)
}

type sliceSource struct {
	buffer []byte
	offset int64
}

func newSource(r io.Reader) source {
	switch v := r.(type) {
	case *bytes.Buffer:
		return newSliceSource(v.Bytes())
	}

	return nil
}

func newSliceSource(b []byte) *sliceSource {
	return &sliceSource{b, 0}
}

func (r *sliceSource) GetOffset() int64 {
	return r.offset
}

func (r *sliceSource) SetOffset(offset int64) {
	r.offset = offset
}

func (r *sliceSource) Read(b []byte) (n int, err error) {
	if r.offset >= int64(len(r.buffer)) {
		return 0, io.EOF
	}

	n = copy(b, r.buffer[r.offset:])
	r.offset += int64(n)
	return
}

func (r *sliceSource) ReadByte() (byte, error) {
	if r.offset >= int64(len(r.buffer)) {
		return 0, io.EOF
	}

	b := r.buffer[r.offset]
	r.offset++
	return b, nil
}

func (r *sliceSource) Slice(n int) ([]byte, error) {
	if r.offset+int64(n) > int64(len(r.buffer)) {
		return nil, io.EOF
	}

	cur := r.offset
	r.offset += int64(n)
	return r.buffer[cur:r.offset], nil
}

type Reader struct {
	src source
}

func NewReader(src io.Reader) *Reader {
	if r, ok := src.(*Reader); ok {
		return r
	}

	return &Reader{
		src: newSource(src),
	}
}

func (r *Reader) GetOffset() int64 {
	return r.src.GetOffset()
}

func (r *Reader) SetOffset(offset int64) {
	r.src.SetOffset(offset)
}

func (r *Reader) Read(p []byte) (n int, err error) {
	return r.src.Read(p)
}

func (r *Reader) ReadByte() (out byte, err error) {
	var b []byte
	if b, err = r.src.Slice(1); err == nil {
		out = b[0]
	}
	return
}

func (r *Reader) ReadUint32() (out uint32, err error) {
	var b []byte
	if b, err = r.src.Slice(4); err == nil {
		_ = b[3] // bounds check hint to compiler
		out = (uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
	}
	return
}

func (r *Reader) ReadUint64() (out uint64, err error) {
	var b []byte
	if b, err = r.src.Slice(8); err == nil {
		_ = b[7] // bounds check hint to compiler
		out = (uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
	}
	return
}
