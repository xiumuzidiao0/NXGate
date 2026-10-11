package nodes

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
)

// countryCentroids maps ISO 3166-1 alpha-2 country codes to approximate center coordinates.
var countryCentroids = map[string][2]float64{
	"JP": {36.2048, 138.2529},
	"US": {37.0902, -95.7129},
	"KR": {35.9078, 127.7669},
	"TW": {23.6978, 120.9605},
	"HK": {22.3193, 114.1694},
	"SG": {1.3521, 103.8198},
	"GB": {55.3781, -3.4360},
	"DE": {51.1657, 10.4515},
	"FR": {46.2276, 2.2137},
	"CA": {56.1304, -106.3468},
	"AU": {-25.2744, 133.7751},
	"NL": {52.1326, 5.2913},
	"RU": {61.5240, 105.3188},
	"VN": {14.0583, 108.2772},
	"TH": {15.8700, 100.9925},
	"MY": {4.2105, 101.9758},
	"IN": {20.5937, 78.9629},
	"BR": {-14.2350, -51.9253},
	"UA": {48.3794, 31.1656},
	"PL": {51.9194, 19.1451},
	"SE": {60.1282, 18.6435},
	"NO": {60.4720, 8.4689},
	"FI": {61.9241, 25.7482},
	"IT": {41.8719, 12.5674},
	"ES": {40.4637, -3.7492},
	"CH": {46.8182, 8.2275},
	"AT": {47.5162, 14.5501},
	"NZ": {-40.9006, 174.8860},
	"ID": {-0.7893, 113.9213},
	"PH": {12.8797, 121.7740},
	"TR": {38.9637, 35.2433},
	"ZA": {-30.5595, 22.9375},
	"MX": {23.6345, -102.5528},
	"AR": {-38.4161, -63.6167},
	"CL": {-35.6751, -71.5430},
}

// GetCountryCentroid returns the centroid coordinates for a given country code.
func GetCountryCentroid(countryCode string) (float64, float64, bool) {
	coords, ok := countryCentroids[strings.ToUpper(strings.TrimSpace(countryCode))]
	if !ok {
		return 0, 0, false
	}
	return coords[0], coords[1], true
}

// ApproximateCoordinates provides stable fallback coordinates for a node based on its
// country code and a deterministic jitter derived from its IP address.
func ApproximateCoordinates(countryCode, ipStr string) (float64, float64, bool) {
	baseLat, baseLon, ok := GetCountryCentroid(countryCode)
	if !ok {
		return 0, 0, false
	}

	// Deterministic pseudo-random jitter based on IP hash
	h := sha256.Sum256([]byte(ipStr))
	val1 := binary.BigEndian.Uint32(h[0:4])
	val2 := binary.BigEndian.Uint32(h[4:8])

	// Offset between -1.5 and +1.5 degrees latitude, -2.0 and +2.0 degrees longitude
	jitterLat := (float64(val1%3000)/1000.0 - 1.5)
	jitterLon := (float64(val2%4000)/1000.0 - 2.0)

	lat := baseLat + jitterLat
	lon := baseLon + jitterLon

	// Clamp to valid geographic ranges
	if lat > 85.0 {
		lat = 85.0
	} else if lat < -85.0 {
		lat = -85.0
	}
	if lon > 180.0 {
		lon = 180.0
	} else if lon < -180.0 {
		lon = -180.0
	}

	return lat, lon, true
}
