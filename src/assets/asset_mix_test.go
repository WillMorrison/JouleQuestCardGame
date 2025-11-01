package assets

import (
	"testing"
)

func TestCoefficientsString(t *testing.T) {
	tests := []struct {
		name         string
		coefficients AssetMixCoefficients
		want         string
	}{
		{
			name:         "All Zero Coefficients",
			coefficients: AssetMixCoefficients{},
			want:         "0",
		},
		{
			name:         "Single Positive Coefficient",
			coefficients: AssetMixCoefficients{Renewables: 3},
			want:         "3*Renewables",
		},
		{
			name:         "Single Negative Coefficient",
			coefficients: AssetMixCoefficients{FossilsCapacity: -2},
			want:         "-2*FossilsCapacity",
		},
		{
			name:         "Some Positive and Negative Coefficients",
			coefficients: AssetMixCoefficients{Renewables: 2, FossilsCapacity: -3},
			want:         "2*Renewables + -3*FossilsCapacity",
		},
		{
			name:         "Plus/Minus One Coefficients",
			coefficients: AssetMixCoefficients{BatteriesArbitrage: 5, BatteriesCapacity: 1, FossilsWholesale: -1},
			want:         "5*BatteriesArbitrage + BatteriesCapacity + -FossilsWholesale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.coefficients.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameCalculation(t *testing.T) {
	resultMap := [4]string{"A>>B", "A>=B", "A<B", "A<<B"}
	tests := []struct {
		name        string
		calculation RatioCalculation
		assetMix    AssetMix
		want        string
	}{
		{
			name: "Zero Coefficients treated as A=B",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{},
				CoefficientsB: AssetMixCoefficients{},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:         1,
				BatteriesArbitrage: 2,
				BatteriesCapacity:  3,
				FossilsWholesale:   4,
				FossilsCapacity:    5,
			},
			want: "A>=B",
		},
		{
			name: "A >> B Simple",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{Renewables: 1},
				CoefficientsB: AssetMixCoefficients{FossilsCapacity: 1},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:      10,
				FossilsCapacity: 5,
			},
			want: "A>>B",
		},
		{
			name: "A << B Simple",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{Renewables: 1},
				CoefficientsB: AssetMixCoefficients{FossilsCapacity: 1},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:      4,
				FossilsCapacity: 8,
			},
			want: "A<<B",
		},
		{
			name: "A < B Simple",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{Renewables: 1},
				CoefficientsB: AssetMixCoefficients{FossilsCapacity: 1},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:      5,
				FossilsCapacity: 7,
			},
			want: "A<B",
		},
		{
			name: "A >= B Simple",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{Renewables: 1},
				CoefficientsB: AssetMixCoefficients{FossilsCapacity: 1},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:      8,
				FossilsCapacity: 7,
			},
			want: "A>=B",
		},
		{
			name: "Coefficient sums",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{Renewables: 4, FossilsCapacity: 1},
				CoefficientsB: AssetMixCoefficients{FossilsCapacity: 1, Renewables: 2},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:      5,
				FossilsCapacity: 7,
			},
			want: "A>=B", // 4*5+1*7=27 vs 1*7+2*5=17
		},
		{
			name: "Negative coefficients",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{Renewables: 1, FossilsCapacity: -1},
				CoefficientsB: AssetMixCoefficients{FossilsCapacity: 1, Renewables: -1},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:      5,
				FossilsCapacity: 7,
			},
			want: "A<<B", // 5-7=-2(0) vs 7-5=2
		},
		{
			name: "Min side is 0",
			calculation: RatioCalculation{
				CoefficientsA: AssetMixCoefficients{Renewables: -1},
				CoefficientsB: AssetMixCoefficients{FossilsCapacity: -1},
				Rollover:      2,
			},
			assetMix: AssetMix{
				Renewables:      1,
				FossilsCapacity: 7,
			},
			want: "A>=B", // would be -1 vs -7, but min is 0 vs 0
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MapRatioTo(tc.calculation, tc.assetMix, resultMap)
			if got != tc.want {
				t.Errorf("%s on %+v =  %q, want %q", tc.calculation.String(), tc.assetMix, got, tc.want)
			}
		})
	}
}
