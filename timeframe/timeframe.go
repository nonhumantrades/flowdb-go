package timeframe

import (
	"errors"
	"fmt"
	"strconv"
)

// Unit represents the timeframe unit type
type Unit int8

const (
	UnitSecond Unit = iota // Duration-based: raw seconds
	UnitMinute             // Duration-based: minutes
	UnitHour               // Duration-based: hours
	UnitDay                // Duration-based: days
	UnitWeek               // Calendar: Monday 00:00 UTC
	UnitMonth              // Calendar: 1st of month 00:00 UTC
	UnitYear               // Calendar: Jan 1st 00:00 UTC
)

// Approximate seconds for calendar-based units
const (
	WeekSeconds  = 7 * 24 * 60 * 60   // 604800
	MonthSeconds = 30 * 24 * 60 * 60  // 2592000
	YearSeconds  = 365 * 24 * 60 * 60 // 31536000
)

var (
	ErrEmptyTimeframe   = errors.New("empty timeframe string")
	ErrInvalidTimeframe = errors.New("invalid timeframe format")
)

// Timeframe represents a time duration with a specific unit
type Timeframe struct {
	Value int64
	Unit  Unit
}

// Parse parses a timeframe string into a Timeframe struct.
// Supported formats: "30s", "5m", "1h", "1D", "1W", "1M", "1Y", "60" (raw seconds)
func Parse(s string) (Timeframe, error) {
	if len(s) == 0 {
		return Timeframe{}, ErrEmptyTimeframe
	}

	suffix := s[len(s)-1]

	switch suffix {
	case 's':
		v, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitSecond}, nil
	case 'm':
		v, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitMinute}, nil
	case 'h':
		v, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitHour}, nil
	case 'd', 'D':
		v, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitDay}, nil
	case 'W':
		v, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitWeek}, nil
	case 'M':
		v, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitMonth}, nil
	case 'Y':
		v, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitYear}, nil
	default:
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return Timeframe{}, fmt.Errorf("%w: %v", ErrInvalidTimeframe, err)
		}
		return Timeframe{Value: v, Unit: UnitSecond}, nil
	}
}

// String returns the canonical string representation
func (t Timeframe) String() string {
	switch t.Unit {
	case UnitSecond:
		return fmt.Sprintf("%ds", t.Value)
	case UnitMinute:
		return fmt.Sprintf("%dm", t.Value)
	case UnitHour:
		return fmt.Sprintf("%dh", t.Value)
	case UnitDay:
		return fmt.Sprintf("%dD", t.Value)
	case UnitWeek:
		return fmt.Sprintf("%dW", t.Value)
	case UnitMonth:
		return fmt.Sprintf("%dM", t.Value)
	case UnitYear:
		return fmt.Sprintf("%dY", t.Value)
	default:
		return fmt.Sprintf("%ds", t.Value)
	}
}

// Seconds converts the timeframe to seconds.
// For calendar-based units, returns approximate values.
func (t Timeframe) Seconds() int64 {
	switch t.Unit {
	case UnitSecond:
		return t.Value
	case UnitMinute:
		return t.Value * 60
	case UnitHour:
		return t.Value * 3600
	case UnitDay:
		return t.Value * 86400
	case UnitWeek:
		return t.Value * WeekSeconds
	case UnitMonth:
		return t.Value * MonthSeconds
	case UnitYear:
		return t.Value * YearSeconds
	default:
		return t.Value
	}
}

// IsCalendarBased returns true if the timeframe requires calendar-aware alignment.
func (t Timeframe) IsCalendarBased() bool {
	return t.Unit == UnitWeek || t.Unit == UnitMonth || t.Unit == UnitYear
}

// IsZero returns true if the timeframe has a zero value.
func (t Timeframe) IsZero() bool {
	return t.Value == 0
}
