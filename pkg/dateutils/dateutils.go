package dateutils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var UA_MONTH = map[string]time.Month{
	"Січ": time.January,
	"Лют": time.February,
	"Бер": time.March,
	"Кві": time.April,
	"Тра": time.May,
	"Чер": time.June,
	"Лип": time.July,
	"Сер": time.August,
	"Вер": time.September,
	"Жов": time.October,
	"Лис": time.November,
	"Гру": time.December,
}

var (
	// Сьогодні о 19:10
	TODAY_FORMAT_UA = 3
	// Вчора о 18:23
	YESTERDAY_FORMAT_UA = 3
	// 06 Лют 2024 о 18:29
	ABSOLUTE_FORMAT_UA = 5
)

var (
	ErrUnsupportedDateFormat = errors.New("unsupported date format")
	ErrInvalidDayFormat      = errors.New("invalid day format")
	ErrInvalidHourFormat     = errors.New("invalid hour format")
	ErrInvalidMinuteFormat   = errors.New("invalid minute format")
	ErrInvalidYearFormat     = errors.New("invalid year format")
	ErrUnknowMonthAbbr       = errors.New("unknown month abbreviation:")
	ErrInvalidMonthFormat    = errors.New("invalid month format")
)

func parseDay(day string) (int, error) {
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return 0, ErrInvalidDayFormat
	}
	return dayInt, nil
}

func parseHour(timeString string) (int, error) {
	dateFormat := strings.Split(timeString, ":")
	hour, err := strconv.Atoi(dateFormat[0])
	if err != nil {
		return 0, ErrInvalidHourFormat
	}
	return hour, nil
}

func parseMinute(timeString string) (int, error) {
	dateFormat := strings.Split(timeString, ":")
	minute, err := strconv.Atoi(dateFormat[1])
	if err != nil {
		return 0, ErrInvalidMinuteFormat
	}
	return minute, nil
}

func parseMonth(day string) (int, error) {
	monthInt, err := strconv.Atoi(day)
	if err != nil {
		return 0, ErrInvalidMonthFormat
	}
	if monthInt > 12 {
		return 0, ErrInvalidMonthFormat
	}
	return monthInt, nil
}

func parseRelativeDate(day, timeString string) (time.Time, error) {
	now := time.Now()
	var targetTime time.Time

	switch day {
	case "Сьогодні":
		hour, err := parseHour(timeString)
		if err != nil {
			return time.Time{}, err
		}
		minute, err := parseMinute(timeString)
		if err != nil {
			return time.Time{}, err
		}

		targetTime = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	case "Вчора":
		yesterday := now.AddDate(0, 0, -1)
		hour, err := parseHour(timeString)
		if err != nil {
			return time.Time{}, err
		}
		minute, err := parseMinute(timeString)
		if err != nil {
			return time.Time{}, err
		}

		targetTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), hour, minute, 0, 0, now.Location())
	default:
		return time.Time{}, ErrUnsupportedDateFormat
	}

	return targetTime, nil
}

func parseAbsoluteDate(day, monthAbbrev, year, timeString string) (time.Time, error) {
	now := time.Now()
	month, ok := UA_MONTH[monthAbbrev]
	if !ok {
		return time.Time{}, ErrUnknowMonthAbbr
	}

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		return time.Time{}, ErrInvalidYearFormat
	}

	dayInt, err := parseDay(day)
	if err != nil {
		return time.Time{}, ErrInvalidDayFormat
	}

	hourInt, err := parseHour(timeString)
	if err != nil {
		return time.Time{}, ErrInvalidHourFormat
	}

	minuteInt, err := parseMinute(timeString)
	if err != nil {
		return time.Time{}, ErrInvalidMinuteFormat
	}

	return time.Date(yearInt, month, dayInt, hourInt, minuteInt, 0, 0, now.Location()), nil
}

func ParseDateUA(str string) (time.Time, error) {
	dateFormat := strings.Fields(str)

	switch len(dateFormat) {
	case 3:
		return parseRelativeDate(dateFormat[0], dateFormat[2])
	case 5:
		return parseAbsoluteDate(dateFormat[0], dateFormat[1], dateFormat[2], dateFormat[4])
	}

	return time.Time{}, ErrUnsupportedDateFormat
}

func ToString(t time.Time) string {
	return t.Format(time.Layout)
}

func ParseString(str string) (time.Time, error) {
	return time.Parse(time.Layout, str)
}

func parseQueryString(queryStr string) (t time.Time, err error) {
	dateAndTime := strings.Split(queryStr, "T")

	var dateStr, timeStr string
	if len(dateAndTime) == 2 {
		dateStr = dateAndTime[0]
		timeStr = dateAndTime[1]
	} else if len(dateAndTime) == 1 {
		dateStr = dateAndTime[0]
	}

	var minute, hour int
	if timeStr != "" {
		hour, err = parseHour(timeStr)
		if err != nil {
			return time.Time{}, err
		}
		minute, err = parseMinute(timeStr)
		if err != nil {
			return time.Time{}, err
		}
	}

	date := strings.Split(dateStr, "-")
	if len(date) != 3 {
		return time.Time{}, fmt.Errorf("Invalid format")
	}

	var year, month, day int
	year, err = strconv.Atoi(date[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid year")
	}
	month, err = parseMonth(date[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid month")
	}

	day, err = parseDay(date[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid day")
	}

	if timeStr != "" {
		t = time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Now().Location())
	} else {
		t = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Now().Location())
	}

	return t, err
}

// Format example: "2024-10-12T10:01", "2024-10-12", "YYYY-MM-DD"
func ParseQueryString(queryTime string) (time.Time, error) {
	t, err := parseQueryString(queryTime)
	if err != nil {
		return time.Time{}, err
	}
	// Ensure that is correct time format by passing it into the time.Parse
	return ParseString(ToString(t))
}

func Pretify(t time.Time) string {
	month := t.Month()
	var monthStr string
	if month < 10 {
		monthStr = fmt.Sprintf("0%d", month)
	} else {
		monthStr = fmt.Sprint(month)
	}

	return fmt.Sprintf("%d:%d %d-%s-%d", t.Hour(), t.Minute(), t.Day(), monthStr, t.Year())
}
