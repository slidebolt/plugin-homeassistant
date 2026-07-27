package app

import (
	"testing"

	domain "github.com/slidebolt/sb-domain"
)

func TestEntityToWireUsesProfileName(t *testing.T) {
	entity := domain.Entity{
		ID:       "default-light",
		Plugin:   "plugin-zigbee",
		DeviceID: "0xabc",
		Type:     "light",
		Name:     "Light",
		Profile:  &domain.EntityProfile{Name: "Main Light Bar 01 Light", ID: "main-light-bar-01-light"},
		State:    domain.Light{Power: true},
	}

	got := entityToWire(entity)
	if got.Name != "Main Light Bar 01 Light" {
		t.Fatalf("wire name = %q, want profile name", got.Name)
	}
	if got.EntityID != "light.main_light_bar_01_light" {
		t.Fatalf("wire entity_id = %q, want profile id", got.EntityID)
	}
}

func TestEntityToWireFallsBackToEntityName(t *testing.T) {
	entity := domain.Entity{
		ID:       "light",
		Plugin:   "plugin-wiz",
		DeviceID: "wiz-1",
		Type:     "light",
		Name:     "WiZ 123456",
		State:    domain.Light{Power: true},
	}

	got := entityToWire(entity)
	if got.Name != "WiZ 123456" {
		t.Fatalf("wire name = %q, want entity name", got.Name)
	}
	if got.EntityID != "light.plugin_wiz_wiz_1_light" {
		t.Fatalf("wire entity_id = %q, want key-based fallback", got.EntityID)
	}
}
