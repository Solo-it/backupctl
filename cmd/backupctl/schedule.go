package main

import (
	"fmt"
	"strconv"
	"strings"
)

// toCron converts an "HH:MM" schedule into a cron expression "MM HH * * *".
func toCron(schedule string) (string, error) {
	parts := strings.SplitN(schedule, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid schedule format %q, expected HH:MM", schedule)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return "", fmt.Errorf("invalid hour in schedule %q", schedule)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return "", fmt.Errorf("invalid minutes in schedule %q", schedule)
	}
	return fmt.Sprintf("%d %d * * *", minute, hour), nil
}

// toSystemdCalendar converts "HH:MM" into OnCalendar="HH:MM:00".
func toSystemdCalendar(schedule string) (string, error) {
	if _, err := toCron(schedule); err != nil {
		return "", err
	}
	return schedule + ":00", nil
}
