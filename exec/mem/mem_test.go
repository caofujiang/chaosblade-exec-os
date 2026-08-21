/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mem

import "testing"

// TestBlockCountForMB verifies the MB->block conversion for 128KB blocks (non big-mem mode):
// 8 blocks equal exactly 1MB, so the conversion is exact.
func TestBlockCountForMB(t *testing.T) {
	tests := []struct {
		name string
		mb   int64
		want int
	}{
		{"zero returns zero", 0, 0},
		{"negative returns zero", -10, 0},
		{"1MB=8 blocks", 1, 8},
		{"100MB=800 blocks", 100, 800},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blockCountForMB(tt.mb)
			if got != tt.want {
				t.Fatalf("blockCountForMB(%d) = %d, want %d", tt.mb, got, tt.want)
			}
			// Each Block is 128KB; the converted capacity should equal the target MB exactly.
			if tt.mb > 0 {
				gotMB := int64(got) * 128 / 1024
				if gotMB != tt.mb {
					t.Fatalf("blockCountForMB(%d) actual %dMB, want exactly %dMB", tt.mb, gotMB, tt.mb)
				}
			}
		})
	}
}

// TestBlocksCountForMB verifies the MB->block conversion for 48MB blocks (big-mem mode):
// rounds up, at least 1 block for a positive target, and the allocated amount falls within
// the [mb, mb+48) range.
func TestBlocksCountForMB(t *testing.T) {
	const blockMB = 48
	tests := []struct {
		name string
		mb   int64
		want int
	}{
		{"zero returns zero", 0, 0},
		{"negative returns zero", -5, 0},
		{"1MB rounds up to 1 block", 1, 1},
		{"48MB=1 block", 48, 1},
		{"49MB=2 blocks", 49, 2},
		{"100MB=3 blocks", 100, 3},
		{"100000MB=2084 blocks", 100000, 2084},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blocksCountForMB(tt.mb)
			if got != tt.want {
				t.Fatalf("blocksCountForMB(%d) = %d, want %d", tt.mb, got, tt.want)
			}
			if tt.mb > 0 {
				allocMB := int64(got) * blockMB
				if allocMB < tt.mb || allocMB >= tt.mb+blockMB {
					t.Fatalf("blocksCountForMB(%d) allocates %dMB, outside round-up range [%d, %d)",
						tt.mb, allocMB, tt.mb, tt.mb+blockMB)
				}
			}
		})
	}
}

// TestBlocksCountNoAmplification is a regression test: the big-mem branch previously reused the
// 8*fillMem factor designed for 128KB blocks (see the big-mem branch in mem.go start), causing
// 48MB blocks to be amplified 384x, which defeated the rate/mem-percent limits and instantly
// exhausted physical memory. This ensures the conversion no longer amplifies severalfold.
func TestBlocksCountNoAmplification(t *testing.T) {
	const fillMemMB = 100000 // about 97.7GB, the single-tick cap for rate=100000
	const blockMB = 48

	allocMB := int64(blocksCountForMB(fillMemMB)) * blockMB

	// Buggy value: 8*fillMem blocks x 48MB = 384x = 36.6TB
	buggyAllocMB := int64(8*fillMemMB) * blockMB
	if allocMB >= buggyAllocMB/10 {
		t.Fatalf("suspected 384x amplification regression: allocates %dMB, close to buggy value %dMB", allocMB, buggyAllocMB)
	}
	// Actual allocation should stay on the same order as the target (within one block granularity).
	if allocMB > fillMemMB+blockMB {
		t.Fatalf("allocates %dMB, clearly exceeds target %dMB", allocMB, fillMemMB)
	}
}
