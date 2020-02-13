package gcalbot

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/keybase/go-keybase-chat-bot/kbchat/types/chat1"

	"google.golang.org/api/calendar/v3"
)

const AllDayDateFormat = "2006-01-02"

func ParseTime(startDateTime, endDateTime *calendar.EventDateTime) (start, end time.Time, isAllDay bool, err error) {
	if startDateTime == nil || endDateTime == nil {
		err = errors.New("empty dates")
		return
	} else if startDateTime.DateTime != "" && endDateTime.DateTime != "" {
		// this is a normal event
		isAllDay = false
		start, err = time.Parse(time.RFC3339, startDateTime.DateTime)
		if err != nil {
			return
		}
		end, err = time.Parse(time.RFC3339, endDateTime.DateTime)
		if err != nil {
			return
		}
	} else if startDateTime.Date != "" && endDateTime.Date != "" {
		// this is an all day event
		isAllDay = true
		start, err = time.Parse(AllDayDateFormat, startDateTime.Date)
		if err != nil {
			return
		}
		end, err = time.Parse(AllDayDateFormat, endDateTime.Date)
		if err != nil {
			return
		}
		end = end.Add(-24 * time.Hour) // the google API sets the end day to the day after, so compensate by one day
	} else {
		err = fmt.Errorf("invalid dates: start: %+v, end: %+v", startDateTime, endDateTime)
		return
	}
	return
}

func FormatTimeRange(
	startDateTime, endDateTime *calendar.EventDateTime,
	timezone *time.Location,
	format24HourTime bool,
) (timeRange string, err error) {
	// For normal events:
	//	If the year, month and day are the same: Wed Jan 1, 2020 6:30pm - 7:30pm (EST)
	//	If just the year and month are the same: Wed Jan 1 4:30pm - Thu Jan 2, 2020 6:30pm (EST)
	//	If just the year is the same (same ^):   Fri Jan 31 5pm - Sat Feb 1, 2020 6pm (EST)
	//	If none of the params are the same:		 Thu Dec 31, 2020 8:30am - Fri Jan 1, 2021 9:30am (EST)
	// For all day:
	//	If the year, month and day are the same: Wed Jan 1, 2020
	//	If just the year and month are the same: Wed Jan 1 - Thu Jan 2, 2020
	//	If just the year is the same (same ^):   Fri Jan 31 - Sat Feb 1, 2020
	//	If none of the params are the same:		 Thu Dec 31, 2020 - Fri Jan 1, 2021

	start, end, isAllDay, err := ParseTime(startDateTime, endDateTime)
	if err != nil {
		return "", err
	}
	if !isAllDay {
		start = start.In(timezone)
		end = end.In(timezone)
	}

	startYear, startMonth, startDay := start.Date()
	endYear, endMonth, endDay := end.Date()

	var startTime string
	var endTime string
	if !isAllDay {
		if format24HourTime {
			startTime = start.Format("15:04")
			endTime = end.Format("15:04")
		} else {
			if start.Minute() == 0 {
				startTime = start.Format("3pm")
			} else {
				startTime = start.Format("3:04pm")
			}
			if end.Minute() == 0 {
				endTime = end.Format("3pm")
			} else {
				endTime = end.Format("3:04pm")
			}
		}
	}

	if startYear == endYear && startMonth == endMonth && startDay == endDay {
		if isAllDay {
			return start.Format("Mon Jan 2, 2006"), nil
		} else {
			return fmt.Sprintf("%s %s - %s (%s)",
				start.Format("Mon Jan 2, 2006"), startTime, endTime, start.Format("MST")), nil
		}
	} else if startYear == endYear {
		if isAllDay {
			return fmt.Sprintf("%s - %s",
				start.Format("Mon Jan 2"), end.Format("Mon Jan 2, 2006")), nil
		} else {
			return fmt.Sprintf("%s %s - %s %s (%s)",
				start.Format("Mon Jan 2"), startTime, end.Format("Mon Jan 2, 2006"), endTime,
				start.Format("MST")), nil
		}
	} else {
		if isAllDay {
			return fmt.Sprintf("%s - %s",
				start.Format("Mon Jan 2, 2006"), end.Format("Mon Jan 2, 2006")), nil
		} else {
			return fmt.Sprintf("%s %s - %s %s (%s)",
				start.Format("Mon Jan 2, 2006"), startTime, end.Format("Mon Jan 2, 2006"), endTime,
				start.Format("MST")), nil
		}
	}
}

func GetUserTimezone(srv *calendar.Service) (timezone *time.Location, err error) {
	timezoneSetting, err := srv.Settings.Get("timezone").Do()
	if err != nil {
		return nil, err
	}
	return time.LoadLocation(timezoneSetting.Value)
}

func GetUserFormat24HourTime(srv *calendar.Service) (format24HourTime bool, err error) {
	format24HourTimeSetting, err := srv.Settings.Get("format24HourTime").Do()
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(format24HourTimeSetting.Value)
}

func GetMinutesFromDuration(duration time.Duration) int {
	return int(duration.Minutes())
}

func GetDurationFromMinutes(minutes int) time.Duration {
	return time.Duration(minutes) * time.Minute
}

func MinutesBeforeString(minutesBefore int) string {
	if minutesBefore == 1 {
		return "1 minute"
	} else {
		return fmt.Sprintf("%d minutes", minutesBefore)
	}
}

func GetConvHelpText(channel chat1.ChatChannel, isKeybaseMessage bool) string {
	if channel.MembersType == "team" {
		teamName := channel.Name
		if isKeybaseMessage {
			teamName = fmt.Sprintf("@%s", teamName)
		}
		return fmt.Sprintf("Configure Google Calendar notifications for a channel in the team %s", teamName)
	}
	return fmt.Sprintf("Configure Google Calendar notifications for the conversation %s", channel.Name)
}
