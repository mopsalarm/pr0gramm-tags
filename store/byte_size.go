// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import "fmt"

type ByteSize float64

const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

func (b ByteSize) String() string {
	switch {
	case b >= YB:
		return fmt.Sprintf("%.2fyb", b/YB)
	case b >= ZB:
		return fmt.Sprintf("%.2fzb", b/ZB)
	case b >= EB:
		return fmt.Sprintf("%.2feb", b/EB)
	case b >= PB:
		return fmt.Sprintf("%.2fpt", b/PB)
	case b >= TB:
		return fmt.Sprintf("%.2ftb", b/TB)
	case b >= GB:
		return fmt.Sprintf("%.2fgb", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.2fmb", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.2fkb", b/KB)
	}
	return fmt.Sprintf("%.2fb", b)
}
