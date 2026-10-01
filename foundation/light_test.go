package foundation

import "testing"

func TestLightFalloff(t *testing.T) {
	if LightFalloff(0, 2) != 1.0 {
		t.Error("center should be full brightness")
	}
	if LightFalloff(2, 2) >= LightFalloff(1, 2) {
		t.Error("falloff should decrease with distance")
	}
	if LightFalloff(100, 2) != 0.16 {
		t.Error("far tiles clamp to 0.16")
	}
}
