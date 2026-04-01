package auth

// Manager handles authentication for providers.
type Manager struct {
	keystore *Keystore
}

// NewManager creates an auth Manager.
func NewManager(ks *Keystore) *Manager {
	return &Manager{keystore: ks}
}

// GetCredential returns the API key for a provider.
func (m *Manager) GetCredential(provider string) (string, error) {
	return m.keystore.GetKey(provider)
}

// SetKey stores an API key for a provider.
func (m *Manager) SetKey(provider, key string) error {
	return m.keystore.SaveKey(provider, key)
}

// HasCredential checks if a provider has stored credentials.
func (m *Manager) HasCredential(provider string) bool {
	_, err := m.keystore.GetKey(provider)
	return err == nil
}
