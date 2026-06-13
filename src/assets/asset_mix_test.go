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

func TestAddOneAsset(t *testing.T) {
	tests := []struct {
		name     string
		assetMix AssetMix
		at       Type
		want     AssetMix
	}{
		{
			name:     "Add one renewable",
			assetMix: AssetMix{},
			at:       TypeRenewable,
			want:     AssetMix{Renewables: 1},
		},
		{
			name:     "Add one battery arbitrage",
			assetMix: AssetMix{},
			at:       TypeBattery,
			want:     AssetMix{BatteriesArbitrage: 1},
		},
		{
			name:     "Add one fossil wholesale",
			assetMix: AssetMix{},
			at:       TypeFossil,
			want:     AssetMix{FossilsWholesale: 1},
		},
		{
			name:     "Add one battery existing",
			assetMix: AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 1},
			at:       TypeBattery,
			want:     AssetMix{BatteriesArbitrage: 3, BatteriesCapacity: 1},
		},
		{
			name:     "Add one fossil existing",
			assetMix: AssetMix{FossilsWholesale: 1, FossilsCapacity: 1},
			at:       TypeFossil,
			want:     AssetMix{FossilsWholesale: 2, FossilsCapacity: 1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.assetMix
			got.AddOneAsset(tc.at)
			if got != tc.want {
				t.Errorf("AddOneAsset(%s) on %+v =  %+v, want %+v", tc.at.String(), tc.assetMix, got, tc.want)
			}
		})
	}
}

func TestRemoveOneAsset(t *testing.T) {
	tests := []struct {
		name     string
		assetMix AssetMix
		at       Type
		want     AssetMix
	}{
		{
			name:     "renewable positive",
			assetMix: AssetMix{Renewables: 2},
			at:       TypeRenewable,
			want:     AssetMix{Renewables: 1},
		},
		{
			name:     "renewable zero",
			assetMix: AssetMix{Renewables: 0},
			at:       TypeRenewable,
			want:     AssetMix{Renewables: 0},
		},
		{
			name:     "battery arbitrage",
			assetMix: AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 0},
			at:       TypeBattery,
			want:     AssetMix{BatteriesArbitrage: 1, BatteriesCapacity: 0},
		},
		{
			name:     "battery capacity first",
			assetMix: AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 1},
			at:       TypeBattery,
			want:     AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 0},
		},
		{
			name:     "battery zero",
			assetMix: AssetMix{BatteriesArbitrage: 0, BatteriesCapacity: 0},
			at:       TypeBattery,
			want:     AssetMix{BatteriesArbitrage: 0, BatteriesCapacity: 0},
		},
		{
			name:     "fossil wholesale",
			assetMix: AssetMix{FossilsWholesale: 2, FossilsCapacity: 0},
			at:       TypeFossil,
			want:     AssetMix{FossilsWholesale: 1, FossilsCapacity: 0},
		},
		{
			name:     "fossil capacity first",
			assetMix: AssetMix{FossilsWholesale: 2, FossilsCapacity: 1},
			at:       TypeFossil,
			want:     AssetMix{FossilsWholesale: 2, FossilsCapacity: 0},
		},
		{
			name:     "fossil zero",
			assetMix: AssetMix{FossilsWholesale: 0, FossilsCapacity: 0},
			at:       TypeFossil,
			want:     AssetMix{FossilsWholesale: 0, FossilsCapacity: 0},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.assetMix
			got.RemoveOneAsset(tc.at)
			if got != tc.want {
				t.Errorf("RemoveOneAsset(%s) on %+v =  %+v, want %+v", tc.at.String(), tc.assetMix, got, tc.want)
			}
		})
	}
}

func TestAssetMixClear(t *testing.T) {
	am := AssetMix{
		Renewables:         2,
		BatteriesArbitrage: 3,
		BatteriesCapacity:  1,
		FossilsWholesale:   4,
		FossilsCapacity:    5,
	}
	am.Clear()
	want := AssetMix{}
	if am != want {
		t.Errorf("Clear() = %+v, want %+v", am, want)
	}
}

func TestPledgeOneAsset(t *testing.T) {
	tests := []struct {
		name     string
		assetMix AssetMix
		at       Type
		want     AssetMix
	}{
		{
			name:     "renewable is no op",
			assetMix: AssetMix{Renewables: 1},
			at:       TypeRenewable,
			want:     AssetMix{Renewables: 1},
		},
		{
			name:     "battery with arbitrage",
			assetMix: AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 1},
			at:       TypeBattery,
			want:     AssetMix{BatteriesArbitrage: 1, BatteriesCapacity: 2},
		},
		{
			name:     "battery with no arbitrage",
			assetMix: AssetMix{BatteriesArbitrage: 0, BatteriesCapacity: 1},
			at:       TypeBattery,
			want:     AssetMix{BatteriesArbitrage: 0, BatteriesCapacity: 1},
		},
		{
			name:     "fossil with wholesale",
			assetMix: AssetMix{FossilsWholesale: 2, FossilsCapacity: 1},
			at:       TypeFossil,
			want:     AssetMix{FossilsWholesale: 1, FossilsCapacity: 2},
		},
		{
			name:     "fossil with no wholesale",
			assetMix: AssetMix{FossilsWholesale: 0, FossilsCapacity: 1},
			at:       TypeFossil,
			want:     AssetMix{FossilsWholesale: 0, FossilsCapacity: 1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.assetMix
			got.PledgeOneAsset(tc.at)
			if got != tc.want {
				t.Errorf("PledgeOneAsset(%s) on %+v =  %+v, want %+v", tc.at.String(), tc.assetMix, got, tc.want)
			}
		})
	}
}
func TestCanPledgeOneAsset(t *testing.T) {
	tests := []struct {
		name     string
		assetMix AssetMix
		at       Type
		want     bool
	}{
		{
			name:     "renewable is no op",
			assetMix: AssetMix{Renewables: 1},
			at:       TypeRenewable,
			want:     false,
		},
		{
			name:     "battery with arbitrage",
			assetMix: AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 1},
			at:       TypeBattery,
			want:     true,
		},
		{
			name:     "battery with no arbitrage",
			assetMix: AssetMix{BatteriesArbitrage: 0, BatteriesCapacity: 1},
			at:       TypeBattery,
			want:     false,
		},
		{
			name:     "fossil with wholesale",
			assetMix: AssetMix{FossilsWholesale: 2, FossilsCapacity: 1},
			at:       TypeFossil,
			want:     true,
		},
		{
			name:     "fossil with no wholesale",
			assetMix: AssetMix{FossilsWholesale: 0, FossilsCapacity: 1},
			at:       TypeFossil,
			want:     false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.assetMix.CanPledgeOneAsset(tc.at)
			if got != tc.want {
				t.Errorf("CanPledgeOneAsset(%s) on %+v = %t, want %t", tc.at.String(), tc.assetMix, got, tc.want)
			}
		})
	}
}

func TestResetAllCapacityPledges(t *testing.T) {
	am := AssetMix{
		Renewables:         2,
		BatteriesArbitrage: 3,
		BatteriesCapacity:  1,
		FossilsWholesale:   4,
		FossilsCapacity:    5,
	}
	am.ResetAllCapacityPledges()
	want := AssetMix{
		Renewables:         2,
		BatteriesArbitrage: 4, // 3+1
		BatteriesCapacity:  0,
		FossilsWholesale:   9, // 4+5
		FossilsCapacity:    0,
	}
	if am != want {
		t.Errorf("ResetAllCapacityPledges() = %+v, want %+v", am, want)
	}
}

func TestTakeOneAssetFrom(t *testing.T) {
	tests := []struct {
		name      string
		assetMix  AssetMix
		other     AssetMix
		at        Type
		want      AssetMix
		wantOther AssetMix
	}{
		{
			name:      "renewable available",
			assetMix:  AssetMix{},
			other:     AssetMix{Renewables: 2},
			at:        TypeRenewable,
			want:      AssetMix{Renewables: 1},
			wantOther: AssetMix{Renewables: 1},
		},
		{
			name:      "renewable not available",
			assetMix:  AssetMix{},
			other:     AssetMix{Renewables: 0},
			at:        TypeRenewable,
			want:      AssetMix{},
			wantOther: AssetMix{Renewables: 0},
		},
		{
			name:      "battery available take capacity first",
			assetMix:  AssetMix{},
			other:     AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 1},
			at:        TypeBattery,
			want:      AssetMix{BatteriesArbitrage: 1, BatteriesCapacity: 0},
			wantOther: AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 0},
		},
		{
			name:      "battery available no capacity",
			assetMix:  AssetMix{},
			other:     AssetMix{BatteriesArbitrage: 2, BatteriesCapacity: 0},
			at:        TypeBattery,
			want:      AssetMix{BatteriesArbitrage: 1, BatteriesCapacity: 0},
			wantOther: AssetMix{BatteriesArbitrage: 1, BatteriesCapacity: 0},
		},
		{
			name:      "battery not available",
			assetMix:  AssetMix{},
			other:     AssetMix{BatteriesArbitrage: 0, BatteriesCapacity: 0},
			at:        TypeBattery,
			want:      AssetMix{},
			wantOther: AssetMix{BatteriesArbitrage: 0, BatteriesCapacity: 0},
		},
		{
			name:      "fossil available take capacity first",
			assetMix:  AssetMix{},
			other:     AssetMix{FossilsWholesale: 2, FossilsCapacity: 1},
			at:        TypeFossil,
			want:      AssetMix{FossilsWholesale: 1, FossilsCapacity: 0},
			wantOther: AssetMix{FossilsWholesale: 2, FossilsCapacity: 0},
		},
		{
			name:      "fossil available no capacity",
			assetMix:  AssetMix{},
			other:     AssetMix{FossilsWholesale: 2, FossilsCapacity: 0},
			at:        TypeFossil,
			want:      AssetMix{FossilsWholesale: 1, FossilsCapacity: 0},
			wantOther: AssetMix{FossilsWholesale: 1, FossilsCapacity: 0},
		},
		{
			name:      "fossil not available",
			assetMix:  AssetMix{},
			other:     AssetMix{FossilsWholesale: 0, FossilsCapacity: 0},
			at:        TypeFossil,
			want:      AssetMix{},
			wantOther: AssetMix{FossilsWholesale: 0, FossilsCapacity: 0},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mix := tc.assetMix
			other := tc.other
			mix.TakeOneAssetFrom(tc.at, &other)
			if mix != tc.want {
				t.Errorf("TakeOneAssetFrom(%s) on %+v from %+v =  %+v, want %+v", tc.at.String(), tc.assetMix, tc.other, mix, tc.want)
			}
			if other != tc.wantOther {
				t.Errorf("TakeOneAssetFrom(%s) on %+v from %+v leaves other =  %+v, want %+v", tc.at.String(), tc.assetMix, tc.other, other, tc.wantOther)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	am := AssetMix{
		Renewables:         2,
		BatteriesArbitrage: 3,
		BatteriesCapacity:  1,
		FossilsWholesale:   4,
		FossilsCapacity:    5,
	}
	other := AssetMix{
		Renewables:         1,
		BatteriesArbitrage: 2,
		BatteriesCapacity:  1,
		FossilsWholesale:   1,
		FossilsCapacity:    2,
	}
	am.Add(other)
	want := AssetMix{
		Renewables:         3, // 2+1
		BatteriesArbitrage: 5, // 3+2
		BatteriesCapacity:  2, // 1+1
		FossilsWholesale:   5, // 4+1
		FossilsCapacity:    7, // 5+2
	}
	if am != want {
		t.Errorf("Add(%+v) = %+v, want %+v", other, am, want)
	}
}

func TestTakeAllAssetsFrom(t *testing.T) {
	am := AssetMix{
		Renewables:         2,
		BatteriesArbitrage: 3,
		BatteriesCapacity:  1,
		FossilsWholesale:   4,
		FossilsCapacity:    5,
	}
	other := AssetMix{
		Renewables:         1,
		BatteriesArbitrage: 2,
		BatteriesCapacity:  1,
		FossilsWholesale:   1,
		FossilsCapacity:    2,
	}
	am.TakeAllAssetsFrom(&other)
	want := AssetMix{
		Renewables:         3, // 2+1
		BatteriesArbitrage: 6, // 3+2+1
		BatteriesCapacity:  1, // unchanged
		FossilsWholesale:   7, // 4+1+2
		FossilsCapacity:    5, // unchanged
	}
	if am != want {
		t.Errorf("TakeAllAssetsFrom() = %+v, want %+v", am, want)
	}
	wantOther := AssetMix{} // other should be cleared
	if other != wantOther {
		t.Errorf("TakeAllAssetsFrom() leaves other = %+v, want %+v", other, wantOther)
	}
}
