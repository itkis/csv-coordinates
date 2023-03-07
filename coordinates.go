package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/wroge/wgs84"
)

func main() {
	entries, err := os.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}
	var fileEntries []string
	for _, e := range entries {
		// Assume we want to handle all .csv files, and exclude previously generated -out.csv files
		if e.Type().IsRegular() && strings.HasSuffix(strings.ToLower(e.Name()), ".csv") && !strings.HasSuffix(strings.ToLower(e.Name()), "-out.csv") {
			fileEntries = append(fileEntries, e.Name())
		}
	}

	for _, s := range fileEntries {
		log.Printf("Parsing file: %s", s)
		convertFile(s)
	}
}

func convertFile(fname string) {
	// Initialize files
	inFile, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	csvReader := csv.NewReader(inFile)
	csvReader.Comma = ';'
	defer inFile.Close()

	var outName = strings.Replace(strings.ToLower(fname), ".csv", "-out.csv", 1)
	outFile, err := os.Create(outName)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer outFile.Close()
	csvwriter := csv.NewWriter(outFile)
	csvwriter.Comma = ';'

	// Read header
	columns, error := csvReader.Read()
	if error != nil {
		log.Fatal(err)
	}

	var coordinateCol = GetColumnForCoordinates(columns)
	if coordinateCol == -1 {
		log.Fatalln("Could not find any header entry that contains 'koordinaatit'.")
	}

	// Add new columns for the latitude and longitude values
	columns = append(columns, "latitude")
	columns = append(columns, "longitude")

	_ = csvwriter.Write(columns)

	from := wgs84.EPSG().Code(25835) // Initialize coordinate conversion from spec
	// Value from https://epsg.io/3067
	// Remarks: Identical to ETRS89 / UTM zone 35N (code 25835) except for area of use. See ETRS89 / TM35FIN(N,E) (code 5048) for more usually used alternative with axis order reversed.

	for {
		line, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if strings.HasPrefix(line[coordinateCol], "[[") {
			// Has multiple values
			var values = ParseCoordinatesFromString(line[coordinateCol])

			for _, value := range values {
				lineCopy := line // Copy the line to not mutate the original values
				var x, _ = strconv.ParseFloat(value[0], 32)
				var y, _ = strconv.ParseFloat(value[1], 32)

				lon, lat, _ := wgs84.LonLat().From(from)(x, y, 0)

				// Update the copy to have the value of one coordinate
				lineCopy[coordinateCol] = fmt.Sprintf("[%f %f]", x, y)
				// Append the latitude and longitude values to end of line
				lineCopy = append(lineCopy, fmt.Sprintf("%f", lat))
				lineCopy = append(lineCopy, fmt.Sprintf("%f", lon))
				_ = csvwriter.Write(lineCopy)
			}

		} else {
			var stripped = RemoveSquareBrackets(line[coordinateCol])
			var values = strings.Split(stripped, " ")
			var x, _ = strconv.ParseFloat(values[0], 32)
			var y, _ = strconv.ParseFloat(values[1], 32)
			lon, lat, _ := wgs84.LonLat().From(from)(x, y, 0)

			// Append the latitude and longitude values to end of line
			line = append(line, fmt.Sprintf("%f", lat))
			line = append(line, fmt.Sprintf("%f", lon))
			_ = csvwriter.Write(line)
		}

	}
	csvwriter.Flush()
}

// getColumnForCoordinates finds the first column index that contains the coordinate values.
// -1 indicates that no column was found.
func GetColumnForCoordinates(columns []string) int {
	var coordinateCol = -1
	for i, header := range columns {
		if strings.Contains(strings.ToLower(header), "koordinaatit") {
			coordinateCol = i
		}
	}
	return coordinateCol
}

// parseCoordinatesFromString parses cell value with multiple coordinates and splits them into own entries.
func ParseCoordinatesFromString(value string) [][]string {
	var result [][]string
	// Remove leading and trailing square brackets
	var stripped = strings.Replace(value, "[[", "", 1)
	stripped = strings.Replace(stripped, "]]", "", 1)

	var values = strings.Split(stripped, "] [")

	for _, rowValue := range values {
		var rowValues = strings.Split(rowValue, " ")
		result = append(result, rowValues)
	}

	return result
}

// removeSquareBrackets removes the first and the last character of the string.
func RemoveSquareBrackets(value string) string {
	return value[1 : len(value)-1]
}
