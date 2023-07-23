package utils

import (
	"fmt"
	"math"
	"math/bits"
	"math/cmplx"
	"os"
	"path/filepath"
)

func NormalizePath(path string) (string, error) {
	// 1. transform to an absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to transform(%s) to an absolute path: %w", path, err)
	}

	// 2. check if the path is valid
	cleanPath := filepath.Clean(absPath)
	if _, err := os.Stat(cleanPath); err != nil {
		return "", fmt.Errorf("failed to get file info(%s): %w", cleanPath, err)
	}

	return cleanPath, nil
}

// FFT is the fast Fourier transform
func FFT(x []complex128) []complex128 {
	N := len(x)
	X := make([]complex128, N)
	for k := 0; k < N; k++ {
		for n := 0; n < N; n++ {
			X[k] += x[n] * cmplx.Exp(-2i*math.Pi*complex(float64(k*n)/float64(N), 0))
		}
	}
	return X
}

// IFFT is the inverse of FFT (iFFT)
func IFFT(x []complex128) []complex128 {
	N := len(x)
	X := make([]complex128, N)
	for k := 0; k < N; k++ {
		for n := 0; n < N; n++ {
			X[k] += x[n] * cmplx.Exp(2i*math.Pi*complex(float64(k*n)/float64(N), 0))
		}
		X[k] /= complex(float64(N), 0)
	}
	return X
}

// BitReverse is the bit-reversal permutation
func BitReverse(x []complex128) []complex128 {
	N := len(x)
	X := make([]complex128, N)
	for k := 0; k < N; k++ {
		b := bits.Reverse(uint(k)) >> (bits.UintSize - uint(math.Log2(float64(N))))
		X[b] = x[k]
	}
	return X
}

// Radix2FFT is the fast Fourier transform for radix-2
func Radix2FFT(x []complex128) []complex128 {
	N := len(x)
	if N&(N-1) != 0 {
		panic("the length of x must be power of 2")
	}
	X := BitReverse(x)
	var s uint
	for s = 1; s <= uint(math.Log2(float64(N))); s++ {
		m := 1 << s
		wm := cmplx.Exp(-2i * math.Pi / complex(float64(m), 0))
		for k := 0; k < N; k += m {
			w := 1 + 0i
			for j := 0; j < m/2; j++ {
				t := w * X[k+j+m/2]
				u := X[k+j]
				X[k+j] = u + t
				X[k+j+m/2] = u - t
				w *= wm
			}
		}
	}
	return X
}

// IRadix2FFT is the inverse of Radix2FFT (iRadix2FFT)
func IRadix2FFT(x []complex128) []complex128 {
	N := len(x)
	if N&(N-1) != 0 {
		panic("the length of x must be power of 2")
	}
	X := BitReverse(x)
	var s uint
	for s = 1; s <= uint(math.Log2(float64(N))); s++ {
		m := 1 << s
		wm := cmplx.Exp(2i * math.Pi / complex(float64(m), 0))
		for k := 0; k < N; k += m {
			w := 1 + 0i
			for j := 0; j < m/2; j++ {
				t := w * X[k+j+m/2]
				u := X[k+j]
				X[k+j] = (u + t) / 2
				X[k+j+m/2] = (u - t) / 2
				w *= wm
			}
		}
	}
	return X
}

func IsComplexEqual(x, y []complex128) bool {
	return IsComplexEqualWithNBit(x, y, 10)
}

func IsComplexEqualWithNBit(x, y []complex128, n int) bool {
	if len(x) != len(y) {
		return false
	}
	if n < 10 {
		n = 10
	}
	for i := 0; i < len(x); i++ {
		// n means 10^-n
		if cmplx.Abs(x[i]-y[i]) > math.Pow10(-n) {
			return false
		}
	}
	return true
}
