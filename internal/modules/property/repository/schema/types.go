package schema

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// GeographyPoint represents a PostGIS geography point (latitude, longitude).
// It implements sql.Scanner and driver.Valuer for GORM compatibility.
type GeographyPoint struct {
	Lat  float64
	Lng  float64
	SRID int // Spatial Reference System Identifier (default: 4326 for WGS84)
}

// NewGeographyPoint creates a new GeographyPoint with WGS84 (SRID 4326).
func NewGeographyPoint(lat, lng float64) *GeographyPoint {
	return &GeographyPoint{
		Lat:  lat,
		Lng:  lng,
		SRID: 4326,
	}
}

// Valid checks if the coordinates are within valid ranges.
func (g *GeographyPoint) Valid() bool {
	return g.Lat >= -90 && g.Lat <= 90 && g.Lng >= -180 && g.Lng <= 180
}

// Scan implements sql.Scanner interface to read from database.
func (g *GeographyPoint) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan GeographyPoint: expected []byte, got %T", value)
	}

	// PostGIS returns WKB (Well-Known Binary) format
	// Format: SRID (4 bytes) + WKB geometry
	if len(bytes) < 5 {
		return fmt.Errorf("invalid WKB data: too short")
	}

	// Parse EWKB (Extended Well-Known Binary) format
	// First byte is endianness
	// Next 4 bytes are geometry type (with SRID flag)
	// Next 4 bytes are SRID
	// Next 16 bytes are X (lng) and Y (lat) as float64

	reader := &wkbReader{data: bytes}

	// Read endianness
	littleEndian := reader.readByte() == 1
	reader.littleEndian = littleEndian

	// Read geometry type (includes SRID flag)
	geomType := reader.readUint32()
	hasSRID := (geomType & 0x20000000) != 0

	// Read SRID if present
	if hasSRID {
		g.SRID = int(reader.readUint32())
	} else {
		g.SRID = 4326 // default to WGS84
	}

	// Read coordinates (X=longitude, Y=latitude)
	g.Lng = reader.readFloat64()
	g.Lat = reader.readFloat64()

	if reader.err != nil {
		return fmt.Errorf("failed to parse WKB: %w", reader.err)
	}

	return nil
}

// Value implements driver.Valuer interface to write to database.
func (g GeographyPoint) Value() (driver.Value, error) {
	if !g.Valid() {
		return nil, fmt.Errorf("invalid coordinates: lat=%f, lng=%f", g.Lat, g.Lng)
	}

	// Return as WKT (Well-Known Text) with SRID
	// PostGIS will convert this to geography type
	wkt := fmt.Sprintf("SRID=%d;POINT(%f %f)", g.SRID, g.Lng, g.Lat)
	return wkt, nil
}

// String returns a human-readable representation.
func (g GeographyPoint) String() string {
	return fmt.Sprintf("Point(lat=%f, lng=%f, SRID=%d)", g.Lat, g.Lng, g.SRID)
}

// WKT returns the Well-Known Text representation.
func (g GeographyPoint) WKT() string {
	return fmt.Sprintf("POINT(%f %f)", g.Lng, g.Lat)
}

// wkbReader is a helper for reading WKB binary data.
type wkbReader struct {
	data         []byte
	offset       int
	littleEndian bool
	err          error
}

func (r *wkbReader) readByte() byte {
	if r.err != nil || r.offset >= len(r.data) {
		r.err = fmt.Errorf("read past end of buffer")
		return 0
	}
	b := r.data[r.offset]
	r.offset++
	return b
}

func (r *wkbReader) readUint32() uint32 {
	if r.err != nil || r.offset+4 > len(r.data) {
		r.err = fmt.Errorf("read past end of buffer")
		return 0
	}
	var v uint32
	if r.littleEndian {
		v = binary.LittleEndian.Uint32(r.data[r.offset : r.offset+4])
	} else {
		v = binary.BigEndian.Uint32(r.data[r.offset : r.offset+4])
	}
	r.offset += 4
	return v
}

func (r *wkbReader) readFloat64() float64 {
	if r.err != nil || r.offset+8 > len(r.data) {
		r.err = fmt.Errorf("read past end of buffer")
		return 0
	}
	var bits uint64
	if r.littleEndian {
		bits = binary.LittleEndian.Uint64(r.data[r.offset : r.offset+8])
	} else {
		bits = binary.BigEndian.Uint64(r.data[r.offset : r.offset+8])
	}
	r.offset += 8
	return math.Float64frombits(bits)
}

// VectorEmbedding represents a pgvector embedding.
// It implements sql.Scanner and driver.Valuer for GORM compatibility.
type VectorEmbedding struct {
	Dimensions int
	Vector     []float32
}

// NewVectorEmbedding creates a new VectorEmbedding from a float slice.
func NewVectorEmbedding(values []float32) *VectorEmbedding {
	return &VectorEmbedding{
		Dimensions: len(values),
		Vector:     values,
	}
}

// Scan implements sql.Scanner interface to read from database.
func (v *VectorEmbedding) Scan(value any) error {
	if value == nil {
		v.Vector = nil
		v.Dimensions = 0
		return nil
	}

	// pgvector returns vectors as string in format: "[1.0,2.0,3.0]"
	str, ok := value.(string)
	if !ok {
		bytes, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("failed to scan VectorEmbedding: expected string or []byte, got %T", value)
		}
		str = string(bytes)
	}

	// Remove brackets and split by comma
	str = strings.TrimSpace(str)
	if !strings.HasPrefix(str, "[") || !strings.HasSuffix(str, "]") {
		return fmt.Errorf("invalid vector format: %s", str)
	}

	str = str[1 : len(str)-1] // Remove brackets
	if str == "" {
		v.Vector = []float32{}
		v.Dimensions = 0
		return nil
	}

	parts := strings.Split(str, ",")
	v.Vector = make([]float32, len(parts))
	v.Dimensions = len(parts)

	for i, part := range parts {
		var val float32
		_, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &val)
		if err != nil {
			return fmt.Errorf("failed to parse vector component at index %d: %w", i, err)
		}
		v.Vector[i] = val
	}

	return nil
}

// Value implements driver.Valuer interface to write to database.
func (v VectorEmbedding) Value() (driver.Value, error) {
	if len(v.Vector) == 0 {
		return nil, nil
	}

	// Format as "[1.0,2.0,3.0]" for pgvector
	parts := make([]string, len(v.Vector))
	for i, val := range v.Vector {
		parts[i] = fmt.Sprintf("%f", val)
	}

	return "[" + strings.Join(parts, ",") + "]", nil
}

// String returns a human-readable representation.
func (v VectorEmbedding) String() string {
	if v.Vector == nil {
		return "Vector(nil)"
	}
	return fmt.Sprintf("Vector(dims=%d, first_3=[%.4f, %.4f, %.4f...])",
		v.Dimensions,
		safeIndex(v.Vector, 0),
		safeIndex(v.Vector, 1),
		safeIndex(v.Vector, 2),
	)
}

// Normalize normalizes the vector to unit length (for cosine similarity).
func (v *VectorEmbedding) Normalize() {
	if len(v.Vector) == 0 {
		return
	}

	var norm float64
	for _, val := range v.Vector {
		norm += float64(val) * float64(val)
	}
	norm = math.Sqrt(norm)

	if norm > 0 {
		for i := range v.Vector {
			v.Vector[i] = float32(float64(v.Vector[i]) / norm)
		}
	}
}

// ToSlice returns the vector as a float32 slice.
func (v *VectorEmbedding) ToSlice() []float32 {
	return v.Vector
}

// FromSlice sets the vector from a float32 slice.
func (v *VectorEmbedding) FromSlice(values []float32) {
	v.Vector = values
	v.Dimensions = len(values)
}

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value between -1 and 1, where 1 means identical direction.
// Note: Assumes vectors are already normalized for best performance.
func (v *VectorEmbedding) CosineSimilarity(other *VectorEmbedding) (float64, error) {
	if v.Dimensions != other.Dimensions {
		return 0, fmt.Errorf("dimension mismatch: %d != %d", v.Dimensions, other.Dimensions)
	}

	var dotProduct float64
	for i := 0; i < v.Dimensions; i++ {
		dotProduct += float64(v.Vector[i]) * float64(other.Vector[i])
	}

	return dotProduct, nil
}

// safeIndex safely accesses a slice with bounds checking.
func safeIndex(slice []float32, index int) float32 {
	if index < len(slice) {
		return slice[index]
	}
	return 0
}

// GormDataType tells GORM what data type to use for this field.
func (VectorEmbedding) GormDataType() string {
	return "vector"
}

// GormDataType tells GORM what data type to use for this field.
func (GeographyPoint) GormDataType() string {
	return "geography(Point,4326)"
}
