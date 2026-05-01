package exporter

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	// ModeSlowestDevice aligns rows on the device with the fewest frames.
	ModeSlowestDevice = "slowest_device"
	// ModeFixedInterval aligns rows on a constant time step.
	ModeFixedInterval = "fixed_interval"
)

// Frame represents one raw NMEA frame loaded from SQLite.
type Frame struct {
	ReceivedAt time.Time
	DeviceName string
	Payload    string
}

// Session describes the exported acquisition session.
type Session struct {
	ID         int64
	Mission    string
	ConfigFile string
	StartedAt  time.Time
}

// Row represents one aligned export row.
type Row struct {
	Timestamp time.Time
	Values    map[string]string
}

// BuildRows aligns raw frames on either the slowest device or a fixed interval.
func BuildRows(frames []Frame, deviceNames []string, mode string, interval time.Duration) ([]Row, error) {
	if len(frames) == 0 {
		return nil, nil
	}
	if len(deviceNames) == 0 {
		return nil, fmt.Errorf("no device names provided")
	}

	sort.Slice(frames, func(i, j int) bool {
		if frames[i].ReceivedAt.Equal(frames[j].ReceivedAt) {
			if frames[i].DeviceName == frames[j].DeviceName {
				return frames[i].Payload < frames[j].Payload
			}
			return frames[i].DeviceName < frames[j].DeviceName
		}
		return frames[i].ReceivedAt.Before(frames[j].ReceivedAt)
	})

	anchors, err := buildAnchors(frames, deviceNames, mode, interval)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(anchors))
	pointers := make(map[string]int, len(deviceNames))
	latest := make(map[string]string, len(deviceNames))
	grouped := groupFramesByDevice(frames, deviceNames)

	for _, anchor := range anchors {
		row := Row{
			Timestamp: anchor,
			Values:    make(map[string]string, len(deviceNames)),
		}
		for _, deviceName := range deviceNames {
			deviceFrames := grouped[deviceName]
			idx := pointers[deviceName]
			for idx < len(deviceFrames) && !deviceFrames[idx].ReceivedAt.After(anchor) {
				latest[deviceName] = deviceFrames[idx].Payload
				idx++
			}
			pointers[deviceName] = idx
			row.Values[deviceName] = latest[deviceName]
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// WriteTSV writes the export rows in a plain-text TSV format.
func WriteTSV(w io.Writer, session Session, deviceNames []string, rows []Row) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	if _, err := fmt.Fprintf(bw, "# mission=%s\n", session.Mission); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(bw, "# session_id=%d\n", session.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(bw, "# config_file=%s\n", session.ConfigFile); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(bw, "# started_at_utc=%s\n", session.StartedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}

	header := append([]string{"timestamp_utc"}, deviceNames...)
	if _, err := fmt.Fprintln(bw, strings.Join(header, "\t")); err != nil {
		return err
	}

	for _, row := range rows {
		fields := make([]string, 0, len(deviceNames)+1)
		fields = append(fields, row.Timestamp.UTC().Format(time.RFC3339Nano))
		for _, deviceName := range deviceNames {
			fields = append(fields, sanitizeTSV(row.Values[deviceName]))
		}
		if _, err := fmt.Fprintln(bw, strings.Join(fields, "\t")); err != nil {
			return err
		}
	}

	return nil
}

func buildAnchors(frames []Frame, deviceNames []string, mode string, interval time.Duration) ([]time.Time, error) {
	switch mode {
	case "", ModeSlowestDevice:
		return anchorsFromSlowestDevice(frames, deviceNames)
	case ModeFixedInterval:
		if interval <= 0 {
			return nil, fmt.Errorf("fixed interval mode requires interval > 0")
		}
		return anchorsFromFixedInterval(frames, interval), nil
	default:
		return nil, fmt.Errorf("unsupported export mode %q", mode)
	}
}

func anchorsFromSlowestDevice(frames []Frame, deviceNames []string) ([]time.Time, error) {
	grouped := groupFramesByDevice(frames, deviceNames)

	var anchorDevice string
	minCount := -1
	for _, deviceName := range deviceNames {
		count := len(grouped[deviceName])
		if count == 0 {
			continue
		}
		if minCount == -1 || count < minCount || (count == minCount && deviceName < anchorDevice) {
			minCount = count
			anchorDevice = deviceName
		}
	}
	if anchorDevice == "" {
		return nil, fmt.Errorf("no frames available for slowest device export")
	}

	anchors := make([]time.Time, 0, len(grouped[anchorDevice]))
	for _, frame := range grouped[anchorDevice] {
		anchors = append(anchors, frame.ReceivedAt)
	}
	return anchors, nil
}

func anchorsFromFixedInterval(frames []Frame, interval time.Duration) []time.Time {
	start := frames[0].ReceivedAt
	end := frames[len(frames)-1].ReceivedAt

	anchors := []time.Time{start}
	for cursor := start.Add(interval); !cursor.After(end); cursor = cursor.Add(interval) {
		anchors = append(anchors, cursor)
	}
	if anchors[len(anchors)-1].Before(end) {
		anchors = append(anchors, end)
	}
	return anchors
}

func groupFramesByDevice(frames []Frame, deviceNames []string) map[string][]Frame {
	grouped := make(map[string][]Frame, len(deviceNames))
	for _, deviceName := range deviceNames {
		grouped[deviceName] = nil
	}
	for _, frame := range frames {
		grouped[frame.DeviceName] = append(grouped[frame.DeviceName], frame)
	}
	return grouped
}

func sanitizeTSV(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
