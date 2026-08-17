package platform

// Bootstrap registers built-in platform providers (safe to call once at startup).
func Bootstrap() {
	Register(newManualProvider())
	Register(newMockProvider())
}
