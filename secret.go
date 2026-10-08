// cli/secret.go
package kli

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/awnumar/memguard"
)

// Secret wraps a memguard LockedBuffer for secure secret storage
type Secret struct {
	buffer  *memguard.LockedBuffer
	cleaned atomic.Int32 // atomic flag for cleanup tracking
}

// newSecret creates a new Secret from a string value
func newSecret(value string) *Secret {
	if value == "" {
		return &Secret{}
	}

	buf := memguard.NewBufferFromBytes([]byte(value))
	s := &Secret{buffer: buf}

	// Set finalizer for automatic cleanup
	runtime.SetFinalizer(s, func(s *Secret) {
		s.finalize()
	})

	return s
}

// Bytes returns the secret value as bytes
// The returned slice is only valid until Destroy() is called
func (s *Secret) Bytes() []byte {
	if s.isDestroyed() {
		return nil
	}
	if s.buffer == nil {
		return nil
	}
	return s.buffer.Bytes()
}

// String returns the secret value as a string
// The returned string is only valid until Destroy() is called
func (s *Secret) String() string {
	if s.isDestroyed() {
		return ""
	}
	if s.buffer == nil {
		return ""
	}
	return string(s.buffer.Bytes())
}

// Destroy securely wipes the secret from memory
// Multiple calls are safe and will not panic
func (s *Secret) Destroy() {
	s.finalize()
}

// finalize performs the actual cleanup with atomic protection
func (s *Secret) finalize() {
	if s.cleaned.CompareAndSwap(0, 1) {
		if s.buffer != nil {
			s.buffer.Destroy()
			s.buffer = nil
		}
		// Prevent finalizer from running again
		runtime.SetFinalizer(s, nil)
	}
}

// isDestroyed returns true if the secret has been destroyed
func (s *Secret) isDestroyed() bool {
	return s.cleaned.Load() == 1
}

// IsSet returns true if the secret has a value and hasn't been destroyed
func (s *Secret) IsSet() bool {
	return !s.isDestroyed() && s.buffer != nil && s.buffer.Size() > 0
}

// Size returns the length of the secret
func (s *Secret) Size() int {
	if s.isDestroyed() {
		return 0
	}
	if s.buffer == nil {
		return 0
	}
	return s.buffer.Size()
}

// SecretStore holds all secrets for cleanup with thread-safe operations
type SecretStore struct {
	secrets   map[string]*Secret
	mu        sync.RWMutex
	destroyed atomic.Int32 // atomic flag for store-wide cleanup tracking
}

func newSecretStore() *SecretStore {
	ss := &SecretStore{
		secrets: make(map[string]*Secret),
	}

	// Set finalizer for automatic cleanup of the entire store
	runtime.SetFinalizer(ss, func(ss *SecretStore) {
		ss.DestroyAll()
	})

	return ss
}

// Store securely stores a secret with thread safety
func (ss *SecretStore) Store(key, value string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Destroy existing secret if present
	if existing, exists := ss.secrets[key]; exists {
		existing.Destroy()
	}

	ss.secrets[key] = newSecret(value)
}

// Get retrieves a secret with thread safety
func (ss *SecretStore) Get(key string) *Secret {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if s, ok := ss.secrets[key]; ok {
		return s
	}
	return &Secret{}
}

// DestroyAll securely destroys all secrets with verification
func (ss *SecretStore) DestroyAll() {
	// Use atomic operation to prevent double cleanup
	if !ss.destroyed.CompareAndSwap(0, 1) {
		return // Already destroyed
	}

	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Destroy all secrets
	for _, s := range ss.secrets {
		s.Destroy()
	}

	// Clear the map
	ss.secrets = make(map[string]*Secret)

	// Prevent finalizer from running again
	runtime.SetFinalizer(ss, nil)
}

// Has checks if a secret exists and is set
func (ss *SecretStore) Has(key string) bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if s, ok := ss.secrets[key]; ok {
		return s.IsSet()
	}
	return false
}
