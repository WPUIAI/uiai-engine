package epwadelivery

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

const DeliveryEventSchema = "uiai.epwa_delivery_event.v1"

var (
	ErrStoreInvalid = errors.New("EPWA delivery store invalid")
	ErrStoreCorrupt = errors.New("EPWA delivery store corrupt")
	ErrNotFound     = errors.New("EPWA delivery not found")
	deliveryStoreMu sync.Mutex
)

type DeliveryEvent struct {
	Schema              string   `json:"schema"`
	Sequence            uint64   `json:"sequence"`
	Delivery            Delivery `json:"delivery"`
	PreviousEventSHA256 string   `json:"previous_event_sha256,omitempty"`
	EventSHA256         string   `json:"event_sha256"`
}

// Record appends a hash-chained delivery revision and atomically refreshes its
// replay projection. The append-only ledger remains authoritative after a crash.
func Record(root string, candidate Delivery) (Delivery, error) {
	deliveryStoreMu.Lock()
	defer deliveryStoreMu.Unlock()
	if err := Validate(candidate); err != nil {
		return Delivery{}, err
	}
	directory, key, err := deliveryStorePath(root, candidate.DeliveryID)
	if err != nil {
		return Delivery{}, err
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return Delivery{}, err
	}
	ledgerPath := filepath.Join(directory, key+".jsonl")
	events, err := loadEvents(ledgerPath, candidate.DeliveryID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Delivery{}, err
	}
	previousSHA := ""
	if len(events) == 0 {
		candidate.Revision = 1
	} else {
		previous := events[len(events)-1]
		if previous.Delivery.DeliveryID != candidate.DeliveryID || previous.Delivery.CreatedAt.After(candidate.ObservedAt) {
			return Delivery{}, ErrStoreInvalid
		}
		candidate.CreatedAt = previous.Delivery.CreatedAt
		if previous.Delivery.State == StateReady && (candidate.State == StatePendingReconcile || candidate.State == StateUnavailable || candidate.State == StateDegraded || candidate.State == StateBlocked) {
			return previous.Delivery, nil
		}
		if equivalentDeliveryState(previous.Delivery, candidate) {
			return previous.Delivery, nil
		}
		candidate.Revision = previous.Delivery.Revision + 1
		if candidate.ObservedAt.Before(previous.Delivery.ObservedAt) {
			return Delivery{}, ErrStoreInvalid
		}
		previousSHA = previous.EventSHA256
	}
	if err := Validate(candidate); err != nil {
		return Delivery{}, err
	}
	event := DeliveryEvent{Schema: DeliveryEventSchema, Sequence: candidate.Revision, Delivery: candidate, PreviousEventSHA256: previousSHA}
	event.EventSHA256, err = digestEvent(event)
	if err != nil {
		return Delivery{}, err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return Delivery{}, err
	}
	file, err := os.OpenFile(ledgerPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640) // #nosec G304 -- validated content-addressed path.
	if err != nil {
		return Delivery{}, err
	}
	if _, err = file.Write(append(body, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return Delivery{}, err
	}
	if closeErr != nil {
		return Delivery{}, closeErr
	}
	projection, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return Delivery{}, err
	}
	if err := writeAtomic(filepath.Join(directory, key+".json"), append(projection, '\n')); err != nil {
		return Delivery{}, err
	}
	return candidate, nil
}

// Load replays and verifies the append-only hash chain; the convenience
// projection is never trusted as authority.
func Load(root, deliveryID string) (Delivery, error) {
	deliveryStoreMu.Lock()
	defer deliveryStoreMu.Unlock()
	directory, key, err := deliveryStorePath(root, deliveryID)
	if err != nil {
		return Delivery{}, err
	}
	events, err := loadEvents(filepath.Join(directory, key+".jsonl"), deliveryID)
	if err != nil {
		return Delivery{}, err
	}
	return events[len(events)-1].Delivery, nil
}

func loadEvents(path, deliveryID string) ([]DeliveryEvent, error) {
	file, err := os.Open(path) // #nosec G304 -- path is rooted and key is validated.
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	events := make([]DeliveryEvent, 0, 4)
	previousSHA := ""
	for scanner.Scan() {
		var event DeliveryEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, ErrStoreCorrupt
		}
		if event.Schema != DeliveryEventSchema || event.Sequence != uint64(len(events)+1) || event.Delivery.Revision != event.Sequence ||
			event.Delivery.DeliveryID != deliveryID || event.PreviousEventSHA256 != previousSHA || Validate(event.Delivery) != nil {
			return nil, ErrStoreCorrupt
		}
		want, err := digestEvent(event)
		if err != nil || event.EventSHA256 != want {
			return nil, ErrStoreCorrupt
		}
		previousSHA = event.EventSHA256
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, ErrNotFound
	}
	return events, nil
}

func digestEvent(event DeliveryEvent) (string, error) {
	event.EventSHA256 = ""
	body, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func equivalentDeliveryState(left, right Delivery) bool {
	left.Revision, right.Revision = 0, 0
	left.ObservedAt, right.ObservedAt = left.CreatedAt, right.CreatedAt
	return reflect.DeepEqual(left, right)
}

func deliveryStorePath(root, deliveryID string) (string, string, error) {
	root = strings.TrimSpace(root)
	const prefix = "uiai-epwa-delivery:sha256:"
	key := strings.TrimPrefix(deliveryID, prefix)
	decoded, err := hex.DecodeString(key)
	if root == "" || !strings.HasPrefix(deliveryID, prefix) || err != nil || len(decoded) != sha256.Size || key != strings.ToLower(key) {
		return "", "", ErrStoreInvalid
	}
	return filepath.Join(root, "epwa-delivery"), key, nil
}

func writeAtomic(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".delivery-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o640); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path)) // #nosec G304 -- caller supplies validated store directory.
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync delivery directory: %w", err)
	}
	return nil
}
