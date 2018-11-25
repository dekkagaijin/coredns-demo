// Copyright (c) 2018 Jacob Sanders, Michael Grosser
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	crlf    = "\r\n" // CSV line breaks per https://tools.ietf.org/html/rfc4180
	header  = "hostname,latency_micros,lookup_error" + crlf
	noError = "nil"
)

type StatsFile struct {
	*os.File
}

func CreateStatsFile(file string) (*StatsFile, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return nil, err
	}
	f, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	// Write the column headers.
	if _, err = fmt.Fprint(f, header); err != nil {
		return nil, err
	}
	return &StatsFile{f}, nil
}

func (s *StatsFile) Emit(hostname string, latency time.Duration, lookupErr error) error {
	if s == nil {
		return nil
	}
	errStr := noError
	if lookupErr != nil {
		// Per RFC 4180, strings containing double-quotes should themselves be double-quoted and each double-quote should be escaped with a second double-quote.
		errStr = strings.Replace(lookupErr.Error(), `"`, `""`, -1)
	}
	_, err := fmt.Fprintf(s, "%s,%d,%q%s", hostname, latency/1e3, errStr, crlf)
	return err
}
