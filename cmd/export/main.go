package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jgrelet/geo-acq/exporter"
	"github.com/jgrelet/geo-acq/storage"
	"github.com/pborman/getopt"
)

var (
	optHelp      = getopt.BoolLong("help", 'h', "display help")
	optRaw       = getopt.BoolLong("raw", 'r', "export raw SQLite input")
	optProcessed = getopt.BoolLong("processed", 'p', "export processed SQLite input")
	optCompact   = getopt.BoolLong("compact", 'c', "write compact processed output")
	optFull      = getopt.BoolLong("full", 'f', "write full output")
	optCSV       = getopt.BoolLong("csv", 'C', "write CSV output")
	optTSV       = getopt.BoolLong("tsv", 'T', "write TSV output")
	optOutput    = getopt.StringLong("output", 'o', "", "output file", "file")
	optMission   = getopt.StringLong("mission", 'm', "", "optional mission filter", "name")
	optSessionID = getopt.Int64Long("session-id", 's', 0, "optional session id", "id")
)

func main() {
	configureUsage()
	getopt.Parse()

	if *optHelp {
		getopt.Usage()
		return
	}

	inputPath := ""
	if getopt.NArgs() > 0 {
		inputPath = getopt.Arg(0)
	}
	if strings.TrimSpace(inputPath) == "" {
		log.Fatal("input SQLite file is required")
	}

	rawSelected, processedSelected, err := resolveSourceMode()
	if err != nil {
		log.Fatal(err)
	}
	compactSelected, err := resolveLayoutMode(rawSelected)
	if err != nil {
		log.Fatal(err)
	}
	format, err := resolveOutputFormat()
	if err != nil {
		log.Fatal(err)
	}

	selection := storage.SessionSelection{
		MissionName: strings.TrimSpace(*optMission),
		SessionID:   *optSessionID,
	}

	path := outputFilePath(*optOutput, inputPath, format)
	file, err := createOutputFile(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if rawSelected {
		session, records, err := storage.LoadRawRecordsForExport(inputPath, selection)
		if err != nil {
			log.Fatal(err)
		}
		if err := writeRaw(file, format, session, records); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("wrote %d rows to %s\n", len(records), path)
		return
	}

	if !processedSelected {
		processedSelected = true
	}
	session, records, err := storage.LoadProcessedRecordsForExport(inputPath, selection)
	if err != nil {
		log.Fatal(err)
	}
	if compactSelected {
		if err := writeCompactProcessed(file, format, session, records); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("wrote %d rows to %s\n", len(exporter.BuildCompactProcessedRecords(records)), path)
		return
	}
	if err := writeProcessed(file, format, session, records); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d rows to %s\n", len(records), path)
}

func configureUsage() {
	getopt.SetParameters("<input.sqlite>")
	getopt.CommandLine.SetProgram("geo-export")
	getopt.CommandLine.SetUsage(func() {
		fmt.Fprintln(os.Stderr, "Export geo-acq SQLite data to TSV or CSV.")
		fmt.Fprintln(os.Stderr)
		getopt.CommandLine.PrintUsage(os.Stderr)
	})
}

func resolveSourceMode() (bool, bool, error) {
	rawSelected := *optRaw
	processedSelected := *optProcessed
	if rawSelected && processedSelected {
		return false, false, fmt.Errorf("use raw or processed export, not both")
	}
	if !rawSelected && !processedSelected {
		processedSelected = true
	}
	return rawSelected, processedSelected, nil
}

func resolveLayoutMode(rawSelected bool) (bool, error) {
	compactSelected := *optCompact
	fullSelected := *optFull
	if compactSelected && fullSelected {
		return false, fmt.Errorf("use compact or full output, not both")
	}
	if !compactSelected && !fullSelected {
		compactSelected = true
	}
	if rawSelected && compactSelected {
		return false, fmt.Errorf("compact output is only available with processed export")
	}
	return compactSelected, nil
}

func resolveOutputFormat() (string, error) {
	if *optCSV && *optTSV {
		return "", fmt.Errorf("use csv or tsv output, not both")
	}
	if *optCSV && !*optTSV {
		return exporter.FormatCSV, nil
	}
	return exporter.FormatTSV, nil
}

func writeRaw(file *os.File, format string, session exporter.Session, records []exporter.RawRecord) error {
	switch format {
	case exporter.FormatCSV:
		return exporter.WriteRawCSV(file, session, records)
	default:
		return exporter.WriteRawTSV(file, session, records)
	}
}

func writeProcessed(file *os.File, format string, session exporter.Session, records []exporter.ProcessedRecord) error {
	switch format {
	case exporter.FormatCSV:
		return exporter.WriteProcessedCSV(file, session, records)
	default:
		return exporter.WriteProcessedTSV(file, session, records)
	}
}

func writeCompactProcessed(file *os.File, format string, session exporter.Session, records []exporter.ProcessedRecord) error {
	switch format {
	case exporter.FormatCSV:
		return exporter.WriteCompactProcessedCSV(file, session, records)
	default:
		return exporter.WriteCompactProcessedTSV(file, session, records)
	}
}

func outputFilePath(explicit string, input string, format string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}

	ext := "." + format
	currentExt := filepath.Ext(input)
	if currentExt == "" {
		return input + ext
	}
	return strings.TrimSuffix(input, currentExt) + ext
}

func createOutputFile(path string) (*os.File, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, err
	}
	return os.Create(path)
}
