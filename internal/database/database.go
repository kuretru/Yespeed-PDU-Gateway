package database

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kuretru/Yespeed-PDU-Gateway/entity"
)

var (
	lock  sync.RWMutex
	memDb map[string]map[string]*MemoryCell
)

type MemoryCell struct {
	LastSeen  time.Time
	Type      entity.DeviceType
	PduDevice *entity.PDUDevice
}

func Init(ctx context.Context) {
	memDb = make(map[string]map[string]*MemoryCell)

	//ticker := time.NewTicker(1 * time.Hour)
	//go func() {
	//	defer ticker.Stop()
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			return
	//		case <-ticker.C:
	//			cleanOfflineDevices()
	//		}
	//	}
	//}()
}

//func cleanOfflineDevices() {
//	now := time.Now()
//	lock.Lock()
//	defer lock.Unlock()
//	for key, value := range memDb {
//		if value.LastSeen.Add(1 * time.Hour).Before(now) {
//			delete(memDb, key)
//		}
//	}
//}

func SetPUDDevice(_ context.Context, nodeId string, deviceId string, device *entity.PDUDevice) {
	now := time.Now()
	lock.Lock()
	defer lock.Unlock()

	nodeDevices, ok := memDb[nodeId]
	if !ok {
		nodeDevices = make(map[string]*MemoryCell)
		memDb[nodeId] = nodeDevices
	}

	if value, ok := nodeDevices[deviceId]; ok {
		if value.Type != entity.DeviceTypePDU {
			slog.Warn("Database: set device failed, type changed",
				"deviceId", deviceId, "nodeId", nodeId, "old", value.Type)
			return
		}
		value.LastSeen = now
		value.PduDevice = device
	} else {
		nodeDevices[deviceId] = &MemoryCell{
			LastSeen:  now,
			Type:      entity.DeviceTypePDU,
			PduDevice: device,
		}
	}
}

func GetAllPDUNodes(_ context.Context) []string {
	lock.RLock()
	defer lock.RUnlock()
	result := make([]string, 0)
	for nodeId := range memDb {
		result = append(result, nodeId)
	}
	return result
}

func GetPDUNodeDevices(_ context.Context, nodeId string) []*MemoryCell {
	lock.RLock()
	defer lock.RUnlock()
	if devices, ok := memDb[nodeId]; ok {
		result := make([]*MemoryCell, 0, len(devices))
		for _, device := range devices {
			result = append(result, device)
		}
		return result
	}
	return nil
}
