package timeframe

import (
	"testing"

	"github.com/nonhumantrades/flowdb-go/proto"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		wantVal  int64
		wantUnit Unit
		wantErr  bool
	}{
		{"60", 60, UnitSecond, false},
		{"30s", 30, UnitSecond, false},
		{"5m", 5, UnitMinute, false},
		{"1h", 1, UnitHour, false},
		{"4h", 4, UnitHour, false},
		{"1D", 1, UnitDay, false},
		{"1W", 1, UnitWeek, false},
		{"1M", 1, UnitMonth, false},
		{"3M", 3, UnitMonth, false},
		{"1Y", 1, UnitYear, false},
		{"", 0, UnitSecond, true},
		{"abc", 0, UnitSecond, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil {
				if got.Value != tt.wantVal || got.Unit != tt.wantUnit {
					t.Errorf("Parse(%q) = {%d, %d}, want {%d, %d}", tt.input, got.Value, got.Unit, tt.wantVal, tt.wantUnit)
				}
			}
		})
	}
}

func TestTimeframeSeconds(t *testing.T) {
	tests := []struct {
		tf   Timeframe
		want int64
	}{
		{Timeframe{60, UnitSecond}, 60},
		{Timeframe{5, UnitMinute}, 300},
		{Timeframe{1, UnitHour}, 3600},
		{Timeframe{1, UnitDay}, 86400},
		{Timeframe{1, UnitWeek}, 604800},
		{Timeframe{1, UnitMonth}, 2592000},
		{Timeframe{1, UnitYear}, 31536000},
	}

	for _, tt := range tests {
		got := tt.tf.Seconds()
		if got != tt.want {
			t.Errorf("Timeframe{%d, %d}.Seconds() = %d, want %d", tt.tf.Value, tt.tf.Unit, got, tt.want)
		}
	}
}

func TestIsCalendarBased(t *testing.T) {
	tests := []struct {
		unit Unit
		want bool
	}{
		{UnitSecond, false},
		{UnitMinute, false},
		{UnitHour, false},
		{UnitDay, false},
		{UnitWeek, true},
		{UnitMonth, true},
		{UnitYear, true},
	}

	for _, tt := range tests {
		tf := Timeframe{1, tt.unit}
		if got := tf.IsCalendarBased(); got != tt.want {
			t.Errorf("Timeframe{1, %d}.IsCalendarBased() = %v, want %v", tt.unit, got, tt.want)
		}
	}
}

func TestSnap(t *testing.T) {
	// Wednesday Jan 3, 2024 12:00:00 UTC
	wednesdayNoon := int64(1704283200)
	// Monday Jan 1, 2024 00:00:00 UTC
	monday := int64(1704067200)
	// Feb 15, 2024 12:00:00 UTC
	feb15 := int64(1707998400)
	// Feb 1, 2024 00:00:00 UTC
	feb1 := int64(1706745600)
	// Jan 1, 2024 00:00:00 UTC
	jan1 := int64(1704067200)

	tests := []struct {
		name  string
		tf    Timeframe
		input int64
		want  int64
	}{
		{"60s snap", Timeframe{60, UnitSecond}, 1704283225, 1704283200},
		{"1W snap wednesday to monday", Timeframe{1, UnitWeek}, wednesdayNoon, monday},
		{"1M snap feb15 to feb1", Timeframe{1, UnitMonth}, feb15, feb1},
		{"1Y snap feb15 to jan1", Timeframe{1, UnitYear}, feb15, jan1},
		{"3M snap feb15 to jan1 (Q1)", Timeframe{3, UnitMonth}, feb15, jan1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tf.Snap(tt.input)
			if got != tt.want {
				t.Errorf("Snap(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNextPeriod(t *testing.T) {
	// Monday Jan 1, 2024 00:00:00 UTC
	jan1 := int64(1704067200)
	// Monday Jan 8, 2024 00:00:00 UTC
	jan8 := int64(1704672000)
	// Feb 1, 2024 00:00:00 UTC
	feb1 := int64(1706745600)
	// Jan 1, 2025 00:00:00 UTC
	jan1_2025 := int64(1735689600)

	tests := []struct {
		name  string
		tf    Timeframe
		input int64
		want  int64
	}{
		{"60s next", Timeframe{60, UnitSecond}, jan1, jan1 + 60},
		{"1W next", Timeframe{1, UnitWeek}, jan1, jan8},
		{"1M next", Timeframe{1, UnitMonth}, jan1, feb1},
		{"1Y next", Timeframe{1, UnitYear}, jan1, jan1_2025},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tf.NextPeriod(tt.input)
			if got != tt.want {
				t.Errorf("NextPeriod(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFromProto(t *testing.T) {
	second := proto.TimeframeUnit_UNIT_SECOND
	week := proto.TimeframeUnit_UNIT_WEEK
	month := proto.TimeframeUnit_UNIT_MONTH

	tests := []struct {
		name     string
		opts     *proto.AggregationOptions
		wantVal  int64
		wantUnit Unit
	}{
		{"nil opts", nil, 0, UnitSecond},
		{"60s duration", &proto.AggregationOptions{TimeBucket: proto.Uint64(60_000_000_000), Unit: &second}, 60, UnitSecond},
		{"1W calendar", &proto.AggregationOptions{TimeBucket: proto.Uint64(1), Unit: &week}, 1, UnitWeek},
		{"3M calendar", &proto.AggregationOptions{TimeBucket: proto.Uint64(3), Unit: &month}, 3, UnitMonth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromProto(tt.opts)
			if got.Value != tt.wantVal || got.Unit != tt.wantUnit {
				t.Errorf("FromProto() = {%d, %d}, want {%d, %d}", got.Value, got.Unit, tt.wantVal, tt.wantUnit)
			}
		})
	}
}

func TestToProto(t *testing.T) {
	tests := []struct {
		name       string
		tf         Timeframe
		wantBucket uint64
		wantUnit   proto.TimeframeUnit
	}{
		{"60s", Timeframe{60, UnitSecond}, 60_000_000_000, proto.TimeframeUnit_UNIT_SECOND},
		{"1W", Timeframe{1, UnitWeek}, 1, proto.TimeframeUnit_UNIT_WEEK},
		{"3M", Timeframe{3, UnitMonth}, 3, proto.TimeframeUnit_UNIT_MONTH},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tf.ToProto()
			if got.GetTimeBucket() != tt.wantBucket {
				t.Errorf("ToProto().TimeBucket = %d, want %d", got.GetTimeBucket(), tt.wantBucket)
			}
			if got.GetUnit() != tt.wantUnit {
				t.Errorf("ToProto().Unit = %v, want %v", got.GetUnit(), tt.wantUnit)
			}
		})
	}
}
