package timeframe

import "github.com/nonhumantrades/flowdb-go/proto"

// FromProto creates a Timeframe from proto AggregationOptions.
func FromProto(opts *proto.AggregationOptions) Timeframe {
	if opts == nil {
		return Timeframe{}
	}

	unit := protoUnitToUnit(opts.GetUnit())
	value := int64(opts.GetTimeBucket())

	// Duration-based: value is nanoseconds, convert to seconds
	if !isCalendarUnit(unit) && value > 0 {
		value = value / 1_000_000_000
	}

	return Timeframe{Value: value, Unit: unit}
}

// ToProto converts a Timeframe to proto AggregationOptions.
func (t Timeframe) ToProto() *proto.AggregationOptions {
	if t.IsZero() {
		return nil
	}

	unit := unitToProtoUnit(t.Unit)
	var bucket uint64

	if t.IsCalendarBased() {
		bucket = uint64(t.Value) // multiplier
	} else {
		bucket = uint64(t.Seconds()) * 1_000_000_000 // nanoseconds
	}

	return &proto.AggregationOptions{
		TimeBucket: &bucket,
		Unit:       &unit,
	}
}

func protoUnitToUnit(pu proto.TimeframeUnit) Unit {
	switch pu {
	case proto.TimeframeUnit_UNIT_MINUTE:
		return UnitMinute
	case proto.TimeframeUnit_UNIT_HOUR:
		return UnitHour
	case proto.TimeframeUnit_UNIT_DAY:
		return UnitDay
	case proto.TimeframeUnit_UNIT_WEEK:
		return UnitWeek
	case proto.TimeframeUnit_UNIT_MONTH:
		return UnitMonth
	case proto.TimeframeUnit_UNIT_YEAR:
		return UnitYear
	default:
		return UnitSecond
	}
}

func unitToProtoUnit(u Unit) proto.TimeframeUnit {
	switch u {
	case UnitMinute:
		return proto.TimeframeUnit_UNIT_MINUTE
	case UnitHour:
		return proto.TimeframeUnit_UNIT_HOUR
	case UnitDay:
		return proto.TimeframeUnit_UNIT_DAY
	case UnitWeek:
		return proto.TimeframeUnit_UNIT_WEEK
	case UnitMonth:
		return proto.TimeframeUnit_UNIT_MONTH
	case UnitYear:
		return proto.TimeframeUnit_UNIT_YEAR
	default:
		return proto.TimeframeUnit_UNIT_SECOND
	}
}

func isCalendarUnit(u Unit) bool {
	return u == UnitWeek || u == UnitMonth || u == UnitYear
}
