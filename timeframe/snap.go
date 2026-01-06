package timeframe

import "time"

// Snap aligns a Unix timestamp to the timeframe boundary.
func (t Timeframe) Snap(unix int64) int64 {
	if t.Value <= 0 || unix <= 0 {
		return unix
	}

	switch t.Unit {
	case UnitWeek:
		return snapToWeek(unix, int(t.Value))
	case UnitMonth:
		return snapToMonth(unix, int(t.Value))
	case UnitYear:
		return snapToYear(unix, int(t.Value))
	default:
		seconds := t.Seconds()
		return (unix / seconds) * seconds
	}
}

// SnapTime aligns a time.Time to the timeframe boundary.
func (t Timeframe) SnapTime(tm time.Time) time.Time {
	return time.Unix(t.Snap(tm.Unix()), 0).UTC()
}

func snapToWeek(unix int64, weeks int) int64 {
	tm := time.Unix(unix, 0).UTC()

	weekday := int(tm.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysToMonday := weekday - 1

	monday := tm.AddDate(0, 0, -daysToMonday)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)

	if weeks > 1 {
		_, week := monday.ISOWeek()
		alignedWeek := ((week - 1) / weeks) * weeks + 1
		diff := week - alignedWeek
		monday = monday.AddDate(0, 0, -diff*7)
	}

	return monday.Unix()
}

func snapToMonth(unix int64, months int) int64 {
	tm := time.Unix(unix, 0).UTC()

	first := time.Date(tm.Year(), tm.Month(), 1, 0, 0, 0, 0, time.UTC)

	if months > 1 {
		monthNum := int(first.Month()) - 1
		alignedMonth := (monthNum / months) * months
		first = time.Date(first.Year(), time.Month(alignedMonth+1), 1, 0, 0, 0, 0, time.UTC)
	}

	return first.Unix()
}

func snapToYear(unix int64, years int) int64 {
	tm := time.Unix(unix, 0).UTC()

	jan1 := time.Date(tm.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	if years > 1 {
		alignedYear := (tm.Year() / years) * years
		jan1 = time.Date(alignedYear, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	return jan1.Unix()
}

// NextPeriod returns the start of the next period.
func (t Timeframe) NextPeriod(unix int64) int64 {
	if t.Value <= 0 {
		return unix
	}

	current := t.Snap(unix)

	switch t.Unit {
	case UnitWeek:
		return time.Unix(current, 0).UTC().AddDate(0, 0, 7*int(t.Value)).Unix()
	case UnitMonth:
		return time.Unix(current, 0).UTC().AddDate(0, int(t.Value), 0).Unix()
	case UnitYear:
		return time.Unix(current, 0).UTC().AddDate(int(t.Value), 0, 0).Unix()
	default:
		return current + t.Seconds()
	}
}

// NextPeriodTime returns the start of the next period as time.Time.
func (t Timeframe) NextPeriodTime(tm time.Time) time.Time {
	return time.Unix(t.NextPeriod(tm.Unix()), 0).UTC()
}

// PrevPeriod returns the start of the previous period.
func (t Timeframe) PrevPeriod(unix int64) int64 {
	if t.Value <= 0 {
		return unix
	}

	current := t.Snap(unix)

	switch t.Unit {
	case UnitWeek:
		return time.Unix(current, 0).UTC().AddDate(0, 0, -7*int(t.Value)).Unix()
	case UnitMonth:
		return time.Unix(current, 0).UTC().AddDate(0, -int(t.Value), 0).Unix()
	case UnitYear:
		return time.Unix(current, 0).UTC().AddDate(-int(t.Value), 0, 0).Unix()
	default:
		return current - t.Seconds()
	}
}
