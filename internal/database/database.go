package database

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
)

var (
	lock  sync.RWMutex
	memDb map[string]*MemoryCell
)

type MemoryCell struct {
	LastSeen  time.Time
	Type      entity.DeviceType
	PduDevice *entity.PDUDevice
}

func Init(ctx context.Context) {
	memDb = make(map[string]*MemoryCell)

	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanOfflineDevices()
			}
		}
	}()
}

func cleanOfflineDevices() {
	now := time.Now()
	lock.Lock()
	defer lock.Unlock()
	for key, value := range memDb {
		if value.LastSeen.Add(1 * time.Hour).Before(now) {
			delete(memDb, key)
		}
	}
}

func SetPUDDevice(_ context.Context, key string, device *entity.PDUDevice) {
	now := time.Now()
	lock.Lock()
	defer lock.Unlock()
	if value, ok := memDb[key]; ok {
		if value.Type != entity.DeviceTypePDU {
			log.Fatalf("Database: set device failed, %v type changed, old=%v new=pdu", key, value.Type)
			return
		}
		value.LastSeen = now
		value.PduDevice = device
	} else {
		memDb[key] = &MemoryCell{
			LastSeen:  now,
			Type:      entity.DeviceTypePDU,
			PduDevice: device,
		}
	}
}

func GetPDUDevice(_ context.Context, key string) (*entity.PDUDevice, bool) {
	lock.RLock()
	defer lock.RUnlock()
	if value, ok := memDb[key]; ok {
		if value.Type == entity.DeviceTypePDU {
			return value.PduDevice, true
		}
		log.Printf("Database: get device failed, %v type is not pdu", key)
		return nil, false
	}
	return nil, false
}
