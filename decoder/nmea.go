package decoder

import (
	"encoding/json"
	"fmt"
	"strings"

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

	msg, err := nmea.Parse(raw)
	if err != nil {
		return DecodedSentence{}, err
	}

	message := msg.GetMessage()
	sentenceType := message.Type.GetTypeID().Serialize()

	payload := buildPayload(sentenceType, msg)
	data, err := json.Marshal(payload)
	if err != nil {
		return DecodedSentence{}, fmt.Errorf("marshal decoded %s sentence: %w", sentenceType, err)
	}

	return DecodedSentence{
		SentenceType: sentenceType,
		JSON:         string(data),
	}, nil
}

func buildPayload(sentenceType string, msg nmea.NMEA) interface{} {
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
		return map[string]interface{}{
			"sentence_type":          sentenceType,
			"datetime_utc":           sentence.DateTimeUTC,
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
