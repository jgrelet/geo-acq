package decoder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	nmea "github.com/jgrelet/go-nmea"
)

// DecodedSentence contains a normalized JSON payload for one NMEA sentence.
type DecodedSentence struct {
	SentenceType string
	JSON         string
}

// DecodeNMEA parses one NMEA sentence and returns a normalized JSON payload.
func DecodeNMEA(raw string) (DecodedSentence, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DecodedSentence{}, fmt.Errorf("empty NMEA sentence")
	}

	raw = normalizeNMEATimeFields(raw)

	if decoded, ok, err := decodeCustomSentence(raw); ok {
		return decoded, err
	}

	msg, err := nmea.Parse(raw)
	if err != nil {
		return DecodedSentence{}, err
	}

	message := msg.GetMessage()
	sentenceType := message.Type.GetTypeID().Serialize()

	payload := buildPayload(raw, sentenceType, msg)
	data, err := json.Marshal(payload)
	if err != nil {
		return DecodedSentence{}, fmt.Errorf("marshal decoded %s sentence: %w", sentenceType, err)
	}

	return DecodedSentence{
		SentenceType: sentenceType,
		JSON:         string(data),
	}, nil
}

func buildPayload(raw string, sentenceType string, msg nmea.NMEA) interface{} {
	switch sentence := msg.(type) {
	case *nmea.GPGGA:
		return map[string]interface{}{
			"sentence_type":      sentenceType,
			"time_utc":           sentence.TimeUTC,
			"latitude":           float64(sentence.Latitude),
			"longitude":          float64(sentence.Longitude),
			"quality_indicator":  sentence.QualityIndicator,
			"satellites_used":    sentence.NbOfSatellitesUsed,
			"hdop":               sentence.HDOP,
			"altitude_m":         sentence.Altitude,
			"geoid_separation_m": optionalFloat(sentence.GeoIDSep),
			"dgps_age_s":         optionalFloat(sentence.DGPSAge),
			"dgps_station_id":    optionalUint8(sentence.DGPSStationID),
		}
	case *nmea.GPRMC:
		datetimeUTC := sentence.DateTimeUTC
		if parsed, ok := extractRMCDatetime(raw); ok {
			datetimeUTC = parsed
		}
		return map[string]interface{}{
			"sentence_type":          sentenceType,
			"datetime_utc":           datetimeUTC,
			"is_valid":               bool(sentence.IsValid),
			"latitude":               float64(sentence.Latitude),
			"longitude":              float64(sentence.Longitude),
			"speed_knots":            sentence.Speed,
			"course_over_deg":        sentence.COG,
			"magnetic_variation_deg": sentence.MagneticVariation,
			"positioning_mode":       string(sentence.PositioningMode),
		}
	case *nmea.GPVTG:
		return map[string]interface{}{
			"sentence_type":    sentenceType,
			"course_over_deg":  sentence.COG,
			"speed_knots":      sentence.SpeedKnots,
			"speed_kmh":        sentence.SpeedKmh,
			"positioning_mode": string(sentence.PositioningMode),
		}
	case *nmea.GPGLL:
		return map[string]interface{}{
			"sentence_type":    sentenceType,
			"time_utc":         sentence.TimeUTC,
			"latitude":         float64(sentence.Latitude),
			"longitude":        float64(sentence.Longitude),
			"is_valid":         bool(sentence.IsValid),
			"positioning_mode": string(sentence.PositioningMode),
		}
	case *nmea.GPGSA:
		return map[string]interface{}{
			"sentence_type":       sentenceType,
			"mode":                string(sentence.Mode),
			"fix_status":          sentence.FixStatus,
			"satellites_channels": channelsInUse(sentence.SatelliteUsedOnChannel),
			"pdop":                sentence.PDOP,
			"hdop":                sentence.HDOP,
			"vdop":                sentence.VDOP,
		}
	case *nmea.GPGSV:
		return map[string]interface{}{
			"sentence_type":      sentenceType,
			"message_count":      sentence.NbOfMessage,
			"sequence_number":    sentence.SequenceNumber,
			"satellites_in_view": sentence.SatellitesInView,
			"satellites":         sentence.Satellites,
		}
	case *nmea.GPDBT:
		return map[string]interface{}{
			"sentence_type": sentenceType,
			"depth_feet":    sentence.DepthInFeet,
			"depth_meters":  sentence.DepthInMeters,
			"depth_fathoms": sentence.DepthInFathoms,
		}
	default:
		return map[string]interface{}{
			"sentence_type": sentenceType,
			"fields":        messageFields(msg),
		}
	}
}

func decodeCustomSentence(raw string) (DecodedSentence, bool, error) {
	sentenceType, parts, ok := splitSentence(raw)
	if !ok {
		return DecodedSentence{}, false, nil
	}
	if !hasValidChecksum(raw) {
		return DecodedSentence{}, false, nil
	}

	var payload map[string]interface{}
	switch {
	case strings.HasSuffix(sentenceType, "RMC"):
		parsed, err := buildRMCPayload(sentenceType, parts)
		if err != nil {
			return DecodedSentence{}, true, err
		}
		payload = parsed
	case strings.HasSuffix(sentenceType, "ZDA"):
		parsed, err := buildZDAPayload(sentenceType, parts)
		if err != nil {
			return DecodedSentence{}, true, err
		}
		payload = parsed
	default:
		return DecodedSentence{}, false, nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return DecodedSentence{}, true, fmt.Errorf("marshal decoded %s sentence: %w", sentenceType, err)
	}

	return DecodedSentence{
		SentenceType: sentenceType,
		JSON:         string(data),
	}, true, nil
}

func extractRMCDatetime(raw string) (time.Time, bool) {
	star := strings.LastIndexByte(raw, '*')
	if star <= 0 || len(raw) < 2 {
		return time.Time{}, false
	}

	body := raw[1:star]
	parts := strings.Split(body, ",")
	if len(parts) < 10 {
		return time.Time{}, false
	}

	sentenceID := strings.TrimSpace(parts[0])
	if sentenceID != "GPRMC" && sentenceID != "GNRMC" {
		return time.Time{}, false
	}

	timeValue := strings.TrimSpace(parts[1])
	dateValue := strings.TrimSpace(parts[9])
	if timeValue == "" || dateValue == "" {
		return time.Time{}, false
	}

	normalizedTime, ok := normalizeUTCTime(timeValue)
	if !ok {
		return time.Time{}, false
	}

	parsed, err := time.Parse("020106 150405.000", dateValue+" "+normalizedTime)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}

func buildRMCPayload(sentenceType string, parts []string) (map[string]interface{}, error) {
	if len(parts) < 10 {
		return nil, fmt.Errorf("incomplete %s sentence", sentenceType)
	}

	datetimeUTC, ok := parseRMCDateTime(parts[1], parts[9])
	if !ok {
		return nil, fmt.Errorf("invalid %s datetime", sentenceType)
	}

	latitude, err := parseLatLong(parts, 3, 4)
	if err != nil {
		return nil, err
	}
	longitude, err := parseLatLong(parts, 5, 6)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"sentence_type": sentenceType,
		"datetime_utc":  datetimeUTC,
		"is_valid":      strings.EqualFold(strings.TrimSpace(fieldAt(parts, 2)), "A"),
		"latitude":      latitude,
		"longitude":     longitude,
	}

	if value, ok := parseOptionalFloat(fieldAt(parts, 7)); ok {
		payload["speed_knots"] = value
	}
	if value, ok := parseOptionalFloat(fieldAt(parts, 8)); ok {
		payload["course_over_deg"] = value
	}
	if value, ok := parseOptionalSignedFloat(fieldAt(parts, 10), fieldAt(parts, 11)); ok {
		payload["magnetic_variation_deg"] = value
	}
	if mode := strings.TrimSpace(fieldAt(parts, 12)); mode != "" {
		payload["positioning_mode"] = mode
	}

	return payload, nil
}

func buildZDAPayload(sentenceType string, parts []string) (map[string]interface{}, error) {
	if len(parts) < 5 {
		return nil, fmt.Errorf("incomplete %s sentence", sentenceType)
	}

	datetimeUTC, ok := parseZDADateTime(fieldAt(parts, 1), fieldAt(parts, 2), fieldAt(parts, 3), fieldAt(parts, 4))
	if !ok {
		return nil, fmt.Errorf("invalid %s datetime", sentenceType)
	}

	payload := map[string]interface{}{
		"sentence_type": sentenceType,
		"datetime_utc":  datetimeUTC,
	}

	if value, ok := parseOptionalInt(fieldAt(parts, 5)); ok {
		payload["local_zone_hours"] = value
	}
	if value, ok := parseOptionalInt(fieldAt(parts, 6)); ok {
		payload["local_zone_minutes"] = value
	}

	return payload, nil
}

func parseRMCDateTime(timeValue string, dateValue string) (time.Time, bool) {
	normalizedTime, ok := normalizeUTCTime(timeValue)
	if !ok || strings.TrimSpace(dateValue) == "" {
		return time.Time{}, false
	}

	parsed, err := time.Parse("020106 150405.000", strings.TrimSpace(dateValue)+" "+normalizedTime)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func parseZDADateTime(timeValue string, dayValue string, monthValue string, yearValue string) (time.Time, bool) {
	normalizedTime, ok := normalizeUTCTime(timeValue)
	if !ok {
		return time.Time{}, false
	}

	day, err := strconv.Atoi(strings.TrimSpace(dayValue))
	if err != nil {
		return time.Time{}, false
	}
	month, err := strconv.Atoi(strings.TrimSpace(monthValue))
	if err != nil {
		return time.Time{}, false
	}
	year, err := strconv.Atoi(strings.TrimSpace(yearValue))
	if err != nil {
		return time.Time{}, false
	}

	parsed, err := time.Parse(time.RFC3339Nano, fmt.Sprintf("%04d-%02d-%02dT%sZ", year, month, day, formatClock(normalizedTime)))
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}

func formatClock(normalizedTime string) string {
	base, frac, _ := strings.Cut(normalizedTime, ".")
	if len(base) != 6 {
		return normalizedTime
	}
	clock := base[:2] + ":" + base[2:4] + ":" + base[4:6]
	if frac == "" || frac == "000" {
		return clock
	}
	return clock + "." + frac
}

func parseLatLong(parts []string, valueIndex int, cardinalIndex int) (float64, error) {
	value := strings.TrimSpace(fieldAt(parts, valueIndex))
	cardinal := strings.TrimSpace(fieldAt(parts, cardinalIndex))
	if value == "" || cardinal == "" {
		return 0, nil
	}

	latLong, err := nmea.NewLatLong(value + " " + cardinal)
	if err != nil {
		return 0, err
	}

	return float64(latLong), nil
}

func parseOptionalFloat(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func parseOptionalSignedFloat(value string, direction string) (float64, bool) {
	parsed, ok := parseOptionalFloat(value)
	if !ok {
		return 0, false
	}
	switch strings.ToUpper(strings.TrimSpace(direction)) {
	case "W":
		return -parsed, true
	case "E", "":
		return parsed, true
	default:
		return parsed, true
	}
}

func parseOptionalInt(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func splitSentence(raw string) (string, []string, bool) {
	if len(raw) < 2 || raw[0] != '$' {
		return "", nil, false
	}
	star := strings.LastIndexByte(raw, '*')
	if star <= 1 {
		return "", nil, false
	}

	body := raw[1:star]
	parts := strings.Split(body, ",")
	if len(parts) == 0 {
		return "", nil, false
	}

	return strings.TrimSpace(parts[0]), parts, true
}

func hasValidChecksum(raw string) bool {
	if len(raw) < 4 || raw[0] != '$' {
		return false
	}
	star := strings.LastIndexByte(raw, '*')
	if star <= 1 || star+3 > len(raw) {
		return false
	}

	return strings.EqualFold(raw[star+1:star+3], checksum(raw[1:star]))
}

func fieldAt(parts []string, index int) string {
	if index < 0 || index >= len(parts) {
		return ""
	}
	return parts[index]
}

func messageFields(msg nmea.NMEA) []string {
	return msg.GetMessage().Fields
}

func channelsInUse(channels [13]int) []int {
	values := make([]int, 0, len(channels)-1)
	for _, channel := range channels[1:] {
		if channel > 0 {
			values = append(values, channel)
		}
	}
	return values
}

func optionalFloat(value *float64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func optionalUint8(value *uint8) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func normalizeNMEATimeFields(raw string) string {
	star := strings.LastIndexByte(raw, '*')
	if star <= 0 {
		return raw
	}

	body := raw[1:star]
	parts := strings.Split(body, ",")
	if len(parts) == 0 {
		return raw
	}

	changed := false
	switch parts[0] {
	case "GPGGA", "GNGGA":
		changed = normalizeField(parts, 1) || changed
	case "GPGLL", "GNGLL":
		changed = normalizeField(parts, 5) || changed
	case "GPRMC", "GNRMC":
		changed = normalizeField(parts, 1) || changed
	}

	if !changed {
		return raw
	}

	payload := strings.Join(parts, ",")
	return "$" + payload + "*" + checksum(payload)
}

func normalizeField(parts []string, index int) bool {
	if index < 0 || index >= len(parts) {
		return false
	}

	normalized, ok := normalizeUTCTime(parts[index])
	if !ok || normalized == parts[index] {
		return false
	}

	parts[index] = normalized
	return true
}

func normalizeUTCTime(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return value, false
	}

	base, frac, hasDot := strings.Cut(value, ".")
	if len(base) != 6 {
		return value, false
	}
	if _, err := strconv.Atoi(base); err != nil {
		return value, false
	}

	if !hasDot {
		return base + ".000", true
	}

	if frac == "" {
		return base + ".000", true
	}

	for _, r := range frac {
		if r < '0' || r > '9' {
			return value, false
		}
	}

	switch {
	case len(frac) == 3:
		return value, true
	case len(frac) < 3:
		return base + "." + frac + strings.Repeat("0", 3-len(frac)), true
	default:
		return base + "." + frac[:3], true
	}
}

func checksum(payload string) string {
	sum := byte(0)
	for i := 0; i < len(payload); i++ {
		sum ^= payload[i]
	}
	return strings.ToUpper(fmt.Sprintf("%02x", sum))
}
