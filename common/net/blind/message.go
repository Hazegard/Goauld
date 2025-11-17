package blind

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	maxDNSPacketSize    = 512
	MaxChunkSize        = 220
	maxLabelSize        = 63
	MaxRetries          = 3
	dnsTimeout          = 5 * time.Second
	RetryDelay          = 500 * time.Millisecond
	PollDelay           = 100 * time.Millisecond
	sshPacketHeaderSize = 5
	SessionIDLength     = 7
	defaultTLD          = "edu"
	maxSafeLabelSize    = 40
)

// DNS-safe base32 alphabet (no padding).
const dnsBase32Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

var dnsBase32 = base32.NewEncoding(dnsBase32Alphabet).WithPadding(base32.NoPadding)

// Encode data preserving SSH packet boundaries.
func EncodeDNSSafe(data []byte) string {
	if len(data) == 0 {
		return "EMPTY"
	}

	// Base32 encode
	encoded := dnsBase32.EncodeToString(data)

	// Split into smaller, safe DNS labels
	var labels []string
	for i := 0; i < len(encoded); i += maxSafeLabelSize {
		end := i + maxSafeLabelSize
		if end > len(encoded) {
			end = len(encoded)
		}
		labels = append(labels, encoded[i:end])
	}

	// Join with dots, ensuring no label is too long
	result := strings.Join(labels, ".")

	// Validate final result
	resultLabels := strings.Split(result, ".")
	for _, label := range resultLabels {
		if len(label) > maxLabelSize {
			// If we somehow still have a long label, split it further
			var subLabels []string
			for i := 0; i < len(label); i += maxSafeLabelSize {
				end := i + maxSafeLabelSize
				if end > len(label) {
					end = len(label)
				}
				subLabels = append(subLabels, label[i:end])
			}
			result = strings.Join(subLabels, ".")
		}
	}

	return result
}

// Decode data.
func DecodeDNSSafe(s string) ([]byte, error) {
	if s == "EMPTY" {
		return []byte{}, nil
	}

	// Join parts if split across labels
	s = strings.ReplaceAll(s, ".", "")

	// Base32 decode
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base32 decode error: %w", err)
	}

	return decoded, nil
}

// Split data into SSH packet-aware chunks.
func splitIntoChunks(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}

	var chunks [][]byte
	remaining := data

	for len(remaining) > 0 {
		chunkSize := MaxChunkSize
		if len(remaining) < chunkSize {
			chunkSize = len(remaining)
		}

		// Check if we have enough data for an SSH packet
		if chunkSize >= 4 {
			packetLen := binary.BigEndian.Uint32(remaining[:4])
			totalLen := int(packetLen) + 4 // Include length field

			if totalLen <= chunkSize && totalLen <= len(remaining) {
				// We can fit the whole SSH packet
				chunkSize = totalLen
			}
		}

		// Create chunk with bounds checking
		chunk := make([]byte, chunkSize)
		copy(chunk, remaining[:chunkSize])
		chunks = append(chunks, chunk)

		// Move to next chunk
		remaining = remaining[chunkSize:]
	}

	return chunks
}

func GenerateSessionID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	result := make([]byte, SessionIDLength)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}

	return string(result)
}

func getRandomTLD() string {
	tlds := []string{"com", "net", "org", "gov", "edu"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(tlds))))

	return tlds[n.Int64()]
}

func addChecksumToData(data []byte) []byte {
	// Add simple checksum
	sum := byte(0)
	for _, b := range data {
		sum ^= b
	}

	return append(data, sum)
}

func verifyAndStripChecksum(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.New("data too short")
	}

	checksum := data[len(data)-1]
	data = data[:len(data)-1]

	sum := byte(0)
	for _, b := range data {
		sum ^= b
	}

	if sum != checksum {
		return nil, errors.New("checksum mismatch")
	}

	return data, nil
}

func SplitDataIntoChunks(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	return chunks
}
