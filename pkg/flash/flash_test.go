// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flash

import (
	"errors"
	"fmt"
	"testing"

	"github.com/u-root/u-root/pkg/flash/spimock"
)

// TestSFDPReader tests reading arbitrary offsets from the SFDP.
func TestSFDPReader(t *testing.T) {
	for _, tt := range []struct {
		name             string
		readOffset       int64
		readSize         int
		forceTransferErr error
		wantData         []byte
		wantErr          error
	}{
		{
			name:       "read sfdp data",
			readOffset: 0x10,
			readSize:   4,
			wantData:   []byte{0xc2, 0x00, 0x01, 0x04},
		},
		{
			name:       "invalid offset",
			readOffset: sfdpMaxAddress + 1,
			readSize:   4,
			wantErr:    &SFDPAddressError{sfdpMaxAddress + 1},
		},
		{
			name:             "transfer error",
			readOffset:       0x10,
			readSize:         4,
			forceTransferErr: errors.New("fake transfer error"),
			wantErr:          errors.New("fake transfer error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := spimock.New()
			s.ForceTransferErr = tt.forceTransferErr
			f := &Flash{SPI: s}

			data := make([]byte, tt.readSize)
			n, err := f.SFDPReader().ReadAt(data, tt.readOffset)
			if gotErrString, wantErrString := fmt.Sprint(err), fmt.Sprint(tt.wantErr); gotErrString != wantErrString {
				t.Errorf("SFDPReader().ReadAt() err = %q; want %q", gotErrString, wantErrString)
			}
			if err == nil && n != len(data) {
				t.Errorf("SFDPReader().ReadAt() n = %d; want %d", n, len(data))
			}

			if err == nil && string(data) != string(tt.wantData) {
				t.Errorf("SFDPReader().ReadAt() data = %#02x; want %#02x", data, tt.wantData)
			}
		})
	}
}

// TestSFDPReadDWORD checks a DWORD can be parsed from the SFDP tables.
func TestSFDPReadDWORD(t *testing.T) {
	s := spimock.New()
	f := &Flash{SPI: s}

	sfdp, err := f.SFDP()
	if err != nil {
		t.Fatal(err)
	}

	dword, err := sfdp.Dword(0, 0)
	if err != nil {
		t.Error(err)
	}
	var want uint32 = 0xfff320e5
	if dword != want {
		t.Errorf("sfdp.TableDword() = %#08x; want %#08x", dword, want)
	}
}

// TestSFDPCaching checks that the SFDP is properly cached.
func TestReadSFDPDCache(t *testing.T) {
	for _, tt := range []struct {
		name     string
		forceErr error
		wantErr  error
	}{
		{
			name: "cache sfdp",
		},
		{
			name:     "cache err",
			forceErr: errors.New("fake transfer error"),
			wantErr:  errors.New("fake transfer error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := spimock.New()
			s.ForceTransferErr = tt.forceErr
			f := &Flash{SPI: s}
			if len(s.Transfers) != 0 {
				t.Fatalf("Expected 0 transfers; got %d", len(s.Transfers))
			}

			// Read the first time.
			_, err := f.SFDP()
			if gotErrString, wantErrString := fmt.Sprint(err), fmt.Sprint(tt.wantErr); gotErrString != wantErrString {
				t.Errorf("SFDP() err = %q; want %q", gotErrString, wantErrString)
			}
			s.ForceTransferErr = nil   // error should be cached now
			expect := len(s.Transfers) // No more transfers should happen due to cache.

			// Read the second time.
			_, err = f.SFDP()
			if gotErrString, wantErrString := fmt.Sprint(err), fmt.Sprint(tt.wantErr); gotErrString != wantErrString {
				t.Errorf("SFDP() err = %q; want %q", gotErrString, wantErrString)
			}
			// The second read of SFDP should be cached, so transfers does not change.
			if len(s.Transfers) != expect {
				t.Fatalf("Expected %d transfers; got %d", expect, len(s.Transfers))
			}
		})
	}
}
