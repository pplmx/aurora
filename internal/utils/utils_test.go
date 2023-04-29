package utils

import (
	"testing"
)

func BenchmarkRecursiveFFT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecursiveFFT([]complex128{1, 2, 3, 4, 5, 6, 7, 8})
	}
}

func BenchmarkIterativeFFT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IterativeFFT([]complex128{1, 2, 3, 4, 5, 6, 7, 8})
	}
}

func BenchmarkRadix2FFT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Radix2FFT([]complex128{1, 2, 3, 4, 5, 6, 7, 8})
	}
}

func BenchmarkIRecursiveFFT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IRecursiveFFT([]complex128{1, 2, 3, 4, 5, 6, 7, 8})
	}
}

func BenchmarkIRadix2FFT(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IRadix2FFT([]complex128{1, 2, 3, 4, 5, 6, 7, 8})
	}
}

func TestDFT(t *testing.T) {
	type args struct {
		x []complex128
	}
	tests := []struct {
		name string
		args args
		want []complex128
	}{
		{
			name: "TestDFT_1",
			args: args{
				x: []complex128{1, 2, 3, 4},
			},
			want: []complex128{10, -2 + 2i, -2, -2 - 2i},
		},
		{
			name: "TestDFT_2",
			args: args{
				x: []complex128{1, 2, 3, 4, 5, 6, 7, 8},
			},
			want: []complex128{36, -4 + 9.65685424949238i, -4 + 4i, -4 + 1.6568542494923806i, -4, -4 - 1.6568542494923806i, -4 - 4i, -4 - 9.65685424949238i},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DFT(tt.args.x); !IsComplexEqual(got, tt.want) {
				t.Errorf("DFT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIDFT(t *testing.T) {
	type args struct {
		x []complex128
	}
	tests := []struct {
		name string
		args args
		want []complex128
	}{
		{
			name: "TestIDFT_1",
			args: args{
				x: []complex128{10, -2 + 2i, -2, -2 - 2i},
			},
			want: []complex128{1, 2, 3, 4},
		},
		{
			name: "TestIDFT_2",
			args: args{
				x: []complex128{36, -4 + 9.65685424949238i, -4 + 4i, -4 + 1.6568542494923806i, -4, -4 - 1.6568542494923806i, -4 - 4i, -4 - 9.65685424949238i},
			},
			want: []complex128{1, 2, 3, 4, 5, 6, 7, 8},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IDFT(tt.args.x); !IsComplexEqual(got, tt.want) {
				t.Errorf("IDFT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFFT(t *testing.T) {
	type args struct {
		x []complex128
	}
	tests := []struct {
		name string
		args args
		want []complex128
	}{
		{
			name: "TestFFT_1",
			args: args{
				x: []complex128{1, 2, 3, 4},
			},
			want: []complex128{10, -2 + 2i, -2, -2 - 2i},
		},
		{
			name: "TestFFT_2",
			args: args{
				x: []complex128{1, 2, 3, 4, 5, 6, 7, 8},
			},
			want: []complex128{36, -4 + 9.65685424949238i, -4 + 4i, -4 + 1.6568542494923806i, -4, -4 - 1.6568542494923806i, -4 - 4i, -4 - 9.65685424949238i},
		},
	}
	for _, tt := range tests {
		// test RecursiveFFT, IterativeFFT, Radix2FFT in parallel
		t.Run(tt.name, func(t *testing.T) {
			if got := RecursiveFFT(tt.args.x); !IsComplexEqual(got, tt.want) {
				t.Errorf("\nFFT() = %v, \nwant %v", got, tt.want)
			}
		})
		t.Run(tt.name, func(t *testing.T) {
			if got := IterativeFFT(tt.args.x); !IsComplexEqual(got, tt.want) {
				t.Errorf("\nRadix2FFT() = %v, \nwant %v", got, tt.want)
			}
		})
		t.Run(tt.name, func(t *testing.T) {
			if got := Radix2FFT(tt.args.x); !IsComplexEqual(got, tt.want) {
				t.Errorf("\nRadix2FFT() = %v, \nwant %v", got, tt.want)
			}
		})
	}
}

func TestIFFT(t *testing.T) {
	type args struct {
		x []complex128
	}
	tests := []struct {
		name string
		args args
		want []complex128
	}{
		{
			name: "TestIFFT_1",
			args: args{
				x: []complex128{10, -2 + 2i, -2, -2 - 2i},
			},
			want: []complex128{1, 2, 3, 4},
		},
		{
			name: "TestIFFT_2",
			args: args{
				x: []complex128{36, -4 + 9.65685424949238i, -4 + 4i, -4 + 1.6568542494923806i, -4, -4 - 1.6568542494923806i, -4 - 4i, -4 - 9.65685424949238i},
			},
			want: []complex128{1, 2, 3, 4, 5, 6, 7, 8},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IRecursiveFFT(tt.args.x); !IsComplexEqual(got, tt.want) {
				t.Errorf("IRecursiveFFT() = %v, want %v", got, tt.want)
			}
		})
		t.Run(tt.name, func(t *testing.T) {
			if got := IRadix2FFT(tt.args.x); !IsComplexEqual(got, tt.want) {
				t.Errorf("IterativeIFFT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsComplexEqual(t *testing.T) {
	type args struct {
		x []complex128
		y []complex128
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestIsComplexEqual_1",
			args: args{
				x: []complex128{1, 2, 3, 4},
				y: []complex128{0.99999999999, 2, 3, 4.00000000001},
			},
			want: true,
		},
		{
			name: "TestIsComplexEqual_2",
			args: args{
				x: []complex128{1, 2, 3, 4},
				y: []complex128{0.999999999, 2, 3, 4.000000001},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsComplexEqual(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("IsComplexEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsComplexEqualWithNBit(t *testing.T) {
	type args struct {
		x []complex128
		y []complex128
		n int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestIsComplexEqualWithNBit_1",
			args: args{
				x: []complex128{1, 2, 3, 4},
				y: []complex128{0.999999999999, 2, 3, 4.000000000001},
				n: 15,
			},
			want: false,
		},
		{
			name: "TestIsComplexEqualWithNBit_2",
			args: args{
				x: []complex128{1, 2, 3, 4},
				y: []complex128{0.999999999999, 2, 3, 4.000000000001},
				n: 11,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsComplexEqualWithNBit(tt.args.x, tt.args.y, tt.args.n); got != tt.want {
				t.Errorf("IsComplexEqualWithNBit() = %v, want %v", got, tt.want)
			}
		})
	}
}
