// Copyright ©2015 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package native

import (
	"github.com/openshift/github.com/gonum/blas"
	"github.com/openshift/github.com/gonum/blas/blas64"
)

// Dtrtrs solves a triangular system of the form a * x = b or a^T * x = b. Dtrtrs
// checks for singularity in a. If a is singular, false is returned and no solve
// is performed. True is returned otherwise.
func (impl Implementation) Dtrtrs(uplo blas.Uplo, trans blas.Transpose, diag blas.Diag, n, nrhs int, a []float64, lda int, b []float64, ldb int) (ok bool) {
	nounit := diag == blas.NonUnit
	if n == 0 {
		return false
	}
	// Check for singularity.
	if nounit {
		for i := 0; i < n; i++ {
			if a[i*lda+i] == 0 {
				return false
			}
		}
	}
	bi := blas64.Implementation()
	bi.Dtrsm(blas.Left, uplo, trans, diag, n, nrhs, 1, a, lda, b, ldb)
	return true
}
